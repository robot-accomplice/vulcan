package rt

import (
	"testing"

	st "github.com/robot-accomplice/vulcan/poc/stockroomtest"
)

func TestRefundLimit(t *testing.T) {
	a := st.Start(t)
	a.Product("A-1", "Lamp", 100, 2)
	a.Order("ada", "A-1", 1)
	a.Want("POST", "/orders/o-1/pay", nil, 200, "")
	a.Want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 101}, 409, "conflict")
	o := a.Want("GET", "/orders/o-1", nil, 200, "")
	if st.Num(o, "refunded_cents") != 0 {
		t.Fatalf("refunded: %v", o)
	}
}
