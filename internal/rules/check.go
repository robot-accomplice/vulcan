package rules

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// The kinds of a part. "leaf" is a plain leaf: not a gateway and not an entry.
var partKinds = []string{"leaf", "gateway", "entry", "library", "main"}

// isSelector reports whether a word names a kind of part or the directory of one.
func isSelector(v string) bool {
	return slices.Contains(partKinds, v) || strings.Contains(v, "/")
}

// Part is one part of a project.
type Part struct {
	Dir  string
	Kind string
}

func selected(words []string, p Part) bool {
	return slices.Contains(words, p.Kind) || slices.Contains(words, p.Dir)
}

// Edge is one dependency between parts.
type Edge struct {
	From, To string // directories
	Via      string // "wire" or "import"
	At       any    // where to report it; the caller gives it, and gets it back
}

// Finding is one thing that a rule rejects.
type Finding struct {
	Code string // the digits: 0802 is K0802 in V and VL0802 in L
	At   any    // a token.Pos, or the At of an Edge
	Msg  string
}

// Set is what a check reads: the rules that are on, and the lock.
type Set struct {
	File  *File
	Lock  map[string]bool // "wire a -> b" and "import a -> b"
	Probs []Problem
	on    []*Rule
}

// Load reads the rules of a project. A project with no rules file has no rules: nothing else
// holds a rule. It reads the rules file and, when a rule needs it, the lock, and nothing else.
func Load(root string) *Set {
	b, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil
	}
	s := &Set{}
	s.File, s.Probs = ParseProject(b)
	for _, r := range s.File.Rules {
		if r.Level == Error {
			s.on = append(s.on, r)
		}
	}
	for _, r := range s.on {
		if r.Kind == "depends" && len(r.Params["map"]) > 0 {
			lock, err := ReadLock(root)
			if err != nil {
				s.Probs = append(s.Probs, Problem{r.Line, fmt.Sprintf("rule %s reads the locked map, but %s; run the command rules lock", r.ID, err)})
				continue
			}
			s.Lock = lock
		}
	}
	return s
}

// Resolve reports each name of the rules that the project does not have. A typing error in a
// directory must not switch a rule off without a word.
func (s *Set) Resolve(parts map[string]Part) {
	for _, r := range s.File.Rules {
		for _, key := range []string{"in", "except", "from", "deny", "load"} {
			for _, v := range r.Params[key] {
				if strings.Contains(v, "/") && parts[v].Dir == "" {
					s.Probs = append(s.Probs, Problem{r.Line, fmt.Sprintf("rule %s: %s names %s, which is not a part of the project", r.ID, key, v)})
				}
			}
		}
		if r.Kind == "depends" {
			for _, v := range r.Params["allow"] {
				if strings.Contains(v, "/") && parts[v].Dir == "" {
					s.Probs = append(s.Probs, Problem{r.Line, fmt.Sprintf("rule %s: allow names %s, which is not a part of the project", r.ID, v)})
				}
			}
		}
		if r.Kind == "transitions" {
			if t := r.Params["type"]; len(t) == 1 {
				dir, _, _ := strings.Cut(t[0], ".")
				if parts[dir].Dir == "" {
					s.Probs = append(s.Probs, Problem{r.Line, fmt.Sprintf("rule %s: type names the directory %s, which is not a part of the project", r.ID, dir)})
				}
			}
		}
	}
}

// Unit is one package that a check reads.
type Unit struct {
	Part   Part
	Module string
	Parts  map[string]Part // by directory
	Fset   *token.FileSet
	Files  []*ast.File
	// PortFiles are V's generated port declarations: only their imports count, which are the
	// imports of component.vulcan, as L's ports.go has them.
	PortFiles []*ast.File
	Info      *types.Info
	IsTest    func(*ast.File) bool
}

func (u *Unit) fileIn(r *Rule, f *ast.File) bool {
	files := r.Params["files"]
	if len(files) == 0 || files[0] == "all" {
		return true
	}
	return (files[0] == "test") == u.IsTest(f)
}

func (r *Rule) tail() string {
	return fmt.Sprintf("%s (rule %s in %s; the team can change or delete it)", r.Reason, r.ID, FileName)
}

