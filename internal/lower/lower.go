package lower

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/blank"
	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/lex"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// RT is the import path of the runtime package that lowered code calls.
const RT = "github.com/robot-accomplice/vulcan/vulcan/rt"

// Input is what the lowering of one package needs.
type Input struct {
	P *project.Project
	// LineFile gives the file name of //line directives for a source file.
	LineFile func(rel string) string
	// Inv lists the struct types with an invariant, by import path.
	Inv map[string]map[string]bool
}

// Typed is the type-checked code of a package before lowering: each code file's syntax, with
// positions in the file set; Base is the length of the //line header of each generated file.
type Typed struct {
	Fset  *token.FileSet
	Info  *types.Info
	Files map[*project.CodeFile]*ast.File
	Base  map[*project.CodeFile]int
}

// Invariants collects the struct types with an invariant in every package.
func Invariants(p *project.Project) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, pkg := range append(append([]*project.Package{}, p.Leaves...), p.Libs...) {
		for _, f := range pkg.Files {
			for _, t := range f.Blank.Types {
				if out[pkg.ImportPath] == nil {
					out[pkg.ImportPath] = map[string]bool{}
				}
				out[pkg.ImportPath][t.Name] = true
			}
		}
	}
	return out
}

type invType struct {
	name    string
	file    *project.CodeFile
	clauses []blank.Clause
	fields  map[string]bool
	generic bool
	methods map[string]bool // pointer-receiver methods with a named receiver: they get a twin
}

// edit replaces src[off:end] with text. At one offset, edits apply in order of prio, then of
// creation: an outer opening (-1) before an inner one (0), an inner closing (0) before an outer
// one (1). twin, when set, is the text in a method's unchecked twin.
type edit struct {
	off, end int
	text     string
	twin     *string
	prio     int
}

type twin struct {
	fd *ast.FuncDecl
}

type file struct {
	in      *Input
	pkg     *project.Package
	cf      *project.CodeFile
	af      *ast.File
	fset    *token.FileSet
	info    *types.Info // nil when the package did not type-check
	base    int
	src     []byte
	lines   *lex.Lines
	lf      string
	edits   []edit
	usesRT  bool
	extra   []string
	twins   []twin
	invs    map[string]*invType
	oldNext *int
}

// Package lowers the contracts of one package. typed may be nil, or miss a file: such a file is
// parsed again and lowered without type information. It returns the Go text of each code file.
func Package(in *Input, pkg *project.Package, typed *Typed) map[*project.CodeFile][]byte {
	out := map[*project.CodeFile][]byte{}
	type parsed struct {
		cf   *project.CodeFile
		af   *ast.File
		fset *token.FileSet
		info *types.Info
		base int
	}
	var files []parsed
	invs := map[string]*invType{}
	for _, cf := range pkg.Files {
		pf := parsed{cf: cf}
		if typed != nil && typed.Files[cf] != nil {
			pf.af, pf.fset, pf.info, pf.base = typed.Files[cf], typed.Fset, typed.Info, typed.Base[cf]
		} else {
			fset := token.NewFileSet()
			af, err := parser.ParseFile(fset, cf.Rel, cf.Blank.Go, parser.SkipObjectResolution)
			if err != nil || af == nil {
				out[cf] = cf.Blank.Go // the go command reports the syntax error
				continue
			}
			pf.af, pf.fset = af, fset
		}
		reservedNames(in.P, cf, pf.af, pf.fset, pf.base)
		files = append(files, pf)
		for _, t := range cf.Blank.Types {
			it := &invType{name: t.Name, file: cf, clauses: t.Clauses, fields: map[string]bool{}, methods: map[string]bool{}}
			for _, d := range pf.af.Decls {
				gd, ok := d.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, sp := range gd.Specs {
					ts := sp.(*ast.TypeSpec)
					if ts.Name.Name != t.Name {
						continue
					}
					it.generic = ts.TypeParams != nil
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						in.P.Diags.Errorf(cf.Pos(t.Clauses[0].WordOff), "K0513", "an invariant is only on a struct type; %s is not a struct", t.Name)
						it.generic = true // skip it
						continue
					}
					for _, fl := range st.Fields.List {
						for _, n := range fl.Names {
							it.fields[n.Name] = true
						}
						if len(fl.Names) == 0 {
							it.fields[embeddedName(fl.Type)] = true
						}
					}
				}
			}
			if it.generic && len(it.fields) > 0 {
				in.P.Deferred(cf.Pos(t.Clauses[0].WordOff), "an invariant on a generic struct", "§7.2")
			}
			invs[t.Name] = it
		}
	}
	for _, f := range files {
		for _, d := range f.af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			rt, ptr := recvType(fd.Recv.List[0].Type)
			if it := invs[rt]; it != nil && ptr && len(fd.Recv.List[0].Names) == 1 && fd.Recv.List[0].Names[0].Name != "_" && fd.Body != nil {
				it.methods[fd.Name.Name] = true
			}
		}
	}
	oldNext := 0
	for _, f := range files {
		lw := &file{in: in, pkg: pkg, cf: f.cf, af: f.af, fset: f.fset, info: f.info, base: f.base, src: f.cf.Blank.Go,
			lines: lex.NewLines(f.cf.Blank.Go), lf: in.LineFile(f.cf.Rel), invs: invs, oldNext: &oldNext}
		out[f.cf] = lw.lower()
	}
	return out
}

