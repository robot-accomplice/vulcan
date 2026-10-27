// Package check type-checks a project's generated build directory and applies the rules of the
// release (spec §5.4, §5.4a, §5.4b, §6).
package check

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/gen"
	"github.com/robot-accomplice/vulcan/internal/lower"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// Checker holds the loaded packages of one check.
type Checker struct {
	P     *project.Project
	Fset  *token.FileSet
	Plain map[string]*packages.Package // by import path, without test files
	Full  map[string]*packages.Package // by import path, with test files when loaded
	D     *diag.List

	badPorts map[string]bool
	scope    *types.Package // the scope of system.vulcan values // ports with a K0112 type, whose method set is not checked
}

// Env is the environment for the go command in the build directory. The module graph is read
// only: a dependency that go.mod lacks is an error (K0209), which `vulcan mod tidy` fixes.
func Env() []string {
	return append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly")
}

// Load type-checks the build directory, which gen.WriteLeaves wrote. With report false, it
// reports nothing: the first pass only gives the lowering its types.
func Load(p *project.Project, tests, report bool) (*Checker, error) {
	c := &Checker{P: p, Fset: token.NewFileSet(), Plain: map[string]*packages.Package{}, Full: map[string]*packages.Package{}, D: p.Diags}
	if !report {
		c.D = &diag.List{}
	}
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedImports | packages.NeedModule,
		Dir:   gen.BuildDir(p),
		Env:   Env(),
		Fset:  c.Fset,
		Tests: tests,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		if strings.HasSuffix(pkg.ID, ".test") {
			continue // the generated test main
		}
		if strings.Contains(pkg.ID, " [") {
			c.Full[pkg.PkgPath] = pkg
		} else {
			c.Plain[pkg.PkgPath] = pkg
			if _, ok := c.Full[pkg.PkgPath]; !ok {
				c.Full[pkg.PkgPath] = pkg
			}
		}
	}
	// A module that go.mod lacks gives a list error on the missing package, and a type error
	// "could not import" in each importer; report the first as K0209 and drop the second.
	missing := map[string]bool{}
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, e := range pkg.Errors {
			if e.Kind == packages.ListError && isMissingModule(e.Msg) {
				missing[pkg.PkgPath] = true
				c.goError(e)
			}
		}
	})
	for _, pkg := range pkgs {
		parseErr := map[string]bool{} // files with a parse error: report only the first
		for _, e := range pkg.Errors {
			if e.Kind == packages.ParseError {
				file := c.relPos(e.Pos).File
				if !parseErr[file] {
					parseErr[file] = true
					c.goError(e)
				}
			}
		}
		for _, e := range pkg.Errors {
			switch {
			case e.Kind == packages.ParseError:
			case len(parseErr) > 0 && e.Kind == packages.TypeError:
				// Type errors after a syntax error repeat it.
			case e.Pos == "" && strings.HasPrefix(e.Msg, "# "):
				// "# pkg" errors of the go command repeat the type errors of the package.
			case e.Kind == packages.ListError && isMissingModule(e.Msg):
			case importsMissing(e.Msg, missing):
			default:
				c.goError(e)
			}
		}
	}
	return c, nil
}

func isMissingModule(msg string) bool {
	return strings.Contains(msg, "no required module provides package") || strings.Contains(msg, "cannot find module providing package") || strings.Contains(msg, "missing go.sum entry")
}

func importsMissing(msg string, missing map[string]bool) bool {
	for path := range missing {
		if strings.Contains(msg, "could not import "+path) {
			return true
		}
	}
	return false
}

var posRE = regexp.MustCompile(`^(.*):(\d+):(\d+)$`)
var posLineRE = regexp.MustCompile(`^(.*):(\d+)$`)

// goError reports an error of the go command or of go/types as K0008 (or K0001 for syntax).
func (c *Checker) goError(e packages.Error) {
	pos := c.relPos(e.Pos)
	code := "K0008"
	if e.Kind == packages.ParseError {
		code = "K0001"
	}
	msg := e.Msg
	if isMissingModule(msg) {
		code = "K0209"
		for _, cut := range []string{"; to add it", ": import lookup disabled"} {
			if i := strings.Index(msg, cut); i > 0 {
				msg = msg[:i]
			}
		}
		msg = strings.Replace(msg, "cannot find module providing package", "no module in go.mod provides package", 1)
		msg += "; run vulcan mod tidy to add it to go.mod and go.sum"
	}
	if pos.File == "" && strings.Contains(msg, ":") {
		// Errors of the go command carry their own position.
		if i := strings.Index(msg, ": "); i > 0 {
			if pp := c.relPos(msg[:i]); pp.File != "" {
				pos, msg = pp, msg[i+2:]
			}
		}
	}
	c.D.Errorf(pos, code, "%s", c.cleanMsg(msg))
}

