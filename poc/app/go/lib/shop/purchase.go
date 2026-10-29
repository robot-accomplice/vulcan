package shop

// SupplierID names a supplier: s-1, s-2, and so on.
type SupplierID string

// PurchaseID names a purchase order: po-1, po-2, and so on.
type PurchaseID string

// Supplier is a supplier of stock.
type Supplier struct {
	ID   SupplierID
	Name string
}

// PurchaseStatus is the state of a purchase order.
type PurchaseStatus string

// The states of a purchase order.
const (
	PurchaseOpen      PurchaseStatus = "open"
	PurchaseReceived  PurchaseStatus = "received"
	PurchaseCancelled PurchaseStatus = "cancelled"
)

// PurchaseLine is one line of a purchase order: a product, the units ordered and received so
// far, and the unit cost.
type PurchaseLine struct {
	SKU      SKU
	Qty      int
	Received int
	Cost     Money
}

// PurchaseWant is one line of a new purchase order, as the buyer asks for it.
type PurchaseWant struct {
	SKU  SKU
	Qty  int
	Cost Money
}

// Arrival is the units of one product that arrive in a delivery from a supplier.
type Arrival struct {
	SKU SKU
	Qty int
}

// PurchaseOrder is an order of stock from a supplier.
type PurchaseOrder struct {
	ID       PurchaseID
	Supplier SupplierID
	Status   PurchaseStatus
	Lines    []PurchaseLine
	Total    Money
}

// PurchaseTotal returns the sum of qty × cost over the lines.
func PurchaseTotal(lines []PurchaseLine) Money {
	var t Money
	for _, l := range lines {
		t += Money(l.Qty) * l.Cost
	}
	return t
}
