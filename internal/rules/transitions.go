package rules

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"slices"
	"strconv"
	"strings"
)

// A transitions rule is a table of the legal changes of a status field. Each write of the field
// must be a constant of the table, and must follow from a state that the function knows: an
// earlier write in the function, or a guard. The state in a store is not known to a check, so a
// guard is the only fact about it.
//
//	rule ORDER-STATE
//	  kind transitions
//	  type lib/shop.Order
//	  field Status
//	  initial pending
//	  allow pending->paid paid->shipped *->cancelled
//	  load gateway
//	  ...
//
// The check reads one function at a time. What it does not know, it rejects: a write through a
// value that it cannot name (ptrs[i].Status), a write after a call that can change the value, and
// a guard through a copy or a method. A part in load may build the field from a value that is not
// a constant, which is how a value comes back from a store.

type table struct {
	pkg, typ, field string
	initial         map[string]bool
	states          map[string]bool
	edges           map[[2]string]bool // from "*" for any state
	load            []string
}

func newTable(r *Rule, module string) *table {
	t := &table{initial: map[string]bool{}, states: map[string]bool{}, edges: map[[2]string]bool{}, load: r.Params["load"]}
	if v := r.Params["type"]; len(v) == 1 {
		dir, name, _ := strings.Cut(v[0], ".")
		t.pkg, t.typ = module+"/"+dir, name
	}
	if v := r.Params["field"]; len(v) == 1 {
		t.field = v[0]
	}
	for _, e := range r.Params["allow"] {
		from, to, _ := strings.Cut(e, "->")
		t.edges[[2]string{from, to}] = true
		if from != "*" {
			t.states[from] = true
		}
		t.states[to] = true
	}
	for _, s := range r.Params["initial"] {
		t.initial[s] = true
		t.states[s] = true
	}
	return t
}

func (t *table) legal(from, to string) bool {
	return from == to || t.edges[[2]string{from, to}] || t.edges[[2]string{"*", to}]
}

// stateOf gives the word of a constant in a table.
func stateOf(v constant.Value) string {
	if v.Kind() == constant.String {
		return constant.StringVal(v)
	}
	return v.ExactString()
}

// A key names one status field of one variable, such as o.Status.
type key struct {
	root types.Object
	path string
}

// A fact says which states a field can be in: in holds them when it is not nil; out holds states
// that it is not in.
type fact struct{ in, out map[string]bool }

type flowState map[key]map[string]bool

func (st flowState) clone() flowState {
	out := flowState{}
	for k, v := range st {
		out[k] = v
	}
	return out
}

func merge(a, b flowState) flowState {
	out := flowState{}
	for k, av := range a {
		if bv, ok := b[k]; ok {
			u := map[string]bool{}
			for s := range av {
				u[s] = true
			}
			for s := range bv {
				u[s] = true
			}
			out[k] = u
		}
	}
	return out
}

type flow struct {
	u    *Unit
	r    *Rule
	t    *table
	add  addFunc
	load bool
}

func (u *Unit) transitions(r *Rule, add addFunc) {
	fl := &flow{u: u, r: r, t: newTable(r, u.Module), add: add, load: selected(r.Params["load"], u.Part)}
	for _, f := range u.Files {
		if u.IsTest(f) {
			continue // a test builds a value in any state that it needs
		}
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Body != nil {
					fl.block(d.Body.List, flowState{})
				}
			case *ast.GenDecl:
				for _, sp := range d.Specs {
					if vs, ok := sp.(*ast.ValueSpec); ok {
						for _, v := range vs.Values {
							fl.expr(v, flowState{})
						}
					}
				}
			}
		}
	}
}

// target reports whether e is a selection of the field of the table, and gives its key. A key with
// no root is a value that the check cannot name.
func (fl *flow) target(e ast.Expr) (key, bool) {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok {
		return key{}, false
	}
	s := fl.u.Info.Selections[sel]
	if s == nil || s.Kind() != types.FieldVal || sel.Sel.Name != fl.t.field {
		return key{}, false
	}
	owner := s.Recv()
	idx := s.Index()
	for _, i := range idx[:len(idx)-1] {
		st, ok := derefStruct(owner)
		if !ok {
			return key{}, false
		}
		owner = st.Field(i).Type()
	}
	n := namedOf(owner)
	if n == nil || n.Obj().Pkg() == nil || n.Obj().Pkg().Path() != fl.t.pkg || n.Obj().Name() != fl.t.typ {
		return key{}, false
	}
	return key{fl.root(sel.X), types.ExprString(sel)}, true
}

