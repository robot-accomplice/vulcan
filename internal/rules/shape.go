package rules

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// The shape kinds of plan-r3, amendment 6: what a rule can read of the form of the code, with no
// flow of values.
//
//	declaration  a declared name, type, tag, doc comment, or comment
//	call-form    an argument of a call
//	literal      a value of a field in a literal, or a string in the code
//	code-form    a form of statement, named by form

// Port is one port of a leaf. Each arm finds its ports in its own way.
type Port struct {
	Name    string
	Type    types.Type
	Doc     bool
	At      token.Pos
	Methods []PortMethod
}

// PortMethod is one method of a port.
type PortMethod struct {
	Name     string
	Sig      *types.Signature
	Doc      bool
	Requires bool // it has a requires clause
	At       token.Pos
}

// Decl is a declaration that is not Go code in V: an external, a library, or an entry of the check
// configuration.
type Decl struct {
	What      string // "external", "library", or "config-entry"
	Name      string
	Commented bool
	At        any
}

var declarationAts = []string{"port-signature", "port-method", "field", "settings-field", "embedded", "param", "var", "doc", "library", "external", "config-entry", "wired-port"}

var codeForms = []string{"dropped-error", "discard-error", "error-compare", "one-value-assertion", "channel-size", "function-complexity", "switch-exhaustive", "test-logic", "suppression", "return-effect", "retry-loop", "duplicate-function", "unused-name", "deprecated-use"}

func (r *Rule) checkShape(bad func(string, ...any)) {
	regex := func(key string) {
		for _, v := range r.Params[key] {
			if _, err := regexp.Compile(v); err != nil {
				bad("rule %s: %s is not a regular expression: %v", r.ID, key, err)
			}
		}
	}
	num := func(key string) {
		for _, v := range r.Params[key] {
			if _, err := strconv.Atoi(v); err != nil {
				bad("rule %s: %s is a number", r.ID, key)
			}
		}
	}
	regex("deny-text")
	regex("require-text")
	num("limit")
	num("similarity")
	switch r.Kind {
	case "declaration":
		for _, v := range r.Params["at"] {
			if !slices.Contains(declarationAts, v) {
				bad("rule %s: at is one of %s", r.ID, strings.Join(declarationAts, ", "))
			}
		}
		for _, v := range r.Params["to"] {
			if !isSelector(v) {
				bad("rule %s: to names %q, which is not a kind or the directory of a part", r.ID, v)
			}
		}
		r.checkDeclaration(bad)
	case "call-form":
		if (len(r.Params["funcs"]) > 0) == (len(r.Params["param-type"]) > 0) {
			bad("rule %s of kind call-form needs funcs or param-type", r.ID)
		}
		for _, v := range r.Params["require"] {
			if v != "constant" && v != "not-nil" && v != "unexported-type" && !strings.HasPrefix(v, "tagged=") {
				bad("rule %s: require is constant, not-nil, unexported-type, or tagged=KEY", r.ID)
			}
		}
		if len(r.Params["require"])+len(r.Params["deny-text"])+len(r.Params["require-text"]) == 0 {
			bad("rule %s of kind call-form needs require, deny-text, or require-text", r.ID)
		}
	case "literal":
		typed, text := len(r.Params["type"]) > 0, len(r.Params["deny-text"])+len(r.Params["names"]) > 0
		if typed == text {
			bad("rule %s of kind literal needs type, or else deny-text or names", r.ID)
		}
		if typed && len(r.Params["require"])+len(r.Params["deny"])+len(r.Params["below"]) == 0 {
			bad("rule %s of kind literal with a type needs require, deny, or below", r.ID)
		}
		for _, v := range r.Params["below"] {
			if _, n, ok := strings.Cut(v, "="); !ok {
				bad("rule %s: below holds Field=NUMBER", r.ID)
			} else if _, err := strconv.Atoi(n); err != nil {
				bad("rule %s: below holds Field=NUMBER", r.ID)
			}
		}
	case "code-form":
		for _, v := range r.Params["form"] {
			if !slices.Contains(codeForms, v) {
				bad("rule %s: form is one of %s", r.ID, strings.Join(codeForms, ", "))
			}
		}
		if len(r.Params["form"]) > 1 {
			bad("rule %s: form names one form", r.ID)
		}
	}
}

// checkDeclaration reports a rule that would read a place and apply nothing to it, and a word of
// require that its place does not take.
func (r *Rule) checkDeclaration(bad func(string, ...any)) {
	has := func(keys ...string) bool {
		for _, k := range keys {
			if len(r.Params[k]) > 0 {
				return true
			}
		}
		return false
	}
	for _, at := range r.Params["at"] {
		applies := true
		switch at {
		case "field", "var", "port-signature":
			applies = has("deny-types")
		case "param":
			applies = has("deny-types", "require")
		case "port-method", "settings-field", "external", "config-entry":
			applies = has("require")
		case "library":
			applies = has("deny-names", "require")
		case "wired-port":
			applies = has("to")
		}
		if !applies {
			bad("rule %s: at %s has nothing to apply: it needs a parameter that says what to reject", r.ID, at)
		}
		for _, q := range r.Params["require"] {
			k, _, hasValue := strings.Cut(q, "=")
			ok := false
			switch at {
			case "port-method":
				ok = q == "requires"
			case "param":
				ok = q == "first" || k == "name" && hasValue
			case "settings-field":
				ok = hasValue
			case "external", "config-entry", "library":
				ok = q == "comment"
			}
			if !ok {
				bad("rule %s: at %s has no require %q", r.ID, at, q)
			}
		}
	}
}

// ResolveTypes reports a table whose type or field the package of its directory does not have. A
// name outside the project is not checked.
func (s *Set) ResolveTypes(u *Unit) []Problem {
	if s == nil || u.Pkg == nil {
		return nil
	}
	var out []Problem
	for _, r := range s.File.Rules {
		if r.Kind != "transitions" || len(r.Params["type"]) != 1 {
			continue
		}
		dir, name, _ := strings.Cut(r.Params["type"][0], ".")
		if dir != u.Part.Dir {
			continue
		}
		obj := u.Pkg.Scope().Lookup(name)
		if obj == nil {
			out = append(out, Problem{r.Line, fmt.Sprintf("rule %s: type names %s, which package %s does not declare", r.ID, name, dir)})
			continue
		}
		st, ok := obj.Type().Underlying().(*types.Struct)
		field := first(r, "field", "")
		found := false
		for i := 0; ok && i < st.NumFields(); i++ {
			found = found || st.Field(i).Name() == field
		}
		if !found {
			out = append(out, Problem{r.Line, fmt.Sprintf("rule %s: type %s has no field %s", r.ID, name, field)})
		}
	}
	return out
}

func first(r *Rule, key, def string) string {
	if v := r.Params[key]; len(v) > 0 {
		return v[0]
	}
	return def
}

func applies(r *Rule, p Part) bool {
	return (len(r.Params["in"]) == 0 || selected(r.Params["in"], p)) && !selected(r.Params["except"], p)
}

// glob matches a name against words where * is any text. It is case-sensitive, so *At is not format.
func glob(words []string, name string) bool {
	for _, w := range words {
		if ok, _ := path.Match(w, name); ok {
			return true
		}
	}
	return false
}

// Comments says which lines of a source file hold a comment. Both arms read the file that a person
// wrote, so a comment that a tool generates (V's /*line*/ and its header) does not count. A comment
// after code on the line above belongs to that code.
type Comments struct {
	trailing, full map[int]bool
}

