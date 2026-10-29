# gen_cases.py writes the Go form of the conformance cases that the desk test maps to the
# analyzers A1 to A8 (docs/poc/desk-test-r2.md), as txtar archives: python gen_cases.py testdata
#
# base.txtar is the Go form of the base system of conformance/gen_cases.py: Web (entry for
# Customer) needs Orders, Orders needs Checkout, Billing provides it, money is a library. Each
# case is an overlay of base.txtar: its files replace the base's, and a file "DELETE" removes one.
# Each expected diagnostic comes from a marker @VL0101@ in a file, at the line and column of the
# token after it; the marker is removed. A case with no marker is "accept".

import os, re, sys

OUT = sys.argv[1]

BASE = {
"arch/arch.go": '''package arch

// The persons and the external systems.
const (
	Customer Person = "Customer" // buys goods
)

// Manifest is the architecture of the shop.
var Manifest = System{
	Parts: []Part{
		Main("cmd/shop"),
		Entry("api/web", Customer),
		Leaf("api/orders"),
		Leaf("api/billing"),
		Library("lib/money"),
		Library("lib/contract"),
	},
}
''',
"lib/money/money.go": '''// Package money holds amounts.
package money

// Money is an amount in cents.
type Money int64

// Receipt records a payment.
type Receipt struct {
	Amount Money
}
''',
"api/billing/billing.go": '''// Package billing takes payments.
package billing

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
"api/orders/ports.go": '''package orders

import "shop/lib/money"

// Checkout takes payment for an order.
type Checkout interface {
	Pay(amt money.Money) (money.Receipt, error)
}
''',
"api/orders/orders.go": '''// Package orders places orders.
package orders

import "shop/lib/money"

// Service places orders.
type Service struct {
	checkout Checkout
}

// NewService returns a Service.
func NewService(checkout Checkout) *Service { return &Service{checkout: checkout} }

// Place takes payment for an order.
func (s *Service) Place(amt money.Money) (money.Receipt, error) {
	return s.checkout.Pay(amt)
}
''',
"api/web/ports.go": '''package web

import "shop/lib/money"

// Orders places orders.
type Orders interface {
	Place(amt money.Money) (money.Receipt, error)
}
''',
"api/web/web.go": '''// Package web is the entry of the customer.
package web

import (
	"context"
	"fmt"
)

// API is the entry of the customer.
type API struct {
	orders Orders
}

// New returns the API.
func New(orders Orders) *API { return &API{orders: orders} }

// Start places one order.
func (a *API) Start(ctx context.Context) error {
	r, err := a.orders.Place(42)
	if err != nil {
		return err
	}
	fmt.Println("paid", r.Amount)
	return nil
}
''',
"cmd/shop/main.go": '''// Command shop runs the shop.
package main

import (
	"context"
	"os"

	"shop/api/billing"
	"shop/api/orders"
	"shop/api/web"
)

func main() {
	payer := billing.NewPayer()
	service := orders.NewService(payer)
	api := web.New(service)
	if err := api.Start(context.Background()); err != nil {
		os.Exit(1)
	}
}
''',
}

def arch(parts, consts='\tCustomer Person = "Customer" // buys goods\n'):
    return ('package arch\n\n// The persons and the external systems.\nconst (\n' + consts + ')\n\n'
            '// Manifest is the architecture of the shop.\nvar Manifest = System{\n' +
            '\tParts: []Part{\n' + ''.join('\t\t' + p + ',\n' for p in parts) + '\t},\n}\n')

PARTS = ['Main("cmd/shop")', 'Entry("api/web", Customer)', 'Leaf("api/orders")', 'Leaf("api/billing")',
         'Library("lib/money")', 'Library("lib/contract")']

def strict(extra=''):
    return {"arch/config.go": 'package arch\n\n// Config is the check configuration of vulcan-lint.\nvar Config = Settings{\n\tStrict: true,\n' + extra + '}\n'}

def billing(decls='', imports='"shop/lib/money"', pay='Pay(amt money.Money) (money.Receipt, error)', body='return money.Receipt{Amount: amt}, nil', extra=''):
    imp = 'import ' + imports + '\n' if not imports.startswith('(') else 'import ' + imports + '\n'
    return ('// Package billing takes payments.\npackage billing\n\n' + imp + '\n' + decls +
            '// Payer takes payments.\ntype Payer struct{}\n\n// NewPayer returns a Payer.\nfunc NewPayer() *Payer { return &Payer{} }\n\n'
            '// Pay takes a payment.\nfunc (p *Payer) ' + pay + ' {\n\t' + body + '\n}\n' + extra)

MONEY = '''// Package money holds amounts.
package money

%s
// Money is an amount in cents.
type Money int64

// Receipt records a payment.
type Receipt struct {
	Amount Money
}
'''

MONEY_VARS = '''// Config holds fees.
type Config struct{ Fee Money }

// Default is the default config.
var Default = Config{Fee: 1}

// Rates are the tax rates.
var Rates = [2]Money{1, 2}

// Limit is the largest order.
var Limit Money = 100

// Fees are the fees by country.
var Fees = map[string]Money{}
'''

# A main that wires no Orders: for a case whose Checkout no provider can have.
MAIN_STUB = '''// Command shop runs the shop.
package main

import (
	"context"
	"os"

	"shop/api/web"
	"shop/lib/money"
)

type place struct{}

func (place) Place(amt money.Money) (money.Receipt, error) { return money.Receipt{Amount: amt}, nil }

func main() {
	api := web.New(place{})
	if err := api.Start(context.Background()); err != nil {
		os.Exit(1)
	}
}
'''

def orders(place='return s.checkout.Pay(amt)', decls='', imports='"shop/lib/money"', ctor='func NewService(checkout Checkout) *Service { return &Service{checkout: checkout} }'):
    return ('// Package orders places orders.\npackage orders\n\nimport ' + imports + '\n\n' + decls +
            '// Service places orders.\ntype Service struct {\n\tcheckout Checkout\n}\n\n'
            '// NewService returns a Service.\n' + ctor + '\n\n'
            '// Place takes payment for an order.\nfunc (s *Service) Place(amt money.Money) (money.Receipt, error) {\n\t' + place + '\n}\n')

def web(decls='', imports='(\n\t"context"\n\t"fmt"\n)'):
    return ('// Package web is the entry of the customer.\npackage web\n\nimport ' + imports + '\n\n' + decls +
            '// API is the entry of the customer.\ntype API struct {\n\torders Orders\n}\n\n'
            '// New returns the API.\nfunc New(orders Orders) *API { return &API{orders: orders} }\n\n'
            '// Start places one order.\nfunc (a *API) Start(ctx context.Context) error {\n\tr, err := a.orders.Place(42)\n'
            '\tif err != nil {\n\t\treturn err\n\t}\n\tfmt.Println("paid", r.Amount)\n\treturn nil\n}\n')

def ports(methods, imports='"shop/lib/money"', extra=''):
    return ('package orders\n\nimport ' + imports + '\n\n// Checkout takes payment for an order.\ntype Checkout interface {\n' +
            ''.join('\t' + m + '\n' for m in methods) + '}\n' + extra)

def main(wire):
    return BASE["cmd/shop/main.go"].replace('\tpayer := billing.NewPayer()\n', '\tpayer := ' + wire + '\n')

FUNC5 = 'Pay(amt money.Money, done func(money.Receipt)) (money.Receipt, error)'
N05 = {"api/billing/billing.go": billing(pay=FUNC5),
       "api/orders/ports.go": ports(['@VL0123@' + FUNC5]),
       "api/orders/orders.go": orders('return s.checkout.Pay(amt, nil)')}
N06 = {"api/billing/billing.go": billing(imports='(\n\t"io"\n\t"strings"\n\n\t"shop/lib/money"\n)',
           extra='\n// Statement returns the statement.\nfunc (p *Payer) Statement() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("")), nil }\n'),
       "api/orders/ports.go": ports(['Pay(amt money.Money) (money.Receipt, error)', '@VL0123@Statement() (io.ReadCloser, error)'],
                                    imports='(\n\t"io"\n\n\t"shop/lib/money"\n)')}
N08_WEB = web('func ping() { _, _ = http.@VL0125@Get("http://127.0.0.1:1") }\n\nvar _ = ping\n\n', '(\n\t"context"\n\t"fmt"\n\t"net/http"\n)')
N10_BILLING = billing('func mode() { _ = os.@VL0211@Setenv("MODE", "x") }\n\nvar _ = mode\n\n', '(\n\t"os"\n\n\t"shop/lib/money"\n)')

def unmark(d):
    return {k: re.sub(r'@VL\d{4}@', '', v) for k, v in d.items()}

CASES = {
# A1 layout
"P10": ("A package that the manifest does not list (K0002: code outside a leaf or a library).",
        {"misc/x.go": "package @VL0002@misc\n\n// X does nothing.\nfunc X() {}\n"}),
"P11": ("A leaf directory that the manifest does not list (K0003). Go has no component file, so a package that the manifest does not list is VL0002, as P10.",
        {"api/extra/extra.go": "// Package extra is not in the manifest.\npackage @VL0002@extra\n\n// X does nothing.\nfunc X() {}\n"}),
"P12": ("A package clause that does not name its directory (K0004).",
        {"api/billing/billing.go": BASE["api/billing/billing.go"].replace("package billing", "package @VL0004@bill"),
         "cmd/shop/main.go": BASE["cmd/shop/main.go"].replace('\t"shop/api/billing"', '\tbill "shop/api/billing"').replace("billing.NewPayer()", "bill.NewPayer()")}),
"P40": ("A library outside lib/ (K0003).",
        {"arch/arch.go": arch(PARTS + ['@VL0003@Library("api/util")']),
         "api/util/util.go": "// Package util is a library in the wrong place.\npackage util\n\n// Twice doubles n.\nfunc Twice(n int) int { return 2 * n }\n"}),
"A16": ("Two parts with one directory (K0121).",
        {"arch/arch.go": arch(PARTS + ['@VL0121@Leaf("api/orders")'])}),
"P15": ("Two gateway leaves for one external system (K0207).",
        {"arch/arch.go": arch(['Main("cmd/shop")', 'Entry("api/web", Customer)', 'Gateway("api/billing", Stripe)', '@VL0207@Gateway("api/orders", Stripe)', 'Library("lib/money")', 'Library("lib/contract")'],
                              consts='\tCustomer Person   = "Customer" // buys goods\n\tStripe   External = "Stripe"   // cards\n')}),
"P33": ("The manifest imports a leaf (K0110: system.vulcan imports a leaf).",
        {"arch/arch.go": arch(PARTS).replace("package arch\n", 'package arch\n\nimport _ @VL0110@"shop/api/billing"\n')}),

# A2 imports
"A01": ("Leaf Orders imports the package of leaf Billing (K0101).",
        {"api/orders/orders.go": orders('return billing.NewPayer().Pay(amt)', imports='(\n\t@VL0101@"shop/api/billing"\n\t"shop/lib/money"\n)')}),
"A02": ("The port of Orders names a type of Billing (K0110). In Go the import is VL0101, and the type in the port signature is VL0112.",
        {"api/orders/ports.go": ports(['@VL0112@Pay(amt money.Money) (*billing.Payer, error)'], imports='(\n\t@VL0101@"shop/api/billing"\n\t"shop/lib/money"\n)'),
         "api/orders/orders.go": orders('_, err := s.checkout.Pay(amt)\n\treturn money.Receipt{Amount: amt}, err'),
         "api/billing/billing.go": billing(pay='Pay(amt money.Money) (*Payer, error)', body='return p, nil')}),
"A13": ("//go:linkname in leaf Orders (K0122).",
        {"api/orders/orders.go": orders(decls='@VL0122@//go:linkname pay shop/api/billing.pay\nfunc pay() error\n\n', imports='(\n\t_ "unsafe"\n\n\t"shop/lib/money"\n)')}),
"P38": ("Near miss: //go:linkname inside a raw string.",
        {"api/orders/orders.go": orders(decls='var doc = `\n//go:linkname x y\n`\n\nvar _ = doc\n\n')}),
"A14": ("A library imports a leaf (K0204).",
        {"arch/arch.go": arch(PARTS + ['Library("lib/util")']),
         "lib/util/util.go": '// Package util imports a leaf.\npackage util\n\nimport @VL0204@"shop/api/billing"\n\n// Pay makes a payer.\nvar Pay = billing.NewPayer\n'}),
"N07": ("Strict: a leaf imports unsafe (K0124).",
        {**strict(),
         "api/billing/billing.go": billing('var size = unsafe.Sizeof(0)\n\nvar _ = size\n\n', '(\n\t@VL0124@"unsafe"\n\n\t"shop/lib/money"\n)')}),

# A3 ports
"A03": ("Orders needs port Checkout, and main gives it no provider (K0104): in Go, main passes nil.",
        {"cmd/shop/main.go": BASE["cmd/shop/main.go"].replace("\tpayer := billing.NewPayer()\n", "\t_ = billing.NewPayer()\n").replace("orders.NewService(payer)", "orders.NewService(@VL0104@nil)")}),
"A09": ("A port method parameter of a type private to Orders (K0112).",
        {"api/orders/ports.go": ports(['@VL0112@Pay(o Order, amt money.Money) (money.Receipt, error)']),
         "api/orders/orders.go": orders('return s.checkout.Pay(Order{}, amt)', decls='// Order is an order.\ntype Order struct{ ID int }\n\n'),
         "cmd/shop/main.go": MAIN_STUB}),
"P36": ("An unexported port method (K0110).",
        {"api/orders/ports.go": ports(['@VL0110@pay(amt money.Money) (money.Receipt, error)']),
         "api/orders/orders.go": orders('return s.checkout.pay(amt)'),
         "cmd/shop/main.go": MAIN_STUB}),
"A10": ("Code assigns to the port (K0116): in Go, to the field that holds it.",
        {"api/orders/orders.go": orders('@VL0116@s.checkout = nil\n\treturn money.Receipt{}, nil')}),
"A10b": ("A range statement that assigns to a port (K0116).",
         {"api/orders/orders.go": orders('var ch chan Checkout\n\tfor @VL0116@s.checkout = range ch {\n\t}\n\treturn money.Receipt{}, nil')}),
"A10c": ("Near miss: a local variable named like the port.",
         {"api/orders/orders.go": orders('r, err := s.checkout.Pay(amt)\n\tcheckout := 3\n\tcheckout = 4\n\t_ = checkout\n\treturn r, err')}),
"A11": ("A leaf declares func init() (K0117).",
        {"api/orders/orders.go": orders(decls='func @VL0117@init() {}\n\n')}),
"P41": ("A library that declares a port (K0201).",
        {"lib/money/ports.go": "package @VL0201@money\n\n// Clock tells the time.\ntype Clock interface {\n\tNow() int64\n}\n"}),
"P52": ("A port contract that names an identifier of its own leaf (K0120).",
        {"api/orders/ports.go": ports(['Pay(amt money.Money) (money.Receipt, error)'], imports='(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)',
             extra='\n// checkedCheckout checks the contract of Checkout at each call.\ntype checkedCheckout struct{ Checkout }\n\n'
                   'func (c checkedCheckout) Pay(amt money.Money) (money.Receipt, error) {\n\tcontract.Requires(amt < @VL0120@MaxAmount)\n\treturn c.Checkout.Pay(amt)\n}\n'),
         "api/orders/orders.go": orders(decls='// MaxAmount is the largest payment.\nconst MaxAmount = 1000\n\n',
             ctor='func NewService(checkout Checkout) *Service { return &Service{checkout: checkedCheckout{checkout}} }')}),
"N05": ("Strict: a port method takes a function (K0123).", {**strict(), **N05}),
"N06": ("Strict: a port method returns io.ReadCloser (K0123).", {**strict(), **N06}),
"N06b": ("Strict: the same with StrictInterfaces: io.ReadCloser: accepted.",
         {**strict('\tStrictInterfaces: []string{"io.ReadCloser"},\n'), **unmark(N06)}),
"P18": ("A constructor with a parameter that is not a settings struct (K0604).",
        {"api/billing/billing.go": billing().replace("func NewPayer() *Payer", "func @VL0604@NewPayer(n int) *Payer"),
         "cmd/shop/main.go": main("billing.NewPayer(1)")}),
"P24": ("A constructor whose settings type is not exported (K0604).",
        {"api/billing/billing.go": billing('type opts struct{ Fee int }\n\n').replace("func NewPayer() *Payer", "func @VL0604@NewPayer(o opts) *Payer"),
         "cmd/shop/main.go": main("billing.NewPayer(struct{ Fee int }{1})")}),

# A4 main
"P27": ("A setting with no value (K0603).",
        {"api/web/web.go": web('// Settings configure Web.\ntype Settings struct {\n\tWait time.Duration\n}\n\n', '(\n\t"context"\n\t"fmt"\n\t"time"\n)')
             .replace("Start(ctx context.Context) error", "Start(ctx context.Context, s Settings) error").replace('"paid", r.Amount)', '"paid", r.Amount, s.Wait)'),
         "cmd/shop/main.go": BASE["cmd/shop/main.go"].replace("api.Start(context.Background())", "api.Start(context.Background(), @VL0603@web.Settings{})")}),
"P32": ("A function literal in a setting (K0606).",
        {"api/billing/billing.go": billing('// Settings configure a Payer.\ntype Settings struct {\n\tFee int64\n}\n\n').replace("func NewPayer() *Payer", "func NewPayer(s Settings) *Payer"),
         "cmd/shop/main.go": main("billing.NewPayer(billing.Settings{Fee: @VL0606@func() int64 { return 1 }()})")}),
"W07": ("A secret setting given a literal (K0605).",
        {"api/billing/billing.go": billing('// Settings configure a Payer.\ntype Settings struct {\n\tKey string `vulcan:"secret"`\n}\n\n').replace("func NewPayer() *Payer", "func NewPayer(s Settings) *Payer"),
         "cmd/shop/main.go": main('billing.NewPayer(billing.Settings{Key: @VL0605@"sk_live_123"})')}),

# A5 effects
"N01": ("A leaf that is not a gateway or an entry leaf calls http.Get (K0205).",
        {"api/billing/billing.go": billing('func ping() error {\n\t_, err := http.@VL0205@Get("https://example.com")\n\treturn err\n}\n\nvar _ = ping\n\n', '(\n\t"net/http"\n\n\t"shop/lib/money"\n)')}),
"N01b": ("Near miss: a gateway leaf calls http.Get.",
         {"arch/arch.go": arch(['Main("cmd/shop")', 'Entry("api/web", Customer)', 'Leaf("api/orders")', 'Gateway("api/billing", Bank)', 'Library("lib/money")', 'Library("lib/contract")'],
                               consts='\tCustomer Person   = "Customer" // buys goods\n\tBank     External = "Bank"     // cards\n'),
          "api/billing/billing.go": billing('func ping() error {\n\t_, err := http.Get("https://example.com")\n\treturn err\n}\n\nvar _ = ping\n\n', '(\n\t"net/http"\n\n\t"shop/lib/money"\n)')}),
"N03": ("A library calls os.Open (K0203).",
        {"lib/money/money.go": MONEY.replace("package money\n\n%s", 'package money\n\nimport "os"\n\n// Load opens a rate file.\nfunc Load() error {\n\t_, err := os.@VL0203@Open("rates.txt")\n\treturn err\n}\n')}),
"N04": ("A library queries through a *sql.DB that it receives: accepted.",
        {"lib/money/money.go": MONEY.replace("package money\n\n%s", 'package money\n\nimport "database/sql"\n\n// Count counts the receipts.\nfunc Count(db *sql.DB) (int, error) {\n\tvar n int\n\terr := db.QueryRow("select count(*) from receipts").Scan(&n)\n\treturn n, err\n}\n')}),
"N08": ("Strict: an entry leaf calls http.Get (K0125).", {**strict(), "api/web/web.go": N08_WEB}),
"N10": ("Strict: a leaf calls os.Setenv (K0211).", {**strict(), "api/billing/billing.go": N10_BILLING}),
"N11": ("Strict: N05, N08, and N10 in leaves listed in StrictExempt: accepted.",
        {**strict('\tStrictExempt: []string{"api/orders", "api/web", "api/billing"},\n'),
         **unmark(N05), "api/web/web.go": unmark({0: N08_WEB})[0],
         "api/billing/billing.go": unmark({0: N10_BILLING})[0].replace("Pay(amt money.Money) (money.Receipt, error)", FUNC5)}),
"N12": ("Ordinary mode: N05, N08, and N10 are accepted.",
        {**unmark(N05), "api/web/web.go": unmark({0: N08_WEB})[0],
         "api/billing/billing.go": unmark({0: N10_BILLING})[0].replace("Pay(amt money.Money) (money.Receipt, error)", FUNC5)}),
"N13": ("Strict: the base system, a serving entry leaf, and a dialing gateway are accepted.",
        {"arch/arch.go": arch(['Main("cmd/shop")', 'Entry("api/web", Customer)', 'Leaf("api/orders")', 'Gateway("api/billing", Bank)', 'Library("lib/money")', 'Library("lib/contract")'],
                              consts='\tCustomer Person   = "Customer" // buys goods\n\tBank     External = "Bank"     // cards\n'),
         **strict(),
         "api/billing/billing.go": billing('func ping() { _, _ = http.Get("http://127.0.0.1:1") }\n\nvar _ = ping\n\n', '(\n\t"net/http"\n\n\t"shop/lib/money"\n)'),
         "api/web/web.go": web('func listen() {\n\tl, err := net.Listen("tcp", "127.0.0.1:0")\n\tif err == nil {\n\t\tl.Close()\n\t}\n}\n\nvar _ = listen\n\n', '(\n\t"context"\n\t"fmt"\n\t"net"\n)')}),

# A6 library state
"A15": ("A library var assigned in a function after startup (K0202).",
        {"lib/money/money.go": MONEY % '// Rate is the tax rate.\nvar Rate = 20\n\n// SetRate changes the rate.\nfunc SetRate(r int) { @VL0202@Rate = r }\n'}),
"A15b": ("A field of a library struct var assigned in a leaf (K0202).",
         {"lib/money/money.go": MONEY % MONEY_VARS,
          "api/billing/billing.go": billing('func tweak() { @VL0202@money.Default.Fee = 99 }\n\nvar _ = tweak\n\n')}),
"A15c": ("An element of a library array var assigned (K0202).",
         {"lib/money/money.go": MONEY % MONEY_VARS,
          "api/billing/billing.go": billing('func tweak() { @VL0202@money.Rates[0] = 7 }\n\nvar _ = tweak\n\n')}),
"A15d": ("A library var assigned in package-level function literals (K0202).",
         {"lib/money/money.go": MONEY % MONEY_VARS,
          "api/billing/billing.go": billing('var _ = func() bool { @VL0202@money.Limit = 1; return true }()\n\nvar reset = func() { @VL0202@money.Limit = 0 }\n\nvar _ = reset\n\n')}),
"A15e": ("Near miss: reading library vars, copying one, and writing through a map are not assignments to the var.",
         {"lib/money/money.go": MONEY % MONEY_VARS,
          "api/billing/billing.go": billing('func use() money.Money {\n\tc := money.Default\n\tc.Fee = 5\n\tmoney.Fees["ch"] = 3\n\treturn c.Fee + money.Limit + money.Rates[1]\n}\n\nvar _ = use\n\n')}),
"N09": ("Strict: a library var holds a channel (K0210).",
        {**strict(), "lib/money/money.go": MONEY % '// Events carries events.\nvar @VL0210@Events = make(chan int)\n'}),
"N09b": ("Strict: a library var of an error type, and of a type in StrictShared, is accepted.",
         {**strict('\tStrictShared: []string{"*regexp.Regexp"},\n'),
          "lib/money/money.go": MONEY.replace("package money\n\n%s", 'package money\n\nimport (\n\t"errors"\n\t"regexp"\n)\n\n// ErrBad is an error.\nvar ErrBad = errors.New("bad")\n\n// Code matches a code.\nvar Code = regexp.MustCompile(`^[A-Z]+$`)\n')}),

# A7 contracts
"C02": ("Requires amt > 0, and a caller passes the constant 0 (K0501).",
        {"api/billing/billing.go": billing('func charge(amt money.Money) money.Money {\n\tcontract.Requires(amt > 0)\n\treturn amt\n}\n\nvar _ = func() money.Money { return @VL0501@charge(0) }\n\n',
                                           '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),
"L4": ("A constant call through the receiver that breaks Requires (K0501).",
       {"api/billing/billing.go": billing('// Account holds money.\ntype Account struct{ Balance int }\n\n// Invariant holds after every change.\nfunc (a Account) Invariant() bool { return a.Balance >= 0 }\n\n'
                                          '// Withdraw takes money out.\nfunc (a *Account) Withdraw(x int) {\n\tcontract.Requires(x > 0)\n\tdefer contract.Keep(a)\n\ta.Balance -= x\n}\n\n'
                                          '// Fix withdraws a constant.\nfunc (a *Account) Fix() {\n\tdefer contract.Keep(a)\n\t@VL0501@a.Withdraw(-5)\n}\n\n',
                                          '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),
"M2": ("Near miss: float arguments that keep Requires are accepted.",
       {"api/billing/billing.go": billing('func half(f float64) float64 {\n\tcontract.Requires(f/2 > 0.4)\n\treturn f / 2\n}\n\nvar _ = half(0.9) + half(1.0)\n\n',
                                          '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),
"H4": ("A contract that calls a function with an effect (K0503).",
       {"api/billing/billing.go": billing('var count int\n\nfunc bump() bool { count++; return true }\n\nfunc charge(x int) int {\n\tcontract.Requires(@VL0503@bump() && x > 0)\n\treturn x\n}\n\nvar _ = charge\n\n',
                                          '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),
"H4b": ("Near miss: len, cap, and conversions in a contract.",
        {"api/billing/billing.go": billing('func fill(xs []int, n int) int {\n\tcontract.Requires(len(xs) <= cap(xs) && money.Money(n) >= 0)\n\treturn n\n}\n\nvar _ = fill\n\n',
                                           '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),
"C06": ("Requires len(xs) > 0 && xs[0] > 0: accepted.",
        {"api/billing/billing.go": billing('func first(xs []int) int {\n\tcontract.Requires(len(xs) > 0 && xs[0] > 0)\n\treturn xs[0]\n}\n\nvar _ = first\n\n',
                                           '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),
"C07": ("Old of a slice (K0509).",
        {"api/billing/billing.go": billing('func grow(xs []int) (out []int) {\n\tbefore := contract.Old(@VL0509@xs)\n\tdefer contract.Ensures(func() bool { return len(out) == len(before)+1 })\n\treturn append(xs, 1)\n}\n\nvar _ = grow\n\n',
                                           '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),
"C17": ("A type assertion in a contract (K0512).",
        {"api/billing/billing.go": billing('func name(v any) string {\n\tcontract.Requires(@VL0512@v.(string) != "")\n\treturn ""\n}\n\nvar _ = name\n\n',
                                           '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')}),

# A8 invariants
"C09": ("Invariant Lo < Hi: the zero value breaks it (K0506).",
        {"api/billing/billing.go": billing('// Window is a range.\ntype Window struct {\n\tLo, Hi int\n}\n\n// Invariant holds for every window.\nfunc (w Window) Invariant() bool { return @VL0506@w.Lo < w.Hi }\n\n')}),
"L3": ("An invariant against a typed constant that the zero value breaks (K0506).",
       {"api/billing/billing.go": billing('// Qty is a count.\ntype Qty int\n\n// MinQty is the smallest line.\nconst MinQty Qty = 1\n\n// Line is an order line.\ntype Line struct{ N Qty }\n\n// Invariant holds for every line.\nfunc (l Line) Invariant() bool { return @VL0506@l.N >= MinQty }\n\n')}),
"C10": ("A field of a struct with an invariant written from a function that is not its method (K0507).",
        {"api/billing/billing.go": billing('// Window is a range.\ntype Window struct {\n\tLo, Hi int\n}\n\n// Invariant holds for every window.\nfunc (w Window) Invariant() bool { return w.Lo <= w.Hi }\n\nfunc widen(w *Window) { @VL0507@w.Hi = w.Hi + 1 }\n\nvar _ = widen\n\n')}),
"C11": ("The address of a field outside the methods (K0508).",
        {"api/billing/billing.go": billing('// Window is a range.\ntype Window struct {\n\tLo, Hi int\n}\n\n// Invariant holds for every window.\nfunc (w Window) Invariant() bool { return w.Lo <= w.Hi }\n\nfunc low(w *Window) *int { return @VL0508@&w.Lo }\n\nvar _ = low\n\n')}),
}

ACCOUNT = ('// Counter counts down.\ntype Counter struct{ N int }\n\n// Dec counts down.\nfunc (c *Counter) Dec() { c.N-- }\n\n'
           '// Account holds money.\ntype Account struct {\n\tBalance int\n\tC       Counter\n}\n\n'
           '// Invariant holds after every change.\nfunc (a Account) Invariant() bool { return a.Balance >= 0 && a.C.N >= 0 }\n\n')
HOLDER = '// Holder embeds an account.\ntype Holder struct{ Account }\n\n'
CASES["M1a"] = ("A write through a promoted field (K0507).",
    {"api/billing/billing.go": billing(ACCOUNT + HOLDER, body='var h Holder\n\t@VL0507@h.Balance = -7\n\treturn money.Receipt{Amount: amt}, nil')})
CASES["M1b"] = ("A pointer method called on a field (K0508).",
    {"api/billing/billing.go": billing(ACCOUNT + HOLDER, body='var a Account\n\t@VL0508@a.C.Dec()\n\treturn money.Receipt{Amount: amt}, nil')})
CASES["M1c"] = ("A closure that a method returns writes a field (K0507).",
    {"api/billing/billing.go": billing(ACCOUNT + '// Drainer returns a function that breaks the account.\nfunc (a *Account) Drainer() func() {\n\tdefer contract.Keep(a)\n\treturn func() { @VL0507@a.Balance = -100 }\n}\n\n' + HOLDER,
                                       '(\n\t"shop/lib/contract"\n\t"shop/lib/money"\n)')})

MARK = re.compile(r'@(VL\d{4})@')

def write(name, desc, files):
    want, out = [], {}
    for path, text in sorted(files.items()):
        clean = []
        for ln, line in enumerate(text.split('\n'), 1):
            while True:
                m = MARK.search(line)
                if not m:
                    break
                col = m.start() + 1
                want.append('error %s:%d:%d %s' % (path, ln, col, m.group(1)))
                line = line[:m.start()] + line[m.end():]
            clean.append(line)
        out[path] = '\n'.join(clean)
    body = desc + '\n-- want.txt --\n' + ('\n'.join(sorted(want)) if want else 'accept') + '\n'
    for path, text in sorted(out.items()):
        body += '-- %s --\n%s' % (path, text if text.endswith('\n') else text + '\n')
    with open(os.path.join(OUT, name + '.txtar'), 'w', newline='\n') as f:
        f.write(body)

os.makedirs(OUT, exist_ok=True)
write('base', 'The base system of the cases, in Go. Each case replaces some of its files.', BASE)
for name, (desc, files) in sorted(CASES.items()):
    write(name, desc, files)
write('accept-base', 'The Go form of the base system is accepted.', {})
