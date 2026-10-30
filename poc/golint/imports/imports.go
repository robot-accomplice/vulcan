// Package imports is analyzer A2 of vulcan-lint: what each part may import.
package imports

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/robot-accomplice/vulcan/internal/rules"
	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A2.
var Analyzer = &analysis.Analyzer{
	Name: "imports",
	Doc:  "check the imports of each part: a leaf or a library imports no leaf, no code imports the manifest, no //go:linkname, no function without a body or assembly file in a leaf or a library, and in strict mode no unsafe, syscall, or cgo (VL0101, VL0118, VL0122, VL0124, VL0126, VL0204)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Manifest || u.Part == nil {
		return nil, nil
	}
	m, self := u.M, u.Part
	if self.Kind != model.Main {
		foreignCode(pass, u)
	}
	for _, f := range pass.Files {
		if self.Kind != model.Main {
			for _, name := range rules.Bodyless(f) {
				model.Report(pass, name.Pos(), "VL0126", "%s", rules.HoleH6Message(name.Name))
			}
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			pos := imp.Path.Pos()
			if p == m.ManifestPath() {
				model.Report(pass, pos, "VL0118", "code imports the manifest %s; the manifest is read by vulcan-lint, never by the code", p)
				continue
			}
			if t := m.PartOf(p); t != nil && t.IsLeaf() && t != self {
				switch {
				case self.IsLeaf():
					model.Report(pass, pos, "VL0101", "leaf %s imports leaf %s; a leaf uses another leaf only through a port that it declares in its ports.go, and main passes the other leaf to its constructor", self.Dir, t.Dir)
				case self.Kind == model.Library:
					model.Report(pass, pos, "VL0204", "library %s imports leaf %s; a library imports only libraries", self.Dir, t.Dir)
				}
			}
			if m.StrictFor(self) && (p == "unsafe" || p == "syscall" || p == "C") {
				what := "imports " + p
				if p == "C" {
					what = "uses cgo"
				}
				model.Report(pass, pos, "VL0124", "strict mode: %s %s %s, which can go around the architecture rules", kindName(self), self.Dir, what)
			}
		}
		if self.Kind == model.Main {
			continue
		}
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if strings.HasPrefix(c.Text, "//go:linkname") {
					model.Report(pass, c.Pos(), "VL0122", "//go:linkname in %s %s reaches code that the rules cannot see; call it through a port or a library", kindName(self), self.Dir)
				}
			}
		}
	}
	return nil, nil
}

// foreignCode applies hole H6 to the files of a package: an assembly file or an object file. It
// reads the directory, as V does, so that a file for another system counts too.
func foreignCode(pass *analysis.Pass, u *model.Unit) {
	dir := filepath.Join(u.M.Root, filepath.FromSlash(u.Dir))
	for _, name := range rules.AsmFiles(dir) {
		file := filepath.Join(dir, name)
		b, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		tf := pass.Fset.AddFile(file, -1, len(b)+1)
		tf.SetLinesForContent(b)
		model.Report(pass, tf.Pos(0), "VL0126", "%s", rules.HoleH6FileMessage(name))
	}
}

func kindName(p *model.Part) string {
	if p.IsLeaf() {
		return "leaf"
	}
	return p.Kind
}
