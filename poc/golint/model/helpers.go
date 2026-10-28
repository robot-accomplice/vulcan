package model

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Report reports a diagnostic with its code. The message starts with the code, so that the
// output of the command reads "file:line:col: VL0101 ...".
func Report(pass *analysis.Pass, pos token.Pos, code, format string, args ...any) {
	pass.Report(analysis.Diagnostic{Pos: pos, Category: code, Message: code + " " + fmt.Sprintf(format, args...)})
}

// IsTest reports whether a file is a test file.
func IsTest(pass *analysis.Pass, f *ast.File) bool {
	return strings.HasSuffix(pass.Fset.PositionFor(f.Package, false).Filename, "_test.go")
}

// FileName returns the base name of a file.
func FileName(pass *analysis.Pass, f *ast.File) string {
	name := pass.Fset.PositionFor(f.Package, false).Filename
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// IsStd reports whether an import path is in the standard library.
func IsStd(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".") && path != "C"
}

// CalledFunc returns the function or method that a call names, or nil.
func CalledFunc(info *types.Info, call *ast.CallExpr) *types.Func {
	fun := ast.Unparen(call.Fun)
	if ix, ok := fun.(*ast.IndexExpr); ok {
		fun = ix.X
	}
	if ix, ok := fun.(*ast.IndexListExpr); ok {
		fun = ix.X
	}
	var id *ast.Ident
	switch f := fun.(type) {
	case *ast.Ident:
		id = f
	case *ast.SelectorExpr:
		id = f.Sel
	default:
		return nil
	}
	fn, _ := info.Uses[id].(*types.Func)
	return fn
}

// NamedOf returns the type name of t or *t, or nil.
func NamedOf(t types.Type) *types.TypeName {
	if t == nil {
		return nil
	}
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		t = p.Elem()
	}
	if n, ok := types.Unalias(t).(*types.Named); ok {
		return n.Origin().Obj()
	}
	return nil
}

// HoldsReference reports whether a value of type t holds a pointer, slice, map, channel,
// function, or interface (the walk of the PoC's holdsReference).
func HoldsReference(t types.Type, seen map[types.Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	if _, ok := t.(*types.TypeParam); ok {
		return false
	}
	switch u := t.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return true
	case *types.Array:
		return HoldsReference(u.Elem(), seen)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if HoldsReference(u.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

// Contract names the function of the contract package that a call calls, or "".
func Contract(info *types.Info, call *ast.CallExpr, contractPath string) string {
	fn := CalledFunc(info, call)
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != contractPath {
		return ""
	}
	return fn.Name()
}

// Prologue is the contract of a function: its first statements, while they are contract
// statements. Each is one of
//
//	contract.Requires(cond)
//	x := contract.Old(expr)
//	defer contract.Ensures(func() bool { return cond })
//	defer contract.Keep(recv)
type Prologue struct {
	Stmts    int // the number of contract statements
	Requires []*ast.CallExpr
	Olds     []*ast.CallExpr
	Ensures  []*ast.CallExpr
	Keeps    []*ast.CallExpr
	Calls    map[*ast.CallExpr]bool // every call above
}

// PrologueOf reads the prologue of a function body.
func PrologueOf(info *types.Info, body *ast.BlockStmt, contractPath string) *Prologue {
	p := &Prologue{Calls: map[*ast.CallExpr]bool{}}
	if body == nil {
		return p
	}
loop:
	for _, s := range body.List {
		var call *ast.CallExpr
		var want []string
		switch s := s.(type) {
		case *ast.ExprStmt:
			call, _ = s.X.(*ast.CallExpr)
			want = []string{"Requires"}
		case *ast.AssignStmt:
			if s.Tok == token.DEFINE && len(s.Lhs) == 1 && len(s.Rhs) == 1 {
				call, _ = s.Rhs[0].(*ast.CallExpr)
				want = []string{"Old"}
			}
		case *ast.DeferStmt:
			call = s.Call
			want = []string{"Ensures", "Keep"}
		}
		if call == nil {
			break loop
		}
		name := Contract(info, call, contractPath)
		if !contains(want, name) {
			break loop
		}
		p.Stmts++
		p.Calls[call] = true
		switch name {
		case "Requires":
			p.Requires = append(p.Requires, call)
		case "Old":
			p.Olds = append(p.Olds, call)
		case "Ensures":
			p.Ensures = append(p.Ensures, call)
		case "Keep":
			p.Keeps = append(p.Keeps, call)
		}
	}
	return p
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// EnsuresCond returns the condition of defer contract.Ensures(func() bool { return cond }), or
// nil when the argument is not of that form.
func EnsuresCond(call *ast.CallExpr) ast.Expr {
	if len(call.Args) != 1 {
		return nil
	}
	lit, ok := ast.Unparen(call.Args[0]).(*ast.FuncLit)
	if !ok || len(lit.Body.List) != 1 {
		return nil
	}
	ret, ok := lit.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return nil
	}
	return ret.Results[0]
}

// SingleReturn returns the expression of a body that is one return statement, or nil.
func SingleReturn(body *ast.BlockStmt) ast.Expr {
	if body == nil || len(body.List) != 1 {
		return nil
	}
	ret, ok := body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return nil
	}
	return ret.Results[0]
}

// InvariantMethod returns the method Invariant() bool of a named type of the module, or nil.
func InvariantMethod(m *Manifest, tn *types.TypeName) *types.Func {
	if tn == nil || tn.Pkg() == nil || !m.InModule(tn.Pkg().Path()) {
		return nil
	}
	obj, _, _ := types.LookupFieldOrMethod(types.NewPointer(tn.Type()), false, tn.Pkg(), "Invariant")
	fn, ok := obj.(*types.Func)
	if !ok {
		return nil
	}
	sig := fn.Type().(*types.Signature)
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 || !types.Identical(sig.Results().At(0).Type(), types.Typ[types.Bool]) {
		return nil
	}
	// Only a method of the type itself, not one promoted from an embedded field.
	if r := NamedOf(sig.Recv().Type()); r != tn {
		return nil
	}
	return fn
}

// FuncDecls maps each function and method of the files to its declaration.
func FuncDecls(info *types.Info, files []*ast.File) map[*types.Func]*ast.FuncDecl {
	out := map[*types.Func]*ast.FuncDecl{}
	for _, f := range files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				if fn, ok := info.Defs[fd.Name].(*types.Func); ok {
					out[fn] = fd
				}
			}
		}
	}
	return out
}

// Assigned calls f with each expression that a statement assigns, or whose address it takes,
// in the node n: the left sides of = and op=, ++ and --, range with =, and &x.
func Assigned(n ast.Node, f func(e ast.Expr, addr bool)) {
	switch x := n.(type) {
	case *ast.AssignStmt:
		if x.Tok != token.DEFINE {
			for _, l := range x.Lhs {
				f(l, false)
			}
		}
	case *ast.IncDecStmt:
		f(x.X, false)
	case *ast.RangeStmt:
		if x.Tok == token.ASSIGN {
			for _, l := range []ast.Expr{x.Key, x.Value} {
				if l != nil {
					f(l, false)
				}
			}
		}
	case *ast.UnaryExpr:
		if x.Op == token.AND {
			f(x.X, true)
		}
	}
}

// TypeName writes a type as the code of the pass names it: shop.Stock, or Stock in its own
// package.
func TypeName(pass *analysis.Pass, t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p == pass.Pkg {
			return ""
		}
		return p.Name()
	})
}
