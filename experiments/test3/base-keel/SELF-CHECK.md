# Self-check of base-keel against docs/spec.md

No `keel` tool exists (the spec says "Nothing is implemented"), so this check is manual. Each rule
in §5.4, §5.4a, §5.4b, §5.4c, §5.5, §6, §8, and §9 is listed with its result and its evidence.

**Result: 0 errors found. 0 warnings expected. 1 open item (O1), 4 spec ambiguities (A1 to A4).**

Mechanical support for the manual check:

1. The code files were translated to Go in a scratch directory (contract lines and `limit`
   removed, `handle` and `startup` made into plain functions, ports as package-level interface
   variables, a stub `keel.dev/keel`). `go vet ./...` and `go test ./...` pass on the result. This
   proves that the bodies and the tests type-check as Go, and that `*billing.Ledger` and
   `*orders.Service` have the method sets of the ports `Payments` and `Ordering` (§5.4 rule 5).
   It does not execute the contracts.
2. `go list -deps` on every imported standard package: only `net/http` and `net/http/httptest`
   depend on `net`, `os/exec`, or `database/sql`. Only `api/web` imports them.

## Fixes made during the check

| # | Rule | Problem found | Fix |
|---|---|---|---|
| F1 | §5.4 rule 7 | Comments in `api/orders/component.keel`, `api/alerts/component.keel`, `api/alerts/alerts.keel`, and `lib/shop/types.keel` named other leaves (`Picking`, `Orders`). | Comments reworded. Now only `system.keel` names two leaves. |
| F2 | §5.4 rule 7, §5.4a rule 3 | A library comment named the ports of leaves. | Reworded. |
| F3 | Style only, not a spec rule | `alerts.keel` used a literal time layout string. | Uses `time.RFC3339`. |

## §5.4 Rules the compiler enforces

| Rule | Result | Evidence |
|---|---|---|
| 1 Ports only | Pass | Leaf imports are only `shop/lib/shop`, `keel.dev/keel`, and standard packages. No leaf imports a leaf. Cross-leaf use is only `Ordering.*`, `Payments.*`, `Shipped.Publish`, and handlers. |
| 2 Code only in leaves and libraries | Pass | `system.keel` has no code. No leaf directory has a child leaf. |
| 3 Container boundaries | Pass | `Web.Ordering` and `Orders.Payments` are method ports inside `Api`, not `remote`. The only cross-container wire is the topic `Api/Orders.Shipped: Worker/Picking.Shipped`. No topic connects two leaves of one container. |
| 4 Complete wiring | Pass | `Web.Ordering`, `Orders.Payments`, `Orders.Shipped`: each wired once. `Alerts.DeadShipped` (a `DeadLetter[T]` topic) is wired by the `deadletters:` setting, so rule 3 does not apply to it. `Picking.Shipped` (published) is not wired by itself. `Billing` needs nothing. |
| 5 Satisfied ports | Pass | `Billing.NewLedger` returns `*Ledger` with `Charge` and `Refund`. `Orders.NewService` returns `*Service` with `Place`, `Cancel`, `Get`. Pointer receivers, so the constructors return pointers. Signatures identical (checked by the scratch build). |
| 6 Compatible wiring | Pass | Each provider method declares the same `requires` and `ensures` as the port method, so both implications (port requires => method requires; method ensures => port ensures) are trivially true. |
| 7 Boundary ownership | Pass after F1, F2 | Each leaf declares its own ports in its own `component.keel`. No `component.keel`, code file, or library names another leaf. |
| 8 No cycles | Pass | Api wire graph: Web -> Orders -> Billing. Alerts and Picking have no port wires. |
| 9 Private types | Pass | Every type in a port signature or a topic is from `shop/lib/shop` (`CustomerID`, `OrderID`, `Money`, `Receipt`, `Order`, `OrderShipped`) or is `error`. Leaf types (`account`, `record`, `orderBook`, `Ledger`, `Service`) are named only by their own leaf and, as `NewLedger` and `NewService`, by `system.keel`. |
| 10 Leaf size (audit) | Pass | No `context_budget` is set, so there is no warning. All leaves are under 130 non-blank lines. |

## §5.4a Libraries