func embeddedName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return embeddedName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.IndexExpr:
		return embeddedName(t.X)
	case *ast.IndexListExpr:
		return embeddedName(t.X)
	}
	return ""
}

// recvType returns the receiver's type name, and whether it is a pointer.
func recvType(e ast.Expr) (string, bool) {
	ptr := false
	if s, ok := e.(*ast.StarExpr); ok {
		ptr, e = true, s.X
	}
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name, ptr
	case *ast.IndexExpr:
		n, _ := recvType(t.X)
		return n, ptr
	case *ast.IndexListExpr:
		n, _ := recvType(t.X)
		return n, ptr
	}
	return "", ptr
}

// reservedNames applies Appendix B.1: a declared name cannot start with vulcan_ (K0005). It runs
// on the code as written, so generated names do not count.
func reservedNames(p *project.Project, cf *project.CodeFile, af *ast.File, fset *token.FileSet, base int) {
	check := func(id *ast.Ident) {
		if id != nil && (strings.HasPrefix(id.Name, "vulcan_") || strings.HasPrefix(id.Name, "Vulcan_")) {
			p.Diags.Errorf(cf.Pos(fset.PositionFor(id.Pos(), false).Offset-base), "K0005", "the name %s starts with %s, which is reserved for generated code", id.Name, id.Name[:7])
		}
	}
	fields := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			for _, n := range f.Names {
				check(n)
			}
		}
	}
	ast.Inspect(af, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			check(x.Name)
		case *ast.FuncType:
			fields(x.TypeParams)
			fields(x.Params)
			fields(x.Results)
		case *ast.TypeSpec:
			check(x.Name)
			fields(x.TypeParams)
		case *ast.StructType:
			fields(x.Fields)
		case *ast.ValueSpec:
			for _, id := range x.Names {
				check(id)
			}
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE {
				for _, l := range x.Lhs {
					if id, ok := l.(*ast.Ident); ok {
						check(id)
					}
				}
			}
		case *ast.RangeStmt:
			if x.Tok == token.DEFINE {
				for _, l := range []ast.Expr{x.Key, x.Value} {
					if id, ok := l.(*ast.Ident); ok {
						check(id)
					}
				}
			}
		case *ast.LabeledStmt:
			check(x.Label)
		case *ast.ImportSpec:
			check(x.Name)
		}
		return true
	})
}

func (lw *file) off(p token.Pos) int { return lw.fset.PositionFor(p, false).Offset - lw.base }

// dir is a directive that gives the next character the position of offset off of this file.
func (lw *file) dir(off int) string {
	l, c := lw.lines.Pos(off)
	return fmt.Sprintf("/*line %s:%d:%d*/", lw.lf, l, c)
}

// clauseDir is a directive to a clause's expression in the source file.
func (lw *file) clauseDir(c blank.Clause) string {
	p := lw.cf.Pos(c.Off)
	return fmt.Sprintf("/*line %s:%d:%d*/", lw.lf, p.Line, p.Col)
}

