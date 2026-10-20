# Development plan: compiler and application proof

Status: draft for discussion, 2026-10-19.

## Objective

Build a working Vulcan compiler and use it to build and maintain a substantive application.
Use that experience to decide whether Vulcan merits continued development as a language.

The lack of a compiler is the reason to run this experiment, not evidence against the project.
The existing writing experiments inform the design but do not establish its benefit. The
decisive evidence must come from real compilation, execution, debugging, and application changes.

This plan proposes an implementation sequence. It does not change the specification or settle
open design questions. Changes to normative rules require an explicit decision and spec update.
Historical experiment fixtures remain unchanged; new experiments get their own directories.

## Scope of the first proof

The proposed first release supports one process with multiple components, consumer-owned ports,
libraries, explicit wiring, gateways, contracts, and generated startup and shutdown code. Bodies
retain Go semantics. The compiler is written in Go and emits Go with source locations preserved.

It includes enough agent-facing tooling to test the language's proposition: check, build, test,
guide, context, query, and machine-readable diagnostics. Command spelling and module paths remain
open until milestone 0.

Defer distributed topics, remote ports, custom durable transport, automatic duplicate extraction,
LLVM, self-hosting, and full editor integrations. Reject unsupported constructs explicitly.
Their deferral does not validate their design; it isolates the first implementation experiment.

The spec currently requires a custom incremental query engine from the start (§11.1). Proposed
change: start with explicit compiler stages and reusable analysis results, and add incremental
execution after measuring rebuild costs. This change needs a recorded decision before
implementation.

## Milestone 0: define guarantees and resolve blockers

Write short design decisions and update the relevant spec and primer sections together.

1. **Contract timing.** Define what postconditions and invariants guarantee when control returns
   to the caller. Cover named results, deferred mutation, deferred unlocking, panic, recovery,
   aliases, and concurrent access. The spec now checks before deferred calls run, does not check
   a deferred change to a named result, and lists such functions in `keel audit` (§7.2, Appendix
   F step 3). Thus a contract can pass and the caller can still receive a value that breaks it.
   Preferred initial direction: reject deferred mutations that can invalidate a checked result
   or invariant, including indirect mutation that cannot be excluded. Validate that this
   restriction still permits ordinary defer-based cleanup. If it requires impractical analysis,
   resolve instrumentation or scope before claiming the guarantee.
2. **Architecture enforcement.** List each guaranteed property and its limits. Distinguish
   checked dependency edges from complete runtime information flow. Decide whether mutable
   library state and unresolved dynamic calls are rejected, explicitly permitted, or reported.
   Preferred direction: an opt-in strict profile rejects unchecked paths; ordinary mode reports
   them without claiming a complete architecture proof. Destination identity is not inferred
   from a gateway label.
3. **Contract failure.** State that a panic does not roll back external effects. Validation that
   protects a side effect must happen before the side effect. Document the responsibility of
   application transactions and idempotency keys.
4. **Compiler structure.** Decide the incremental-engine proposal above. Keep parsing, name
   resolution, type checking, architecture analysis, contract instrumentation, and Go lowering
   separate enough to test independently. Avoid building a second backend abstraction until it
   has a concrete requirement.
5. **Names and reproducibility.** Decide extension, CLI name, runtime module path, supported Go
   version, and host platforms. Preserve the old experiment names as historical records.
6. **Evidence protocol.** Define application tasks, evaluation rules, measurement methods, and
   acceptance thresholds before evaluation begins. Separate feasibility, usability, and
   comparative productivity claims.

Exit gate: every implemented guarantee has a precise rule and an acceptance or rejection example.
No decision may silently weaken the promise while retaining its stronger description.

## Milestone 1: compile a vertical slice

Implement declarations, Go-body parsing, symbol resolution, type checking, and Go generation.
Support libraries, two or more leaves, consumer-owned ports, constructor wiring, and one entry
leaf. Generate lifecycle code and preserve source mappings in diagnostics and stack traces.
Provide check, build, test, and module dependency resolution for the supported subset.

Start with a small executable that calls through a port, uses a shared library type, and exits
cleanly. Exercise constructor failure and shutdown as well as success. Use the real compiler
in every test; do not strip language features or substitute handwritten generated code.

Exit gate: a clean checkout builds and tests the example with documented commands. Illegal
imports, missing wires, cycles, incompatible methods, and unsupported syntax fail with stable
diagnostics at the correct source location.

## Milestone 2: enforce the language's distinctive guarantees

Implement the agreed architecture rules and the contract subset needed by the application.
Include requires, ensures, old values, struct invariants, port wrappers, and contract-aware fakes.
Make unsupported contract expressions explicit errors. Add pure predicates only where required
and where their permitted effects are defined and tested.

Deliver guide, context, query, and JSON diagnostics. Context must expose the declarations and
contracts needed for a local change; query must return actual compiler symbols.

Build a conformance suite with both valid programs and deliberate violations:

- Deferred result and receiver mutations, indirect callbacks, aliases, panic, and recovery.
- Invariant checks with locks, nested methods, collection aliases, and external calls.
- Leaf imports, shared mutable library state, dynamic calls, gateways, and escape hatches.
- Integer boundaries and contract failures before and after external effects.
- Fake providers that violate a port contract and isolation between tests.
- Correct source positions, deterministic diagnostics, and clean dependency resolution.

