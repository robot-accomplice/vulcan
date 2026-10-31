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
    "lib/money/money.go": "// Package money holds amounts.\n" + MONEY,
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
        out += f"  {k.rstrip('_').replace('_', '-')} {v}\n"
    return out + "  reason A reason.\n  source A source.\n"


def rules(*blocks):
    return HEAD + "\n".join(blocks)


def rawcode(path, text):
    """One code file, the same text in both arms, with no package line of its own to add."""
    return {"v": {path + ".vulcan": text}, "l": {path + ".go": text}}


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



# The shape kinds of plan-r3, amendment 6. A digit string that starts with w is a warning.
def same(path, snippet, digits, offset=0):
    return e(path, snippet, digits, offset)


def rawfile(path, text):
    return {"v": {path: text}, "l": {path: text}}


CTX_FIELD = extra("orders", "type holder struct {\n\tctx context.Context\n}\n\nvar _ = holder{}\n", "context")
SETTINGS = extra("orders", "// Settings say how.\ntype Settings struct {\n\tApiKey string\n}\n\nfunc useIt(s Settings) {}\n")
SETTINGS_TAGGED = extra("orders", '// Settings say how.\ntype Settings struct {\n\tApiKey string `vulcan:"secret"`\n}\n\nfunc useIt(s Settings) {}\n')
EMBED = code("lib/money/extra", "package money\n\n// Wrap holds an order.\ntype Wrap struct {\n\tOrder\n}\n")
CTX_LAST = extra("orders", "func doIt(x int, ctx context.Context) {}\n", "context")
CTX_NAME = extra("orders", "func doIt(c context.Context) {}\n", "context")
FLOAT = extra("orders", "func priceIt() float64 {\n\tamount := 1.5\n\treturn amount\n}\n")
UNDOC = extra("orders", "func Exported() {}\n")
PORT_NODOC = {"l": {"api/orders/ports.go": 'package orders\n\nimport "shop/lib/money"\n\ntype Checkout interface {\n\tPay(amt money.Money) (money.Receipt, error)\n}\n'}}
PORT_DOC = {"v": {"api/orders/component.vulcan": 'import "shop/lib/money"\n\ncomponent Orders {\n    // Checkout takes payment.\n    needs port Checkout {\n        // Pay takes a payment.\n        func Pay(amt money.Money) (money.Receipt, error)\n    }\n}\n'},
            "l": {"api/orders/ports.go": 'package orders\n\nimport "shop/lib/money"\n\n// Checkout takes payment.\ntype Checkout interface {\n\t// Pay takes a payment.\n\tPay(amt money.Money) (money.Receipt, error)\n}\n'}}
CONFIG_BARE = {"v": {"system.vulcan": V_BASE["system.vulcan"].replace('    container Api', '    config { strict_shared: ["sync.Mutex"] }\n\n    container Api')},
               "l": {"arch/config.go": 'package arch\n\n// Config is the check configuration.\nvar Config = Settings{\n\tStrictShared: []string{\n\t\t"sync.Mutex",\n\t},\n}\n'}}
CONFIG_COMMENT = {"v": {"system.vulcan": V_BASE["system.vulcan"].replace('    container Api', '    config {\n        // a lock is safe to share\n        strict_shared: ["sync.Mutex"]\n    }\n\n    container Api')},
                  "l": {"arch/config.go": 'package arch\n\n// Config is the check configuration.\nvar Config = Settings{\n\tStrictShared: []string{\n\t\t"sync.Mutex", // a lock is safe to share\n\t},\n}\n'}}
