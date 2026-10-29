// Package wiring is analyzer A4 of vulcan-lint: main only wires the leaves.
package wiring

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
	"github.com/robot-accomplice/vulcan/poc/golint/ports"
)

// Analyzer is A4.
var Analyzer = &analysis.Analyzer{
	Name: "wiring",
	Doc:  "check main: it calls only the constructors and the lifecycle methods of the leaves, and the standard library; it passes no nil port; each settings value is a literal with every field, of constants and flag values; a secret field gets no constant; in strict mode, it calls each constructor once (VL0104, VL0603, VL0605, VL0606, VL0607, VL0907, VL0908)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Part == nil || u.Part.Kind != model.Main {
		return nil, nil
	}
	m, info := u.M, pass.TypesInfo
	leafOf := func(p *types.Package) *model.Part {
		if p == nil {
			return nil
		}
		if part := m.PartOf(p.Path()); part != nil && part.IsLeaf() {
			return part
		}
		return nil
	}
	called := map[*types.Func]token.Pos{} // the first call of each constructor
	for _, f := range pass.Files {
		if model.IsTest(pass, f) {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				t := info.TypeOf(x)
				if p, ok := t.(*types.Pointer); ok {
					t = p.Elem()
				}
				if tn := model.NamedOf(t); tn != nil && leafOf(tn.Pkg()) != nil && !settingsValue(t) {
					model.Report(pass, x.Pos(), "VL0908", "main builds a %s with a literal; a leaf's value comes from its constructor, which takes its ports", model.TypeName(pass, t))
				}
			case *ast.CallExpr:
				if tv, ok := info.Types[x.Fun]; ok && tv.IsType() {
					return true
				}
				if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok {
					if b, ok := info.Uses[id].(*types.Builtin); ok {
						if b.Name() == "new" && len(x.Args) == 1 {
							if tn := model.NamedOf(info.TypeOf(x.Args[0])); tn != nil && leafOf(tn.Pkg()) != nil && !settingsValue(tn.Type()) {
								model.Report(pass, x.Pos(), "VL0908", "main builds a %s with new; a leaf's value comes from its constructor", model.TypeName(pass, tn.Type()))
							}
						}
						return true
					}
				}
				fn := model.CalledFunc(info, x)
				if fn == nil || fn.Pkg() == nil || fn.Pkg() == pass.Pkg || !m.InModule(fn.Pkg().Path()) {
					return true // the standard library, other modules, main's own functions, func values
				}
				part := m.PartOf(fn.Pkg().Path())
				sig := fn.Type().(*types.Signature)
				switch {
				case part == nil || !part.IsLeaf():
					model.Report(pass, x.Pos(), "VL0907", "main calls %s; main only wires: it calls the constructors and the lifecycle methods of the leaves, and the standard library", fn.FullName())
				case sig.Recv() == nil && ports.IsConstructor(fn):
					// A second call makes a second instance, such as a second database: an error
					// in strict mode, as K0607 is in Vulcan (parity review R2, item 2).
					if first, ok := called[fn]; ok && m.StrictFor(part) {
						model.Report(pass, x.Pos(), "VL0607", "main calls %s again (first at line %d), so it makes a second instance; call each constructor once and pass its value to every leaf that needs it", fn.FullName(), pass.Fset.Position(first).Line)
					} else if !ok {
						called[fn] = x.Pos()
					}
					args(pass, x, sig)
				case sig.Recv() != nil && Lifecycle(sig):
					args(pass, x, sig)
				case sig.Recv() != nil:
					model.Report(pass, x.Pos(), "VL0907", "main calls the method %s, which is not a lifecycle method (func (ctx context.Context[, s Settings]) error); main only wires, and the leaves do the work", fn.FullName())
				default:
					model.Report(pass, x.Pos(), "VL0907", "main calls %s, which is not a constructor of %s; main only wires", fn.FullName(), part.Dir)
				}
			}
			return true
		})
	}
	return nil, nil
}

// settingsValue reports whether main may build a value of a leaf's type t with a literal: a
// settings struct with at least one field.
func settingsValue(t types.Type) bool {
	st, ok := t.Underlying().(*types.Struct)
	return ok && st.NumFields() > 0 && ports.IsSettings(t)
}

