// Package lower turns the contracts of code files and ports into Go (spec §7, Appendix F).
package lower

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"regexp"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/lex"
)

// implies is the name of the call that stands for "a => b" while a contract is analyzed.
const implies = "vulcan_implies"

var resultN = regexp.MustCompile(`\bresult\.([0-9]+)\b`)

// ParseContract parses a contract expression. "=>" becomes a call vulcan_implies(a, b), with
// the lowest precedence, grouping to the right (Appendix B.3), and result.N becomes the
// identifier vulcan_result_N.
func ParseContract(text string) (ast.Expr, error) {
	text = resultN.ReplaceAllString(text, "vulcan_result_$1")
	src := []byte(text)
	toks, errs := lex.Scan(src)
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", errs[0].Msg)
	}
	var n int // the last real token
	for n = 0; n < len(toks) && toks[n].Tok != token.EOF; n++ {
	}
	// Drop a trailing automatic semicolon.
	if n > 0 && toks[n-1].AutoSemi() {
		n--
	}
	out := group(src, toks, 0, n)
	e, err := parser.ParseExpr(out)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// isImplies reports whether toks[i] and toks[i+1] spell "=>".
func isImplies(toks []lex.Tok, i int) bool {
	return toks[i].Tok == token.ASSIGN && toks[i+1].Tok == token.GTR && toks[i].End == toks[i+1].Off
}

// group rewrites toks[i:j], one bracket level: each comma-separated item has its "=>" chain
// turned into nested calls.
func group(src []byte, toks []lex.Tok, i, j int) string {
	var items []string
	start := i
	depth := 0
	for k := i; k <= j; k++ {
		if k == j || (depth == 0 && toks[k].Tok == token.COMMA) {
			items = append(items, chain(src, toks, start, k))
			start = k + 1
			continue
		}
		switch toks[k].Tok {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			depth++
		case token.RPAREN, token.RBRACK, token.RBRACE:
			depth--
		}
	}
	return strings.Join(items, ", ")
}

// chain rewrites one item: parts separated by "=>" at depth 0.
func chain(src []byte, toks []lex.Tok, i, j int) string {
	var parts []string
	var b strings.Builder
	depth := 0
	for k := i; k < j; k++ {
		t := toks[k]
		if depth == 0 && k+1 < j && isImplies(toks, k) {
			parts = append(parts, b.String())
			b.Reset()
			k++
			continue
		}
		switch t.Tok {
		case token.LPAREN, token.LBRACK, token.LBRACE:
			m := lex.Match(toks, k)
			if m > j {
				m = j
			}
			b.WriteString(string(src[t.Off:t.End]))
			b.WriteString(group(src, toks, k+1, m))
			if m < j {
				b.WriteString(string(src[toks[m].Off:toks[m].End]))
			}
			k = m
			b.WriteString(" ")
			continue
		}
		if t.AutoSemi() {
			b.WriteString(" ")
			continue
		}
		b.WriteString(string(src[t.Off:t.End]))
		b.WriteString(" ")
	}
	parts = append(parts, b.String())
	acc := strings.TrimSpace(parts[len(parts)-1])
	for k := len(parts) - 2; k >= 0; k-- {
		acc = implies + "(" + strings.TrimSpace(parts[k]) + ", " + acc + ")"
	}
	return acc
}

// ImpliesArgs returns a and b when e is vulcan_implies(a, b).
func ImpliesArgs(e ast.Expr) (ast.Expr, ast.Expr, bool) {
	c, ok := e.(*ast.CallExpr)
	if !ok || len(c.Args) != 2 {
		return nil, nil, false
	}
	if id, ok := c.Fun.(*ast.Ident); !ok || id.Name != implies {
		return nil, nil, false
	}
	return c.Args[0], c.Args[1], true
}

// Rewrite returns a copy of e in which f replaces nodes: f gets each expression and returns
// a replacement, or nil to keep it (and descend into it).
func Rewrite(e ast.Expr, f func(ast.Expr) ast.Expr) ast.Expr {
	if r := f(e); r != nil {
		return r
	}
	switch x := e.(type) {
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{X: Rewrite(x.X, f), Op: x.Op, Y: Rewrite(x.Y, f)}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{Op: x.Op, X: Rewrite(x.X, f)}
	case *ast.ParenExpr:
		return &ast.ParenExpr{X: Rewrite(x.X, f)}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{X: Rewrite(x.X, f), Sel: x.Sel}
	case *ast.CallExpr:
		args := make([]ast.Expr, len(x.Args))
		for i, a := range x.Args {
			args[i] = Rewrite(a, f)
		}
		return &ast.CallExpr{Fun: Rewrite(x.Fun, f), Args: args, Ellipsis: x.Ellipsis}
	case *ast.IndexExpr:
		return &ast.IndexExpr{X: Rewrite(x.X, f), Index: Rewrite(x.Index, f)}
	case *ast.SliceExpr:
		s := &ast.SliceExpr{X: Rewrite(x.X, f), Slice3: x.Slice3}
		if x.Low != nil {
			s.Low = Rewrite(x.Low, f)
		}
		if x.High != nil {
			s.High = Rewrite(x.High, f)
		}
		if x.Max != nil {
			s.Max = Rewrite(x.Max, f)
		}
		return s
	case *ast.StarExpr:
		return &ast.StarExpr{X: Rewrite(x.X, f)}
	case *ast.CompositeLit:
		elts := make([]ast.Expr, len(x.Elts))
		for i, el := range x.Elts {
			elts[i] = Rewrite(el, f)
		}
		return &ast.CompositeLit{Type: x.Type, Elts: elts}
	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{Key: x.Key, Value: Rewrite(x.Value, f)}
	}
	return e
}

// GoText prints a lowered contract expression on one line, with "=>" as "!a || b".
func GoText(e ast.Expr) string {
	e = Rewrite(e, func(n ast.Expr) ast.Expr {
		if a, b, ok := ImpliesArgs(n); ok {
			return &ast.ParenExpr{X: &ast.BinaryExpr{
				X:  &ast.UnaryExpr{Op: token.NOT, X: &ast.ParenExpr{X: GoTextExpr(a)}},
				Op: token.LOR,
				Y:  &ast.ParenExpr{X: GoTextExpr(b)},
			}}
		}
		return nil
	})
	var b bytes.Buffer
	printer.Fprint(&b, token.NewFileSet(), e)
	return strings.Join(strings.Fields(b.String()), " ")
}

// GoTextExpr lowers the implications inside e.
func GoTextExpr(e ast.Expr) ast.Expr {
	text := GoText(e)
	x, err := parser.ParseExpr(text)
	if err != nil {
		return e
	}
	return x
}

// Idents calls f for each identifier of e that is not a field or method name of a selector.
func Idents(e ast.Expr, f func(*ast.Ident)) {
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			ast.Inspect(x.X, func(m ast.Node) bool {
				if id, ok := m.(*ast.Ident); ok {
					f(id)
				}
				_, isSel := m.(*ast.SelectorExpr)
				if isSel {
					Idents(m.(*ast.SelectorExpr).X, f)
					return false
				}
				return true
			})
			return false
		case *ast.KeyValueExpr:
			Idents(x.Value, f)
			return false
		case *ast.Ident:
			f(x)
		}
		return true
	})
}
