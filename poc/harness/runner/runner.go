// Package runner runs one agent on one task in one arm (docs/poc/plan.md, section 7; round 2:
// docs/poc/plan-r2.md, section 6): a fresh copy of the arm's workspace, the task brief as the
// prompt, a budget of turns and time, the review loop, and the logs that the measurements need.
package runner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/robot-accomplice/vulcan/poc/harness/agentlog"
	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Config is one run.
type Config struct {
	Template string // the arm's workspace: poc/app/vulcan, poc/app/go, or arm L's
	Arm      string
	Task     string
	Index    int
	Order    int
	Brief    string // the task brief; {{language}} becomes Vulcan or Go
	RulesMD  string // the team rules (RULES.md), copied into the workspace; may be ""
	OutDir   string // the run's directory: workspace/, base/, scratch/, stream.jsonl, agent.err, diff.txt, run.json
	ToolDir  string // directory with the vulcan and fakepay commands, put first on the PATH

	// Agent is the command that runs the agent; the prompt is its last argument. When it is
	// empty, the runner uses Claude Code in headless mode with Model and MaxBudgetUSD.
	Agent        []string
	Model        string
	BaseURL      string        // an Anthropic-compatible endpoint of another provider, or "" for Claude Code's own login
	Token        string        // the key of BaseURL
	MaxTurns     int           // 60
	MaxTime      time.Duration // 45 minutes
	MaxBudgetUSD float64       // the token cap, as a cost

	Sandbox *Sandbox // round 2 (F4): run the agent in WSL with its own /tmp and home; nil runs it on the host
	Review  *Review  // round 2 (F9): the review loop; nil for none
	Prices  *Price   // round 2 (F2): the provider's prices of the model, for ProviderCostUSD
}

// Price is a model's price, in USD per million tokens.
type Price struct {
	Input       float64 `json:"input"`
	CachedInput float64 `json:"cached_input"`
	Output      float64 `json:"output"`
}

// Cost is the cost of a run's tokens.
func (p *Price) Cost(in, cached, out int64) float64 {
	return (float64(in-cached)*p.Input + float64(cached)*p.CachedInput + float64(out)*p.Output) / 1e6
}

// Preamble is the start of every prompt, the same in every arm except the language name and the
// scratch directory. Round 2 (F3, F6) moved scratch files out of the workspace and added the stop
// sentence; round 1 asked for scratch files "inside the current directory".
const Preamble = `You are working on Stockroom, a service written in {{language}}, in the current directory.
Read README.md first: it gives the layout, the check command, and how to run the service.
The team rules are in RULES.md. Make the change below. You are done when the change works and
the check command of README.md passes. Do not ask questions; if something is unclear, choose the
simplest reading of the request. Keep scratch files (notes, test scripts, logs, binaries) in
{{scratch}}, not in the current directory. When the check passes and the request is met, stop.

`

func language(arm string) string {
	if arm == record.ArmV {
		return "Vulcan"
	}
	return "Go"
}

// Prompt returns the prompt of a run.
func (c *Config) Prompt() string {
	return strings.NewReplacer("{{language}}", language(c.Arm), "{{scratch}}", c.agentScratch()).Replace(Preamble + c.Brief)
}

// scratch is the run's scratch directory on the host; agentScratch is the same directory as
// the agent sees it.
func (c *Config) scratch() string {
	s, _ := filepath.Abs(filepath.Join(c.OutDir, "scratch"))
	return s
}

func (c *Config) agentScratch() string {
	if c.Sandbox != nil {
		return sandboxRun + "/scratch"
	}
	return c.scratch()
}

// AllowedTools are the only tools of a run, the same in every arm: the file tools, the task
// tools of background commands, and the whole shell. Any other tool is denied without a prompt
// (--permission-mode dontAsk). The shell is not restricted: Claude Code denies every shell loop
// unless the whole shell is allowed, and the owner decided to allow loops after the pilots
// (experiments/poc/pilot-p5.md). In round 2 the sandbox (F4) limits what the shell can reach.
var AllowedTools = []string{
	"Read", "Edit", "Write", "Glob", "Grep", "TodoWrite",
	"BashOutput", "KillShell", "TaskOutput", "TaskStop",
	"Bash",
}

// claudeArgs is the headless Claude Code command: no user or project settings (so no
// CLAUDE.md or memories of anyone reach the agent), no MCP servers, no web tools, and only the
// tools of AllowedTools. Partial messages give the output tokens of a run that the runner
// stops (F2). resume continues a session in the review loop.
func (c *Config) claudeArgs(resume string) []string {
	args := []string{"claude", "-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages",
		"--setting-sources", "local", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
		"--disallowedTools", "WebFetch,WebSearch", "--permission-mode", "dontAsk",
		"--allowedTools", strings.Join(AllowedTools, ","), "--add-dir", c.agentScratch()}
	if resume != "" {
		args = append(args, "--resume", resume)
	}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	if c.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(c.MaxBudgetUSD, 'f', 2, 64))
	}
	return args
}