func derefStruct(t types.Type) (*types.Struct, bool) {
	if p, ok := types.Unalias(t).(*types.Pointer); ok {
		t = p.Elem()
	}
	st, ok := t.Underlying().(*types.Struct)
	return st, ok
}

// root gives the variable at the left of a chain of fields, or nil for anything else.
func (fl *flow) root(e ast.Expr) types.Object {
	switch x := ast.Unparen(e).(type) {
	case *ast.Ident:
		return fl.u.Info.Uses[x]
	case *ast.SelectorExpr:
		if s := fl.u.Info.Selections[x]; s != nil && s.Kind() == types.FieldVal {
			return fl.root(x.X)
		}
		if v, ok := fl.u.Info.Uses[x.Sel].(*types.Var); ok {
			return v // a package-level variable of another package
		}
	case *ast.StarExpr:
		return fl.root(x.X)
	}
	return nil
}

func (fl *flow) constState(e ast.Expr) (string, bool) {
	if tv, ok := fl.u.Info.Types[e]; ok && tv.Value != nil {
		return stateOf(tv.Value), true
	}
	return "", false
}

// forget drops what is known of each field reached through e.
func (fl *flow) forget(e ast.Expr, st flowState) {
	root := fl.root(e)
	if root == nil {
		return
	}
	prefix := types.ExprString(ast.Unparen(e))
	whole := len(prefix) > 0 && prefix[0] == '*'
	for k := range st {
		if k.root == root && (whole || k.path == prefix || strings.HasPrefix(k.path, prefix+".")) {
			delete(st, k)
		}
	}
}

func (fl *flow) forgetRoot(root types.Object, st flowState) {
	for k := range st {
		if k.root == root {
			delete(st, k)
		}
	}
}

// write checks one write of the field.
func (fl *flow) write(k key, value ast.Expr, st flowState, at token.Pos) {
	to, isConst := "", false
	if value != nil {
		to, isConst = fl.constState(value)
	}
	if !isConst {
		if !fl.load {
			fl.add("0805", at, "%s writes %s with a value that is not a constant, so no table can check it; write a constant of the table, or load the value in a part that the rule names in load: %s", fl.u.Part.Dir, k.path, fl.r.tail())
		}
		delete(st, k)
		return
	}
	if !fl.t.states[to] {
		fl.add("0805", at, "%s writes %s = %q, which is not a state of the table: %s", fl.u.Part.Dir, k.path, to, fl.r.tail())
		st[k] = map[string]bool{to: true}
		return
	}
	prior, known := st[k]
	if k.root == nil {
		known = false
	}
	if !known {
		if !fl.t.edges[[2]string{"*", to}] {
			fl.add("0805", at, "%s writes %s = %q, and the state before it is not known here; guard the write with a test of %s: %s", fl.u.Part.Dir, k.path, to, k.path, fl.r.tail())
		}
	} else {
		var bad []string
		for from := range prior {
			if !fl.t.legal(from, to) {
				bad = append(bad, from)
			}
		}
		if len(bad) > 0 {
			slices.Sort(bad)
			fl.add("0805", at, "%s changes %s from %s to %q, which the table does not allow: %s", fl.u.Part.Dir, k.path, strings.Join(bad, " or "), to, fl.r.tail())
		}
	}
	if k.root != nil {
		st[k] = map[string]bool{to: true}
	}
}

// expr reads an expression: its literals, closures, addresses, and calls.
func (fl *flow) expr(e ast.Expr, st flowState) {
	if e == nil {
		return
	}
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			fl.forgetWritten(x.Body, st)
			fl.block(x.Body.List, flowState{})
			return false
		case *ast.CompositeLit:
			fl.literal(x)
		case *ast.UnaryExpr:
			if x.Op == token.AND {
				if k, ok := fl.target(x.X); ok {
					if !fl.load {
						fl.add("0805", x.Pos(), "%s takes the address of %s, so a write through it cannot be checked: %s", fl.u.Part.Dir, k.path, fl.r.tail())
					}
					delete(st, k)
				}
			}
		case *ast.CallExpr:
			fl.call(x, st)
		}
		return true
	})
}