func (lw *file) add(off, end int, text string, prio int) int {
	lw.edits = append(lw.edits, edit{off: off, end: end, text: text, prio: prio})
	return len(lw.edits) - 1
}

func (lw *file) lower() []byte {
	byKey := map[string]*blank.Func{}
	for _, f := range lw.cf.Blank.Funcs {
		byKey[f.Recv+"."+f.Name] = f
	}
	for _, d := range lw.af.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		recv, ptr := "", false
		if fd.Recv != nil && len(fd.Recv.List) == 1 {
			recv, ptr = recvType(fd.Recv.List[0].Type)
		}
		bf := byKey[recv+"."+fd.Name.Name]
		var it *invType
		if inv := lw.invs[recv]; inv != nil && ptr && inv.methods[fd.Name.Name] && !inv.generic {
			it = inv
		}
		if bf == nil && it == nil {
			continue
		}
		var clauses []blank.Clause
		if bf != nil {
			clauses = bf.Clauses
			if !bf.HasBody || fd.Body == nil {
				lw.in.P.Diags.Errorf(lw.cf.Pos(clauses[0].WordOff), "K0514", "function %s has a contract but no body", fd.Name.Name)
				continue
			}
		}
		lw.function(fd, clauses, it)
	}
	lw.literals()
	for _, it := range lw.invs {
		if it.file == lw.cf && !it.generic {
			lw.invariantMethods(it)
		}
	}
	if len(lw.edits) == 0 && len(lw.extra) == 0 && len(lw.twins) == 0 {
		return lw.src // nothing to lower: the file stays as it is
	}
	if lw.usesRT {
		e := lw.off(lw.af.Name.End())
		lw.add(e, e, "; import vulcan_rt "+strconv.Quote(RT), -1)
	}
	var b strings.Builder
	b.WriteString(apply(lw.src, lw.edits, 0, len(lw.src), false))
	for _, tw := range lw.twins {
		edits := append([]edit{}, lw.edits...)
		name := tw.fd.Name
		edits = append(edits, edit{off: lw.off(name.Pos()), end: lw.off(name.End()),
			text: lw.dir(lw.off(name.Pos())) + "vulcan_" + name.Name + lw.dir(lw.off(name.End()))})
		start := lw.off(tw.fd.Pos())
		l, c := lw.lines.Pos(start)
		fmt.Fprintf(&b, "\n\n//line %s:%d:1\n%s%s", lw.lf, l, strings.Repeat(" ", c-1), apply(lw.src, edits, start, lw.off(tw.fd.End()), true))
	}
	for _, x := range lw.extra {
		b.WriteString("\n\n")
		b.WriteString(x)
	}
	b.WriteString("\n")
	return []byte(b.String())
}

// apply applies the edits inside [from, to) to src[from:to]; for a twin, it uses each edit's
// twin text where one is set.
func apply(src []byte, edits []edit, from, to int, forTwin bool) string {
	var in []edit
	for _, e := range edits {
		if e.off >= from && e.end <= to {
			in = append(in, e)
		}
	}
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].off != in[j].off {
			return in[i].off < in[j].off
		}
		return in[i].prio < in[j].prio
	})
	var b strings.Builder
	at := from
	for _, e := range in {
		if e.off < at {
			continue // overlapping: keep the first
		}
		b.Write(src[at:e.off])
		if forTwin && e.twin != nil {
			b.WriteString(*e.twin)
		} else {
			b.WriteString(e.text)
		}
		at = e.end
	}
	b.Write(src[at:to])
	return b.String()
}

type result struct {
	name string // the name in the source, or ""
	typ  string
}

// contract is one parsed clause, with old() replaced by entry variables.
type contract struct {
	c     blank.Clause
	e     ast.Expr
	ensur bool
}

