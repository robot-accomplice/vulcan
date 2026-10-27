package check

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/robot-accomplice/vulcan/internal/blank"
	"github.com/robot-accomplice/vulcan/internal/lower"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// Contracts applies the contract rules that need types (spec §7.1, §7.2, Appendix F).
func (c *Checker) Contracts() {
	inv := c.invariantTypes()
	callees := c.requiresOf()
	for _, pkg := range append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...) {
		gp := c.Full[pkg.ImportPath]
		if gp == nil || gp.TypesInfo == nil {
			continue
		}
		for _, f := range gp.Syntax {
			if c.Generated(f) {
				if pkg.IsLeaf && strings.HasSuffix(c.Fset.PositionFor(f.Package, false).Filename, "vulcan_ports.go") {
					c.portContractNames(pkg, gp, f)
					c.purity(gp.TypesInfo, f)
				}
				continue
			}
			c.purity(gp.TypesInfo, f)
			c.fieldWrites(gp.TypesInfo, f, inv)
			c.constantCalls(gp.TypesInfo, f, callees)
		}
		c.olds(pkg, gp)
	}
	c.zeroValues()
}

// invariantTypes returns the struct types with an invariant.
func (c *Checker) invariantTypes() map[*types.TypeName]bool {
	out := map[*types.TypeName]bool{}
	for path, names := range lower.Invariants(c.P) {
		for _, gp := range []*packages.Package{c.Plain[path], c.Full[path]} {
			if gp == nil || gp.Types == nil {
				continue
			}
			for name := range names {
				if tn, ok := gp.Types.Scope().Lookup(name).(*types.TypeName); ok {
					out[tn] = true
				}
			}
		}
	}
	return out
}

func namedOf(t types.Type) *types.TypeName {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	if n, ok := types.Unalias(t).(*types.Named); ok {
		return n.Origin().Obj()
	}
	return nil
}

