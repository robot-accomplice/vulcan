// Package ports is analyzer A3 of vulcan-lint: the ports of each leaf, its constructors, and the
// fields that hold its ports.
package ports

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A3.
var Analyzer = &analysis.Analyzer{
	Name: "ports",
	Doc:  "check the ports of each leaf (exported interfaces in ports.go, with their contract wrappers), its constructors (ports and one settings struct), and the fields that hold the ports (VL0104, VL0110, VL0112, VL0116, VL0117, VL0120, VL0123, VL0201, VL0604, VL0906)",
	Run:  run,
}

// PortsFile is the file of a leaf that declares its ports.
const PortsFile = "ports.go"

type checker struct {
	pass     *analysis.Pass
	u        *model.Unit
	ports    map[*types.TypeName]bool
	wrappers map[*types.TypeName]*types.TypeName // port -> its wrapper
	wrapped  map[*types.TypeName]*types.TypeName // wrapper -> its port
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Manifest || u.Part == nil {
		return nil, nil
	}
	if u.Part.Kind == model.Library {
		for _, f := range pass.Files {
			if model.FileName(pass, f) == PortsFile && !model.IsTest(pass, f) {
				model.Report(pass, f.Name.Pos(), "VL0201", "library %s has a %s; a library has no ports: it uses only what its callers give it", u.Dir, PortsFile)
			}
		}
		return nil, nil
	}
	if !u.Part.IsLeaf() {
		return nil, nil
	}
	c := &checker{pass: pass, u: u, ports: map[*types.TypeName]bool{}, wrappers: map[*types.TypeName]*types.TypeName{}, wrapped: map[*types.TypeName]*types.TypeName{}}
	var portsFile *ast.File
	for _, f := range pass.Files {
		if model.FileName(pass, f) == PortsFile && !model.IsTest(pass, f) {
			portsFile = f
		}
	}
	if portsFile != nil {
		c.portsFile(portsFile)
	}
	for tn := range c.ports {
		c.portMethods(tn)
	}
	used := c.constructors()
	for tn := range c.ports {
		if !used[tn] {
			model.Report(pass, tn.Pos(), "VL0104", "port %s is not a parameter of any constructor of %s, so main cannot give it a provider; take it in the constructor, or remove it", tn.Name(), u.Dir)
		}
	}
	c.portFields()
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Name.Name == "init" {
				model.Report(pass, fd.Name.Pos(), "VL0117", "leaf %s declares init, which runs before main wires the leaves; do the work in a constructor or a lifecycle method", u.Dir)
			}
		}
	}
	return nil, nil
}

// portsFile reads the ports and the wrappers of ports.go, and rejects anything else in it.
func (c *checker) portsFile(f *ast.File) {
	pass, info := c.pass, c.pass.TypesInfo
	only := "ports.go holds only the ports of the leaf (exported interfaces) and their contract wrappers (type checkedX struct{ X }, with methods)"
	var structs []*ast.TypeSpec
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			switch d.Tok.String() {
			case "import":
			case "type":
				for _, s := range d.Specs {
					ts := s.(*ast.TypeSpec)
					tn, _ := info.Defs[ts.Name].(*types.TypeName)
					if tn == nil {
						continue
					}
					if _, ok := tn.Type().Underlying().(*types.Interface); ok && !ts.Assign.IsValid() {
						if !tn.Exported() {
							model.Report(pass, ts.Name.Pos(), "VL0110", "port %s is not exported; a port is an exported interface, so that main can pass its provider", tn.Name())
						}
						c.ports[tn] = true
						continue
					}
					structs = append(structs, ts)
				}
			default:
				model.Report(pass, d.Pos(), "VL0110", "%s; move this %s to another file", only, d.Tok)
			}
		case *ast.FuncDecl:
			if d.Recv == nil {
				model.Report(pass, d.Name.Pos(), "VL0110", "%s; move func %s to another file", only, d.Name.Name)
			}
		}
	}
	for _, ts := range structs {
		tn := info.Defs[ts.Name].(*types.TypeName)
		st, ok := tn.Type().Underlying().(*types.Struct)
		var port *types.TypeName
		if ok && st.NumFields() == 1 && st.Field(0).Embedded() {
			if n, ok := types.Unalias(st.Field(0).Type()).(*types.Named); ok && c.ports[n.Obj()] {
				port = n.Obj()
			}
		}
		if port == nil || tn.Name() != "checked"+port.Name() {
			model.Report(pass, ts.Name.Pos(), "VL0110", "%s; %s is neither", only, tn.Name())
			continue
		}
		c.wrappers[port], c.wrapped[tn] = tn, port
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Recv == nil {
			continue
		}
		recv := model.NamedOf(info.TypeOf(fd.Recv.List[0].Type))
		if port := c.wrapped[recv]; port != nil {
			c.wrapperMethod(fd, port)
		} else {
			model.Report(pass, fd.Name.Pos(), "VL0110", "%s; move method %s to another file", only, fd.Name.Name)
		}
	}
}