// CheckUnit applies the rules that read the code of one package. A rule does not read a main
// package: in V, main is generated, so a rule that fired on L's main would have no equal in V.
func (s *Set) CheckUnit(u *Unit) []Finding {
	if s == nil || u.Part.Kind == "main" {
		return nil
	}
	var out []Finding
	seen := map[string]bool{}
	add := func(code string, at token.Pos, format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		key := fmt.Sprint(code, at, msg)
		if !seen[key] {
			seen[key] = true
			out = append(out, Finding{code, at, msg})
		}
	}
	for _, r := range s.on {
		switch r.Kind {
		case "forbid-call":
			if (len(r.Params["in"]) == 0 || selected(r.Params["in"], u.Part)) && !selected(r.Params["except"], u.Part) {
				u.forbidCall(r, add)
			}
		case "only-in":
			if !selected(r.Params["in"], u.Part) {
				u.onlyIn(r, add)
			}
		case "depends":
			u.imports(r, s.Lock, add)
		case "transitions":
			u.transitions(r, add)
		}
	}
	return out
}

type addFunc func(code string, at token.Pos, format string, args ...any)

// usedName gives the full name of a use of a function or a package-level variable, or "".
func usedName(obj types.Object) string {
	switch o := obj.(type) {
	case *types.Func:
		return o.FullName()
	case *types.Var:
		if o.Pkg() != nil && o.Parent() == o.Pkg().Scope() {
			return o.Pkg().Path() + "." + o.Name()
		}
	}
	return ""
}

func (u *Unit) forbidCall(r *Rule, add addFunc) {
	funcs := map[string]bool{}
	for _, f := range r.Params["funcs"] {
		funcs[f] = true
	}
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if name := usedName(u.Info.Uses[id]); name != "" && matches(funcs, name) {
					add("0802", id.Pos(), "%s uses %s, which a rule forbids here: %s", u.Part.Dir, name, r.tail())
				}
			}
			return true
		})
	}
}

func (u *Unit) onlyIn(r *Rule, add addFunc) {
	where := strings.Join(r.Params["in"], ", ")
	for _, eff := range r.Params["effect"] {
		e := effects[eff]
		if e == nil {
			continue
		}
		for _, f := range u.Files {
			if !u.fileIn(r, f) {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.Ident:
					switch obj := u.Info.Uses[x].(type) {
					case *types.Func:
						if matches(e.funcs, obj.FullName()) {
							add("0803", x.Pos(), "%s uses %s, which has the effect %s, allowed only in %s: %s", u.Part.Dir, obj.FullName(), eff, where, r.tail())
						}
					case *types.Var:
						if name := usedName(obj); name != "" && e.values[name] {
							add("0803", x.Pos(), "%s uses %s, which has the effect %s, allowed only in %s: %s", u.Part.Dir, name, eff, where, r.tail())
						}
					}
				case *ast.CompositeLit:
					if name := fullTypeName(u.Info.TypeOf(x)); name != "" && e.types[name] {
						add("0803", x.Pos(), "%s builds a %s, which has the effect %s, allowed only in %s: %s", u.Part.Dir, name, eff, where, r.tail())
					}
				}
				return true
			})
		}
	}
}

func namedOf(t types.Type) *types.Named {
	if t == nil {
		return nil
	}
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		t = p.Elem()
	}
	n, _ := types.Unalias(t).(*types.Named)
	return n
}

func fullTypeName(t types.Type) string {
	n := namedOf(t)
	if n == nil || n.Obj().Pkg() == nil {
		return ""
	}
	return n.Obj().Pkg().Path() + "." + n.Obj().Name()
}

// partOfImport returns the part that an import path names.
func (u *Unit) partOfImport(path string) (Part, bool) {
	if !strings.HasPrefix(path, u.Module+"/") {
		return Part{}, false
	}
	p, ok := u.Parts[strings.TrimPrefix(path, u.Module+"/")]
	return p, ok
}

