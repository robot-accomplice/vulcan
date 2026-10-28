// Package record holds the record of one run of the PoC evaluation (docs/poc/plan.md, sections
// 7 and 8), and the task and rule metadata that the analysis reads.
package record

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Arms of the evaluation.
const (
	ArmV = "V" // Vulcan
	ArmG = "G" // plain Go
)

// Run is one run: one agent, one task, one arm.
type Run struct {
	Task  string `json:"task"`
	Arm   string `json:"arm"`
	Index int    `json:"index"` // 1, 2, 3 for the runs of one task and arm
	Order int    `json:"order"` // the place in the schedule

	Start       time.Time `json:"start"`
	WallSeconds float64   `json:"wall_seconds"`
	BudgetHit   string    `json:"budget_hit,omitempty"` // "turns", "time", "tokens", or ""

	// M4: cost.
	Turns      int     `json:"turns"`
	ToolCalls  int     `json:"tool_calls"`
	TokensIn   int64   `json:"tokens_in"` // input, with cache reads and writes
	TokensOut  int64   `json:"tokens_out"`
	CostUSD    float64 `json:"cost_usd"`
	AgentError string  `json:"agent_error,omitempty"`
	Denied     int     `json:"denied"` // tool calls that the permission rules denied

	// M5 and M6: the check loop. Each run of the arm's check command, in order.
	Checks []Check `json:"checks"`

	// M7: review load.
	LinesChanged int `json:"lines_changed"`
	FilesChanged int `json:"files_changed"`
	UnitsTouched int `json:"units_touched"` // leaves (V) or packages (G)

	// Grading.
	Builds       bool            `json:"builds"`
	Acceptance   Tests           `json:"acceptance"`
	Holdout      Tests           `json:"holdout"`                 // M8
	Violations   map[string]bool `json:"violations"`              // rule id -> broken in the final change (M2)
	Targeted     []string        `json:"targeted"`                // the task's targeted rules
	ToolchainBug bool            `json:"toolchain_bug,omitempty"` // M10, decided by hand
	Excluded     bool            `json:"excluded,omitempty"`      // infrastructure failure
	Note         string          `json:"note,omitempty"`
}

// Check is one run of the check command by the agent.
type Check struct {
	OK    bool           `json:"ok"`
	Codes map[string]int `json:"codes,omitempty"` // Vulcan diagnostic codes in the output
	GoErr bool           `json:"go_error,omitempty"`
}

// Tests is the outcome of a hidden test suite.
type Tests struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// OK reports whether every test passed, and at least one ran.
func (t Tests) OK() bool { return t.Failed == 0 && t.Passed > 0 }

// TaskSuccess is M3: the acceptance tests pass.
func (r *Run) TaskSuccess() bool { return r.Builds && r.Acceptance.OK() }

// CleanSuccess is M1: the acceptance tests pass, and no targeted rule is broken.
func (r *Run) CleanSuccess() bool {
	if !r.TaskSuccess() {
		return false
	}
	for _, id := range r.Targeted {
		if r.Violations[id] {
			return false
		}
	}
	return true
}

// Task describes one task.
type Task struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"` // "tempting" or "ordinary"
	Targets []string `json:"targets"`
	Pilot   bool     `json:"pilot,omitempty"`
}

// Rule describes one hidden rule (plan section 4).
type Rule struct {
	ID                string `json:"id"`
	Class             string `json:"class"`  // "A", "B", "C", or "D"
	Method            string `json:"method"` // "test" or "detector"
	EnforcedByV       bool   `json:"enforced_by_v"`
	AnalyzerDecidable bool   `json:"analyzer_decidable"`
	Text              string `json:"text,omitempty"`
}

// Meta is the task and rule metadata of an evaluation.
type Meta struct {
	Tasks []Task `json:"tasks"`
	Rules []Rule `json:"rules"`
}

// Append writes runs to a JSON-lines file.
func Append(path string, runs ...Run) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, r := range runs {
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
	return nil
}

// Load reads a JSON-lines file of runs.
func Load(path string) ([]Run, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Run
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for n := 1; sc.Scan(); n++ {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var r Run
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("%s:%d: %v", path, n, err)
		}
		out = append(out, r)
	}
	return out, sc.Err()
}

// LoadMeta reads a metadata file.
func LoadMeta(path string) (*Meta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Meta
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return &m, nil
}
