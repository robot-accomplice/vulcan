// T22: a diagnostic with an exact edit. Each pointer method of a type with Invariant starts with
// defer contract.Keep(recv). The fix inserts it.
package main

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

var A = &analysis.Analyzer{Name: "fixkeep", Doc: "insert the missing Keep", Run: run}

func run(pass *analysis.Pass) (any, error) {
	for _, f := range pass.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || fd.Body == nil || len(fd.Recv.List[0].Names) == 0 {
				continue
			}
			p, ok := pass.TypesInfo.TypeOf(fd.Recv.List[0].Type).(*types.Pointer)
			if !ok {
				continue
			}
			if m, _, _ := types.LookupFieldOrMethod(p.Elem(), true, pass.Pkg, "Invariant"); m == nil {
				continue
			}
			has := false
			if len(fd.Body.List) > 0 {
				if ds, ok := fd.Body.List[0].(*ast.DeferStmt); ok {
					has = types.ExprString(ds.Call.Fun) == "contract.Keep"
				}
			}
			if !has {
				recv := fd.Recv.List[0].Names[0].Name
				pass.Report(analysis.Diagnostic{Pos: fd.Pos(), Message: "pointer method without defer contract.Keep",
					SuggestedFixes: []analysis.SuggestedFix{{Message: "insert it", TextEdits: []analysis.TextEdit{{Pos: fd.Body.Lbrace + 1, End: fd.Body.Lbrace + 1, NewText: []byte(fmt.Sprintf("\n\tdefer contract.Keep(%s)", recv))}}}}})
			}
		}
	}
	return nil, nil
}

func main() { singlechecker.Main(A) }
