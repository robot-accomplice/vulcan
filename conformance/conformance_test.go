// Package conformance runs the conformance cases of docs/release-1/conformance.md.
//
// Each case is one txtar archive, <id>.txtar, that holds the files of the case directory of the
// spec's format: a small Vulcan project and want.txt. A go.mod with "module shop" is added when
// the archive has none. Each line of want.txt is one expectation:
//
//	error  api/orders/orders.vulcan:12:5  K0101
//	warn   api/billing/ledger.vulcan:40:2  K0525
//	accept
//	run    exit=0  stdout=out.txt
//	cmd    exit=2  vulcan fmt
//
// An error case expects exactly the listed errors, no more. Warnings are compared only when
// want.txt lists one.
package conformance

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"

	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/driver"
)

func TestConformance(t *testing.T) {
	cases, err := filepath.Glob("*.txtar")
	if err != nil || len(cases) == 0 {
		t.Fatalf("no cases: %v", err)
	}
	for _, path := range cases {
		id := strings.TrimSuffix(filepath.Base(path), ".txtar")
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			runCase(t, path)
		})
	}
}

type want struct {
	errors, warns []string // "file:line:col code"
	accept        bool
	run           *runWant
	cmds          []cmdWant
}

type runWant struct {
	exit   int
	stdout string
	args   []string
}

type cmdWant struct {
	exit int
	args []string
}

func parseWant(t *testing.T, b []byte, files map[string][]byte) want {
	var w want
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || strings.HasPrefix(f[0], "#") {
			continue
		}
		switch f[0] {
		case "error", "warn":
			if len(f) != 3 {
				t.Fatalf("bad want line %q", line)
			}
			e := f[1] + " " + f[2]
			if f[0] == "error" {
				w.errors = append(w.errors, e)
			} else {
				w.warns = append(w.warns, e)
			}
		case "accept":
			w.accept = true
		case "run":
			r := &runWant{}
			for _, kv := range f[1:] {
				k, v, _ := strings.Cut(kv, "=")
				switch k {
				case "exit":
					r.exit, _ = strconv.Atoi(v)
				case "stdout":
					r.stdout = string(files[v])
				case "arg":
					r.args = append(r.args, v)
				}
			}
			w.run = r
		case "cmd":
			k, v, _ := strings.Cut(f[1], "=")
			if k != "exit" {
				t.Fatalf("bad cmd line %q", line)
			}
			n, _ := strconv.Atoi(v)
			w.cmds = append(w.cmds, cmdWant{exit: n, args: f[3:]})
		default:
			t.Fatalf("bad want line %q", line)
		}
	}
	return w
}

func runCase(t *testing.T, path string) {
	ar, err := txtar.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files := map[string][]byte{}
	for _, f := range ar.Files {
		files[f.Name] = f.Data
		full := filepath.Join(dir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, f.Data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := files["go.mod"]; !ok {
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module shop\n\ngo 1.26\n"), 0o644)
	}
	w := parseWant(t, files["want.txt"], files)
	for _, c := range w.cmds {
		got := vulcanExit(t, dir, c.args)
		if got != c.exit {
			t.Errorf("vulcan %s: exit %d, want %d", strings.Join(c.args, " "), got, c.exit)
		}
	}
	if !w.accept && w.run == nil && len(w.errors) == 0 && len(w.warns) == 0 {
		return
	}
	r, err := driver.Check(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("VULCAN_SHOW") != "" {
		var all bytes.Buffer
		r.Diags.Print(&all)
		t.Logf("diagnostics:\n%s", all.String())
	}
	var gotErr, gotWarn []string
	for _, d := range r.Diags.Items {
		e := d.Pos.String() + " " + d.Code
		if d.Severity == diag.Error {
			gotErr = append(gotErr, e)
		} else {
			gotWarn = append(gotWarn, e)
		}
	}
	sort.Strings(gotErr)
	sort.Strings(w.errors)
	if strings.Join(gotErr, "\n") != strings.Join(w.errors, "\n") {
		var all bytes.Buffer
		r.Diags.Print(&all)
		t.Errorf("errors:\n got:\n  %s\n want:\n  %s\n diagnostics:\n%s", strings.Join(gotErr, "\n  "), strings.Join(w.errors, "\n  "), all.String())
	}
	if len(w.warns) > 0 {
		sort.Strings(gotWarn)
		sort.Strings(w.warns)
		if strings.Join(gotWarn, "\n") != strings.Join(w.warns, "\n") {
			t.Errorf("warnings: got %v, want %v", gotWarn, w.warns)
		}
	}
	if w.run != nil && r.OK {
		cmd := exec.Command(driver.Binary(r.P), w.run.args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		if code != w.run.exit {
			t.Errorf("exit %d, want %d; output:\n%s", code, w.run.exit, out.String())
		}
		if w.run.stdout != "" && strings.TrimSpace(out.String()) != strings.TrimSpace(w.run.stdout) {
			t.Errorf("output:\n%s\nwant:\n%s", out.String(), w.run.stdout)
		}
	}
}

var vulcanBin string

func vulcanExit(t *testing.T, dir string, args []string) int {
	if vulcanBin == "" {
		t.Fatal("no vulcan binary")
	}
	cmd := exec.Command(vulcanBin, args...)
	cmd.Dir = dir
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0
}

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "vulcan-conformance")
	if err != nil {
		panic(err)
	}
	vulcanBin = filepath.Join(tmp, "vulcan")
	if runtime.GOOS == "windows" {
		vulcanBin += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", vulcanBin, "../cmd/vulcan").CombinedOutput(); err != nil {
		panic(string(out))
	}
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}
