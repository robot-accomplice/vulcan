package conformance

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"
)

// ruleCase matches the cases of plan-r3, step 2: the rule kinds, the rules file, and holes H6 and H7.
var ruleCase = regexp.MustCompile(`^(R\d+|H[67][a-z])$`)

// TestRuleCasesAgree checks that V and L have the same cases, and that each case lists the same
// codes in both, with K for VL. A position can differ, because the arms declare ports and wires
// in other files.
func TestRuleCasesAgree(t *testing.T) {
	ids := func(dir string) []string {
		paths, err := filepath.Glob(filepath.Join(dir, "*.txtar"))
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, p := range paths {
			if id := strings.TrimSuffix(filepath.Base(p), ".txtar"); ruleCase.MatchString(id) {
				out = append(out, id)
			}
		}
		return out
	}
	vdir := filepath.Join("..", "..", "..", "conformance")
	l, v := ids("testdata"), ids(vdir)
	if len(l) < 50 || !slices.Equal(l, v) {
		t.Fatalf("the cases differ:\n L %v\n V %v", l, v)
	}
	for _, id := range l {
		lw := codes(t, filepath.Join("testdata", id+".txtar"), "VL")
		vw := codes(t, filepath.Join(vdir, id+".txtar"), "K")
		if !slices.Equal(lw, vw) {
			t.Errorf("%s lists other codes: L %v, V %v", id, lw, vw)
		}
	}
}

func codes(t *testing.T, path, prefix string) []string {
	t.Helper()
	a := lintest.Read(t, path)
	var out []string
	for _, f := range a.Files {
		if f.Name != "want.txt" {
			continue
		}
		for _, line := range strings.Split(string(f.Data), "\n") {
			if fs := strings.Fields(line); len(fs) == 3 && fs[0] == "error" {
				out = append(out, strings.TrimPrefix(fs[2], prefix))
			}
		}
	}
	slices.Sort(out)
	return out
}

var vulcanBin string

