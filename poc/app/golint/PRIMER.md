# Primer: the architecture and the contracts that `vulcan-lint` checks

This project is plain Go with an enforced architecture and contracts. **Function bodies are plain
Go**: write them exactly as you would without the linter. This primer lists only what the rules
add. `vulcan-lint ./...` checks the rules below; each diagnostic has a code (`VL0101`) and says
how to fix it. It honors no `//nolint` or other suppression comment, and rejects a file that the
build ignores (a build tag).

## 1. Project layout and the manifest

```
go.mod                   a Go module
arch/                    the manifest (arch.go), the check configuration (config.go), the kinds
cmd/shop/main.go         main: wires the leaves
api/orders/ports.go      the ports of the leaf orders
api/orders/*.go          code of orders; tests in *_test.go
lib/money/               a library: shared types and reusable code
lib/contract/            the contract checks
```

`arch/arch.go` lists every package of the module with its kind. It is the architecture.
`arch/config.go` holds the check configuration, and `arch/schema.go` declares the kinds.

```go
const (
    Customer Person   = "Customer"   // someone who uses the system
    Stripe   External = "Stripe"     // a system that it uses
)

var Manifest = System{
    Parts: []Part{
        Main("cmd/shop"),              // wires the leaves
        Entry("api/web", Customer),    // a leaf that serves a person or an external
        Leaf("api/orders"),            // a leaf
        Gateway("api/cards", Stripe),  // the one leaf that talks to Stripe
        Library("lib/money"),          // shared types and code, in lib/
    },
}

var Config = Settings{Strict: true} // arch/config.go
```

- Each package is one part: a leaf is one package, with no package inside it. A package is
  named after its directory; tests are in the same package (no `x_test` package).
- A **leaf** never imports another leaf. It uses another leaf only through a port. A library
  imports no leaf. No code imports `arch`.
- Only a **gateway** or an **entry** leaf uses the network: dial, listen, send an HTTP request,
  `sql.Open`, `exec.Command`, `http.DefaultClient`, building an `http.Client`. Main does not.
  One gateway leaf for each external system. An entry is for a declared `Person` or `External`.
- A **library** has no ports. A package-level `var` in it is set once, in its initializer or an
  `init` of the library; nothing assigns it later, also not a field or an element of it. It never
  chooses a file or a resource by name (`os.Open`, `os.ReadFile`); it uses a handle that its
  caller gives it.
- Import any Go package. No `//go:linkname`.

## 2. Leaves, ports, constructors, and main

A leaf declares what it needs as **ports**: exported interfaces in its `ports.go`. The consumer
owns the port; the provider never names the consumer.

```go
// api/orders/ports.go
package orders

// Checkout takes payments.
type Checkout interface {
    Pay(c money.CustomerID, amt money.Money) (money.Receipt, error)
}

// checkedCheckout checks the contract of Checkout at each call.
type checkedCheckout struct{ Checkout }

func (c checkedCheckout) Pay(cu money.CustomerID, amt money.Money) (money.Receipt, error) {
    contract.Requires(amt > 0)
    return c.Checkout.Pay(cu, amt)
}
```

- `ports.go` holds only the ports and their contract wrappers. A wrapper of port `X` is
  `type checkedX struct{ X }`; each of its methods is the port contract (section 3), then
  `return c.X.Method(its parameters, in order)`. A port contract names only its parameters,
  results, old values, the standard library, and libraries; never a name of its own leaf.
- Any type with the port's methods satisfies it, as a Go interface does. In code, call the port
  through its field: `r, err := s.checkout.Pay(c, amt)`. It is an ordinary call; the wrapper
  checks the port's contract at each call.
- Port methods are exported. A type in a port signature is predeclared, from the standard
  library, or from a library; never a type of a leaf or of another module.
- A **constructor** is an exported function that returns `T` or `(T, error)`, where `T` is an
  exported type of the leaf (or a pointer to one). It takes the leaf's ports and, last, at most
  one settings struct of the leaf (exported fields of basic types; tag a secret field
  `vulcan:"secret"`). It stores each port in a field; a port with a wrapper is stored wrapped:

```go
func NewService(checkout Checkout, s Settings) *Service {
    return &Service{checkout: checkedCheckout{checkout}, limit: s.Limit}
}
```

- Every port is a parameter of a constructor. Only a constructor sets a port field; nothing takes
  its address. A leaf has no package-level variable that holds a port, and no `init`.