// Run runs the agent, and the review loop if there is one, and returns the record, without the
// grading.
func Run(c *Config) (*record.Run, error) {
	if c.MaxTurns == 0 {
		c.MaxTurns = 60
	}
	if c.MaxTime == 0 {
		c.MaxTime = 45 * time.Minute
	}
	ws := filepath.Join(c.OutDir, "workspace")
	base := filepath.Join(c.OutDir, "base")
	for _, d := range []string{ws, base} {
		if err := copyTree(c.Template, d); err != nil {
			return nil, err
		}
		if c.RulesMD != "" {
			if err := copyFile(c.RulesMD, filepath.Join(d, "RULES.md")); err != nil {
				return nil, err
			}
		}
	}
	for _, d := range []string{"tmp", "scratch", "home"} {
		if err := os.MkdirAll(filepath.Join(c.OutDir, d), 0o755); err != nil {
			return nil, err
		}
	}
	tempBefore := tempEntries()
	r := &record.Run{Task: c.Task, Arm: c.Arm, Index: c.Index, Order: c.Order, Start: time.Now().UTC()}

	errLog, err := os.Create(filepath.Join(c.OutDir, "agent.err"))
	if err != nil {
		return nil, err
	}
	defer errLog.Close()
	streamLog, err := os.Create(filepath.Join(c.OutDir, "stream.jsonl"))
	if err != nil {
		return nil, err
	}
	defer streamLog.Close()

	reader := agentlog.NewReader(c.Arm)
	reader.Scratch = c.agentScratch()
	deadline := time.Now().Add(c.MaxTime)
	prompt := c.Prompt()
	resume := ""
	for round := 1; ; round++ {
		waitErr, err := c.agentRound(reader, r, prompt, resume, deadline, streamLog, errLog)
		if err != nil {
			return nil, err
		}
		s := reader.S
		if s.Provider != "" || r.BudgetHit != "" || s.Error != "" {
			break
		}
		if waitErr != nil && !s.Done {
			r.AgentError = waitErr.Error()
			break
		}
		if c.Review == nil {
			break
		}
		if err := diffStats(c, base, ws, r); err != nil {
			return nil, err
		}
		rev, comments, err := c.review(round, ws)
		if err != nil {
			return nil, err
		}
		r.Reviews = append(r.Reviews, rev)
		if rev.Accept || round > c.Review.MaxResubmits || s.Session == "" || reader.S.Turns >= c.MaxTurns || time.Now().After(deadline) {
			break
		}
		prompt, resume = rejectionPrompt(comments, rev.Rules), s.Session
	}
	r.WallSeconds = time.Since(r.Start).Seconds()
	s := reader.S
	r.Turns, r.ToolCalls, r.TokensIn, r.TokensOut, r.CostUSD, r.Checks, r.Denied = s.Turns, s.ToolCalls, s.TokensIn, s.TokensOut, s.CostUSD, s.Checks, s.Denied
	r.TokensCached = s.Cached
	r.FirstEditTurn, r.FirstPassTurn, r.TokensInAtFirstPass = s.FirstEdit, s.FirstPass, s.TokensInAtFirstPass
	r.UsagePartial = !s.Done
	if c.Prices != nil {
		r.ProviderCostUSD = c.Prices.Cost(r.TokensIn, r.TokensCached, r.TokensOut)
	}
	switch {
	case s.Provider != "":
		r.AgentError, r.ProviderFailed = "provider: "+s.Provider, true
	case s.Error == "error_max_budget_usd" || strings.Contains(s.Error, "budget"):
		r.BudgetHit = "tokens"
	case s.Error != "":
		r.AgentError = s.Error
	}
	r.TempLeftovers = newEntries(tempBefore, tempEntries())
	if err := diffStats(c, base, ws, r); err != nil {
		return nil, err
	}
	r.ConfigChanges = configChanges(c.Arm, base, ws)
	r.StrayFiles = strayFiles(base, ws)
	if c.Review != nil && !r.ProviderFailed {
		// The second review of the final change, only for the agreement of the reviewer.
		rev, _, err := c.review(0, ws)
		if err != nil {
			return nil, err
		}
		r.Reviews = append(r.Reviews, rev)
	}
	return r, nil
}

