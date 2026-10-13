// Package orders places, cancels, and ships orders. It keeps orders in memory.
package orders

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"shop/shop"
)

// Billing is what Orders needs from a payment provider.
type Billing interface {
	Charge(customer shop.CustomerID, order shop.OrderID, amount shop.Money) (shop.Receipt, error)
	Refund(order shop.OrderID, amount shop.Money) (shop.Receipt, error)
}

// Status is the state of an order.
type Status string

const (
	Placed    Status = "placed"
	Cancelled Status = "cancelled"
	Shipped   Status = "shipped"
)

var (
	ErrNotFound   = errors.New("orders: order not found")
	ErrBadAmount  = errors.New("orders: amount must be positive")
	ErrCancelled  = errors.New("orders: order is cancelled")
	ErrShipped    = errors.New("orders: order is shipped")
	ErrNoteTooBig = fmt.Errorf("orders: note longer than %d bytes", shop.MaxNoteBytes)
)

// Order is one customer order.
type Order struct {
	ID           shop.OrderID    `json:"id"`
	Customer     shop.CustomerID `json:"customer"`
	Amount       shop.Money      `json:"amount"`
	Status       Status          `json:"status"`
	ShippingNote string          `json:"shippingNote,omitempty"`
}

// Service holds the orders.
type Service struct {
	billing Billing

	mu     sync.Mutex
	nextID int
	orders map[shop.OrderID]*Order
}

// New returns a Service that charges and refunds through b.
func New(b Billing) *Service {
	return &Service{billing: b, orders: map[shop.OrderID]*Order{}}
}

// Place creates an order and charges the customer for it.
func (s *Service) Place(customer shop.CustomerID, amount shop.Money) (Order, error) {
	if amount <= 0 {
		return Order{}, ErrBadAmount
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := shop.OrderID("o-" + strconv.Itoa(s.nextID))
	if _, err := s.billing.Charge(customer, id, amount); err != nil {
		return Order{}, fmt.Errorf("place %s: %w", id, err)
	}
	o := &Order{ID: id, Customer: customer, Amount: amount, Status: Placed}
	s.orders[id] = o
	return *o, nil
}

// Cancel cancels a placed order and refunds its full amount.
func (s *Service) Cancel(id shop.OrderID) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	switch o.Status {
	case Cancelled:
		return Order{}, ErrCancelled
	case Shipped:
		return Order{}, ErrShipped
	}
	if _, err := s.billing.Refund(id, o.Amount); err != nil {
		return Order{}, fmt.Errorf("cancel %s: %w", id, err)
	}
	o.Status = Cancelled
	return *o, nil
}

// Get returns one order.
func (s *Service) Get(id shop.OrderID) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return Order{}, ErrNotFound
	}
	return *o, nil
}

// HandleShipped marks an order shipped. The bus can deliver the same event
// more than once, so a repeat for a shipped order succeeds and changes nothing.
func (s *Service) HandleShipped(ev shop.OrderShipped) error {
	if len(ev.Note) > shop.MaxNoteBytes {
		return ErrNoteTooBig
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[ev.Order]
	if !ok {
		return ErrNotFound
	}
	switch o.Status {
	case Shipped:
		return nil
	case Cancelled:
		return fmt.Errorf("ship %s: %w", ev.Order, ErrCancelled)
	}
	o.Status = Shipped
	o.ShippingNote = ev.Note
	return nil
}
