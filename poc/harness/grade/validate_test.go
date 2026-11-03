package grade

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

var (
	toolsOnce sync.Once
	toolDir   string
	toolErr   error
)

// tools builds the commands of the arms once.
func tools(t *testing.T) string {
	t.Helper()
	toolsOnce.Do(func() {
		toolDir, toolErr = os.MkdirTemp("", "harness-tools")
		if toolErr != nil {
			return
		}
		for name, pkg := range map[string]string{"vulcan": "cmd/vulcan", "vulcan-lint": "poc/golint/cmd/vulcan-lint"} {
			out, err := exec.Command("go", "build", "-o", filepath.Join(toolDir, name+exe()), "github.com/robot-accomplice/vulcan/"+pkg).CombinedOutput()
			if err != nil {
				toolErr = fmt.Errorf("%s: %v\n%s", name, err, out)
				return
			}
		}
	})
	if toolErr != nil {
		t.Fatal(toolErr)
	}
	return toolDir
}

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..")
}

// validate runs Validate on a copy of the demo material, with the given templates.
func validate(t *testing.T, material string, tmpl Templates, match Matches) *Validation {
	t.Helper()
	meta, err := record.LoadMeta(filepath.Join(material, "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Grader{Repo: repoRoot(), Material: material, ToolDir: tools(t), Meta: meta, Match: match}
	v, err := g.Validate(tmpl, t.TempDir(), 3)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func demoMaterial() string {
	return filepath.Join(repoRoot(), "poc", "harness", "testdata", "material")
}

// baseTemplates are the applications as they are.
func baseTemplates() Templates {
	app := filepath.Join(repoRoot(), "poc", "app")
	return Templates{record.ArmP: filepath.Join(app, "go"), record.ArmL: filepath.Join(app, "golint"), record.ArmV: filepath.Join(app, "vulcan")}
}

// shape gives what the test compares: each item with no scratch path, and the summary.
func shape(v *Validation) string {
	var lines []string
	for _, it := range v.Items {
		lines = append(lines, fmt.Sprintf("%s %s %s ok=%v rejected=%v also=%v", it.Kind, it.ID, it.Arm, it.OK, it.Rejected, it.Also))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n") + "\n--\n" + regexp.MustCompile(`[A-Za-z]:\\\S+|/tmp/\S+`).ReplaceAllString(v.Summary(), "PATH")
}

// TestValidateNoRulesFile holds round 1's and round 2's behavior: with no rules file, the items, the
// labels, and the summary are as they were before the rules file existed. The golden file was
// written at commit 73ad2eb, before the code that reads a rules file.
func TestValidateNoRulesFile(t *testing.T) {
	v := validate(t, demoMaterial(), baseTemplates(), nil)
	got := strings.ReplaceAll(shape(v), "\r\n", "\n")
	path := filepath.Join("testdata", "no-rules.golden")
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("with no rules file, the result changed:\n got:\n%s\nwant:\n%s", got, want)
	}
}

// withRules copies a template, and gives it the rules file of init and a baseline of its findings.
func withRules(t *testing.T, from, arm string, baseline bool) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "workspace")
	if err := copyTemplate(from, dir); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(tools(t), "vulcan"+exe())
	if arm == record.ArmL {
		tool = filepath.Join(tools(t), "vulcan-lint"+exe())
	}
	args := [][]string{{"init"}}
	if baseline {
		args = append(args, []string{"rules", "baseline"})
	}
	for _, a := range args {
		cmd := exec.Command(tool, a...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off", "GOPROXY=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", a, err, out)
		}
	}
	return dir
}

// demoWithShapes copies the demo material, and adds the audit's two harmless shapes of the behavior
// rule RT, and one shape that breaks an unrelated default.
func demoWithShapes(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := copyDir(demoMaterial(), dir); err != nil {
		t.Fatal(err)
	}
	harmless := func(shape, body string) {
		for arm, name := range map[string]string{record.ArmP: "extra.go", record.ArmL: "extra.go", record.ArmV: "extra.vulcan"} {
			if err := writeFile(filepath.Join(dir, "seeded", "RT", shape, arm, "api", "orders", name), []byte("package orders\n\n"+body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	harmless("one", "// Harmless does nothing.\nfunc Harmless() {}\n\nvar _ = Harmless\n")
	harmless("two", "// Also does nothing.\nfunc Also() int { return 1 }\n\nvar _ = Also\n")
	harmless("unrelated", "func NoComment() {}\n")
	return dir
}

// TestValidateWithRules is the audit's demo of N17: with all defaults on and a baseline, the base and
// the reference pass in both arms, V builds and runs its holdout as L does, harmless shapes do not
// land in S, a shape that only an unrelated rule rejects does not count, and with no match file no
// label comes out.
func TestValidateWithRules(t *testing.T) {
	app := baseTemplates()
	tmpl := Templates{record.ArmP: app[record.ArmP], record.ArmL: withRules(t, app[record.ArmL], record.ArmL, true), record.ArmV: withRules(t, app[record.ArmV], record.ArmV, true)}
	material := demoWithShapes(t)

	v := validate(t, material, tmpl, nil)
	t.Logf("no match file:\n%s", v.Summary())
	if v.NoLabels == "" || len(v.EnforcedByV) > 0 || len(v.EnforcedByL) > 0 {
		t.Errorf("with no match file, the run gives labels: %q %v %v", v.NoLabels, v.EnforcedByV, v.EnforcedByL)
	}
	if !strings.Contains(v.Summary(), "no enforced label") {
		t.Errorf("the summary does not say why there is no label:\n%s", v.Summary())
	}
	for _, it := range v.Items {
		if (it.Kind == "base" || it.Kind == "reference") && it.Arm != record.ArmP && !it.OK {
			t.Errorf("%s %s %s: %s", it.Kind, it.ID, it.Arm, it.Problem)
		}
	}
	j := v.ExportJudge(mustMeta(t, material))
	text, _ := json.Marshal(j)
	scanJudge(t, j, string(text))

	// The judge's decisions: the model check that rejects RD matches it. CODE-08 does not match RT.
	yes, no := true, false
	for i := range j.Pairs {
		if j.Pairs[i].TeamRule == "RD" && strings.HasPrefix(j.Pairs[i].FiredRule, "check-") {
			j.Pairs[i].Match = &yes
		} else {
			j.Pairs[i].Match = &no
		}
	}
	path := filepath.Join(t.TempDir(), "judge.json")
	if err := j.Save(path); err != nil {
		t.Fatal(err)
	}
	match, err := ReadMatches(path)
	if err != nil {
		t.Fatal(err)
	}
	v = validate(t, material, tmpl, match)
	t.Logf("with a match file:\n%s", v.Summary())
	for _, m := range []map[string]bool{v.EnforcedByV, v.EnforcedByL} {
		if m["RT"] {
			t.Errorf("harmless shapes and an unrelated rule enforce RT: %v", m)
		}
		if !m["RD"] {
			t.Errorf("RD is not enforced: %v", m)
		}
	}
	// As each arm ships, the model check puts RD in S. For the rules file alone, no rule matches it.
	for _, want := range []string{"the gate reads these", "set S: 1 (RD)", "rules file alone: set S: 0 ()", "rules file alone: set N: 2 (RD, RT)"} {
		if !strings.Contains(v.Summary(), want) {
			t.Errorf("the summary does not say %q:\n%s", want, v.Summary())
		}
	}
	found := false
	for _, it := range v.Items {
		if strings.HasPrefix(it.ID, "RT/unrelated") && it.Arm != record.ArmP {
			found = true
			if !it.Rejected || it.Matched || len(it.Fired) == 0 || it.Fired[0].Rule != "CODE-08" {
				t.Errorf("RT/unrelated %s: %+v", it.Arm, it)
			}
		}
		if (it.ID == "RT/one" || it.ID == "RT/two") && it.Arm != record.ArmP && it.Rejected {
			t.Errorf("a harmless shape is rejected: %+v", it)
		}
	}
	if !found {
		t.Errorf("no item RT/unrelated")
	}
	if !strings.Contains(v.Summary(), "unrelated rule") {
		t.Errorf("the report does not list the case that only an unrelated rule rejects:\n%s", v.Summary())
	}
}

func mustMeta(t *testing.T, material string) *record.Meta {
	t.Helper()
	m, err := record.LoadMeta(filepath.Join(material, "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// scanJudge reads the judge's file for each word that names an arm or shows a code of one.
func scanJudge(t *testing.T, j *JudgeFile, text string) {
	t.Helper()
	for _, word := range []string{`"arm"`, "enforced", "set S", "vulcan-lint", "VL0", "model-"} {
		if strings.Contains(text, word) {
			t.Errorf("the judge's file holds %q: %s", word, text)
		}
	}
	if m := regexp.MustCompile(`\bK\d{4}\b|\b0\d{3}\b`).FindString(text); m != "" {
		t.Errorf("the judge's file holds the code %q: %s", m, text)
	}
	checkWords := regexp.MustCompile(`(?i)vulcan|§|component|container|contract\.|checked|wrapper|gateway for|\bmain\b|manifest|\.go\b|arch\b`)
	for _, p := range j.Pairs {
		if strings.HasPrefix(p.FiredRule, "check-") {
			if m := checkWords.FindString(p.FiredText); m != "" {
				t.Errorf("the text of a model check holds %q: %s", m, p.FiredText)
			}
		}
	}
	for digits, c := range modelChecks {
		if m := checkWords.FindString(c.text); m != "" {
			t.Errorf("the sentence of %s holds %q: %s", digits, m, c.text)
		}
	}
}

// TestModelChecksTable holds the table to the code: each code of each arm has a sentence, and the
// table knows which arm has it alone.
func TestModelChecksTable(t *testing.T) {
	root := repoRoot()
	codes := func(re string, dirs ...string) map[string]bool {
		out := map[string]bool{}
		for _, dir := range dirs {
			err := filepath.WalkDir(filepath.Join(root, dir), func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
					return err
				}
				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				for _, m := range regexp.MustCompile(re).FindAllStringSubmatch(string(b), -1) {
					out[m[1]] = true
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	v := codes(`"K(\d{4})"`, "internal", "cmd")
	l := codes(`\bVL(\d{4})\b`, filepath.Join("poc", "golint"))
	for digits := range v {
		c, ok := modelChecks[digits]
		if !ok {
			t.Errorf("K%s has no sentence", digits)
		} else if want := map[bool]string{true: "", false: "V"}[l[digits]]; c.arm != want {
			t.Errorf("K%s: the table says arm %q, and the code says %q", digits, c.arm, want)
		}
	}
	for digits := range l {
		if digits >= "0802" && digits <= "0810" {
			continue // the rule kinds: a finding of these carries its rule's id
		}
		c, ok := modelChecks[digits]
		if !ok {
			t.Errorf("VL%s has no sentence", digits)
		} else if want := map[bool]string{true: "", false: "L"}[v[digits]]; c.arm != want {
			t.Errorf("VL%s: the table says arm %q, and the code says %q", digits, c.arm, want)
		}
	}
	oneArm := map[string]int{}
	for _, c := range modelChecks {
		oneArm[c.arm]++
	}
	// Audit N33: these are a difference between the arms, and they do not fail M3.
	t.Logf("codes that one arm alone emits: V %d, L %d", oneArm["V"], oneArm["L"])
	if oneArm["V"] != 19 || oneArm["L"] != 8 {
		t.Errorf("one-arm codes: V %d, L %d; the audit counts V 19, L 8", oneArm["V"], oneArm["L"])
	}
	for digits := range modelChecks {
		if id := judgeID("model-" + digits); fromJudgeID(id) != "model-"+digits {
			t.Errorf("the id of %s does not come back", digits)
		}
	}
}

// TestNoLabelsWhenBaseRejected: defaults with no baseline reject the base, so no label is written, even
// with a match file (audit N25).
func TestNoLabelsWhenBaseRejected(t *testing.T) {
	app := baseTemplates()
	tmpl := Templates{record.ArmL: withRules(t, app[record.ArmL], record.ArmL, false), record.ArmV: withRules(t, app[record.ArmV], record.ArmV, false)}
	material := demoWithShapes(t)
	meta := mustMeta(t, material)
	g := &Grader{Repo: repoRoot(), Material: material, ToolDir: tools(t), Meta: meta, Match: Matches{"RT": {"OBS-01": true, "CTX-06": true}}, Only: []string{"base", "seeded/RT/one"}}
	v, err := g.Validate(tmpl, t.TempDir(), 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(v.NoLabels, "rejects the base") || len(v.EnforcedByV) > 0 || len(v.EnforcedByL) > 0 {
		t.Errorf("labels with a rejected base: %q %v %v\n%s", v.NoLabels, v.EnforcedByV, v.EnforcedByL, v.Summary())
	}
}

// TestBuildsMeansCompiles is audit N28: with a rules file, Builds means the same in each arm, and the
// check is recorded apart. With no rules file, a record has no final check.
func TestBuildsMeansCompiles(t *testing.T) {
	app := baseTemplates()
	material := demoMaterial()
	meta := mustMeta(t, material)
	g := &Grader{Repo: repoRoot(), Material: material, ToolDir: tools(t), Meta: meta}
	if err := g.Prepare(filepath.Join(t.TempDir(), "prepared")); err != nil {
		t.Fatal(err)
	}
	ext := map[string]string{record.ArmL: ".go", record.ArmV: ".vulcan"}
	grade := func(t *testing.T, arm, from, extra string, more ...func(ws string)) *record.Run {
		t.Helper()
		out := t.TempDir()
		ws := filepath.Join(out, "workspace")
		if err := copyTemplate(from, ws); err != nil {
			t.Fatal(err)
		}
		if extra != "" {
			if err := writeFile(filepath.Join(ws, "api", "orders", "extra"+ext[arm]), []byte(extra)); err != nil {
				t.Fatal(err)
			}
		}
		for _, f := range more {
			f(ws)
		}
		r := &record.Run{Task: "DEMO", Arm: arm}
		if err := g.Grade(out, r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	const newFinding = "package orders\n\nfunc NoComment() {}\n"
	const modelError = "package orders\n\nimport \"stockroom/api/billing\"\n\nvar _ = billing.Ledger{}\n"
	for _, arm := range []string{record.ArmL, record.ArmV} {
		t.Run(arm, func(t *testing.T) {
			if r := grade(t, arm, app[arm], ""); r.FinalCheck != nil {
				t.Errorf("no rules file: a final check %+v", r.FinalCheck)
			}
			rules := withRules(t, app[arm], arm, true)
			clean := grade(t, arm, rules, "")
			if !clean.Builds || clean.FinalCheck == nil || !clean.FinalCheck.Passed {
				t.Errorf("only old findings: builds %v, check %+v", clean.Builds, clean.FinalCheck)
			}
			r := grade(t, arm, rules, newFinding)
			if !r.Builds || !r.Holdout.OK() {
				t.Errorf("a new finding: builds %v, holdout %+v", r.Builds, r.Holdout)
			}
			if r.FinalCheck == nil || r.FinalCheck.Passed || !slicesContains(r.FinalCheck.Fired, "CODE-08") {
				t.Errorf("a new finding: check %+v", r.FinalCheck)
			}
			m := grade(t, arm, rules, modelError)
			t.Logf("%s, a model error: builds %v, check %+v", arm, m.Builds, m.FinalCheck)
			if want := arm == record.ArmL; m.Builds != want {
				t.Errorf("a model error: builds %v, and as today it builds only in L", m.Builds)
			}
			// Audit N33: a code of the shared model check is no success in either arm.
			if m.TaskSuccess() || m.FinalCheck == nil || len(m.FinalCheck.Model) == 0 {
				t.Errorf("a model error: success %v, check %+v", m.TaskSuccess(), m.FinalCheck)
			}
			// Audit N34: an embedded *.db and a link that nothing uses build in both arms.
			embed := func(ws string) {
				if err := writeFile(filepath.Join(ws, "api", "orders", "seed.db"), []byte("seed")); err != nil {
					t.Fatal(err)
				}
			}
			e := grade(t, arm, rules, "package orders\n\nimport _ \"embed\"\n\n//go:embed seed.db\nvar seed []byte\n\nvar _ = seed\n", embed)
			if !e.Builds || !e.TaskSuccess() {
				t.Errorf("an embedded *.db: builds %v, success %v", e.Builds, e.TaskSuccess())
			}
			l := grade(t, arm, rules, "", func(ws string) {
				if err := os.Symlink(filepath.Join(ws, "api"), filepath.Join(ws, "link")); err == nil {
					return
				}
				if runtime.GOOS == "windows" {
					if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(ws, "link"), filepath.Join(ws, "api")).CombinedOutput(); err != nil {
						t.Fatalf("mklink: %v %s", err, out)
					}
				}
			})
			if !l.Builds || !l.TaskSuccess() {
				t.Errorf("a link that nothing uses: builds %v, success %v", l.Builds, l.TaskSuccess())
			}
		})
	}
}

func slicesContains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
