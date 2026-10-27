package analysis

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
	"github.com/robot-accomplice/vulcan/poc/harness/synth"
)

// Each synthetic scenario has a known effect; with 3 runs per task and arm, as in the plan, the
// analysis must read it right.
func TestScenarios(t *testing.T) {
	want := map[string]string{
		"promising":      "Promising",
		"no-signal":      "No signal",
		"costly":         "Costly",
		"tool-may-do-it": "Promising, but a tool may do it",
	}
	for _, s := range synth.Scenarios {
		meta := synth.Meta(s.OnlyAD)
		runs := synth.Runs(s, meta, 3, 1)
		rep := Analyze(runs, meta)
		if rep.Verdict != want[s.Name] {
			var b bytes.Buffer
			rep.Markdown(&b, s.Name)
			t.Errorf("%s: verdict %q, want %q\n%s", s.Name, rep.Verdict, want[s.Name], b.String())
		}
	}
}

func TestSignFlip(t *testing.T) {
	// All positive differences of 4 tasks: only the two all-same-sign assignments reach the mean.
	d := compare([]float64{1, 1, 1, 1})
	if math.Abs(d.P-2.0/16) > 1e-9 || d.Mean != 1 {
		t.Fatalf("p %v mean %v", d.P, d.Mean)
	}
	z := compare([]float64{1, -1, 1, -1})
	if z.P != 1 {
		t.Fatalf("p of a zero mean: %v", z.P)
	}
}

func TestToolchainBugsAndExclusions(t *testing.T) {
	meta := &record.Meta{Tasks: []record.Task{{ID: "T1", Kind: "tempting", Targets: []string{"R1"}}}, Rules: []record.Rule{{ID: "R1", Class: "A", EnforcedByV: true}}}
	ok := record.Run{Task: "T1", Builds: true, Acceptance: record.Tests{Passed: 1}, Violations: map[string]bool{}}
	v1, v2, g := ok, ok, ok
	v1.Arm, v2.Arm, g.Arm = "V", "V", "G"
	v2.ToolchainBug = true
	g.Violations = map[string]bool{"R1": true}
	ex := ok
	ex.Arm, ex.Excluded = "G", true
	rep := Analyze([]record.Run{v1, v2, g, ex}, meta)
	if rep.Bugs != 1 || rep.Excluded != 1 {
		t.Fatalf("bugs %d excluded %d", rep.Bugs, rep.Excluded)
	}
	// With the bug run counted as a failure, V's clean rate is 0.5; without it, 1.
	if rep.Clean["all"].Mean != 0.5 {
		t.Fatalf("clean difference %v", rep.Clean["all"].Mean)
	}
	var b bytes.Buffer
	rep.Markdown(&b, "x")
	if !strings.Contains(b.String(), "Without the 1 runs with a toolchain bug") {
		t.Fatal(b.String())
	}
}
