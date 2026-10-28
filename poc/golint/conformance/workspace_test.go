package conformance

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"

	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"
)

// TestWorkspace checks arm L's workspace with its own linter: no diagnostic.
func TestWorkspace(t *testing.T) {
	if got, out := Lint(t, lintest.Workspace()); len(got) > 0 {
		t.Errorf("vulcan-lint on poc/app/golint: %v\n%s", got, out)
	}
}

// TestReadme checks that arm L's README is arm P's, but for the check command and one line that
// points to the primer (plan-r2, section 2).
func TestReadme(t *testing.T) {
	p, err := os.ReadFile(filepath.Join(lintest.Workspace(), "..", "go", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	l, err := os.ReadFile(filepath.Join(lintest.Workspace(), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	norm := func(b []byte) string { return strings.ReplaceAll(string(b), "\r\n", "\n") }
	want := strings.Replace(norm(p), "go vet ./... && go test ./...\n```\n",
		"go vet ./... && vulcan-lint ./... && go test ./...\n```\n\n"+
			"`PRIMER.md` explains the architecture and the contracts that `vulcan-lint` checks, and each of its codes.\n", 1)
	if want == norm(p) || norm(l) != want {
		t.Errorf("poc/app/golint/README.md is not poc/app/go/README.md with the check command of arm L and the line of the primer")
	}
}

// runtimeTest checks the contract package when the code runs: the Go form of the run cases C01
// (requires), C03 (ensures), C18 (a literal that breaks its invariant), and C12b (a method call
// that breaks an invariant), and that a check does not hide a panic of the body.
const runtimeTest = `
-- lib/contract/contract_test.go --
package contract

import (
	"errors"
	"strings"
	"testing"
)

type window struct{ Lo, Hi int }

func (w window) Invariant() bool { return w.Lo <= w.Hi }

func (w *window) shrink(n int) {
	defer Keep(w)
	w.Hi -= n
}

func charge(amt int) int {
	Requires(amt > 0)
	return amt
}

func pay(amt int) (got int) {
	defer Ensures(func() bool { return got == amt })
	return amt - 1
}

func boom() (got int) {
	defer Ensures(func() bool { return got == 1 })
	panic("boom")
}

func failure(t *testing.T, f func()) (v any) {
	t.Helper()
	defer func() { v = recover() }()
	f()
	t.Fatal("no panic")
	return nil
}

func want(t *testing.T, v any, kind, says string) {
	t.Helper()
	var f *Failure
	if err, ok := v.(error); !ok || !errors.As(err, &f) || f.Kind != kind || !strings.Contains(f.Error(), says) {
		t.Fatalf("panic %v, want a %s failure that says %q", v, kind, says)
	}
}

func TestRequires(t *testing.T) {
	want(t, failure(t, func() { charge(0) }), "requires", "contract.charge requires is false")
	if charge(1) != 1 {
		t.Fatal("charge(1)")
	}
}

func TestEnsures(t *testing.T) {
	want(t, failure(t, func() { pay(5) }), "ensures", "contract.pay ensures is false")
}

func TestEnsuresKeepsPanic(t *testing.T) {
	if v := failure(t, func() { boom() }); v != "boom" {
		t.Fatalf("panic %v, want boom", v)
	}
}

func TestValid(t *testing.T) {
	want(t, failure(t, func() { _ = Valid(window{Lo: 2, Hi: 1}) }), "invariant", "the invariant of contract.window")
	_ = Valid(window{Lo: 1, Hi: 2})
}

func TestKeep(t *testing.T) {
	w := Valid(&window{Lo: 0, Hi: 1})
	w.shrink(1)
	want(t, failure(t, func() { w.shrink(1) }), "invariant", "after contract.(*window).shrink")
	var nilw *window
	Keep(nilw)
}

func TestOld(t *testing.T) {
	if Old(3) != 3 {
		t.Fatal("Old")
	}
}
`

func TestContractRuntime(t *testing.T) {
	dir := lintest.Write(t, txtar.Parse([]byte(runtimeTest)))
	cmd := exec.Command("go", "test", "./lib/contract")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=", "GOWORK=off", "GOPROXY=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test ./lib/contract: %v\n%s", err, out)
	}
}
