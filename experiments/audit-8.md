# Audit 8: independent re-audit of the development plan

I audited commit `2b22fcb`, read only. Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `acc`, `conf`, `syn` = `docs/release-1/acceptance.md`, `conformance.md`, `syntax.md`. I wrote the Verdict through "What holds up" sections before I read `experiments/audit-7.md`. I did not read audits 3 to 6.

*Provenance: written by a fifth freshly spawned agent, independent of the conversation that wrote
the plan and of the earlier auditors. Stored as returned. The blocker's premise was spot-checked
and held: the protocol fixes only a floor of targeted tasks (`d6:125-127`), not the number the
simulation assumes.*

To check power, I ran a small Monte Carlo in my scratchpad, outside the repository. It uses the plan's own fixed parameters: R at 0.4 and V at 0.05 violations per run under scenario L, gamma task variation with CV 1.0 (the floor in `d6:254-255`), an exact sign-flip test, the 5-informative-task minimum, the 10-violation minimum, and the half-rate condition (`d6:210-218`). The violation-test powers below come from exact enumeration. The full milestone-4 figures are rough, because they use t-approximations for the bounds N0 to N2.

## Verdict

**No. The plan does not survive as written. One Blocker remains.**

The structure is now strong. The plan has a preregistered decision table, a freeze before the rules exist, and owner isolation. It runs a cheap test without a compiler first, and its sizing gate comes before any compiler work. Arm C now neutralizes contract-based inflation of U\*. But the sizing machinery does not size the design that will run.

- **The decisive input is not fixed.** The decisive power parameter is the number of tempting tasks that target the tested set. The simulation does not fix it, and the task writer is held only to a floor.
- **At the floor, the gates cannot pass.** Milestone 0.5 and milestone 4 then cannot reach the 0.8 power target anywhere in the allowed search space. Both end as "stop, not judged" by construction.
- **Above the floor, the guarantee is fictional.** If the analyst assumes more targeted tasks than the floor, nothing binds the real tasks to that assumption. The real power can then be about half of what the gate certified, and a non-answer is reported as "Stop".

## Blockers

### B1. The sizing gates do not bind the design they size, and at the binding floor neither gate can pass

**What the protocol fixes and what it leaves open:**
- A violation test uses only the tempting tasks that target its set (`d6:208`). The task writer must aim "at least five tempting tasks, and at least half of them" at each of S and U\* (`d6:125-127`).
- Above 14 tasks, only the share of tempting tasks is fixed (`d6:124`). The search runs from 14 to 24 tasks and from 5 to 10 runs (`d6:259-260`). Milestone 0.5 runs from 8 to 16 tasks, three quarters of them tempting (`d6:327-333`).
- No text says how many targeted tasks the simulation assumes. The "fixed parameters" (`d6:251-256`) do not include this number.

**Power at the floor (exact sign-flip, CV 1.0, scenario L):**
- **Milestone 4 at its largest design:** 24 tasks give 14 tempting tasks, so at least 7 target U\*. Q2 alone passes with probability about 0.77 at 10 runs. So P(Continue | L) ≤ 0.77 < 0.8 at every design, before N0 to N3 are counted.
- **Milestone 4, all bounds together:** with widened margins, my estimate of P(Continue | L) is 0.42 to 0.52 at 24 × 10, and about 0.05 at 24 × 5.
- **Milestone 0.5 at its largest design:** 16 tasks give 12 tempting tasks, so at least 6 target S. P(Q1 passes | L) is about 0.54 at 10 runs, against the required 0.8 (`d6:331`). At the minimum (8 tasks, 5 targeted, 5 runs) it is about 0.08.
- **With more targeted tasks:** power is about 0.95 for 9 tasks at 10 runs, and about 0.91 for 12 tasks at 5 runs. So the outcome depends on a number that nobody has fixed.

**Consequences:**
- **(a) The conservative reading ends the project unanswered.** The analyst uses the floor, both gates fail, and the project ends as "stop, not judged" before anything is learned (`d6:332-333`, `:262`).
- **(b) The optimistic reading gives a false guarantee.** The analyst assumes that all tempting tasks target the set, and the gate passes. Then the task writer, bound only by the floor, writes fewer. Real power falls to about 0.5, and a large real effect ends in row 5, "Stop", about half the time. The plan describes that outcome as "enforcing the rules does not change outcomes" (`plan:40`).
- **(c) The baseline is fixed by fiat.** R's U\* violation rate is fixed at 0.4 per run (`d6:244-249`), although milestone 0.5 measures exactly this quantity. If the real rate is lower, the sized design is underpowered in the same way.
- **(d) Milestone 0.5's sizing is undefined in two ways.**
  - It is circular: its CV and time spread are "measured in milestone 0.5" (`d6:252-255`), before milestone 0.5 has run.
  - Its order is unclear: the task writer writes the tasks (step 5, `d6:171`), but the number of tasks is set "before the runs" (`d6:329`).

