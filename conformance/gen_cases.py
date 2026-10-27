# gen_cases.py writes the conformance cases as txtar archives: python gen_cases.py .
# Each expected position comes from a text marker, @@text@ (or @n@text@ for the nth match).

import os, re, sys

OUT = sys.argv[1]

BASE = {
"system.vulcan": '''// Shop is the online store.
system Shop {
    person Customer "Buys goods"

    container Api {
        Web, Orders, Billing

        Web.Orders:      Orders.NewService
        Orders.Checkout: Billing.NewPayer
    }
}
''',
"lib/money/library.vulcan": '''// money holds amounts.
library money {
}
''',
"lib/money/money.vulcan": '''package money

// Money is an amount in cents.
type Money int64

// Receipt records a payment.
type Receipt struct {
	Amount Money
}
''',
"api/web/component.vulcan": '''import "shop/lib/money"

component Web {
    entry for Customer

    needs port Orders {
        func Place(amt money.Money) (money.Receipt, error)
    }
}
''',
"api/web/web.vulcan": '''package web

import (
	"context"
	"fmt"
)

func startup(ctx context.Context) error {
	r, err := Orders.Place(42)
	if err != nil {
		return err
	}
	fmt.Println("paid", r.Amount)
	return nil
}
''',
"api/orders/component.vulcan": '''import "shop/lib/money"

component Orders {
    needs port Checkout {
        func Pay(amt money.Money) (money.Receipt, error)
    }
}
''',
"api/orders/orders.vulcan": '''package orders

import "shop/lib/money"

// Service places orders.
type Service struct{}

// NewService returns a Service.
func NewService() *Service { return &Service{} }

// Place takes payment for an order.
func (s *Service) Place(amt money.Money) (money.Receipt, error) {
	return Checkout.Pay(amt)
}
''',
"api/billing/component.vulcan": '''component Billing {
}
''',
"api/billing/billing.vulcan": '''package billing

import "shop/lib/money"

// Payer takes payments.
type Payer struct{}

// NewPayer returns a Payer.
func NewPayer() *Payer { return &Payer{} }

// Pay takes a payment.
func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {
	return money.Receipt{Amount: amt}, nil
}
''',
}

def sub(files, path, old, new):
    s = files[path]
    assert s.count(old) == 1, (path, old)
    files[path] = s.replace(old, new)

CASES = []
def case(id, doc, edits, want, extra=None, base=True):
    files = dict(BASE) if base else {}
    for e in edits:
        if len(e) == 2:
            files[e[0]] = e[1]
        elif e[1] is None:
            del files[e[0]]
        else:
            sub(files, *e)
    if extra:
        files.update(extra)
    CASES.append((id, doc, files, want))

def resolve(files, want):
    out = []
    for line in want.strip().split("\n"):
        m = re.match(r'^(error|warn)\s+(\S+?):@(\d*)@(.+?)@\s+(K\d{4})$', line.strip())
        if m:
            kind, path, nth, text, code = m.groups()
            s = files[path]
            n = int(nth or 1)
            idx = -1
            for _ in range(n):
                idx = s.index(text, idx + 1)
            line_no = s[:idx].count("\n") + 1
            col = idx - (s.rfind("\n", 0, idx) + 1) + 1
            out.append(f"{kind} {path}:{line_no}:{col} {code}")
        else:
            out.append(line.strip())
    return "\n".join(out) + "\n"

# ---- Startup and shutdown
case("W01", "Two leaves, a port between them, a library type in the signature, an entry leaf whose startup calls the port and returns.",
     [], "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})

case("W02", "A constructor returns (T, error) and fails: exit 1 before any setup, with a report that names the constructor.",
     [("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }",
       'func NewPayer() (*Payer, error) { return nil, errors.New("no card network") }'),
      ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"errors"\n\n\t"shop/lib/money"\n)'),
      ("api/web/web.vulcan", "func startup(", 'func setup(ctx context.Context) error {\n\tfmt.Println("setup ran")\n\treturn nil\n}\n\nfunc startup(')],
     "run exit=1 stdout=out.txt", {"out.txt": "Shop/Api: startup failed:\nconstructor Billing.NewPayer: no card network\n"})

case("W03", "A setup returns an error: exit 1; no startup runs.",
     [("api/web/web.vulcan", "func startup(", 'func setup(ctx context.Context) error {\n\treturn fmt.Errorf("no database")\n}\n\nfunc startup(')],
     "run exit=1 stdout=out.txt", {"out.txt": "Shop/Api: setup failed:\nWeb: no database\n"})

case("W05", "A constructor calls its own port: accepted; the call reaches the provider.",
     [("api/orders/orders.vulcan", "func NewService() *Service { return &Service{} }",
       'func NewService() *Service {\n\tr, _ := Checkout.Pay(1)\n\tfmt.Println("warm", r.Amount)\n\treturn &Service{}\n}'),
      ("api/orders/orders.vulcan", 'import "shop/lib/money"', 'import (\n\t"fmt"\n\n\t"shop/lib/money"\n)')],
     "run exit=0 stdout=out.txt", {"out.txt": "warm 1\npaid 42\n"})

case("W06", "setup with the wrong signature.",
     [("api/web/web.vulcan", "func startup(", "func setup() error { return nil }\n\nfunc startup(")],
     "error api/web/web.vulcan:@@setup@ K0601")

case("W07", "A secret setting given a literal.",
     [("system.vulcan", "Orders.Checkout: Billing.NewPayer", 'Orders.Checkout: Billing.NewPayer{ Key: "sk_live_123" }'),
      ("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }",
       '// Settings configure a Payer.\ntype Settings struct {\n\tKey string `vulcan:"secret"`\n}\n\nfunc NewPayer(s Settings) *Payer { return &Payer{} }')],
     'error system.vulcan:@@"sk_live_123"@ K0605')

case("W08", "Two flags that map to one environment variable.",
     [("system.vulcan", "Orders.Checkout: Billing.NewPayer", 'Orders.Checkout: Billing.NewPayer{ Key: flag("card-key"), Alt: flag("card.key") }'),
      ("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }",
       '// Settings configure a Payer.\ntype Settings struct {\n\tKey, Alt string\n}\n\nfunc NewPayer(s Settings) *Payer { return &Payer{} }')],
     'error system.vulcan:@@"card.key"@ K0602')

case("P01", "A required flag is missing at startup: exit 1 with a report that names the flag and its environment variable.",
     [("system.vulcan", "Orders.Checkout: Billing.NewPayer", 'Orders.Checkout: Billing.NewPayer{ Key: flag("card-key") }'),
      ("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }",
       '// Settings configure a Payer.\ntype Settings struct {\n\tKey string\n}\n\nfunc NewPayer(s Settings) *Payer { return &Payer{} }')],
     "run exit=1 stdout=out.txt", {"out.txt": "Shop/Api: reading flags failed:\nflag --card-key (or CARD_KEY) is required\n"})

case("P02", "Leaf settings with a flag default, and a flag given on the command line.",
     [("system.vulcan", "Web, Orders, Billing", 'Web{ Greeting: flag("greeting", default: "paid") }, Orders, Billing'),
      ("api/web/web.vulcan", "func startup(ctx context.Context) error {", "// Settings configure Web.\ntype Settings struct {\n\tGreeting string\n}\n\nfunc startup(ctx context.Context, s Settings) error {"),
      ("api/web/web.vulcan", 'fmt.Println("paid", r.Amount)', "fmt.Println(s.Greeting, r.Amount)")],
     "run exit=0 arg=--greeting=charged stdout=out.txt", {"out.txt": "charged 42\n"})

