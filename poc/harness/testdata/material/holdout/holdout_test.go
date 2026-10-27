package holdout

import (
	"testing"

	st "github.com/robot-accomplice/vulcan/poc/stockroomtest"
)

func TestOrdersOfNobody(t *testing.T) {
	a := st.Start(t)
	a.Want("GET", "/orders?customer=nobody", nil, 200, "")
}
