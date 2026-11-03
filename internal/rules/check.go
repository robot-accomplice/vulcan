package rules

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
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
	From, To string       // directories
	Via      string       // "wire" or "import"
	At       any          // where to report it; the caller gives it, and gets it back
	Methods  []PortMethod // of the port, for a wire
}

// Finding is one thing that a rule rejects.
type Finding struct {
	Code string // the digits: 0802 is K0802 in V and VL0802 in L
	At   any    // a token.Pos, or the At of an Edge
	Msg  string
	Warn bool // the rule has level warn, or the baseline holds it: it reports, and the check passes

	Rule, Group, Part, Symbol string // the key of the finding (baseline.go)
	Text                      string // Msg with no reason of the rule
	Old                       bool   // the baseline holds it
}

// finding makes a finding of a rule.
func (r *Rule) finding(code string, at any, msg, group, part, symbol string) Finding {
	return Finding{Code: code, At: at, Msg: msg, Warn: r.Level == Warn, Rule: r.ID, Group: group, Part: part, Symbol: symbol, Text: strings.TrimSuffix(msg, ": "+r.tail())}
}

// Set is what a check reads: the rules that are on, and the lock.
type Set struct {
	File  *File
	Lock  map[string]bool // "wire a -> b" and "import a -> b"
	Probs []Problem
	// Baseline is the file of accepted findings, or nil.
	Baseline *Baseline
	on       []*Rule

	reported sync.Map // the messages of ResolveTypes that an arm has: a package reads each name once
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
	base, probs := ReadBaseline(root)
	s.Baseline = base
	s.Probs = append(s.Probs, probs...)
	for _, r := range s.File.Rules {
		if r.Level == Error || r.Level == Warn {
			s.on = append(s.on, r)
		}
	}
	for _, r := range s.on {
		if r.Kind == "depends" && len(r.Params["map"]) > 0 {
			lock, err := ReadLock(root)
			if err != nil {
				s.Probs = append(s.Probs, Problem{Line: r.Line, Msg: fmt.Sprintf("rule %s reads the locked map, but %s; run the command rules lock", r.ID, err)})
				continue
			}
			s.Lock = lock
		}
	}
	return s
}

