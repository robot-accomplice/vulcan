// Package imports is analyzer A2 of vulcan-lint: what each part may import.
package imports

import (
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

// Analyzer is A2.
var Analyzer = &analysis.Analyzer{
	Name: "imports",
	Doc:  "check the imports of each part: a leaf or a library imports no leaf, no code imports the manifest, no //go:linkname, and in strict mode no unsafe, syscall, or cgo (VL0101, VL0118, VL0122, VL0124, VL0204)",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	u := model.For(pass)
	if u == nil || u.Manifest || u.Part == nil {
		return nil, nil
	}
	m, self := u.M, u.Part
	for _, f := range pass.Files {
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

func kindName(p *model.Part) string {
	if p.IsLeaf() {
		return "leaf"
	}
	return p.Kind
}
