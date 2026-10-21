# Audit 10: independent re-audit of the development plan

I audited commit `c528366` and changed nothing in the repository (`git status` is clean). Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `acc`/`conf`/`syn` = `docs/release-1/acceptance.md`, `conformance.md`, `syntax.md`, `sz` = `experiments/sizing/sizing.py`. I wrote every section up to "What holds up" before I read `experiments/audit-9.md`. I did not read audits 3 to 8.

*Provenance: written by a seventh freshly spawned agent, independent of the conversation that wrote
the plan and of the earlier auditors. Stored as returned. This is the first independent audit that
found no Blocker; by the owner's criterion, the plan is accepted.*

**Method.** I copied `sizing.py` to my scratchpad and ran it with `python -I -B`. Its `results.md` is byte-identical to the committed file. Then I ran my own variants in the scratchpad, 800 simulations per cell (about ±0.015), and 20,000 per cell for the gate-2 and pilot checks:
- I split each sizing outcome into the gate that failed.
- I varied the effect size and the true rate.
- I added a raw cost (a lower success probability and a slower time for V, not net of R's rule breaks), on all tasks or on ordinary tasks only.
- I modelled the cost pilot.

## Verdict

**Yes, the plan survives as written: I found no Blocker.** I found three High findings. They should be fixed before recruiting, because each one weakens a safeguard that the plan relies on.

- **H1.** The clean-success fix for audit 9's B1 reversed the bias. Milestone 4's cost bounds are now almost inert when Vulcan has a large effect. The only real check of cost on ordinary work is an unsized pilot.
- **H2.** Milestone 0.5's paper labels let Vulcan's design be written after the hidden rules are known. Arm R's design is frozen before them. This inflates the cheap gate that decides whether to build the compiler.
- **H3.** One blind grader, who can see the language, alone judges the violations that no test can observe. Those judgments feed Q2, N0, and N1, and nobody checks them a second time.

None of the three makes the main question unanswerable. None decisively biases the final result: milestone 4 labels again from frozen material, and the cost pilot screens large costs.

## Blockers

None.

## High

### H1. N0 and N1 now credit Vulcan with R's rule breaks; the milestone-4 cost bounds hardly bind, and the real cost check is an unsized pilot

**What the text says:**
- Clean success (X1c) counts a run as a success only if it keeps every targeted hidden rule (`d6:212`).
- A run without a clean success counts at the full budget (`d6:215`).
- N0 and N1 use these over all tasks (`d6:247-254`), and three quarters of the tasks are tempting (`d6:134-135`).
- The tasks are built so that breaking a U\* rule is the easiest path (`d6:142-143`).
- So on 21 of 28 tasks, every R violation lowers R's clean success and charges R the budget, before any cost of V is counted.

**Decomposition of the committed model at 28 × 10, scenario L:**
- At r = 0.4, N0 and N1 never failed in 1,000 simulations. All 125 failures came from N2.
- At r = 0.2, N0 failed 2 times and N1 16 times, against 109 failures from N2.
- So under L the cost bounds hardly ever decide.

**Raw cost (my runs, L, 28 × 10, P that Q2, N0, and N1 all pass):**

| V's raw cost (success, time) | r = 0.4 | r = 0.2 |
|---|---|---|
| 12 points, 15% (the margins) on all tasks | 0.48 | 0.16 |
| 15 points, 20% on all tasks | 0.24 | 0.05 |
| 20 points, 30% on ordinary tasks only | 0.92 | |
| 30 points, 50% on ordinary tasks only | 0.80 | |

- **Consequence.** The plan says that "Continue" means the language "costs no more than the agreed margins in task success and working time" (`plan:31-32`). In fact the margins bound a net cost on a mix that is three quarters tempting. The benefit on tasks built to tempt pays for the cost on ordinary work.
- Q2 passes with probability of about 1 once R's rate is at least 0.1 (my runs: 0.94 at r = 0.1). So both primary tests of milestone 4 are driven mostly by R's temptation rate.
- **The pilot is the only real guard on ordinary-work cost.** It runs 8 ordinary tasks × 5 runs and compares point estimates, with one retry (`d6:398-404`). It is not sized in `sz`, and no error rates are published. My model of it (budget 3× median, geometric mean of per-task time ratios):

| True raw cost | P(pass, one try) | P(pass within two tries) |
|---|---|---|
| 0 | 0.80 | 0.96 |
| 6 points, 7% | 0.56 | 0.80 |
| 12 points, 15% (the margins) | 0.29 | 0.50 |
| 20 points, 30% | 0.07 | 0.14 |

- Combined, a language that costs exactly the margins on ordinary work continues about 40% of the time (0.50 × about 0.95 × 0.87). The stated limit is 20% at half the margins (`d6:298-300`), so the protocol is more permissive than it says.
- **Remediation:**
  - In milestone 4, add a cost test on the ordinary tasks alone: a point estimate against the margins, as in the pilot, or a bound with a stated power.
  - Add the pilot to the frozen simulation and publish its error rates.
  - Require that a change is made between the two pilot tries. Otherwise the retry is a best-of-two.
  - Reword `plan:31-32` and `plan:36-38` to say "net cost on the evaluated mix".

### H2. In milestone 0.5, Vulcan's paper design is written after the rules; arm R's is frozen before them

- R's analyzers and configuration file are frozen before the rule writer starts (`d6:187-191`).
- Vulcan's side in milestone 0.5 is a paper call: the grader and the Go developer write down the diagnostic or the contract "that would reject each seeded case" (`d6:370-377`).
- The Go developer writes the contracts "after labelling" (`d6:84-86`).
- No frozen Vulcan design of the milestone-0.5 service exists anywhere: no `system`, `component`, or gateway declarations. So the declarations that a diagnostic depends on can be chosen after the rules are known. Contracts are written to reject the seeded cases.
- Contracts go mostly to UC, because arm C gets them too. But architecture calls, and contract semantics that the generator lists as unbuilt for reasons (a) or (b), stay in U\*.
- This defeats the independent rule writer in exactly the gate that decides whether to build the compiler: gate 2 needs 8 U\* rules (`d6:384-389`, `plan:136-138`).
- In milestone 4 the Vulcan application is frozen before the rules (`d6:83-84`). But its "too few rules" check fires only after milestones 1 to 3 (`d6:181-183`).
- **Remediation.** Freeze a paper Vulcan design of the milestone-0.5 service before the rules exist, hashed with arm R: declarations, wires, gateways, and the contracts. Label only from that design. A rule that the frozen design does not reject is not "enforced by V".

### H3. The blind grader is a single, unaudited, language-visible judge on the primary outcome

- The blind grader judges every rule that no hidden test can observe, and every final change that does not build (`d6:113`, `d6:176-178`, `d6:213`). The blind grader sees the language (`d6:113`).
- U\* holds, by definition, rules that Go static analysis cannot decide (`d6:166-169`). So many U\* rules that are not runtime-observable will be judged this way.
- These judgments set X2, which feeds Q2, and X1c, which feeds N0 and N1 (`d6:212`).
- The reviewer has a second reviewer, an agreement limit, and both-judgment computations (`d6:118`, `d6:247-252`). The blind grader has none of these.
- A blind grader who favours Vulcan can raise R's U\* counts. That passes Q2 and widens V's N0 and N1 head start (H1).
- **Remediation:**
  - Add a second blind grader who re-judges a random 30% or more of the judgments, with an agreement limit.
  - Compute Q2, N0, and N1 under both graders' judgments, as for the reviewers.
  - Report the share of U\* violations that a judge decided rather than a test.

## Medium

- **M1. Milestone 4 lacks the "same reaching test" rule.**
  - In milestone 0.5, arm C's label must use the test that reaches V's contract (`d6:372-375`). Milestone 4 has no such rule.
  - V's tests (written by the owner) and the Go tests (written by the Go developer) differ. A contract that V's tests reach but R's do not keeps a rule in U\* that a generator would deliver.
  - `acc:13` says "the same tests" but defines this only at the HTTP level (`acc:88`).
  - **Fix:** apply the reaching-test rule in milestone 4, or label contracts only through the shared HTTP tests.
- **M2. The run budget is unfixed, and it drives N1.**
  - "The full budget" charged to every non-clean run (`d6:215`, `d6:251`) has no value and no named owner. It lives in the harness that the owner builds (`d6:102`, `d6:94-95`).
  - `sz` assumes 3× the task median (`sz:121`). The sizing gate does not check it (`d6:302-306`).
  - A larger budget widens V's N1 credit under any Q2 effect.
  - **Fix:** fix the budget as a multiple of a measured median, set by the run adjudicator, and include it in the sizing gate.
- **M3. N2 fails about 12% of the time with no real difference.**
  - All of L's non-continues at r = 0.4 came from N2 (125 of 1,000).
  - Row 3 then requires "a new decision [that] changes or drops the feature behind the failing bound" (`d6:330`). No feature lies behind sampling noise in escaped defects. It still costs a full rerun of milestone 4.
  - **Fix:** define what N2's "feature" is, or size N2 so that its false-fail rate under L is stated and small.
- **M4. Gate 2's threshold sits at the bottom of the rates that milestone 4 can handle.**
  - At 8 tasks × 5 runs, the gate passes only 51% of the time at a true rate of 0.2 (`results.md:21`), and 68% at 0.25 and 79% at 0.3 (my runs).
  - Milestone 4 at 28 × 10 still has P(Q2, N0, N1) of 0.89 at a true r of 0.1 (my runs).
  - A false fail ends the language. "Matters" is a judgment, but its false-fail rate at the sized rate should be stated where the gate is defined (`d6:292-293` gives only 0.1 and 0.4).
  - **Fix:** lower the threshold, or add runs or tasks that target U\*.
- **M5. The cost pilot has no builder, no staff, and no budget.**
  - "A small application" in Vulcan and Go (`d6:399`) is not assigned to anyone and has no parity review.
  - The pilot's task writer, reviewer, grader, and operator loads, and its tokens, are not in section 3's loads (`d6:107-118`) or in the committed budgets (`plan:74-77`).
  - It sits inside milestone 2's 10-week box.
- **M6. The loads are understated, and the seven-person roster forces heavy doubling.**
  - In milestone 4 the grader's 15 days (`d6:112`) must cover about 30 rules × 2 shapes × 3 arms of seeded cases, 56 reference solutions, hidden and holdout tests, and pilot tests.
  - The run adjudicator's 4 days must cover every paper call and the one-to-one contract and seeded-case checks.
  - The separations, plus the implicit rule that a bug fixer sees no hidden material (`d6:201-202`), leave only the grader or the adjudicator able to be task writer and operator.
  - **Fix:** re-estimate the loads and name the allowed role combinations.
- **M7. The Go developer sees hidden material, but must get bug reports without it.**
  - The Go developer makes milestone 0.5's paper calls and writes arm C's contracts after labelling (`d6:84-86`, `d6:370-376`).
  - The same person also fixes analyzer bugs from "a minimal reproduction with no hidden material" (`d6:200-202`). The two rules conflict.
- **M8. Governance gaps.**
  - The owner recruits every role, including the run adjudicator, who signs every gate. No conflict-of-interest rule exists.
  - The Go developer, who builds Go Stockroom, also judges parity (`d6:96-98`, `plan:195-196`).
  - Nobody reviews the owner-built harness for arm parity.
  - **Fix:** add an eligibility rule for the run adjudicator, and a second parity check of the applications and the harness.

## Low

- **L1. Stale counts.**
  - `docs/status.md:78` says "audits 4 to 8".
  - `docs/decisions/README.md:19` says "amended by audits 3 to 8".
  - `README.md:101` says "audits 3 to 8".
  - The d6 amendments table lists audit 8's M1 to M9 after audit 9 (`d6:437`).
- **L2. "Success" is undefined in gate 1.** Gate 1 says "R's success" (`d6:391`), but row 4 says "R's clean success" (`d6:331`). X1c is defined only "for N0 and N1" (`d6:212`).
- **L3. "All tasks included" is not true for every bound.** Section 8 says every bound includes all tasks (`d6:236-238`), but N2 and the half-bound use subsets (`d6:242-243`, `d6:255-256`).
- **L4. No milestone-0.5 rule for some failures.** The gate list has no explicit rule for a failed integrity check, a toolchain-inconclusive result, or a budget overrun (`d6:382-396`). The plan's catch-all covers it (`plan:43-46`).
- **L5. The recruiting scope is unclear.** It is not clear whether milestone 4's rule writer and second reviewer must be recruited within the 8-week deadline (`plan:72-74`, `d6:123-126`).
- **L6. The generator's conformance test is undefined.**
  - The generator "must pass the conformance cases" (`d6:79-80`), but those are `.vulcan` projects with `.vulcan` positions (`conf:11-19`). No translation is defined.
  - Reason (a), "static analysis cannot decide", has no meaning for a generator (`d6:78-79`).
- **L7. The model-change rule has gaps.**
  - It adds arm P to milestone 4 without sizing or budget (`d6:319-320`).
  - The pilot's model version is not tied to milestone 4's.
- **L8. The reviewer's hours are low.** The 180 hours (`d6:117`) assume one review per run, with no resubmissions and no pilot.
- **L9. The sizing gate re-measures too little.**
  - It re-measures only the CV and the time spread (`d6:302-306`). Milestone 0.5 also measures base success and interaction, using R against P, but these are not used.
  - At r = 0.4 the design choice sits on a knife edge: 32 × 8 passes L+inter at exactly 0.80 (`results.md:37`).

## What holds up

- `sizing.py` reproduces `results.md` byte for byte. Its tests match section 8:
  - an exact sign-flip test over informative tasks;
  - the half-rate condition;
  - the minimum of 5 tasks and 10 violations;
  - t-bounds over tasks;
  - clean success for N0 and N1;
  - N2 per accepted change over the tasks where both arms have one.
- P(Continue | Z) is about 0 at every design. Power holds for moderate effects too. P(Q2, N0, N1) is 0.86 at r = 0.4 and 0.77 at r = 0.2 with V's rate at r/2.5, and 0.89 at a true r of 0.1.
- The decision tables are ordered, first match wins, and they are complete. Row 6 and the plan's catch-all take every other result.
- Cheaper falsification comes first: gate 2, then gate 1, then the sizing gate, all before compiler work, and the cost pilot before Stockroom.
- The toolchain is frozen before the rules. Hashes are committed with an outside timestamp. The owner is kept from hidden material. Gap rules exist for both arm R and arm C.
- The counts are correct:
  - 32 acceptance criteria (6+7+5+5+5+4), 28 without V1 to V4 (`acc:40-98`);
  - 87 conformance cases;
  - K0009 is in Appendix E (`spec.md:2214`);
  - §1.1, §5.4d, §7.3, §7.4, §11.1, and Q1 (closed) exist as the plan cites them.
- The boxes are signed by a non-owner, and the time between milestones now counts (`plan:80-88`).

## Status of audit 9's findings

- **B1 (N0 and N1 penalise enforcement): resolved.**
  - N0 and N1 use clean success (`d6:212`, `d6:247-254`).
  - The sizing check ran again with L+coupled, L+inter, and L+half (`d6:278-296`, `results.md:27-40`).
  - **New problem:** the bias now runs the other way. Under L, N0 and N1 almost never fail, and a raw cost at the margins passes them 48% of the time at r = 0.4 (H1).
- **H1 (the sizing gate tests only a zero-cost case): partly resolved.**
  - The gate now requires L+coupled and L+inter (`d6:308-309`). L+half is published as an accepted limit (`d6:298-300`).
  - **Not done:**
    - Interaction is still not measured at the sizing gate (`d6:302-306`).
    - The Z and L+cost criteria still cannot fail (`results.md:29-40`, all 0.00 to 0.01).
- **H2 (cost not tested before the compiler): partly resolved.**
  - A cost pilot exists (`d6:398-404`, `plan:187-189`).
  - **New problems:**
    - It is unsized, uses point estimates, and allows a retry with no required change (H1).
    - Its application, staff, and budget are not assigned (M5).
- **H3 (N2 a hidden success bound): resolved.**
  - X6 counts failures per accepted change (`d6:217`). N2 uses tasks where both arms have one (`d6:255-256`).
  - **Residual:** N2 still fails about 12% under L, and row 3 still asks for "the feature behind" it (M3).
- **H4 (gate 2 unsized): resolved.**
  - At least 8 tasks target U\* (`d6:140-143`). The rate test is simulated (`results.md:15-21`, `d6:291-293`).
  - **Residual:** the false-fail rate at the sized rate of 0.2 is 49%, and section 9's text does not state it (M4).
- **H5 (generator gaps stay in U\*): mostly resolved.**
  - The generator-gap rule exists (`d6:170-172`). Its semantics are listed with reasons (`d6:76-80`).
  - **Residual:**
    - The conformance-case translation is undefined (L6).
    - Milestone 4 has no reaching-test rule (M1).
    - Milestone 0.5's post-hoc paper design remains (H2). Audit 9 noted this only in passing.

## Minimum changes

None are needed for the plan to survive, because no Blocker remains. Before recruiting, I recommend these, in order:

1. **H1:**
   - Add an ordinary-task cost check to milestone 4.
   - Simulate the cost pilot and publish its error rates.
   - Require a change before the pilot's retry.
   - Reword "costs no more than the margins" as a net cost on the evaluated mix.
2. **H2:** freeze and hash a paper Vulcan design of the milestone-0.5 service, with arm R, before the rules exist. Label only from it.
3. **H3:**
   - Add a second blind grader on a sample, with an agreement limit, and compute Q2, N0, and N1 under both.
   - Report the share of U\* violations that a judge decided.
4. **M1 and M2:** apply the reaching-test rule to milestone 4. Fix the run budget and who sets it, and check it at the sizing gate.

## Resolution

Audit 10 found no Blocker, and the owner accepted the plan. The owner then asked for its three High
findings to be fixed, on 2026-10-21, in the commit that adds this resolution:

| Finding | Change |
|---|---|
| H1 | Decision 6 adds N6, a point-estimate cost check on the ordinary tasks of milestone 4, required for "continue" and "narrow"; its failure means "the language costs too much". Half of the tasks are now tempting. The sizing check ran again with N6, a raw-cost scenario (L+raw: a cost equal to the margins on every task), and the cost pilot. Before recruiting, 32 tasks × 10 runs gives P(Continue) of 0.85 to 0.90 under L, 0.84 to 0.88 with coupled cost, 0.81 to 0.86 with interaction, 0.05 to 0.12 under L+raw, and about 0 under Z and a 15-point net cost. The sizing gate now also requires P(Continue \| L+raw) ≤ 0.2. The pilot's pass rates are published (96% at no cost, 51% at the margins, within two tries), and its retry needs a named change. The plan describes the margins as a net cost on the evaluated mix, plus a cost check on ordinary work. |
| H2 | Milestone 0.5 has a paper Vulcan design (declarations and contracts), written by the owner, reviewed for parity, and frozen and hashed with arm R before the rules exist. Labels come only from it, and arm C's contracts are its contracts. |
| H3 | A second blind grader re-judges 30% of the blind grader's judgments. Every test and bound that uses judged violations (Q1, Q2, gate 2, N0, N1, N6) must pass under both graders' judgments; more than 20% disagreement makes them inconclusive. The report gives the share of U\* violations decided by judgment. The evaluation now needs at least eight people besides the owner. |

The Medium and Low findings stay open, to be handled as they become relevant.