**Remediation:**
1. Fix the number of targeted tasks as a parameter of the simulation, and bind the task writer to it. For example, every tempting task targets a U\* rule in milestone 4, and every tempting task targets an S rule in milestone 0.5. Tasks may target two sets (`d6:127`).
2. Before recruiting, run the frozen simulation with the floor values (CV 1.0, a stated conservative time spread, and the cost of a failed run). Publish whether any design in the search space reaches 0.8 in each milestone. If none does, change the design now: more targeted tasks, a wider search, or a lower target with honest labelling.
3. For milestone 4, size with the lower of 0.4 and R's measured U\* rate on targeted tasks in milestone 0.5.
4. Size milestone 0.5 with the CV floor and stated defaults, and set the number of tasks before step 5.

## High

### H1. Gate 2 of milestone 0.5 is close to a pass by construction, so the expensive work is gated mainly by labelling

- **The bar:** gate 2 sends the project to the compiler when U\* holds at least 4 rules with "at least 5 violations" over all tasks and both arms (`d6:337-340`).
- **Why it is low:**
  - Arms R and P cannot enforce U\* rules, by definition.
  - At least five tempting tasks target U\* (`d6:126`).
  - A tempting brief is "easiest to meet by breaking a hidden rule" (`d6:123-124`).
  - Five violations in 50 or more unenforced, tempted runs is nearly certain. The gate therefore reduces to |U\*| ≥ 4 after labelling.
- **The labels lean toward the compiler:**
  - V's milestone-0.5 labels are paper calls that "assume ... tests that reach each contract" (`d6:323-325`).
  - Arm C's labels are real runs of `go test` (`d6:55`, `:322`). A contract rule that the tests do not reach fails in C but passes on paper in V, so it moves into U\*. This asymmetry inflates U\* exactly where the gate counts it.
- **The plan contradicts itself:** its objective says the cheaper tests show "that the rules Vulcan enforces change outcomes" before the compiler (`plan:12-14`). On the gate-2 path, Q1 is ignored (`d6:338-340`, `plan:125-126`).
- **Remediation:**
  - Make the paper V call cite the test that reaches the contract, and require the same test in C. Otherwise label both arms on the paper assumption.
  - Require R's measured U\* rate on targeted tasks to be at least the rate that the milestone-4 sizing assumes.
  - Stop if Q1's bound rules out a halving on S, even when gate 2 passes. Or correct the objective text.

### H2. A non-building final workspace has no defined violation or defect count, and the gap favours V

- **The rules:** X2 and X6 grade "the workspace when the budget ran out" (`d6:192`, `:196`). Violations are found by hidden tests "which may run the application" (`d6:157-158`).
- **Why V gains:** in V, a broken rule fails the build (`vulcan check`). A budget-exhausted V workspace often does not build, so runtime hidden tests observe nothing, and Q2 records zero violations. Go workspaces usually still build, because R's analyzers do not stop compilation.
- **X6 is also undefined** for these runs: are the holdout tests counted as all failing, or as none?
- **Remediation:** define both measures for workspaces that do not build, the same way in every arm. For example, the blinded grader inspects the diff statically, or the run counts as violating each rule its task targets. Report how many runs fall in this case per arm.

### H3. One unblinded reviewer controls three non-inferiority bounds, and the favourable direction is now unaudited

- **What the reviewer controls:** the reviewer decides acceptance (X1, N0), adds their own minutes to X4 (N1), and alone judges V's false positives (X8, N3) (`d6:107`, `:191`, `:194`, `:198`).
- **Blinding:** the reviewer is blind only to R against P.
- **The audit covers one direction:** the second reviewer now samples only rejected changes (`d6:108`). Leniency toward V, fast reviews of V, and lenient X8 calls are never checked.
- **Effect:** once Q2 passes, these bounds decide between Continue (row 2) and Pivot (row 4).
- **The counterfactual is undefined:** N0 "with every rejection ... reversed" (`d6:223-225`) does not say how to recount a run whose agent revised and resubmitted after the rejection.
- **Remediation:**
  - Sample accepted changes too.
  - Have the second reviewer double-judge a sample of X8 calls.
  - Compute N1 with and without reviewer minutes, or time reviews by a fixed procedure.
  - Define reversal as "accepted at the first disputed submission, with time up to that point".

