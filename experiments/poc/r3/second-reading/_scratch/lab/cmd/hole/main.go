// T24: a typed hole on plain Go. An unfinished program with the name _hole type-checks with errors;
// the tool answers with the expected type, the names in scope, and the ports on the receiver.
package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports | packages.NeedFiles}
	pkgs, err := packages.Load(cfg, os.Args[1])
	if err != nil {
		panic(err)
	}
	p := pkgs[0]
	fmt.Println("checker errors, tolerated:", len(p.Errors))
	for _, f := range p.Syntax {
		ast.Inspect(f, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok || id.Name != "_hole" {
				return true
			}
			path, _ := astutil.PathEnclosingInterval(f, id.Pos(), id.End())
			fmt.Println("hole at", p.Fset.Position(id.Pos()))
			switch par := path[1].(type) {
			case *ast.CallExpr:
				if sig, ok := p.TypesInfo.TypeOf(par.Fun).(*types.Signature); ok {
					for i, a := range par.Args {
						if a == id {
							fmt.Println("expected type:", sig.Params().At(i).Type())
						}
					}
				}
			case *ast.ReturnStmt:
				for _, q := range path {
					if fd, ok := q.(*ast.FuncDecl); ok {
						sig := p.TypesInfo.Defs[fd.Name].Type().(*types.Signature)
						for i, r := range par.Results {
							if r == id {
								fmt.Println("expected type:", sig.Results().At(i).Type())
							}
						}
					}
				}
			}
			for s := p.Types.Scope().Innermost(id.Pos()); s != nil && s != p.Types.Scope() && s != types.Universe; s = s.Parent() {
				for _, name := range s.Names() {
					o := s.Lookup(name)
					fmt.Printf("in scope: %s %s\n", name, o.Type())
					ptr, ok := o.Type().(*types.Pointer)
					if !ok {
						continue
					}
					st, ok := ptr.Elem().Underlying().(*types.Struct)
					if !ok {
						continue
					}
					for i := 0; i < st.NumFields(); i++ {
						if iface, ok := st.Field(i).Type().Underlying().(*types.Interface); ok {
							fmt.Printf("  port %s.%s %s\n", name, st.Field(i).Name(), st.Field(i).Type())
							for m := 0; m < iface.NumMethods(); m++ {
								fmt.Printf("    %s\n", iface.Method(m))
							}
						}
					}
				}
			}
			return true
		})
	}
}
