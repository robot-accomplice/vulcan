# Contracts of the Stockroom app

What this is: a plain-English statement of every contract in the Vulcan version of the Stockroom app, each tied to the matching operation or type in the plain Go app.
Source: the `.vulcan` files in `C:\code\vulcan\poc\app\vulcan`, at git commit `ceb0ebedfd486a1cc05efd3e1b23f82682525663` of `C:\code\vulcan`. The meaning of the clauses is taken from `docs/spec.md` (sections 5.3, 7.1 to 7.4).
Count: 29 contracts (K01 to K29): 14 preconditions (6 on operations, 8 on interface methods), 12 postconditions, 3 invariants.
No code. Written for a builder who sees only this file and the Go app.

## Terms used in every contract

- A **precondition** is a condition on the inputs that must be true when the operation is called.
- A **postcondition** is a condition on what an operation returns (and on the receiver after the call). It is evaluated at every point where the operation returns.
- "Returns no error" means the error result is nil. A postcondition written "if the call returns no error, then X" says nothing when the call returns an error. An error return satisfies it.
- "The returned value" or "the result" means the first non-error result of the operation.
- "The old value of F" means the value F had at the moment the operation was entered, before any of its work. It is a copy taken at entry, not the value at the end.
- An **invariant** is a condition on the fields of one value of a struct type. It is judged from that value's own fields only.
- A **port contract** is a precondition attached to an interface method that one part needs from another (the Go interface named below). It applies to every call made through that interface, whatever type stands behind it, including test fakes. The caller is responsible for it. Where the type behind the interface has the same condition as its own contract, that is a separate contract, listed on its own below. Both are checked.

## What happens when a contract fails at run time

- The check always runs. It cannot be switched off.
- A failed contract stops the running code with a panic. The panic value reports which contract failed, its place in the source, and the values involved.
- Nothing is undone. Whatever the operation already did before the failed check stays done: database writes, requests sent to the card processor, events published, changes to in-memory values. The panic only unwinds the call stack. Deferred calls (for example, unlocking a mutex) still run as the stack unwinds.
- A failed precondition stops the operation before any of its work is done. A failed postcondition or invariant is found after the work is done, so the work stays done.
- The failure is not turned into an error return. The caller receives no error value. The panic travels up the call stack like any other panic, until some caller recovers it or the process ends.
- This app does not set the "exit on contract failure" option, so the default applies: panic, not process exit.
- A postcondition that begins "if the call returns no error" cannot fail on a call that returns an error.

## When each kind is checked

- Precondition: on entry to the operation, before the first statement of its body runs. For a port contract: at each call made through the interface, before the call reaches the type behind it.
- Postcondition: at every return of the operation (every `return` statement and the end of the body), after the returned values are fixed.
- Invariant: see K27 to K29 for the exact moments for each type.

## Preconditions and postconditions on operations

### package api/billing

**K01** Type `account`, method `charge` (unexported, in `api/billing/billing.go`). Kind: precondition.
Condition: the amount argument is greater than zero.
Checked: on entry.

**K02** Type `account`, method `charge`. Kind: postcondition.
Condition: after the call, the account's `Charged` field equals the old value of `Charged` plus the amount argument. "Old value" is the account's `Charged` at entry. It applies on every call. The method has no error result.
Checked: at every return.

**K03** Type `account`, method `refund` (unexported). Kind: precondition.
Condition: the amount argument is greater than zero.
Checked: on entry.

**K04** Type `account`, method `refund`. Kind: postcondition.
Condition: after the call, the account's `Refunded` field equals the old value of `Refunded` plus the amount argument. "Old value" is the account's `Refunded` at entry. It applies on every call. The method has no error result.
Checked: at every return.

**K05** Type `Ledger`, method `Charge`. Kind: precondition.
Condition: in the order argument, `Charged` is zero and `Total` is greater than zero. Both parts must hold.
Checked: on entry.

**K06** Type `Ledger`, method `Charge`. Kind: postcondition.
Condition: if the call returns no error, then the returned payment has `Kind` equal to the charge kind (`shop.KindCharge`) and `Amount` equal to the order argument's `Total`. Nothing is required when the call returns an error.
Checked: at every return.

