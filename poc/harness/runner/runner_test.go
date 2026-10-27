package runner

import "testing"

func TestChangedLines(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"x\ny\nz\n", "x\ny\nz\n", 0},
		{"x\ny\nz\n", "x\nY\nz\n", 2},
		{"x\nz\n", "x\ny\nz\n", 1},
		{"", "a\nb\n", 2},
		{"a\nb\nc\nd\n", "b\nc\n", 2},
	} {
		if got := changedLines(splitLines([]byte(c.a)), splitLines([]byte(c.b))); got != c.want {
			t.Errorf("%q -> %q: %d, want %d", c.a, c.b, got, c.want)
		}
	}
	if unitOf("api/orders/orders.go") != "api/orders" || unitOf("system.vulcan") != "system.vulcan" {
		t.Fatal("unitOf")
	}
}
