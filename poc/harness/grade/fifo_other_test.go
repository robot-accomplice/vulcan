//go:build !unix

package grade

import "errors"

// mkfifo makes a FIFO: this host has none.
func mkfifo(string) error { return errors.ErrUnsupported }
