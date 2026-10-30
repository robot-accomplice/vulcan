// Counts the arithmetic operators on integers, the map ranges, and the appends in arm L bodies,
// outside contracts, by type information.
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	cfg := &packages.Config{Dir: os.Args[1], Mode: packages.NeedName | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports | packages.NeedFiles}
	pkgs, err := packages.Load(cfg, "./api/...", "./lib/shop")
	if err != nil {
		panic(err)
	}
	var arith, incdec, compound, mapRange, appendN, sliceExpr, ptrParams, goStmts, sliceParams int
	for _, p := range pkgs {
		isInt := func(e ast.Expr) bool {
			t := p.TypesInfo.TypeOf(e)
			if t == nil {
				return false
			}
			b, ok := t.Underlying().(*types.Basic)
			return ok && b.Info()&types.IsInteger != 0
		}
		for _, f := range p.Syntax {
			if strings.HasSuffix(p.Fset.Position(f.Pos()).Filename, "ports.go") {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.BinaryExpr:
					if (x.Op == token.ADD || x.Op == token.SUB || x.Op == token.MUL) && isInt(x) {
						if tv := p.TypesInfo.Types[x]; tv.Value == nil {
							arith++
						}
					}
				case *ast.IncDecStmt:
					if isInt(x.X) {
						incdec++
					}
				case *ast.AssignStmt:
					if (x.Tok == token.ADD_ASSIGN || x.Tok == token.SUB_ASSIGN || x.Tok == token.MUL_ASSIGN) && isInt(x.Lhs[0]) {
						compound++
					}
				case *ast.RangeStmt:
					if _, ok := p.TypesInfo.TypeOf(x.X).Underlying().(*types.Map); ok {
						mapRange++
					}
				case *ast.CallExpr:
					if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "append" {
						appendN++
					}
				case *ast.SliceExpr:
					sliceExpr++
				case *ast.GoStmt:
					goStmts++
				case *ast.FuncType:
					if x.Params != nil {
						for _, fl := range x.Params.List {
							switch p.TypesInfo.TypeOf(fl.Type).(type) {
							case *types.Pointer:
								ptrParams++
							case *types.Slice:
								sliceParams++
							}
						}
					}
				}
				return true
			})
		}
	}
	fmt.Printf("int arith binary=%d incdec=%d compound=%d mapRange=%d append=%d sliceExpr=%d go=%d ptrParams=%d sliceParams=%d\n", arith, incdec, compound, mapRange, appendN, sliceExpr, goStmts, ptrParams, sliceParams)
}
