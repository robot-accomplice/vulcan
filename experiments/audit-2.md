# Audit 2: specification, evidence, and test 3 code

Date: 2026-10-20. Commit audited: `665af46`.

This audit repeats the external audit (audit 1) that the
[development plan](../docs/development-plan.md) answers. The report of audit 1 is not stored, so
this audit starts again from the sources. It covers:

1. `docs/spec.md` and `docs/guide.md` as a language design: the contract, architecture, and
   distributed guarantees, and the feasibility of the scope.
2. Whether the evidence supports the premise in spec §1.1.
3. The Go code in `experiments/test3/work/go` (and `base-go` where they are the same): correctness,
   overflow, ledger and audit consistency, security, resource bounds, and concurrency. The Keel
   version (`work/keel`) is used for comparison.

The spec still uses the working name Keel. This report uses the spec's names (`keel audit`,
`.keel`).

## Executive summary

The spec is careful and honest about its limits, but three of its central promises are weaker
than its text says. A deferred `recover` returns from a function with no contract check. The
network rule depends on how leaves are labelled, and it does not follow calls through function
values. The `direct` transport has unauthenticated peers and defaults that let a dead subscriber
stop its publisher. The premise rests on experiments whose data is lost, and on a test that is not
finished. The test 3 Go code has a critical data-integrity defect: after a restart, audit trails of
different orders mix. **Fix first:** do not call the premise a finding until the experiments in
milestone 4 of the development plan can show it. In the code, make order IDs unique across
restarts.

## Findings by severity

### CRITICAL

1 finding

- **C1. Audit trails of different orders mix after a restart.** `work/go/orders/orders.go:66-67`
  makes IDs `o-1`, `o-2`, and so on from an in-memory counter. `work/go/audit/postgres.go:42-50`
  stores audit lines in Postgres, which persists. After a restart, the new `o-1` shows the charges
  and refunds of the old `o-1` (`GET /orders/o-1/audit`, `web/web.go:73-86`). The audit line of the
  old order is also unreachable, because `web.go:76` returns 404 for an order that is no longer
  in memory. The writer reported this (`results/go-writer-notes.md`, open issue 1) and shipped
  it. **Remediation:** make order IDs unique across restarts (UUIDs or a database sequence). Store
  orders in the same database as the audit lines. Add a test that restarts the service on one
  database and checks that each trail holds only its own order's lines.

### HIGH

11 findings

**Code (test 3)**

- **H1. Refund check overflows.** `work/go/billing/billing.go:103`:
  `if e.refunded+amount > e.charged`. With `refunded = 50` and `amount = math.MaxInt64 - 10`, the
  sum wraps to a negative number, the check passes, and `refunded` becomes negative. The Keel
  version has the same body (`work/keel/api/billing/ledger.keel:27`). There, the exact-arithmetic
  contract on `account.refund` (`ledger.keel:36`) and the invariant `refunded <= charged` turn the
  silent overflow into a panic. That panic comes after the audit line is written (`ledger.keel:94`,
  then `:97`), so the audit shows a refund that never happened. Today only a direct caller of
  `Ledger.Refund` can reach this: `Cancel` passes `o.Amount`, which is never more than the charge.
  `POST /orders` also accepts any positive `int64` (`web.go:40-55`, `orders.go:61`) with no upper
  bound. **Remediation:** write the check as `amount > e.charged - e.refunded`, which cannot
  overflow because both values are at least 0. Set a maximum order amount. Add tests at
  `math.MaxInt64`.
- **H2. A lost reply leaves a phantom audit line.** `billing.go:79` and `:107` write the audit line
  first. If the `INSERT` commits but its reply is lost, or `AuditTimeout` (`billing.go:17`) expires
  after the commit, Billing returns `ErrAudit` and does not change the ledger. The audit table then
  records a charge that the ledger and the caller say did not happen. This breaks the intent of
  hidden rule 4. The writer reported it (open issue 3). **Remediation:** keep the ledger in
  Postgres and write the ledger change and the audit line in one transaction, with an operation ID
  that the caller supplies. On an unclear result, read the operation ID back before you report.
- **H3. Placing an order is not idempotent.** `POST /orders` (`web.go:40`) has no idempotency key.
  A client that retries after a timeout places and charges a second order. **Remediation:** accept
  an `Idempotency-Key` header. Store it with the order in the same transaction. Return the stored
  result on a repeat.
