package grade

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/robot-accomplice/vulcan/internal/rules"
	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Validation checks the hidden material before the runs (docs/poc/plan.md, section 6):
//
//   - The base application of each arm breaks no rule (no false positive).
//   - The reference solution of each task, in each arm, builds, passes the task's acceptance
//     tests and the holdout tests, and breaks no rule.
//   - Each seeded case of a rule, in each arm, is flagged by the rule's test or detector. A case
//     in arm V that the compiler rejects counts as caught for a test rule, since it can never
//     build. Other rules that also fire are listed.
//   - Each rule has at least two seeded shapes in each arm.
//   - Enforced by V: the compiler (vulcan check) rejects every seeded case of the rule in arm V.
//   - Enforced by L (round 2): the linter (vulcan-lint ./...) rejects every seeded case of the rule
//     in arm L. Arm P has no check beyond the build, so it gives no label.
//   - With a rules file beside a workspace (plan-r3, section 5): a workspace is rejected only by a
//     finding that its baseline does not hold, each item records what fired, and a label needs a
//     session's decision that a rule that fired matches the team rule (fired.go).
//
// A seeded case or a reference solution is an overlay: a directory whose files replace or add
// files of the arm's workspace; a file .delete in it lists workspace paths to remove, one a line.
//
//	seeded/<rule>/<shape>/<arm>/...     two or more shapes for each rule, in each arm
//	reference/<task>/<arm>/...
//
// The arms are those of the templates: G and V in round 1; P, L, and V in round 2.
type Validation struct {
	Items       []Item          `json:"items"`
	EnforcedByV map[string]bool `json:"enforced_by_v"`
	EnforcedByL map[string]bool `json:"enforced_by_l,omitempty"`
	ShapeCounts map[string]int  `json:"shape_counts"` // "<rule>/<arm>" -> shapes
	Problems    int             `json:"problems"`
	// NoLabels says why this run writes no enforced label: a workspace has a rules file, and a
	// session has not decided which rule that fired matches which team rule, or a base is rejected.
	NoLabels string `json:"no_labels,omitempty"`
	// With a rules file: EnforcedByV and EnforcedByL are as each arm ships, with a model check as a
	// rule that fired, and the gate of plan-r3, section 5 reads them. These read the rules file alone.
	WithRules       bool                `json:"with_rules,omitempty"`
	RulesOnlyByV    map[string]bool     `json:"rules_only_by_v,omitempty"`
	RulesOnlyByL    map[string]bool     `json:"rules_only_by_l,omitempty"`
	DecidedByOneArm map[string][]string `json:"decided_by_one_arm,omitempty"` // team rule -> codes that only one arm has
}

// Item is one validated workspace.
type Item struct {
	Kind     string   `json:"kind"` // "base", "reference", "seeded"
	ID       string   `json:"id"`   // task, or rule/shape
	Arm      string   `json:"arm"`
	OK       bool     `json:"ok"`
	Rejected bool     `json:"rejected_by_check,omitempty"` // by vulcan check in V, by vulcan-lint in L
	Problem  string   `json:"problem,omitempty"`
	Also     []string `json:"also_fired,omitempty"` // other rules that a seeded case breaks
	// With a rules file: the new findings, and whether a rule that fired matches the team rule.
	HasRules bool    `json:"has_rules,omitempty"`
	Fired    []Fired `json:"fired,omitempty"`
	Matched  bool    `json:"matched,omitempty"`
	// MatchedFile says that a rule of the rules file matched, and not only a model check.
	MatchedFile bool `json:"matched_file,omitempty"`
}

// Templates names the workspace of each arm.
type Templates map[string]string

// Arms lists the arms of the templates, in a fixed order.
func (t Templates) Arms() []string {
	var out []string
	for _, a := range []string{record.ArmP, record.ArmL, record.ArmG, record.ArmV} {
		if _, ok := t[a]; ok {
			out = append(out, a)
		}
	}
	return out
}

// checked tells whether an arm has a check whose rejection labels a rule.
func checked(arm string) bool { return arm == record.ArmV || arm == record.ArmL }

