// Command vulcan-lint checks a Go module against Vulcan's model of architecture and contracts:
// the analyzers A1 to A8 of docs/poc/desk-test-r2.md (poc/golint/README.md).
//
//	vulcan-lint ./...
//
// Each diagnostic has a code VL and four digits. vulcan-lint honors no suppression comment.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/robot-accomplice/vulcan/poc/golint/contracts"
	"github.com/robot-accomplice/vulcan/poc/golint/effects"
	"github.com/robot-accomplice/vulcan/poc/golint/imports"
	"github.com/robot-accomplice/vulcan/poc/golint/invariants"
	"github.com/robot-accomplice/vulcan/poc/golint/layout"
	"github.com/robot-accomplice/vulcan/poc/golint/libstate"
	"github.com/robot-accomplice/vulcan/poc/golint/ports"
	"github.com/robot-accomplice/vulcan/poc/golint/wiring"
)

func main() {
	multichecker.Main(
		layout.Analyzer, imports.Analyzer, ports.Analyzer, wiring.Analyzer,
		effects.Analyzer, libstate.Analyzer, contracts.Analyzer, invariants.Analyzer,
	)
}
