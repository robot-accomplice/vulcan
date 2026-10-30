// T7: a type marked //vl:nozero has no zero value in checked code: no var without a value, no new,
// no literal that omits a field, no make with a length, no map read, no named result, no generic
// instantiation. The rule follows the types that hold it.
package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// NoZero marks the type.
type NoZero struct{}

func (*NoZero) AFact() {}

var A = &analysis.Analyzer{Name: "nozero", Doc: "types with no zero value", Run: run, FactTypes: []analysis.Fact{new(NoZero)}}

func holds(pass *analysis.Pass, t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	switch x := types.Unalias(t).(type) {
	case *types.Named:
		var z NoZero
		return pass.ImportObjectFact(x.Origin().Obj(), &z) || holds(pass, x.Underlying(), seen)
	case *types.Array:
		return holds(pass, x.Elem(), seen)
	case *types.Struct:
		for i := 0; i < x.NumFields(); i++ {
			if holds(pass, x.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

func run(pass *analysis.Pass) (any, error) {
	h := func(t types.Type) bool { return holds(pass, t, map[types.Type]bool{}) }
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE && g.Doc != nil {
				for _, c := range g.Doc.List {
					if strings.HasPrefix(c.Text, "//vl:nozero") {
						for _, s := range g.Specs {
							pass.ExportObjectFact(pass.TypesInfo.Defs[s.(*ast.TypeSpec).Name], &NoZero{})
						}
					}
				}
			}
		}
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.ValueSpec:
				if len(x.Values) == 0 {
					for _, id := range x.Names {
						if h(pass.TypesInfo.TypeOf(id)) {
							pass.Reportf(id.Pos(), "var with no value makes a zero of a no-zero type")
						}
					}
				}
			case *ast.CompositeLit:
				t := pass.TypesInfo.TypeOf(x)
				if st, ok := t.Underlying().(*types.Struct); ok && h(t) && len(x.Elts) < st.NumFields() {
					pass.Reportf(x.Pos(), "a literal omits a field of a no-zero type")
				}
			case *ast.CallExpr:
				if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok {
					if b, ok := pass.TypesInfo.Uses[id].(*types.Builtin); ok {
						if b.Name() == "new" && h(pass.TypesInfo.TypeOf(x.Args[0])) {
							pass.Reportf(x.Pos(), "new makes a zero of a no-zero type")
						}
						if b.Name() == "make" {
							if s, ok := pass.TypesInfo.TypeOf(x.Args[0]).Underlying().(*types.Slice); ok && h(s.Elem()) && len(x.Args) > 1 {
								if tv := pass.TypesInfo.Types[x.Args[1]]; tv.Value == nil || tv.Value.String() != "0" {
									pass.Reportf(x.Pos(), "make with a length makes zeros of a no-zero type")
								}
							}
						}
					}
				}
			case *ast.IndexExpr:
				if m, ok := pass.TypesInfo.TypeOf(x.X).Underlying().(*types.Map); ok && h(m.Elem()) {
					pass.Reportf(x.Pos(), "a map read can give a zero of a no-zero type")
				}
			case *ast.FuncType:
				if x.Results != nil {
					for _, fld := range x.Results.List {
						if len(fld.Names) > 0 && h(pass.TypesInfo.TypeOf(fld.Type)) {
							pass.Reportf(fld.Pos(), "a named result starts as a zero of a no-zero type")
						}
					}
				}
			case *ast.Ident:
				if inst, ok := pass.TypesInfo.Instances[x]; ok {
					for i := 0; i < inst.TypeArgs.Len(); i++ {
						if h(inst.TypeArgs.At(i)) {
							pass.Reportf(x.Pos(), "a generic gets a no-zero type; its body can make a zero")
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
