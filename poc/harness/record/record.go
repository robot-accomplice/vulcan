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

// Arms of the evaluation. Round 1 had V and G; round 2 (docs/poc/plan-r2.md) has P, L, and V.
const (
	ArmV = "V" // Vulcan
	ArmG = "G" // plain Go (round 1)
	ArmP = "P" // plain Go (round 2)
	ArmL = "L" // Go with Vulcan's model as a linter and a primer (round 2)
)

// IsGo reports whether an arm's agent works in Go.
func IsGo(arm string) bool { return arm != ArmV }

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
	Turns     int   `json:"turns"`
	ToolCalls int   `json:"tool_calls"`
	TokensIn  int64 `json:"tokens_in"` // input, with cache reads and writes
	TokensOut int64 `json:"tokens_out"`
	// Input tokens read from the provider's cache, part of TokensIn (round 2).
	TokensCached int64   `json:"tokens_cached,omitempty"`
	CostUSD      float64 `json:"cost_usd"`
	// The cost at the provider's prices (-prices), when known; CostUSD is what Claude Code
	// reports, at Claude's prices.
	ProviderCostUSD float64 `json:"provider_cost_usd,omitempty"`
	// The stream has no final result (the runner stopped the run), so TokensOut and CostUSD
	// are not known: the events of a message carry its input tokens, not its output.
	UsagePartial bool   `json:"usage_partial,omitempty"`
	AgentError   string `json:"agent_error,omitempty"`
	Denied       int    `json:"denied"` // tool calls that the permission rules denied
	// An API error of the provider ended the run. Such a run says nothing about the agent: it is
	// not counted, and evaluate runs the slot again.
	ProviderFailed bool `json:"provider_failed,omitempty"`
	// New entries in the user's temp directory after the run: files that a later run could see.
	TempLeftovers []string `json:"temp_leftovers,omitempty"`

	// M5 and M6: the check loop. Each run of the arm's check command, in order.
	Checks []Check `json:"checks"`

	// Round 2, measures that start at the first edit (F5): the message of the first Edit or
	// Write of a file in the workspace, the message of the first passing check after it, and the
	// input tokens up to that check (O8). 0 when it did not happen.
	FirstEditTurn       int   `json:"first_edit_turn,omitempty"`
	FirstPassTurn       int   `json:"first_pass_turn,omitempty"`
	TokensInAtFirstPass int64 `json:"tokens_in_at_first_pass,omitempty"`

	// Round 2: reviews of the change (O3b, O4), in order. The loop's reviews go back to the
	// agent; the second review of the final change is only for agreement.
	Reviews []Review `json:"reviews,omitempty"`

	// Round 2: check configuration that the run changed (F7): each entry names a file and what
	// changed. A run that changes it fails, unless its task allows it.
	ConfigChanges []string `json:"config_changes,omitempty"`
	// The task's brief asks for a change of the check configuration, so ConfigChanges do not fail
	// the run (set by the grader).
	ConfigAllowed bool `json:"config_allowed,omitempty"`
	// Round 2: files that the run left in the workspace outside the product source (F3).
	StrayFiles []string `json:"stray_files,omitempty"`

	// M7: review load.
	LinesChanged int `json:"lines_changed"`
	FilesChanged int `json:"files_changed"`
	UnitsTouched int `json:"units_touched"` // leaves (V) or packages (G)

	// Grading.
	Builds     bool            `json:"builds"`
	Acceptance Tests           `json:"acceptance"`
	Holdout    Tests           `json:"holdout"`    // M8
	Violations map[string]bool `json:"violations"` // rule id -> broken in the final change (M2)
	// Round 2, O7, from the product source, final minus base: lines in duplicated blocks, and
	// errors dropped or masked.
	CloneLines   int      `json:"clone_lines"`
	MaskedErrors int      `json:"masked_errors"`
	Targeted     []string `json:"targeted"`                // the task's targeted rules
	ToolchainBug bool     `json:"toolchain_bug,omitempty"` // M10, decided by hand
	Excluded     bool     `json:"excluded,omitempty"`      // infrastructure failure
	Note         string   `json:"note,omitempty"`
}

// Check is one run of the check command by the agent.
type Check struct {
	OK    bool           `json:"ok"`
	Codes map[string]int `json:"codes,omitempty"` // rule codes in the output: Vulcan's K-codes, and the linter's VL-codes in arm L
	GoErr bool           `json:"go_error,omitempty"`
	// "undefined:" errors in the output: names the agent used that do not exist (O7, R4).
	Undefined int `json:"undefined,omitempty"`
	Turn      int `json:"turn,omitempty"` // the message that ran it
}

// Review is one review of a change by the reviewer session (round 2, F9).
type Review struct {
	Round     int      `json:"round"` // 1, 2, ... in the loop; 0 for the second review of the final change
	Accept    bool     `json:"accept"`
	Rules     []string `json:"rules,omitempty"` // the rule ids the reviewer found broken
	Valid     bool     `json:"valid"`           // the reviewer gave a verdict in the required form
	Turns     int      `json:"turns"`
	TokensIn  int64    `json:"tokens_in"`
	TokensOut int64    `json:"tokens_out"`
	Seconds   float64  `json:"seconds"`
}

// Accepted reports whether the last review of the loop accepted the change. A run with no
// review counts as accepted, as in round 1.
func (r *Run) Accepted() bool {
	last := -1
	for i, v := range r.Reviews {
		if v.Round > 0 {
			last = i
		}
	}
	return last < 0 || r.Reviews[last].Accept
}

// Rejections is the number of the loop's reviews that rejected the change (O4).
func (r *Run) Rejections() int {
	n := 0
	for _, v := range r.Reviews {
		if v.Round > 0 && !v.Accept {
			n++
		}
	}
	return n
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

// CleanSuccess is M1: the acceptance tests pass, and no targeted rule is broken. In round 2 a
// run that changed its check configuration (F7) is not a clean success.
func (r *Run) CleanSuccess() bool {
	if !r.TaskSuccess() || (len(r.ConfigChanges) > 0 && !r.ConfigAllowed) {
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
	// The brief asks for a change of the check configuration (round 2, F7).
	AllowConfig bool `json:"allow_config,omitempty"`
}

// Rule describes one hidden rule (plan section 4).
type Rule struct {
	ID                string `json:"id"`
	Class             string `json:"class"`  // "A", "B", "C", or "D"
	Method            string `json:"method"` // "test" or "detector"
	EnforcedByV       bool   `json:"enforced_by_v"`
	AnalyzerDecidable bool   `json:"analyzer_decidable"`
	EnforcedByL       bool   `json:"enforced_by_l,omitempty"` // round 2: arm L's check rejects every seeded case
	Text              string `json:"text,omitempty"`
}

// Rule sets of round 2 (docs/poc/plan-r2.md, section 4; decision 6, section 5).
const (
	SetS = "S" // enforced by L and V
	SetG = "G" // enforced by L only
	SetU = "U" // enforced by V only
	SetN = "N" // enforced by neither
)

// Set is the rule's set in round 2.
func (r Rule) Set() string {
	switch {
	case r.EnforcedByL && r.EnforcedByV:
		return SetS
	case r.EnforcedByL:
		return SetG
	case r.EnforcedByV:
		return SetU
	}
	return SetN
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