# ---- Architecture and wiring
case("A01", "Leaf Orders imports the package of leaf Billing.",
     [("api/orders/orders.vulcan", 'import "shop/lib/money"', 'import (\n\t"shop/api/billing"\n\t"shop/lib/money"\n)'),
      ("api/orders/orders.vulcan", "return Checkout.Pay(amt)", "return billing.NewPayer().Pay(amt)")],
     'error api/orders/orders.vulcan:@@"shop/api/billing"@ K0101')

case("A02", "Leaf Orders names a type of Billing in its component.vulcan.",
     [("api/orders/component.vulcan", 'import "shop/lib/money"', 'import (\n    "shop/api/billing"\n    "shop/lib/money"\n)'),
      ("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(p *billing.Payer, amt money.Money) (money.Receipt, error)"),
      ("api/orders/orders.vulcan", "return Checkout.Pay(amt)", "return Checkout.Pay(nil, amt)"),
      ("api/billing/billing.vulcan", "Pay(amt money.Money) (money.Receipt, error) {", "Pay(_ *Payer, amt money.Money) (money.Receipt, error) {")],
     'error api/orders/component.vulcan:@@"shop/api/billing"@ K0110')

case("A03", "Orders needs port Checkout, and system.vulcan has no wire for it.",
     [("system.vulcan", "        Orders.Checkout: Billing.NewPayer\n", "")],
     "error api/orders/component.vulcan:@@Checkout@ K0104")

case("A04", "Two wires for Orders.Checkout.",
     [("system.vulcan", "        Orders.Checkout: Billing.NewPayer\n", "        Orders.Checkout: Billing.NewPayer\n        Orders.Checkout: Billing.Payer\n")],
     "error system.vulcan:@2@Orders.Checkout@ K0105")

case("A05", "A wire names Billing.NewPayr, which does not exist.",
     [("system.vulcan", "Billing.NewPayer", "Billing.NewPayr")],
     "error system.vulcan:@@NewPayr@ K0107")

case("A06", "The provider has the port method with a different parameter type.",
     [("api/billing/billing.vulcan", "Pay(amt money.Money) (money.Receipt, error) {\n\treturn money.Receipt{Amount: amt}, nil",
       "Pay(amt int) (money.Receipt, error) {\n\treturn money.Receipt{Amount: money.Money(amt)}, nil")],
     "error system.vulcan:@@NewPayer@ K0108")

case("A07", "Pay has a pointer receiver, and the constructor NewPayer returns Payer, not *Payer.",
     [("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }", "func NewPayer() Payer { return Payer{} }")],
     "error system.vulcan:@@NewPayer@ K0108")

case("A07b", "A wire that names the type Payer provides new(Payer), a *Payer, and is accepted.",
     [("system.vulcan", "Billing.NewPayer", "Billing.Payer")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})

CYCLE = {
"system.vulcan": '''system Loop {
    container Api {
        A, B, C

        A.P: B.New
        B.Q: C.New
        C.R: A.New
    }
}
''',
}
for leaf, port in (("a", "P"), ("b", "Q"), ("c", "R")):
    CYCLE[f"api/{leaf}/component.vulcan"] = f"component {leaf.upper()} {{\n    needs port {port} {{\n        func Do() error\n    }}\n}}\n"
    CYCLE[f"api/{leaf}/{leaf}.vulcan"] = f"package {leaf}\n\n// T does the work.\ntype T struct{{}}\n\n// New returns a T.\nfunc New() *T {{ return &T{{}} }}\n\n// Do does it.\nfunc (*T) Do() error {{ return nil }}\n"
case("A08", "Wires A.P: B.New, B.Q: C.New, C.R: A.New.", [], "error system.vulcan:@@A.P: B.New@ K0111", CYCLE, base=False)

case("A09", "A port method parameter of a type private to Orders.",
     [("api/orders/component.vulcan", "component Orders {", "component Orders {\n    // Checkout takes payment for an order."),
      ("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(o Order, amt money.Money) (money.Receipt, error)"),
      ("api/orders/orders.vulcan", "// Service places orders.", "// Order is an order.\ntype Order struct{ ID int }\n\n// Service places orders."),
      ("api/orders/orders.vulcan", "return Checkout.Pay(amt)", "return Checkout.Pay(Order{}, amt)")],
     "error api/orders/component.vulcan:@@Pay@ K0112")

case("A10", "Code assigns to the port name Checkout.",
     [("api/orders/orders.vulcan", "return Checkout.Pay(amt)", "Checkout = nil\n\treturn money.Receipt{}, nil")],
     "error api/orders/orders.vulcan:@@Checkout = nil@ K0116")

case("A11", "A leaf declares func init().",
     [("api/orders/orders.vulcan", "// Service places orders.", "func init() {}\n\n// Service places orders.")],
     "error api/orders/orders.vulcan:@@init@ K0117")

case("A12", "A package-level var in a leaf whose initializer calls a port.",
     [("api/orders/orders.vulcan", "// Service places orders.", "var first, _ = Checkout.Pay(1)\n\n// Service places orders.")],
     "error api/orders/orders.vulcan:@@Checkout.Pay(1)@ K0117")

case("A13", "//go:linkname in leaf Orders that names a function of Billing.",
     [("api/orders/orders.vulcan", 'import "shop/lib/money"', 'import (\n\t_ "unsafe"\n\n\t"shop/lib/money"\n)\n\n//go:linkname pay shop/api/billing.pay\nfunc pay() error')],
     "error api/orders/orders.vulcan:@@//go:linkname@ K0122")

case("A14", "A library imports a leaf.",
     [], 'error lib/util/util.vulcan:@@"shop/api/billing"@ K0204',
     {"lib/util/library.vulcan": "library util {\n}\n",
      "lib/util/util.vulcan": 'package util\n\nimport "shop/api/billing"\n\n// P is a payer.\nvar P = billing.NewPayer\n'})

case("A15", "A library var assigned in a function after startup.",
     [("lib/money/money.vulcan", "// Money is an amount in cents.", "// Rate is the tax rate.\nvar Rate = 20\n\n// SetRate changes the rate.\nfunc SetRate(r int) { Rate = r }\n\n// Money is an amount in cents.")],
     "error lib/money/money.vulcan:@@Rate = r@ K0202")

case("A16", "Two leaves with one name in different groups.",
     [("system.vulcan", "Web, Orders, Billing", "Web\n        group Front { Orders }\n        group Back { Billing, Orders }")],
     "error system.vulcan:@2@Orders }@ K0121")

case("A17", "A wire names a group.",
     [("system.vulcan", "Web, Orders, Billing", "Web, Orders\n        group Back { Billing }"),
      ("system.vulcan", "Orders.Checkout: Billing.NewPayer", "Orders.Checkout: Back.NewPayer")],
     "error system.vulcan:@@Back.NewPayer@ K0113")

# ---- Lexing and positions
case("S01", "A method with three contract lines between its signature and a { on its own line: accepted, and it runs.",
     [("api/billing/billing.vulcan", "func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {",
       "func (p *Payer) Pay(amt money.Money) (money.Receipt, error)\n\trequires amt > 0\n\trequires amt < 1_000_000\n\tensures err == nil => result.Amount == amt\n{")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})

case("S02", "func F() requires { … }: a result of a type named requires.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "type requires *int\n\nfunc f() requires { return nil }\n\nvar _ = f\n\n// Payer takes payments.")],
     "accept")

case("S03", "A struct field named limit, a local variable named requires, a parameter named timeout.",
     [("api/billing/billing.vulcan", "// Payer takes payments.\ntype Payer struct{}", "// Payer takes payments.\ntype Payer struct {\n\tlimit int\n}\n\nfunc wait(timeout int) int {\n\trequires := timeout + 1\n\treturn requires\n}\n\nvar _ = wait")],
     "accept")

