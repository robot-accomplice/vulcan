# Audit 7: independent re-audit of the development plan

Commit audited: `c688410`. Read only. Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `spec` = `docs/spec.md`. I wrote sections Verdict through "What holds up" before I read `experiments/audit-6.md`. I did not read audits 3 to 5.

*Provenance: written by a fourth freshly spawned agent, independent of the conversation that wrote
the plan and of the earlier auditors. Stored as returned. Both blocker claims were spot-checked and
held: "informative" is undefined (`d6:183`), and no document assigns or defines arm C's checked
comments (the only mention is `d6:53`).*

## Verdict

**No. The plan does not survive as written. Two Blockers remain.**

The structure is much stronger than the version audit 6 saw. The sizing gate now runs before any compiler work. The check commands are defined. Arm R is not patched after the rules exist. The reviewer is isolated. "Narrow" is a hold.

Two defects still let the result be decisively biased:

1. Gate 1 of milestone 0.5 can end the project as "Stop" when a large real effect exists. The gate has no power analysis. Its outcome depends on an undefined word ("informative"). It also tests set S, but "continue" in milestone 4 depends only on U\*.
2. Arm C labels the rules that a generator would deliver. But nothing says that the Go application carries the contracts that arm C's generator needs. Arm C can therefore label nothing, and U\* then contains every contract rule. Audit 6's B1 tried to prevent exactly this.

## Blockers

### B1. Milestone 0.5's gate 1 can stop the project on an unsized, ambiguous test, and it tests the wrong set

- Gate 1: "if Q1 fails, **stop**". If Q1 is inconclusive, the run extends once to twelve tasks, then ends as "stop, not judged" (`d6:281-283`, `plan:118-119`). "Stop" is reported as a finding: "enforcing the rules does not change outcomes" (`plan:38`).
- Q1 uses only the tempting tasks that target S (`d6:180-181`). Milestone 0.5 needs only five of them (`d6:279-280`). The exact sign-flip test then passes only when every task favours R: the smallest p-values are 1/32 and 2/32 = 0.0625.
- "A test needs at least 5 informative tasks" (`d6:183`). The protocol never defines "informative". Two readings:
  - **Reading A:** a task with a zero difference does not count. Then a zero-difference task makes the test inconclusive.
  - **Reading B:** every targeted task counts. Then one zero-difference task makes the test fail, which means Stop.
- I simulated the gate:
  - Assumptions: 5 S-targeted tasks, 5 runs each. R leaks 0.02 violations per run. P has 0.4 or 0.8 per run, the plan's own scenario rate (`d6:214`). Task rates are equal or vary between tasks (gamma, CV 1). One extension adds 3 S-targeted tasks.
  - Reading B: the gate returns "Stop" in 29% (equal tasks, 0.4) to 54-63% (varied tasks) of trials, although R removes nearly all violations.
  - Reading A: "Stop" in 4-11% of trials, and "not judged" in up to 28%.
  - These numbers are my estimates. The point stands: nobody sized this gate. The sizing gate is gate 3 and covers only milestone 4 (`d6:206-225`, `:288`).
- The gate also tests the wrong set. Row 2 ("continue") needs Q2 on U\* and does not need Q1 (`d6:234`). Gate 1 can kill the language because statically checkable rules (S) did not matter. It ignores the U\* evidence that gate 2 collects in the same runs (`d6:284-285`).
- **Remediation:**
  - Define "informative" in section 8 as a task with a nonzero paired difference.
  - Size gate 1 with the frozen simulation before milestone 0.5's runs. Require enough S-targeted tasks for the power that section 9 requires, for example P(pass | L) ≥ 0.8.
  - Report "Stop" only when the bound excludes a meaningful effect. Otherwise report "stop, not judged".
  - Do not stop the language on Q1 alone while gate 2 finds at least 5 U\* violations. In that case, continue to gate 3.

### B2. Arm C has no defined contracts to check, so U\* can include every contract rule

