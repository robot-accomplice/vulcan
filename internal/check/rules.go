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
			c.names(gp.TypesInfo, f)
			if pkg.IsLeaf {
				c.leafRules(pkg, gp.TypesInfo, f, ports)
			}
			c.libVarAssigns(pkg, gp.TypesInfo, f, libVars)
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

// names applies the reserved-name rule (K0005).
func (c *Checker) names(info *types.Info, f *ast.File) {
	for id := range info.Defs {
		if id == nil || c.Fset.File(id.Pos()) != c.Fset.File(f.Pos()) {
			continue
		}
		if strings.HasPrefix(id.Name, "vulcan_") || strings.HasPrefix(id.Name, "Vulcan_") {
			c.D.Errorf(c.Pos(id.Pos()), "K0005", "the name %s starts with %s, which is reserved for generated code", id.Name, id.Name[:7])
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
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "init" {
				c.D.Errorf(c.Pos(d.Name.Pos()), "K0117", "leaf %s declares init, which would run before its ports are bound; use a constructor, setup, or startup (§5.4 rule 12)", pkg.Name)
			}
		case *ast.GenDecl:
			if d.Tok != token.VAR {
				continue
			}
			for _, s := range d.Specs {
				for _, v := range s.(*ast.ValueSpec).Values {
					ast.Inspect(v, func(n ast.Node) bool {
						if id, ok := n.(*ast.Ident); ok {
							if name, ok := ports[info.Uses[id]]; ok {
								c.D.Errorf(c.Pos(id.Pos()), "K0117", "the initializer of a package-level variable uses port %s, which is not bound yet when it runs (§5.4 rule 12)", name)
							}
						}
						return true
					})
				}
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, l := range n.Lhs {
				if name, ok := portOf(l); ok && n.Tok != token.DEFINE {
					c.D.Errorf(c.Pos(l.Pos()), "K0116", "code assigns to port %s; a port is bound once, at startup (in a test, use vulcan.Bind)", name)
				}
			}
		case *ast.UnaryExpr:
			if n.Op == token.AND && !bindArgs[n] {
				if name, ok := portOf(n.X); ok {
					c.D.Errorf(c.Pos(n.Pos()), "K0116", "code takes the address of port %s; only vulcan.Bind may", name)
				}
			}
		}
		return true
	})
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

// libVarAssigns applies §5.4a rule 1: a library var is not assigned after startup (K0202).
func (c *Checker) libVarAssigns(pkg *project.Package, info *types.Info, f *ast.File, own map[types.Object]bool) {
	isLibVar := func(obj types.Object) bool {
		v, ok := obj.(*types.Var)
		if !ok || v.Pkg() == nil || v.Parent() != v.Pkg().Scope() {
			return false
		}
		lib := c.P.ByPath[v.Pkg().Path()]
		return lib != nil && !lib.IsLeaf
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		inInit := fd.Recv == nil && fd.Name.Name == "init"
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			var lhs []ast.Expr
			switch n := n.(type) {
			case *ast.AssignStmt:
				if n.Tok == token.DEFINE {
					return true
				}
				lhs = n.Lhs
			case *ast.IncDecStmt:
				lhs = []ast.Expr{n.X}
			default:
				return true
			}
			for _, l := range lhs {
				var id *ast.Ident
				switch l := ast.Unparen(l).(type) {
				case *ast.Ident:
					id = l
				case *ast.SelectorExpr:
					id = l.Sel
				}
				if id == nil {
					continue
				}
				obj := info.Uses[id]
				if !isLibVar(obj) {
					continue
				}
				if inInit && own[obj] {
					continue
				}
				c.D.Errorf(c.Pos(id.Pos()), "K0202", "library variable %s is assigned after startup; a library var is set once, in its initializer or in init (§5.4a rule 1)", obj.Name())
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
var ContractsEnabled = false