case("S04", "limit on the line after its field: K0009 at the limit, because release 1 defers limit.",
     [("lib/money/money.vulcan", "\tAmount Money\n", "\tAmount Money\n\tNote string\n\t\tlimit 200\n")],
     "error lib/money/money.vulcan:@@limit@ K0009")

case("S05", "A declared name vulcan_x.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "var vulcan_x = 1\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@vulcan_x@ K0005")

case("S06", "A body with a Go type error, inside a file with contracts above it: K0008 at the true .vulcan position.",
     [("api/billing/billing.vulcan", "func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {\n\treturn money.Receipt{Amount: amt}, nil",
       "func (p *Payer) Pay(amt money.Money) (money.Receipt, error)\n\trequires amt > 0\n{\n\tvar s string = amt; _ = s\n\treturn money.Receipt{Amount: amt}, nil")],
     "error api/billing/billing.vulcan:@@amt; _@ K0008")

for sid, doc, edits, marker, path in (
    ("S08a", "A topic: publishes.", [("api/orders/component.vulcan", "    needs port Checkout {", "    publishes Placed topic[money.Receipt]\n\n    needs port Checkout {")], "publishes", "api/orders/component.vulcan"),
    ("S08b", "A remote port.", [("api/orders/component.vulcan", "needs port Checkout", "needs remote port Checkout")], "remote", "api/orders/component.vulcan"),
    ("S08c", "via on a wire.", [("system.vulcan", "Orders.Checkout: Billing.NewPayer", "Orders.Checkout: Billing.NewPayer via direct{}")], "direct", "system.vulcan"),
    ("S08d", "A second container.", [("system.vulcan", "    }\n}\n", "    }\n\n    container Worker {\n    }\n}\n")], "Worker", "system.vulcan"),
    ("S08e", "pure.", [("api/billing/billing.vulcan", "// Payer takes payments.", "pure func positive(m money.Money) bool { return m > 0 }\n\nvar _ = positive\n\n// Payer takes payments.")], "pure", "api/billing/billing.vulcan"),
    ("S08f", "handle.", [("api/billing/billing.vulcan", "// Payer takes payments.", "handle Placed(r money.Receipt) error {\n\treturn nil\n}\n\n// Payer takes payments.")], "handle", "api/billing/billing.vulcan"),
):
    case(sid, "Deferred construct: " + doc, edits, f"error {path}:@@{marker}@ K0009")

case("S09", "A Go feature newer than Go 1.22 in a body (a range-over-func iterator): accepted, with Go's output.",
     [("api/web/web.vulcan", 'fmt.Println("paid", r.Amount)', 'for i := range count(2) {\n\t\tfmt.Println("paid", r.Amount, i)\n\t}'),
      ("api/web/web.vulcan", "func startup(", "func count(n int) func(func(int) bool) {\n\treturn func(yield func(int) bool) {\n\t\tfor i := 0; i < n; i++ {\n\t\t\tif !yield(i) {\n\t\t\t\treturn\n\t\t\t}\n\t\t}\n\t}\n}\n\nfunc startup(")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42 0\npaid 42 1\n"})

case("S10", "Each deferred command prints that it is not in the release, and exits 2.", [],
     "\n".join(f"cmd exit=2 say=not-in-this-release vulcan {c}" for c in ("fmt", "explain", "doc", "diagram", "archdiff", "fake", "trace")))

# ---- Layout (PoC additions)
case("P10", "A code file outside a leaf or a library.", [], "error misc/x.vulcan:@@package@ K0002".replace(":@@package@", ""),
     {"misc/x.vulcan": "package misc\n"})
case("P11", "A leaf directory that system.vulcan does not list.", [], "error api/extra/component.vulcan:1:1 K0003",
     {"api/extra/component.vulcan": "component Extra {\n}\n"})
case("P12", "A package clause that does not name its directory.",
     [("api/billing/billing.vulcan", "package billing", "package bill")],
     "error api/billing/billing.vulcan:@@bill@ K0004")
case("P13", "An unknown config key.",
     [("system.vulcan", '    person Customer "Buys goods"', '    person Customer "Buys goods"\n\n    config { colour: "blue" }')],
     "error system.vulcan:@@colour@ K0006")
case("P14", "gateway for a name that is not an external.",
     [("api/billing/component.vulcan", "component Billing {", "component Billing {\n    gateway for Stripe")],
     "error api/billing/component.vulcan:@@Stripe@ K0206")
case("P15", "Two gateway leaves for one external system.",
     [("system.vulcan", '    person Customer "Buys goods"', '    person Customer "Buys goods"\n    external Stripe "Cards"'),
      ("api/billing/component.vulcan", "component Billing {", "component Billing {\n    gateway for Stripe"),
      ("api/orders/component.vulcan", "component Orders {", "component Orders {\n    gateway for Stripe")],
     "error api/billing/component.vulcan:@@Stripe@ K0207")
case("P16", "A test binds a fake with vulcan.Bind; an unbound port fails with a message.",
     [], "accept",
     {"api/orders/orders_test.vulcan": '''package orders

import (
	"testing"

	"github.com/robot-accomplice/vulcan/vulcan"
	"shop/lib/money"
)

type fake struct{}

func (fake) Pay(amt money.Money) (money.Receipt, error) { return money.Receipt{Amount: amt * 2}, nil }

func TestPlace(t *testing.T) {
	vulcan.Bind(t, &Checkout, fake{})
	r, err := NewService().Place(2)
	if err != nil || r.Amount != 4 {
		t.Fatal(r, err)
	}
}
'''})
case("P17", "Code imports the runtime package rt, which only generated code may use.",
     [("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"github.com/robot-accomplice/vulcan/vulcan/rt"\n\t"shop/lib/money"\n)\n\nvar _ = rt.Fail')],
     'error api/billing/billing.vulcan:@@"github.com/robot-accomplice/vulcan/vulcan/rt"@ K0118')
case("P18", "A constructor with a parameter that is not a settings struct.",
     [("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }", "func NewPayer(n int) *Payer { return &Payer{} }")],
     "error system.vulcan:@@NewPayer@ K0604")

# ---- Cases added after the P1 audit (experiments/poc/audit-p1.md)
WEB_SETTINGS = [("api/web/web.vulcan", "func startup(ctx context.Context) error {", "// Settings configure Web.\ntype Settings struct {\n\tWait time.Duration\n}\n\nfunc startup(ctx context.Context, s Settings) error {"),
                ("api/web/web.vulcan", 'fmt.Println("paid", r.Amount)', 'fmt.Println("paid", r.Amount, s.Wait)'),
                ("api/web/web.vulcan", 'import (\n\t"context"\n\t"fmt"\n)', 'import (\n\t"context"\n\t"fmt"\n\t"time"\n)')]
case("P20", "A leaf setting of type time.Duration, given as a constant that uses time (audit H1).",
     WEB_SETTINGS + [("system.vulcan", "Web, Orders, Billing", "Web{ Wait: 2 * time.Second }, Orders, Billing")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42 2s\n"})
case("P21", "The identifier limit in valid Go: a parameter, an array length, a field name (audit H2).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "const limit = 4\n\ntype pager struct {\n\tfetch func(limit int) []int\n\tbuf   [limit]byte\n\tlimit int\n}\n\nvar _ = pager{}\n\n// Payer takes payments.")],
     "accept")
LIB_STATE = ("lib/money/money.vulcan", "// Money is an amount in cents.", "// Config holds fees.\ntype Config struct{ Fee Money }\n\n// Default is the default config.\nvar Default = Config{Fee: 1}\n\n// Rates are the tax rates.\nvar Rates = [2]Money{1, 2}\n\n// Limit is the largest order.\nvar Limit Money = 100\n\n// Fees are the fees by country.\nvar Fees = map[string]Money{}\n\n// Money is an amount in cents.")
case("A15b", "A field of a library struct var assigned in a leaf (audit H3).",
     [LIB_STATE, ("api/billing/billing.vulcan", "// Payer takes payments.", "func tweak() { money.Default.Fee = 99 }\n\nvar _ = tweak\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@money.Default.Fee@ K0202")
case("A15c", "An element of a library array var assigned (audit H3).",
     [LIB_STATE, ("api/billing/billing.vulcan", "// Payer takes payments.", "func tweak() { money.Rates[0] = 7 }\n\nvar _ = tweak\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@money.Rates[0]@ K0202")
case("A15d", "A library var assigned in package-level function literals (audit H3).",
     [LIB_STATE, ("api/billing/billing.vulcan", "// Payer takes payments.", "var _ = func() bool { money.Limit = 1; return true }()\n\nvar reset = func() { money.Limit = 0 }\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@money.Limit = 1@ K0202\nerror api/billing/billing.vulcan:@@money.Limit = 0@ K0202")
case("A15e", "Near miss: reading library vars, copying one, and writing through a map are not assignments to the var.",
     [LIB_STATE, ("api/billing/billing.vulcan", "// Payer takes payments.", "func use() money.Money {\n\tc := money.Default\n\tc.Fee = 5\n\tmoney.Fees[\"ch\"] = 3\n\treturn c.Fee + money.Limit + money.Rates[1]\n}\n\nvar _ = use\n\n// Payer takes payments.")],
     "accept")
case("A12b", "A package-level var whose initializer reaches a port through a function (audit M2).",
     [("api/orders/orders.vulcan", "// Service places orders.", "func warm() money.Receipt {\n\tr, _ := Checkout.Pay(1)\n\treturn r\n}\n\nvar first = warm()\n\n// Service places orders.")],
     "error api/orders/orders.vulcan:@2@warm()@ K0117")
case("A12c", "Near miss: a package-level var that stores a function literal, which calls a port later.",
     [("api/orders/orders.vulcan", "// Service places orders.", "var later = func() { _, _ = Checkout.Pay(2) }\n\n// Service places orders.")],
     "accept")
case("A10b", "A range statement that assigns to a port (audit L1).",
     [("api/orders/orders.vulcan", "return Checkout.Pay(amt)", "var ch chan interface {\n\t\tPay(money.Money) (money.Receipt, error)\n\t}\n\tfor Checkout = range ch {\n\t}\n\treturn money.Receipt{}, nil")],
     "error api/orders/orders.vulcan:@@Checkout = range@ K0116")
case("A10c", "Near miss: a local variable named like the port.",
     [("api/orders/orders.vulcan", "return Checkout.Pay(amt)", "r, err := Checkout.Pay(amt)\n\tCheckout := 3\n\tCheckout = 4\n\t_ = Checkout\n\treturn r, err")],
     "accept")
case("A06b", "The provider lacks the port method.",
     [("api/billing/billing.vulcan", "// Pay takes a payment.\nfunc (p *Payer) Pay(", "// Charge takes a payment.\nfunc (p *Payer) Charge(")],
     "error system.vulcan:@@NewPayer@ K0108")
case("P22", "A string constant for a time.Duration setting (audit H4).",
     WEB_SETTINGS + [("system.vulcan", "Web, Orders, Billing", 'Web{ Wait: "soon" }, Orders, Billing')],
     'error system.vulcan:@@"soon"@ K0606')
case("P23", "A flag default of the wrong type for a time.Duration setting (audit H4).",
     WEB_SETTINGS + [("system.vulcan", "Web, Orders, Billing", 'Web{ Wait: flag("wait", default: "5s") }, Orders, Billing')],
     'error system.vulcan:@@"5s"@ K0606')
case("P24", "A constructor whose settings type is not exported (audit H4).",
     [("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }", "type opts struct{ Fee int }\n\nfunc NewPayer(o opts) *Payer { return &Payer{} }")],
     "error system.vulcan:@@NewPayer@ K0604")
case("P25", "import \"time\" in system.vulcan is harmless; a flag default uses time (audit H4).",
     WEB_SETTINGS + [("system.vulcan", "// Shop is the online store.", 'import "time"\n\n// Shop is the online store.'),
                     ("system.vulcan", "Web, Orders, Billing", 'Web{ Wait: flag("wait", default: 2 * time.Second) }, Orders, Billing')],
     "run exit=0 arg=--wait=3s stdout=out.txt", {"out.txt": "paid 42 3s\n"})
case("P26", "A truncated code file gives a syntax error, not a crash (audit M1).",
     [], "error api/billing/extra.vulcan:3:8 K0001", {"api/billing/extra.vulcan": "package billing\n\nfunc ("})
case("P27", "A leaf setting with no value in system.vulcan (audit M3).",
     WEB_SETTINGS, "error system.vulcan:@@Web,@ K0603")
case("P28", "A failed startup while another startup ignores ctx: report and exit 1 within shutdown_timeout (audit M4).",
     [("system.vulcan", '    person Customer "Buys goods"', '    person Customer "Buys goods"\n\n    config { shutdown_timeout: 1 * time.Second }'),
      ("api/web/web.vulcan", "\tr, err := Orders.Place(42)", "\tif ctx != nil {\n\t\treturn fmt.Errorf(\"boom\")\n\t}\n\tr, err := Orders.Place(42)"),
      ("api/billing/billing.vulcan", "// Payer takes payments.", "func startup(ctx context.Context) error {\n\tselect {}\n}\n\n// Payer takes payments."),
      ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"context"\n\n\t"shop/lib/money"\n)')],
     "run exit=1 stdout=out.txt", {"out.txt": "Shop/Api: startup failed:\nWeb: boom\nShop/Api: 1 startup functions did not stop within 1s\n"})
case("P29", "A .go file in a leaf directory (audit M5).",
     [], "error api/billing/helper.go:1:1 K0002", {"api/billing/helper.go": "package billing\n\nfunc fee() int { return 1 }\n"})
case("P30", "A test reads a file in testdata (audit M5).",
     [], "accept\ncmd exit=0 vulcan test Billing",
     {"api/billing/testdata/fee.txt": "7\n",
      "api/billing/billing_test.vulcan": 'package billing\n\nimport (\n\t"os"\n\t"testing"\n)\n\nfunc TestFee(t *testing.T) {\n\tb, err := os.ReadFile("testdata/fee.txt")\n\tif err != nil || string(b[:1]) != "7" {\n\t\tt.Fatal(err)\n\t}\n}\n'})
case("P31", "An import of a module that go.mod does not require (audit M6).",
     [("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"example.com/notreal/pkg"\n\n\t"shop/lib/money"\n)\n\nvar _ = pkg.X')],
     'error api/billing/billing.vulcan:@@"example.com/notreal/pkg"@ K0209')
case("P32", "A function literal in a system.vulcan value (audit M7).",
     [("system.vulcan", "Orders.Checkout: Billing.NewPayer", "Orders.Checkout: Billing.NewPayer{ Fee: func() int64 { return 1 }() }"),
      ("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }", "// Settings configure a Payer.\ntype Settings struct {\n\tFee int64\n}\n\nfunc NewPayer(s Settings) *Payer { return &Payer{} }")],
     "error system.vulcan:@@func()@ K0606")
case("P33", "system.vulcan imports a leaf (audit M7).",
     [("system.vulcan", "// Shop is the online store.", 'import "shop/api/billing"\n\n// Shop is the online store.')],
     'error system.vulcan:@@"shop/api/billing"@ K0110')
case("P34", "A port declared twice in one component (audit M8).",
     [("api/orders/component.vulcan", "    needs port Checkout {", "    needs port Checkout {\n        func Refund(amt money.Money) error\n    }\n\n    needs port Checkout {")],
     "error api/orders/component.vulcan:@2@Checkout@ K0110")
case("P35", "A port with the name of a function of its leaf (audit M8).",
     [("api/orders/orders.vulcan", "// Service places orders.", "func Checkout() {}\n\n// Service places orders.")],
     "error api/orders/component.vulcan:@@Checkout@ K0110")
case("P36", "An unexported port method (audit L10).",
     [("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func pay(amt money.Money) (money.Receipt, error)")],
     "error api/orders/component.vulcan:@@pay@ K0110")
case("P37", "vulcan.Bind with a fake that does not satisfy the port (audit L2).",
     [], "error api/orders/orders_test.vulcan:@@fake{}@ K0108",
     {"api/orders/orders_test.vulcan": 'package orders\n\nimport (\n\t"testing"\n\n\t"github.com/robot-accomplice/vulcan/vulcan"\n)\n\ntype fake struct{}\n\nfunc (fake) Pay(amt int) error { return nil }\n\nfunc TestPlace(t *testing.T) {\n\tvulcan.Bind(t, &Checkout, fake{})\n}\n'})
case("P38", "Near miss: //go:linkname inside a raw string (audit L4).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "var doc = `\n//go:linkname x y\n`\n\nvar _ = doc\n\n// Payer takes payments.")],
     "accept")
case("P39", "Near miss: a type named invariant in a type group (audit L5).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "type (\n\tlevel     int\n\tinvariant func() bool\n)\n\nvar _ level\nvar _ invariant\n\n// Payer takes payments.")],
     "accept")
case("P40", "A library outside lib/ (audit L9).",
     [], "error api/util/library.vulcan:1:1 K0003",
     {"api/util/library.vulcan": "library util {\n}\n", "api/util/util.vulcan": "package util\n\n// One is one.\nconst One = 1\n"})
case("P41", "A library that declares a port.",
     [("lib/money/library.vulcan", "library money {\n}", "library money {\n    needs port Rates {\n        func Get() int\n    }\n}")],
     "error lib/money/library.vulcan:@@needs@ K0201")
case("P42", "One flag with two types.",
     [("system.vulcan", "Orders.Checkout: Billing.NewPayer", 'Orders.Checkout: Billing.NewPayer{ Key: flag("card"), Max: flag("card") }'),
      ("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }",
       '// Settings configure a Payer.\ntype Settings struct {\n\tKey string\n\tMax int\n}\n\nfunc NewPayer(s Settings) *Payer { return &Payer{} }')],
     'error system.vulcan:@2@"card"@ K0602')
case("P43", "A wire that names a type takes no settings.",
     [("system.vulcan", "Orders.Checkout: Billing.NewPayer", "Orders.Checkout: Billing.Payer{ Fee: 1 }")],
     "error system.vulcan:@@{ Fee: 1 }@ K0604")
case("S07", "A panic in a body: the stack trace names the .vulcan file and line.",
     [("api/web/web.vulcan", "\tr, err := Orders.Place(42)", "\tdone := make(chan bool)\n\tgo func() {\n\t\tvar m map[string]int\n\t\tm[\"x\"] = 1\n\t\tdone <- true\n\t}()\n\t<-done\n\tr, err := Orders.Place(42)")],
     "run exit=2 contains=want-trace.txt", {"want-trace.txt": "assignment to entry in nil map\napi/web/web.vulcan:12\n"})

# ---- Phase P2: contracts, the network rule, guide
PAY = "func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {"
def pay_contract(clauses):
    return ("api/billing/billing.vulcan", PAY, "func (p *Payer) Pay(amt money.Money) (money.Receipt, error)\n" + "".join("\t" + c + "\n" for c in clauses) + "{")
ZERO_CALL = ("api/web/web.vulcan", "\tr, err := Orders.Place(42)", "\tvar amt money.Money\n\tr, err := Orders.Place(amt)")
WEB_MONEY = ("api/web/web.vulcan", 'import (\n\t"context"\n\t"fmt"\n)', 'import (\n\t"context"\n\t"fmt"\n\n\t"shop/lib/money"\n)')
case("C01", "requires amt > 0, and a caller passes 0 at runtime: a contract failure that names the clause and the value.",
     [pay_contract(["requires amt > 0"]), ZERO_CALL, WEB_MONEY],
     "run exit=1 contains=want.out", {"want.out": "contract failure: Payer.Pay requires: amt > 0 (api/billing/billing.vulcan:13:11)\namt = 0\n"})
case("C02", "requires amt > 0, and a caller passes the constant 0: K0501 at the call.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func charge(amt money.Money) money.Money\n\trequires amt > 0\n{\n\treturn amt\n}\n\nvar _ = func() money.Money { return charge(0) }\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@charge(0)@ K0501")
case("C03", "ensures err == nil => result.Amount == amt, and the body returns a wrong amount.",
     [pay_contract(["ensures err == nil => result.Amount == amt"]),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "return money.Receipt{Amount: amt + 1}, nil")],
     "run exit=1 contains=want.out", {"want.out": "contract failure: Payer.Pay ensures: err == nil => result.Amount == amt\nresult = money.Receipt{Amount:43}\n"})
case("C04", "ensures result.Amount == amt in a function with an error result: K0502.",
     [pay_contract(["ensures result.Amount == amt"])],
     "error api/billing/billing.vulcan:@@result.Amount == amt@ K0502")
case("C06", "requires len(xs) > 0 && xs[0] > 0: accepted.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func first(xs []int) int\n\trequires len(xs) > 0 && xs[0] > 0\n{\n\treturn xs[0]\n}\n\nvar _ = first\n\n// Payer takes payments.")],
     "accept")
case("C07", "old(xs) with a slice xs: K0509.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func grow(xs []int) []int\n\tensures len(result) == len(old(xs)) + 1\n{\n\treturn append(xs, 1)\n}\n\nvar _ = grow\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@len(result)@ K0509")
WINDOW = ("api/billing/billing.vulcan", "// Payer takes payments.", "// Window is a range.\ntype Window struct {\n\tLo, Hi int\n}\n\tinvariant Lo <= Hi\n\n// Payer takes payments.")
case("C09", "invariant Lo < Hi: K0506, because the zero value breaks it.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "// Window is a range.\ntype Window struct {\n\tLo, Hi int\n}\n\tinvariant Lo < Hi\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@Lo < Hi@ K0506")
case("C10", "A field of a struct with an invariant written from a function that is not its method: K0507.",
     [WINDOW, ("api/billing/billing.vulcan", "// Payer takes payments.\ntype Payer", "func widen(w *Window) { w.Hi = w.Hi + 1 }\n\nvar _ = widen\n\n// Payer takes payments.\ntype Payer")],
     "error api/billing/billing.vulcan:@@w.Hi =@ K0507")
case("C11", "&w.Lo outside the methods: K0508.",
     [WINDOW, ("api/billing/billing.vulcan", "// Payer takes payments.\ntype Payer", "func low(w *Window) *int { return &w.Lo }\n\nvar _ = low\n\n// Payer takes payments.\ntype Payer")],
     "error api/billing/billing.vulcan:@@&w.Lo@ K0508")
ACCOUNT = '''// Account holds money.
type Account struct {
	Balance int
}
	invariant Balance >= 0

// Deposit adds money.
func (a *Account) Deposit(x int) { a.Balance += x }

// Rebalance passes through a negative balance, then restores it.
func (a *Account) Rebalance(x int) {
	a.Balance -= 2 * x
	a.Deposit(x)
	a.Deposit(x)
}

// Payer takes payments.'''
case("C12", "A method breaks its invariant for a moment and calls methods through its own receiver: no check on those returns, only on its own.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", ACCOUNT),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "acct := &Account{Balance: 1}\n\tacct.Rebalance(5)\n\treturn money.Receipt{Amount: amt + money.Money(acct.Balance) - 1}, nil")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})
case("C12b", "A call through another expression is checked at its return: the invariant of the other account fails.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", ACCOUNT),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "acct := &Account{Balance: 1}\n\tacct.Deposit(-5)\n\treturn money.Receipt{Amount: amt}, nil")],
     "run exit=1 contains=want.out", {"want.out": "contract failure: Account.Deposit invariant: Balance >= 0\nBalance = -4\n"})
case("C13", "A port contract, and a fake bound with vulcan.Bind that breaks it: the test fails at the port.",
     [("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(amt money.Money) (money.Receipt, error)\n            ensures err == nil => result.Amount == amt")],
     "accept\ncmd exit=1 say=contract-failure:-Orders.Checkout.Pay-ensures vulcan test Orders",
     {"api/orders/orders_test.vulcan": 'package orders\n\nimport (\n\t"testing"\n\n\t"github.com/robot-accomplice/vulcan/vulcan"\n\t"shop/lib/money"\n)\n\ntype double struct{}\n\nfunc (double) Pay(amt money.Money) (money.Receipt, error) { return money.Receipt{Amount: amt * 2}, nil }\n\nfunc TestPlace(t *testing.T) {\n\tvulcan.Bind(t, &Checkout, double{})\n\tNewService().Place(2)\n}\n'})
case("C17", "A type assertion in a contract: K0512.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func name(v any) string\n\trequires v.(string) != \"\"\n{\n\treturn \"\"\n}\n\nvar _ = name\n\n// Payer takes payments.")],
     'error api/billing/billing.vulcan:@@v.(string)@ K0512')
case("C18", "A struct literal that breaks its invariant fails where the code builds it.",
     [WINDOW, ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "w := Window{Lo: int(amt), Hi: 1}\n\t_ = w\n\treturn money.Receipt{Amount: amt}, nil")],
     "run exit=1 contains=want.out", {"want.out": "contract failure: the value built at api/billing/billing.vulcan:19:7 invariant: Lo <= Hi\nLo = 42\n"})
case("P50", "Plain result in a function with two results other than error: K0510.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func split(n int) (int, int)\n\tensures result >= 0\n{\n\treturn n, n\n}\n\nvar _ = split\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@result >= 0@ K0510")
case("P51", "A function with a contract that names a parameter result: K0511.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func keep(result int) int\n\trequires result > 0\n{\n\treturn result\n}\n\nvar _ = keep\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@requires@ K0511\nerror api/billing/billing.vulcan:@@result > 0@ K0510")
case("P52", "A port contract that names an identifier of its own leaf: K0120.",
     [("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(amt money.Money) (money.Receipt, error)\n            requires amt < MaxAmount"),
      ("api/orders/orders.vulcan", "// Service places orders.", "// MaxAmount is the largest payment.\nconst MaxAmount = 1000\n\n// Service places orders.")],
     "error api/orders/component.vulcan:@@amt < MaxAmount@ K0120")
case("P53", "Named results can be used by name in ensures; error wrapping in a deferred call keeps the result.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func half(n int) (h int, e error)\n\trequires n >= 0\n\tensures err == nil => h*2 <= n\n{\n\tdefer func() {\n\t\tif e != nil {\n\t\t\te = fmt.Errorf(\"half: %w\", e)\n\t\t}\n\t}()\n\th = n / 2\n\treturn\n}\n\n// Payer takes payments."),
      ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"fmt"\n\n\t"shop/lib/money"\n)'),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "h, err := half(int(amt))\n\treturn money.Receipt{Amount: money.Money(h * 2)}, err")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})
case("E04", "on_contract_failure: exit: a contract failure ends the process with exit code 3.",
     [pay_contract(["requires amt > 0"]), ZERO_CALL, WEB_MONEY,
      ("system.vulcan", '    person Customer "Buys goods"', '    person Customer "Buys goods"\n\n    config { on_contract_failure: exit }')],
     "run exit=3 contains=want.out", {"want.out": "contract failure: Payer.Pay requires: amt > 0\n"})
NET_BILLING = [("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"net/http"\n\n\t"shop/lib/money"\n)\n\nfunc ping() error {\n\t_, err := http.Get("https://example.com")\n\treturn err\n}\n\nvar _ = ping')]
case("N01", "A leaf that is not a gateway or an entry leaf calls http.Get: K0205.",
     NET_BILLING, "error api/billing/billing.vulcan:@@Get@ K0205")
case("N01b", "Near miss: a gateway leaf calls http.Get.",
     NET_BILLING + [("system.vulcan", '    person Customer "Buys goods"', '    person Customer "Buys goods"\n    external Bank "Card network"'),
                    ("api/billing/component.vulcan", "component Billing {", "component Billing {\n    gateway for Bank")],
     "accept")
case("N03", "A library calls os.Open: K0203.",
     [("lib/money/money.vulcan", "package money\n", 'package money\n\nimport "os"\n\n// Load opens a rate file.\nfunc Load() error {\n\t_, err := os.Open("rates.txt")\n\treturn err\n}\n')],
     "error lib/money/money.vulcan:@@Open@ K0203")
case("N04", "A library queries through a *sql.DB that it receives: accepted.",
     [("lib/money/money.vulcan", "package money\n", 'package money\n\nimport "database/sql"\n\n// Count counts the receipts.\nfunc Count(db *sql.DB) (int, error) {\n\tvar n int\n\terr := db.QueryRow("select count(*) from receipts").Scan(&n)\n\treturn n, err\n}\n')],
     "accept")
case("T05", "vulcan guide prints the primer.", [], "cmd exit=0 say=Vulcan-primer vulcan guide")

# ---- Cases added after the P2 audit (experiments/poc/audit-p2.md)
SYNC = ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"sync"\n\n\t"shop/lib/money"\n)')
LEDGER = '''// Ledger counts charges and refunds under a lock.
type Ledger struct {
	mu       sync.Mutex
	Charged  int
	Refunded int
}
	invariant Charged >= Refunded

// ChargeAndRefund changes both counts under the lock.
func (l *Ledger) ChargeAndRefund(x int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Charged += x
	l.Refunded += x
}

// Payer takes payments.'''
case("B1", "A mutex-guarded invariant under concurrent calls: the check runs before the deferred unlock, so it never sees a half-done change (audit P2, B1).",
     [SYNC, ("api/billing/billing.vulcan", "// Payer takes payments.", LEDGER),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "l := &Ledger{}\n\tvar wg sync.WaitGroup\n\tfor range 8 {\n\t\twg.Add(1)\n\t\tgo func() {\n\t\t\tdefer wg.Done()\n\t\t\tfor range 20000 {\n\t\t\t\tl.ChargeAndRefund(1)\n\t\t\t}\n\t\t}()\n\t}\n\twg.Wait()\n\treturn money.Receipt{Amount: amt}, nil")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})
case("H1", "A port contract that uses a package only in the contract, and a test of the leaf (audit P2, H1).",
     [("api/orders/component.vulcan", 'import "shop/lib/money"', 'import (\n    "math"\n    "shop/lib/money"\n)'),
      ("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(amt money.Money) (money.Receipt, error)\n            requires amt < math.MaxInt32")],
     "accept\ncmd exit=0 vulcan test Orders",
     {"api/orders/orders_test.vulcan": 'package orders\n\nimport "testing"\n\nfunc TestNothing(t *testing.T) {}\n'})
SPREAD = '''// Account holds money.
type Account struct {
	Balance int
}
	invariant Balance >= 0

// Withdraw takes money out.
func (a *Account) Withdraw(x int) { a.Balance -= x }

// Spread takes x from each other account.
func (a *Account) Spread(others []*Account, x int) {
	for _, a := range others {
		a.Withdraw(x)
	}
}

// Payer takes payments.'''
case("H2", "A loop variable that reuses the receiver's name: calls on it are checked (audit P2, H2).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", SPREAD),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "me := &Account{Balance: 10}\n\tme.Spread([]*Account{{Balance: 1}}, 5)\n\treturn money.Receipt{Amount: amt}, nil")],
     "run exit=1 contains=want.out", {"want.out": "contract failure: Account.Withdraw invariant: Balance >= 0\nBalance = -4\n"})
