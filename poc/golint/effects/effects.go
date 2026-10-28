// Package effects is analyzer A5 of vulcan-lint: which parts may reach the network, choose a
// resource by name, call code by reflection, or change process-wide state.
package effects

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A5.
var Analyzer = &analysis.Analyzer{
	Name: "effects",
	Doc:  "check the effects of each part: only gateways and entry leaves use the network, a library opens nothing by name, and in strict mode an entry leaf only serves, no reflect calls, and no global mutators outside main (VL0124, VL0125, VL0203, VL0205, VL0211)",
	Run:  run,
}

// The lists of the PoC (internal/check/network.go and strict.go, spec Appendix D), by
// types.Func.FullName. A name that ends in "*" is a prefix.
var networkFuncs = names(`
net.Dial net.DialTimeout net.DialTCP net.DialUDP net.DialIP net.DialUnix net.Listen net.ListenPacket
net.ListenTCP net.ListenUDP net.ListenIP net.ListenUnix net.ListenUnixgram net.ListenMulticastUDP
net.FileConn net.FileListener net.FilePacketConn (*net.Dialer).Dial (*net.Dialer).DialContext
(*net.ListenConfig).Listen (*net.ListenConfig).ListenPacket net.Lookup* (*net.Resolver).Lookup*
crypto/tls.Dial crypto/tls.DialWithDialer crypto/tls.Listen (*crypto/tls.Dialer).Dial (*crypto/tls.Dialer).DialContext
net/http.Get net/http.Head net/http.Post net/http.PostForm (*net/http.Client).Do (*net/http.Transport).RoundTrip
(*net/http.Client).Get (*net/http.Client).Head (*net/http.Client).Post (*net/http.Client).PostForm
net/http.ListenAndServe net/http.ListenAndServeTLS net/http.Serve net/http.ServeTLS
(*net/http.Server).ListenAndServe (*net/http.Server).ListenAndServeTLS (*net/http.Server).Serve (*net/http.Server).ServeTLS
net/smtp.Dial net/smtp.SendMail net/rpc.Dial net/rpc.DialHTTP net/rpc.DialHTTPPath
database/sql.Open database/sql.OpenDB os/exec.Command os/exec.CommandContext os.StartProcess
syscall.Socket syscall.Connect syscall.Bind syscall.Listen syscall.Accept syscall.Exec syscall.ForkExec syscall.StartProcess
(net/http.RoundTripper).RoundTrip
`)

// networkValues are the values of the standard library that reach the network when they are
// used, so that a call through an interface does not hide the dial (desk test, hole H3). The
// PoC does not list them.
var networkValues = names(`net/http.DefaultClient net/http.DefaultTransport net.DefaultResolver`)

// networkTypes are the standard types whose values dial; building one is a use of the network.
var networkTypes = names(`net/http.Client net/http.Transport net.Dialer crypto/tls.Dialer net.Resolver`)

var acquiringFuncs = names(`
os.Open os.OpenFile os.Create os.CreateTemp os.MkdirTemp os.OpenRoot os.OpenInRoot os.NewFile os.Pipe os.DirFS
os.ReadFile os.WriteFile os.ReadDir os.Remove os.RemoveAll os.Rename os.Mkdir os.MkdirAll os.Truncate os.Chmod
os.Chown os.Lchown os.Chtimes os.Link os.Symlink os.Readlink os.Stat os.Lstat os.Chdir os.CopyFS
io/ioutil.ReadFile io/ioutil.WriteFile io/ioutil.ReadDir io/ioutil.TempFile io/ioutil.TempDir
path/filepath.Walk path/filepath.WalkDir path/filepath.Glob path/filepath.EvalSymlinks plugin.Open
syscall.Open syscall.Openat syscall.Creat syscall.Mkdir syscall.Mkdirat syscall.Rmdir syscall.Unlink
syscall.Unlinkat syscall.Rename syscall.Renameat syscall.Link syscall.Symlink syscall.Chdir syscall.Chroot syscall.Mount
`)

