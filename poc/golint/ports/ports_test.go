package ports_test

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/robot-accomplice/vulcan/poc/golint/internal/lintest"
	"github.com/robot-accomplice/vulcan/poc/golint/ports"
)

// TestPorts runs the analyzer on the base system of the conformance cases with the changes of
// testdata/ports.txtar; each // want comment names the code of an expected diagnostic.
func TestPorts(t *testing.T) {
	base := lintest.Read(t, filepath.Join("..", "conformance", "testdata", "base.txtar"))
	dir := lintest.Write(t, base, lintest.Read(t, filepath.Join("testdata", "ports.txtar")))
	analysistest.Run(t, dir, ports.Analyzer, "./...")
}