case("H2b", "A loop variable of another type that reuses the receiver's name compiles (audit P2, H2).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "// Log records.\ntype Log struct{ N int }\n\n// Deposit counts.\nfunc (l *Log) Deposit(x int) { l.N += x }\n\n" + SPREAD.replace("for _, a := range others {\n\t\ta.Withdraw(x)", "for _, a := range []*Log{{}} {\n\t\ta.Deposit(x)").replace("// Payer takes payments.", "func (a *Account) Deposit(x int) { a.Balance += x }\n\n// Payer takes payments.")),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "me := &Account{Balance: 10}\n\tme.Spread(nil, 5)\n\treturn money.Receipt{Amount: amt}, nil")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})
for hid, build in (("H3a", "xs := []Account{{Balance: -1}}\n\t_ = xs"), ("H3b", "ps := []*Account{{Balance: -2}}\n\t_ = ps"),
                   ("H3c", 'm := map[string]Account{"k": {Balance: -3}}\n\t_ = m'), ("H3d", "c := Account(Raw{Balance: -4})\n\t_ = c"),
                   ("H3e", "al := Alias{Balance: -5}\n\t_ = al")):
    case(hid, "A value with an invariant built by an elided literal, a conversion, or an alias is checked (audit P2, H3).",
         [("api/billing/billing.vulcan", "// Payer takes payments.", "// Account holds money.\ntype Account struct{ Balance int }\n\tinvariant Balance >= 0\n\n// Raw is unchecked.\ntype Raw struct{ Balance int }\n\n// Alias names Account.\ntype Alias = Account\n\n// Payer takes payments."),
          ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", build + "\n\treturn money.Receipt{Amount: amt}, nil")],
         "run exit=1 contains=want.out", {"want.out": "invariant: Balance >= 0\n"})
case("H4", "A contract that calls a function with an effect: K0503 (audit P2, H4).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "var count int\n\nfunc bump() bool { count++; return true }\n\nfunc charge(x int) int\n\trequires bump() && x > 0\n{\n\treturn x\n}\n\nvar _ = charge\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@bump() && x > 0@ K0503")
case("H4b", "Near miss: len, cap, and conversions in a contract.",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func fill(xs []int, n int) int\n\trequires len(xs) <= cap(xs) && money.Money(n) >= 0\n{\n\treturn n\n}\n\nvar _ = fill\n\n// Payer takes payments.")],
     "accept")
case("H5", "Port contracts with named results, two old() clauses, and a parameter named w compile and run (audit P2, H5).",
     [("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(w money.Money) (r money.Receipt, e error)\n            requires w > 0\n            ensures e == nil => r.Amount == w && w == old(w)\n            ensures w + 1 == old(w + 1)"),
      ("api/billing/billing.vulcan", "func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {", "func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {")],
     "run exit=0 stdout=out.txt", {"out.txt": "paid 42\n"})
EMBED = '''// Counter counts down.
type Counter struct{ N int }

// Dec counts down.
func (c *Counter) Dec() { c.N-- }

// Account holds money.
type Account struct {
	Balance int
	C       Counter
}
	invariant Balance >= 0 && C.N >= 0

// Holder embeds an account.
type Holder struct{ Account }

// Payer takes payments.'''
case("M1a", "A write through a promoted field: K0507 (audit P2, M1).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", EMBED),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "var h Holder\n\th.Balance = -7\n\treturn money.Receipt{Amount: amt}, nil")],
     "error api/billing/billing.vulcan:@@h.Balance@ K0507")
