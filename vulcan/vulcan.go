// Package vulcan is the runtime package that Vulcan code and tests import (spec Appendix C).
// Release 1 declares only ContractFailure, Value, and Bind.
package vulcan

import (
	"fmt"
	"testing"

	"github.com/robot-accomplice/vulcan/vulcan/rt"
)

// ContractFailure is the panic value of a failed contract (spec §7).
type ContractFailure = rt.ContractFailure

// Value is one named value that a contract failure reports.
type Value = rt.Value

// Bind binds impl to a port name for one test, and restores the binding when the test ends
// (spec §8). The compiler checks impl against the port (K0108).
func Bind[P any](t testing.TB, port *P, impl any) {
	t.Helper()
	v, ok := impl.(P)
	if !ok {
		panic(fmt.Sprintf("vulcan.Bind: %T does not satisfy the port", impl))
	}
	old := *port
	*port = v
	t.Cleanup(func() { *port = old })
}