// call forgets what a call can change: a method with a pointer receiver on a value, and each value
// that the call gets by its address or as a pointer.
func (fl *flow) call(c *ast.CallExpr, st flowState) {
	if sel, ok := ast.Unparen(c.Fun).(*ast.SelectorExpr); ok {
		if s := fl.u.Info.Selections[sel]; s != nil && s.Kind() == types.MethodVal {
			if sig, ok := s.Obj().Type().(*types.Signature); ok && sig.Recv() != nil {
				if _, ptr := types.Unalias(sig.Recv().Type()).(*types.Pointer); ptr {
					if root := fl.root(sel.X); root != nil {
						fl.forgetRoot(root, st)
					}
				}
			}
		}
	}
	for _, a := range c.Args {
		a := ast.Unparen(a)
		if u, ok := a.(*ast.UnaryExpr); ok && u.Op == token.AND {
			if root := fl.root(u.X); root != nil {
				fl.forgetRoot(root, st)
			}
			continue
		}
		if _, ptr := types.Unalias(fl.u.Info.TypeOf(a)).(*types.Pointer); ptr {
			if root := fl.root(a); root != nil {
				fl.forgetRoot(root, st)
			}
		}
	}
}

// literal checks a value of the type that a literal builds: its state is a state in which a value
// can start. An empty literal is no value yet (return T{}, err).
func (fl *flow) literal(x *ast.CompositeLit) {
	n := namedOf(fl.u.Info.TypeOf(x))
	if n == nil || n.Obj().Pkg() == nil || n.Obj().Pkg().Path() != fl.t.pkg || n.Obj().Name() != fl.t.typ {
		return
	}
	st, ok := n.Underlying().(*types.Struct)
	if !ok || len(x.Elts) == 0 {
		return
	}
	check := func(v ast.Expr) {
		s, isConst := fl.constState(v)
		switch {
		case !isConst:
			if !fl.load {
				fl.add("0805", v.Pos(), "%s builds a %s with %s that is not a constant, so no table can check it; use a constant of the table, or build it in a part that the rule names in load: %s", fl.u.Part.Dir, fl.t.typ, fl.t.field, fl.r.tail())
			}
		case !fl.t.states[s]:
			fl.add("0805", v.Pos(), "%s builds a %s with %s = %q, which is not a state of the table: %s", fl.u.Part.Dir, fl.t.typ, fl.t.field, s, fl.r.tail())
		case len(fl.t.initial) > 0 && !fl.t.initial[s]:
			fl.add("0805", v.Pos(), "%s builds a %s with %s = %q, and a value starts only in %s: %s", fl.u.Part.Dir, fl.t.typ, fl.t.field, s, words(fl.t.initial), fl.r.tail())
		}
	}
	for i, e := range x.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok && id.Name == fl.t.field {
				check(kv.Value)
				return
			}
			continue
		}
		if i < st.NumFields() && st.Field(i).Name() == fl.t.field {
			check(e)
			return
		}
	}
	// A value that omits the field starts in the zero value of the field.
	if fl.load || len(fl.t.initial) == 0 {
		return
	}
	for i := 0; i < st.NumFields(); i++ {
		if st.Field(i).Name() != fl.t.field {
			continue
		}
		zero := "0"
		if b, ok := st.Field(i).Type().Underlying().(*types.Basic); ok && b.Info()&types.IsString != 0 {
			zero = ""
		}
		if !fl.t.initial[zero] {
			fl.add("0805", x.Pos(), "%s builds a %s that omits %s, so it starts in %s, and a value starts only in %s: %s", fl.u.Part.Dir, fl.t.typ, fl.t.field, strconv.Quote(zero), words(fl.t.initial), fl.r.tail())
		}
	}
}

func words(s map[string]bool) string {
	var out []string
	for w := range s {
		out = append(out, w)
	}
	slices.Sort(out)
	return strings.Join(out, ", ")
}

// forgetWritten drops what a loop body or a closure can change.
func (fl *flow) forgetWritten(body ast.Node, st flowState) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, l := range x.Lhs {
				if id, ok := l.(*ast.Ident); ok && id.Name == "_" {
					continue
				}
				fl.forget(l, st)
			}
		case *ast.IncDecStmt:
			fl.forget(x.X, st)
		case *ast.RangeStmt:
			for _, l := range []ast.Expr{x.Key, x.Value} {
				if l != nil && x.Tok == token.ASSIGN {
					fl.forget(l, st)
				}
			}
		case *ast.UnaryExpr:
			if x.Op == token.AND {
				if root := fl.root(x.X); root != nil {
					fl.forgetRoot(root, st)
				}
			}
		case *ast.CallExpr:
			fl.call(x, st)
		}
		return true
	})
}