case("M1b", "A pointer method called on a field: K0508 (audit P2, M1).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", EMBED),
      ("api/billing/billing.vulcan", "return money.Receipt{Amount: amt}, nil", "var a Account\n\ta.C.Dec()\n\treturn money.Receipt{Amount: amt}, nil")],
     "error api/billing/billing.vulcan:@@a.C.Dec@ K0508")
case("M1c", "A closure that a method returns writes a field: K0507 (audit P2, M1).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", EMBED.replace("// Holder embeds", "// Drainer returns a function that breaks the account.\nfunc (a *Account) Drainer() func() { return func() { a.Balance = -100 } }\n\n// Holder embeds"))],
     "error api/billing/billing.vulcan:@@a.Balance = -100@ K0507")
case("M2", "Near miss: float arguments that keep requires are accepted (audit P2, M2).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func half(f float64) float64\n\trequires f / 2 > 0.4\n{\n\treturn f / 2\n}\n\nvar _ = half(0.9) + half(1.0)\n\n// Payer takes payments.")],
     "accept")
case("M3", "A type error inside a contract is reported at the clause, with the source names (audit P2, M3).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func get(x int) (n int, e error)\n\tensures e == nil => n == \"x\"\n{\n\treturn x, nil\n}\n\nvar _ = get\n\n// Payer takes payments.")],
     'error api/billing/billing.vulcan:@@e == nil => n == "x"@ K0008')
