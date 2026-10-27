package check

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/robot-accomplice/vulcan/internal/gen"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// RuntimeRT is the generated-code-only runtime package (K0118).
const RuntimeRT = gen.RuntimeModule + "/rt"

// Rules applies the rules that read the code of leaves and libraries.
func (c *Checker) Rules() {
	for _, pkg := range append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...) {
		gp := c.Full[pkg.ImportPath]
		if gp == nil || gp.TypesInfo == nil {
			continue
		}
		ports := c.portVars(pkg)
		var libVars map[types.Object]bool
		if !pkg.IsLeaf {
			libVars = packageVars(gp.Types)
		}
		for _, f := range gp.Syntax {
			if c.Generated(f) {
				continue
			}
			c.imports(pkg, f)
			if pkg.IsLeaf {
				c.leafRules(pkg, gp.TypesInfo, f, ports)
			}
			c.libVarAssigns(pkg, gp.TypesInfo, f, libVars)
			c.network(pkg, gp.TypesInfo, f)
		}
		if pkg.IsLeaf {
			var files []*ast.File
			for _, f := range gp.Syntax {
				if !c.Generated(f) && !strings.HasSuffix(c.Fset.PositionFor(f.Package, false).Filename, "_test.go") {
					files = append(files, f)
				}
			}
			c.initReach(pkg, files, gp.TypesInfo, ports)
		}
	}
	c.componentImports()
	c.contractsDeferred()
}

// portVars returns the package-level port variables of a leaf.
func (c *Checker) portVars(pkg *project.Package) map[types.Object]string {
	out := map[types.Object]string{}
	if !pkg.IsLeaf || pkg.Comp == nil {
		return out
	}
	for _, gp := range []*packages.Package{c.Full[pkg.ImportPath], c.Plain[pkg.ImportPath]} {
		if gp == nil || gp.Types == nil {
			continue
		}
		for _, port := range pkg.Comp.Ports {
			if v, ok := gp.Types.Scope().Lookup(port.Name).(*types.Var); ok {
				out[v] = port.Name
			}
		}
	}
	return out
}

func packageVars(p *types.Package) map[types.Object]bool {
	out := map[types.Object]bool{}
	if p == nil {
		return out
	}
	for _, name := range p.Scope().Names() {
		if v, ok := p.Scope().Lookup(name).(*types.Var); ok {
			out[v] = true
		}
	}
	return out
}

// imports applies rule 1 (K0101), K0204, and K0118 to the imports of one file.
func (c *Checker) imports(pkg *project.Package, f *ast.File) {
	for _, imp := range f.Imports {
		if imp.Name != nil && imp.Name.Name == "vulcan_rt" {
			continue // inserted by the lowering of contracts; K0005 stops code from naming it
		}
		path, _ := strconv.Unquote(imp.Path.Value)
		pos := c.Pos(imp.Path.Pos())
		if path == RuntimeRT || strings.HasPrefix(path, RuntimeRT+"/") {
			c.D.Errorf(pos, "K0118", "code imports %s, which only generated code may use; import %s", path, gen.RuntimeModule)
			continue
		}
		target := c.P.ByPath[path]
		if target == nil || !target.IsLeaf || target == pkg {
			continue
		}
		if pkg.IsLeaf {
			c.D.Errorf(pos, "K0101", "leaf %s imports leaf %s; a leaf uses another leaf only through a port that it needs (declare one in %s/component.vulcan, and wire it in system.vulcan)", pkg.Name, target.Name, pkg.Dir)
		} else {
			c.D.Errorf(pos, "K0204", "library %s imports leaf %s; a library imports only libraries", pkg.GoName, target.Name)
		}
	}
}