**K07** Type `Ledger`, method `Refund`. Kind: precondition.
Condition: the amount argument is greater than zero.
Checked: on entry.

**K08** Type `Ledger`, method `Refund`. Kind: postcondition.
Condition: if the call returns no error, then the returned payment has `Kind` equal to the refund kind (`shop.KindRefund`) and `Amount` equal to the amount argument. Nothing is required when the call returns an error.
Checked: at every return.

**K09** Interface `Card` in `api/billing`, method `Charge` (the card processor port; the type behind it in the app is `Client` in `api/payments`, method `Charge`). Kind: precondition (port contract).
Condition: the amount argument is greater than zero.
Checked: at each call through the `Card` interface, before the call is made. The caller (`Ledger`) is responsible. The payments `Client` is not required to check it.

**K10** Interface `Card` in `api/billing`, method `Refund` (type behind it: `Client` in `api/payments`, method `Refund`). Kind: precondition (port contract).
Condition: the charge ID argument is not the empty string, and the amount argument is greater than zero. Both parts must hold.
Checked: at each call through the `Card` interface, before the call is made.

### package api/events

**K11** Interface `Sender` in `api/events`, method `Send` (type behind it: `Sender` in `api/webhooks`, method `Send`). Kind: precondition (port contract).
Condition: the url argument is not the empty string, the signature argument is not the empty string, and the body argument has length greater than zero. All three parts must hold.
Checked: at each call through the `Sender` interface, before the call is made. The caller (the events hub) is responsible.

### package api/inventory

**K12** Interface `Store` in `api/inventory`, method `Receive` (type behind it: `DB` in `api/store`, method `Receive`). Kind: precondition (port contract).
Condition: the qty argument is greater than zero.
Checked: at each call through the `Store` interface, before the call is made. The caller (`Catalog.Receive`) is responsible.

**K13** Type `Catalog` in `api/inventory`, method `Reserve`. Kind: precondition.
Condition: the lines argument has length greater than zero.
Checked: on entry.

**K14** Type `Catalog` in `api/inventory`, method `Restock`. Kind: precondition.
Condition: the arrivals argument has length greater than zero.
Checked: on entry.

### package api/orders

**K15** Interface `Stock` in `api/orders`, method `Reserve` (type behind it: `Catalog` in `api/inventory`, method `Reserve`). Kind: precondition (port contract).
Condition: the lines argument has length greater than zero.
Checked: at each call through the `Stock` interface, before the call is made. The caller (`Service`) is responsible.

**K16** Interface `Billing` in `api/orders`, method `Charge` (type behind it: `Ledger` in `api/billing`, method `Charge`). Kind: precondition (port contract).
Condition: in the order argument, `Charged` is zero and `Total` is greater than zero. Both parts must hold.
Checked: at each call through the `Billing` interface, before the call is made.

**K17** Interface `Billing` in `api/orders`, method `Refund` (type behind it: `Ledger` in `api/billing`, method `Refund`). Kind: precondition (port contract).
Condition: the amount argument is greater than zero.
Checked: at each call through the `Billing` interface, before the call is made.

**K18** Type `Service` in `api/orders`, method `Create`. Kind: postcondition.
Condition: if the call returns no error, then the returned order has `Status` equal to pending (`shop.Pending`) and `Charged` equal to zero.
Checked: at every return.

**K19** Type `Service` in `api/orders`, method `Pay`. Kind: postcondition.
Condition: if the call returns no error, then the returned order has `Status` equal to paid (`shop.Paid`) and its `Charged` equals its own `Total`.
Checked: at every return.

**K20** Type `Service` in `api/orders`, method `Ship`. Kind: postcondition.
Condition: if the call returns no error, then the returned order has `Status` equal to shipped (`shop.Shipped`).
Checked: at every return.

**K21** Type `Service` in `api/orders`, method `Cancel`. Kind: postcondition.
Condition: if the call returns no error, then the returned order has `Status` equal to cancelled (`shop.Cancelled`) and its `Refunded` equals its own `Charged`.
Checked: at every return.

**K22** Type `Service` in `api/orders`, method `Refund`. Kind: postcondition.
Condition: if the call returns no error, then the returned order's `Refunded` is less than or equal to its own `Charged`.
Checked: at every return.

### package api/purchasing

