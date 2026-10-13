// Package bus is a small in-memory event bus for OrderShipped events.
//
// It stands in for the channel between the worker process and the API process.
// Delivery is at least once: a handler that returns an error gets the event again,
// and a handler must accept the same event more than once.
package bus

import (
	"fmt"

	"shop/shop"
)

// MaxAttempts is how many times the bus delivers one event to one handler before it gives up.
const MaxAttempts = 5

// Handler processes one OrderShipped event.
type Handler func(shop.OrderShipped) error

// Bus delivers each published event to every subscriber.
type Bus struct {
	handlers []Handler
	// DeadLetter receives an event that a handler failed MaxAttempts times. It can be nil.
	DeadLetter func(ev shop.OrderShipped, err error)
}

// New returns an empty bus.
func New() *Bus { return &Bus{} }

// Subscribe adds a handler.
func (b *Bus) Subscribe(h Handler) { b.handlers = append(b.handlers, h) }

// Publish delivers ev to each handler, and retries a handler that fails.
func (b *Bus) Publish(ev shop.OrderShipped) {
	for _, h := range b.handlers {
		var err error
		for attempt := 1; attempt <= MaxAttempts; attempt++ {
			if err = h(ev); err == nil {
				break
			}
		}
		if err != nil && b.DeadLetter != nil {
			b.DeadLetter(ev, fmt.Errorf("after %d attempts: %w", MaxAttempts, err))
		}
	}
}