- Other interfaces of a leaf (outside `ports.go`) are fine, but a constructor never takes one.
- **Main** only wires. It calls the standard library, its own functions, the constructors of the
  leaves, and their lifecycle methods (`func (x *T) Start(ctx context.Context[, s Settings])
  error`); nothing else of the module. It builds no leaf value with a literal or `new`, and passes
  no `nil` port. Each settings value is a literal in the call with every field
  (`store.Settings{Path: db}`), whose values are constants or flag values (a local variable, or
  `flags["db"]`); a secret field gets a flag, never a constant. A wire is an argument:
  `orders.NewService(cards, orders.Settings{Limit: 500})`.
- One value shared by several leaves is one constructor call whose result main passes to each,
  so one database is one `store.Open(...)`. In strict mode main calls each constructor once: a
  second call is a second instance.
- **Strict mode** (`Strict: true`, on in this project) adds, outside main: a port method takes
  and returns no function, channel, or interface type except `error` and `context.Context`; no
  `unsafe`, `syscall`, cgo, or `reflect` calls; an entry leaf only serves (listens), and only a
  gateway dials; a library `var` holds no pointer, slice, map, channel, function, or interface;
  no code changes process-wide state (`os.Setenv`, `http.HandleFunc`, `log.SetOutput`,
  `flag.Parse`). Main may do these. `StrictExempt: []string{"api/web"}` keeps a leaf in ordinary
  mode.

## 3. Contracts

`stockroom/lib/contract` checks contracts when the code runs. A failed check panics with a
`*contract.Failure` that names the function and the line. It undoes nothing: check before you
commit (`defer tx.Rollback()`, checks, then `return tx.Commit()`). The contract statements are **the first statements** of a function, before any other:

```go
func (a *Account) Withdraw(amt int) (left int, err error) {
    contract.Requires(amt > 0)
    before := contract.Old(a.Balance)
    defer contract.Keep(a)
    defer contract.Ensures(func() bool { return err != nil || left == before-amt })
    ...                 // plain Go
}
```

- `contract.Requires(cond)` is checked at entry.
- `defer contract.Ensures(func() bool { return cond })` is checked when the function returns,
  after the deferred calls written below it. Its literal is one `return`. Name the results to
  use them; write "err == nil implies X" as `err != nil || X`.
- `x := contract.Old(expr)` keeps a value at entry. Its type holds no reference:
  `contract.Old(len(xs))`, not `contract.Old(xs)`.
- A condition has no effects: comparisons, arithmetic, `&&`, `||`, `!`, fields, indexes, `len`,
  `cap`, and conversions. It calls no other function and has no type assertion or function
  literal. Guard its own index on its left: `len(xs) > 0 && xs[0] > 0`.
- A call whose constant arguments make a `Requires` false is rejected by the check.
- An **invariant** is a method `func (x T) Invariant() bool { return cond }` of a struct type,
  with a value receiver and one pure `return`. It must hold for the zero value. Each method of
  `T` with a pointer receiver starts with `defer contract.Keep(recv)`, which checks the invariant
  when the method returns, so a method can pass through a broken state. Each value of `T` that
  the code builds with fields or by a conversion is checked where it is built:
  `contract.Valid(T{Lo: 1, Hi: 2})` or `contract.Valid(&T{...})`; an empty `T{}` needs none.
- Only the methods of `T` write its fields or take a field's address (`&t.Lo`), also through a
  field of another struct; elsewhere, call a method or build a new value with `contract.Valid`.

## 4. Tests

Tests are Go tests in `*_test.go`, in the leaf's package. A fake is any type that has the
port's methods. Build the value with its constructor and the fakes:

```go
type checkoutOK struct{}

func (checkoutOK) Pay(c money.CustomerID, a money.Money) (money.Receipt, error) {
    return money.Receipt{Amount: a}, nil
}

func TestPlace(t *testing.T) {
    s := NewService(checkoutOK{}, Settings{Limit: 500})
    if err := s.Place(o); err != nil {
        t.Fatal(err)
    }
}
```

A fake must meet the port's contract: the wrapper checks it at each call.

## 5. Commands

- `go vet ./... && vulcan-lint ./... && go test ./...` is the check. Each diagnostic of
  `vulcan-lint` has a code (`VL0101`), a position, and the fix.
- `go build`, `go run ./cmd/stockroom [--flag=value]`, `go test ./api/orders`, and
  `go mod tidy` work as usual.