// fieldWrites applies §7.2: only a struct's own methods write its fields or take a field's
// address (K0507, K0508). A field reached through embedded fields counts for each struct on the
// path. A call of a pointer-receiver method on a field takes the field's address. A function
// literal inside a method counts as outside it, unless the method calls it directly or with
// defer.
func (c *Checker) fieldWrites(info *types.Info, f *ast.File, inv map[*types.TypeName]bool) {
	// owner returns the invariant type whose field e selects, through the chain of e.
	var owner func(e ast.Expr) *types.TypeName
	owner = func(e ast.Expr) *types.TypeName {
		switch x := ast.Unparen(e).(type) {
		case *ast.SelectorExpr:
			sel := info.Selections[x]
			if sel == nil || sel.Kind() != types.FieldVal {
				return nil
			}
			// Each struct on the path of embedded fields, then the struct that declares it.
			t := sel.Recv()
			for _, i := range sel.Index() {
				if p, ok := t.Underlying().(*types.Pointer); ok {
					t = p.Elem()
				}
				if tn := namedOf(t); tn != nil && inv[tn] {
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
			if t := info.TypeOf(fd.Recv.List[0].Type); t != nil {
				self = namedOf(t)
			}
		}
		called := map[*ast.FuncLit]bool{}
		var visit func(n ast.Node, self *types.TypeName)
		visit = func(root ast.Node, self *types.TypeName) {
			write := func(l ast.Expr) {
				if tn := owner(l); tn != nil && tn != self {
					c.D.Errorf(c.Pos(l.Pos()), "K0507", "this writes a field of %s outside its methods, which would go around its invariant; add a method of %s that makes the change, or build a new value with a literal", tn.Name(), tn.Name())
				}
			}
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
					// A pointer-receiver method called on a field takes the field's address.
					if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
						if s := info.Selections[sel]; s != nil && s.Kind() == types.MethodVal {
							if sig, ok := s.Obj().Type().(*types.Signature); ok && sig.Recv() != nil {
								if _, ptrRecv := sig.Recv().Type().(*types.Pointer); ptrRecv {
									if _, ptrX := info.TypeOf(sel.X).Underlying().(*types.Pointer); !ptrX {
										if tn := owner(sel.X); tn != nil && tn != self {
											c.D.Errorf(c.Pos(sel.Pos()), "K0508", "this calls the pointer method %s on a field of %s outside its methods, which takes the field's address and would go around its invariant", sel.Sel.Name, tn.Name())
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
				case *ast.AssignStmt:
					if x.Tok != token.DEFINE {
						for _, l := range x.Lhs {
							write(l)
						}
					}
				case *ast.IncDecStmt:
					write(x.X)
				case *ast.RangeStmt:
					if x.Tok == token.ASSIGN {
						for _, l := range []ast.Expr{x.Key, x.Value} {
							if l != nil {
								write(l)
							}
						}
					}
				case *ast.UnaryExpr:
					if x.Op == token.AND {
						if tn := owner(x.X); tn != nil && tn != self {
							c.D.Errorf(c.Pos(x.Pos()), "K0508", "this takes the address of a field of %s outside its methods; a write through it would go around its invariant, so read the field into a local variable instead", tn.Name())
						}
					}
				}
				return true
			})
		}
		visit(d, self)
	}
}

// olds applies §7.1: old(x) needs a type that holds no reference (K0509).
func (c *Checker) olds(pkg *project.Package, gp *packages.Package) {
	if len(pkg.Olds) == 0 {
		return
	}
	for id, obj := range gp.TypesInfo.Defs {
		if obj == nil {
			continue
		}
		pos, ok := pkg.Olds[id.Name]
		if !ok {
			continue
		}
		if holdsReference(obj.Type(), map[types.Type]bool{}) {
			c.D.Errorf(pos, "K0509", "old() of a value of type %s, which holds a reference: the copy would show the new data; write old() of a value, for example old(len(xs)) or old(a.Balance)", types.TypeString(obj.Type(), qualifier))
		}
		delete(pkg.Olds, id.Name)
	}
}

func holdsReference(t types.Type, seen map[types.Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	if _, ok := t.(*types.TypeParam); ok {
		return false // the PoC does not decide a type set
	}
	switch u := t.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature, *types.Interface:
		return true
	case *types.Array:
		return holdsReference(u.Elem(), seen)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if holdsReference(u.Field(i).Type(), seen) {
				return true
			}
		}
	}
	return false
}

// zeroValues applies §7.2: the zero value satisfies the invariant (K0506). The PoC decides an
// invariant whose fields are all numbers, strings, or booleans.
func (c *Checker) zeroValues() {
	for _, pkg := range append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...) {
		gp := c.Plain[pkg.ImportPath]
		if gp == nil || gp.Types == nil {
			continue
		}
		for _, f := range pkg.Files {
			for _, t := range f.Blank.Types {
				tn, ok := gp.Types.Scope().Lookup(t.Name).(*types.TypeName)
				if !ok {
					continue
				}
				st, ok := tn.Type().Underlying().(*types.Struct)
				if !ok {
					continue
				}
				zero := map[string]string{}
				for i := 0; i < st.NumFields(); i++ {
					ft := st.Field(i).Type()
					if b, ok := ft.Underlying().(*types.Basic); ok {
						// The zero as a conversion to the field's own type, so that it compares
						// with typed constants (audit P2, L3).
						name := types.TypeString(ft, types.RelativeTo(gp.Types))
						switch {
						case b.Info()&types.IsNumeric != 0:
							zero[st.Field(i).Name()] = name + "(0)"
						case b.Info()&types.IsString != 0:
							zero[st.Field(i).Name()] = name + `("")`
						case b.Info()&types.IsBoolean != 0:
							zero[st.Field(i).Name()] = name + "(false)"
						}
					}
				}
				for _, cl := range t.Clauses {
					c.zeroClause(gp.Types, f, cl, zero, t.Name)
				}
			}
		}
	}
}

func (c *Checker) zeroClause(pkg *types.Package, f *project.CodeFile, cl blank.Clause, zero map[string]string, name string) {
	e, err := lower.ParseContract(cl.Text)
	if err != nil {
		return
	}
	decidable := true
	x := lower.Rewrite(e, func(n ast.Expr) ast.Expr {
		if id, ok := n.(*ast.Ident); ok {
			if z, ok := zero[id.Name]; ok {
				return ast.NewIdent(z)
			}
			if pkg.Scope().Lookup(id.Name) == nil && types.Universe.Lookup(id.Name) == nil {
				decidable = false // a field that is not a number, a string, or a boolean
			}
		}
		return nil
	})
	if !decidable {
		return
	}
	tv, err := types.Eval(c.Fset, pkg, token.NoPos, lower.GoText(x))
	if err != nil || tv.Value == nil || tv.Value.Kind() != constant.Bool || constant.BoolVal(tv.Value) {
		return
	}
	c.D.Errorf(f.Pos(cl.Off), "K0506", "the zero value of %s breaks its invariant %s; Go makes zero values in many places, so the invariant must hold for it (for example Lo <= Hi, not Lo < Hi)", name, cl.Text)
}

type callee struct {
	params   []string
	types    []types.Type
	requires []blank.Clause
	file     *project.CodeFile
	pkg      *types.Package
	variadic bool
}

// requiresOf maps each function and method that has a requires clause to its clauses.
func (c *Checker) requiresOf() map[string]*callee {
	out := map[string]*callee{}
	for _, pkg := range append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...) {
		gp := c.Plain[pkg.ImportPath]
		if gp == nil || gp.Types == nil {
			continue
		}
		for _, f := range pkg.Files {
			for _, bf := range f.Blank.Funcs {
				var req []blank.Clause
				for _, cl := range bf.Clauses {
					if cl.Word == "requires" {
						req = append(req, cl)
					}
				}
				if len(req) == 0 {
					continue
				}
				var fn *types.Func
				if bf.Recv == "" {
					fn, _ = gp.Types.Scope().Lookup(bf.Name).(*types.Func)
				} else if tn, ok := gp.Types.Scope().Lookup(bf.Recv).(*types.TypeName); ok {
					obj, _, _ := types.LookupFieldOrMethod(types.NewPointer(tn.Type()), true, gp.Types, bf.Name)
					fn, _ = obj.(*types.Func)
				}
				if fn == nil {
					continue
				}
				sig := fn.Type().(*types.Signature)
				cal := &callee{requires: req, file: f, pkg: gp.Types, variadic: sig.Variadic()}
				for i := 0; i < sig.Params().Len(); i++ {
					cal.params = append(cal.params, sig.Params().At(i).Name())
					cal.types = append(cal.types, sig.Params().At(i).Type())
				}
				out[fn.FullName()] = cal
				if bf.Recv != "" {
					// The unchecked twin of a method has the same requires (audit P2, L4).
					full := fn.FullName()
					if i := strings.LastIndex(full, "."); i >= 0 {
						out[full[:i+1]+"vulcan_"+full[i+1:]] = cal
					}
				}
			}
		}
	}
	return out
}

// constantCalls applies the literal case of K0501: a call whose constant arguments break the
// callee's requires is a compile error.
func (c *Checker) constantCalls(info *types.Info, f *ast.File, callees map[string]*callee) {
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fn := calledFunc(info, call)
		if fn == nil {
			return true
		}
		cal := callees[fn.Origin().FullName()]
		if cal == nil || cal.variadic || len(call.Args) != len(cal.params) {
			return true
		}
		values := map[string]string{}
		for i, a := range call.Args {
			tv, ok := info.Types[a]
			if !ok || tv.Value == nil {
				continue
			}
			// A typed constant of the parameter's type, so that 0.9 stays a float.
			pt := cal.types[i]
			if _, basic := pt.Underlying().(*types.Basic); !basic {
				continue
			}
			lit := tv.Value.ExactString()
			switch tv.Value.Kind() {
			case constant.String:
				lit = strconv.Quote(constant.StringVal(tv.Value))
			case constant.Float:
				lit = tv.Value.String()
				if f, exact := constant.Float64Val(tv.Value); exact {
					lit = strconv.FormatFloat(f, 'g', -1, 64)
				}
			}
			tname := pt.Underlying().(*types.Basic).Name()
			if n, ok := types.Unalias(pt).(*types.Named); ok && n.Obj().Pkg() == cal.pkg {
				tname = n.Obj().Name() // a type of the callee's own package is in scope there
			}
			values[cal.params[i]] = tname + "(" + lit + ")"
		}
		if len(values) == 0 {
			return true
		}
		for _, cl := range cal.requires {
			e, err := lower.ParseContract(cl.Text)
			if err != nil {
				continue
			}
			known := true
			x := lower.Rewrite(e, func(n ast.Expr) ast.Expr {
				if id, ok := n.(*ast.Ident); ok {
					if v, ok := values[id.Name]; ok {
						return ast.NewIdent(v)
					}
					for _, p := range cal.params {
						if p == id.Name {
							known = false // a parameter with an argument that is not a constant
						}
					}
				}
				return nil
			})
			if !known {
				continue
			}
			tv, err := types.Eval(c.Fset, cal.pkg, token.NoPos, lower.GoText(x))
			if err == nil && tv.Value != nil && tv.Value.Kind() == constant.Bool && !constant.BoolVal(tv.Value) {
				c.D.Errorf(c.Pos(call.Pos()), "K0501", "this call breaks the contract of %s: requires %s (at %s) is false for these arguments", fn.Name(), cl.Text, cal.file.Pos(cl.Off))
			}
		}
		return true
	})
}

