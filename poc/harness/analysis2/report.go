package analysis2

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
)

func f2(x float64) string {
	if math.IsNaN(x) {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", x)
}

func pts(x float64) string {
	if math.IsNaN(x) {
		return "n/a"
	}
	return fmt.Sprintf("%+.1f", 100*x)
}

func pct(x float64) string {
	if math.IsNaN(x) {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*x)
}

func pval(x float64) string {
	if math.IsNaN(x) {
		return "n/a"
	}
	return fmt.Sprintf("%.4f", x)
}

func boundRow(name string, b Bound, points bool) string {
	f := f2
	if points && !b.Ratio {
		f = pts
	}
	holds := ""
	if b.Condition != "" {
		holds = fmt.Sprintf("%s: %s", b.Condition, yes(b.Holds))
	}
	return fmt.Sprintf("| %s | %d | %s | %s | %s | %s |", name, b.N, f(b.Mean), f(b.Lo), f(b.Hi), holds)
}

func boundTable(w io.Writer, title string, m map[string]Bound, points bool) {
	fmt.Fprintf(w, "\n## %s\n\n| Measure | Tasks | Mean | Lower bound | Upper bound | Condition |\n|---|---|---|---|---|---|\n", title)
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintln(w, boundRow(k, m[k], points))
	}
}

// Markdown writes the report.
func (r *Report) Markdown(w io.Writer, title string) {
	fmt.Fprintf(w, "# %s\n\n**Verdict (row %d): %s.**\n\n", title, r.Row, r.Verdict)
	fmt.Fprintf(w, "A verdict holds only if the replication on a second model gives the same row (plan-r2, sections 3 and 5). Every verdict is read with all the effect sizes and intervals below.\n\n")
	for _, s := range r.Reasons {
		fmt.Fprintf(w, "- %s\n", s)
	}
	fmt.Fprintf(w, "\nTasks with runs: %d. Excluded runs: %d. Rules by set: %v. The reading rules are in `poc/harness/analysis2/analysis2.go`, frozen before the runs (docs/poc/plan-r2.md, section 3). Points are percentage points; bounds are one-sided 95%% t-bounds over tasks; ratios are geometric means of per-task ratios.\n", r.Tasks, r.Excluded, r.Rules)

	fmt.Fprintf(w, "\n## Validity gate\n\n| Condition | Value |\n|---|---|\n")
	fmt.Fprintf(w, "| P's runs on tempting tasks that break a targeted rule (at least 30%%) | %s |\n", pct(r.PBreakTempting))
	fmt.Fprintf(w, "| P's success on ordinary tasks (at least 70%%) | %s |\n", pct(r.OrdSuccess["P"]))
	for _, arm := range Arms {
		fmt.Fprintf(w, "| %s: runs with a toolchain bug (at most 10%%) | %s |\n", arm, pct(r.BugShare[arm]))
	}
	for _, arm := range Arms[1:] {
		fmt.Fprintf(w, "| %s's success on ordinary tasks (reported) | %s |\n", arm, pct(r.OrdSuccess[arm]))
	}

	t := r.QA
	fmt.Fprintf(w, "\n## QA: do enforced rules keep the agent within them? (O1)\n\n| Test | Informative tasks | Rules | Violations | Mean L | Mean P | p | Conclusive | Passes | L at most half of P | P1 |\n|---|---|---|---|---|---|---|---|---|---|---|\n")
	fmt.Fprintf(w, "| %s | %d | %d | %.0f | %s | %s | %s | %s | %s | %s | %s |\n", t.Name, t.Tasks, t.Rules, t.Violations, f2(t.MeanX), f2(t.MeanY), pval(t.P),
		yes(t.Conclusive), yes(t.Passes), yes(t.HalfOrLess), yes(t.P1))

	fmt.Fprintf(w, "\n## QV: does the language add to the linter? (O1 to O4)\n\nState: **%s**.\n\n| Measure | Tasks | Mean | Lower bound | Upper bound | Condition |\n|---|---|---|---|---|---|\n", r.QVState)
	fmt.Fprintln(w, boundRow("clean success, V − L, tempting tasks that target S", r.QV, true))
	fmt.Fprintln(w, boundRow("reviewer tokens, V / L", r.QVRev, false))

	boundTable(w, "QR: review load (O3)", r.QR, false)
	boundTable(w, "QC: cost, L against P (tested)", r.NL, true)
	boundTable(w, "QC: cost, V against L (reported)", r.NV, true)
	boundTable(w, "O5: controls (no arm should differ)", r.Control, false)
	boundTable(w, "O7, O8: reported, not credited to Vulcan (the PoC does not build these features)", r.Report, false)

	fmt.Fprintf(w, "\n## Repeatability\n\n| Measure | Pairs | Agree |\n|---|---|---|\n")
	fmt.Fprintf(w, "| Test-retest: clean success of the two runs of a slot | %d | %s |\n", r.RetestPairs, pct(ratioOrNaN(float64(r.RetestAgree), float64(r.RetestPairs))))
	fmt.Fprintf(w, "| Reviewer: the loop's last verdict and the second review of the final change | %d | %s |\n", r.ReviewPairs, pct(ratioOrNaN(float64(r.ReviewAgree), float64(r.ReviewPairs))))
	fmt.Fprintf(w, "\n## Per task and arm\n\n| Task | Kind | Arm | Runs | Clean | Success | S breaks | N breaks | Tokens | Turns | Lines | Review tokens | Rejections | Holdout per accepted | Clones | Masked | Undefined | Rule fails | Tokens to pass |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, m := range r.Means {
		fmt.Fprintf(w, "| %s | %s | %s | %d | %s | %s | %s | %s | %.0f | %.1f | %.0f | %.0f | %.2f | %s | %.1f | %.1f | %.1f | %.1f | %s |\n",
			m.Task, m.Kind, m.Arm, m.Runs, f2(m.Clean), f2(m.Success), f2(m.SRate), f2(m.NRate), m.Tokens, m.Turns, m.Lines, m.ReviewTokens,
			m.Rejections, f2(m.HoldoutPerAcc), m.Clones, m.Masked, m.Undefined, m.RuleFails, strings.TrimSuffix(f2(m.TokensToPass), ".00"))
	}
}