SQL_BUILT = extra("store", 'func runIt(db *sql.DB, t string) {\n\t_, _ = db.Exec("DELETE FROM " + t)\n}\n', "database/sql")
SQL_COMMENT = extra("store", 'func runIt(db *sql.DB, t string) {\n\t// t comes from a fixed list.\n\t_, _ = db.Exec("DELETE FROM " + t)\n}\n', "database/sql")
SQL_BEGIN = extra("store", 'func runIt(db *sql.DB) {\n\t_, _ = db.Exec("BEGIN")\n}\n', "database/sql")
ERR_TEXT = extra("orders", 'func badIt() error { return errors.New("Bad.") }\n', "errors")
NIL_CTX = extra("orders", "func takeIt(ctx context.Context) {}\n\nfunc callIt() { takeIt(nil) }\n", "context")
CTX_KEY = extra("orders", 'func valIt(ctx context.Context) context.Context {\n\treturn context.WithValue(ctx, "k", 1)\n}\n', "context")
UNTAGGED = extra("orders", "type wire struct {\n\tName string\n}\n\nfunc sendIt() { _, _ = json.Marshal(wire{}) }\n", "encoding/json")
TAGGED = extra("orders", 'type wire struct {\n\tName string `json:"name"`\n}\n\nfunc sendIt() { _, _ = json.Marshal(wire{}) }\n', "encoding/json")
SERVER = extra("web", 'func serveIt() *http.Server {\n\treturn &http.Server{Addr: "x", ReadTimeout: 1}\n}\n', "net/http")
SERVER_OK = extra("web", 'func serveIt() *http.Server {\n\treturn &http.Server{Addr: "x", ReadTimeout: 1, WriteTimeout: 1}\n}\n', "net/http")
SERVER_NEW = extra("web", "func serveIt() *http.Server {\n\treturn new(http.Server)\n}\n", "net/http")
TLS_SKIP = extra("store", "func tlsIt() *tls.Config {\n\treturn &tls.Config{InsecureSkipVerify: true}\n}\n", "crypto/tls")
TLS_OLD = extra("store", "func tlsIt() *tls.Config {\n\treturn &tls.Config{MinVersion: tls.VersionTLS10}\n}\n", "crypto/tls")
TLS_ASSIGN = extra("store", "func tlsIt(c *tls.Config) {\n\tc.InsecureSkipVerify = true\n}\n", "crypto/tls")
COOKIE = extra("web", 'func cookieIt() *http.Cookie {\n\treturn &http.Cookie{Secure: true, SameSite: http.SameSiteNoneMode}\n}\n', "net/http")
COOKIE_OK = extra("web", 'func cookieIt() *http.Cookie {\n\treturn &http.Cookie{Secure: true, SameSite: http.SameSiteLaxMode}\n}\n', "net/http")
SECRET_TEXT = extra("orders", 'const keyIt = "sk_live_abc"\n\nvar _ = keyIt\n')
PASSWORD = extra("orders", 'func passIt() string {\n\tpassword := "hunter2"\n\treturn password\n}\n')
DROPPED = extra("orders", 'func failIt() error { return nil }\n\nfunc callIt() { failIt() }\n')
DEFERRED = extra("orders", 'func closeIt(tx *sql.Tx) { defer tx.Rollback() }\n', "database/sql")
DISCARD = extra("orders", 'func failIt() error { return nil }\n\nfunc callIt() {\n\t_ = failIt()\n}\n')
DISCARD_OK = extra("orders", 'func failIt() error { return nil }\n\nfunc callIt() {\n\t// nothing can be done here\n\t_ = failIt()\n}\n')
COMPARE = extra("orders", 'func sameIt(a, b error) bool { return a == b }\n')
COMPARE_OK = extra("orders", 'func sameIt(a, b error) bool { return errors.Is(a, b) }\n', "errors")
TEXT_COMPARE = extra("orders", 'func sameIt(a error) bool { return a.Error() == "x" }\n')
ASSERT = extra("orders", 'func castIt(x any) int { return x.(int) }\n')
ASSERT_OK = extra("orders", 'func castIt(x any) int {\n\tn, _ := x.(int)\n\treturn n\n}\n')
CHAN = extra("orders", 'func chanIt() chan int { return make(chan int, 5) }\n')
CHAN_OK = extra("orders", 'func chanIt() chan int {\n\t// a batch is five items\n\treturn make(chan int, 5)\n}\n')
BRANCHY = extra("orders", 'func manyIt(a, b int) int {\n\tif a > 0 && b > 0 {\n\t\treturn 1\n\t}\n\treturn 0\n}\n')
COLOR = 'type color int\n\nconst (\n\tred color = iota\n\tblue\n)\n\nvar _ = []color{red, blue}\n\n'
SWITCH_BAD = extra("orders", COLOR + 'func nameIt(c color) string {\n\tswitch c {\n\tcase red:\n\t\treturn "red"\n\t}\n\treturn ""\n}\n')
SWITCH_DEFAULT = extra("orders", COLOR + 'func nameIt(c color) string {\n\tswitch c {\n\tcase red:\n\t\treturn "red"\n\tdefault:\n\t\treturn ""\n\t}\n}\n')
SWITCH_REASON = extra("orders", COLOR + 'func nameIt(c color) string {\n\tswitch c {\n\tcase red:\n\t\treturn "red"\n\tdefault:\n\t\t// blue has no name yet\n\t\treturn ""\n\t}\n}\n')
TEST_IF = code("api/orders/logic_test", 'package orders\n\nimport "testing"\n\nfunc TestLogic(t *testing.T) {\n\tif t == nil {\n\t\tt.Fatal("x")\n\t}\n}\n')
TEST_TABLE = code("api/orders/logic_test", 'package orders\n\nimport "testing"\n\nfunc TestLogic(t *testing.T) {\n\tcases := []struct{ a int }{{1}}\n\tfor _, c := range cases {\n\t\tt.Log(c.a)\n\t}\n}\n')
NOLINT = extra("orders", 'func quietIt() int {\n\treturn 1 //nolint\n}\n')
NOLINT_OK = extra("orders", 'func quietIt() int {\n\treturn 1 //nolint:gomnd // a fixed answer\n}\n')
RETURN_V = {"api/orders/commit.vulcan": 'package orders\n\nimport "database/sql"\n\nfunc commitIt(tx *sql.Tx) error\n\tensures tx != nil\n{\n\treturn tx.Commit()\n}\n'}
RETURN_L = {"api/orders/commit.go": 'package orders\n\nimport (\n\t"database/sql"\n\n\t"shop/lib/contract"\n)\n\nfunc commitIt(tx *sql.Tx) error {\n\tdefer contract.Ensures(func() bool { return tx != nil })\n\treturn tx.Commit()\n}\n'}
RETURN_PLAIN = extra("orders", 'func commitIt(tx *sql.Tx) error {\n\treturn tx.Commit()\n}\n', "database/sql")
RETRY = extra("orders", 'func tryIt() {\n\tfor {\n\t\ttime.Sleep(1)\n\t}\n}\n', "time")
RETRY_OK = extra("orders", 'func tryIt() {\n\td := time.Duration(1)\n\tfor i := 0; i < 3; i++ {\n\t\ttime.Sleep(d + time.Duration(rand.IntN(3)))\n\t\td *= 2\n\t}\n}\n', "math/rand/v2 time")
DUP_BODY = 'if a > 0 {\n\t\ta = a + 1\n\t}\n\tif a > 1 {\n\t\ta = a + 2\n\t}\n\treturn a\n'
DUP = extra("orders", 'func oneIt(a int) int {\n\t' + DUP_BODY + '}\n\nfunc twoIt(a int) int {\n\t' + DUP_BODY + '}\n')
UNUSED = extra("orders", 'func aloneIt() {}\n')
USED = extra("orders", 'func aloneIt() {}\n\nvar _ = aloneIt\n')
DEPRECATED = extra("orders", 'func readIt(r io.Reader) { _, _ = ioutil.ReadAll(r) }\n', "io io/ioutil")

DECL = lambda **ps: rule("X", "declaration", **ps)
CALL = lambda **ps: rule("X", "call-form", **ps)
LIT = lambda **ps: rule("X", "literal", **ps)
FORM = lambda form, level="error", **ps: rule("X", "code-form", level=level, form=form, **ps)
SERVER_RULE = LIT(type="net/http.Server", require="ReadTimeout WriteTimeout")
TLS_RULE = LIT(type="crypto/tls.Config", deny="InsecureSkipVerify=true", below="MinVersion=771")
COOKIE_RULE = LIT(type="net/http.Cookie", require="Secure=true SameSite=2,3")
PORT_V = "api/orders/component.vulcan"
PORT_L = "api/orders/ports.go"
LEDGER_V = "api/billing/component.vulcan"
LEDGER_L = "api/billing/ports.go"

