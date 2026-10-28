// Package invariants is analyzer A8 of vulcan-lint: the types with an Invariant method.
package invariants

import (
	"bytes"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/robot-accomplice/vulcan/poc/golint/contracts"
	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A8.
var Analyzer = &analysis.Analyzer{
	Name: "invariants",
	Doc:  "check the types with an invariant (func (x T) Invariant() bool): one pure return that the zero value keeps, defer contract.Keep in each pointer method, contract.Valid on each value that the code builds, and no write to a field outside the type's methods (VL0503, VL0506, VL0507, VL0508, VL0513, VL0904, VL0905)",
	Run:  run,
}

type checker struct {
	pass *analysis.Pass
	u    *model.Unit
	inv  map[*types.TypeName]bool
}

// is reports whether tn is a struct type with an invariant.
func (c *checker) is(tn *types.TypeName) bool {
	if tn == nil {
		return false
	}
	if v, ok := c.inv[tn]; ok {
		return v
	}
	_, isStruct := tn.Type().Underlying().(*types.Struct)
	v := isStruct && model.InvariantMethod(c.u.M, tn) != nil
	c.inv[tn] = v
	return v
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Manifest || u.Part == nil || u.Contract {
		return nil, nil
	}
	c := &checker{pass: pass, u: u, inv: map[*types.TypeName]bool{}}
	c.declarations()
	c.literals()
	for _, f := range pass.Files {
		c.fieldWrites(f)
	}
	return nil, nil
}

// declarations checks each Invariant method and the pointer methods of its type.
func (c *checker) declarations() {
	pass, info := c.pass, c.pass.TypesInfo
	cp := c.u.M.ContractPath()
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			rt := info.TypeOf(fd.Recv.List[0].Type)
			tn := model.NamedOf(rt)
			if tn == nil {
				continue
			}
			_, ptr := types.Unalias(rt).(*types.Pointer)
			if fd.Name.Name == "Invariant" {
				fn, _ := info.Defs[fd.Name].(*types.Func)
				sig, _ := fn.Type().(*types.Signature)
				if sig == nil || sig.Params().Len() != 0 || sig.Results().Len() != 1 || !types.Identical(sig.Results().At(0).Type(), types.Typ[types.Bool]) {
					continue
				}
				if _, isStruct := tn.Type().Underlying().(*types.Struct); !isStruct {
					model.Report(pass, fd.Name.Pos(), "VL0513", "an invariant is a method of a struct type; %s is not a struct", tn.Name())
					continue
				}
				if ptr {
					model.Report(pass, fd.Name.Pos(), "VL0513", "declare Invariant with a value receiver, func (x %s) Invariant() bool, so that %s and *%s both have it", tn.Name(), tn.Name(), tn.Name())
				}
				expr := model.SingleReturn(fd.Body)
				if expr == nil {
					model.Report(pass, fd.Name.Pos(), "VL0503", "the body of Invariant is one return statement of a condition on the fields")
					continue
				}
				contracts.Pure(pass, expr, "Invariant")
				c.zero(fd, tn, expr)
				continue
			}
			if !ptr || !c.is(tn) || tn.Pkg() != pass.Pkg {
				continue
			}
			var recv types.Object
			if names := fd.Recv.List[0].Names; len(names) == 1 {
				recv = info.Defs[names[0]]
			}
			pro := model.PrologueOf(info, fd.Body, cp)
			kept := false
			for _, k := range pro.Keeps {
				if len(k.Args) == 1 {
					if id, ok := ast.Unparen(k.Args[0]).(*ast.Ident); ok && recv != nil && info.Uses[id] == recv {
						kept = true
					}
				}
			}
			if !kept {
				name := "recv"
				if recv != nil && recv.Name() != "_" {
					name = recv.Name()
				}
				model.Report(pass, fd.Name.Pos(), "VL0904", "%s.%s has a pointer receiver, so it can change a %s; write defer contract.Keep(%s) among its first statements, so that the invariant is checked when it returns", tn.Name(), fd.Name.Name, tn.Name(), name)
			}
		}
	}
	// A Keep on anything but the pointer receiver of a method of an invariant type.
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			pro := model.PrologueOf(info, fd.Body, cp)
			for _, k := range pro.Keeps {
				ok := false
				if fd.Recv != nil && len(fd.Recv.List) == 1 && len(fd.Recv.List[0].Names) == 1 && len(k.Args) == 1 {
					rt := info.TypeOf(fd.Recv.List[0].Type)
					_, ptr := types.Unalias(rt).(*types.Pointer)
					id, isID := ast.Unparen(k.Args[0]).(*ast.Ident)
					ok = ptr && isID && c.is(model.NamedOf(rt)) && info.Uses[id] == info.Defs[fd.Recv.List[0].Names[0]]
				}
				if !ok {
					model.Report(pass, k.Pos(), "VL0904", "defer contract.Keep(recv) belongs in a method with a pointer receiver of a type with an invariant, and takes that receiver")
				}
			}
		}
	}
}

