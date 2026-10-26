package check

import (
	"fmt"
	"go/types"
	"reflect"
	"sort"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/decl"
	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/gen"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// Instance is one provider object that main constructs.
type Instance struct {
	ID         int
	Leaf       *project.Package
	Name       string // the constructor or the type
	IsType     bool   // a wire that names a type provides new(T)
	ReturnsErr bool
	Settings   *decl.Settings
	SetType    types.Type // the constructor's settings parameter, or nil
	Result     types.Type
}

// Bind binds one port to an instance.
type Bind struct {
	Need *project.Package
	Port string
	Inst *Instance
	Wire *decl.Wire
}

// LeafFunc is a setup or startup function of a leaf.
type LeafFunc struct {
	Leaf    *project.Package
	Kind    string // "setup" or "startup"
	SetType types.Type
}

// Flag is one flag of the container.
type Flag struct {
	Name    string
	Getter  string // the rt.Flags method: String, Int, Int64, Float64, Bool, Duration
	GoType  string // the Go type of the default
	Default *decl.Value
}

// Plan is what the generated main does.
type Plan struct {
	Order     []*project.Package // leaves, providers first
	Instances []*Instance
	Binds     []Bind
	Funcs     []LeafFunc
	Flags     []*Flag
}

// Wiring checks the wires of the container and returns the plan of main.
func (c *Checker) Wiring() *Plan {
	plan := &Plan{}
	ct := c.P.Container
	if ct == nil {
		return plan
	}
	sysPos := c.P.SysSrc.Pos
	type wireKey struct{ leaf, port string }
	wired := map[wireKey]*decl.Wire{}
	instKey := map[string]*Instance{}
	edges := map[*project.Package][]*project.Package{}
	edgeWire := map[[2]*project.Package]*decl.Wire{}
	groups := map[string]bool{}
	for _, g := range ct.Groups {
		groups[g.Name] = true
	}
	c.badPorts = map[string]bool{}
	for _, le := range ct.Leaves {
		if l := c.P.ByName[le.Name]; l != nil && l.Container == ct.Name {
			c.boundaryTypes(l)
		}
	}
	for _, w := range ct.Wires {
		need := c.P.ByName[w.Need.Leaf]
		if need == nil && groups[w.Need.Leaf] {
			continue // K0113
		}
		if need == nil {
			c.D.Errorf(sysPos(w.Need.Pos), "K0107", "the wire names leaf %s, which container %s does not list", w.Need.Leaf, ct.Name)
			continue
		}
		port := findPort(need, w.Need.Name)
		if port == nil {
			c.D.Errorf(sysPos(w.Need.NamePos), "K0107", "leaf %s needs no port %s (its ports: %s)", need.Name, w.Need.Name, portNames(need))
			continue
		}
		k := wireKey{need.Name, port.Name}
		if prev := wired[k]; prev != nil {
			pp := sysPos(prev.Pos)
			c.D.Errorf(sysPos(w.Pos), "K0105", "port %s.%s is wired twice; the first wire is at line %d", need.Name, port.Name, pp.Line)
			continue
		}
		wired[k] = w
		sup := c.P.ByName[w.Supply.Leaf]
		if sup == nil && groups[w.Supply.Leaf] {
			continue // K0113
		}
		if sup == nil {
			c.D.Errorf(sysPos(w.Supply.Pos), "K0107", "the wire names leaf %s, which container %s does not list", w.Supply.Leaf, ct.Name)
			continue
		}
		inst := c.instance(sup, w, plan, instKey)
		if inst == nil {
			continue
		}
		if !c.badPorts[need.Name+"."+port.Name] {
			c.methodSet(need, port, inst, w)
		}
		plan.Binds = append(plan.Binds, Bind{Need: need, Port: port.Name, Inst: inst, Wire: w})
		edges[need] = append(edges[need], sup)
		edgeWire[[2]*project.Package{need, sup}] = w
	}
	// Each port is wired once (rule 4).
	for _, le := range ct.Leaves {
		l := c.P.ByName[le.Name]
		if l == nil || l.Comp == nil || l.Container != ct.Name {
			continue
		}
		for _, port := range l.Comp.Ports {
			if wired[wireKey{l.Name, port.Name}] == nil {
				c.D.Errorf(l.CompSrc.Pos(port.Pos), "K0104", "port %s of %s is not wired; add a wire \"%s.%s: Leaf.Constructor\" to container %s in system.vulcan", port.Name, l.Name, l.Name, port.Name, ct.Name)
			}
		}
	}
	plan.Order = c.order(ct, edges, edgeWire)
	c.leafFuncs(ct, plan)
	return plan
}

func findPort(l *project.Package, name string) *decl.Port {
	if l.Comp == nil {
		return nil
	}
	for _, p := range l.Comp.Ports {
		if p.Name == name {
			return p
		}
	}
	return nil
}

func portNames(l *project.Package) string {
	if l.Comp == nil || len(l.Comp.Ports) == 0 {
		return "none"
	}
	var out []string
	for _, p := range l.Comp.Ports {
		out = append(out, p.Name)
	}
	return strings.Join(out, ", ")
}

// instance resolves the supplier of a wire (K0107, K0604).
func (c *Checker) instance(sup *project.Package, w *decl.Wire, plan *Plan, seen map[string]*Instance) *Instance {
	sysPos := c.P.SysSrc.Pos
	gp := c.Plain[sup.ImportPath]
	if gp == nil || gp.Types == nil {
		return nil
	}
	name := w.Supply.Name
	obj := gp.Types.Scope().Lookup(name)
	if obj == nil || !obj.Exported() {
		what := "does not exist"
		if obj != nil {
			what = "is not exported"
		}
		c.D.Errorf(sysPos(w.Supply.NamePos), "K0107", "%s.%s %s; a wire names an exported constructor or type of the leaf", sup.Name, name, what)
		return nil
	}
	settingsText := ""
	if w.Settings != nil {
		settingsText = string(c.P.SysSrc.Src[w.Settings.Pos:settingsEnd(w.Settings, c.P.SysSrc.Src)])
	}
	key := sup.Name + "." + name + settingsText
	if inst := seen[key]; inst != nil {
		return inst
	}
	inst := &Instance{ID: len(plan.Instances), Leaf: sup, Name: name, Settings: w.Settings}
	switch o := obj.(type) {
	case *types.TypeName:
		if w.Settings != nil {
			c.D.Errorf(sysPos(w.Settings.Pos), "K0604", "the wire names type %s.%s, which provides new(%s) and takes no settings", sup.Name, name, name)
		}
		if types.IsInterface(o.Type()) {
			c.D.Errorf(sysPos(w.Supply.NamePos), "K0604", "%s.%s is an interface; a wire names a concrete type or a constructor", sup.Name, name)
			return nil
		}
		inst.IsType = true
		inst.Result = types.NewPointer(o.Type())
	case *types.Func:
		sig := o.Type().(*types.Signature)
		ok := true
		if sig.Params().Len() > 1 || sig.Variadic() {
			ok = false
		} else if sig.Params().Len() == 1 {
			pt := sig.Params().At(0).Type()
			if _, isStruct := pt.Underlying().(*types.Struct); !isStruct {
				ok = false
			} else {
				inst.SetType = pt
			}
		}
		res := sig.Results()
		switch {
		case res.Len() == 1:
		case res.Len() == 2 && isError(res.At(1).Type()):
			inst.ReturnsErr = true
		default:
			ok = false
		}
		if ok {
			inst.Result = res.At(0).Type()
			if !exportedConcrete(inst.Result) {
				ok = false
			}
		}
		if !ok {
			c.D.Errorf(sysPos(w.Supply.NamePos), "K0604", "constructor %s.%s has the signature %s; a constructor takes nothing or one settings struct, and returns T or (T, error), where T is an exported concrete type or a pointer to one", sup.Name, name, types.TypeString(sig, qualifier))
			return nil
		}
		if inst.SetType != nil && w.Settings != nil {
			c.settings(w.Settings, inst.SetType, fmt.Sprintf("constructor %s.%s", sup.Name, name), plan)
		} else if inst.SetType == nil && w.Settings != nil {
			c.D.Errorf(sysPos(w.Settings.Pos), "K0604", "constructor %s.%s takes no settings", sup.Name, name)
		}
	default:
		c.D.Errorf(sysPos(w.Supply.NamePos), "K0107", "%s.%s is not a constructor or a type", sup.Name, name)
		return nil
	}
	seen[key] = inst
	plan.Instances = append(plan.Instances, inst)
	return inst
}

func settingsEnd(s *decl.Settings, src []byte) int {
	depth := 0
	for i := s.Pos; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(src)
}

func isError(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func exportedConcrete(t types.Type) bool {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	n, ok := types.Unalias(t).(*types.Named)
	if !ok || !n.Obj().Exported() {
		return false
	}
	return !types.IsInterface(n)
}

func qualifier(p *types.Package) string { return p.Name() }

// methodSet applies rule 5: the supplier's method set matches the port (K0108).
func (c *Checker) methodSet(need *project.Package, port *decl.Port, inst *Instance, w *decl.Wire) {
	gp := c.Plain[need.ImportPath]
	if gp == nil || gp.Types == nil {
		return
	}
	tn, ok := gp.Types.Scope().Lookup(gen.PortType(port.Name)).(*types.TypeName)
	if !ok {
		return
	}
	iface, ok := tn.Type().Underlying().(*types.Interface)
	if !ok {
		return
	}
	if types.Implements(inst.Result, iface) {
		return
	}
	m, wrong := types.MissingMethod(inst.Result, iface, true)
	if m == nil {
		return
	}
	got := types.TypeString(inst.Result, qualifier)
	want := m.Name() + strings.TrimPrefix(types.TypeString(m.Type(), qualifier), "func")
	msg := fmt.Sprintf("%s does not satisfy port %s.%s: it has no method %s", got, need.Name, port.Name, want)
	if wrong {
		if hm, _, _ := types.LookupFieldOrMethod(inst.Result, true, m.Pkg(), m.Name()); hm != nil {
			if fn, ok := hm.(*types.Func); ok {
				have := fn.Name() + strings.TrimPrefix(types.TypeString(fn.Type(), qualifier), "func")
				msg = fmt.Sprintf("%s does not satisfy port %s.%s: method %s has the signature %s; the port wants %s", got, need.Name, port.Name, m.Name(), have, want)
				if _, isPtr := inst.Result.(*types.Pointer); !isPtr && types.Implements(types.NewPointer(inst.Result), iface) {
					msg = fmt.Sprintf("%s does not satisfy port %s.%s: method %s has a pointer receiver, so only *%s has it; return a pointer from the constructor", got, need.Name, port.Name, m.Name(), got)
				}
			}
		}
	}
	c.D.Errorf(c.P.SysSrc.Pos(w.Supply.NamePos), "K0108", "%s", msg)
}

// boundaryTypes applies rule 9: a type in a port signature is predeclared, from the standard
// library, or from a library (K0112).
func (c *Checker) boundaryTypes(l *project.Package) {
	gp := c.Plain[l.ImportPath]
	if gp == nil || gp.Types == nil || l.Comp == nil {
		return
	}
	for _, port := range l.Comp.Ports {
		tn, ok := gp.Types.Scope().Lookup(gen.PortType(port.Name)).(*types.TypeName)
		if !ok {
			continue
		}
		iface, ok := tn.Type().Underlying().(*types.Interface)
		if !ok {
			continue
		}
		for i := 0; i < iface.NumExplicitMethods(); i++ {
			m := iface.ExplicitMethod(i)
			var bad []string
			visitNamed(m.Type(), map[types.Type]bool{}, func(n *types.Named) {
				pkg := n.Obj().Pkg()
				if pkg == nil {
					return
				}
				if p := c.P.ByPath[pkg.Path()]; p != nil {
					// A type of another leaf needs an import of that leaf, which K0110 reports.
					if p == l {
						bad = append(bad, fmt.Sprintf("%s.%s, which is private to leaf %s", pkg.Name(), n.Obj().Name(), p.Name))
					}
					return
				}
				if !isStd(pkg.Path()) {
					bad = append(bad, fmt.Sprintf("%s.%s, which is from a module outside the project", pkg.Name(), n.Obj().Name()))
				}
			})
			if len(bad) > 0 {
				c.badPorts[l.Name+"."+port.Name] = true
			}
			for _, b := range bad {
				d := diag.Diagnostic{Severity: diag.Error, Code: "K0112", Pos: l.CompSrc.Pos(findMethodPos(port, m.Name())),
					Message: fmt.Sprintf("port %s.%s method %s uses %s; a type that crosses a boundary is predeclared, from the standard library, or from a library", l.Name, port.Name, m.Name(), b),
					Fix:     "move the type to a library (lib/<name>/library.vulcan) and import it in both leaves"}
				c.D.Add(d)
			}
		}
	}
}

func findMethodPos(port *decl.Port, name string) int {
	for _, m := range port.Methods {
		if m.Name == name {
			return m.Pos
		}
	}
	return port.Pos
}

func isStd(path string) bool {
	first, _, _ := strings.Cut(path, "/")
	return !strings.Contains(first, ".")
}

func visitNamed(t types.Type, seen map[types.Type]bool, f func(*types.Named)) {
	if t == nil || seen[t] {
		return
	}
	seen[t] = true
	switch t := t.(type) {
	case *types.Alias:
		visitNamed(types.Unalias(t), seen, f)
	case *types.Named:
		f(t)
		if args := t.TypeArgs(); args != nil {
			for i := 0; i < args.Len(); i++ {
				visitNamed(args.At(i), seen, f)
			}
		}
	case *types.Pointer:
		visitNamed(t.Elem(), seen, f)
	case *types.Slice:
		visitNamed(t.Elem(), seen, f)
	case *types.Array:
		visitNamed(t.Elem(), seen, f)
	case *types.Map:
		visitNamed(t.Key(), seen, f)
		visitNamed(t.Elem(), seen, f)
	case *types.Chan:
		visitNamed(t.Elem(), seen, f)
	case *types.Signature:
		for i := 0; i < t.Params().Len(); i++ {
			visitNamed(t.Params().At(i).Type(), seen, f)
		}
		for i := 0; i < t.Results().Len(); i++ {
			visitNamed(t.Results().At(i).Type(), seen, f)
		}
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			visitNamed(t.Field(i).Type(), seen, f)
		}
	case *types.Interface:
		for i := 0; i < t.NumMethods(); i++ {
			visitNamed(t.Method(i).Type(), seen, f)
		}
	}
}

// order sorts the leaves so that each provider comes before the leaves that need it, and
// reports cycles (rule 8, K0111).
func (c *Checker) order(ct *decl.Container, edges map[*project.Package][]*project.Package, edgeWire map[[2]*project.Package]*decl.Wire) []*project.Package {
	var out []*project.Package
	state := map[*project.Package]int{} // 1 visiting, 2 done
	var stack []*project.Package
	reported := map[string]bool{}
	var visit func(l *project.Package)
	visit = func(l *project.Package) {
		switch state[l] {
		case 2:
			return
		case 1:
			i := len(stack) - 1
			for i >= 0 && stack[i] != l {
				i--
			}
			cyc := append(append([]*project.Package{}, stack[i:]...), l)
			var names []string
			for _, n := range cyc {
				names = append(names, n.Name)
			}
			key := canonical(names[:len(names)-1])
			if !reported[key] {
				reported[key] = true
				w := edgeWire[[2]*project.Package{cyc[0], cyc[1]}]
				c.D.Errorf(c.P.SysSrc.Pos(w.Pos), "K0111", "the wires form a cycle: %s; break it with a new leaf that both sides need, or by moving code", strings.Join(names, " -> "))
			}
			return
		}
		state[l] = 1
		stack = append(stack, l)
		for _, s := range edges[l] {
			visit(s)
		}
		stack = stack[:len(stack)-1]
		state[l] = 2
		out = append(out, l)
	}
	for _, le := range ct.Leaves {
		if l := c.P.ByName[le.Name]; l != nil {
			visit(l)
		}
	}
	return out
}

func canonical(names []string) string {
	s := append([]string{}, names...)
	sort.Strings(s)
	return strings.Join(s, ",")
}

// leafFuncs finds setup and startup, and checks their signatures and settings (K0601, K0603).
func (c *Checker) leafFuncs(ct *decl.Container, plan *Plan) {
	for _, l := range plan.Order {
		gp := c.Plain[l.ImportPath]
		if gp == nil || gp.Types == nil {
			continue
		}
		var settingsType types.Type
		for _, kind := range []string{"setup", "startup"} {
			fn, ok := gp.Types.Scope().Lookup(kind).(*types.Func)
			if !ok {
				continue
			}
			sig := fn.Type().(*types.Signature)
			ok = sig.Results().Len() == 1 && isError(sig.Results().At(0).Type()) && sig.Params().Len() >= 1 && sig.Params().Len() <= 2 &&
				types.TypeString(sig.Params().At(0).Type(), nil) == "context.Context" && sig.TypeParams() == nil
			var st types.Type
			if ok && sig.Params().Len() == 2 {
				st = sig.Params().At(1).Type()
				n, isNamed := types.Unalias(st).(*types.Named)
				if _, isStruct := st.Underlying().(*types.Struct); !isStruct || !isNamed || n.Obj().Pkg() != gp.Types {
					ok = false
				}
			}
			if !ok {
				c.D.Errorf(c.Pos(fn.Pos()), "K0601", "%s has the signature %s; it must be func(ctx context.Context) error, or func(ctx context.Context, s Settings) error with a struct type of the leaf", kind, types.TypeString(sig, qualifier))
				continue
			}
			if st != nil {
				if settingsType != nil && !types.Identical(settingsType, st) {
					c.D.Errorf(c.Pos(fn.Pos()), "K0601", "setup and startup of %s take different settings types", l.Name)
				}
				settingsType = st
			}
			plan.Funcs = append(plan.Funcs, LeafFunc{Leaf: l, Kind: kind, SetType: st})
		}
		pos := c.P.SysSrc.Pos(l.Entry.Pos)
		switch {
		case settingsType == nil && l.Entry.Settings != nil:
			c.D.Errorf(pos, "K0603", "leaf %s has settings in system.vulcan, but its setup and startup take none", l.Name)
		case settingsType != nil:
			if l.Entry.Settings == nil {
				l.Entry.Settings = &decl.Settings{Pos: l.Entry.Pos}
			}
			c.settings(l.Entry.Settings, settingsType, "leaf "+l.Name, plan)
		}
	}
}

// settings checks the values of a settings struct, and collects its flags (K0602, K0603, K0605).
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
		given[it.Key] = true
		f := st.Field(i)
		secret := reflect.StructTag(st.Tag(i)).Get("vulcan") == "secret"
		switch it.Value.Kind {
		case decl.FlagValue:
			c.flag(it.Value, f.Type(), owner+" setting "+it.Key, plan)
		case decl.ExprValue:
			if secret {
				c.D.Errorf(pos(it.Value.Pos), "K0605", "%s: setting %s is a secret (tag vulcan:\"secret\"); give it a flag, not a literal", owner, it.Key)
			}
		default:
			c.P.Deferred(pos(it.Value.Pos), "a list or a named settings value", "§5.5.1")
		}
	}
}

func (c *Checker) flag(v *decl.Value, t types.Type, where string, plan *Plan) {
	getter, goType := flagGetter(t)
	pos := c.P.SysSrc.Pos(v.Flag.Pos)
	if getter == "" {
		c.D.Errorf(pos, "K0602", "flag %q fills %s of type %s; a flag gives a string, int, int64, float64, bool, or time.Duration", v.Flag.Name, where, types.TypeString(t, qualifier))
		return
	}
	if v.Flag.Max != nil {
		c.P.Deferred(c.P.SysSrc.Pos(v.Flag.Max.Pos), "max on a flag", "§5.5.1")
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
