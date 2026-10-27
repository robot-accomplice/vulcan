// Package grade grades one run (docs/poc/plan.md, section 6): it builds the final workspace, runs
// the task's acceptance tests, the rule tests and the rule detectors, and the holdout tests, the
// same way in both arms.
//
// The hidden material is a directory, unsealed only for grading:
//
//	meta.json               tasks and rules (record.Meta)
//	tasks/<id>/brief.md     the brief, the only thing the agent sees
//	tasks/<id>/accept/      the acceptance tests: a Go test package that uses poc/stockroomtest
//	rules/<id>/             a rule test (a Go test package) or a detector (a main package that
//	                        uses poc/harness/detect), as meta.json says
//	holdout/                the holdout tests: a Go test package
//	RULES.md                the team rules that both arms get
package grade

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Grader grades runs.
type Grader struct {
	Repo     string // the root of the vulcan repository, for the test module
	Material string // the unsealed hidden material
	ToolDir  string // the directory of the vulcan command
	Meta     *record.Meta
}

// Grade fills the grading fields of r, for the run in outDir (its workspace is outDir/workspace).
func (g *Grader) Grade(outDir string, r *record.Run) error {
	ws := filepath.Join(outDir, "workspace")
	logf, err := os.Create(filepath.Join(outDir, "grade.log"))
	if err != nil {
		return err
	}
	defer logf.Close()
	var task record.Task
	for _, t := range g.Meta.Tasks {
		if t.ID == r.Task {
			task = t
		}
	}
	r.Targeted = task.Targets
	r.Violations = map[string]bool{}

	bin, err := filepath.Abs(filepath.Join(outDir, "bin", "stockroom"+exe()))
	if err != nil {
		return err
	}
	r.Builds = g.build(r.Arm, ws, bin, logf)
	mod, err := g.testModule(outDir, r.Task)
	if err != nil {
		return err
	}
	env := append(g.env(), "STOCKROOM_BIN="+bin)
	if r.Builds {
		r.Acceptance = runTests(mod, "./accept/", env, logf)
		r.Holdout = runTests(mod, "./holdout/", env, logf)
	} else {
		r.Acceptance = record.Tests{Failed: 1}
	}
	for _, rule := range g.Meta.Rules {
		switch rule.Method {
		case "test":
			if !r.Builds {
				continue // a build failure fails the task; it says nothing about the rule
			}
			t := runTests(mod, "./rules/"+rule.ID+"/", env, logf)
			r.Violations[rule.ID] = t.Failed > 0
		case "detector":
			v, err := g.detector(mod, rule.ID, r.Arm, ws, logf)
			if err != nil {
				fmt.Fprintf(logf, "detector %s: %v\n", rule.ID, err)
				r.Note = strings.TrimSpace(r.Note + " detector " + rule.ID + " failed")
				continue
			}
			r.Violations[rule.ID] = v
		}
	}
	return nil
}

func exe() string {
	if os.PathSeparator == '\\' {
		return ".exe"
	}
	return ""
}

func (g *Grader) env() []string {
	path := os.Getenv("PATH")
	if g.ToolDir != "" {
		path = g.ToolDir + string(os.PathListSeparator) + path
	}
	var out []string
	for _, e := range os.Environ() {
		if k, _, _ := strings.Cut(e, "="); !strings.EqualFold(k, "PATH") && !strings.EqualFold(k, "GOFLAGS") {
			out = append(out, e)
		}
	}
	return append(out, "PATH="+path, "GOFLAGS=-mod=mod", "GOWORK=off")
}

// build builds the final workspace with the arm's command.
func (g *Grader) build(arm, ws, bin string, logf *os.File) bool {
	var cmd *exec.Cmd
	if arm == record.ArmV {
		// exec looks a bare name up in this process's PATH, not in cmd.Env: name the tool.
		vulcan := filepath.Join(g.ToolDir, "vulcan"+exe())
		if g.ToolDir == "" {
			vulcan = "vulcan"
		}
		cmd = exec.Command(vulcan, "build", "-o", bin)
	} else {
		cmd = exec.Command("go", "build", "-o", bin, "./cmd/stockroom")
	}
	cmd.Dir = ws
	cmd.Env = append(g.env(), "GOPROXY=off")
	out, err := cmd.CombinedOutput()
	fmt.Fprintf(logf, "== build (%s)\n%s\n", arm, out)
	return err == nil
}

// testModule writes a Go module with the task's acceptance tests, the holdout tests, and the
// rules, which imports the repository for poc/stockroomtest and poc/harness/detect.
func (g *Grader) testModule(outDir, task string) (string, error) {
	mod := filepath.Join(outDir, "grade")
	if err := os.RemoveAll(mod); err != nil {
		return "", err
	}
	repo, err := filepath.Abs(g.Repo)
	if err != nil {
		return "", err
	}
	gomod := fmt.Sprintf("module grading\n\ngo 1.26\n\nrequire github.com/robot-accomplice/vulcan v0.0.0\n\nreplace github.com/robot-accomplice/vulcan => %s\n", filepath.ToSlash(repo))
	if err := writeFile(filepath.Join(mod, "go.mod"), []byte(gomod)); err != nil {
		return "", err
	}
	sum, err := os.ReadFile(filepath.Join(repo, "go.sum"))
	if err != nil {
		return "", err
	}
	if err := writeFile(filepath.Join(mod, "go.sum"), sum); err != nil {
		return "", err
	}
	for src, dst := range map[string]string{
		filepath.Join(g.Material, "tasks", task, "accept"): "accept",
		filepath.Join(g.Material, "holdout"):               "holdout",
		filepath.Join(g.Material, "rules"):                 "rules",
	} {
		if err := copyDir(src, filepath.Join(mod, dst)); err != nil {
			return "", err
		}
	}
	return mod, nil
}

// runTests runs a test package with -json and counts the top-level tests.
func runTests(mod, pkg string, env []string, logf *os.File) record.Tests {
	if _, err := os.Stat(filepath.Join(mod, filepath.FromSlash(pkg))); err != nil {
		return record.Tests{}
	}
	cmd := exec.Command("go", "test", "-json", "-count=1", pkg)
	cmd.Dir = mod
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run()
	fmt.Fprintf(logf, "== go test %s\n", pkg)
	var t record.Tests
	sc := bufio.NewScanner(bytes.NewReader(out.Bytes()))
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var ev struct {
			Action, Test, Output string
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			fmt.Fprintln(logf, sc.Text())
			continue
		}
		if ev.Output != "" {
			logf.WriteString(ev.Output)
		}
		if ev.Test == "" || strings.Contains(ev.Test, "/") {
			if ev.Test == "" && ev.Action == "fail" && t.Passed+t.Failed == 0 {
				t.Failed++ // the package failed to build or to start
			}
			continue
		}
		switch ev.Action {
		case "pass":
			t.Passed++
		case "fail":
			t.Failed++
		}
	}
	return t
}

// detector runs a rule's detector: exit 1 is a violation, 0 none, anything else an error.
func (g *Grader) detector(mod, id, arm, ws string, logf *os.File) (bool, error) {
	cmd := exec.Command("go", "run", "./rules/"+id, "-arm", arm, "-ws", ws)
	cmd.Dir = mod
	cmd.Env = g.env()
	out, err := cmd.CombinedOutput()
	fmt.Fprintf(logf, "== detector %s\n%s\n", id, out)
	if err == nil {
		return false, nil
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("%v", err)
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func copyDir(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		return nil // an optional part of the material
	}
	return filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return writeFile(filepath.Join(dst, rel), b)
	})
}
