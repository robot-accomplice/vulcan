package arch

// The persons and the external systems.
const (
	Customer        Person   = "Customer"        // places, pays for, and cancels orders
	Database        External = "Database"        // the SQLite file
	PaymentProvider External = "PaymentProvider" // the card processor
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
		Gateway("api/store", Database),
		Gateway("api/payments", PaymentProvider),
		Library("lib/shop"),
		Library("lib/contract"),
	},
}
