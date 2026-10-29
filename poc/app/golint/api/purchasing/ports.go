package purchasing

import (
	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Stock looks up products and adds received units to their stock.
type Stock interface {
	Product(sku shop.SKU) (shop.Item, error)
	Restock(arrivals []shop.Arrival) error
}

// Store persists suppliers and purchase orders.
type Store interface {
	AddSupplier(name string) (shop.SupplierID, error)
	Supplier(id shop.SupplierID) (shop.Supplier, error)
	Suppliers() ([]shop.Supplier, error)
	CreatePurchase(po shop.PurchaseOrder) (shop.PurchaseID, error)
	Purchase(id shop.PurchaseID) (shop.PurchaseOrder, error)
	PurchasesOf(s shop.SupplierID) ([]shop.PurchaseOrder, error)
	SavePurchase(po shop.PurchaseOrder) error
}

// Events tells the subscribers what happened.
type Events interface {
	Publish(t shop.EventType, subject string)
}

// checkedStock checks the contract of Stock at each call.
type checkedStock struct{ Stock }

func (s checkedStock) Restock(arrivals []shop.Arrival) error {
	contract.Requires(len(arrivals) > 0)
	return s.Stock.Restock(arrivals)
}
