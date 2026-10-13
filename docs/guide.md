# Keel primer (`keel guide`)

Keel is Go with an enforced architecture and contracts. **Function bodies are plain Go**: write
them exactly as you would in Go. This primer lists only what Keel adds. Budget: 4,000 tokens.

## 1. Project layout

```
system.keel                 architecture: containers, wires, config
api/orders/component.keel   boundary of the leaf Orders
api/orders/*.keel           code of Orders (plain Go)
lib/money/library.keel      shared types and reusable code
transports/kafka/           a transport
```

- A **leaf** is a directory of code with one `component.keel`. A leaf never imports another leaf.
  It uses another leaf only through a port it needs, a topic, or a library.
- A **container** is one process. Most systems have one container.
- A **library** is a Go package that holds shared types and code: a `library.keel` with
  `library money {}`, plus Go files. It has no ports or topics, its package-level `var` is set once
  at startup, and it never acquires a resource (`os.Open`, `sql.Open`, a client): only leaves do.
- Every code file is a Go file: `package orders`, imports, code. Import a library by its Go path
  (`import "shop/lib/money"`) and use its package name (`money.Money`).
- Go packages are imported as in Go. A package that depends on `net`, `os/exec`, or `database/sql`
  can be imported
  only by a **gateway** leaf (`gateway for Stripe`) or an **entry** leaf (`entry for Customer`).
- `main` is generated. A leaf's code runs only from a handler, a port that it provides, an entry
  leaf's server, or a plain Go function `func startup(ctx context.Context) error` (tests can call
  it). `main` reads flags, calls constructors, binds ports, then runs every `startup` at once. So
  open a provider's resources (a database) in its constructor, not in `startup`. A container with
  no subscription, served remote port, or entry leaf exits after its startup functions finish.

## 2. Boundaries and wiring

```go
// api/orders/component.keel
import "shop/lib/money"

component Orders {
    needs port Checkout {
        func Pay(c money.CustomerID, amt money.Money) (money.Receipt, error)
            requires amt > 0
    }
    publishes  Placed  topic[OrderPlaced]
    subscribes Shipped topic[ShipmentSent]
}
```

- The consumer declares the port. Any exported type with matching methods satisfies it, as a Go
  interface does. The provider never names the consumer.
- In code, call the port by its name: `r, err := Checkout.Pay(c, amt)`. It is an ordinary call.
- Types in a port signature come from a library, imported at the top of `component.keel`.

```go
// system.keel
system Shop {
    container Api {
        Web, Orders, Billing
        Orders.Checkout: Billing.NewPayer
    }
    container Worker {
        Alerts
        group Fulfilment { Picking }       // groups are diagram labels only
    }
    person   Customer "Buys goods"
    external Stripe   "Card payments"
    Worker/Picking.Placed: Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }
    config { context_budget: 400 }
}
```

- Every wire reads `needer: supplier`. Every needed port and subscribed topic is wired once.
- A wire can fill its constructor's settings struct:
  `Billing.Cards: Stripe.NewCards{ Key: flag("stripe-key") }`. A constructor can return
  `(T, error)`. A leaf with no constructor (an entry leaf) gets settings in the container list,
  `Web{ Addr: flag("web-addr", default: ":8080") }`, as the second parameter of
  `startup(ctx, s Settings)`.
- `flag("x")` reads `--x`, else the variable `X`. Without `default:` it is required. A size needs
  `max:`: `capacity: flag("orders-cap", max: 10_000)`.
- Between containers only topics and `needs remote port` cross.

## 3. Contracts

```go
func Withdraw(a *Account, amt Money) (Money, error)
    requires amt > 0
    ensures  err == nil => a.Balance == old(a.Balance) - amt
    ensures  err != nil => a.Balance == old(a.Balance)
{
    ...                 // plain Go
}

type Range struct{ Lo, Hi int }
    invariant Lo <= Hi
```

- `requires` is checked at entry, `ensures` at every return, `invariant` after the outermost
  method. `err` names the error result and `result` the first other one; use `result` only after
  `err == nil =>`. A failed contract panics with a report.
- Contracts run in every build. They may call only `pure` functions: no I/O, no mutation.
- An invariant must hold for the zero value. Only a struct's own methods write its fields.

## 4. Topics and remote calls

```go
Placed.Publish(OrderPlaced{ID: id})   // returns nothing

idempotent handle Shipped(m ShipmentSent) error {
    return orders.MarkShipped(m.OrderID)
}                   // acknowledged when it returns nil
```

- An error or a panic in a handler means the message is not acknowledged; the transport retries,
  then sends it to the wire's `deadletters` route: a topic declared
  `subscribes DeadPlaced topic[keel.DeadLetter[OrderPlaced]]`, with fields `Message`, `Error`,
  `Attempts`, `Wire`, `Time`.
- `direct` delivers at least once, so every handler on it, including a dead-letter handler, is
  `idempotent`.
- A handler runs one message at a time unless the wire sets `concurrency`.
- `direct` needs every variable-size message field bounded with `limit`, including named string
  types: `ID OrderID limit 64`, `Note string limit 2000`, `Lines []Line limit 100`. A `limit`
  counts bytes for a string, elements for a slice or a map. It takes a constant: `limit MaxNote`.
  A publish that breaks a limit panics.
- Message types are locked in `topics.lock`. Adding or removing a field is compatible if the zero
  value passes the invariant. A rename, a type change, or an invariant change needs a new type.
- `needs remote port Stock { func Level(sku SKU) (int, error) timeout 200 * time.Millisecond }`.
  Errors can also be `Unavailable`, `Timeout`, or `RemotePanic`. The runtime never retries a call.

## 5. Tests

Tests are Go tests in `*_test.keel`. The runtime package is `import "keel.dev/keel"`. A fake is any type that satisfies the port.

```go
type CheckoutOK struct{}
func (CheckoutOK) Pay(c money.CustomerID, a money.Money) (money.Receipt, error) { return money.Receipt{Amount: a}, nil }

func TestPlace(t *testing.T) {
    keel.Bind(t, &Checkout, CheckoutOK{})          // this test only
    if err := Place(o); err != nil { t.Fatal(err) }
    if got := keel.Published(t, Placed); len(got) != 1 { t.Fatalf("got %d", len(got)) }
}
```

## 6. Commands

`keel check`, `keel build`, `keel test <leaf>`, `keel context <leaf>` (what to read before an edit),
`keel query <symbol>` (never guess an API), `keel explain <code>`, `keel fake <Port>`, `keel audit`
(warnings on Go code, escape hatches, unproved claims), `keel archdiff`, `keel diagram`,
`keel doc`, `keel trace <id>`.
