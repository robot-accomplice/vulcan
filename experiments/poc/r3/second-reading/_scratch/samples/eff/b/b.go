package b

import "samples/eff/a"

//vl:effects clock
func OK() { a.Now() }

//vl:effects clock
func Hidden() { a.Dial() }

//vl:effects
func PolyPure(xs []int) { a.Map(xs, func(x int) int { return x + 1 }) }

//vl:effects
func PolyNet(xs []int) {
	a.Map(xs, func(x int) int { a.Dial(); return x })
}

// A value stored in a slice: in a language with effects in types this is []func() !net, and legal.
func Stored(fs []func()) {
	for _, f := range fs {
		f()
	}
}

func Param(f func()) { f() }