// Validate checks the material, with up to jobs workspaces at once, in dir.
func (g *Grader) Validate(tmpl Templates, dir string, jobs int) (*Validation, error) {
	if err := g.Prepare(filepath.Join(dir, "prepared")); err != nil {
		return nil, err
	}
	type work struct {
		item    Item
		overlay string
		task    string
		rule    string
	}
	var todo []work
	arms := tmpl.Arms()
	for _, arm := range arms {
		todo = append(todo, work{item: Item{Kind: "base", ID: "base", Arm: arm}})
		for _, t := range g.Meta.Tasks {
			todo = append(todo, work{item: Item{Kind: "reference", ID: t.ID, Arm: arm}, task: t.ID,
				overlay: filepath.Join(g.Material, "reference", t.ID, arm)})
		}
	}
	v := &Validation{EnforcedByV: map[string]bool{}, ShapeCounts: map[string]int{}}
	if _, ok := tmpl[record.ArmL]; ok {
		v.EnforcedByL = map[string]bool{}
	}
	for _, r := range g.Meta.Rules {
		shapes, _ := os.ReadDir(filepath.Join(g.Material, "seeded", r.ID))
		for _, sh := range shapes {
			if !sh.IsDir() {
				continue
			}
			for _, arm := range arms {
				ov := filepath.Join(g.Material, "seeded", r.ID, sh.Name(), arm)
				if _, err := os.Stat(ov); err != nil {
					continue
				}
				v.ShapeCounts[r.ID+"/"+arm]++
				todo = append(todo, work{item: Item{Kind: "seeded", ID: r.ID + "/" + sh.Name(), Arm: arm}, rule: r.ID, overlay: ov})
			}
		}
	}
	if len(g.Only) > 0 {
		var kept []work
		for _, w := range todo {
			key := w.item.Kind + "/" + w.item.ID
			for _, p := range g.Only {
				if strings.HasPrefix(key, p) {
					kept = append(kept, w)
					break
				}
			}
		}
		todo = kept
	}
	results := make([]Item, len(todo))
	var wg sync.WaitGroup
	sem := make(chan bool, max(1, jobs))
	for i, w := range todo {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- true
			defer func() { <-sem }()
			results[i] = g.validateOne(tmpl, filepath.Join(dir, fmt.Sprintf("%03d", i)), w.item, w.overlay, w.task, w.rule)
		}()
	}
	wg.Wait()
	v.Items = results
	byFile := false
	enforcedBy := func(arm, rule string) bool {
		enforced, cases := true, 0
		for _, it := range results {
			if it.Kind == "seeded" && it.Arm == arm && strings.HasPrefix(it.ID, rule+"/") {
				cases++
				if byFile {
					enforced = enforced && it.Rejected && it.MatchedFile
				} else if it.HasRules {
					enforced = enforced && it.Rejected && it.Matched
				} else {
					enforced = enforced && it.Rejected
				}
			}
		}
		return enforced && cases > 0
	}
	v.NoLabels = g.noLabels(results)
	for _, it := range results {
		v.WithRules = v.WithRules || it.HasRules
	}
	if v.WithRules {
		v.RulesOnlyByV, v.RulesOnlyByL = map[string]bool{}, map[string]bool{}
		v.DecidedByOneArm = g.decidedByOneArm(results)
	}
	for _, r := range g.Meta.Rules {
		if v.NoLabels == "" {
			v.EnforcedByV[r.ID] = enforcedBy(record.ArmV, r.ID)
			if v.EnforcedByL != nil {
				v.EnforcedByL[r.ID] = enforcedBy(record.ArmL, r.ID)
			}
			if v.WithRules {
				byFile = true
				v.RulesOnlyByV[r.ID] = enforcedBy(record.ArmV, r.ID)
				if v.EnforcedByL != nil {
					v.RulesOnlyByL[r.ID] = enforcedBy(record.ArmL, r.ID)
				}
				byFile = false
			}
		}
		for _, arm := range arms {
			if v.ShapeCounts[r.ID+"/"+arm] < 2 && len(g.Only) == 0 {
				v.Items = append(v.Items, Item{Kind: "shapes", ID: r.ID, Arm: arm, Problem: fmt.Sprintf("%d seeded shapes; at least 2 are needed", v.ShapeCounts[r.ID+"/"+arm])})
			}
		}
	}
	for _, it := range v.Items {
		if !it.OK {
			v.Problems++
		}
	}
	return v, nil
}

// noLabels says why a run with a rules file cannot write an enforced label, or "".
func (g *Grader) noLabels(items []Item) string {
	withRules := false
	var missing, rejected []string
	for _, it := range items {
		withRules = withRules || it.HasRules
		if it.Kind == "base" && it.HasRules && it.Rejected {
			rejected = append(rejected, it.Arm)
		}
		if it.Kind != "seeded" {
			continue
		}
		team, _, _ := strings.Cut(it.ID, "/")
		for _, f := range it.Fired {
			if _, ok := g.Match[team][f.Rule]; !ok && !slices.Contains(missing, team+" and "+f.Rule) {
				missing = append(missing, team+" and "+f.Rule)
			}
		}
	}
	switch {
	case !withRules:
		return ""
	case len(rejected) > 0:
		return "the check rejects the base of arm " + strings.Join(rejected, ", ") + ", so a rejection says nothing about a seeded case; write a baseline first"
	case g.Match == nil:
		return "a workspace has a rules file, and no match file was given: a label needs a session's decision on each rule that fired (plan-r3, section 5). Export the pairs with -export-judge, and give the decisions with -match"
	case len(missing) > 0:
		slices.Sort(missing)
		return "the match file has no decision on: " + strings.Join(missing, "; ")
	}
	return ""
}

