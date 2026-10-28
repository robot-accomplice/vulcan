// Package synth2 makes synthetic runs of round 2 with a known effect, so that the analysis of
// round 2 (package analysis2) can be checked before any run (docs/poc/plan-r2.md, section 6, F10).
package synth2

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Arm is the true behavior of one arm.
type Arm struct {
	BreakS  float64 // the chance that a run breaks each targeted rule of S
	Success float64 // the chance that a run passes its acceptance tests, on tempting tasks
	OrdSucc float64 // the same on ordinary tasks
	Cost    float64 // tokens and turns, as a multiple of P's
	Review  float64 // the reviewer's tokens, as a multiple of P's
}

// Scenario is a true state of the world, and the verdict rows that the analysis may give for it.
type Scenario struct {
	Name    string
	P, L, V Arm
	Expect  []int
}

// Scenarios are the cases that the analysis must read right (plan-r2, section 6, F10).
var Scenarios = []Scenario{
	{Name: "not-tempting", Expect: []int{1},
		P: Arm{0.05, 0.85, 0.9, 1, 1}, L: Arm{0.02, 0.85, 0.9, 1, 1}, V: Arm{0.02, 0.85, 0.9, 1, 1}},
	{Name: "model-helps", Expect: []int{2},
		P: Arm{0.5, 0.85, 0.9, 1, 1}, L: Arm{0.08, 0.85, 0.9, 1.03, 1}, V: Arm{0.08, 0.85, 0.9, 1.08, 1}},
	{Name: "v-worse-than-l", Expect: []int{2},
		P: Arm{0.5, 0.85, 0.9, 1, 1}, L: Arm{0.08, 0.85, 0.9, 1.03, 1}, V: Arm{0.08, 0.65, 0.75, 1.4, 1}},
	{Name: "language-adds", Expect: []int{3},
		P: Arm{0.5, 0.85, 0.9, 1, 1}, L: Arm{0.2, 0.85, 0.9, 1.03, 1}, V: Arm{0.0, 0.85, 0.9, 1.05, 0.7}},
	{Name: "costly", Expect: []int{4},
		P: Arm{0.5, 0.85, 0.9, 1, 1}, L: Arm{0.08, 0.85, 0.9, 1.6, 1}, V: Arm{0.08, 0.85, 0.9, 1.6, 1}},
	{Name: "no-effect", Expect: []int{5, 6},
		P: Arm{0.5, 0.85, 0.9, 1, 1}, L: Arm{0.5, 0.85, 0.9, 1, 1}, V: Arm{0.5, 0.85, 0.9, 1, 1}},
}

// Meta is the synthetic design of plan-r2 section 4: 24 tempting and 8 ordinary tasks; 10 rules
// in S, 2 in G, and 4 in N (class D). Each tempting task targets one or two rules of S.
func Meta() *record.Meta {
	m := &record.Meta{}
	for i := 1; i <= 16; i++ {
		r := record.Rule{ID: fmt.Sprintf("R%02d", i), Method: "test"}
		switch {
		case i <= 10:
			r.Class, r.EnforcedByL, r.EnforcedByV = []string{"A", "B", "C"}[i%3], true, true
		case i <= 12:
			r.Class, r.EnforcedByL = "B", true
		default:
			r.Class = "D"
		}
		m.Rules = append(m.Rules, r)
	}
	for i := 1; i <= 32; i++ {
		t := record.Task{ID: fmt.Sprintf("T%02d", i), Kind: "ordinary"}
		if i <= 24 {
			t.Kind = "tempting"
			t.Targets = []string{fmt.Sprintf("R%02d", (i-1)%10+1)}
			if i%3 == 0 {
				t.Targets = append(t.Targets, fmt.Sprintf("R%02d", i%10+1))
			}
		}
		m.Tasks = append(m.Tasks, t)
	}
	return m
}

// Runs makes runs per task and arm of a scenario: runs on tempting tasks, ordRuns on ordinary
// ones. Tasks differ in how tempting and how large they are, as real tasks do.
func Runs(s Scenario, meta *record.Meta, runs, ordRuns int, seed uint64) []record.Run {
	rng := rand.New(rand.NewPCG(seed, seed^0x51ed270b2a2a1d9b))
	lognorm := func(sd float64) float64 { return math.Exp(rng.NormFloat64()*sd - sd*sd/2) }
	var out []record.Run
	for _, t := range meta.Tasks {
		tempt := math.Min(1.8, lognorm(0.4)) // how tempting the task is
		size := lognorm(0.35)                // how large
		for _, arm := range []string{record.ArmP, record.ArmL, record.ArmV} {
			a := map[string]Arm{record.ArmP: s.P, record.ArmL: s.L, record.ArmV: s.V}[arm]
			n := runs
			if t.Kind == "ordinary" {
				n = ordRuns
			}
			for i := 1; i <= n; i++ {
				r := record.Run{Task: t.ID, Arm: arm, Index: i, Builds: true, Violations: map[string]bool{}}
				succ := a.Success
				if t.Kind == "ordinary" {
					succ = a.OrdSucc
				}
				if rng.Float64() < succ {
					r.Acceptance = record.Tests{Passed: 5}
				} else {
					r.Acceptance = record.Tests{Passed: 3, Failed: 2}
				}
				r.Holdout = record.Tests{Passed: 14}
				if rng.Float64() < 0.05 {
					r.Holdout = record.Tests{Passed: 13, Failed: 1}
				}
				for _, id := range t.Targets {
					r.Violations[id] = rng.Float64() < math.Min(1, a.BreakS*tempt)
				}
				if rng.Float64() < 0.08 {
					r.Violations[fmt.Sprintf("R%02d", 13+rng.IntN(4))] = true // an N rule, equal in every arm (O5)
				}
				cost := a.Cost * size * lognorm(0.2)
				r.TokensIn, r.TokensOut = int64(800000*cost), int64(20000*cost)
				r.Turns = int(math.Max(1, math.Round(22*cost)))
				r.LinesChanged = int(math.Max(1, 80*size*lognorm(0.3)))
				r.Reviews = []record.Review{{Round: 1, Accept: true, Valid: true, TokensIn: int64(50000 * a.Review * size * lognorm(0.2)), TokensOut: 2000}}
				out = append(out, r)
			}
		}
	}
	return out
}
