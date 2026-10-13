package orders

import (
	"errors"
	"testing"

	"shop/shop"
)

// fakeBilling records refunds so a test can see that Cancel asked for one.
type fakeBilling struct {
	refunds   []shop.Money
	refundErr error
}

func (f *fakeBilling) Charge(c shop.CustomerID, o shop.OrderID, a shop.Money) (shop.Receipt, error) {
	return shop.Receipt{Customer: c, Order: o, Amount: a}, nil
}

func (f *fakeBilling) Refund(o shop.OrderID, a shop.Money) (shop.Receipt, error) {
	if f.refundErr != nil {
		return shop.Receipt{}, f.refundErr
	}
	f.refunds = append(f.refunds, a)
	return shop.Receipt{Order: o, Amount: a}, nil
}

// Cancel must refund the full amount, because the customer paid it at Place.
func TestCancelRefundsFullAmount(t *testing.T) {
	b := &fakeBilling{}
	s := New(b)
	o, err := s.Place("c1", 2500)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Cancel(o.ID); err != nil {
		t.Fatal(err)
	}
	if len(b.refunds) != 1 || b.refunds[0] != 2500 {
		t.Fatalf("refunds = %v, want [2500]", b.refunds)
	}
	if _, err := s.Cancel(o.ID); !errors.Is(err, ErrCancelled) {
		t.Fatalf("second cancel: got %v, want ErrCancelled", err)
	}
	if len(b.refunds) != 1 {
		t.Fatalf("second cancel refunded again: %v", b.refunds)
	}
}

// A failed refund must leave the order placed, so the customer is not left unpaid.
func TestCancelKeepsOrderWhenRefundFails(t *testing.T) {
	b := &fakeBilling{refundErr: errors.New("down")}
	s := New(b)
	o, _ := s.Place("c1", 100)
	if _, err := s.Cancel(o.ID); err == nil {
		t.Fatal("cancel succeeded with failed refund")
	}
	got, _ := s.Get(o.ID)
	if got.Status != Placed {
		t.Fatalf("status = %s, want placed", got.Status)
	}
}

// The bus delivers at least once, so a repeated event must not fail or change the order.
func TestShippedTwiceIsIdempotent(t *testing.T) {
	s := New(&fakeBilling{})
	o, _ := s.Place("c1", 100)
	ev := shop.OrderShipped{Order: o.ID, Note: "left at door"}
	for i := 0; i < 2; i++ {
		if err := s.HandleShipped(ev); err != nil {
			t.Fatalf("delivery %d: %v", i+1, err)
		}
	}
	got, _ := s.Get(o.ID)
	if got.Status != Shipped || got.ShippingNote != "left at door" {
		t.Fatalf("got %+v", got)
	}
	if _, err := s.Cancel(o.ID); !errors.Is(err, ErrShipped) {
		t.Fatalf("cancel shipped: got %v, want ErrShipped", err)
	}
}
