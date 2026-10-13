// Package audit records each charge and refund that Billing makes, and reads
// the trail back per order.
package audit

import (
	"context"
	"sync"
	"time"

	"shop/shop"
)

// Kind says what Billing did.
type Kind string

const (
	Charge Kind = "charge"
	Refund Kind = "refund"
)

// Entry is one audit line.
type Entry struct {
	Order    shop.OrderID    `json:"order"`
	Customer shop.CustomerID `json:"customer"`
	Kind     Kind            `json:"kind"`
	Amount   shop.Money      `json:"amount"`
	At       time.Time       `json:"at"`
}

// Memory keeps audit lines in memory. Use it in tests and local runs that have no database.
type Memory struct {
	mu      sync.Mutex
	entries []Entry
}

// Record appends e.
func (m *Memory) Record(_ context.Context, e Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, e)
	return nil
}

// ForOrder returns the lines for order, oldest first.
func (m *Memory) ForOrder(_ context.Context, order shop.OrderID) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Entry{}
	for _, e := range m.entries {
		if e.Order == order {
			out = append(out, e)
		}
	}
	return out, nil
}
