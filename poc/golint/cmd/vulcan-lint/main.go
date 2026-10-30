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
	"os"

	"golang.org/x/tools/go/analysis/multichecker"

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

func main() {
	if len(os.Args) > 1 {
		tool := scaffold.Tool(rulecheck.Edges)
		switch os.Args[1] {
		case "init":
			os.Exit(tool.Init(os.Args[2:], os.Stdout, os.Stderr))
		case "rules":
			os.Exit(tool.Rules(os.Args[2:], os.Stdout, os.Stderr))
		}
	}
	multichecker.Main(
		layout.Analyzer, imports.Analyzer, ports.Analyzer, wiring.Analyzer,
		effects.Analyzer, libstate.Analyzer, contracts.Analyzer, invariants.Analyzer,
		rulecheck.Analyzer,
	)
}
