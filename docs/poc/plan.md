# Proof of concept: compiler subset and measurement

Status: **accepted**, 2026-10-26.

The owner's decision (2026-10-26): keep the plan and the spec as they are, and build a proof of
concept (PoC) that lets us test the approach ourselves. This is step 2 of the owner's roadmap in
`docs/status.md` (prototype and evaluation). It does not replace the
[development plan](../development-plan.md) or [decision 6](../decisions/0006-evidence-protocol.md).
Its result is internal evidence. It tells us whether the full protocol is worth its cost; it does
not pass any gate of decision 6.

The PoC has two parts:

1. **A compiler subset** that is enough to build and change one real application in Vulcan.
2. **A measurement system** that compares an AI agent working in Vulcan with the same agent
   working in plain Go, on the same application and the same tasks.

## 1. The question

**When an AI agent changes an application, does Vulcan make it keep the team's rules more often
than plain Go does, at an acceptable cost in task success, time, and tokens?**

The spec's premise (§1.1) gives two hypotheses. The PoC measures both:

- **H-A:** the compiler takes the architecture rules out of review: fewer rule violations reach
  the final change.
- **H-B:** the compiler rejects rule-breaking changes before review, and the agent fixes them
  against the diagnostics, at a small cost.

The PoC also measures where Vulcan does **not** help (logic defects that no rule covers), so that
the result is not one-sided.

## 2. Arms

| Arm | What the agent gets | Check command |
|---|---|---|
| **V** | The application in Vulcan, the PoC compiler, and the release-1 primer (`vulcan guide`) | `vulcan check && vulcan test` |
| **G** | The same application in plain Go, with the same team rules written in a `RULES.md`, in Go terms (packages, interfaces) | `go vet ./... && go test ./...` |

Everything else is the same in both arms: the model and its version, the harness, the budgets, the
machine, and the task briefs, word for word except for the language name.

Arm R of decision 6 (Go with static analyzers) is **not** built in the PoC. Instead, each hidden
rule is labelled **analyzer-decidable** or not (section 4). The report then gives the result
separately on the rules that an analyzer could also enforce. This shows how much of any gain a
Go tool could get, without the cost of building arm R. If the PoC is positive, arm R is the next
comparison, as decision 6 requires.

## 3. The application

**Stockroom-lite**: an order, inventory, and billing service in one process. It is a reduced
Stockroom ([release-1/acceptance.md](../release-1/acceptance.md)), cut to what release 1 supports:

| Leaf | Kind | Job |
|---|---|---|
| `Web` | entry for `Customer` | HTTP API (JSON) |
| `Orders` | component | Order life cycle: create, confirm, cancel |
| `Inventory` | component | Stock levels and reservations |
| `Billing` | component | Charges and refunds, a ledger with an invariant |
| `Store` | gateway for `Database` | Persistence (SQLite through a pure-Go driver, no cgo) |
| `Payments` | gateway for `PaymentProvider` | A fake card processor over HTTP, started by the test harness |
| `lib/shop` | library | Shared value types: `Money`, `OrderID`, `SKU` |

Size: about 1,500 to 2,500 lines of Go bodies, so that a task needs real navigation but fits in
an agent's context.

The Go version has one package per leaf, interfaces owned by the consumer, and constructor
wiring in `main.go`, so the two versions have the same shape. A **parity review** by a separate
agent session lists each difference that would change a task's difficulty. Each is fixed before
the freeze.

## 4. Hidden rules

A **rule-writer** agent session writes the team rules. It gets only the functional description of
Stockroom-lite and a neutral package diagram, with no Vulcan material. It writes 12 to 16 rules in
four classes:

| Class | Example | How Vulcan can enforce it |
|---|---|---|
| A. Boundaries | `Orders` does not use `Billing`'s internals | K0101, K0110, K0112 |
| B. Effects | Only `Store` and `Payments` open network or database connections | K0205 (gateway rule) |
| C. Values | A refund is never more than the amount charged, and never 0 or less | `requires`, `ensures`, `invariant` |
| D. Not covered | A failed payment is not reported to the customer as success | Nothing. A control: Vulcan should not help here |

The arms get the rules in different forms: arm G gets all of them in `RULES.md`. Arm V gets the
same `RULES.md`, and the application's code already expresses some of them as ports, gateways,
and contracts. That difference is the thing under test.

Each rule gets two labels, written before the runs:

- **Enforced by V:** the PoC compiler rejects every seeded violation of the rule (section 6).
- **Analyzer-decidable:** a `go/analysis` analyzer could reject the same seeded violations in arm
  G. A separate agent session decides this, with a reason for each rule.

