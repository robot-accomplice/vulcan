package bus

import (
	"errors"
	"testing"

	"shop/shop"
)

// A failing handler gets the event again, then the event goes to the dead letter.
func TestRetryThenDeadLetter(t *testing.T) {
	b := New()
	calls := 0
	b.Subscribe(func(shop.OrderShipped) error { calls++; return errors.New("no") })
	var dead []shop.OrderShipped
	b.DeadLetter = func(ev shop.OrderShipped, _ error) { dead = append(dead, ev) }

	b.Publish(shop.OrderShipped{Order: "o-1"})
	if calls != MaxAttempts {
		t.Fatalf("calls = %d, want %d", calls, MaxAttempts)
	}
	if len(dead) != 1 {
		t.Fatalf("dead letters = %d, want 1", len(dead))
	}
}
