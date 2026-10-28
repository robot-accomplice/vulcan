// Package analysis2 is the analysis script of round 2 (docs/poc/plan-r2.md, sections 1 and 3),
// frozen before the runs. Round 1's script, package analysis, stays as it was, so that round 1
// can be analyzed again.
//
// The unit is the task. Each measurement is averaged over the runs of a task and an arm; arms are
// compared by the paired per-task difference. The statistics are those of decision 6, section 8:
//
//   - A violation test of "X has fewer violations than Y" on a rule set uses the tempting tasks
//     that target a rule of the set. A task's rate is the mean number of the set's rules broken
//     per run. A task is informative when its two rates differ. The test is a one-sided exact
//     sign-flip test of the mean of (Y − X) over the informative tasks, at α = 0.05. It needs at
//     least 5 informative tasks, 4 rules in the set, and 10 violations in the two arms together;
//     otherwise it is inconclusive, and inconclusive counts as fail.
//   - P1(X better than Y) passes when the violation test passes and X's mean rate is at most half
//     of Y's, over the tasks of the test.
//   - A bound is a one-sided 95% t-bound over the tasks: the mean plus (or minus) t(0.95, k−1)
//     times the standard deviation over √k.
//
// The questions and the verdicts are those of plan-r2 section 3:
//
//   - QA: P1(L better than P) on set S.
//   - QV: V against L on clean success, tempting tasks that target S. V is "better" when the mean
//     of (V − L) is above 12 points and its lower bound above 0, or when the upper bound of the
//     reviewer's tokens, as the per-task log ratio V/L, is below log 0.87. V is "equivalent" when
//     the two bounds of (V − L) are within ±12 points.
//   - QR: review load (lines changed, reviewer tokens) as per-task log ratios L/P and V/L, with
//     upper bounds. Reported.
//   - QC: decision 6's N0 (the upper bound of P − L clean success at most 12 points), N1 (the
//     upper bound of the per-task log ratio L/P of tokens, and of turns, at most log 1.15; tokens
//     and turns stand for working time), N2 (the upper bound of L − P holdout failures per accepted
//     change at most 0.1, over the tasks where both have one), and N6 (on ordinary tasks, P − L
//     task success at most 12 points and the geometric mean ratio L/P of tokens at most 1.15, as
//     point estimates). The same for V against L, reported.
//   - The validity gate: P breaks a targeted rule in at least 30% of its runs on tempting tasks,
//     P succeeds in at least 70% of its runs on ordinary tasks, and no arm has more than 10% of
//     its runs blocked by a toolchain bug. The success condition reads P only: it asks whether
//     the setup is too hard, and an arm that is worse than P is a result, not a broken setup
//     (the dry run of F10 showed that a condition on every arm hides that result).
//
// Clean success is record.Run.CleanSuccess: the acceptance tests pass, no targeted rule is
// broken, and the check configuration is unchanged (or the task allows it). The reviewer's
// verdict is a measure (O3b, O4), not part of success: it is a model's judgment.
package analysis2

import (
	"math"
	"math/rand/v2"
	"sort"
	"strconv"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
	"github.com/robot-accomplice/vulcan/poc/harness/schedule"
)

// Seed fixes the Monte Carlo sign-flip tests, so that the same data gives the same report.
const Seed = 20261028

// Margins of decision 6, section 8.
const (
	MarginPoints = 0.12 // N0, N6, and QV's equivalence
	MarginTime   = 1.15 // N1, N6
	MarginDefect = 0.1  // N2
	ReviewGain   = 0.87 // QV: a review-load ratio bound below this is "better"
	Alpha        = 0.05
	MinBreakP    = 0.30 // the validity gate
	MinOrdinary  = 0.70
	MaxBugShare  = 0.10
)

// Arms of round 2.
var Arms = []string{record.ArmP, record.ArmL, record.ArmV}

