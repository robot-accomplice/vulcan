# Release 1: application acceptance criteria

Status: accepted, 2026-10-21. Part of the milestone-1 scope of the
[development plan](../development-plan.md). The application is built in milestone 3 and evaluated
in milestone 4 ([decision 6](../decisions/0006-evidence-protocol.md)).

## The application

**Stockroom** is an order and inventory service. A customer places, pays for, and cancels orders.
An operator manages stock and reads the audit trail. An internal ledger records each charge and
refund. It does not process real payments.

It is built twice, with the same HTTP API and the same tests: once in Vulcan (strict mode), and
once in Go. The Go version is arm R of decision 6, with Vulcan's rules as static analyzers, and its
labelling copy is arm C. (Arm P runs only in milestone 0.5, on a smaller service.) Every criterion
below applies to both versions, unless it says "Vulcan arm". "Both arms" below means both versions.

### Shape

| Part | Vulcan | Go arm |
|---|---|---|
| HTTP API | Entry leaf `Web` (`entry for Customer`, `entry for Operator`) | Package `web` |
| Orders | Leaf `Orders` | Package `orders` |
| Inventory | Leaf `Inventory` | Package `inventory` |
| Ledger | Leaf `Ledger` | Package `ledger` |
| Audit trail | Leaf `Audit` | Package `audit` |
| Database | Gateway leaf `Store` (`gateway for Postgres`) | Package `store` |
| Shared types | Libraries `money`, `ids`, `authz` | Same packages |

The domain leaves need ports; only `Store` opens the database. One container, one binary.

## Criteria

Each criterion has an ID, so that a test, a task, or a review can name it. Each one is checked by
an automated test unless it says otherwise. The **From** column names where it comes from: a
finding of [audit 2](../../experiments/audit-2.md) or a decision.

### Function

| ID | Criterion | From |
|---|---|---|
| F1 | `POST /orders` places an order of up to 50 lines, reserves stock for each line, and charges the order total, all in one transaction. | |
| F2 | `POST /orders/{id}/cancel` cancels a placed order, releases its stock, and refunds the charged amount. A shipped order cannot be cancelled. | |
| F3 | `POST /orders/{id}/refunds` refunds part of an order. The sum of refunds never exceeds the charge. | H1 |
| F4 | `GET /orders/{id}` and `GET /orders?cursor=` return orders. The list returns at most 100 orders for each page. | M3 |
| F5 | `GET /orders/{id}/audit?cursor=` returns the audit lines of one order, oldest first, at most 100 for each page. | M3 |
| F6 | Operators manage stock: `PUT /stock/{sku}` sets a level; `GET /stock/{sku}` reads it. | |

### Data integrity

| ID | Criterion | From |
|---|---|---|
| D1 | Each change writes its state, its ledger entry, and its audit line in one database transaction. A test that fails each statement in turn finds no state where these three disagree. | H2, decision 3 |
| D2 | Order IDs, ledger entries, and audit lines are unique across restarts. A test restarts the service on one database and checks that each order's audit trail holds only its own lines. | C1 |
| D3 | Each `POST` takes an `Idempotency-Key` header. The key and the response are stored in the change's transaction. A repeat with the same key returns the stored response and changes nothing. A repeat with the same key and a different body is rejected with 422. | H3, decision 3 |
| D4 | A lost response is recoverable: a test commits a change, drops the response, and retries with the same key. The retry returns the original outcome, and the ledger has one entry. | H2 |
| D5 | Amounts are `int64` cents. Each guard is written so that it cannot overflow. Tests cover 0, 1, the maximum order total, `math.MaxInt64`, and negative values for each money input. | H1, decision 3 |
| D6 | Two concurrent refunds that together exceed the charge: exactly one succeeds. Two concurrent orders for the last unit of stock: exactly one succeeds. | H1, H5 |
| D7 | Migrations are versioned files that run once. Running them on a database at the current version changes nothing. A test applies them to an empty database, and to a copy at the previous version. | |

### Security

| ID | Criterion | From |
|---|---|---|
| S1 | Every route requires a bearer token. A request without a valid token gets 401. | H4 |
| S2 | A customer can read, cancel, and refund only their own orders, and gets 404 for any other order. An operator can read every order and audit trail. Stock routes need the operator role. | H4 |
| S3 | IDs cannot be guessed: random, with at least 122 bits. | H4 |
| S4 | A 5xx response has a fixed message and a request ID. The full error goes to the log with that ID, never to the client. | M1 |
| S5 | Tokens, secrets, and database credentials come from flags or the environment (a `vulcan:"secret"` setting in the Vulcan arm), never from source. | §5.3 |

### Resource bounds and operation

| ID | Criterion | From |
|---|---|---|
| R1 | The HTTP server sets read-header, read, write, and idle timeouts. A request body is limited to 64 KiB. | M2 |
| R2 | Each database call has a timeout from its request's context. No process-wide lock is held across a database call. | H5 |
| R3 | `GET /healthz` reports that the process runs. `GET /readyz` pings the database and fails while it is down. | M5 |
| R4 | On `SIGTERM`, the server stops taking requests, finishes the running ones within `shutdown_timeout`, closes the database, and exits 0. | M5, §6 |
| R5 | A database outage makes write requests fail with 503 and a fixed message. Reads that need the database fail the same way. The service recovers without a restart when the database returns. | |

### Tests and evidence

| ID | Criterion | From |
|---|---|---|
| T1 | The integration tests run against a real Postgres in CI. If the database is missing, the CI job fails. It does not skip. | M4 |
| T2 | Fault injection covers a failure before commit, after commit and before the response, during startup, during shutdown, and across a restart. Each test checks both the response and the stored state. | Audit 2 (H2); decision 3 |
| T3 | A load test runs a fixed mix (70% reads, 20% orders, 10% cancels and refunds) for 10 minutes with 50 clients, and reports p50 and p99 latency and throughput. It is the benchmark of decision 6 (X11). | Decision 6 |
| T4 | The hidden acceptance tests of decision 6 use only the HTTP API, so that the same tests grade both arms. | Decision 6 |
| T5 | All tests pass under the race detector on a host that has it. | §6 |

### Vulcan arm only

| ID | Criterion | From |
|---|---|---|
| V1 | Strict mode is on. Each `strict_exempt`, `strict_interfaces`, and `strict_shared` entry is listed with a reason in the application's README. The target is no exemption. | Decision 2 |
| V2 | Ledger and inventory operations carry `requires` that guard their effects, and `ensures` and invariants that state their results. Each K0525 warning is reviewed and kept or fixed. | Decisions 1, 3 |
| V3 | `on_contract_failure` is `exit` for this container. | Decision 3 |
| V4 | `vulcan audit` runs in CI. Its "Unchecked paths" count and its "detects after effects" list are stored with each build. | Decisions 2, 3 |

## Exit gate of milestone 3

The Vulcan version passes all 32 criteria, and the Go version the 28 that apply to it, from a
clean checkout, with documented commands. Then a fresh agent session checks each version as the
development plan, milestone 3 describes. Three criteria need a person: D5 (each guard cannot
overflow), V1 (the reasons for exemptions), and V2 (each K0525 warning reviewed). The Go developer
of decision 6 checks them, not the author; on the Go version, the grader checks D5.

## Out of scope

Payments with a real provider, email, a user interface, multi-container deployment, topics, and
remote ports. A task in milestone 4 can add a gateway to an external service, which is one of its
tempting changes.
