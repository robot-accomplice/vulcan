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
	"strconv"
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
			if fs := strings.Fields(line); len(fs) == 3 && (fs[0] == "error" || fs[0] == "warn") {
				out = append(out, fs[0]+" "+strings.TrimPrefix(fs[2], prefix))
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
// application, and compares the findings by rule, part, and count. Which rules fire is a measure
// for QN, not a fault: the test fails only when the arms differ.
func TestDefaultsOnBase(t *testing.T) {
	code := regexp.MustCompile(`(K|VL)08\d\d\b`)
	id := regexp.MustCompile(`\(rule ([\w.-]+) in rules\.vulcan`)
	part := regexp.MustCompile(`((?:api|lib)[/\\][\w]+|system\.vulcan|cmd[/\\]\w+)`)
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
			where := part.FindString(strings.TrimPrefix(line, arm.dir))
			where = strings.ReplaceAll(where, "\\", "/")
			if where == "system.vulcan" || strings.HasPrefix(where, "cmd/") {
				where = "wires" // V declares them in system.vulcan, and L makes them in main
			}
			fired[arm.name][m[1]+" "+where]++
		}
	}
	// The one difference is in the applications: V's lib/shop has a comment on its library.vulcan,
	// and L's has no package comment.
	fired["V"]["CODE-08 lib/shop"]++
	for key, n := range fired["V"] {
		if fired["L"][key] != n {
			t.Errorf("%s: %d findings in V, and %d in L", key, n, fired["L"][key])
		}
	}
	for key, n := range fired["L"] {
		if fired["V"][key] != n {
			t.Errorf("%s: %d findings in L, and %d in V", key, n, fired["V"][key])
		}
	}
}

// TestListsInUpgrade checks that rules upgrade shows a change of the tool's lists, with the source
// the same, and that accepting it records the new version (audit M8).
func TestListsInUpgrade(t *testing.T) {
	for _, arm := range arms(t) {
		t.Run(arm.name, func(t *testing.T) {
			if c, out := run(t, arm.dir, arm.bin, "init"); c != 0 {
				t.Fatal(out)
			}
			path := filepath.Join(arm.dir, "rules.vulcan")
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(b)
			lists := regexp.MustCompile(`(?m)^lists (sha256:\w+)$`).FindStringSubmatch(text)
			if lists == nil {
				t.Fatalf("no lists in the head:\n%s", text[:200])
			}
			if _, out := run(t, arm.dir, arm.bin, "rules", "upgrade"); !strings.Contains(out, "has not changed") {
				t.Errorf("with the same lists: %s", out)
			}
			old := strings.Replace(text, lists[1], "sha256:0", 1)
			if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
				t.Fatal(err)
			}
			_, out := run(t, arm.dir, arm.bin, "rules", "upgrade")
			if !strings.Contains(out, "lists changed") || !strings.Contains(out, "nothing changed") {
				t.Fatalf("rules upgrade does not show the change:\n%s", out)
			}
			if _, d := run(t, arm.dir, arm.bin, "rules", "diff"); !strings.Contains(d, "lists sha256:0 here") {
				t.Errorf("rules diff does not show the change:\n%s", d)
			}
			hash := regexp.MustCompile(`--accept (sha256:\w+)`).FindStringSubmatch(out)[1]
			if c, o := run(t, arm.dir, arm.bin, "rules", "upgrade", "--accept", hash); c != 0 {
				t.Fatal(o)
			}
			if b, _ := os.ReadFile(path); !strings.Contains(string(b), "lists "+lists[1]) {
				t.Errorf("the accept did not record the lists")
			}
		})
	}
}