// Means are the means of one task in one arm.
type Means struct {
	Task, Kind, Arm string
	Runs            int
	TargetsS        bool    // the task targets a rule of S
	Clean           float64 // M1
	Success         float64 // M3
	SRate           float64 // rules of S broken per run (all S rules, on tempting tasks that target S)
	NRate           float64 // rules of N broken per run (the control, O5)
	BreakTarget     float64 // share of runs that break a targeted rule
	SViolations     float64 // the sum, for the violation test's minimum
	Tokens, Turns   float64 // M4
	Lines, Files    float64 // M7
	Units           float64
	ReviewTokens    float64 // the loop's reviewer tokens per run (O3b)
	Rejections      float64 // O4
	Accepted        float64 // share of runs whose last review accepted
	HoldoutPerAcc   float64 // holdout failures per accepted change (N2, X6); NaN without one
	Clones, Masked  float64 // O7
	Undefined       float64 // O7: "undefined:" errors in check runs
	RuleFails       float64 // O2: failed check runs with a rule code
	TokensToPass    float64 // O8: input tokens to the first passing check after the first edit; NaN if none
	Bugs            int
}

// Test is a violation test.
type Test struct {
	Name        string
	Tasks       int     // informative tasks
	Rules       int     // rules in the set
	Violations  float64 // in the two arms together
	MeanX       float64 // the better arm's mean rate
	MeanY       float64
	P           float64
	Conclusive  bool
	Passes      bool // the test passes
	HalfOrLess  bool // MeanX ≤ MeanY / 2
	P1          bool // Passes and HalfOrLess
	Description string
}

// Bound is a paired difference (or log ratio) over tasks with its one-sided 95% bounds.
type Bound struct {
	N            int
	Mean, Lo, Hi float64
	Ratio        bool // Mean, Lo, Hi are exp of the log ratio
	Holds        bool // the condition of the row
	Condition    string
}

// Report is the outcome of the analysis.
type Report struct {
	Means    []Means
	Tasks    int
	Rules    map[string]int // rules by set
	Excluded int
	Bugs     map[string]int

	// The validity gate.
	PBreakTempting float64
	OrdSuccess     map[string]float64
	BugShare       map[string]float64
	Valid          bool

	QA      Test
	QV      Bound // V − L clean success
	QVRev   Bound // V/L reviewer tokens
	QVState string
	QR      map[string]Bound // "lines L/P", "lines V/L", "review L/P", "review V/L"
	NL      map[string]Bound // N0, N1 tokens, N1 turns, N2, N6 success, N6 tokens: L against P
	NV      map[string]Bound // the same, V against L
	Control map[string]Bound // O5: N-rule breaks and holdout failures, L − P and V − L
	Report  map[string]Bound // O7, O8: clones, masked, undefined, tokens to pass

	// Repeatability (plan-r2, section 5): how often the two runs of a retest slot agree on clean
	// success, and how often the reviewer's second review of a final change agrees with the
	// loop's last verdict.
	RetestPairs, RetestAgree int
	ReviewPairs, ReviewAgree int

	Verdict string
	Row     int
	Reasons []string
}

type taskInfo struct {
	kind    string
	targets []string
	s       bool
}