- Arm C is "a contract generator that turns checked comments into runtime checks" (`d6:53`). It exists to remove from Q2 the rules that a generator would deliver (`d6:125-126`, `:310`).
- No text says who writes those checked comments in the Go application, or that they must mirror the Vulcan application's contracts. The grep is empty outside `d6:53`. The owner writes the Vulcan contracts (`d6:79`). Acceptance criterion V2 requires them in the Vulcan arm only (`docs/release-1/acceptance.md:97`).
- Parity review lets the reviewer "fix or log" a difference (`d6:74-75`). So a Go application with no checked comments is allowed.
- Then arm C rejects no seeded contract case, UC is empty, and U\* = U. Q2, and with it "continue" against "pivot", then rests on rules that a generator would deliver. The owner's decision for arm C (`d6:66-68`) has no effect.
- Arm C also "carries over" from milestone 0.5's shop to Stockroom (`plan:125`). Nothing says who annotates Stockroom's Go version.
- **Remediation:**
  - Before the freeze, the Go developer writes each contract of the frozen Vulcan application as a checked comment in the Go application.
  - Parity review verifies the comments one to one. A missing contract is fixed, not logged.
  - The generator supports the release-1 tier-1 contract forms (`docs/release-1/syntax.md:53-56`): requires, ensures, `old`, and invariants. An unsupported form is listed in the coverage map, and the run adjudicator reviews the list.
  - Arm P's application does not run the generator.

## High

### H1. The grader can steer set labels, and one person may build arm R and judge it

- The grader writes seeded cases "for each rule and each arm" (`d6:118-119`). Each arm gets its own cases, with no requirement that they encode the same change. A seed shape that R misses moves a rule into U. A shape that V misses moves it out.
- Roles may combine, except the rule writer and the reviewer (`d6:96`). So one person can be:
  - the Go developer who builds arm R and writes the coverage map;
  - the grader who labels arm R;
  - the run adjudicator who "reviews the map" and decides gaps (`d6:61-64`, `:127-128`).
- That person reviews their own work and sets S, U, and U\*.
- **Remediation:**
  - Write one seeded change per shape, and translate it across the arms. The run adjudicator checks that the translations are the same.
  - Forbid combining the Go developer with the grader or with the run adjudicator.

### H2. The sizing gate can multiply the volunteer load, and nobody re-commits before the compiler is built

- Loads are estimated for the minimum design. For example, the reviewer is "about 80 hours in total" (`d6:93`), and people commit to them while recruiting (`d6:99`, `plan:65-67`).
- The sizing search can raise milestone 4 to 24 tasks × 10 runs × 3 arms = 720 runs, against 210 runs today (`d6:220-221`). At 15 minutes per review, that is about 180 hours of review in milestone 4 alone. The grader's, operator's, and adjudicator's loads grow too.
- Sizing publishes "expected volunteer hours" as the committed budget (`d6:223-225`). But gate 3 does not require the role holders to accept the new load.
- If they cannot, row 1 ("budget would be exceeded" or "a role is empty", `d6:233`) ends the project as "not judged" after milestones 1 to 3. Those are 24 weeks of boxes (`plan:80-82`).
- Milestone 4's box stays 8 weeks whatever the design size (`plan:83`).
- **Remediation:**
  - Make gate 3 include written re-commitment by each role holder to the sized load.
  - Scale milestone 4's box with the sized number of runs.

### H3. One unblinded reviewer decides V's task success and adds to its working time, and rejections are never audited

- The reviewer's pass is part of X1 (`d6:163`). The reviewer's minutes are part of X4 (`d6:166`). The reviewer judges X8 from V's logs (`d6:93`).
- The reviewer is blind only to R against P, not to V against Go (`d6:93`).
- The N0 margin is 10 points (`d6:192-193`). At 70 V runs, that is 7 rejections.
- The second reviewer audits only accepted changes (`d6:94`). A biased rejection of a V change, or of a Go change, is never checked.
- **Remediation:**
  - The second reviewer re-reviews a random sample of rejected changes from each arm.
  - Report agreement between the two reviewers. Run N0 with and without the disputed rejections.
  - Define whether a rejected change can be revised within the run.

### H4. "Narrow, as a hold" has no procedure, no budget, no limit, and it re-measures on released material

