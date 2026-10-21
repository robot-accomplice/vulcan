# Development plan: compiler and application proof

Status: **accepted**, 2026-10-21. The owner's condition was that the plan survive an antagonistic
audit. [Audit 3](../experiments/audit-3.md) (not independent) and the independent audits
[4](../experiments/audit-4.md) to [9](../experiments/audit-9.md) found blockers, which this version
answers. The independent [audit 10](../experiments/audit-10.md) found none; its three High findings
(H1 to H3) were fixed on the same day, and a fresh independent audit checks the fixes.

## Objective

Find out, at the lowest cost that gives a real answer, whether Vulcan merits continued
development as a language. Build a compiler and use it to build and maintain a substantive
application, but only after cheaper tests show two things: that some rules Vulcan enforces need
more than Go's tools can give, and are broken often enough to matter (or, failing that, whether the
rules help at all as Go tools); and that the final evaluation can give an answer.

The lack of a compiler is the reason to run this experiment, not evidence against the project.
The existing writing experiments inform the design but do not establish its benefit. The
decisive evidence must come from real compilation, execution, debugging, and application changes.

Changes to normative rules need an explicit decision and a spec update. Milestone 0 made six such
decisions, and they changed the spec. Historical experiment fixtures remain unchanged; new
experiments get their own directories.

## Possible outcomes

The plan ends in one of six ways. The first five come from the fixed rules of
[decision 6](decisions/0006-evidence-protocol.md) (section 10 for milestone 4, section 13 for
milestone 0.5):

- **Continue the language:** Vulcan enforces rules that neither Go's static analyzers nor a Go
  contract generator can enforce, that changes outcomes on tasks built to test it. Its net cost on
  the evaluated mix of tasks stays within the agreed margins (12 points of clean success, 1.15
  times the working time), and its point-estimate cost on ordinary, untempting work does too.
- **Narrow, as a hold:** as above, but a secondary bound fails (defects, false positives, runtime,
  or latency). A new decision changes the feature behind it, and the bound is measured again.
  Nothing else proceeds meanwhile.
- **Pivot to the rules:** the rules help, but either Go tools deliver them, or the language
  costs too much in task success or time. Continue as a Go tool (analyzers, plus a contract
  generator where it helps), not as a language.
- **Stop:** a bound shows that the tested enforcement keeps more than half of the violations it
  should remove, so enforcing the rules does not change outcomes enough to matter; or the rules
  help but cost too much (in milestone 0.5, the analyzers against plain Go; in milestone 2's cost
  pilot or milestone 4, the language against the analyzers).
- **Stop, not judged:** the independent roles were not filled in time, the budget would be
  exceeded, the evaluation cannot reach a useful power, the integrity check of the seeded cases
  failed, or the evaluation could not finish or gave no clear answer. Any result that matches no
  row above ends here too.
- **Stopped by schedule:** a milestone overran its box by more than half, and then overran its new
  box (see "Time boxes").

Decision 6's question Q2 shows that an effect exists on tasks built to tempt agents. Its size in
ordinary work is estimated and reported, not tested.

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

## Recruiting, budget, and time boxes

**Recruiting comes first.** Fill the roles of decision 6, section 3, and get each person's
commitment to the estimated load listed there. The deadline is **8 weeks** from the plan's
acceptance. If the roles are not filled by then, the project ends as "stop, not judged". The
budget of milestone 0.5 (volunteer hours and tokens, for its fixed design of 16 tasks, 5 runs, and
2 arms) is committed at the same time. The budget of milestone 4 comes from decision 6's sizing
gate. The design sizes were checked before recruiting by simulation
([experiments/sizing/results.md](../experiments/sizing/results.md)).

**Time boxes** are in calendar weeks, counted from each milestone's start date in `docs/status.md`,
which the run adjudicator of decision 6 signs. When a milestone runs over its box by more than half,
work stops for a re-plan: a note in `docs/status.md` that says why and what changes, with one new
box **no longer than the original box**, starting on the date of the note. If the milestone overruns
the new box, the project ends as **stopped by schedule**. The run adjudicator of decision 6 signs
each overrun and each new box.

The next milestone's box starts on the day the previous gate is signed, or 4 weeks later at the
latest, so that time between milestones counts.

The compiler milestones (1 to 3) are large for one person. "Stopped by schedule" is a likely
outcome, and an honest one.

| Milestone | Box |
|---|---|
| 0.5 Rules without a compiler | 24 weeks, from the day the roles are filled, with the Go developer at least 2.5 days a week |
| 1 Vertical slice | 6 weeks |
| 2 Guarantees | 10 weeks |
| 3 Application | 8 weeks |
| 4 Evaluation | 8 weeks at 640 runs, scaled by decision 6's sizing gate |

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
6. Evidence protocol: two run arms in each milestone, a generator check, independent roles, fixed
   thresholds, designs sized before recruiting, a sizing gate, and a fixed decision rule.

