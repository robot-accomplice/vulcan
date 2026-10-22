# Audit 12: independent check of the development plan after audit 11's fixes

I audited commit `99d9e7c` and did not change the repository (`git status` is clean). Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `sz` = `experiments/sizing/sizing.py`, `res` = `experiments/sizing/results.md`. I wrote every section up to "What holds up" before I read `experiments/audit-11.md`. I read no other audit.

*Provenance: written by a ninth freshly spawned agent, independent of the conversation that wrote
the plan and of the earlier auditors, to check that audit 11's fixes added no Blocker. Stored as
returned.*

**Method.** I copied `sz` to a scratch directory outside the repository and ran it with `python -I -B` (43 s). Its output is byte-identical to `res`. I also ran scratch variants for gate 2's rate test (3,000 simulations per cell) and for the pilot with toolchain-bug failures in the first try (3,000 per cell).

## Verdict

**Yes, the plan survives as written. I found no Blocker.** Audit 11's fixes did not add one. The main question can still be answered. P(Continue | Z) is about 0.01 (`res:40-51`). Labels come from material frozen before the rules (`d6:206-212`, `d6:406-410`). No single person can decisively bias the outcome without leaving a record that the report must publish.

Two High findings remain:

- **H1.** The cost pilot is now staffed and boxed, but it has no run rules. It does not refer to section 6 (toolchain bugs, exclusions, unanswered questions). It gives the second reviewer no role in the verdict. It has no outcome for an unfinished or inconclusive try. Yet it can end the language as "costs too much".
- **H2.** The new agreement floor for the blind graders (audit 11's H1 fix) is unsized and statistically biased, and it can end milestone 0.5 as "stop, not judged". The pre-recruiting sizing never modelled it, and the statistic overstates agreement when the first grader misses violations.

## Blockers

None.

## High

### H1. The cost pilot has no run rules, and it can end the language

- **What the text says.** The pilot can end the project "by row 4 ... and otherwise as **stop: the language costs too much**" (`d6:462-463`). Its rules are only staff, box, pass, and retry (`d6:443-463`). It does not refer to section 6 (`d6:213-225`). The only reviewing rule is "the reviewer and the second reviewer review as in section 3" (`d6:449-450`).
- **Defect 1: toolchain bugs.** This is the first time agents use a new compiler. Section 6 says a run that a toolchain bug blocks "ends as a failure of its arm". It also says more than 10% of such runs forces a fix and a full rerun (`d6:219-224`). Nothing applies that rule to the pilot. If it does not apply, bug failures count as V failures. The pooled retry then carries them into the second try (`d6:459-461`), even after the named change fixes the bug. My scratch run: when 10% of V's first-try runs fail from a bug, the zero-cost pass rate falls from 0.94 to 0.83. At half the margins it falls from 0.71 to 0.55 (15% of runs: 0.78 and 0.46). A compiler bug is then reported as "the language costs too much".
- **Defect 2: the second reviewer has no effect.** In milestone 4, N0 is recomputed with the second reviewer's disputed verdicts, and more than 20% disagreement makes it inconclusive (`d6:267-272`). The pilot has neither rule. One reviewer decides acceptance and adds minutes to X4 on all 80 runs. Nothing in the pilot rule uses the re-review.
- **Defect 3: no outcome for an unfinished try.** A role vacancy, an infrastructure failure, or an overrun of the 6-week box has no defined result. The plan's catch-all ("any result that matches no row above", `plan:47-48`) is written for milestone 4's rows.
- **Remediation.** State that section 6 applies to each pilot try. Let a try that more than 10% of bug-blocked runs stopped run again on new tasks, and leave it out of the pool. Compute the pass rule under both reviewers' verdicts, with the same 20% limit as N0. Map an unfinished try to "stop, not judged".

### H2. The positive-agreement floor is unsized and biased upward

- **What the text says.** The second blind grader re-judges every pair the first marked as a violation, plus 30% of the "none" pairs (`d6:184-187`). Agreement is "of the re-judged pairs that either grader marks as a violation, the share that both mark". Below 0.8, "every test and bound that uses judged violations is inconclusive" (`d6:188-190`). This includes Q1 and gate 2 in milestone 0.5 (`d6:191`).
- **Defect 1: it can end milestone 0.5 unsized.** If agreement falls below 0.8 in milestone 0.5, gate 2 and Q1 are both inconclusive. Gate 1 then gives "stop, not judged" (`d6:436`). The pre-recruiting numbers for milestone 0.5 (Q1 0.93, gate 2 0.91; `d6:319-321`) leave this condition out (`d6:300-303`, `sz:39-41`). Only the later frozen simulation for milestone 4 models it (`d6:345-347`). The floor also has no minimum count, and its value is undefined when no pair is positive. With 10 positives, three disagreements end the project.
- **Defect 2: the statistic overstates agreement.** The second grader sees only 30% of the first grader's negatives. So violations that only the second grader finds are undercounted about 3.3 times. Example: the first grader marks 40 violations, and the second confirms 36. The first grader also missed 20 real violations, which the second grader would find. The measured agreement is 36/(40+6) = 0.78. The true agreement is 36/60 = 0.60. Misses by the first grader are the very bias the floor exists to catch.
- **Defect 3: the worst-case rule names a milestone without V.** "Q2 and gate 2 also pass only if they pass with each of V's final changes that does not build counted under whichever grader found more" (`d6:194-196`). Gate 2 runs in milestone 0.5, where only R and P run (`d6:411`). For V in milestone 4, the second grader judged only a sample of each non-building change's negative pairs. So the rule covers only part of each change.
- **Remediation.** Weight sampled negatives by 1/0.3 in the agreement statistic, or use kappa on the stratified sample. Set a minimum number of positives, and say what happens below it. Add the floor to the milestone 0.5 sizing, or state its expected trip rate. Have the second grader judge every pair of every non-building V change. Drop "gate 2" from the worst-case rule, or restate it for R's judged violations.

## Medium

- **M1. The end-to-end power is not reported, and the omission misleads.** Gate 2 at a true rate of 0.2, which `d6:428-429` calls "the lower rate that section 9 sizes for", passes only 0.51 of the time (`res:21`). `d6:320-321` reports only the 0.4 and 0.1 rates. Chained with the pilot (0.94) and milestone 4 (0.85 to 0.88), a language with a large effect continues about 0.75 of the time at r = 0.4 and about 0.41 at r = 0.2. This leaves out schedule stops and the unmodelled conditions. Also, r at the sizing gate is the measured rate, given that it already passed 0.2 (`d6:316-317`). This selection overstates r near the threshold. **Fix:** publish gate 2's rate at 0.2 and the chained P(Continue | L).
- **M2. Milestone 4 has no reaching-test rule (audit 11 M5, still open).** Only milestone 0.5 requires arm C's label to use the test that reaches V's contract (`d6:415-417`). In milestone 4, the owner writes V's tests and the Go developer writes arm C's tests (`d6:85-89`). Only "the same tests" in `acceptance.md:13` constrains this. When V's tests reach a contract that arm C's do not, the contract's rule lands in U\* when it belongs in UC. That favors "continue". **Fix:** apply the milestone 0.5 rule in milestone 4.
- **M3. The boxes around the pilot conflict.** The next box starts "4 weeks later at the latest" after the previous gate is signed (`plan:89-90`). Milestone 3's previous gate is milestone 2's, but the pilot (6 to 12 weeks, `d6:451-453`, `d6:459`) sits in between. Read literally, milestone 3's box runs during the pilot. The text also does not say when the second try's box starts, or whether the owner's change counts in it. **Fix:** make the pilot's signed result the gate that starts milestone 3. Start the second box on the day the named change is frozen, at most 4 weeks after the first try is signed.
- **M4. A failed sizing gate discards a pivot (audit 11 M2, still open).** Row 1 comes before row 4 (`d6:364`, `plan:145-147`). If Q1 passed in milestone 0.5, a sizing failure still ends as "stop, not judged". **Fix:** apply gate 1 when gate 3 fails.
- **M5. The model-change fallback is unsized (audit 11 M6, still open).** Recruiting to milestone 4 takes more than 60 weeks (`plan:95-102`). The fallback adds arm P with no budget or sizing (`d6:355-356`). The text does not say which model the pilot uses.
- **M6. The run adjudicator has wide powers and no check (audit 11 M7, still open).** One person decides gap calls, coverage-map reasons, paper calls, seeded-case equivalence, exclusions, and the pilot's "named change" (`d6:118`, `d6:457-458`). These decide U\* membership. **Fix:** an eligibility rule, and a second signer for gap calls and reason (a) calls.
- **M7. The plan's milestone 0.5 section omits the paper design (audit 11 M4, part still open).** `plan:128-136` does not mention the frozen paper design that `d6:406-410` requires. Nothing checks the design against release 1. A contract that K0502, K0504, or K0509 would reject still counts as V enforcing a rule (`conformance.md:85-88`).

## Low

- `plan:3-7` still says that "a fresh independent audit checks" audit 10's fixes. Audit 11 did, and its fixes now need this check.
- `docs/decisions/README.md:19` says "amended by audits 3 to 10". `README.md:101` lists "audits 3 to 10". Audit 11 exists and amended decision 6.
- `d6:461` says a change that does not work "does not pass by a second chance alone". The pooled retry still raises the pass rate from 0.30 to 0.39 at the margins, and from 0.54 to 0.73 at half the margins (`res:29-31`). Say "passes less often".
- Row 5's "Q2 passes, and N0 or N1 fails ..., or N6 fails" (`d6:368`) can be read with N6 independent of Q2. Add brackets.
- It is unclear whether N0 needs 2 or 4 computations when the dual-reviewer and dual-grader rules combine (`d6:191-193`, `d6:267-271`).
- `d6:497` (audit 9 row) says 28 × 10 passes L+inter at about 0.84 or more. `res:42-43` shows 0.78 and 0.70. This is history that reads as current.
- The amendments table is out of order: the audit 11 rows sit before audit 10's and audit 9's (`d6:492-497`).
- Gate 2's "at least 5 violations" (`d6:427`) is implied by its rate test: 8 tasks × 5 runs × 0.2 = 8 violations.
- Fewer than 4 U\* rules in milestone 4 ends as "stop, not judged" (`d6:200-202`). This is a negative answer about the language (audit 11 Low, still open).
- `0004-compiler-structure.md:111` still says `keel check`.
- The 5% hole cap has a growing denominator (`plan:187`, `conformance.md:7`).
- Nobody is named to run milestone 3's fresh agent session (`plan:206-211`).

## What holds up

- `sz` reproduces `res` byte for byte. Its tests match section 8: the exact sign-flip test, the half-rate condition, the minimum counts, t-bounds, clean success, N2 per accepted change, and N6 as point estimates on ordinary tasks.
- The pilot's pooled retry is simulated correctly. The second try runs only after a first failure, with an unchanged cost. Pooling over 16 tasks matches `d6:459-461`, and the rates match `res:29-32`.
- The published milestone 4 rates in `d6:322-325` match `res:46-47`.
- Cheaper falsification comes first: gate 2, gate 1, and the sizing gate all run before any compiler work, and the pilot runs before Stockroom.
- Decision tables: section 10 is ordered, and the first match wins. Milestone 0.5's gates cover every case. Row 4 before row 5 keeps a pivot when V costs too much.
- Isolation holds. The toolchain, paper design, and arms are frozen before the rules. Material is sealed with an outside timestamp. The owner is kept from hidden material, including milestone 0.5's (`d6:376-380`).
- Gap rules move reason (c) misses out of U and U\*, so an under-built arm R or arm C cannot inflate U\*.
- Counts and references check out: 32 and 28 acceptance criteria; 87 conformance cases; K0009 in Appendix E (`spec.md:2214`); §1.1, §5.4d, §7.3, §7.4, and §11.1 exist as cited; eight people besides the owner can fill the roles within the separation rules.

## Status of audit 11's High findings

- **H1 (the second blind grader is a weak check): mostly resolved, with new problems.**
  - A judgment is now one (run, rule) pair (`d6:184`).
  - Every positive and 30% of negatives are re-judged, mixed together (`d6:185-187`).
  - Agreement is measured on positives, with a floor of 0.8 (`d6:188-190`).
  - Q1, Q2, gate 2, N0, and N1 must pass under both verdicts (`d6:191-193`).
  - A worst-case rule covers V's non-building changes (`d6:194-196`).
  - The frozen sizing simulation must model these conditions (`d6:345-347`).
  - Base-rate dilution (audit 11 defect 1) is fixed.
  - Over-counting by the first grader is now fully corrected.
  - **Still partial.** Under-counting by the first grader is still corrected for only 30% of pairs. The positive-agreement statistic now inherits that gap and reads high.
  - **New problems.** The floor is unsized for milestone 0.5, and it has no minimum count. The worst-case rule names gate 2, where V does not run. See H2 of this audit.
- **H2 (the cost pilot cannot run): mostly resolved, with new problems.**
  - The pilot now has an application: milestone 0.5's service, ported by the owner in milestone 2 and parity-reviewed (`d6:443-446`, `plan:189-191`).
  - It has staff (`d6:447-450`) and a signer, the run adjudicator (`d6:450`).
  - Its budget is committed at the sizing gate (`d6:349-351`, `d6:451-452`).
  - It has its own box outside milestone 2's (`d6:452-453`, `plan:100`).
  - A retry needs a change that the adjudicator agrees addresses the failure, and is judged on both tries pooled (`d6:456-461`).
  - The outcome with no named change is defined (`d6:462-463`).
  - **New problems.** The pilot has no run rules, and its second reviewer has no effect (H1 of this audit). Pooling makes first-try bug failures permanent (H1). The box start of the second try, and of milestone 3, is unclear (M3). The claim at `d6:461` is overstated (Low).

## Minimum changes

None are required for the plan to survive, because no Blocker remains. Before recruiting, I recommend these, in order:

1. **H1:** apply section 6 to each pilot try. Exclude a try halted by toolchain bugs from the pool, and rerun it. Compute the pilot pass rule under both reviewers' verdicts, with N0's 20% limit. Map an unfinished try to "stop, not judged".
2. **H2:** weight sampled negatives in the agreement statistic, or use kappa. Set a minimum positive count and a rule for zero positives. Include the floor in milestone 0.5's sizing, or report its trip rate. Fix the worst-case rule's reference to gate 2. Have the second grader judge every pair of V's non-building changes.
3. **M1 to M3:** publish gate 2's 0.51 at a rate of 0.2 and the chained P(Continue | L). Add milestone 4's reaching-test rule. Make the pilot's signed result the gate that starts milestone 3, and fix when the second box starts.

## Resolution

The owner asked for this audit's two High findings to be fixed as the last round, on 2026-10-22, in
the commit that adds this resolution:

| Finding | Change |
|---|---|
| H1 | The cost pilot follows section 6's run rules (unanswered questions, rejected changes, exclusions, toolchain bugs). A try in which more than 10% of Vulcan's runs end on a toolchain bug is void: it is not pooled, the bug is fixed under section 6, and the try reruns on new tasks; a second void try is "stop, not judged". The pass rule holds under both reviewers' verdicts, with N0's 20% disagreement limit. An unfinished or inconclusive try is "stop, not judged". The pilot's signed result is the gate that starts milestone 3, and the second try's box starts on the day the named change is frozen, at most 4 weeks after the first try. The claim about the second try now says it passes "less often than it would with a free second try". |
| H2 | The sizing check simulated the agreement floor: with two honest graders and a false-positive rate of 0.005, a floor of 0.8 on weighted positive agreement trips 95% of the time or more. So the floor is dropped. Instead, a judgment is **decisive** when its rule is a targeted rule of its task, and the second blind grader re-judges every decisive pair in every arm, so the both-verdicts rule corrects a grader who over-counts or under-counts in either arm. Kappa on the decisive pairs is reported. The worst-case rule for non-building changes now applies to Q2 only. |
| M1 | Section 9 publishes gate 2's rate at its threshold (0.51) and the end-to-end chance that a language with a large effect continues: about 0.75 at a true rate of 0.4, and 0.41 at 0.2. It also notes that 32 × 10 sits within simulation error of 0.8 under L+inter at r = 0.2, and plans budgets for 36 × 10. |
| M3 | The pilot's boxes and its place before milestone 3 are defined (see H1). |

The other Medium and Low findings stay open.
