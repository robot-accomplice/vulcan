# Development plan: compiler and application proof

Status: proposed, 2026-10-21. The owner accepts it only if it survives an antagonistic audit.
[Audit 3](../experiments/audit-3.md) (not independent) and [audit 4](../experiments/audit-4.md)
(independent) both found blockers, and so did [audit 5](../experiments/audit-5.md) (independent).
This version applies their amendments and waits for the next independent audit.

## Objective

Find out, at the lowest cost that gives a real answer, whether Vulcan merits continued
development as a language. Build a compiler and use it to build and maintain a substantive
application, but only after a cheaper test shows that the rules Vulcan enforces change outcomes
at all.

The lack of a compiler is the reason to run this experiment, not evidence against the project.
The existing writing experiments inform the design but do not establish its benefit. The
decisive evidence must come from real compilation, execution, debugging, and application changes.

Changes to normative rules need an explicit decision and a spec update. Milestone 0 made six such
decisions, and they changed the spec. Historical experiment fixtures remain unchanged; new
experiments get their own directories.

## Possible outcomes

The plan ends in one of six ways. The first five come from the fixed decision rule of
[decision 6](decisions/0006-evidence-protocol.md), section 10:

- **Continue the language:** Vulcan enforces rules that Go's static analyzers cannot check, that
  changes outcomes, and it costs no more than 10% in task success or working time.
- **Narrow:** as above, but a secondary bound fails, or contract rules carried the result (a Go
  code generator might deliver those). Keep the language features behind the rules that helped,
  and drop or redesign the rest.
- **Pivot to the rules:** the rules help, and Go analyzers deliver them. Continue as a Go tool, not
  as a language.
- **Stop:** enforcing the rules does not change outcomes, or nothing above applies.
- **Stop, not judged:** the independent roles of decision 6, section 3 were not filled, or the
  evaluation could not finish.
- **Stopped by schedule:** a milestone overran its time box twice.

## Scope of the first proof

Release 1 supports one process with several leaves, consumer-owned ports, libraries, explicit
wiring, gateways and entry leaves, contracts, and generated startup and shutdown code. Bodies keep
Go semantics. The compiler is written in Go, as a staged compiler (decision 4), and emits Go with
source locations preserved. The full scope is in [release-1/syntax.md](release-1/syntax.md).

It includes enough agent-facing tooling to test the language's proposition: check, build, test,
guide, context, query, audit, and machine-readable diagnostics. The names are fixed by decision 5:
`.vulcan` files, the `vulcan` command, and the runtime module
`github.com/robot-accomplice/vulcan/vulcan`.

Topics, remote ports, transports, near-duplicate detection, LLVM, self-hosting, and editor
integrations are deferred. Release 1 rejects each deferred source construct with K0009, and each
deferred command with exit code 2 ([release-1/syntax.md](release-1/syntax.md)). Their deferral does
not validate their design; it isolates the first implementation experiment.

## Time boxes

Each milestone from 0.5 to 4 has a time box in calendar weeks, counted from its start date in
`docs/status.md`. When a milestone runs over its box by more than half, work stops for a re-plan:
a written note in `docs/status.md` that says why, what changes, and one new box. If the milestone
then overruns the new box, the project ends as **stopped by schedule**. The run adjudicator of
decision 6 signs each overrun.

Recruiting the roles of decision 6, section 3 comes before milestone 0.5 and has its own deadline:
**8 weeks** from the plan's acceptance. If the roles are not filled by then, the project ends as
"stop, not judged". Milestone 0.5's box starts when the roles are filled.

| Milestone | Box |
|---|---|
| 0.5 Rules without a compiler | 6 weeks |
| 1 Vertical slice | 6 weeks |
| 2 Guarantees | 10 weeks |
| 3 Application | 8 weeks |
| 4 Evaluation | 6 weeks |

## Milestone 0: define guarantees and resolve blockers (done)

Done on 2026-10-20 and 2026-10-21: [decisions 1 to 6](decisions/README.md), each applied to the
spec and the primer.

1. Contract timing: checks after deferred calls and recovered panics; a panic that leaves a
   method with its invariant false becomes a contract failure (spec §7.3).
2. Architecture enforcement: the guarantee matrix, ordinary and strict modes (spec §5.4d).
3. Contract failure: nothing is undone; the application's transaction and idempotency
   responsibilities (spec §7.4).
