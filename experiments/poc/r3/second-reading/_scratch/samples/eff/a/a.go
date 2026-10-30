package a

import (
	"net"
	"time"
)

func Dial() { net.Dial("tcp", "x:1") }

func Now() time.Time { return time.Now() }

// Map has the effects of its parameter f.
//
//vl:poly f
func Map(xs []int, f func(int) int) {
	for i := range xs {
		xs[i] = f(xs[i])
	}
}
