# Vulcan primer (`vulcan guide`, proof of concept)

Vulcan is Go with an enforced architecture and contracts. **Function bodies are plain Go**: write
them exactly as you would in Go. This primer lists only what Vulcan adds, and only what this
release accepts. It is the release-1 primer of `docs/guide.md`, cut to the proof of concept.

## 1. Project layout

```
go.mod                        a Go module
system.vulcan                 architecture: the container, its leaves, wires, config
api/orders/component.vulcan   boundary of the leaf Orders (container Api, leaf Orders)
api/orders/*.vulcan           code of Orders (plain Go); tests in *_test.vulcan
lib/money/library.vulcan      a library: shared types and reusable code
```

- Code files end in `.vulcan`, never `.go`. Each is a Go file: `package orders` (the directory
  name), imports, code. `testdata/` and other files next to the code work as in Go.
- A **leaf** is a directory with one `component.vulcan`. A leaf never imports another leaf. It
  uses another leaf only through a port that it needs, or through a library.
- A **library** (`lib/<name>/library.vulcan` with `library money {}`) holds shared types and
  code. It has no ports. A package-level `var` in it is set once, in its initializer or `init`;
  nothing assigns it later, also not a field or an element of it. It never chooses a file or a
  network target (`os.Open`, `sql.Open`, `http.Get`); it uses a handle that its caller gives it.
- Import any Go package. Only a **gateway** leaf (`gateway for Stripe`) or an **entry** leaf
  (`entry for Customer`) can use a network function: dial, listen, send an HTTP request,
  `sql.Open`, `exec.Command`. Never write `//go:linkname`.
- **Strict mode** (`config { strict: true }`, on in this project) adds: a port method takes and
  returns no function, channel, or interface type except `error` and `context.Context` (K0123);
  no `unsafe`, `syscall`, cgo, or `reflect` calls (K0124); an entry leaf only serves (listens),
  and only a gateway dials (K0125); a library `var` holds no pointer, slice, map, channel,
  function, or interface (K0210); no code changes process-wide state such as `os.Setenv`,
  `http.HandleFunc`, `log.SetOutput`, or `flag.Parse` (K0211). `strict_exempt: ["Leaf"]` keeps
  a leaf in ordinary mode.
- `main` is generated. It reads flags, calls the constructors and binds the ports, runs every
  `func setup(ctx context.Context) error`, then runs every `func startup(ctx context.Context) error`
  at once. Plain Go functions: tests can call them. A leaf cannot declare `init`, and the
  initializer of a package-level `var` cannot reach a port. Code never assigns a port name.
- The process exits 0 when every startup function has returned `nil`, and 1 when one fails. On a
  stop signal, `main` cancels the startup `ctx` and waits for at most `shutdown_timeout` (30 s).

## 2. Boundaries and wiring

```go
// api/orders/component.vulcan
import "shop/lib/money"

component Orders {
    needs port Checkout {
        func Pay(c money.CustomerID, amt money.Money) (money.Receipt, error)
            requires amt > 0
            ensures  err == nil => result.Amount == amt
    }
}
```

- The consumer declares the port. Any exported type with matching methods satisfies it, as a Go
  interface does. The provider never names the consumer. Port methods are exported.
- In code, call the port by its name: `r, err := Checkout.Pay(c, amt)`. It is an ordinary call.
  Its contract is checked at each call.
- A type in a port signature is a Go built-in or standard type, or comes from a library; never a
  type of a leaf.
- A port's name must differ from every package-level name of its leaf.

```go
// system.vulcan
import "shop/lib/money"

system Shop {
    person   Customer "Buys goods"
    external Stripe   "Card payments"

    config { shutdown_timeout: 10 * time.Second }

    container Api {
        Web{ Addr: flag("web-addr", default: ":8080") }, Orders, Billing, StripeGateway
        group Back { Ledger }                 // a label only

        Web.Orders:          Orders.NewService
        Orders.Checkout:     Billing.NewPayer{ Limit: money.MaxOrder }
        Billing.CardNetwork: StripeGateway.NewCards{ Key: flag("stripe-key") }
        Billing.Store:       Ledger.Memory     // a type: provides new(Memory)
    }
}
```

- Every wire reads `needer: supplier`. Every needed port is wired exactly once. The wires have no
  cycle. One gateway leaf for each external system.
- Wires that name the same constructor with the same settings share one instance; other
  settings make a second instance (K0607 warns). So one database is one `Store.Open{...}`, written
  the same on every wire.
- A wire names an exported constructor or type of the supplier leaf. A constructor takes nothing
  or one exported settings struct, and returns `T` or `(T, error)`; `T` is an exported concrete
  type or a pointer to one. Pointer receivers need a pointer: return `*Payer`.