Use differential tests against Go for supported ordinary Go bodies. Add parser fuzzing and
malformed-input tests. Run race detection in a configured CI environment even if a local host
cannot run it. Record instrumentation overhead and build time without claiming targets met.

Exit gate: the supported subset has executable conformance tests, and known holes are either
rejected or accurately exposed. No missing compiler feature is replaced with manual checking.

## Milestone 3: build and operate a substantive application

Proposed application: an order and inventory service with an internal charge/refund ledger,
Postgres persistence, an authenticated HTTP API, and an audit trail. It does not process real
payments. Use the old shop experiment as domain inspiration, not as a production foundation.

Build it with the actual compiler. It must exercise ports, libraries, gateways, contracts,
lifecycle management, agent context, and compiler-guided correction.

Application acceptance requirements:

- Store order, ledger, operation identity, and audit changes in a coherent transaction model.
  Retrying after a committed operation with a lost response must recover its recorded outcome.
- Persist identifiers and state across restart; never mix unrelated orders' audit histories.
- Validate amounts with overflow-safe arithmetic before writes. Test maximum and minimum
  values, partial refunds, concurrent refunds, duplicate requests, and rejected operations.
- Enforce authentication and authorization for each order and audit operation. Return safe
  public errors and retain useful internal diagnostics.
- Bound requests and audit queries. Avoid a process-wide mutex across database operations.
  Define timeouts, cancellation, readiness, shutdown, and resource cleanup.
- Run real database tests in CI. An absent database must fail the required integration job,
  rather than turn the evidence into a skipped test.

Inject failures before commit, after commit but before response, during startup, during shutdown,
and across restart. Verify both returned outcomes and persisted state. Run a representative
concurrency and load test. Test migrations and recovery on disposable databases.

Exit gate: the application survives these tests and can be built, run, diagnosed, and changed
from documented instructions without its author's intervention.

## Milestone 4: evaluate development and maintenance

Use several tasks on the application, not just its first construction: a new feature, a boundary
refactor, a dependency update, a failure-handling change, and a deliberately invalid change.

Compare against a functionally equivalent Go application with the same written requirements,
tests, and reasonable architecture tooling. The Go comparison is an evaluation baseline, not a
prerequisite for implementing Vulcan. Keep model versions, task briefs, budgets, and environments
comparable. Use held-out tasks or counterbalanced ordering to reduce learning effects.

Record:

- Total time to an accepted change, including compiler fixes and human intervention.
- Review time, escaped defects, and architectural violations caught before review.
- False positives, workarounds, and occasions where developers bypassed the intended model.
- Build latency, runtime overhead, and agent context usage.
- Compiler and runtime maintenance effort separately from application effort.

Keep prompts, commits, compiler versions, raw logs, failures, and grading results under a new
experiments directory. Use independent or blinded grading where practical. Report variation
and sample size; a single successful task does not establish a productivity advantage.

Exit gate: evaluate against the thresholds agreed in milestone 0. Continue if the implementation
is trustworthy and repeated tasks show enough practical benefit to justify the measured burden.
Narrow the language if only some features earn their cost. Stop or pivot if ordinary Go achieves
the same outcomes with materially less total effort. Treat inconclusive results as inconclusive.

## Milestone 5: consider distributed features separately

Proceed only after the application establishes a need and the first evaluation supports further
investment. Prefer an existing transport behind the language's interface before implementing a
custom durable broker.

Before implementation, resolve peer authentication, encryption, request size limits, disk quotas,
crash recovery, acknowledgement persistence, delivery identity, retry behavior, and shutdown.
Label idempotent handlers as developer assertions unless the implementation provides a stronger
mechanism. Test repeated delivery around every side-effect and acknowledgement boundary.

Revise compatibility rules: removing a remote method is not safe for an old caller merely
because the schema remains readable. The §5.5.2 table now calls this change compatible. Specify
staged retirement and test mixed-version callers, providers, and rollback. Bound malformed messages and dead letters as well as valid messages.

Exit gate: fault-injection and mixed-version tests support each advertised delivery and
compatibility guarantee before those guarantees appear in user-facing documentation.

## Audit traceability

The findings below come from an external audit of the project; its report is not stored here.
[Audit 2](../experiments/audit-2.md) repeats that audit from the sources.

| Finding | Planned response |
|---|---|
| 1. Evidence not yet established | Real compiler and application in milestones 1-3; comparative evidence in milestone 4 |
| 2. Deferred mutations invalidate contracts | Semantic decision in milestone 0; conformance tests in milestone 2 |
| 3. Architecture enforcement has unchecked paths | Guarantee matrix in milestone 0; enforcement and escape tests in milestone 2 |
| 4. Distributed runtime obligations | Deferred from initial scope; separate design and failure tests in milestone 5 |
| 5. Audit and ledger inconsistency | Transactional state and ambiguous-commit tests in milestone 3 |
| 6. Refund overflow | Boundary conformance tests in milestone 2; safe application arithmetic in milestone 3 |
| 7. Excessive initial scope | Small compiler release; explicit decisions on deferred infrastructure |
| 8. Remote-method removal breaks old callers | Retirement protocol and mixed-version testing in milestone 5 |
| 9. Operational and resource weaknesses | Security, persistence, bounded I/O, and concurrency tests in milestone 3 |

## Immediate next deliverable

Produce the milestone-0 decision packet and compiler design, including the supported syntax,
guarantee matrix, first conformance cases, and application acceptance criteria. Estimate the
implementation after this scope is concrete. Do not attach a full-language delivery date to the
current specification.