CASES += [
    # declaration
    ("R70", "declaration: a port signature carries a listed type", rules(DECL(at="port-signature", deny_types="*net/http.Request")), PORT_HTTP,
     {"v": [(PORT_V, "Raw(r", "0807", 0)], "l": [(PORT_L, "Raw(r", "0807", 0)]}),
    ("R71", "A deleted rule of kind declaration stops firing: the same code as R70", rules(), PORT_HTTP, NO),
    ("R72", "declaration: a field of a listed type", rules(DECL(at="field", deny_types="context.Context")), CTX_FIELD, e("api/orders/extra", "ctx context.Context", "0807")),
    ("R73", "declaration: a settings field with no tag", rules(DECL(at="settings-field", names="*Key", require="vulcan=secret")), SETTINGS, e("api/orders/extra", "ApiKey", "0807")),
    ("R74", "declaration: a settings field with its tag", rules(DECL(at="settings-field", names="*Key", require="vulcan=secret")), SETTINGS_TAGGED, NO),
    ("R75", "declaration: an exported struct embeds a type", rules(DECL(at="embedded", **{"in": "library"})), EMBED, e("lib/money/extra", "Order\n}", "0807")),
    ("R76", "declaration: a parameter of a listed type is not first", rules(DECL(at="param", types="context.Context", require="first")), CTX_LAST, e("api/orders/extra", "ctx context", "0807")),
    ("R77", "declaration: a parameter of a listed type has another name", rules(DECL(at="param", types="context.Context", require="name=ctx")), CTX_NAME, e("api/orders/extra", "c context", "0807")),
    ("R78", "declaration: a variable with a listed name and a listed type", rules(DECL(at="var", names="*amount*", deny_types="float64")), FLOAT, e("api/orders/extra", "amount :=", "0807")),
    ("R79", "declaration: an exported name with no doc comment", rules(DECL(at="doc", **{"in": "api/orders"})), UNDOC, NO),
    ("R80", "declaration: a port and its method with doc comments", rules(DECL(at="doc", **{"in": "api/orders"})), PORT_DOC, NO),
    ("R81", "declaration: a port and its method with no doc comment", rules(DECL(at="doc", **{"in": "api/orders"})), PORT_NODOC,
     {"v": [(PORT_V, "Checkout {", "0807", 0), (PORT_V, "Pay(amt", "0807", 0)], "l": [(PORT_L, "Checkout interface", "0807", 0), (PORT_L, "Pay(amt", "0807", 0)]}),
    ("R82", "declaration: a library with a listed name", rules(DECL(at="library", deny_names="money")), NO,
     {"v": [("lib/money/library.vulcan", "money {", "0807", 0)], "l": [("lib/money/money.go", "package money", "0807", 0)]}),
    ("R83", "declaration: an entry of the configuration with no comment", rules(DECL(at="config-entry", require="comment")), CONFIG_BARE,
     {"v": [("system.vulcan", '"sync.Mutex"', "0807", 0)], "l": [("arch/config.go", '"sync.Mutex"', "0807", 0)]}),
    ("R84", "declaration: an entry of the configuration with a comment", rules(DECL(at="config-entry", require="comment")), CONFIG_COMMENT, NO),
    ("R85", "declaration: a port that a gateway provides has no context", rules(DECL(at="wired-port", to="gateway")), NO,
     {"v": [("system.vulcan", "Billing.Ledger:", "0807", 0)], "l": [("cmd/shop/main.go", "(keeper)", "0807", 1)]}),
    ("R86", "declaration: a port method with no requires", rules(DECL(at="port-method", require="requires", **{"in": "api/billing"})), NO,
     {"v": [(LEDGER_V, "Save(id", "0807", 0)], "l": [(LEDGER_L, "Save(id", "0807", 0)]}),
    ("R87", "declaration: except-names leaves out a port method", rules(DECL(at="port-method", require="requires", except_names="Save", **{"in": "api/billing"})), NO, NO),
    # call-form
    ("R88", "call-form: an argument that is not a constant", rules(CALL(funcs="(*database/sql.DB).Exec", require="constant")), SQL_BUILT, e("api/store/extra", '"DELETE FROM', "0808")),
    ("R89", "A deleted rule of kind call-form stops firing: the same code as R88", rules(), SQL_BUILT, NO),
    ("R90", "call-form: a comment removes the finding, with unless-comment", rules(CALL(funcs="(*database/sql.DB).Exec", require="constant", unless_comment="yes")), SQL_COMMENT, NO),
    ("R91", "call-form: a text that the rule denies", rules(CALL(funcs="(*database/sql.DB).Exec", deny_text="BEGIN")), SQL_BEGIN, e("api/store/extra", '"BEGIN"', "0808")),
    ("R92", "call-form: a text that the rule requires", rules(CALL(funcs="errors.New", require_text="^[a-z]")), ERR_TEXT, e("api/orders/extra", '"Bad.', "0808")),
    ("R93", "call-form: nil for a parameter of a listed type", rules(CALL(param_type="context.Context", require="not-nil")), NIL_CTX, e("api/orders/extra", "nil)", "0808")),
    ("R94", "call-form: a key that is not of an unexported type", rules(CALL(funcs="context.WithValue", arg="1", require="unexported-type")), CTX_KEY, e("api/orders/extra", '"k"', "0808")),
    ("R95", "call-form: a marshaled field with no tag", rules(CALL(funcs="encoding/json.Marshal", arg="0", require="tagged=json")), UNTAGGED, e("api/orders/extra", "wire{}", "0808")),
    ("R96", "call-form: a marshaled field with its tag", rules(CALL(funcs="encoding/json.Marshal", arg="0", require="tagged=json")), TAGGED, NO),
    # literal
    ("R97", "literal: a field that the rule requires is not set", rules(SERVER_RULE), SERVER, e("api/web/extra", "http.Server{", "0809")),
    ("R98", "A deleted rule of kind literal stops firing: the same code as R97", rules(), SERVER, NO),
    ("R99", "literal: each required field is set", rules(SERVER_RULE), SERVER_OK, NO),
    ("R100", "literal: new makes a value with no fields set", rules(LIT(type="net/http.Server", require="ReadTimeout")), SERVER_NEW, e("api/web/extra", "new(http", "0809")),
    ("R101", "literal: a value that the rule denies", rules(TLS_RULE), TLS_SKIP, e("api/store/extra", "true}", "0809")),
    ("R102", "literal: a value below the limit", rules(TLS_RULE), TLS_OLD, e("api/store/extra", "tls.VersionTLS10", "0809")),
    ("R103", "literal: an assignment of a value that the rule denies", rules(TLS_RULE), TLS_ASSIGN, e("api/store/extra", "true", "0809")),
    ("R104", "literal: a field with a value outside the list", rules(COOKIE_RULE), COOKIE, e("api/web/extra", "http.Cookie{", "0809")),
    ("R105", "literal: a field with a value in the list", rules(COOKIE_RULE), COOKIE_OK, NO),
    ("R106", "literal: a string that the rule denies", rules(LIT(deny_text="sk_live_")), SECRET_TEXT, e("api/orders/extra", '"sk_live', "0809")),
    ("R107", "literal: a text assigned to a listed name", rules(LIT(names="*password*")), PASSWORD, e("api/orders/extra", '"hunter2"', "0809")),
    # code-form
    ("R110", "code-form dropped-error", rules(FORM("dropped-error", **{"in": "api/orders"})), DROPPED, e("api/orders/extra", "failIt() }", "0810")),
    ("R111", "A deleted rule of kind code-form stops firing: the same code as R110", rules(), DROPPED, NO),
    ("R112", "code-form dropped-error: deferred-ok allows a listed call", rules(FORM("dropped-error", deferred_ok="(*database/sql.Tx).Rollback", **{"in": "api/orders"})), DEFERRED, NO),
    ("R113", "code-form discard-error with no comment", rules(FORM("discard-error", unless_comment="yes")), DISCARD, e("api/orders/extra", "_ = failIt", "0810")),
    ("R114", "code-form discard-error with a comment", rules(FORM("discard-error", unless_comment="yes")), DISCARD_OK, NO),
    ("R115", "code-form error-compare with ==", rules(FORM("error-compare")), COMPARE, e("api/orders/extra", "a == b", "0810")),
    ("R116", "code-form error-compare with errors.Is", rules(FORM("error-compare")), COMPARE_OK, NO),
    ("R117", "code-form error-compare of the text", rules(FORM("error-compare")), TEXT_COMPARE, e("api/orders/extra", 'a.Error() ==', "0810")),
    ("R118", "code-form one-value-assertion", rules(FORM("one-value-assertion")), ASSERT, e("api/orders/extra", "x.(int)", "0810")),
    ("R119", "code-form one-value-assertion with two values", rules(FORM("one-value-assertion")), ASSERT_OK, NO),
    ("R120", "code-form channel-size at level warn: it reports, and the check passes", rules(FORM("channel-size", level="warn", unless_comment="yes")), CHAN, e("api/orders/extra", "make(chan", "w0810")),
    ("R121", "code-form channel-size with a comment", rules(FORM("channel-size", level="warn", unless_comment="yes")), CHAN_OK, NO),
    ("R122", "code-form function-complexity at level warn", rules(FORM("function-complexity", level="warn", limit="2", **{"in": "api/orders"})), BRANCHY, e("api/orders/extra", "manyIt", "w0810")),
    ("R123", "code-form switch-exhaustive: a value is not covered", rules(FORM("switch-exhaustive")), SWITCH_BAD, e("api/orders/extra", "switch c", "0810")),
    ("R124", "code-form switch-exhaustive: a default with no reason", rules(FORM("switch-exhaustive")), SWITCH_DEFAULT, e("api/orders/extra", "default:", "0810")),
    ("R125", "code-form switch-exhaustive: a default with a reason", rules(FORM("switch-exhaustive")), SWITCH_REASON, NO),
    ("R126", "code-form test-logic: a branch in a test", rules(FORM("test-logic")), TEST_IF, e("api/orders/logic_test", "if t == nil", "0810")),
    ("R127", "code-form test-logic: one loop over a table", rules(FORM("test-logic")), TEST_TABLE, NO),
    ("R128", "code-form suppression with no check and no reason", rules(FORM("suppression")), NOLINT, e("api/orders/extra", "//nolint", "0810")),
    ("R129", "code-form suppression with a check and a reason", rules(FORM("suppression")), NOLINT_OK, NO),
    ("R130", "code-form return-effect in a function with ensures", rules(FORM("return-effect", level="warn", funcs="(*database/sql.Tx).Commit")), {"v": RETURN_V, "l": RETURN_L},
     {"v": [("api/orders/commit.vulcan", "tx.Commit()", "w0810", 0)], "l": [("api/orders/commit.go", "tx.Commit()", "w0810", 0)]}),
    ("R131", "code-form return-effect in a function with no ensures", rules(FORM("return-effect", funcs="(*database/sql.Tx).Commit")), RETURN_PLAIN, NO),
    ("R132", "code-form retry-loop: no limit, no backoff, no jitter", rules(FORM("retry-loop")), RETRY, e("api/orders/extra", "for {", "0810")),
    ("R133", "code-form retry-loop with all three", rules(FORM("retry-loop")), RETRY_OK, NO),
    ("R134", "code-form duplicate-function", rules(FORM("duplicate-function", limit="10", **{"in": "api/orders"})), DUP, e("api/orders/extra", "twoIt", "0810")),
    ("R135", "code-form unused-name", rules(FORM("unused-name", **{"in": "api/orders"})), UNUSED, e("api/orders/extra", "aloneIt", "0810")),
    ("R136", "code-form unused-name: a name in use", rules(FORM("unused-name", **{"in": "api/orders"})), USED, NO),
    ("R137", "code-form deprecated-use", rules(FORM("deprecated-use", **{"in": "api/orders"})), DEPRECATED, e("api/orders/extra", "ReadAll", "0810")),
]
# R79: the port has its doc comments, so that the exported name is the only finding.
CASES[[c[0] for c in CASES].index("R79")] = ("R79", "declaration: an exported name with no doc comment", rules(DECL(at="doc", **{"in": "api/orders"})), join(UNDOC, PORT_DOC), e("api/orders/extra", "Exported", "0807"))


