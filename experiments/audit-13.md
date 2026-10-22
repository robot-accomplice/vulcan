# Audit 13: final independent check of the development plan

I audited commit `81f9eda` read-only. Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `sz` = `experiments/sizing/sizing.py`, `res` = `experiments/sizing/results.md`. I wrote every section up to "What holds up" before I read `experiments/audit-12.md`. I read no other audit.

*Provenance: written by a tenth freshly spawned agent, independent of the conversation that wrote
the plan and of the earlier auditors, as the final check after the owner's last round of fixes.
Stored as returned. Its findings are recorded as open; the owner chose to stop fixing here.*

**Method.** I copied `sz` to a scratch directory outside the repository and ran variants with `python -I -B`: milestone 4 power at 2,000 simulations, power under higher task heterogeneity, the noise of the CV estimate, gate 2's selection effect, a rough model of two honest graders and two reviewers, and binomial trip rates for the pilot's 20% disagreement rule. I changed nothing in the repository.

## Verdict

**Yes, the plan survives as written. I found no Blocker.** The main question can still be answered. P(Continue | Z) stays near 0 (`res:52-63`). Labels come from frozen material (`d6:206-214`, `d6:416-420`). Cheap gates come before the compiler. But audit 12's H2 fix brought back a hole that audit 11's design had closed. The second blind grader now re-judges only "decisive" pairs, but the tests also count pairs that are not decisive. One High finding follows from that. The H1 fix (cost pilot) brought in one new Medium finding.

## Blockers

None.

## High

### H1. The tests count judged violations that only one grader sees, so one grader can still move Q2 and gate 2 (a regression from audit 12's H2 fix)

- **What the text says.** A pair is decisive "when its rule is a targeted rule of its task, because only those pairs enter the tests that decide the outcome: Q1, Q2, and gate 2 through X2" (`d6:184-187`). The second blind grader re-judges every decisive pair and "a random 30% of the other judged pairs" (`d6:187-188`). Both computations use the second grader's verdicts "on all decisive pairs" only (`d6:190-191`). Then the text claims that "a grader who over-counts or under-counts in either arm cannot move the result alone" (`d6:191-192`).
- **The defect.** That premise is false as the tests are defined:
  - X2 counts "hidden-rule violations in the run's final change ... By set" (`d6:235`).
  - A violation test on a set "uses only the tempting tasks that target that set" (`d6:253-254`). It does not limit the count to the targeted rules. So on a task that targets U\* rule A, a violation of U\* rule B counts in Q2. That pair is not decisive.
  - Gate 2 counts "the U\* violations (X2) in both arms, over all tasks" (`d6:436-437`). That includes ordinary tasks and tasks that do not target U\*.
- **The consequence.** For judged rules that are not targeted, only the first blind grader's verdict enters both computations. In milestone 4 the blind grader sees the language (`d6:115`). Such a grader can add violations to R on judged U\* rules that the task does not target, and Q2 can then pass under Z. N0 and N1 still pass under Z, so the result is "continue". The 30% sample of these pairs is only reported; nothing acts on it.
- **Why this is a regression.** Audit 11's design re-judged every pair that the first grader marked, so over-counting anywhere was caught. Audit 12 recorded this as "fully corrected". The new design drops that for pairs that are not decisive.
- **A second inconsistency.** The sizing check counts only "violations of the targeted rules" (`sz:10-11`). So the frozen analysis could compute a different statistic from the one that was sized.
- **Why High and not Blocker.** It needs a biased grader. It works only through judged rules that the task does not target. The reported 30% sample would show the asymmetry. It is the same class of risk as audit 10's H3, which was rated High.
- **Remediation (one sentence either way).**
  - Preferred: define Q1's, Q2's, and gate 2's violation counts as violations of each task's targeted rules in the tested set, and count gate 2 over the tasks that target U\*. This matches `sz` and makes `d6:185-187` true.
  - Or: make every judged pair that enters any test decisive.