### H4. Milestone 0.5's 10-week box does not fit its critical path, even with full-time volunteers

- **The Go developer's path:** before the freeze, the Go developer builds the service, the coverage map, arm R (15 days), the generator (15 days), the rule document, and the checklist (`plan:115-117`, `d6:69-71`, `:168-169`). That is at least 35 working days, or 7 weeks full time.
- **The rest of the path:** rules, grading (10 days, `d6:103`), arm C's contracts after labelling (`d6:79-80`), tasks, sealing with an outside timestamp, runs (about 1 week), analysis, sizing, and re-commitment.
- **The box:** 10 weeks (`plan:85`). With part-time volunteers, the cheapest gate is likely to end "stopped by schedule" after one re-plan, before it measures anything.
- **The load estimate is low:** the Go developer's "up to 65 working days" (`d6:100`) is low if they also build Go Stockroom (see M2).
- **Remediation:**
  - Re-estimate the box from the critical path at a stated weekly volunteer load.
  - Let the owner build the milestone-0.5 Go service, with the Go developer's review and parity sign-off. That service is not the milestone-4 baseline.

## Medium

- **M1. The grader can be the run adjudicator, so key checks become self-checks.**
  - Only the rule writer, the reviewer, and the Go developer's two exclusions are restricted (`d6:110-111`).
  - One person can therefore write the seeded translations and certify their equivalence (`d6:140-141`).
  - The same person can review their own paper V calls (`d6:321`), make gap calls on rules they labelled (`d6:149-151`), write the analysis, and operate the runs.
  - Fix: the grader is neither the run adjudicator nor the analyst.
- **M2. The owner may build the Go baseline.** Milestone 3 says "The Go developer builds the Go version, or reviews it" (`plan:181-182`, `d6:100`). Parity review mitigates this but does not prevent a weak baseline. Fix: the Go developer builds the Go Stockroom, and the load estimate is updated.
- **M3. "Informative" is defined for tests, but bounds are found "by inverting the test"** (`d6:210-213`). Under inversion, the shifted differences are no longer zero, so the informative set changes with the shift. Fix: say whether bounds use all tasks (`d6:209` suggests so for N0 to N2), and how zeros are handled when inverting.
- **M4. Gate 1's "stop" needs a bound on a ratio, but the test bounds a difference.** "Its bound shows that R removes less than half of P's violations" (`d6:343-344`). The section-8 bound is on the mean difference (`d6:210-211`). Fix: define the bound, for example on (P − R) − 0.5·P per task, or on a log ratio.
- **M5. The "blinded human grader" is an undefined role** (`d6:158`). It is not in the roles table (`d6:98-108`), it has no load, and its blinding cannot hide Vulcan from Go. Fix: name the role, its holder rules, and its load.
- **M6. The second reviewer may be the Go developer or the grader** (`d6:108`, `:110-113`). Both know the arms, and the grader knows the hidden rules. Fix: give the second reviewer the reviewer's blinding.
- **M7. U\* ≥ 4 in milestone 0.5 says little about milestone 4.** Milestone 4 has a new rule writer and a new application. If its U\* holds fewer than 4 rules, only R and P run (`d6:162-164`), after 24 weeks of compiler work. Fix: state this risk, or require a margin in milestone 0.5, for example U\* ≥ 8.
- **M8. The outcome wording mislabels non-answers.**
  - In milestone 4, an inconclusive result counts as a fail and leads to row 5, "Stop" (`d6:216`, `:280`). The plan describes that outcome as "enforcing the rules does not change outcomes" (`plan:40`).
  - The plan also lists "gave no clear answer" under "stop, not judged" (`plan:43`).
  - Fix: report row 5 as "stop, not judged" unless a bound rules out the effect of scenario L, as milestone 0.5 already does.
- **M9. "Violations at review" in gate 2 is undefined** (`d6:337`). It could mean X2 or X3.

## Low

