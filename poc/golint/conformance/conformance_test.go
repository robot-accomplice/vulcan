// Package conformance runs the Go form of the conformance cases that the desk test maps to the
// analyzers A1 to A8 (docs/poc/desk-test-r2.md, section 2) through the vulcan-lint command.
//
// testdata/base.txtar is the base system; each other archive is a case: its files replace the
// base's, and want.txt lists the expected diagnostics, one per line, or "accept":
//
//	error api/orders/orders.go:4:2 VL0101
//
// A case expects exactly the listed diagnostics, no more. gen_cases.py writes the archives.
package conformance

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"

	// The analyzers, so that a change to them is a change to this test (go test caches results).
	_ "github.com/robot-accomplice/vulcan/poc/golint/contracts"
	_ "github.com/robot-accomplice/vulcan/poc/golint/effects"
	_ "github.com/robot-accomplice/vulcan/poc/golint/imports"
	_ "github.com/robot-accomplice/vulcan/poc/golint/invariants"
	_ "github.com/robot-accomplice/vulcan/poc/golint/layout"
	_ "github.com/robot-accomplice/vulcan/poc/golint/libstate"
	_ "github.com/robot-accomplice/vulcan/poc/golint/ports"
	_ "github.com/robot-accomplice/vulcan/poc/golint/wiring"
)

var lintBin string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "vulcan-lint")
	if err != nil {
		panic(err)
	}
	lintBin = filepath.Join(dir, "vulcan-lint")
	if runtime.GOOS == "windows" {
		lintBin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", lintBin, "github.com/robot-accomplice/vulcan/poc/golint/cmd/vulcan-lint")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic(string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// diagRE reads a line of the command's output: file:line:col: VL0101 message.
var diagRE = regexp.MustCompile(`^(.+?):(\d+):(\d+): (VL\d{4}) `)

// Lint runs vulcan-lint ./... in dir and returns its diagnostics as "path:line:col CODE", with
// paths relative to dir, and the whole output.
func Lint(t *testing.T, dir string) ([]string, string) {
	t.Helper()
	cmd := exec.Command(lintBin, "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off", "GOPROXY=off")
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	var exit *exec.ExitError
	if err != nil && !(errors.As(err, &exit) && exit.ExitCode() == 3) {
		t.Fatalf("vulcan-lint: %v\n%s", err, out.String())
	}
	var got []string
	seen := map[string]bool{}
	for _, line := range strings.Split(out.String(), "\n") {
		m := diagRE.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		rel, err := filepath.Rel(dir, m[1])
		if err != nil {
			rel = m[1]
		}
		d := filepath.ToSlash(rel) + ":" + m[2] + ":" + m[3] + " " + m[4]
		if !seen[d] {
			seen[d] = true
			got = append(got, d)
		}
	}
	sort.Strings(got)
	return got, out.String()
}

func TestConformance(t *testing.T) {
	base := lintest.Read(t, filepath.Join("testdata", "base.txtar"))
	cases, err := filepath.Glob(filepath.Join("testdata", "*.txtar"))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, path := range cases {
		id := strings.TrimSuffix(filepath.Base(path), ".txtar")
		if id == "base" {
			continue
		}
		n++
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			c := lintest.Read(t, path)
			var want []string
			accept := false
			for _, f := range c.Files {
				if f.Name != "want.txt" {
					continue
				}
				for _, line := range strings.Split(string(f.Data), "\n") {
					fields := strings.Fields(line)
					switch {
					case len(fields) == 1 && fields[0] == "accept":
						accept = true
					case len(fields) == 3 && fields[0] == "error":
						want = append(want, fields[1]+" "+fields[2])
					case len(fields) > 0:
						t.Fatalf("bad want line %q", line)
					}
				}
			}
			if accept == (len(want) > 0) {
				t.Fatalf("want.txt lists accept, or errors, not both")
			}
			sort.Strings(want)
			got, out := Lint(t, lintest.Write(t, base, c))
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Errorf("diagnostics\n got: %v\nwant: %v\noutput:\n%s", got, want, out)
			}
		})
	}
	if n < 50 {
		t.Errorf("only %d cases", n)
	}
}
