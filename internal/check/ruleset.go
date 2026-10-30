package check

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/project"
	"github.com/robot-accomplice/vulcan/internal/rules"
)

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
		wires = append(wires, rules.Edge{From: b.Need.Dir, To: b.Inst.Leaf.Dir, Via: "wire", At: c.P.SysSrc.Pos(b.Wire.Pos)})
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
	for _, f := range set.CheckWires(parts, wires) {
		c.D.Errorf(f.At.(diag.Pos), "K"+f.Code, "%s", f.Msg)
	}
	for _, pkg := range all {
		gp := c.Full[pkg.ImportPath]
		if gp == nil || gp.TypesInfo == nil {
			continue
		}
		u := &rules.Unit{
			Part: parts[pkg.Dir], Module: c.P.Module, Parts: parts, Fset: c.Fset, Info: gp.TypesInfo,
			IsTest: func(f *ast.File) bool {
				return strings.HasSuffix(c.Fset.PositionFor(f.Package, false).Filename, "_test.go")
			},
		}
		for _, f := range gp.Syntax {
			if !c.Generated(f) {
				u.Files = append(u.Files, f)
			} else if strings.HasSuffix(c.Fset.PositionFor(f.Package, false).Filename, "vulcan_ports.go") {
				u.PortFiles = append(u.PortFiles, f)
			}
		}
		for _, f := range set.CheckUnit(u) {
			c.D.Errorf(c.Pos(f.At.(token.Pos)), "K"+f.Code, "%s", f.Msg)
		}
	}
	return edges
}