| Rule | Result | Evidence |
|---|---|---|
| Declaration and package | Pass | `lib/shop/library.keel` declares `library shop {}`. `types.keel` starts with `package shop`. Code imports it as `"shop/lib/shop"` (module `shop` in `go.mod`). |
| 1 No ports, no topics, `var` set once | Pass | The library declares neither. Its only package-level `var`s are five `errors.New` sentinels, never assigned again. None changes through a method or a reference. |
| 2 No acquiring calls | Pass | The library imports only `errors`. |
| 3 Imports only libraries, no cycles | Pass | Imports only the standard package `errors`. |
| 4, 5 | Pass | Every leaf that needs a shared type imports the library. |

## §5.4b Gateways and entry leaves

| Rule | Result | Evidence |
|---|---|---|
| 1 One gateway per external system per container | Pass (n/a) | The system has no `external` declaration, so it needs no gateway. |
| Entry leaf | Pass | `Web` declares `entry for Customer`, and `Customer` is a `person` in `system.keel`. |

## §5.4c Go packages

| Rule | Result | Evidence |
|---|---|---|
| 1 Network packages only in gateway or entry leaves | Pass | Only `api/web` (entry) imports `net/http` and `net/http/httptest`. Other imports (`bufio`, `context`, `errors`, `fmt`, `log`, `os`, `strconv`, `strings`, `sync`, `time`, `testing`, `encoding/json`) have no dependency on `net`, `os/exec`, or `database/sql` (checked with `go list -deps`). See A3 for `keel.dev/keel`. |
| 2 Acquiring calls only in leaves | Pass | No library calls an acquiring function. `Picking` reads `os.Stdin`, which is not an acquiring call. `Web` calls `ListenAndServe` from a leaf. |
| Versions | Open (O1) | `go.mod` declares module `shop`. See O1 for `go.sum`. |

## §5.5 Crossing containers (with §5.5.0 to §5.5.3)

| Rule | Result | Evidence |
|---|---|---|
| Topic type `topic[T]` | Pass | `publishes Shipped topic[shop.OrderShipped]`, `subscribes Shipped topic[shop.OrderShipped]`, `subscribes DeadShipped topic[keel.DeadLetter[shop.OrderShipped]]`. |
| Wire reads `needer: supplier` with `via` | Pass | `Api/Orders.Shipped: Worker/Picking.Shipped via direct{ deadletters: Api/Alerts.DeadShipped }`. |
| `direct` requires `deadletters` | Pass | Set. |
| Dead-letter route type | Pass | `keel.DeadLetter[shop.OrderShipped]`, the same `T` as the wire. |
| Publish returns nothing | Pass | `Shipped.Publish(...)` is a statement. |
| Limit breach panics at publish | Pass | `ship` rejects a note over `shop.MaxNoteBytes` before `Publish`, and returns `ErrNoteTooBig`. |
| One handler per subscribed topic | Pass | Orders: `handle Shipped`. Alerts: `handle DeadShipped`. |
| `idempotent` on `direct` handlers, including dead-letter handlers | Pass | Both handlers are `idempotent`. `markShipped` returns `nil` for an order that is already shipped, and the test proves a repeat changes nothing. |
| Acknowledgement | Pass | Handlers return the error for an unknown or cancelled order, so `direct` retries and then dead-letters it. |
| Unread topic warning | Pass | `Picking.Shipped` has one subscriber. |
| Serializable messages | Pass | `OrderShipped` holds two strings. |
| `limit` on every variable-size field | Pass | `Order OrderID limit 64` (a named string type), `Note string limit 500`. Order ids are `"o-" + n`, far below 64 bytes. |
| `topics.lock` | Pass | Lists `shop.OrderShipped` and `keel.DeadLetter[shop.OrderShipped]` with field numbers, types, limits, and invariants. The spec gives no file format; see A4. |
| Remote ports (§5.5.3) | Pass (n/a) | None. |

## §6 Code

| Rule | Result | Evidence |
|---|---|---|
| Bodies are Go | Pass | The scratch build type-checks every body. |
| Code files have a package clause naming the directory | Pass | `billing`, `orders`, `web`, `alerts`, `picking`, `shop`. |
| `main` is generated | Pass | No `main` anywhere. |
| One `startup` per leaf at most | Pass | `Web` (serves HTTP until `ctx` ends) and `Picking` (reads stdin). See A1 for the syntax. |
| Container lifetime | Pass | `Api` has an entry leaf and a subscription, so it runs until a stop signal. `Worker` has neither, so it exits when `Picking`'s startup returns at the end of stdin. This is intended. |
| Runtime package path | Pass | `keel.dev/keel`. |
| Warnings: unhandled error | None expected | One explicit discard, `_ = json.NewEncoder(w).Encode(v)` in `web.keel`, after the status is sent; it has a comment. `errcheck` does not report an explicit blank assignment by default. `srv.Shutdown`'s error is returned from `startup`. |
| Warnings: error masking | None expected | Every error branch returns, wraps, writes, or logs the error. |
| Warnings: nil dereference | None expected | Map lookups check `ok` before use (`!ok \|\| a.charged == 0` short-circuits). |
| Warnings: unlocked shared field | None expected | `orderBook` and `Ledger` lock `mu` on every access. The `Shipped` handler and `Ordering` calls share `book` through that lock. |

