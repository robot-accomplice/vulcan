package arch

// The persons and the external systems.
const (
	Customer        Person   = "Customer"        // places, pays for, and cancels orders
	Database        External = "Database"        // the SQLite file
	PaymentProvider External = "PaymentProvider" // the card processor
	Subscribers     External = "Subscribers"     // the systems that receive webhooks
)

// Manifest is the architecture of Stockroom.
var Manifest = System{
	Strict: true,
	Parts: []Part{
		Main("cmd/stockroom"),
		Entry("api/web", Customer),
		Leaf("api/inventory"),
		Leaf("api/orders"),
		Leaf("api/billing"),
		Leaf("api/purchasing"),
		Leaf("api/events"),
		Gateway("api/store", Database),
		Gateway("api/payments", PaymentProvider),
		Gateway("api/webhooks", Subscribers),
		Library("lib/shop"),
		Library("lib/contract"),
	},
}