// ScanComments reads the comments of a source file. Go's scanner reads a .vulcan file too: a
// comment there has Go's form.
func ScanComments(src []byte) *Comments {
	c := &Comments{map[int]bool{}, map[int]bool{}}
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(src))
	var s scanner.Scanner
	s.Init(file, src, func(token.Position, string) {}, scanner.ScanComments)
	last := 0
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return c
		}
		line := fset.Position(pos).Line
		if tok != token.COMMENT {
			if !(tok == token.SEMICOLON && lit == "\n") {
				last = line
			}
			continue
		}
		if strings.HasPrefix(lit, "//line") || strings.HasPrefix(lit, "/*line") {
			continue
		}
		end := line + strings.Count(lit, "\n")
		for l := line; l <= end; l++ {
			if l == last {
				c.trailing[l] = true
			} else {
				c.full[l] = true
			}
		}
	}
}

// Near reports whether a comment is on a line (from 1), or fills the line above it.
func (c *Comments) Near(line int) bool {
	return c.trailing[line] || c.full[line] || c.full[line-1]
}

func (c *Comments) any(from, to int) bool {
	for l := from; l <= to; l++ {
		if c.trailing[l] || c.full[l] {
			return true
		}
	}
	return false
}

var scanned sync.Map // file name -> *Comments

func commentsOf(file string) *Comments {
	if c, ok := scanned.Load(file); ok {
		return c.(*Comments)
	}
	b, err := os.ReadFile(file)
	if err != nil {
		b = nil
	}
	c := ScanComments(b)
	scanned.Store(file, c)
	return c
}

// commentNear reports whether a comment is on the line of pos or fills the line above it.
func (u *Unit) commentNear(pos token.Pos) bool {
	p := u.Fset.Position(pos)
	return commentsOf(p.Filename).Near(p.Line)
}

// generated reports V's names of generated code, which no rule reads. In L, such a name is
// an ordinary name.
func (u *Unit) generated(name string) bool {
	return u.Generates && (strings.HasPrefix(name, "vulcan_") || strings.HasPrefix(name, "Vulcan_"))
}

// Declaration ------------------------------------------------------------------------------------

func (u *Unit) declaration(r *Rule, add addFunc) {
	if !applies(r, u.Part) {
		return
	}
	for _, at := range r.Params["at"] {
		switch at {
		case "port-signature":
			u.portSignature(r, add)
		case "port-method":
			for _, p := range u.Ports {
				for _, m := range p.Methods {
					if glob(r.Params["except-names"], m.Name) || (m.Sig != nil && !hasArguments(m.Sig)) {
						continue
					}
					if slices.Contains(r.Params["require"], "requires") && !m.Requires {
						add("0807", m.At, "port method %s.%s has no requires clause for the limits of its arguments: %s", p.Name, m.Name, r.tail())
					}
				}
			}
		case "doc":
			u.docs(r, add)
		case "field", "settings-field", "embedded", "param", "var":
			u.declared(r, at, add)
		}
	}
}

func hasArguments(sig *types.Signature) bool {
	for i := range sig.Params().Len() {
		if types.TypeString(sig.Params().At(i).Type(), nil) != "context.Context" {
			return true
		}
	}
	return false
}

func (u *Unit) portSignature(r *Rule, add addFunc) {
	for _, p := range u.Ports {
		for _, m := range p.Methods {
			if m.Sig == nil {
				continue
			}
			for _, tuple := range []*types.Tuple{m.Sig.Params(), m.Sig.Results()} {
				for i := range tuple.Len() {
					if bad := carried(tuple.At(i).Type(), r.Params["deny-types"], map[types.Type]bool{}); bad != "" {
						add("0807", m.At, "port method %s.%s carries %s: %s", p.Name, m.Name, bad, r.tail())
					}
				}
			}
		}
	}
}

// carried names a listed type that t is or holds, or "".
func carried(t types.Type, deny []string, seen map[types.Type]bool) string {
	if seen[t] {
		return ""
	}
	seen[t] = true
	t = types.Unalias(t)
	if s := types.TypeString(t, nil); slices.Contains(deny, s) {
		return s
	}
	switch x := t.Underlying().(type) {
	case *types.Pointer:
		return carried(x.Elem(), deny, seen)
	case *types.Slice:
		return carried(x.Elem(), deny, seen)
	case *types.Array:
		return carried(x.Elem(), deny, seen)
	case *types.Map:
		if s := carried(x.Key(), deny, seen); s != "" {
			return s
		}
		return carried(x.Elem(), deny, seen)
	case *types.Struct:
		for i := range x.NumFields() {
			if s := carried(x.Field(i).Type(), deny, seen); s != "" {
				return s
			}
		}
	}
	return ""
}

func (u *Unit) docs(r *Rule, add addFunc) {
	for _, p := range u.Ports {
		if !p.Doc {
			add("0807", p.At, "port %s has no doc comment: %s", p.Name, r.tail())
		}
		for _, m := range p.Methods {
			if !m.Doc {
				add("0807", m.At, "port method %s.%s has no doc comment: %s", p.Name, m.Name, r.tail())
			}
		}
	}
	missing := func(id *ast.Ident) {
		if !u.generated(id.Name) {
			add("0807", id.Pos(), "the exported name %s has no doc comment: %s", id.Name, r.tail())
		}
	}
	fields := func(st *ast.StructType) {
		for _, fd := range st.Fields.List {
			if fd.Doc == nil && fd.Comment == nil {
				for _, id := range fd.Names {
					if id.IsExported() {
						missing(id)
					}
				}
			}
		}
	}
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Name.IsExported() && d.Doc == nil && (d.Recv == nil || exportedRecv(d)) && !isTestName(u, f, d) {
					missing(d.Name)
				}
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					switch sp := sp.(type) {
					case *ast.TypeSpec:
						if sp.Name.IsExported() && d.Doc == nil && sp.Doc == nil && sp.Comment == nil {
							missing(sp.Name)
						}
						if st, ok := sp.Type.(*ast.StructType); ok && sp.Name.IsExported() {
							fields(st)
						}
						if it, ok := sp.Type.(*ast.InterfaceType); ok && sp.Name.IsExported() {
							for _, fd := range it.Methods.List {
								if fd.Doc == nil && fd.Comment == nil {
									for _, id := range fd.Names {
										if id.IsExported() && !u.isPortName(sp.Name.Name) {
											missing(id)
										}
									}
								}
							}
						}
					case *ast.ValueSpec:
						if d.Doc == nil && sp.Doc == nil && sp.Comment == nil {
							for _, id := range sp.Names {
								if id.IsExported() {
									missing(id)
								}
							}
						}
					}
				}
			}
		}
	}
}

func (u *Unit) isPortName(name string) bool {
	return slices.ContainsFunc(u.Ports, func(p Port) bool { return p.Name == name })
}

func isTestName(u *Unit, f *ast.File, d *ast.FuncDecl) bool {
	return u.IsTest(f) && strings.HasPrefix(d.Name.Name, "Test")
}

func exportedRecv(d *ast.FuncDecl) bool {
	if len(d.Recv.List) != 1 {
		return false
	}
	t := d.Recv.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	if ix, ok := t.(*ast.IndexExpr); ok {
		t = ix.X
	}
	id, ok := t.(*ast.Ident)
	return ok && id.IsExported()
}

