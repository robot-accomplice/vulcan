//go:build unix

package grade

import "syscall"

// mkfifo makes a FIFO.
func mkfifo(p string) error { return syscall.Mkfifo(p, 0o644) }
