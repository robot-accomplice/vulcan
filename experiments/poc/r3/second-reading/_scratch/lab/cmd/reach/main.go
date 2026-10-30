// T2: facts that follow static calls through every dependency, so a dial inside an SDK is seen.
package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// Reaches says that a function statically reaches a network function; Via is the path.
type Reaches struct{ Via string }

func (*Reaches) AFact() {}

var seeds = map[string]bool{"net.Dial": true, "net.DialTimeout": true, "(*net.Dialer).Dial": true, "(*net.Dialer).DialContext": true,
	"net.Listen": true, "(*net/http.Client).Do": true, "(*net/http.Transport).RoundTrip": true, "database/sql.Open": true, "os/exec.Command": true}

var A = &analysis.Analyzer{Name: "reach", Doc: "reach through dependencies", Run: run, FactTypes: []analysis.Fact{new(Reaches)}}

func callees(pass *analysis.Pass, body *ast.BlockStmt) []*types.Func {
	var out []*types.Func
	ast.Inspect(body, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			if fn, ok := pass.TypesInfo.Uses[id].(*types.Func); ok {
				out = append(out, fn.Origin())
			}
		}
		return true
	})
	return out
}

func run(pass *analysis.Pass) (any, error) {
	bodies := map[*types.Func]*ast.BlockStmt{}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Body != nil {
				if fn, ok := pass.TypesInfo.Defs[fd.Name].(*types.Func); ok {
					bodies[fn] = fd.Body
				}
			}
		}
	}
	reach := func(fn *types.Func) (string, bool) {
		if seeds[fn.FullName()] {
			return fn.FullName(), true
		}
		var r Reaches
		if pass.ImportObjectFact(fn, &r) {
			return r.Via, true
		}
		return "", false
	}
	for changed := true; changed; {
		changed = false
		for fn, body := range bodies {
			var r Reaches
			if pass.ImportObjectFact(fn, &r) {
				continue
			}
			for _, c := range callees(pass, body) {
				if v, ok := reach(c); ok {
					pass.ExportObjectFact(fn, &Reaches{Via: c.FullName() + " -> " + v})
					changed = true
					break
				}
			}
		}
	}
	p := pass.Pkg.Path()
	if !strings.HasPrefix(p, "stockroom/") || strings.Contains(p, "/store") || strings.Contains(p, "/payments") || strings.Contains(p, "/webhooks") || strings.Contains(p, "/web") || strings.Contains(p, "/cmd/") {
		return nil, nil
	}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if fn, ok := pass.TypesInfo.Uses[id].(*types.Func); ok && fn.Pkg() != pass.Pkg {
					if v, ok := reach(fn.Origin()); ok {
						pass.Reportf(id.Pos(), "a plain part reaches the network: %s -> %s", fn.FullName(), v)
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
