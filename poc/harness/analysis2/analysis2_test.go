package analysis2

import (
	"testing"

	"github.com/robot-accomplice/vulcan/poc/harness/synth2"
)

// Runs of the planned design (plan-r2, section 7): 3 per tempting task and arm, 5 per ordinary
// task and arm.
const runs, ordRuns = 3, 5

// TestScenarios is the dry run of F10. Over 200 seeds of each synthetic scenario, the analysis
// gives an expected row at least 80% of the time (decision 6's power), and a world with no effect
// gives row 2 or 3 at most 5% of the time.
func TestScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("200 seeds per scenario")
	}
	meta := synth2.Meta()
	const seeds = 200
	for _, s := range synth2.Scenarios {
		hits := map[int]int{}
		for seed := uint64(1); seed <= seeds; seed++ {
			hits[Analyze(synth2.Runs(s, meta, runs, ordRuns, seed), meta).Row]++
		}
		right := 0
		for _, row := range s.Expect {
			right += hits[row]
		}
		if share := float64(right) / seeds; share < 0.8 {
			t.Errorf("%s: expected rows %v in %.0f%% of seeds, want at least 80%%: %v", s.Name, s.Expect, 100*share, hits)
		}
		if s.Name == "no-effect" {
			if fp := float64(hits[2]+hits[3]) / seeds; fp > 0.05 {
				t.Errorf("no-effect: rows 2 or 3 in %.1f%% of seeds, want at most 5%%: %v", 100*fp, hits)
			}
		}
		t.Logf("%s: %v", s.Name, hits)
	}
}
