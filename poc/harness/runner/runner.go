// Package runner runs one agent on one task in one arm (docs/poc/plan.md, section 7): a fresh
// copy of the arm's workspace, the task brief as the prompt, a budget of turns and time, and the
// logs that the measurements need.
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
	Template string // the arm's workspace: poc/app/vulcan or poc/app/go
	Arm      string
	Task     string
	Index    int
	Order    int
	Brief    string // the task brief; {{language}} becomes Vulcan or Go
	RulesMD  string // the team rules (RULES.md), copied into the workspace; may be ""
	OutDir   string // the run's directory: workspace/, base/, stream.jsonl, agent.err, diff.txt, run.json
	ToolDir  string // directory with the vulcan and fakepay commands, put first on the PATH

	// Agent is the command that runs the agent; the prompt is its last argument. When it is
	// empty, the runner uses Claude Code in headless mode with Model and MaxBudgetUSD.
	Agent        []string
	Model        string
	MaxTurns     int           // 60
	MaxTime      time.Duration // 45 minutes
	MaxBudgetUSD float64       // the token cap, as a cost
}

// Preamble is the start of every prompt, the same in both arms except the language name.
const Preamble = `You are working on Stockroom, a service written in {{language}}, in the current directory.
Read README.md first: it gives the layout, the check command, and how to run the service.
The team rules are in RULES.md. Make the change below. You are done when the change works and
the check command of README.md passes. Do not ask questions; if something is unclear, choose the
simplest reading of the request.

`

// Prompt returns the prompt of a run.
func (c *Config) Prompt() string {
	lang := "Go"
	if c.Arm == record.ArmV {
		lang = "Vulcan"
	}
	return strings.ReplaceAll(Preamble+c.Brief, "{{language}}", lang)
}

// AllowedTools are the only tools of a run, the same in both arms: the file tools, the task
// tools of background commands, and the whole shell. Any other tool is denied without a prompt
// (--permission-mode dontAsk). The shell is not restricted: Claude Code denies every shell loop
// unless the whole shell is allowed, and the owner decided to allow loops after the pilots
// (experiments/poc/pilot-p5.md). So the agent can run any command with the user's rights.
var AllowedTools = []string{
	"Read", "Edit", "Write", "Glob", "Grep", "TodoWrite",
	"BashOutput", "KillShell", "TaskOutput", "TaskStop",
	"Bash",
}

// claudeArgs is the headless Claude Code command: no user or project settings (so no
// CLAUDE.md or memories of anyone reach the agent), no MCP servers, no web tools, and only the
// tools of AllowedTools.
func (c *Config) claudeArgs() []string {
	args := []string{"claude", "-p", "--output-format", "stream-json", "--verbose",
		"--setting-sources", "local", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
		"--disallowedTools", "WebFetch,WebSearch", "--permission-mode", "dontAsk",
		"--allowedTools", strings.Join(AllowedTools, ",")}
	if c.Model != "" {
		args = append(args, "--model", c.Model)
	}
	if c.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(c.MaxBudgetUSD, 'f', 2, 64))
	}
	return args
}

// Run runs the agent and returns the record, without the grading.
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
	r := &record.Run{Task: c.Task, Arm: c.Arm, Index: c.Index, Order: c.Order, Start: time.Now().UTC()}

	args := c.Agent
	if len(args) == 0 {
		args = c.claudeArgs()
	}
	args = append(append([]string{}, args...), c.Prompt())
	ctx, cancel := context.WithTimeout(context.Background(), c.MaxTime)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = ws
	cmd.Env = c.env()
	cmd.Cancel = func() error { return killTree(cmd.Process) }
	cmd.WaitDelay = 10 * time.Second
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	errLog, err := os.Create(filepath.Join(c.OutDir, "agent.err"))
	if err != nil {
		return nil, err
	}
	defer errLog.Close()
	cmd.Stderr = errLog
	streamLog, err := os.Create(filepath.Join(c.OutDir, "stream.jsonl"))
	if err != nil {
		return nil, err
	}
	defer streamLog.Close()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	reader := agentlog.NewReader(c.Arm)
	sc := bufio.NewScanner(io.TeeReader(stdout, streamLog))
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		reader.Line(sc.Bytes())
		if reader.S.Turns > c.MaxTurns && r.BudgetHit == "" {
			r.BudgetHit = "turns"
			cancel()
		}
	}
	waitErr := cmd.Wait()
	r.WallSeconds = time.Since(r.Start).Seconds()
	if r.BudgetHit == "" && ctx.Err() == context.DeadlineExceeded {
		r.BudgetHit = "time"
	}
	s := reader.S
	r.Turns, r.ToolCalls, r.TokensIn, r.TokensOut, r.CostUSD, r.Checks, r.Denied = s.Turns, s.ToolCalls, s.TokensIn, s.TokensOut, s.CostUSD, s.Checks, s.Denied
	switch {
	case s.Error == "error_max_budget_usd" || strings.Contains(s.Error, "budget"):
		r.BudgetHit = "tokens"
	case s.Error != "":
		r.AgentError = s.Error
	case waitErr != nil && r.BudgetHit == "" && !s.Done:
		r.AgentError = waitErr.Error()
	}
	if err := diffStats(c, base, ws, r); err != nil {
		return nil, err
	}
	return r, nil
}

// env is the agent's environment: the tools first on the PATH, and Go modules from the local
// cache only (no network for modules, plan section 7).
func (c *Config) env() []string {
	var out []string
	for _, e := range os.Environ() {
		k, _, _ := strings.Cut(e, "=")
		switch u := strings.ToUpper(k); {
		case u == "PATH", u == "GOPROXY", u == "GOFLAGS", u == "GONOSUMDB", u == "GOSUMDB":
			continue
		case strings.HasPrefix(u, "CLAUDE"), strings.HasPrefix(u, "ANTHROPIC"), u == "AI_AGENT", u == "BAGGAGE":
			// The session that starts the harness may itself run in Claude Code: its variables
			// change the agent (CLAUDE_CODE_SIMPLE, CLAUDE_EFFORT, ANTHROPIC_BASE_URL) and are
			// not the same in every run, so the agent gets none of them.
			continue
		}
		out = append(out, e)
	}
	path := os.Getenv("PATH")
	if c.ToolDir != "" {
		path = c.ToolDir + string(os.PathListSeparator) + path
	}
	return append(out, "PATH="+path, "GOPROXY=off", "GOSUMDB=off")
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

// diffStats compares the workspace with its base (M7): the lines that each changed file adds and
// removes, the files, and the units. It writes diff.txt, a unified diff for people to read.
func diffStats(c *Config, base, ws string, r *record.Run) error {
	files := map[string]bool{}
	for _, root := range []string{base, ws} {
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if rel != "." && skipped(rel+"/") {
					return filepath.SkipDir
				}
				return nil
			}
			if !skipped(rel) {
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
		if bytes.Equal(a, b) {
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
