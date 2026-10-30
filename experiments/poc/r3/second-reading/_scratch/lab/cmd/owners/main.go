// T15: a table of which package owns which table. A constant SQL write from another package is
// rejected. A query text that is not a constant is rejected too, so the rule has no silent hole.
package main

import (
	"go/ast"
	"go/constant"
	"path"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var owner = map[string]string{"orders": "ordersdb", "stock": "stockdb"}
var write = regexp.MustCompile(`(?i)\b(insert\s+into|update|delete\s+from)\s+(\w+)`)

var A = &analysis.Analyzer{Name: "owners", Doc: "declared ownership of data", Run: run}

func run(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			s, ok := c.Fun.(*ast.SelectorExpr)
			if !ok || !strings.HasPrefix(s.Sel.Name, "Exec") || len(c.Args) == 0 {
				return true
			}
			tv := pass.TypesInfo.Types[c.Args[0]]
			if tv.Value == nil || tv.Value.Kind() != constant.String {
				pass.Reportf(c.Pos(), "a query that is not a constant cannot be checked against the table of owners")
				return true
			}
			if m := write.FindStringSubmatch(constant.StringVal(tv.Value)); m != nil {
				if o, ok := owner[m[2]]; ok && o != path.Base(pass.Pkg.Path()) {
					pass.Reportf(c.Pos(), "package %s writes table %s, which %s owns", pass.Pkg.Name(), m[2], o)
				}
			}
			return true
		})
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
