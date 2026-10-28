package runner

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/robot-accomplice/vulcan/poc/harness/agentlog"
	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Review is the review loop of round 2 (F9; decision 6, section 6: "the agent gets the
// reviewer's comments and can revise and resubmit within the run's budget"). The reviewer is a
// headless Claude Code session with read-only tools, a fixed prompt and checklist, the brief,
// RULES.md, and the diff. It runs on the host: it cannot change files or run commands.
type Review struct {
	Model        string        // the reviewer's model; "" is the run's model
	MaxResubmits int           // resubmissions after a rejection, within the run's budget
	MaxTurns     int           // the reviewer's turn budget (30)
	MaxTime      time.Duration // the reviewer's time budget (10 minutes)
	Agent        []string      // replaces claude, for tests of the harness
}

// ReviewTools are the reviewer's only tools.
var ReviewTools = []string{"Read", "Glob", "Grep"}

// ReviewPrompt is the reviewer's prompt, the same in every arm except the language name. It is
// fixed for a round, so that two reviews of one change differ only by the model's variation.
const ReviewPrompt = `You review a change to Stockroom, a service written in {{language}}. The current directory holds
the code with the change; its diff is below. Review the change against the request and the team
rules in RULES.md, with this checklist. You can read any file. Do not change anything.

1. Does the change do what the request asks?
2. Does it break any rule of RULES.md? Name each broken rule by its id in RULES.md.
3. Does it add a defect: a wrong result, a lost error, a race, or broken behavior elsewhere?

Accept the change only if all three answers are good. End your answer with one line of JSON,
and nothing after it:
{"verdict": "accept" or "reject", "broken_rules": ["<rule id>", ...], "comments": "<what the author must change>"}

The request:

{{brief}}

The diff:

{{diff}}
`

// maxDiff caps the diff in the reviewer's prompt.
const maxDiff = 200_000

type verdict struct {
	Verdict  string   `json:"verdict"`
	Broken   []string `json:"broken_rules"`
	Comments string   `json:"comments"`
}

// parseVerdict reads the last JSON line of the reviewer's answer.
func parseVerdict(text string) (verdict, bool) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.Trim(strings.TrimSpace(lines[i]), "`")
		if !strings.HasPrefix(l, "{") {
			continue
		}
		var v verdict
		if json.Unmarshal([]byte(l), &v) == nil && (v.Verdict == "accept" || v.Verdict == "reject") {
			return v, true
		}
		return verdict{}, false
	}
	return verdict{}, false
}

func (c *Config) reviewPrompt() (string, error) {
	diff, err := os.ReadFile(filepath.Join(c.OutDir, "diff.txt"))
	if err != nil {
		return "", err
	}
	d := string(diff)
	if len(d) > maxDiff {
		d = d[:maxDiff] + "\n[the diff is cut here]\n"
	}
	if strings.TrimSpace(d) == "" {
		d = "(no change)"
	}
	return strings.NewReplacer("{{language}}", language(c.Arm), "{{brief}}", strings.TrimSpace(c.Brief), "{{diff}}", d).Replace(ReviewPrompt), nil
}

// review runs the reviewer once on the workspace. Round 0 is the second review of the final
// change. A review that ends without a verdict in the required form counts as an acceptance, so
// that a fault of the reviewer never fails the agent's run, and is marked not Valid.
func (c *Config) review(round int, ws string) (record.Review, string, error) {
	rv := c.Review
	if rv.MaxTurns == 0 {
		rv.MaxTurns = 30
	}
	if rv.MaxTime == 0 {
		rv.MaxTime = 10 * time.Minute
	}
	prompt, err := c.reviewPrompt()
	if err != nil {
		return record.Review{}, "", err
	}
	args := rv.Agent
	if len(args) == 0 {
		args = []string{"claude", "-p", "--output-format", "stream-json", "--verbose", "--include-partial-messages",
			"--setting-sources", "local", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
			"--permission-mode", "dontAsk", "--allowedTools", strings.Join(ReviewTools, ",")}
		model := rv.Model
		if model == "" {
			model = c.Model
		}
		if model != "" {
			args = append(args, "--model", model)
		}
	}
	args = append(append([]string{}, args...), prompt)
	ctx, cancel := context.WithTimeout(context.Background(), rv.MaxTime)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = ws
	cmd.Env = c.env()
	cmd.Cancel = func() error { return killTree(cmd.Process) }
	cmd.WaitDelay = 10 * time.Second
	name := "review-" + strconv.Itoa(round)
	logf, err := os.Create(filepath.Join(c.OutDir, name+".jsonl"))
	if err != nil {
		return record.Review{}, "", err
	}
	defer logf.Close()
	cmd.Stderr = logf
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return record.Review{}, "", err
	}
	start := time.Now()
	if err := cmd.Start(); err != nil {
		return record.Review{}, "", err
	}
	reader := agentlog.NewReader("")
	sc := bufio.NewScanner(io.TeeReader(stdout, logf))
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		reader.Line(sc.Bytes())
		if reader.S.Turns > rv.MaxTurns {
			cancel()
		}
	}
	_ = cmd.Wait()
	s := reader.S
	out := record.Review{Round: round, Accept: true, Turns: s.Turns, TokensIn: s.TokensIn, TokensOut: s.TokensOut,
		Seconds: time.Since(start).Seconds()}
	v, ok := parseVerdict(s.Result)
	if !ok {
		return out, "", nil
	}
	out.Valid, out.Accept, out.Rules = true, v.Verdict == "accept", v.Broken
	return out, v.Comments, nil
}
