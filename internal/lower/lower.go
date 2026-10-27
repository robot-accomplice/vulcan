package lower

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
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
	// Inv lists the struct types with an invariant, by import path, for literal checks in
	// other packages.
	Inv map[string]map[string]bool
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

type edit struct {
	off, end int
	text     string
}

type twin struct {
	fd        *ast.FuncDecl
	closeEdit int    // index of the edit that closes the body
	closeText string // the same text without the invariant check
}

type file struct {
	in      *Input
	pkg     *project.Package
	cf      *project.CodeFile
	af      *ast.File
	tf      *token.File
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

// Package lowers the contracts of one package. It returns the Go text of each code file, and
// reports contract diagnostics.
func Package(in *Input, pkg *project.Package) map[*project.CodeFile][]byte {
	out := map[*project.CodeFile][]byte{}
	type parsed struct {
		cf *project.CodeFile
		af *ast.File
		tf *token.File
	}
	var files []parsed
	invs := map[string]*invType{}
	for _, cf := range pkg.Files {
		fset := token.NewFileSet()
		af, err := parser.ParseFile(fset, cf.Rel, cf.Blank.Go, parser.SkipObjectResolution)
		if err != nil || af == nil {
			out[cf] = cf.Blank.Go // the go command reports the syntax error
			continue
		}
		reservedNames(in.P, cf, af, fset)
		files = append(files, parsed{cf, af, fset.File(af.Pos())})
		for _, t := range cf.Blank.Types {
			it := &invType{name: t.Name, file: cf, clauses: t.Clauses, fields: map[string]bool{}, methods: map[string]bool{}}
			for _, d := range af.Decls {
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
	// Methods of invariant types, from every file of the package.
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
		lw := &file{in: in, pkg: pkg, cf: f.cf, af: f.af, tf: f.tf, src: f.cf.Blank.Go, lines: lex.NewLines(f.cf.Blank.Go),
			lf: in.LineFile(f.cf.Rel), invs: invs, oldNext: &oldNext}
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
func reservedNames(p *project.Project, cf *project.CodeFile, af *ast.File, fset *token.FileSet) {
	check := func(id *ast.Ident) {
		if id != nil && (strings.HasPrefix(id.Name, "vulcan_") || strings.HasPrefix(id.Name, "Vulcan_")) {
			pos := fset.Position(id.Pos())
			p.Diags.Errorf(diag.Pos{File: cf.Rel, Line: pos.Line, Col: pos.Column}, "K0005", "the name %s starts with %s, which is reserved for generated code", id.Name, id.Name[:7])
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

func (lw *file) off(p token.Pos) int { return lw.tf.Offset(p) }

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

func (lw *file) add(off, end int, text string) int {
	lw.edits = append(lw.edits, edit{off: off, end: end, text: text})
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
	if lw.usesRT {
		lw.add(lw.off(lw.af.Name.End()), lw.off(lw.af.Name.End()), "; import vulcan_rt "+strconv.Quote(RT))
	}
	var b strings.Builder
	b.WriteString(apply(lw.src, lw.edits, 0, len(lw.src)))
	for _, tw := range lw.twins {
		edits := append([]edit{}, lw.edits...)
		edits[tw.closeEdit].text = tw.closeText
		name := tw.fd.Name
		edits = append(edits, edit{off: lw.off(name.Pos()), end: lw.off(name.End()),
			text: lw.dir(lw.off(name.Pos())) + "vulcan_" + name.Name + lw.dir(lw.off(name.End()))})
		start := lw.off(tw.fd.Pos())
		l, c := lw.lines.Pos(start)
		fmt.Fprintf(&b, "\n\n//line %s:%d:1\n%s%s", lw.lf, l, strings.Repeat(" ", c-1), apply(lw.src, edits, start, lw.off(tw.fd.End())))
	}
	for _, x := range lw.extra {
		b.WriteString("\n\n")
		b.WriteString(x)
	}
	b.WriteString("\n")
	return []byte(b.String())
}

// apply applies the edits inside [from, to) to src[from:to]. Edits at one offset keep their order.
func apply(src []byte, edits []edit, from, to int) string {
	var in []edit
	for _, e := range edits {
		if e.off >= from && e.end <= to {
			in = append(in, e)
		}
	}
	sort.SliceStable(in, func(i, j int) bool { return in[i].off < in[j].off })
	var b strings.Builder
	at := from
	for _, e := range in {
		if e.off < at {
			continue // overlapping: keep the first
		}
		b.Write(src[at:e.off])
		b.WriteString(e.text)
		at = e.end
	}
	b.Write(src[at:to])
	return b.String()
}

type result struct {
	name string // the name in the source, or ""
	typ  string
}

// function lowers one function or method (Appendix F, with the check after the body).
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
			params[n.Name] = true
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
	rv := func(i int) string { return "vulcan_r" + strconv.Itoa(i) }
	named := map[string]int{}
	for i, r := range results {
		if r.name != "" && r.name != "_" {
			named[r.name] = i
		}
	}
	var open, closeChecks strings.Builder
	var olds []string
	for _, c := range clauses {
		e, err := ParseContract(c.Text)
		if err != nil {
			p.Diags.Errorf(lw.cf.Pos(c.Off), "K0001", "the contract does not parse: %v", err)
			continue
		}
		ensures := c.Word == "ensures"
		lw.checkNames(e, c, ensures, errIdx, len(nonErr))
		var shows []string
		seen := map[string]bool{}
		show := func(label, expr string) {
			if !seen[label] {
				seen[label] = true
				shows = append(shows, fmt.Sprintf("vulcan_rt.Show(%q, %s)", label, expr))
			}
		}
		x := Rewrite(e, func(n ast.Expr) ast.Expr {
			switch v := n.(type) {
			case *ast.Ident:
				switch {
				case params[v.Name]:
					show(v.Name, v.Name)
				case v.Name == "err" && errIdx >= 0 && ensures:
					show("err", rv(errIdx))
					return ast.NewIdent(rv(errIdx))
				case v.Name == "result" && len(nonErr) > 0 && ensures:
					show("result", rv(nonErr[0]))
					return ast.NewIdent(rv(nonErr[0]))
				case strings.HasPrefix(v.Name, "vulcan_result_") && ensures:
					k, _ := strconv.Atoi(strings.TrimPrefix(v.Name, "vulcan_result_"))
					if k < len(nonErr) {
						show("result."+strconv.Itoa(k), rv(nonErr[k]))
						return ast.NewIdent(rv(nonErr[k]))
					}
				default:
					if i, ok := named[v.Name]; ok && ensures {
						show(v.Name, rv(i))
						return ast.NewIdent(rv(i))
					}
				}
			case *ast.CallExpr:
				if id, ok := v.Fun.(*ast.Ident); ok && id.Name == "old" && len(v.Args) == 1 && ensures {
					name := fmt.Sprintf("vulcan_old%d", *lw.oldNext)
					*lw.oldNext++
					olds = append(olds, fmt.Sprintf("%s%s := %s; ", lw.clauseDir(c), name, GoText(v.Args[0])))
					if lw.pkg.Olds == nil {
						lw.pkg.Olds = map[string]diag.Pos{}
					}
					lw.pkg.Olds[name] = lw.cf.Pos(c.Off)
					show("old("+GoText(v.Args[0])+")", name)
					return ast.NewIdent(name)
				}
			}
			return nil
		})
		goText := GoText(x)
		args := fmt.Sprintf("%q, %q, %q, %q", c.Word, c.Text, lw.cf.Pos(c.Off).String(), caller)
		if len(shows) > 0 {
			args += ", " + strings.Join(shows, ", ")
		}
		check := fmt.Sprintf("if !(%s%s) { vulcan_rt.Fail(%s) }; ", lw.clauseDir(c), goText, args)
		lw.usesRT = true
		if ensures {
			closeChecks.WriteString(check)
		} else {
			open.WriteString(check)
		}
	}
	for _, o := range olds {
		open.WriteString(o)
	}
	var vars []string
	for i := range results {
		vars = append(vars, rv(i))
	}
	lbrace, rbrace := lw.off(fd.Body.Lbrace), lw.off(fd.Body.Rbrace)
	resText := ""
	if fd.Type.Results != nil {
		resText = string(lw.src[lw.off(fd.Type.Results.Pos()):lw.off(fd.Type.Results.End())])
	}
	if len(vars) > 0 {
		fmt.Fprintf(&open, "%s := func() %s {", strings.Join(vars, ", "), resText)
	} else {
		open.WriteString("func() {")
	}
	open.WriteString(lw.dir(lbrace + 1))
	lw.add(lbrace+1, lbrace+1, " "+open.String())
	ret := "return; "
	if len(vars) > 0 {
		ret = "return " + strings.Join(vars, ", ") + "; "
	}
	tail := lw.dir(rbrace)
	withoutInv := "}(); " + closeChecks.String() + ret + tail
	withInv := withoutInv
	if it != nil {
		withInv = "}(); " + closeChecks.String() + recvName + ".Vulcan_invariant(); " + ret + tail
		lw.receiverCalls(fd, recvName, it)
	}
	idx := lw.add(rbrace, rbrace, withInv)
	if it != nil {
		lw.twins = append(lw.twins, twin{fd: fd, closeEdit: idx, closeText: withoutInv})
	}
}

// receiverCalls makes a call through the method's own receiver call the twin without the
// invariant check (§7.2, "outer calls only"), unless the method assigns the receiver.
func (lw *file) receiverCalls(fd *ast.FuncDecl, recv string, it *invType) {
	assigned := false
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		if a, ok := n.(*ast.AssignStmt); ok {
			for _, l := range a.Lhs {
				if id, ok := l.(*ast.Ident); ok && id.Name == recv {
					assigned = true
				}
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
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == recv && it.methods[sel.Sel.Name] {
			s, e := lw.off(sel.Sel.Pos()), lw.off(sel.Sel.End())
			lw.add(s, e, lw.dir(s)+"vulcan_"+sel.Sel.Name+lw.dir(e))
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
			walk(x.X, errNil)
			walk(x.Y, errNil)
		case *ast.ParenExpr:
			walk(x.X, errNil)
		case *ast.UnaryExpr:
			walk(x.X, errNil)
		case *ast.CallExpr:
			for _, a := range x.Args {
				walk(a, errNil)
			}
		case *ast.SelectorExpr:
			walk(x.X, errNil)
		case *ast.IndexExpr:
			walk(x.X, errNil)
			walk(x.Index, errNil)
		case *ast.Ident:
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
			case x.Name == "err" && errIdx < 0:
				p.Diags.Errorf(pos, "K0510", "err names the last result when its type is error; this function has no error result")
			}
		}
	}
	walk(e, false)
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

// literals checks each struct literal of a type with an invariant where it is built (§7.2).
func (lw *file) literals() {
	names := map[string]string{} // package name in this file -> import path
	for _, imp := range lw.af.Imports {
		ip, _ := strconv.Unquote(imp.Path.Value)
		name := path.Base(ip)
		if pkg := lw.in.P.ByPath[ip]; pkg != nil {
			name = pkg.GoName
		}
		if imp.Name != nil {
			name = imp.Name.Name
		}
		names[name] = ip
	}
	checker := func(t ast.Expr) (string, bool) {
		switch x := t.(type) {
		case *ast.Ident:
			if it := lw.invs[x.Name]; it != nil && !it.generic {
				return "Vulcan_check" + x.Name, true
			}
		case *ast.SelectorExpr:
			if pk, ok := x.X.(*ast.Ident); ok {
				if ip, ok := names[pk.Name]; ok && lw.in.Inv[ip][x.Sel.Name] && ip != lw.pkg.ImportPath {
					return pk.Name + ".Vulcan_check" + x.Sel.Name, true
				}
			}
		}
		return "", false
	}
	done := map[*ast.CompositeLit]bool{}
	wrap := func(n ast.Node, fn string) {
		s, e := lw.off(n.Pos()), lw.off(n.End())
		lw.add(s, s, fn+"("+lw.dir(s))
		lw.add(e, e, ")"+lw.dir(e))
	}
	ast.Inspect(lw.af, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.UnaryExpr:
			if lit, ok := x.X.(*ast.CompositeLit); ok && x.Op == token.AND {
				if fn, ok := checker(lit.Type); ok {
					done[lit] = true
					i := strings.LastIndex(fn, "Vulcan_check")
					wrap(x, fn[:i]+"Vulcan_checkPtr"+fn[i+len("Vulcan_check"):])
				}
			}
		case *ast.CompositeLit:
			if done[x] {
				return true
			}
			if fn, ok := checker(x.Type); ok {
				wrap(x, fn)
			}
		}
		return true
	})
}

// invariantMethods writes Vulcan_invariant, Vulcan_checkT, and Vulcan_checkPtrT (Appendix F).
func (lw *file) invariantMethods(it *invType) {
	lw.usesRT = true
	var b strings.Builder
	first := it.clauses[0]
	fp := lw.cf.Pos(first.WordOff)
	fmt.Fprintf(&b, "//line %s:%d:1\nfunc (vulcan_v *%s) Vulcan_invariant() { if vulcan_v == nil { return }; ", lw.lf, fp.Line, it.name)
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
		args := fmt.Sprintf("%q, %q, %q, %q", "invariant", c.Text, lw.cf.Pos(c.Off).String(), it.name)
		if len(shows) > 0 {
			args += ", " + strings.Join(shows, ", ")
		}
		fmt.Fprintf(&b, "if !(%s%s) { vulcan_rt.Fail(%s) }; ", lw.clauseDir(c), GoText(x), args)
	}
	b.WriteString("}\n")
	fmt.Fprintf(&b, "func Vulcan_check%s(vulcan_x %s) %s { (&vulcan_x).Vulcan_invariant(); return vulcan_x }\n", it.name, it.name, it.name)
	fmt.Fprintf(&b, "func Vulcan_checkPtr%s(vulcan_p *%s) *%s { vulcan_p.Vulcan_invariant(); return vulcan_p }", it.name, it.name, it.name)
	lw.extra = append(lw.extra, b.String())
}
