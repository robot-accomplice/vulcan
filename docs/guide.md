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
- A **library** holds shared types and code. It has no ports or topics, its package-level `var` is
  set once at startup, and it never acquires a resource (`os.Open`, `sql.Open`, a client): only
  leaves do.
- Go packages are imported as in Go. A package that depends on `net` or `os/exec` can be imported
  only by a **gateway** leaf (`gateway for Stripe`) or an **entry** leaf (`entry for Customer`).
- `main` is generated. A leaf starts work in `startup func(ctx context.Context) error`.

## 2. Boundaries and wiring

```go
// api/orders/component.keel
component Orders {
    needs port Checkout {
        func Pay(c CustomerID, amt Money) (Receipt, error)
            requires amt > 0
    }
    publishes  Placed  topic[OrderPlaced]
    subscribes Shipped topic[ShipmentSent]
}
```

- The consumer declares the port. Any exported type with matching methods satisfies it, as a Go
  interface does. The provider never names the consumer.
- In code, call the port by its name: `r, err := Checkout.Pay(c, amt)`. It is an ordinary call.
- Types in a port signature come from a library.

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
- Settings can come from startup flags: `capacity: flag("orders-cap", max: 10_000)`.
- Between containers only topics and `needs remote port` cross.

## 3. Contracts

```go
func Withdraw(a *Account, amt Money) (Money, error)
    requires amt > 0
    ensures  a.Balance == old(a.Balance) - amt

type Range struct{ Lo, Hi int }
    invariant Lo <= Hi
```

- `requires` is checked at entry, `ensures` on success (`result` is the value), `invariant` after
  the outermost method. A failed contract panics with a report.
- Contracts run in every build. They may call only `pure` functions: no I/O, no mutation.
- An invariant must hold for the zero value. Only a struct's own methods write its fields.

## 4. Topics and remote calls

```go
Placed.Publish(OrderPlaced{ID: id})

idempotent handle Shipped(m ShipmentSent) error {
    return orders.MarkShipped(m.OrderID)
}                   // acknowledged when it returns nil
```

- An error or a panic in a handler means the message is not acknowledged; the transport retries,
  then sends it to the wire's `deadletters` route, a subscribed `DeadLetter[T]` topic.
- A handler runs one message at a time unless the wire sets `concurrency`.
- `direct` needs bounded message types: `Note string limit 4096`, `Lines []Line limit 100`.
- Message types are locked in `topics.lock`. Adding or removing a field is compatible if the zero
  value passes the invariant. A rename, a type change, or an invariant change needs a new type.
- `needs remote port Stock { func Level(sku SKU) (int, error) timeout 200 * time.Millisecond }`.
  Errors can also be `Unavailable`, `Timeout`, or `RemotePanic`. The runtime never retries a call.

## 5. Tests

```go
fake CheckoutOK for Checkout { func Pay(c CustomerID, a Money) (Receipt, error) { return Receipt{Amount: a}, nil } }

test "pays for an order" {
    bind { Checkout: fake CheckoutOK }
    ...
}
```

In a test, each published topic records its messages, and the test can read them.

## 6. Commands

`keel check`, `keel build`, `keel test <leaf>`, `keel context <leaf>` (what to read before an edit),
`keel query <symbol>` (never guess an API), `keel explain <code>`, `keel fake <Port>`, `keel audit`
(warnings on Go code, escape hatches, unproved claims), `keel archdiff`, `keel diagram`,
`keel doc`, `keel trace <id>`.
