package analysis2

import (
	"math"
	"testing"
)

// The 0.95 quantiles of Student's t from a printed table.
func TestTQuantile(t *testing.T) {
	for df, want := range map[int]float64{1: 6.3138, 2: 2.9200, 5: 2.0150, 10: 1.8125, 23: 1.7139, 31: 1.6955, 100: 1.6602} {
		if got := tQuantile95(df); math.Abs(got-want) > 5e-4 {
			t.Errorf("df %d: %.4f, want %.4f", df, got, want)
		}
	}
}

func TestSignFlip(t *testing.T) {
	// All 5 positive: only the all-positive flip reaches the mean, p = 1/32.
	if p := signFlipOneSided([]float64{1, 1, 1, 1, 1}); math.Abs(p-1.0/32) > 1e-12 {
		t.Fatalf("p %v", p)
	}
	if p := signFlipOneSided([]float64{1, -1}); p != 0.75 {
		t.Fatalf("p %v", p)
	}
}
