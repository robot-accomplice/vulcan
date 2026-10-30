// T14: a table of allowed wires. The edges come from main: the argument of a constructor call is
// a local variable that an earlier constructor call defined.
package main

import (
	"go/ast"
	"go/types"
	"path"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var allowed = map[string]bool{"inventory>store": true, "billing>payments": true, "billing>store": true, "events>store": true,
	"events>webhooks": true, "orders>inventory": true, "orders>billing": true, "orders>store": true, "orders>events": true,
	"purchasing>inventory": true, "purchasing>store": true, "purchasing>events": true, "web>inventory": true, "web>orders": true,
	"web>purchasing": true}

var A = &analysis.Analyzer{Name: "wires", Doc: "allowed wires", Run: run}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}
	owner := map[types.Object]string{}
	ctor := func(c *ast.CallExpr) string {
		if s, ok := c.Fun.(*ast.SelectorExpr); ok {
			if fn, ok := pass.TypesInfo.Uses[s.Sel].(*types.Func); ok && fn.Pkg() != nil && strings.HasPrefix(fn.Pkg().Path(), "stockroom/") {
				return path.Base(fn.Pkg().Path())
			}
		}
		return ""
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if as, ok := n.(*ast.AssignStmt); ok && len(as.Rhs) == 1 {
				if c, ok := as.Rhs[0].(*ast.CallExpr); ok {
					if id, ok := as.Lhs[0].(*ast.Ident); ok {
						if o := pass.TypesInfo.Defs[id]; o != nil {
							owner[o] = ctor(c)
						}
					}
				}
			}
			if c, ok := n.(*ast.CallExpr); ok {
				from := ctor(c)
				for _, a := range c.Args {
					if id, ok := a.(*ast.Ident); ok {
						if to := owner[pass.TypesInfo.Uses[id]]; to != "" && from != "" && from != "fmt" && !allowed[from+">"+to] {
							pass.Reportf(a.Pos(), "a wire %s -> %s that the table does not allow", from, to)
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