4. Compiler structure: six cached stages that reuse `go/parser` and `go/types` (spec §11.1).
5. Names and reproducibility (spec §6, §11, Q1 closed).
6. Evidence protocol: three arms, independent roles, fixed thresholds, a fixed decision rule.

Exit gate (met): every implemented guarantee has a precise rule and an acceptance or rejection
example. No decision silently weakens a promise while keeping its stronger description.

## Milestone 0.5: the rules without a compiler

H-A and H-B (spec §1.1) are claims about rules, not about syntax. So test the rules first, in the
cheapest form, and find out whether a language is needed at all:

- Start only when the roles of decision 6, section 3 are filled (see "Time boxes").
- Write arm R of decision 6: static analyzers and one configuration file, nothing else, within the
  Go developer's 15-day cap. The Go developer signs that the coverage is complete.
- Build a small order service in Go (the test 3 shop, rebuilt with a database is enough).
- Run the reduced protocol of decision 6, section 12: arm R against arm P, eight tasks, six of
  them tempting, with hidden rules from the same rule-writer process as milestone 4.

Exit gate, in two parts (decision 6, section 12), signed by the run adjudicator:

1. If the rules do not help (T1 fails), **stop**.
2. If fewer than 4 hidden rules need a language, or their violations number fewer than 5,
   **pivot to the rules** without building a compiler (or stop, if arm R costs too much against
   arm P).

Otherwise continue to milestone 1. The analyzers become arm R of milestone 4.

## Milestone 1: compile a vertical slice

Implement declarations, Go-body parsing, symbol resolution, type checking, and Go generation.
Support libraries, two or more leaves, consumer-owned ports, constructor wiring, and one entry
leaf. Generate lifecycle code and preserve source mappings in diagnostics and stack traces.
Provide check, build, test, and module dependency resolution for the supported subset.

Start with a small executable that calls through a port, uses a shared library type, and exits
cleanly. Exercise constructor failure and shutdown as well as success. Use the real compiler
in every test; do not strip language features or substitute handwritten generated code.

Exit gate: a clean checkout builds and tests the example with documented commands. The milestone-1
cases of [release-1/conformance.md](release-1/conformance.md) pass: illegal imports, missing
wires, cycles, incompatible methods, and unsupported syntax fail with stable diagnostics at the
correct source location.

## Milestone 2: enforce the language's distinctive guarantees

Implement the architecture rules of decision 2 and the contract subset of release 1: requires,
ensures, old values, struct invariants, port wrappers, contract-aware fakes, and the rules of
decisions 1 and 3. Make unsupported contract expressions explicit errors (K0009). Add `pure`
predicates only where the application needs them.

Deliver guide, context, query, audit, and JSON diagnostics. Context must expose the declarations
and contracts needed for a local change; query must return actual compiler symbols.

Make the conformance suite of release 1 pass, and grow it with each bug. Use differential tests
against Go for supported ordinary Go bodies. Add parser fuzzing and malformed-input tests. Run
race detection in CI on a host that has it. Measure check and build latency against the targets
of decision 6, section 9, on a synthetic application of fixed size. A miss starts the incremental
work of decision 4, rule 4. Decide `pure` (release 1) from the written design of the milestone-3
application, which exists by the end of milestone 1.

Exit gate: every case of `docs/release-1/conformance.md` passes, or is marked as a known hole with
its K0009 or "Unchecked paths" entry. A script lists both. Someone other than the owner (the Go
developer of decision 6) checks the list against §5.4d and signs it.

## Milestone 3: build and operate a substantive application

Build Stockroom, the order and inventory service of
[release-1/acceptance.md](release-1/acceptance.md), in Vulcan with the compiler, and in Go for the
two Go arms of decision 6. The Go developer of decision 6 builds the Go version, or reviews it and
writes a list of changes that are applied before sealing. The Go developer also reviews the Vulcan
version for parity (decision 6, section 2).

Exit gate: the Vulcan version passes all 32 acceptance criteria, and the Go version passes the 28
that apply to it (V1 to V4 are for Vulcan only). Then a **fresh agent session**, with
only the repository and its README and no contact with the owner, builds and runs each version,
finds the cause of one fault that the grader of decision 6 injects (named in a sealed file), and
makes one small change that the grader specifies, from the documented instructions. The run
adjudicator compares the result with the sealed file and signs the gate. The transcript is kept
in `experiments/m3/`.

## Milestone 4: evaluate development and maintenance

