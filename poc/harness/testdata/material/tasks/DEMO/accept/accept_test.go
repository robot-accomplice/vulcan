package accept

import (
	"testing"

	st "github.com/robot-accomplice/vulcan/poc/stockroomtest"
)

func TestHealth(t *testing.T) {
	a := st.Start(t)
	a.Want("GET", "/health", nil, 200, "")
}

func TestProduct(t *testing.T) {
	a := st.Start(t)
	a.Product("A-1", "Lamp", 100, 2)
	a.Stock(t, "A-1", 2, 0)
}