- A leaf's own settings are in the container list, as the second parameter of
  `setup(ctx, s Settings)` or `startup(ctx, s Settings)`.
- A setting is a Go **constant** of its field's type (`5 * time.Second`, `"eu"`, `money.MaxOrder`)
  or a flag. `system.vulcan` holds no code: no function literals, no calls, no variables. Every
  exported field of a settings struct gets a value.
- `flag("x")` reads `--x`, else the variable `X`. Without `default:` it is required; a default is
  a constant of the field's type. A flag fills a string, int, int64, float64, bool, or
  `time.Duration` field. A field tagged `vulcan:"secret"` takes a flag with no default.
- `system.vulcan` imports only libraries; `time` needs no import.

## 3. Contracts

```go
type Account struct{ Balance int }
    invariant Balance >= 0

func (a *Account) Withdraw(amt int) (int, error)
    requires amt > 0
    ensures  err == nil => result == old(a.Balance) - amt
    ensures  err != nil => a.Balance == old(a.Balance)
{
    ...                 // plain Go
}
```

- `requires` is checked at entry. `ensures` and the invariant are checked at each `return`,
  before the deferred calls run (so a deferred `Unlock` comes after the check). `=>` means
  "implies".
- `err` names the error result and `result` the first other one (`result.0`, `result.1` for
  several). Use `result` only after `err == nil =>`. A function with a contract cannot name a
  parameter or a result `err` or `result`. Named results can be used by name in `ensures`.
- `old(x)` is the value of `x` at entry. Its type holds no reference: `old(len(xs))`, not
  `old(xs)`.
- A contract is a Go expression with no effects: comparisons, arithmetic, `&&`, `||`, `!`, `=>`,
  fields, indexes, `len`, `cap`, and conversions. It calls no other function (K0503) and has no
  type assertion. Guard its own index and dereference on its left: `len(xs) > 0 && xs[0] > 0`,
  `a != nil => a.Balance >= 0`. Put a composite literal in parentheses, as in an `if`:
  `ensures err == nil => result == (money.Receipt{Amount: amt})`.
- A port contract names only its parameters, `err`, `result`, `old`, the standard library, and
  libraries; never a name of its own leaf (K0120).
- A failed contract panics with a report of the clause and its values. It undoes nothing: check
  before you commit (`defer tx.Rollback()`, checks, then `return tx.Commit()`).
  `config { on_contract_failure: exit }` makes a failure exit the process with code 3.
- An invariant is checked when each method call on a pointer receiver returns, and on each value
  of the type that the code builds (a literal, also inside `[]T{{...}}`, and a conversion), except
  a call through the method's own receiver (`a.Withdraw(x)` inside a method of `a`), so a method
  can pass through a broken state. It must hold for the zero value.
- Only the struct's own methods write its fields or take a field's address (`&e.Amount`);
  elsewhere, call a method or build a new literal.

## 4. Tests

Tests are Go tests in `*_test.vulcan`, in the leaf's package. Import the runtime package
`github.com/robot-accomplice/vulcan/vulcan`. A fake is any type that satisfies the port.

```go
type CheckoutOK struct{}

func (CheckoutOK) Pay(c money.CustomerID, a money.Money) (money.Receipt, error) {
    return money.Receipt{Amount: a}, nil
}

func TestPlace(t *testing.T) {
    vulcan.Bind(t, &Checkout, CheckoutOK{})   // this test only; not with t.Parallel
    if err := Place(o); err != nil {
        t.Fatal(err)
    }
}
```

A fake must meet the port's contract: it is checked at each call. A port that a test does not
bind panics with "port X is not bound".

## 5. Commands

- `vulcan check` checks the architecture, the contracts, and the Go types, and runs `go vet`. `--json` prints the
  diagnostics as JSON. Each diagnostic has a code (`K0101`) and a `.vulcan` position.
- `vulcan build`, `vulcan run [-- --flag=value]`, `vulcan test [Leaf]`. The `go` commands do not
  work in the project (its code is in `.vulcan` files); `.vulcan/build` is generated, so never
  edit it.
- `vulcan mod tidy` adds the modules that the code imports to `go.mod` and `go.sum`. Use it, not
  `go mod tidy`, which cannot see `.vulcan` files.
- `vulcan guide` prints this primer.

Not in this release (K0009): topics, remote ports, transports, a second container, `pure`
functions, `limit`, `handle`, and `vulcan audit`, `context`, `query`, `explain`,
`fake`, `fmt`, `diagram`, `archdiff`, `doc`, `trace`.