// settingsTypes lists the structs of the package that a function takes and that hold only basic
// exported fields: the form of settings in both arms.
func (u *Unit) settingsTypes() map[string]bool {
	out := map[string]bool{}
	for _, f := range u.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv != nil {
				continue
			}
			for _, field := range fd.Type.Params.List {
				n := namedOf(u.Info.TypeOf(field.Type))
				if n == nil || n.Obj().Pkg() == nil || !strings.HasPrefix(n.Obj().Pkg().Path(), u.Module) {
					continue
				}
				st, ok := n.Underlying().(*types.Struct)
				if !ok || st.NumFields() == 0 {
					continue
				}
				basic := true
				for i := range st.NumFields() {
					_, b := st.Field(i).Type().Underlying().(*types.Basic)
					basic = basic && b && st.Field(i).Exported()
				}
				if basic {
					out[n.Obj().Name()] = true
				}
			}
		}
	}
	return out
}

// declared reads each declared field, parameter, or variable.
func (u *Unit) declared(r *Rule, at string, add addFunc) {
	wantTypes := r.Params["types"]
	checkName := func(name string, t types.Type, at token.Pos) {
		if t == nil || name == "_" || u.generated(name) {
			return
		}
		if names := r.Params["names"]; len(names) > 0 && !glob(names, name) {
			return
		}
		if len(wantTypes) > 0 && typeIn(t, wantTypes, map[types.Type]bool{}) == "" {
			return
		}
		if s := typeIn(t, r.Params["deny-types"], map[types.Type]bool{}); s != "" {
			add("0807", at, "%s has the type %s here: %s", name, s, r.tail())
		}
	}
	check := func(id *ast.Ident, t types.Type) {
		if id == nil {
			return
		}
		checkName(id.Name, t, id.Pos())
	}
	settings := map[string]bool{}
	if at == "settings-field" {
		settings = u.settingsTypes()
	}
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.TypeSpec:
				if at == "field" && x.Assign.IsValid() {
					// An alias of a generic struct: its fields hold the type argument.
					if n, ok := types.Unalias(u.Info.TypeOf(x.Type)).(*types.Named); ok && n.TypeArgs() != nil {
						checkName(x.Name.Name, n, x.Name.Pos())
					}
				}
				st, ok := x.Type.(*ast.StructType)
				if !ok {
					return true
				}
				switch at {
				case "embedded":
					if x.Name.IsExported() {
						for _, fd := range st.Fields.List {
							if len(fd.Names) == 0 {
								add("0807", fd.Pos(), "the exported struct %s embeds %s: %s", x.Name.Name, types.ExprString(fd.Type), r.tail())
							}
						}
					}
				case "settings-field":
					if settings[x.Name.Name] {
						u.settingsFields(r, st, add)
					}
				}
			case *ast.StructType:
				if at == "field" {
					for _, fd := range x.Fields.List {
						for _, id := range fd.Names {
							check(id, u.Info.TypeOf(fd.Type))
						}
						if len(fd.Names) == 0 {
							if s := typeIn(u.Info.TypeOf(fd.Type), r.Params["deny-types"], map[types.Type]bool{}); s != "" {
								add("0807", fd.Pos(), "a struct holds %s: %s", s, r.tail())
							}
						}
					}
				}
			case *ast.FuncType:
				if at == "param" {
					u.params(r, x, check, add)
				}
			case *ast.ValueSpec:
				if at == "var" {
					for _, id := range x.Names {
						if obj := u.Info.Defs[id]; obj != nil {
							check(id, obj.Type())
						}
					}
				}
			case *ast.AssignStmt:
				if at == "var" && x.Tok == token.DEFINE {
					for _, l := range x.Lhs {
						if id, ok := l.(*ast.Ident); ok {
							if obj := u.Info.Defs[id]; obj != nil {
								check(id, obj.Type())
							}
						}
					}
				}
			}
			return true
		})
	}
	// In L, ports.go is among the files, so the walk above read each port. In V, it is generated.
	if at == "param" && len(u.PortFiles) > 0 {
		for _, p := range u.Ports {
			for _, m := range p.Methods {
				if m.Sig == nil {
					continue
				}
				params := m.Sig.Params()
				for i := range params.Len() {
					checkName(params.At(i).Name(), params.At(i).Type(), m.At)
				}
				results := m.Sig.Results()
				for i := range results.Len() {
					checkName(results.At(i).Name(), results.At(i).Type(), m.At)
				}
				u.portParams(r, m, add)
			}
		}
	}
}

// typeIn names a listed type that t is, through an alias, a pointer, a slice, an array, a map, a
// type argument, or an embedded interface; or "". A type defined from a listed type is another
// type, and is not found.
func typeIn(t types.Type, list []string, seen map[types.Type]bool) string {
	if t == nil || len(list) == 0 || seen[t] {
		return ""
	}
	seen[t] = true
	t = types.Unalias(t)
	if s := types.TypeString(t, nil); slices.Contains(list, s) {
		return s
	}
	switch x := t.(type) {
	case *types.Pointer:
		return typeIn(x.Elem(), list, seen)
	case *types.Slice:
		return typeIn(x.Elem(), list, seen)
	case *types.Array:
		return typeIn(x.Elem(), list, seen)
	case *types.Map:
		if s := typeIn(x.Key(), list, seen); s != "" {
			return s
		}
		return typeIn(x.Elem(), list, seen)
	case *types.Named:
		if args := x.TypeArgs(); args != nil {
			for i := range args.Len() {
				if s := typeIn(args.At(i), list, seen); s != "" {
					return s
				}
			}
		}
		if _, ok := x.Underlying().(*types.Interface); ok {
			return typeIn(x.Underlying(), list, seen)
		}
	case *types.Interface:
		for i := range x.NumEmbeddeds() {
			if s := typeIn(x.EmbeddedType(i), list, seen); s != "" {
				return s
			}
		}
	}
	return ""
}

func (u *Unit) settingsFields(r *Rule, st *ast.StructType, add addFunc) {
	for _, fd := range st.Fields.List {
		for _, id := range fd.Names {
			if !glob(r.Params["names"], id.Name) {
				continue
			}
			tag := ""
			if fd.Tag != nil {
				tag, _ = strconv.Unquote(fd.Tag.Value)
			}
			for _, want := range r.Params["require"] {
				k, v, _ := strings.Cut(want, "=")
				if !strings.Contains(tag, k+`:"`+v+`"`) {
					add("0807", id.Pos(), "the settings field %s has no tag %s:%q: %s", id.Name, k, v, r.tail())
				}
			}
		}
	}
}

// params checks the place and the name of a parameter of a listed type.
func (u *Unit) params(r *Rule, ft *ast.FuncType, check func(*ast.Ident, types.Type), add addFunc) {
	if ft.Results != nil {
		for _, fd := range ft.Results.List {
			for _, id := range fd.Names {
				check(id, u.Info.TypeOf(fd.Type))
			}
		}
	}
	if ft.Params == nil {
		return
	}
	i := 0
	for _, fd := range ft.Params.List {
		t := u.Info.TypeOf(fd.Type)
		names := fd.Names
		if len(names) == 0 {
			names = []*ast.Ident{nil}
		}
		for _, id := range names {
			if id != nil {
				check(id, t)
			}
			if t != nil && typeIn(t, r.Params["types"], map[types.Type]bool{}) != "" {
				at := fd.Pos()
				name := ""
				if id != nil {
					at, name = id.Pos(), id.Name
				}
				u.paramForm(r, i, name, at, add)
			}
			i++
		}
	}
}

