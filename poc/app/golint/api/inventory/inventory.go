// Package inventory keeps the catalog and the stock levels.
package inventory

import (
	"regexp"
	"unicode/utf8"

	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Catalog is the catalog and the stock.
type Catalog struct {
	store Store
}

// NewCatalog returns the catalog.
func NewCatalog(store Store) *Catalog { return &Catalog{store: checkedStore{store}} }

var skuPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,32}$`)

// AddProduct adds a product to the catalog, with no stock.
func (c *Catalog) AddProduct(p shop.Product) (shop.Item, error) {
	switch {
	case !skuPattern.MatchString(string(p.SKU)):
		return shop.Item{}, shop.Errorf(shop.CodeInvalid, "sku must be 1 to 32 letters, digits, or -")
	case p.Name == "" || utf8.RuneCountInString(p.Name) > 100:
		return shop.Item{}, shop.Errorf(shop.CodeInvalid, "name must be 1 to 100 characters")
	case p.Price <= 0 || p.Price > shop.MaxPrice:
		return shop.Item{}, shop.Errorf(shop.CodeInvalid, "price_cents must be more than 0 and at most %d", shop.MaxPrice)
	}
	if err := c.store.AddProduct(p); err != nil {
		return shop.Item{}, err
	}
	return c.store.Product(p.SKU)
}

// Product returns a product with its stock.
func (c *Catalog) Product(sku shop.SKU) (shop.Item, error) {
	return c.store.Product(sku)
}

// Products returns the catalog, by SKU.
func (c *Catalog) Products() ([]shop.Item, error) {
	return c.store.Products()
}

// Receive adds received units to the stock of a product.
func (c *Catalog) Receive(sku shop.SKU, qty int) (shop.Item, error) {
	if qty <= 0 || qty > shop.MaxQty {
		return shop.Item{}, shop.Errorf(shop.CodeInvalid, "qty must be more than 0 and at most %d", shop.MaxQty)
	}
	return c.store.Receive(sku, qty)
}

// Reserve reserves the units of every line, or of none.
func (c *Catalog) Reserve(lines []shop.Line) error {
	contract.Requires(len(lines) > 0)
	return c.store.Reserve(lines)
}

// Release releases the reservation of every line.
func (c *Catalog) Release(lines []shop.Line) error {
	return c.store.Release(lines)
}

// Ship takes the reserved units of every line out of stock.
func (c *Catalog) Ship(lines []shop.Line) error {
	return c.store.Ship(lines)
}
