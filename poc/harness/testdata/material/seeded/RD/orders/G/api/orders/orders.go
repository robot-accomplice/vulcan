// Package orders runs the life cycle of an order: create, pay, ship, cancel, refund.
package orders

import (
	"database/sql"
	"fmt"
	"sync"
	"unicode/utf8"

	"stockroom/lib/shop"
)

// Stock reserves, releases, and ships stock.
type Stock interface {
	Product(sku shop.SKU) (shop.Item, error)
	Reserve(lines []shop.Line) error
	Release(lines []shop.Line) error
	Ship(lines []shop.Line) error
}

// Billing takes and returns the money of an order.
type Billing interface {
	Charge(o shop.Order) (shop.Payment, error)
	Refund(o shop.Order, amount shop.Money) (shop.Payment, error)
	Payments(id shop.OrderID) ([]shop.Payment, error)
}

// Store persists orders.
type Store interface {
	CreateOrder(o shop.Order) (shop.OrderID, error)
	Order(id shop.OrderID) (shop.Order, error)
	OrdersOf(c shop.CustomerID) ([]shop.Order, error)
	SaveOrder(o shop.Order) error
}

// Service runs the life cycle of orders. One lock serializes the changes of orders, so that two
// requests never change one order at once.
type Service struct {
	mu      sync.Mutex
	stock   Stock
	billing Billing
	store   Store
}

// NewService returns the service.
func NewService(stock Stock, billing Billing, store Store) *Service {
	return &Service{stock: stock, billing: billing, store: store}
}

// Create creates an order and reserves its stock: all lines, or none.
func (s *Service) Create(customer shop.CustomerID, wants []shop.Want) (shop.Order, error) {
	if customer == "" || utf8.RuneCountInString(string(customer)) > 64 {
		return shop.Order{}, shop.Errorf(shop.CodeInvalid, "customer must be 1 to 64 characters")
	}
	if len(wants) == 0 {
		return shop.Order{}, shop.Errorf(shop.CodeInvalid, "an order has at least one line")
	}
	seen := map[shop.SKU]bool{}
	var lines []shop.Line
	for _, w := range wants {
		if w.Qty <= 0 || w.Qty > shop.MaxQty {
			return shop.Order{}, shop.Errorf(shop.CodeInvalid, "qty must be more than 0 and at most %d", shop.MaxQty)
		}
		if seen[w.SKU] {
			return shop.Order{}, shop.Errorf(shop.CodeInvalid, "sku %s is on two lines", w.SKU)
		}
		seen[w.SKU] = true
		it, err := s.stock.Product(w.SKU)
		if err != nil {
			return shop.Order{}, err
		}
		lines = append(lines, shop.Line{SKU: w.SKU, Qty: w.Qty, Price: it.Product.Price})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.stock.Reserve(lines); err != nil {
		return shop.Order{}, err
	}
	o := shop.Order{Customer: customer, Status: shop.Pending, Lines: lines, Total: shop.Total(lines)}
	id, err := s.store.CreateOrder(o)
	if err != nil {
		if rerr := s.stock.Release(lines); rerr != nil {
			return shop.Order{}, fmt.Errorf("%w; and the release of its stock failed: %v", err, rerr)
		}
		return shop.Order{}, err
	}
	o.ID = id
	return o, nil
}

// Get returns an order.
func (s *Service) Get(id shop.OrderID) (shop.Order, error) {
	return s.store.Order(id)
}

// List returns the orders of a customer.
func (s *Service) List(customer shop.CustomerID) ([]shop.Order, error) {
	return s.store.OrdersOf(customer)
}

// Pay charges the total of a pending order.
func (s *Service) Pay(id shop.OrderID) (shop.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.store.Order(id)
	if err != nil {
		return shop.Order{}, err
	}
	if o.Status != shop.Pending {
		return shop.Order{}, shop.Errorf(shop.CodeConflict, "order %s is %s, not pending", id, o.Status)
	}
	p, err := s.billing.Charge(o)
	if err != nil {
		return shop.Order{}, err
	}
	o.Status, o.Charged, o.PaymentID = shop.Paid, p.Amount, p.ProviderID
	if err := s.store.SaveOrder(o); err != nil {
		return shop.Order{}, err
	}
	return o, nil
}

// Ship ships a paid order: its units leave the stock.
func (s *Service) Ship(id shop.OrderID) (shop.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.store.Order(id)
	if err != nil {
		return shop.Order{}, err
	}
	if o.Status != shop.Paid {
		return shop.Order{}, shop.Errorf(shop.CodeConflict, "order %s is %s, not paid", id, o.Status)
	}
	if err := s.stock.Ship(o.Lines); err != nil {
		return shop.Order{}, err
	}
	o.Status = shop.Shipped
	if err := s.store.SaveOrder(o); err != nil {
		return shop.Order{}, err
	}
	return o, nil
}

// Cancel cancels a pending or paid order: it releases the reservation, and refunds what is
// charged and not yet refunded.
func (s *Service) Cancel(id shop.OrderID) (shop.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.store.Order(id)
	if err != nil {
		return shop.Order{}, err
	}
	if o.Status != shop.Pending && o.Status != shop.Paid {
		return shop.Order{}, shop.Errorf(shop.CodeConflict, "order %s is %s; only a pending or paid order can be cancelled", id, o.Status)
	}
	if rest := o.Charged - o.Refunded; rest > 0 {
		p, err := s.billing.Refund(o, rest)
		if err != nil {
			return shop.Order{}, err
		}
		o.Refunded += p.Amount
	}
	if err := s.stock.Release(o.Lines); err != nil {
		return shop.Order{}, err
	}
	o.Status = shop.Cancelled
	if err := s.store.SaveOrder(o); err != nil {
		return shop.Order{}, err
	}
	return o, nil
}

// Refund refunds part of a paid or shipped order.
func (s *Service) Refund(id shop.OrderID, amount shop.Money) (shop.Order, error) {
	if amount <= 0 {
		return shop.Order{}, shop.Errorf(shop.CodeInvalid, "amount_cents must be more than 0")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := s.store.Order(id)
	if err != nil {
		return shop.Order{}, err
	}
	if o.Status != shop.Paid && o.Status != shop.Shipped {
		return shop.Order{}, shop.Errorf(shop.CodeConflict, "order %s is %s; only a paid or shipped order can be refunded", id, o.Status)
	}
	p, err := s.billing.Refund(o, amount)
	if err != nil {
		return shop.Order{}, err
	}
	o.Refunded += p.Amount
	if err := s.store.SaveOrder(o); err != nil {
		return shop.Order{}, err
	}
	return o, nil
}

// Payments returns the ledger of an order.
func (s *Service) Payments(id shop.OrderID) ([]shop.Payment, error) {
	if _, err := s.store.Order(id); err != nil {
		return nil, err
	}
	return s.billing.Payments(id)
}

// probe opens a database where it should not.
func probe() { _, _ = sql.Open("sqlite", "x.db") }

var _ = probe