func (u *Unit) portParams(r *Rule, m PortMethod, add addFunc) {
	for i := range m.Sig.Params().Len() {
		v := m.Sig.Params().At(i)
		if typeIn(v.Type(), r.Params["types"], map[types.Type]bool{}) != "" {
			u.paramForm(r, i, v.Name(), m.At, add)
		}
	}
}

func (u *Unit) paramForm(r *Rule, index int, name string, at token.Pos, add addFunc) {
	for _, want := range r.Params["require"] {
		k, v, _ := strings.Cut(want, "=")
		switch k {
		case "first":
			if index != 0 {
				add("0807", at, "a parameter of a listed type is not the first parameter: %s", r.tail())
			}
		case "name":
			if name != v {
				add("0807", at, "a parameter of a listed type is named %q, not %q: %s", name, v, r.tail())
			}
		}
	}
}

// CheckDecls applies the declaration rules to what is not Go code in V.
func (s *Set) CheckDecls(decls []Decl) []Finding {
	if s == nil {
		return nil
	}
	var out []Finding
	for _, r := range s.on {
		if r.Kind != "declaration" {
			continue
		}
		for _, d := range decls {
			for _, at := range r.Params["at"] {
				switch {
				case at == "doc" && (d.What == "external" || d.What == "library") && !d.Commented:
					out = append(out, Finding{"0807", d.At, fmt.Sprintf("%s %s has no doc comment: %s", d.What, d.Name, r.tail()), r.Level == Warn})
				case at == d.What && slices.Contains(r.Params["require"], "comment") && !d.Commented:
					out = append(out, Finding{"0807", d.At, fmt.Sprintf("the %s %s has no comment that gives the reason: %s", d.What, d.Name, r.tail()), r.Level == Warn})
				case at == d.What && slices.Contains(r.Params["deny-names"], d.Name):
					out = append(out, Finding{"0807", d.At, fmt.Sprintf("the %s has the name %s: %s", d.What, d.Name, r.tail()), r.Level == Warn})
				}
			}
		}
	}
	return out
}

// checkWiredPorts applies a declaration rule to the ports of each wire.
func wiredPorts(r *Rule, parts map[string]Part, wires []Edge) []Finding {
	var out []Finding
	for _, e := range wires {
		if !selected(r.Params["to"], parts[e.To]) {
			continue
		}
		for _, m := range e.Methods {
			if m.Sig == nil {
				continue
			}
			p := m.Sig.Params()
			if p.Len() == 0 || types.TypeString(p.At(0).Type(), nil) != "context.Context" {
				out = append(out, Finding{"0807", e.At, fmt.Sprintf("port method %s of %s has no context.Context as its first parameter, and %s provides it: %s", m.Name, e.From, e.To, r.tail()), r.Level == Warn})
			}
		}
	}
	return out
}

// Call form --------------------------------------------------------------------------------------

func (u *Unit) callForm(r *Rule, add addFunc) {
	if !applies(r, u.Part) {
		return
	}
	funcs := map[string]bool{}
	for _, f := range r.Params["funcs"] {
		funcs[f] = true
	}
	paramType := first(r, "param-type", "")
	comment := first(r, "unless-comment", "") == "yes"
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		u.funcValues(r, f, funcs, add)
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok || u.Info.Types[c.Fun].IsType() {
				return true
			}
			sig, _ := u.Info.TypeOf(c.Fun).(*types.Signature)
			if sig == nil {
				return true
			}
			if paramType != "" {
				for i, a := range c.Args {
					if i < sig.Params().Len() && types.TypeString(sig.Params().At(i).Type(), nil) == paramType {
						u.argument(r, a, f, comment, add)
					}
				}
				return true
			}
			fn := calledFunc(u.Info, c)
			if fn == nil || !matches(funcs, fn.FullName()) {
				return true
			}
			i := argIndex(r, sig)
			if i >= 0 && i < len(c.Args) {
				u.argument(r, c.Args[i], f, comment, add)
			}
			return true
		})
	}
}

// funcValues rejects a listed function that the code uses as a value (f := d.Exec), because this
// kind then cannot read the argument of its call.
func (u *Unit) funcValues(r *Rule, f *ast.File, funcs map[string]bool, add addFunc) {
	if len(funcs) == 0 {
		return
	}
	called := map[*ast.Ident]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			fun := ast.Unparen(c.Fun)
			if ix, ok := fun.(*ast.IndexExpr); ok {
				fun = ast.Unparen(ix.X)
			}
			switch x := fun.(type) {
			case *ast.Ident:
				called[x] = true
			case *ast.SelectorExpr:
				called[x.Sel] = true
			}
		}
		return true
	})
	ast.Inspect(f, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && !called[id] {
			if fn, ok := u.Info.Uses[id].(*types.Func); ok && matches(funcs, fn.FullName()) {
				add("0808", id.Pos(), "this uses %s as a value, so its argument cannot be read: %s", fn.FullName(), r.tail())
			}
		}
		return true
	})
}

func calledFunc(info *types.Info, c *ast.CallExpr) *types.Func {
	fun := ast.Unparen(c.Fun)
	if ix, ok := fun.(*ast.IndexExpr); ok {
		fun = ix.X
	}
	var id *ast.Ident
	switch x := fun.(type) {
	case *ast.Ident:
		id = x
	case *ast.SelectorExpr:
		id = x.Sel
	}
	if id == nil {
		return nil
	}
	fn, _ := info.Uses[id].(*types.Func)
	return fn
}

// argIndex gives the argument that a rule reads: arg N, or by default the first of type string.
func argIndex(r *Rule, sig *types.Signature) int {
	if v := first(r, "arg", ""); v != "" {
		n, _ := strconv.Atoi(v)
		return n
	}
	for i := range sig.Params().Len() {
		if b, ok := sig.Params().At(i).Type().Underlying().(*types.Basic); ok && b.Info()&types.IsString != 0 {
			return i
		}
	}
	return -1
}

func (u *Unit) argument(r *Rule, a ast.Expr, f *ast.File, comment bool, add addFunc) {
	if comment && u.commentNear(a.Pos()) {
		return
	}
	tv := u.Info.Types[a]
	for _, want := range r.Params["require"] {
		k, v, _ := strings.Cut(want, "=")
		switch k {
		case "constant":
			if tv.Value == nil {
				add("0808", a.Pos(), "this argument is not a constant: %s", r.tail())
			}
		case "not-nil":
			if tv.IsNil() {
				add("0808", a.Pos(), "this argument is nil: %s", r.tail())
			}
		case "unexported-type":
			n := namedOf(tv.Type)
			if n == nil || n.Obj().Exported() || n.Obj().Pkg() == nil {
				add("0808", a.Pos(), "this argument is not of an unexported named type: %s", r.tail())
			}
		case "tagged":
			if bad := untagged(tv.Type, v, map[types.Type]bool{}); bad != "" {
				add("0808", a.Pos(), "the field %s has no %s tag: %s", bad, v, r.tail())
			}
		}
	}
	if tv.Value != nil && tv.Value.Kind() == constant.String {
		text := constant.StringVal(tv.Value)
		for _, re := range r.Params["deny-text"] {
			if m, _ := regexp.MatchString(re, text); m {
				add("0808", a.Pos(), "the text %q matches %s: %s", text, re, r.tail())
			}
		}
		for _, re := range r.Params["require-text"] {
			if m, _ := regexp.MatchString(re, text); !m {
				add("0808", a.Pos(), "the text %q does not match %s: %s", text, re, r.tail())
			}
		}
	}
}

