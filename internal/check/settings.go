package check

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/decl"
	"github.com/robot-accomplice/vulcan/internal/gen"
)

// This file checks the values of system.vulcan: settings, flags, and config (§5.3, App. B.2).

func normalSettings(s *decl.Settings) string {
	if s == nil {
		return ""
	}
	var parts []string
	for _, it := range s.Items {
		v := it.Value
		text := v.Text
		if v.Kind == decl.FlagValue {
			text = "flag:" + v.Flag.Name + ":" + defText(v.Flag.Default)
		}
		parts = append(parts, it.Key+"="+strings.Join(strings.Fields(text), ""))
	}
	sort.Strings(parts)
	return "{" + strings.Join(parts, ",") + "}"
}

// settings checks the values of a settings struct, and collects its flags (K0602, K0603, K0605,
// K0606). Each exported field needs a value or a flag.
func (c *Checker) settings(s *decl.Settings, t types.Type, owner string, plan *Plan) {
	st := t.Underlying().(*types.Struct)
	fields := map[string]int{}
	for i := 0; i < st.NumFields(); i++ {
		fields[st.Field(i).Name()] = i
	}
	pos := c.P.SysSrc.Pos
	given := map[string]bool{}
	for _, it := range s.Items {
		i, ok := fields[it.Key]
		if !ok || !st.Field(i).Exported() {
			c.D.Errorf(pos(it.KeyPos), "K0603", "%s: settings type %s has no exported field %s", owner, types.TypeString(t, qualifier), it.Key)
			continue
		}
		if given[it.Key] {
			c.D.Errorf(pos(it.KeyPos), "K0603", "%s: setting %s is given twice", owner, it.Key)
			continue
		}
		given[it.Key] = true
		f := st.Field(i)
		secret := isSecret(st.Tag(i))
		where := owner + " setting " + it.Key
		switch it.Value.Kind {
		case decl.FlagValue:
			c.flag(it.Value, f.Type(), where, secret, plan)
		case decl.ExprValue:
			if secret {
				c.D.Errorf(pos(it.Value.Pos), "K0605", "%s is a secret (tag vulcan:\"secret\"); give it a flag, not a literal", where)
				continue
			}
			c.constant(it.Value, f.Type(), where)
		default:
			c.P.Deferred(pos(it.Value.Pos), "a list or a named settings value", "§5.5.1")
		}
	}
	var missing []string
	for i := 0; i < st.NumFields(); i++ {
		if f := st.Field(i); f.Exported() && !given[f.Name()] {
			missing = append(missing, f.Name())
		}
	}
	if len(missing) > 0 {
		c.D.Errorf(pos(s.Pos), "K0603", "%s: settings type %s has no value for %s; give each exported field a value or a flag in system.vulcan", owner, types.TypeString(t, qualifier), strings.Join(missing, ", "))
	}
}

func isSecret(tag string) bool {
	for _, opt := range strings.Split(reflect.StructTag(tag).Get("vulcan"), ",") {
		if strings.TrimSpace(opt) == "secret" {
			return true
		}
	}
	return false
}

// constant checks that a value is a Go constant that fits the field's type, and stores its Go
// literal in v.Lit (K0606). The value is evaluated in a scope that holds only the package time
// and the libraries that system.vulcan imports, so a value cannot name or run code (§5.4 rule 2).
func (c *Checker) constant(v *decl.Value, t types.Type, where string) bool {
	pos := c.P.SysSrc.Pos(v.Pos)
	if bad := codeIn(v.Expr); bad != "" {
		c.D.Errorf(pos, "K0606", "%s: %s in system.vulcan; a value is a constant, because system.vulcan holds no code", where, bad)
		return false
	}
	tv, err := types.Eval(c.Fset, c.valueScope(), token.NoPos, v.Text)
	if err != nil {
		c.D.Errorf(pos, "K0606", "%s: %s", where, evalMessage(err))
		return false
	}
	if tv.Value == nil {
		c.D.Errorf(pos, "K0606", "%s: %s is not a constant; a setting takes a constant or a flag", where, v.Text)
		return false
	}
	if !c.fits(tv, t) {
		hint := ""
		if isDuration(t) {
			hint = "; write a duration as a Go constant, for example 5 * time.Second"
		}
		c.D.Errorf(pos, "K0606", "%s: %s (%s) does not fit the field's type %s%s", where, v.Text, typeName(tv.Type), types.TypeString(t, qualifier), hint)
		return false
	}
	v.Lit = literal(tv.Value)
	return true
}

