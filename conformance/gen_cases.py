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
case("S01", "A method with three contract lines between its signature and a { on its own line: K0009 at the first requires before contracts are supported.",
     [("api/billing/billing.vulcan", "func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {",
       "func (p *Payer) Pay(amt money.Money) (money.Receipt, error)\n\trequires amt > 0\n\trequires amt < 1_000_000\n\tensures err == nil => result.Amount == amt\n{")],
     "error api/billing/billing.vulcan:@@requires@ K0009")

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
     "error api/billing/billing.vulcan:@@requires@ K0009\nerror api/billing/billing.vulcan:@@amt; _@ K0008")

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
     "\n".join(f"cmd exit=2 vulcan {c}" for c in ("fmt", "explain", "doc", "diagram", "archdiff", "fake", "trace")))

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