// portContractNames applies Appendix F, "Port contracts": a port contract names only
// predeclared identifiers, the standard library, and libraries; a name of its own leaf is a
// compile error (K0120).
func (c *Checker) portContractNames(pkg *project.Package, gp *packages.Package, f *ast.File) {
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil || fd.Body == nil {
			continue
		}
		if n := namedOf(gp.TypesInfo.TypeOf(fd.Recv.List[0].Type)); n == nil || !strings.HasPrefix(n.Name(), "Vulcan_wrap_") {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := gp.TypesInfo.Uses[id]
			if obj == nil || obj.Pkg() != gp.Types || obj.Parent() != gp.Types.Scope() {
				return true
			}
			if strings.HasPrefix(obj.Name(), "vulcan_") || strings.HasPrefix(obj.Name(), "Vulcan_") {
				return true
			}
			c.D.Errorf(c.Pos(id.Pos()), "K0120", "the port contract names %s, which belongs to leaf %s; a port contract can name only predeclared identifiers, the standard library, and libraries", obj.Name(), pkg.Name)
			return true
		})
	}
}

// purity applies §7.1 to tier 1, the only tier of this release: a contract calls only len, cap,
// and conversions, and has no function literal and no receive (K0503). It reads the lowered code:
// the condition of each generated check, and each old() capture.
func (c *Checker) purity(info *types.Info, f *ast.File) {
	inspect := func(e ast.Expr) {
		ast.Inspect(e, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncLit:
				c.D.Errorf(c.Pos(x.Pos()), "K0503", "a contract cannot hold a function literal; contracts have no effects")
				return false
			case *ast.UnaryExpr:
				if x.Op == token.ARROW {
					c.D.Errorf(c.Pos(x.Pos()), "K0503", "a contract cannot receive from a channel; contracts have no effects")
				}
			case *ast.CallExpr:
				if tv, ok := info.Types[x.Fun]; ok && tv.IsType() {
					return true // a conversion
				}
				if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok {
					if b, ok := info.Uses[id].(*types.Builtin); ok && (b.Name() == "len" || b.Name() == "cap") {
						return true
					}
				}
				c.D.Errorf(c.Pos(x.Pos()), "K0503", "a contract calls %s; this release has no pure functions, so a contract calls only len, cap, and conversions (§7.1)", types.ExprString(x.Fun))
			}
			return true
		})
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.IfStmt:
			if isFailBlock(x.Body) {
				inspect(x.Cond)
			}
		case *ast.AssignStmt:
			if len(x.Lhs) == 1 && len(x.Rhs) == 1 {
				if id, ok := x.Lhs[0].(*ast.Ident); ok && strings.HasPrefix(id.Name, "vulcan_old") {
					inspect(x.Rhs[0])
				}
			}
		}
		return true
	})
}

// isFailBlock reports whether b is the body of a generated check: one call of vulcan_rt.Fail.
func isFailBlock(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) != 1 {
		return false
	}
	es, ok := b.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := es.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "vulcan_rt" && sel.Sel.Name == "Fail"
}