// agentRound runs the agent once: the first prompt, or the reviewer's comments in a resumed
// session. The turn budget and the time budget count over all rounds.
func (c *Config) agentRound(reader *agentlog.Reader, r *record.Run, prompt, resume string, deadline time.Time, streamLog, errLog io.Writer) (waitErr, err error) {
	args := c.Agent
	if len(args) == 0 {
		args = c.claudeArgs(resume)
	}
	args = append(append([]string{}, args...), prompt)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	cmd, cleanup, err := c.command(ctx, args)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = errLog
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if resume != "" {
		reader.NewRound()
	}
	sc := bufio.NewScanner(io.TeeReader(stdout, streamLog))
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		reader.Line(sc.Bytes())
		if reader.S.Turns > c.MaxTurns && r.BudgetHit == "" {
			r.BudgetHit = "turns"
			cancel()
		}
	}
	waitErr = cmd.Wait()
	if r.BudgetHit == "" && ctx.Err() == context.DeadlineExceeded {
		r.BudgetHit = "time"
	}
	return waitErr, nil
}

// command makes the agent's process: on the host in the workspace, or in the sandbox.
func (c *Config) command(ctx context.Context, args []string) (*exec.Cmd, func(), error) {
	if c.Sandbox != nil {
		return c.Sandbox.command(ctx, c, args)
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = filepath.Join(c.OutDir, "workspace")
	cmd.Env = c.env()
	cmd.Cancel = func() error { return killTree(cmd.Process) }
	cmd.WaitDelay = 10 * time.Second
	return cmd, func() {}, nil
}

func rejectionPrompt(comments string, rules []string) string {
	var b strings.Builder
	b.WriteString("A reviewer rejected the change.\n\n")
	b.WriteString(strings.TrimSpace(comments))
	if len(rules) > 0 {
		b.WriteString("\n\nRules the reviewer found broken: " + strings.Join(rules, ", ") + ".")
	}
	b.WriteString("\n\nRevise the change. The same rules and the same check command apply. When the check passes and the request is met, stop.\n")
	return b.String()
}

// stripped reports whether the agent must not get an environment variable: the session that
// starts the harness may itself run in Claude Code, and its variables change the agent
// (CLAUDE_CODE_SIMPLE, CLAUDE_EFFORT, ANTHROPIC_BASE_URL) and are not the same in every run.
func stripped(k string) bool {
	switch u := strings.ToUpper(k); {
	case u == "PATH", u == "GOPROXY", u == "GOFLAGS", u == "GONOSUMDB", u == "GOSUMDB", u == "VULCAN_AGENT_TOKEN",
		u == "TMP", u == "TEMP", u == "TMPDIR":
		return true
	case strings.HasPrefix(u, "CLAUDE"), strings.HasPrefix(u, "ANTHROPIC"), u == "AI_AGENT", u == "BAGGAGE":
		return true
	}
	return false
}

// providerEnv is the environment of another provider: every model role is the run's model, so
// no request goes to a Claude model. The cost that Claude Code reports, and so
// --max-budget-usd, uses Claude prices; the tokens are exact. The token is not in it.
func (c *Config) providerEnv() []string {
	if c.BaseURL == "" {
		return nil
	}
	out := []string{"ANTHROPIC_BASE_URL=" + c.BaseURL, "ANTHROPIC_API_KEY=", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1"}
	if c.Model != "" {
		for _, k := range []string{"ANTHROPIC_MODEL", "ANTHROPIC_SMALL_FAST_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL",
			"ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL", "CLAUDE_CODE_SUBAGENT_MODEL"} {
			out = append(out, k+"="+c.Model)
		}
	}
	return out
}

// env is the agent's environment on the host: the tools first on the PATH, and Go modules from
// the local cache only (no network for modules, plan section 7).
func (c *Config) env() []string {
	var out []string
	for _, e := range os.Environ() {
		if k, _, _ := strings.Cut(e, "="); !stripped(k) {
			out = append(out, e)
		}
	}
	path := os.Getenv("PATH")
	if c.ToolDir != "" {
		path = c.ToolDir + string(os.PathListSeparator) + path
	}
	tmp, _ := filepath.Abs(filepath.Join(c.OutDir, "tmp"))
	out = append(out, "PATH="+path, "GOPROXY=off", "GOSUMDB=off", "TMP="+tmp, "TEMP="+tmp, "TMPDIR="+tmp)
	out = append(out, c.providerEnv()...)
	if c.BaseURL != "" {
		out = append(out, "ANTHROPIC_AUTH_TOKEN="+c.Token)
	}
	return out
}

// The shell of Git for Windows maps /tmp to the user's temp directory whatever TMP says, so a
// run on the host can leave files there that a later run sees. tempEntries lists that directory,
// and the run records what is new in it (TempLeftovers), without removing anything: other
// programs use it too. In the sandbox (F4) a run has its own /tmp.
func tempEntries() map[string]bool {
	m := map[string]bool{}
	des, _ := os.ReadDir(os.TempDir())
	for _, d := range des {
		m[d.Name()] = true
	}
	return m
}

func newEntries(before, after map[string]bool) []string {
	var out []string
	for n := range after {
		if !before[n] {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

func killTree(p *os.Process) error {
	if p == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		// The agent starts child processes; kill the tree.
		return exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(p.Pid)).Run()
	}
	return p.Kill()
}

// diffStats compares the product source of the workspace with its base (M7): the lines that each
// changed file adds and removes, the files, and the units. Files outside the product source and
// binary files are not counted (F3). It writes diff.txt, a unified diff for people to read.
func diffStats(c *Config, base, ws string, r *record.Run) error {
	r.LinesChanged, r.FilesChanged, r.UnitsTouched = 0, 0, 0
	tops := topEntries(base)
	files := map[string]bool{}
	for _, root := range []string{base, ws} {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if rel != "." && (skipped(rel+"/") || !tops[strings.Split(rel, "/")[0]]) {
					return filepath.SkipDir
				}
				return nil
			}
			if !skipped(rel) && tops[strings.Split(rel, "/")[0]] {
				files[rel] = true
			}
			return nil
		})
	}
	var names []string
	for f := range files {
		names = append(names, f)
	}
	sort.Strings(names)
	units := map[string]bool{}
	var patch bytes.Buffer
	for _, rel := range names {
		a, _ := os.ReadFile(filepath.Join(base, filepath.FromSlash(rel)))
		b, _ := os.ReadFile(filepath.Join(ws, filepath.FromSlash(rel)))
		if bytes.Equal(a, b) || isBinary(a) || isBinary(b) {
			continue
		}
		r.LinesChanged += changedLines(splitLines(a), splitLines(b))
		r.FilesChanged++
		units[unitOf(rel)] = true
		cmd := exec.Command("git", "diff", "--no-index", "--", filepath.Join(base, filepath.FromSlash(rel)), filepath.Join(ws, filepath.FromSlash(rel)))
		var out bytes.Buffer
		cmd.Stdout = &out
		_ = cmd.Run() // exit code 1 means "differences"
		fmt.Fprintf(&patch, "=== %s\n%s\n", rel, out.String())
	}
	r.UnitsTouched = len(units)
	return os.WriteFile(filepath.Join(c.OutDir, "diff.txt"), patch.Bytes(), 0o644)
}

