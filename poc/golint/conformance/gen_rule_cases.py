#!/usr/bin/env python3
"""Writes the conformance cases of plan-r3, step 2, for both arms, from one table.

Each case has the same rules file and the same code in V (conformance/<id>.txtar) and in L
(poc/golint/conformance/testdata/<id>.txtar). A case differs between the arms only where the arms
differ: the file that declares a port or a wire. TestRuleCasesAgree checks that each case lists the
same codes in both. Run it from the root of the repository.
"""
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[3]
V_DIR = ROOT / "conformance"
L_DIR = ROOT / "poc/golint/conformance/testdata"

MONEY = """package money

// Money is an amount in cents.
type Money int64

// Receipt records a payment.
type Receipt struct {
	Amount Money
}

// The states of an order.
const (
	Pending   = "pending"
	Paid      = "paid"
	Shipped   = "shipped"
	Cancelled = "cancelled"
)

// Order is an order.
type Order struct {
	ID     string
	Status string
}
"""

STORE = """package store

// Store keeps things.
type Store struct{}

// Open returns a Store.
func Open() *Store { return &Store{} }

// Save keeps one thing.
func (s *Store) Save(id string) error { return nil }
"""

V_BASE = {
    "system.vulcan": """// Shop is the online store.
system Shop {
    person Customer "Buys goods"
    external Database "The file"

    container Api {
        Web, Orders, Billing, Store

        Web.Orders:      Orders.NewService
        Orders.Checkout: Billing.NewPayer
        Billing.Ledger:  Store.Open
    }
}
""",
    "lib/money/library.vulcan": "// money holds amounts.\nlibrary money {\n}\n",
    "lib/money/money.vulcan": MONEY,
    "api/store/component.vulcan": "component Store {\n    gateway for Database\n}\n",
    "api/store/store.vulcan": STORE,
    "api/billing/component.vulcan": """component Billing {
    needs port Ledger {
        func Save(id string) error
    }
}
""",
    "api/billing/billing.vulcan": """package billing

import "shop/lib/money"

// Payer takes payments.
type Payer struct{}

// NewPayer returns a Payer.
func NewPayer() *Payer { return &Payer{} }

// Pay takes a payment.
func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {
	return money.Receipt{Amount: amt}, Ledger.Save("x")
}
""",
    "api/orders/component.vulcan": """import "shop/lib/money"

component Orders {
    needs port Checkout {
        func Pay(amt money.Money) (money.Receipt, error)
    }
}
""",
    "api/orders/orders.vulcan": """package orders

import "shop/lib/money"

// Service places orders.
type Service struct{}

// NewService returns a Service.
func NewService() *Service { return &Service{} }

// Place takes payment for an order.
func (s *Service) Place(amt money.Money) (money.Receipt, error) {
	return Checkout.Pay(amt)
}
""",
    "api/web/component.vulcan": """import "shop/lib/money"

component Web {
    entry for Customer

    needs port Orders {
        func Place(amt money.Money) (money.Receipt, error)
    }
}
""",
    "api/web/web.vulcan": """package web

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
""",
}

L_BASE = {
    "arch/arch.go": """package arch

// The persons and the external systems.
const (
	Customer Person   = "Customer" // buys goods
	Database External = "Database" // the file
)

// Manifest is the architecture of the shop.
var Manifest = System{
	Parts: []Part{
		Main("cmd/shop"),
		Entry("api/web", Customer),
		Leaf("api/orders"),
		Leaf("api/billing"),
		Gateway("api/store", Database),
		Library("lib/money"),
		Library("lib/contract"),
	},
}
""",
    "cmd/shop/main.go": """// Command shop runs the shop.
package main

import (
	"context"
	"os"

	"shop/api/billing"
	"shop/api/orders"
	"shop/api/store"
	"shop/api/web"
)

func main() {
	keeper := store.Open()
	payer := billing.NewPayer(keeper)
	service := orders.NewService(payer)
	api := web.New(service)
	if err := api.Start(context.Background()); err != nil {
		os.Exit(1)
	}
}
""",
    "lib/money/money.go": MONEY,
    "api/store/store.go": STORE,
    "api/billing/ports.go": """package billing

// Ledger keeps payments.
type Ledger interface {
	Save(id string) error
}
""",
    "api/billing/billing.go": """package billing

import "shop/lib/money"

// Payer takes payments.
type Payer struct {
	ledger Ledger
}

// NewPayer returns a Payer.
func NewPayer(ledger Ledger) *Payer { return &Payer{ledger: ledger} }

// Pay takes a payment.
func (p *Payer) Pay(amt money.Money) (money.Receipt, error) {
	return money.Receipt{Amount: amt}, p.ledger.Save("x")
}
""",
}

