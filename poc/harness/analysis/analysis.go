// Package analysis is the analysis script of the PoC evaluation (docs/poc/plan.md, section 9).
// It is frozen before the runs. The unit is the task: each measurement is averaged over the runs
// of a task and an arm, and the arms are compared by the paired per-task difference, V minus G.
//
// The reading rules of section 9, made exact here before any data exists:
//
//   - "Clean success" (M1) is a run whose acceptance tests pass and that breaks no targeted rule.
//   - "Points" are percentage points of a rate.
//   - The tempting gain is the mean over tempting tasks of V's clean-success rate minus G's.
//   - The ordinary condition holds when the mean over ordinary tasks of V's task-success rate
//     minus G's is at least -10 points.
//   - The cost ratio is the geometric mean over tasks of V's mean tokens (input with cache, and
//     output) divided by G's. The condition holds at 1.3 or less. Time and turns are reported.
//   - Class D is "about equal" when the mean over tasks of V's rate of runs that break a class D
//     rule minus G's is within ±10 points (it holds when there is no class D rule).
//   - "The gain comes only from analyzer-decidable rules" when the tempting gain, computed with
//     only the targeted rules that are not analyzer-decidable, is less than 10 points.
//   - The verdict: "No signal" if the tempting gain is less than 10 points; else "Promising"
//     (or "Promising, but a tool may do it") if it is at least 20 points and the ordinary, cost,
//     and class D conditions hold; else "Costly" if the ordinary or the cost condition fails;
//     else "Unclear".
//   - A run marked Excluded is left out. A run with a toolchain bug counts as a failure of its
//     arm; the report also gives the verdict without those runs (section 7).
package analysis

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// goCodes are the diagnostics of plain Go problems; every other K-code is a rule of Vulcan (M5).
var goCodes = map[string]bool{"K0001": true, "K0008": true, "K0209": true, "K0706": true}

// Seed fixes the bootstrap, so that the same data gives the same report.
const Seed = 20261027

// Bootstraps is the number of bootstrap resamples.
const Bootstraps = 10000

// TaskMeans are the means of one task in one arm.
type TaskMeans struct {
	Task, Kind, Arm string
	Runs            int
	Clean           float64            // M1
	CleanNonAD      float64            // M1 with only the targeted rules that are not analyzer-decidable
	Success         float64            // M3
	Violations      map[string]float64 // M2: broken rules per run, by class
	BreaksD         float64            // share of runs that break a class D rule
	Tokens          float64            // M4
	Wall            float64
	Turns           float64
	Checks          float64 // M5
	FailsBeforePass float64
	VulcanShare     float64 // share of failed checks with a Vulcan rule code (arm V)
	Recovered       float64 // M6: share of Vulcan codes gone within the next two checks
	Lines, Files    float64 // M7
	Units           float64
	HoldoutFails    float64 // M8
	Bugs            int     // M10
}

// Diff is a paired comparison over tasks.
type Diff struct {
	N      int
	Mean   float64
	Lo, Hi float64 // bootstrap 95% interval
	P      float64 // exact two-sided sign-flip p-value
}

// Report is the outcome of the analysis.
type Report struct {
	Means         []TaskMeans
	Clean         map[string]Diff // "all", "tempting", "ordinary"
	Success       map[string]Diff
	Violations    map[string]Diff // by class
	BreaksD       Diff
	CleanNonAD    Diff // tempting tasks
	CostRatio     float64
	WallRatio     float64
	TurnsRatio    float64
	Verdict       string
	Reasons       []string
	Bugs          int
	Excluded      int
	VerdictNoBugs string
	Tasks         int
}

type taskInfo struct {
	kind    string
	targets []string
}

// Analyze computes the report.
func Analyze(runs []record.Run, meta *record.Meta) *Report {
	rep := analyze(runs, meta, false)
	rep.VerdictNoBugs = analyze(runs, meta, true).Verdict
	return rep
}

