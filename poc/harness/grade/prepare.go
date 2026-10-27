package grade

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Prepare compiles every acceptance test, holdout test, rule test, and detector of the material
// once, into dir/bin, so that grading a run only executes them. Grade uses them from then on.
func (g *Grader) Prepare(dir string) error {
	mod := filepath.Join(dir, "module")
	if err := os.RemoveAll(mod); err != nil {
		return err
	}
	if err := g.writeModule(mod); err != nil {
		return err
	}
	for _, t := range g.Meta.Tasks {
		if err := copyDir(filepath.Join(g.Material, "tasks", t.ID, "accept"), filepath.Join(mod, "accept", t.ID)); err != nil {
			return err
		}
	}
	for _, part := range []string{"holdout", "rules"} {
		if err := copyDir(filepath.Join(g.Material, part), filepath.Join(mod, part)); err != nil {
			return err
		}
	}
	bin := filepath.Join(dir, "bin")
	build := func(args ...string) error {
		cmd := exec.Command("go", args...)
		cmd.Dir = mod
		cmd.Env = g.env()
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("go %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return nil
	}
	compileTest := func(pkg, out string) error {
		if _, err := os.Stat(filepath.Join(mod, filepath.FromSlash(pkg))); err != nil {
			return nil
		}
		return build("test", "-c", "-o", out, "./"+pkg+"/")
	}
	for _, t := range g.Meta.Tasks {
		if err := compileTest("accept/"+t.ID, filepath.Join(bin, "accept", t.ID+".test"+exe())); err != nil {
			return err
		}
	}
	if err := compileTest("holdout", filepath.Join(bin, "holdout.test"+exe())); err != nil {
		return err
	}
	for _, r := range g.Meta.Rules {
		switch r.Method {
		case "test":
			if err := compileTest("rules/"+r.ID, filepath.Join(bin, "rules", r.ID+".test"+exe())); err != nil {
				return err
			}
		case "detector":
			if err := build("build", "-o", filepath.Join(bin, "rules", r.ID+exe()), "./rules/"+r.ID); err != nil {
				return err
			}
		default:
			return fmt.Errorf("rule %s: method %q is not test or detector", r.ID, r.Method)
		}
	}
	g.prepared = bin
	return nil
}

// writeModule writes a Go module that imports the repository, for poc/stockroomtest and
// poc/harness/detect.
func (g *Grader) writeModule(mod string) error {
	repo, err := filepath.Abs(g.Repo)
	if err != nil {
		return err
	}
	gomod := fmt.Sprintf("module grading\n\ngo 1.26\n\nrequire github.com/robot-accomplice/vulcan v0.0.0\n\nreplace github.com/robot-accomplice/vulcan => %s\n", filepath.ToSlash(repo))
	if err := writeFile(filepath.Join(mod, "go.mod"), []byte(gomod)); err != nil {
		return err
	}
	sum, err := os.ReadFile(filepath.Join(repo, "go.sum"))
	if err != nil {
		return err
	}
	return writeFile(filepath.Join(mod, "go.sum"), sum)
}

// runBinary runs a compiled test binary and counts its top-level tests.
func runBinary(bin string, env []string, logf *os.File) record.Tests {
	if _, err := os.Stat(bin); err != nil {
		return record.Tests{}
	}
	cmd := exec.Command(bin, "-test.v", "-test.count=1")
	cmd.Env = env
	cmd.Dir = filepath.Dir(bin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	fmt.Fprintf(logf, "== %s\n%s\n", filepath.Base(bin), out.String())
	var t record.Tests
	sc := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "--- PASS: "):
			t.Passed++
		case strings.HasPrefix(line, "--- FAIL: "):
			t.Failed++
		}
	}
	if err != nil && t.Failed == 0 {
		t.Failed++ // the binary failed without a failed test: a panic or a timeout
	}
	return t
}
