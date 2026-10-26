// Package gen writes the Go code that the compiler generates (spec Appendix F) into the build
// directory, .vulcan/build, which is an ordinary Go module.
package gen

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/decl"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// RuntimeModule is the import path of the runtime module (decision 5).
const RuntimeModule = "github.com/robot-accomplice/vulcan/vulcan"

// MainDir is the build-directory path of the generated main package.
const MainDir = "vulcan_main"

// BuildDir returns the build directory of a project.
func BuildDir(p *project.Project) string { return filepath.Join(p.Root, ".vulcan", "build") }

// GoFileName maps a .vulcan file name to its generated .go file name.
func GoFileName(rel string) string { return strings.TrimSuffix(rel, ".vulcan") + ".go" }

// LineFile is the file name that //line directives use for a source file.
func LineFile(p *project.Project, rel string) string {
	return filepath.ToSlash(filepath.Join(p.Root, filepath.FromSlash(rel)))
}

// RuntimeDir finds the runtime module's source: $VULCAN_RUNTIME, or the vulcan directory next to
// the compiler's own source.
func RuntimeDir() (string, error) {
	if d := os.Getenv("VULCAN_RUNTIME"); d != "" {
		return d, nil
	}
	_, file, _, ok := runtime.Caller(0)
	if ok {
		d := filepath.Join(filepath.Dir(file), "..", "..", "vulcan")
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return filepath.Abs(d)
		}
	}
	return "", fmt.Errorf("cannot find the Vulcan runtime module; set VULCAN_RUNTIME to its directory")
}

var replaceRel = regexp.MustCompile(`(?m)^(\s*(?:replace\s+)?[^\s=]+(?:\s+v[^\s=]+)?\s*=>\s*)(\.{1,2}/\S*)`)

