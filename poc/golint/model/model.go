// Package model reads the manifest of a project's architecture and classifies its packages, for
// the analyzers of vulcan-lint (README.md).
//
// The manifest is the Go package arch at the root of the module. It declares the persons and the
// externals as typed constants, one variable Manifest, a System literal that lists every package
// of the module with its kind, and, in config.go, one variable Config, a Settings literal that
// holds the check configuration:
//
//	const Customer Person = "Customer"
//	const Database External = "Database"
//
//	var Manifest = System{
//		Parts: []Part{
//			Main("cmd/shop"),
//			Entry("api/web", Customer),
//			Leaf("api/orders"),
//			Gateway("api/store", Database),
//			Library("lib/shop"),
//		},
//	}
//
//	var Config = Settings{Strict: true} // in config.go
//
// The build type-checks the manifest; the analyzers read its syntax.
package model

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/analysis"
)

// ManifestDir is the directory of the manifest package, relative to the module root.
const ManifestDir = "arch"

// ConfigFile is the file of the manifest package that declares the check configuration.
const ConfigFile = "config.go"

// ContractDir is the directory of the contract package, relative to the module root.
const ContractDir = "lib/contract"

// The kinds of a part.
const (
	Leaf    = "leaf"
	Gateway = "gateway"
	Entry   = "entry"
	Library = "library"
	Main    = "main"
)

// Part is one package of the manifest.
type Part struct {
	Dir  string // relative to the module root, with slashes
	Kind string
	For  string // the external of a gateway, the person or external of an entry
	Pos  token.Pos
}

// IsLeaf reports whether the part is a leaf of any kind: a leaf, a gateway, or an entry.
func (p *Part) IsLeaf() bool { return p.Kind == Leaf || p.Kind == Gateway || p.Kind == Entry }

// Problem is a manifest that the analyzers cannot read.
type Problem struct {
	Pos token.Pos
	Msg string
}

// Manifest is the architecture of a module.
type Manifest struct {
	Root    string // the module's directory
	Module  string // the module path
	Found   bool   // the manifest package exists
	Persons map[string]token.Pos
	Extern  map[string]token.Pos
	Parts   []*Part
	ByDir   map[string]*Part

	Strict     bool
	Exempt     map[string]bool // dirs of leaves that strict mode skips
	Interfaces map[string]bool // interface types a port method may carry
	Shared     map[string]bool // types a library var may have
	Mutators   map[string]bool // more global mutators
	ListPos    map[string]token.Pos

	Problems []Problem
}

// ContractPath is the import path of the contract package.
func (m *Manifest) ContractPath() string { return m.Module + "/" + ContractDir }

// ManifestPath is the import path of the manifest package.
func (m *Manifest) ManifestPath() string { return m.Module + "/" + ManifestDir }

// PartOf returns the part of an import path of the module, or nil.
func (m *Manifest) PartOf(importPath string) *Part {
	rel, ok := m.Rel(importPath)
	if !ok {
		return nil
	}
	return m.ByDir[rel]
}

// Rel returns the directory of an import path relative to the module root.
func (m *Manifest) Rel(importPath string) (string, bool) {
	if importPath == m.Module {
		return ".", true
	}
	if strings.HasPrefix(importPath, m.Module+"/") {
		return strings.TrimPrefix(importPath, m.Module+"/"), true
	}
	return "", false
}

// InModule reports whether an import path is in the module.
func (m *Manifest) InModule(importPath string) bool {
	_, ok := m.Rel(importPath)
	return ok
}

// StrictFor reports whether strict mode applies to a part.
func (m *Manifest) StrictFor(p *Part) bool {
	return m.Strict && p != nil && p.Kind != Main && !(p.IsLeaf() && m.Exempt[p.Dir])
}

// Unit is the package that an analyzer pass sees, placed in its module.
type Unit struct {
	M        *Manifest
	Dir      string // relative to the module root
	Part     *Part  // nil when the manifest does not list the directory
	Manifest bool   // the package is the manifest
	Contract bool   // the package is the contract package
}

var cache sync.Map // root -> *Manifest

