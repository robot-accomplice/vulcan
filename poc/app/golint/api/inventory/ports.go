package inventory

import (
	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Store persists products and stock.
type Store interface {
	AddProduct(p shop.Product) error
	Product(sku shop.SKU) (shop.Item, error)
	Products() ([]shop.Item, error)
	Receive(sku shop.SKU, qty int) (shop.Item, error)
	Reserve(lines []shop.Line) error
	Release(lines []shop.Line) error
	Ship(lines []shop.Line) error
}

// checkedStore checks the contract of Store at each call.
type checkedStore struct{ Store }

func (s checkedStore) Receive(sku shop.SKU, qty int) (shop.Item, error) {
	contract.Requires(qty > 0)
	return s.Store.Receive(sku, qty)
}
