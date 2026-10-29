// Package orders runs the life cycle of an order: create, pay, ship, cancel, refund.
package orders

import (
	"fmt"
	"sync"
	"unicode/utf8"

	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Service runs the life cycle of orders. One lock serializes the changes of orders, so that two
// requests never change one order at once.
type Service struct {
	mu      sync.Mutex
	stock   Stock
	billing Billing
	store   Store
	events  Events
}

// NewService returns the service.
func NewService(stock Stock, billing Billing, store Store, events Events) *Service {
	return &Service{stock: checkedStock{stock}, billing: checkedBilling{billing}, store: store, events: events}
}

// Create creates an order and reserves its stock: all lines, or none.
func (s *Service) Create(customer shop.CustomerID, wants []shop.Want) (order shop.Order, err error) {
	defer contract.Ensures(func() bool { return err != nil || order.Status == shop.Pending && order.Charged == 0 })
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
	s.events.Publish(shop.EventOrderCreated, string(o.ID))
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
func (s *Service) Pay(id shop.OrderID) (order shop.Order, err error) {
	defer contract.Ensures(func() bool { return err != nil || order.Status == shop.Paid && order.Charged == order.Total })
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
	s.events.Publish(shop.EventOrderPaid, string(o.ID))
	return o, nil
}

// Ship ships a paid order: its units leave the stock.
func (s *Service) Ship(id shop.OrderID) (order shop.Order, err error) {
	defer contract.Ensures(func() bool { return err != nil || order.Status == shop.Shipped })
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
	s.events.Publish(shop.EventOrderShipped, string(o.ID))
	return o, nil
}

// Cancel cancels a pending or paid order: it releases the reservation, and refunds what is
// charged and not yet refunded.
func (s *Service) Cancel(id shop.OrderID) (order shop.Order, err error) {
	defer contract.Ensures(func() bool { return err != nil || order.Status == shop.Cancelled && order.Refunded == order.Charged })
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
	s.events.Publish(shop.EventOrderCancelled, string(o.ID))
	return o, nil
}

// Refund refunds part of a paid or shipped order.
func (s *Service) Refund(id shop.OrderID, amount shop.Money) (order shop.Order, err error) {
	defer contract.Ensures(func() bool { return err != nil || order.Refunded <= order.Charged })
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
	s.events.Publish(shop.EventOrderRefunded, string(o.ID))
	return o, nil
}

// Payments returns the ledger of an order.
func (s *Service) Payments(id shop.OrderID) ([]shop.Payment, error) {
	if _, err := s.store.Order(id); err != nil {
		return nil, err
	}
	return s.billing.Payments(id)
}