HEAD = "origin test\nhash sha256:0\n\n"


def rule(rid, kind, level="error", **ps):
    out = f"rule {rid}\n  kind {kind}\n  level {level}\n"
    for k, v in ps.items():
        out += f"  {k.rstrip('_')} {v}\n"
    return out + "  reason A reason.\n  source A source.\n"


def rules(*blocks):
    return HEAD + "\n".join(blocks)


def code(path, text):
    """One code file, the same text in both arms."""
    return {"v": {path + ".vulcan": text}, "l": {path + ".go": text}}


def join(*parts):
    out = {"v": {}, "l": {}}
    for p in parts:
        out["v"].update(p.get("v", {}))
        out["l"].update(p.get("l", {}))
    return out


def extra(pkg, body, imports=""):
    """A file of unexported functions in a part, the same in both arms."""
    imp = ""
    if imports:
        imp = "import (\n" + "".join(f'\t"{i}"\n' for i in imports.split()) + ")\n\n"
    return code(f"api/{pkg}/extra", f"package {pkg}\n\n{imp}{body}")


def at(path, snippet, digits, offset=0):
    return (path, snippet, digits, offset)


def e(path, snippet, digits, offset=0):
    """A finding in a code file, at the same place in both arms."""
    return {"v": [(path + ".vulcan", snippet, digits, offset)], "l": [(path + ".go", snippet, digits, offset)]}


def both(*wants):
    out = {"v": [], "l": []}
    for w in wants:
        out["v"] += w["v"]
        out["l"] += w["l"]
    return out


NO = {"v": [], "l": []}
GETENV = extra("orders", 'func readIt() string { return os.Getenv("X") }\n', "os")
ORDER = extra("orders", "%s", "shop/lib/money")
TABLE = dict(type="lib/money.Order", field="Status", initial="pending", allow="pending->paid paid->shipped")
STATE = rule("S", "transitions", **TABLE)
ANYSTATE = rule("S", "transitions", type="lib/money.Order", field="Status", initial="pending", allow="pending->paid *->cancelled")
LOADED = rule("S", "transitions", load="leaf", **TABLE)


def order(body):
    return extra("orders", body, "shop/lib/money")


LOCK = """wire api/billing -> api/store
wire api/orders -> api/billing
wire api/web -> api/orders
import api/billing -> lib/money
import api/orders -> lib/money
import api/web -> lib/money
"""
LOCK_NO_WIRE = LOCK.replace("wire api/billing -> api/store\n", "")
LOCK_NO_IMPORT = LOCK.replace("import api/billing -> lib/money\n", "")
LOCKED = rule("L", "depends", map="locked")
WIRE_V = ("system.vulcan", "Billing.Ledger:", "0806", 0)
WIRE_L = ("cmd/shop/main.go", "(keeper)", "0806", 1)
WIRE_V4 = ("system.vulcan", "Billing.Ledger:", "0804", 0)
WIRE_L4 = ("cmd/shop/main.go", "(keeper)", "0804", 1)
PORT_HTTP = {
    "v": {"api/orders/component.vulcan": """import (
	"net/http"

	"shop/lib/money"
)

component Orders {
    needs port Checkout {
        func Pay(amt money.Money) (money.Receipt, error)
        func Raw(r *http.Request) error
    }
}
""", "api/billing/raw.vulcan": """package billing

import "net/http"

// Raw reads a request.
func (p *Payer) Raw(r *http.Request) error { return nil }
"""},
    "l": {"api/orders/ports.go": """package orders

import (
	"net/http"

	"shop/lib/money"
)

// Checkout takes payment for an order.
type Checkout interface {
	Pay(amt money.Money) (money.Receipt, error)
	Raw(r *http.Request) error
}
""", "api/billing/raw.go": """package billing

import "net/http"

// Raw reads a request.
func (p *Payer) Raw(r *http.Request) error { return nil }
"""},
}
TWICE = len(rule("B", "forbid-call", funcs="os.Exit")) + 1  # to the second rule
ASM = 'TEXT ·holeB(SB),4,$0-8\n\tRET\n'