- Row 3: "the bound is measured again" (`d6:235`, `plan:33-35`). Nothing says on which tasks, at what size, by whom, with what budget, how many times, or what a second failure means.
- After milestone 4, the sealed material is released into `experiments/m4/` (`d6:252-253`). The owner then knows every task and rule, and can tune the compiler, for example its false positives (N3), to them before the re-measurement.
- **Remediation:**
  - Re-measurement uses new hidden rules and tasks from a new rule writer, at the sized design, with a committed budget.
  - Allow one attempt. A second failure maps to row 4 or row 5.

### H5. U\* mixes "Go tools cannot enforce this" with "15 days of work did not enforce this"

- Arm R is built "within 15 working days" (`d6:64-65`). The coverage map gives "a reason when not" covered (`d6:63`), with no fixed categories.
- A rule that static analysis could check but that the cap left out falls into U, and then into U\* if arm C does not cover it.
- The plan states the "continue" claim as rules "that neither Go's static analyzers nor a Go contract generator **can** enforce" (`plan:30-31`; also `d6:41`, `:126`). The data supports only "did not enforce".
- **Remediation:**
  - Classify each "not at all" entry as one of: not decidable statically; needs runtime checks; or not built within the cap.
  - Move the "not built" rules out of U\*, as gaps are moved, and report them.

### H6. The sizing target accepts a coin flip under a large effect, and the margins can widen to fit the sample

- The design passes with P(Continue | L) ≥ 0.5 (`d6:219`). Scenario L is an 8-fold reduction. A large real effect therefore still ends in "pivot" or "stop" half the time. Those outcomes are reported as findings (`plan:36-38`), not as "not judged".
- If no design passes, the margins may widen to 15 points and 1.20 (`d6:221-222`). That is exactly the "real cost" of scenario L+cost (`d6:217`). So the acceptable cost is set by what sample the project can afford, not by what matters.
- **Remediation:**
  - Raise the target to P(Continue | L) ≥ 0.8.
  - Report rows 4 and 5 as "not judged" when the bounds were too wide to exclude L. Or publish P(Stop | L) and P(Pivot | L) next to the outcome.
  - Remove the widening, or forbid it from reaching L+cost.

## Medium

- **M1. The sizing simulation is underspecified.**
  - Above 14 tasks, only "the same share of tempting tasks" is fixed (`d6:107`). The number that target U\* stays "at least five" (`d6:108-109`).
  - The scenarios set no values for X6, X8, X11, or latency, but "continue" needs N2 to N5 (`d6:212-219`).
  - Variance comes from a different application (the shop, not Stockroom) and from arms R and P only.
  - Fix: name each parameter, and freeze the parameter values with the simulation.
- **M2. "Arms R and C carry over to milestone 4"** (`plan:125`), but Stockroom is a new application.
  - Nobody is assigned Stockroom's configuration file, coverage-map revision, or checked comments, or their time cap.
  - Fix: assign them to the Go developer before milestone 4's freeze.
- **M3. "Too few rules" contradicts the decision table.**
  - The protocol records the outcome "then, without runs" (`d6:137-138`). But row 4, and gate 2's "pivot" condition (`d6:285-287`), need success and time data from runs.
  - Fix: when U\* < 4 and S ≥ 4, run arms R and P only.
- **M4. Nobody proves that a tempting task has a compliant solution.**
  - "Easiest to meet by breaking a hidden rule" (`d6:106-107`) does not guarantee that a compliant path exists in arm V. An impossible task costs V on X1 and X4.
  - Fix: the grader writes a reference solution that complies with the rules, in each arm, for each task.
- **M5. X6 and N2 are inconsistent across runs.**
  - The second reviewer samples 30% of accepted changes (`d6:94`), so X6 has different parts for different runs (`d6:168`).
  - Failed runs have no defined X6.
  - An arm that fails more often has fewer accepted changes in which to escape defects.
  - Fix: define X6 for failed runs. Scale the sampled part, or review every accepted change.
- **M6. N3 treats runs as independent** (`d6:196`), although the unit is the task (`d6:178`).
  - A false positive in the compiler tends to repeat on every run of the same task.
  - Fix: bound N3 over tasks, or use a cluster-adjusted bound.
