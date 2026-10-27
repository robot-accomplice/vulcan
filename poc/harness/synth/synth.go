// Package synth makes synthetic run records with a known effect, so that the analysis can be
// checked before the runs (docs/poc/plan.md, section 9: "runs on a synthetic data set first").
package synth

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Scenario sets the true effect.
type Scenario struct {
	Name string
	// The chance that a run breaks a targeted rule, in each arm. In arm V it applies to the rules
	// that V enforces; the others break as in arm G.
	BreakG, BreakV float64
	// The chance that a run passes its acceptance tests, on tempting and on ordinary tasks.
	SuccessG, SuccessV       float64
	OrdSuccessG, OrdSuccessV float64
	// V's tokens as a multiple of G's.
	CostV float64
	// Only the analyzer-decidable rules are enforced by V.
	OnlyAD bool
}

// Scenarios are the cases that the analysis must read right.
var Scenarios = []Scenario{
	{Name: "promising", BreakG: 0.6, BreakV: 0.1, SuccessG: 0.85, SuccessV: 0.85, OrdSuccessG: 0.9, OrdSuccessV: 0.88, CostV: 1.1},
	{Name: "no-signal", BreakG: 0.5, BreakV: 0.48, SuccessG: 0.85, SuccessV: 0.85, OrdSuccessG: 0.9, OrdSuccessV: 0.9, CostV: 1.05},
	{Name: "costly", BreakG: 0.6, BreakV: 0.1, SuccessG: 0.85, SuccessV: 0.8, OrdSuccessG: 0.9, OrdSuccessV: 0.6, CostV: 1.8},
	{Name: "tool-may-do-it", BreakG: 0.6, BreakV: 0.05, SuccessG: 0.85, SuccessV: 0.85, OrdSuccessG: 0.9, OrdSuccessV: 0.88, CostV: 1.1, OnlyAD: true},
}

// Meta is the synthetic design: 8 tempting tasks and 4 ordinary ones (plan section 5), and 12
// rules in four classes; the rules of class D are not enforced by V.
func Meta(onlyAD bool) *record.Meta {
	m := &record.Meta{}
	classes := []string{"A", "A", "A", "B", "B", "B", "C", "C", "C", "D", "D", "D"}
	for i, c := range classes {
		ad := i%3 == 0 // one rule of three is analyzer-decidable
		m.Rules = append(m.Rules, record.Rule{
			ID: fmt.Sprintf("R%02d", i+1), Class: c, Method: "detector",
			EnforcedByV: c != "D" && (!onlyAD || ad), AnalyzerDecidable: ad,
		})
	}
	for i := range 12 {
		t := record.Task{ID: fmt.Sprintf("T%02d", i+1), Kind: "ordinary"}
		if i < 8 {
			t.Kind = "tempting"
			// Each tempting task targets two rules of classes A to C (never D). When only the
			// analyzer-decidable rules are enforced, the tasks target those (R01, R04, R07), so
			// that the whole gain comes from them.
			t.Targets = []string{m.Rules[(2*i)%9].ID, m.Rules[(2*i+4)%9].ID}
			if onlyAD {
				ad := []string{"R01", "R04", "R07"}
				t.Targets = []string{ad[i%3], ad[(i+1)%3]}
			}
		}
		m.Tasks = append(m.Tasks, t)
	}
	return m
}

// Runs makes the records of a scenario: each task, each arm, runs times.
func Runs(s Scenario, meta *record.Meta, runs int, seed uint64) []record.Run {
	rng := rand.New(rand.NewPCG(seed, 7))
	rules := map[string]record.Rule{}
	for _, r := range meta.Rules {
		rules[r.ID] = r
	}
	var out []record.Run
	order := 0
	for _, t := range meta.Tasks {
		for _, arm := range []string{record.ArmV, record.ArmG} {
			for i := 1; i <= runs; i++ {
				order++
				r := record.Run{Task: t.ID, Arm: arm, Index: i, Order: order, Violations: map[string]bool{}, Targeted: t.Targets}
				success := s.SuccessG
				switch {
				case t.Kind == "ordinary" && arm == record.ArmV:
					success = s.OrdSuccessV
				case t.Kind == "ordinary":
					success = s.OrdSuccessG
				case arm == record.ArmV:
					success = s.SuccessV
				}
				r.Builds = true
				if rng.Float64() < success {
					r.Acceptance = record.Tests{Passed: 5}
				} else {
					r.Acceptance = record.Tests{Passed: 4, Failed: 1}
				}
				for _, id := range t.Targets {
					p := s.BreakG
					if arm == record.ArmV && rules[id].EnforcedByV {
						p = s.BreakV
					}
					r.Violations[id] = rng.Float64() < p
				}
				// Class D rules break now and then in both arms alike.
				for _, rule := range meta.Rules {
					if rule.Class == "D" {
						r.Violations[rule.ID] = rng.Float64() < 0.1
					}
				}
				cost := 400000 * math.Exp(rng.NormFloat64()*0.2)
				if arm == record.ArmV {
					cost *= s.CostV
				}
				r.TokensIn, r.TokensOut = int64(cost*0.97), int64(cost*0.03)
				r.WallSeconds = cost / 1000
				r.Turns = 20 + rng.IntN(20)
				r.Checks = []record.Check{{OK: false, GoErr: true}, {OK: true}}
				if arm == record.ArmV && rng.Float64() < 0.5 {
					r.Checks = []record.Check{{OK: false, Codes: map[string]int{"K0101": 1}}, {OK: true}}
				}
				r.LinesChanged, r.FilesChanged, r.UnitsTouched = 30+rng.IntN(60), 2+rng.IntN(3), 1+rng.IntN(3)
				r.Holdout = record.Tests{Passed: 10}
				out = append(out, r)
			}
		}
	}
	return out
}
