// Package rulecheck is analyzer A9 of vulcan-lint: the team's rules file (plan-r3, section 2). The
// rule kinds are in internal/rules, which `vulcan check` calls too, so each rule means the same in
// both. This package gives them Go's facts: the manifest's parts, and the wires of main.
package rulecheck

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"github.com/robot-accomplice/vulcan/internal/rules"
	"github.com/robot-accomplice/vulcan/poc/golint/model"
	"github.com/robot-accomplice/vulcan/poc/golint/ports"
)

// Analyzer is A9.
var Analyzer = &analysis.Analyzer{
	Name: "rules",
	Doc:  "apply the team's rules file, rules.vulcan: forbid-call, only-in, depends, transitions, declaration, call-form, literal, and code-form (VL0801 to VL0810)",
	Run:  run,
}

var cache sync.Map // root -> *rules.Set

func load(m *model.Manifest) *rules.Set {
	if s, ok := cache.Load(m.Root); ok {
		return s.(*rules.Set)
	}
	s := rules.Load(m.Root)
	if s != nil {
		s.Resolve(partsOf(m))
	}
	v, _ := cache.LoadOrStore(m.Root, s)
	return v.(*rules.Set)
}

func partsOf(m *model.Manifest) map[string]rules.Part {
	out := map[string]rules.Part{}
	for dir, p := range m.ByDir {
		out[dir] = rules.Part{Dir: dir, Kind: p.Kind}
	}
	return out
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil {
		return nil, nil
	}
	set := load(u.M)
	if set == nil {
		return nil, nil
	}
	if u.Manifest {
		problems(pass, u, set)
		for _, f := range set.CheckDecls(manifestDecls(pass)) {
			emit(pass, f.At.(token.Pos), f)
		}
		return nil, nil
	}
	if u.Part == nil {
		return nil, nil
	}
	parts := partsOf(u.M)
	if u.Part.Kind == model.Main {
		for _, f := range set.CheckWires(parts, Wires(u.M, pass.TypesInfo, plainFiles(pass))) {
			emit(pass, f.At.(token.Pos), f)
		}
		return nil, nil
	}
	for _, f := range set.CheckDecls(libraryDecl(pass, u)) {
		emit(pass, f.At.(token.Pos), f)
	}
	unit := &rules.Unit{
		Part: parts[u.Part.Dir], Module: u.M.Module, Parts: parts, Fset: pass.Fset, Files: pass.Files, Info: pass.TypesInfo,
		IsTest: func(f *ast.File) bool { return model.IsTest(pass, f) },
		Ports:  portsOf(pass, u), TestsAbsent: testsAbsent(pass, u),
		HasEnsures: func(fd *ast.FuncDecl) bool {
			return len(model.PrologueOf(pass.TypesInfo, fd.Body, u.M.ContractPath()).Ensures) > 0
		},
	}
	for _, f := range set.CheckUnit(unit) {
		emit(pass, f.At.(token.Pos), f)
	}
	return nil, nil
}

var warned sync.Map

// emit reports a finding. A go/analysis driver has no warning level, and a diagnostic fails the
// check, so a rule with level warn prints its finding and leaves the check to pass.
func emit(pass *analysis.Pass, at token.Pos, f rules.Finding) {
	if !f.Warn {
		model.Report(pass, at, "VL"+f.Code, "%s", f.Msg)
		return
	}
	line := fmt.Sprintf("%s: warning VL%s %s", pass.Fset.Position(at), f.Code, f.Msg)
	if _, dup := warned.LoadOrStore(line, true); !dup {
		fmt.Fprintln(os.Stderr, line)
	}
}

