package shop

import (
	"errors"
	"fmt"
)

// Error codes. The HTTP API maps each to a status (docs/poc/stockroom.md).
const (
	CodeInvalid     = "invalid"
	CodeNotFound    = "not_found"
	CodeConflict    = "conflict"
	CodeOutOfStock  = "out_of_stock"
	CodeDeclined    = "payment_declined"
	CodeUnavailable = "payment_unavailable"
)

// Error is an error with a code.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Errorf returns an error with a code.
func Errorf(code, format string, args ...any) error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// CodeOf returns the code of an error, or "" for an error without one.
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