// untagged names an exported field of a struct that has no tag with the key, or "".
func untagged(t types.Type, key string, seen map[types.Type]bool) string {
	if t == nil || seen[t] {
		return ""
	}
	seen[t] = true
	switch x := t.Underlying().(type) {
	case *types.Pointer:
		return untagged(x.Elem(), key, seen)
	case *types.Slice:
		return untagged(x.Elem(), key, seen)
	case *types.Array:
		return untagged(x.Elem(), key, seen)
	case *types.Map:
		return untagged(x.Elem(), key, seen)
	case *types.Struct:
		for i := range x.NumFields() {
			fd := x.Field(i)
			if !fd.Exported() {
				continue
			}
			if !strings.Contains(x.Tag(i), key+`:"`) {
				return fd.Name()
			}
			if s := untagged(fd.Type(), key, seen); s != "" {
				return s
			}
		}
	}
	return ""
}

// Literal ----------------------------------------------------------------------------------------

func (u *Unit) literal(r *Rule, add addFunc) {
	if !applies(r, u.Part) {
		return
	}
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		if len(r.Params["type"]) > 0 {
			u.typeLiterals(r, f, add)
		} else {
			u.stringLiterals(r, f, add)
		}
	}
}

func isZeroConstant(tv types.TypeAndValue) bool {
	if tv.IsNil() {
		return true
	}
	if tv.Value == nil {
		return false
	}
	switch tv.Value.Kind() {
	case constant.Bool:
		return !constant.BoolVal(tv.Value)
	case constant.String:
		return constant.StringVal(tv.Value) == ""
	case constant.Int, constant.Float:
		return constant.Sign(tv.Value) == 0
	}
	return false
}

func (u *Unit) typeLiterals(r *Rule, f *ast.File, add addFunc) {
	want := r.Params["type"]
	isT := func(t types.Type) bool { return t != nil && slices.Contains(want, fullTypeName(t)) }
	valueBad := func(field string, tv types.TypeAndValue, at token.Pos) {
		for _, d := range r.Params["deny"] {
			k, v, _ := strings.Cut(d, "=")
			if k == field && tv.Value != nil && stateOf(tv.Value) == v {
				add("0809", at, "%s is %s here: %s", field, v, r.tail())
			}
		}
		for _, b := range r.Params["below"] {
			k, v, _ := strings.Cut(b, "=")
			limit, _ := strconv.ParseInt(v, 10, 64)
			if k == field && tv.Value != nil {
				if n, ok := constant.Int64Val(constant.ToInt(tv.Value)); ok && n < limit {
					add("0809", at, "%s is below %d here: %s", field, limit, r.tail())
				}
			}
		}
	}
	requires := func(set map[string]types.TypeAndValue, name string, at token.Pos) {
		for _, q := range r.Params["require"] {
			k, v, hasValue := strings.Cut(q, "=")
			tv, ok := set[k]
			switch {
			case !ok || isZeroConstant(tv):
				add("0809", at, "this %s does not set %s: %s", name, k, r.tail())
			case hasValue && tv.Value != nil && !slices.Contains(strings.Split(v, ","), stateOf(tv.Value)):
				add("0809", at, "this %s sets %s to another value than %s: %s", name, k, v, r.tail())
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CompositeLit:
			if !isT(u.Info.TypeOf(x)) {
				return true
			}
			set := map[string]types.TypeAndValue{}
			for _, e := range x.Elts {
				if kv, ok := e.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok {
						tv := u.Info.Types[kv.Value]
						set[id.Name] = tv
						valueBad(id.Name, tv, kv.Value.Pos())
					}
				}
			}
			requires(set, fullTypeName(u.Info.TypeOf(x)), x.Pos())
		case *ast.CallExpr:
			if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok && id.Name == "new" && len(x.Args) == 1 && isT(u.Info.TypeOf(x.Args[0])) {
				if _, builtin := u.Info.Uses[id].(*types.Builtin); builtin {
					requires(nil, fullTypeName(u.Info.TypeOf(x.Args[0])), x.Pos())
				}
			}
		case *ast.ValueSpec:
			if len(x.Values) == 0 && x.Type != nil && isT(u.Info.TypeOf(x.Type)) {
				if _, ptr := types.Unalias(u.Info.TypeOf(x.Type)).(*types.Pointer); !ptr {
					requires(nil, fullTypeName(u.Info.TypeOf(x.Type)), x.Pos())
				}
			}
		case *ast.AssignStmt:
			for i, l := range x.Lhs {
				sel, ok := ast.Unparen(l).(*ast.SelectorExpr)
				if !ok || len(x.Lhs) != len(x.Rhs) || !isT(u.Info.TypeOf(sel.X)) {
					continue
				}
				valueBad(sel.Sel.Name, u.Info.Types[x.Rhs[i]], x.Rhs[i].Pos())
			}
		}
		return true
	})
}

func (u *Unit) stringLiterals(r *Rule, f *ast.File, add addFunc) {
	names := r.Params["names"]
	named := func(id *ast.Ident, v ast.Expr) {
		if id == nil || !glob(names, id.Name) {
			return
		}
		if tv := u.Info.Types[v]; tv.Value != nil && tv.Value.Kind() == constant.String && constant.StringVal(tv.Value) != "" {
			add("0809", v.Pos(), "%s holds a text in the code: %s", id.Name, r.tail())
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ImportSpec:
			return false
		case *ast.Field:
			if x.Tag != nil {
				ast.Inspect(x.Type, func(ast.Node) bool { return true })
				return false // a struct with a tag holds no string value here
			}
		case *ast.BasicLit:
			if x.Kind == token.STRING {
				if s, err := strconv.Unquote(x.Value); err == nil {
					for _, re := range r.Params["deny-text"] {
						if m, _ := regexp.MatchString(re, s); m {
							add("0809", x.Pos(), "the text %q matches %s: %s", s, re, r.tail())
						}
					}
				}
			}
		case *ast.ValueSpec:
			for i, id := range x.Names {
				if i < len(x.Values) {
					named(id, x.Values[i])
				}
			}
		case *ast.AssignStmt:
			if len(x.Lhs) == len(x.Rhs) {
				for i, l := range x.Lhs {
					switch t := ast.Unparen(l).(type) {
					case *ast.Ident:
						named(t, x.Rhs[i])
					case *ast.SelectorExpr:
						named(t.Sel, x.Rhs[i])
					}
				}
			}
		case *ast.KeyValueExpr:
			if id, ok := x.Key.(*ast.Ident); ok {
				named(id, x.Value)
			}
		}
		return true
	})
}

// Code form --------------------------------------------------------------------------------------

func (u *Unit) codeForm(r *Rule, add addFunc) {
	if !applies(r, u.Part) {
		return
	}
	form := first(r, "form", "")
	if form == "duplicate-function" || form == "unused-name" {
		if form == "duplicate-function" {
			u.duplicates(r, add)
		} else {
			u.unused(r, add)
		}
		return
	}
	comment := first(r, "unless-comment", "") == "yes"
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		report := func(at token.Pos, format string, args ...any) {
			if comment && u.commentNear(at) {
				return
			}
			add("0810", at, "%s: %s", fmt.Sprintf(format, args...), r.tail())
		}
		switch form {
		case "dropped-error":
			u.droppedErrors(r, f, report)
		case "discard-error":
			u.discards(f, report)
		case "error-compare":
			u.errorCompares(f, report)
		case "one-value-assertion":
			u.oneValue(f, report)
		case "channel-size":
			u.channelSizes(f, report)
		case "function-complexity":
			u.complexity(r, f, report)
		case "switch-exhaustive":
			u.exhaustive(f, report)
		case "test-logic":
			u.testLogic(f, report)
		case "suppression":
			u.suppressions(f, report)
		case "return-effect":
			u.returnEffects(r, f, report)
		case "retry-loop":
			u.retries(f, report)
		case "deprecated-use":
			u.deprecated(f, report)
		}
	}
}

