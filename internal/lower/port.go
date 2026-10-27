package lower

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/blank"
	"github.com/robot-accomplice/vulcan/internal/decl"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// HasContracts reports whether a port has a contract on a method.
func HasContracts(port *decl.Port) bool {
	for _, m := range port.Methods {
		if len(m.Contracts) > 0 {
			return true
		}
	}
	return false
}

// WrapType is the generated type that checks a port's contract (Appendix F, "Ports").
func WrapType(port string) string { return "Vulcan_wrap_" + port }

// PortWrapper writes the type that checks a port's contract around a provider. It lives in the
// consumer's package, so that main binds a provider through it and vulcan.Bind binds a fake
// through it (§8: a fake meets the port's contract). lf is the //line file of component.vulcan.
func PortWrapper(p *project.Project, leaf *project.Package, port *decl.Port, lf string) string {
	src := leaf.CompSrc
	iface := "vulcan_" + port.Name
	wrap := WrapType(port.Name)
	var b strings.Builder
	fmt.Fprintf(&b, "type %s struct{ Impl %s }\n\n", wrap, iface)
	fmt.Fprintf(&b, "func (w %s) Vulcan_wrap(impl any) any { return %s{Impl: impl.(%s)} }\n\n", wrap, wrap, iface)
	for _, m := range port.Methods {
		sig := string(src.Src[m.SigOff:m.SigEnd])
		ft, fset, err := parseSig(sig)
		if err != nil {
			continue // the type checker reports the signature
		}
		at := func(n ast.Node) string {
			return sig[fset.Position(n.Pos()).Offset-len(sigPrefix) : fset.Position(n.End()).Offset-len(sigPrefix)]
		}
		var params, args []string
		names := map[string]bool{}
		k := 0
		for _, f := range ft.Params.List {
			typ := at(f.Type)
			_, variadic := f.Type.(*ast.Ellipsis)
			fieldNames := f.Names
			if len(fieldNames) == 0 {
				fieldNames = []*ast.Ident{ast.NewIdent(fmt.Sprintf("vulcan_a%d", k))}
			}
			for _, n := range fieldNames {
				name := n.Name
				if name == "_" {
					name = fmt.Sprintf("vulcan_a%d", k)
				}
				k++
				names[name] = true
				params = append(params, name+" "+typ)
				if variadic {
					args = append(args, name+"...")
				} else {
					args = append(args, name)
				}
			}
		}
		var results []result
		if ft.Results != nil {
			for _, f := range ft.Results.List {
				n := len(f.Names)
				if n == 0 {
					n = 1
				}
				for range n {
					results = append(results, result{typ: at(f.Type)})
				}
			}
		}
		var res, vars []string
		for i, r := range results {
			vars = append(vars, fmt.Sprintf("vulcan_r%d", i))
			res = append(res, fmt.Sprintf("vulcan_r%d %s", i, r.typ))
		}
		errIdx := -1
		if n := len(results); n > 0 && results[n-1].typ == "error" {
			errIdx = n - 1
		}
		var nonErr []int
		for i := range results {
			if i != errIdx {
				nonErr = append(nonErr, i)
			}
		}
		caller := leaf.Name + "." + port.Name + "." + m.Name
		var pre, post strings.Builder
		for _, c := range m.Contracts {
			text := string(src.Src[c.Off:c.End])
			pos := src.Pos(c.Off)
			e, err := ParseContract(text)
			if err != nil {
				p.Diags.Errorf(pos, "K0001", "the contract does not parse: %v", err)
				continue
			}
			ensures := c.Word == "ensures"
			fl := &file{in: &Input{P: p}, cf: &project.CodeFile{Source: src}}
			fl.checkNames(e, blank.Clause{Word: c.Word, Off: c.Off, End: c.End, Text: text}, ensures, errIdx, len(nonErr))
			var shows []string
			seen := map[string]bool{}
			show := func(label, expr string) {
				if !seen[label] {
					seen[label] = true
					shows = append(shows, fmt.Sprintf("vulcan_rt.Show(%q, %s)", label, expr))
				}
			}
			var olds []string
			x := Rewrite(e, func(n ast.Expr) ast.Expr {
				switch v := n.(type) {
				case *ast.Ident:
					switch {
					case names[v.Name]:
						show(v.Name, v.Name)
					case v.Name == "err" && errIdx >= 0 && ensures:
						show("err", vars[errIdx])
						return ast.NewIdent(vars[errIdx])
					case v.Name == "result" && len(nonErr) > 0 && ensures:
						show("result", vars[nonErr[0]])
						return ast.NewIdent(vars[nonErr[0]])
					case strings.HasPrefix(v.Name, "vulcan_result_") && ensures:
						i, _ := strconv.Atoi(strings.TrimPrefix(v.Name, "vulcan_result_"))
						if i < len(nonErr) {
							show("result."+strconv.Itoa(i), vars[nonErr[i]])
							return ast.NewIdent(vars[nonErr[i]])
						}
					}
				case *ast.CallExpr:
					if id, ok := v.Fun.(*ast.Ident); ok && id.Name == "old" && len(v.Args) == 1 && ensures {
						name := fmt.Sprintf("vulcan_old%d", len(olds))
						olds = append(olds, fmt.Sprintf("%s := %s; ", name, GoText(v.Args[0])))
						show("old("+GoText(v.Args[0])+")", name)
						return ast.NewIdent(name)
					}
				}
				return nil
			})
			args := fmt.Sprintf("%q, %q, %q, %q", c.Word, text, pos.String(), caller)
			if len(shows) > 0 {
				args += ", " + strings.Join(shows, ", ")
			}
			check := fmt.Sprintf("if !(/*line %s:%d:%d*/%s) { vulcan_rt.Fail(%s) }; ", lf, pos.Line, pos.Col, GoText(x), args)
			if ensures {
				for _, o := range olds {
					pre.WriteString(o)
				}
				post.WriteString(check)
			} else {
				pre.WriteString(check)
			}
		}
		call := "w.Impl." + m.Name + "(" + strings.Join(args, ", ") + ")"
		if len(vars) > 0 {
			call = strings.Join(vars, ", ") + " = " + call
		}
		mp := src.Pos(m.Pos)
		fmt.Fprintf(&b, "func (w %s) /*line %s:%d:%d*/%s(%s) (%s) {\n\t%s%s; %sreturn\n}\n\n", wrap, lf, mp.Line, mp.Col, m.Name, strings.Join(params, ", "), strings.Join(res, ", "), pre.String(), call, post.String())
	}
	return b.String()
}

const sigPrefix = "package p;var _ func"

// parseSig parses a method signature alone. Offsets in the file set, less len(sigPrefix), are
// offsets in sig.
func parseSig(sig string) (*ast.FuncType, *token.FileSet, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sig.go", sigPrefix+sig, 0)
	if err != nil {
		return nil, nil, err
	}
	return f.Decls[0].(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Type.(*ast.FuncType), fset, nil
}
