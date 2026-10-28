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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/robot-accomplice/vulcan/poc/harness/detect"
	"github.com/robot-accomplice/vulcan/poc/harness/measure"
	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// Grader grades runs.
type Grader struct {
	Repo     string // the root of the vulcan repository, for the test module
	Material string // the unsealed hidden material
	ToolDir  string // the directory of the vulcan command
	Meta     *record.Meta

	prepared string   // the directory of the compiled tests and detectors (Prepare)
	Only     []string // Validate checks only the items whose kind/id starts with one of these
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
	r.ConfigAllowed = task.AllowConfig
	r.Violations = map[string]bool{}
	g.measures(outDir, r, logf)

	bin, err := filepath.Abs(filepath.Join(outDir, "bin", "stockroom"+exe()))
	if err != nil {
		return err
	}
	r.Builds = g.build(r.Arm, ws, bin, logf)
	if g.prepared == "" {
		if err := g.Prepare(filepath.Join(outDir, "grade")); err != nil {
			return err
		}
	}
	env := append(g.env(), "STOCKROOM_BIN="+bin)
	if r.Builds {
		if r.Task != "" {
			r.Acceptance = runBinary(filepath.Join(g.prepared, "accept", r.Task+".test"+exe()), env, logf)
		}
		r.Holdout = runBinary(filepath.Join(g.prepared, "holdout.test"+exe()), env, logf)
	} else {
		r.Acceptance = record.Tests{Failed: 1}
	}
	for _, rule := range g.Meta.Rules {
		switch rule.Method {
		case "test":
			if !r.Builds {
				continue // a build failure fails the task; it says nothing about the rule
			}
			t := runBinary(filepath.Join(g.prepared, "rules", rule.ID+".test"+exe()), env, logf)
			r.Violations[rule.ID] = t.Failed > 0
		case "detector":
			v, err := g.detector(filepath.Join(g.prepared, "rules", rule.ID+exe()), rule.ID, r.Arm, ws, logf)
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

// measures fills the code measures of round 2 (O7): final minus base, in the product source.
// A workspace that does not load counts 0 dropped errors, with a note.
func (g *Grader) measures(outDir string, r *record.Run, logf *os.File) {
	ws, base := filepath.Join(outDir, "workspace"), filepath.Join(outDir, "base")
	if _, err := os.Stat(base); err != nil {
		return // a run without its base (round 1 archives keep it; tests may not)
	}
	if g.ToolDir != "" && !strings.Contains(os.Getenv("PATH"), g.ToolDir) {
		// detect.Load runs `vulcan check` in this process for arm V.
		os.Setenv("PATH", g.ToolDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	r.CloneLines = measure.CloneLines(ws) - measure.CloneLines(base)
	masked := func(dir string) (int, error) {
		s, err := detect.Load(r.Arm, dir)
		if err != nil {
			return 0, err
		}
		return measure.MaskedErrors(s), nil
	}
	after, errA := masked(ws)
	before, errB := masked(base)
	if errA != nil || errB != nil {
		fmt.Fprintf(logf, "masked errors: %v %v\n", errA, errB)
		r.Note = strings.TrimSpace(r.Note + " masked errors not measured")
		return
	}
	r.MaskedErrors = after - before
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
	_ = os.MkdirAll(filepath.Dir(bin), 0o755)
	out, err := cmd.CombinedOutput()
	fmt.Fprintf(logf, "== build (%s)\n%s\n", arm, out)
	return err == nil
}

// detector runs a rule's compiled detector: exit 1 is a violation, 0 none, anything else an
// error.
func (g *Grader) detector(bin, id, arm, ws string, logf *os.File) (bool, error) {
	cmd := exec.Command(bin, "-arm", arm, "-ws", ws)
	cmd.Env = g.env()
	out, err := cmd.CombinedOutput()
	fmt.Fprintf(logf, "== detector %s\n%s\n", id, out)
	if err == nil {
		return false, nil
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return true, nil
	}
	return false, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
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