// portMethods checks the methods of a port: exported, and their types (VL0110, VL0112, VL0123).
func (c *checker) portMethods(tn *types.TypeName) {
	iface := tn.Type().Underlying().(*types.Interface)
	for i := 0; i < iface.NumMethods(); i++ {
		meth := iface.Method(i)
		pos := meth.Pos()
		if meth.Pkg() != c.pass.Pkg {
			pos = tn.Pos()
		}
		if !meth.Exported() {
			model.Report(c.pass, pos, "VL0110", "port %s has the unexported method %s; a provider in another package cannot have it, so port methods are exported", tn.Name(), meth.Name())
			continue
		}
		sig := meth.Type().(*types.Signature)
		var bad, why string
		for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
			for j := 0; j < tuple.Len() && bad == ""; j++ {
				bad, why = c.foreign(tuple.At(j).Type(), map[types.Type]bool{})
			}
		}
		if bad != "" {
			model.Report(c.pass, pos, "VL0112", "port %s method %s uses the type %s, %s; a port signature uses predeclared types, the standard library, and libraries of the module (lib/), so that the provider can name every type", tn.Name(), meth.Name(), bad, why)
		}
		if !c.u.M.StrictFor(c.u.Part) {
			continue
		}
		bad = ""
		for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
			for j := 0; j < tuple.Len() && bad == ""; j++ {
				bad = Behavior(tuple.At(j).Type(), c.u.M.Interfaces, map[types.Type]bool{})
			}
		}
		if bad != "" {
			model.Report(c.pass, pos, "VL0123", "strict mode: port %s method %s carries %s, which lets the callee run the caller's code; pass values, or list the interface in StrictInterfaces of the manifest", tn.Name(), meth.Name(), bad)
		}
	}
}

// foreign names a type in t that a port signature may not use, with the reason, or "".
func (c *checker) foreign(t types.Type, seen map[types.Type]bool) (string, string) {
	if seen[t] {
		return "", ""
	}
	seen[t] = true
	switch u := types.Unalias(t).(type) {
	case *types.Named:
		pkg := u.Obj().Pkg()
		if pkg == nil {
			return "", ""
		}
		name := types.TypeString(u, nil)
		switch p := pkg.Path(); {
		case pkg == c.pass.Pkg:
			return u.Obj().Name(), "which belongs to leaf " + c.u.Dir
		case c.u.M.InModule(p):
			if part := c.u.M.PartOf(p); part == nil || part.Kind != model.Library {
				return name, "which is not in a library"
			}
		case model.IsStd(p):
		default:
			return name, "which comes from another module"
		}
		if args := u.TypeArgs(); args != nil {
			for i := 0; i < args.Len(); i++ {
				if b, w := c.foreign(args.At(i), seen); b != "" {
					return b, w
				}
			}
		}
	case *types.Pointer:
		return c.foreign(u.Elem(), seen)
	case *types.Slice:
		return c.foreign(u.Elem(), seen)
	case *types.Array:
		return c.foreign(u.Elem(), seen)
	case *types.Chan:
		return c.foreign(u.Elem(), seen)
	case *types.Map:
		if b, w := c.foreign(u.Key(), seen); b != "" {
			return b, w
		}
		return c.foreign(u.Elem(), seen)
	case *types.Signature:
		for _, tuple := range []*types.Tuple{u.Params(), u.Results()} {
			for i := 0; i < tuple.Len(); i++ {
				if b, w := c.foreign(tuple.At(i).Type(), seen); b != "" {
					return b, w
				}
			}
		}
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if b, w := c.foreign(u.Field(i).Type(), seen); b != "" {
				return b, w
			}
		}
	}
	return "", ""
}

