package check

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/decl"
	"github.com/robot-accomplice/vulcan/internal/gen"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// Strict mode (spec §5.4d, decision 2): config { strict: true } adds K0123, K0124, K0125, K0210,
// and K0211 in each leaf that is not in strict_exempt, and in each library. Like the network rule
// of the PoC, it reads the uses in the project's own code, not the calls inside other modules.

type strictConfig struct {
	on         bool
	exempt     map[string]bool
	interfaces map[string]bool // full names of interfaces a port may carry
	shared     map[string]bool // full type names a library var may have
	mutators   map[string]bool // more global mutators
}

// strictConf reads the strict keys of config, and reports a bad value (K0606).
func (c *Checker) strictConf() *strictConfig {
	s := &strictConfig{exempt: map[string]bool{}, interfaces: map[string]bool{}, shared: map[string]bool{}, mutators: map[string]bool{}}
	if v := c.P.Config("strict"); v != nil {
		tv, err := types.Eval(c.Fset, types.NewPackage("vulcan/strict", "strict"), token.NoPos, v.Text)
		if v.Kind != decl.ExprValue || err != nil || tv.Value == nil || tv.Value.Kind() != constant.Bool {
			c.D.Errorf(c.P.SysSrc.Pos(v.Pos), "K0606", "config strict is true or false")
		} else {
			s.on = constant.BoolVal(tv.Value)
		}
	}
	for key, set := range map[string]map[string]bool{"strict_exempt": s.exempt, "strict_interfaces": s.interfaces, "strict_shared": s.shared, "mutators": s.mutators} {
		v := c.P.Config(key)
		if v == nil {
			continue
		}
		if v.Kind != decl.ListValue {
			c.D.Errorf(c.P.SysSrc.Pos(v.Pos), "K0606", "config %s is a list of names, for example [\"x\"]", key)
			continue
		}
		for _, it := range v.List {
			name := strings.TrimSpace(it.Text)
			if u, err := strconv.Unquote(name); err == nil {
				name = u
			}
			if it.Kind != decl.ExprValue || name == "" {
				c.D.Errorf(c.P.SysSrc.Pos(it.Pos), "K0606", "config %s holds names, for example \"x\"", key)
				continue
			}
			set[name] = true
		}
	}
	for name := range s.exempt {
		if c.P.ByName[name] == nil {
			c.D.Errorf(c.P.SysSrc.Pos(c.P.Config("strict_exempt").Pos), "K0606", "strict_exempt names %s, which is not a leaf", name)
		}
	}
	return s
}

// serveFuncs are the serve functions of Appendix D; every other network function dials.
var serveFuncs = names(`
net.Listen net.ListenPacket net.ListenTCP net.ListenUDP net.ListenIP net.ListenUnix net.ListenUnixgram
net.ListenMulticastUDP net.FileListener (*net.ListenConfig).Listen (*net.ListenConfig).ListenPacket crypto/tls.Listen
net/http.ListenAndServe net/http.ListenAndServeTLS net/http.Serve net/http.ServeTLS
(*net/http.Server).ListenAndServe (*net/http.Server).ListenAndServeTLS (*net/http.Server).Serve (*net/http.Server).ServeTLS
syscall.Bind syscall.Listen syscall.Accept
`)

// globalMutators are the global mutators of Appendix D.
var globalMutators = names(`
os.Setenv os.Unsetenv os.Clearenv os.Chdir net/http.Handle net/http.HandleFunc
expvar.Publish expvar.NewInt expvar.NewFloat expvar.NewMap expvar.NewString
flag.Parse flag.Set flag.String flag.Int flag.Int64 flag.Uint flag.Uint64 flag.Bool flag.Float64 flag.Duration
flag.StringVar flag.IntVar flag.Int64Var flag.UintVar flag.Uint64Var flag.BoolVar flag.Float64Var flag.DurationVar
flag.Var flag.Func flag.BoolFunc flag.TextVar
log.SetOutput log.SetFlags log.SetPrefix log/slog.SetDefault log/slog.SetLogLoggerLevel
runtime/debug.SetGCPercent runtime/debug.SetMemoryLimit runtime/debug.SetMaxThreads runtime/debug.SetMaxStack
`)

var reflectCalls = names(`(reflect.Value).Call (reflect.Value).CallSlice (reflect.Value).Method (reflect.Value).MethodByName`)