**K23** Interface `Stock` in `api/purchasing`, method `Restock` (type behind it: `Catalog` in `api/inventory`, method `Restock`). Kind: precondition (port contract).
Condition: the arrivals argument has length greater than zero.
Checked: at each call through the `Stock` interface, before the call is made. The caller (`Service`) is responsible.

**K24** Type `Service` in `api/purchasing`, method `Create`. Kind: postcondition.
Condition: if the call returns no error, then the returned purchase order has `Status` equal to open (`shop.PurchaseOpen`).
Checked: at every return.

**K25** Type `Service` in `api/purchasing`, method `Receive`. Kind: postcondition.
Condition: if the call returns no error, then the returned purchase order has `Status` equal to open (`shop.PurchaseOpen`) or equal to received (`shop.PurchaseReceived`).
Checked: at every return.

**K26** Type `Service` in `api/purchasing`, method `Cancel`. Kind: postcondition.
Condition: if the call returns no error, then the returned purchase order has `Status` equal to cancelled (`shop.PurchaseCancelled`).
Checked: at every return.

## Invariants on value types

An invariant uses only the fields of the one value. The zero value of each type satisfies its invariant.

**K27** Type `account` in `api/billing` (unexported). Kind: invariant.
Condition: `Refunded` is greater than or equal to zero, and `Refunded` is less than or equal to `Charged`. Both parts must hold.
Checked: (a) when an `account` value is built (`Ledger.accountOf` builds it as a zero value); (b) after every call of the `account` methods `charge` and `refund` returns, when the call is made from code that is not itself a method of `account` (the calls in `Ledger.accountOf` and in `Ledger.Refund` are such calls). It is judged on the state after the call returns, not between statements. Note: `Ledger.accountOf` replays the stored ledger entries in stored order, so a refund entry that comes before enough charge entries makes this invariant false at that point.

**K28** Type `Stock` in `lib/shop` (fields `OnHand` and `Reserved`). Kind: invariant.
Condition: `OnHand` is greater than or equal to zero, `Reserved` is greater than or equal to zero, and `Reserved` is less than or equal to `OnHand`. All three parts must hold.
Checked: every time a `Stock` value is built. The Go app builds `Stock` values in `api/store/store.go` where it reads a product row (the `Item` it returns) and in the stock-change helpers behind `DB.Reserve`, `DB.Release`, `DB.Ship`, `DB.Receive` and `DB.Restock` (each builds the new `Stock` from the old levels and the quantity). `Stock` has no method that changes its fields.

*Correction, 2026-11-10 (parity audit 1b, F4):* the list above is not complete. `Stock` is also built in `store.changeStock` in `api/store/store.go`, from the levels read from the stock row before each change, and the invariant is checked there too. `DB.Receive` is not one of the stock-change helpers. P's builder worked from the uncorrected text, and the fix in P is parity-1b F1.

**K29** Type `PurchaseLine` in `lib/shop` (fields `Qty` and `Received`). Kind: invariant.
Condition: `Received` is greater than or equal to zero, and `Received` is less than or equal to `Qty`. Both parts must hold.
Checked: every time a `PurchaseLine` value is built. The Go app builds one in `api/purchasing/purchasing.go` (in `Service.Create`, with `Received` zero; in `Service.Receive`, where a line is rebuilt with `Received` raised by the arrival quantity) and in `api/store/purchases.go` (where lines are read from the database). `PurchaseLine` has no method that changes its fields.

## Summary

| Kind | Ids | Count |
|---|---|---|
| Precondition on an operation | K01, K03, K05, K07, K13, K14 | 6 |
| Precondition on an interface method (port contract) | K09, K10, K11, K12, K15, K16, K17, K23 | 8 |
| Postcondition | K02, K04, K06, K08, K18, K19, K20, K21, K22, K24, K25, K26 | 12 |
| Invariant | K27, K28, K29 | 3 |
| Total | | 29 |

Other contract forms: none. The files have no `pure` functions, no message or topic contracts, and no use of `old(...)` other than in K02 and K04. The port contracts are declared in the `component.vulcan` files of `api/billing`, `api/events`, `api/inventory`, `api/orders` and `api/purchasing`. The files of `api/store`, `api/web`, `api/payments` and `api/webhooks` carry none.
