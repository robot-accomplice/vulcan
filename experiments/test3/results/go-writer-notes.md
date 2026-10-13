# Go writer notes: Billing audit trail

## Files changed

- `audit/audit.go` (new): `Entry`, `Kind` (`charge` / `refund`), and `Memory`, an in-memory store for tests.
- `audit/postgres.go` (new): `Postgres` store on `database/sql`. Has `Schema` (`billing_audit` table plus index), `Migrate`, `Record`, `ForOrder`.
- `audit/postgres_test.go` (new): Postgres round-trip test. It runs only when `SHOP_TEST_DATABASE_URL` is set. **It skipped in this run** (no database available).
- `billing/billing.go`: `NewLedger(a Auditor)` now needs an auditor. `Charge` and `Refund` write one audit line. New `ErrAudit`, `AuditTimeout` (5 s).
- `billing/billing_test.go`: existing tests use `&audit.Memory{}`. New tests: lines are written in order; a rejected refund writes no line; an audit failure leaves the ledger unchanged.
- `web/web.go`: `Handler(o Orders, a AuditTrail)`. New route `GET /orders/{id}/audit` returns a JSON array of entries.
- `web/web_test.go`: existing test updated. New end-to-end test: place + cancel gives charge + refund lines; unknown order gives 404.
- `cmd/api/main.go`: reads `-db` or `DATABASE_URL`, pings, runs `Migrate`, and wires the Postgres store into Billing and the web handler. The process exits at start if there is no database.
- `go.mod`, `go.sum`: added `github.com/lib/pq v1.11.2` (taken from the local module cache, offline).

`go build ./...`, `go vet ./...`, `go test ./...` and `go test -race ./...` pass. One test skipped (the Postgres test).

## Design choices

1. **The audit write gates the money movement (fail closed).** Billing validates, writes the audit line, and only then changes the ledger. If the write fails, the charge or refund does not happen and the caller gets `ErrAudit` (HTTP 500). Reason: an audit trail that can silently miss a charge is not an audit trail. The alternative (record the money, log the audit failure) keeps orders working while the database is down, but loses lines.
2. **The audit write happens under the ledger mutex.** If the ledger applied first and rolled back on audit failure, a concurrent refund could spend an un-audited charge, and the rollback would then break "refunded never exceeds charged". Holding the lock avoids that. Cost: all billing is serialized behind one DB round trip. `orders.Service` already serializes `Place` / `Cancel` under its own lock, so in practice nothing is lost today. `AuditTimeout` bounds how long the lock is held.
3. **Billing owns the audit, not Orders.** The request says "when Billing records a charge or a refund". Putting it in `Ledger` means every caller is audited, and a rejected refund (over-refund, not charged) writes nothing, because no money moved.
4. **`NewLedger` requires an auditor and panics on nil.** No silent "no audit" mode in production. Tests and local runs use `audit.Memory`.
5. **Only the `cmd/api` binary and the test import the driver.** The `audit` package depends on `database/sql` only. I chose `lib/pq` because it has no transitive dependencies and was in the local cache. `pgx` is the more active driver; switching is a two-line change in `cmd/api/main.go` and the test.
6. **Schema by `CREATE TABLE IF NOT EXISTS` at start.** The repo has no migration tool. This is the simplest thing that works; replace it if the team adopts one.
7. **The audit route returns 404 for an unknown order** (checked through `Orders.Get`), and `[]` (never `null`) for a known order with no lines.
8. **Timestamps come from the Go process** (`time.Now().UTC()`), and rows are ordered by the `BIGSERIAL id`, not by time.

## Open issues and things I was unsure about

1. **Order IDs repeat after a restart. This mixes audit trails.** Orders live in memory and IDs restart at `o-1`. The audit lines persist in Postgres. After a restart, a new `o-1` shows the old `o-1` lines too. Also, after a restart, `GET /orders/o-1/audit` for an old order returns 404, because the order is gone from memory. I did not fix this: the fix is in Orders (IDs that are unique across restarts, or persisted orders), which is outside this request. Decide before release: (a) make order IDs unique (for example a UUID or a per-start prefix), or (b) accept the mix for now. I recommend (a).
2. **Fail closed means a database outage stops all orders.** This follows from choice 1. If the team prefers availability, change `record` to log and continue. That loses audit lines, so it needs a retry or outbox.
3. **Phantom line on a lost reply.** If the INSERT commits but the reply times out, Billing reports `ErrAudit` and does not move the money, but the line exists. A full fix needs a status column or an outbox. I judged it too much for "ship today".
4. **No auth on the audit route.** The other routes have none either, but the audit trail shows customer IDs and amounts.
5. **The Postgres store is not tested here.** No database was available. Run `SHOP_TEST_DATABASE_URL=postgres://... go test ./audit/` against a scratch database before release.
6. **`Handler` and `NewLedger` signatures changed.** All callers in this module are updated. Any caller outside this module breaks at compile time, which is the intent.
