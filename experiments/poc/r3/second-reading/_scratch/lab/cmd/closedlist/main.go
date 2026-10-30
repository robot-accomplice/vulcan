// T1: a closed list of packages per kind. A plain leaf or library may use only listed packages;
// a body-less function and an assembly file are rejected.
package main

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var plainAllowed = map[string]bool{"context": true, "errors": true, "fmt": true, "sort": true, "slices": true, "maps": true,
	"strings": true, "strconv": true, "time": true, "sync": true, "cmp": true, "math": true, "unicode": true,
	"unicode/utf8": true, "net/url": true, "log": true, "crypto/rand": true, "crypto/hmac": true, "crypto/sha256": true, "encoding/json": true, "encoding/hex": true, "hash": true, "io": true, "regexp": true}

var A = &analysis.Analyzer{Name: "closedlist", Doc: "closed list per kind", Run: run}

func run(pass *analysis.Pass) (any, error) {
	path := pass.Pkg.Path()
	privileged := strings.Contains(path, "/store") || strings.Contains(path, "/payments") || strings.Contains(path, "/webhooks") ||
		strings.Contains(path, "/web") || strings.HasSuffix(path, "/contract") || strings.Contains(path, "/cmd/") || strings.HasSuffix(path, "/arch")
	if privileged {
		return nil, nil
	}
	for _, name := range pass.OtherFiles {
		if strings.HasSuffix(name, ".s") {
			pass.Reportf(pass.Files[0].Package, "assembly file %s in a plain part", name)
		}
	}
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Body == nil {
				pass.Reportf(fd.Pos(), "function %s has no body", fd.Name.Name)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := pass.TypesInfo.Uses[id]
			if obj == nil || obj.Pkg() == nil || obj.Pkg() == pass.Pkg {
				return true
			}
			if _, isPkg := obj.(*types.PkgName); isPkg {
				return true
			}
			p := obj.Pkg().Path()
			if strings.HasPrefix(p, "stockroom/lib/") || plainAllowed[p] {
				return true
			}
			pass.Reportf(id.Pos(), "%s.%s is not on the closed list of a plain part", p, obj.Name())
			return true
		})
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
