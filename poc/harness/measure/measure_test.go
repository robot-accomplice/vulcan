package measure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robot-accomplice/vulcan/poc/harness/detect"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const block = `a := 1
b := a + 2
c := b * 3
d := c - 4
e := d / 5
f := e % 6
`

func TestCloneLines(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "api/x/x.go", "package x\n\nfunc A() {\n"+block+"}\n")
	if n := CloneLines(ws); n != 0 {
		t.Fatalf("one copy: %d", n)
	}
	write(t, ws, "api/y/y.go", "package y\n\n// B copies A.\nfunc B() {\n"+block+"}\n")
	write(t, ws, "scratch/z.go", "package z\n\nfunc C() {\n"+block+"}\n")
	if n := CloneLines(ws); n != 12 {
		t.Fatalf("two copies in the product: %d, want 12", n)
	}
}

func TestMaskedErrors(t *testing.T) {
	ws := t.TempDir()
	write(t, ws, "go.mod", "module stockroom\n\ngo 1.22\n")
	write(t, ws, "api/x/x.go", `package x

import (
	"errors"
	"fmt"
	"os"
)

func f() error          { return errors.New("x") }
func g() (int, error)   { return 0, nil }

// Use drops errors three ways, and checks one.
func Use() {
	f()
	_ = f()
	_, _ = g()
	n, _ := g()
	fmt.Println(n)
	if err := f(); err != nil {
		os.Exit(1)
	}
	defer f()
}
`)
	write(t, ws, "api/x/x_test.go", "package x\n\nfunc helper() { f() }\n")
	s, err := detect.Load("P", ws)
	if err != nil {
		t.Fatal(err)
	}
	if n := MaskedErrors(s); n != 4 {
		t.Fatalf("masked %d, want 4", n)
	}
}