// function lowers one function or method (Appendix F): requires and old() at entry, ensures and
// the invariant at each return, before the deferred calls run, and at a reachable end of a body
// that has no results.
func (lw *file) function(fd *ast.FuncDecl, clauses []blank.Clause, it *invType) {
	p := lw.in.P
	recvName, recvTypeName := "", ""
	params := map[string]bool{}
	if fd.Recv != nil && len(fd.Recv.List) == 1 {
		recvTypeName, _ = recvType(fd.Recv.List[0].Type)
		if len(fd.Recv.List[0].Names) == 1 {
			recvName = fd.Recv.List[0].Names[0].Name
			params[recvName] = true
		}
	}
	caller := lw.pkg.GoName + "." + fd.Name.Name
	if recvTypeName != "" {
		caller = recvTypeName + "." + fd.Name.Name
	}
	var firstClause int
	if len(clauses) > 0 {
		firstClause = clauses[0].WordOff
	}
	reserved := func(id *ast.Ident) {
		if len(clauses) > 0 && (id.Name == "err" || id.Name == "result") {
			p.Diags.Errorf(lw.cf.Pos(firstClause), "K0511", "%s has a contract, so it cannot name a parameter or a result %s; rename it", fd.Name.Name, id.Name)
		}
	}
	for _, f := range fd.Type.Params.List {
		for _, n := range f.Names {
			if n.Name != "_" {
				params[n.Name] = true
			}
			reserved(n)
		}
	}
	var results []result
	if fd.Type.Results != nil {
		for _, f := range fd.Type.Results.List {
			typ := string(lw.src[lw.off(f.Type.Pos()):lw.off(f.Type.End())])
			if len(f.Names) == 0 {
				results = append(results, result{typ: typ})
			}
			for _, n := range f.Names {
				reserved(n)
				results = append(results, result{name: n.Name, typ: typ})
			}
		}
	}
	errIdx := -1
	if n := len(results); n > 0 && results[n-1].typ == "error" {
		errIdx = n - 1
	}
	var nonErr []int
	for i := range results {
		if i != errIdx {
			nonErr = append(nonErr, i)
		}
	}
	named := map[string]int{}
	for i, r := range results {
		if r.name != "" && r.name != "_" {
			named[r.name] = i
		}
	}
	// Parse each clause, and replace old(x) with a variable set at entry. Parameters that an
	// ensures reads are saved at entry, so that a local name in the body cannot hide them.
	var open strings.Builder
	var contracts []contract
	saved := map[string]bool{}
	for _, c := range clauses {
		e, err := ParseContract(c.Text)
		if err != nil {
			p.Diags.Errorf(lw.cf.Pos(c.Off), "K0001", "the contract does not parse: %v", err)
			continue
		}
		ensures := c.Word == "ensures"
		lw.checkNames(e, c, ensures, errIdx, len(nonErr))
		if ensures {
			e = Rewrite(e, func(n ast.Expr) ast.Expr {
				if v, ok := n.(*ast.CallExpr); ok {
					if id, ok := v.Fun.(*ast.Ident); ok && id.Name == "old" && len(v.Args) == 1 {
						name := fmt.Sprintf("vulcan_old%d", *lw.oldNext)
						*lw.oldNext++
						fmt.Fprintf(&open, "%s := %s%s; _ = %s; ", name, lw.clauseDir(c), GoText(v.Args[0]), name)
						if lw.pkg.Olds == nil {
							lw.pkg.Olds = map[string]diag.Pos{}
						}
						lw.pkg.Olds[name] = lw.cf.Pos(c.Off)
						return &ast.CallExpr{Fun: ast.NewIdent("vulcan_oldvalue"), Args: []ast.Expr{ast.NewIdent(name), v.Args[0]}}
					}
				}
				return nil
			})
			Idents(e, func(id *ast.Ident) {
				if params[id.Name] {
					saved[id.Name] = true
				}
			})
		}
		contracts = append(contracts, contract{c: c, e: e, ensur: ensures})
	}
	// check writes one clause's check, with resVar naming each result at this point.
	check := func(ct contract, resVar func(int) string) string {
		var shows []string
		seen := map[string]bool{}
		show := func(label, expr string) {
			if !seen[label] {
				seen[label] = true
				shows = append(shows, fmt.Sprintf("vulcan_rt.Show(%q, %s)", label, expr))
			}
		}
		x := Rewrite(ct.e, func(n ast.Expr) ast.Expr {
			switch v := n.(type) {
			case *ast.CallExpr:
				if id, ok := v.Fun.(*ast.Ident); ok && id.Name == "vulcan_oldvalue" {
					name := v.Args[0].(*ast.Ident).Name
					show("old("+GoText(v.Args[1])+")", name)
					return ast.NewIdent(name)
				}
			case *ast.Ident:
				switch {
				case params[v.Name]:
					if ct.ensur {
						show(v.Name, "vulcan_p_"+v.Name)
						return ast.NewIdent("vulcan_p_" + v.Name)
					}
					show(v.Name, v.Name)
				case !ct.ensur:
				case v.Name == "err" && errIdx >= 0:
					show("err", resVar(errIdx))
					return ast.NewIdent(resVar(errIdx))
				case v.Name == "result" && len(nonErr) > 0:
					show("result", resVar(nonErr[0]))
					return ast.NewIdent(resVar(nonErr[0]))
				case strings.HasPrefix(v.Name, "vulcan_result_"):
					k, _ := strconv.Atoi(strings.TrimPrefix(v.Name, "vulcan_result_"))
					if k < len(nonErr) {
						show("result."+strconv.Itoa(k), resVar(nonErr[k]))
						return ast.NewIdent(resVar(nonErr[k]))
					}
				default:
					if i, ok := named[v.Name]; ok {
						show(v.Name, resVar(i))
						return ast.NewIdent(resVar(i))
					}
				}
			}
			return nil
		})
		args := fmt.Sprintf("%q, %q, %q, %q", ct.c.Word, ct.c.Text, lw.cf.Pos(ct.c.Off).String(), caller)
		if len(shows) > 0 {
			args += ", " + strings.Join(shows, ", ")
		}
		lw.usesRT = true
		return fmt.Sprintf("if !(%s%s) { vulcan_rt.Fail(%s) }; ", lw.clauseDir(ct.c), GoText(x), args)
	}
	// Entry: requires, saved parameters, old().
	var entry strings.Builder
	for _, ct := range contracts {
		if !ct.ensur {
			entry.WriteString(check(ct, nil))
		}
	}
	var savedNames []string
	for n := range saved {
		savedNames = append(savedNames, n)
	}
	sort.Strings(savedNames)
	for _, n := range savedNames {
		fmt.Fprintf(&entry, "vulcan_p_%s := %s; _ = vulcan_p_%s; ", n, n, n)
	}
	entry.WriteString(open.String())
	lbrace := lw.off(fd.Body.Lbrace)
	if entry.Len() > 0 {
		lw.add(lbrace+1, lbrace+1, " "+entry.String()+lw.dir(lbrace+1), -1)
	}
	inv := ""
	if it != nil {
		inv = fmt.Sprintf("%s.Vulcan_invariant(%q); ", recvName, caller)
		lw.receiverCalls(fd, recvName, it)
		lw.twins = append(lw.twins, twin{fd: fd})
	}
	ensures := func(resVar func(int) string) string {
		var b strings.Builder
		for _, ct := range contracts {
			if ct.ensur {
				b.WriteString(check(ct, resVar))
			}
		}
		return b.String()
	}
	addPair := func(off, end int, with, without string, prio int) {
		i := lw.add(off, end, with, prio)
		if with != without {
			w := without
			lw.edits[i].twin = &w
		}
	}
	// Each return, outside the function literals of the body.
	var hasEnsures bool
	for _, ct := range contracts {
		hasEnsures = hasEnsures || ct.ensur
	}
	if !hasEnsures && it == nil {
		return
	}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			kw := lw.off(x.Return)
			end := lw.off(x.End())
			if len(x.Results) == 0 {
				// A bare return: the results are the named results, by name.
				byName := func(i int) string { return results[i].name }
				checks := ensures(byName)
				addPair(kw, kw+len("return"), "{ "+checks+inv+"return }"+lw.dir(kw+len("return")), "{ "+checks+"return }"+lw.dir(kw+len("return")), -1)
				return false
			}
			var tvars, decls []string
			for i, r := range results {
				t := fmt.Sprintf("vulcan_t%d", i)
				tvars = append(tvars, t)
				decls = append(decls, fmt.Sprintf("var %s %s; ", t, r.typ))
			}
			byTemp := func(i int) string { return tvars[i] }
			checks := ensures(byTemp)
			openText := "{ " + strings.Join(decls, "") + strings.Join(tvars, ", ") + " = " + lw.dir(kw+len("return"))
			lw.add(kw, kw+len("return"), openText, -1)
			ret := "return " + strings.Join(tvars, ", ") + " }"
			addPair(end, end, "; "+checks+inv+ret+lw.dir(end), "; "+checks+ret+lw.dir(end), 1)
			return true
		}
		return true
	})
	if len(results) == 0 {
		// The end of the body can be reached.
		rbrace := lw.off(fd.Body.Rbrace)
		checks := ensures(func(int) string { return "" })
		if checks != "" || inv != "" {
			addPair(rbrace, rbrace, "; "+checks+inv+lw.dir(rbrace), "; "+checks+lw.dir(rbrace), 1)
		}
	}
}