- **L1. The re-plan rule has gaps.** It does not say when the new box starts (`plan:76-78`). "Overran its time box twice" (`plan:44`) does not match the rule: the first trigger is an overrun of more than half the box.
- **L2. Two budgets are committed before they are known.** Milestone 0.5's budget is committed while recruiting (`plan:70-71`, `d6:328`), before sizing fixes its size, which can vary about fourfold. The narrow rerun needs a third rule writer (`d6:278`) whom nobody recruits, and the rerun has no box.
- **L3. Two rules conflict on fixing tools.** The toolchain-bug procedure has "the owner" fix the bug (`d6:181-182`), but arm R is "not patched" (`d6:151`, `:70`). The text does not say who fixes a bug in an analyzer or the generator.
- **L4. The reviewer's load is understated.** "About 80 hours in total" (`d6:107`) covers only milestone 4 at its minimum size, not milestone 0.5.
- **L5. Conformance case S04 conflicts with the K0009 rule.** S04 expects K0001 for a misplaced `limit` (`conf:35`). But `limit` is deferred with K0009 (`syn:59`, `conf:39`), and an `error` case allows "no more" diagnostics (`conf:22-23`).
- **L6. Acceptance criterion T2 cites a dead source.** It cites "Development plan, milestone 3" (`acc:87`), but that milestone no longer describes fault injection.
- **L7. Three documents are stale.** `docs/status.md:76-79` stops at audit 6. `README.md:113` says "audits 3 to 6". `docs/decisions/README.md:19` says "amended by audits 3 to 6".

## What holds up

- **Governance:** preregistration, a freeze and hashes before the rules exist, sealing with an outside timestamp, and the owner kept from hidden material (`d6:93-95`, `:287-298`).
- **Rules and labels:**
  - An independent rule writer works from a functional brief.
  - Seeded shapes are written once and translated.
  - "Enforced" has one definition.
  - Arm C mirrors V's contracts one to one, so contract rules cannot inflate U\*.
  - Rules missed for lack of time leave U\* (`d6:149-153`).
- **Statistics:** the task is the unit, tests are exact sign-flip tests on paired differences, inconclusive counts as fail, and P(Continue | Z) ≤ 0.05 holds by α. The widening stops below L+cost.
- **The decision rule:** first-match order makes the table complete. Every combination maps to one row, and the narrow rerun is bounded.
- **Gates in the right order:** milestone 0.5 and a sizing gate with re-commitment come before any compiler work. Milestone 4's box scales with its size (210 = 14 × 5 × 3, `d6:267-268`).
- **Release-1 consistency:**
  - The acceptance counts are right: F 6 + D 7 + S 5 + R 5 + T 5 + V 4 = 32, and 28 without V1 to V4.
  - Except for S04, the conformance codes match the milestone table of `syn:63-77`.
  - The plan's spec citations hold (§5.4d, §7.3 rule 5, §7.4, §11.1), and decision 4 rule 4 exists (`0004:109`).

## Status of audit 7's findings

- **B1 (gate 1 of milestone 0.5 unsized and ambiguous): resolved in text, with new problems.**
  - What changed: "informative" is defined (`d6:211-213`). Milestone 0.5 is sized for P(Q1 passes | L) ≥ 0.8 (`d6:329-333`). Gate 2 comes first (`d6:337-340`). A plain "stop" needs a bound (`d6:343-345`).
  - New problems: the target cannot be reached at the targeting floor (my B1). The "half" bound is undefined (M4). Gate 2 is now close to a pass by construction and bypasses Q1 (my H1).
- **B2 (arm C has no contracts): resolved.** The Go developer writes the contracts one to one, and the run adjudicator checks them (`d6:77-81`, `:55`). The generator covers tier 1 or lists each gap (`d6:71-74`).
  - New problem: in milestone 0.5, C is labelled by real test runs while V is labelled on paper (my H1).
- **H1 (the grader steers labels; one person builds and judges arm R): partly resolved.** Seeded shapes are translated (`d6:138-141`), and the Go developer is neither grader nor adjudicator (`d6:111`). But the grader may still be the adjudicator, who then certifies their own translations (M1).
- **H2 (load and re-commitment): resolved.** Gate 3 requires written re-commitment, and the box scales (`d6:265-268`, `plan:89`, `:130-131`).
  - Residual: re-commitment comes about 34 weeks before milestone 4, and milestone 0.5's own box is infeasible (my H4).
- **H3 (unblinded reviewer; rejections never audited): partly resolved, with a new problem.**
  - What changed: rejections are now sampled, N0 is computed twice, and resubmission is defined (`d6:108`, `:177-178`, `:223-225`).
  - New problem: the earlier sample of accepted changes was dropped, so leniency is now unaudited. Reviewer minutes still count in X4, X8 is still judged by one person, and the reversal counterfactual is undefined (my H3).
- **H4 ("narrow" has no procedure): resolved.** There is one rerun with new material, and a second "narrow" maps to "stop" (`d6:278`).
  - Residual: nobody recruits the third rule writer, and the rerun has no box (L2).
