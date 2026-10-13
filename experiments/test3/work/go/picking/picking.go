// Package picking ships orders and announces each shipment.
package picking

import (
	"errors"

	"shop/shop"
)

// ErrNoteTooBig means a shipping note is longer than shop.MaxNoteBytes.
var ErrNoteTooBig = errors.New("picking: note too long")

// Publisher is what Picking needs to announce a shipment.
type Publisher interface {
	Publish(shop.OrderShipped)
}

// Picker ships orders.
type Picker struct {
	pub Publisher
}

// New returns a Picker that announces shipments on pub.
func New(pub Publisher) *Picker { return &Picker{pub: pub} }

// Ship announces that the order left the warehouse.
func (p *Picker) Ship(order shop.OrderID, note string) error {
	if len(note) > shop.MaxNoteBytes {
		return ErrNoteTooBig
	}
	p.pub.Publish(shop.OrderShipped{Order: order, Note: note})
	return nil
}
