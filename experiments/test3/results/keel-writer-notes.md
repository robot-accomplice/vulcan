# Keel writer notes: audit trail feature

## Files changed

- `lib/shop/types.keel`: new types `AuditKind` (`AuditCharge`, `AuditRefund`) and `AuditEntry` (Order, Customer, Kind, Amount, At; `invariant Amount >= 0`).
- `api/billing/component.keel`: Billing now `needs port Audit { Record(e shop.AuditEntry) error }` with a `requires` on amount, order, and kind.
- `api/billing/ledger.keel`: `Charge` and `Refund` write the audit line before they change the ledger. New `account.checkRefund`, shared by `Refund` and `account.refund`, so the over-refund rule is in one place.
- `api/billing/ledger_test.keel`: existing tests bind an `auditFake`. New tests: each movement is audited once, rejected refunds are not audited, a failed audit write moves no money.
- `api/auditlog/component.keel` (new): leaf `AuditLog`, `gateway for AuditDB`.
- `api/auditlog/store.keel` (new): `Store` with `Record` and `ForOrder`. `startup` opens Postgres (pgx driver) from `AUDIT_DATABASE_URL`, pings it, and creates table `billing_audit` and its index if missing.
- `api/auditlog/store_test.keel` (new): Postgres tests that skip when `AUDIT_DATABASE_URL` is not set, plus one test that a call before startup returns `ErrNotStarted`.
- `api/web/component.keel`: Web now `needs port AuditTrail { ForOrder(id) ([]shop.AuditEntry, error) }`.
- `api/web/web.keel`: new route `GET /orders/{id}/audit`.
- `api/web/web_test.keel`: new `TestAuditTrail` (order of entries, `[]` for no entries, 500 when the store fails).
- `system.keel`: `external AuditDB`, leaf `AuditLog` in container Api, wires `Billing.Audit: AuditLog.NewStore` and `Web.AuditTrail: AuditLog.NewStore`.
- `go.mod`: requires `github.com/jackc/pgx/v5 v5.7.1`. No `go.sum` change: I cannot run `go`.
- `topics.lock`: not changed. The feature adds no topic.

## Design choices

1. **A gateway leaf owns Postgres.** The primer says only a gateway or entry leaf can import `database/sql`. Billing is neither, so Billing gets a port and the new gateway leaf `AuditLog` provides it. Web reads through a second port on the same leaf. Billing and Web stay free of database code and test with plain fakes.
2. **No money moves without an audit line (fail closed).** Billing validates, then writes the audit line, then changes the ledger, all under the ledger lock. If the write fails, the charge or refund fails and the ledger is unchanged. A refund that the ledger rejects writes no line. Cost: when Postgres is down, orders cannot be placed or cancelled. The other option, log the failure and continue, makes the trail incomplete without notice. I chose fail closed because an audit trail with gaps is not an audit trail. Residual gap: if the audit insert commits but the response is lost (timeout after commit), the ledger has no movement but the trail has a line. Fixing that needs an idempotency key or an outbox; I left it out because the request said "keep it simple".
3. **The database sets the time.** `At` comes from `DEFAULT now()`, so Billing needs no clock.
4. **`GET /orders/{id}/audit` does not check that the order exists.** Orders are in memory and the trail is in Postgres, so after a restart the trail outlives the order. The route returns `[]` for an order with no entries, never `null` (the port `ensures result != nil`). A store failure is a 500 via the existing `writeErr`.
5. **Connection string from the environment** (`AUDIT_DATABASE_URL`). See the unclear points below about `flag(...)`.
6. **Table created in `startup`** with `CREATE TABLE IF NOT EXISTS`. A real team may want migrations; this was the simplest option for "ship today".
7. **Each DB call has a 2 s timeout**, because port methods take no `context.Context`.

## Unclear or missing in the primer

