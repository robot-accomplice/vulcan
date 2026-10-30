// T11: a type marked //vl:linear must be used up exactly once on every path. A method marked
// //vl:consumes uses up its receiver. A return moves the value. Flow over go/cfg, as vet's lostcancel does.
package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/singlechecker"
	"golang.org/x/tools/go/cfg"
)

var A = &analysis.Analyzer{Name: "linear", Doc: "linear types", Run: run, Requires: []*analysis.Analyzer{ctrlflow.Analyzer}}

// state: var -> 1 live, 2 used up, 3 maybe used up.
type state map[*types.Var]int

func run(pass *analysis.Pass) (any, error) {
	linear := map[*types.TypeName]bool{}
	consumes := map[*types.Func]bool{}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			switch x := d.(type) {
			case *ast.GenDecl:
				if x.Doc == nil {
					continue
				}
				for _, c := range x.Doc.List {
					if strings.HasPrefix(c.Text, "//vl:linear") {
						for _, s := range x.Specs {
							if ts, ok := s.(*ast.TypeSpec); ok {
								linear[pass.TypesInfo.Defs[ts.Name].(*types.TypeName)] = true
							}
						}
					}
				}
			case *ast.FuncDecl:
				if x.Doc == nil {
					continue
				}
				for _, c := range x.Doc.List {
					if strings.HasPrefix(c.Text, "//vl:consumes") {
						consumes[pass.TypesInfo.Defs[x.Name].(*types.Func)] = true
					}
				}
			}
		}
	}
	isLinear := func(t types.Type) bool {
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
		}
		n, ok := types.Unalias(t).(*types.Named)
		return ok && linear[n.Origin().Obj()]
	}
	cf := pass.ResultOf[ctrlflow.Analyzer].(*ctrlflow.CFGs)
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Body != nil {
				checkFunc(pass, cf.FuncDecl(fd), isLinear, consumes)
			}
		}
	}
	return nil, nil
}

func checkFunc(pass *analysis.Pass, g *cfg.CFG, isLinear func(types.Type) bool, consumes map[*types.Func]bool) {
	if g == nil || len(g.Blocks) == 0 {
		return
	}
	deferred := map[*types.Var]bool{}
	in := map[*cfg.Block]state{g.Blocks[0]: {}}
	seen := map[token.Pos]bool{}
	report := func(p token.Pos, format string, a ...any) {
		if !seen[p] {
			seen[p] = true
			pass.Reportf(p, format, a...)
		}
	}
	transfer := func(b *cfg.Block, s state) state {
		out := state{}
		for k, v := range s {
			out[k] = v
		}
		use := func(id *ast.Ident, consume bool) {
			v, ok := pass.TypesInfo.Uses[id].(*types.Var)
			if !ok || !isLinear(v.Type()) || out[v] == 0 {
				return
			}
			if out[v] != 1 {
				report(id.Pos(), "%s is used after it was used up", v.Name())
			}
			if consume {
				out[v] = 2
			}
		}
		for _, n := range b.Nodes {
			handled := map[*ast.Ident]bool{}
			ast.Inspect(n, func(x ast.Node) bool {
				switch c := x.(type) {
				case *ast.FuncLit:
					ast.Inspect(c, func(y ast.Node) bool {
						if id, ok := y.(*ast.Ident); ok {
							if v, ok := pass.TypesInfo.Uses[id].(*types.Var); ok && isLinear(v.Type()) && out[v] != 0 {
								report(id.Pos(), "a linear value %s is captured by a closure", v.Name())
							}
						}
						return true
					})
					return false
				case *ast.CallExpr:
					if sel, ok := ast.Unparen(c.Fun).(*ast.SelectorExpr); ok {
						if fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func); ok && consumes[fn] {
							if id, ok := sel.X.(*ast.Ident); ok {
								handled[id] = true
								use(id, true)
							}
						}
					}
				case *ast.DeferStmt:
					// a deferred use-up runs at exit: the value stays usable, and no path forgets it.
					if sel, ok := ast.Unparen(c.Call.Fun).(*ast.SelectorExpr); ok {
						if fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func); ok && consumes[fn] {
							if id, ok := sel.X.(*ast.Ident); ok {
								if v, ok := pass.TypesInfo.Uses[id].(*types.Var); ok && isLinear(v.Type()) {
									deferred[v] = true
								}
							}
						}
					}
					return false
				case *ast.AssignStmt:
					for _, l := range c.Lhs {
						if id, ok := l.(*ast.Ident); ok {
							if v, ok := pass.TypesInfo.Defs[id].(*types.Var); ok && isLinear(v.Type()) {
								out[v] = 1
							}
						}
					}
				case *ast.ReturnStmt:
					for _, r := range c.Results {
						if id, ok := ast.Unparen(r).(*ast.Ident); ok {
							handled[id] = true
							use(id, true)
						}
					}
				case *ast.Ident:
					if !handled[c] {
						use(c, false)
					}
				}
				return true
			})
		}
		return out
	}
	join := func(a, b state) state {
		r := state{}
		for k, v := range a {
			if w, ok := b[k]; ok && w != v {
				r[k] = 3
			} else {
				r[k] = v
			}
		}
		for k, v := range b {
			if _, ok := a[k]; !ok {
				r[k] = v
			}
		}
		return r
	}
	for i := 0; i < len(g.Blocks)*4; i++ {
		for _, b := range g.Blocks {
			s, ok := in[b]
			if !ok || !b.Live {
				continue
			}
			out := transfer(b, s)
			if len(b.Succs) == 0 {
				for v, st := range out {
					if (st == 1 || st == 3) && !deferred[v] {
						pos := v.Pos()
						if b.Return() != nil {
							pos = b.Return().Pos()
						}
						report(pos, "%s is not used up on this path", v.Name())
					}
				}
			}
			for _, t := range b.Succs {
				if p, ok := in[t]; ok {
					in[t] = join(p, out)
				} else {
					in[t] = out
				}
			}
		}
	}
}

func main() { singlechecker.Main(A) }