// testsAbsent reports a package whose test files this pass does not hold: the driver reads each
// package without its tests too.
func testsAbsent(pass *analysis.Pass, u *model.Unit) bool {
	for _, f := range pass.Files {
		if model.IsTest(pass, f) {
			return false
		}
	}
	entries, err := os.ReadDir(filepath.Join(u.M.Root, filepath.FromSlash(u.Dir)))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// portsOf gives the exported interfaces of ports.go, with the contracts of their wrappers.
func portsOf(pass *analysis.Pass, u *model.Unit) []rules.Port {
	if !u.Part.IsLeaf() {
		return nil
	}
	info := pass.TypesInfo
	requires := map[string]bool{}
	var out []rules.Port
	for _, f := range pass.Files {
		if model.FileName(pass, f) != "ports.go" {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			if tn := model.NamedOf(info.TypeOf(fd.Recv.List[0].Type)); tn != nil && len(model.PrologueOf(info, fd.Body, u.M.ContractPath()).Requires) > 0 {
				requires[strings.TrimPrefix(tn.Name(), "checked")+"."+fd.Name.Name] = true
			}
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, sp := range gd.Specs {
				ts := sp.(*ast.TypeSpec)
				it, ok := ts.Type.(*ast.InterfaceType)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				obj := info.Defs[ts.Name]
				if obj == nil {
					continue
				}
				iface, ok := obj.Type().Underlying().(*types.Interface)
				if !ok {
					continue
				}
				p := rules.Port{Name: ts.Name.Name, Type: obj.Type(), At: ts.Name.Pos(), Doc: gd.Doc != nil || ts.Doc != nil}
				for _, fd := range it.Methods.List {
					for _, id := range fd.Names {
						pm := rules.PortMethod{Name: id.Name, At: id.Pos(), Doc: fd.Doc != nil || fd.Comment != nil, Requires: requires[ts.Name.Name+"."+id.Name]}
						for i := range iface.NumExplicitMethods() {
							if m := iface.ExplicitMethod(i); m.Name() == id.Name {
								pm.Sig = m.Type().(*types.Signature)
							}
						}
						p.Methods = append(p.Methods, pm)
					}
				}
				out = append(out, p)
			}
		}
	}
	return out
}

// libraryDecl gives the library that a pass reads, with whether it has a package comment.
func libraryDecl(pass *analysis.Pass, u *model.Unit) []rules.Decl {
	if u.Part.Kind != model.Library || rules.IsRuntimeLib(u.Dir) {
		return nil
	}
	d := rules.Decl{What: "library", Name: path.Base(u.Dir), At: token.NoPos}
	for _, f := range pass.Files {
		if model.IsTest(pass, f) {
			continue
		}
		if !d.At.(token.Pos).IsValid() {
			d.At = f.Package
		}
		d.Commented = d.Commented || f.Doc != nil
	}
	return []rules.Decl{d}
}

// manifestDecls gives the externals and the entries of the check configuration.
func manifestDecls(pass *analysis.Pass) []rules.Decl {
	var out []rules.Decl
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range gd.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if t, ok := vs.Type.(*ast.Ident); ok && gd.Tok == token.CONST && t.Name == "External" {
					for _, id := range vs.Names {
						out = append(out, rules.Decl{What: "external", Name: id.Name, Commented: gd.Doc != nil || vs.Doc != nil || vs.Comment != nil, At: id.Pos()})
					}
				}
				if gd.Tok != token.VAR {
					continue
				}
				for i, id := range vs.Names {
					if id.Name != "Config" || i >= len(vs.Values) {
						continue
					}
					lit, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					for _, e := range lit.Elts {
						kv, ok := e.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						list, ok := kv.Value.(*ast.CompositeLit)
						if !ok {
							continue
						}
						for _, x := range list.Elts {
							line := pass.Fset.Position(x.Pos()).Line
							commented := false
							for _, cg := range f.Comments {
								for _, c := range cg.List {
									if l := pass.Fset.Position(c.End()).Line; l == line || l == line-1 {
										commented = true
									}
								}
							}
							name := strings.Trim(types.ExprString(x), `"`)
							out = append(out, rules.Decl{What: "config-entry", Name: name, Commented: commented, At: x.Pos()})
						}
					}
				}
			}
		}
	}
	return out
}

func plainFiles(pass *analysis.Pass) []*ast.File {
	var out []*ast.File
	for _, f := range pass.Files {
		if !model.IsTest(pass, f) {
			out = append(out, f)
		}
	}
	return out
}

// problems reports what is wrong with the rules file, at its own lines.
func problems(pass *analysis.Pass, u *model.Unit, set *rules.Set) {
	if len(set.Probs) == 0 {
		return
	}
	file := filepath.Join(u.M.Root, rules.FileName)
	b, err := os.ReadFile(file)
	if err != nil {
		return
	}
	tf := pass.Fset.AddFile(file, -1, len(b)+1)
	tf.SetLinesForContent(b)
	for _, p := range set.Probs {
		line := min(max(p.Line, 1), tf.LineCount())
		model.Report(pass, tf.LineStart(line), "VL0801", "%s", p.Msg)
	}
}

