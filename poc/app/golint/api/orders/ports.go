package orders

import (
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

// checkedStock checks the contract of Stock at each call.
type checkedStock struct{ Stock }

func (s checkedStock) Reserve(lines []shop.Line) error {
	contract.Requires(len(lines) > 0)
	return s.Stock.Reserve(lines)
}

// checkedBilling checks the contract of Billing at each call.
type checkedBilling struct{ Billing }

func (b checkedBilling) Charge(o shop.Order) (shop.Payment, error) {
	contract.Requires(o.Charged == 0 && o.Total > 0)
	return b.Billing.Charge(o)
}

func (b checkedBilling) Refund(o shop.Order, amount shop.Money) (shop.Payment, error) {
	contract.Requires(amount > 0)
	return b.Billing.Refund(o, amount)
}