// Analyze computes the report.
func Analyze(runs []record.Run, meta *record.Meta) *Report {
	rules := map[string]record.Rule{}
	rep := &Report{Rules: map[string]int{}, Bugs: map[string]int{}, OrdSuccess: map[string]float64{}, BugShare: map[string]float64{},
		QR: map[string]Bound{}, NL: map[string]Bound{}, NV: map[string]Bound{}, Control: map[string]Bound{}, Report: map[string]Bound{}}
	for _, r := range meta.Rules {
		rules[r.ID] = r
		rep.Rules[r.Set()]++
	}
	tasks := map[string]taskInfo{}
	for _, t := range meta.Tasks {
		if t.Pilot {
			continue
		}
		ti := taskInfo{kind: t.Kind, targets: t.Targets}
		for _, id := range t.Targets {
			ti.s = ti.s || rules[id].Set() == record.SetS
		}
		tasks[t.ID] = ti
	}
	groups := map[[2]string][]record.Run{}
	armRuns := map[string]int{}
	retests := map[[3]string]record.Run{}
	var main []record.Run
	for _, r := range runs {
		if _, ok := tasks[r.Task]; ok && !r.Excluded && r.Index > schedule.RetestOffset {
			retests[[3]string{r.Task, r.Arm, strconv.Itoa(r.Index - schedule.RetestOffset)}] = r
			continue
		}
		main = append(main, r)
	}
	runs = main
	for _, r := range runs {
		if _, ok := tasks[r.Task]; !ok {
			continue
		}
		r.Targeted = tasks[r.Task].targets
		if t, ok := retests[[3]string{r.Task, r.Arm, strconv.Itoa(r.Index)}]; ok && !r.Excluded {
			t.Targeted = r.Targeted
			rep.RetestPairs++
			if t.CleanSuccess() == r.CleanSuccess() {
				rep.RetestAgree++
			}
		}
		var loop, second *record.Review
		for i := range r.Reviews {
			if v := &r.Reviews[i]; v.Valid && v.Round > 0 {
				loop = v
			} else if v.Valid && v.Round == 0 {
				second = v
			}
		}
		if loop != nil && second != nil {
			rep.ReviewPairs++
			if loop.Accept == second.Accept {
				rep.ReviewAgree++
			}
		}
		if r.Excluded {
			rep.Excluded++
			continue
		}
		armRuns[r.Arm]++
		if r.ToolchainBug {
			rep.Bugs[r.Arm]++
		}
		groups[[2]string{r.Task, r.Arm}] = append(groups[[2]string{r.Task, r.Arm}], r)
	}
	var ids []string
	for id := range tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	by := map[string]map[string]*Means{}
	for _, id := range ids {
		for _, arm := range Arms {
			rs := groups[[2]string{id, arm}]
			if len(rs) == 0 {
				continue
			}
			m := means(id, tasks[id], arm, rs, rules)
			rep.Means = append(rep.Means, *m)
			if by[id] == nil {
				by[id] = map[string]*Means{}
			}
			by[id][arm] = m
		}
	}
	rep.Tasks = len(by)

	// The validity gate, pooled over runs.
	var pBreak, pTempt float64
	ordOK, ordN := map[string]float64{}, map[string]float64{}
	for _, r := range runs {
		ti, ok := tasks[r.Task]
		if !ok || r.Excluded {
			continue
		}
		if ti.kind == "tempting" && r.Arm == record.ArmP {
			pTempt++
			for _, id := range ti.targets {
				if r.Violations[id] {
					pBreak++
					break
				}
			}
		}
		if ti.kind == "ordinary" {
			ordN[r.Arm]++
			r.Targeted = ti.targets
			if r.TaskSuccess() && !r.ToolchainBug {
				ordOK[r.Arm]++
			}
		}
	}
	rep.PBreakTempting = ratioOrNaN(pBreak, pTempt)
	rep.Valid = rep.PBreakTempting >= MinBreakP && ratioOrNaN(ordOK[record.ArmP], ordN[record.ArmP]) >= MinOrdinary
	for _, arm := range Arms {
		rep.OrdSuccess[arm] = ratioOrNaN(ordOK[arm], ordN[arm])
		rep.BugShare[arm] = ratioOrNaN(float64(rep.Bugs[arm]), float64(armRuns[arm]))
		if rep.BugShare[arm] > MaxBugShare {
			rep.Valid = false
		}
	}

	pairs := func(x, y string, pick func(taskInfo) bool, f func(*Means) float64) []float64 {
		var d []float64
		for _, id := range ids {
			a, b := by[id][x], by[id][y]
			if a == nil || b == nil || !pick(tasks[id]) {
				continue
			}
			va, vb := f(a), f(b)
			if math.IsNaN(va) || math.IsNaN(vb) {
				continue
			}
			d = append(d, va-vb)
		}
		return d
	}
	logPairs := func(x, y string, pick func(taskInfo) bool, f func(*Means) float64) []float64 {
		var d []float64
		for _, id := range ids {
			a, b := by[id][x], by[id][y]
			if a == nil || b == nil || !pick(tasks[id]) || !(f(a) > 0) || !(f(b) > 0) {
				continue
			}
			d = append(d, math.Log(f(a)/f(b)))
		}
		return d
	}
	all := func(taskInfo) bool { return true }
	temptS := func(t taskInfo) bool { return t.kind == "tempting" && t.s }
	ordinary := func(t taskInfo) bool { return t.kind == "ordinary" }

	// QA: P1(L better than P) on S.
	rep.QA = violationTest("L fewer S violations than P", ids, by, tasks, record.ArmL, record.ArmP, rep.Rules[record.SetS])

	// QV.
	rep.QV = bound(pairs(record.ArmV, record.ArmL, temptS, func(m *Means) float64 { return m.Clean }), false)
	rep.QVRev = bound(logPairs(record.ArmV, record.ArmL, temptS, func(m *Means) float64 { return m.ReviewTokens }), true)
	better := (rep.QV.Mean > MarginPoints && rep.QV.Lo > 0) || (rep.QVRev.N > 0 && rep.QVRev.Hi < ReviewGain)
	equivalent := rep.QV.N > 0 && rep.QV.Lo >= -MarginPoints && rep.QV.Hi <= MarginPoints
	switch {
	case better:
		rep.QVState = "better"
	case equivalent:
		rep.QVState = "equivalent"
	case rep.QV.N > 0 && rep.QV.Hi < 0:
		rep.QVState = "worse"
	default:
		rep.QVState = "unclear"
	}

	// QR.
	lines := func(m *Means) float64 { return m.Lines }
	review := func(m *Means) float64 { return m.ReviewTokens }
	rep.QR["lines L/P"] = bound(logPairs(record.ArmL, record.ArmP, all, lines), true)
	rep.QR["lines V/L"] = bound(logPairs(record.ArmV, record.ArmL, all, lines), true)
	rep.QR["review L/P"] = bound(logPairs(record.ArmL, record.ArmP, all, review), true)
	rep.QR["review V/L"] = bound(logPairs(record.ArmV, record.ArmL, all, review), true)

	// QC, for L against P (tested) and V against L (reported).
	costs := func(x, y string, out map[string]Bound) bool {
		n0 := bound(pairs(y, x, all, func(m *Means) float64 { return m.Clean }), false)
		n0.Holds, n0.Condition = n0.N > 0 && n0.Hi <= MarginPoints, "upper bound of the clean-success loss at most 12 points"
		tok := bound(logPairs(x, y, all, func(m *Means) float64 { return m.Tokens }), true)
		tok.Holds, tok.Condition = tok.N > 0 && tok.Hi <= MarginTime, "upper bound of the token ratio at most 1.15"
		turns := bound(logPairs(x, y, all, func(m *Means) float64 { return m.Turns }), true)
		turns.Holds, turns.Condition = turns.N > 0 && turns.Hi <= MarginTime, "upper bound of the turn ratio at most 1.15"
		n2 := bound(pairs(x, y, all, func(m *Means) float64 { return m.HoldoutPerAcc }), false)
		n2.Holds, n2.Condition = n2.N == 0 || n2.Hi <= MarginDefect, "upper bound of extra holdout failures per accepted change at most 0.1"
		ord := bound(pairs(y, x, ordinary, func(m *Means) float64 { return m.Success }), false)
		ord.Holds, ord.Condition = ord.N > 0 && ord.Mean <= MarginPoints, "ordinary tasks: success loss at most 12 points (point estimate)"
		ordTok := bound(logPairs(x, y, ordinary, func(m *Means) float64 { return m.Tokens }), true)
		ordTok.Holds, ordTok.Condition = ordTok.N > 0 && ordTok.Mean <= MarginTime, "ordinary tasks: token ratio at most 1.15 (point estimate)"
		out["N0 clean success"], out["N1 tokens"], out["N1 turns"], out["N2 escaped defects"], out["N6 ordinary success"], out["N6 ordinary tokens"] = n0, tok, turns, n2, ord, ordTok
		return n0.Holds && tok.Holds && turns.Holds && ord.Holds && ordTok.Holds
	}
	lCostsOK := costs(record.ArmL, record.ArmP, rep.NL)
	costs(record.ArmV, record.ArmL, rep.NV)

	// O5: the controls.
	nrate := func(m *Means) float64 { return m.NRate }
	hold := func(m *Means) float64 { return m.HoldoutPerAcc }
	rep.Control["N-rule breaks L − P"] = bound(pairs(record.ArmL, record.ArmP, all, nrate), false)
	rep.Control["N-rule breaks V − L"] = bound(pairs(record.ArmV, record.ArmL, all, nrate), false)
	rep.Control["holdout per accepted L − P"] = bound(pairs(record.ArmL, record.ArmP, all, hold), false)
	rep.Control["holdout per accepted V − L"] = bound(pairs(record.ArmV, record.ArmL, all, hold), false)
	// O7, O8: reported.
	for name, f := range map[string]func(*Means) float64{
		"clone lines": func(m *Means) float64 { return m.Clones }, "masked errors": func(m *Means) float64 { return m.Masked },
		"undefined names": func(m *Means) float64 { return m.Undefined }} {
		rep.Report[name+" L − P"] = bound(pairs(record.ArmL, record.ArmP, all, f), false)
		rep.Report[name+" V − L"] = bound(pairs(record.ArmV, record.ArmL, all, f), false)
	}
	pass := func(m *Means) float64 { return m.TokensToPass }
	rep.Report["tokens to first pass L/P"] = bound(logPairs(record.ArmL, record.ArmP, all, pass), true)
	rep.Report["tokens to first pass V/L"] = bound(logPairs(record.ArmV, record.ArmL, all, pass), true)

	rep.Row, rep.Verdict, rep.Reasons = verdict(rep, lCostsOK)
	return rep
}