# The cases of the parity audit, experiments/poc/parity-r3.md. Each failed before its fix.
def ensured(pkg, sig, cond, body, imports="", before=""):
    """A function with an ensures clause, in the form of each arm: forms that one arm has alone."""
    names = imports.split()
    vimp = "import (\n" + "".join(f'\t"{i}"\n' for i in names) + ")\n\n" if names else ""
    limp = "import (\n" + "".join(f'\t"{i}"\n' for i in names) + '\n\t"shop/lib/contract"\n)\n\n'
    return {"v": {f"api/{pkg}/f.vulcan": f"package {pkg}\n\n{vimp}{before}func {sig}\n\tensures {cond}\n{{\n{body}}}\n"},
            "l": {f"api/{pkg}/f.go": f"package {pkg}\n\n{limp}{before}func {sig} {{\n\tdefer contract.Ensures(func() bool {{ return {cond} }})\n{body}}}\n"}}


def ef(pkg, snippet, digits, offset=0):
    return {"v": [(f"api/{pkg}/f.vulcan", snippet, digits, offset)], "l": [(f"api/{pkg}/f.go", snippet, digits, offset)]}


def both_at(pv, pl, snippet, digits, offset=0):
    return {"v": [(pv, snippet, digits, offset)], "l": [(pl, snippet, digits, offset)]}


