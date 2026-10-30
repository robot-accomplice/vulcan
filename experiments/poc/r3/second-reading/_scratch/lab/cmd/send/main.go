// T20: what may cross between goroutines. A type crosses when it holds no reference, is marked
// //vl:send, or is on a table. A go statement may not capture a variable that the function writes;
// a captured variable, a go argument, and a channel element must cross.
package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// Send marks a type.
type Send struct{}

func (*Send) AFact() {}

var table = map[string]bool{"sync.Mutex": true, "sync.WaitGroup": true, "net.Listener": true, "*net/http.Server": true}

var A = &analysis.Analyzer{Name: "send", Doc: "race freedom by type", Run: run, FactTypes: []analysis.Fact{new(Send)}}

func crosses(pass *analysis.Pass, t types.Type, seen map[types.Type]bool) bool {
	if seen[t] || table[t.String()] {
		return true
	}
	seen[t] = true
	switch x := types.Unalias(t).(type) {
	case *types.Basic:
		return true
	case *types.Named:
		var s Send
		if pass.ImportObjectFact(x.Origin().Obj(), &s) {
			return true
		}
		return crosses(pass, x.Underlying(), seen)
	case *types.Chan:
		return crosses(pass, x.Elem(), seen)
	case *types.Pointer:
		if n, ok := types.Unalias(x.Elem()).(*types.Named); ok {
			var s Send
			return pass.ImportObjectFact(n.Origin().Obj(), &s)
		}
		return false
	case *types.Struct:
		for i := 0; i < x.NumFields(); i++ {
			if !crosses(pass, x.Field(i).Type(), seen) {
				return false
			}
		}
		return true
	case *types.Array:
		return crosses(pass, x.Elem(), seen)
	}
	return false
}

func run(pass *analysis.Pass) (any, error) {
	c := func(t types.Type) bool {
		return t.String() == "error" || crosses(pass, t, map[types.Type]bool{})
	}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE && g.Doc != nil {
				for _, cm := range g.Doc.List {
					if strings.HasPrefix(cm.Text, "//vl:send") {
						for _, s := range g.Specs {
							pass.ExportObjectFact(pass.TypesInfo.Defs[s.(*ast.TypeSpec).Name], &Send{})
						}
					}
				}
			}
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			written := map[*types.Var]int{}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.AssignStmt:
					for _, l := range x.Lhs {
						if id, ok := l.(*ast.Ident); ok {
							if v, ok := pass.TypesInfo.Uses[id].(*types.Var); ok {
								written[v]++
							}
						}
					}
				case *ast.IncDecStmt:
					if id, ok := x.X.(*ast.Ident); ok {
						if v, ok := pass.TypesInfo.Uses[id].(*types.Var); ok {
							written[v]++
						}
					}
				}
				return true
			})
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.SendStmt:
					if !c(pass.TypesInfo.TypeOf(x.Value)) {
						pass.Reportf(x.Pos(), "a value of type %s crosses a channel, but the type does not cross", pass.TypesInfo.TypeOf(x.Value))
					}
				case *ast.GoStmt:
					for _, a := range x.Call.Args {
						if !c(pass.TypesInfo.TypeOf(a)) {
							pass.Reportf(a.Pos(), "a go argument of type %s does not cross", pass.TypesInfo.TypeOf(a))
						}
					}
					if fl, ok := x.Call.Fun.(*ast.FuncLit); ok {
						ast.Inspect(fl.Body, func(y ast.Node) bool {
							if id, ok := y.(*ast.Ident); ok {
								if v, ok := pass.TypesInfo.Uses[id].(*types.Var); ok && !v.IsField() && v.Parent() != pass.Pkg.Scope() && v.Pos() < fl.Pos() {
									if written[v] > 0 {
										pass.Reportf(id.Pos(), "a go statement captures %s, which this function writes", v.Name())
									} else if !c(v.Type()) {
										pass.Reportf(id.Pos(), "a go statement captures %s of type %s, which does not cross", v.Name(), v.Type())
									}
								}
							}
							return true
						})
					}
				}
				return true
			})
		}
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
