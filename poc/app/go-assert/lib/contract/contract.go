// Package contract reports the failure of a run-time contract. The checks are always on.
//
// A check is written as an ordinary if statement at the place the contract names, so that the
// message is built only when the check fails:
//
//	if amount <= 0 {
//		contract.Fail("K01", "amount %d must be more than 0", amount)
//	}
package contract

import (
	"fmt"
	"runtime"
)

// Violation is the panic value of a failed contract.
type Violation struct {
	ID      string // the contract id, such as K01
	Place   string // file:line of the failed check
	Message string // the values involved
}

// Error makes Violation an error, so that a recover can use errors.As.
func (v *Violation) Error() string {
	return fmt.Sprintf("contract %s violated at %s: %s", v.ID, v.Place, v.Message)
}

// Fail stops the running code with a panic that names the contract id, the place of the check
// (the caller of Fail), and the values involved. Nothing is undone.
func Fail(id, format string, args ...any) {
	place := "unknown"
	if _, file, line, ok := runtime.Caller(1); ok {
		place = fmt.Sprintf("%s:%d", file, line)
	}
	panic(&Violation{ID: id, Place: place, Message: fmt.Sprintf(format, args...)})
}