// receiverCalls makes a call through the method's own receiver variable call the twin without
// the invariant check (§7.2, "outer calls only"), unless the method assigns the receiver. The
// receiver is found by its object, so a local variable of the same name does not count.
func (lw *file) receiverCalls(fd *ast.FuncDecl, recv string, it *invType) {
	if lw.info == nil {
		return // without types, every call is checked
	}
	recvObj := lw.info.Defs[fd.Recv.List[0].Names[0]]
	if recvObj == nil {
		return
	}
	assigned := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		var lhs []ast.Expr
		switch a := n.(type) {
		case *ast.AssignStmt:
			lhs = a.Lhs
		case *ast.RangeStmt:
			lhs = []ast.Expr{a.Key, a.Value}
		case *ast.UnaryExpr:
			if a.Op == token.AND {
				lhs = []ast.Expr{a.X}
			}
		}
		for _, l := range lhs {
			if id, ok := l.(*ast.Ident); ok && lw.info.Uses[id] == recvObj {
				assigned = true
			}
		}
		return true
	})
	if assigned {
		return
	}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false // a function literal counts as outside the method (§7.2)
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && lw.info.Uses[id] == recvObj && it.methods[sel.Sel.Name] {
			s, e := lw.off(sel.Sel.Pos()), lw.off(sel.Sel.End())
			lw.add(s, e, lw.dir(s)+"vulcan_"+sel.Sel.Name+lw.dir(e), 0)
		}
		return true
	})
}