// block walks statements in order, and reports whether the last one ends the path.
func (fl *flow) block(list []ast.Stmt, st flowState) bool {
	for _, s := range list {
		if fl.stmt(s, st) {
			return true
		}
	}
	return false
}

func replace(st, with flowState) {
	for k := range st {
		delete(st, k)
	}
	for k, v := range with {
		st[k] = v
	}
}

// stmt walks one statement, changes st to the state after it, and reports whether it ends the path.
func (fl *flow) stmt(s ast.Stmt, st flowState) bool {
	switch x := s.(type) {
	case *ast.ExprStmt:
		fl.expr(x.X, st)
		if c, ok := ast.Unparen(x.X).(*ast.CallExpr); ok {
			if id, ok := ast.Unparen(c.Fun).(*ast.Ident); ok {
				if _, builtin := fl.u.Info.Uses[id].(*types.Builtin); builtin && id.Name == "panic" {
					return true
				}
			}
		}
	case *ast.ReturnStmt:
		for _, e := range x.Results {
			fl.expr(e, st)
		}
		return true
	case *ast.BranchStmt:
		return x.Tok != token.FALLTHROUGH
	case *ast.AssignStmt:
		fl.assign(x, st)
	case *ast.IncDecStmt:
		if k, ok := fl.target(x.X); ok {
			fl.write(k, nil, st, x.Pos())
		} else {
			fl.forget(x.X, st)
		}
	case *ast.DeclStmt:
		if gd, ok := x.Decl.(*ast.GenDecl); ok {
			for _, sp := range gd.Specs {
				if vs, ok := sp.(*ast.ValueSpec); ok {
					for _, v := range vs.Values {
						fl.expr(v, st)
					}
				}
			}
		}
	case *ast.DeferStmt:
		fl.expr(x.Call, st)
	case *ast.GoStmt:
		fl.expr(x.Call, st)
	case *ast.SendStmt:
		fl.expr(x.Chan, st)
		fl.expr(x.Value, st)
	case *ast.BlockStmt:
		return fl.block(x.List, st)
	case *ast.LabeledStmt:
		// A goto can come here from anywhere, so nothing is known.
		replace(st, flowState{})
		return fl.stmt(x.Stmt, st)
	case *ast.IfStmt:
		return fl.ifStmt(x, st)
	case *ast.ForStmt:
		if x.Init != nil {
			fl.stmt(x.Init, st)
		}
		fl.forgetWritten(x.Body, st)
		if x.Post != nil {
			fl.forgetWritten(x.Post, st)
		}
		fl.expr(x.Cond, st)
		body := st.clone()
		if x.Cond != nil {
			fl.applyFact(fl.facts(x.Cond, true), body)
		}
		fl.block(x.Body.List, body)
		if x.Post != nil {
			fl.stmt(x.Post, body)
		}
	case *ast.RangeStmt:
		fl.expr(x.X, st)
		if x.Tok == token.ASSIGN {
			for _, l := range []ast.Expr{x.Key, x.Value} {
				if l == nil {
					continue
				}
				if k, ok := fl.target(l); ok {
					fl.write(k, nil, st, l.Pos())
				}
			}
		}
		fl.forgetWritten(x.Body, st)
		fl.block(x.Body.List, st.clone())
	case *ast.SwitchStmt:
		if x.Init != nil {
			fl.stmt(x.Init, st)
		}
		fl.expr(x.Tag, st)
		return fl.switchStmt(x, st)
	case *ast.TypeSwitchStmt:
		if x.Init != nil {
			fl.stmt(x.Init, st)
		}
		return fl.clauses(x.Body, st, false, func(ast.Stmt, flowState) {})
	case *ast.SelectStmt:
		return fl.clauses(x.Body, st, true, func(c ast.Stmt, in flowState) {
			if comm := c.(*ast.CommClause).Comm; comm != nil {
				fl.stmt(comm, in)
			}
		})
	}
	return false
}

