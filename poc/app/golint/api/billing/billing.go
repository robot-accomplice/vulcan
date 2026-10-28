// Package billing charges and refunds orders, and keeps the ledger of each order.
package billing

import (
	"fmt"

	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// account is the money of one order: what was charged and what was refunded.
type account struct {
	Charged  shop.Money
	Refunded shop.Money
}

// Invariant holds after every change.
func (a account) Invariant() bool { return a.Refunded >= 0 && a.Refunded <= a.Charged }

// charge records a charge.
func (a *account) charge(amount shop.Money) {
	contract.Requires(amount > 0)
	before := contract.Old(a.Charged)
	defer contract.Keep(a)
	defer contract.Ensures(func() bool { return a.Charged == before+amount })
	a.Charged += amount
}

// refund records a refund.
func (a *account) refund(amount shop.Money) {
	contract.Requires(amount > 0)
	before := contract.Old(a.Refunded)
	defer contract.Keep(a)
	defer contract.Ensures(func() bool { return a.Refunded == before+amount })
	a.Refunded += amount
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
	for _, p := range entries {
		switch p.Kind {
		case shop.KindCharge:
			a.charge(p.Amount)
		case shop.KindRefund:
			a.refund(p.Amount)
		}
	}
	return a, nil
}

// Charge charges the total of an order that is not charged yet, and records it.
func (l *Ledger) Charge(o shop.Order) (pay shop.Payment, err error) {
	contract.Requires(o.Charged == 0 && o.Total > 0)
	defer contract.Ensures(func() bool { return err != nil || pay.Kind == shop.KindCharge && pay.Amount == o.Total })
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
func (l *Ledger) Refund(o shop.Order, amount shop.Money) (pay shop.Payment, err error) {
	contract.Requires(amount > 0)
	defer contract.Ensures(func() bool { return err != nil || pay.Kind == shop.KindRefund && pay.Amount == amount })
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
