// Package layout is analyzer A1 of vulcan-lint: the manifest against the packages of the module.
package layout

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A1.
var Analyzer = &analysis.Analyzer{
	Name: "layout",
	Doc:  "check the manifest (arch) against the packages of the module: every package has one part of a known kind, and each name is its directory (VL0002, VL0003, VL0004, VL0110, VL0121, VL0206, VL0207, VL0901, VL0902)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil {
		return nil, nil
	}
	first := firstFile(pass)
	if !u.M.Found {
		model.Report(pass, first.Package, "VL0902", "the module has no manifest: the package %s declares var Manifest = System{...}, which lists every package with its kind", model.ManifestDir)
		return nil, nil
	}
	ignored(pass, u, first)
	if u.Manifest {
		manifest(pass, u)
		return nil, nil
	}
	m := u.M
	if u.Part == nil {
		for _, d := range m.Dirs() {
			if p := m.ByDir[d]; p.IsLeaf() && strings.HasPrefix(u.Dir, d+"/") {
				model.Report(pass, first.Name.Pos(), "VL0002", "package %s is inside leaf %s; a leaf is one package, with no packages inside it", u.Dir, d)
				return nil, nil
			}
		}
		model.Report(pass, first.Name.Pos(), "VL0002", "package %s is not in the manifest; every package of the module is a part of the manifest (%s)", u.Dir, model.ManifestDir)
		return nil, nil
	}
	for _, f := range pass.Files {
		name := f.Name.Name
		switch {
		case strings.HasSuffix(name, "_test") && model.IsTest(pass, f):
			model.Report(pass, f.Name.Pos(), "VL0004", "the test package %s is an external test package; write the tests in package %s", name, strings.TrimSuffix(name, "_test"))
		case u.Part.Kind == model.Main && name != "main":
			model.Report(pass, f.Name.Pos(), "VL0004", "the main part %s holds package %s; a main part is package main", u.Dir, name)
		case u.Part.Kind != model.Main && name != path.Base(u.Dir):
			model.Report(pass, f.Name.Pos(), "VL0004", "the package clause names %s, but the directory is %s; a package is named after its directory", name, path.Base(u.Dir))
		}
	}
	return nil, nil
}

func firstFile(pass *analysis.Pass) *ast.File {
	for _, f := range pass.Files {
		if !model.IsTest(pass, f) {
			return f
		}
	}
	return pass.Files[0]
}

// ignored rejects the files of a project package that the build ignores, such as a file behind
// a build tag: no analyzer would read them.
func ignored(pass *analysis.Pass, u *model.Unit, first *ast.File) {
	for _, name := range pass.IgnoredFiles {
		rel, err := filepath.Rel(u.M.Root, name)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		model.Report(pass, first.Package, "VL0901", "the build ignores %s (a build constraint or a file name for another system); every file of a package is built, so remove the constraint", filepath.ToSlash(rel))
	}
}

// manifest checks the manifest itself, at its own positions.
func manifest(pass *analysis.Pass, u *model.Unit) {
	m := &model.Manifest{Root: u.M.Root, Module: u.M.Module}
	var files []*ast.File
	for _, f := range pass.Files {
		if !model.IsTest(pass, f) {
			files = append(files, f)
		}
	}
	model.Parse(m, pass.Fset, files)
	for _, p := range m.Problems {
		model.Report(pass, p.Pos, "VL0902", "%s", p.Msg)
	}
	for _, f := range pass.Files {
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			model.Report(pass, imp.Path.Pos(), "VL0110", "the manifest imports %s; it names directories, not packages, and imports nothing", p)
		}
	}
	seen := map[string]bool{}
	gateways := map[string]string{}
	for _, p := range m.Parts {
		if seen[p.Dir] {
			model.Report(pass, p.Pos, "VL0121", "the manifest lists %s twice; each package is one part", p.Dir)
			continue
		}
		seen[p.Dir] = true
		pkgName, hasCode := packageOf(filepath.Join(m.Root, filepath.FromSlash(p.Dir)))
		switch {
		case !hasCode:
			model.Report(pass, p.Pos, "VL0003", "the manifest lists %s, but no Go package is there; remove the part or add its package", p.Dir)
		case p.Kind == model.Library && !strings.HasPrefix(p.Dir, "lib/"):
			model.Report(pass, p.Pos, "VL0003", "library %s is outside lib/; a library lives in lib/<name>", p.Dir)
		case p.Kind != model.Library && strings.HasPrefix(p.Dir, "lib/"):
			model.Report(pass, p.Pos, "VL0003", "%s is in lib/, which holds only libraries", p.Dir)
		case p.Kind == model.Main && pkgName != "main":
			model.Report(pass, p.Pos, "VL0003", "main part %s holds package %s, not package main", p.Dir, pkgName)
		case p.Kind != model.Main && pkgName == "main":
			model.Report(pass, p.Pos, "VL0003", "%s is package main, so its part is Main(%q)", p.Dir, p.Dir)
		case p.Dir == model.ManifestDir:
			model.Report(pass, p.Pos, "VL0003", "the manifest is not a part of itself")
		}
		for _, q := range m.Parts {
			if q != p && q.IsLeaf() && strings.HasPrefix(p.Dir, q.Dir+"/") {
				model.Report(pass, p.Pos, "VL0002", "%s is inside leaf %s; a leaf is one package, with no packages inside it", p.Dir, q.Dir)
			}
		}
		switch p.Kind {
		case model.Gateway:
			if _, ok := m.Extern[p.For]; !ok {
				model.Report(pass, p.Pos, "VL0206", "gateway %s is for %s, which is not an external of the manifest (a const of type External)", p.Dir, p.For)
			} else if other, dup := gateways[p.For]; dup {
				model.Report(pass, p.Pos, "VL0207", "%s and %s are both gateways for %s; one gateway leaf for each external system", other, p.Dir, p.For)
			} else {
				gateways[p.For] = p.Dir
			}
		case model.Entry:
			_, person := m.Persons[p.For]
			_, ext := m.Extern[p.For]
			if !person && !ext {
				model.Report(pass, p.Pos, "VL0206", "entry %s is for %s, which is not a person or an external of the manifest", p.Dir, p.For)
			}
		}
	}
	for _, name := range sortedKeys(m.Exempt) {
		if p := m.ByDir[name]; p == nil || !p.IsLeaf() {
			model.Report(pass, m.ListPos["StrictExempt"], "VL0902", "StrictExempt names %s, which is not a leaf of the manifest", name)
		}
	}
}

// packageOf returns the package name of the Go files in dir, and whether it has any.
func packageOf(dir string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.PackageClauseOnly)
		if err == nil {
			return f.Name.Name, true
		}
	}
	return "", false
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