## Medium

- **M1. The pilot's 20% disagreement rule can end the project on a small sample (new with audit 12's H1 fix).**
  - A pilot try is "inconclusive" when the reviewers "disagree on more than 20% of the sampled verdicts" (`d6:472-473`). An inconclusive try ends the project as "stop, not judged" (`d6:483-484`).
  - One try has 80 runs, and the second reviewer samples 30% (`d6:121`). That is about 25 to 35 verdicts, and the rule sets no minimum.
  - My binomial check: at an honest disagreement rate of 15%, a 30-verdict sample exceeds 20% about 15% of the time. At 20%, it does so about 39% of the time.
  - Neither the sizing check nor the end-to-end figure (`d6:335-338`) includes this. The frozen simulation covers only rows 2 and 5 of section 10 (`d6:355-357`), not the pilot.
  - **Fix:** set a minimum sample. Or let a third reviewer settle disputed verdicts. Or treat an excess as a void try, not as the end of the project. Model it in the frozen simulation.
- **M2. "Both must pass" gives each grader and each reviewer a veto, and nothing checks the veto.**
  - Under the conjunction (`d6:189-191`, `d6:270-274`), a strict or hostile second grader alone can fail Q2, N0, or N1.
  - Row 4 then gives "pivot" whenever Q1 passed in milestone 0.5 (`d6:377`). That is a substantive outcome.
  - The agreement floor is gone, and kappa is only reported (`d6:195-199`). So "cannot move the result alone" (`d6:191-192`) holds only in the pass direction.
  - **Fix:** a third judge settles disputed decisive pairs (majority of three), or a low kappa triggers a recorded review by the run adjudicator.
- **M3. The model-change path is likely, and it is unplanned (audit 12 M5, still open).**
  - The boxes add up to about 1.5 years from milestone 0.5 to milestone 4 (`plan:97-104`, plus up to 4 weeks between milestones, `plan:91-92`).
  - The fallback adds arm P to milestone 4 (`d6:365-366`). But budgets are planned for 720 runs and two arms (`d6:363`), and the box scales from 640 two-arm runs (`d6:361-363`).
  - Q1 in milestone 4 has no sized power: tasks target S only "where the rules allow" (`d6:144-145`).
  - r, CV, and the disagreement rates were measured on the old model (`d6:344-357`).
  - **Fix:** decide now on a budget, a box, and sizing for a third arm. Or pin a model version that stays available.
- **M4. The sizing gate is sensitive to noisy measured inputs.**
  - The CV comes from about 8 to 12 tasks with a floor of 1.0 (`d6:345-347`). At a true CV of 1.0, my estimate of it from 8 tasks × 5 runs exceeds 1.1 in 25% of draws and 1.39 in 10%.
  - At CV 2.0, 36 × 10 under L+inter at r = 0.2 gives 0.73 (1,500 simulations). Under the gate's rule (`d6:349-352`) the sizing gate then fails, and the project stops as "not judged".
  - Selection biases r: given that gate 2 passed at a true rate of 0.2, the mean measured rate is 0.28 (`d6:318-319`). The design is then sized for a higher rate than the true one.
  - The `res` cells use 600 simulations. At 2,000 simulations, 36 × 10 under L+inter at r = 0.2 gives 0.865, against 0.82 in `res:63`. So the tables are coarse but not optimistic.
  - **Fix:** size at r = 0.2 always. Use a shrunk or capped CV estimate. Allow one larger design (for example 40 × 10) in the search space, with its budget.
- **M5. Nothing checks that the grader's hidden tests are equivalent across arms.**
  - The grader writes hidden tests "for each task and arm" (`d6:114`). The run adjudicator checks only seeded-case translations (`d6:160-162`). Acceptance criterion T4 covers the acceptance tests only (`acceptance.md:96`).
  - In milestone 4 the grader knows which arm is which, and could write weaker violation tests for one arm.
  - **Fix:** each violation-detecting test must flag every seeded case of its rule in its arm, and the run adjudicator checks this at labelling.