func ratioOrNaN(a, b float64) float64 {
	if b == 0 {
		return math.NaN()
	}
	return a / b
}

func means(id string, ti taskInfo, arm string, rs []record.Run, rules map[string]record.Rule) *Means {
	m := &Means{Task: id, Kind: ti.kind, Arm: arm, Runs: len(rs), TargetsS: ti.s}
	var accChanges, accHoldout, passN float64
	for _, r := range rs {
		r.Targeted = ti.targets
		ok := !r.ToolchainBug
		if ok && r.CleanSuccess() {
			m.Clean++
		}
		if ok && r.TaskSuccess() {
			m.Success++
			if r.Accepted() {
				accChanges++
				accHoldout += float64(r.Holdout.Failed)
			}
		}
		for _, id := range ti.targets {
			if r.Violations[id] {
				m.BreakTarget++
				break
			}
		}
		for id, broken := range r.Violations {
			if !broken {
				continue
			}
			switch rules[id].Set() {
			case record.SetS:
				m.SRate++
			case record.SetN:
				m.NRate++
			}
		}
		m.Tokens += float64(r.TokensIn + r.TokensOut)
		m.Turns += float64(r.Turns)
		m.Lines += float64(r.LinesChanged)
		m.Files += float64(r.FilesChanged)
		m.Units += float64(r.UnitsTouched)
		for _, v := range r.Reviews {
			if v.Round > 0 {
				m.ReviewTokens += float64(v.TokensIn + v.TokensOut)
			}
		}
		m.Rejections += float64(r.Rejections())
		if r.Accepted() {
			m.Accepted++
		}
		m.Clones += float64(r.CloneLines)
		m.Masked += float64(r.MaskedErrors)
		for _, c := range r.Checks {
			m.Undefined += float64(c.Undefined)
			if !c.OK && len(c.Codes) > 0 && (r.FirstPassTurn == 0 || c.Turn < r.FirstPassTurn) {
				m.RuleFails++
			}
		}
		if r.TokensInAtFirstPass > 0 {
			m.TokensToPass += float64(r.TokensInAtFirstPass)
			passN++
		}
		if r.ToolchainBug {
			m.Bugs++
		}
	}
	n := float64(len(rs))
	m.SViolations = m.SRate
	for _, p := range []*float64{&m.Clean, &m.Success, &m.SRate, &m.NRate, &m.BreakTarget, &m.Tokens, &m.Turns, &m.Lines, &m.Files,
		&m.Units, &m.ReviewTokens, &m.Rejections, &m.Accepted, &m.Clones, &m.Masked, &m.Undefined, &m.RuleFails} {
		*p /= n
	}
	m.HoldoutPerAcc = ratioOrNaN(accHoldout, accChanges)
	m.TokensToPass = ratioOrNaN(m.TokensToPass, passN)
	if ti.kind != "tempting" || !ti.s {
		m.SRate = math.NaN() // the violation test reads only tempting tasks that target S
	}
	return m
}

