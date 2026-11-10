// Package billing charges and refunds orders, and keeps the ledger of each order.
package billing

import (
	"fmt"

	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// Card is the card processor.
type Card interface {
	Charge(customer shop.CustomerID, amount shop.Money) (string, error)
	Refund(chargeID string, amount shop.Money) (string, error)
}

// checkedCard enforces the port contracts of Card (K09, K10) at each call that Ledger makes.
// The caller is responsible for these contracts, so the check sits at the call site and covers
// every Card behind it, fakes included.
type checkedCard struct{ Card }

func (c checkedCard) Charge(customer shop.CustomerID, amount shop.Money) (string, error) {
	if amount <= 0 {
		contract.Fail("K09", "Card.Charge: amount %d must be more than 0", amount)
	}
	return c.Card.Charge(customer, amount)
}

func (c checkedCard) Refund(chargeID string, amount shop.Money) (string, error) {
	if chargeID == "" || amount <= 0 {
		contract.Fail("K10", "Card.Refund: chargeID %q must not be empty and amount %d must be more than 0", chargeID, amount)
	}
	return c.Card.Refund(chargeID, amount)
}

// Store persists the ledger.
type Store interface {
	AddPayment(id shop.OrderID, p shop.Payment) error
	Payments(id shop.OrderID) ([]shop.Payment, error)
}

// account is the money of one order: what was charged and what was refunded.
type account struct {
	Charged  shop.Money
	Refunded shop.Money
}

// checkInvariant checks the invariant of account (K27). Callers that are not methods of account
// call it: when an account is built, and after each charge or refund returns.
func (a account) checkInvariant() {
	if a.Refunded < 0 || a.Refunded > a.Charged {
		contract.Fail("K27", "account needs 0 <= Refunded <= Charged, got Charged=%d Refunded=%d", a.Charged, a.Refunded)
	}
}

// charge records a charge.
func (a *account) charge(amount shop.Money) {
	if amount <= 0 { // K01
		contract.Fail("K01", "account.charge: amount %d must be more than 0", amount)
	}
	old := a.Charged
	a.Charged += amount
	if a.Charged != old+amount { // K02
		contract.Fail("K02", "account.charge: Charged is %d, want old %d + amount %d", a.Charged, old, amount)
	}
}

// refund records a refund.
func (a *account) refund(amount shop.Money) {
	if amount <= 0 { // K03
		contract.Fail("K03", "account.refund: amount %d must be more than 0", amount)
	}
	old := a.Refunded
	a.Refunded += amount
	if a.Refunded != old+amount { // K04
		contract.Fail("K04", "account.refund: Refunded is %d, want old %d + amount %d", a.Refunded, old, amount)
	}
}

// Ledger charges and refunds orders.
type Ledger struct {
	card  Card
	store Store
}

// NewLedger returns the ledger.
func NewLedger(card Card, store Store) *Ledger { return &Ledger{card: checkedCard{card}, store: store} }

// accountOf sums the ledger of an order.
func (l *Ledger) accountOf(id shop.OrderID) (*account, error) {
	entries, err := l.store.Payments(id)
	if err != nil {
		return nil, err
	}
	a := &account{}
	a.checkInvariant() // K27, built
	for _, p := range entries {
		switch p.Kind {
		case shop.KindCharge:
			a.charge(p.Amount)
			a.checkInvariant() // K27, after charge
		case shop.KindRefund:
			a.refund(p.Amount)
			a.checkInvariant() // K27, after refund; fails when the stored entries put a refund before its charges
		}
	}
	return a, nil
}

// Charge charges the total of an order that is not charged yet, and records it.
func (l *Ledger) Charge(o shop.Order) (shop.Payment, error) {
	if o.Charged != 0 || o.Total <= 0 { // K05
		contract.Fail("K05", "Ledger.Charge: order %s needs Charged 0 and Total > 0, got Charged=%d Total=%d", o.ID, o.Charged, o.Total)
	}
	p, err := l.charge(o)
	if err == nil && (p.Kind != shop.KindCharge || p.Amount != o.Total) { // K06
		contract.Fail("K06", "Ledger.Charge: payment is kind %q amount %d, want %q amount %d", p.Kind, p.Amount, shop.KindCharge, o.Total)
	}
	return p, err
}

func (l *Ledger) charge(o shop.Order) (shop.Payment, error) {
	id, err := l.card.Charge(o.Customer, o.Total)
	if err != nil {
		return shop.Payment{}, err
	}
	p := shop.Payment{Kind: shop.KindCharge, Amount: o.Total, ProviderID: id}
	if err := l.store.AddPayment(o.ID, p); err != nil {
		return shop.Payment{}, fmt.Errorf("record the charge %s: %w", id, err)
	}
	return p, nil
}

// Refund refunds part of what an order was charged, and records it.
func (l *Ledger) Refund(o shop.Order, amount shop.Money) (shop.Payment, error) {
	if amount <= 0 { // K07
		contract.Fail("K07", "Ledger.Refund: amount %d must be more than 0", amount)
	}
	p, err := l.refund(o, amount)
	if err == nil && (p.Kind != shop.KindRefund || p.Amount != amount) { // K08
		contract.Fail("K08", "Ledger.Refund: payment is kind %q amount %d, want %q amount %d", p.Kind, p.Amount, shop.KindRefund, amount)
	}
	return p, err
}

func (l *Ledger) refund(o shop.Order, amount shop.Money) (shop.Payment, error) {
	a, err := l.accountOf(o.ID)
	if err != nil {
		return shop.Payment{}, err
	}
	if amount > a.Charged-a.Refunded {
		return shop.Payment{}, shop.Errorf(shop.CodeConflict, "refund %d is more than the %d that can be refunded", amount, a.Charged-a.Refunded)
	}
	id, err := l.card.Refund(o.PaymentID, amount)
	if err != nil {
		return shop.Payment{}, err
	}
	a.refund(amount)
	a.checkInvariant() // K27, after refund
	p := shop.Payment{Kind: shop.KindRefund, Amount: amount, ProviderID: id}
	if err := l.store.AddPayment(o.ID, p); err != nil {
		return shop.Payment{}, fmt.Errorf("record the refund %s: %w", id, err)
	}
	return p, nil
}

// Payments returns the ledger of an order.
func (l *Ledger) Payments(id shop.OrderID) ([]shop.Payment, error) {
	return l.store.Payments(id)
}
