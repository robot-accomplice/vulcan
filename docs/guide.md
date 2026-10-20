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
  at startup, and it never chooses a file or a network target (`os.Open`, `sql.Open`, `http.Get`):
  only leaves do. It can use a handle that its caller gives it (an `io.Reader`, a `*sql.DB`).
- Every code file is a Go file: `package orders`, imports, code. Import a library by its Go path
  (`import "shop/lib/money"`) and use its package name (`money.Money`).
- Import any Go package. Only a **gateway** leaf (`gateway for Stripe`) or an **entry** leaf
  (`entry for Customer`) can reach a network function: dial, listen, send an HTTP request,
  `sql.Open`, `exec.Command`, also inside an SDK. The diagnostic shows the call path.
- Never write `//go:linkname` in a leaf or a library. `keel audit` lists each path around the
  architecture rules (callbacks through ports, shared library state). `config { strict: true }`
  rejects most of them: then port methods take no function, channel, or interface types except
  `error` and `context.Context`, library `var`s hold no references, and only gateways dial.
- `main` is generated. It reads flags, calls constructors and binds ports, runs every
  `func setup(ctx context.Context) error`, starts subscriptions, then runs every
  `func startup(ctx context.Context) error` at once (plain Go functions; tests can call them). Open
  a provider's resources (a database) in its constructor, and a subscriber's in `setup`. A leaf
  cannot declare `init`, and code never assigns a port name. On a stop signal, `main` cancels the
  startup `ctx`, waits for running work (`shutdown_timeout`, 30 s), then exits.
- Leaf names are unique in the system, also across containers.

## 2. Boundaries and wiring

```go
// api/orders/component.keel
import ("shop/lib/money"; "shop/lib/events")

component Orders {
    needs port Checkout {
        func Pay(c money.CustomerID, amt money.Money) (money.Receipt, error)
            requires amt > 0
    }
    publishes  Placed  topic[events.OrderPlaced]
    subscribes Shipped topic[events.ShipmentSent]
}
```

- The consumer declares the port. Any exported type with matching methods satisfies it, as a Go
  interface does. The provider never names the consumer.
- In code, call the port by its name: `r, err := Checkout.Pay(c, amt)`. It is an ordinary call.
- Every type that crosses a boundary (port, topic, remote call) is a Go built-in or standard type,
  or comes from a library. Topic and remote types must come from a library.

```go
// system.keel
system Shop {
    container Api {
        Web{ Addr: flag("web-addr", default: ":8080") }, Orders, Billing, StripeGateway
        Orders.Checkout:     Billing.NewPayer
        Billing.CardNetwork: StripeGateway.NewCards{ Key: flag("stripe-key") }
    }
    container Worker {
        Alerts
        group Fulfilment { Picking }       // groups are diagram labels only
    }
    person   Customer "Buys goods"
    external Stripe   "Card payments"
    Worker/Picking.Placed: Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }
    Api/Orders.Shipped: Worker/Picking.Shipped via direct{ deadletters: Worker/Alerts.DeadShipped }
    config { context_budget: 400 }
}
```

- Every wire reads `needer: supplier`. Every needed port and subscribed topic is wired once.
- A wire names a constructor, which can return `(T, error)` and take one settings struct, filled
  at the wire. A wire that names a type `T` provides `new(T)`. An entry leaf gets settings in the
  container list, as the second parameter of `startup(ctx, s Settings)`.
- `flag("x")` reads `--x`, else the variable `X`. Without `default:` it is required. A size needs
  `max:`: `capacity: flag("orders-cap", max: 10_000)`. A `secret` setting accepts only a flag.
- Between containers only topics and `needs remote port` cross.

## 3. Contracts

```go
func Withdraw(a *Account, amt Money) (Money, error)
    requires a != nil && amt > 0
    ensures  err == nil => a.Balance == old(a.Balance) - amt
    ensures  err != nil => a.Balance == old(a.Balance)
{
    ...                 // plain Go
}

type Range struct{ Lo, Hi int }
    invariant Lo <= Hi

pure func Positive(x int) bool { return x > 0 }   // contracts call only pure functions
```

- `requires` is checked at entry, `ensures` at every return, `invariant` after each method call
  except a call through the method's own receiver. `err` names the error result and `result` the
  first other one (`result.0`, `result.1` for several); use `result` only after `err == nil =>`.
  A function with a contract cannot name a parameter or a result `err` or `result`.