// WriteLeaves writes go.mod, every leaf and library file, and the port declarations.
func WriteLeaves(p *project.Project, tests bool) error {
	dir := BuildDir(p)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	gomod := replaceRel.ReplaceAllStringFunc(string(p.GoMod), func(m string) string {
		sub := replaceRel.FindStringSubmatch(m)
		abs := filepath.ToSlash(filepath.Join(p.Root, filepath.FromSlash(sub[2])))
		return sub[1] + abs
	})
	// The runtime is not published yet, so the build directory resolves it from its source.
	if !regexp.MustCompile(`(?m)^\s*(require\s+)?` + regexp.QuoteMeta(RuntimeModule) + `\s+v`).MatchString(gomod) {
		gomod += "\nrequire " + RuntimeModule + " v0.0.0\n"
	}
	if !strings.Contains(gomod, RuntimeModule+" =>") {
		rt, err := RuntimeDir()
		if err != nil {
			return err
		}
		gomod += "\nreplace " + RuntimeModule + " => " + filepath.ToSlash(rt) + "\n"
	}
	if err := write(dir, "go.mod", []byte(gomod)); err != nil {
		return err
	}
	if len(p.GoSum) > 0 {
		if err := write(dir, "go.sum", p.GoSum); err != nil {
			return err
		}
	}
	for _, pkg := range append(append([]*project.Package{}, p.Libs...), p.Leaves...) {
		for _, f := range pkg.Files {
			if f.Test && !tests {
				continue
			}
			var b bytes.Buffer
			fmt.Fprintf(&b, "//line %s:1:1\n", LineFile(p, f.Rel))
			b.Write(f.Blank.Go)
			if err := write(dir, GoFileName(f.Rel), b.Bytes()); err != nil {
				return err
			}
		}
		if pkg.IsLeaf && pkg.Comp != nil {
			if err := write(dir, pkg.Dir+"/vulcan_ports.go", portsFile(p, pkg)); err != nil {
				return err
			}
			if tests {
				if err := write(dir, pkg.Dir+"/vulcan_stub_test.go", stubFile(p, pkg)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func write(dir, rel string, b []byte) error {
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, b, 0o644)
}

// importName is the package name that an import gives.
func importName(p *project.Project, imp decl.Import) string {
	if imp.Name != "" {
		return imp.Name
	}
	if pkg := p.ByPath[imp.Path]; pkg != nil {
		return pkg.GoName
	}
	base := path.Base(imp.Path)
	if regexp.MustCompile(`^v[0-9]+$`).MatchString(base) {
		base = path.Base(path.Dir(imp.Path))
	}
	return base
}

// usedImports returns the imports of a component that the signature texts name.
func usedImports(p *project.Project, pkg *project.Package, texts []string) []string {
	var out []string
	all := strings.Join(texts, "\n")
	for _, imp := range pkg.Comp.Imports {
		name := importName(p, imp)
		if regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\.`).MatchString(all) {
			if imp.Name != "" {
				out = append(out, fmt.Sprintf("%s %q", imp.Name, imp.Path))
			} else {
				out = append(out, fmt.Sprintf("%q", imp.Path))
			}
		}
	}
	return out
}

func sigTexts(pkg *project.Package) []string {
	var out []string
	for _, port := range pkg.Comp.Ports {
		for _, m := range port.Methods {
			out = append(out, string(pkg.CompSrc.Src[m.SigOff:m.SigEnd]))
		}
	}
	return out
}

// PortType is the generated interface type of a port.
func PortType(port string) string { return "vulcan_" + port }

func portsFile(p *project.Project, pkg *project.Package) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by vulcan from %s. DO NOT EDIT.\n\npackage %s\n\n", pkg.CompSrc.Rel, pkg.GoName)
	if imps := usedImports(p, pkg, sigTexts(pkg)); len(imps) > 0 {
		fmt.Fprintf(&b, "import (\n\t%s\n)\n\n", strings.Join(imps, "\n\t"))
	}
	file := LineFile(p, pkg.CompSrc.Rel)
	for _, port := range pkg.Comp.Ports {
		pos := pkg.CompSrc.Pos(port.Pos)
		fmt.Fprintf(&b, "type %s interface {\n", PortType(port.Name))
		for _, m := range port.Methods {
			mp := pkg.CompSrc.Pos(m.Pos)
			fmt.Fprintf(&b, "\t/*line %s:%d:%d*/%s%s\n", file, mp.Line, mp.Col, m.Name, pkg.CompSrc.Src[m.SigOff:m.SigEnd])
		}
		fmt.Fprintf(&b, "}\n\n")
		fmt.Fprintf(&b, "/*line %s:%d:%d*/var %s %s\n\n", file, pos.Line, pos.Col, port.Name, PortType(port.Name))
	}
	return b.Bytes()
}

func stubFile(p *project.Project, pkg *project.Package) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by vulcan. DO NOT EDIT.\n\npackage %s\n\n", pkg.GoName)
	if imps := usedImports(p, pkg, sigTexts(pkg)); len(imps) > 0 {
		fmt.Fprintf(&b, "import (\n\t%s\n)\n\n", strings.Join(imps, "\n\t"))
	}
	if len(pkg.Comp.Ports) == 0 {
		return b.Bytes()
	}
	fmt.Fprintf(&b, "func init() {\n")
	for _, port := range pkg.Comp.Ports {
		fmt.Fprintf(&b, "\t%s = vulcan_stub_%s{}\n", port.Name, port.Name)
	}
	fmt.Fprintf(&b, "}\n\n")
	for _, port := range pkg.Comp.Ports {
		fmt.Fprintf(&b, "type vulcan_stub_%s struct{}\n\n", port.Name)
		for _, m := range port.Methods {
			sig := string(pkg.CompSrc.Src[m.SigOff:m.SigEnd])
			fmt.Fprintf(&b, "func (vulcan_stub_%s) %s%s {\n\tpanic(\"port %s is not bound; use vulcan.Bind\")\n}\n\n", port.Name, m.Name, sig, port.Name)
		}
	}
	return b.Bytes()
}
