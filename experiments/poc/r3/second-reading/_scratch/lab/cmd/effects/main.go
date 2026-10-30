// T13: effects inferred through static calls and stored as facts, so each package is checked alone.
// A structured comment //vl:effects a,b states a function's effects; //vl:poly f gives it the effects
// of its parameter f. A call of any other function value is rejected, because the Go type holds no effect.
package main

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// Eff is the inferred effects of a function, and its polymorphic parameter indexes.
type Eff struct {
	Set  []string
	Poly []int
}

func (*Eff) AFact() {}

var base = map[string]string{"net.Dial": "net", "os.Open": "file", "time.Now": "clock", "os.ReadFile": "file"}

var A = &analysis.Analyzer{Name: "effects", Doc: "effects in signatures", Run: run, FactTypes: []analysis.Fact{new(Eff)}}

func add(s []string, more ...string) []string {
	for _, m := range more {
		if !slices.Contains(s, m) {
			s = append(s, m)
		}
	}
	sort.Strings(s)
	return s
}

func directive(fd *ast.FuncDecl, key string) ([]string, bool) {
	if fd.Doc == nil {
		return nil, false
	}
	for _, c := range fd.Doc.List {
		if v, ok := strings.CutPrefix(c.Text, "//vl:"+key); ok {
			return strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' }), true
		}
	}
	return nil, false
}

type info struct {
	fd   *ast.FuncDecl
	fn   *types.Func
	poly map[*types.Var]bool
}

func run(pass *analysis.Pass) (any, error) {
	var fns []*info
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			fn := pass.TypesInfo.Defs[fd.Name].(*types.Func)
			in := &info{fd, fn, map[*types.Var]bool{}}
			e := &Eff{}
			if names, ok := directive(fd, "poly"); ok {
				sig := fn.Type().(*types.Signature)
				for i := 0; i < sig.Params().Len(); i++ {
					if slices.Contains(names, sig.Params().At(i).Name()) {
						in.poly[sig.Params().At(i)] = true
						e.Poly = append(e.Poly, i)
					}
				}
			}
			pass.ExportObjectFact(fn, e)
			fns = append(fns, in)
		}
	}
	var scan func(body ast.Node, in *info, report bool) []string
	scan = func(body ast.Node, in *info, report bool) []string {
		var set []string
		ast.Inspect(body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || pass.TypesInfo.Types[call.Fun].IsType() {
				return true
			}
			var obj types.Object
			switch f := ast.Unparen(call.Fun).(type) {
			case *ast.Ident:
				obj = pass.TypesInfo.Uses[f]
			case *ast.SelectorExpr:
				obj = pass.TypesInfo.Uses[f.Sel]
			case *ast.FuncLit:
				return true
			default:
				if report {
					pass.Reportf(call.Pos(), "a call of a function value whose Go type holds no effect (%s)", types.ExprString(call.Fun))
				}
				return true
			}
			switch o := obj.(type) {
			case *types.Func:
				if b, ok := base[o.FullName()]; ok {
					set = add(set, b)
				}
				var e Eff
				if pass.ImportObjectFact(o.Origin(), &e) {
					set = add(set, e.Set...)
					for _, i := range e.Poly {
						if i >= len(call.Args) {
							continue
						}
						switch a := ast.Unparen(call.Args[i]).(type) {
						case *ast.FuncLit:
							set = add(set, scan(a.Body, in, report)...)
						case *ast.Ident:
							if fo, ok := pass.TypesInfo.Uses[a].(*types.Func); ok {
								var fe Eff
								if pass.ImportObjectFact(fo, &fe) {
									set = add(set, fe.Set...)
								}
								if b, ok := base[fo.FullName()]; ok {
									set = add(set, b)
								}
							}
						}
					}
				}
			case *types.Var:
				if !in.poly[o] && report {
					pass.Reportf(call.Pos(), "a call of the function value %s, whose Go type holds no effect; mark the parameter //vl:poly", o.Name())
				}
			}
			return true
		})
		return set
	}
	for changed := true; changed; {
		changed = false
		for _, in := range fns {
			set := scan(in.fd.Body, in, false)
			var e Eff
			pass.ImportObjectFact(in.fn, &e)
			if m := add(append([]string{}, e.Set...), set...); len(m) != len(e.Set) {
				e.Set = m
				pass.ExportObjectFact(in.fn, &e)
				changed = true
			}
		}
	}
	for _, in := range fns {
		scan(in.fd.Body, in, true)
		var e Eff
		pass.ImportObjectFact(in.fn, &e)
		declared, ok := directive(in.fd, "effects")
		if !ok {
			continue
		}
		for _, x := range e.Set {
			if !slices.Contains(declared, x) {
				pass.Reportf(in.fd.Pos(), "%s has effect %s, but its signature comment states %v", in.fn.Name(), x, declared)
			}
		}
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