DISCARD_RULE = rule("X", "code-form", form="discard-error", unless_comment="yes")
GFN = "func g() error { return nil }\n\n"
REFUND = {"v": {"api/orders/component.vulcan": 'import "shop/lib/money"\n\ncomponent Orders {\n    needs port Checkout {\n        func Pay(amt money.Money) (money.Receipt, error)\n        func Refund(amount float64) error\n    }\n}\n',
                "api/billing/raw.vulcan": "package billing\n\n// Refund pays back.\nfunc (p *Payer) Refund(x float64) error { return nil }\n"},
          "l": {"api/orders/ports.go": 'package orders\n\nimport "shop/lib/money"\n\n// Checkout takes payment for an order.\ntype Checkout interface {\n\tPay(amt money.Money) (money.Receipt, error)\n\tRefund(amount float64) error\n}\n',
                "api/billing/raw.go": "package billing\n\n// Refund pays back.\nfunc (p *Payer) Refund(x float64) error { return nil }\n"}}
PING = {"v": {"api/orders/component.vulcan": 'import (\n\t"context"\n\n\t"shop/lib/money"\n)\n\ncomponent Orders {\n    needs port Checkout {\n        func Pay(amt money.Money) (money.Receipt, error)\n        func Ping(c context.Context) error\n    }\n}\n',
              "api/billing/raw.vulcan": 'package billing\n\nimport "context"\n\n// Ping answers.\nfunc (p *Payer) Ping(ctx context.Context) error { return nil }\n'},
        "l": {"api/orders/ports.go": 'package orders\n\nimport (\n\t"context"\n\n\t"shop/lib/money"\n)\n\n// Checkout takes payment for an order.\ntype Checkout interface {\n\tPay(amt money.Money) (money.Receipt, error)\n\tPing(c context.Context) error\n}\n',
              "api/billing/raw.go": 'package billing\n\nimport "context"\n\n// Ping answers.\nfunc (p *Payer) Ping(ctx context.Context) error { return nil }\n'}}
SAVES_V = "\tif err := Ledger.Save(x); err != nil {\n\t\treturn err\n\t}\n"
SAVES_L = "\tif err := p.ledger.Save(x); err != nil {\n\t\treturn err\n\t}\n"
PAIR = lambda s, n: f"package billing\n\nfunc (p *Payer) oneIt(x string) error {{\n{s * n}\treturn nil\n}}\n\nfunc (p *Payer) twoIt(x string) error {{\n{s * n}\treturn nil\n}}\n"
DUP_PORT = lambda n: {"v": {"api/billing/dup.vulcan": PAIR(SAVES_V, n)}, "l": {"api/billing/dup.go": PAIR(SAVES_L, n)}}
OLD = join(code("lib/money/old", "package money\n\n// Old is old.\n//\n// Deprecated: use New.\nfunc Old() int { return 1 }\n"),
           extra("orders", "func useIt() int { return money.Old() }\n", "shop/lib/money"))
ALIAS_SWITCH = extra("orders", "type s int\n\ntype z = s\n\nconst (\n\ta s = iota\n\tb\n)\n\nvar _ = b\n\nfunc pickIt(v z) int {\n\tswitch v {\n\tcase a:\n\t\treturn 1\n\t}\n\treturn 0\n}\n")
LISTS3 = lambda text: {"v": {"system.vulcan": V_BASE["system.vulcan"].replace("    container Api", "    config {\n        strict_shared: [\n            " + text.replace("\n", "\n            ") + "\n        ]\n    }\n\n    container Api")},
                       "l": {"arch/config.go": 'package arch\n\n// Config is the check configuration.\nvar Config = Settings{\n\tStrictShared: []string{\n\t\t' + text.replace("\n", "\n\t\t") + '\n\t},\n}\n'}}
CFGRULE = rule("X", "declaration", at="config-entry", require="comment")
SQLRULE = rule("X", "call-form", funcs="(*database/sql.DB).Exec", require="constant", unless_comment="yes")
CTX = lambda body: extra("orders", body, "context")
FIELDCTX = rule("X", "declaration", at="field", deny_types="context.Context")
CTXFIRST = rule("X", "declaration", at="param", types="context.Context", require="first")
FLOATVAR = rule("X", "declaration", at="var", names="*amount*", deny_types="float64")
FLOATFIELD = rule("X", "declaration", at="field", names="*Amount*", deny_types="float64")
LOGATTRS = rule("X", "call-form", funcs="log/slog.LogAttrs (*log/slog.Logger).LogAttrs", require="constant")
ONLY = {"in": "api/orders"}
ALLFORMS = rules(*[rule(f"F{i}", "code-form", form=fm, **ONLY) for i, fm in enumerate(["dropped-error", "discard-error", "error-compare", "one-value-assertion", "channel-size", "function-complexity", "switch-exhaustive", "test-logic", "suppression", "return-effect", "retry-loop", "duplicate-function", "unused-name", "deprecated-use"])],
                 rule("D1", "declaration", at="doc", **ONLY), rule("D2", "declaration", at="param", types="context.Context", require="first", **ONLY), rule("D3", "declaration", at="field", deny_types="context.Context", **ONLY),
                 rule("C1", "call-form", funcs="errors.New", require="constant", **ONLY), rule("C2", "call-form", param_type="context.Context", require="not-nil", **ONLY), rule("L1", "literal", type="net/http.Server", require="ReadTimeout", **ONLY))
BROKEN_TEST = 'package orders\n\nimport "testing"\n\nfunc TestX(t *testing.T) {\n\tcases := []struct{ A int }{{1}}\n\tfor _, c := range casess {\n\t\t_ = c\n\t}\n\t_ = cases\n}\n'

