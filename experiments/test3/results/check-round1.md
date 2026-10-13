# keel check: experiments/test3/work/keel (round 1)

## Diagnostics

error go.mod:7: module github.com/jackc/pgx/v5 v5.7.1 has no go.sum entry (no go.sum file exists); keel check cannot resolve the Go dependencies that the network rule reads. Imported at api/auditlog/store.keel:11 ("github.com/jackc/pgx/v5/stdlib"). Reproduced: `go build` on a copy of this go.mod reports "missing go.sum entry for module providing package github.com/jackc/pgx/v5/stdlib". hint: run `go mod tidy` and commit go.sum (go.mod will probably also need pgx's indirect requirements; not verified offline) (§6 "The runtime package", §5.4c "Versions")
warning api/auditlog/store.keel:79: unhandled error: the error result of (*sql.Rows).Close is discarded (`defer rows.Close()`) (§6, errcheck)
warning api/auditlog/store.keel:108: unhandled error: the error result of (*sql.DB).Close is discarded (§6, errcheck)
warning api/auditlog/store.keel:113: unhandled error: the error result of (*sql.DB).Close is discarded (§6, errcheck)
warning api/auditlog/store.keel:118: unhandled error: the error result of (*sql.DB).Close is discarded (§6, errcheck)
warning api/auditlog/store_test.keel:30: unhandled error: the error result of (*sql.DB).Close is discarded in the t.Cleanup function (§6, errcheck)

## Ambiguities (not counted)

1. api/auditlog/store.keel:83: `rows.Scan(&e.Order, &e.Customer, &e.Kind, &e.Amount, &e.At)` writes the fields of shop.AuditEntry, which has `invariant Amount >= 0`, through pointers, outside any method of AuditEntry (it has none).
   - Reading A: §8.2 "A write from anywhere else is a compile error" covers a write through a field address. Then this is an error.
   - Reading B: the compiler does not track writes through a reference (the stance §5.4a rule 1 takes for library vars). Then there is no diagnostic.
2. api/web/web.keel:30, 40, 48, 56, 64: each `if err != nil` branch passes the error to the HTTP client (`http.Error(... err.Error() ...)` or `writeErr(w, err)`) and returns nothing.
   - Reading A: §6 error masking is literal: "neither returns it, logs it, nor wraps it". Then it reports 5 warnings.
   - Reading B: sending the error to the client handles it. Then there is no warning.
3. api/web/web.keel:87: `_ = json.NewEncoder(w).Encode(v)`.
   - Reading A: §6 "an error result that the code discards" covers `_ =`. Then it is a warning.
   - Reading B: "as errcheck reports". By default errcheck does not report a blank assignment (only with `-blank`). Then there is no warning.
4. api/auditlog/store.keel:37/115 vs 49, 54, 68, 73: the package var `db` is written by AuditLog's startup and read by Store methods. Web's startup can serve requests at the same moment, because main runs all startup functions at once (§6). There is no lock.
   - Reading A: §6 "as the race detector reports in tests". The translated tests pass under `-race`. Then there is no warning.
   - Reading B: a static "shared field accessed without a lock". Then it is a warning. (`db` is also a package var, not a field.)
5. Near-duplicate code (§10) has no threshold in the spec. `dupl` finds the three handler closures in api/web/web.keel:46-53, 54-61, 62-69 to be clones inside one leaf. It finds no clone across leaves. If those closures count, there is one in-leaf K0412-style warning. If they do not, there is none.
6. keel.DeadLetter[T] (topics.lock:13-22) has `Wire string` with no limit. §5.5.1 says direct "requires topic message types to be bounded with `limit` fields".
   - Reading A: the rule applies to the runtime type too. Then every direct dead-letter route fails.
   - Reading B: the spec defines the type, so it is exempt. This is a spec issue, not caused by the writer. No diagnostic.
7. Possible nil dereference (§6, "as nilaway reports"): nilaway is not installed. Manual review found none: `db` is nil-checked before each use, and `rows` is used only when err == nil. I could not confirm whether nilaway narrows a nil check on a package-level var.

## Known noise

- keel.dev/keel v0.1.0 has no go.sum entry (go.mod:6). The module is unpublished, and the base has the same gap. For the same reason, the network-package status of keel.dev/keel (§5.4c rule 1) cannot be resolved.

## Checked and clean

- Architecture: directories agree with system.keel (§5.2). Every required port and subscribed topic is wired exactly once, and Alerts.DeadShipped is wired by `deadletters:` (§5.4 rule 4). The only cross-container wire is a topic (rule 3). There are no cycles: Web->Orders->Billing->AuditLog, and Web->AuditLog (rule 8). Port signatures use only lib/shop types (rule 9). No leaf references another leaf (rules 1, 7).
- Wires (§5.3, §5.4 rules 5 and 6): Orders.NewService, AuditLog.NewStore (x2) and Billing.NewLedger return pointer types whose method sets match. The provider contracts are identical to the port contracts, so the implications hold.
- Gateway/entry (§5.4b): AuditLog is the one gateway for the declared external AuditDB in Api. Web is the entry for the declared person Customer.
- Network/acquire (§5.4c): only Web (net/http, net/http/httptest) and AuditLog (database/sql, pgx) import network packages. `go list -deps` confirms that log, testing, encoding/json, bufio, os and the rest have no net/os/exec/database/sql dependency. The library calls no acquiring function.
- Library (§5.4a): lib/shop has no ports or topics. No package var is reassigned. It imports only errors and time.
- Topics (§5.5 to §5.5.2): both handlers are `idempotent` (direct is at-least-once). Each subscribed topic has one handler. OrderShipped fields carry `limit`. `deadletters` is present. Picking.Shipped has a subscriber (no unread-topic warning). topics.lock matches the current OrderShipped (field names, types, limits, no invariants) and the dead-letter route.
- Contracts (§8): each `result` use is guarded by `err == nil =>`. All expressions are tier 1. The zero values of record, account, Receipt and AuditEntry satisfy their invariants. Fields of invariant structs are written only in their own methods or in literals (except ambiguity 1).
- Type-check: I translated the project to Go in scratchpad/check1/tc. I stripped contracts, invariants and `limit`, renamed handle/startup, declared ports as package interface vars, stubbed keel.dev/keel and pgx, and added a main that does the four wire assignments. `go build`, `go vet` and `go test -race` passed (the Postgres tests skip without AUDIT_DATABASE_URL).

## Total

1 error, 5 warnings (7 ambiguities and 1 known-noise item not counted).