- **H5 (U\* mixes "cannot" with "did not"): resolved.** The three reasons are in `d6:66-69`, and cap gaps leave U\* (`d6:149-153`).
  - Residual: the run adjudicator's review of an (a) "undecidable" claim has no criterion.
- **H6 (coin-flip target; margin widening): resolved in text, with a new problem.**
  - What changed: the target is 0.8, the widening stops at 12 points and 1.15, and P(Stop | L) and P(Pivot | L) are published (`d6:258-263`).
  - New problem: at the binding targeting floor, 0.8 cannot be reached (my B1). Audit 7's alternative, reporting underpowered rows 4 and 5 as "not judged", was not adopted (M8).

## Minimum changes

To remove the Blocker:

1. **Bind the targeting count.** Fix the number of tempting tasks that target S (milestone 0.5) and U\* (milestone 4) as a frozen parameter of the simulation. Bind the task writer to it, for example all tempting tasks.
2. **Size now, before recruiting.** Run the frozen simulation now with the floor parameters and stated defaults for time spread and the cost of a failed run. Publish whether milestone 0.5 and milestone 4 can reach 0.8 in their search spaces, and redesign before recruiting if they cannot.
3. **Use the measured baseline.** Size milestone 4 with the lower of 0.4 and R's measured U\* rate on targeted tasks in milestone 0.5.
4. **Fix milestone 0.5's sizing.** Define its sizing inputs (the CV floor and the defaults), and fix its number of tasks before task writing.

Strongly recommended, so that a result is credible: define X2 and X6 for workspaces that do not build (H2), and audit the reviewer in both directions, X8 included (H3). Tighten gate 2 and make milestone 0.5's labelling symmetric (H1). Re-estimate milestone 0.5's box (H4). Separate the grader from the run adjudicator (M1).

As written, the plan does not survive. Its governance and decision table are sound. But the sizing gates certify power for a design the protocol does not require, and at the design it does require, both gates fail by construction. Either the project ends "stop, not judged" with nothing learned, or it passes on an assumption that the real tasks need not meet, and then risks reporting a non-answer as "Stop". The four changes above fix this without altering the given design choices.

## Resolution

Applied on 2026-10-21, in the commit that adds this report, as round 8 of the owner's audit and
remediation cycle:

| Finding | Change |
|---|---|
| B1 | The targeting rule binds: every tempting task targets the set its test uses. A sizing check ran before recruiting (`experiments/sizing/sizing.py`, results in `results.md`). It showed that milestone 0.5 reaches P(Q1 passes \| L) ≈ 0.94 at 16 tasks (12 tempting) × 5 runs, and that milestone 4 cannot reach 0.8 at the old margins (10 points, 1.10) anywhere up to 32 tasks × 10 runs, but does at 12 points and 1.15: 28 tasks (21 tempting) × 10 runs gives about 0.85 (0.82 if R's rate is halved), with P(Continue \| Z) and P(Continue \| L+cost) near 0. The margins are now 12 points and 1.15, with no widening; milestone 0.5's design is fixed; milestone 4 chooses among 24, 28, and 32 tasks × 8 or 10 runs at the sizing gate; r is the lower of 0.4 and the measured rate. Arm P now runs only in milestone 0.5, which cuts milestone 4 to two arms. |
| H1 | Gate 2 needs 8 U\* rules and R's measured U\* rate of at least 0.2. Paper contract calls name the reaching test, and arm C uses the same test. The plan's objective now matches the gate order. |
| H2 | Non-building final changes: X2 by the blind grader from the diff, X6 as failing every holdout test, counted per arm. |
| H3 | The second reviewer samples accepted and rejected changes and X8 calls; N0, N1, and N3 must hold under both judgments and without reviewer minutes; reversal defined. |
| H4 | The owner builds milestone 0.5's service; its box is 16 weeks with the Go developer at least 2.5 days a week. |
| M1 to M9 | Role separations (the grader apart from adjudicator, analyst, and blind grader); the Go developer builds Go Stockroom; t-bounds over all tasks and a defined half-bound; a named blind grader; second-reviewer blinding; U\* ≥ 8; "stop" only on a half-bound, else "not judged"; gate 2 counts X2. |
| L1 to L7 | Re-plan dates and wording; milestone 0.5's budget for a fixed design, and a recruited rule writer and box for a re-run; who fixes which tool bug; reviewer load; S04; acceptance T2's source; stale indexes. |
