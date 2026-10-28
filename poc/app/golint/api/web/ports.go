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