case("L2", "result under * without a guard: K0502 (audit P2, L2).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "func find(k int) (*int, error)\n\tensures *result > 0\n{\n\treturn &k, nil\n}\n\nvar _ = find\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@*result > 0@ K0502")
case("L3", "An invariant against a typed constant that the zero value breaks: K0506 (audit P2, L3).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "// Qty is a count.\ntype Qty int\n\n// MinQty is the smallest line.\nconst MinQty Qty = 1\n\n// Line is an order line.\ntype Line struct{ N Qty }\n\tinvariant N >= MinQty\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@N >= MinQty@ K0506")
case("L4", "A constant call through the receiver that breaks requires: K0501 (audit P2, L4).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "// Account holds money.\ntype Account struct{ Balance int }\n\tinvariant Balance >= 0\n\n// Withdraw takes money out.\nfunc (a *Account) Withdraw(x int)\n\trequires x > 0\n{\n\ta.Balance -= x\n}\n\n// Fix withdraws a constant.\nfunc (a *Account) Fix() { a.Withdraw(-5) }\n\n// Payer takes payments.")],
     "error api/billing/billing.vulcan:@@Withdraw(-5)@ K0501".replace("@@Withdraw(-5)@", "@@a.Withdraw(-5)@"))
case("L5", "A literal that ends at the closing brace of a body with a contract (audit P2, L5).",
     [("api/billing/billing.vulcan", "// Payer takes payments.", "// Account holds money.\ntype Account struct{ Balance int }\n\tinvariant Balance >= 0\n\nfunc open(x int) *Account\n\trequires x >= 0\n{ return &Account{Balance: x}}\n\nvar _ = open\n\n// Payer takes payments.")],
     "accept")

# ---- Cases added after the P3 parity review (experiments/poc/parity-p3.md)
case("V01", "vulcan check runs go vet: a value receiver that copies a mutex is K0706 (parity P3, item 1).",
     [("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"sync"\n\n\t"shop/lib/money"\n)'),
      ("api/billing/billing.vulcan", "// Payer takes payments.\ntype Payer struct{}", "// Payer takes payments.\ntype Payer struct{ mu sync.Mutex }\n\n// Name names the payer.\nfunc (p Payer) Name() string { return \"payer\" }")],
     "error api/billing/billing.vulcan:@@Payer) Name@ K0706")
case("V02", "One constructor wired with two settings: K0607 warns of the second instance (parity P3, item 2).",
     [("system.vulcan", "        Web.Orders:      Orders.NewService\n        Orders.Checkout: Billing.NewPayer",
       "        Web.Orders:      Orders.NewService\n        Orders.Checkout: Billing.NewPayer{ Fee: 1 }\n        Web.Extra:       Billing.NewPayer{ Fee: 2 }"),
      ("api/billing/billing.vulcan", "func NewPayer() *Payer { return &Payer{} }", "// Settings configure a Payer.\ntype Settings struct {\n\tFee int\n}\n\nfunc NewPayer(s Settings) *Payer { return &Payer{} }"),
      ("api/web/component.vulcan", "    needs port Orders {", "    needs port Extra {\n        func Pay(amt money.Money) (money.Receipt, error)\n    }\n\n    needs port Orders {")],
     "accept\nwarn system.vulcan:@2@NewPayer@ K0607")

# ---- Strict mode (spec §5.4d, decision 2): N05 to N12 of conformance.md
def strict(extra=""):
    return ("system.vulcan", '    person Customer "Buys goods"', '    person Customer "Buys goods"\n\n    config { strict: true' + extra + ' }')
FUNC_PORT = [("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(amt money.Money, done func(money.Receipt)) (money.Receipt, error)"),
             ("api/orders/orders.vulcan", "return Checkout.Pay(amt)", "return Checkout.Pay(amt, nil)"),
             ("api/billing/billing.vulcan", "Pay(amt money.Money) (money.Receipt, error) {", "Pay(amt money.Money, done func(money.Receipt)) (money.Receipt, error) {")]
case("N05", "Strict: a port method takes a function: K0123.", [strict()] + FUNC_PORT, "error api/orders/component.vulcan:@@Pay@ K0123")
RC_PORT = [("api/orders/component.vulcan", 'import "shop/lib/money"', 'import (\n    "io"\n    "shop/lib/money"\n)'),
           ("api/orders/component.vulcan", "func Pay(amt money.Money) (money.Receipt, error)", "func Pay(amt money.Money) (money.Receipt, error)\n        func Statement() (io.ReadCloser, error)"),
           ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"io"\n\t"strings"\n\n\t"shop/lib/money"\n)'),
           ("api/billing/billing.vulcan", "// Pay takes a payment.", "// Statement returns the statement.\nfunc (p *Payer) Statement() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader(\"\")), nil }\n\n// Pay takes a payment.")]
case("N06", "Strict: a port method returns io.ReadCloser: K0123.", [strict()] + RC_PORT, "error api/orders/component.vulcan:@@Statement@ K0123")
case("N06b", "Strict: the same with strict_interfaces: [\"io.ReadCloser\"]: accepted.", [strict(', strict_interfaces: ["io.ReadCloser"]')] + RC_PORT, "accept")
case("N07", "Strict: a leaf imports unsafe: K0124.",
     [strict(), ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"unsafe"\n\n\t"shop/lib/money"\n)\n\nvar _ = unsafe.Sizeof(0)')],
     'error api/billing/billing.vulcan:@@"unsafe"@ K0124')
DIAL = [("api/web/web.vulcan", 'import (\n\t"context"\n\t"fmt"\n)', 'import (\n\t"context"\n\t"fmt"\n\t"net/http"\n)'),
        ("api/web/web.vulcan", "func startup(", "func ping() { _, _ = http.Get(\"http://127.0.0.1:1\") }\n\nvar _ = ping\n\nfunc startup(")]
case("N08", "Strict: an entry leaf calls http.Get: K0125.", [strict()] + DIAL, "error api/web/web.vulcan:@@Get@ K0125")
case("N09", "Strict: a library var holds a channel: K0210.",
     [strict(), ("lib/money/money.vulcan", "// Money is an amount in cents.", "// Events carries events.\nvar Events = make(chan int)\n\n// Money is an amount in cents.")],
     "error lib/money/money.vulcan:@@Events =@ K0210")
case("N09b", "Strict: a library var of an error type, and of a type in strict_shared, is accepted.",
     [strict(', strict_shared: ["*regexp.Regexp"]'), ("lib/money/money.vulcan", "package money\n", 'package money\n\nimport (\n\t"errors"\n\t"regexp"\n)\n\n// ErrBad is an error.\nvar ErrBad = errors.New("bad")\n\n// Code matches a code.\nvar Code = regexp.MustCompile(`^[A-Z]+$`)\n')],
     "accept")
SETENV = [("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"os"\n\n\t"shop/lib/money"\n)\n\nfunc mode() { _ = os.Setenv("MODE", "x") }\n\nvar _ = mode')]
case("N10", "Strict: a leaf calls os.Setenv: K0211.", [strict()] + SETENV, "error api/billing/billing.vulcan:@@Setenv@ K0211")
case("N11", "Strict: N05, N08, and N10 in leaves listed in strict_exempt: accepted.",
     [strict(', strict_exempt: ["Orders", "Web", "Billing"]')] + FUNC_PORT + DIAL + [
      ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"os"\n\n\t"shop/lib/money"\n)\n\nfunc mode() { _ = os.Setenv("MODE", "x") }\n\nvar _ = mode')],
     "accept")
case("N12", "Ordinary mode: N05, N08, and N10 are accepted.", FUNC_PORT + DIAL + SETENV, "accept")
case("N13", "Strict: the base system, a serving entry leaf, and a dialing gateway are accepted.",
     [strict(), ("system.vulcan", '    person Customer "Buys goods"', '    person Customer "Buys goods"\n    external Bank "Cards"'),
      ("api/billing/component.vulcan", "component Billing {", "component Billing {\n    gateway for Bank"),
      ("api/billing/billing.vulcan", 'import "shop/lib/money"', 'import (\n\t"net/http"\n\n\t"shop/lib/money"\n)\n\nfunc ping() { _, _ = http.Get("http://127.0.0.1:1") }\n\nvar _ = ping'),
      ("api/web/web.vulcan", 'import (\n\t"context"\n\t"fmt"\n)', 'import (\n\t"context"\n\t"fmt"\n\t"net"\n)'),
      ("api/web/web.vulcan", "func startup(", "func listen() { l, err := net.Listen(\"tcp\", \"127.0.0.1:0\"); if err == nil { l.Close() } }\n\nvar _ = listen\n\nfunc startup(")],
     "accept")

os.makedirs(OUT, exist_ok=True)
for f in os.listdir(OUT):
    if f.endswith(".txtar"):
        os.remove(os.path.join(OUT, f))
for id, doc, files, want in CASES:
    with open(os.path.join(OUT, id + ".txtar"), "w", encoding="utf-8", newline="\n") as fh:
        fh.write(doc + "\n")
        fh.write("-- want.txt --\n" + resolve(files, want))
        for path in sorted(files):
            body = files[path]
            if not body.endswith("\n"):
                body += "\n"
            fh.write(f"-- {path} --\n{body}")
print(len(CASES), "cases")