- **M6. A failed sizing gate throws away a pivot (audit 12 M4, still open).**
  - Gate 3 failing gives "stop, not judged" even when Q1 passed (`plan:147-149`, `d6:447`, `d6:374`).
  - **Fix:** apply gate 1 when gate 3 fails.
- **M7. The run adjudicator is a single, unchecked signer (audit 12 M6, still open).**
  - One person makes the gap calls, the reason (a) reviews, the paper-call reviews, the exclusions, and the pilot's "named change" agreement (`d6:118`, `d6:426-427`, `d6:474-476`). Several of these decide what goes into U\*.
  - **Fix:** a second signer for gap calls and reason (a) calls.

## Low

- **L1.** `docs/decisions/README.md:19` says "amended by audits 3 to 11". Audit 12 also amended decision 6.
- **L2.** "Continues only about one time in five" at half the margins (`d6:340`) does not match 0.26 to 0.30 at the planned 36 × 10 (`d6:326`). It holds only when the pilot is chained in. Say which.
- **L3.** The pilot's void rule covers only V's bug-blocked runs (`d6:462-465`). For R's analyzer bugs, section 6's full-rerun rule applies (`d6:221-226`). Two procedures overlap.
- **L4.** A void try reruns "in the same box" of 6 weeks (`d6:464`), and that box must also hold the bug fix and new tasks and tests. A box overrun in the pilot is "stop, not judged" (`d6:483`), but milestones get one re-plan (`plan:86-89`).
- **L5.** If the run adjudicator does not agree that the named change addresses the failure, the outcome is unstated (`d6:474-482`).
- **L6.** Row 4 recommends arm C's generator "when Q2 would pass on U (with UC included)" (`d6:377`). U contains U\*, so V's U\* advantage alone can pass that test. It does not show that a generator helps, and no task targets UC.
- **L7.** It is unclear whether N0 needs 2 or 4 computations when the dual-reviewer and dual-grader rules combine (`d6:189-191`, `d6:270-274`). This is audit 12's Low, still open.
- **L8.** The loads look low. The blind grader judges every judged pair of about 720 runs in "about 2 days", and the second blind grader, who re-judges a subset, gets 4 days (`d6:115-116`). The re-commitment at the sizing gate corrects this only if the simulation's hour model is realistic.
- **L9.** `plan:185-186` says "A miss starts the incremental work of decision 4, rule 4". Decision 4's rule triggers only on the one-edit p95 (`0004-compiler-structure.md:112-113`).
- **L10.** A failed milestone-3 gate (`plan:212-213`) maps to an outcome only through the catch-all at `plan:48-49`.
- **L11.** The floor check in `sz:209-225` uses both/(union). That is stricter than positive specific agreement, 2a/(2a+b+c). At q = 0.95 the usual statistic is about 0.81, so the floor would trip about half the time, not 0.95 (`res:40`). Dropping the floor still makes sense, but the stated reason is overstated.
- **L12.** The amendments table is out of order: the audit 12 rows sit above audit 11 and 10 (`d6:513-519`). The audit 11 H1 row still describes the 0.8 floor and does not mark it as replaced. The `sz` docstring still says "(audits 8 and 9)" (`sz:1`).
- **L13.** Rule writers for later rule sets "hold no other role" and must not have read the spec (`d6:112`). The plan's 8-week recruiting deadline (`plan:76-78`) and row 1's "the recruiting deadline passed" (`d6:374`) can be read to require the milestone-4 rule writer up front, about 1.5 years early.

## What holds up

