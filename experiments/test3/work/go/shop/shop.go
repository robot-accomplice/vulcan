// Package shop holds the types that Orders, Billing, Picking, and the web entry point share.
package shop

// CustomerID identifies a customer.
type CustomerID string

// OrderID identifies an order.
type OrderID string

// Money is an amount in cents.
type Money int64

// Receipt records one charge or one refund.
type Receipt struct {
	Customer CustomerID
	Order    OrderID
	Amount   Money
}

// MaxNoteBytes is the largest Note that an OrderShipped event can carry.
const MaxNoteBytes = 500

// OrderShipped tells the API process that Picking shipped an order.
type OrderShipped struct {
	Order OrderID
	Note  string
}
