package rules

import (
	"slices"
	"strings"
)

// The effects that only-in names. The functions are by types.Func.FullName, and a name that ends
// in "*" is a prefix. Values are package-level variables, and types are the types whose literal
// builds a thing that reaches the effect.

type effect struct {
	funcs, values, types map[string]bool
}

func set(s string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.Fields(s) {
		out[f] = true
	}
	return out
}

// matches reports whether a name is in a set, where a name of the set that ends in "*" is a prefix.
func matches(s map[string]bool, name string) bool {
	if s[name] {
		return true
	}
	for k := range s {
		if strings.HasSuffix(k, "*") && strings.HasPrefix(name, strings.TrimSuffix(k, "*")) {
			return true
		}
	}
	return false
}

// dialFuncs and serveFuncs are the network functions of spec Appendix D, with net/textproto.Dial
// (hole H7) and RoundTripper.RoundTrip (hole H3).
const dialFuncs = `
net.Dial net.DialTimeout net.DialTCP net.DialUDP net.DialIP net.DialUnix
net.FileConn net.FilePacketConn (*net.Dialer).Dial (*net.Dialer).DialContext
net.Lookup* (*net.Resolver).Lookup*
crypto/tls.Dial crypto/tls.DialWithDialer (*crypto/tls.Dialer).Dial (*crypto/tls.Dialer).DialContext
net/http.Get net/http.Head net/http.Post net/http.PostForm (*net/http.Client).Do (*net/http.Transport).RoundTrip
(*net/http.Client).Get (*net/http.Client).Head (*net/http.Client).Post (*net/http.Client).PostForm
(net/http.RoundTripper).RoundTrip
net/smtp.Dial net/smtp.SendMail net/rpc.Dial net/rpc.DialHTTP net/rpc.DialHTTPPath net/textproto.Dial
net/rpc/jsonrpc.Dial log/syslog.Dial log/syslog.New log/syslog.NewLogger
(*net.Dialer).DialTCP (*net.Dialer).DialUDP (*net.Dialer).DialIP (*net.Dialer).DialUnix
(*os/exec.Cmd).Run (*os/exec.Cmd).Start (*os/exec.Cmd).Output (*os/exec.Cmd).CombinedOutput
database/sql.Open database/sql.OpenDB os/exec.Command os/exec.CommandContext os.StartProcess
syscall.Socket syscall.Connect syscall.Exec syscall.ForkExec syscall.StartProcess
`

const serveFuncs = `
net.Listen net.ListenPacket net.ListenTCP net.ListenUDP net.ListenIP net.ListenUnix net.ListenUnixgram
net.ListenMulticastUDP net.FileListener (*net.ListenConfig).Listen (*net.ListenConfig).ListenPacket crypto/tls.Listen
net/http.ListenAndServe net/http.ListenAndServeTLS net/http.Serve net/http.ServeTLS
(*net/http.Server).ListenAndServe (*net/http.Server).ListenAndServeTLS (*net/http.Server).Serve (*net/http.Server).ServeTLS
syscall.Bind syscall.Listen syscall.Accept net/http/fcgi.Serve
`

const dialValues = `net/http.DefaultClient net/http.DefaultTransport net.DefaultResolver`

const dialTypes = `net/http.Client net/http.Transport net.Dialer crypto/tls.Dialer net.Resolver`

var effects = map[string]*effect{
	"network": {funcs: set(dialFuncs + serveFuncs), values: set(dialValues), types: set(dialTypes)},
	"dial":    {funcs: set(dialFuncs), values: set(dialValues), types: set(dialTypes)},
	"serve":   {funcs: set(serveFuncs)},
	"database": {funcs: set(`
database/sql.Open database/sql.OpenDB (*database/sql.DB).* (*database/sql.Tx).* (*database/sql.Stmt).*
(*database/sql.Conn).* (*database/sql.Rows).* (*database/sql.Row).*
`)},
	"files": {funcs: set(`
os.Open os.OpenFile os.Create os.CreateTemp os.MkdirTemp os.OpenRoot os.OpenInRoot os.NewFile os.Pipe os.DirFS
os.ReadFile os.WriteFile os.ReadDir os.Remove os.RemoveAll os.Rename os.Mkdir os.MkdirAll os.Truncate os.Chmod
os.Chown os.Lchown os.Chtimes os.Link os.Symlink os.Readlink os.Stat os.Lstat os.Chdir os.CopyFS
io/ioutil.ReadFile io/ioutil.WriteFile io/ioutil.ReadDir io/ioutil.TempFile io/ioutil.TempDir
path/filepath.Walk path/filepath.WalkDir path/filepath.Glob path/filepath.EvalSymlinks plugin.Open
html/template.ParseFiles html/template.ParseGlob (*html/template.Template).ParseFiles (*html/template.Template).ParseGlob
text/template.ParseFiles text/template.ParseGlob (*text/template.Template).ParseFiles (*text/template.Template).ParseGlob
archive/zip.OpenReader net/http.ServeFile
syscall.Open syscall.Openat syscall.Creat syscall.Mkdir syscall.Mkdirat syscall.Rmdir syscall.Unlink
syscall.Unlinkat syscall.Rename syscall.Renameat syscall.Link syscall.Symlink syscall.Chdir syscall.Chroot syscall.Mount
`)},
	"clock": {funcs: set(`
time.Now time.Since time.Until time.After time.AfterFunc time.NewTimer time.NewTicker time.Tick time.Sleep
context.WithTimeout context.WithDeadline context.WithTimeoutCause context.WithDeadlineCause
`)},
	"randomness": {funcs: set(`
crypto/rand.* math/rand.* (*math/rand.Rand).* math/rand/v2.* (*math/rand/v2.Rand).*
`), values: set(`crypto/rand.Reader`)},
	"processes": {funcs: set(`
os/exec.Command os/exec.CommandContext os.StartProcess os.FindProcess
(*os/exec.Cmd).Run (*os/exec.Cmd).Start (*os/exec.Cmd).Output (*os/exec.Cmd).CombinedOutput
syscall.Exec syscall.ForkExec syscall.StartProcess
`)},
}

// Lists gives a version of what the tool holds and a rules file does not: the effect lists, the
// forms, and the places. The head of a rules file records it, so that rules upgrade can show that a
// new build changes what a rule enforces. It does not cover the code of a kind, or the model's
// own lists.
func Lists() string {
	var b strings.Builder
	for _, name := range EffectNames() {
		e := effects[name]
		for _, m := range []map[string]bool{e.funcs, e.values, e.types} {
			var words []string
			for w := range m {
				words = append(words, w)
			}
			slices.Sort(words)
			b.WriteString(name + " " + strings.Join(words, " ") + "\n")
		}
	}
	b.WriteString(strings.Join(codeForms, " ") + "\n" + strings.Join(declarationAts, " ") + "\n")
	return Hash([]byte(b.String()))
}

// EffectNames lists the effects.
func EffectNames() []string {
	var out []string
	for k := range effects {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