func (c *Checker) leafRules(pkg *project.Package, info *types.Info, f *ast.File, ports map[types.Object]string) {
	// Arguments of vulcan.Bind may take a port's address.
	bindArgs := map[ast.Node]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if fn := calledFunc(info, call); fn != nil && fn.Pkg() != nil && fn.Pkg().Path() == gen.RuntimeModule && fn.Name() == "Bind" && len(call.Args) >= 2 {
			bindArgs[call.Args[1]] = true
		}
		return true
	})
	portOf := func(e ast.Expr) (string, bool) {
		id, ok := ast.Unparen(e).(*ast.Ident)
		if !ok {
			return "", false
		}
		name, ok := ports[info.Uses[id]]
		return name, ok
	}
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && fd.Name.Name == "init" {
			c.D.Errorf(c.Pos(fd.Name.Pos()), "K0117", "leaf %s declares init, which would run before its ports are bound; use a constructor, setup, or startup (§5.4 rule 12)", pkg.Name)
		}
	}
	assigned := func(l ast.Expr) {
		if name, ok := portOf(l); ok {
			c.D.Errorf(c.Pos(l.Pos()), "K0116", "code assigns to port %s; a port is bound once, at startup (in a test, use vulcan.Bind)", name)
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			if n.Tok != token.DEFINE {
				for _, l := range n.Lhs {
					assigned(l)
				}
			}
		case *ast.RangeStmt:
			if n.Tok == token.ASSIGN {
				for _, l := range []ast.Expr{n.Key, n.Value} {
					if l != nil {
						assigned(l)
					}
				}
			}
		case *ast.UnaryExpr:
			if n.Op == token.AND && !bindArgs[n] {
				if name, ok := portOf(n.X); ok {
					c.D.Errorf(c.Pos(n.Pos()), "K0116", "code takes the address of port %s; only vulcan.Bind may", name)
				}
			}
		case *ast.CallExpr:
			c.bindFake(info, n, portOf)
		}
		return true
	})
}

// bindFake checks the fake of vulcan.Bind(t, &Port, fake) against the port (K0108).
func (c *Checker) bindFake(info *types.Info, call *ast.CallExpr, portOf func(ast.Expr) (string, bool)) {
	fn := calledFunc(info, call)
	if fn == nil || fn.Pkg() == nil || fn.Pkg().Path() != gen.RuntimeModule || fn.Name() != "Bind" || len(call.Args) != 3 {
		return
	}
	u, ok := ast.Unparen(call.Args[1]).(*ast.UnaryExpr)
	if !ok || u.Op != token.AND {
		return
	}
	name, ok := portOf(u.X)
	if !ok {
		return
	}
	iface, ok := info.TypeOf(u.X).Underlying().(*types.Interface)
	impl := info.TypeOf(call.Args[2])
	if !ok || impl == nil || types.Implements(impl, iface) {
		return
	}
	c.D.Errorf(c.Pos(call.Args[2].Pos()), "K0108", "%s", missingMessage(impl, iface, "the fake", name))
}

// initReach applies the second half of rule 12: the initializer of a package-level variable in
// a leaf cannot reach a port use, directly or through the leaf's functions and methods (K0117).
// A function literal that the initializer stores, and does not call, is not followed.
func (c *Checker) initReach(pkg *project.Package, files []*ast.File, info *types.Info, ports map[types.Object]string) {
	decls := map[*types.Func]*ast.FuncDecl{}
	for _, f := range files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Body != nil {
				if fn, ok := info.Defs[fd.Name].(*types.Func); ok {
					decls[fn] = fd
				}
			}
		}
	}
	type hit struct {
		port string
		via  []string
	}
	done := map[*types.Func]*hit{}
	var reach func(n ast.Node) *hit
	reach = func(root ast.Node) *hit {
		var found *hit
		called := map[*ast.FuncLit]bool{}
		ast.Inspect(root, func(n ast.Node) bool {
			if found != nil {
				return false
			}
			switch n := n.(type) {
			case *ast.FuncLit:
				return called[n]
			case *ast.CallExpr:
				if lit, ok := ast.Unparen(n.Fun).(*ast.FuncLit); ok {
					called[lit] = true
				}
				if fn := calledFunc(info, n); fn != nil {
					if fd := decls[fn]; fd != nil {
						h, seen := done[fn]
						if !seen {
							done[fn] = nil // a cycle reaches nothing new
							h = reach(fd.Body)
							done[fn] = h
						}
						if h != nil {
							found = &hit{port: h.port, via: append([]string{fn.Name()}, h.via...)}
							return false
						}
					}
				}
			case *ast.Ident:
				if name, ok := ports[info.Uses[n]]; ok {
					found = &hit{port: name}
				}
			}
			return true
		})
		return found
	}
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, s := range gd.Specs {
				for _, v := range s.(*ast.ValueSpec).Values {
					if h := reach(v); h != nil {
						path := ""
						if len(h.via) > 0 {
							path = " through " + strings.Join(h.via, " -> ")
						}
						c.D.Errorf(c.Pos(v.Pos()), "K0117", "the initializer of a package-level variable reaches port %s%s; ports are not bound yet when it runs, so do this in a constructor, setup, or startup (§5.4 rule 12)", h.port, path)
					}
				}
			}
		}
	}
}