// imports applies a depends rule to the imports of one package.
func (u *Unit) imports(r *Rule, lock map[string]bool, add addFunc) {
	if len(r.Params["map"]) > 0 {
		for _, f := range slices.Concat(u.Files, u.PortFiles) {
			if u.IsTest(f) {
				continue
			}
			for _, imp := range f.Imports {
				path, _ := strconv.Unquote(imp.Path.Value)
				if t, ok := u.partOfImport(path); ok && t.Dir != u.Part.Dir && t.Kind == "library" && !IsRuntimeLib(t.Dir) && lock != nil {
					e := Edge{From: u.Part.Dir, To: t.Dir, Via: "import"}
					if !lock[e.key()] {
						add("0806", imp.Path.Pos(), "%s: %s", newDep(e), r.tail())
					}
				}
			}
		}
		return
	}
	if !selected(r.Params["from"], u.Part) || selected(r.Params["except"], u.Part) {
		return
	}
	wire := len(r.Params["via"]) > 0 && r.Params["via"][0] == "wire"
	for _, f := range slices.Concat(u.Files, u.PortFiles) {
		if !u.fileIn(r, f) {
			continue
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if slices.Contains(r.Params["packages"], path) {
				add("0804", imp.Path.Pos(), "%s imports %s, which a rule forbids here: %s", u.Part.Dir, path, r.tail())
				continue
			}
			if wire {
				continue
			}
			if t, ok := u.partOfImport(path); ok && t.Dir != u.Part.Dir && !IsRuntimeLib(t.Dir) && edgeDenied(r, t) {
				add("0804", imp.Path.Pos(), "%s imports %s (%s), which a rule forbids here: %s", u.Part.Dir, t.Dir, t.Kind, r.tail())
			}
		}
	}
}

func edgeDenied(r *Rule, to Part) bool {
	if allow := r.Params["allow"]; len(allow) > 0 {
		return !selected(allow, to)
	}
	return selected(r.Params["deny"], to)
}

// CheckWires applies the depends rules to the wires of a project.
func (s *Set) CheckWires(parts map[string]Part, wires []Edge) []Finding {
	if s == nil {
		return nil
	}
	var out []Finding
	for _, r := range s.on {
		if r.Kind != "depends" {
			continue
		}
		for _, e := range wires {
			if len(r.Params["map"]) > 0 {
				if s.Lock != nil && !s.Lock[e.key()] {
					out = append(out, Finding{"0806", e.At, newDep(e) + ": " + r.tail()})
				}
				continue
			}
			from, to := parts[e.From], parts[e.To]
			if via := r.Params["via"]; len(via) > 0 && via[0] == "import" {
				continue
			}
			if selected(r.Params["from"], from) && !selected(r.Params["except"], from) && (len(r.Params["deny"])+len(r.Params["allow"]) > 0) && edgeDenied(r, to) {
				out = append(out, Finding{"0804", e.At, fmt.Sprintf("a wire makes %s (%s) depend on %s (%s), which a rule forbids: %s", e.From, from.Kind, e.To, to.Kind, r.tail())})
			}
		}
	}
	return out
}

func newDep(e Edge) string {
	return fmt.Sprintf("the %s %s -> %s is a dependency that %s does not hold; if it is wanted, run the command rules lock and commit the change", e.Via, e.From, e.To, LockName)
}

func (e Edge) key() string { return e.Via + " " + e.From + " -> " + e.To }

// IsForeign reports whether a file name is an assembly file or an object file.
func IsForeign(name string) bool {
	switch filepath.Ext(name) {
	case ".s", ".S", ".sx", ".syso":
		return true
	}
	return false
}

// AsmFiles lists the files of a directory that hold code for the Go toolchain, but not Go code:
// an assembly file or an object file gives a body-less Go function its body (hole H6).
func AsmFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if IsForeign(e.Name()) {
			out = append(out, e.Name())
		}
	}
	return out
}

// Bodyless lists the functions of a file that have no body (hole H6). A function that a
// //go:linkname names is not listed: K0122 and VL0122 reject that directive.
func Bodyless(f *ast.File) []*ast.Ident {
	linked := map[string]bool{}
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if fs := strings.Fields(c.Text); len(fs) >= 2 && fs[0] == "//go:linkname" {
				linked[fs[1]] = true
			}
		}
	}
	var out []*ast.Ident
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Body == nil && !linked[fd.Name.Name] {
			out = append(out, fd.Name)
		}
	}
	return out
}

// HoleH6Message is the message for a body-less function, the same in both checkers.
func HoleH6Message(name string) string {
	return fmt.Sprintf("function %s has no body, so its code is outside Go and no rule can read it; write the body in Go, or move the call behind a port", name)
}

// HoleH6FileMessage is the message for an assembly file or an object file.
func HoleH6FileMessage(name string) string {
	return fmt.Sprintf("%s holds code that is not Go, so no rule can read it; a leaf or a library holds only Go code", name)
}
