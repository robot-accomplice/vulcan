// Package contracts is analyzer A7 of vulcan-lint: the calls of the contract package.
package contracts

import (
	"bytes"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A7.
var Analyzer = &analysis.Analyzer{
	Name: "contracts",
	Doc:  "check the contracts: contract.Requires, contract.Old, defer contract.Ensures, and defer contract.Keep are the first statements of a function; a contract has no effects; old values hold no reference; a call with constant arguments keeps the callee's Requires (VL0501, VL0503, VL0509, VL0512, VL0903)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Manifest || u.Part == nil || u.Contract {
		return nil, nil
	}
	cp, info := u.M.ContractPath(), pass.TypesInfo
	placed := map[*ast.CallExpr]bool{}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			pro := model.PrologueOf(info, fd.Body, cp)
			for call := range pro.Calls {
				placed[call] = true
			}
			for _, call := range pro.Requires {
				for _, a := range call.Args {
					Pure(pass, a, "contract.Requires")
				}
			}
			for _, call := range pro.Olds {
				for _, a := range call.Args {
					Pure(pass, a, "contract.Old")
					if t := info.TypeOf(a); t != nil && model.HoldsReference(t, map[types.Type]bool{}) {
						model.Report(pass, a.Pos(), "VL0509", "contract.Old of a value of type %s, which holds a reference: the copy would show the new data; take the old value of a value, for example contract.Old(len(xs)) or contract.Old(a.Balance)", model.TypeName(pass, t))
					}
				}
			}
			for _, call := range pro.Ensures {
				if cond := model.EnsuresCond(call); cond != nil {
					Pure(pass, cond, "contract.Ensures")
				} else {
					model.Report(pass, call.Pos(), "VL0503", "contract.Ensures takes func() bool { return cond }, with one return statement and no other code")
				}
			}
		}
	}
	callees := &requires{pass: pass, cp: cp, m: u.M, cache: map[*types.Func]*callee{}, files: map[string]*ast.File{}, fset: token.NewFileSet()}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch name := model.Contract(info, call, cp); name {
			case "Requires", "Ensures", "Old", "Keep":
				if !placed[call] {
					form := map[string]string{"Requires": "contract.Requires(cond)", "Ensures": "defer contract.Ensures(func() bool { return cond })", "Old": "x := contract.Old(expr)", "Keep": "defer contract.Keep(recv)"}[name]
					model.Report(pass, call.Pos(), "VL0903", "contract.%s is not among the first statements of a function; a contract is written first, as %s, before any other statement", name, form)
				}
			case "":
				callees.check(call)
			}
			return true
		})
	}
	return nil, nil
}

// Pure reports what makes a contract expression impure: a call of a function other than len,
// cap, and conversions, a function literal, a receive (VL0503), and a type assertion (VL0512).
func Pure(pass *analysis.Pass, e ast.Expr, where string) {
	info := pass.TypesInfo
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			model.Report(pass, x.Pos(), "VL0503", "the condition of %s holds a function literal; a contract has no effects", where)
			return false
		case *ast.UnaryExpr:
			if x.Op == token.ARROW {
				model.Report(pass, x.Pos(), "VL0503", "the condition of %s receives from a channel; a contract has no effects", where)
			}
		case *ast.TypeAssertExpr:
			model.Report(pass, x.Pos(), "VL0512", "a type assertion is not allowed in a contract (%s)", where)
		case *ast.CallExpr:
			if tv, ok := info.Types[x.Fun]; ok && tv.IsType() {
				return true // a conversion
			}
			if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok {
				if b, ok := info.Uses[id].(*types.Builtin); ok && (b.Name() == "len" || b.Name() == "cap") {
					return true
				}
			}
			model.Report(pass, x.Pos(), "VL0503", "the condition of %s calls %s; a contract calls only len, cap, and conversions, so that it has no effects", where, types.ExprString(x.Fun))
		}
		return true
	})
}

// callee is a function with contract.Requires clauses.
type callee struct {
	params   []string
	types    []types.Type
	text     []string
	at       []string
	pkg      *types.Package
	pos      token.Pos // a position in its body, when it is in the package of the pass
	variadic bool
}

type requires struct {
	pass  *analysis.Pass
	cp    string
	m     *model.Manifest
	cache map[*types.Func]*callee
	files map[string]*ast.File
	fset  *token.FileSet
}

