package contracts_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/robot-accomplice/vulcan/poc/golint/contracts"
	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"
)

// TestContracts runs the analyzer on the base system of the conformance cases with the changes of
// testdata/contracts.txtar; each // want comment names the code of an expected diagnostic.
func TestContracts(t *testing.T) {
	base := lintest.Read(t, filepath.Join("..", "conformance", "testdata", "base.txtar"))
	dir := lintest.Write(t, base, lintest.Read(t, filepath.Join("testdata", "contracts.txtar")))
	analysistest.Run(t, dir, contracts.Analyzer, "./...")
}
