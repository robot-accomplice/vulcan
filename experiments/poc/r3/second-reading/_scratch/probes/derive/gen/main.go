// T26: derive a property test from the contract statements of plain Go. The output is ordinary Go.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
)

func main() {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "target/t.go", nil, 0)
	if err != nil {
		panic(err)
	}
	fmt.Print("package target\n\nimport (\n\t\"testing\"\n\t\"testing/quick\"\n)\n\n")
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		var names, guards []string
		for _, p := range fd.Type.Params.List {
			for _, n := range p.Names {
				names = append(names, n.Name)
			}
		}
		for _, s := range fd.Body.List {
			es, ok := s.(*ast.ExprStmt)
			if !ok {
				continue
			}
			if c, ok := es.X.(*ast.CallExpr); ok {
				if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Requires" {
					var b bytes.Buffer
					printer.Fprint(&b, fs, c.Args[0])
					guards = append(guards, b.String())
				}
			}
		}
		a := strings.Join(names, ", ")
		fmt.Printf("func TestDerived%s(t *testing.T) {\n\tcfg := &quick.Config{MaxCount: 200000, Values: small}\n\tif err := quick.Check(func(%s int) bool {\n\t\tif !(%s) {\n\t\t\treturn true\n\t\t}\n\t\t%s(%s)\n\t\treturn true\n\t}, cfg); err != nil {\n\t\tt.Fatal(err)\n\t}\n}\n", fd.Name.Name, a, strings.Join(guards, " && "), fd.Name.Name, a)
	}
}