// Behavior names a function, channel, or interface type that t carries, or "" (the walk of the
// PoC's behavior).
func Behavior(t types.Type, allowed map[string]bool, seen map[types.Type]bool) string {
	if seen[t] {
		return ""
	}
	seen[t] = true
	full := types.TypeString(t, nil)
	if full == "error" || full == "context.Context" || allowed[full] {
		return ""
	}
	switch u := t.Underlying().(type) {
	case *types.Signature:
		return "the function type " + full
	case *types.Chan:
		return "the channel type " + full
	case *types.Interface:
		return "the interface type " + full
	case *types.Pointer:
		return Behavior(u.Elem(), allowed, seen)
	case *types.Slice:
		return Behavior(u.Elem(), allowed, seen)
	case *types.Array:
		return Behavior(u.Elem(), allowed, seen)
	case *types.Map:
		if b := Behavior(u.Key(), allowed, seen); b != "" {
			return b
		}
		return Behavior(u.Elem(), allowed, seen)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if b := Behavior(u.Field(i).Type(), allowed, seen); b != "" {
				return b + " (in field " + u.Field(i).Name() + " of " + full + ")"
			}
		}
	}
	return ""
}

// wrapperMethod checks a method of a contract wrapper: its contract, then one call of the same
// method of the port with the same arguments (VL0906), and the names in its contract (VL0120).
func (c *checker) wrapperMethod(fd *ast.FuncDecl, port *types.TypeName) {
	pass, info := c.pass, c.pass.TypesInfo
	shape := "a wrapper method is its contract, then return w." + port.Name() + "." + fd.Name.Name + "(its parameters, in order)"
	obj, _, _ := types.LookupFieldOrMethod(port.Type(), false, port.Pkg(), fd.Name.Name)
	if _, ok := obj.(*types.Func); !ok {
		model.Report(pass, fd.Name.Pos(), "VL0906", "%s is not a method of port %s, so its wrapper cannot have it", fd.Name.Name, port.Name())
		return
	}
	pro := model.PrologueOf(info, fd.Body, c.u.M.ContractPath())
	rest := fd.Body.List[pro.Stmts:]
	var call *ast.CallExpr
	if len(rest) == 1 {
		switch s := rest[0].(type) {
		case *ast.ReturnStmt:
			if len(s.Results) == 1 {
				call, _ = s.Results[0].(*ast.CallExpr)
			}
		case *ast.ExprStmt:
			if fd.Type.Results == nil {
				call, _ = s.X.(*ast.CallExpr)
			}
		}
	}
	if call == nil || !c.forwards(fd, call, port) {
		model.Report(pass, fd.Name.Pos(), "VL0906", "the wrapper method %s.%s does more than check the contract: %s", c.wrappers[port].Name(), fd.Name.Name, shape)
	}
	// A port contract names only its parameters, results, old values, the standard library, and
	// libraries: never a name of its own leaf.
	for s := 0; s < pro.Stmts; s++ {
		ast.Inspect(fd.Body.List[s], func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			o := info.Uses[id]
			if o == nil || o.Pkg() != pass.Pkg || o.Parent() != pass.Pkg.Scope() {
				return true
			}
			model.Report(pass, id.Pos(), "VL0120", "the contract of port %s names %s, which belongs to leaf %s; a port contract names only its parameters, results, old values, the standard library, and libraries", port.Name(), o.Name(), c.u.Dir)
			return true
		})
	}
}