CASES += [
    # H1
    ("R140", "H1: a discard after a return, in a function with ensures", rules(DISCARD_RULE), ensured("store", "F(a int) int", "a > 0", "\tif a > 1 { return 1 }\n\t_ = g()\n\treturn 2\n", before=GFN), ef("store", "_ = g()", "0810")),
    ("R141", "H1: a built query after a return, in a function with ensures", rules(SQLRULE), ensured("store", 'F(d *sql.DB, q string) int', "d != nil", '\tif q == "" { return 1 }\n\t_, _ = d.Exec(q)\n\treturn 2\n', imports="database/sql"), ef("store", "q)\n", "0808")),
    ("R142", "H1: a default that follows a return, in a function with ensures", rules(rule("X", "code-form", form="switch-exhaustive")), ensured("orders", "pickIt(c color) int", "c >= 0", "\tswitch c {\n\tcase red:\n\t\treturn 1\n\tdefault:\n\t\treturn 2\n\t}\n", before=COLOR), ef("orders", "default:", "0810")),
    ("R143", "H1: return-effect with unless-comment fires at warn in both", rules(rule("X", "code-form", level="warn", form="return-effect", funcs="(*database/sql.Tx).Commit", unless_comment="yes")), ensured("orders", "commitIt(t *sql.Tx) error", "t != nil", "\treturn t.Commit()\n", imports="database/sql"), ef("orders", "t.Commit()", "w0810")),
    ("R144", "H1: a discard on line 1 of a file, where V's generated header would be", rules(DISCARD_RULE), rawcode("api/store/one", "package store; func g() error { return nil }; func aIt() { _ = g() }\n"), {"v": [("api/store/one.vulcan", "_ = g()", "0810", 0)], "l": [("api/store/one.go", "_ = g()", "0810", 0)]}),
    ("R145", "H1: a real comment still removes the finding, with ensures", rules(DISCARD_RULE), ensured("store", "F(a int) int", "a > 0", "\tif a > 1 { return 1 }\n\t// nothing can be done here\n\t_ = g()\n\treturn 2\n", before=GFN), NO),
    ("R146", "H1: a comment after the code on the line above is not a reason", rules(DISCARD_RULE), extra("store", "func g() error { return nil }\n\nfunc aIt() {\n\tx := 1 // about x\n\t_ = g()\n\t_ = x\n}\n"), e("api/store/extra", "_ = g()", "0810")),
    # M1
    ("R147", "M1: a port parameter with a listed name and type", rules(rule("X", "declaration", at="param", names="*amount*", deny_types="float64")), REFUND, {"v": [("api/orders/component.vulcan", "Refund(amount", "0807", 0)], "l": [("api/orders/ports.go", "amount float64", "0807", 0)]}),
    ("R148", "M1: a port parameter of another kind of name stays accepted", rules(rule("X", "declaration", at="param", names="*timeout*", deny_types="float64")), REFUND, NO),
    ("R149", "M1: a port parameter with another name is one finding in both", rules(rule("X", "declaration", at="param", types="context.Context", require="name=ctx", **{"in": "api/orders"})), PING, {"v": [("api/orders/component.vulcan", "Ping(c", "0807", 0)], "l": [("api/orders/ports.go", "c context", "0807", 0)]}),
    # M2
    ("R150", "M2: a file behind a build tag", None, {"v": {"api/orders/tagged.vulcan": '//go:build foo\n\npackage orders\n\nimport "os"\n\nfunc stopIt() { os.Exit(1) }\n'}, "l": {"api/orders/tagged.go": '//go:build foo\n\npackage orders\n\nimport "os"\n\nfunc stopIt() { os.Exit(1) }\n'}},
     {"v": [("api/orders/tagged.vulcan", "//go:build", "0901", 0)], "l": [("api/orders/orders.go", (2, 1), "0901", 0)]}),
    ("R151", "M2: a file for another system", None, {"v": {"api/orders/extra_plan9.vulcan": "package orders\n\nfunc stopIt() {}\n"}, "l": {"api/orders/extra_plan9.go": "package orders\n\nfunc stopIt() {}\n"}},
     {"v": [("api/orders/extra_plan9.vulcan", "package", "0901", 0)], "l": [("api/orders/orders.go", (2, 1), "0901", 0)]}),
    # M3
    ("R152", "M3: one port call counts the same in both: under the limit in both", rules(rule("X", "code-form", form="duplicate-function", limit="60", similarity="90", **{"in": "api/billing"})), DUP_PORT(4), NO),
    ("R185", "M3: one port call counts the same in both: over the limit in both", rules(rule("X", "code-form", form="duplicate-function", limit="60", similarity="90", **{"in": "api/billing"})), DUP_PORT(6), both_at("api/billing/dup.vulcan", "api/billing/dup.go", "twoIt", "0810")),
    # M4
    ("R153", "M4: a deprecated function of the module itself", rules(rule("X", "code-form", form="deprecated-use", **{"in": "api/orders"})), OLD, e("api/orders/extra", "Old()", "0810")),
    # M5
    ("R154", "M5: a switch over an alias of a type with unexported constants", rules(rule("X", "code-form", form="switch-exhaustive")), ALIAS_SWITCH, e("api/orders/extra", "switch v", "0810")),
    # M7
    ("R155", "M7: a word of require that the place does not take", rules(rule("X", "declaration", at="port-method", require="requirs")), NO, both_at("rules.vulcan", "rules.vulcan", "rule X", "0801")),
    ("R156", "M7: a place with nothing to apply", rules(rule("X", "declaration", at="field")), NO, both_at("rules.vulcan", "rules.vulcan", "rule X", "0801")),
    ("R157", "M7: a table of a type that the package does not declare", rules(rule("X", "transitions", type="lib/money.Ordr", field="Status", allow="a->b")), NO, both_at("rules.vulcan", "rules.vulcan", "rule X", "0801")),
    ("R158", "M7: a table of a field that the type does not have", rules(rule("X", "transitions", type="lib/money.Order", field="Statuss", allow="a->b")), NO, both_at("rules.vulcan", "rules.vulcan", "rule X", "0801")),
    ("R159", "M7: to names a directory that is not a part", rules(rule("X", "declaration", at="wired-port", to="api/nope")), NO, both_at("rules.vulcan", "rules.vulcan", "rule X", "0801")),
    # M11
    ("R160", "M11: a listed method as a value leaves no argument to read", rules(rule("X", "call-form", funcs="(*database/sql.DB).Exec", require="constant")), extra("store", "func runIt(d *sql.DB, q string) {\n\tf := d.Exec\n\t_, _ = f(q)\n}\n", "database/sql"), e("api/store/extra", "Exec\n", "0808")),
    ("R161", "M11, a limit: a call through a local interface passes in both", rules(rule("X", "call-form", funcs="(*database/sql.DB).Exec", require="constant")), extra("store", "type runner interface {\n\tExec(string, ...any) (sql.Result, error)\n}\n\nfunc runIt(d runner, q string) {\n\t_, _ = d.Exec(q)\n}\n", "database/sql"), NO),
    # M12
    ("R162", "M12: a field of an alias of a listed type", rules(FIELDCTX), CTX("type c = context.Context\n\ntype holder struct {\n\tX c\n}\n\nvar _ = holder{}\n"), e("api/orders/extra", "X c", "0807")),
    ("R163", "M12: a field of a pointer to a listed type", rules(FIELDCTX), CTX("type holder struct {\n\tX *context.Context\n}\n\nvar _ = holder{}\n"), e("api/orders/extra", "X *context", "0807")),
    ("R164", "M12: a field of a slice of a listed type", rules(FIELDCTX), CTX("type holder struct {\n\tX []context.Context\n}\n\nvar _ = holder{}\n"), e("api/orders/extra", "X []context", "0807")),
    ("R165", "M12: an alias of a generic type with a listed type argument", rules(FIELDCTX), CTX("type box[v any] struct {\n\tX v\n}\n\ntype t = box[context.Context]\n\nvar _ = t{}\n"), e("api/orders/extra", "t = box", "0807")),
    ("R166", "M12: a field of an interface that embeds a listed type", rules(FIELDCTX), CTX("type i interface{ context.Context }\n\ntype holder struct {\n\tX i\n}\n\nvar _ = holder{}\n"), e("api/orders/extra", "X i", "0807")),
    ("R167", "M12: a parameter of an alias of a listed type", rules(CTXFIRST), CTX("type c = context.Context\n\nfunc doIt(a int, x c) {}\n"), e("api/orders/extra", "x c", "0807")),
    ("R168", "M12: a variable of an alias of a listed type", rules(FLOATVAR), extra("orders", "type m = float64\n\nfunc priceIt() m {\n\tamount := m(1)\n\treturn amount\n}\n"), e("api/orders/extra", "amount :=", "0807")),
    ("R169", "M12: a field that is a slice of a listed type", rules(FLOATFIELD), extra("orders", "type bill struct {\n\tAmounts []float64\n}\n\nvar _ = bill{}\n"), e("api/orders/extra", "Amounts", "0807")),
    ("R170", "M12: a named result", rules(rule("X", "declaration", at="param", names="timeout", deny_types="int")), extra("orders", "func waitIt() (timeout int) {\n\treturn 1\n}\n"), e("api/orders/extra", "timeout int", "0807")),
    ("R171", "M12, a limit: a type defined from a listed type passes in both", rules(FLOATFIELD), extra("orders", "type m float64\n\ntype bill struct {\n\tAmount m\n}\n\nvar _ = bill{}\n"), NO),
    # M13
    ("R172", "M13: a method of net.Dialer", rules(rule("X", "only-in", effect="network", **{"in": "gateway entry"})), extra("orders", "func dialIt(x *net.Dialer) any { return x.DialTCP }\n", "net"), both(e("api/orders/extra", "DialTCP", "0205"), e("api/orders/extra", "DialTCP", "0803"))),
    ("R173", "M13: a method of exec.Cmd", rules(rule("X", "only-in", effect="processes", **{"in": "gateway"})), extra("orders", 'func runIt() {\n\tc := exec.Cmd{Path: "x"}\n\t_ = c.Run()\n}\n', "os/exec"), both(e("api/orders/extra", "Run()", "0205"), e("api/orders/extra", "Run()", "0803"))),
    ("R174", "M13: the value crypto/rand.Reader", rules(rule("X", "only-in", effect="randomness", **{"in": "gateway"})), extra("orders", "func randIt(b []byte) { _, _ = io.ReadFull(rand.Reader, b) }\n", "crypto/rand io"), e("api/orders/extra", "Reader,", "0803")),
    ("R175", "M13: slog.LogAttrs", rules(LOGATTRS), extra("orders", "func logIt(c context.Context, m string) { slog.LogAttrs(c, slog.LevelInfo, m) }\n", "context log/slog"), e("api/orders/extra", "m) }", "0808")),
    ("R176", "M13: context.WithTimeout reads the clock", rules(rule("X", "only-in", effect="clock", **{"in": "gateway"})), extra("orders", "func timeIt(c context.Context) { _, f := context.WithTimeout(c, 1); f() }\n", "context"), e("api/orders/extra", "WithTimeout", "0803")),
    ("R177", "M13: template.ParseFiles reads a file", rules(rule("X", "only-in", effect="files", **{"in": "gateway"})), extra("orders", 'func readIt() { _, _ = template.ParseFiles("x") }\n', "html/template"), e("api/orders/extra", "ParseFiles", "0803")),
    ("R178", "M13: net/rpc/jsonrpc.Dial", rules(rule("X", "only-in", effect="network", **{"in": "gateway entry"})), extra("orders", 'func dialIt() { _, _ = jsonrpc.Dial("tcp", "x") }\n', "net/rpc/jsonrpc"), both(e("api/orders/extra", "Dial", "0205"), e("api/orders/extra", "Dial", "0803"))),
    # L1
    ("R179", "L1: a comment on the entry before is not a reason", rules(CFGRULE), LISTS3('"a.B", // why\n"c.D",'), both_at("system.vulcan", "arch/config.go", '"c.D"', "0807")),
    ("R180", "L1: a block comment above an entry is a reason", rules(CFGRULE), LISTS3('/* why */\n"a.B",'), NO),
    ("R181", "L1: // inside an entry's text is no comment", rules(CFGRULE), LISTS3('"http://x",'), both_at("system.vulcan", "arch/config.go", '"http://x"', "0807")),
    # L3, L4
    ("R183", "L3: ensures on a method of a generic type", rules(rule("X", "code-form", level="warn", form="return-effect", funcs="(*database/sql.Tx).Commit")), ensured("orders", "(b *box[T]) commitIt(t *sql.Tx) error", "t != nil", "\treturn t.Commit()\n", imports="database/sql", before="type box[T any] struct{}\n\n"), ef("orders", "t.Commit()", "w0810")),
    ("R184", "L4: at var does not read V's lowered temporaries", rules(rule("X", "declaration", at="var", deny_types="float64")), ensured("orders", "priceIt(a float64) float64", "a > 0", "\treturn a\n"), NO),
]

