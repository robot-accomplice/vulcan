# Round 2: Stockroom-lite with two more services

Plan-r2 section 4 asks for "a larger application (Stockroom-lite with two more services)". This
note records what was added, in all three arms, before any hidden rule existed. The functional
description is [docs/poc/stockroom.md](../docs/poc/stockroom.md).

## The two services, and why

1. **Purchasing.** Suppliers, and purchase orders with partial receiving into stock (rules 7 to
   10). A stockroom that sells stock must also buy it, so this is the first thing a real product
   adds. Its seams: a new leaf (`purchasing`), work that crosses parts (receiving changes stock
   through `inventory`, not through the store), a new atomic multi-line stock change
   (`Restock`), new tables, and a state machine of its own.
2. **Webhooks.** Subscriptions, events raised by orders and purchasing, and signed HTTP deliveries
   to the subscribers, with a delivery log and a retry (rules 11 to 14). Integrating with other
   systems is the usual next need of an order service. Its seams: a new leaf (`events`), a new
   external system and gateway (`webhooks`, for `Subscribers`), a secret per subscription, and
   two parts (`orders`, `purchasing`) that now call a third part on every change.

Choices made for an even fit, not for any arm:

- **No new startup flag.** Subscriptions are made through the API, so round 1's `stockroomtest.Start`
  and the existing tests (`TestRestartKeepsData`, `TestMissingSettings`) run the grown apps
  unchanged. A notifier configured by flags would have needed a required secret flag, which
  would have broken them.
- **Synchronous delivery**, with a 5 second timeout, inside the action. A background dispatcher
  would need a lifecycle method outside the entry leaf, which the three arms express very
  differently; it would also make the tests poll.
- **Thin events** (`{"id", "type", "subject"}`): the events part needs none of web's JSON forms.
- **`shop.PurchaseWant`** for the lines of a new purchase order, as `shop.Want` is for an order.
  `shop.PurchaseLine` has an invariant in L and V (`0 <= received <= qty`), so web cannot build
  one from unchecked request data without a panic. P uses the same type, so the code is the same.

## Files changed, per arm

Lines added and removed, from 19df87b (before this work) to the last commit of the work. READMEs
excluded: each gained 5 lines and lost 2, the same text in each arm.

| Part | P (`poc/app/go`) | L (`poc/app/golint`) | V (`poc/app/vulcan`) |
|---|---|---|---|
| shop | `event.go` +60, `purchase.go` +63 | `event.go` +60, `purchase.go` +66 | `event.vulcan` +60, `purchase.vulcan` +64 |
| purchasing (new) | `purchasing.go` +209 | `purchasing.go` +193, `ports.go` +36 | `purchasing.vulcan` +186, `component.vulcan` +28 |
| events (new) | `events.go` +181 | `events.go` +166, `ports.go` +31 | `events.vulcan` +160, `component.vulcan` +23 |
| webhooks (new) | `webhooks.go` +44 | `webhooks.go` +44 | `webhooks.vulcan` +43, `component.vulcan` +4 |
| store | `store.go` +52 −1, `purchases.go` +206, `events.go` +174 | `store.go` +52 −1, `purchases.go` +207, `events.go` +174 | `store.vulcan` +50, `purchases.vulcan` +206, `events.vulcan` +174, `component.vulcan` +2 −1 |
| inventory | `inventory.go` +6 | `inventory.go` +6, `ports.go` +1 | `inventory.vulcan` +7, `component.vulcan` +1 |
| orders | `orders.go` +13 −2 | `orders.go` +8 −2, `ports.go` +5 | `orders.vulcan` +5, `component.vulcan` +5 |
| web | `web.go` +296 −3 | `web.go` +274 −3, `ports.go` +22 | `web.vulcan` +267, `component.vulcan` +22 |
| wiring | `cmd/stockroom/main.go` +12 −5 | `cmd/stockroom/main.go` +12 −5, `arch/arch.go` +4 | `system.vulcan` +19 −10 |
| **Total added** | **1,316** | **1,361** | **1,326** |

Code lines in the whole app (`.go` or `.vulcan`): P 1,407 → 2,712; L 1,699 → 3,049; V 1,366 → 2,681.

Shared files: `docs/poc/stockroom.md` (the two services, a new parts diagram in plain words),
`poc/acceptance/growth_test.go` (new: 7 tests), `poc/stockroomtest/stockroomtest.go` (added only:
`Receiver`, `NewReceiver`, `Hook`, `Hook.Signed`, and `Receiver.FailNext` and `Hooks`).