Exit gate (met): every implemented guarantee has a precise rule and an acceptance or rejection
example. No decision silently weakens a promise while keeping its stronger description.

## Milestone 0.5: the rules without a compiler

H-A and H-B (spec §1.1) are claims about rules, not about syntax. So test the rules first, in the
cheapest form, and find out whether a language is needed at all. Decision 6, section 13 gives the
protocol:

- The owner builds a small order service in Go (the test 3 shop, rebuilt with a database is
  enough), and the Go developer reviews it. The Go developer writes the coverage map, and builds
  arm R (static analyzers and one configuration file) and arm C's contract generator, each within
  its 15-day cap.
- A rule writer who has not read the spec writes the hidden rules. The grader labels them, using
  the release-1 diagnostics and contracts that Vulcan would apply.
- Run arm R against arm P: 16 tasks, 12 of them tempting (8 of those also targeting the rules
  that need more than Go tools), 5 runs each (decision 6, section 9).

Exit gate, in this order, signed by the run adjudicator (decision 6, section 13):

1. **Is a language needed?** If at least 8 rules need more than Go's analyzers and a contract
   generator, they show at least 5 violations, and arm R breaks them at least 0.2 times per run on
   the tasks that target them, go to step 3.
2. **Do the rules help?** Otherwise: if they help, **pivot to the rules** without building a
   compiler (or **stop**, if arm R costs too much against arm P). If a bound shows that the
   analyzers keep more than half of the violations, **stop**. Otherwise, **stop, not judged**.
3. **Can milestone 4 give an answer?** The sizing gate of decision 6, section 9, including each
   role holder's commitment to the sized load. If no design reaches the required power within the
   caps, **stop, not judged**.

Otherwise continue to milestone 1. For milestone 4, the Go developer revises arms R and C for
Stockroom (decision 6, section 3).

## Milestone 1: compile a vertical slice

Implement declarations, Go-body parsing, symbol resolution, type checking, and Go generation.
Support libraries, two or more leaves, consumer-owned ports, constructor wiring, and one entry
leaf. Generate lifecycle code and preserve source mappings in diagnostics and stack traces.
Provide check, build, test, and module dependency resolution for the supported subset. Write the
design of the milestone-3 application (its leaves, ports, and contracts), so that milestone 2 can
decide whether release 1 needs `pure`.

Start with a small executable that calls through a port, uses a shared library type, and exits
cleanly. Exercise constructor failure and shutdown as well as success. Use the real compiler
in every test; do not strip language features or substitute handwritten generated code.

Exit gate, signed by the Go developer: a clean checkout builds and tests the example with
documented commands. The milestone-1 cases of [release-1/conformance.md](release-1/conformance.md)
pass: illegal imports, missing wires, cycles, incompatible methods, and unsupported syntax fail
with stable diagnostics at the correct source location.

## Milestone 2: enforce the language's distinctive guarantees

Implement the architecture rules of decision 2 and the contract subset of release 1: requires,
ensures, old values, struct invariants, port wrappers, contract-aware fakes, and the rules of
decisions 1 and 3. Make unsupported contract expressions explicit errors (K0009). Add `pure`
predicates only if the design written in milestone 1 needs them.

Deliver guide, context, query, audit, and JSON diagnostics. Context must expose the declarations
and contracts needed for a local change; query must return actual compiler symbols.

Make the conformance suite of release 1 pass, and grow it with each bug. Use differential tests
against Go for supported ordinary Go bodies. Add parser fuzzing and malformed-input tests. Run
race detection in CI on a host that has it. Measure check and build latency against the targets
of decision 6, section 12, on a synthetic application of fixed size. A miss starts the incremental
work of decision 4, rule 4.

Exit gate: every case of `docs/release-1/conformance.md` passes, or is marked as a known hole with
its K0009 or "Unchecked paths" entry. A script lists both. At most 5% of the cases may be holes;
each hole in a construct that release 1 supports needs the run adjudicator's signature, and every
hole is listed in the milestone-4 report. The Go developer checks the list against spec §5.4d and
§7 and signs it. Then the **cost pilot** of decision 6, section 13 runs: 8 ordinary tasks, Vulcan
against the analyzers. If the agents' cost in Vulcan exceeds the margins twice, the project pivots
or stops there, before the application is built.

## Milestone 3: build and operate a substantive application

Build Stockroom, the order and inventory service of
[release-1/acceptance.md](release-1/acceptance.md), in Vulcan with the compiler (by the owner), and
in Go for arm R of decision 6 (by the Go developer). The Go developer reviews both versions for
parity.