// forwards reports whether call is recv.Port.Method(params...).
func (c *checker) forwards(fd *ast.FuncDecl, call *ast.CallExpr, port *types.TypeName) bool {
	info := c.pass.TypesInfo
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != fd.Name.Name {
		return false
	}
	inner, ok := ast.Unparen(sel.X).(*ast.SelectorExpr)
	if !ok || inner.Sel.Name != port.Name() || len(fd.Recv.List[0].Names) != 1 {
		return false
	}
	recv, ok := ast.Unparen(inner.X).(*ast.Ident)
	if !ok || info.Uses[recv] != info.Defs[fd.Recv.List[0].Names[0]] {
		return false
	}
	var params []types.Object
	for _, field := range fd.Type.Params.List {
		for _, n := range field.Names {
			params = append(params, info.Defs[n])
		}
	}
	if len(call.Args) != len(params) {
		return false
	}
	for i, a := range call.Args {
		id, ok := a.(*ast.Ident)
		if !ok || info.Uses[id] != params[i] || params[i] == nil {
			return false
		}
	}
	sig := info.TypeOf(fd.Name).(*types.Signature)
	return sig.Variadic() == call.Ellipsis.IsValid()
}

// IsConstructor reports whether a function is a constructor of its leaf: exported, with the
// results T or (T, error), where T is a type of the leaf.
func IsConstructor(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Recv() != nil || !fn.Exported() || sig.TypeParams() != nil {
		return false
	}
	res := sig.Results()
	switch res.Len() {
	case 1:
	case 2:
		if types.TypeString(res.At(1).Type(), nil) != "error" {
			return false
		}
	default:
		return false
	}
	tn := model.NamedOf(res.At(0).Type())
	if tn == nil || tn.Pkg() != fn.Pkg() {
		return false
	}
	_, isIface := tn.Type().Underlying().(*types.Interface)
	return !isIface
}

// IsSettings reports whether t is a settings struct: a named struct type whose fields are all
// exported, of a basic type.
func IsSettings(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	st, ok := n.Underlying().(*types.Struct)
	if !ok {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if _, basic := f.Type().Underlying().(*types.Basic); !basic || !f.Exported() {
			return false
		}
	}
	return true
}

// constructors checks the constructors of the leaf (VL0604, VL0906), and returns the ports that
// some constructor takes.
func (c *checker) constructors() map[*types.TypeName]bool {
	pass, info := c.pass, c.pass.TypesInfo
	used := map[*types.TypeName]bool{}
	shape := "a constructor takes the ports of its leaf (the interfaces of ports.go) and, last, at most one settings struct of the leaf (exported fields of basic types)"
	for _, f := range pass.Files {
		if model.IsTest(pass, f) {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv != nil {
				continue
			}
			fn, _ := info.Defs[fd.Name].(*types.Func)
			if fn == nil || !IsConstructor(fn) {
				continue
			}
			sig := fn.Type().(*types.Signature)
			if tn := model.NamedOf(sig.Results().At(0).Type()); !tn.Exported() {
				model.Report(pass, fd.Name.Pos(), "VL0604", "constructor %s returns the unexported type %s; a constructor returns an exported type of its leaf, or a pointer to one", fn.Name(), tn.Name())
			}
			if sig.Variadic() {
				model.Report(pass, fd.Name.Pos(), "VL0604", "constructor %s is variadic; %s", fn.Name(), shape)
			}
			params := sig.Params()
			for i := 0; i < params.Len(); i++ {
				p := params.At(i)
				pt := p.Type()
				if n, ok := types.Unalias(pt).(*types.Named); ok && c.ports[n.Obj()] {
					used[n.Obj()] = true
					if w := c.wrappers[n.Obj()]; w != nil {
						c.storedWrapped(fd, p, w)
					}
					continue
				}
				n, _ := types.Unalias(pt).(*types.Named)
				if i == params.Len()-1 && n != nil && n.Obj().Pkg() == pass.Pkg && n.Obj().Exported() && IsSettings(pt) {
					continue
				}
				why := "which is not a port of " + c.u.Dir
				switch {
				case n != nil && n.Obj().Pkg() == pass.Pkg && IsSettings(pt) && !n.Obj().Exported():
					why = "an unexported settings type"
				case n != nil && n.Obj().Pkg() == pass.Pkg && IsSettings(pt):
					why = "a settings struct that is not the last parameter"
				case types.IsInterface(pt):
					why = "an interface that is not declared in ports.go"
				}
				model.Report(pass, fd.Name.Pos(), "VL0604", "constructor %s takes %s %s, %s; %s", fn.Name(), p.Name(), model.TypeName(pass, pt), why, shape)
			}
		}
	}
	return used
}