// checkNames applies the rules on err and result (K0502, K0510) and on type assertions (K0512).
func (lw *file) checkNames(e ast.Expr, c blank.Clause, ensures bool, errIdx, nonErr int) {
	p := lw.in.P
	pos := lw.cf.Pos(c.Off)
	ast.Inspect(e, func(n ast.Node) bool {
		if _, ok := n.(*ast.TypeAssertExpr); ok {
			p.Diags.Errorf(pos, "K0512", "a type assertion is not allowed in a contract")
		}
		return true
	})
	var walk func(e ast.Expr, errNil bool)
	walk = func(e ast.Expr, errNil bool) {
		if a, b, ok := ImpliesArgs(e); ok {
			walk(a, errNil)
			walk(b, errNil || hasErrNil(a))
			return
		}
		switch x := e.(type) {
		case *ast.BinaryExpr:
			if x.Op == token.LAND {
				walk(x.X, errNil)
				walk(x.Y, errNil || hasErrNil(x.X))
				return
			}
		case *ast.SelectorExpr:
			walk(x.X, errNil)
			return
		case *ast.KeyValueExpr:
			walk(x.Value, errNil)
			return
		case *ast.Ident:
			lw.checkIdent(x, pos, ensures, errIdx, nonErr, errNil)
			return
		}
		// Every other expression: walk its direct sub-expressions.
		ast.Inspect(e, func(n ast.Node) bool {
			if n == nil || n == ast.Node(e) {
				return true
			}
			if x, ok := n.(ast.Expr); ok {
				walk(x, errNil)
				return false
			}
			return true
		})
	}
	walk(e, false)
}

