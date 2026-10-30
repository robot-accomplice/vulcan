package target

import "derive/contract"

// Clamp has a defect on a path that a unit test may miss.
func Clamp(x, lo, hi int) (r int) {
	contract.Requires(lo <= hi)
	defer contract.Ensures(func() bool { return lo <= r && r <= hi })
	if x < lo {
		return lo
	}
	if x > hi+1 {
		return hi
	}
	return x
}