// zero rejects an invariant that the zero value breaks (VL0506). It decides an invariant whose
// fields are numbers, strings, or booleans, as the PoC does.
func (c *checker) zero(fd *ast.FuncDecl, tn *types.TypeName, expr ast.Expr) {
	pass, info := c.pass, c.pass.TypesInfo
	names := fd.Recv.List[0].Names
	if len(names) != 1 {
		return
	}
	recv := info.Defs[names[0]]
	st := tn.Type().Underlying().(*types.Struct)
	qual := func(p *types.Package) string {
		if p == pass.Pkg {
			return ""
		}
		return p.Name()
	}
	zero := map[string]string{}
	for i := 0; i < st.NumFields(); i++ {
		ft := st.Field(i).Type()
		b, ok := ft.Underlying().(*types.Basic)
		if !ok {
			continue
		}
		name := types.TypeString(ft, qual)
		switch {
		case b.Info()&types.IsNumeric != 0:
			zero[st.Field(i).Name()] = name + "(0)"
		case b.Info()&types.IsString != 0:
			zero[st.Field(i).Name()] = name + `("")`
		case b.Info()&types.IsBoolean != 0:
			zero[st.Field(i).Name()] = name + "(false)"
		}
	}
	// Mark the receiver's field selections, then rewrite a copy of the expression.
	fields := map[ast.Node]string{}
	decidable := true
	ast.Inspect(expr, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := ast.Unparen(x.X).(*ast.Ident); ok && info.Uses[id] == recv {
				if z, ok := zero[x.Sel.Name]; ok {
					fields[x] = z
					return false
				}
				decidable = false
			}
		case *ast.Ident:
			if info.Uses[x] == recv {
				decidable = false
			}
		}
		return true
	})
	if !decidable {
		return
	}
	text := rewrite(pass.Fset, expr, func(n ast.Node) string { return fields[n] })
	if text == "" {
		return
	}
	tv, err := types.Eval(pass.Fset, pass.Pkg, expr.Pos(), text)
	if err != nil || tv.Value == nil || tv.Value.Kind() != constant.Bool || constant.BoolVal(tv.Value) {
		return
	}
	model.Report(pass, expr.Pos(), "VL0506", "the zero value of %s breaks its invariant %s; Go makes zero values in many places, so the invariant must hold for it (for example Lo <= Hi, not Lo < Hi)", tn.Name(), printed(pass.Fset, expr))
}