// TestDiffSaysTurnedOff checks the words of rules diff for each level (audit L6).
func TestDiffSaysTurnedOff(t *testing.T) {
	src := "rule A\n  kind forbid-call\n  level warn\n  funcs net/url.NoSuch\n  reason R.\n  source S.\n\n" +
		"rule B\n  kind forbid-call\n  level error\n  funcs net/url.NoOther\n  reason R.\n  source S.\n"
	source := filepath.Join(t.TempDir(), "team.rules")
	if err := os.WriteFile(source, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, arm := range arms(t) {
		if c, out := run(t, arm.dir, arm.bin, "init", "--rules", source); c != 0 {
			t.Fatal(out)
		}
		path := filepath.Join(arm.dir, "rules.vulcan")
		b, _ := os.ReadFile(path)
		text := strings.Replace(strings.ReplaceAll(string(b), "\r\n", "\n"), "level warn", "level off", 1)
		text = strings.Replace(text, "level error", "level warn", 1)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		_, out := run(t, arm.dir, arm.bin, "rules", "diff")
		if !strings.Contains(out, "~ rule A: turned off") || !strings.Contains(out, "~ rule B: changed") {
			t.Errorf("%s: %s", arm.name, out)
		}
	}
}

// TestLockRefusesModelError checks that rules lock stops on an error of the model, in both arms
// (audit L5), and that L stops on a wire that it cannot read.
func TestLockRefusesModelError(t *testing.T) {
	for _, arm := range arms(t) {
		t.Run(arm.name, func(t *testing.T) {
			name, text := "api/orders/extra.vulcan", "package orders\n\nimport \"stockroom/api/billing\"\n\nvar _ = billing.Ledger{}\n"
			if arm.name == "L" {
				name = "api/orders/extra.go"
			}
			if err := os.WriteFile(filepath.Join(arm.dir, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			code, out := run(t, arm.dir, arm.bin, "rules", "lock")
			if code == 0 {
				t.Errorf("rules lock passes with a leaf that imports a leaf:\n%s", out)
			}
			if _, err := os.Stat(filepath.Join(arm.dir, "rules.lock")); err == nil {
				t.Errorf("rules lock wrote a lock")
			}
		})
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

// TestBaselineCommands runs rules baseline on a copy of each base application. A check and a build then
// pass with each finding printed as old; a new finding fails both arms; a stale entry fails nothing;
// and rules diff shows the size.
func TestBaselineCommands(t *testing.T) {
	keys := map[string][]string{}
	for _, arm := range arms(t) {
		t.Run(arm.name, func(t *testing.T) {
			if c, out := run(t, arm.dir, arm.bin, "init"); c != 0 {
				t.Fatal(out)
			}
			if c, _ := run(t, arm.dir, arm.bin, arm.check...); c == 0 {
				t.Fatal("the defaults fire on the base application, and the check passes")
			}
			if c, out := run(t, arm.dir, arm.bin, "rules", "baseline"); c != 0 {
				t.Fatalf("rules baseline: %s", out)
			}
			c, out := run(t, arm.dir, arm.bin, arm.check...)
			if c != 0 || !strings.Contains(out, "old finding") {
				t.Fatalf("a check with a baseline: exit %d\n%s", c, out)
			}
			if arm.name == "V" {
				if c, out := run(t, arm.dir, arm.bin, "build"); c != 0 {
					t.Errorf("vulcan build with only old findings: %s", out)
				}
			}
			_, diff := run(t, arm.dir, arm.bin, "rules", "diff")
			if !regexp.MustCompile(`rules\.baseline holds \d+ findings, in \d+ keys`).MatchString(diff) {
				t.Errorf("rules diff does not show the baseline: %s", diff)
			}
			b, err := os.ReadFile(filepath.Join(arm.dir, "rules.baseline"))
			if err != nil {
				t.Fatal(err)
			}
			for _, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
				if line != "" && !strings.HasPrefix(line, "#") {
					keys[arm.name] = append(keys[arm.name], line)
				}
			}
			// A new finding fails. It is an exported name with no comment in a leaf.
			name, text := "api/orders/newone.vulcan", "package orders\n\nfunc NewOne() {}\n"
			if arm.name == "L" {
				name = "api/orders/newone.go"
			}
			if err := os.WriteFile(filepath.Join(arm.dir, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			c, out = run(t, arm.dir, arm.bin, arm.check...)
			if c == 0 || !regexp.MustCompile(`(K|VL)0807`).MatchString(out) {
				t.Errorf("a new finding passes: exit %d\n%s", c, out)
			}
			if arm.name == "V" {
				if c, _ := run(t, arm.dir, arm.bin, "build"); c == 0 {
					t.Errorf("vulcan build passes with a new finding")
				}
			}
			if err := os.Remove(filepath.Join(arm.dir, filepath.FromSlash(name))); err != nil {
				t.Fatal(err)
			}
			// A stale entry fails nothing.
			if err := os.WriteFile(filepath.Join(arm.dir, "rules.baseline"), append(b, []byte("1\tCODE-08\tcode\tapi/orders\tGone\tthe exported name Gone has no doc comment\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
			c, out = run(t, arm.dir, arm.bin, arm.check...)
			if c != 0 || !regexp.MustCompile(`(K|VL)0811`).MatchString(out) {
				t.Errorf("a stale entry: exit %d\n%s", c, out)
			}
			// No baseline: as before.
			if err := os.Remove(filepath.Join(arm.dir, "rules.baseline")); err != nil {
				t.Fatal(err)
			}
			if c, _ := run(t, arm.dir, arm.bin, arm.check...); c == 0 {
				t.Errorf("with no baseline, the check passes")
			}
		})
	}
	only := func(a, b []string) []string {
		var out []string
		for _, k := range a {
			if !slices.Contains(b, k) {
				out = append(out, k)
			}
		}
		return out
	}
	t.Logf("only V: %q\nonly L: %q", only(keys["V"], keys["L"]), only(keys["L"], keys["V"]))
	for _, k := range only(keys["V"], keys["L"]) {
		if !knownKeyDifference(k) {
			t.Errorf("a key in V only: %q", k)
		}
	}
	for _, k := range only(keys["L"], keys["V"]) {
		if !knownKeyDifference(k) {
			t.Errorf("a key in L only: %q", k)
		}
	}
}

// knownKeyDifference names what differs on the base applications because the applications differ:
// V's startup is L's Serve, and L's lib/shop has no package comment.
func knownKeyDifference(k string) bool {
	return strings.Contains(k, "library shop has no doc comment") || strings.Contains(k, "\tstartup\t") || strings.Contains(k, "\tServe\t")
}

// TestBaselineKeysAgree writes a baseline in each arm for forms that one arm has alone: a lowered
// return and a clause in V, contract calls in L, a port declared in another file, a wire, and a
// library. The files must be the same.
func TestBaselineKeysAgree(t *testing.T) {
	vulcan := buildVulcan(t)
	base := lintest.Read(t, filepath.Join("testdata", "base.txtar"))
	for _, id := range []string{"R01", "R70", "R85", "R86", "R82", "R130", "R140", "R147", "R149", "R183", "R243", "R290", "R291", "R292", "R293"} {
		t.Run(id, func(t *testing.T) {
			ldir := lintest.Write(t, base, lintest.Read(t, filepath.Join("testdata", id+".txtar")))
			varch := lintest.Read(t, filepath.Join("..", "..", "..", "conformance", id+".txtar"))
			vdir := t.TempDir()
			for _, f := range varch.Files {
				p := filepath.Join(vdir, filepath.FromSlash(f.Name))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, f.Data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(vdir, "go.mod"), []byte("module shop\n\ngo 1.26\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			var got [2]string
			for i, a := range []struct{ dir, bin string }{{vdir, vulcan}, {ldir, lintBin}} {
				if c, out := run(t, a.dir, a.bin, "rules", "baseline"); c != 0 {
					t.Fatalf("rules baseline: %s", out)
				}
				b, err := os.ReadFile(filepath.Join(a.dir, "rules.baseline"))
				if err != nil {
					t.Fatal(err)
				}
				got[i] = strings.ReplaceAll(string(b), "\r\n", "\n")
			}
			if got[0] != got[1] {
				t.Errorf("the keys differ:\nV:\n%s\nL:\n%s", got[0], got[1])
			}
			if !strings.Contains(got[0], "\t") {
				t.Errorf("no finding in the baseline:\n%s", got[0])
			}
		})
	}
}

// TestBaselineRefusesModelError checks that a model check never goes in the file.
func TestBaselineRefusesModelError(t *testing.T) {
	for _, arm := range arms(t) {
		t.Run(arm.name, func(t *testing.T) {
			if c, out := run(t, arm.dir, arm.bin, "init"); c != 0 {
				t.Fatal(out)
			}
			name := "api/orders/extra.vulcan"
			if arm.name == "L" {
				name = "api/orders/extra.go"
			}
			text := "package orders\n\nimport \"stockroom/api/billing\"\n\nvar _ = billing.Ledger{}\n"
			if err := os.WriteFile(filepath.Join(arm.dir, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			if c, out := run(t, arm.dir, arm.bin, "rules", "baseline"); c == 0 {
				t.Errorf("rules baseline passes with a model error: %s", out)
			}
			if _, err := os.Stat(filepath.Join(arm.dir, "rules.baseline")); err == nil {
				t.Errorf("rules baseline wrote a file with a model error")
			}
		})
	}
}

// fixture writes the small project of case R05 in an arm, with no rules file.
func fixture(t *testing.T, arm, dir string) {
	t.Helper()
	write := func(name string, data []byte) {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if arm == "L" {
		src := lintest.Write(t, lintest.Read(t, filepath.Join("testdata", "base.txtar")), lintest.Read(t, filepath.Join("testdata", "R05.txtar")))
		_ = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(src, p)
				b, _ := os.ReadFile(p)
				write(filepath.ToSlash(rel), b)
			}
			return nil
		})
		return
	}
	for _, f := range lintest.Read(t, filepath.Join("..", "..", "..", "conformance", "R05.txtar")).Files {
		write(f.Name, f.Data)
	}
}

// TestReplaceByFileSystem is audit N23, N24, N31, and N32 (plan-r3, amendment 12): each replace by a
// local path is rejected, in both arms, inside the project or outside, in each form of the path, and
// whether it exists or not. It holds es_build, es_git, es_bin, li_, ct_bld, and ct_obj of the audit.
func TestReplaceByFileSystem(t *testing.T) {
	type form struct {
		name   string
		target func(root, dir string) string
	}
	forms := []form{
		{"relative", func(_, dir string) string { return "./" + dir }},
	}
	if runtime.GOOS == "windows" {
		forms = append(forms,
			form{"unc", func(root, dir string) string { return `\\?\` + filepath.Join(root, filepath.FromSlash(dir)) }},
			form{"short", func(root, dir string) string { return shortPath(filepath.Join(root, filepath.FromSlash(dir))) }},
			form{"junction", func(root, dir string) string {
				link := filepath.Join(filepath.Dir(root), "o")
				if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, root).CombinedOutput(); err != nil {
					t.Fatalf("mklink: %v\n%s", err, out)
				}
				return "../o/" + dir
			}},
			form{"direct-junction", func(root, dir string) string {
				link := filepath.Join(filepath.Dir(root), "o")
				if out, err := exec.Command("cmd", "/c", "mklink", "/J", link, filepath.Join(root, filepath.FromSlash(dir))).CombinedOutput(); err != nil {
					t.Fatalf("mklink: %v\n%s", err, out)
				}
				return "../o"
			}},
			form{"drive", func(root, dir string) string { return filepath.Join(root, filepath.FromSlash(dir)) }},
		)
	} else {
		forms = append(forms,
			form{"absolute", func(root, dir string) string { return filepath.Join(root, dir) }},
			form{"proc", func(_, dir string) string { return "/proc/self/cwd/" + dir }},
			form{"link", func(root, dir string) string {
				link := filepath.Join(filepath.Dir(root), "o")
				if err := os.Symlink(root, link); err != nil {
					t.Fatal(err)
				}
				return "../o/" + dir
			}},
			form{"direct-link", func(root, dir string) string {
				link := filepath.Join(filepath.Dir(root), "o")
				if err := os.Symlink(filepath.Join(root, filepath.FromSlash(dir)), link); err != nil {
					t.Fatal(err)
				}
				return "../o"
			}},
		)
	}
	vulcan := buildVulcan(t)
	for _, arm := range []struct {
		name, bin string
		check     []string
	}{{"V", vulcan, []string{"check"}}, {"L", lintBin, []string{"./..."}}} {
		for _, f := range forms {
			for _, dir := range []string{".vulcan/build/evil", ".vulcan/bin/evil", ".git/evil", ".git/objects/evil", "third", "../out/evil"} {
				t.Run(arm.name+"/"+f.name+"/"+dir, func(t *testing.T) {
					base, err := filepath.EvalSymlinks(t.TempDir())
					if err != nil {
						t.Fatal(err)
					}
					root := filepath.Join(base, "project")
					fixture(t, arm.name, root)
					evil := filepath.Join(root, filepath.FromSlash(dir))
					if err := os.MkdirAll(evil, 0o755); err != nil {
						t.Fatal(err)
					}
					put := func(p, text string) {
						if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
							t.Fatal(err)
						}
					}
					put(filepath.Join(evil, "go.mod"), "module example.com/evil\n\ngo 1.26.0\n")
					put(filepath.Join(evil, "s.go"), "package evil\n\nimport \"os\"\n\n// X stops.\nfunc X() { os.Exit(1) }\n")
					use := filepath.Join(root, "api", "orders", "extra.go")
					if arm.name == "V" {
						use = filepath.Join(root, "api", "orders", "extra.vulcan")
					}
					put(use, "package orders\n\nimport \"example.com/evil\"\n\nfunc stopIt() { evil.X() }\n")
					put(filepath.Join(root, "go.mod"), "module shop\n\ngo 1.26.0\n\nrequire example.com/evil v0.0.0\n\nreplace example.com/evil => "+strconv.Quote(f.target(root, dir))+"\n")
					code, out := run(t, root, arm.bin, arm.check...)
					if code == 0 || !regexp.MustCompile(`go\.mod:7:1: (error K0002|VL0002)`).MatchString(out) {
						t.Errorf("a replace by a local path passes: exit %d\n%s", code, out)
					}
				})
			}
		}
		t.Run(arm.name+"/missing", func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "project")
			fixture(t, arm.name, root)
			for _, target := range []string{"./missing/evil", "../missing/evil"} {
				if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module shop\n\ngo 1.26.0\n\nrequire example.com/evil v0.0.0\n\nreplace example.com/evil => "+target+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				code, out := run(t, root, arm.bin, arm.check...)
				if strings.Contains(out, "panic") {
					t.Errorf("%s: %s", target, out)
				}
				if code == 0 || !strings.Contains(out, "0002") {
					t.Errorf("%s: exit %d\n%s", target, code, out)
				}
			}
		})
	}
}

// TestLinkedPart is audit N40: a leaf's directory that is a link fails in both arms, at the link, by
// the shared walk. The go command does not follow a link when it expands ./..., and a build does. The
// link leads inside the project, to a directory that the go command skips, or outside it.
func TestLinkedPart(t *testing.T) {
	vulcan := buildVulcan(t)
	for _, arm := range []struct {
		name, bin string
		check     []string
	}{{"V", vulcan, []string{"check"}}, {"L", lintBin, []string{"./..."}}} {
		for _, to := range []string{"", "_keep/orders", "testdata/orders", ".keep/orders", "../out/orders"} {
			name := to
			if name == "" {
				name = "plain"
			}
			t.Run(arm.name+"/"+name, func(t *testing.T) {
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				root := filepath.Join(base, "project")
				fixture(t, arm.name, root)
				if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
					if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module shop\n\ngo 1.26.0\n"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				leaf := filepath.Join(root, "api", "orders")
				if to != "" {
					target := filepath.Join(root, filepath.FromSlash(to))
					if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.Rename(leaf, target); err != nil {
						t.Fatal(err)
					}
					if runtime.GOOS == "windows" {
						if out, err := exec.Command("cmd", "/c", "mklink", "/J", leaf, target).CombinedOutput(); err != nil {
							t.Fatalf("mklink: %v\n%s", err, out)
						}
					} else if err := os.Symlink(target, leaf); err != nil {
						t.Fatal(err)
					}
				}
				code, out := run(t, root, arm.bin, arm.check...)
				linked := regexp.MustCompile(`api[/\\]orders:1:1: (error K0002|VL0002)`).MatchString(out)
				switch {
				case to == "" && code != 0:
					t.Errorf("the plain project fails: exit %d\n%s", code, out)
				case to != "" && (code == 0 || !linked):
					t.Errorf("a leaf's directory that is a link passes: exit %d\n%s", code, out)
				}
			})
		}
	}
}

// TestLinkedFile is audit N44: vendor/modules.txt or a nested go.mod that is a link to a file fails in
// both arms, as the plain file does. The go command reads each through the link, so with GOFLAGS empty
// L built vendored code that no rule reads. A link to a file needs a right on Windows that an account
// may lack; the test then skips.
func TestLinkedFile(t *testing.T) {
	vulcan := buildVulcan(t)
	for _, arm := range []struct {
		name, bin string
		check     []string
	}{{"V", vulcan, []string{"check"}}, {"L", lintBin, []string{"./..."}}} {
		for _, name := range []string{"vendor/modules.txt", "api/orders/go.mod"} {
			t.Run(arm.name+"/"+name, func(t *testing.T) {
				base, err := filepath.EvalSymlinks(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				root := filepath.Join(base, "project")
				fixture(t, arm.name, root)
				if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
					if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module shop\n\ngo 1.26.0\n"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				target := filepath.Join(base, "target")
				if err := os.WriteFile(target, []byte("# example.com/evil v1.0.0\n## explicit; go 1.26.0\nexample.com/evil\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(root, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, link); err != nil {
					t.Skipf("no link to a file on this host: %v", err)
				}
				code, out := run(t, root, arm.bin, arm.check...)
				at := regexp.MustCompile(regexp.QuoteMeta(name) + `:1:1: (error K0002|VL0002)`)
				if code == 0 || !at.MatchString(filepath.ToSlash(out)) {
					t.Errorf("%s as a link to a file passes: exit %d\n%s", name, code, out)
				}
			})
		}
	}
}

// TestAgentK35 is what an agent sees during a run, with one new finding (audit k35, amendment 11).
// V's build and test refuse it, and L's go command does not read rules. That stays, as a measured cost.
func TestAgentK35(t *testing.T) {
	for _, arm := range arms(t) {
		t.Run(arm.name, func(t *testing.T) {
			if c, out := run(t, arm.dir, arm.bin, "init"); c != 0 {
				t.Fatal(out)
			}
			if c, out := run(t, arm.dir, arm.bin, "rules", "baseline"); c != 0 {
				t.Fatal(out)
			}
			name := "api/orders/stopit.vulcan"
			if arm.name == "L" {
				name = "api/orders/stopit.go"
			}
			text := "package orders\n\nimport \"os\"\n\nfunc stopIt() { os.Exit(1) }\n\nvar _ = stopIt\n"
			if err := os.WriteFile(filepath.Join(arm.dir, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			type step struct {
				bin  string
				args []string
				want int
			}
			steps := []step{{arm.bin, arm.check, 3}, {"go", []string{"build", "./cmd/stockroom"}, 0}, {"go", []string{"test", "./..."}, 0}}
			if arm.name == "V" {
				steps = []step{{arm.bin, []string{"check"}, 1}, {arm.bin, []string{"build"}, 1}, {arm.bin, []string{"test"}, 1}}
			}
			for _, s := range steps {
				code, out := run(t, arm.dir, s.bin, s.args...)
				if code != s.want {
					t.Errorf("%v: exit %d, want %d\n%s", s.args, code, s.want, out)
				}
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
			if b, _ := os.ReadFile(filepath.Join(dir, "go.mod")); !strings.Contains(string(b), "go 1.27.3") {
				t.Errorf("go.mod does not name the current toolchain:\n%s", b)
			}
			if code, out := run(t, dir, arm.bin, arm.check...); code != 0 {
				t.Errorf("the check of a new project: %s", out)
			}
		})
	}
}