// decidedByOneArm lists, for each team rule, each code that one arm alone has and that matches it. Such
// a code can put a rule in G or U by a form that only that arm has.
func (g *Grader) decidedByOneArm(items []Item) map[string][]string {
	out := map[string][]string{}
	for _, it := range items {
		if it.Kind != "seeded" {
			continue
		}
		team, _, _ := strings.Cut(it.ID, "/")
		for _, f := range it.Fired {
			if c, ok := modelChecks[f.Code]; ok && c.arm != "" && f.Rule == "model-"+f.Code && g.Match[team][f.Rule] && !slices.Contains(out[team], f.Rule) {
				out[team] = append(out[team], f.Rule)
			}
		}
	}
	return out
}

func (g *Grader) validateOne(tmpl Templates, dir string, it Item, overlay, task, rule string) Item {
	ws := filepath.Join(dir, "workspace")
	if err := os.RemoveAll(dir); err != nil {
		it.Problem = err.Error()
		return it
	}
	if err := copyTemplate(tmpl[it.Arm], ws); err != nil {
		it.Problem = err.Error()
		return it
	}
	if overlay != "" {
		if err := applyOverlay(overlay, ws); err != nil {
			it.Problem = "overlay: " + err.Error()
			return it
		}
	}
	if rules := filepath.Join(g.Material, "RULES.md"); fileExists(rules) {
		b, _ := os.ReadFile(rules)
		_ = writeFile(filepath.Join(ws, "RULES.md"), b)
	}
	if checked(it.Arm) {
		out, err := g.checkCommand(it.Arm, ws).CombinedOutput()
		it.Rejected = err != nil
		_ = os.WriteFile(filepath.Join(dir, "check.log"), out, 0o644)
		if fileExists(filepath.Join(ws, rules.FileName)) {
			it.HasRules = true
			if it.Rejected {
				it.Fired = parseFired(out, ws)
			}
			if rule != "" {
				for _, f := range it.Fired {
					it.Matched = it.Matched || g.Match[rule][f.Rule]
					it.MatchedFile = it.MatchedFile || (g.Match[rule][f.Rule] && !strings.HasPrefix(f.Rule, "model-"))
				}
			}
		}
	}
	r := &record.Run{Task: task, Arm: it.Arm}
	if err := g.Grade(dir, r); err != nil {
		it.Problem = "grade: " + err.Error()
		return it
	}
	var broken []string
	for id, b := range r.Violations {
		if b {
			broken = append(broken, id)
		}
	}
	sort.Strings(broken)
	switch it.Kind {
	case "base":
		it.OK = r.Builds && len(broken) == 0 && r.Holdout.OK() && !it.Rejected
		if !it.OK {
			it.Problem = fmt.Sprintf("builds %v, rules broken %v, holdout %+v", r.Builds, broken, r.Holdout)
		}
	case "reference":
		it.OK = r.Builds && r.Acceptance.OK() && r.Holdout.OK() && len(broken) == 0 && !it.Rejected
		if !fileExists(overlay) {
			it.OK, it.Problem = false, "no reference solution"
		} else if !it.OK {
			it.Problem = fmt.Sprintf("builds %v, acceptance %+v, holdout %+v, rules broken %v, rejected by the check %v", r.Builds, r.Acceptance, r.Holdout, broken, it.Rejected)
		}
	case "seeded":
		caught := r.Violations[rule]
		var method string
		for _, ru := range g.Meta.Rules {
			if ru.ID == rule {
				method = ru.Method
			}
		}
		if !caught && checked(it.Arm) && it.Rejected && method == "test" {
			// The arm's check rejects it, so it never passes the check. With a rules file, a match
			// must also say that the rule that fired is this one, once a session has decided.
			caught = !it.HasRules || g.Match == nil || it.Matched
		}
		it.OK = caught
		if !caught {
			it.Problem = fmt.Sprintf("the %s of %s does not flag it (builds %v)", method, rule, r.Builds)
		}
		for _, b := range broken {
			if b != rule {
				it.Also = append(it.Also, b)
			}
		}
	}
	if r.Note != "" {
		it.Problem = strings.TrimSpace(it.Problem + "; " + r.Note)
		it.OK = false
	}
	return it
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func copyTemplate(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() && (d.Name() == ".vulcan" || d.Name() == ".git") {
			return filepath.SkipDir
		}
		if d.IsDir() || strings.HasSuffix(d.Name(), ".db") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(dst, rel), b)
	})
}