func buildVulcan(t *testing.T) string {
	t.Helper()
	if vulcanBin != "" {
		return vulcanBin
	}
	vulcanBin = filepath.Join(filepath.Dir(lintBin), "vulcan")
	if runtime.GOOS == "windows" {
		vulcanBin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", vulcanBin, "github.com/robot-accomplice/vulcan/cmd/vulcan")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return vulcanBin
}

func run(t *testing.T, dir, bin string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off", "GOPROXY=off")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), out.String()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0, out.String()
}

// copyTree copies a tree. It skips what the tools generate.
func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		if d.IsDir() {
			if d.Name() == ".vulcan" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(to, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(to, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// arms gives a scratch copy of each base application, with the command of its arm.
func arms(t *testing.T) []struct {
	name, dir, bin string
	check          []string
} {
	t.Helper()
	app := filepath.Dir(lintest.Workspace())
	vdir, ldir := t.TempDir(), t.TempDir()
	copyTree(t, filepath.Join(app, "vulcan"), vdir)
	copyTree(t, filepath.Join(app, "golint"), ldir)
	return []struct {
		name, dir, bin string
		check          []string
	}{
		{"V", vdir, buildVulcan(t), []string{"check"}},
		{"L", ldir, lintBin, []string{"./..."}},
	}
}

// TestRulesCommands runs init, rules diff, rules upgrade, and rules lock in both arms, on a copy of
// each base application, and compares what they write and say.
func TestRulesCommands(t *testing.T) {
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "team.rules")
	first := "rule A\n  kind forbid-call\n  level error\n  funcs net/url.NoSuch\n  reason First.\n  source S.\n\n" +
		"rule B\n  kind forbid-call\n  level error\n  funcs net/url.NoOther\n  reason Second.\n  source S.\n\n" +
		"rule C\n  kind forbid-call\n  level error\n  funcs net/url.NoThird\n  reason Third.\n  source S.\n"
	second := strings.ReplaceAll(first, "First.", "First, better.") + "\nrule D\n  kind forbid-call\n  level error\n  funcs net/url.NoFourth\n  reason Fourth.\n  source S.\n"
	second = strings.Replace(second, "rule C\n  kind forbid-call\n  level error\n  funcs net/url.NoThird\n  reason Third.\n  source S.\n\n", "", 1)
	said := map[string][]string{}
	locks := map[string]string{}
	for _, arm := range arms(t) {
		t.Run(arm.name, func(t *testing.T) {
			write := func(name, text string) {
				if err := os.WriteFile(filepath.Join(arm.dir, name), []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			read := func(name string) string {
				b, err := os.ReadFile(filepath.Join(arm.dir, name))
				if err != nil {
					t.Fatal(err)
				}
				return string(b)
			}
			step := func(want int, args ...string) string {
				t.Helper()
				code, out := run(t, arm.dir, arm.bin, args...)
				if code != want {
					t.Fatalf("%v: exit %d, want %d\n%s", args, code, want, out)
				}
				said[arm.name] = append(said[arm.name], strings.Join(args, " ")+"\n"+strings.ReplaceAll(strings.ReplaceAll(out, "vulcan-lint", "TOOL"), "vulcan ", "TOOL "))
				return out
			}
			if err := os.WriteFile(source, []byte(first), 0o644); err != nil {
				t.Fatal(err)
			}
			step(0, "init", "--rules", source)
			head := read("rules.vulcan")
			if !strings.Contains(head, "origin ") || !strings.Contains(head, "hash sha256:") {
				t.Fatalf("no head:\n%s", head)
			}
			step(1, "init", "--rules", source) // the team's file stays

			// A deleted rule, a rule turned off, and a rule added, and what diff says of them.
			text := strings.Replace(head, "rule B\n  kind forbid-call\n  level error\n  funcs net/url.NoOther\n  reason Second.\n  source S.\n\n", "", 1)
			text = strings.Replace(text, "funcs net/url.NoSuch\n", "funcs net/url.NoSuch\n", 1)
			text = strings.Replace(text, "rule C\n  kind forbid-call\n  level error", "rule C\n  kind forbid-call\n  level off", 1)
			text += "\n# the team's own rule\nrule T\n  kind forbid-call\n  level error\n  funcs net/url.Nothing\n  reason Ours.\n  source Us.\n"
			write("rules.vulcan", text)
			out := step(0, "rules", "diff")
			for _, w := range []string{"- rule B", "~ rule C: turned off", "+ rule T"} {
				if !strings.Contains(out, w) {
					t.Errorf("rules diff does not say %q:\n%s", w, out)
				}
			}

			// An upgrade shows a difference, and changes nothing until it is accepted.
			if err := os.WriteFile(source, []byte(second), 0o644); err != nil {
				t.Fatal(err)
			}
			before := read("rules.vulcan")
			out = step(0, "rules", "upgrade")
			if read("rules.vulcan") != before {
				t.Fatal("rules upgrade changed the rules without --accept")
			}
			for _, w := range []string{"~ rule A", "+ rule D", "= rule C", "nothing changed"} {
				if !strings.Contains(out, w) {
					t.Errorf("rules upgrade does not say %q:\n%s", w, out)
				}
			}
			step(1, "rules", "upgrade", "--accept", "sha256:0")
			hash := regexp.MustCompile(`--accept (sha256:\w+)`).FindStringSubmatch(out)[1]
			step(0, "rules", "upgrade", "--accept", hash)
			after := read("rules.vulcan")
			for _, w := range []string{"First, better.", "rule D", "# the team's own rule", "rule T", "hash " + hash} {
				if !strings.Contains(after, w) {
					t.Errorf("after the upgrade, the rules do not hold %q:\n%s", w, after)
				}
			}
			if strings.Contains(after, "rule B") {
				t.Errorf("the upgrade brought back a rule that the team deleted:\n%s", after)
			}
			if !strings.Contains(after, "rule C\n  kind forbid-call\n  level off") {
				t.Errorf("the upgrade removed a rule that the team changed:\n%s", after)
			}
			if code, out := run(t, arm.dir, arm.bin, arm.check...); code != 0 {
				t.Errorf("the check fails after the upgrade:\n%s", out)
			}

			// The lock. The check never reads the source: remove it and the check passes.
			step(0, "rules", "lock")
			locks[arm.name] = read("rules.lock")
			if err := os.Remove(source); err != nil {
				t.Fatal(err)
			}
			write("rules.vulcan", strings.Replace(after, "rule A\n", "rule L\n  kind depends\n  level error\n  map locked\n  reason Locked.\n  source S.\n\nrule A\n", 1))
			if code, out := run(t, arm.dir, arm.bin, arm.check...); code != 0 {
				t.Errorf("the check with the lock and no source fails:\n%s", out)
			}
			line := "wire api/events -> api/store\n"
			if !strings.Contains(read("rules.lock"), line) {
				t.Fatalf("the lock has no %q", line)
			}
			write("rules.lock", strings.Replace(read("rules.lock"), line, "", 1))
			code, out := run(t, arm.dir, arm.bin, arm.check...)
			if code == 0 || !regexp.MustCompile(`(K|VL)0806`).MatchString(out) {
				t.Errorf("a wire that the lock does not hold passes (exit %d):\n%s", code, out)
			}
			step(0, "rules", "lock") // a rule's error does not stop the lock that clears it
			if code, out := run(t, arm.dir, arm.bin, arm.check...); code != 0 {
				t.Errorf("the check fails after rules lock:\n%s", out)
			}
		})
	}
	if locks["V"] != locks["L"] {
		t.Errorf("the arms write other locks for one architecture:\nV:\n%s\nL:\n%s", locks["V"], locks["L"])
	}
	if strings.Join(said["V"], "\n") != strings.Join(said["L"], "\n") {
		t.Errorf("the arms say other things:\nV:\n%s\nL:\n%s", strings.Join(said["V"], "\n"), strings.Join(said["L"], "\n"))
	}
}

// TestDefaultsOnBase runs both arms with the draft of defaults v1 on a copy of each base
// application, and compares the rules that fire. Which rules fire is a measure for QN, not a fault:
// the test fails only when the arms differ.
func TestDefaultsOnBase(t *testing.T) {
	code := regexp.MustCompile(`(K|VL)08\d\d\b`)
	id := regexp.MustCompile(`\(rule ([\w.-]+) in rules\.vulcan`)
	fired := map[string]map[string]int{}
	for _, arm := range arms(t) {
		if c, out := run(t, arm.dir, arm.bin, "init"); c != 0 {
			t.Fatalf("%s init: %s", arm.name, out)
		}
		_, out := run(t, arm.dir, arm.bin, arm.check...)
		fired[arm.name] = map[string]int{}
		seen := map[string]bool{}
		for _, line := range strings.Split(out, "\n") {
			if !code.MatchString(line) || seen[line] {
				continue
			}
			seen[line] = true
			m := id.FindStringSubmatch(line)
			if m == nil {
				t.Fatalf("%s: a rule's finding with no id: %s", arm.name, line)
			}
			fired[arm.name][m[1]]++
		}
		t.Logf("%s fires %v", arm.name, fired[arm.name])
	}
	for rule := range fired["V"] {
		if fired["L"][rule] == 0 {
			t.Errorf("rule %s fires in V and not in L", rule)
		}
	}
	for rule := range fired["L"] {
		if fired["V"][rule] == 0 {
			t.Errorf("rule %s fires in L and not in V", rule)
		}
	}
}

// TestTransitionsOnStockroom applies a table of the order's status to both base applications. The
// table of the application passes. Without one change that the application makes, both arms reject
// that write.
func TestTransitionsOnStockroom(t *testing.T) {
	table := func(allow string) string {
		return "origin test\nhash sha256:0\n\nrule ORDER\n  kind transitions\n  level error\n  type lib/shop.Order\n  field Status\n  initial pending\n  allow " + allow + "\n  load gateway\n  reason R.\n  source S.\n"
	}
	for _, arm := range arms(t) {
		t.Run(arm.name, func(t *testing.T) {
			path := filepath.Join(arm.dir, "rules.vulcan")
			if err := os.WriteFile(path, []byte(table("pending->paid paid->shipped pending->cancelled paid->cancelled")), 0o644); err != nil {
				t.Fatal(err)
			}
			if code, out := run(t, arm.dir, arm.bin, arm.check...); code != 0 {
				t.Errorf("the table of the application: %s", out)
			}
			if err := os.WriteFile(path, []byte(table("paid->shipped pending->cancelled paid->cancelled")), 0o644); err != nil {
				t.Fatal(err)
			}
			code, out := run(t, arm.dir, arm.bin, arm.check...)
			if code == 0 || !regexp.MustCompile(`(K|VL)0805`).MatchString(out) || !strings.Contains(out, `from pending to "paid"`) {
				t.Errorf("a change that the table does not allow passes (exit %d):\n%s", code, out)
			}
		})
	}
}

// TestScaffoldMatchesWorkspace keeps the files of vulcan-lint init equal to the workspace's.
func TestScaffoldMatchesWorkspace(t *testing.T) {
	for file, src := range map[string]string{"schema.go.txt": "arch/schema.go", "contract.go.txt": "lib/contract/contract.go"} {
		a, err := os.ReadFile(filepath.Join("..", "scaffold", "files", file))
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(lintest.Workspace(), filepath.FromSlash(src)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.ReplaceAll(string(a), "\r\n", "\n") != strings.ReplaceAll(string(b), "\r\n", "\n") {
			t.Errorf("scaffold/files/%s is not %s", file, src)
		}
	}
}

// TestInitIsClean checks that a project made by init passes its own check, in both arms.
func TestInitIsClean(t *testing.T) {
	for _, arm := range []struct {
		name, bin string
		check     []string
	}{{"V", buildVulcan(t), []string{"check"}}, {"L", lintBin, []string{"./..."}}} {
		t.Run(arm.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "newapp")
			if code, out := run(t, filepath.Dir(dir), arm.bin, "init", "newapp"); code != 0 {
				t.Fatalf("init: %s", out)
			}
			for _, f := range []string{"go.mod", "rules.vulcan", "rules.origin"} {
				if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
					t.Error(err)
				}
			}
			if code, out := run(t, dir, arm.bin, arm.check...); code != 0 {
				t.Errorf("the check of a new project: %s", out)
			}
		})
	}
}