// For places the package of a pass. It returns nil for a package outside a module with a
// go.mod, such as the generated main of a test binary.
func For(pass *analysis.Pass) *Unit {
	var file string
	for _, f := range pass.Files {
		if name := pass.Fset.PositionFor(f.Package, false).Filename; name != "" {
			file = name
			break
		}
	}
	if file == "" {
		return nil
	}
	dir := filepath.Dir(file)
	root := findRoot(dir)
	if root == "" {
		return nil
	}
	m := Load(root)
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return nil
	}
	rel = filepath.ToSlash(rel)
	u := &Unit{M: m, Dir: rel, Part: m.ByDir[rel]}
	u.Manifest = rel == ManifestDir
	u.Contract = rel == ContractDir
	return u
}

func findRoot(dir string) string {
	for d := dir; ; {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// Load reads the manifest of the module at root; the result is cached.
func Load(root string) *Manifest {
	if m, ok := cache.Load(root); ok {
		return m.(*Manifest)
	}
	v, _ := cache.LoadOrStore(root, Read(root))
	return v.(*Manifest)
}

// Read reads the manifest of the module at root, with no cache.
func Read(root string) *Manifest {
	m := &Manifest{Root: root}
	if b, err := os.ReadFile(filepath.Join(root, "go.mod")); err == nil {
		m.Module = modfile.ModulePath(b)
	}
	fset := token.NewFileSet()
	files := parseDir(fset, filepath.Join(root, ManifestDir))
	Parse(m, fset, files)
	return m
}

func parseDir(fset *token.FileSet, dir string) []*ast.File {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution)
		if err == nil {
			out = append(out, f)
		}
	}
	return out
}

// Parse fills m from the syntax of the manifest package. The positions are those of fset, so
// the analyzer of the manifest package can parse its own files and report at them.
func Parse(m *Manifest, fset *token.FileSet, files []*ast.File) {
	m.Persons, m.Extern = map[string]token.Pos{}, map[string]token.Pos{}
	m.ByDir = map[string]*Part{}
	m.Exempt, m.Interfaces, m.Shared, m.Mutators = map[string]bool{}, map[string]bool{}, map[string]bool{}, map[string]bool{}
	m.ListPos = map[string]token.Pos{}
	m.Parts, m.Problems, m.Found, m.Strict = nil, nil, len(files) > 0, false
	bad := func(n ast.Node, format string, args ...any) {
		m.Problems = append(m.Problems, Problem{n.Pos(), fmt.Sprintf(format, args...)})
	}
	var lit, cfg *ast.CompositeLit
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range gd.Specs {
				vs, ok := s.(*ast.ValueSpec)
				if !ok {
					continue
				}
				switch gd.Tok {
				case token.CONST:
					t, _ := vs.Type.(*ast.Ident)
					if t == nil || (t.Name != "Person" && t.Name != "External") {
						continue
					}
					for _, n := range vs.Names {
						if t.Name == "Person" {
							m.Persons[n.Name] = n.Pos()
						} else {
							m.Extern[n.Name] = n.Pos()
						}
					}
				case token.VAR:
					for i, n := range vs.Names {
						var v *ast.CompositeLit
						if i < len(vs.Values) {
							v, _ = vs.Values[i].(*ast.CompositeLit)
						}
						switch n.Name {
						case "Manifest":
							if lit = v; lit == nil {
								bad(n, "Manifest is a System{...} literal")
							}
						case "Config":
							if cfg = v; cfg == nil {
								bad(n, "Config is a Settings{...} literal")
							} else if filepath.Base(fset.Position(n.Pos()).Filename) != ConfigFile {
								bad(n, "Config is declared in %s/%s, which holds the check configuration", ManifestDir, ConfigFile)
							}
						}
					}
				}
			}
		}
	}
	if len(files) == 0 {
		return
	}
	if lit == nil {
		if len(m.Problems) == 0 {
			bad(files[0].Name, "the manifest package declares no var Manifest = System{...}")
		}
		return
	}
	fields(lit, "Parts", m, bad)
	if cfg != nil {
		fields(cfg, "Strict StrictExempt StrictInterfaces StrictShared Mutators", m, bad)
	}
}