func hasBranch(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.BranchStmt:
			if x.Tok != token.FALLTHROUGH {
				found = true
			}
		}
		return !found
	})
	return found
}

func (fl *flow) assign(x *ast.AssignStmt, st flowState) {
	for _, r := range x.Rhs {
		fl.expr(r, st)
	}
	for i, l := range x.Lhs {
		var v ast.Expr
		if len(x.Lhs) == len(x.Rhs) {
			v = x.Rhs[i]
		}
		if k, ok := fl.target(l); ok {
			if x.Tok != token.ASSIGN && x.Tok != token.DEFINE {
				v = nil // += and the others
			}
			fl.write(k, v, st, l.Pos())
			continue
		}
		fl.forget(l, st)
		// A new value of a literal has the state that the literal gives it.
		if id, ok := l.(*ast.Ident); ok && v != nil {
			if lit, ok := ast.Unparen(v).(*ast.CompositeLit); ok {
				fl.fromLiteral(id, lit, st)
			}
		}
	}
}

func (fl *flow) fromLiteral(id *ast.Ident, lit *ast.CompositeLit, st flowState) {
	n := namedOf(fl.u.Info.TypeOf(lit))
	if n == nil || n.Obj().Pkg() == nil || n.Obj().Pkg().Path() != fl.t.pkg || n.Obj().Name() != fl.t.typ {
		return
	}
	var root types.Object = fl.u.Info.Defs[id]
	if root == nil {
		root = fl.u.Info.Uses[id]
	}
	for _, e := range lit.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			if k, ok := kv.Key.(*ast.Ident); ok && k.Name == fl.t.field {
				if s, isConst := fl.constState(kv.Value); isConst {
					st[key{root, id.Name + "." + fl.t.field}] = map[string]bool{s: true}
				}
			}
		}
	}
}

func (fl *flow) ifStmt(x *ast.IfStmt, st flowState) bool {
	if x.Init != nil {
		fl.stmt(x.Init, st)
	}
	fl.expr(x.Cond, st)
	then, els := st.clone(), st.clone()
	fl.applyFact(fl.facts(x.Cond, true), then)
	fl.applyFact(fl.facts(x.Cond, false), els)
	thenEnd := fl.block(x.Body.List, then)
	elseEnd := false
	if x.Else != nil {
		elseEnd = fl.stmt(x.Else, els)
	}
	switch {
	case thenEnd && elseEnd:
		return true
	case thenEnd:
		replace(st, els)
	case elseEnd:
		replace(st, then)
	default:
		replace(st, merge(then, els))
	}
	return false
}

func (fl *flow) switchStmt(x *ast.SwitchStmt, st flowState) bool {
	var tag key
	hasTag := false
	if x.Tag != nil {
		tag, hasTag = fl.target(x.Tag)
		hasTag = hasTag && tag.root != nil
	}
	seen := map[string]bool{}
	return fl.clauses(x.Body, st, false, func(c ast.Stmt, in flowState) {
		cc := c.(*ast.CaseClause)
		for _, e := range cc.List {
			fl.expr(e, in)
		}
		switch {
		case hasTag && cc.List == nil:
			fl.applyFact(map[key]fact{tag: {out: seen}}, in)
		case hasTag:
			states := map[string]bool{}
			for _, e := range cc.List {
				s, ok := fl.constState(e)
				if !ok {
					states = nil
					break
				}
				states[s], seen[s] = true, true
			}
			if states != nil {
				fl.applyFact(map[key]fact{tag: {in: states}}, in)
			}
		case x.Tag == nil && cc.List != nil:
			f := fl.facts(cc.List[0], true)
			for _, e := range cc.List[1:] {
				f = union(f, fl.facts(e, true))
			}
			fl.applyFact(f, in)
		}
	})
}

