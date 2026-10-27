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
	if err := gen.WriteLeaves(p, tests); err != nil {
		return nil, err
	}
	c, err := check.Load(p, tests)
	if err != nil {
		return nil, err
	}
	c.Rules()
	plan := c.Wiring()
	if !p.Diags.HasErrors() {
		if err := link.Write(p, plan); err != nil {
			return nil, err
		}
		goBuild(p, r.Diags)
	}
	r.Diags.Sort()
	r.OK = !r.Diags.HasErrors()
	return r, nil
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
