package billing

import (
	"context"
	"errors"
	"testing"

	"shop/audit"
)

// A refund must never return money that was not taken.
func TestRefundNeverExceedsCharge(t *testing.T) {
	l := NewLedger(&audit.Memory{})
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
	if _, err := NewLedger(&audit.Memory{}).Refund("o9", 1); !errors.Is(err, ErrNotCharged) {
		t.Fatalf("got %v, want ErrNotCharged", err)
	}
}

func TestAmountsMustBePositive(t *testing.T) {
	l := NewLedger(&audit.Memory{})
	if _, err := l.Charge("c1", "o1", 0); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("charge 0: got %v", err)
	}
	if _, err := l.Refund("o1", -5); !errors.Is(err, ErrBadAmount) {
		t.Fatalf("refund -5: got %v", err)
	}
}

// failingAuditor fails every write, as a down database does.
type failingAuditor struct{}

func (failingAuditor) Record(context.Context, audit.Entry) error { return errors.New("db down") }

// Every charge and refund that takes effect leaves one audit line, in order.
func TestChargeAndRefundAreAudited(t *testing.T) {
	a := &audit.Memory{}
	l := NewLedger(a)
	if _, err := l.Charge("c1", "o1", 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Refund("o1", 300); err != nil {
		t.Fatal(err)
	}
	got, _ := a.ForOrder(context.Background(), "o1")
	if len(got) != 2 ||
		got[0].Kind != audit.Charge || got[0].Amount != 1000 || got[0].Customer != "c1" ||
		got[1].Kind != audit.Refund || got[1].Amount != 300 || got[1].Customer != "c1" {
		t.Fatalf("audit trail: %+v", got)
	}
}

// A refund that Billing rejects moved no money, so it must not appear in the audit trail.
func TestRejectedRefundIsNotAudited(t *testing.T) {
	a := &audit.Memory{}
	l := NewLedger(a)
	if _, err := l.Charge("c1", "o1", 100); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Refund("o1", 101); !errors.Is(err, ErrOverRefund) {
		t.Fatalf("got %v, want ErrOverRefund", err)
	}
	if got, _ := a.ForOrder(context.Background(), "o1"); len(got) != 1 {
		t.Fatalf("audit trail: %+v", got)
	}
}

// If the audit line cannot be written, the money must not move: an unaudited
// charge or refund is exactly what the audit trail exists to prevent.
func TestAuditFailureLeavesLedgerUnchanged(t *testing.T) {
	l := NewLedger(failingAuditor{})
	if _, err := l.Charge("c1", "o1", 100); !errors.Is(err, ErrAudit) {
		t.Fatalf("charge: got %v, want ErrAudit", err)
	}
	// The failed charge did not count, so there is nothing to refund.
	l.auditor = &audit.Memory{}
	if _, err := l.Refund("o1", 1); !errors.Is(err, ErrNotCharged) {
		t.Fatalf("refund after failed charge: got %v, want ErrNotCharged", err)
	}

	if _, err := l.Charge("c1", "o1", 100); err != nil {
		t.Fatal(err)
	}
	l.auditor = failingAuditor{}
	if _, err := l.Refund("o1", 100); !errors.Is(err, ErrAudit) {
		t.Fatalf("refund: got %v, want ErrAudit", err)
	}
	// The failed refund did not count, so the full amount is still refundable.
	l.auditor = &audit.Memory{}
	if _, err := l.Refund("o1", 100); err != nil {
		t.Fatalf("full refund after failed refund: %v", err)
	}
}
