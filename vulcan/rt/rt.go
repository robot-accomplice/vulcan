// Package rt is the runtime that generated code uses (spec Appendix C). Code that a developer
// writes must not import it (K0118); it imports package vulcan instead.
package rt

import (
	"fmt"
	"os"
	"strings"
)

// Value is one named value that a contract failure reports.
type Value struct {
	Name, Text string
}

// ContractFailure is the panic value of a failed contract (spec §7).
type ContractFailure struct {
	Kind     string // "requires", "ensures", "invariant", "limit", or "panic"
	Contract string // the source text of the clause
	Position string // the .vulcan position of the clause
	Caller   string // the function or port method that the clause belongs to
	Values   []Value
	Cause    any
}

func (f *ContractFailure) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "contract failure: %s %s: %s", f.Caller, f.Kind, f.Contract)
	if f.Position != "" {
		fmt.Fprintf(&b, " (%s)", f.Position)
	}
	for _, v := range f.Values {
		fmt.Fprintf(&b, "\n\t%s = %s", v.Name, v.Text)
	}
	return b.String()
}

// Unwrap returns Cause when it is an error.
func (f *ContractFailure) Unwrap() error {
	if err, ok := f.Cause.(error); ok {
		return err
	}
	return nil
}

// Fail panics with a *ContractFailure.
func Fail(kind, contract, position, caller string, values ...Value) {
	f := &ContractFailure{Kind: kind, Contract: contract, Position: position, Caller: caller, Values: values}
	if ExitOnFailure {
		// on_contract_failure: exit (§7.4): no deferred call and no recover runs.
		fmt.Fprintln(os.Stderr, f.Error())
		os.Exit(3)
	}
	panic(f)
}

// ExitOnFailure makes a contract failure end the process with exit code 3 (§7.4).
var ExitOnFailure bool

// Show formats a value for a contract failure report.
func Show(name string, v any) Value {
	return Value{Name: name, Text: fmt.Sprintf("%#v", v)}
}
