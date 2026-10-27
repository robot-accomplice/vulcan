package shop

// Money is an amount in cents.
type Money int64

// Limits of the data (docs/poc/stockroom.md): with them, no sum of an order overflows.
const (
	MaxPrice Money = 100_000_000 // cents, for one unit
	MaxQty         = 10_000      // units, for one line or one receipt
)

// SKU names a product.
type SKU string

// OrderID names an order: o-1, o-2, and so on.
type OrderID string

// CustomerID names a customer.
type CustomerID string

// Status is the state of an order.
type Status string

// The states of an order.
const (
	Pending   Status = "pending"
	Paid      Status = "paid"
	Shipped   Status = "shipped"
	Cancelled Status = "cancelled"
)

// Product is an entry of the catalog.
type Product struct {
	SKU   SKU
	Name  string
	Price Money
}

// Stock is the stock level of a product.
type Stock struct {
	OnHand   int
	Reserved int
}

// Available is the stock that a new order can reserve.
func (s Stock) Available() int { return s.OnHand - s.Reserved }

// Item is a product with its stock.
type Item struct {
	Product Product
	Stock   Stock
}

// Line is one line of an order: a product, a quantity, and the unit price when it was ordered.
type Line struct {
	SKU   SKU
	Qty   int
	Price Money
}

// Want is one line of a new order, as the customer asks for it.
type Want struct {
	SKU SKU
	Qty int
}

// Order is an order.
type Order struct {
	ID        OrderID
	Customer  CustomerID
	Status    Status
	Lines     []Line
	Total     Money
	Charged   Money
	Refunded  Money
	PaymentID string
}

// Kinds of payment entries.
const (
	KindCharge = "charge"
	KindRefund = "refund"
)

// Payment is one entry of an order's ledger.
type Payment struct {
	Kind       string
	Amount     Money
	ProviderID string
}

// Total returns the sum of qty × price over the lines.
func Total(lines []Line) Money {
	var t Money
	for _, l := range lines {
		t += Money(l.Qty) * l.Price
	}
	return t
}