- **H4. No authentication or authorization.** Any client can cancel any order and get the refund
  (`web.go:57`). Any client can read any order's audit trail, with customer IDs and amounts
  (`web.go:73`). Order IDs are sequential, so they are easy to guess. **Remediation:** authenticate
  each request. Check that the caller owns the order, or has an operator role for the audit route.
  Use IDs that cannot be guessed.
- **H5. One slow database call blocks the whole service.** `Place` holds `orders.Service.mu`
  (`orders.go:64-68`) while it calls `Charge`. `Charge` holds `Ledger.mu` (`billing.go:77-79`)
  while it waits for Postgres, for up to 5 s. During that wait, every `Get`, every `Cancel`,
  `HandleShipped`, and the audit route (which calls `Get` first, `web.go:76`) wait for the same
  lock. **Remediation:** do not hold a process-wide mutex across I/O. With the transactional design
  of H2, use row locks in the database, or lock per order.

**Specification**

- **H6. A deferred `recover` goes around every contract check.** §7.2 and Appendix F step 3 check
  `ensures` and the invariant at each `return`, before deferred calls run. They also state that "a
  panic reaches no `return`". In Go, a deferred function that calls `recover` makes the function
  return normally to its caller, with named results that the deferred function sets. Thus a method
  that panics halfway through an update, and recovers in a `defer`, returns to its caller with no
  `ensures` check and no invariant check. The struct can then break its invariant, and nothing
  reports it. `keel audit` lists only deferred calls that change a named result (spec line 2007).
  It does not list recovery, and the spec does not mention this path. This is the core promise of
  §7 ("is true after the outermost method call returns"). **Remediation:** after a recovered panic
  in a function with a contract or in a method of a struct with an invariant, check the contract
  again on the deferred return path, or reject `recover` in such functions. Add the case to the
  conformance suite (development plan, milestone 2).
- **H7. The network rule depends on labels, and it has unchecked paths.** §5.4b says that "the
  compiler rejects network access in any other leaf, so the reviewer does not have to find it".
  The rules allow these paths:
  - An entry leaf can reach every network function, and "the compiler does not tell serving from
    dialing" (§5.4b). Any leaf marked `entry for` can dial anywhere.
  - A gateway's target is not checked (§5.4b rule 3). A gateway for Stripe can call any host.
  - §5.4c does not follow a call through a function value or an interface value beyond two
    conversion steps. A gateway can give a port method that returns `func(host string, body []byte)
    error`. A domain leaf can then choose any network target through it, and no diagnostic shows
    the edge.
  - A library can hold state that two leaves share, through methods and references (§5.4a rule 1).
    This is a channel between leaves that does not go through a port. `keel audit` lists it, but
    the compiler accepts it.

  Each limit is stated somewhere in the spec, but the summary claims in §1.1, §5.4b, and the README
  do not mention them. **Remediation:** define a guarantee matrix (development plan, milestone 0,
  item 2). Say what the compiler proves (static dependency edges) and what it only reports.
  Consider a strict mode that rejects function values and interfaces that come from a gateway, and
  rejects mutable library state.
- **H8. Removing a remote method is called compatible, but it breaks old callers.** The §5.5.2
  table (spec line 940) lists "Remove a method from a remote port | Compatible". The section also
  says "any deploy order works". If the provider deploys without the method while an old consumer
  build still calls it, each call fails at runtime. **Remediation:** make removal a staged change.
  First remove the method from all consumer builds and record that in `topics.lock`. Remove it from
  the provider only after that. Then state the one deploy order this needs, as the table already
  does for an added method.
- **H9. The `direct` transport has no peer security, and its defaults couple availability.**
  §5.5.1 sends messages and calls over plain TCP to the addresses in `--transport.direct.peers`,
  and listens on `:7400`. The spec does not mention authentication, encryption, or which peers can
  connect. Any host that can reach the port can publish messages and call remote ports. The
  defaults are also unsafe together: `whenFull: block`, no `maxWait`, no `ttl`, `capacity`
  10,000, and `durable: true`. When a subscriber's container is down, its queue fills, and then
  every publish to that wire blocks the publisher with no limit. The spec says a publisher "does
  not wait for a subscriber", but with these defaults it does. Durable queues have a memory bound
  but no disk bound. **Remediation:** require mutual TLS or a shared-key handshake for `direct`.
  Make `maxWait` or a drop policy a required setting, or default it to a bounded value. Add a disk
  quota. The development plan defers distributed features (milestone 5). Until then, mark §5.5.1
  as not ready to implement.
