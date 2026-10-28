package schedule

import "testing"

// Round 1's committed schedule must stay the same: Make is MakeArms with V and G.
func TestMakeStable(t *testing.T) {
	a := Make([]string{"T01", "T02"}, 3, 20261028)
	b := MakeArms([]string{"T01", "T02"}, []string{"V", "G"}, 3, 20261028)
	if len(a) != 12 || len(b) != 12 {
		t.Fatal("size")
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("slot %d: %+v != %+v", i, a[i], b[i])
		}
	}
	c := MakeArms([]string{"T01"}, []string{"P", "L", "V"}, 2, 1)
	if len(c) != 6 || c[5].Order != 6 {
		t.Fatalf("%+v", c)
	}
}

func TestRetest(t *testing.T) {
	base := MakeArms([]string{"T01", "T02", "T03", "T04", "T05"}, []string{"P", "L"}, 2, 7)
	out := AddRetest(base, 0.1, 7)
	n := 0
	for _, s := range out {
		if s.Index > RetestOffset {
			n++
		}
	}
	if len(out) != 22 || n != 2 {
		t.Fatalf("%d slots, %d retests", len(out), n)
	}
}
