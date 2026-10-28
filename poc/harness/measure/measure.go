// Package measure computes the measures of round 2 that read the code itself (docs/poc/plan-r2.md,
// objective O7): lines in duplicated blocks, and errors that the code drops or masks. Both are
// counted in the product source of the final workspace and of its base, and a run's value is the
// difference. They are fixed rules, the same in every arm, and approximate on purpose: they are
// reported, not tested, and the PoC compiler does not claim either (its K0701, K0702, and
// duplication warning are not built).
package measure

import (
	"crypto/sha256"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/robot-accomplice/vulcan/poc/harness/detect"
)

// Window is the size of a duplicated block, in normalized lines.
const Window = 6

var (
	spaceRE   = regexp.MustCompile(`\s+`)
	trivialRE = regexp.MustCompile(`^[{}()\[\],;]*$`)
)

// normalize gives the lines that count: without comments, blank lines, and lines of brackets
// only, with runs of spaces made one.
func normalize(src string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n") {
		if i := strings.Index(l, "//"); i >= 0 {
			l = l[:i]
		}
		l = strings.TrimSpace(spaceRE.ReplaceAllString(l, " "))
		if l == "" || trivialRE.MatchString(l) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// CloneLines counts the lines of the product source (.go and .vulcan files under api/, lib/, and
// cmd/, tests included) that lie in a block of Window normalized lines that occurs more than once
// in the workspace.
func CloneLines(ws string) int {
	var files [][]string
	_ = filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(ws, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && !detect.Product(rel+"/") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") || strings.HasSuffix(p, ".vulcan") {
			b, _ := os.ReadFile(p)
			files = append(files, normalize(string(b)))
		}
		return nil
	})
	key := func(ls []string) [32]byte { return sha256.Sum256([]byte(strings.Join(ls, "\n"))) }
	count := map[[32]byte]int{}
	for _, f := range files {
		for i := 0; i+Window <= len(f); i++ {
			count[key(f[i:i+Window])]++
		}
	}
	n := 0
	for _, f := range files {
		in := make([]bool, len(f))
		for i := 0; i+Window <= len(f); i++ {
			if count[key(f[i:i+Window])] > 1 {
				for j := i; j < i+Window; j++ {
					in[j] = true
				}
			}
		}
		for _, b := range in {
			if b {
				n++
			}
		}
	}
	return n
}

// errorType is the predeclared error interface.
var errorType = types.Universe.Lookup("error").Type()

// safeDrops are calls whose error, by common practice, nobody checks.
var safeDrops = []string{"fmt.Print", "fmt.Fprint", "(*strings.Builder).Write", "(*bytes.Buffer).Write", "(*hash.Hash).Write"}

// MaskedErrors counts, in the non-test product code of a loaded workspace, the errors that the
// code drops: a call whose last result is an error, used as a statement (not in go or defer),
// and an error assigned to the blank identifier.
func MaskedErrors(s *detect.Source) int {
	n := 0
	for _, p := range s.Pkgs {
		for _, f := range p.Syntax {
			name := s.Fset.Position(f.Package).Filename
			if s.Generated(f) || strings.HasSuffix(name, "_test.go") {
				continue
			}
			ast.Inspect(f, func(node ast.Node) bool {
				switch x := node.(type) {
				case *ast.ExprStmt:
					if call, ok := x.X.(*ast.CallExpr); ok && returnsError(p.TypesInfo, call) && !safe(p.TypesInfo, call) {
						n++
					}
				case *ast.AssignStmt:
					for i, lhs := range x.Lhs {
						if id, ok := lhs.(*ast.Ident); ok && id.Name == "_" && blankIsError(p.TypesInfo, x, i) {
							n++
						}
					}
				}
				return true
			})
		}
	}
	return n
}

func returnsError(info *types.Info, call *ast.CallExpr) bool {
	t := info.TypeOf(call)
	if tup, ok := t.(*types.Tuple); ok && tup.Len() > 0 {
		t = tup.At(tup.Len() - 1).Type()
	}
	return t != nil && types.Identical(t, errorType)
}

func safe(info *types.Info, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	obj := info.Uses[sel.Sel]
	f, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	full := f.FullName()
	for _, p := range safeDrops {
		if strings.HasPrefix(full, p) {
			return true
		}
	}
	return false
}

// blankIsError reports whether the i-th left side of an assignment receives an error.
func blankIsError(info *types.Info, a *ast.AssignStmt, i int) bool {
	if len(a.Rhs) == len(a.Lhs) {
		t := info.TypeOf(a.Rhs[i])
		return t != nil && types.Identical(t, errorType)
	}
	if len(a.Rhs) == 1 {
		if tup, ok := info.TypeOf(a.Rhs[0]).(*types.Tuple); ok && i < tup.Len() {
			return types.Identical(tup.At(i).Type(), errorType)
		}
	}
	return false
}