// violationTest is the test of "x has fewer violations of set S than y" (decision 6, section 8).
func violationTest(name string, ids []string, by map[string]map[string]*Means, tasks map[string]taskInfo, x, y string, rules int) Test {
	t := Test{Name: name, Rules: rules, Description: "one-sided exact sign-flip test over informative tempting tasks that target S"}
	var d, xs, ys []float64
	for _, id := range ids {
		a, b := by[id][x], by[id][y]
		if a == nil || b == nil || tasks[id].kind != "tempting" || !tasks[id].s {
			continue
		}
		xs, ys = append(xs, a.SRate), append(ys, b.SRate)
		t.Violations += a.SViolations*float64(a.Runs) + b.SViolations*float64(b.Runs)
		if a.SRate != b.SRate {
			d = append(d, b.SRate-a.SRate)
		}
	}
	t.Tasks = len(d)
	t.MeanX, t.MeanY = mean(xs), mean(ys)
	t.P = signFlipOneSided(d)
	t.Conclusive = t.Tasks >= 5 && t.Rules >= 4 && t.Violations >= 10
	t.Passes = t.Conclusive && t.P < Alpha
	t.HalfOrLess = t.MeanX <= t.MeanY/2
	t.P1 = t.Passes && t.HalfOrLess
	return t
}

