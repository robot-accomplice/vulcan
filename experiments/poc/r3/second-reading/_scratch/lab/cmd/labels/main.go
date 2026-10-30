// T9: a defined type marked //vl:label. A labeled value reaches no interface and no conversion to
// an unlabeled type; no index, range, compare, or len; no generic of another package. Only a package
// named declassify may convert.
package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// Labeled marks a labeled type across packages.
type Labeled struct{}

func (*Labeled) AFact() {}

var A = &analysis.Analyzer{Name: "labels", Doc: "labels on values", Run: run, FactTypes: []analysis.Fact{new(Labeled)}}

func holds(pass *analysis.Pass, t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}
	seen[t] = true
	switch x := types.Unalias(t).(type) {
	case *types.Named:
		var l Labeled
		if pass.ImportObjectFact(x.Origin().Obj(), &l) {
			return true
		}
		return holds(pass, x.Underlying(), seen)
	case *types.Pointer:
		return holds(pass, x.Elem(), seen)
	case *types.Slice:
		return holds(pass, x.Elem(), seen)
	case *types.Array:
		return holds(pass, x.Elem(), seen)
	case *types.Map:
		return holds(pass, x.Key(), seen) || holds(pass, x.Elem(), seen)
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
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE && g.Doc != nil {
				for _, c := range g.Doc.List {
					if strings.HasPrefix(c.Text, "//vl:label") {
						for _, s := range g.Specs {
							pass.ExportObjectFact(pass.TypesInfo.Defs[s.(*ast.TypeSpec).Name], &Labeled{})
						}
					}
				}
			}
		}
	}
	if pass.Pkg.Name() == "declassify" {
		return nil, nil
	}
	lab := func(e ast.Expr) bool { return holds(pass, pass.TypesInfo.TypeOf(e), map[types.Type]bool{}) }
	toIface := func(e ast.Expr, t types.Type) {
		if t != nil && types.IsInterface(t) && lab(e) {
			pass.Reportf(e.Pos(), "a labeled value goes into interface type %s, so it can reach a log, a response, or an external", t)
		}
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CallExpr:
				tv := pass.TypesInfo.Types[x.Fun]
				if tv.IsType() {
					if len(x.Args) == 1 && lab(x.Args[0]) && !holds(pass, tv.Type, map[types.Type]bool{}) {
						pass.Reportf(x.Pos(), "a conversion removes the label; only package declassify may do it")
					}
					return true
				}
				if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok {
					if b, ok := pass.TypesInfo.Uses[id].(*types.Builtin); ok && (b.Name() == "len" || b.Name() == "cap") && lab(x.Args[0]) {
						pass.Reportf(x.Pos(), "%s of a labeled value leaks a fact about it", b.Name())
					}
				}
				if sig, ok := pass.TypesInfo.TypeOf(x.Fun).(*types.Signature); ok {
					for i, a := range x.Args {
						var pt types.Type
						if sig.Variadic() && i >= sig.Params().Len()-1 {
							if sl, ok := sig.Params().At(sig.Params().Len() - 1).Type().(*types.Slice); ok {
								pt = sl.Elem()
							}
						} else if i < sig.Params().Len() {
							pt = sig.Params().At(i).Type()
						}
						toIface(a, pt)
					}
				}
			case *ast.AssignStmt:
				if len(x.Lhs) == len(x.Rhs) {
					for i, l := range x.Lhs {
						toIface(x.Rhs[i], pass.TypesInfo.TypeOf(l))
					}
				}
			case *ast.IndexExpr:
				if lab(x.X) && !holds(pass, pass.TypesInfo.TypeOf(x), map[types.Type]bool{}) {
					pass.Reportf(x.Pos(), "an index of a labeled value loses the label")
				}
			case *ast.RangeStmt:
				if lab(x.X) {
					pass.Reportf(x.Pos(), "a range over a labeled value loses the label")
				}
			case *ast.BinaryExpr:
				if (x.Op == token.EQL || x.Op == token.NEQ || x.Op == token.LSS || x.Op == token.GTR) && (lab(x.X) || lab(x.Y)) {
					pass.Reportf(x.Pos(), "a comparison of a labeled value is an implicit flow")
				}
			case *ast.Ident:
				if inst, ok := pass.TypesInfo.Instances[x]; ok {
					for i := 0; i < inst.TypeArgs.Len(); i++ {
						if holds(pass, inst.TypeArgs.At(i), map[types.Type]bool{}) {
							if obj := pass.TypesInfo.Uses[x]; obj != nil && obj.Pkg() != pass.Pkg {
								pass.Reportf(x.Pos(), "a generic of another package gets a labeled type; it could log it")
							}
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