type reportFunc func(at token.Pos, format string, args ...any)

func hasErrorResult(t types.Type) bool {
	switch x := t.(type) {
	case *types.Tuple:
		for i := range x.Len() {
			if types.TypeString(x.At(i).Type(), nil) == "error" {
				return true
			}
		}
	case nil:
	default:
		return types.TypeString(t, nil) == "error"
	}
	return false
}

func (u *Unit) droppedErrors(r *Rule, f *ast.File, report reportFunc) {
	deferredOK := map[string]bool{}
	for _, v := range r.Params["deferred-ok"] {
		deferredOK[v] = true
	}
	call := func(c *ast.CallExpr, deferred bool) {
		if u.Info.Types[c.Fun].IsType() || !hasErrorResult(u.Info.TypeOf(c)) {
			return
		}
		fn := calledFunc(u.Info, c)
		name := types.ExprString(c.Fun)
		if fn != nil {
			name = fn.FullName()
			if deferred && matches(deferredOK, name) {
				return
			}
		}
		report(c.Pos(), "this call drops the error of %s", name)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.ExprStmt:
			if c, ok := ast.Unparen(x.X).(*ast.CallExpr); ok {
				call(c, false)
			}
		case *ast.DeferStmt:
			call(x.Call, true)
		case *ast.GoStmt:
			call(x.Call, false)
		}
		return true
	})
}

func (u *Unit) discards(f *ast.File, report reportFunc) {
	ast.Inspect(f, func(n ast.Node) bool {
		x, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		tuple := func(i int) types.Type {
			if len(x.Rhs) == len(x.Lhs) {
				return u.Info.TypeOf(x.Rhs[i])
			}
			if len(x.Rhs) == 1 {
				if t, ok := u.Info.TypeOf(x.Rhs[0]).(*types.Tuple); ok && i < t.Len() {
					return t.At(i).Type()
				}
			}
			return nil
		}
		for i, l := range x.Lhs {
			if id, ok := l.(*ast.Ident); ok && id.Name == "_" {
				if t := tuple(i); t != nil && types.TypeString(t, nil) == "error" {
					report(x.Pos(), "this assignment discards an error, with no comment that gives the reason")
				}
			}
		}
		return true
	})
}

func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	if types.TypeString(t, nil) == "error" {
		return true
	}
	errType := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	return types.Implements(t, errType)
}

func (u *Unit) isErrorText(e ast.Expr) bool {
	c, ok := ast.Unparen(e).(*ast.CallExpr)
	if !ok || len(c.Args) != 0 {
		return false
	}
	sel, ok := ast.Unparen(c.Fun).(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Error" && isErrorType(u.Info.TypeOf(sel.X))
}

func (u *Unit) errorCompares(f *ast.File, report reportFunc) {
	textFuncs := set(`strings.Contains strings.HasPrefix strings.HasSuffix strings.EqualFold`)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr:
			if x.Op != token.EQL && x.Op != token.NEQ {
				return true
			}
			if u.isErrorText(x.X) || u.isErrorText(x.Y) {
				report(x.Pos(), "this compares the text of an error")
				return true
			}
			if u.Info.Types[x.X].IsNil() || u.Info.Types[x.Y].IsNil() {
				return true
			}
			if isErrorType(u.Info.TypeOf(x.X)) && isErrorType(u.Info.TypeOf(x.Y)) {
				report(x.Pos(), "this compares errors with %s", x.Op)
			}
		case *ast.TypeAssertExpr:
			if x.Type != nil && types.TypeString(u.Info.TypeOf(x.X), nil) == "error" {
				report(x.Pos(), "this uses a type assertion on an error")
			}
		case *ast.TypeSwitchStmt:
			var e ast.Expr
			switch a := x.Assign.(type) {
			case *ast.ExprStmt:
				e = a.X
			case *ast.AssignStmt:
				if len(a.Rhs) == 1 {
					e = a.Rhs[0]
				}
			}
			if t, ok := ast.Unparen(e).(*ast.TypeAssertExpr); ok && types.TypeString(u.Info.TypeOf(t.X), nil) == "error" {
				report(x.Pos(), "this uses a type switch on an error")
			}
		case *ast.SwitchStmt:
			if x.Tag != nil && isErrorType(u.Info.TypeOf(x.Tag)) {
				report(x.Pos(), "this compares errors with a switch")
			}
		case *ast.CallExpr:
			if fn := calledFunc(u.Info, x); fn != nil && textFuncs[fn.FullName()] && len(x.Args) > 0 && u.isErrorText(x.Args[0]) {
				report(x.Pos(), "this reads the text of an error")
			}
		}
		return true
	})
}

func (u *Unit) oneValue(f *ast.File, report reportFunc) {
	ok2 := map[ast.Expr]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if len(x.Lhs) == 2 && len(x.Rhs) == 1 {
				ok2[ast.Unparen(x.Rhs[0])] = true
			}
		case *ast.ValueSpec:
			if len(x.Names) == 2 && len(x.Values) == 1 {
				ok2[ast.Unparen(x.Values[0])] = true
			}
		case *ast.TypeAssertExpr:
			if x.Type != nil && !ok2[x] {
				report(x.Pos(), "this type assertion has one value, so it panics when it fails")
			}
		}
		return true
	})
}

func (u *Unit) channelSizes(f *ast.File, report reportFunc) {
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok || len(c.Args) != 2 {
			return true
		}
		id, ok := ast.Unparen(c.Fun).(*ast.Ident)
		if !ok || id.Name != "make" {
			return true
		}
		t := u.Info.TypeOf(c.Args[0])
		if t == nil {
			return true
		}
		if _, isChan := t.Underlying().(*types.Chan); !isChan {
			return true
		}
		tv := u.Info.Types[c.Args[1]]
		if tv.Value == nil || constant.Compare(tv.Value, token.GTR, constant.MakeInt64(1)) {
			report(c.Pos(), "this channel has a buffer larger than one, with no comment that defends it")
		}
		return true
	})
}

// skipped reports code that a contract adds: V's lowered checks and L's calls of the contract
// package. A measure of the code does not count it, so that both arms count the same.
func (u *Unit) skipped(n ast.Node) bool {
	isContract := func(c *ast.CallExpr) bool {
		sel, ok := ast.Unparen(c.Fun).(*ast.SelectorExpr)
		if !ok {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return false
		}
		if id.Name == "vulcan_rt" {
			return true
		}
		pn, ok := u.Info.Uses[id].(*types.PkgName)
		return ok && strings.HasSuffix(pn.Imported().Path(), "/lib/contract")
	}
	switch x := n.(type) {
	case *ast.IfStmt:
		if len(x.Body.List) == 1 {
			if s, ok := x.Body.List[0].(*ast.ExprStmt); ok {
				if c, ok := s.X.(*ast.CallExpr); ok && isContract(c) {
					if sel := c.Fun.(*ast.SelectorExpr); sel.X.(*ast.Ident).Name == "vulcan_rt" {
						return true
					}
				}
			}
		}
	case *ast.CallExpr:
		return isContract(x)
	case *ast.DeferStmt:
		return isContract(x.Call)
	}
	return false
}