Run [decision 6](decisions/0006-evidence-protocol.md): three arms (Vulcan; Go with the rules as
static analyzers; Go with the rules written down but not enforced), at least fourteen tasks sized by
simulation, written by a non-owner from sealed hidden rules, rules labelled for each arm by seeded
cases, a toolchain frozen before the rules exist, mandatory independent roles, exact permutation
tests on paired per-task differences, a committed budget, and sealed files kept from the owner and
the agents, with an outside timestamp.

Exit gate: the run adjudicator applies the decision rule of decision 6, section 10. Inconclusive
counts as fail. The result is one of the outcomes above.

## Milestone 5: consider distributed features separately

**Entry test.** Start only when milestone 4 ends in "continue" or "narrow", and the milestone-4
records show a task that the one-container design could not meet without a workaround that the
evaluation counted (M9), or without a second process.

Prefer an existing transport behind the language's interface before implementing a custom durable
broker. Before implementation, resolve peer authentication, encryption, request size limits, disk
quotas, crash recovery, acknowledgement persistence, delivery identity, retry behavior, and
shutdown. Label idempotent handlers as developer assertions unless the implementation provides a
stronger mechanism. Test repeated delivery around every side-effect and acknowledgement boundary.

Revise compatibility rules: removing a remote method is not safe for an old caller merely because
the schema remains readable, but the §5.5.2 table calls this change compatible. Specify staged
retirement and test mixed-version callers, providers, and rollback. Bound malformed messages and
dead letters as well as valid messages. Answer audit 2's findings H8, H9, and M7 here.

Exit gate: fault-injection and mixed-version tests support each advertised delivery and
compatibility guarantee before those guarantees appear in user-facing documentation.

## Audit traceability

| Source | Finding | Response |
|---|---|---|
| Audit 1 (external, not stored) | 1. Evidence not yet established | Milestones 0.5 to 4; decision 6 |
| Audit 1 | 2. Deferred mutations invalidate contracts | Decision 1; conformance cases D01 to D11 |
| Audit 1 | 3. Architecture enforcement has unchecked paths | Decision 2; cases N01 to N12 |
| Audit 1 | 4. Distributed runtime obligations | Deferred; milestone 5 |
| Audit 1 | 5. Audit and ledger inconsistency | Decision 3; acceptance criteria D1, D4 |
| Audit 1 | 6. Refund overflow | Case C08; acceptance criterion D5 |
| Audit 1 | 7. Excessive initial scope | Release 1 scope; decision 4 |
| Audit 1 | 8. Remote-method removal breaks old callers | Milestone 5 |
| Audit 1 | 9. Operational and resource weaknesses | Acceptance criteria S1 to S5, R1 to R5 |
| [Audit 2](../experiments/audit-2.md) | C1, H1 to H5, M1 to M5 (test 3 code) | Acceptance criteria; the test 3 code is a record and stays |
| Audit 2 | H6, H7, H10, H11, M6, L1, L3 | Decisions 1, 2, 4, 5, 6; spec §1.1 |
| Audit 2 | H8, H9, M7 | Milestone 5 |
| Audit 2 | L2 (line wrapping) | Open |
| [Audit 3](../experiments/audit-3.md) | B1, B2, B3, H2, H3 | Decision 6, amended |
| Audit 3 | H1 | Milestone 0.5 and the time boxes |
| Audit 3 | M1 | This rewrite |
| Audit 3 | M2 | Spec §2, §5.5, §11.3 |
| Audit 3 | M3, M4 | Milestone 5 entry test; milestone 3 exit gate |
| Audit 3 | L1 | Decision 1 and spec §7.3 state the limit; conformance case D12 |
| [Audit 4](../experiments/audit-4.md) | B1 to B4, H1, H2, H4, H6 | Decision 6, amended; milestone 0.5 |
| Audit 4 | M3, M4, L1 | This plan; conformance S08, S10, D12; decision 1; notes in decisions 1 to 4 |
| Audit 4 | H3, H5, H7, M1, M2, M5 to M10, L2 to L6 | Mostly answered by the audit-5 changes; see audit 5's resolution |
| [Audit 5](../experiments/audit-5.md) | B1, B2, H1 to H4, M1 to M6, Lows | Decision 6, amended; this plan; see audit 5's resolution |

## Next deliverable

An independent audit of this version. If it survives, milestone 0.5 starts with recruiting the
roles of decision 6, section 3, because nothing can be sealed without them.
