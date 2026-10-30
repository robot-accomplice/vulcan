// Package rulecheck is analyzer A9 of vulcan-lint: the team's rules file (plan-r3, section 2). The
// rule kinds are in internal/rules, which `vulcan check` calls too, so each rule means the same in
// both. This package gives them Go's facts: the manifest's parts, and the wires of main.
package rulecheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strconv"
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
	Doc:  "apply the team's rules file, rules.vulcan: forbid-call, only-in, depends, and transitions (VL0801 to VL0806)",
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
		return nil, nil
	}
	if u.Part == nil {
		return nil, nil
	}
	parts := partsOf(u.M)
	if u.Part.Kind == model.Main {
		for _, f := range set.CheckWires(parts, Wires(u.M, pass.TypesInfo, plainFiles(pass))) {
			model.Report(pass, f.At.(token.Pos), "VL"+f.Code, "%s", f.Msg)
		}
		return nil, nil
	}
	unit := &rules.Unit{
		Part: parts[u.Part.Dir], Module: u.M.Module, Parts: parts, Fset: pass.Fset, Files: pass.Files, Info: pass.TypesInfo,
		IsTest: func(f *ast.File) bool { return model.IsTest(pass, f) },
	}
	for _, f := range set.CheckUnit(unit) {
		model.Report(pass, f.At.(token.Pos), "VL"+f.Code, "%s", f.Msg)
	}
	return nil, nil
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
					out = append(out, rules.Edge{From: need.Dir, To: sup, Via: "wire", At: a.Pos()})
				}
			}
			return true
		})
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