func calledFunc(info *types.Info, call *ast.CallExpr) *types.Func {
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

// libVarAssigns applies §5.4a rule 1: a library var is not assigned after startup (K0202). An
// assignment to the var, to a field of a struct var, or to an element of an array var assigns
// the var itself. A change through a pointer, a slice, a map, or a method is not tracked.
func (c *Checker) libVarAssigns(pkg *project.Package, info *types.Info, f *ast.File, own map[types.Object]bool) {
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
	isLibVar := func(obj types.Object) bool {
		v, ok := obj.(*types.Var)
		if !ok || v.Pkg() == nil || v.Parent() != v.Pkg().Scope() {
			return false
		}
		lib := c.P.ByPath[v.Pkg().Path()]
		return lib != nil && !lib.IsLeaf
	}
	check := func(l ast.Expr, inInit bool) {
		obj := root(l)
		if !isLibVar(obj) || inInit && own[obj] {
			return
		}
		c.D.Errorf(c.Pos(l.Pos()), "K0202", "library variable %s is assigned after startup; a library var is set once, in its initializer or in an init function of its library (§5.4a rule 1)", obj.Name())
	}
	for _, d := range f.Decls {
		inInit := false
		if fd, ok := d.(*ast.FuncDecl); ok {
			inInit = fd.Recv == nil && fd.Name.Name == "init"
		}
		ast.Inspect(d, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if n.Tok != token.DEFINE {
					for _, l := range n.Lhs {
						check(l, inInit)
					}
				}
			case *ast.IncDecStmt:
				check(n.X, inInit)
			case *ast.RangeStmt:
				if n.Tok == token.ASSIGN {
					for _, l := range []ast.Expr{n.Key, n.Value} {
						if l != nil {
							check(l, inInit)
						}
					}
				}
			}
			return true
		})
	}
}

// componentImports applies rule 7: a boundary never names another leaf (K0110).
func (c *Checker) componentImports() {
	for _, l := range c.P.Leaves {
		if l.Comp == nil {
			continue
		}
		for _, imp := range l.Comp.Imports {
			if t := c.P.ByPath[imp.Path]; t != nil && t.IsLeaf {
				c.D.Errorf(l.CompSrc.Pos(imp.Pos), "K0110", "the boundary of %s imports leaf %s; a boundary names only libraries, the standard library, and predeclared types", l.Name, t.Name)
			}
		}
	}
}

// contractsDeferred reports contracts until phase P2 implements them.
func (c *Checker) contractsDeferred() {
	if ContractsEnabled {
		return
	}
	for _, pkg := range append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...) {
		for _, f := range pkg.Files {
			// One diagnostic for each declaration, at its first clause.
			for _, fn := range f.Blank.Funcs {
				c.P.Deferred(f.Pos(fn.Clauses[0].WordOff), "a contract ("+fn.Clauses[0].Word+") on a function", "§7.1")
			}
			for _, ty := range f.Blank.Types {
				c.P.Deferred(f.Pos(ty.Clauses[0].WordOff), "an invariant on a struct", "§7.2")
			}
		}
		if pkg.Comp != nil {
			for _, port := range pkg.Comp.Ports {
				for _, m := range port.Methods {
					if len(m.Contracts) > 0 {
						c.P.Deferred(pkg.CompSrc.Pos(m.Contracts[0].WordPos), "a contract ("+m.Contracts[0].Word+") on a port method", "§5.3")
					}
				}
			}
		}
	}
}

// ContractsEnabled turns on contracts (phase P2).
var ContractsEnabled = true
