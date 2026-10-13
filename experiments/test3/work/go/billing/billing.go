// Package billing charges and refunds customers, and keeps an in-memory ledger.
// Each charge and refund also writes one audit line.
package billing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"shop/audit"
	"shop/shop"
)

// AuditTimeout is the longest Billing waits for one audit write.
const AuditTimeout = 5 * time.Second

// Auditor stores audit lines.
type Auditor interface {
	Record(ctx context.Context, e audit.Entry) error
}

var (
	// ErrNotCharged means a refund names an order with no charge.
	ErrNotCharged = errors.New("billing: order not charged")
	// ErrOverRefund means a refund would return more than was charged.
	ErrOverRefund = errors.New("billing: refund exceeds charge")
	// ErrBadAmount means an amount is zero or negative.
	ErrBadAmount = errors.New("billing: amount must be positive")
	// ErrAudit means the audit line was not written, so the charge or refund was not made.
	ErrAudit = errors.New("billing: audit write failed")
)

type entry struct {
	customer shop.CustomerID
	charged  shop.Money
	refunded shop.Money
}

// Ledger records charges and refunds per order.
//
// A charge or refund takes effect only after its audit line is written. The
// audit write happens under the ledger lock, so no other charge or refund can
// use a change that is not yet audited.
type Ledger struct {
	auditor Auditor

	mu      sync.Mutex
	byOrder map[shop.OrderID]*entry
}

// NewLedger returns an empty ledger that writes audit lines to a.
func NewLedger(a Auditor) *Ledger {
	if a == nil {
		panic("billing: NewLedger needs an Auditor")
	}
	return &Ledger{auditor: a, byOrder: map[shop.OrderID]*entry{}}
}

// record writes one audit line. The caller holds l.mu.
func (l *Ledger) record(kind audit.Kind, customer shop.CustomerID, order shop.OrderID, amount shop.Money) error {
	ctx, cancel := context.WithTimeout(context.Background(), AuditTimeout)
	defer cancel()
	e := audit.Entry{Order: order, Customer: customer, Kind: kind, Amount: amount, At: time.Now().UTC()}
	if err := l.auditor.Record(ctx, e); err != nil {
		return fmt.Errorf("%s %s of %d: %w: %w", kind, order, amount, ErrAudit, err)
	}
	return nil
}

// Charge takes amount from the customer for the order.
func (l *Ledger) Charge(customer shop.CustomerID, order shop.OrderID, amount shop.Money) (shop.Receipt, error) {
	if amount <= 0 {
		return shop.Receipt{}, ErrBadAmount
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.record(audit.Charge, customer, order, amount); err != nil {
		return shop.Receipt{}, err
	}
	e, ok := l.byOrder[order]
	if !ok {
		e = &entry{customer: customer}
		l.byOrder[order] = e
	}
	e.charged += amount
	return shop.Receipt{Customer: customer, Order: order, Amount: amount}, nil
}

// Refund returns amount to the customer who paid for the order.
// The total refunded never exceeds the total charged.
func (l *Ledger) Refund(order shop.OrderID, amount shop.Money) (shop.Receipt, error) {
	if amount <= 0 {
		return shop.Receipt{}, ErrBadAmount
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.byOrder[order]
	if !ok || e.charged == 0 {
		return shop.Receipt{}, fmt.Errorf("refund %s: %w", order, ErrNotCharged)
	}
	if e.refunded+amount > e.charged {
		return shop.Receipt{}, fmt.Errorf("refund %s of %d (charged %d, refunded %d): %w",
			order, amount, e.charged, e.refunded, ErrOverRefund)
	}
	if err := l.record(audit.Refund, e.customer, order, amount); err != nil {
		return shop.Receipt{}, err
	}
	e.refunded += amount
	return shop.Receipt{Customer: e.customer, Order: order, Amount: amount}, nil
}
