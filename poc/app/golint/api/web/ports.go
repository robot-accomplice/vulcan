package web

import "stockroom/lib/shop"

// Catalog is the catalog and the stock.
type Catalog interface {
	AddProduct(p shop.Product) (shop.Item, error)
	Product(sku shop.SKU) (shop.Item, error)
	Products() ([]shop.Item, error)
	Receive(sku shop.SKU, qty int) (shop.Item, error)
}

// Orders runs the life cycle of orders.
type Orders interface {
	Create(customer shop.CustomerID, wants []shop.Want) (shop.Order, error)
	Get(id shop.OrderID) (shop.Order, error)
	List(customer shop.CustomerID) ([]shop.Order, error)
	Pay(id shop.OrderID) (shop.Order, error)
	Ship(id shop.OrderID) (shop.Order, error)
	Cancel(id shop.OrderID) (shop.Order, error)
	Refund(id shop.OrderID, amount shop.Money) (shop.Order, error)
	Payments(id shop.OrderID) ([]shop.Payment, error)
}

// Purchasing keeps the suppliers and the purchase orders.
type Purchasing interface {
	AddSupplier(name string) (shop.Supplier, error)
	Supplier(id shop.SupplierID) (shop.Supplier, error)
	Suppliers() ([]shop.Supplier, error)
	Create(supplier shop.SupplierID, wants []shop.PurchaseWant) (shop.PurchaseOrder, error)
	Get(id shop.PurchaseID) (shop.PurchaseOrder, error)
	List(supplier shop.SupplierID) ([]shop.PurchaseOrder, error)
	Receive(id shop.PurchaseID, arrivals []shop.Arrival) (shop.PurchaseOrder, error)
	Cancel(id shop.PurchaseID) (shop.PurchaseOrder, error)
}

// Events keeps the webhook subscriptions and their deliveries.
type Events interface {
	Subscribe(url string, types []shop.EventType) (shop.Subscription, error)
	Subscription(id shop.SubscriptionID) (shop.Subscription, error)
	Subscriptions() ([]shop.Subscription, error)
	Unsubscribe(id shop.SubscriptionID) error
	Deliveries(id shop.SubscriptionID) ([]shop.Delivery, error)
	Retry(id shop.SubscriptionID) ([]shop.Delivery, error)
}