- **H10. The premise rests on evidence that cannot be checked.** Spec §1.1 states three
  conclusions from "two experiments" (2026-09-29). `docs/status.md` says that the data of
  experiments 1 and 2 was lost. Test 3, the only stored experiment, is not finished: round 2 of the
  check and all grading are not done, and the Keel base is older than reviews 8 and 9
  (`status.md`, "Test 3"). Its `keel check` was done by hand, not by a compiler
  (`results/check-round1.md`). Each language had one writer, so the sample size is one. The §3
  table cites good external sources, but they show the problems, not that Vulcan solves them.
  **Remediation:** in §1.1, call the three bullets hypotheses until a compiler-based experiment
  supports them. Keep the raw data of each new experiment in `experiments/`. The development plan,
  milestone 4, gives a suitable protocol.
- **H11. The committed scope is much larger than a first implementation can test.** The spec
  requires, before any code exists: a custom incremental query engine "from the start" (§11.1), an
  LSP server and an MCP server (§10), C4 diagram generation, `archdiff`, near-duplicate detection,
  reachability analysis over all dependency modules (§5.4c), a proto3 wire format with a schema
  lock, a durable transport with dead letters, tracing, and later self-hosting and an LLVM backend.
  The spec is 2,054 lines and has had nine reviews, but there is no compiler. More reviews of
  unimplemented text find fewer problems each time (55, then 29), but they cannot find the problems
  that only an implementation shows. **Remediation:** accept the scope cut in the development plan
  (one process, no topics or remote ports, staged compiler). Mark the deferred sections of the spec
  as deferred in the spec itself.

### MEDIUM

8 findings

- **M1. Error responses show internal details.** `web.go:100` sends `err.Error()` for every
  error, including the 500 case. A database failure sends the driver's error text and the SQL
  context of `audit/postgres.go:47` to the client. `web.go:43` repeats the JSON decoder's error.
  **Remediation:** send a fixed message for 500 errors, and log the full error with a request ID.
- **M2. No HTTP limits.** `cmd/api/main.go:61` uses `http.ListenAndServe`, which has no read,
  write, or idle timeouts. `web.go:42` decodes a request body of any size. A slow or large client
  can hold connections and memory. **Remediation:** use an `http.Server` with
  `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`. Wrap the body in
  `http.MaxBytesReader`.
- **M3. The audit query has no bound.** `audit/postgres.go:53-72` and `audit/audit.go:45-55`
  return all lines for an order with no limit or pagination. **Remediation:** add a limit and a
  cursor (the `id` column) to `ForOrder` and to the route.
- **M4. The only database code has no test that runs.** `audit/postgres_test.go:24-26` skips
  when `SHOP_TEST_DATABASE_URL` is not set. `go test -cover` gives `shop/audit` 0.0%. `cmd/api`
  and `cmd/worker` also have 0.0%. A green test run says nothing about Postgres. **Remediation:**
  run the Postgres test in CI against a real database, and fail the job when the database is
  missing.
- **M5. No shutdown, health, or readiness.** `cmd/api/main.go:61` ends with `log.Fatal`, which
  exits without running the deferred `db.Close` (`:39`). The service has no readiness check,
  so a load balancer cannot see that the database is down, which stops every order (fail closed,
  writer open issue 2). **Remediation:** handle `SIGTERM` with `Server.Shutdown`. Add `/healthz`
  and `/readyz`, and make readiness ping the database.
- **M6. The invariant guarantee is narrower than the §7 table says.** The table says that an
  invariant "is true after construction and after the outermost method call returns". The rules
  in §7.2 then exclude elements of slice and map fields, work in goroutines (K0520), and recovered
  panics (H6). They also assume that no other goroutine writes the fields during the check: the
  spec covers `defer mu.Unlock()`, but not a method that does not lock. **Remediation:** state
  these conditions in the §7 table row itself, and in the primer.
- **M7. "Remove a field" depends on a check that the spec does not define.** The §5.5.2 table says
  that an older subscriber reads the zero value of a removed field, and that "its invariant must
  accept it". The spec does not say that the compiler checks this. The old invariant is in
  `topics.lock`, so the compiler could evaluate it with the field at its zero value.
  **Remediation:** define that check, with a diagnostic code in Appendix E.