func (lw *file) checkIdent(x *ast.Ident, pos diag.Pos, ensures bool, errIdx, nonErr int, errNil bool) {
	p := lw.in.P
	isResult := x.Name == "result" || strings.HasPrefix(x.Name, "vulcan_result_")
	switch {
	case (isResult || x.Name == "err") && !ensures:
		p.Diags.Errorf(pos, "K0510", "%s names a result, which requires cannot see; requires is checked at entry", displayName(x.Name))
	case isResult && nonErr == 0:
		p.Diags.Errorf(pos, "K0510", "%s names a result, but the function has no result other than error", displayName(x.Name))
	case x.Name == "result" && nonErr > 1:
		p.Diags.Errorf(pos, "K0510", "the function has %d results other than error; write result.0 to result.%d, not result", nonErr, nonErr-1)
	case strings.HasPrefix(x.Name, "vulcan_result_"):
		k, _ := strconv.Atoi(strings.TrimPrefix(x.Name, "vulcan_result_"))
		if k >= nonErr {
			p.Diags.Errorf(pos, "K0510", "%s is out of range: the function has %d results other than error", displayName(x.Name), nonErr)
		} else if errIdx >= 0 && !errNil {
			p.Diags.Errorf(pos, "K0502", "%s is used without an err == nil guard; write err == nil => ...", displayName(x.Name))
		}
	case isResult && errIdx >= 0 && !errNil:
		p.Diags.Errorf(pos, "K0502", "result is used without an err == nil guard; write err == nil => ...")
	case x.Name == "err" && errIdx < 0 && ensures:
		p.Diags.Errorf(pos, "K0510", "err names the last result when its type is error; this function has no error result")
	}
}

func displayName(n string) string {
	if strings.HasPrefix(n, "vulcan_result_") {
		return "result." + strings.TrimPrefix(n, "vulcan_result_")
	}
	return n
}

// hasErrNil reports whether e, as a conjunction, states err == nil.
func hasErrNil(e ast.Expr) bool {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return hasErrNil(x.X)
	case *ast.BinaryExpr:
		if x.Op == token.LAND {
			return hasErrNil(x.X) || hasErrNil(x.Y)
		}
		if x.Op == token.EQL {
			a, aok := x.X.(*ast.Ident)
			b, bok := x.Y.(*ast.Ident)
			return aok && bok && (a.Name == "err" && b.Name == "nil" || a.Name == "nil" && b.Name == "err")
		}
	}
	return false
}

// invariantOf returns the qualifier and the name of the invariant type t, or ok false.
func (lw *file) invariantOf(t types.Type) (qual, name string, ok bool) {
	n, isNamed := types.Unalias(t).(*types.Named)
	if !isNamed || n.Obj().Pkg() == nil || n.TypeArgs() != nil {
		return "", "", false
	}
	path := n.Obj().Pkg().Path()
	if !lw.in.Inv[path][n.Obj().Name()] {
		return "", "", false
	}
	if path == lw.pkg.ImportPath {
		return "", n.Obj().Name(), true
	}
	for _, imp := range lw.af.Imports {
		if ip, _ := strconv.Unquote(imp.Path.Value); ip == path {
			q := n.Obj().Pkg().Name()
			if imp.Name != nil {
				q = imp.Name.Name
			}
			if q == "_" || q == "." {
				return "", "", false
			}
			return q + ".", n.Obj().Name(), true
		}
	}
	return "", "", false // the file does not import the package: nothing to name it with
}

