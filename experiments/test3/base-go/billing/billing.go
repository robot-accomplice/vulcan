// Package billing charges and refunds customers, and keeps an in-memory ledger.
package billing

import (
	"errors"
	"fmt"
	"sync"

	"shop/shop"
)

var (
	// ErrNotCharged means a refund names an order with no charge.
	ErrNotCharged = errors.New("billing: order not charged")
	// ErrOverRefund means a refund would return more than was charged.
	ErrOverRefund = errors.New("billing: refund exceeds charge")
	// ErrBadAmount means an amount is zero or negative.
	ErrBadAmount = errors.New("billing: amount must be positive")
)

type entry struct {
	customer shop.CustomerID
	charged  shop.Money
	refunded shop.Money
}

// Ledger records charges and refunds per order.
type Ledger struct {
	mu      sync.Mutex
	byOrder map[shop.OrderID]*entry
}

// NewLedger returns an empty ledger.
func NewLedger() *Ledger { return &Ledger{byOrder: map[shop.OrderID]*entry{}} }

// Charge takes amount from the customer for the order.
func (l *Ledger) Charge(customer shop.CustomerID, order shop.OrderID, amount shop.Money) (shop.Receipt, error) {
	if amount <= 0 {
		return shop.Receipt{}, ErrBadAmount
	}
	l.mu.Lock()
	defer l.mu.Unlock()
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
	e.refunded += amount
	return shop.Receipt{Customer: e.customer, Order: order, Amount: amount}, nil
}