## Contracts (the same in L and V)

Seven new clauses, each tied to a rule of `stockroom.md`:

| Clause | Rule |
|---|---|
| invariant of `shop.PurchaseLine`: `Received >= 0 && Received <= Qty` | Data, purchase order |
| `inventory` `Catalog.Restock` requires `len(arrivals) > 0` | 8 (as `Reserve` for rule 1) |
| `purchasing` port `Stock.Restock` requires `len(arrivals) > 0` | 8 |
| `purchasing` `Create` ensures the status is `open` | 7 |
| `purchasing` `Receive` ensures the status is `open` or `received` | 8, 9 |
| `purchasing` `Cancel` ensures the status is `cancelled` | 10 |
| `events` port `Sender.Send` requires a URL, a signature, and a body | Subscribers section: every delivery is signed |

## Checks run

- P: `go vet ./... && go test ./...` pass.
- L: `go vet ./... && vulcan-lint ./... && go test ./...` pass (the `vulcan-lint` from
  `harness tools`). A probe confirmed that the linter checks the new code: removing one
  `contract.Valid` gives VL0905.
- V: `vulcan check && vulcan test --race=false` pass; strict mode is on. The generated `main` has
  one store, one catalog, and one hub.
- `STOCKROOM_BIN=<binary> go test ./poc/acceptance/...` passes against each of the three binaries
  (17 tests: the 10 of round 1 and 7 new).
- `go test ./poc/...` passes.

## Where the arms could not be kept equal

Behavior is the same in the three arms; every difference below is in the code's form.

1. **The shared database, in V, is now five identical wire lines** (it was three):
   `Orders.Store`, `Inventory.Store`, `Billing.Store`, `Purchasing.Store`, and `Events.Store`.
   P and L pass one `db` value. This is the limit of V's wiring syntax that parity review P3
   (item 2) noted; it grows with each part that needs the store. `system.vulcan` says that the
   lines must stay the same, and K0607 warns if they differ.
2. **One hub, three wires in V.** `Web.Events`, `Orders.Events`, and `Purchasing.Events` each name
   `Events.NewHub`, and `Inventory.NewCatalog` is now wired twice. They share one instance
   because the constructors take no settings. P and L pass `hub` and `catalog`.
3. **The manifest in L.** The new parts and the `Subscribers` external go into `arch/arch.go`,
   which `PROTECTED` lists. In V they are `component.vulcan` files and a line of
   `system.vulcan`; in P, only `main.go`.
4. **Invariant checks are written in L, implicit in V.** L wraps each built `PurchaseLine` in
   `contract.Valid` (in `purchasing` twice and in `store` once), and the new `Restock` stock value
   too, as the linter requires. V checks the same literals without extra code. P has no check.
   This is the treatment, as for `shop.Stock` in round 1.
5. **Ports and constructors.** P and L keep a field per port and constructors with port parameters
   (`NewService(stock, store, events)`, `NewHub(store, sender)`); L stores the ports with a
   wrapper (`checkedStock`, `checkedSender`). V has empty constructors and calls ports by name
   (`Store.Purchase(id)`). The bodies are otherwise the same Go code: `store/purchases` and
   `store/events` are byte for byte the same in P and V, and differ in L only by the
   `contract.Valid` of item 4.
6. **Line wrapping.** gofmt wraps L's `NewHub` (longer because of the wrapper) onto three lines.

## Notes for the parity review and the rule writer

- The webhook secret is made with `crypto/rand` (`rand.Text`) inside the `events` leaf; deliveries
  are made while the order or purchase-order lock is held; and a failure to record an event is
  logged by `events` and never reaches the caller. Each is the same in the three arms, and each
  is a natural place for a rule.
- `shop.EventTypes` is a package-level array in the library: allowed in strict mode in both L and
  V, because an array is a value type.
- The `go.mod` files still say `go 1.26.0` while the repository builds with Go 1.27.3. They are
  equal across the arms, so this is no parity issue; upgrading them is its own change, in all
  three arms at once, before the freeze.
- The brief named `poc/stockroom.md` and `docs/poc/parity-p3.md`; the files are
  `docs/poc/stockroom.md` and `experiments/poc/parity-p3.md`.