# Forms that one arm has alone. These are not in TestRuleCasesAgree.
def main_with(body, decls=""):
    return L_BASE["cmd/shop/main.go"].replace("func main() {", decls + "func main() {").replace("\tkeeper := store.Open()\n\tpayer := billing.NewPayer(keeper)\n", body)


UNREAD = [
    ("LW1", "H2: a helper that returns the provider", main_with("\tpayer := billing.NewPayer(mk())\n", "func mk() billing.Ledger { return store.Open() }\n\n"), [("mk())", 0)]),
    ("LW2", "H2: a helper that takes the provider", main_with("\tpayer := mk(store.Open())\n", "func mk(l billing.Ledger) *billing.Payer { return billing.NewPayer(l) }\n\n"), [("l) }", 0), ("(payer)", 1)]),
    ("LW3", "H2: a field", main_with("\tb := box{store.Open()}\n\tpayer := billing.NewPayer(b.l)\n", "type box struct{ l billing.Ledger }\n\n"), [("b.l)", 0)]),
    ("LW4", "H2: a slice", main_with("\tls := []billing.Ledger{store.Open()}\n\tpayer := billing.NewPayer(ls[0])\n"), [("ls[0])", 0)]),
    ("LW5", "H2: a closure", main_with("\tf := func() billing.Ledger { return store.Open() }\n\tpayer := billing.NewPayer(f())\n"), [("f())", 0)]),
    ("LW6", "H2: a wire inside a closure", main_with("\tvar payer *billing.Payer\n\tfunc() { payer = billing.NewPayer(store.Open()) }()\n"), [("store.Open()) }", 0)]),
    ("LW7", "H2: a variable with a second value", main_with("\tk := store.Open()\n\tk = store.Open()\n\tpayer := billing.NewPayer(k)\n"), [("k)", 0)]),
    ("LW8", "H2: a map", main_with('\tm := map[string]billing.Ledger{"a": store.Open()}\n\tpayer := billing.NewPayer(m["a"])\n'), [('m["a"])', 0)]),
    ("LW9", "H2: a function variable", main_with("\tf := store.Open\n\tpayer := billing.NewPayer(f())\n"), [("Open\n", 0), ("f())", 0)]),
]
PLAIN = main_with("\tvar k billing.Ledger = store.Open()\n\tj := k\n\tpayer := billing.NewPayer(j)\n")
LOCK_HELPER = main_with("\tpayer := billing.NewPayer(mk())\n", "func mk() billing.Ledger { return store.Open() }\n\n")
TESTDATA_S = 'package sub\n\nimport "os"\n\n// X stops.\nfunc X() { os.Exit(1) }\n'
TESTDATA_USE = 'package orders\n\nimport "shop/api/orders/testdata/sub"\n\nfunc stopIt() { sub.X() }\n'
ERR08 = rule("X", "forbid-call", funcs="os.Exit", **{"in": "leaf gateway entry library"})


