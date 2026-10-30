package contract

// Requires panics when false.
func Requires(c bool) {
	if !c {
		panic("requires")
	}
}

// Ensures panics when false.
func Ensures(c func() bool) {
	if r := recover(); r != nil {
		panic(r)
	}
	if !c() {
		panic("ensures")
	}
}
