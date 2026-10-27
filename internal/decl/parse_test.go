package decl

import "testing"

func TestParseSystem(t *testing.T) {
	src := []byte(`// Shop is the online store.
system Shop {
    person   Customer "Buys goods"
    external Stripe   "Card payment processor"

    config { shutdown_timeout: 5 * time.Second }

    container Api {
        Web{ Addr: flag("web-addr", default: ":8080") }, Billing
        Orders
        group Back { LedgerMemory, StripeGateway }

        Orders.Checkout: Billing.NewPayer
        Billing.CardNetwork: StripeGateway.NewCards{
            APIKey: flag("stripe-key"),
            Limit:  10,
        }
        Billing.Clock: SystemClock.Wall
    }
}
`)
	sys, probs := ParseSystem(src)
	if len(probs) > 0 {
		t.Fatalf("problems: %+v", probs)
	}
	c := sys.Containers[0]
	if len(c.Leaves) != 5 || c.Leaves[3].Group != "Back" {
		t.Fatalf("leaves: %+v", c.Leaves)
	}
	if len(c.Wires) != 3 || c.Wires[1].Settings == nil || len(c.Wires[1].Settings.Items) != 2 {
		t.Fatalf("wires: %+v", c.Wires)
	}
	if f := c.Leaves[0].Settings.Items[0].Value.Flag; f == nil || f.Name != "web-addr" || f.Default.Text != `":8080"` {
		t.Fatalf("flag: %+v", c.Leaves[0].Settings.Items[0].Value)
	}
	if sys.Config == nil || sys.Config.Items[0].Value.Text != "5 * time.Second" {
		t.Fatalf("config: %+v", sys.Config)
	}
}

func TestParseComponent(t *testing.T) {
	src := []byte(`import "shop/lib/money"

component Orders {
    entry for Customer
    // Checkout takes payment.
    needs port Checkout {
        func Pay(customer money.CustomerID, amount money.Money) (money.Receipt, error)
            requires amount > 0 && amount < money.MaxOrder
            ensures  err == nil => result.Amount == amount
        func Ping() error
    }
}
`)
	c, probs := ParseComponent(src)
	if len(probs) > 0 {
		t.Fatalf("problems: %+v", probs)
	}
	if len(c.Ports) != 1 || len(c.Ports[0].Methods) != 2 || len(c.Entries) != 1 {
		t.Fatalf("component: %+v", c)
	}
	m := c.Ports[0].Methods[0]
	if got := string(src[m.SigOff:m.SigEnd]); got != "(customer money.CustomerID, amount money.Money) (money.Receipt, error)" {
		t.Fatalf("sig %q", got)
	}
	if len(m.Contracts) != 2 || string(src[m.Contracts[1].Off:m.Contracts[1].End]) != "err == nil => result.Amount == amount" {
		t.Fatalf("contracts %+v", m.Contracts)
	}
	if got := string(src[c.Ports[0].Methods[1].SigOff:c.Ports[0].Methods[1].SigEnd]); got != "() error" {
		t.Fatalf("sig2 %q", got)
	}
}

func TestTruncated(t *testing.T) {
	ParseComponent([]byte("component Billing {\n    publishes P topic["))
	ParseSystem([]byte("system A0{A0.A:A0.A00{A0:00000\x97"))
	ParseSystem([]byte("system S { container A { X.P: Y.N{ a: flag("))
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("system S {\n container Api {\n  Web{ Addr: flag(\"a\", default: \":8080\") }, B\n  Web.P: B.New{ X: 1 }\n }\n}\n"))
	f.Add([]byte("import \"x/lib/m\"\ncomponent C {\n needs port P {\n  func F(a m.T) (m.R, error)\n   requires a > 0\n }\n gateway for X\n}\n"))
	f.Fuzz(func(t *testing.T, src []byte) {
		ParseSystem(src)
		ParseComponent(src)
		ParseLibrary(src)
	})
}