// literals checks each value of a type with an invariant where the code builds it: a struct
// literal, also with its type elided, and a conversion (§7.2). It needs the types.
func (lw *file) literals() {
	if lw.info == nil {
		return
	}
	done := map[ast.Node]bool{}
	wrap := func(n ast.Node, before, after string) {
		s, e := lw.off(n.Pos()), lw.off(n.End())
		lw.add(s, s, before+lw.dir(s), 0)
		lw.add(e, e, after+lw.dir(e), 0)
	}
	at := func(n ast.Node) string { return strconv.Quote(lw.cf.Pos(lw.off(n.Pos())).String()) }
	ast.Inspect(lw.af, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.UnaryExpr:
			if lit, ok := x.X.(*ast.CompositeLit); ok && x.Op == token.AND && lit.Type != nil {
				if q, name, ok := lw.invariantOf(lw.info.TypeOf(lit)); ok {
					done[lit] = true
					wrap(x, q+"Vulcan_checkPtr"+name+"(", ", "+at(x)+")")
				}
			}
		case *ast.CompositeLit:
			if done[x] {
				return true
			}
			t := lw.info.TypeOf(x)
			if t == nil {
				return true
			}
			if p, isPtr := t.(*types.Pointer); isPtr && x.Type == nil {
				// An elided &T{...} in a slice or map literal of *T.
				if q, name, ok := lw.invariantOf(p.Elem()); ok {
					wrap(x, q+"Vulcan_checkPtr"+name+"(&"+q+name, ", "+at(x)+")")
				}
				return true
			}
			if q, name, ok := lw.invariantOf(t); ok {
				if x.Type == nil {
					wrap(x, q+"Vulcan_check"+name+"("+q+name, ", "+at(x)+")")
				} else {
					wrap(x, q+"Vulcan_check"+name+"(", ", "+at(x)+")")
				}
			}
		case *ast.CallExpr:
			if tv, ok := lw.info.Types[x.Fun]; ok && tv.IsType() && len(x.Args) == 1 {
				if q, name, ok := lw.invariantOf(tv.Type); ok {
					wrap(x, q+"Vulcan_check"+name+"(", ", "+at(x)+")")
				}
			}
		}
		return true
	})
}

// invariantMethods writes Vulcan_invariant, Vulcan_checkT, and Vulcan_checkPtrT (Appendix F).
// The argument of each names the call or the position that the check follows.
func (lw *file) invariantMethods(it *invType) {
	lw.usesRT = true
	var b strings.Builder
	first := it.clauses[0]
	fp := lw.cf.Pos(first.WordOff)
	fmt.Fprintf(&b, "//line %s:%d:1\nfunc (vulcan_v *%s) Vulcan_invariant(vulcan_at string) { if vulcan_v == nil { return }; ", lw.lf, fp.Line, it.name)
	for _, c := range it.clauses {
		e, err := ParseContract(c.Text)
		if err != nil {
			lw.in.P.Diags.Errorf(lw.cf.Pos(c.Off), "K0001", "the invariant does not parse: %v", err)
			continue
		}
		var shows []string
		seen := map[string]bool{}
		x := Rewrite(e, func(n ast.Expr) ast.Expr {
			if id, ok := n.(*ast.Ident); ok && it.fields[id.Name] {
				if !seen[id.Name] {
					seen[id.Name] = true
					shows = append(shows, fmt.Sprintf("vulcan_rt.Show(%q, vulcan_v.%s)", id.Name, id.Name))
				}
				return &ast.SelectorExpr{X: ast.NewIdent("vulcan_v"), Sel: ast.NewIdent(id.Name)}
			}
			return nil
		})
		args := fmt.Sprintf("%q, %q, %q, vulcan_at", "invariant", c.Text, lw.cf.Pos(c.Off).String())
		if len(shows) > 0 {
			args += ", " + strings.Join(shows, ", ")
		}
		fmt.Fprintf(&b, "if !(%s%s) { vulcan_rt.Fail(%s) }; ", lw.clauseDir(c), GoText(x), args)
	}
	b.WriteString("}\n")
	fmt.Fprintf(&b, "func Vulcan_check%s(vulcan_x %s, vulcan_at string) %s { (&vulcan_x).Vulcan_invariant(\"the value built at \" + vulcan_at); return vulcan_x }\n", it.name, it.name, it.name)
	fmt.Fprintf(&b, "func Vulcan_checkPtr%s(vulcan_p *%s, vulcan_at string) *%s { vulcan_p.Vulcan_invariant(\"the value built at \" + vulcan_at); return vulcan_p }", it.name, it.name, it.name)
	lw.extra = append(lw.extra, b.String())
}

// Needed reports whether a project has a contract or an invariant, so that the lowering needs
// the types of a first pass.
func Needed(p *project.Project) bool {
	for _, pkg := range append(append([]*project.Package{}, p.Leaves...), p.Libs...) {
		for _, f := range pkg.Files {
			if len(f.Blank.Funcs) > 0 || len(f.Blank.Types) > 0 {
				return true
			}
		}
	}
	return false
}