// storedWrapped checks that the constructor uses the port parameter p only as checkedX{p}, so
// that its contract is checked at every call (VL0906).
func (c *checker) storedWrapped(fd *ast.FuncDecl, p *types.Var, w *types.TypeName) {
	info := c.pass.TypesInfo
	ok := map[*ast.Ident]bool{}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		cl, isLit := n.(*ast.CompositeLit)
		if isLit && len(cl.Elts) == 1 && model.NamedOf(info.TypeOf(cl)) == w {
			e := cl.Elts[0]
			if kv, isKV := e.(*ast.KeyValueExpr); isKV {
				e = kv.Value
			}
			if id, isID := ast.Unparen(e).(*ast.Ident); isID && info.Uses[id] == p {
				ok[id] = true
			}
		}
		return true
	})
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if id, isID := n.(*ast.Ident); isID && info.Uses[id] == p && !ok[id] {
			model.Report(c.pass, id.Pos(), "VL0906", "port %s has a contract, but the constructor uses %s unwrapped; store %s{%s}, so that the contract is checked at every call", p.Type().(*types.Named).Obj().Name(), p.Name(), w.Name(), p.Name())
		}
		return true
	})
}

// portFields rejects a write to a field that holds a port outside a constructor, and a
// package-level variable that holds a port (VL0116).
func (c *checker) portFields() {
	pass, info := c.pass, c.pass.TypesInfo
	isPort := func(t types.Type) bool {
		n, ok := types.Unalias(t).(*types.Named)
		return ok && c.ports[n.Obj()]
	}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.GenDecl:
				if d.Tok.String() != "var" {
					continue
				}
				for _, s := range d.Specs {
					for _, n := range s.(*ast.ValueSpec).Names {
						if v, ok := info.Defs[n].(*types.Var); ok && holdsPort(v.Type(), isPort, map[types.Type]bool{}) {
							model.Report(pass, n.Pos(), "VL0116", "package-level variable %s holds a port; a port is a field of the leaf's type, set once by its constructor", n.Name)
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil {
					if fn, ok := info.Defs[d.Name].(*types.Func); ok && IsConstructor(fn) && !model.IsTest(pass, f) {
						continue
					}
				}
				ast.Inspect(d, func(n ast.Node) bool {
					model.Assigned(n, func(e ast.Expr, addr bool) {
						sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
						if !ok {
							return
						}
						s := info.Selections[sel]
						if s == nil || s.Kind() != types.FieldVal || !isPort(s.Obj().Type()) {
							return
						}
						what := "assigns to"
						if addr {
							what = "takes the address of"
						}
						model.Report(pass, sel.Pos(), "VL0116", "this %s the port field %s outside a constructor; a port is set once, by the constructor (in a test, build the value with its constructor and a fake)", what, sel.Sel.Name)
					})
					return true
				})
			}
		}
	}
}

func holdsPort(t types.Type, isPort func(types.Type) bool, seen map[types.Type]bool) bool {
	if seen[t] {
		return false
	}
	seen[t] = true
	if isPort(t) {
		return true
	}
	switch u := t.Underlying().(type) {
	case *types.Pointer:
		return holdsPort(u.Elem(), isPort, seen)
	case *types.Slice:
		return holdsPort(u.Elem(), isPort, seen)
	case *types.Array:
		return holdsPort(u.Elem(), isPort, seen)
	case *types.Map:
		return holdsPort(u.Key(), isPort, seen) || holdsPort(u.Elem(), isPort, seen)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if holdsPort(u.Field(i).Type(), isPort, seen) {
				return true
			}
		}
	}
	return false
}