// Lifecycle reports whether a method signature is a lifecycle method: it takes a context and at
// most one settings struct, and returns an error.
func Lifecycle(sig *types.Signature) bool {
	p, r := sig.Params(), sig.Results()
	if r.Len() != 1 || types.TypeString(r.At(0).Type(), nil) != "error" || p.Len() < 1 || p.Len() > 2 {
		return false
	}
	if types.TypeString(p.At(0).Type(), nil) != "context.Context" {
		return false
	}
	return p.Len() == 1 || ports.IsSettings(p.At(1).Type())
}

// args checks the arguments of a constructor or lifecycle call: no nil port, and settings as a
// literal of constants and flag values.
func args(pass *analysis.Pass, call *ast.CallExpr, sig *types.Signature) {
	info := pass.TypesInfo
	params := sig.Params()
	for i, a := range call.Args {
		if i >= params.Len() {
			break
		}
		pt := params.At(i).Type()
		if types.IsInterface(pt) {
			if tv, ok := info.Types[a]; ok && tv.IsNil() {
				model.Report(pass, a.Pos(), "VL0104", "main passes nil for the port %s; every port gets a provider", params.At(i).Name())
			}
			continue
		}
		if !ports.IsSettings(pt) {
			continue
		}
		settings(pass, a, pt)
	}
}

func settings(pass *analysis.Pass, a ast.Expr, t types.Type) {
	info := pass.TypesInfo
	name := model.TypeName(pass, t)
	cl, ok := ast.Unparen(a).(*ast.CompositeLit)
	if !ok {
		model.Report(pass, a.Pos(), "VL0606", "settings %s are a literal in the call, %s{Field: value, ...}, so that each value can be checked", name, name)
		return
	}
	st := t.Underlying().(*types.Struct)
	given := map[string]bool{}
	keyed := len(cl.Elts) == 0
	for i, e := range cl.Elts {
		var field *types.Var
		v := e
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			keyed = true
			id, _ := kv.Key.(*ast.Ident)
			if id != nil {
				given[id.Name] = true
				for j := 0; j < st.NumFields(); j++ {
					if st.Field(j).Name() == id.Name {
						field = st.Field(j)
					}
				}
			}
			v = kv.Value
		} else if i < st.NumFields() {
			field = st.Field(i)
		}
		tv := info.Types[v]
		constant := tv.Value != nil
		if !constant && !flagValue(info, v) {
			model.Report(pass, v.Pos(), "VL0606", "a setting is a constant of its field's type or a flag value (a variable, or flags[\"name\"]); compute nothing in main")
		}
		if field != nil && constant {
			idx := fieldIndex(st, field)
			if tag := reflect.StructTag(st.Tag(idx)); tag.Get("vulcan") == "secret" {
				model.Report(pass, v.Pos(), "VL0605", "field %s of %s is secret, so its value is a flag with no default, never a constant in the code", field.Name(), name)
			}
		}
	}
	if keyed {
		var missing []string
		for i := 0; i < st.NumFields(); i++ {
			if !given[st.Field(i).Name()] {
				missing = append(missing, st.Field(i).Name())
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			model.Report(pass, cl.Pos(), "VL0603", "settings %s have no value for %s; every field gets a value", name, strings.Join(missing, ", "))
		}
	}
}

func fieldIndex(st *types.Struct, f *types.Var) int {
	for i := 0; i < st.NumFields(); i++ {
		if st.Field(i) == f {
			return i
		}
	}
	return -1
}

// flagValue reports whether e is a flag value: a local variable, or an index with a constant
// key, such as flags["db"].
func flagValue(info *types.Info, e ast.Expr) bool {
	switch x := ast.Unparen(e).(type) {
	case *ast.Ident:
		v, ok := info.Uses[x].(*types.Var)
		return ok && v.Pkg() != nil && v.Parent() != v.Pkg().Scope()
	case *ast.IndexExpr:
		tv := info.Types[x.Index]
		id, ok := ast.Unparen(x.X).(*ast.Ident)
		return ok && tv.Value != nil && flagValue(info, id)
	}
	return false
}