- A contract must prove its own index, dereference, and divisor safe, from the conditions on its
  left, or from a `requires` conjunct that reads only parameters the body never assigns (not their
  fields): `len(xs) > 0 && xs[0] > 0`, `requires a != nil`. `old(x)` needs a type with no
  references: `old(len(xs))`.
- A `pure` function has no effects: it writes only values that it created, and calls only `pure`
  functions. Its body is plain Go. Use `keel.All(xs, Positive)` for a rule over a collection.
- Contracts run in every build. A failed contract panics with a report. It undoes nothing:
  check before you commit (`defer tx.Rollback()`, checks, then `return tx.Commit()`), and store an
  idempotency key with each retryable change (`keel.DeliveryID(ctx)` in a handler).
- Deferred calls and `recover` cannot hide a broken contract: when a deferred call can change a
  result or recover a panic, `ensures` and the invariant are checked again after it. A deferred
  call in a method cannot write the struct's fields. A panic that leaves a method with its
  invariant false becomes a contract failure.
- An invariant must hold for the zero value. Only a struct's own methods write its fields or take
  a field's address (`&e.Amount`): elsewhere, read into locals and build a literal.

## 4. Topics and remote calls

```go
Placed.Publish(events.OrderPlaced{ID: id})   // returns nothing

idempotent handle Shipped(ctx context.Context, m events.ShipmentSent) error {
    return orders.MarkShipped(m.OrderID)
}                   // acknowledged when it returns nil; ctx is optional
```

- An error or a panic in a handler means the message is not acknowledged; the transport retries,
  then sends it to the wire's `deadletters` route: a topic declared
  `subscribes DeadPlaced topic[keel.DeadLetter[events.OrderPlaced]]`, with fields `Message`,
  `Error`, `Attempts`, `Wire`, `Time`. A message that fails its invariant goes there at once.
- `direct` delivers at least once, so every handler on it, including a dead-letter handler, is
  `idempotent`. It promises no order.
- A handler runs one message at a time unless the wire sets `concurrency`.
- `direct` needs every variable-size message field bounded with `limit`, one bound per
  variable-size part: `ID OrderID limit 64`, `Note string limit MaxNote`, `Lines []Line limit 100`,
  `Tags []string limit 10, 64`. A bound counts bytes for a string, elements for a slice or a map. A
  publish that breaks a limit panics.
- A message type is a struct from a library. It holds only `bool`, numbers, `string`, `[]byte`,
  `time.Time`, `time.Duration`, arrays, slices, maps with integer, `bool`, or `string` keys, and
  such structs: no pointers (use a zero value or a separate field for "absent"), no interfaces.
- Message types are locked in `topics.lock`. Adding or removing a field is compatible if the zero
  value passes the invariant. A rename, a type change, a `limit` change, or an invariant change
  needs a new type.
- `needs remote port Stock { func Level(sku stock.SKU) (int, error) timeout 200 * time.Millisecond }`.
  The last result is `error`; it can also be `keel.Unavailable`, `keel.Timeout`, or
  `keel.RemotePanic`. The runtime never retries a call. An error crosses as text; declare it with
  `keel.NewError("stock.unknown-sku", "unknown SKU")` so that `errors.Is` matches on the caller.

## 5. Tests

Tests are Go tests in `*_test.keel`. The runtime package is `import "keel.dev/keel"`. A fake is any type that satisfies the port.

```go
type CheckoutOK struct{}
func (CheckoutOK) Pay(c money.CustomerID, a money.Money) (money.Receipt, error) { return money.Receipt{Amount: a}, nil }

func TestPlace(t *testing.T) {
    keel.Bind(t, &Checkout, CheckoutOK{})          // this test only; not with t.Parallel
    if err := Place(o); err != nil { t.Fatal(err) }
    if got := keel.Published(t, Placed); len(got) != 1 { t.Fatalf("got %d", len(got)) }
    if err := keel.Deliver(t, Shipped, events.ShipmentSent{OrderID: "o-1"}); err != nil { t.Fatal(err) }
}
```

A fake must meet the port's contract. `keel test` runs the race detector; a race fails the test.

## 6. Commands

`keel mod tidy` (not `go mod tidy`, which cannot see `.keel` imports), `keel check`, `keel build`,
`keel test <leaf>`, `keel context <leaf>` (what to read before an edit),
`keel query <symbol>` (never guess an API), `keel explain <code>`, `keel fake <Port>`, `keel audit`
(warnings on Go code, escape hatches, unproved claims), `keel archdiff`, `keel diagram`,
`keel doc`, `keel trace <id>`.