- **M7. Attrition and concentration.**
  - All roles, including milestone 4's second rule writer, are recruited at the start (`plan:65`). That is up to about a year before milestone 4, or longer under re-plans (`plan:72-74`).
  - No rule says how to replace someone who leaves. A vacancy ends the evaluation (`d6:99`).
  - One volunteer may carry the Go developer's load (about 55 days) plus four other roles.
  - Fix: define replacement before sealing, with the run adjudicator's sign-off.
- **M8. Milestone 0.5's "V enforces" is a paper call** (`d6:274-276`).
  - It assumes no compiler bugs, and that the tests reach every contract. This inflates U\* in milestone 0.5 and favours building the compiler.
  - Gate 2 does not say which tasks' violations count (`d6:284-285`).
- **M9. In-source suppression is not covered.**
  - The configuration rule (`d6:58-60`) and X7 (`d6:169`) cover configuration files. They do not cover in-source directives such as `//lint:ignore` or ignore comments in custom analyzers.
  - An R agent can pass its check that way.
  - Fix: count and forbid them as configuration changes.
- **M10. Isolation in practice.**
  - The owner builds the harness (`d6:79`). The model account and the log storage are unassigned.
  - Fix: run the harness under the operator's account, and keep the logs in the private store.
- **M11. The compiler scope is large for one person in 16 weeks** (`plan:80-81`).
  - The scope includes reach analysis through third-party SDKs (`docs/release-1/conformance.md:59`, N02), strict mode, contracts with exact arithmetic and second check points, tools, fuzzing, and differential tests.
  - "Stopped by schedule" is a likely outcome. That is honest, but it should be expected.

## Low

- **L1.** Decision 4 rule 4 cites "decision 6, section 9" for the synthetic application (`docs/decisions/0004-compiler-structure.md:110`). The latency targets are in section 12.
- **L2.** Acceptance criterion T3 cites "(M12)" (`docs/release-1/acceptance.md:88`). Decision 6 calls the benchmark X11 (`d6:173`).
- **L3.** `spec:1739` says the latency is measured "on a real application". Decision 4 and `plan:157-158` say a synthetic application in milestone 2.
- **L4.** `plan:184-185` reads as if the second rule writer writes the tasks.
- **L5.** The plan's gate 2 (`plan:120-121`) omits the "otherwise **stop**" condition of `d6:285-287`.
- **L6.** Conformance case S08 expects K0009 for `pure` (`docs/release-1/conformance.md:39`), but milestone 2 may add `pure` (`docs/release-1/syntax.md:57`). D08 handles this case; S08 does not.
- **L7.** "Pivot: ... Go tools deliver them" (`plan:36`) is wrong when row 4 fires because Q2 passed but N0 or N1 failed.
- **L8.** Row 4 judges R against P by point estimates. Rows 2 and 3 use bounds (`d6:236`). State why the standards differ.
- **L9.** The plan's "stop, not judged" list (`plan:39-40`) omits a failed integrity check (`d6:233`).
- **L10.** The milestone 1 and milestone 2 gate signatures (`plan:140`, `:165`) are not in the Go developer's duties or load (`d6:86`).
- **L11.** The owner records milestone start dates in `docs/status.md` (`plan:71-72`). The run adjudicator does not sign them.
- **L12.** Decision 6 is "accepted" (`d6:3`) while the plan that it serves is still "proposed" (`plan:3`).

## What holds up

- **Cheap gates come before the compiler.** The sizing gate now ends the project as "not judged" before milestone 1 (`d6:206-225`, `plan:122-123`).
- **Preregistration.**
  - The coverage map, arms, analysis script, simulation, harness, and applications are frozen and hashed before the rules exist (`d6:142-146`).
  - There is a private store run by a non-owner, with an outside timestamp (`d6:244-251`).
  - The owner is kept from hidden material (`d6:79-81`).