Exit gate: the Vulcan version passes all 32 acceptance criteria, and the Go version passes the 28
that apply to it (V1 to V4 are for Vulcan only). Then a **fresh agent session**, with only the
repository and its README and no contact with the owner, builds and runs each version, finds the
cause of one fault that the grader of decision 6 injects (named in a sealed file), and makes one
small change that the grader specifies. The run adjudicator compares the result with the sealed
file and signs the gate. If the fresh agent fails, the owner may fix the documentation once, and a
new fresh agent tries again with a new sealed fault; a second failure fails the gate. The grader,
not the Go developer, checks acceptance criterion D5 on the Go version. The transcripts are kept in
`experiments/m3/`.

## Milestone 4: evaluate development and maintenance

Run [decision 6](decisions/0006-evidence-protocol.md) at the size that its sizing gate fixed: arms V
and R, with arm C's labels (arm P ran in milestone 0.5, whose result row 4 uses); hidden rules from
a second rule writer; tasks written by a non-owner task writer from those sealed rules; rules
labelled for each arm by seeded cases; a toolchain, harness, and analysis frozen before the rules
exist; mandatory independent roles; exact permutation tests on paired per-task differences; the
committed budget; and sealed material kept from the owner and the agents, with an outside timestamp.

Exit gate: the run adjudicator applies the decision rule of decision 6, section 10. Inconclusive
counts as fail. The result is one of the outcomes above.

## Milestone 5: consider distributed features separately

**Entry test.** Start only when milestone 4 ends in "continue", and the milestone-4 records show a
task that the one-container design could not meet without a second process or a workaround that
the evaluation counted (X7).

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

| Source | Findings | Response |
|---|---|---|
| Audit 1 (external, not stored) | 1 to 9 | Milestones 0.5 to 4; decisions 1 to 6; release-1 scope; milestone 5 for 4 and 8 |
| [Audit 2](../experiments/audit-2.md) | C1, H1 to H5, M1 to M5 (test 3 code) | Stockroom acceptance criteria; the test 3 code is a record and stays |
| Audit 2 | H6, H7, H10, H11, M6, L1, L3 | Decisions 1, 2, 4, 5, 6; spec §1.1 |
| Audit 2 | H8, H9, M7 | Milestone 5 |
| Audit 2 | L2 (line wrapping) | Open |
| [Audit 3](../experiments/audit-3.md) | All | Superseded by audits 4 to 6, which re-examined the same areas |
| [Audit 4](../experiments/audit-4.md) | B1 to B4, H1 to H7, M1 to M10, L1 to L6 | Answered by decision 6 as amended through audit 6, this plan, and the release-1 files, except: L2 (H-C mapping) partly; L5 (one model) stated as a limit |
| [Audit 5](../experiments/audit-5.md) | B1, B2, H1 to H4, M1 to M6, Lows | Answered through audit 6's amendments |
| [Audit 6](../experiments/audit-6.md) | B1, B2, H1 to H7, M1 to M11, L1 to L8 | Decision 6, amendments for audit 6; this plan (recruiting, budgets, boxes, gates, milestone 1 design, milestone 2 hole cap, milestone 5 entry) |
| [Audit 7](../experiments/audit-7.md) | B1, B2, H1 to H6, M1 to M10, L1 to L12 | Decision 6, amendments for audit 7; this plan (gate order of milestone 0.5, outcome wording, scaled milestone-4 box, signed start dates). M11 (compiler scope) is accepted as a risk: "stopped by schedule" is a likely outcome, and the plan says so in "Recruiting, budget, and time boxes" |
| [Audit 8](../experiments/audit-8.md) | B1, H1 to H4, M1 to M9, L1 to L7 | Decision 6, amendments for audit 8, and a sizing check run before recruiting (`experiments/sizing/`); this plan (objective, outcome wording, fixed milestone-0.5 design, its box and builder, re-plan dates, the Go version of Stockroom, two arms in milestone 4) |
| [Audit 9](../experiments/audit-9.md) | B1, H1 to H5, M1 to M7, L1 to L11 | Decision 6, amendments for audit 9: clean success for N0 and N1, a re-run sizing check with coupled-cost, interaction, and net-cost scenarios, a cost pilot after milestone 2; this plan (outcome wording, milestone 0.5's box, the time between milestones, the cost pilot, the milestone-3 retry) |
| [Audit 10](../experiments/audit-10.md) | No Blocker; H1 to H3, M1 to M8, L1 to L9 | Plan accepted. H1 to H3 are the next deliverable; the rest follow |

## Next deliverable

Audit 10's High findings are fixed in decision 6 (see its amendments for audit 10): a cost check on
ordinary work (N6) and a sized cost pilot whose retry needs a named change (H1); a paper design of
milestone 0.5's Vulcan side, frozen before the rules (H2); and a second blind grader (H3). Audit
10's Medium and Low findings follow as they become relevant. Next, recruiting starts (see
"Recruiting, budget, and time boxes").