func analyze(runs []record.Run, meta *record.Meta, dropBugs bool) *Report {
	tasks := map[string]taskInfo{}
	for _, t := range meta.Tasks {
		if !t.Pilot {
			tasks[t.ID] = taskInfo{t.Kind, t.Targets}
		}
	}
	rules := map[string]record.Rule{}
	for _, r := range meta.Rules {
		rules[r.ID] = r
	}
	rep := &Report{Clean: map[string]Diff{}, Success: map[string]Diff{}, Violations: map[string]Diff{}}
	groups := map[[2]string][]record.Run{}
	for _, r := range runs {
		if _, ok := tasks[r.Task]; !ok {
			continue // a pilot task, or one not in the metadata
		}
		if r.Excluded {
			rep.Excluded++
			continue
		}
		if r.ToolchainBug {
			rep.Bugs++
			if dropBugs {
				continue
			}
		}
		groups[[2]string{r.Task, r.Arm}] = append(groups[[2]string{r.Task, r.Arm}], r)
	}
	byTask := map[string]map[string]*TaskMeans{}
	var ids []string
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, arm := range []string{record.ArmV, record.ArmG} {
			rs := groups[[2]string{id, arm}]
			if len(rs) == 0 {
				continue
			}
			m := means(id, tasks[id], arm, rs, rules)
			rep.Means = append(rep.Means, *m)
			if byTask[id] == nil {
				byTask[id] = map[string]*TaskMeans{}
			}
			byTask[id][arm] = m
		}
	}
	paired := func(kind string, f func(*TaskMeans) float64) Diff {
		var d []float64
		for _, id := range ids {
			v, g := byTask[id][record.ArmV], byTask[id][record.ArmG]
			if v == nil || g == nil || (kind != "all" && tasks[id].kind != kind) {
				continue
			}
			d = append(d, f(v)-f(g))
		}
		return compare(d)
	}
	for _, kind := range []string{"all", "tempting", "ordinary"} {
		rep.Clean[kind] = paired(kind, func(m *TaskMeans) float64 { return m.Clean })
		rep.Success[kind] = paired(kind, func(m *TaskMeans) float64 { return m.Success })
	}
	for _, class := range []string{"A", "B", "C", "D"} {
		rep.Violations[class] = paired("all", func(m *TaskMeans) float64 { return m.Violations[class] })
	}
	rep.BreaksD = paired("all", func(m *TaskMeans) float64 { return m.BreaksD })
	rep.CleanNonAD = paired("tempting", func(m *TaskMeans) float64 { return m.CleanNonAD })
	ratio := func(f func(*TaskMeans) float64) float64 {
		var logs []float64
		for _, id := range ids {
			v, g := byTask[id][record.ArmV], byTask[id][record.ArmG]
			if v == nil || g == nil || f(v) <= 0 || f(g) <= 0 {
				continue
			}
			logs = append(logs, math.Log(f(v)/f(g)))
		}
		if len(logs) == 0 {
			return math.NaN()
		}
		return math.Exp(mean(logs))
	}
	rep.CostRatio = ratio(func(m *TaskMeans) float64 { return m.Tokens })
	rep.WallRatio = ratio(func(m *TaskMeans) float64 { return m.Wall })
	rep.TurnsRatio = ratio(func(m *TaskMeans) float64 { return m.Turns })
	rep.Tasks = len(byTask)
	hasD := false
	for _, r := range rules {
		hasD = hasD || r.Class == "D"
	}
	rep.Verdict, rep.Reasons = verdict(rep, hasD)
	return rep
}

func means(id string, ti taskInfo, arm string, rs []record.Run, rules map[string]record.Rule) *TaskMeans {
	m := &TaskMeans{Task: id, Kind: ti.kind, Arm: arm, Runs: len(rs), Violations: map[string]float64{}}
	var vulcanFails, fails, codes, recovered float64
	for _, r := range rs {
		r.Targeted = ti.targets
		ok := !r.ToolchainBug
		if ok && r.CleanSuccess() {
			m.Clean++
		}
		if ok && r.TaskSuccess() {
			m.Success++
			clean := true
			for _, id := range ti.targets {
				if r.Violations[id] && !rules[id].AnalyzerDecidable {
					clean = false
				}
			}
			if clean {
				m.CleanNonAD++
			}
		}
		breaksD := false
		for id, broken := range r.Violations {
			if broken {
				m.Violations[rules[id].Class]++
				breaksD = breaksD || rules[id].Class == "D"
			}
		}
		if breaksD {
			m.BreaksD++
		}
		m.Tokens += float64(r.TokensIn + r.TokensOut)
		m.Wall += r.WallSeconds
		m.Turns += float64(r.Turns)
		m.Checks += float64(len(r.Checks))
		for i, c := range r.Checks {
			if c.OK {
				m.FailsBeforePass += float64(i)
				break
			}
			if i == len(r.Checks)-1 {
				m.FailsBeforePass += float64(len(r.Checks))
			}
		}
		for i, c := range r.Checks {
			if c.OK {
				continue
			}
			fails++
			vulcan := false
			for code := range c.Codes {
				if goCodes[code] {
					continue
				}
				vulcan = true
				codes++
				if gone(r.Checks, i, code) {
					recovered++
				}
			}
			if vulcan {
				vulcanFails++
			}
		}
		m.Lines += float64(r.LinesChanged)
		m.Files += float64(r.FilesChanged)
		m.Units += float64(r.UnitsTouched)
		m.HoldoutFails += float64(r.Holdout.Failed)
		if r.ToolchainBug {
			m.Bugs++
		}
	}
	n := float64(len(rs))
	for _, p := range []*float64{&m.Clean, &m.CleanNonAD, &m.Success, &m.BreaksD, &m.Tokens, &m.Wall, &m.Turns, &m.Checks, &m.FailsBeforePass, &m.Lines, &m.Files, &m.Units, &m.HoldoutFails} {
		*p /= n
	}
	for k := range m.Violations {
		m.Violations[k] /= n
	}
	m.VulcanShare, m.Recovered = math.NaN(), math.NaN()
	if fails > 0 {
		m.VulcanShare = vulcanFails / fails
	}
	if codes > 0 {
		m.Recovered = recovered / codes
	}
	return m
}