func typeName(t types.Type) string {
	if b, ok := t.(*types.Basic); ok && b.Info()&types.IsUntyped != 0 {
		return strings.TrimPrefix(b.Name(), "untyped ") + " constant"
	}
	return types.TypeString(t, qualifier)
}

func evalMessage(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, ": "); i >= 0 && strings.Contains(msg[:i], ":") {
		msg = msg[i+2:]
	}
	if strings.HasPrefix(msg, "undefined: ") {
		msg += " (a value can use only constants, the package time, and the libraries that system.vulcan imports)"
	}
	return msg
}

func isDuration(t types.Type) bool {
	n, ok := types.Unalias(t).(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == "time" && n.Obj().Name() == "Duration"
}

// codeIn names the first construct of e that is not allowed in a constant value.
func codeIn(e ast.Expr) string {
	bad := ""
	ast.Inspect(e, func(n ast.Node) bool {
		if bad != "" {
			return false
		}
		switch n := n.(type) {
		case *ast.FuncLit:
			bad = "a function literal"
		case *ast.CompositeLit:
			bad = "a composite literal"
		case *ast.IndexExpr, *ast.IndexListExpr, *ast.SliceExpr:
			bad = "an index or slice expression"
		case *ast.StarExpr:
			bad = "a pointer expression"
		case *ast.TypeAssertExpr:
			bad = "a type assertion"
		case *ast.UnaryExpr:
			if n.Op == token.AND || n.Op == token.ARROW {
				bad = "an address or a receive"
			}
		}
		return true
	})
	return bad
}

// fits reports whether a constant can be assigned to a field of type t.
func (c *Checker) fits(tv types.TypeAndValue, t types.Type) bool {
	b, ok := tv.Type.(*types.Basic)
	if !ok || b.Info()&types.IsUntyped == 0 {
		return types.AssignableTo(tv.Type, t)
	}
	fb, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	switch {
	case b.Info()&types.IsString != 0:
		if fb.Info()&types.IsString == 0 {
			return false
		}
	case b.Info()&types.IsBoolean != 0:
		if fb.Info()&types.IsBoolean == 0 {
			return false
		}
	case b.Info()&types.IsNumeric != 0:
		if fb.Info()&types.IsNumeric == 0 {
			return false
		}
	default:
		return false
	}
	// Representable: the conversion to the field's basic type is a valid constant expression.
	_, err := types.Eval(c.Fset, types.NewPackage("vulcan/fit", "fit"), token.NoPos, fb.Name()+"("+literal(tv.Value)+")")
	return err == nil
}

// literal writes a constant as Go source.
func literal(v constant.Value) string {
	switch v.Kind() {
	case constant.String:
		return strconv.Quote(constant.StringVal(v))
	case constant.Float:
		if f, exact := constant.Float64Val(v); exact || !math.IsInf(f, 0) {
			return strconv.FormatFloat(f, 'g', -1, 64)
		}
	}
	return v.ExactString()
}

// valueScope is the scope of a value in system.vulcan: time and the imported libraries.
func (c *Checker) valueScope() *types.Package {
	if c.scope != nil {
		return c.scope
	}
	c.scope = types.NewPackage("vulcan/system", "system")
	main := c.Plain[c.P.Module+"/"+gen.MainDir]
	if main == nil {
		return c.scope
	}
	add := func(name, path string) {
		if imp := main.Imports[path]; imp != nil && imp.Types != nil {
			c.scope.Scope().Insert(types.NewPkgName(token.NoPos, c.scope, name, imp.Types))
		}
	}
	add("time", "time")
	for _, imp := range c.P.System.Imports {
		if lib := c.P.ByPath[imp.Path]; lib != nil && !lib.IsLeaf {
			name := imp.Name
			if name == "" {
				name = lib.GoName
			}
			add(name, imp.Path)
		}
	}
	return c.scope
}

// systemImports applies §5.4 rule 2 to the imports of system.vulcan: only libraries, so that a
// value can name their constants (K0110).
func (c *Checker) systemImports() {
	for _, imp := range c.P.System.Imports {
		p := c.P.ByPath[imp.Path]
		switch {
		case p != nil && p.IsLeaf:
			c.D.Errorf(c.P.SysSrc.Pos(imp.Pos), "K0110", "system.vulcan imports leaf %s; system.vulcan names leaves only in wires, and imports only libraries", p.Name)
		case p == nil && imp.Path != "time":
			c.D.Errorf(c.P.SysSrc.Pos(imp.Pos), "K0110", "system.vulcan imports %s; it imports only libraries of the project (the package time needs no import)", imp.Path)
		}
	}
}

func (c *Checker) flag(v *decl.Value, t types.Type, where string, secret bool, plan *Plan) {
	getter, goType := flagGetter(t)
	pos := c.P.SysSrc.Pos(v.Flag.Pos)
	if getter == "" {
		c.D.Errorf(pos, "K0602", "flag %q fills %s of type %s; a flag gives a string, int, int64, float64, bool, or time.Duration", v.Flag.Name, where, types.TypeString(t, qualifier))
		return
	}
	if v.Flag.Max != nil {
		c.P.Deferred(c.P.SysSrc.Pos(v.Flag.Max.Pos), "max on a flag", "§5.5.1")
	}
	if d := v.Flag.Default; d != nil {
		if secret {
			c.D.Errorf(c.P.SysSrc.Pos(d.Pos), "K0605", "%s is a secret (tag vulcan:\"secret\"); its flag has no default, so that no secret is in source", where)
		} else {
			c.constant(d, t, where+", flag "+strconv.Quote(v.Flag.Name)+" default")
		}
	}
	env := strings.NewReplacer("-", "_", ".", "_").Replace(strings.ToUpper(v.Flag.Name))
	for _, f := range plan.Flags {
		fenv := strings.NewReplacer("-", "_", ".", "_").Replace(strings.ToUpper(f.Name))
		if f.Name == v.Flag.Name {
			if f.Getter != getter || defText(f.Default) != defText(v.Flag.Default) {
				c.D.Errorf(pos, "K0602", "flag %q is used with two types or two defaults", v.Flag.Name)
			}
			return
		}
		if fenv == env {
			c.D.Errorf(pos, "K0602", "flags %q and %q both read the environment variable %s", f.Name, v.Flag.Name, env)
			return
		}
	}
	plan.Flags = append(plan.Flags, &Flag{Name: v.Flag.Name, Getter: getter, GoType: goType, Default: v.Flag.Default})
}

func defText(v *decl.Value) string {
	if v == nil {
		return ""
	}
	return v.Text
}

func flagGetter(t types.Type) (string, string) {
	if n, ok := types.Unalias(t).(*types.Named); ok && n.Obj().Pkg() != nil && n.Obj().Pkg().Path() == "time" && n.Obj().Name() == "Duration" {
		return "Duration", "time.Duration"
	}
	b, ok := t.(*types.Basic)
	if !ok {
		return "", ""
	}
	switch b.Kind() {
	case types.String:
		return "String", "string"
	case types.Int:
		return "Int", "int"
	case types.Int64:
		return "Int64", "int64"
	case types.Float64:
		return "Float64", "float64"
	case types.Bool:
		return "Bool", "bool"
	}
	return "", ""
}

// config checks the config values that the PoC uses.
func (c *Checker) config() {
	for _, key := range []string{"shutdown_timeout", "context_budget"} {
		v := c.P.Config(key)
		if v == nil {
			continue
		}
		if v.Kind != decl.ExprValue {
			c.D.Errorf(c.P.SysSrc.Pos(v.Pos), "K0606", "config %s takes a constant, not a flag or a list", key)
			continue
		}
		var t types.Type = types.Typ[types.Int]
		if key == "shutdown_timeout" {
			if tp := c.valueScope().Scope().Lookup("time"); tp != nil {
				t = tp.(*types.PkgName).Imported().Scope().Lookup("Duration").Type()
			}
		}
		c.constant(v, t, "config "+key)
	}
	if v := c.P.Config("on_contract_failure"); v != nil {
		mode := strings.Trim(v.Text, `"`)
		if v.Kind != decl.ExprValue || mode != "panic" && mode != "exit" {
			c.D.Errorf(c.P.SysSrc.Pos(v.Pos), "K0606", "config on_contract_failure is panic or exit (§7.4)")
		} else {
			v.Lit = mode
		}
	}
}