- The published figures match `res`: 36 × 10 (`d6:324-326` against `res:62-63`), the pilot (`d6:331-333` against `res:29-32`), gate 2 (`d6:321-323`, `d6:334` against `res:21`), and the end-to-end products (0.91 × 0.94 × 0.87 is about 0.75; 0.51 × 0.94 × 0.87 is about 0.42).
- `sz` matches section 8: sign-flip over informative tasks, the half-rate condition, the minimum counts, t-bounds over all tasks, clean success with a budget for failures, N2 per accepted change, and N6 as point estimates.
- My rough model of two honest graders (q = 0.9 on half of the violations) and two reviewers (10% to 15% disagreement) did not lower P(Continue) much at 36 × 10. This model leaves out N2 and N3. So the conditions that the check does not model are probably not fatal at milestone 4's size.
- Cheaper falsification comes first. Gate 2, gate 1, and the sizing gate run before any compiler work. The pilot runs before Stockroom.
- The decision tables are complete. Section 10 is ordered and the first match wins. Milestone 0.5's gates cover every case. The pilot now maps every end, including unfinished and inconclusive tries (`d6:481-484`).
- Isolation holds. The toolchain, the paper design, and the arms are frozen before the rules exist. Material is sealed with an outside timestamp. The owner never sees hidden material.
- Gap and generator-gap rules stop an under-built arm R or arm C from inflating U\*.
- The counts and references check out. There are 32 acceptance criteria, and 28 for Go. K0009 is in Appendix E (`spec.md:2214`). The cited spec sections exist. Eight people besides the owner can fill the roles within the separation rules. The plan, `status.md`, and decision 6 agree on the boxes, the 16 × 5 design, gate thresholds, and the pilot.

## Status of audit 12's High findings

- **H1 (the cost pilot has no run rules): resolved, with one new problem.**
  - Section 6 now applies to each try (`d6:461-462`).
  - A try in which more than 10% of V's runs are bug-blocked is void and not pooled (`d6:462-465`).
  - The pass rule holds under both reviewers, with N0's 20% limit (`d6:469-473`).
  - An unfinished or inconclusive try is "stop, not judged" (`d6:483-484`).
  - The pilot's result is the gate that starts milestone 3, and the second box starts when the change is frozen (`d6:466-468`, `d6:476-478`, `plan:102`).
  - **New problems:** the 20% limit on about 30 verdicts, with no minimum, now ends the project with a real chance (M1). The void rule overlaps section 6 for R (L3). The pilot's box has no re-plan (L4).
- **H2 (the agreement floor was unsized and biased): the floor is resolved by removal, but the replacement adds a High.**
  - The floor is gone (`d6:195-199`), and kappa is reported.
  - The worst-case rule now applies to Q2 only (`d6:193-194`).
  - Every decisive pair is re-judged in every arm (`d6:187`).
  - **Not done:** audit 12 asked that the second grader judge every pair of V's non-building changes. The text still covers only the decisive pairs in those changes and 30% of the rest, so "whichever grader found more" (`d6:194`) still compares partial judgments.
  - **New problem:** pairs that are not decisive still enter Q1, Q2, and gate 2 through X2 (`d6:235`, `d6:253-254`, `d6:436-437`). Over-counting by the first grader on those pairs, which audit 11's design caught, is no longer caught (this audit's H1). The veto direction is unchecked (M2).

## Minimum changes

None are required for the plan to survive, because no Blocker remains. Before recruiting, I recommend these, in order:

1. **H1:** in section 8 and section 13, count violations for Q1, Q2, and gate 2 only on each task's targeted rules in the tested set, with gate 2 over the U\*-targeting tasks. This matches `sz` and makes `d6:185-187` true. Or make every judged pair that enters a test decisive. Have the second grader judge every pair of V's non-building changes.
2. **M1:** give the pilot's disagreement rule a minimum sample, or settle disputes with a third reviewer instead of ending the project.
3. **M2 and M3:** add a tie-break for disputed decisive pairs, and plan the budget and box for the model-change fallback.