var serveFuncs = names(`
net.Listen net.ListenPacket net.ListenTCP net.ListenUDP net.ListenIP net.ListenUnix net.ListenUnixgram
net.ListenMulticastUDP net.FileListener (*net.ListenConfig).Listen (*net.ListenConfig).ListenPacket crypto/tls.Listen
net/http.ListenAndServe net/http.ListenAndServeTLS net/http.Serve net/http.ServeTLS
(*net/http.Server).ListenAndServe (*net/http.Server).ListenAndServeTLS (*net/http.Server).Serve (*net/http.Server).ServeTLS
syscall.Bind syscall.Listen syscall.Accept
`)

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

func names(s string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.Fields(s) {
		out[f] = true
	}
	return out
}

func listed(set map[string]bool, name string) bool {
	if set[name] {
		return true
	}
	for k := range set {
		if strings.HasSuffix(k, "*") && strings.HasPrefix(name, strings.TrimSuffix(k, "*")) {
			return true
		}
	}
	return false
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Manifest || u.Part == nil {
		return nil, nil
	}
	m, p, info := u.M, u.Part, pass.TypesInfo
	strict := m.StrictFor(p)
	mayNet := p.Kind == model.Gateway || p.Kind == model.Entry
	who := func() string {
		switch p.Kind {
		case model.Library:
			return "library " + p.Dir
		case model.Main:
			return "main " + p.Dir
		}
		return "leaf " + p.Dir
	}
	network := func(pos ast.Node, what string) {
		hint := "move it to a gateway leaf (Gateway(dir, External) in the manifest) and reach that leaf through a port"
		switch p.Kind {
		case model.Library:
			hint = "let the caller give the library a handle, or move it to a gateway leaf"
		case model.Main:
			hint = "main only wires; the gateway leaves dial and the entry leaves serve"
		}
		model.Report(pass, pos.Pos(), "VL0205", "%s uses %s, which reaches the network, but it is not a gateway or an entry leaf; %s", who(), what, hint)
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				if t := model.NamedOf(pass.TypesInfo.TypeOf(x)); t != nil && !mayNet && listed(networkTypes, full(t)) {
					network(x, "a "+full(t))
				}
			case *ast.Ident:
				switch obj := info.Uses[x].(type) {
				case *types.Var:
					if !mayNet && obj.Pkg() != nil && obj.Parent() == obj.Pkg().Scope() && listed(networkValues, obj.Pkg().Path()+"."+obj.Name()) {
						network(x, obj.Pkg().Path()+"."+obj.Name())
					}
				case *types.Func:
					name := obj.FullName()
					switch {
					case !mayNet && listed(networkFuncs, name):
						network(x, "the network function "+name)
					case p.Kind == model.Library && listed(acquiringFuncs, name):
						model.Report(pass, x.Pos(), "VL0203", "library %s uses %s, which chooses a file or a resource by name; a library uses only the handles that its caller gives it", p.Dir, name)
					case strict && listed(reflectCalls, name):
						model.Report(pass, x.Pos(), "VL0124", "strict mode: %s calls %s, which can call any code", who(), name)
					case strict && p.Kind == model.Entry && listed(networkFuncs, name) && !listed(serveFuncs, name):
						model.Report(pass, x.Pos(), "VL0125", "strict mode: entry leaf %s dials with %s; an entry leaf only serves, so move the call to a gateway leaf and reach it through a port", p.Dir, name)
					case strict && (listed(globalMutators, name) || m.Mutators[name]):
						model.Report(pass, x.Pos(), "VL0211", "strict mode: %s calls %s, which changes state that every part shares; set it once in main, or pass the value through a port or a setting", who(), name)
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func full(t *types.TypeName) string {
	if t.Pkg() == nil {
		return t.Name()
	}
	return t.Pkg().Path() + "." + t.Name()
}
