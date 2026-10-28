# Status and handoff

This file records where the language work stands, so that work can continue on another machine
without the history of the sessions that produced it.

## Name

The language is **Vulcan** (decided 2026-10-19). Earlier working names were Keel and Domí.
[Decision 5](decisions/0005-names-and-reproducibility.md) (2026-10-20) set the extension
`.vulcan`, the command `vulcan`, and the runtime module `github.com/robot-accomplice/vulcan/vulcan`.
The spec and the primer use these names. The experiment files are records: they keep "Keel",
`.keel`, and `keel`.

## What the spec is

- `docs/spec.md`: about 25,000 words. Sections 1 to 13, Appendix A (sources), B (grammar), C
  (runtime API), D (network and acquiring functions), E (diagnostics), F (lowering to Go).
- `docs/guide.md`: the primer, about 2,400 tokens of a hard 4,000-token budget (§10).
- Function bodies are plain Go. Vulcan adds the architecture (§5), contracts (§7), tests (§8), and
  tools (§9, §10) around them.

## Design rules that decide new questions

These come from the decisions so far. Apply them before proposing a change.

1. **Mechanism, not policy.** The language gives mechanisms and overridable defaults. Operational
   policy (delivery, retries, queues, build output, tracing) is the developer's decision. Offer
   "mechanism + default + override", not one imposed policy.
2. **Compiler first.** Each rule that the compiler can check, it checks. Correctness checks go in
   the compiler, not in review.
3. **Do not reinvent paradigms.** Components in one process talk by ordinary calls, passing values
   or references, usually on the same thread. Topics and transports exist only between processes.
4. **Bodies stay Go.** A rule may reject a body; it must never change what a body means.
5. **Do not present an extrapolation as a decision.** Mark it as open until it is decided.

## Review history

Each review was a fresh, adversarial read of the spec by an implementer with no other context.
Each finding was then resolved and logged.

| Review | Findings | Record |
|---|---|---|
| 1 to 7 | resolved; records lost when a scratch directory was wiped | commit messages |
| 8 | 55 (5 blockers) | `experiments/review-8.md`, `review-8-resolution.md` |
| 9 | 29 (3 blockers) | `experiments/review-9.md`, `review-9-resolution.md` |

## Experiments

- **Experiments 1 and 2** (writing and review, with written rules): their data was lost. Their
  conclusions are in the premise, §1.1, as hypotheses (audit 2, H10): Vulcan moves architecture
  rules out of review and rejects rule-breaking changes before review. It does not shorten a
  thorough reviewer's reading, and it does not reduce logic defects.
- **Test 3** (`experiments/test3/`): a Go writer and a Keel writer each add a feature
  (`feature.md`) to the same shop system, with no written rules. The grader alone reads
  `hidden/rules.md`.
  - Done: both bases (`base-go` builds and tests; `base-keel` passes a manual spec check), both
    writers (`work/go`, `work/keel`, notes in `results/`), the first simulated `keel check`
    (`results/check-round1.md`: 1 error, 5 warnings), and the Keel writer's round 2.
  - Not done: the round-2 check, and the grading of both changes against `hidden/rules.md`.
  - For the grading: audit 2 found that the Go change mixes the audit trails of different orders
    after a restart (C1), and that both changes can overflow in the refund check (H1). The writers'
    code is a record; do not fix it.
  - Test 3 will not be graded: milestones 0.5 and 4 of the development plan replace it. Its files
    stay as a record.

## Open work, in order

[Audit 2](../experiments/audit-2.md) (2026-10-20) lists the findings against the spec and test 3,
with a remediation roadmap.

The milestone-0 decision packet is complete: [decisions 1 to 6](decisions/README.md) are accepted
and applied to the spec. They answer audit 2's spec findings H6, H7, H10, H11, M6, L1, and L3.
H8, H9, and M7 are about distributed features, which milestone 5 takes up. L2 (line wrapping) is
open.

