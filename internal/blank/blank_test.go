package blank

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestBlank(t *testing.T) {
	src := `package bank

type Account struct {
	Balance int
}
	invariant Balance >= 0

func Withdraw(a *Account, amt int) (int, error)
    requires a != nil && amt > 0
    ensures  err != nil => a.Balance == old(a.Balance)
{
	a.Balance -= amt
	return a.Balance, nil
}

func One(x int) requires x > 0 { return }

func requiresType() requires { return nil }

type requires *int

func (a *Account) Deposit(x int)
	requires x > 0
{
	a.Balance += x
}
`
	r := File([]byte(src))
	if len(r.Funcs) != 3 || len(r.Types) != 1 {
		t.Fatalf("funcs %d types %d", len(r.Funcs), len(r.Types))
	}
	if r.Funcs[0].Clauses[1].Text != "err != nil => a.Balance == old(a.Balance)" {
		t.Fatalf("clause %q", r.Funcs[0].Clauses[1].Text)
	}
	if r.Funcs[2].Recv != "Account" || r.Funcs[2].Name != "Deposit" {
		t.Fatalf("method %+v", r.Funcs[2])
	}
	if r.Types[0].Clauses[0].Text != "Balance >= 0" {
		t.Fatalf("invariant %q", r.Types[0].Clauses[0].Text)
	}
	out := string(r.Go)
	if strings.Count(out, "\n") != strings.Count(src, "\n") {
		t.Fatalf("line count changed")
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "x.go", out, 0)
	if err != nil {
		t.Fatalf("go parse: %v\n%s", err, out)
	}
	// The statement "a.Balance -= amt" keeps its line and column.
	if !strings.Contains(strings.Split(out, "\n")[11], "\ta.Balance -= amt") {
		t.Fatalf("moved line:\n%s", out)
	}
	_ = f
}

func TestLimit(t *testing.T) {
	src := `package p

const limit = 4

type pager struct {
	fetch func(limit int) []int
	buf   [limit]byte
	limit int
	n     limit2
	Name  string limit 200 ` + "`json:\"name\"`" + `
	Note  string
		limit 300
}

type limit2 int
`
	r := File([]byte(src))
	if len(r.Deferred) != 2 {
		t.Fatalf("deferred: %+v", r.Deferred)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "x.go", r.Go, 0); err != nil {
		t.Fatalf("go parse: %v\n%s", err, r.Go)
	}
}

func TestTruncated(t *testing.T) {
	for _, src := range []string{"package p\n\nfunc (", "package p\n\nfunc F[", "func A()(0000", "package p\ntype T struct {", "package p\nfunc F() requires"} {
		File([]byte(src))
	}
}

func FuzzFile(f *testing.F) {
	f.Add([]byte("package p\n\nfunc F(x int) (int, error)\n\trequires x > 0\n{\n\treturn x, nil\n}\n"))
	f.Add([]byte("package p\ntype T struct {\n\tA int limit 3\n}\n\tinvariant A >= 0\n"))
	f.Fuzz(func(t *testing.T, src []byte) {
		File(src)
	})
}