// Wires gives the wires that main makes: each port that a constructor call gets, with the leaf that
// provides it. The provider is the package of the argument's type. For an argument of an interface
// type, it is the type of the value that main assigned to the variable.
func Wires(m *model.Manifest, info *types.Info, files []*ast.File) []rules.Edge {
	from := map[types.Object]ast.Expr{} // the value that main assigned to each variable
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				if len(x.Rhs) == 1 && len(x.Lhs) >= 1 {
					if id, ok := x.Lhs[0].(*ast.Ident); ok {
						record(info, from, id, x.Rhs[0])
					}
				} else if len(x.Rhs) == len(x.Lhs) {
					for i, l := range x.Lhs {
						if id, ok := l.(*ast.Ident); ok {
							record(info, from, id, x.Rhs[i])
						}
					}
				}
			case *ast.ValueSpec:
				for i, id := range x.Names {
					if i < len(x.Values) {
						record(info, from, id, x.Values[i])
					}
				}
			}
			return true
		})
	}
	var out []rules.Edge
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fn := model.CalledFunc(info, call)
			if fn == nil || fn.Pkg() == nil {
				return true
			}
			need := m.PartOf(fn.Pkg().Path())
			if need == nil || !need.IsLeaf() || !ports.IsConstructor(fn) {
				return true
			}
			params := fn.Type().(*types.Signature).Params()
			for i, a := range call.Args {
				if i >= params.Len() || !types.IsInterface(params.At(i).Type()) {
					continue
				}
				if tv, ok := info.Types[a]; ok && tv.IsNil() {
					continue // VL0104
				}
				if sup := provider(m, info, from, a, 0); sup != "" && sup != need.Dir {
					out = append(out, rules.Edge{From: need.Dir, To: sup, Via: "wire", At: a.Pos(), Methods: portMethods(params.At(i).Type())})
				}
			}
			return true
		})
	}
	return out
}

func portMethods(t types.Type) []rules.PortMethod {
	iface, ok := t.Underlying().(*types.Interface)
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

func record(info *types.Info, from map[types.Object]ast.Expr, id *ast.Ident, v ast.Expr) {
	obj := info.Defs[id]
	if obj == nil {
		obj = info.Uses[id]
	}
	if obj != nil {
		from[obj] = v
	}
}

func provider(m *model.Manifest, info *types.Info, from map[types.Object]ast.Expr, e ast.Expr, depth int) string {
	if depth > 16 {
		return ""
	}
	e = ast.Unparen(e)
	if t := info.TypeOf(e); t != nil && !types.IsInterface(t) {
		if tup, ok := t.(*types.Tuple); ok && tup.Len() > 0 {
			t = tup.At(0).Type()
		}
		if tn := model.NamedOf(t); tn != nil && tn.Pkg() != nil {
			if p := m.PartOf(tn.Pkg().Path()); p != nil {
				return p.Dir
			}
		}
	}
	switch x := e.(type) {
	case *ast.Ident:
		if v, ok := from[info.Uses[x]]; ok {
			return provider(m, info, from, v, depth+1)
		}
	case *ast.CallExpr:
		if fn := model.CalledFunc(info, x); fn != nil && fn.Pkg() != nil {
			if p := m.PartOf(fn.Pkg().Path()); p != nil && p.IsLeaf() {
				return p.Dir
			}
		}
	}
	return ""
}

// Edges gives what rules lock writes: the wires of main, and the imports of libraries. It refuses a
// module that does not type-check.
func Edges(root string) ([]rules.Edge, error) {
	m := model.Read(root)
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Dir:  root,
		Fset: token.NewFileSet(),
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	if n := packages.PrintErrors(pkgs); n > 0 {
		return nil, errBuild
	}
	var out []rules.Edge
	for _, pkg := range pkgs {
		part := m.PartOf(pkg.PkgPath)
		if part == nil {
			continue
		}
		if part.Kind == model.Main {
			for _, e := range Wires(m, pkg.TypesInfo, pkg.Syntax) {
				e.At = nil
				out = append(out, e)
			}
			continue
		}
		for _, f := range pkg.Syntax {
			for _, imp := range f.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				t := m.PartOf(path)
				if t != nil && t != part && t.Kind == model.Library && !rules.IsRuntimeLib(t.Dir) {
					out = append(out, rules.Edge{From: part.Dir, To: t.Dir, Via: "import"})
				}
			}
		}
	}
	return out, nil
}

type buildError struct{}

func (buildError) Error() string { return "the module does not build; fix it first" }

var errBuild = buildError{}
