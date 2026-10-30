package target

import (
	"testing"
	"testing/quick"
)

func TestDerivedClamp(t *testing.T) {
	cfg := &quick.Config{MaxCount: 200000, Values: small}
	if err := quick.Check(func(x, lo, hi int) bool {
		if !(lo <= hi) {
			return true
		}
		Clamp(x, lo, hi)
		return true
	}, cfg); err != nil {
		t.Fatal(err)
	}
}
