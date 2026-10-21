# Audit 11: independent check of the development plan after audit 10's fixes

I audited commit `820af42` and changed nothing in the repository (`git status` is clean). Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `sz` = `experiments/sizing/sizing.py`, `res` = `experiments/sizing/results.md`. I wrote every section up to "What holds up" before I read `experiments/audit-10.md`. I did not read audits 3 to 9.

*Provenance: written by an eighth freshly spawned agent, independent of the conversation that
wrote the plan and of the earlier auditors, to check that audit 10's fixes added no Blocker.
Stored as returned.*

**Method.** I copied `sizing.py` to my scratchpad and ran it with `python -I -B`. Its `results.md` is byte-identical to the committed file. Then I ran variants in the scratchpad at 600 simulations per cell (about ±0.02), and 3,000 per cell for the pilot.

## Verdict

**Yes, the plan survives as written. I found no Blocker.** The fixes for audit 10 did not add one. Two High findings remain, and both weaken safeguards that the fixes rely on:

- **H1.** The second blind grader (audit 10's H3 fix) barely checks a biased first grader. The 20% disagreement limit is diluted by the many trivial "no violation" judgments, and only 30% of judgments are ever replaced.
- **H2.** The cost pilot is now load-bearing (it is half of audit 10's H1 fix). But it has no application in any milestone's scope, no staff, no budget, no signer, and no defined outcome when the owner names no change.

Neither makes the main question unanswerable. Neither can decisively bias the result alone: milestone 4 labels from frozen material, ordinary-task cost is now checked twice (pilot and N6), and P(Continue | Z) is about 0.

## Blockers

None.

## High

### H1. The second-blind-grader safeguard has low sensitivity on the primary outcome

- **What the text says.** The second blind grader re-judges "a random 30% sample of the blind grader's judgments" (`d6:115`, `d6:182`). Each test is computed with the first grader's judgments, and again with only the sampled, disputed judgments replaced (`d6:183-185`). More than 20% disagreement "of the sampled judgments" makes them inconclusive (`d6:185-186`). Both graders see the language in milestone 4 (`d6:114-115`).
- **Defect 1: base-rate dilution.** No text defines what a "judgment" is. If it is one (diff, rule) pair, almost all pairs are "no violation". Two graders who disagree on half of the real violations still agree on more than 90% of pairs, so the 20% limit never fires.
- **Defect 2: partial correction.** The second computation replaces only the 30% sample. A first grader who under-counts V's judged violations, or over-counts R's, keeps about 70% of that bias in both computations. "Passes only if both pass" then passes twice.
- **Defect 3: double effect.** Judged violations feed Q2 through X2 and, through X1c, R's clean success in N0 and N1 (`d6:222`). One biased judge moves the benefit test and the cost bounds in the same direction.
- **Defect 4: asymmetric method.** A V final change with a caught architecture violation does not build. The blind grader judges it from the diff (`d6:223`). A change in a Go arm with the same kind of violation builds, and hidden tests can grade it where they reach it. The report gives only the share of U\* violations decided by judgment (`d6:186-187`). It has no sensitivity analysis.
- **Not modelled.** `d6:289-291` and `sz:36-37` do not list the dual-grader condition among the unmodelled conditions that lower P(Continue).
- **Remediation:**
  - Define a judgment as one (run, rule) pair.
  - Measure agreement on positives: of the pairs that either grader marks as a violation, the share where both agree. Or use Cohen's kappa with a stated floor.
  - Have the second blind grader re-judge every judged U\* violation on tempting tasks, in both arms, plus a random sample of the "no violation" judgments.
  - Require Q2 to pass with V's non-building final changes counted under whichever grader finds more violations.

### H2. The cost pilot cannot run as written

- **What the text says.** The pilot runs "8 ordinary tasks on a small application ... 5 runs each in arms V and R" at the end of milestone 2 (`d6:423-431`, `plan:189-191`). It decides pivot or stop before Stockroom is built.
- **No application.** Milestones 1 and 2 build only conformance cases and a synthetic latency application (`plan:153-183`). No milestone builds a Vulcan version of a small service with Go bodies, nor its parity-reviewed Go twin. Milestone 0.5's Vulcan design is paper only (`d6:389-393`).
- **No staff or budget.** Pilot tasks need acceptance tests (grader), reviews (reviewer), an operator, and tokens. Only the reviewer's hours mention the pilot (`d6:119`). The committed budgets cover milestone 0.5 and milestone 4 only (`plan:76-79`).
- **No signer.** Only the re-plan note is signed (`d6:426-427`). Nobody is named to judge pass or fail.
- **Undefined outcome.** "Without such a change, there is no second try" (`d6:428-429`), but the text does not say what follows. It does not say that the project then ends by row 4 or as "costs too much".
- **The named change has no teeth.** Any edit to the primer qualifies (`d6:427-428`). The retry is in effect still best-of-two, and the published 96% and 51% pass rates (`res:29-31`, `d6:314-315`) assume exactly that.
- **Box conflict.** The pilot and its retry sit inside milestone 2's 10-week box (`plan:99`). Nothing says whether the pilot re-plan is the same as the overrun re-plan of `plan:82-87`. A retry can turn a cost verdict into "stopped by schedule".
- **Remediation:**
  - Name the pilot application: milestone 0.5's service, ported to Vulcan by the owner in milestone 2, with a parity review by the Go developer.
  - Assign the grader, task writer, reviewer, and operator, and add the pilot's hours and tokens to the committed budget.
  - Name the run adjudicator as the person who signs the pilot result.
  - State the outcome when there is no second try.
  - Give the pilot its own box, outside milestone 2's.

## Medium

- **M1. The false-continue control covers one cost profile.** The sizing gate requires P(Continue) ≤ 0.2 only under L+raw, which has a success cost and a time cost together (`d6:326-328`). My runs, all at 32 × 10, r = 0.4:
  - A 12-point success cost on every task, with no slowdown, continues 0.31. Combined with the pilot (0.65 pass), that is about 0.20 overall.
  - The same cost on ordinary tasks only continues 0.41, or about 0.27 overall.
  - A cost of 9 points and 11% (three quarters of the margins) continues 0.33.
  - A 15% slowdown on every clean run, with equal success, continues 0.74.
  - The last case follows from X4. Each failed run is charged 3 times the task median (`sz:130`, `d6:226`), which dilutes a per-run slowdown: 15% per run is only about a 1.07 ratio on X4. In per-run terms, the 1.15 margin is about 1.3.
  - The plan's "1.15 times the working time" (`plan:34`) reads as a 15% slowdown.
  - The run budget is still unfixed (audit 10's M2 is open).
  - **Fix:** add success-only and time-only profiles to the gate's ≤ 0.2 criterion, fix the budget multiple, and report V's per-run time ratio on clean runs.
- **M2. A failed sizing gate throws away a pivot.** If gate 2 passes but the sizing gate fails, the outcome is "stop, not judged" (`plan:144-146`, `d6:409-414`, `d6:420`), even when Q1 passed and R was within P's margins. Gate 1 never runs.
  - **Fix:** when gate 3 fails, apply gate 1.
- **M3. The power is on a knife edge, and unmodelled conditions lower it.** At r = 0.2, the only qualifying design passes L+inter at 0.81 (`res:47`). None of these conditions is modelled, and `d6:289-291` lists only some of them:
  - the dual-reviewer and dual-grader conditions;
  - N1 computed without the reviewer's minutes;
  - N4 and N5;
  - the 4-rule minimum.
  - The analyst's frozen simulation is not required to model them (`d6:321-330`). A design can pass the gate with a true power below 0.8.
  - **Fix:** require the frozen simulation to model every condition of rows 2 and 5.
- **M4. The paper design's validity is unchecked.** The owner writes contracts "in release-1 syntax" with no compiler (`d6:389-393`). The Go developer reviews the design only for parity. A contract that release 1 would reject (K0502, K0504, K0509, K0512; `conformance.md:85-98`) still counts as V enforcing a rule. This matters most for semantics only V has, which stay in U\*.
  - The plan's milestone-0.5 section does not mention the paper design at all (`plan:127-134`).
  - The role table omits the Go developer's paper-design review and paper calls, and the adjudicator's review of those calls (`d6:110`, `d6:117`).
  - **Fix:** have the run adjudicator check each contract against syntax.md and Appendix E, and update the plan and the role table.
- **M5. Milestone 4 still has no reaching-test rule.** Only milestone 0.5 requires arm C's label to use the test that reaches V's contract (`d6:397-400`). For milestone 4, `d6:84-86` has no such rule. Audit 10's M1 is open.
- **M6. The model-change fallback is unsized and likely.** From recruiting to milestone 4 is about 70 weeks or more (`plan:95-101`, plus up to 4 weeks between milestones). The fallback adds arm P to milestone 4 with no sizing or budget (`d6:338-339`). Its Q1 would run on tasks that target U\*, and S only "where the rules allow" (`d6:143-144`). r and CV were measured on the old model.
  - **Fix:** size and budget a three-arm milestone 4, or make a model change end the project as "stop, not judged".
- **M7. Governance gaps are still open (audit 10's M8).**
  - The owner builds the harness, and nobody reviews it for arm parity (`d6:103`, `d6:96-99`).
  - No rule covers who recruits the first role holders, or conflicts of interest. The run adjudicator, who signs every gate, is recruited with no eligibility rule (`d6:131-134`).
  - The Go developer classifies each "not at all" as reason (a), (b), or (c). That choice decides U\* membership (`d6:66-73`, `d6:169-176`), and reasons (a) and (b) are ill-defined for a runtime generator (`d6:79-80`).
  - **Fix:** add a harness parity review by the run adjudicator, an eligibility rule, and generator-specific reasons.
- **M8. The loads are understated.**
  - The blind grader's "about 2 days" must cover every rule that no test can observe, across about 640 diffs, and every non-building diff (`d6:114`).
  - The grader's 15 days must cover 64 to 72 reference solutions in milestone 4, plus seeded cases, tests, and pilot tests (`d6:113`).
  - A rushed judge tends toward "no violation", which feeds H1.
- **M9. N2's noise still triggers a full rerun.** Row 3 asks for "the feature behind the failing bound" (`d6:349`), even when N2 failed by chance. Audit 10's M3 is open.

## Low

- `docs/decisions/README.md:16` says "amended by audits 3 to 9". Audit 10 also amended decision 6.
- `plan:258` still says "H1 to H3 are the next deliverable". `plan:262-265` says they are fixed.
- `d6:461` says the sizing check "fixed milestone 4 at 32 tasks × 10 runs". But the gate chooses among 28, 32, and 36 tasks with 8 or 10 runs, fewest runs first (`d6:325-326`). At r = 0.4, 36 × 8 (576 runs) qualifies (`res:48`) and would be chosen.
- `d6:464` says 28 × 10 passes L+inter "at about 0.84 or more", but `res:42-43` shows 0.78 and 0.70. The row is history, but it reads as current.
- N6 is listed among the tests that use X1c (`d6:183`), but it uses X1 (`d6:275`). Unlike N0, it is not computed under the second reviewer's verdicts (`d6:257-262`).
- The pilot's estimator ("1.15 times slower") is not defined (`d6:426`). `sz:153` uses the geometric mean of per-task ratios.
- The forbidden configuration changes (`d6:62-65`) omit `strict`, `mutators`, `network`, and `acquiring` (`syntax.md:36`).
- `plan:148` says the Go developer "revises arms R and C". `d6:110` allows only the coverage map, the configuration file, and arm C's contracts.
- Milestone 4 with fewer than 4 U\* rules ends as "stop, not judged" (`d6:190-192`), although that is a negative answer about the language.
- The CV measured at the sizing gate does not say which set it uses, S or U\* (`d6:322-324`).
- The plan numbers milestone 0.5's gates as steps 1 to 3 (`plan:138-146`). Decision 6 calls the same gates 2, 1, and 3 (`d6:409-421`).

## What holds up

- `sz` reproduces `res` byte for byte. Its tests match section 8: exact sign-flip, the half-rate condition, the minimum counts, t-bounds, clean success, N2 per accepted change, and N6 as point estimates on the ordinary tasks. The published numbers match `d6:307-315`.
- P(Continue | Z) is at most 0.01, and P(Continue | L+cost) is 0.00. With N6 and the pilot, a raw cost equal to the margins on every task now continues 0.05 to 0.12 (audit 10 found 0.48 without them).
- The decision tables are ordered, first match wins, and they are complete. The plan's six outcomes match rows 1 to 6 and gates 1 to 3.
- Cheaper falsification comes first: gate 2, then gate 1, then the sizing gate, all before compiler work, and then the pilot before Stockroom.
- The toolchain is frozen before the rules, and the paper design now is too. The material is hashed with an outside timestamp, and the owner is kept from hidden material. Gap rules exist for arm R and arm C.
- The counts are correct:
  - 32 acceptance criteria, and 28 without V1 to V4 (`acceptance.md:40-98`);
  - 87 conformance cases;
  - every K-code in release 1 exists in Appendix E;
  - §1.1, §5.4d, §7.3, §7.4, §11.1, and Q1 (closed) exist as the plan cites them.
- The roster is consistent: eight people besides the owner can fill every role within the separation rules.

## Status of audit 10's High findings

- **H1 (the cost bounds are inert, and the pilot is unsized): mostly resolved.**
  - N6 adds a point-estimate cost check on ordinary tasks (`d6:275-279`), required by rows 2 and 3 and failing to row 5 (`d6:348-351`).
  - Half of the tasks are tempting (`d6:139`). The pilot is simulated and its rates are published (`d6:314-315`, `res:23-32`). L+raw is in the gate (`d6:328`).
  - The plan's wording now says "net cost on the evaluated mix" plus a cost on ordinary work (`plan:33-34`).
  - **Not done:**
    - The "named change" has no teeth (`d6:427-429`).
    - The gate controls one cost profile only (M1).
    - The pilot's application, staff, signer, and missing-change outcome are undefined (H2 of this audit; audit 10's M5).
    - The run budget is still unfixed (M1).
- **H2 (the paper design is written after the rules): resolved.**
  - The owner writes the paper design, it is frozen and hashed with arm R before the rules, and it never changes (`d6:389-393`, `d6:196-199`).
  - Labels come only from it, and a rule it does not reject is not enforced by V (`d6:395-398`). Arm C's contracts are its contracts, written before the freeze (`d6:84-88`).
  - **New problems:**
    - The design's validity against release 1 is unchecked.
    - The plan and the role table do not mention the design (M4).
- **H3 (a single language-visible judge): partly resolved.**
  - A second blind grader exists, with separations (`d6:115`, `d6:125-126`). Dual computation and an agreement limit are added (`d6:182-186`), the judged share is reported (`d6:186-187`), and the roster rises to eight (`d6:126-129`).
  - **New problems:**
    - The limit is diluted by base rate, and the substitution corrects only 30% (H1 of this audit).
    - The dual-grader condition is missing from the sizing model's list of unmodelled conditions (`d6:289-291`).
    - N6 is wrongly listed as a test that uses judgments (Low).

## Minimum changes

None are needed for the plan to survive, because no Blocker remains. Before recruiting, I recommend these, in order:

1. **H1:**
   - Define the judgment unit, and measure agreement on positives or with kappa.
   - Have the second blind grader re-judge every judged U\* violation on tempting tasks, in both arms.
   - Add the worst-grader sensitivity rule for V's non-building changes.
2. **H2:**
   - Name the pilot's application, its builder, and a parity review.
   - Assign the pilot's staff and budget, and name its signer.
   - Define the outcome when there is no second try, and give the pilot its own box.
3. **M1 and M3:**
   - Add success-only and time-only profiles to the gate's ≤ 0.2 criterion.
   - Fix the run budget.
   - Require the frozen simulation to model every condition of rows 2 and 5.
4. **M2:** when the sizing gate fails, apply gate 1 instead of ending as "stop, not judged".