- **One definition of "enforced"**, by seeded cases, in both milestones (`d6:119-120`, `:277`). The integrity check uses two seed shapes (`d6:135-136`).
- **The check commands are explicit,** and arm P has no extra linter (`d6:51-53`).
- **Gaps are decided by the run adjudicator,** and arm R is not patched (`d6:127-128`).
- **The reviewer holds no other role** and works from a sealed checklist (`d6:93`).
- **The decision table is ordered** and ends in a catch-all row, so every result maps to one row (`d6:229-239`).
- **The time boxes are bounded.** A re-plan box is no longer than the original box, and a second overrun ends the project (`plan:72-74`).
- **Milestone 2's holes are capped at 5%,** with signatures (`plan:161-165`).
- **The counts are right.**
  - The criteria add to 6 + 7 + 5 + 5 + 5 + 4 = 32, and 28 for the Go version.
  - Milestone 4's task split adds to 8 + 6 = 14.
  - The reviewer's 80 hours match the minimum design: about 290 reviews.
  - K0009 is in Appendix E (`spec:2212`). The plan's references to §5.4d, §7.3, §7.4, and §11.1 are right.
- **The release-1 scope cut is real and testable** (`docs/release-1/syntax.md:13-16`, `:101`). The conformance cases fix positions.

## Status of audit 6's findings

- **B1 (gate 2 cannot count contract rules): resolved.**
  - Milestone 0.5 now counts "the release-1 contract (spec §7) that would reject each seeded case" (`d6:274-276`).
  - There is one definition of "enforced" (`d6:119-120`).
  - Arm C labels UC, and gate 2 pivots "plus arm C's generator for the rules of UC" (`d6:285-286`).
  - New problems: arm C's contracts are undefined (my B2). Paper labels are optimistic (my M8).
- **B2 (power checked only after the compiler exists): resolved in structure, with new problems.**
  - Sizing now runs at the end of milestone 0.5, with four fixed scenarios, a search over tasks and runs, one bounded widening, and "stop, not judged" (`d6:206-225`).
  - New problems: the target is P ≥ 0.5 under a large effect, and the margins can widen to the "real cost" scenario (my H6). The parameters are underspecified (M1). Volunteers do not re-commit to the sized load (H2). Milestone 0.5's own gate 1 is still unsized (my B1, which audit 6 had as M5).
- **H1 (undefined `make check`): resolved.**
  - Each arm's check command is defined (`d6:51-53`), with `-race` (`d6:55`). P has no linter.
  - New problem: in-source suppression is not covered (my M9).
- **H2 (gap fix breaks the freeze): mostly resolved.**
  - The coverage map comes before the freeze, and arm R is "not changed after the freeze" (`d6:61-65`). The run adjudicator decides gaps, and "Arm R is not patched" (`d6:127-128`).
  - Remaining: one person may still be the Go developer, the grader, and the adjudicator (`d6:96`), so self-review is still possible (my H1).
- **H3 (reviewer may know hidden rules): resolved as remediated.**
  - The reviewer "Sees the task brief of the change, and nothing else of the hidden material. Holds no other role" (`d6:93`), and follows a sealed checklist.
  - Remaining: the reviewer is not blind to V against Go, and rejections are unaudited (my H3).
- **H4 (analysis not preregistered): mostly resolved.**
  - Rates are defined (`d6:178-179`), the run without review counts (X2, `d6:164`), task selection is defined (`d6:180-181`), and so is the split above 14 (`d6:107`).
  - The script, simulation, and harness are frozen (`d6:144-146`, `:246-248`), and the analyst is a non-owner (`d6:90`).
  - Remaining: "informative" is undefined (my B1). The number of U\*-targeted tasks above 14 is undefined (M1).
- **H5 (Q2 measures temptation): resolved as remediated.**
  - Q2 is stated as existence, and the untargeted rates are reported (`d6:43-44`, `:203-204`; `plan:43-44`).
  - The optional minimum effect on untargeted tasks was not adopted. So "continue" still rests on tempting tasks. This is a stated limit, not a defect.
- **H6 ("narrow" does not answer the question): resolved.**
  - "Narrow" is a hold, and "milestone 5 and new features wait" (`d6:235`, `plan:33-35`). Arm C answers the generator question before the runs (`d6:66-68`).
  - New problem: the hold has no procedure or limit, and it re-measures on released material (my H4).
