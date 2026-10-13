package billing

import (
	"errors"
	"testing"
)

// A refund must never return money that was not taken.
func TestRefundNeverExceedsCharge(t *testing.T) {
	l := NewLedger()
	if _, err := l.Charge("c1", "o1", 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Refund("o1", 600); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Refund("o1", 401); !errors.Is(err, ErrOverRefund) {
		t.Fatalf("second refund: got %v, want ErrOverRefund", err)
	}
	if _, err := l.Refund("o1", 400); err != nil {
		t.Fatalf("exact remainder must refund: %v", err)
	}
}

func TestRefundWithoutChargeFails(t *testing.T) {
	if _, err := NewLedger().Refund("o9", 1); !errors.Is(err, ErrNotCharged) {
		t.Fatalf("got %v, want ErrNotCharged", err)
	}
}

func TestAmountsMustBePositive(t *testing.T) {
	l := NewLedger()
	if _, err := l.Charge("c1", "o1", 0); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("charge 0: got %v", err)
	}
	if _, err := l.Refund("o1", -5); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("refund -5: got %v", err)
	}
}
