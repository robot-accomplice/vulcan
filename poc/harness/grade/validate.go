package grade

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

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
	enforcedBy := func(arm, rule string) bool {
		enforced, cases := true, 0
		for _, it := range results {
			if it.Kind == "seeded" && it.Arm == arm && strings.HasPrefix(it.ID, rule+"/") {
				cases++
				enforced = enforced && it.Rejected
			}
		}
		return enforced && cases > 0
	}
	for _, r := range g.Meta.Rules {
		v.EnforcedByV[r.ID] = enforcedBy(record.ArmV, r.ID)
		if v.EnforcedByL != nil {
			v.EnforcedByL[r.ID] = enforcedBy(record.ArmL, r.ID)
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
		// exec looks a bare name up in this process's PATH, not in cmd.Env: name the tool.
		cmd := exec.Command(filepath.Join(g.ToolDir, "vulcan"+exe()), "check")
		if it.Arm == record.ArmL {
			cmd = exec.Command(filepath.Join(g.ToolDir, "vulcan-lint"+exe()), "./...")
		}
		cmd.Dir = ws
		cmd.Env = append(g.env(), "GOPROXY=off")
		out, err := cmd.CombinedOutput()
		it.Rejected = err != nil
		_ = os.WriteFile(filepath.Join(dir, "check.log"), out, 0o644)
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
			caught = true // the arm's check rejects it, so it never passes the check
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
	if v.EnforcedByL != nil {
		label(record.ArmL, v.EnforcedByL)
	}
	label(record.ArmV, v.EnforcedByV)
	if v.EnforcedByL != nil {
		sets := map[string][]string{}
		for id := range v.EnforcedByV {
			s := record.Rule{EnforcedByL: v.EnforcedByL[id], EnforcedByV: v.EnforcedByV[id]}.Set()
			sets[s] = append(sets[s], id)
		}
		for _, s := range []string{record.SetS, record.SetG, record.SetU, record.SetN} {
			sort.Strings(sets[s])
			fmt.Fprintf(&b, "set %s: %d (%s)\n", s, len(sets[s]), strings.Join(sets[s], ", "))
		}
	}
	for _, it := range v.Items {
		if !it.OK {
			fmt.Fprintf(&b, "PROBLEM %s %s %s: %s\n", it.Kind, it.ID, it.Arm, it.Problem)
		}
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