// topEntries are the names at the top of the base workspace: the product source lives under them.
func topEntries(base string) map[string]bool {
	m := map[string]bool{}
	des, _ := os.ReadDir(base)
	for _, d := range des {
		m[d.Name()] = true
	}
	return m
}

func isBinary(b []byte) bool { return bytes.IndexByte(b, 0) >= 0 }

func splitLines(b []byte) []string {
	s := strings.ReplaceAll(string(b), "\r\n", "\n")
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// changedLines counts the lines that a line diff adds and removes: the lines outside a longest
// common subsequence.
func changedLines(a, b []string) int {
	// Trim the common head and tail, which keeps the table small for small edits.
	for len(a) > 0 && len(b) > 0 && a[0] == b[0] {
		a, b = a[1:], b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1] == b[len(b)-1] {
		a, b = a[:len(a)-1], b[:len(b)-1]
	}
	if len(a) == 0 || len(b) == 0 {
		return len(a) + len(b)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			switch {
			case a[i-1] == b[j-1]:
				cur[j] = prev[j-1] + 1
			case prev[j] >= cur[j-1]:
				cur[j] = prev[j]
			default:
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
	}
	return len(a) + len(b) - 2*prev[len(b)]
}

func skipped(rel string) bool {
	return strings.HasPrefix(rel, ".vulcan/") || strings.HasPrefix(rel, ".claude/") || strings.HasPrefix(rel, ".git/") ||
		strings.HasSuffix(rel, ".db") || strings.HasSuffix(rel, ".db-journal")
}

// unitOf names the leaf or package of a file: api/orders, lib/shop, cmd/stockroom, or the file
// itself at the top (system.vulcan).
func unitOf(rel string) string {
	parts := strings.Split(rel, "/")
	if len(parts) >= 3 {
		return parts[0] + "/" + parts[1]
	}
	return rel
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() && (d.Name() == ".vulcan" || d.Name() == ".git" || d.Name() == ".claude") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".db") {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyFile(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// Save writes the record of a run into its directory.
func Save(c *Config, r *record.Run) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.OutDir, "run.json"), b, 0o644)
}

// Units lists the units of a workspace, for reports.
func Units(ws string) []string {
	var out []string
	for _, top := range []string{"api", "lib", "cmd"} {
		entries, _ := os.ReadDir(filepath.Join(ws, top))
		for _, e := range entries {
			if e.IsDir() {
				out = append(out, top+"/"+e.Name())
			}
		}
	}
	sort.Strings(out)
	return out
}
