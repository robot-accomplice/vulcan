package synct

import (
	"testing"
	"testing/synctest"
	"time"
)

func TestFakeClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		time.Sleep(48 * time.Hour)
		if got := time.Since(start); got != 48*time.Hour {
			t.Fatal(got)
		}
		t.Log("bubble start", start.UTC())
	})
}