def arm_cases():
    out = []
    for id_, desc, main_text, wants in UNREAD:
        out.append(("l", id_, desc, {"cmd/shop/main.go": main_text}, [("cmd/shop/main.go", s, "0909", o) for s, o in wants]))
    out.append(("l", "LW10", "H2: plain forms stay accepted", {"cmd/shop/main.go": PLAIN}, []))
    out.append(("l", "LW11", "H2: a helper cannot hide a new dependency from the lock", {"cmd/shop/main.go": LOCK_HELPER, "rules.vulcan": rules(LOCKED), "rules.lock": LOCK_NO_WIRE}, [("cmd/shop/main.go", "mk())", "0909", 0)]))
    out.append(("l", "LT1", "H3: code under testdata", {"api/orders/testdata/sub/s.go": TESTDATA_S, "api/orders/extra.go": TESTDATA_USE, "rules.vulcan": rules(ERR08)}, [("api/orders/extra.go", '"shop/api', "0002", 0)]))
    out.append(("l", "LT2", "H3: code under testdata at the root of the module", {"testdata/sub/s.go": TESTDATA_S, "api/orders/extra.go": TESTDATA_USE.replace("shop/api/orders/testdata", "shop/testdata")}, [("api/orders/extra.go", '"shop/testdata', "0002", 0)]))
    out.append(("v", "VT1", "H3: code under testdata", {"api/orders/testdata/sub/s.vulcan": TESTDATA_S, "api/orders/extra.vulcan": TESTDATA_USE, "rules.vulcan": rules(ERR08)}, [("api/orders/extra.vulcan", '"shop/api', "0008", 0)]))
    out.append(("l", "LN1", "L2: in L, the prefix vulcan_ is an ordinary name", {"api/orders/extra.go": "package orders\n\nfunc vulcan_x() {}\n", "rules.vulcan": rules(rule("X", "code-form", form="unused-name", **{"in": "api/orders"}))}, [("api/orders/extra.go", "vulcan_x", "0810", 0)]))
    out.append(("l", "ET1", "H4: a type error in a test, with each form on: the error, and no panic", {"api/orders/x_test.go": BROKEN_TEST, "rules.vulcan": ALLFORMS, **PORT_DOC["l"]}, "broken"))
    out.append(("v", "ET1", "H4: a type error in a test, with each form on: the error, and no panic", {"api/orders/x_test.vulcan": BROKEN_TEST, "rules.vulcan": ALLFORMS, **PORT_DOC["v"]}, [("api/orders/x_test.vulcan", "casess", "0008", 0)]))
    return out


def pos(text, snippet, offset):
    if isinstance(snippet, tuple):
        return snippet
    i = text.index(snippet) + offset
    line = text.count("\n", 0, i) + 1
    col = i - (text.rfind("\n", 0, i) + 1) + 1
    return line, col


def archive(desc, files, wants, prefix, base):
    all_files = dict(base)
    all_files.update(files)
    lines = []
    for path, snippet, digits, offset in sorted(wants):
        text = all_files.get(path, "")
        line, col = pos(text, snippet, offset)
        word = "warn" if digits.startswith("w") else "error"
        lines.append(f"{word} {path}:{line}:{col} {prefix}{digits.lstrip('w')}")
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
    for arm, id_, desc, files, wants in arm_cases():
        base = dict(V_BASE if arm == "v" else L_ALL)
        base.update(files)
        prefix = "K" if arm == "v" else "VL"
        if wants == "broken":
            text = archive(desc, base, [], prefix, base).replace("-- want.txt --\naccept\n", "-- want.txt --\nbroken\n")
        else:
            text = archive(desc, base, wants, prefix, base)
        (V_DIR if arm == "v" else L_DIR).joinpath(f"{id_}.txtar").write_text(text, encoding="utf-8", newline="\n")


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
