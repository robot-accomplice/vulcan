package check

import (
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/gen"
	"github.com/robot-accomplice/vulcan/internal/project"
	"github.com/robot-accomplice/vulcan/internal/rules"
)

// wireMethods gives the methods of the port that a wire binds.
func (c *Checker) wireMethods(b Bind) []rules.PortMethod {
	gp := c.Plain[b.Need.ImportPath]
	if gp == nil || gp.Types == nil {
		return nil
	}
	tn, ok := gp.Types.Scope().Lookup(gen.PortType(b.Port)).(*types.TypeName)
	if !ok {
		return nil
	}
	iface, ok := tn.Type().Underlying().(*types.Interface)
	if !ok {
		return nil
	}
	var out []rules.PortMethod
	for i := range iface.NumExplicitMethods() {
		m := iface.ExplicitMethod(i)
		out = append(out, rules.PortMethod{Name: m.Name(), Sig: m.Type().(*types.Signature)})
	}
	return out
}

// ports gives the ports of a leaf.
func (c *Checker) ports(pkg *project.Package, gp *packages.Package) []rules.Port {
	if !pkg.IsLeaf || pkg.Comp == nil || gp.Types == nil {
		return nil
	}
	var out []rules.Port
	comments := rules.ScanComments(pkg.CompSrc.Src)
	for _, port := range pkg.Comp.Ports {
		tn, ok := gp.Types.Scope().Lookup(gen.PortType(port.Name)).(*types.TypeName)
		if !ok {
			continue
		}
		iface, ok := tn.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		p := rules.Port{Name: port.Name, Type: tn.Type(), At: tn.Pos(), Doc: comments.Near(pkg.CompSrc.Pos(port.Pos).Line)}
		for _, m := range port.Methods {
			pm := rules.PortMethod{Name: m.Name, At: tn.Pos(), Doc: comments.Near(pkg.CompSrc.Pos(m.Pos).Line)}
			for i := range iface.NumExplicitMethods() {
				if x := iface.ExplicitMethod(i); x.Name() == m.Name {
					pm.Sig, pm.At = x.Type().(*types.Signature), x.Pos()
				}
			}
			for _, cl := range m.Contracts {
				pm.Requires = pm.Requires || cl.Word == "requires"
			}
			p.Methods = append(p.Methods, pm)
		}
		out = append(out, p)
	}
	return out
}

// buildFile maps a .vulcan file of a position to its copy in the build directory, which is Go.
func (c *Checker) buildFile(file string) string {
	if !strings.HasSuffix(file, ".vulcan") {
		return file
	}
	return filepath.Join(gen.BuildDir(c.P), filepath.FromSlash(gen.GoFileName(c.rel(file))))
}

// ensuresOf says whether a function of a package has an ensures clause.
func ensuresOf(pkg *project.Package) func(*ast.FuncDecl) bool {
	with := map[string]bool{}
	for _, f := range pkg.Files {
		for _, fn := range f.Blank.Funcs {
			for _, cl := range fn.Clauses {
				if cl.Word == "ensures" {
					with[fn.Recv+"."+fn.Name] = true
				}
			}
		}
	}
	return func(fd *ast.FuncDecl) bool {
		recv := ""
		if fd.Recv != nil && len(fd.Recv.List) == 1 {
			t := fd.Recv.List[0].Type
			if s, ok := t.(*ast.StarExpr); ok {
				t = s.X
			}
			switch g := t.(type) {
			case *ast.IndexExpr:
				t = g.X
			case *ast.IndexListExpr:
				t = g.X
			}
			if id, ok := t.(*ast.Ident); ok {
				recv = id.Name
			}
		}
		return with[recv+"."+fd.Name.Name]
	}
}

// testsAbsent reports a package with test files that this load left out.
func testsAbsent(pkg *project.Package, gp *packages.Package) bool {
	has := false
	for _, f := range pkg.Files {
		has = has || f.Test
	}
	if !has {
		return false
	}
	for _, name := range gp.GoFiles {
		if strings.HasSuffix(name, "_test.go") {
			return false
		}
	}
	return true
}

// decls gives what is not Go code: the externals, the libraries, and the check configuration.
func (c *Checker) decls() []rules.Decl {
	var out []rules.Decl
	sys, src := c.P.System, c.P.SysSrc
	comments := rules.ScanComments(src.Src)
	if sys != nil {
		for _, n := range sys.Externals {
			pos := src.Pos(n.Pos)
			out = append(out, rules.Decl{What: "external", Name: n.Name, Commented: n.Desc != "" || comments.Near(pos.Line), At: pos})
		}
	}
	for _, key := range []string{"strict_exempt", "strict_interfaces", "strict_shared", "mutators"} {
		v := c.P.Config(key)
		if v == nil {
			continue
		}
		for _, it := range v.List {
			pos := src.Pos(it.Pos)
			name := strings.Trim(strings.TrimSpace(it.Text), `"`)
			out = append(out, rules.Decl{What: "config-entry", Name: name, Commented: comments.Near(pos.Line), At: pos})
		}
	}
	for _, lib := range c.P.Libs {
		d := rules.Decl{What: "library", Name: lib.GoName, At: diag.Pos{File: lib.Dir, Line: 0}}
		if lib.LibSrc != nil && lib.Lib != nil {
			pos := lib.LibSrc.Pos(lib.Lib.Pos)
			d.Commented, d.At = rules.ScanComments(lib.LibSrc.Src).Near(pos.Line), pos
		}
		out = append(out, d)
	}
	return out
}

