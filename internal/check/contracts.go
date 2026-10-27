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
				}
				continue
			}
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
// address (K0507, K0508).
func (c *Checker) fieldWrites(info *types.Info, f *ast.File, inv map[*types.TypeName]bool) {
	// owner returns the invariant type whose field e selects, through the chain of e.
	var owner func(e ast.Expr) *types.TypeName
	owner = func(e ast.Expr) *types.TypeName {
		switch x := ast.Unparen(e).(type) {
		case *ast.SelectorExpr:
			if sel := info.Selections[x]; sel != nil && sel.Kind() == types.FieldVal {
				if tn := namedOf(sel.Recv()); tn != nil && inv[tn] {
					return tn
				}
				return owner(x.X)
			}
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
		write := func(l ast.Expr) {
			if tn := owner(l); tn != nil && tn != self {
				c.D.Errorf(c.Pos(l.Pos()), "K0507", "this writes a field of %s outside its methods, which would go around its invariant; add a method of %s that makes the change, or build a new value with a literal", tn.Name(), tn.Name())
			}
		}
		ast.Inspect(d, func(n ast.Node) bool {
			switch x := n.(type) {
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
					if b, ok := st.Field(i).Type().Underlying().(*types.Basic); ok {
						switch {
						case b.Info()&types.IsNumeric != 0:
							zero[st.Field(i).Name()] = b.Name() + "(0)"
						case b.Info()&types.IsString != 0:
							zero[st.Field(i).Name()] = `""`
						case b.Info()&types.IsBoolean != 0:
							zero[st.Field(i).Name()] = "false"
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
				}
				out[fn.FullName()] = cal
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
			if tv, ok := info.Types[a]; ok && tv.Value != nil {
				values[cal.params[i]] = tv.Value.ExactString()
				if tv.Value.Kind() == constant.String {
					values[cal.params[i]] = strconv.Quote(constant.StringVal(tv.Value))
				}
			}
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
