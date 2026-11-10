# Contracts K01-K29 in plain Go

Helper: `app/lib/contract/contract.go`. `contract.Fail(id, format, args...)` panics with a `*contract.Violation` (an `error`). Its message names the id, the file:line of the check, and the values. Checks are plain `if` statements, so a message is built only on failure. No build tag, no flag: the checks are always on. Nothing is undone on failure; deferred unlocks still run.

## Where each contract is checked

| Id | File | Function |
|---|---|---|
| K01 | app/api/billing/billing.go | `account.charge` (entry) |
| K02 | same | `account.charge` (end; old `Charged` copied at entry) |
| K03 | same | `account.refund` (entry) |
| K04 | same | `account.refund` (end; old `Refunded` copied at entry) |
| K05 | same | `Ledger.Charge` (entry) |
| K06 | same | `Ledger.Charge` (after inner `charge` returns) |
| K07 | same | `Ledger.Refund` (entry) |
| K08 | same | `Ledger.Refund` (after inner `refund` returns) |
| K09 | same | `checkedCard.Charge` |
| K10 | same | `checkedCard.Refund` |
| K11 | app/api/events/events.go | `checkedSender.Send` |
| K12 | app/api/inventory/inventory.go | `checkedStore.Receive` |
| K13 | same | `Catalog.Reserve` (entry) |
| K14 | same | `Catalog.Restock` (entry) |
| K15 | app/api/orders/orders.go | `checkedStock.Reserve` |
| K16 | same | `checkedBilling.Charge` |
| K17 | same | `checkedBilling.Refund` |
| K18-K22 | same | `Service.Create`, `Pay`, `Ship`, `Cancel`, `Refund` (after the inner method returns) |
| K23 | app/api/purchasing/purchasing.go | `checkedStock.Restock` |
| K24-K26 | same | `Service.Create`, `Receive`, `Cancel` (after the inner method returns) |
| K27 | app/api/billing/billing.go | `account.checkInvariant`, called in `Ledger.accountOf` and `Ledger.refund` |
| K28 | app/lib/shop/types.go | `shop.NewStock`; used in `app/api/store/store.go` (`scanItem`, and the new Stock of `Reserve`, `Release`, `Ship`, `Restock`; `Receive` ends in `Product`, which reads through `scanItem`) |
| K29 | app/lib/shop/purchase.go | `shop.NewPurchaseLine`; used in `app/api/purchasing/purchasing.go` (`create`, `receive`) and `app/api/store/purchases.go` (`Purchase`) |

## Choices

1. **Postconditions at every return.** Each checked operation is split in two. The public method keeps its name and signature. It calls an unexported method with the original body, then checks the result. A `return` in the body cannot skip the check. A `defer` check was rejected: it also runs while another panic unwinds, and it could replace that panic. Cost: the old body now sits in a lowercase method (`create`, `pay`, `ship`, `cancel`, `refund`, `receive`, `charge`). Only the "if no error" form is checked when `err == nil`; an error return passes.
2. **Order of work.** The postcondition runs after the inner method has released its lock. This does not matter: a failed postcondition leaves the work done in any case (contracts.md).
3. **Interface-method (port) contracts K09-K12, K15-K17, K23, K11: a wrapper at the call site, not in the provider.** Each consumer package defines a small `checked...` type that embeds the interface and overrides only the contracted methods. The constructor (`NewLedger`, `NewHub`, `NewCatalog`, `NewService`) wraps what it receives. Reasons: (a) the contract says the caller is responsible and that it holds for every type behind the interface, test fakes included. A check in `payments.Client` or `DB` would miss fakes. (b) contracts.md says the payments `Client` need not check K09. (c) One wrapper per interface keeps the check in one place, whatever the number of callers. Providers (`payments`, `webhooks`, `store`, `Catalog`) are unchanged. Not done in both places: a second check in the provider would be dead duplication. The panic place is the wrapper line, which names the contract id.
   - Some of these cannot fire through today's callers, because the caller validates first (for example, `Catalog.Receive` returns an error for `qty <= 0` before the store call, so K12 never fires on that path). They still guard any other caller or fake.
4. **K13, K14 are separate from K15, K23.** `Catalog.Reserve`/`Restock` check on entry (own contract). The `Stock` wrappers in orders and purchasing check at the call (port contract). Both are present, as the file says.
5. **K27 invariant.** `account.checkInvariant` is a method on `account`. It is called by non-`account` code only, per the file: right after `&account{}` in `accountOf` (built), after each `a.charge`/`a.refund` in the `accountOf` replay, and after `a.refund` in `Ledger.refund`. `account.charge`/`refund` do not call it (their callers do), so no check happens "between statements".
6. **K27 replay-order note.** `accountOf` replays stored entries in stored order and checks after each entry. A stored refund entry before enough charge entries therefore panics with K27 at that entry, even if the final sums would be valid. Implemented as written; not softened. This can only happen with data that is already inconsistent; the app itself writes a charge before any refund.
7. **K27 and K03 together.** A stored refund entry with `Amount <= 0` fails K03 first (it is checked on entry to `refund`), as the contracts say.
8. **K28, K29 via constructors.** `shop.NewStock` and `shop.NewPurchaseLine` build the value and check it. Struct literals of these types are replaced at the sites contracts.md lists. Go cannot forbid a literal elsewhere; the types have no mutating methods, as the file says. One literal stays in `store.changeStock`: the `Stock` read from the row as input to the change function. The file lists only the new Stock built by each helper and the `Item` from `scanItem`, so the input is not checked. If it is invalid, the new Stock check or the earlier `scanItem` read would catch the data in most cases.
9. **No behaviour change when contracts hold.** No export, signature or output changed. New exports: `lib/contract` (new package), `shop.NewStock`, `shop.NewPurchaseLine`. No new modules; standard library only. `app` has no test files.
10. **K02, K04** compare `a.Charged`/`a.Refunded` with the copy taken at entry plus the amount. With a plain add this is true by construction in one goroutine; it would fail on a data race or an overflow (wrap-around). It is implemented because the contract is listed.

## Results

`go build ./...`, `go vet ./...`: pass. `go test ./...`: no test files. Service started (no `fakepay` available, so the payments URL pointed to an unused port); `GET /health` returned 200 `{"status":"ok"}`. A smoke run of add product, supplier, purchase order, receive, order create and cancel returned normal output with no contract panic.