- **H7 (volunteer load): partly resolved.**
  - Each role now has an estimated load (`d6:84-94`). Commitments are made while recruiting (`d6:99`). Milestone 0.5 has a budget (`plan:67-68`). The follow-on tasks are gone.
  - Remaining: the loads assume the minimum design, while sizing can multiply them about threefold with no re-commitment (my H2). The volunteers must stay for about a year (M7).

## Minimum changes

To remove the two Blockers:

1. **Gate 1 (B1):**
   - Define "informative task" as a nonzero paired difference (`d6:183`).
   - Size milestone 0.5's gate 1 with the frozen simulation, and require P(pass | L) ≥ 0.8.
   - Record a non-pass whose bound does not exclude a meaningful effect as "stop, not judged", not "stop".
   - Do not stop on Q1 while gate 2 finds at least 5 U\* violations; go to gate 3 instead.
2. **Arm C (B2):**
   - Before each freeze, the Go developer writes every contract of the frozen Vulcan application as a checked comment in the Go application.
   - Parity review verifies the comments one to one; a missing contract is fixed, not logged.
   - The generator covers release-1 tier-1 contracts, or lists each form it does not cover in the coverage map for the run adjudicator.

Strongly recommended, so that the result is credible:

- Translate seeded cases across the arms, and separate the Go developer from the grader and the run adjudicator (H1).
- Make re-commitment to the sized load part of gate 3, and scale milestone 4's box (H2).
- Audit a sample of rejections (H3).
- Define one re-measurement for "narrow", on fresh material (H4).
- Move "not built within the cap" rules out of U\* (H5).
- Raise the sizing target and stop margin widening from reaching L+cost (H6).

## Resolution

Applied on 2026-10-21, in the commit that adds this report, as round 7 of the owner's audit and
remediation cycle:

| Finding | Change |
|---|---|
| B1 | "Informative" is a nonzero paired difference. Milestone 0.5 is sized before its runs for P(Q1 passes \| L) ≥ 0.8. Gate 2 is checked first: evidence that a language is needed goes to gate 3 whatever Q1 shows. A plain "stop" needs a bound that rules out a meaningful effect; otherwise "stop, not judged". |
| B2 | Arm C is a labelling copy that holds, one to one, the contracts on which V's labels rely, written by the Go developer and checked by the run adjudicator. The generator covers the tier-1 contract forms, or lists each form it cannot. Arms R and P never run it. |
| H1 | Seeded shapes are written once and translated into each arm, checked by the run adjudicator. The Go developer is neither the grader nor the run adjudicator (now at least four people besides the owner). |
| H2 | Gate 3 needs each role holder's written commitment to the sized load. Milestone 4's box scales with the sized number of runs. |
| H3 | The second reviewer re-reviews 30% of rejected changes in each arm. N0 must hold with and without the disputed rejections. Rejected changes can be revised within the budget. |
| H4 | "Narrow" re-runs milestone 4 once, with new rules, tasks, and rule writer, at the sized design and budget; a second "narrow" counts as "stop". |
| H5 | The coverage map gives one of three reasons for each rule arm R does not cover. Rules missed only for lack of time are gaps and leave U\*. |
| H6 | Sizing target P(Continue \| L) ≥ 0.8. The one widening stops at 12 points and 1.15, below scenario L+cost. P(Stop \| L) and P(Pivot \| L) are published. |
| M1 | Fixed sizing parameters, frozen with the simulation; target shares above fourteen tasks. |
| M2 | The Go developer revises arms R and C for Stockroom within 10 days. |
| M3 | When only U\* is too small, arms R and P still run. |
| M4 | Reference solutions that keep every rule, per task and arm. |
| M5 | X6 is the holdout-test failures of every run's final change. |
| M6 | N3 is bounded over tasks. |
| M7 | Replacements approved by the run adjudicator. |
| M8 | Stated as a limit; gate 2 counts all tasks. |
| M9 | In-source suppressions count as configuration changes. |
| M10 | The harness runs under the operator's account; logs go to the private store. |
| M11 | Accepted as a risk, stated in the plan. |
| L1 to L12 | Section references, the benchmark name, the spec's latency text, the plan's milestone-4 and gate wording, S08 and `pure`, the pivot wording, the reason for point estimates in row 4, the integrity check in the plan's list, the Go developer's signatures, signed start dates, and decision 6's status line. |