func (u *Unit) complexity(r *Rule, f *ast.File, report reportFunc) {
	limit, _ := strconv.Atoi(first(r, "limit", "15"))
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil || u.generated(fd.Name.Name) {
			continue
		}
		n := 1
		ast.Inspect(fd.Body, func(x ast.Node) bool {
			if x == nil || u.skipped(x) {
				return false
			}
			switch y := x.(type) {
			case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
				n++
			case *ast.CaseClause:
				if y.List != nil {
					n++
				}
			case *ast.CommClause:
				if y.Comm != nil {
					n++
				}
			case *ast.BinaryExpr:
				if y.Op == token.LAND || y.Op == token.LOR {
					n++
				}
			}
			return true
		})
		if n > limit {
			report(fd.Name.Pos(), "function %s has a complexity of %d, above the limit of %d", fd.Name.Name, n, limit)
		}
	}
}

func (u *Unit) exhaustive(f *ast.File, report reportFunc) {
	ast.Inspect(f, func(n ast.Node) bool {
		x, ok := n.(*ast.SwitchStmt)
		if !ok || x.Tag == nil {
			return true
		}
		named, ok := types.Unalias(u.Info.TypeOf(x.Tag)).(*types.Named)
		if !ok || named.Obj().Pkg() == nil {
			return true
		}
		if _, basic := named.Underlying().(*types.Basic); !basic {
			return true
		}
		all := map[string]string{}
		scope := named.Obj().Pkg().Scope()
		for _, name := range scope.Names() {
			if c, ok := scope.Lookup(name).(*types.Const); ok && types.Identical(c.Type(), named) && (c.Exported() || c.Pkg() == u.pkgOf(x.Tag)) {
				all[stateOf(c.Val())] = name
			}
		}
		if len(all) < 2 {
			return true
		}
		var deflt *ast.CaseClause
		for _, s := range x.Body.List {
			cc := s.(*ast.CaseClause)
			if cc.List == nil {
				deflt = cc
			}
			for _, e := range cc.List {
				if tv := u.Info.Types[e]; tv.Value != nil {
					delete(all, stateOf(tv.Value))
				}
			}
		}
		if len(all) == 0 {
			return true
		}
		var missing []string
		for _, name := range all {
			missing = append(missing, name)
		}
		slices.Sort(missing)
		switch {
		case deflt == nil:
			report(x.Pos(), "this switch over %s does not cover %s, and has no default", named.Obj().Name(), strings.Join(missing, ", "))
		case !u.commentNear(deflt.Pos()) && !u.commentInside(deflt):
			report(deflt.Pos(), "this default of a switch over %s has no comment that gives a reason", named.Obj().Name())
		}
		return true
	})
}

// commentInside reports a comment on a line of a node.
func (u *Unit) commentInside(n ast.Node) bool {
	from, to := u.Fset.Position(n.Pos()), u.Fset.Position(n.End())
	return commentsOf(from.Filename).any(from.Line, to.Line)
}

// pkgOf gives the package whose code holds an expression.
func (u *Unit) pkgOf(e ast.Expr) *types.Package {
	if u.Pkg != nil {
		return u.Pkg
	}
	if id, ok := e.(*ast.Ident); ok {
		if obj := u.Info.Uses[id]; obj != nil {
			return obj.Pkg()
		}
	}
	return nil
}

func (u *Unit) testLogic(f *ast.File, report reportFunc) {
	if !u.IsTest(f) {
		return
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil || fd.Recv != nil || !strings.HasPrefix(fd.Name.Name, "Test") {
			continue
		}
		var table *ast.RangeStmt
		for _, s := range fd.Body.List {
			if rs, ok := s.(*ast.RangeStmt); ok && u.isTable(rs) {
				table = rs
				break
			}
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.ForStmt:
				report(x.Pos(), "the body of %s has a branch or a loop", fd.Name.Name)
			case *ast.RangeStmt:
				if x != table {
					report(x.Pos(), "the body of %s has a loop that is not over a table of cases", fd.Name.Name)
				}
			}
			return true
		})
	}
}

// isTable reports a range over a slice, an array, or a map of structs: a table of cases.
func (u *Unit) isTable(rs *ast.RangeStmt) bool {
	var elem types.Type
	rt := u.Info.TypeOf(rs.X)
	if rt == nil {
		return false
	}
	switch t := rt.Underlying().(type) {
	case *types.Slice:
		elem = t.Elem()
	case *types.Array:
		elem = t.Elem()
	case *types.Map:
		elem = t.Elem()
	default:
		return false
	}
	_, ok := elem.Underlying().(*types.Struct)
	return ok
}

var (
	nolintRE  = regexp.MustCompile(`^//\s*nolint(:\S+)?(\s+//\s*\S.*)?$`)
	ignoreRE  = regexp.MustCompile(`^//\s*lint:(file-)?ignore(\s+\S+)?(\s+\S.*)?$`)
	nosecRE   = regexp.MustCompile(`^(//|/\*)\s*#nosec(\s+[A-Z]+\d+)?(\s+\S.*)?$`)
	vignoreRE = regexp.MustCompile(`^//\s*vulcan:ignore(\s+\S+)?(\s+\S.*)?$`)
)

func (u *Unit) suppressions(f *ast.File, report reportFunc) {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			t := strings.TrimSpace(c.Text)
			var named, reason bool
			switch {
			case nolintRE.MatchString(t):
				m := nolintRE.FindStringSubmatch(t)
				named, reason = m[1] != "", m[2] != ""
			case ignoreRE.MatchString(t):
				m := ignoreRE.FindStringSubmatch(t)
				named, reason = m[2] != "", m[3] != ""
			case nosecRE.MatchString(t):
				m := nosecRE.FindStringSubmatch(t)
				named, reason = m[2] != "", m[3] != ""
			case vignoreRE.MatchString(t):
				m := vignoreRE.FindStringSubmatch(t)
				named, reason = m[1] != "", m[2] != ""
			case strings.HasPrefix(t, "//nolint"), strings.HasPrefix(t, "// nolint"):
				named, reason = false, false
			default:
				continue
			}
			if !named || !reason {
				report(c.Pos(), "this suppression does not name its check and give a reason")
			}
		}
	}
}

func (u *Unit) returnEffects(r *Rule, f *ast.File, report reportFunc) {
	funcs := map[string]bool{}
	for _, v := range r.Params["funcs"] {
		funcs[v] = true
	}
	isEffect := func(c *ast.CallExpr) (string, bool) {
		fn := calledFunc(u.Info, c)
		if fn == nil {
			return "", false
		}
		if matches(funcs, fn.FullName()) {
			return fn.FullName(), true
		}
		if sel, ok := ast.Unparen(c.Fun).(*ast.SelectorExpr); ok {
			if s := u.Info.Selections[sel]; s != nil {
				recv := types.TypeString(s.Recv(), nil)
				for _, p := range u.Ports {
					if p.Type != nil && types.TypeString(p.Type, nil) == recv {
						return "the port " + p.Name, true
					}
				}
			}
		}
		return "", false
	}
	scan := func(e ast.Expr) {
		c, ok := ast.Unparen(e).(*ast.CallExpr)
		if !ok {
			return
		}
		if name, ok := isEffect(c); ok {
			report(c.Pos(), "a return in a function with ensures calls %s, so the effect happens before the ensures runs", name)
		}
	}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil || u.HasEnsures == nil || u.generated(fd.Name.Name) || !u.HasEnsures(fd) {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.FuncLit:
				return false
			case *ast.ReturnStmt:
				for _, e := range x.Results {
					scan(e)
				}
			case *ast.AssignStmt:
				// V lowers a return to an assignment to its temporaries, vulcan_t0 and so on.
				if id, ok := x.Lhs[0].(*ast.Ident); ok && strings.HasPrefix(id.Name, "vulcan_t") {
					for _, e := range x.Rhs {
						scan(e)
					}
				}
			}
			return true
		})
	}
}