// partKind gives the kind of a leaf or a library in a rules file.
func partKind(pkg *project.Package) string {
	switch {
	case !pkg.IsLeaf:
		return "library"
	case pkg.Gateway:
		return "gateway"
	case pkg.EntryLeaf:
		return "entry"
	}
	return "leaf"
}

// ApplyRules applies the team's rules file (plan-r3, section 2): K0801 to K0806. It reads
// rules.vulcan and, when a rule needs it, rules.lock, and nothing else. It returns the current
// wires and the imports of libraries, which rules lock writes.
func (c *Checker) ApplyRules(plan *Plan) []rules.Edge {
	parts := map[string]rules.Part{}
	all := append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...)
	for _, pkg := range all {
		parts[pkg.Dir] = rules.Part{Dir: pkg.Dir, Kind: partKind(pkg)}
	}
	var wires []rules.Edge
	for _, b := range plan.Binds {
		wires = append(wires, rules.Edge{From: b.Need.Dir, To: b.Inst.Leaf.Dir, Via: "wire", At: c.P.SysSrc.Pos(b.Wire.Pos), Methods: c.wireMethods(b)})
	}
	edges := append([]rules.Edge{}, wires...)
	for _, pkg := range all {
		gp := c.Plain[pkg.ImportPath]
		if gp == nil {
			continue
		}
		for path := range gp.Imports {
			if t := c.P.ByPath[path]; t != nil && t != pkg && !t.IsLeaf && !rules.IsRuntimeLib(t.Dir) {
				edges = append(edges, rules.Edge{From: pkg.Dir, To: t.Dir, Via: "import"})
			}
		}
	}
	set := rules.Load(c.P.Root)
	if set == nil {
		return edges
	}
	set.Resolve(parts)
	for _, p := range set.Probs {
		c.D.Errorf(diag.Pos{File: rules.FileName, Line: p.Line, Col: 1}, "K0801", "%s", p.Msg)
	}
	report := func(pos diag.Pos, f rules.Finding) {
		if f.Warn {
			c.D.Warnf(pos, "K"+f.Code, "%s", f.Msg)
		} else {
			c.D.Errorf(pos, "K"+f.Code, "%s", f.Msg)
		}
	}
	for _, f := range set.CheckWires(parts, wires) {
		report(f.At.(diag.Pos), f)
	}
	for _, f := range set.CheckDecls(c.decls()) {
		report(f.At.(diag.Pos), f)
	}
	for _, pkg := range all {
		gp := c.Full[pkg.ImportPath]
		// As L's driver does, a rule does not read a package with a type error. A type error in a
		// test leaves the package without its tests, which L's driver reads too.
		if gp != nil && len(gp.Errors) > 0 {
			gp = c.Plain[pkg.ImportPath]
		}
		if gp == nil || gp.TypesInfo == nil || len(gp.Errors) > 0 {
			continue
		}
		u := &rules.Unit{
			Part: parts[pkg.Dir], Module: c.P.Module, Parts: parts, Fset: c.Fset, Info: gp.TypesInfo,
			IsTest: func(f *ast.File) bool {
				return strings.HasSuffix(c.Fset.PositionFor(f.Package, false).Filename, "_test.go")
			},
			Pkg: gp.Types, Generates: true, SourceFile: c.buildFile,
			Ports:       c.ports(pkg, gp),
			HasEnsures:  ensuresOf(pkg),
			TestsAbsent: testsAbsent(pkg, gp),
		}
		for _, f := range gp.Syntax {
			if !c.Generated(f) {
				u.Files = append(u.Files, f)
			} else if strings.HasSuffix(c.Fset.PositionFor(f.Package, false).Filename, "vulcan_ports.go") {
				u.PortFiles = append(u.PortFiles, f)
			}
		}
		for _, p := range set.ResolveTypes(u) {
			c.D.Errorf(diag.Pos{File: rules.FileName, Line: p.Line, Col: 1}, "K0801", "%s", p.Msg)
		}
		for _, f := range set.CheckUnit(u) {
			report(c.Pos(f.At.(token.Pos)), f)
		}
	}
	return edges
}
