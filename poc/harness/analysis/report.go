package analysis

import (
	"fmt"
	"io"
	"math"
	"strings"
)

func pts(x float64) string {
	if math.IsNaN(x) {
		return "n/a"
	}
	return fmt.Sprintf("%+.1f", 100*x)
}

func num(x float64) string {
	if math.IsNaN(x) {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", x)
}

func diffRow(name string, d Diff, scale bool) string {
	f := num
	if scale {
		f = pts
	}
	return fmt.Sprintf("| %s | %d | %s | %s to %s | %s |", name, d.N, f(d.Mean), f(d.Lo), f(d.Hi), num(d.P))
}

// Markdown writes the report.
func (r *Report) Markdown(w io.Writer, title string) {
	fmt.Fprintf(w, "# %s\n\n", title)
	fmt.Fprintf(w, "**Verdict: %s.** Without the %d runs with a toolchain bug: %s. Tasks with runs in both arms: %d. Excluded runs: %d.\n\n", r.Verdict, r.Bugs, r.VerdictNoBugs, r.Tasks, r.Excluded)
	for _, s := range r.Reasons {
		fmt.Fprintf(w, "- %s\n", s)
	}
	fmt.Fprintf(w, "\nThe reading rules are in `poc/harness/analysis/analysis.go`, fixed before the runs (docs/poc/plan.md, section 9). Differences are V minus G, paired by task; the interval is a bootstrap 95%% interval over tasks; p is an exact two-sided sign-flip p-value.\n\n")

	fmt.Fprintf(w, "## M1 and M3: clean success and task success (points)\n\n| Measure | Tasks | V - G | 95%% interval | p |\n|---|---|---|---|---|\n")
	for _, k := range []string{"all", "tempting", "ordinary"} {
		fmt.Fprintln(w, diffRow("Clean success, "+k, r.Clean[k], true))
	}
	fmt.Fprintln(w, diffRow("Clean success, tempting, rules an analyzer cannot decide", r.CleanNonAD, true))
	for _, k := range []string{"all", "tempting", "ordinary"} {
		fmt.Fprintln(w, diffRow("Task success, "+k, r.Success[k], true))
	}

	fmt.Fprintf(w, "\n## M2: broken rules per run, by class\n\n| Class | Tasks | V - G | 95%% interval | p |\n|---|---|---|---|---|\n")
	for _, c := range []string{"A", "B", "C", "D"} {
		fmt.Fprintln(w, diffRow(c, r.Violations[c], false))
	}
	fmt.Fprintln(w, diffRow("D, runs that break a rule (points)", r.BreaksD, true))

	fmt.Fprintf(w, "\n## M4: cost (geometric mean over tasks of V / G)\n\n| Tokens | Wall time | Turns |\n|---|---|---|\n| %s | %s | %s |\n", num(r.CostRatio), num(r.WallRatio), num(r.TurnsRatio))

	fmt.Fprintf(w, "\n## Per task and arm\n\n| Task | Kind | Arm | Runs | Clean | Success | Tokens | Wall s | Turns | Checks | Fails before pass | Vulcan share of failed checks (M5) | Recovered (M6) | Lines | Files | Units (M7) | Holdout fails (M8) | Bugs (M10) |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for _, m := range r.Means {
		fmt.Fprintf(w, "| %s | %s | %s | %d | %.2f | %.2f | %.0f | %.0f | %.1f | %.1f | %.1f | %s | %s | %.0f | %.1f | %.1f | %.1f | %d |\n",
			m.Task, m.Kind, m.Arm, m.Runs, m.Clean, m.Success, m.Tokens, m.Wall, m.Turns, m.Checks, m.FailsBeforePass,
			num(m.VulcanShare), num(m.Recovered), m.Lines, m.Files, m.Units, m.HoldoutFails, m.Bugs)
	}
	fmt.Fprintf(w, "\n%s\n", strings.TrimSpace(`M9 (false positives) comes from the judge session's file, not from the runs; the report of the
evaluation adds it.`))
}