The [development plan](development-plan.md) is **accepted** (2026-10-21). Audit 3 (not
independent) and the independent audits 4 to 9 found blockers, which the plan now answers; the
independent [audit 10](../experiments/audit-10.md) found none. The plan answers item 1 below:
milestone 0.5 tests the rules without a compiler, and decision 6 decides by a fixed rule.
Audit 10's three High findings are fixed in decision 6, and the independent
[audit 11](../experiments/audit-11.md) found that the fixes added no Blocker. Its two High findings are
fixed too, and the independent [audit 12](../experiments/audit-12.md) found no Blocker in those
fixes. Its two High findings are fixed as the last round, and the final independent
[audit 13](../experiments/audit-13.md) found no Blocker. No people are recruited
(owner, 2026-10-22): the roles of decision 6 are AI agent sessions.

**Owner's roadmap (2026-10-22):**

1. **Colleague review.** Share the internally vetted spec with colleagues and collect their
   feedback.
2. **Prototype and evaluation.** Build a prototype and evaluate it.
3. **MVP.** If the prototype passes muster, build an MVP.

The development plan's milestones and decision 6 were written for a different sequence (a
compiler-free test in milestone 0.5 before any compiler work, then a release-1 compiler, an
application, and a formal evaluation).

**Proof of concept (owner, 2026-10-26).** The plan and the spec stay as they are. Step 2 of the
roadmap is the [PoC plan](poc/plan.md): a compiler subset, Stockroom-lite in Vulcan and in Go, and
72 agent runs that compare arm V with plain Go (arm G) by clean success, violations, and cost,
against thresholds fixed before the runs. Its result is internal evidence; it passes no gate of
decision 6.