func printed(fset *token.FileSet, e ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

// rewrite prints e, parses the text again, and replaces each node of the copy for which f,
// given the node of e that it copies, gives a text.
func rewrite(fset *token.FileSet, e ast.Expr, f func(orig ast.Node) string) string {
	x, err := parser.ParseExpr(printed(fset, e))
	if err != nil {
		return ""
	}
	var origs, copies []ast.Node
	ast.Inspect(e, func(n ast.Node) bool {
		if n != nil {
			origs = append(origs, n)
		}
		return true
	})
	ast.Inspect(x, func(n ast.Node) bool {
		if n != nil {
			copies = append(copies, n)
		}
		return true
	})
	if len(origs) != len(copies) {
		return ""
	}
	orig := map[ast.Node]ast.Node{}
	for i := range copies {
		orig[copies[i]] = origs[i]
	}
	out := astutil.Apply(x, func(c *astutil.Cursor) bool {
		if n := c.Node(); n != nil {
			if t := f(orig[n]); t != "" {
				c.Replace(ast.NewIdent(t))
				return false
			}
		}
		return true
	}, nil)
	return printed(token.NewFileSet(), out.(ast.Expr))
}

// literals rejects a value of a type with an invariant that the code builds without
// contract.Valid: a non-empty literal, also with its type elided, and a conversion (VL0905).
func (c *checker) literals() {
	pass, info := c.pass, c.pass.TypesInfo
	cp := c.u.M.ContractPath()
	ins := inspector.New(pass.Files)
	valid := func(stack []ast.Node) bool {
		i := len(stack) - 1
		node := stack[i]
		if i > 0 {
			if u, ok := stack[i-1].(*ast.UnaryExpr); ok && u.Op == token.AND {
				node, i = u, i-1
			}
		}
		for i > 0 {
			if p, ok := stack[i-1].(*ast.ParenExpr); ok {
				node, i = p, i-1
				continue
			}
			break
		}
		if i == 0 {
			return false
		}
		call, ok := stack[i-1].(*ast.CallExpr)
		return ok && len(call.Args) == 1 && call.Args[0] == node && model.Contract(info, call, cp) == "Valid"
	}
	ins.WithStack([]ast.Node{(*ast.CompositeLit)(nil), (*ast.CallExpr)(nil)}, func(n ast.Node, push bool, stack []ast.Node) bool {
		if !push {
			return true
		}
		var t types.Type
		switch x := n.(type) {
		case *ast.CompositeLit:
			if len(x.Elts) == 0 {
				return true // the zero value keeps the invariant (VL0506)
			}
			t = info.TypeOf(x)
			if p, ok := types.Unalias(t).(*types.Pointer); ok && x.Type == nil {
				t = p.Elem()
			}
		case *ast.CallExpr:
			if tv, ok := info.Types[x.Fun]; !ok || !tv.IsType() || len(x.Args) != 1 {
				return true
			}
			t = info.TypeOf(x)
		}
		tn, _ := types.Unalias(t).(*types.Named)
		if tn == nil || !c.is(tn.Origin().Obj()) {
			return true
		}
		if !valid(stack) {
			name := model.TypeName(pass, t)
			model.Report(pass, n.Pos(), "VL0905", "this builds a %s without checking its invariant; write contract.Valid(%s{...}) (or contract.Valid(&%s{...})), which checks it where the value is built", name, name, name)
		}
		return true
	})
}

// fieldWrites applies the rule that only a struct's own methods write its fields or take a
// field's address (VL0507, VL0508), as the PoC's fieldWrites does. A field reached through
// embedded fields counts for each struct on the path. A call of a pointer method on a field
// takes the field's address. A function literal inside a method counts as outside it, unless
// the method calls it directly or with defer.
func (c *checker) fieldWrites(f *ast.File) {
	pass, info := c.pass, c.pass.TypesInfo
	var owner func(e ast.Expr) *types.TypeName
	owner = func(e ast.Expr) *types.TypeName {
		switch x := ast.Unparen(e).(type) {
		case *ast.SelectorExpr:
			sel := info.Selections[x]
			if sel == nil || sel.Kind() != types.FieldVal {
				return nil
			}
			t := sel.Recv()
			for _, i := range sel.Index() {
				if p, ok := t.Underlying().(*types.Pointer); ok {
					t = p.Elem()
				}
				if tn := model.NamedOf(t); c.is(tn) {
					return tn
				}
				st, ok := t.Underlying().(*types.Struct)
				if !ok {
					break
				}
				t = st.Field(i).Type()
			}
			if _, ptr := info.TypeOf(x.X).Underlying().(*types.Pointer); ptr {
				return nil // a field of a value reached through a pointer is that value's own
			}
			return owner(x.X)
		case *ast.IndexExpr:
			if _, isArray := info.TypeOf(x.X).Underlying().(*types.Array); isArray {
				return owner(x.X)
			}
		}
		return nil
	}
	for _, d := range f.Decls {
		var self *types.TypeName
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv != nil && len(fd.Recv.List) == 1 {
			self = model.NamedOf(info.TypeOf(fd.Recv.List[0].Type))
		}
		called := map[*ast.FuncLit]bool{}
		var visit func(root ast.Node, self *types.TypeName)
		visit = func(root ast.Node, self *types.TypeName) {
			ast.Inspect(root, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncLit:
					if x == root {
						return true
					}
					inner := self
					if !called[x] {
						inner = nil
					}
					visit(x, inner)
					return false
				case *ast.CallExpr:
					if lit, ok := ast.Unparen(x.Fun).(*ast.FuncLit); ok {
						called[lit] = true
					}
					if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
						if s := info.Selections[sel]; s != nil && s.Kind() == types.MethodVal {
							if sig, ok := s.Obj().Type().(*types.Signature); ok && sig.Recv() != nil {
								if _, ptrRecv := sig.Recv().Type().(*types.Pointer); ptrRecv {
									if _, ptrX := info.TypeOf(sel.X).Underlying().(*types.Pointer); !ptrX {
										if tn := owner(sel.X); tn != nil && tn != self {
											model.Report(pass, sel.Pos(), "VL0508", "this calls the pointer method %s on a field of %s outside its methods, which takes the field's address and would go around its invariant", sel.Sel.Name, tn.Name())
										}
									}
								}
							}
						}
					}
				case *ast.DeferStmt:
					if lit, ok := ast.Unparen(x.Call.Fun).(*ast.FuncLit); ok {
						called[lit] = true
					}
				}
				model.Assigned(n, func(l ast.Expr, addr bool) {
					tn := owner(l)
					if tn == nil || tn == self {
						return
					}
					if addr {
						model.Report(pass, n.Pos(), "VL0508", "this takes the address of a field of %s outside its methods; a write through it would go around its invariant, so read the field into a local variable instead", tn.Name())
					} else {
						model.Report(pass, l.Pos(), "VL0507", "this writes a field of %s outside its methods, which would go around its invariant; add a method of %s that makes the change, or build a new value with contract.Valid(%s{...})", tn.Name(), tn.Name(), tn.Name())
					}
				})
				return true
			})
		}
		visit(d, self)
	}
}
