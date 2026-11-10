# Parity audit of build 1b (P's assertions, rows 5a and 19)

Status: independent, antagonistic audit, 2026-11-10. No change in the repo except this file.
Scope: `experiments/poc/build-1b/contracts.md` (the statement), `poc/app/go-assert` (P), its notes `NOTES-p.md`, against `poc/app/vulcan` (V) as lowered by `vulcan` at commit `ceb0ebed` (the statement's source commit; `git diff --stat ceb0ebed -- poc/app/vulcan poc/app/go vulcan/rt internal/lower` is empty). L (`poc/app/golint`) was read only where it decides a question.
Plan rows read: plan-coverage.md row 5a (line 449), row 19 (line 492), build order 1b (line 578).

**Verdict: build 1b passes the "no open High finding" gate.** One Medium finding (F1, a missed K28 check point that the statement caused) should be closed before phase A' runs row 5a.

## Method

All work was in a scratch directory outside the repo (copies of `go`, `go-assert`, `vulcan`). Tools: `go1.27.3 windows/amd64`; `vulcan` built from `./cmd/vulcan` of this repo.

1. Lowered V (`vulcan build`) and read the lowered Go with the `/*line*/` directives removed (`.vulcan/build/**`).
2. Diffed `poc/app/go` against `poc/app/go-assert` (`diff -ru`).
3. Ran one end-to-end script of 47 HTTP requests against G (plain Go), P and V, each with a fresh database and `poc/fakepay`.
4. Injected 15 faults (12 by mutation of the same source text in P and V, 3 by data or by a call that breaks a precondition) and ran package tests in P (`go test`) and V (`vulcan test`, fakes bound with `vulcan.Bind`).
5. Ran `go build -gcflags=-m` and `testing.AllocsPerRun` for the cost questions. Nothing was timed.

## 1. Statement fidelity (contracts.md against V)

Count: 29 clauses in the `.vulcan` files (`grep -rn -E "requires|ensures|invariant"`): 6 operation `requires`, 8 port `requires` (in `component.vulcan` of billing 2, events 1, inventory 1, orders 3, purchasing 1), 12 `ensures`, 3 `invariant`. This matches the statement's 14 / 12 / 3. `old(...)` occurs only in K02 and K04 (billing.vulcan:19, :27). Every condition, every `err == nil =>` guard, and the failure effect (panic, `OnContractFailure: "panic"` in the lowered `vulcan_main/main.go:44`) match.

Check points: the statement matches the lowered code for K01 to K27. For K27 the lowered code checks the invariant at the end of the outer `account.charge` and `account.refund` (lowered billing.go:34, :53) and at `&account{}` in `accountOf` (:68); the inner variants `vulcan_charge`/`vulcan_refund` (:195, :212) skip it. The statement's rule (outer calls only, after the call returns) is exact.

**K28 is not exact.** See F1 and F4.

## 2. P fidelity (go-assert against V at run time)

| Id | Condition | Check point | Count per call | Result |
|---|---|---|---|---|
| K01-K04 | same | entry; end (old copied at entry) | same | equal |
| K05-K08 | same | entry; after the inner method (V: at each return) | 1 = 1 | equal |
| K09-K12, K15-K17, K23, K11 | same | wrapper at the consumer, built in the constructor (V: generated wrapper set in `main.go:55-75`, and by `vulcan.Bind` in tests) | same; same set of ports wrapped | equal |
| K13, K14 | same | entry | same | equal |
| K18-K26 | same | after the inner method returns, after its deferred unlock (V: at each return, before the deferred unlock) | 1 = 1 | equal effect, see F6 |
| K27 | same | at build and after each outer charge/refund (P: by the caller; V: inside the outer method) | 3 sites = 3 sites | equal |
| K28 | same | `scanItem` and the four new-Stock closures | **5 sites, V has 6** | **F1** |
| K29 | same | `create`, `receive`, `store.Purchase` | 3 = 3 | equal |

Negations of each condition were read one by one (go-assert diff); all are exact complements.

The builder's choices:
- Wrappers for port contracts at the caller: equal to V's generated `Vulcan_wrap_*` types. Uncontracted methods of a wrapped port are forwarded in both (P by embedding, V by explicit forwarding).
- Split public/inner methods for postconditions: correct; equal to "every return".
- K27 replay: equal (fault B6 below).
- Constructors `shop.NewStock`, `shop.NewPurchaseLine`: equal at every site that the statement lists.
- **The one `Stock` literal left in `store.changeStock`** (go-assert `api/store/store.go:251`): V checks it (`api/store/store.vulcan:249`, lowered `store.go:250`: `change(shop.Vulcan_checkStock(shop.Stock{...}, "api/store/store.vulcan:249:23"), l.Qty)`). L checks it too (`poc/app/golint/api/store/store.go:252`). P is the only arm that does not. See F1.

## 3. Behaviour when every contract holds

`diff -r` of `go` and `go-assert`: no change in `api/web`, `api/webhooks`, `api/payments`, `cmd`, `go.mod`, `README.md`, `lib/shop/errors.go`, `lib/shop/event.go`, `api/store/events.go`. Exported signatures are the same; the old bodies moved to lowercase methods. New exports: package `lib/contract` (`Fail`, `Violation`), `shop.NewStock`, `shop.NewPurchaseLine`. `go build ./...` and `go vet ./...` pass in both.

End-to-end: one script of 47 requests (products, receive, orders create/pay/decline/refund/over-refund/ship/cancel, suppliers, purchase orders create/receive/over-receive/cancel, webhooks subscribe/deliveries/retry/delete), fresh database per arm, secrets and provider ids masked:

```
diff g.txt p.txt && echo "G==P identical"   ->  G==P identical
diff g.txt v.txt && echo "G==V identical"   ->  G==V identical
grep -ci "panic|contract" *.stderr          ->  0, 0, 0
```

## 4. Fault injection

The same text change in P and V (P file / V file), then one package test with fakes. "Effects" are the counters of the fakes or the database after the panic.

| # | Contract (kind) | Injected fault | P | V |
|---|---|---|---|---|
| B1 | K01 (pre) | `a.charge(p.Amount)` -> `a.charge(p.Amount - p.Amount)` | K01 panic; card refunds 0 | `account.charge requires` panic; card refunds 0 |
| B2 | K02 (post, old) | `a.Charged += amount` -> `a.Charged = amount` | K02 panic | `ensures a.Charged == old(a.Charged) + amount` panic |
| B3 | K04 (post, old) | `a.Refunded += amount` -> `a.Refunded = amount` | K04 panic | `ensures ... old(a.Refunded)` panic |
| B4 | K06 (post, err-guarded) | `Amount: o.Total` -> `Amount: o.Total + 1` in Charge | K06 panic; card charged 1, payment stored 1 | panic; card charged 1, payment stored 1 |
| B5 | K09 (port) | `Charge(o.Customer, o.Total)` -> `Charge(o.Customer, o.Total-o.Total)` | K09 panic; card charges 0 | `Billing.Card.Charge requires` panic; card charges 0 |
| B6 | K27 (invariant, data) | stored entries `[refund 5, charge 10]` | K27 panic at replay | invariant panic at `account.refund` |
| B7 | K27 (invariant, code) | over-refund guard disabled (`if false && ...`) | K27 panic; card refunds 1, stored 0 | invariant panic; card refunds 1, stored 0 |
| B8 | K05 (pre, call) | `Charge(Order{Charged: 5, Total: 10})` | K05 panic; card 0 | `Ledger.Charge requires` panic; card 0 |
| I1 | K12 (port) | `if qty <= 0 \|\| qty > MaxQty` -> `if qty > MaxQty` | K12 panic; store receives 0 | `Inventory.Store.Receive requires` panic; store receives 0 |
| I2 | K13 (pre, call) | `Catalog.Reserve(nil)` | K13 panic | `Catalog.Reserve requires` panic |
| O1 | K20 (post) | `o.Status = shop.Shipped` -> `shop.Paid` | K20 panic; shipped 1, saved 1, published 1, mutex free | panic; shipped 1, saved 1, published 1, mutex free |
| U1 | K29 (invariant) | received-overflow guard disabled | K29 panic; restocks 0, saves 0 | invariant panic at `purchasing.vulcan:130:18`; restocks 0, saves 0 |
| S2 | K28 (invariant, code) | `if st.Available() < qty` disabled; reserve 7 of 5 | K28 panic; rows after `on_hand=5 reserved=0` (rolled back) | invariant panic at `store.vulcan:204:10`; rows after `on_hand=5 reserved=0` |
| **S1** | **K28 (invariant, data)** | stored row `on_hand=5 reserved=7`, then `Release 2` | **no panic, err nil; row after `on_hand=5 reserved=5`** | **invariant panic at `store.vulcan:249:23`; row after `on_hand=5 reserved=7`** |

Without mutation, every test gives the same result in P and V (B6, B8, I2 panic as designed; the others do not panic). 14 of 15 rows are equal in catch and in effect. S1 differs (F1). After the runs, `diff -r -x .vulcan -x 'zz_fi_test*'` of each scratch copy against the repo was empty, and nothing in the repo changed.

## 5. Cost parity for row 19 (no timing)

Per request, in the steady state of the request paths of `experiments/poc/timing/requests.json`:

| Difference | Where | Paths | Leans (on time) |
|---|---|---|---|
| V allocates one string per invariant check at construction: `"the value built at " + vulcan_at` is built before the check, and escapes (`-gcflags=-m`: `types.vulcan:44:26: "the value built at " + vulcan_at escapes to heap`; same for `purchase.vulcan:33:16`, `billing.vulcan:14:19`). `testing.AllocsPerRun`: V `Vulcan_checkStock` 1 alloc/op; P `NewStock` 0 alloc/op | every Stock, PurchaseLine and account built | catalog-list (+1 per product, 5), product-get (+1), stock-receive (+1), order-create (+1 per line for `Product`, +2 per line in `changeStock`), purchase-create and purchase-receive (+1 per line built or read, +2 per line in `changeStock`), order-payments 0 (no `accountOf` on a read) | favours P (F2) |
| V checks the input Stock in `changeStock`; P does not | store `changeStock` | order-create, purchase-receive (+1 compare per line) | favours P (F1) |
| P has one more call frame for each postcondition method (public calls inner, then returns the result struct again) | `Ledger.Charge/Refund`, orders `Create/Pay/Ship/Cancel/Refund`, purchasing `Create/Receive/Cancel` | order-create, purchase-create, purchase-receive | favours V (very small) |
| V copies parameters at entry (`vulcan_p_*`) and the results into temporaries at each return | functions with `ensures` | same paths | favours P (very small; registers) |
| V runs postconditions before the deferred unlock; P after it | orders, purchasing | write paths | favours P only under lock contention; row 19 requests are sequential, so about 0 |
| Port wrappers: same ports wrapped, two dynamic calls per wrapped call in both | all ports with contracts | all paths that call them | equal |
| Failure message work | only on failure, in both (`if` first, then `Fail`/`Show`) | none | equal |

No check runs more often in P than in V. Everything above is nanoseconds per request against SQLite work of microseconds to milliseconds, well inside the ln 1.1 margin. The net lean is toward P.

## Findings

### F1. Medium: P skips the K28 check of the input Stock in `store.changeStock`, which V and L check

Evidence: go-assert `api/store/store.go:251` `next, err := change(shop.Stock{OnHand: onHand, Reserved: reserved}, l.Qty)`; V `api/store/store.vulcan:249` (lowered with `Vulcan_checkStock`); L `poc/app/golint/api/store/store.go:252` `contract.Valid(...)`. Fault S1: V panics and leaves the row unchanged; P returns nil and writes `reserved=5` over a corrupt row. The builder's claim in NOTES-p.md, choice 8 ("the new Stock check or the earlier scanItem read would catch the data in most cases"), is false on the `Release` path: no `scanItem` runs there, and the new Stock (5,5) is valid.
Effect: in row 5a, a seed that corrupts stored stock and then calls `Release`, `Ship` or `Restock` is caught by V and L and missed by P (favours V). In row 19, P does one compare less per line (favours P, very small).
Root cause: the statement (F4). The builder followed the list in K28.
Fix: in go-assert `store.changeStock`, replace the literal with `shop.NewStock(onHand, reserved)`. Correct K28 in contracts.md (F4) with a dated erratum. Re-run fault S1; expect a K28 panic in P.

### F2. Low: V allocates a string at each invariant check of a built value (a real V cost)

Evidence: section 5 (escape analysis, 1 alloc/op against 0). The cause is in the lowering: `Vulcan_check<T>` builds `"the value built at " + vulcan_at` before it calls `Vulcan_invariant`, not only on failure.
Effect: row 19 leans toward P. This is V's real cost, not a parity defect of P, so row 19 may count it.
Fix: none is needed for parity. If the owner wants V's lowering fixed, decide and change it **before** any row 19 timing, and record the decision. A change after timing data exists would be a data-driven change (rule 6).

### F3. Low: P's invariant failure names the helper, not the place that built the value

Evidence: S2 in P reports `lib/shop/types.go:106` (inside `NewStock`); B6 and B7 report `billing.go:52` (inside `checkInvariant`). V reports the construction site (`the value built at api/store/store.vulcan:204:10`) or the method (`account.refund`). `contract.Fail` uses `runtime.Caller(1)`.
Effect: none on row 5a (it counts catches). In the agent campaigns where P+ runs with these assertions (row 8), V's report points nearer to the fault (favours V).
Fix: give `NewStock`, `NewPurchaseLine` and `checkInvariant` a call-site place (for example `runtime.Caller(2)` through a `contract.FailAt(skip, ...)`), or record this as an accepted difference.

### F4. Low: contracts.md K28 lists its check sites incompletely and names one wrongly

Evidence: contracts.md:160 says "every time a Stock value is built", then lists `scanItem` and "the stock-change helpers behind DB.Reserve, DB.Release, DB.Ship, DB.Receive and DB.Restock (each builds the new Stock from the old levels and the quantity)". It omits the input Stock that `changeStock` builds from the row (V store.vulcan:249). It also names `DB.Receive`, which does not use `changeStock`: it runs an SQL update and then reads through `Product` and `scanItem` (NOTES-p.md, row K28, says so).
Effect: the cause of F1. The general rule and the site list disagree, and the builder followed the list.
Fix: add the `changeStock` input to K28, and replace "DB.Receive" with "DB.Receive (through `Product`, which reads with `scanItem`)". Add a dated erratum line, because L and P were both built from this file.

### F5. Low: port checks in P bind only values built through the constructors

Evidence: P puts the wrapper in `NewLedger`, `NewHub`, `NewCatalog`, `NewService` (go-assert billing.go:87, events.go:57, inventory.go:40, orders.go:81, purchasing.go:57). A test in the same package that writes `&Service{stock: fake}` bypasses the check. In V, `vulcan.Bind` always wraps (`vulcan/vulcan.go:28-30`), but a test can also assign the port global directly and bypass it.
Effect: none on this build (no test files exist in P). It can matter for row 9 and the test-time count of row 5a.
Fix: state in the row 9 fake-suite brief that P's fakes are passed through the constructors.

### F6. Low (no effect found): postconditions run after the deferred unlock in P, before it in V

Evidence: NOTES-p.md, choice 2; spec 7.2, "Checks run before deferred calls". The postconditions of K18 to K26 read only the returned values, so no shared state is read after the unlock. Fault O1 shows the same effects in both arms (mutex free after the panic).
Fix: none.

## What I did not check

- L's fidelity against the statement. I read it only for the `changeStock` question.
- Panics inside the HTTP layer (both arms use `net/http`'s per-request recover). Both web packages are byte-identical between G and P.
- The race detector: it could not run on this host (no cgo, K0705).

## Recheck of F1 and F4 (2026-11-10)

The same scratch method as above. The scratch copies were synced from the repo first: `diff -r` showed only one changed line in go-assert, `api/store/store.go:251`. The V copy was the same as `poc/app/vulcan`.

1. **F1 fix in P.** `poc/app/go-assert/api/store/store.go:251` now reads `next, err := change(shop.NewStock(onHand, reserved), l.Qty) // K28 on the row read (parity-1b F1)`.
2. **No other Stock literal.** `grep -rn "shop\.Stock\b\|\bStock{" poc/app/go-assert --include=*.go` finds only the closure and `changeStock` signatures (`store.go:202, 212, 219, 230, 236`), the `checkedStock{...}` port wrappers in orders and purchasing, and the one literal inside `NewStock` itself (`lib/shop/types.go:108`). No `var x shop.Stock` exists.
3. **Fault S1 re-run** (stored row `on_hand=5 reserved=7`, then `Release 2`):
   - P: `contract K28 violated at .../lib/shop/types.go:106: Stock needs 0 <= Reserved <= OnHand, got OnHand=5 Reserved=7`, err nil, row after `on_hand=5 reserved=7` (rolled back).
   - V: `contract failure: the value built at api/store/store.vulcan:249:23 invariant: ...`, row after `on_hand=5 reserved=7`.
   - Same catch, same effect. Fault S2 re-run is also unchanged and equal in both arms (K28 panic, row rolled back to `on_hand=5 reserved=0`).
4. **Behaviour.** `go build` and `go vet ./...` of go-assert pass. The end-to-end script, run again on the rebuilt P: `diff g.txt p2.txt` gives `G==P identical`, and the server stderr has 0 lines with "panic" or "contract". Correction to section 3: the script sends **48** requests, not 47 (`grep -c " -> " p2.txt` gives 48).
5. **F4 fix in the statement.** contracts.md:162 has a dated correction under K28. It adds the `changeStock` input Stock as a check point and removes `DB.Receive` from the stock-change helpers. It does not say that `DB.Receive` reads its result through `Product` and `scanItem`, but the `scanItem` site already covers that path, so the check points are now exact.

Open findings: **0 High, 0 Medium, 4 Low** (F2, F3, F5, F6). F1 and F4 are closed. Build 1b passes the "no open High finding" gate.