// fields reads the fields of the System or Settings literal x; allowed lists the field names of
// its type.
func fields(x *ast.CompositeLit, allowed string, m *Manifest, bad func(ast.Node, string, ...any)) {
	for _, e := range x.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		var key *ast.Ident
		if ok {
			key, _ = kv.Key.(*ast.Ident)
		}
		if key == nil {
			bad(e, "each field of the manifest is written as Name: value")
			continue
		}
		if !slices.Contains(strings.Fields(allowed), key.Name) {
			if key.Name == "Parts" {
				bad(key, "the parts are in var Manifest = System{...}, not in the check configuration")
			} else if slices.Contains([]string{"Strict", "StrictExempt", "StrictInterfaces", "StrictShared", "Mutators"}, key.Name) {
				bad(key, "%s is check configuration: it is in var Config = Settings{...} of %s/%s", key.Name, ManifestDir, ConfigFile)
			} else {
				bad(key, "the manifest has no field %s", key.Name)
			}
			continue
		}
		switch key.Name {
		case "Strict":
			id, ok := ast.Unparen(kv.Value).(*ast.Ident)
			if !ok || (id.Name != "true" && id.Name != "false") {
				bad(kv.Value, "Strict is true or false")
				continue
			}
			m.Strict = id.Name == "true"
		case "StrictExempt", "StrictInterfaces", "StrictShared", "Mutators":
			set := map[string]map[string]bool{"StrictExempt": m.Exempt, "StrictInterfaces": m.Interfaces, "StrictShared": m.Shared, "Mutators": m.Mutators}[key.Name]
			m.ListPos[key.Name] = kv.Pos()
			cl, ok := ast.Unparen(kv.Value).(*ast.CompositeLit)
			if !ok {
				bad(kv.Value, "%s is a []string{...} literal", key.Name)
				continue
			}
			for _, x := range cl.Elts {
				s, ok := stringLit(x)
				if !ok {
					bad(x, "%s holds string literals", key.Name)
					continue
				}
				set[s] = true
			}
		case "Parts":
			cl, ok := ast.Unparen(kv.Value).(*ast.CompositeLit)
			if !ok {
				bad(kv.Value, "Parts is a []Part{...} literal")
				continue
			}
			for _, x := range cl.Elts {
				if p := part(x, bad); p != nil {
					m.Parts = append(m.Parts, p)
					if _, dup := m.ByDir[p.Dir]; !dup {
						m.ByDir[p.Dir] = p
					}
				}
			}
		}
	}
}

func part(x ast.Expr, bad func(ast.Node, string, ...any)) *Part {
	call, ok := ast.Unparen(x).(*ast.CallExpr)
	var fn *ast.Ident
	if ok {
		fn, _ = ast.Unparen(call.Fun).(*ast.Ident)
	}
	if fn == nil {
		bad(x, "a part is Leaf(dir), Gateway(dir, External), Entry(dir, Person or External), Library(dir), or Main(dir)")
		return nil
	}
	kinds := map[string]string{"Leaf": Leaf, "Gateway": Gateway, "Entry": Entry, "Library": Library, "Main": Main}
	kind, ok := kinds[fn.Name]
	if !ok {
		bad(fn, "%s is not a kind of part; the kinds are Leaf, Gateway, Entry, Library, and Main", fn.Name)
		return nil
	}
	want := 1
	if kind == Gateway || kind == Entry {
		want = 2
	}
	if len(call.Args) != want {
		bad(call, "%s takes %d arguments", fn.Name, want)
		return nil
	}
	dir, ok := stringLit(call.Args[0])
	if !ok || dir == "" || path.Clean(dir) != dir || path.IsAbs(dir) || strings.HasPrefix(dir, "..") || dir == "." {
		bad(call.Args[0], "the directory of a part is a string literal, relative to the module root, such as \"api/orders\"")
		return nil
	}
	p := &Part{Dir: dir, Kind: kind, Pos: call.Pos()}
	if want == 2 {
		id, ok := ast.Unparen(call.Args[1]).(*ast.Ident)
		if !ok {
			bad(call.Args[1], "the second argument of %s names a constant of type Person or External", fn.Name)
			return nil
		}
		p.For = id.Name
	}
	return p
}

func stringLit(x ast.Expr) (string, bool) {
	bl, ok := ast.Unparen(x).(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(bl.Value)
	return s, err == nil
}

// Dirs returns the directories of the parts, sorted.
func (m *Manifest) Dirs() []string {
	var out []string
	for d := range m.ByDir {
		out = append(out, d)
	}
	sort.Strings(out)
	return out
}
