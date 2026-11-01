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
	for _, word := range []string{`"arm"`, "enforced", "set S", "vulcan-lint", "VL0", "K02", "K01"} {
		if strings.Contains(string(text), word) {
			t.Errorf("the judge's file holds %q: %s", word, text)
		}
	}

	// The judge's decisions: the model check that rejects RD matches it. CODE-08 does not match RT.
	yes, no := true, false
	for i := range j.Pairs {
		if j.Pairs[i].TeamRule == "RD" && strings.HasPrefix(j.Pairs[i].FiredRule, "model-") {
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
