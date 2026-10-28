# Primer: the architecture and the contracts that `vulcan-lint` checks

This project is plain Go with an enforced architecture and contracts. **Function bodies are plain
Go.** `vulcan-lint ./...` checks the rules below; each diagnostic has a code (`VL0101`) and says
how to fix it. It honors no `//nolint` or other suppression comment, and rejects a file that the
build ignores (a build tag).

## 1. The manifest

`arch/arch.go` lists every package of the module with its kind. It is the architecture: change it
only when the request asks for a change to the architecture. `arch/schema.go` declares the kinds.

```go
const (
    Customer Person   = "Customer"   // someone who uses the system
    Stripe   External = "Stripe"     // a system that it uses
)

var Manifest = System{
    Strict: true,
    Parts: []Part{
        Main("cmd/shop"),              // wires the leaves
        Entry("api/web", Customer),    // a leaf that serves a person or an external
        Leaf("api/orders"),            // a leaf
        Gateway("api/cards", Stripe),  // the one leaf that talks to Stripe
        Library("lib/money"),          // shared types and code, in lib/
    },
}
```

- Each package is one part: a leaf is one package, with no package inside it. A package is
  named after its directory; tests are in the same package (no `x_test` package).
- A **leaf** never imports another leaf. It uses another leaf only through a port. A library
  imports no leaf. No code imports `arch`.
- Only a **gateway** or an **entry** leaf uses the network: dial, listen, send an HTTP request,
  `sql.Open`, `exec.Command`, `http.DefaultClient`, building an `http.Client`. Main does not.
- A **library** has no ports. A package-level `var` in it is set once, in its initializer or an
  `init` of the library; nothing assigns it later, also not a field or an element of it. It never
  chooses a file or a resource by name (`os.Open`, `os.ReadFile`); it uses a handle that its
  caller gives it.
- No `//go:linkname`.

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
- **Strict mode** (`Strict: true`, on in this project) adds, outside main: a port method takes
  and returns no function, channel, or interface type except `error` and `context.Context`; no
  `unsafe`, `syscall`, cgo, or `reflect` calls; an entry leaf only serves (listens), and only a
  gateway dials; a library `var` holds no pointer, slice, map, channel, function, or interface;
  no code changes process-wide state (`os.Setenv`, `http.HandleFunc`, `log.SetOutput`,
  `flag.Parse`). Main may do these.

## 3. Contracts

`stockroom/lib/contract` checks contracts when the code runs. A failed check panics with a
`*contract.Failure` that names the function and the line; it undoes nothing, so check before you
commit. The contract statements are **the first statements** of a function, before any other:

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
  `T` with a pointer receiver starts with `defer contract.Keep(recv)`. Each value of `T` that
  the code builds with fields or by a conversion is checked where it is built:
  `contract.Valid(T{Lo: 1, Hi: 2})` or `contract.Valid(&T{...})`; an empty `T{}` needs none.
- Only the methods of `T` write its fields or take a field's address (`&t.Lo`), also through a
  field of another struct; elsewhere, call a method or build a new value with `contract.Valid`.

## 4. Tests

Tests are Go tests in the leaf's package. Build the value with its constructor and a fake: any
type that has the port's methods. The wrapper checks the fake against the port's contract too.

## 5. Codes

| Code | What is wrong | Fix |
|---|---|---|
| VL0002 | A package not in the manifest, or inside a leaf | Put the code in a listed package |
| VL0003 | A part without its package; a library outside `lib/`; main not `package main` | Fix the part or the package |
| VL0004 | Package name is not the directory; an `x_test` package | Rename; test in the same package |
| VL0101 | A leaf imports a leaf | Declare a port; main passes the other leaf |
| VL0104 | A port that no constructor takes, or a `nil` port in main | Take it; pass a provider |
| VL0110 | Something else in `ports.go`; an unexported port or port method; the manifest imports | Keep `ports.go` to ports and wrappers |
| VL0112 | A port signature uses a type of the leaf or of another module | Move the type to a library |
| VL0116 | A port field written outside a constructor; a package-level var holds a port | Set ports in the constructor only |
| VL0117 | `init` in a leaf | Use the constructor or a lifecycle method |
| VL0118 | Code imports `arch` | Remove the import |
| VL0120 | A port contract names a name of its own leaf | Use parameters, constants of libraries |
| VL0121 | A directory listed twice in the manifest | One part per package |
| VL0122 | `//go:linkname` | Call through a port or a library |
| VL0123 | Strict: a port method carries a function, channel, or interface | Pass values |
| VL0124 | Strict: `unsafe`, `syscall`, cgo, or a `reflect` call | Remove it |
| VL0125 | Strict: an entry leaf dials | Dial in a gateway, through a port |
| VL0201 | A library has a `ports.go` | Libraries have no ports |
| VL0202 | A library var is assigned after startup | Set it once, in its initializer |
| VL0203 | A library opens a file or a resource by name | Take a handle from the caller |
| VL0204 | A library imports a leaf | Import only libraries |
| VL0205 | Network use outside a gateway or an entry leaf | Move it to a gateway, reach it by a port |
| VL0206 | A gateway or entry is for a name that is not declared | Use a declared Person or External |
| VL0207 | Two gateways for one external | One gateway per external |
| VL0210 | Strict: a library var holds a reference | Use a value type |
| VL0211 | Strict: a call that changes process-wide state, outside main | Do it in main, or pass a value |
| VL0501 | Constant arguments break the callee's `Requires` | Pass values that keep it |
| VL0503 | A contract or invariant with effects, or not of its form | Comparisons, `len`, `cap`, conversions |
| VL0506 | The zero value breaks the invariant | Make it hold for zero (`Lo <= Hi`) |
| VL0507 | A field of a type with an invariant written outside its methods | Add a method, or `contract.Valid` |
| VL0508 | The address of such a field, or a pointer method on it, outside its methods | Copy it to a local |
| VL0509 | `contract.Old` of a value that holds a reference | Old of a value: `len(xs)`, a field |
| VL0512 | A type assertion in a contract | Remove it |
| VL0513 | `Invariant` on a non-struct, or with a pointer receiver | `func (x T) Invariant() bool` |
| VL0603 | A settings literal in main lacks a field | Give every field |
| VL0604 | A constructor's parameters or result are not of the form | Ports, then one settings struct |
| VL0605 | A secret setting gets a constant | Pass a flag value |
| VL0606 | A setting that is not a constant or a flag value; settings not a literal | Compute nothing in main |
| VL0901 | The build ignores a file | Remove the build constraint |
| VL0902 | The manifest cannot be read | Keep its literal form |
| VL0903 | A contract call that is not among the first statements | Move it to the top |
| VL0904 | A pointer method of a type with an invariant without `defer contract.Keep(recv)` | Add it first |
| VL0905 | A value of a type with an invariant built without `contract.Valid` | Wrap the literal |
| VL0906 | A wrapper method that does more than check, or a port stored unwrapped | Fix its form; store `checkedX{x}` |
| VL0907 | Main calls something that is not a constructor, lifecycle method, or standard library | Main only wires |
| VL0908 | Main builds a leaf value without its constructor | Call the constructor |

## 6. Commands

`go vet ./... && vulcan-lint ./... && go test ./...` is the check. `go build`, `go run`, and
`go test` work as usual.
