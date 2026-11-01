// Command vulcan-lint checks a Go module against Vulcan's model of architecture and contracts:
// the analyzers A1 to A8 of docs/poc/desk-test-r2.md (poc/golint/README.md).
//
//	vulcan-lint ./...
//	vulcan-lint init [--rules SOURCE] [dir]
//	vulcan-lint rules diff|upgrade|lock
//
// Each diagnostic has a code VL and four digits. vulcan-lint honors no suppression comment.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/robot-accomplice/vulcan/internal/rules"
	"github.com/robot-accomplice/vulcan/poc/golint/contracts"
	"github.com/robot-accomplice/vulcan/poc/golint/effects"
	"github.com/robot-accomplice/vulcan/poc/golint/imports"
	"github.com/robot-accomplice/vulcan/poc/golint/invariants"
	"github.com/robot-accomplice/vulcan/poc/golint/layout"
	"github.com/robot-accomplice/vulcan/poc/golint/libstate"
	"github.com/robot-accomplice/vulcan/poc/golint/ports"
	"github.com/robot-accomplice/vulcan/poc/golint/rulecheck"
	"github.com/robot-accomplice/vulcan/poc/golint/scaffold"
	"github.com/robot-accomplice/vulcan/poc/golint/wiring"
)

// foreignRoots reports what brings code in past every check, the same as V's K0002, and stops. It runs
// before the driver, because the go command reads these in a way that differs by host, and a load
// that fails runs no analyzer.
func foreignRoots() {
	root, err := filepath.Abs(".")
	if err != nil {
		return
	}
	for d := root; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			root = d
			break
		}
		if filepath.Dir(d) == d {
			return
		}
	}
	found := rules.ForeignRoots(root)
	for _, f := range found {
		fmt.Fprintf(os.Stderr, "%s:%d:1: VL0002 %s\n", filepath.Join(root, filepath.FromSlash(f.File)), f.Line, f.Msg)
	}
	if len(found) > 0 {
		os.Exit(3)
	}
}

func main() {
	if len(os.Args) > 1 {
		tool := scaffold.Tool(rulecheck.Edges, rulecheck.Findings)
		switch os.Args[1] {
		case "init":
			os.Exit(tool.Init(os.Args[2:], os.Stdout, os.Stderr))
		case "rules":
			os.Exit(tool.Rules(os.Args[2:], os.Stdout, os.Stderr))
		}
	}
	foreignRoots()
	multichecker.Main(
		layout.Analyzer, imports.Analyzer, ports.Analyzer, wiring.Analyzer,
		effects.Analyzer, libstate.Analyzer, contracts.Analyzer, invariants.Analyzer,
		rulecheck.Analyzer,
	)
}
