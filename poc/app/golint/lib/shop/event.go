package shop

// EventType is the kind of an event.
type EventType string

// The types of events.
const (
	EventOrderCreated     EventType = "order.created"
	EventOrderPaid        EventType = "order.paid"
	EventOrderShipped     EventType = "order.shipped"
	EventOrderCancelled   EventType = "order.cancelled"
	EventOrderRefunded    EventType = "order.refunded"
	EventPurchaseReceived EventType = "purchase_order.received"
)

// EventTypes lists every type of event.
var EventTypes = [...]EventType{
	EventOrderCreated, EventOrderPaid, EventOrderShipped, EventOrderCancelled, EventOrderRefunded, EventPurchaseReceived,
}

// EventID names an event: ev-1, ev-2, and so on.
type EventID string

// Event is something that happened, for the subscribers: its type and the id of its order or
// purchase order.
type Event struct {
	ID      EventID
	Type    EventType
	Subject string
}

// SubscriptionID names a webhook subscription: wh-1, wh-2, and so on.
type SubscriptionID string

// Subscription is a webhook subscription: where to send the events of its types, and the secret
// that signs them.
type Subscription struct {
	ID     SubscriptionID
	URL    string
	Events []EventType
	Secret string
}

// DeliveryStatus is the state of a delivery.
type DeliveryStatus string

// The states of a delivery.
const (
	Delivered DeliveryStatus = "delivered"
	Failed    DeliveryStatus = "failed"
)

// Delivery is one event sent to one subscription.
type Delivery struct {
	Subscription SubscriptionID
	Event        Event
	Status       DeliveryStatus
	Attempts     int
	Response     int // the HTTP status of the last attempt, or 0
}
