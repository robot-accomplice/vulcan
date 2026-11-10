// Package orders runs the life cycle of an order: create, pay, ship, cancel, refund.
package orders

import (
	"fmt"
	"sync"
	"unicode/utf8"

	"stockroom/lib/contract"
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

// Events tells the subscribers what happened.
type Events interface {
	Publish(t shop.EventType, subject string)
}

// checkedStock enforces the port contract of Stock.Reserve (K15) at each call that Service makes.
type checkedStock struct{ Stock }

func (s checkedStock) Reserve(lines []shop.Line) error {
	if len(lines) == 0 {
		contract.Fail("K15", "Stock.Reserve: lines must not be empty")
	}
	return s.Stock.Reserve(lines)
}

// checkedBilling enforces the port contracts of Billing.Charge and Billing.Refund (K16, K17) at
// each call that Service makes.
type checkedBilling struct{ Billing }

func (b checkedBilling) Charge(o shop.Order) (shop.Payment, error) {
	if o.Charged != 0 || o.Total <= 0 {
		contract.Fail("K16", "Billing.Charge: order %s needs Charged 0 and Total > 0, got Charged=%d Total=%d", o.ID, o.Charged, o.Total)
	}
	return b.Billing.Charge(o)
}

func (b checkedBilling) Refund(o shop.Order, amount shop.Money) (shop.Payment, error) {
	if amount <= 0 {
		contract.Fail("K17", "Billing.Refund: amount %d must be more than 0", amount)
	}
	return b.Billing.Refund(o, amount)
}

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

// create creates an order and reserves its stock: all lines, or none.
func (s *Service) create(customer shop.CustomerID, wants []shop.Want) (shop.Order, error) {
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

// pay charges the total of a pending order.
func (s *Service) pay(id shop.OrderID) (shop.Order, error) {
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

// ship ships a paid order: its units leave the stock.
func (s *Service) ship(id shop.OrderID) (shop.Order, error) {
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

// cancel cancels a pending or paid order: it releases the reservation, and refunds what is
// charged and not yet refunded.
func (s *Service) cancel(id shop.OrderID) (shop.Order, error) {
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

// refund refunds part of a paid or shipped order.
func (s *Service) refund(id shop.OrderID, amount shop.Money) (shop.Order, error) {
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

// Create creates an order and reserves its stock: all lines, or none.
func (s *Service) Create(customer shop.CustomerID, wants []shop.Want) (shop.Order, error) {
	o, err := s.create(customer, wants)
	if err == nil && (o.Status != shop.Pending || o.Charged != 0) { // K18
		contract.Fail("K18", "Service.Create: order %s has Status=%s Charged=%d, want %s and 0", o.ID, o.Status, o.Charged, shop.Pending)
	}
	return o, err
}

// Pay charges the total of a pending order.
func (s *Service) Pay(id shop.OrderID) (shop.Order, error) {
	o, err := s.pay(id)
	if err == nil && (o.Status != shop.Paid || o.Charged != o.Total) { // K19
		contract.Fail("K19", "Service.Pay: order %s has Status=%s Charged=%d Total=%d, want %s and Charged=Total", o.ID, o.Status, o.Charged, o.Total, shop.Paid)
	}
	return o, err
}

// Ship ships a paid order: its units leave the stock.
func (s *Service) Ship(id shop.OrderID) (shop.Order, error) {
	o, err := s.ship(id)
	if err == nil && o.Status != shop.Shipped { // K20
		contract.Fail("K20", "Service.Ship: order %s has Status=%s, want %s", o.ID, o.Status, shop.Shipped)
	}
	return o, err
}

// Cancel cancels a pending or paid order: it releases the reservation, and refunds what is
// charged and not yet refunded.
func (s *Service) Cancel(id shop.OrderID) (shop.Order, error) {
	o, err := s.cancel(id)
	if err == nil && (o.Status != shop.Cancelled || o.Refunded != o.Charged) { // K21
		contract.Fail("K21", "Service.Cancel: order %s has Status=%s Refunded=%d Charged=%d, want %s and Refunded=Charged", o.ID, o.Status, o.Refunded, o.Charged, shop.Cancelled)
	}
	return o, err
}

// Refund refunds part of a paid or shipped order.
func (s *Service) Refund(id shop.OrderID, amount shop.Money) (shop.Order, error) {
	o, err := s.refund(id, amount)
	if err == nil && o.Refunded > o.Charged { // K22
		contract.Fail("K22", "Service.Refund: order %s has Refunded=%d more than Charged=%d", o.ID, o.Refunded, o.Charged)
	}
	return o, err
}