func (u *Unit) retries(f *ast.File, report reportFunc) {
	rnd := effects["randomness"]
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		jitter := false
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if fn, ok := u.Info.Uses[id].(*types.Func); ok && matches(rnd.funcs, fn.FullName()) {
					jitter = true
				}
			}
			return true
		})
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			var body *ast.BlockStmt
			limited := true
			switch x := n.(type) {
			case *ast.ForStmt:
				body, limited = x.Body, x.Cond != nil
			case *ast.RangeStmt:
				body = x.Body
			default:
				return true
			}
			sleeps, grows := false, false
			ast.Inspect(body, func(m ast.Node) bool {
				switch y := m.(type) {
				case *ast.FuncLit:
					return false
				case *ast.CallExpr:
					if fn := calledFunc(u.Info, y); fn != nil && fn.FullName() == "time.Sleep" {
						sleeps = true
					}
				case *ast.BinaryExpr:
					grows = grows || y.Op == token.MUL || y.Op == token.SHL
				case *ast.AssignStmt:
					grows = grows || y.Tok == token.MUL_ASSIGN || y.Tok == token.SHL_ASSIGN
				}
				return true
			})
			if !sleeps {
				return true
			}
			var lacks []string
			if !limited {
				lacks = append(lacks, "a limit on attempts")
			}
			if !grows {
				lacks = append(lacks, "exponential backoff")
			}
			if !jitter {
				lacks = append(lacks, "jitter")
			}
			if len(lacks) > 0 {
				report(n.Pos(), "this loop sleeps and has no %s", strings.Join(lacks, ", "))
			}
			return true
		})
	}
}

// tokens gives the form of a function: the kinds of its nodes, with names and values left out.
func (u *Unit) tokens(fd *ast.FuncDecl) []string {
	var out []string
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if n == nil || u.skipped(n) {
			return false
		}
		switch x := n.(type) {
		case *ast.SelectorExpr:
			// A chain of names counts as a selector of one name, so that L's p.ledger.Save and V's
			// Ledger.Save count the same.
			if isNameChain(x) {
				out = append(out, "*ast.SelectorExpr", "*ast.Ident", "*ast.Ident")
				return false
			}
			out = append(out, fmt.Sprintf("%T", n))
		case *ast.BinaryExpr:
			out = append(out, x.Op.String())
		case *ast.AssignStmt:
			out = append(out, x.Tok.String())
		default:
			out = append(out, fmt.Sprintf("%T", n))
		}
		return true
	})
	return out
}

func isNameChain(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return isNameChain(x.X)
	}
	return false
}

func trigrams(toks []string) map[string]bool {
	out := map[string]bool{}
	for i := 0; i+2 < len(toks); i++ {
		out[toks[i]+" "+toks[i+1]+" "+toks[i+2]] = true
	}
	return out
}

func (u *Unit) duplicates(r *Rule, add addFunc) {
	limit, _ := strconv.Atoi(first(r, "limit", "60"))
	similarity, _ := strconv.Atoi(first(r, "similarity", "90"))
	type fn struct {
		name string
		decl *ast.FuncDecl
		set  map[string]bool
	}
	var all []fn
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil || u.generated(fd.Name.Name) {
				continue
			}
			if toks := u.tokens(fd); len(toks) >= limit {
				all = append(all, fn{recvName(fd) + "." + fd.Name.Name, fd, trigrams(toks)})
			}
		}
	}
	// By name, not by place: the arms have other files, and a pair must read the same way in both.
	slices.SortFunc(all, func(a, b fn) int { return strings.Compare(a.name, b.name) })
	for i, a := range all {
		for _, b := range all[i+1:] {
			both := 0
			for t := range a.set {
				if b.set[t] {
					both++
				}
			}
			union := len(a.set) + len(b.set) - both
			if union > 0 && both*100 >= similarity*union {
				add("0810", b.decl.Name.Pos(), "function %s is a near-duplicate of %s: %s", b.decl.Name.Name, a.decl.Name.Name, r.tail())
			}
		}
	}
}

func recvName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) != 1 {
		return ""
	}
	return types.ExprString(fd.Recv.List[0].Type)
}

// unused reports an unexported name at package level that nothing in the package uses. It cannot
// read a package with tests that the load left out, because only a test may use a name there.
func (u *Unit) unused(r *Rule, add addFunc) {
	if u.TestsAbsent {
		return
	}
	used := map[types.Object]bool{}
	for _, obj := range u.Info.Uses {
		used[obj] = true
	}
	for _, f := range u.Files {
		if !u.fileIn(r, f) {
			continue
		}
		for _, d := range f.Decls {
			var ids []*ast.Ident
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					ids = append(ids, d.Name)
				}
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					switch sp := sp.(type) {
					case *ast.TypeSpec:
						ids = append(ids, sp.Name)
					case *ast.ValueSpec:
						ids = append(ids, sp.Names...)
					}
				}
			}
			for _, id := range ids {
				if id.IsExported() || id.Name == "_" || id.Name == "init" || id.Name == "main" || id.Name == "setup" || id.Name == "startup" || u.generated(id.Name) {
					continue
				}
				if obj := u.Info.Defs[id]; obj != nil && !used[obj] {
					add("0810", id.Pos(), "nothing in the package uses %s: %s", id.Name, r.tail())
				}
			}
		}
	}
}

var (
	parsed     sync.Map // file name -> map of function name to true when it is deprecated
	goRootOnce = sync.OnceValue(func() string {
		out, err := exec.Command("go", "env", "GOROOT").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	})
)

// deprecatedIn lists the deprecated functions and methods of a Go source file.
func deprecatedIn(file string) map[string]bool {
	if v, ok := parsed.Load(file); ok {
		return v.(map[string]bool)
	}
	out := map[string]bool{}
	name := file
	if rest, ok := strings.CutPrefix(file, "$GOROOT"); ok {
		name = goRootOnce() + rest
	}
	f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err == nil {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Doc == nil {
				continue
			}
			for _, c := range fd.Doc.List {
				if strings.HasPrefix(c.Text, "// Deprecated:") {
					out[fd.Name.Name] = true
				}
			}
		}
	}
	parsed.Store(file, out)
	return out
}

func (u *Unit) deprecated(f *ast.File, report reportFunc) {
	ast.Inspect(f, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		fn, ok := u.Info.Uses[id].(*types.Func)
		if !ok || fn.Pkg() == nil || !fn.Pos().IsValid() {
			return true
		}
		file := u.Fset.Position(fn.Pos()).Filename
		if u.SourceFile != nil {
			file = u.SourceFile(file)
		}
		if file != "" && strings.HasSuffix(file, ".go") && deprecatedIn(file)[fn.Name()] {
			report(id.Pos(), "this uses %s, which is deprecated", fn.FullName())
		}
		return true
	})
}