**Phase P1 (2026-10-26):** the compiler skeleton works. It parses the declaration files, blanks
code files to Go with lines and columns kept, type-checks with `go/types`, applies the
architecture, wiring, library, and startup rules, generates `main`, and runs `check`, `build`,
`run`, `test`, and `mod tidy`. Contracts are rejected with K0009 until phase P2. The independent
[P1 audit](../experiments/poc/audit-p1.md) found 0 Blocker, 4 High, 9 Medium, and 10 Low findings;
its resolution section lists what was fixed. 84 conformance cases pass (`conformance/`).
Spec changes from the audit: K0606 (a `system.vulcan` value is a constant of its field's type),
every exported settings field needs a value (K0603), and K0110 covers more boundary errors.

PoC choices to know: a container exits 0 when all its startup functions return `nil`, also with
an entry leaf, as conformance case W01 expects (§6 says it runs until a stop signal; release 1 has
nothing else that keeps it alive). Open after P1: W04 (stop signal) needs a host with signals;
the differential runner of conformance.md; flags for named types; `-trimpath` and relative
`//line` paths.

**Phase P2 (2026-10-27):** contracts, the network rule, and `vulcan guide`. `requires`,
`ensures`, `old`, `err`, `result`, and `result.N` on functions, methods, and port methods; struct
invariants, checked after each outer method call and on each struct literal, with unchecked twins
for calls through the method's own receiver (§7.2); port contracts checked through a wrapper that
`main` and `vulcan.Bind` use (C13); `on_contract_failure: exit`. Compile-time rules: K0501
(constant arguments), K0502, K0506 (zero value), K0507, K0508, K0509, K0510, K0511, K0512, K0514,
K0120. K0205 and K0203 check the network and acquiring functions of Appendix D that the project's
code uses. `vulcan guide` prints [the PoC primer](poc/guide.md). 108 conformance cases pass.

PoC deviations from §7 and §5.4c, to know when reading results. The independent
[P2 audit](../experiments/poc/audit-p2.md) found 1 Blocker, 5 High, 3 Medium, and 8 Low findings;
its resolution section lists the fixes. After them:

- `ensures` and the invariant are checked at each `return` and at a reachable end of the body,
  before the deferred calls run (Appendix F). The second check point of §7.3, after a risky
  deferred call or a recovered panic, is not built (D01 is not met). Parameters that an `ensures`
  reads are their values at entry.
- The lowering runs on the types of a first pass over the code without contracts: receivers,
  literals (also elided ones), conversions, and aliases are found by type.
- A contract calls only `len`, `cap`, and conversions (K0503), because `pure` is deferred.
- Contract arithmetic is Go arithmetic, not exact (C08 is not met). The proof rule is not
  implemented: K0504 is not reported, and a contract that panics panics as Go does. K0109 is not
  checked.
- A panic that leaves a method with its invariant false is not turned into a contract failure
  (§7.3 rule 5). Calls that receive a struct (`json.Unmarshal(b, a)`) are not followed by a
  check, so the PoC application must not decode a type with an invariant from JSON.
- K0205 and K0203 look at the project's own code; they do not follow calls into other modules
  (N02 is not met), nor the interface steps of §5.4c (audit P2, L7).
- A type error inside a contract is reported at the start of its clause.

**Phase P3 (2026-10-27):** Stockroom-lite exists in both arms: `poc/app/vulcan` and `poc/app/go`,
from the functional description [stockroom.md](poc/stockroom.md), with the test card processor
`poc/fakepay`. Both pass the shared black-box suite `poc/acceptance`. The independent
[parity review](../experiments/poc/parity-p3.md) found 2 High, 4 Medium, and 5 Low differences;
all but one Low are fixed. From the review: `vulcan check` now runs `go vet` (K0706), and K0607
warns of a second instance of one constructor.

**Phase P4 (2026-10-27):** the harness, `poc/harness` (its README). It runs headless Claude Code
in a fresh workspace with a budget, reads the stream (turns, tokens, the check loop), grades
(build, acceptance, rule tests, detectors, holdout), seals the hidden material, and holds the
frozen analysis of plan section 9. The analysis reads four synthetic scenarios right, and the
pipeline ran end to end with a scripted agent on demo material in both arms. Next: phase P5, the
hidden material from separate sessions, its validation, and two pilot tasks with the real agent.

**Phase P5 (2026-10-27):** separate sessions wrote the hidden material: 16 rules, 14 tasks, the
tests, detectors, seeded cases, and reference solutions. It validates and is sealed outside the
repository ([material-p5.md](../experiments/poc/material-p5.md)). The owner then had strict mode
(§5.4d) implemented and turned on in arm V; after it, 2 of 16 rules are enforced by V (R03, R05),
and the compiler rejects 9 of the 36 seeded cases of arm V (5 before). The runner now starts the
agent with `dontAsk` and a tool allowlist. The two [pilot tasks](../experiments/poc/pilot-p5.md)
ran in both arms; all four runs build, pass every acceptance and holdout test, and break no rule.
Next: the decisions that the pilot record lists, then the main runs.

**Round 1 (2026-10-28):** 72 main runs on deepseek-v4-pro through Claude Code
([results-p5.md](../experiments/poc/results-p5.md)). The frozen analysis reads No signal: arm V
succeeded in 28 of 36 runs, arm G in 35 of 36, and neither arm broke a rule in the product code.
Faults found after the runs: the turn budget counted stream events (about 26 real turns, not 60;
fixed in `agentlog`, and `harness restream` corrected the data), the usage of stopped runs is
partial (`usage_partial`), scratch files in the workspace were graded as code, and runs still
shared `/tmp`. The [desk test](poc/desk-test-r2.md) then found every PoC check "Same in Go" (gate 0), and its hole H1 (code files named `vulcan_*` skipped every rule) is fixed. Next: the owner reviews the second draft of round 2, [plan-r2.md](poc/plan-r2.md): each objective of spec §1.1 and §3 measured by script against plain Go, with a third arm that is Vulcan's model as a Go linter and primer, and verdicts written before the runs.

1. **Go or no go.** Planned: milestones 0.5 and 4 of the development plan, by decision 6.
2. **Names.** Done: decision 5, and the rename of the spec and the primer.
3. **Optional review 10.** Reviews 8 and 9 changed much text. The trend is 55, then 29 findings,
   mostly implementation detail.
4. **Test 3.** Replaced by milestones 0.5 and 4. Its records stay as they are.
5. **The compiler.** Milestones 1 and 2, after milestone 0.5 passes. Staged, in Go
   (§11.1, decision 4), to the scope of `docs/release-1/`.
6. **Q22** (§13): numeric performance targets against Go, Java, C++, and Rust.

## Working conventions

- Write prose for other people in Simplified Technical English: short active sentences, one idea
  each. Do not use em-dashes.
- Commits and comments carry no AI attribution.
- Keep experiment data and reports in `experiments/` and commit them. A scratch directory does not
  survive across days.
- Every question to the owner states its context, the options, and a recommendation.