## 5. Tasks

A **task-writer** agent session writes **12 tasks** from the sealed rules:

- **8 tempting tasks.** The easiest way to meet the brief breaks at least one rule. For example,
  "show the refund total on the order page" is easiest done by reading `Billing`'s ledger from
  `Orders`. Each tempting task names its **targeted rules**, sealed with it.
- **4 ordinary tasks.** Features, a refactor, and a failure-handling change, with no planned
  temptation. These measure Vulcan's cost on normal work.

Each task has a brief (the only thing the agent sees), black-box acceptance tests over HTTP, and a
reference solution in each arm that passes the tests and keeps every rule. A task without a
reference solution in both arms is replaced.

## 6. Grading

Grading is automatic, and the same for both arms, so that no judgment differs by arm.

- **Acceptance tests:** HTTP tests against the built binary. The API is identical in both arms,
  so one test suite serves both.
- **Behavioral rule tests:** hidden HTTP tests for the rules of classes C and D (for example, a
  refund larger than the charge must fail with 4xx and must not change the ledger).
- **Structural rule detectors:** for classes A and B, a detector reads the final source. It runs on
  the Go source in arm G, and on the leaf bodies of arm V (which are Go). It uses the import
  graph, the types, and the calls (for example: "a package other than `store` or `payments`
  imports `database/sql` or `net/http` as a client").
- **Holdout tests:** extra hidden tests of behavior that no task touches. They find regressions.
- **Validation before the runs:** for each rule, a **grader** agent session writes two seeded
  violations of different shapes in each arm. A detector or test is valid only if it flags every
  seeded violation and passes both reference solutions. The same seeded cases give the "enforced
  by V" label of section 4.

A **judge** agent session reviews a 25% sample of the structural verdicts by reading the diff, to
estimate detector errors. Its agreement is reported.

## 7. Runs

- **Agent:** Claude Code in headless mode (`claude -p`), one fixed model and version, in a fresh
  copy of the arm's workspace, with no network except the local fake payment provider and the Go
  module proxy cache. It sees only the workspace and the task brief.
- **Budget per run:** 60 agent turns, 45 minutes of wall time, and a fixed token cap. A run that
  hits the budget ends, and its workspace is graded as it stands.
- **Design:** 12 tasks × 3 runs × 2 arms = **72 runs**, in a random interleaved order from a
  committed seed. Two extra pilot tasks test the harness first; they are not counted.
- **Toolchain bugs:** a run blocked by a bug in the PoC compiler is logged and counted as a failure
  of arm V in the main result. The report also gives the result without those runs, and the
  number of bugs.
- **Logs:** the full JSON stream of each run (turns, tool calls, tokens), every run of the check
  command with its output, and the final diff.

## 8. Measurements

| # | Measurement | Definition |
|---|---|---|
| M1 | **Clean success** (primary) | The acceptance tests pass, and no targeted rule is broken |
| M2 | Violations | The number of broken rules in the final change, by class (A to D) |
| M3 | Task success | The acceptance tests pass, whatever the rules |
| M4 | Cost | Tokens in and out, wall time, and turns, to the end of the run |
| M5 | Check loop | Runs of the check command; failed runs before the first pass; for V, the share of failures that were Vulcan diagnostics (K-codes) and not Go errors |
| M6 | Recovery | For V, the share of Vulcan diagnostics that the agent fixed within its next two check runs without breaking the rule another way |
| M7 | Review load | Lines changed, files changed, and the number of leaves (or packages) that the change touches |
| M8 | Regressions | Failures of the holdout tests |
| M9 | False positives | Vulcan diagnostics that the judge session finds wrong, per run |
| M10 | Toolchain bugs | Runs blocked by a PoC compiler bug |

## 9. Analysis

The unit is the **task**. For each task and arm, take the mean of each measurement over its runs.
Compare the arms by the paired per-task difference (V minus G).

- For M1 and M2: the mean difference, a bootstrap 95% interval over tasks, and an exact sign-flip
  p-value. Reported for all tasks, for tempting and ordinary tasks apart, and for each rule class.
- For M4: the geometric mean over tasks of the ratio V / G.
- The analysis script is written and frozen before the runs, and runs on a synthetic data set
  first.

With 12 tasks, the PoC can show a large effect, not a small one. It is a screen, not a proof.

**Reading the result.** These thresholds are written now, before any run, so that we cannot move
them after we see the data:

| Result | Condition |
|---|---|
| **Promising** | On tempting tasks, V's clean success is at least 20 points above G's. On ordinary tasks, V's task success is no more than 10 points below G's. V's cost ratio is at most 1.3. On class D rules, V and G are about equal (as expected) |
| **Promising, but a tool may do it** | As above, but the gain comes only from rules labelled analyzer-decidable. Next: build arm R |
| **Costly** | V keeps more rules, but fails the ordinary-task or cost condition. Next: find the cause in M5 and M6 (diagnostics, primer) and fix it, then repeat once |
| **No signal** | V's clean success on tempting tasks is less than 10 points above G's. Next: stop, or change the rules or the tasks with a reason written down first |
| **Unclear** | Anything else. Next: more runs on the same tasks |

## 10. Compiler scope

The PoC compiler is a subset of release 1 ([release-1/syntax.md](../release-1/syntax.md)), enough
for Stockroom-lite and its tasks. It follows decision 4: staged, reusing `go/parser` and
`go/types` on blanked source, with no IR. Each construct outside the subset is rejected with K0009.

| Area | In the PoC | Codes |
|---|---|---|
| Layout and declarations | `system` with one `container`, leaves, wires, constructor settings, `flag`, `person`, `external`, `component` with `needs port`, `gateway for`, `entry for`, `library` | K0001 to K0004, K0008, K0009 |
| Architecture | No leaf names another leaf; ports owned by the consumer; each port wired once; method sets match; no wire cycles; boundary types from libraries; no `init`; library rules | K0101, K0104, K0105, K0107, K0108, K0110 to K0112, K0117, K0121, K0201, K0204 |
| Gateways and effects | Only gateways and entry leaves reach a network or database function. The PoC checks direct calls and imports, not the full reach analysis of Appendix D | K0205 (simplified), K0206, K0207 |
| Contracts | `requires` and `ensures` on functions, methods, and port methods; `err`, `result`, `old`; `invariant` on structs; tier-1 expressions; every check runs; field writes only in methods | K0501 (literal cases), K0502, K0507, K0510, K0511 |
| Startup | Generated wiring, startup, and shutdown in `main` | K0601 to K0605 |
| Commands | `check`, `build`, `test`, `run`, `guide`, with `--json` diagnostics; source positions mapped back to `.vulcan` files by `//line` directives | |

Deferred in the PoC, and rejected with K0009: strict mode, the rules of §7.3 beyond the return
check, K0525, `audit`, `context`, `query`, and everything that release 1 defers.

## 11. Who does what

The owner's session builds the compiler, the two versions of the application, and the harness.
To keep the hidden material away from the builder, separate agent sessions, each with only its own
inputs, do these jobs:

| Session | Job |
|---|---|
| Rule writer | Writes the hidden rules (section 4) |
| Labeller | Decides the analyzer-decidable label for each rule, with reasons |
| Task writer | Writes the tasks, their targets, and their briefs (section 5) |
| Grader | Writes the acceptance tests, rule tests, detectors, seeded violations, and reference solutions (section 6) |
| Parity reviewer | Compares the two versions of the application (section 3) |
| Judge | Reviews the sample of structural verdicts and the Vulcan diagnostics (M9) |

The hidden material is committed with its hashes before the runs, in a directory that the agents
under test cannot see. The report states that all roles were AI agent sessions under the owner's
control.

## 12. Work order

| Phase | Work | Done when |
|---|---|---|
| P1 | Compiler: declaration parser, blanking, `go/types`, architecture rules, wiring, Go generation, `check`, `build`, `test` | A two-leaf example builds and runs; the milestone-1 conformance cases in the PoC scope pass |
| P2 | Compiler: contracts, the gateway rule, `guide`, `--json` | The milestone-2 conformance cases in the PoC scope pass |
| P3 | Stockroom-lite in Vulcan and in Go; parity review | Both versions pass the same acceptance suite |
| P4 | Harness: run runner, logs, detectors framework, analysis script | A dry run on a synthetic data set gives a report |
| P5 | Hidden rules, labels, tasks, tests, and seeded cases by separate sessions; validation; pilot | Every detector is valid; both pilot tasks run end to end |
| P6 | The 72 runs and the report | The report is in `experiments/poc/` |

Each phase ends with a commit. P1 and P2 end with an independent audit by a fresh agent session of
the compiler against the spec subset.

## 13. Repository layout

```
cmd/vulcan/          the command
internal/...         compiler stages (decl, blank, types, rules, contracts, gen)
vulcan/              the runtime module package (ContractFailure, Value, Bind)
poc/app/vulcan/      Stockroom-lite in Vulcan
poc/app/go/          Stockroom-lite in Go
poc/harness/         runner, graders, detectors, analysis
experiments/poc/     run logs and the report (hidden material sealed until the runs end)
```
