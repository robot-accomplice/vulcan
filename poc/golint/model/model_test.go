package model

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestParseConfig(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string // a problem, or "" for none
	}{
		{"split", map[string]string{
			"arch.go":   "package arch\nvar Manifest = System{Parts: []Part{Leaf(\"api/a\")}}\n",
			"config.go": "package arch\nvar Config = Settings{Strict: true, StrictExempt: []string{\"api/a\"}}\n",
		}, ""},
		{"config elsewhere", map[string]string{
			"arch.go": "package arch\nvar Manifest = System{Parts: []Part{Leaf(\"api/a\")}}\nvar Config = Settings{Strict: true}\n",
		}, "Config is declared in arch/config.go"},
		{"setting in the manifest", map[string]string{
			"arch.go": "package arch\nvar Manifest = System{Strict: true, Parts: []Part{Leaf(\"api/a\")}}\n",
		}, "Strict is check configuration"},
		{"parts in the configuration", map[string]string{
			"arch.go":   "package arch\nvar Manifest = System{}\n",
			"config.go": "package arch\nvar Config = Settings{Parts: []Part{}}\n",
		}, "the parts are in var Manifest"},
	} {
		fset := token.NewFileSet()
		var files []*ast.File
		for name, src := range tc.files {
			f, err := parser.ParseFile(fset, name, src, 0)
			if err != nil {
				t.Fatal(err)
			}
			files = append(files, f)
		}
		m := &Manifest{}
		Parse(m, fset, files)
		var got []string
		for _, p := range m.Problems {
			got = append(got, p.Msg)
		}
		switch {
		case tc.want == "" && len(got) > 0:
			t.Errorf("%s: problems %v", tc.name, got)
		case tc.want != "" && (len(got) != 1 || !strings.Contains(got[0], tc.want)):
			t.Errorf("%s: got %v, want %q", tc.name, got, tc.want)
		}
		if tc.name == "split" && (!m.Strict || !m.Exempt["api/a"] || m.ByDir["api/a"] == nil) {
			t.Errorf("split: not read: %+v", m)
		}
	}
}
