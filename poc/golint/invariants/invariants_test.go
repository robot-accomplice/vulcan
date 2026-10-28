package invariants_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"
	"github.com/robot-accomplice/vulcan/poc/golint/invariants"
)

// TestInvariants runs the analyzer on the base system of the conformance cases with the changes of
// testdata/invariants.txtar; each // want comment names the code of an expected diagnostic.
func TestInvariants(t *testing.T) {
	base := lintest.Read(t, filepath.Join("..", "conformance", "testdata", "base.txtar"))
	dir := lintest.Write(t, base, lintest.Read(t, filepath.Join("testdata", "invariants.txtar")))
	analysistest.Run(t, dir, invariants.Analyzer, "./...")
}