// cleanMsg removes generated names from Go messages.
func (c *Checker) cleanMsg(msg string) string {
	msg = strings.ReplaceAll(msg, filepath.ToSlash(c.P.Root)+"/", "")
	msg = regexp.MustCompile(`\bvulcan_([A-Z]\w*)\b`).ReplaceAllString(msg, "port $1")
	return msg
}

// relPos converts "file:line:col" from the go command to a project position.
func (c *Checker) relPos(s string) diag.Pos {
	var file string
	var line, col int
	if m := posRE.FindStringSubmatch(s); m != nil {
		file = m[1]
		line, _ = strconv.Atoi(m[2])
		col, _ = strconv.Atoi(m[3])
	} else if m := posLineRE.FindStringSubmatch(s); m != nil {
		file = m[1]
		line, _ = strconv.Atoi(m[2])
	} else {
		return diag.Pos{}
	}
	if !filepath.IsAbs(file) && !strings.HasPrefix(filepath.ToSlash(file), "/") {
		file = filepath.Join(gen.BuildDir(c.P), file) // relative to the build directory
	}
	rel := c.rel(file)
	// A position in the build directory's copy of a code file: map it back to the .vulcan file,
	// whose first line is the //line directive's line.
	build := filepath.ToSlash(filepath.Join(".vulcan", "build")) + "/"
	if strings.HasPrefix(rel, build) && strings.HasSuffix(rel, ".go") && !strings.HasPrefix(filepath.Base(rel), "vulcan_") {
		if src := strings.TrimSuffix(strings.TrimPrefix(rel, build), ".go") + ".vulcan"; c.isSource(src) {
			rel = src
			line -= gen.HeaderLines
		}
	}
	return diag.Pos{File: rel, Line: line, Col: col}
}

func (c *Checker) isSource(rel string) bool {
	for _, pkg := range append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...) {
		for _, f := range pkg.Files {
			if f.Rel == rel {
				return true
			}
		}
	}
	return false
}

func (c *Checker) rel(file string) string {
	file = filepath.ToSlash(file)
	root := filepath.ToSlash(c.P.Root) + "/"
	if strings.HasPrefix(strings.ToLower(file), strings.ToLower(root)) {
		return file[len(root):]
	}
	return file
}

// Pos gives the project position of a node.
func (c *Checker) Pos(n token.Pos) diag.Pos {
	p := c.Fset.Position(n)
	return diag.Pos{File: c.rel(p.Filename), Line: p.Line, Col: p.Column}
}

// Generated reports whether a syntax file was generated by the compiler (not a .vulcan file).
func (c *Checker) Generated(f *ast.File) bool {
	name := filepath.Base(c.Fset.PositionFor(f.Package, false).Filename)
	return strings.HasPrefix(name, "vulcan_")
}

// Package returns the project package for an import path, or nil.
func (c *Checker) Package(path string) *project.Package { return c.P.ByPath[path] }

// Typed gives the lowering the syntax and types of a package's code files.
func (c *Checker) Typed(pkg *project.Package) *lower.Typed {
	gp := c.Full[pkg.ImportPath]
	if gp == nil || gp.TypesInfo == nil {
		return nil
	}
	byRel := map[string]*project.CodeFile{}
	for _, f := range pkg.Files {
		byRel[gen.GoFileName(f.Rel)] = f
	}
	build := filepath.ToSlash(gen.BuildDir(c.P)) + "/"
	t := &lower.Typed{Fset: c.Fset, Info: gp.TypesInfo, Files: map[*project.CodeFile]*ast.File{}, Base: map[*project.CodeFile]int{}}
	for _, f := range gp.Syntax {
		name := filepath.ToSlash(c.Fset.PositionFor(f.Package, false).Filename)
		if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(build)) {
			continue
		}
		if cf := byRel[name[len(build):]]; cf != nil {
			t.Files[cf] = f
			t.Base[cf] = len(gen.Header(c.P, cf.Rel))
		}
	}
	return t
}