// clauses walks the clauses of a switch or a select. A path that leaves by break or goto joins the
// code after the statement with what it knows, so then nothing is known there.
func (fl *flow) clauses(body *ast.BlockStmt, st flowState, isSelect bool, enter func(ast.Stmt, flowState)) bool {
	entry := st.clone()
	var after flowState
	hasDefault := false
	branches := hasBranch(body)
	for _, c := range body.List {
		in := entry.clone()
		enter(c, in)
		var list []ast.Stmt
		switch c := c.(type) {
		case *ast.CaseClause:
			list = c.Body
			if c.List == nil {
				hasDefault = true
			}
		case *ast.CommClause:
			list = c.Body
			hasDefault = hasDefault || c.Comm == nil
		}
		// A clause that falls through can come here from another clause.
		if i := slices.Index(body.List, c); i > 0 {
			if prev, ok := body.List[i-1].(*ast.CaseClause); ok && len(prev.Body) > 0 {
				if b, ok := prev.Body[len(prev.Body)-1].(*ast.BranchStmt); ok && b.Tok == token.FALLTHROUGH {
					replace(in, flowState{})
				}
			}
		}
		if fl.block(list, in) {
			continue
		}
		if after == nil {
			after = in
		} else {
			after = merge(after, in)
		}
	}
	if !hasDefault && !isSelect {
		if after == nil {
			after = entry
		} else {
			after = merge(after, entry)
		}
	}
	if branches {
		replace(st, flowState{})
		return false
	}
	if after == nil {
		return true
	}
	replace(st, after)
	return false
}

// applyFact narrows a state by what a condition says.
func (fl *flow) applyFact(f map[key]fact, st flowState) {
	for k, x := range f {
		prior, known := st[k]
		switch {
		case x.in != nil && known:
			now := map[string]bool{}
			for s := range prior {
				if x.in[s] {
					now[s] = true
				}
			}
			st[k] = now
		case x.in != nil:
			st[k] = x.in
		case known:
			now := map[string]bool{}
			for s := range prior {
				if !x.out[s] {
					now[s] = true
				}
			}
			st[k] = now
		}
	}
}

// facts says what a condition tells when it is true (want) or false.
func (fl *flow) facts(e ast.Expr, want bool) map[key]fact {
	switch x := ast.Unparen(e).(type) {
	case *ast.UnaryExpr:
		if x.Op == token.NOT {
			return fl.facts(x.X, !want)
		}
	case *ast.BinaryExpr:
		switch x.Op {
		case token.LAND:
			if want {
				return both(fl.facts(x.X, true), fl.facts(x.Y, true))
			}
			return union(fl.facts(x.X, false), fl.facts(x.Y, false))
		case token.LOR:
			if want {
				return union(fl.facts(x.X, true), fl.facts(x.Y, true))
			}
			return both(fl.facts(x.X, false), fl.facts(x.Y, false))
		case token.EQL, token.NEQ:
			k, s, ok := fl.compare(x)
			if !ok {
				return nil
			}
			one := map[string]bool{s: true}
			if (x.Op == token.EQL) == want {
				return map[key]fact{k: {in: one}}
			}
			return map[key]fact{k: {out: one}}
		}
	}
	return nil
}

func (fl *flow) compare(x *ast.BinaryExpr) (key, string, bool) {
	for _, p := range [][2]ast.Expr{{x.X, x.Y}, {x.Y, x.X}} {
		if k, ok := fl.target(p[0]); ok && k.root != nil {
			if s, ok := fl.constState(p[1]); ok {
				return k, s, true
			}
		}
	}
	return key{}, "", false
}

// both is what two conditions tell when both hold.
func both(a, b map[key]fact) map[key]fact {
	out := map[key]fact{}
	for k, v := range a {
		out[k] = v
	}
	for k, y := range b {
		x, ok := out[k]
		if !ok {
			out[k] = y
			continue
		}
		var f fact
		switch {
		case x.in != nil && y.in != nil:
			f.in = map[string]bool{}
			for s := range x.in {
				if y.in[s] {
					f.in[s] = true
				}
			}
		case x.in != nil:
			f.in = minus(x.in, y.out)
		case y.in != nil:
			f.in = minus(y.in, x.out)
		default:
			f.out = map[string]bool{}
			for s := range x.out {
				f.out[s] = true
			}
			for s := range y.out {
				f.out[s] = true
			}
		}
		out[k] = f
	}
	return out
}

func minus(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for s := range a {
		if !b[s] {
			out[s] = true
		}
	}
	return out
}

// union is what one of two conditions tells: only a field that both name, as a set of states.
func union(a, b map[key]fact) map[key]fact {
	out := map[key]fact{}
	for k, x := range a {
		y, ok := b[k]
		if !ok || x.in == nil || y.in == nil {
			continue
		}
		f := fact{in: map[string]bool{}}
		for s := range x.in {
			f.in[s] = true
		}
		for s := range y.in {
			f.in[s] = true
		}
		out[k] = f
	}
	return out
}