// signFlipOneSided is the one-sided p-value of "the mean of d is above 0": the share of sign
// flips whose mean is at least the observed mean. Exact up to 20 values, Monte Carlo beyond.
func signFlipOneSided(d []float64) float64 {
	if len(d) == 0 {
		return math.NaN()
	}
	obs := mean(d)
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
			if s/float64(len(d)) >= obs-eps {
				extreme++
			}
			total++
		}
	} else {
		rng := rand.New(rand.NewPCG(Seed, uint64(len(d))))
		for range 200000 {
			s := 0.0
			for _, x := range d {
				if rng.IntN(2) == 0 {
					s -= x
				} else {
					s += x
				}
			}
			if s/float64(len(d)) >= obs-eps {
				extreme++
			}
			total++
		}
	}
	return float64(extreme) / float64(total)
}

// bound gives the mean and the one-sided 95% t-bounds of d; for a log ratio, as ratios.
func bound(d []float64, ratio bool) Bound {
	b := Bound{N: len(d), Mean: mean(d), Lo: math.NaN(), Hi: math.NaN(), Ratio: ratio}
	if len(d) >= 2 {
		sd := stddev(d)
		h := tQuantile95(len(d)-1) * sd / math.Sqrt(float64(len(d)))
		b.Lo, b.Hi = b.Mean-h, b.Mean+h
	}
	if ratio {
		b.Mean, b.Lo, b.Hi = math.Exp(b.Mean), math.Exp(b.Lo), math.Exp(b.Hi)
	}
	return b
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

func stddev(xs []float64) float64 {
	m := mean(xs)
	s := 0.0
	for _, x := range xs {
		s += (x - m) * (x - m)
	}
	return math.Sqrt(s / float64(len(xs)-1))
}

func verdict(rep *Report, lCostsOK bool) (int, string, []string) {
	reasons := []string{}
	add := func(format string, ok bool) { reasons = append(reasons, format+": "+yes(ok)) }
	add("validity gate (P breaks a targeted rule in ≥30% of tempting runs; P ≥70% ordinary success; ≤10% toolchain bugs in every arm)", rep.Valid)
	add("QA: P1(L better than P) on S", rep.QA.P1)
	add("L within the margins of P (N0, N1, N6)", lCostsOK)
	reasons = append(reasons, "QV: V against L: "+rep.QVState)
	switch {
	case !rep.Valid:
		return 1, "Not informative", reasons
	case rep.QA.P1 && lCostsOK && rep.QVState != "better":
		return 2, "The model helps; the language does not. Pivot to the rules, as a Go tool (decision 6, row 4)", reasons
	case rep.QA.P1 && rep.QVState == "better":
		return 3, "The language adds something the linter does not. Find the cause before any claim; test again with a fresh rule set", reasons
	case rep.QA.P1 && !lCostsOK:
		return 4, "The model helps, at a cost too high", reasons
	case !rep.QA.P1 && !(rep.QA.MeanX < rep.QA.MeanY):
		return 5, "The model does not help this agent. Neither the language nor the linter is supported", reasons
	}
	return 6, "Unclear: more runs on the same tasks, decided and sized before they run", reasons
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