1. **How leaf code reads a config value.** The primer shows `capacity: flag("orders-cap", max: 10_000)` in `system.keel`, but not where `config {}` goes for one leaf, or how code reads the value. I used an environment variable instead.
2. **Constructors.** Wires name constructors (`Billing.NewLedger`), but the primer does not say when they run, whether they may return an error, or whether two wires to one constructor (`AuditLog.NewStore` twice) make one instance or two. The list of places where leaf code runs does not mention constructors. So I opened the database in `startup` and kept it in a package `var`, not in `NewStore`.
3. **Startup order.** The primer does not say if all `startup` functions finish before the entry leaf serves, or if they run concurrently. If they run concurrently, an early request could reach the store before `db` is set: a data race on the package `var` and an `ErrNotStarted` error. I added the `ErrNotStarted` check, but the race stays if startups are concurrent.
4. **Which packages "depend on" `net`/`database/sql`.** Is a third-party driver (`pgx`) such a package, so that only a gateway can import it? I assumed yes and imported it only in the gateway.
5. **Gateway leaves.** The primer names `gateway for Stripe` but does not show a full gateway component. I do not know if it must also declare something about the external (for example a timeout), or if `external X` in `system.keel` is required (I added it).
6. **Contracts with `[]T` results and `pure`.** I wanted `ensures err == nil => every entry has Order == id`. The primer says contracts may call only `pure` functions, but does not show how to declare one. I used only `result != nil`.
7. **Tests and `startup`.** A `startup func` has no name, so a test cannot call it. My Postgres test copies its steps (open, create schema). The primer could say how to test startup code.
8. **Rebinding a port inside one test.** `TestAuditTrail` calls `keel.Bind` twice on `AuditTrail`. The primer does not say if the second call replaces the first.
9. **Contracts on an unexported helper.** I left `account.checkRefund` without contracts. The primer does not say if `invariant` checks run after unexported non-mutating methods, or if a method that only reads is treated differently.

## Round 2

Reviewer output: 1 error (no `go.sum` for pgx) and 5 warnings (discarded `Close` errors). I fixed all six.

### Changes

1. **`go.sum` (new) and `go.mod`.** The module cache had only pgx v5.9.2 and v5.10.0. Both declare `go 1.25.0`. To use either, this module's `go 1.22` floor must go up for every developer, which is a bigger change than this feature. So I kept v5.7.1. I resolved it in a scratch module with a real `.go` file that imports `github.com/jackc/pgx/v5/stdlib`, so `go mod tidy` sees the import. This fetched v5.7.1 and its requirements through the Go proxy, and the checksum database verified them. I copied the resulting `go.sum` into `work/keel/` and added the indirect requirements to `go.mod` (`pgpassfile`, `pgservicefile`, `puddle/v2`, `x/crypto v0.27.0`, `x/sync v0.8.0`, `x/text v0.18.0`). The scratch module builds with `GOTOOLCHAIN=local`. The `keel.dev/keel` entry still has no `go.sum` line, as the coordinator expects.
2. **`api/auditlog/store.keel`, `ForOrder`.** I removed `defer rows.Close()`. A new helper `scanEntries` reads the rows. `ForOrder` then returns `errors.Join(scanErr, rows.Close())`, so a close error is reported and not lost.
3. **`api/auditlog/store.keel`, `startup` error paths.** When ping or table creation fails, `startup` returns `errors.Join(cause, conn.Close())`. The cause comes first, and a close failure is added to it.
4. **`api/auditlog/store.keel`, shutdown goroutine.** There is no caller to return to after `ctx` is cancelled, so the close error goes to `log.Printf`. This matches how other leaves report errors that have no caller (`alerts`, `picking`).
5. **`api/auditlog/store_test.keel`.** The `t.Cleanup` function reports a close error with `t.Errorf`.

### Verification

`keel check` is still not available to me. I converted `store.keel` and `store_test.keel` to plain Go (contract lines removed, `startup func` given a name) against a stub `shop` package and the new `go.sum`. Then I ran `go vet` (clean) and `go test` (`TestNotStarted` passes; the two Postgres tests skip because there is no database). This shows that the Go bodies compile and that `go.sum` covers the imports. It does not show that `keel check` passes.

### Primer gaps found this round

1. The primer does not say that `keel check` reads `go.sum` to apply the network rule, or that `go mod tidy` ignores imports in `.keel` files. A team adding a third-party dependency needs to know how to produce `go.sum` (for example a `keel mod tidy` command, or a documented workaround).
2. The primer does not say that discarded error results are a `keel check` warning.
