// Package libstate is analyzer A6 of vulcan-lint: the package-level variables of libraries.
package libstate

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A6.
var Analyzer = &analysis.Analyzer{
	Name: "libstate",
	Doc:  "check the package-level variables of libraries: set once, in their initializer or an init of their library, and in strict mode of a type that holds no reference (VL0202, VL0210)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Manifest || u.Part == nil {
		return nil, nil
	}
	m, info := u.M, pass.TypesInfo
	// root returns the variable that an assignment to e assigns: a write to a field of a struct
	// var, or to an element of an array var, writes the var itself. A change through a pointer,
	// a slice, a map, or a method is not tracked (as in the PoC).
	var root func(e ast.Expr) types.Object
	root = func(e ast.Expr) types.Object {
		switch e := ast.Unparen(e).(type) {
		case *ast.Ident:
			return info.Uses[e]
		case *ast.SelectorExpr:
			if v, ok := info.Uses[e.Sel].(*types.Var); ok && v.Pkg() != nil && v.Parent() == v.Pkg().Scope() {
				return v // a qualified package-level var
			}
			if sel := info.Selections[e]; sel != nil && sel.Kind() == types.FieldVal && !sel.Indirect() {
				if _, ptr := info.TypeOf(e.X).Underlying().(*types.Pointer); !ptr {
					return root(e.X)
				}
			}
		case *ast.IndexExpr:
			if _, isArray := info.TypeOf(e.X).Underlying().(*types.Array); isArray {
				return root(e.X)
			}
		}
		return nil
	}
	libVar := func(obj types.Object) bool {
		v, ok := obj.(*types.Var)
		if !ok || v.Pkg() == nil || v.Parent() != v.Pkg().Scope() {
			return false
		}
		p := m.PartOf(v.Pkg().Path())
		return p != nil && p.Kind == model.Library
	}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			inInit := false
			if fd, ok := d.(*ast.FuncDecl); ok {
				inInit = fd.Recv == nil && fd.Name.Name == "init"
			}
			ast.Inspect(d, func(n ast.Node) bool {
				if _, ok := n.(*ast.UnaryExpr); ok {
					return true // taking an address is not an assignment (as in the PoC)
				}
				model.Assigned(n, func(l ast.Expr, _ bool) {
					obj := root(l)
					if !libVar(obj) || inInit && obj.Pkg() == pass.Pkg {
						return
					}
					model.Report(pass, l.Pos(), "VL0202", "library variable %s is assigned after startup; a library variable is set once, in its initializer or in an init function of its library", obj.Name())
				})
				return true
			})
		}
	}
	if u.Part.Kind != model.Library || !m.Strict {
		return nil, nil
	}
	scope := pass.Pkg.Scope()
	for _, name := range scope.Names() {
		v, ok := scope.Lookup(name).(*types.Var)
		if !ok {
			continue
		}
		full := types.TypeString(v.Type(), nil)
		if full == "error" || m.Shared[full] || !model.HoldsReference(v.Type(), map[types.Type]bool{}) {
			continue
		}
		model.Report(pass, v.Pos(), "VL0210", "strict mode: library variable %s has the type %s, which holds a reference, so two leaves could share and change it; use a value type, or list the type in StrictShared of the manifest", v.Name(), full)
	}
	return nil, nil
}
