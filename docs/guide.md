# Keel primer (`keel guide`)

Keel is Go with an enforced architecture, explicit errors, nil proofs, and contracts. If you know Go,
write Go. This primer lists only what differs. Budget: 4,000 tokens.

## 1. Rule of thumb

Keel may reject Go code. It never accepts Go code and runs it differently, except the cases in §6.
When the compiler rejects code, read the hint: it shows the fix.

## 2. Project layout

```
system.keel                 architecture: containers, wires, config
api/orders/component.keel   boundary of the leaf Orders
api/orders/*.keel           code of Orders (Go syntax)
lib/money/library.keel      shared types and pure code
transports/kafka/           a transport
```

- A **leaf** is a directory of code with one `component.keel`. A leaf uses another leaf only
  through a port it needs, a topic, or a library. It never imports another leaf.
- A **container** is one process. Most systems have one container.
- A **library** holds shared types and code. It uses resources that callers pass in, but it cannot
  call an `acquires` function (open a file, open a database, create a client); only leaves can.
  Its package-level `var` is set once, at startup.
- Only a **foreign library** imports Go packages. A foreign library that depends on `net` or
  `os/exec` is a network library; only a **gateway** leaf (`gateway for Stripe`) or an **entry**
  leaf (`entry for Customer`) imports one.

## 3. Boundaries and wiring

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
- In code, call the port by its name: `r, err := Checkout.Pay(c, amt)`.
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
- Between containers only topics and `needs remote port` cross. Inside a container, a port call is
  an ordinary function call.

## 4. Errors

- `(T, error)` returns a value **or** an error. Write Go: `return v, nil`, `return Receipt{}, err`.
- An unused error is a compile error. Handle it, or write `ignore err, "reason"` or
  `ignore f.Close(), "reason"`.
- Postfix `?` returns the error: `r := Checkout.Pay(c, amt)?`.
- A function whose value and error both matter (like `io.Reader`) declares `partial`.
- Sealed errors: `type PayError sealed interface { Declined | Expired }`. A type switch on it must
  cover each case and `nil`.
- No exceptions. `panic(v)` and a broken rule **abort** the task. Only the task's scope sees it.

## 5. Nil

`nil` and zero values are Go's. The compiler must prove a value is not `nil` before you
dereference it, call a method on an interface, call a function value, or close a channel.

Proofs: an `if x != nil` check or early return; `&T{}`, `&x`, `new`, `make`, a function literal;
a result marked `nonnil`; a pointer, interface, function, or channel parameter (callers prove it);
a port name.

- A field does not narrow. Copy it to a local first: `u := s.user; if u != nil { ... }`.
- Slices, maps, and `error` behave as in Go. A write to a `nil` map aborts, as Go panics.
- `?T` means "a T or absent" for values that cannot be `nil`: `func Find(id ID) ?User`.
- `maps.Lookup(m, k)` returns `?V`.

## 6. Numbers and bounds

- Unsigned arithmetic wraps, as in Go. **Signed** `+ - * / <<` abort on overflow. Use unsigned
  types or `wrapping.Add` for hashes. `checked.Add` returns `?T`.
- Conversions truncate, as in Go.
- An index needs a proof: `for i := range xs`, `for i := 0; i < len(xs); i++`, or
  `if 0 <= i && i < len(xs)`. Otherwise you get a warning and a runtime check.

Differences from Go at runtime: signed overflow aborts; an abort that unwinds into a function that
holds a lock (`Lock` without `Unlock` yet, or `holds mu`) stops the process before that function's
deferred calls.

## 7. Contracts

```go
func Withdraw(a *Account, amt Money) (Money, error)
    requires amt > 0
    ensures  a.Balance == old(a.Balance) - amt

type Range struct{ Lo, Hi int }
    invariant Lo <= Hi
```

- `requires` is checked at entry, `ensures` on success (`result` is the value), `invariant` after
  the outermost method.
- Contracts run in every build. They may call only `pure` functions. No I/O, no mutation.
- An invariant must hold for the zero value.
- Only a struct's own methods write the fields of a struct with an invariant.

## 8. Concurrency

```go
tasks ctx := parent {
    for _, it := range items {
        go process(ctx, it)
    }
}?                  // waits; returns all task errors, joined
```

- `go` works only inside `tasks { }`. `detach f(), "reason"` starts a background task.
- A failing task cancels `ctx`. `t := go f()` gives a task value; `t.Wait()` gives
  `Completed`, `Failed`, or `Aborted`.
- Mark shared fields: `m map[string]int guarded by mu`. A helper called with the lock held says
  `holds mu`.
- A leaf starts work in `startup func(ctx context.Context) error`. `main` is generated.

## 9. Topics and remote calls

```go
Placed.Publish(OrderPlaced{ID: id})

idempotent handle Shipped(m ShipmentSent) error {
    return orders.MarkShipped(m.OrderID)
}                   // acknowledged when it returns nil
```

- A handler runs one message at a time unless the wire sets `concurrency`.
- A failed message goes to the wire's `deadletters` route, a subscribed `DeadLetter[T]` topic.
- `direct` needs bounded message types: `Note string limit 4096`, `Lines []Line limit 100`.
- Message types are locked in `topics.lock`. Adding or removing a `?T` field is compatible. A
  rename, a type change, a required field, or an invariant change needs a new type (`OrderPlacedV2`).
- `needs remote port Stock { func Level(sku SKU) (int, error) timeout 200 * time.Millisecond }`.
  Handle `Unavailable`, `Timeout`, and `RemoteAbort`; for a sealed error `E` you get `Remote[E]`.
  The runtime never retries a call.

## 10. Tests

```go
fake CheckoutOK for Checkout { func Pay(c CustomerID, a Money) (Receipt, error) { return Receipt{Amount: a}, nil } }

test "pays for an order" {
    bind { Checkout: fake CheckoutOK }
    ...
}
```

## 11. Commands

`keel check`, `keel build`, `keel test <leaf>`, `keel context <leaf>` (what to read before an edit),
`keel query <symbol>` (never guess an API), `keel explain <code>`, `keel fake <Port>`, `keel audit`
(escape hatches and unproved claims), `keel archdiff`, `keel diagram`, `keel doc`, `keel trace <id>`.