func applyOverlay(overlay, ws string) error {
	if !fileExists(overlay) {
		return nil
	}
	err := filepath.WalkDir(overlay, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(overlay, p)
		if rel == ".delete" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(ws, rel), b)
	})
	if err != nil {
		return err
	}
	f, err := os.Open(filepath.Join(overlay, ".delete"))
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			if err := os.Remove(filepath.Join(ws, filepath.FromSlash(line))); err != nil {
				return err
			}
		}
	}
	return nil
}

// Summary writes one line for each problem, and the counts. It names rules and tasks by id
// only, so that the builder can read it without seeing the hidden material.
func (v *Validation) Summary() string {
	var b strings.Builder
	counts := map[string][2]int{}
	for _, it := range v.Items {
		c := counts[it.Kind]
		if it.OK {
			c[0]++
		} else {
			c[1]++
		}
		counts[it.Kind] = c
	}
	var kinds []string
	for k := range counts {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Fprintf(&b, "%s: %d ok, %d with a problem\n", k, counts[k][0], counts[k][1])
	}
	if v.NoLabels != "" {
		fmt.Fprintf(&b, "no enforced label: %s\n", v.NoLabels)
	}
	label := func(arm string, m map[string]bool) {
		var enforced []string
		for id, e := range m {
			if e {
				enforced = append(enforced, id)
			}
		}
		sort.Strings(enforced)
		fmt.Fprintf(&b, "enforced by %s: %d of %d rules (%s)\n", arm, len(enforced), len(m), strings.Join(enforced, ", "))
	}
	if v.EnforcedByL != nil && v.NoLabels == "" {
		label(record.ArmL, v.EnforcedByL)
	}
	if v.NoLabels == "" {
		label(record.ArmV, v.EnforcedByV)
	}
	printSets := func(prefix string, l, vm map[string]bool) {
		sets := map[string][]string{}
		for id := range vm {
			s := record.Rule{EnforcedByL: l[id], EnforcedByV: vm[id]}.Set()
			sets[s] = append(sets[s], id)
		}
		for _, s := range []string{record.SetS, record.SetG, record.SetU, record.SetN} {
			sort.Strings(sets[s])
			fmt.Fprintf(&b, "%sset %s: %d (%s)\n", prefix, s, len(sets[s]), strings.Join(sets[s], ", "))
		}
	}
	if v.EnforcedByL != nil && v.NoLabels == "" {
		if v.WithRules {
			b.WriteString("the sets as each arm ships, with a model check as a rule that fired; the gate reads these:\n")
		}
		printSets("", v.EnforcedByL, v.EnforcedByV)
		if v.WithRules {
			b.WriteString("the sets for the rules file alone:\n")
			printSets("rules file alone: ", v.RulesOnlyByL, v.RulesOnlyByV)
			var teams []string
			for team := range v.DecidedByOneArm {
				teams = append(teams, team)
			}
			sort.Strings(teams)
			for _, team := range teams {
				fmt.Fprintf(&b, "decided by a code of one arm: %s (%s)\n", team, strings.Join(v.DecidedByOneArm[team], ", "))
			}
		}
	}
	for _, it := range v.Items {
		if !it.OK {
			fmt.Fprintf(&b, "PROBLEM %s %s %s: %s\n", it.Kind, it.ID, it.Arm, it.Problem)
		}
	}
	for _, it := range v.Items {
		if !it.HasRules || len(it.Fired) == 0 {
			continue
		}
		var names []string
		for _, f := range it.Fired {
			names = append(names, f.Rule+" ("+f.Part+")")
		}
		status := "rejected"
		if it.Kind == "seeded" && v.NoLabels == "" && !it.Matched {
			status = "rejected only by an unrelated rule, so it does not count"
		}
		fmt.Fprintf(&b, "fired %s %s %s, %s: %s\n", it.Kind, it.ID, it.Arm, status, strings.Join(names, ", "))
	}
	for _, it := range v.Items {
		if len(it.Also) > 0 {
			fmt.Fprintf(&b, "note %s %s %s also breaks %s\n", it.Kind, it.ID, it.Arm, strings.Join(it.Also, ", "))
		}
	}
	return b.String()
}

// Save writes the validation as JSON.
func (v *Validation) Save(path string) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
