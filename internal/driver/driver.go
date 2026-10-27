// Package driver runs the stages of the compiler in order (spec §11.1).
package driver

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/check"
	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/gen"
	"github.com/robot-accomplice/vulcan/internal/link"
	"github.com/robot-accomplice/vulcan/internal/lower"
	"github.com/robot-accomplice/vulcan/internal/project"
)

// FindRoot walks up from dir to the directory that holds system.vulcan.
func FindRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "system.vulcan")); err == nil {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", fmt.Errorf("no system.vulcan in %s or a parent directory", abs)
		}
	}
}

// Result is the outcome of a check.
type Result struct {
	P     *project.Project
	Diags *diag.List
	OK    bool
}

// Check runs every stage up to and including the Go build of main. With tests, it also
// type-checks the test files.
func Check(root string, tests bool) (*Result, error) {
	p, err := project.Load(root)
	if err != nil {
		return nil, err
	}
	r := &Result{P: p, Diags: p.Diags}
	if p.System == nil || p.Container == nil || p.Fatal || stops(p.Diags) {
		r.Diags.Sort()
		return r, nil
	}
	// First pass: the code without its contracts, for the types that the lowering needs.
	var c0 *check.Checker
	if lower.Needed(p) {
		if err := gen.WriteLeaves(p, tests, nil); err != nil {
			return nil, err
		}
		if c0, err = check.Load(p, tests, false); err != nil {
			return nil, err
		}
	}
	in := &lower.Input{P: p, LineFile: func(rel string) string { return gen.LineFile(p, rel) }, Inv: lower.Invariants(p)}
	lowered := map[*project.CodeFile][]byte{}
	for _, pkg := range append(append([]*project.Package{}, p.Libs...), p.Leaves...) {
		var typed *lower.Typed
		if c0 != nil {
			typed = c0.Typed(pkg)
		}
		for cf, text := range lower.Package(in, pkg, typed) {
			lowered[cf] = text
		}
	}
	// Second pass: the lowered code, which every rule reads.
	if err := gen.WriteLeaves(p, tests, lowered); err != nil {
		return nil, err
	}
	c, err := check.Load(p, tests, true)
	if err != nil {
		return nil, err
	}
	c.Rules()
	c.Contracts()
	plan := c.Wiring()
	if !p.Diags.HasErrors() {
		if err := link.Write(p, plan); err != nil {
			return nil, err
		}
		goBuild(p, r.Diags)
		if !r.Diags.HasErrors() {
			goVet(p, r.Diags)
		}
	}
	contractPositions(p, r.Diags)
	r.Diags.Sort()
	r.OK = !r.Diags.HasErrors()
	return r, nil
}

var generatedName = regexp.MustCompile(`\bvulcan_(t|r)[0-9]+\b|\bvulcan_p_(\w+)\b|\bvulcan_old\w*\b`)

// contractPositions maps a diagnostic inside a contract to the start of its clause, because the
// lowered text of a clause has other columns than its source, and writes the generated names of
// the message as source names (audit P2, M3).
func contractPositions(p *project.Project, d *diag.List) {
	starts := clauseStarts(p)
	// A go vet finding on the line of a clause is in the generated check code (for example a
	// check function that copies a struct with a lock), not in code that anyone wrote.
	kept := d.Items[:0]
	for _, x := range d.Items {
		if x.Code == "K0706" && len(starts[x.Pos.File][x.Pos.Line]) > 0 {
			continue
		}
		kept = append(kept, x)
	}
	d.Items = kept
	mapToClauses(d, starts)
}

// clauseStarts gives the columns where a contract clause starts, by file and line.
func clauseStarts(p *project.Project) map[string]map[int][]int {
	starts := map[string]map[int][]int{} // file -> line -> clause columns
	add := func(pos diag.Pos) {
		if starts[pos.File] == nil {
			starts[pos.File] = map[int][]int{}
		}
		starts[pos.File][pos.Line] = append(starts[pos.File][pos.Line], pos.Col)
	}
	for _, pkg := range append(append([]*project.Package{}, p.Leaves...), p.Libs...) {
		for _, f := range pkg.Files {
			for _, fn := range f.Blank.Funcs {
				for _, c := range fn.Clauses {
					add(f.Pos(c.Off))
				}
			}
			for _, t := range f.Blank.Types {
				for _, c := range t.Clauses {
					add(f.Pos(c.Off))
				}
			}
		}
		if pkg.Comp != nil {
			for _, port := range pkg.Comp.Ports {
				for _, m := range port.Methods {
					for _, c := range m.Contracts {
						add(pkg.CompSrc.Pos(c.Off))
					}
				}
			}
		}
	}
	return starts
}

