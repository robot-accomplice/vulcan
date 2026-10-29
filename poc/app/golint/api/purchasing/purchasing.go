// Package purchasing keeps the suppliers and the purchase orders: it orders stock from a supplier
// and receives its deliveries.
package purchasing

import (
	"fmt"
	"sync"
	"unicode/utf8"

	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Service keeps the suppliers and the purchase orders. One lock serializes the changes of
// purchase orders, so that two requests never change one purchase order at once.
type Service struct {
	mu     sync.Mutex
	stock  Stock
	store  Store
	events Events
}

// NewService returns the service.
func NewService(stock Stock, store Store, events Events) *Service {
	return &Service{stock: checkedStock{stock}, store: store, events: events}
}

// AddSupplier adds a supplier.
func (s *Service) AddSupplier(name string) (shop.Supplier, error) {
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return shop.Supplier{}, shop.Errorf(shop.CodeInvalid, "name must be 1 to 100 characters")
	}
	id, err := s.store.AddSupplier(name)
	if err != nil {
		return shop.Supplier{}, err
	}
	return shop.Supplier{ID: id, Name: name}, nil
}

// Supplier returns a supplier.
func (s *Service) Supplier(id shop.SupplierID) (shop.Supplier, error) {
	return s.store.Supplier(id)
}

// Suppliers returns the suppliers, by id.
func (s *Service) Suppliers() ([]shop.Supplier, error) {
	return s.store.Suppliers()
}

// Create creates an open purchase order. It changes no stock.
func (s *Service) Create(supplier shop.SupplierID, wants []shop.PurchaseWant) (purchase shop.PurchaseOrder, err error) {
	defer contract.Ensures(func() bool { return err != nil || purchase.Status == shop.PurchaseOpen })
	if len(wants) == 0 {
		return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "a purchase order has at least one line")
	}
	seen := map[shop.SKU]bool{}
	var lines []shop.PurchaseLine
	for _, w := range wants {
		switch {
		case w.Qty <= 0 || w.Qty > shop.MaxQty:
			return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "qty must be more than 0 and at most %d", shop.MaxQty)
		case w.Cost <= 0 || w.Cost > shop.MaxPrice:
			return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "cost_cents must be more than 0 and at most %d", shop.MaxPrice)
		case seen[w.SKU]:
			return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "sku %s is on two lines", w.SKU)
		}
		seen[w.SKU] = true
		lines = append(lines, contract.Valid(shop.PurchaseLine{SKU: w.SKU, Qty: w.Qty, Cost: w.Cost}))
	}
	if _, err := s.store.Supplier(supplier); err != nil {
		return shop.PurchaseOrder{}, err
	}
	for _, l := range lines {
		if _, err := s.stock.Product(l.SKU); err != nil {
			return shop.PurchaseOrder{}, err
		}
	}
	po := shop.PurchaseOrder{Supplier: supplier, Status: shop.PurchaseOpen, Lines: lines, Total: shop.PurchaseTotal(lines)}
	id, err := s.store.CreatePurchase(po)
	if err != nil {
		return shop.PurchaseOrder{}, err
	}
	po.ID = id
	return po, nil
}

// Get returns a purchase order.
func (s *Service) Get(id shop.PurchaseID) (shop.PurchaseOrder, error) {
	return s.store.Purchase(id)
}

// List returns the purchase orders of a supplier, by id.
func (s *Service) List(supplier shop.SupplierID) ([]shop.PurchaseOrder, error) {
	if _, err := s.store.Supplier(supplier); err != nil {
		return nil, err
	}
	return s.store.PurchasesOf(supplier)
}

// Receive receives a delivery for an open purchase order: every arrival goes into stock, or
// none. When every line is received in full, the purchase order is received.
func (s *Service) Receive(id shop.PurchaseID, arrivals []shop.Arrival) (purchase shop.PurchaseOrder, err error) {
	defer contract.Ensures(func() bool {
		return err != nil || purchase.Status == shop.PurchaseOpen || purchase.Status == shop.PurchaseReceived
	})
	if len(arrivals) == 0 {
		return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "a delivery has at least one line")
	}
	seen := map[shop.SKU]bool{}
	for _, a := range arrivals {
		if a.Qty <= 0 || a.Qty > shop.MaxQty {
			return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "qty must be more than 0 and at most %d", shop.MaxQty)
		}
		if seen[a.SKU] {
			return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "sku %s is on two lines", a.SKU)
		}
		seen[a.SKU] = true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	po, err := s.store.Purchase(id)
	if err != nil {
		return shop.PurchaseOrder{}, err
	}
	if po.Status != shop.PurchaseOpen {
		return shop.PurchaseOrder{}, shop.Errorf(shop.CodeConflict, "purchase order %s is %s, not open", id, po.Status)
	}
	lines := append([]shop.PurchaseLine(nil), po.Lines...)
	for _, a := range arrivals {
		i := lineOf(lines, a.SKU)
		if i < 0 {
			return shop.PurchaseOrder{}, shop.Errorf(shop.CodeInvalid, "sku %s is not on purchase order %s", a.SKU, id)
		}
		l := lines[i]
		if l.Received+a.Qty > l.Qty {
			return shop.PurchaseOrder{}, shop.Errorf(shop.CodeConflict, "%s: %d more units is more than the %d still to receive", a.SKU, a.Qty, l.Qty-l.Received)
		}
		lines[i] = contract.Valid(shop.PurchaseLine{SKU: l.SKU, Qty: l.Qty, Received: l.Received + a.Qty, Cost: l.Cost})
	}
	if err := s.stock.Restock(arrivals); err != nil {
		return shop.PurchaseOrder{}, err
	}
	po.Lines = lines
	if complete(lines) {
		po.Status = shop.PurchaseReceived
	}
	if err := s.store.SavePurchase(po); err != nil {
		return shop.PurchaseOrder{}, fmt.Errorf("the stock is received, but purchase order %s is not saved: %w", id, err)
	}
	if po.Status == shop.PurchaseReceived {
		s.events.Publish(shop.EventPurchaseReceived, string(po.ID))
	}
	return po, nil
}

// Cancel cancels an open purchase order. The units already received stay in stock.
func (s *Service) Cancel(id shop.PurchaseID) (purchase shop.PurchaseOrder, err error) {
	defer contract.Ensures(func() bool { return err != nil || purchase.Status == shop.PurchaseCancelled })
	s.mu.Lock()
	defer s.mu.Unlock()
	po, err := s.store.Purchase(id)
	if err != nil {
		return shop.PurchaseOrder{}, err
	}
	if po.Status != shop.PurchaseOpen {
		return shop.PurchaseOrder{}, shop.Errorf(shop.CodeConflict, "purchase order %s is %s, not open", id, po.Status)
	}
	po.Status = shop.PurchaseCancelled
	if err := s.store.SavePurchase(po); err != nil {
		return shop.PurchaseOrder{}, err
	}
	return po, nil
}

// lineOf returns the index of the line of a product, or -1.
func lineOf(lines []shop.PurchaseLine, sku shop.SKU) int {
	for i, l := range lines {
		if l.SKU == sku {
			return i
		}
	}
	return -1
}

// complete reports whether every line is received in full.
func complete(lines []shop.PurchaseLine) bool {
	for _, l := range lines {
		if l.Received < l.Qty {
			return false
		}
	}
	return true
}