CASES = [
    # forbid-call
    ("R01", "forbid-call rejects a call", rules(rule("F", "forbid-call", funcs="os.Getenv", **{"in": "leaf"})), GETENV, e("api/orders/extra", "Getenv", "0802")),
    ("R02", "forbid-call: except lets a part call it", rules(rule("F", "forbid-call", funcs="os.Getenv", except_="api/orders")), GETENV, NO),
    ("R03", "forbid-call: a rule that is off does not fire", rules(rule("F", "forbid-call", level="off", funcs="os.Getenv")), GETENV, NO),
    ("R04", "A deleted rule stops firing: the same code as R01, and the file has no rule", rules(), GETENV, NO),
    ("R05", "No rules file: no rule fires", None, GETENV, NO),
    ("R06", "forbid-call: in names another kind", rules(rule("F", "forbid-call", funcs="os.Getenv", **{"in": "gateway"})), GETENV, NO),
    ("R07", "forbid-call: files test reads only test files", rules(rule("F", "forbid-call", funcs="time.Sleep", files="test")),
     join(extra("orders", "func waitIt() { time.Sleep(1) }\n", "time"),
          code("api/orders/slow_test", 'package orders\n\nimport (\n\t"testing"\n\t"time"\n)\n\nfunc TestSlow(t *testing.T) { time.Sleep(1) }\n')),
     e("api/orders/slow_test", "Sleep", "0802")),
    ("R08", "forbid-call names a package-level value", rules(rule("F", "forbid-call", funcs="net/http.DefaultServeMux")),
     extra("web", "func muxIt() any { return http.DefaultServeMux }\n", "net/http"), e("api/web/extra", "DefaultServeMux", "0802")),
    ("R09", "forbid-call: a name that ends in * is a prefix", rules(rule("F", "forbid-call", funcs="os.Get*")), GETENV, e("api/orders/extra", "Getenv", "0802")),
    # only-in
    ("R10", "only-in clock", rules(rule("O", "only-in", effect="clock", **{"in": "gateway"})), extra("orders", "func nowIt() any { return time.Now() }\n", "time"), e("api/orders/extra", "Now", "0803")),
    ("R11", "only-in randomness", rules(rule("O", "only-in", effect="randomness", **{"in": "gateway"})), extra("orders", "func randIt() int { return rand.IntN(3) }\n", "math/rand/v2"), e("api/orders/extra", "IntN", "0803")),
    ("R12", "only-in processes", rules(rule("O", "only-in", effect="processes", **{"in": "gateway"})), extra("orders", "func findIt() { _, _ = os.FindProcess(1) }\n", "os"), e("api/orders/extra", "FindProcess", "0803")),
    ("R13", "only-in files", rules(rule("O", "only-in", effect="files", **{"in": "gateway"})), extra("orders", 'func statIt() { _, _ = os.Stat("x") }\n', "os"), e("api/orders/extra", "Stat", "0803")),
    ("R14", "only-in database", rules(rule("O", "only-in", effect="database", **{"in": "gateway"})), extra("orders", "func pingIt(db *sql.DB) error { return db.Ping() }\n", "database/sql"), e("api/orders/extra", "Ping", "0803")),
    ("R15", "only-in network, with the model's check too", rules(rule("O", "only-in", effect="network", **{"in": "gateway entry"})), extra("orders", 'func dialIt() { _, _ = http.Get("x") }\n', "net/http"),
     both(e("api/orders/extra", "Get", "0205"), e("api/orders/extra", "Get", "0803"))),
    ("R16", "only-in: a part of the kind may", rules(rule("O", "only-in", effect="clock", **{"in": "gateway"})), extra("store", "func nowIt() any { return time.Now() }\n", "time"), NO),
    ("R17", "only-in dial rejects an entry leaf", rules(rule("O", "only-in", effect="dial", **{"in": "gateway"})), extra("web", 'func dialIt() { _, _ = http.Get("x") }\n', "net/http"), e("api/web/extra", "Get", "0803")),
    ("R18", "only-in: a literal of a type that dials, in a part that the rule allows", rules(rule("O", "only-in", effect="network", **{"in": "gateway entry"})), extra("web", "func clientIt() any { return &http.Client{} }\n", "net/http"), NO),
    ("R19", "only-in: a literal of a type that dials, in a part that the rule does not allow", rules(rule("O", "only-in", effect="dial", **{"in": "entry"})), extra("store", "func clientIt() any { return &http.Client{} }\n", "net/http"), e("api/store/extra", "http.Client{}", "0803")),
    # depends
    ("R20", "depends rejects an import of a package", rules(rule("D", "depends", **{"from": "leaf"}, packages="net/url")), extra("orders", "func parseIt() { _, _ = url.Parse(\"x\") }\n", "net/url"), e("api/orders/extra", '"net/url"', "0804")),
    ("R21", "depends reads the imports of a port's declaration", rules(rule("D", "depends", **{"from": "api/orders"}, packages="net/http")), PORT_HTTP,
     {"v": [("api/orders/component.vulcan", '"net/http"', "0804", 0)], "l": [("api/orders/ports.go", '"net/http"', "0804", 0)]}),
    ("R22", "depends rejects an import of a kind, with the model's check too", rules(rule("D", "depends", **{"from": "leaf gateway entry"}, deny="leaf", via="import")),
     extra("orders", "func payerIt() any { return billing.Payer{} }\n", "shop/api/billing"),
     both(e("api/orders/extra", '"shop/api/billing"', "0804"), e("api/orders/extra", '"shop/api/billing"', "0101"))),
    ("R23", "depends rejects an import of one part", rules(rule("D", "depends", **{"from": "api/web"}, deny="lib/money")), NO,
     {"v": [("api/web/component.vulcan", '"shop/lib/money"', "0804", 0)], "l": [("api/web/ports.go", '"shop/lib/money"', "0804", 0)]}),
    ("R24", "depends rejects a wire by kind", rules(rule("D", "depends", **{"from": "leaf"}, deny="gateway", via="wire")), NO, {"v": [WIRE_V4], "l": [WIRE_L4]}),
    ("R25", "depends: via import does not read wires", rules(rule("D", "depends", **{"from": "leaf"}, deny="gateway", via="import")), NO, NO),
    ("R26", "depends: allow rejects every other wire", rules(rule("D", "depends", **{"from": "api/billing"}, allow="lib/money", via="wire")), NO, {"v": [WIRE_V4], "l": [WIRE_L4]}),
    ("R27", "depends: except leaves out a part", rules(rule("D", "depends", **{"from": "leaf"}, except_="api/billing", deny="gateway", via="wire")), NO, NO),
    ("R28", "The locked map holds each dependency", rules(LOCKED), {"lock": LOCK}, NO),
    ("R29", "The locked map: a new wire fails", rules(LOCKED), {"lock": LOCK_NO_WIRE}, {"v": [WIRE_V], "l": [WIRE_L]}),
    ("R30", "The locked map: a new import of a library fails", rules(LOCKED), {"lock": LOCK_NO_IMPORT}, e("api/billing/billing", '"shop/lib/money"', "0806")),
    ("R31", "The locked map is off: no lock is needed", rules(rule("L", "depends", level="off", map="locked")), NO, NO),
    ("R32", "The locked map is on, and there is no lock", rules(LOCKED), NO, {"v": [("rules.vulcan", "rule L", "0801", 0)], "l": [("rules.vulcan", "rule L", "0801", 0)]}),
    # transitions
    ("R40", "transitions: an early return guards a write", rules(STATE), order('func payIt(o *money.Order) {\n\tif o.Status != money.Pending {\n\t\treturn\n\t}\n\to.Status = money.Paid\n}\n'), NO),
    ("R41", "transitions: the table does not allow the change", rules(STATE), order('func shipIt(o *money.Order) {\n\tif o.Status != money.Pending {\n\t\treturn\n\t}\n\to.Status = money.Shipped\n}\n'), e("api/orders/extra", "o.Status = money.Shipped", "0805")),
    ("R42", "transitions: no guard, so the state is not known", rules(STATE), order("func payIt(o *money.Order) {\n\to.Status = money.Paid\n}\n"), e("api/orders/extra", "o.Status = money.Paid", "0805")),
    ("R43", "transitions: not a state of the table", rules(STATE), order('func oddIt(o *money.Order) {\n\tif o.Status == money.Pending {\n\t\to.Status = "odd"\n\t}\n}\n'), e("api/orders/extra", 'o.Status = "odd"', "0805")),
    ("R44", "transitions: not a constant", rules(STATE), order("func setIt(o *money.Order, s string) {\n\to.Status = s\n}\n"), e("api/orders/extra", "o.Status = s", "0805")),
    ("R45", "transitions: load lets a part write a value from a store", rules(LOADED), order("func setIt(o *money.Order, s string) {\n\to.Status = s\n}\n"), NO),
    ("R46", "transitions: a value starts in an initial state", rules(STATE), order("func newIt() money.Order {\n\treturn money.Order{ID: \"a\", Status: money.Paid}\n}\n"), e("api/orders/extra", "money.Paid}", "0805")),
    ("R47", "transitions: a literal, then a write that follows from it", rules(STATE), order('func newIt() money.Order {\n\to := money.Order{ID: "a", Status: money.Pending}\n\to.Status = money.Paid\n\treturn o\n}\n'), NO),
    ("R48", "transitions: a switch guards each write", rules(STATE), order("func stepIt(o *money.Order) {\n\tswitch o.Status {\n\tcase money.Pending:\n\t\to.Status = money.Paid\n\tcase money.Paid:\n\t\to.Status = money.Shipped\n\t}\n}\n"), NO),
    ("R49", "transitions: a case of a switch breaks the table", rules(STATE), order("func stepIt(o *money.Order) {\n\tswitch o.Status {\n\tcase money.Paid:\n\t\to.Status = money.Pending\n\t}\n}\n"), e("api/orders/extra", "o.Status = money.Pending", "0805")),
    ("R50", "transitions: * allows a change from any state", rules(ANYSTATE), order("func stopIt(o *money.Order) {\n\to.Status = money.Cancelled\n}\n"), NO),
    ("R51", "transitions: a call can change a value that it gets by its address", rules(STATE), order("func other(o *money.Order) {}\n\nfunc payIt(o money.Order) money.Order {\n\tif o.Status == money.Pending {\n\t\tother(&o)\n\t\to.Status = money.Paid\n\t}\n\treturn o\n}\n"), e("api/orders/extra", "o.Status = money.Paid", "0805")),
    ("R52", "transitions: the address of the field", rules(STATE), order("func addrIt(o *money.Order) *string {\n\treturn &o.Status\n}\n"), e("api/orders/extra", "&o.Status", "0805")),
    ("R53", "transitions: a guard with || gives both states", rules(rule("S", "transitions", type="lib/money.Order", field="Status", allow="pending->cancelled paid->cancelled shipped->paid")), order("func stopIt(o *money.Order) {\n\tif o.Status == money.Pending || o.Status == money.Paid {\n\t\to.Status = money.Cancelled\n\t}\n}\n"), NO),
    ("R54", "transitions: a write that one state of a guard reaches and that the other already holds", rules(STATE), order("func stopIt(o *money.Order) {\n\tif o.Status == money.Pending || o.Status == money.Paid {\n\t\to.Status = money.Paid\n\t}\n}\n"), NO),
    ("R59", "transitions: a guard with ||, and a change that the table does not allow from one of the two", rules(STATE), order("func stopIt(o *money.Order) {\n\tif o.Status == money.Pending || o.Status == money.Paid {\n\t\to.Status = money.Shipped\n\t}\n}\n"), e("api/orders/extra", "o.Status = money.Shipped", "0805")),
    ("R55", "transitions: a write in a loop does not keep a guard from outside it", rules(STATE), order("func loopIt(o *money.Order) {\n\tif o.Status == money.Pending {\n\t\tfor i := 0; i < 2; i++ {\n\t\t\to.Status = money.Paid\n\t\t}\n\t}\n}\n"), e("api/orders/extra", "o.Status = money.Paid", "0805")),
    ("R56", "transitions: a test builds a value in any state", rules(STATE), code("api/orders/state_test", 'package orders\n\nimport (\n\t"testing"\n\n\t"shop/lib/money"\n)\n\nfunc TestState(t *testing.T) {\n\to := money.Order{Status: money.Paid}\n\to.Status = money.Shipped\n\t_ = o\n}\n'), NO),
    ("R57", "transitions: a parallel assignment is a write", rules(STATE), order("func payIt(o *money.Order) {\n\tif o.Status != money.Pending {\n\t\treturn\n\t}\n\to.ID, o.Status = \"a\", money.Shipped\n}\n"), {"v": [("api/orders/extra.vulcan", "o.ID, o.Status", "0805", 6)], "l": [("api/orders/extra.go", "o.ID, o.Status", "0805", 6)]}),
    ("R58", "transitions: a guard through a method is no fact", rules(STATE), order("func isNew(o *money.Order) bool { return o.Status == money.Pending }\n\nfunc payIt(o *money.Order) {\n\tif isNew(o) {\n\t\to.Status = money.Paid\n\t}\n}\n"), e("api/orders/extra", "o.Status = money.Paid", "0805")),
    # the rules file
    ("R60", "The rules file: an unknown kind", rules(rule("B", "forbid-everything")), NO, {"v": [("rules.vulcan", "rule B", "0801", 0)], "l": [("rules.vulcan", "rule B", "0801", 0)]}),
    ("R61", "The rules file: no reason", HEAD + "rule B\n  kind forbid-call\n  level error\n  funcs os.Exit\n  source A source.\n", NO, {"v": [("rules.vulcan", "rule B", "0801", 0)], "l": [("rules.vulcan", "rule B", "0801", 0)]}),
    ("R62", "The rules file: an id twice", rules(rule("B", "forbid-call", funcs="os.Exit"), rule("B", "forbid-call", funcs="os.Exit")), NO, {"v": [("rules.vulcan", "rule B", "0801", TWICE)], "l": [("rules.vulcan", "rule B", "0801", TWICE)]}),
    ("R63", "The rules file: a directory that is not a part", rules(rule("B", "forbid-call", funcs="os.Exit", **{"in": "api/nowhere"})), NO, {"v": [("rules.vulcan", "rule B", "0801", 0)], "l": [("rules.vulcan", "rule B", "0801", 0)]}),
    ("R64", "The rules file: an unknown effect", rules(rule("B", "only-in", effect="weather", **{"in": "leaf"})), NO, {"v": [("rules.vulcan", "rule B", "0801", 0)], "l": [("rules.vulcan", "rule B", "0801", 0)]}),
    ("R65", "The rules file: no origin", "rule B\n  kind forbid-call\n  level off\n  funcs os.Exit\n  reason A reason.\n  source A source.\n", NO, {"v": [("rules.vulcan", "rule B", "0801", 0)], "l": [("rules.vulcan", "rule B", "0801", 0)]}),
    # holes
    ("H6a", "Hole H6: a function with no body, and an assembly file, in a leaf",  None,
     join(code("api/orders/hole", "package orders\n\nfunc holeB() int\n"), {"v": {"api/orders/h_b_amd64.s": ASM}, "l": {"api/orders/h_b_amd64.s": ASM}}),
     {"v": [("api/orders/hole.vulcan", "holeB", "0126", 0), ("api/orders/h_b_amd64.s", "TEXT", "0126", 0)], "l": [("api/orders/hole.go", "holeB", "0126", 0), ("api/orders/h_b_amd64.s", "TEXT", "0126", 0)]}),
    ("H6b", "Hole H6: in a library", None,
     {"v": {"lib/money/hole.vulcan": "package money\n\nfunc holeB() int\n", "lib/money/h_b_amd64.s": ASM}, "l": {"lib/money/hole.go": "package money\n\nfunc holeB() int\n", "lib/money/h_b_amd64.s": ASM}},
     {"v": [("lib/money/hole.vulcan", "holeB", "0126", 0), ("lib/money/h_b_amd64.s", "TEXT", "0126", 0)], "l": [("lib/money/hole.go", "holeB", "0126", 0), ("lib/money/h_b_amd64.s", "TEXT", "0126", 0)]}),
    ("H6c", "Hole H6: an assembly file for another system", None,
     {"v": {"api/orders/h_b_plan9.s": ASM}, "l": {"api/orders/h_b_plan9.s": ASM}},
     {"v": [("api/orders/h_b_plan9.s", "TEXT", "0126", 0)], "l": [("api/orders/h_b_plan9.s", "TEXT", "0126", 0)]}),
    ("H7a", "Hole H7: net/textproto.Dial in a plain leaf", None, extra("orders", 'func dialIt() { _, _ = textproto.Dial("tcp", "x") }\n', "net/textproto"), e("api/orders/extra", "Dial", "0205")),
    ("H7b", "Hole H7: a control, http.Get in the same leaf", None, extra("orders", 'func dialIt() { _, _ = http.Get("x") }\n', "net/http"), e("api/orders/extra", "Get", "0205")),
]


