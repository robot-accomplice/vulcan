package effects_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/robot-accomplice/vulcan/poc/golint/effects"
	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"
)

// TestEffects runs the analyzer on the base system of the conformance cases with the changes of
// testdata/effects.txtar; each // want comment names the code of an expected diagnostic.
func TestEffects(t *testing.T) {
	base := lintest.Read(t, filepath.Join("..", "conformance", "testdata", "base.txtar"))
	dir := lintest.Write(t, base, lintest.Read(t, filepath.Join("testdata", "effects.txtar")))
	analysistest.Run(t, dir, effects.Analyzer, "./...")
}