// of returns the Requires of a function of the module, or nil.
func (r *requires) of(fn *types.Func) *callee {
	if c, ok := r.cache[fn]; ok {
		return c
	}
	r.cache[fn] = nil
	if fn.Pkg() == nil || !r.m.InModule(fn.Pkg().Path()) || !fn.Pos().IsValid() {
		return nil
	}
	sig := fn.Type().(*types.Signature)
	if sig.Recv() != nil && types.IsInterface(sig.Recv().Type()) {
		return nil
	}
	posn := r.pass.Fset.Position(fn.Pos())
	fset := r.fset
	var file *ast.File
	same := fn.Pkg() == r.pass.Pkg
	if same {
		for _, f := range r.pass.Files {
			if r.pass.Fset.Position(f.Package).Filename == posn.Filename {
				file, fset = f, r.pass.Fset
			}
		}
	} else {
		file = r.files[posn.Filename]
		if file == nil {
			f, err := parser.ParseFile(fset, posn.Filename, nil, parser.SkipObjectResolution)
			if err != nil {
				return nil
			}
			file, r.files[posn.Filename] = f, f
		}
	}
	if file == nil {
		return nil
	}
	contractName := ""
	for _, imp := range file.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == r.cp {
			contractName = "contract"
			if imp.Name != nil {
				contractName = imp.Name.Name
			}
		}
	}
	if contractName == "" {
		return nil
	}
	var fd *ast.FuncDecl
	for _, d := range file.Decls {
		if x, ok := d.(*ast.FuncDecl); ok {
			p := fset.Position(x.Name.Pos())
			if p.Line == posn.Line && p.Column == posn.Column {
				fd = x
			}
		}
	}
	if fd == nil || fd.Body == nil {
		return nil
	}
	c := &callee{pkg: fn.Pkg(), variadic: sig.Variadic()}
	if same {
		c.pos = fd.Body.Lbrace + 1
	}
	for i := 0; i < sig.Params().Len(); i++ {
		c.params = append(c.params, sig.Params().At(i).Name())
		c.types = append(c.types, sig.Params().At(i).Type())
	}
	isContract := func(call *ast.CallExpr, names ...string) bool {
		sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
		if !ok {
			return false
		}
		if ix, ok := sel.X.(*ast.Ident); !ok || ix.Name != contractName {
			return false
		}
		for _, n := range names {
			if sel.Sel.Name == n {
				return true
			}
		}
		return false
	}
loop:
	for _, s := range fd.Body.List {
		switch s := s.(type) {
		case *ast.ExprStmt:
			call, ok := s.X.(*ast.CallExpr)
			if !ok || !isContract(call, "Requires") || len(call.Args) != 1 {
				break loop
			}
			c.text = append(c.text, exprText(fset, call.Args[0]))
			c.at = append(c.at, fset.Position(call.Pos()).String())
		case *ast.AssignStmt:
			call, ok := s.Rhs[0].(*ast.CallExpr)
			if !ok || !isContract(call, "Old") {
				break loop
			}
		case *ast.DeferStmt:
			if !isContract(s.Call, "Ensures", "Keep") {
				break loop
			}
		default:
			break loop
		}
	}
	if len(c.text) == 0 {
		return nil
	}
	r.cache[fn] = c
	return c
}

func exprText(fset *token.FileSet, e ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

// check applies the constant case of VL0501: a call whose constant arguments make a Requires of
// the callee false.
func (r *requires) check(call *ast.CallExpr) {
	info := r.pass.TypesInfo
	fn := model.CalledFunc(info, call)
	if fn == nil {
		return
	}
	cal := r.of(fn.Origin())
	if cal == nil || cal.variadic || len(call.Args) != len(cal.params) {
		return
	}
	values := map[string]string{}
	for i, a := range call.Args {
		tv, ok := info.Types[a]
		if !ok || tv.Value == nil {
			continue
		}
		pt := cal.types[i]
		b, basic := pt.Underlying().(*types.Basic)
		if !basic {
			continue
		}
		lit := tv.Value.ExactString()
		switch tv.Value.Kind() {
		case constant.String:
			lit = strconv.Quote(constant.StringVal(tv.Value))
		case constant.Float:
			lit = tv.Value.String()
			if f, exact := constant.Float64Val(tv.Value); exact {
				lit = strconv.FormatFloat(f, 'g', -1, 64)
			}
		}
		tname := b.Name()
		if n, ok := types.Unalias(pt).(*types.Named); ok && n.Obj().Pkg() == cal.pkg {
			tname = n.Obj().Name()
		}
		values[cal.params[i]] = tname + "(" + lit + ")"
	}
	if len(values) == 0 {
		return
	}
	for k, text := range cal.text {
		e, err := parser.ParseExpr(text)
		if err != nil {
			continue
		}
		known := true
		x := astutil.Apply(e, func(c *astutil.Cursor) bool {
			id, ok := c.Node().(*ast.Ident)
			if !ok {
				return true
			}
			if sel, ok := c.Parent().(*ast.SelectorExpr); ok && sel.Sel == id {
				return true
			}
			if kv, ok := c.Parent().(*ast.KeyValueExpr); ok && kv.Key == id {
				return true
			}
			if v, ok := values[id.Name]; ok {
				c.Replace(ast.NewIdent(v))
				return true
			}
			for _, p := range cal.params {
				if p == id.Name {
					known = false // a parameter whose argument is not a constant
				}
			}
			return true
		}, nil).(ast.Expr)
		if !known {
			continue
		}
		tv, err := types.Eval(r.pass.Fset, cal.pkg, cal.pos, exprText(token.NewFileSet(), x))
		if err == nil && tv.Value != nil && tv.Value.Kind() == constant.Bool && !constant.BoolVal(tv.Value) {
			model.Report(r.pass, call.Pos(), "VL0501", "this call breaks the contract of %s: contract.Requires(%s) (at %s) is false for these arguments", fn.Name(), text, cal.at[k])
		}
	}
}
