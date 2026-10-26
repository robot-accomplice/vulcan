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
