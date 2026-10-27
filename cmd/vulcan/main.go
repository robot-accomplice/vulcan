// Command vulcan is the Vulcan compiler and its tools (spec §10). This is the proof-of-concept
// subset of release 1 (docs/poc/plan.md).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	vulcanroot "github.com/robot-accomplice/vulcan"
	"github.com/robot-accomplice/vulcan/internal/check"
	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/driver"
	"github.com/robot-accomplice/vulcan/internal/gen"
)

const usage = `vulcan is the Vulcan compiler (proof of concept).

Usage:
	vulcan check [--json] [dir]           check the project: architecture, wiring, and Go types
	vulcan build [--json] [-o file] [dir] check, then build the program
	vulcan run [dir] [-- flags]           build, then run the program with flags
	vulcan test [--json] [-v] [-run re] [Leaf|library ...]
	                                      check with tests, then run the tests of each leaf and library
	vulcan mod tidy                       add each module that the code imports to go.mod and go.sum
	vulcan guide                          print the primer: what Vulcan adds to Go

The project root is the directory that holds system.vulcan; dir defaults to the current one.
Exit codes: 0 success, 1 diagnostics or failures, 2 a usage error or a command that this release
does not have.
`

var deferred = map[string]bool{"fmt": true, "explain": true, "doc": true, "diagram": true, "archdiff": true, "fake": true, "trace": true, "context": true, "query": true, "audit": true}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(stdout, usage)
		return 0
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "check", "build":
		return checkOrBuild(cmd, rest, stdout, stderr)
	case "run":
		return runCmd(rest, stdout, stderr)
	case "test":
		return testCmd(rest, stdout, stderr)
	case "mod":
		return modCmd(rest, stdout, stderr)
	case "guide":
		fmt.Fprint(stdout, vulcanroot.Guide)
		return 0
	}
	if deferred[cmd] {
		fmt.Fprintf(stderr, "vulcan %s is not in this release of Vulcan (the PoC subset of release 1)\n", cmd)
		return 2
	}
	fmt.Fprintf(stderr, "vulcan: unknown command %q\n\n%s", cmd, usage)
	return 2
}

func report(r *driver.Result, asJSON bool, stdout, stderr io.Writer) {
	if asJSON {
		_ = r.Diags.PrintJSON(stdout)
		return
	}
	r.Diags.Print(stderr)
}

func load(dir string, tests bool, stderr io.Writer) (*driver.Result, bool) {
	root, err := driver.FindRoot(dir)
	if err != nil {
		fmt.Fprintf(stderr, "vulcan: %v\n", err)
		return nil, false
	}
	r, err := driver.Check(root, tests)
	if err != nil {
		fmt.Fprintf(stderr, "vulcan: %v\n", err)
		return nil, false
	}
	return r, true
}

func checkOrBuild(cmd string, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print diagnostics as JSON")
	out := ""
	if cmd == "build" {
		fs.StringVar(&out, "o", "", "write the program to this file")
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}
	r, ok := load(dir, cmd == "check", stderr)
	if !ok {
		return 1
	}
	report(r, *asJSON, stdout, stderr)
	if !r.OK {
		return 1
	}
	if cmd == "build" {
		bin := driver.Binary(r.P)
		if out != "" {
			b, err := os.ReadFile(bin)
			if err == nil {
				err = os.WriteFile(out, b, 0o755)
			}
			if err != nil {
				fmt.Fprintf(stderr, "vulcan: %v\n", err)
				return 1
			}
			bin = out
		}
		if !*asJSON {
			fmt.Fprintf(stderr, "built %s\n", bin)
		}
	}
	return 0
}

func runCmd(args []string, stdout, stderr io.Writer) int {
	dir := "."
	var flags []string
	for i, a := range args {
		if a == "--" {
			flags = args[i+1:]
			args = args[:i]
			break
		}
	}
	if len(args) > 0 {
		dir = args[0]
	}
	r, ok := load(dir, false, stderr)
	if !ok {
		return 1
	}
	report(r, false, stdout, stderr)
	if !r.OK {
		return 1
	}
	c := exec.Command(driver.Binary(r.P), flags...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, stdout, stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintf(stderr, "vulcan: %v\n", err)
		return 1
	}
	return 0
}

func testCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "print diagnostics as JSON, and go test -json output")
	verbose := fs.Bool("v", false, "verbose test output")
	runRE := fs.String("run", "", "run only the tests that match")
	race := fs.Bool("race", true, "run with the race detector where the host has it")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r, ok := load(".", true, stderr)
	if !ok {
		return 1
	}
	var pkgs []string
	if fs.NArg() == 0 {
		for _, l := range r.P.Leaves {
			pkgs = append(pkgs, "./"+l.Dir)
		}
		for _, l := range r.P.Libs {
			pkgs = append(pkgs, "./"+l.Dir)
		}
	} else {
		for _, name := range fs.Args() {
			found := false
			for _, l := range r.P.Leaves {
				if strings.EqualFold(l.Name, name) {
					pkgs, found = append(pkgs, "./"+l.Dir), true
				}
			}
			for _, l := range r.P.Libs {
				if l.GoName == name {
					pkgs, found = append(pkgs, "./"+l.Dir), true
				}
			}
			if !found {
				fmt.Fprintf(stderr, "vulcan test: no leaf or library %s\n", name)
				return 2
			}
		}
	}
	goArgs := []string{"test"}
	if *race {
		if driver.RaceSupported() {
			goArgs = append(goArgs, "-race")
		} else {
			r.Diags.Warnf(diag.Pos{}, "K0705", "vulcan test ran without the race detector, because this host cannot run it (it needs cgo and a C compiler); run the tests in CI on a host that has it")
		}
	}
	report(r, *asJSON, stdout, stderr)
	if !r.OK {
		return 1
	}
	if *verbose {
		goArgs = append(goArgs, "-v")
	}
	if *asJSON {
		goArgs = append(goArgs, "-json")
	}
	if *runRE != "" {
		goArgs = append(goArgs, "-run", *runRE)
	}
	goArgs = append(goArgs, pkgs...)
	c := exec.Command("go", goArgs...)
	c.Dir = gen.BuildDir(r.P)
	c.Env = check.Env()
	c.Stdout, c.Stderr = stdout, stderr
	if err := c.Run(); err != nil {
		return 1
	}
	return 0
}

func modCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "tidy" {
		fmt.Fprintf(stderr, "vulcan mod: this release has only vulcan mod tidy\n")
		return 2
	}
	root, err := driver.FindRoot(".")
	if err != nil {
		fmt.Fprintf(stderr, "vulcan: %v\n", err)
		return 1
	}
	out, err := driver.ModTidy(root)
	if err != nil {
		fmt.Fprintf(stderr, "vulcan mod tidy: %v\n%s", err, out)
		return 1
	}
	return 0
}
