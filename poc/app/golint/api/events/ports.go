package events

import (
	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Store persists the subscriptions, the events, and the deliveries.
type Store interface {
	AddSubscription(sub shop.Subscription) (shop.SubscriptionID, error)
	Subscription(id shop.SubscriptionID) (shop.Subscription, error)
	Subscriptions() ([]shop.Subscription, error)
	DeleteSubscription(id shop.SubscriptionID) error
	AddEvent(e shop.Event) (shop.EventID, error)
	AddDelivery(d shop.Delivery) error
	SaveDelivery(d shop.Delivery) error
	Deliveries(id shop.SubscriptionID) ([]shop.Delivery, error)
}

// Sender sends one webhook, and returns the HTTP status of the answer.
type Sender interface {
	Send(url string, eventType string, signature string, body []byte) (int, error)
}

// checkedSender checks the contract of Sender at each call.
type checkedSender struct{ Sender }

func (s checkedSender) Send(url string, eventType string, signature string, body []byte) (int, error) {
	contract.Requires(url != "" && signature != "" && len(body) > 0)
	return s.Sender.Send(url, eventType, signature, body)
}
