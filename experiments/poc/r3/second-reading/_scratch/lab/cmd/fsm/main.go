// T10: a table of legal transitions, checked at each write of Status. The known state comes from
// the guards before the write: if o.Status != A && o.Status != B { return }.
package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var table = map[string][]string{"Pending": {"Paid", "Cancelled"}, "Paid": {"Shipped", "Cancelled"}, "Shipped": {}, "Cancelled": {}}
var initial = map[string]bool{"Pending": true}

var A = &analysis.Analyzer{Name: "fsm", Doc: "declared state machine", Run: run}

func isStatus(pass *analysis.Pass, e ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(e)
	return t != nil && strings.HasSuffix(t.String(), "shop.Status")
}

func constName(pass *analysis.Pass, e ast.Expr) string {
	if s, ok := e.(*ast.SelectorExpr); ok {
		if c, ok := pass.TypesInfo.Uses[s.Sel].(*types.Const); ok {
			return c.Name()
		}
	}
	return ""
}

func guard(pass *analysis.Pass, s *ast.IfStmt) []string {
	if len(s.Body.List) == 0 {
		return nil
	}
	if _, ok := s.Body.List[len(s.Body.List)-1].(*ast.ReturnStmt); !ok {
		return nil
	}
	var out []string
	var walk func(e ast.Expr) bool
	walk = func(e ast.Expr) bool {
		b, ok := ast.Unparen(e).(*ast.BinaryExpr)
		if !ok {
			return false
		}
		if b.Op == token.LAND {
			return walk(b.X) && walk(b.Y)
		}
		if b.Op == token.NEQ && isStatus(pass, b.X) {
			if c := constName(pass, b.Y); c != "" {
				out = append(out, c)
				return true
			}
		}
		return false
	}
	if walk(s.Cond) {
		return out
	}
	return nil
}

func run(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			var known []string
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.IfStmt:
					if g := guard(pass, x); g != nil {
						known = g
					}
				case *ast.CompositeLit:
					for _, e := range x.Elts {
						if kv, ok := e.(*ast.KeyValueExpr); ok && isStatus(pass, kv.Value) {
							if c := constName(pass, kv.Value); !initial[c] {
								pass.Reportf(kv.Pos(), "a new value starts in state %q, which is not an initial state", c)
							}
						}
					}
				case *ast.AssignStmt:
					for i, l := range x.Lhs {
						if !isStatus(pass, l) || x.Tok != token.ASSIGN || i >= len(x.Rhs) {
							continue
						}
						to := constName(pass, x.Rhs[i])
						if to == "" {
							pass.Reportf(l.Pos(), "a write of a status that is not a declared state")
							continue
						}
						if known == nil {
							pass.Reportf(l.Pos(), "a write of state %s with no known state before it", to)
							continue
						}
						for _, from := range known {
							legal := false
							for _, t := range table[from] {
								legal = legal || t == to
							}
							if !legal {
								pass.Reportf(l.Pos(), "illegal transition %s -> %s", from, to)
							}
						}
					}
				}
				return true
			})
		}
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
