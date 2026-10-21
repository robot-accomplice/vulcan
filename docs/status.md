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
fixes. **Next:** decide on audit 12's two High findings (the cost pilot's run rules and the
agreement floor), then recruit the roles of decision 6, section 3 (at least eight people besides
the owner), and start milestone 0.5.

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