// Resolve reports each name of the rules that the project does not have. A typing error in a
// directory must not switch a rule off without a word.
func (s *Set) Resolve(parts map[string]Part, module string) {
	for _, r := range s.File.Rules {
		for _, key := range nameKeys {
			if key == "type" && r.Kind != "literal" {
				continue
			}
			for _, name := range r.Params[key] {
				for _, one := range projectNames(name, module) {
					dir, _, _, ok := projectName(one, module)
					switch {
					case !ok || IsRuntimeLib(dir):
					case parts[dir].Dir == "":
						s.Probs = append(s.Probs, Problem{Line: r.Line, Msg: fmt.Sprintf("rule %s: %s names %s, and %s is not a part of the project", r.ID, key, name, dir)})
					case parts[dir].Kind == "main":
						s.Probs = append(s.Probs, Problem{Line: r.Line, Msg: fmt.Sprintf("rule %s: %s names %s, in main, which no rule reads", r.ID, key, name)})
					}
				}
			}
		}
		for _, key := range []string{"in", "except", "from", "deny", "load", "to"} {
			if r.Kind == "literal" && key == "deny" {
				continue
			}
			for _, v := range r.Params[key] {
				if strings.Contains(v, "/") && parts[v].Dir == "" {
					s.Probs = append(s.Probs, Problem{Line: r.Line, Msg: fmt.Sprintf("rule %s: %s names %s, which is not a part of the project", r.ID, key, v)})
				}
			}
		}
		if r.Kind == "depends" {
			for _, v := range r.Params["allow"] {
				if strings.Contains(v, "/") && parts[v].Dir == "" {
					s.Probs = append(s.Probs, Problem{Line: r.Line, Msg: fmt.Sprintf("rule %s: allow names %s, which is not a part of the project", r.ID, v)})
				}
			}
		}
		if r.Kind == "transitions" {
			if t := r.Params["type"]; len(t) == 1 {
				dir, _, _ := strings.Cut(t[0], ".")
				if parts[dir].Dir == "" {
					s.Probs = append(s.Probs, Problem{Line: r.Line, Msg: fmt.Sprintf("rule %s: type names the directory %s, which is not a part of the project", r.ID, dir)})
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
	// Pkg is the package. Generates says that this arm's lowering adds names with the prefix
	// vulcan_, which no rule reads. SourceFile maps a file of a position to one that Go can parse.
	Pkg       *types.Package
	Generates bool
	// ContractLib says that lib/contract is this arm's contract package, which no rule reads: L's. In
	// V, a library of that name is an ordinary library.
	ContractLib bool
	SourceFile  func(string) string
	Info        *types.Info
	// Ports are the ports of the leaf. HasEnsures says whether a function has an ensures clause.
	// TestsAbsent says that the package has tests, and this load left them out.
	Ports       []Port
	HasEnsures  func(*ast.FuncDecl) bool
	TestsAbsent bool
	IsTest      func(*ast.File) bool
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
	if s == nil || u.Part.Kind == "main" || u.isRuntimeLib(u.Part.Dir) {
		return nil
	}
	var out []Finding
	seen := map[string]bool{}
	var cur *Rule
	add := func(code string, at token.Pos, format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		// By the place in the source: V's twin of a method holds the same places as the method.
		key := fmt.Sprint(code, u.Fset.Position(at), msg)
		if !seen[key] {
			seen[key] = true
			out = append(out, cur.finding(code, at, msg, GroupCode, u.Part.Dir, u.symbolAt(at)))
		}
	}
	for _, r := range s.on {
		cur = r
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
		case "declaration":
			u.declaration(r, add)
		case "call-form":
			u.callForm(r, add)
		case "literal":
			u.literal(r, add)
		case "code-form":
			u.codeForm(r, add)
		}
	}
	return out
}

type addFunc func(code string, at token.Pos, format string, args ...any)

// usedName gives the full name of a use of a function or a package-level variable, or "".
func usedName(obj types.Object) string {
	switch o := obj.(type) {
	case *types.Func:
		return portName(o.FullName())
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
				if t, ok := u.partOfImport(path); ok && t.Dir != u.Part.Dir && t.Kind == "library" && !u.isRuntimeLib(t.Dir) && lock != nil {
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
			if t, ok := u.partOfImport(path); ok && t.Dir != u.Part.Dir && !u.isRuntimeLib(t.Dir) && edgeDenied(r, t) {
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

func (u *Unit) isRuntimeLib(dir string) bool { return u.ContractLib && IsRuntimeLib(dir) }

// symbolAt names the function that holds a place, with no receiver. A place outside a function has no
// symbol: its text names it.
func (u *Unit) symbolAt(at token.Pos) string {
	for _, f := range u.Files {
		if at < f.Pos() || at > f.End() {
			continue
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && at >= fd.Pos() && at <= fd.End() {
				return fd.Name.Name
			}
		}
	}
	return ""
}

// CheckWires applies the depends rules to the wires of a project.
func (s *Set) CheckWires(parts map[string]Part, wires []Edge) []Finding {
	if s == nil {
		return nil
	}
	var out []Finding
	for _, r := range s.on {
		if r.Kind == "declaration" && slices.Contains(r.Params["at"], "wired-port") {
			out = append(out, wiredPorts(r, parts, wires)...)
		}
		if r.Kind != "depends" {
			continue
		}
		for _, e := range wires {
			if len(r.Params["map"]) > 0 {
				if s.Lock != nil && !s.Lock[e.key()] {
					out = append(out, r.finding("0806", e.At, newDep(e)+": "+r.tail(), GroupWires, e.From, ""))
				}
				continue
			}
			from, to := parts[e.From], parts[e.To]
			if via := r.Params["via"]; len(via) > 0 && via[0] == "import" {
				continue
			}
			if selected(r.Params["from"], from) && !selected(r.Params["except"], from) && (len(r.Params["deny"])+len(r.Params["allow"]) > 0) && edgeDenied(r, to) {
				out = append(out, r.finding("0804", e.At, fmt.Sprintf("a wire makes %s (%s) depend on %s (%s), which a rule forbids: %s", e.From, from.Kind, e.To, to.Kind, r.tail()), GroupWires, e.From, ""))
			}
		}
	}
	return out
}

func newDep(e Edge) string {
	return fmt.Sprintf("the %s %s -> %s is a dependency that %s does not hold; if it is wanted, run the command rules lock and commit the change", e.Via, e.From, e.To, LockName)
}

func (e Edge) key() string { return e.Via + " " + e.From + " -> " + e.To }

// Foreign is one thing that can bring code into a project past every check.
type Foreign struct {
	File string // relative to the root, with slashes
	Line int
	Msg  string
}

// ForeignRoots reads what brings code in: each replace of the root go.mod and of a root go.work whose
// target is a local path, and a root go.work itself, which changes what the go command builds
// (plan-r3, amendment 12). A local path is one that modfile.IsDirectoryPath accepts: relative or
// absolute, in the form of any host. The check reads the text alone, and asks the file system
// nothing, so no link, junction, or other form of the path changes its result. A replace by a module
// path and a version passes. An ignore directive of the root go.mod fails too (audit N41): the go
// command leaves an ignored directory out of ./..., so no check loads it, and still builds it when
// something imports it.
//
// The walk rejects each link to a directory in the project, a symlink or a junction (audit N40). The
// go command does not follow one when it expands ./..., so L's check would not load a part behind it,
// and go build follows it. Each arm reads the same walk, so both fail it the same way.
//
// The walk stays for a go.mod in any directory, and for a root vendor/modules.txt. A go.mod that
// nothing replaces brings in nothing. It stays a fault because V's K0002 rejects a Go file in a plain
// directory, and go ./... does not list a nested module, so without the walk L would pass that file.
// It leaves out two places at the root, the same in both arms: .vulcan/build, which holds V's own
// go.mod, and .git/objects, which git alone writes and which can be large.
func ForeignRoots(root string) []Foreign {
	var out []Foreign
	local := func(name string, line int, old, target string) {
		out = append(out, Foreign{name, line, fmt.Sprintf("%s replaces %s with the local path %s, so it can bring in code that the check does not read; a replace must name a module path and a version", name, old, target)})
	}
	replaces := func(name string, reps []*modfile.Replace) {
		for _, rep := range reps {
			if rep.New.Version == "" || modfile.IsDirectoryPath(rep.New.Path) {
				local(name, rep.Syntax.Start.Line, rep.Old.Path, rep.New.Path)
			}
		}
	}
	// A host that is not Windows refuses a replace by a path with a backslash, so the file does not
	// parse there. Each such replace is read from its line, so that it fails the same on each host.
	refused := func(name string, b []byte, err error) {
		var list modfile.ErrorList
		if !errors.As(err, &list) {
			return
		}
		lines := strings.Split(string(b), "\n")
		for _, e := range list {
			if e.Pos.Line < 1 || e.Pos.Line > len(lines) || !strings.Contains(e.Err.Error(), "Windows path") {
				continue
			}
			before, after, ok := strings.Cut(lines[e.Pos.Line-1], "=>")
			left, right := strings.Fields(before), strings.Fields(after)
			if !ok || len(left) == 0 || len(right) == 0 {
				continue
			}
			target := right[0]
			if t, err := strconv.Unquote(target); err == nil {
				target = t
			}
			old := left[0]
			if old == "replace" && len(left) > 1 {
				old = left[1]
			}
			local(name, e.Pos.Line, old, target)
		}
	}
	if b, err := os.ReadFile(filepath.Join(root, "go.mod")); err == nil {
		if f, err := modfile.Parse("go.mod", b, nil); err == nil {
			replaces("go.mod", f.Replace)
			for _, ig := range f.Ignore {
				out = append(out, Foreign{"go.mod", ig.Syntax.Start.Line, fmt.Sprintf("go.mod ignores %s, so the check does not load it and a build can still hold it; remove the ignore directive", ig.Path)})
			}
		} else {
			refused("go.mod", b, err)
		}
	}
	if b, err := os.ReadFile(filepath.Join(root, "go.work")); err == nil {
		out = append(out, Foreign{"go.work", 1, "go.work changes what the go command builds, so the check may read other code than a build does; remove it from the project"})
		if f, err := modfile.ParseWork("go.work", b, nil); err == nil {
			replaces("go.work", f.Replace)
		} else {
			refused("go.work", b, err)
		}
	}
	_ = filepath.WalkDir(root, func(full string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(root, full)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		// A junction is ModeIrregular on Windows, a symlink ModeSymlink. Only one that leads to a
		// directory can hold a package; os.Stat follows it.
		if d.Type()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			if fi, err := os.Stat(full); err == nil && fi.IsDir() {
				out = append(out, Foreign{rel, 1, fmt.Sprintf("%s is a link to a directory, which go ./... does not follow and a build does, so the check may not read code that the build holds; make it a plain directory", rel)})
			}
			return nil
		}
		if d.IsDir() {
			if rel == ".git/objects" || rel == ".vulcan/build" {
				return filepath.SkipDir
			}
			return nil
		}
		// By any case: a file system that ignores case reads GO.MOD as go.mod. It is the same fault
		// on one that does not, so a program has one result on each host.
		switch {
		case strings.EqualFold(d.Name(), "go.mod") && rel != "go.mod":
			out = append(out, Foreign{rel, 1, fmt.Sprintf("%s makes another module inside the project, so a replace can bring in code that no rule reads; move its code into a part, or use the module from outside", rel)})
		case strings.EqualFold(rel, "vendor/modules.txt"):
			out = append(out, Foreign{rel, 1, fmt.Sprintf("%s means that the project builds from vendored code, which no rule reads; remove the vendor directory, and build from the module cache", rel)})
		}
		return nil
	})
	slices.SortFunc(out, func(a, b Foreign) int {
		if c := strings.Compare(a.File, b.File); c != 0 {
			return c
		}
		return a.Line - b.Line
	})
	return out
}

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
	return fmt.Sprintf("%s holds code that is not Go, so no rule can read it; each part, main too, holds only Go code", name)
}