def pos(text, snippet, offset):
    i = text.index(snippet) + offset
    line = text.count("\n", 0, i) + 1
    col = i - (text.rfind("\n", 0, i) + 1) + 1
    return line, col


def archive(desc, files, wants, prefix, base):
    all_files = dict(base)
    all_files.update(files)
    lines = []
    for path, snippet, digits, offset in sorted(wants):
        text = all_files[path]
        line, col = pos(text, snippet, offset)
        lines.append(f"error {path}:{line}:{col} {prefix}{digits}")
    want = "\n".join(lines) + "\n" if lines else "accept\n"
    out = desc + "\n-- want.txt --\n" + want
    for name in sorted(files):
        out += f"-- {name} --\n{files[name]}"
        if not files[name].endswith("\n"):
            out += "\n"
    return out


def main():
    L_ALL = dict(L_BASE)
    for id_, desc, rules_text, change, wants in CASES:
        v_files, l_files = {}, {}
        if rules_text is not None:
            v_files["rules.vulcan"] = rules_text
            l_files["rules.vulcan"] = rules_text
        if isinstance(change, dict) and "lock" in change:
            v_files["rules.lock"] = change["lock"]
            l_files["rules.lock"] = change["lock"]
        elif change:
            v_files.update(change.get("v", {}))
            l_files.update(change.get("l", {}))
        vf = dict(V_BASE)
        vf.update(v_files)
        # Each V archive is a whole project; each L archive overlays base.txtar.
        lf = dict(L_ALL)
        lf.update(l_files)
        (V_DIR / f"{id_}.txtar").write_text(archive(desc, vf, wants["v"], "K", vf), encoding="utf-8", newline="\n")
        (L_DIR / f"{id_}.txtar").write_text(archive(desc, lf, wants["l"], "VL", {**L_FROM_BASE, **lf}), encoding="utf-8", newline="\n")


# The files of base.txtar that a case can name in want.txt.
L_FROM_BASE = {
    "api/orders/orders.go": "",
    "api/orders/ports.go": """package orders

import "shop/lib/money"

// Checkout takes payment for an order.
type Checkout interface {
	Pay(amt money.Money) (money.Receipt, error)
}
""",
    "api/web/ports.go": """package web

import "shop/lib/money"

// Orders places orders.
type Orders interface {
	Place(amt money.Money) (money.Receipt, error)
}
""",
}


if __name__ == "__main__":
    main()