// Strict applies strict mode, when config turns it on.
func (c *Checker) Strict() {
	s := c.strictConf()
	if !s.on {
		return
	}
	for _, pkg := range append(append([]*project.Package{}, c.P.Leaves...), c.P.Libs...) {
		if pkg.IsLeaf && s.exempt[pkg.Name] {
			continue
		}
		gp := c.Full[pkg.ImportPath]
		if gp == nil || gp.TypesInfo == nil {
			continue
		}
		if pkg.IsLeaf {
			c.portBehavior(pkg, s)
		} else {
			c.libraryState(pkg, s)
		}
		for _, f := range gp.Syntax {
			if c.Generated(f) {
				continue
			}
			c.strictUses(pkg, gp.TypesInfo, f, s)
		}
	}
}

// portBehavior: no behavior across a boundary (K0123).
func (c *Checker) portBehavior(l *project.Package, s *strictConfig) {
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
			sig := m.Type().(*types.Signature)
			var bad string
			for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
				for j := 0; j < tuple.Len() && bad == ""; j++ {
					bad = behavior(tuple.At(j).Type(), s, map[types.Type]bool{})
				}
			}
			if bad != "" {
				c.D.Errorf(l.CompSrc.Pos(findMethodPos(port, m.Name())), "K0123",
					"strict mode: port %s.%s method %s carries %s, which lets the callee run the caller's code; pass values, or list the interface in config strict_interfaces", l.Name, port.Name, m.Name(), bad)
			}
		}
	}
}

// behavior names a function, channel, or interface type that t carries, or "".
func behavior(t types.Type, s *strictConfig, seen map[types.Type]bool) string {
	if seen[t] {
		return ""
	}
	seen[t] = true
	full := types.TypeString(t, nil)
	if full == "error" || full == "context.Context" || s.interfaces[full] {
		return ""
	}
	switch u := t.Underlying().(type) {
	case *types.Signature:
		return "the function type " + full
	case *types.Chan:
		return "the channel type " + full
	case *types.Interface:
		return "the interface type " + full
	case *types.Pointer:
		return behavior(u.Elem(), s, seen)
	case *types.Slice:
		return behavior(u.Elem(), s, seen)
	case *types.Array:
		return behavior(u.Elem(), s, seen)
	case *types.Map:
		if b := behavior(u.Key(), s, seen); b != "" {
			return b
		}
		return behavior(u.Elem(), s, seen)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if b := behavior(u.Field(i).Type(), s, seen); b != "" {
				return b + " (in field " + u.Field(i).Name() + " of " + full + ")"
			}
		}
	}
	return ""
}

// libraryState: immutable library state (K0210).
func (c *Checker) libraryState(lib *project.Package, s *strictConfig) {
	gp := c.Plain[lib.ImportPath]
	if gp == nil || gp.Types == nil {
		return
	}
	scope := gp.Types.Scope()
	for _, name := range scope.Names() {
		v, ok := scope.Lookup(name).(*types.Var)
		if !ok {
			continue
		}
		full := types.TypeString(v.Type(), nil)
		if full == "error" || s.shared[full] || !holdsReference(v.Type(), map[types.Type]bool{}) {
			continue
		}
		c.D.Errorf(c.Pos(v.Pos()), "K0210", "strict mode: library variable %s has the type %s, which holds a reference, so two leaves could share and change it; use a value type, or list the type in config strict_shared", v.Name(), full)
	}
}

// strictUses: no escape hatches (K0124), entry leaves serve and gateways dial (K0125), and no
// known global mutators (K0211).
func (c *Checker) strictUses(pkg *project.Package, info *types.Info, f *ast.File, s *strictConfig) {
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path == "unsafe" || path == "syscall" || path == "C" {
			what := "imports " + path
			if path == "C" {
				what = "uses cgo"
			}
			c.D.Errorf(c.Pos(imp.Path.Pos()), "K0124", "strict mode: %s %s, which can go around the architecture rules", unitName(pkg), what)
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		fn, ok := info.Uses[id].(*types.Func)
		if !ok {
			return true
		}
		name := fn.FullName()
		switch {
		case listed(reflectCalls, name):
			c.D.Errorf(c.Pos(id.Pos()), "K0124", "strict mode: %s calls %s, which can call any code", unitName(pkg), name)
		case pkg.IsLeaf && pkg.EntryLeaf && !pkg.Gateway && listed(networkFuncs, name) && !listed(serveFuncs, name):
			c.D.Errorf(c.Pos(id.Pos()), "K0125", "strict mode: entry leaf %s dials with %s; an entry leaf only serves, so move the call to a gateway leaf and reach it through a port", pkg.Name, name)
		case listed(globalMutators, name) || s.mutators[name]:
			c.D.Errorf(c.Pos(id.Pos()), "K0211", "strict mode: %s calls %s, which changes state that every leaf shares; set it once in main, or pass the value through a port", unitName(pkg), name)
		}
		return true
	})
}

func unitName(pkg *project.Package) string {
	if pkg.IsLeaf {
		return "leaf " + pkg.Name
	}
	return "library " + pkg.GoName
}
