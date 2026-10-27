package check

import (
	"go/ast"
	"go/types"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/project"
)

// The network and acquiring functions of Appendix D, by types.Func.FullName. A name that ends
// in "*" is a prefix.
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
`)

var acquiringFuncs = names(`
os.Open os.OpenFile os.Create os.CreateTemp os.MkdirTemp os.OpenRoot os.OpenInRoot os.NewFile os.Pipe os.DirFS
os.ReadFile os.WriteFile os.ReadDir os.Remove os.RemoveAll os.Rename os.Mkdir os.MkdirAll os.Truncate os.Chmod
os.Chown os.Lchown os.Chtimes os.Link os.Symlink os.Readlink os.Stat os.Lstat os.Chdir os.CopyFS
io/ioutil.ReadFile io/ioutil.WriteFile io/ioutil.ReadDir io/ioutil.TempFile io/ioutil.TempDir
path/filepath.Walk path/filepath.WalkDir path/filepath.Glob path/filepath.EvalSymlinks plugin.Open
syscall.Open syscall.Openat syscall.Creat syscall.Mkdir syscall.Mkdirat syscall.Rmdir syscall.Unlink
syscall.Unlinkat syscall.Rename syscall.Renameat syscall.Link syscall.Symlink syscall.Chdir syscall.Chroot syscall.Mount
`)

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

// network applies §5.4c: only gateway and entry leaves use a network function (K0205), and a
// library uses no acquiring function (K0203). The PoC checks the uses in the project's own code;
// it does not follow calls into other modules (Appendix D's reach analysis).
func (c *Checker) network(pkg *project.Package, info *types.Info, f *ast.File) {
	if pkg.IsLeaf && (pkg.Gateway || pkg.EntryLeaf) {
		return
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
		case listed(networkFuncs, name):
			who := "leaf " + pkg.Name + " is not a gateway or an entry leaf"
			hint := "move the call to a gateway leaf (gateway for <External>) and reach it through a port"
			if !pkg.IsLeaf {
				who = "library " + pkg.GoName + " is not a leaf"
				hint = "let the caller give the library a handle, or move the call to a gateway leaf"
			}
			c.D.Errorf(c.Pos(id.Pos()), "K0205", "%s uses the network function %s, but %s; %s (§5.4c rule 1)", pkg.Dir, name, who, hint)
		case !pkg.IsLeaf && listed(acquiringFuncs, name):
			c.D.Errorf(c.Pos(id.Pos()), "K0203", "library %s uses %s, which chooses a file or a resource by name; a library uses only the handles that its caller gives it (§5.4a rule 2)", pkg.GoName, name)
		}
		return true
	})
}
