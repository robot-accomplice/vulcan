// Package detect is the library of the structural rule detectors (docs/poc/plan.md, section 6).
// A detector reads the final source of a run, with types: the Go code of arm G, and the leaf and
// library bodies of arm V (the Go code that `vulcan check` writes in .vulcan/build, whose
// positions map back to the .vulcan files). It names each part of the application by its unit:
// api/orders, lib/shop, cmd/stockroom.
//
// A detector is a small program:
//
//	func main() {
//		detect.Main(func(s *detect.Source) []detect.Violation {
//			var out []detect.Violation
//			for _, u := range s.UsesOf("database/sql.Open") {
//				if u.Unit != "api/store" {
//					out = append(out, detect.Violation{Pos: u.Pos, Text: "opens a database outside store"})
//				}
//			}
//			return out
//		})
//	}
//
// It prints each violation and exits 1 when there is one, 0 when there is none, and 2 when it
// cannot read the source.
package detect

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Module is the module path of Stockroom in both arms.
const Module = "stockroom"

// Source is the type-checked code of one workspace.
type Source struct {
	Arm  string
	WS   string
	Fset *token.FileSet
	Pkgs []*packages.Package // the application's packages, without generated main and tests
}

// Violation is one finding.
type Violation struct {
	Pos  string
	Text string
}

// Load reads the workspace of a run. For arm V it runs `vulcan check` first when the build
// directory is missing, with vulcan on the PATH.
func Load(arm, ws string) (*Source, error) {
	dir := ws
	if arm == "V" {
		dir = filepath.Join(ws, ".vulcan", "build")
		if _, err := os.Stat(dir); err != nil {
			cmd := exec.Command("vulcan", "check")
			cmd.Dir = ws
			if out, err := cmd.CombinedOutput(); err != nil {
				if _, statErr := os.Stat(dir); statErr != nil {
					return nil, fmt.Errorf("vulcan check: %v\n%s", err, out)
				}
			}
		}
	}
	s := &Source{Arm: arm, WS: ws, Fset: token.NewFileSet()}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedImports,
		Dir:  dir,
		Fset: s.Fset,
		Env:  append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off"),
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	for _, p := range pkgs {
		if strings.HasSuffix(p.PkgPath, "vulcan_main") {
			continue
		}
		s.Pkgs = append(s.Pkgs, p)
	}
	sort.Slice(s.Pkgs, func(i, j int) bool { return s.Pkgs[i].PkgPath < s.Pkgs[j].PkgPath })
	return s, nil
}

// UnitOf names the part of a package path: stockroom/api/orders is api/orders.
func UnitOf(pkgPath string) string { return strings.TrimPrefix(pkgPath, Module+"/") }

// Generated reports whether a syntax file was written by the Vulcan compiler, not by anyone
// working on the application.
func (s *Source) Generated(f *ast.File) bool {
	name := filepath.Base(s.Fset.PositionFor(f.Package, false).Filename)
	return strings.HasPrefix(name, "vulcan_")
}

// Pos gives a position in the source that the agent edited (.vulcan files in arm V).
func (s *Source) Pos(p token.Pos) string {
	pos := s.Fset.Position(p)
	file := filepath.ToSlash(pos.Filename)
	ws := filepath.ToSlash(s.WS) + "/"
	if i := strings.Index(strings.ToLower(file), strings.ToLower(ws)); i >= 0 {
		file = file[i+len(ws):]
	}
	return fmt.Sprintf("%s:%d:%d", file, pos.Line, pos.Column)
}

// Imports maps each unit to the import paths of its code (not of generated files).
func (s *Source) Imports() map[string][]string {
	out := map[string][]string{}
	for _, p := range s.Pkgs {
		seen := map[string]bool{}
		for _, f := range p.Syntax {
			if s.Generated(f) {
				continue
			}
			for _, imp := range f.Imports {
				path := strings.Trim(imp.Path.Value, `"`)
				if !seen[path] {
					seen[path] = true
					out[UnitOf(p.PkgPath)] = append(out[UnitOf(p.PkgPath)], path)
				}
			}
		}
	}
	return out
}

// Use is one use of a function, a method, or a variable.
type Use struct {
	Unit string
	Pos  string
	Func string // the enclosing function, or ""
}

// UsesOf lists the uses of objects whose full name is name: "database/sql.Open",
// "(*net/http.Client).Do", or "stockroom/lib/shop.MaxPrice". A name that ends in "*" is a
// prefix.
func (s *Source) UsesOf(name string) []Use {
	return s.Find(func(obj types.Object) bool { return matches(fullName(obj), name) })
}

// Find lists the uses of objects for which match is true, in code that is not generated.
func (s *Source) Find(match func(types.Object) bool) []Use {
	var out []Use
	for _, p := range s.Pkgs {
		for _, f := range p.Syntax {
			if s.Generated(f) {
				continue
			}
			for _, d := range f.Decls {
				fn := ""
				if fd, ok := d.(*ast.FuncDecl); ok {
					fn = fd.Name.Name
				}
				ast.Inspect(d, func(n ast.Node) bool {
					id, ok := n.(*ast.Ident)
					if !ok {
						return true
					}
					if obj := p.TypesInfo.Uses[id]; obj != nil && match(obj) {
						out = append(out, Use{Unit: UnitOf(p.PkgPath), Pos: s.Pos(id.Pos()), Func: fn})
					}
					return true
				})
			}
		}
	}
	return out
}

func fullName(obj types.Object) string {
	switch o := obj.(type) {
	case *types.Func:
		return o.FullName()
	case *types.Var, *types.Const, *types.TypeName:
		if o.Pkg() != nil {
			return o.Pkg().Path() + "." + o.Name()
		}
	}
	return obj.Name()
}

func matches(full, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(full, strings.TrimSuffix(pattern, "*"))
	}
	return full == pattern
}

// Main runs a detector as a command: -arm V|G -ws <workspace>.
func Main(check func(*Source) []Violation) {
	arm := flag.String("arm", "", "the arm: V or G")
	ws := flag.String("ws", "", "the workspace of the run")
	flag.Parse()
	s, err := Load(*arm, *ws)
	if err != nil {
		fmt.Fprintln(os.Stderr, "detect:", err)
		os.Exit(2)
	}
	vs := check(s)
	for _, v := range vs {
		fmt.Printf("%s: %s\n", v.Pos, v.Text)
	}
	if len(vs) > 0 {
		os.Exit(1)
	}
}