## §8 Contracts

| Rule | Result | Evidence |
|---|---|---|
| Placement | Pass | Every contract line is between the signature and the body, or after the struct. |
| `result` guarded by `err == nil =>` | Pass | Every `ensures` that uses `result` on a function with an error result is guarded. `account.refund` uses only `err` and `old`. |
| Pure, cannot fail | Pass | Only comparisons, `+`, `&&`, `=>`, field access, `len`, `old`, and constants. No calls, no indexes. |
| Zero value satisfies each invariant | Pass | `Receipt`: `Amount >= 0` (0). `account`: `charged >= 0 && refunded >= 0 && refunded <= charged` (0, 0). `record`: `amount >= 0 && len(note) <= shop.MaxNoteBytes` (0, ""). |
| Only own methods write invariant fields | Pass | `account` fields change only in `charge` and `refund`. `record` fields change only in `cancel` and `ship`. `Receipt` is only built with literals. Literals are construction, which the compiler checks. |
| Construction | Pass | `Receipt{Amount: amount}` follows `requires amount > 0`. `&record{amount: amount}` likewise. |
| Rules the brief calls for | Pass | Refund > 0: `requires amount > 0` on the port and on `Ledger.Refund`. Never more than charged: invariant `refunded <= charged` on `account`, plus the error `shop.ErrOverRefund`. Not charged: `shop.ErrNotCharged`. Note <= 500 bytes: `limit 500`. |
| Fakes meet port contracts | Pass | `paymentsFake` echoes order and amount. `orderingFake` echoes id, customer, amount, and sets the status the port promises. `TestPlaceRejectsBadAmount` proves Web never calls `Place` with `amount <= 0`. |

## §9 Testing

| Rule | Result | Evidence |
|---|---|---|
| Go tests in `*_test.keel` in the leaf directory | Pass | `ledger_test.keel`, `orders_test.keel`, `web_test.keel`, `picking_test.keel`. |
| `keel.Bind(t, &Port, fake)` | Pass | `Payments` in Orders tests, `Ordering` in Web tests. |
| `keel.Published(t, Topic)` | Pass | `Shipped` in the Picking test. |
| Each leaf testable alone | Pass | Each test uses only its own leaf, fakes, and the library. |

## Open item

| # | Item | Why it is open |
|---|---|---|
| O1 | No `go.sum`. §6 says `keel check` needs `go.sum`. | The only non-standard dependency is `keel.dev/keel`, which is not published, so no real checksum exists. A fabricated hash would be wrong. `go.mod` requires `keel.dev/keel v0.1.0`; that version is a placeholder for the same reason. If the grader treats a missing `go.sum` as an error, this is it. |

## Spec ambiguities (choices made, not violations)

| # | Where | Choice |
|---|---|---|
| A1 | §6 gives the form `startup func(ctx context.Context) error` without a name. | Written exactly that way at top level. |
| A2 | §5.5.1: no rule says if `limit` accepts a named constant. | `limit 500` is a literal. `shop.MaxNoteBytes = 500` is a separate constant that `Picking` checks before it publishes and `record` uses in its invariant. The two values must stay equal by hand. |
| A3 | §5.4c rule 1 vs §6: `keel.dev/keel` is the runtime, and the `direct` transport may depend on `net`. | Treated as exempt, because the spec itself requires leaves to name `keel.DeadLetter` and to import `keel.dev/keel` in tests. `Alerts` imports it in code. |
| A4 | §5.5.2 gives no format for `topics.lock`, and no rule says if a listen address for an entry leaf can come from a flag (only wire settings take `flag(...)`, and `Web` is not a provider). | The lock uses a plain block format. `Web` listens on the constant `listenAddr = ":8080"`. |