// gone reports whether code is absent from one of the next two checks (M6).
func gone(checks []record.Check, i int, code string) bool {
	for j := i + 1; j < len(checks) && j <= i+2; j++ {
		if checks[j].Codes[code] == 0 {
			return true
		}
	}
	return false
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// compare gives the mean of the paired differences, a percentile bootstrap 95% interval, and an
// exact two-sided sign-flip p-value (Monte Carlo past 20 tasks).
func compare(d []float64) Diff {
	out := Diff{N: len(d), Mean: mean(d), Lo: math.NaN(), Hi: math.NaN(), P: math.NaN()}
	if len(d) == 0 {
		return out
	}
	rng := rand.New(rand.NewPCG(Seed, uint64(len(d))))
	boots := make([]float64, Bootstraps)
	for b := range boots {
		s := 0.0
		for range d {
			s += d[rng.IntN(len(d))]
		}
		boots[b] = s / float64(len(d))
	}
	sort.Float64s(boots)
	out.Lo, out.Hi = boots[Bootstraps*25/1000], boots[Bootstraps*975/1000-1]
	obs := math.Abs(out.Mean)
	const eps = 1e-12
	extreme, total := 0, 0
	if len(d) <= 20 {
		for mask := 0; mask < 1<<len(d); mask++ {
			s := 0.0
			for i, x := range d {
				if mask&(1<<i) != 0 {
					s -= x
				} else {
					s += x
				}
			}
			if math.Abs(s/float64(len(d))) >= obs-eps {
				extreme++
			}
			total++
		}
	} else {
		for range 100000 {
			s := 0.0
			for _, x := range d {
				if rng.IntN(2) == 0 {
					s -= x
				} else {
					s += x
				}
			}
			if math.Abs(s/float64(len(d))) >= obs-eps {
				extreme++
			}
			total++
		}
	}
	out.P = float64(extreme) / float64(total)
	return out
}

func verdict(rep *Report, hasD bool) (string, []string) {
	gain := 100 * rep.Clean["tempting"].Mean
	ord := 100 * rep.Success["ordinary"].Mean
	gainNonAD := 100 * rep.CleanNonAD.Mean
	dDiff := 100 * rep.BreaksD.Mean
	ordOK := !math.IsNaN(ord) && ord >= -10
	costOK := !math.IsNaN(rep.CostRatio) && rep.CostRatio <= 1.3
	dOK := !hasD || (!math.IsNaN(dDiff) && math.Abs(dDiff) <= 10)
	reasons := []string{
		fmt.Sprintf("tempting gain in clean success: %+.1f points (Promising needs +20, No signal is below +10)", gain),
		fmt.Sprintf("ordinary tasks, task success V minus G: %+.1f points (the condition is at least -10): %v", ord, ordOK),
		fmt.Sprintf("cost ratio V/G (tokens, geometric mean): %.2f (the condition is at most 1.30): %v", rep.CostRatio, costOK),
		fmt.Sprintf("class D, runs that break a rule, V minus G: %+.1f points (about equal within 10): %v", dDiff, dOK),
		fmt.Sprintf("tempting gain with only the rules that an analyzer cannot decide: %+.1f points", gainNonAD),
	}
	switch {
	case math.IsNaN(gain):
		return "Unclear", append(reasons, "no tempting task has runs in both arms")
	case gain < 10:
		return "No signal", reasons
	case gain >= 20 && ordOK && costOK && dOK:
		if gainNonAD < 10 {
			return "Promising, but a tool may do it", reasons
		}
		return "Promising", reasons
	case !ordOK || !costOK:
		return "Costly", reasons
	}
	return "Unclear", reasons
}