func mapToClauses(d *diag.List, starts map[string]map[int][]int) {
	for i := range d.Items {
		x := &d.Items[i]
		best := 0
		for _, col := range starts[x.Pos.File][x.Pos.Line] {
			if col <= x.Pos.Col && col > best {
				best = col
			}
		}
		if best == 0 {
			continue
		}
		x.Pos.Col = best
		x.Message = generatedName.ReplaceAllStringFunc(x.Message, func(m string) string {
			switch {
			case strings.HasPrefix(m, "vulcan_p_"):
				return strings.TrimPrefix(m, "vulcan_p_")
			case strings.HasPrefix(m, "vulcan_old"):
				return "old(...)"
			}
			return "the result"
		})
	}
}

// stops reports whether a syntax or package-clause error makes the Go stages useless.
func stops(d *diag.List) bool {
	for _, x := range d.Items {
		if x.Severity == diag.Error && (x.Code == "K0001" || x.Code == "K0004") {
			return true
		}
	}
	return false
}

// Binary is the path of the built program.
func Binary(p *project.Project) string {
	name := strings.ToLower(p.System.Name)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(p.Root, ".vulcan", "bin", name)
}

var goErrLine = regexp.MustCompile(`^(.+?):(\d+):(\d+): (.*)$`)

func goBuild(p *project.Project, d *diag.List) {
	cmd := exec.Command("go", "build", "-o", Binary(p), "./"+gen.MainDir)
	cmd.Dir = gen.BuildDir(p)
	cmd.Env = check.Env()
	out, err := cmd.CombinedOutput()
	if err == nil {
		return
	}
	found := false
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		m := goErrLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		file := filepath.ToSlash(m[1])
		if !filepath.IsAbs(m[1]) && !strings.HasPrefix(file, "/") {
			file = filepath.ToSlash(filepath.Join(gen.BuildDir(p), m[1]))
		}
		ln, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		d.Errorf(diag.Pos{File: relTo(p.Root, file), Line: ln, Col: col}, "K0008", "%s", m[4])
		found = true
	}
	if !found {
		d.Errorf(diag.Pos{}, "K0008", "go build failed:\n%s", strings.TrimSpace(string(out)))
	}
}

// goVet runs go vet on the generated code, as a Go team runs it, and reports each finding at
// its .vulcan position as an error (K0706): the PoC gives both arms the same Go analysis
// (parity review P3, item 1).
func goVet(p *project.Project, d *diag.List) {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = gen.BuildDir(p)
	cmd.Env = check.Env()
	out, err := cmd.CombinedOutput()
	if err == nil {
		return
	}
	sc := bufio.NewScanner(bytes.NewReader(out))
	found := false
	for sc.Scan() {
		m := goErrLine.FindStringSubmatch(strings.TrimSpace(sc.Text()))
		if m == nil {
			continue
		}
		file := filepath.ToSlash(m[1])
		if !filepath.IsAbs(m[1]) && !strings.HasPrefix(file, "/") {
			file = filepath.ToSlash(filepath.Join(gen.BuildDir(p), m[1]))
		}
		ln, _ := strconv.Atoi(m[2])
		col, _ := strconv.Atoi(m[3])
		d.Errorf(diag.Pos{File: relTo(p.Root, file), Line: ln, Col: col}, "K0706", "go vet: %s", m[4])
		found = true
	}
	if !found {
		d.Errorf(diag.Pos{}, "K0706", "go vet failed:\n%s", strings.TrimSpace(string(out)))
	}
}

func relTo(root, file string) string {
	r := filepath.ToSlash(root) + "/"
	if strings.HasPrefix(strings.ToLower(file), strings.ToLower(r)) {
		return file[len(r):]
	}
	return file
}

// RaceSupported reports whether go test -race can run on this host.
func RaceSupported() bool {
	out, err := exec.Command("go", "env", "CGO_ENABLED").Output()
	if err != nil || strings.TrimSpace(string(out)) != "1" {
		return false
	}
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("gcc"); err != nil {
			return false
		}
	}
	return true
}
