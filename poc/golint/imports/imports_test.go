package imports_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/robot-accomplice/vulcan/poc/golint/imports"
	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"
)

// TestImports runs the analyzer on the base system of the conformance cases with the changes of
// testdata/imports.txtar; each // want comment names the code of an expected diagnostic.
func TestImports(t *testing.T) {
	base := lintest.Read(t, filepath.Join("..", "conformance", "testdata", "base.txtar"))
	dir := lintest.Write(t, base, lintest.Read(t, filepath.Join("testdata", "imports.txtar")))
	analysistest.Run(t, dir, imports.Analyzer, "./...")
}
