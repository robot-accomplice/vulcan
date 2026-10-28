package billing

import (
	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Card is the card processor.
type Card interface {
	Charge(customer shop.CustomerID, amount shop.Money) (string, error)
	Refund(chargeID string, amount shop.Money) (string, error)
}

// Store persists the ledger.
type Store interface {
	AddPayment(id shop.OrderID, p shop.Payment) error
	Payments(id shop.OrderID) ([]shop.Payment, error)
}

// checkedCard checks the contract of Card at each call.
type checkedCard struct{ Card }

func (c checkedCard) Charge(customer shop.CustomerID, amount shop.Money) (string, error) {
	contract.Requires(amount > 0)
	return c.Card.Charge(customer, amount)
}

func (c checkedCard) Refund(chargeID string, amount shop.Money) (string, error) {
	contract.Requires(chargeID != "" && amount > 0)
	return c.Card.Refund(chargeID, amount)
}