- **M8. The in-memory bus retries permanent errors at once.** `bus/bus.go:37-41` delivers again
  immediately, five times, with no backoff. `orders.HandleShipped` returns `ErrNotFound` and
  `ErrCancelled`, which no retry can fix. The worker's bus (`cmd/worker/main.go:18`) is not
  connected to the API process, so shipments never reach Orders. The comment says this is a
  stand-in. **Remediation:** this is acceptable for a fixture. Mark permanent errors so that they
  go to the dead letter at once, and add backoff, if the fixture is used again.

### LOW

3 findings

- **L1. Typo in the §10 table.** Spec line 1385: "Builds and runs the tests of one leaf (§8).,
  with the race detector (§6)." **Remediation:** remove ".,".
- **L2. Uneven line wrapping in §5.4 and §5.3.** Rules 1, 3, and 9 of §5.4 and a paragraph of
  §5.3 have lines much longer than the rest of the file, left over from edits. **Remediation:**
  rewrap when you next edit these sections.
- **L3. The rename is not done.** The spec, the primer, and Appendix E still use Keel, `.keel`,
  `keel`, and `keel.dev/keel` (Q1). New readers see two names. **Remediation:** decide the three
  open names (development plan, milestone 0, item 5), and rename in one change.

## What is done well

- The spec states most of its limits where it makes a rule, and lists each gap in `keel audit`.
  Few drafts are this candid.
- Each review is stored with a resolution file that maps each finding to its fix.
- The Go writer's notes list the critical and high defects of their own code (C1, H2, H4, M4).
  The process found these problems. The process did not stop them from shipping.
- The billing tests cover the rejected-refund and failed-audit paths (`billing` 97.1% coverage),
  and all tests pass under `-race`.
- Fail closed on audit failure (`billing.go:43-45`) is the right default for a money trail.

## Metrics snapshot

- Files audited: 2 spec documents (`spec.md` 2,054 lines, `guide.md` 173 lines), the test 3
  results, 30 Go files (1,741 lines), and the Keel billing leaf.
- God files (more than 500 lines): 1 (`docs/spec.md`; a specification, so this is expected).
- God functions (more than 50 lines): 1 (`web.Handler`, 50 lines of route closures).
- Test coverage (`work/go`): billing 97.1%, picking 100%, bus 88.9%, orders 83.7%, web 73.9%,
  audit 0.0%, commands 0.0%.
- Critical-path coverage: the money paths are covered in memory. The Postgres path is not covered.
- Known security issues: 4 (H4, H9, M1, M2).
- Idempotency gaps: 2 (H3; and H2, where a retry after an unclear audit write repeats a charge).
- Documentation rating: **Incomplete.** The spec is detailed but overstates three guarantees (H6,
  H7, H9) and presents unchecked evidence as fact (H10).

## Remediation roadmap

### Immediate

- C1: unique order IDs in `work/go/orders` if test 3 continues. Otherwise, record the defect in the
  test 3 grading.
- H10: change spec §1.1 from findings to hypotheses.

### Short term

- H6, H7, H8, H9, M6, M7: write each into the milestone-0 decision packet of the development plan.
  Each one needs a precise rule and an accept or reject example.
- H11: mark the deferred parts of the spec (§5.5 transports, §11.1 incremental engine, LLVM,
  self-hosting) as deferred in the spec itself.

### Medium term

- H1 to H5 and M1 to M5: make these the acceptance criteria of the milestone-3 application. The
  development plan already lists them.
- L1 to L3: fix them with the rename.

### Ongoing

- Keep the raw data of every experiment and audit in `experiments/`. Audit 1 was lost because it
  was not stored.
- Run each check with a real compiler as soon as one exists, and stop review-only rounds of the
  spec.

## Architectural recommendations

- **Separate what the compiler proves from what it reports.** The spec mixes the two, so the
  summary claims are stronger than the rules. A guarantee matrix, and a strict profile that turns
  each reported gap into an error, fix this at the root.
- **Put transactional state in the application, not in the language.** H2 and C1 show that a
  consistent audit needs one database transaction. Contracts cannot give that. §7 states this for
  external effects, and the primer should say it too.
- **Build the smallest compiler that can falsify the premise.** Most high findings are about
  promises that only an implementation can test. The development plan's milestones 1 to 4 are the
  shortest path to evidence.
