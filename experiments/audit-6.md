# Audit 6: independent re-audit of the development plan

Commit audited: `fbb7614`. Read only. Paths are relative to `C:\code\vulcan`. Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `spec` = `docs/spec.md`.

*Provenance: written by a third freshly spawned agent, independent of the conversation that wrote
the plan and of the earlier auditors. It wrote its findings before it read `experiments/audit-5.md`,
and did not read audits 3 or 4. Stored as returned. Two claims were spot-checked and held: milestone
0.5 labels U by "a diagnostic in spec Appendix E" (`d6:289`), and `make check` has no defined
contents (`d6:66`).*

## Verdict

**No. The plan does not survive as written.** The structure is sound: a cheap gate before the compiler, preregistration, sealing, mandatory non-owner roles, and an ordered decision table. Two defects break it, though. First, the cheap gate (milestone 0.5, gate 2) defines set U so that contract rules cannot enter it. In milestone 4, contract rules are the main source of U. So the gate leans toward "pivot to the rules" by construction, and it measures a different U than milestone 4 does. Second, rows 2 to 4 all need non-inferiority bounds N0 and N1 at a 10% margin. With 14 to 24 tasks and 5 runs, those bounds probably fail even when the true cost difference is zero. The sizing rule has no instruction for that case, and it runs only after 24 weeks of compiler work. Several High findings leave levers that can bias the result: an undefined `make check`, a post-rule "gap fix" that breaks the freeze, a reviewer who may know the hidden rules, and an analysis that is not fully preregistered.

## Blockers

### B1. Milestone 0.5's gate 2 cannot count contract rules, so it leans toward "pivot" and does not predict milestone 4

- In milestone 0.5, a rule is in U only when "the grader and the Go developer find a diagnostic in spec Appendix E ... that would reject them" (`d6:287-290`).
- Appendix E lists only compile-time codes. Warnings never stop a build (`spec:2196-2197`). A contract violation is a runtime panic, not an Appendix E diagnostic (`docs/release-1/conformance.md:82-84`, C01 and C03). K0501 covers only a contract that the compiler proves false (`spec:2275`). So data-integrity rules enforced by `requires`, `ensures`, or `invariant` cannot enter U in milestone 0.5.
- Every Appendix E error is a static check over the source and the declarations. Arm R's one configuration file can carry the declarations, such as gateways and entry points (`d6:56-57`). The Go developer signs that arm R covers every rule that static analysis can check (`d6:59-61`, `plan:100-101`). So milestone 0.5's U holds only the rules that 15 days of analyzer work missed.
- Gate 2 pivots when U has fewer than 4 rules or fewer than 5 violation events (`d6:296-298`, `plan:109-111`). The likely result is "pivot to the rules". The compiler is then never built, and the contract evidence is never collected.
- Milestone 4 labels U differently. It uses seeded cases rejected by V's check command, which includes `vulcan test` and therefore runtime contracts (`d6:65-66`, `:130-141`). The plan itself expects contract rules to carry T2 (row 3, `d6:262`). So gate 2 can stop the language in exactly the case where milestone 4 would say "narrow".
- **Remediation:** In milestone 0.5, also put a rule in U when a release-1 contract would reject its seeded cases at runtime. The grader and the Go developer write that contract down, record the call, and seal it, as they do for diagnostics. Make gate 2's outcome for a U made mostly of contract rules match row 3 ("narrow", or a named codegen comparison), not "pivot". Use the same "not enforced by R" definition in both milestones (see M9).

### B2. The non-inferiority bounds make "stop" likely regardless of truth, and power is checked only after the compiler exists

- Rows 2, 3, and 4 all need N0 and N1 (`d6:261-263`). "Inconclusive counts as fail" for every bound (`d6:201`).
- N0: the one-sided 95% bound of (R minus V) task success is at most 10 percentage points (`d6:217-218`). With 5 runs per task (`d6:153`) and a success rate near 0.7, the per-task rate has a standard deviation of about 0.20. The paired difference has about 0.29. The bound is about mean + 1.77 × 0.29 / √14 ≈ mean + 0.14 at 14 tasks, and about mean + 0.10 at 24 tasks. At a true difference of zero, N0 fails in most runs at 14 tasks and in about half at 24 tasks.
- N1 is worse. A failed run counts as the full budget (`d6:173`), so time is bimodal. With the same assumptions, the per-task standard deviation of the log ratio is about 0.4. The bound is about mean + 0.19 at 14 tasks, against a margin of log 1.10 ≈ 0.095. These figures are my estimates. The point is that the protocol has no rule for the case where they hold:
  - Sizing raises the task count only "to at most 24" (`d6:236`). It says nothing about what happens if P(Continue | large effect) is still below 0.5. It does not allow more runs per task. It does not define the three scenarios ("moderate" and "large" have no magnitudes, and no scenario fixes a zero cost difference) (`d6:232-238`). It does not name who runs the simulation.
  - Sizing happens "before the files are sealed" in milestone 4. That is after milestones 1 to 3 (24 weeks of boxes, `plan:72-74`). The variance data it needs comes partly from milestone 0.5 (`d6:233-234`), so it could run then.
- So the expensive work is not gated behind the cheapest falsification of all: whether milestone 4 can give an answer at all.
- **Remediation:** Run the sizing at the end of milestone 0.5, before milestone 1, on milestone 0.5's variance data. Fix the scenario magnitudes now, including "large T2 effect, zero cost difference". Allow more runs per task as well as more tasks. State the rule when power stays below the target at the cap: "stop, not judged" before any compiler work, or margins revised and justified before sealing. Publish P(Pivot) and P(Stop) under each scenario too.

## High

### H1. `make check` is undefined, and it decides the labels

- V's check command is `vulcan check && vulcan test`. R and P use `make check`, with no stated contents (`d6:65-67`).
- A rule is "enforced" when the check command rejects every seeded case (`d6:130-131`). If R's `make check` does not run `go test`, any rule that the shared test suite catches becomes U. That is a gain for V by construction that has nothing to do with the language. If it does run the tests, the rule is S. Either way, the label depends on an undefined command.
- P has "a dependency linter whose sealed configuration says what it covers" (`d6:62-64`). An import linter enforces G1-type rules. So P is not purely "written down, not enforced", and its configuration can move T1. No role writes that configuration, and nothing says when.
- **Remediation:** Define each arm's check command in decision 6: `go vet`, `staticcheck`, the dependency linter, R's analyzers, and `go test -race` for the Go arms. Freeze and hash it with the toolchain. Assign P's linter configuration to the Go developer, and write it before the rules exist.

### H2. The "gap fix" breaks the freeze, and the Go developer judges their own gaps

- The order is: freeze, then rules, then labels (`d6:126-127`, `:149-151`, `plan:173`). Yet "a U rule that an analyzer could check within the 15-day cap is a gap in arm R: arm R is fixed and the toolchain re-frozen before sealing" (`d6:134-136`).
- So arm R is changed after the hidden rules are known, and only arm R. V's compiler is never fixed for G rules. The Go developer writes the reason for each U rule (`d6:133-134`), so the person whose coverage is in question decides whether a gap exists. "Within the 15-day cap" does not say whether that means total or remaining days.
- This choice also decides "continue" against "narrow". Row 2 needs at most half of U's events to come from contract rules (`d6:261`). Non-contract U rules exist mainly where arm R is incomplete. So the Go developer's thoroughness sets the outcome row.
- Nothing says who re-labels after a re-freeze.
- **Remediation:** Forbid changes to arm R after the rules exist. Before the freeze, the Go developer certifies coverage against an explicit list (the release-1 error codes). After labelling, the run adjudicator, not the Go developer, decides whether a U rule is an R gap. A gap rule is moved out of U and reported. Arm R is not patched.

### H3. The reviewer may know the hidden rules, and the reviewer decides M1 and part of M5

- Only the rule writer is isolated, and the two reviewers must be different people (`d6:100-101`). Nothing stops the reviewer from also being the task writer, grader, or Go developer, all of whom know the rules or labels (`d6:91-94`). At the minimum staffing of three people, that overlap is the natural assignment.
- The reviewer's pass or fail is part of M1 task success (`d6:169`). The reviewer's minutes are part of M5 (`d6:173`). The reviewer is blind only to R against P, not to V against Go (`d6:97`).
- A reviewer who knows the rules will tend to reject Go changes that break U rules. That lowers R's success and raises its time, which helps V pass N0 and N1. It also moves P against R in row 4.
- **Remediation:** The reviewer holds no role that sees hidden rules, labels, targets, or the FAQ. Seal written review criteria before the runs. If that needs a fourth person, say so in section 3.

### H4. The analysis is not fully preregistered, and the owner builds the measuring tools

- Undefined terms that change results:
  - What "rate" averages: violation count per run, or the share of runs with a violation (`d6:189-190`).
  - How a run that never reaches review counts in M2.
  - Which tasks enter each test: all tasks, or only the targeted ones.
  - Whose events form the denominator of "U's violation events from contract rules": R's, or both arms' (`d6:261-262`).
  - Whether follow-on runs count in M1 and M5 (`d6:115-117`).
  - What "the same split" of tempting tasks means above 14 tasks (`d6:236-237`).
- The analysis script, the harness, and the sizing simulation are not in the freeze list (`d6:149-151`) or the sealed store (`d6:272-274`). No role owns the analysis script or the simulation. The owner builds the harness (`d6:84`), which measures agent time (M5) and enforces budgets.
- **Remediation:** Define each term in section 8. Write, hash, and timestamp the analysis script, the harness, and the simulation with the freeze. A non-owner writes the analysis script.

### H5. T2 passes almost by construction once U has 4 rules, so it measures temptation, not practical benefit

- V's check rejects every seeded U case (`d6:130-133`). A change goes to review only after the check passes (`d6:66-67`). So V's U violations at review are near zero, except where an agent's violation differs from the seeded case or uses an exemption.
- T2 then passes when R's agents break U rules in at least 5 targeted tasks with at least 10 events (`d6:192-199`). Tempting tasks are built to be "easiest to meet by breaking a hidden rule" (`d6:109-110`). They are written by someone who knows the labels (`d6:150-151`).
- So T2 shows that enforcement exists. It does not estimate how often the language prevents a violation in ordinary work. Eight of fourteen tasks are tempting, which also loads the cost bounds.
- **Remediation:** Report violation rates on untargeted tasks as the estimate of practical benefit. State in the report and in `plan:28-29` that T2 shows that an effect exists, not its size in normal work. Consider a minimum effect on untargeted tasks as a reported condition of "continue".

### H6. The likely positive result, "narrow", does not answer the plan's question

- This is a consequence of the owner's scope choice, which I accept as given. Arm R has no code generation, so contract rules fall to V "by definition" (`d6:78-80`).
- When contract rules carry T2, row 3 says "narrow" and names a Go code generator comparison as "the next step" (`d6:262`). No milestone, budget, owner, or decision rule exists for that comparison.
- The plan's Narrow keeps "the language features behind the rules that helped" (`plan:30-32`). Milestone 5 may then start on "narrow" (`plan:182`). So in the probable positive case, language work continues before the language-versus-generator question is answered.
- The keep criterion, "U rules with violations in arm R" (`d6:262`), covers nearly every U rule.
- **Remediation:** Make "narrow" a hold, not a continuation. The next step is a preregistered codegen-arm comparison with its own budget and gate. No milestone 5 work and no new language features start until it ends.

### H7. The volunteer load is probably not feasible, and it is checked late

- The Go developer carries:
  - milestone 0.5 analyzers (up to 15 days);
  - the Stockroom Go build or a timed review;
  - the parity review of the V application;
  - the milestone 2 sign-off;
  - the D5, V1, and V2 checks;
  - a written reason for each U rule.

  Sources: `d6:91`, `plan:149-150`, `docs/release-1/acceptance.md:217-220`.
- The reviewer times each reviewed change:
  - Milestone 0.5: 8 × 5 × 2 = 80 runs.
  - Milestone 4: 14 × 5 × 3 = 210 main runs.
  - Pilot: 2 × 5 × 3 = 30 runs.
  - Follow-ons, if run 5 times per arm: 8 × 5 × 3 = 120 runs.
  - Total: about 440 reviews, plus the second reviewer's 30% sample.
- The same volunteers must stay for about 44 weeks: 8 for recruiting plus 36 weeks of boxes (`plan:65-74`). Milestone 4's box is 6 weeks. The budget is committed only before sealing in milestone 4 (`d6:237-238`). Section 12 has no budget at all.
- **Remediation:** Estimate the hours for each role at plan acceptance, and get commitments while recruiting. Make the follow-on runs (M14, reported only) optional or smaller. Commit a budget for milestone 0.5 and check milestone 4's budget at the end of milestone 0.5.

## Medium

- **M1. The label check is a tautology.** S and U are defined by the same seeded-case test that the label check repeats (`d6:130-137` against `:211-213`). It cannot fail unless the toolchain changed. It does not show that agents' violations, as opposed to the grader's seeded ones, are rejected. **Fix:** require at least two differently shaped seeded cases per rule and arm, and re-run them on the frozen hash at run time. Call the step an integrity check.
- **M2. The re-plan box has no bound.** The owner writes the re-plan note with "one new box" (`plan:60-62`). The adjudicator only signs it. A large new box defeats the schedule stop. **Fix:** cap the new box, for example at the original box.
- **M3. Milestone 2's gate has an unlimited escape hatch.** Any case can be "marked as a known hole" (`plan:148-150`). The Go developer checks the list "against §5.4d", which does not cover contracts (`spec:612-688`). **Fix:** cap the holes. Each hole in a supported release-1 construct needs the adjudicator's signature and goes into the milestone 4 report.
- **M4. The number of hidden rules is not set.** If the rule writer writes few rules, S or U falls below 4 (`d6:197-199`), and the result is fixed as "fail" before any run. **Fix:** set a minimum rule count in the brief. If S or U has fewer than 4 rules after labelling, record the outcome then and skip the runs.
- **M5. Milestone 0.5 is unsized, and its inconclusive result is reported as a finding.** Eight tasks, with at least five targeting S (`d6:292-293`), means T1 passes only if nearly every informative task favors R. The plan says "If the rules do not help (T1 fails), stop" (`plan:108`), but inconclusive also counts as fail (`d6:300`). **Fix:** size milestone 0.5 by the same simulation, and record an inconclusive stop as "not judged", not as "the rules do not help".
- **M6. Material leaks between milestones.** Milestone 0.5's sealed material is copied into `experiments/m0.5/` after its runs (`d6:279-280`). That is before the owner writes Stockroom's contracts in milestone 3. Milestone 4 uses the same rule-writer process on a similar order domain (`plan:102`, `docs/release-1/acceptance.md:122-124`). The Go developer also saw milestone 0.5's rules while fixing gaps. The owner also wrote the public F, D, S, and R criteria that form the rule writer's brief (`d6:123-124`). The contract-share cap in row 2 limits the effect. **Fix:** keep milestone 0.5's material in the private store until milestone 4 ends, and use a different rule writer in milestone 4.
- **M7. The toolchain-bug restart exposes hidden material.** The owner fixes the bug "under a new tag" (`d6:158-161`). To do that, the owner must read failing runs, which contain task briefs (`d6:85-86`). Nothing limits what the new tag may change, and nothing says whether the first schedule's data is reported. **Fix:** a non-owner shows the owner a minimal reproduction. The adjudicator reviews the diff of the new tag. Report the first schedule.
- **M8. Plan and decision 6 disagree on "stop, not judged".** Row 1 lists empty roles, a failed label check, and section 6 inconclusive (`d6:260`). The plan lists unfilled roles and "could not finish" (`plan:36-37`). Neither covers a budget that would be exceeded (`d6:237-238`). The recruiting deadline appears only in the plan (`plan:65-67`). **Fix:** align row 1 with the plan, and add the budget and deadline cases.
- **M9. Milestones 0.5 and 4 define "not enforced by R" differently.** Milestone 0.5 says "reject none of its seeded cases" (`d6:288`). Milestone 4 says R does not reject every seeded case (`d6:130-133`). In milestone 0.5, a rule that R rejects partly is neither S nor U. **Fix:** use one definition.
- **M10. False positives (N3) are undercounted.** M8 counts diagnostics "that the reviewer judges wrong" (`d6:176`), and N3 divides by accepted changes (`d6:223-224`). A false positive that the agent worked around, or that made a run fail, never reaches the reviewer or the denominator. **Fix:** judge every V diagnostic in the harness log, per run.
- **M11. Duties have no owner.** Nobody is named to write the FAQ (`d6:96`), build milestone 0.5's application (`plan:102`), run the harness during runs, or run the sizing simulation. If the owner runs the harness, the owner sees the task briefs, which contradicts `d6:85-86`. **Fix:** assign each of these in section 3.

## Low

- **L1. Name collisions.** Acceptance criteria T1 to T5 (`docs/release-1/acceptance.md:199-203`) share names with tests T1 and T2. Measurements M1 to M14 share names with milestones. `plan:184` writes "(M9)". **Fix:** rename one family.
- **L2. Milestone 1 omits a deliverable.** Milestone 1 does not list the written design of Stockroom. Milestone 2 needs it to decide `pure` (`plan:145-146`, `docs/release-1/syntax.md:57`).
- **L3. Spec §1.1 overstates N2.** It says N2 "tests the defect part of the third hypothesis" (`spec:51-52`). N2 is a non-inferiority bound. It cannot test "does not reduce defects". The premise's main benefit, shorter review (`spec:27-29`), decides nothing (M6, `d6:230`).
- **L4. M14 follow-ons do not fit arm V.** A V base that contains a seeded S or U violation fails V's own check, so V's follow-on starts from a failing state (`d6:115-117`).
- **L5. Agents can edit the configuration.** Nothing says whether an agent may edit `system.vulcan` (`strict_exempt`, `allow_network`) or R's configuration file to pass the check. M9 decides nothing (`d6:230`).
- **L6. The milestone 1 gate has no signer.** The run adjudicator signs only milestones 0.5, 3, and 4 (`d6:95`).
- **L7. The traceability table is vague.** It says "Mostly answered by the audit-5 changes" for audit 4 findings, without naming which remain open (`plan:225`).
- **L8. Some text is stale.**
  - The repository layout in `README.md:101` omits audits 3 to 5.
  - `docs/status.md:232-233` still asks to rebuild `base-keel` before grading, while `docs/status.md:256` says milestones 0.5 and 4 replace test 3.

## What holds up

- **A cheap gate first.** Milestone 0.5 runs before any compiler work, and the roles come first with an 8-week deadline (`plan:65-67`, `:94-113`).
- **Preregistration.** The text has fixed thresholds, a private store run by a non-owner, an outside timestamp, isolated workspaces, and release with hash checks (`d6:270-280`).
- **The freeze order, in principle** (`d6:149-151`), apart from H2.
- **Mandatory roles with no escape clause** (`d6:100-103`). A rule writer who has not read the spec. Reviewers who differ, and a reviewer blind to R against P.
- **The decision table.** The rows apply in order with a catch-all, so every result maps to exactly one row (`d6:256-266`). "Narrow" now needs N0 and N1. A result carried by contracts cannot reach "continue".
- **The arithmetic is right.** 1/32 < 0.05 < 1/16, so a test needs at least 5 informative tasks (`d6:194-195`). The task split is 8 + 2 + 1 + 1 + 2 = 14. The criteria are 6 + 7 + 5 + 5 + 5 + 4 = 32, and 28 for Go.
- **References are right.** Every K-code I checked exists in Appendix E: K0009, K0101 to K0125, K0201 to K0211, K0501 to K0525, K0601 to K0605, and K0701 to K0705. The plan's section references to decision 6 (sections 2, 3, 9, 10, 12) are right. Its references to spec §5.4d, §7.3, §7.4, and §11.1 are right.
- **The scope cut is real.** Release 1 rejects deferred constructs with K0009 and deferred commands with exit 2 (`docs/release-1/syntax.md:13-16`, `:101`).
- **Merged violations are visible.** Acceptance no longer requires the hidden-rule checks (`d6:184-185`).
- **The caveat about arm R is stated in every report** (`d6:78-80`).

## Status of audit 5's findings

- **B1 (Continue unreachable; Narrow without a cost floor): resolved as written, with new problems.**
  - T3 is gone. Equal enforcement of S is now a label check: "every seeded case of every S rule is rejected by both V and R" (`d6:211-213`).
  - Narrow needs N0 and N1: "T2 passes, N0 and N1 pass, and ..." (`d6:262`).
  - New problems: the label check is a tautology (M1). The difficulty of reaching "continue" has moved to the power of N0 and N1, which now also gate "narrow" and "pivot" (my B2).
- **B2 (set U has no workable definition): partly resolved.**
  - R's means are now operational: "static analyzers and one configuration file, and nothing else", with a 15-day cap (`d6:55-61`).
  - U is labelled by seeded cases (`d6:130-136`), and U is graded by runtime tests and a blinded grader (`d6:142-145`).
  - Milestone 0.5 uses the same rule process (`d6:291`).
  - Still open:
    - Milestone 0.5 labels U by "a diagnostic in spec Appendix E" (`d6:289`), which excludes contract rules. So gate 2 still measures a different U than milestone 4 (my B1).
    - The "one person's call" problem has moved into the gap rule, where the Go developer judges their own coverage (my H2).
- **H1 (temptations decide; the owner writes them): resolved as remediated, but the mechanism remains.**
  - A non-owner task writer writes all tasks and tags their targets, at least five per set (`d6:93`, `:107-114`).
  - Acceptance no longer requires the hidden checks (`d6:184-185`).
  - Spec §1.1 says that H-A is "measured, but not tested" (`spec:48-52`).
  - T2 still passes if and only if R's agents break U rules on at least 5 tempting tasks (my H5).
  - Milestone 0.5 has no minimum number of U-targeted tasks (`d6:292-293`).
- **H2 (statistics and power): partly resolved.**
  - The number of tasks is now sized by simulation (`d6:232-238`).
  - N3 to N5 have defined tests (`d6:223-229`). The bounds are one-sided 95% (`d6:193`).
  - Still open:
    - The N1 variance problem that audit 5 computed remains, and N0 and N1 now gate three rows.
    - The sizing rule has no instruction when 24 tasks are not enough, no defined scenarios, and no option to add runs.
    - The sizing runs only after milestone 3 (my B2).
  - N3's measurement is incomplete (my M10).
- **H3 (owner control): mostly resolved, with new gaps.**
  - The owner does not see hidden material (`d6:85-86`). The freeze comes first (`d6:149-151`).
  - A non-owner grader writes the tests (`d6:94`). The reviewers differ, and the reviewer is blind to the Go arm (`d6:97-101`).
  - New or remaining gaps:
    - The post-rule re-freeze of arm R (my H2).
    - A reviewer who may know the hidden rules (my H3).
    - An owner-built harness and an analysis script that are not frozen (my H4).
    - The toolchain-bug fix, which exposes runs to the owner (my M7).
    - Milestone 0.5 material released before Stockroom is built (my M6).
- **H4 (schedule; milestone 0.5 not cheap): partly resolved.**
  - Recruiting now has its own 8-week deadline, and milestone 0.5's box starts when the roles are filled (`plan:65-67`). The box is 6 weeks (`plan:71`).
  - A budget exists (`d6:237-238`).
  - Still open:
    - The scope of arm R in milestone 0.5 was not cut. Up to 15 of its working days fall inside a 6-week box, before the rules, tasks, sealing, and 80 runs.
    - The budget covers milestone 4 only, and it is committed late.
    - The re-plan box is unbounded (my M2, H7).
- **Audit 5's Medium and Low items: mostly fixed.**
  - Fixed: the section references (`plan:144`), the outcome count (`plan:25-26`), the decisions index (`docs/decisions/README.md:19`), the Go-plain wording (`docs/release-1/acceptance.md:128-129`), the order and wording of S09 and S10 (`docs/release-1/conformance.md:40-41`), decision 4's latency text (`docs/decisions/0004-compiler-structure.md:109-110`), and the person-checked criteria (`docs/release-1/acceptance.md:217-220`).
  - The fix for unanswered runs holds (`d6:155-156`).
  - Remaining: Row 3's "features it keeps" is now defined, but too broadly (my H6). Milestone 2's gate has a script and a signer, but its "known hole" escape is unlimited (my M3).

## Minimum changes

1. **Gate 2 (B1).** In milestone 0.5, count a rule in U when a release-1 contract would reject its seeded cases, not only an Appendix E diagnostic. Use one "not enforced by R" definition in both milestones. When U is made mostly of contract rules, gate 2's outcome follows row 3, not "pivot".
2. **Sizing (B2).** Move the sizing simulation to the end of milestone 0.5, as a gate before milestone 1. Fix the scenario magnitudes, including a zero-cost-difference scenario. Allow more runs per task. State the outcome when power is unreachable within the cap and the budget.
3. **Check commands (H1).** Define `make check` for R and P, including `go test`. Freeze it. Assign and pre-date P's linter configuration.
4. **Gap rule (H2).** Remove the post-rule fix of arm R. The run adjudicator classifies gaps, and gap rules are reported, not patched.
5. **Reviewer (H3).** Bar the reviewer from every role that sees hidden material. Seal the review criteria.
6. **Analysis (H4).** Define the open statistical terms. Freeze and hash the harness, the analysis script, and the simulation. Assign the script to a non-owner.
7. **Narrow (H6).** Make "narrow" a hold until a preregistered codegen comparison, with its own budget, ends. No milestone 5 work starts on "narrow".
8. **Workload (H7).** Commit volunteer hours for each role at recruiting. Budget milestone 0.5. Trim or drop the follow-on runs.

## Resolution

On 2026-10-21 the owner asked for the audit and remediation cycle to repeat until an independent
audit finds no blocker. For this round, decision 6 and the development plan were rewritten in full,
in the commit that adds this report:

| Finding | Change |
|---|---|
| B1 | Milestone 0.5 counts release-1 contracts, as well as diagnostics, as Vulcan's enforcement. One "enforced" definition for both milestones. Arm C, a contract generator used only to label rules, splits U into UC (a generator would deliver it) and U\* (nothing in Go would); Q2 uses U\*. The contract-share rule is gone. |
| B2 | A sizing gate at the end of milestone 0.5, before any compiler work: four fixed scenarios, a search over 14 to 24 tasks and 5 to 10 runs, one bounded widening of the margins, and "stop, not judged" if no design passes. |
| H1 | Every arm's check command is written out. Arm P has no dependency linter. Agents may not change check configuration. |
| H2 | A coverage map before the freeze, reviewed by the run adjudicator. Arm R is never patched after the freeze. The run adjudicator moves gap rules out of U. |
| H3 | The reviewer holds no other role, sees only the task brief, and follows a checklist written before the rules exist. |
| H4 | Rate, the run with no review, task selection, and the split above fourteen are defined. The analysis script, simulation, and harness are frozen and hashed; a non-owner analyst writes the first two. |
| H5 | Q2 is described as showing that an effect exists. Rates on untargeted tasks are reported as the estimate of benefit. |
| H6 | "Narrow" is a hold, used only when a secondary bound fails. The generator question is answered by arm C before the runs. |
| H7 | Estimated load per role; commitments while recruiting; a budget for milestone 0.5 at recruiting and for milestone 4 from the sizing gate; follow-on tasks removed; milestone 0.5's box is 10 weeks from the day the roles are filled. |
| M1 | Two differently shaped seeded cases per rule and arm; an integrity check before and after the runs. |
| M2 | A new box after a re-plan is no longer than the original box. |
| M3 | At most 5% of conformance cases may be holes at milestone 2's gate, each signed, all reported. |
| M4 | The rule writer is asked for at least 30 rules; too few rules are recorded as an outcome without runs. |
| M5 | Milestone 0.5 may extend once; a still-inconclusive gate 1 is "stop, not judged". |
| M6 | Milestone 0.5's material stays private until milestone 4 ends; a different rule writer for milestone 4. |
| M7 | A non-owner gives the owner a minimal reproduction; the run adjudicator reviews the fix; the first schedule is reported. |
| M8 | Row 1 lists every "not judged" cause, matching the plan. |
| M9 | One definition of "enforced". |
| M10 | False positives are judged from every Vulcan diagnostic in the run logs, per run. |
| M11 | The FAQ, milestone 0.5's application, the harness during runs, and the simulation each have an owner. |
| L1 | Questions are Q1 and Q2; measurements are X1 to X12. |
| L2 | Milestone 1 writes the application's design. |
| L3 | Spec §1.1: N2 only checks that Vulcan does not add defects. |
| L4 | Follow-on tasks removed. |
| L5 | Agents may not change check configuration. |
| L6 | The Go developer signs milestone 1's gate. |
| L7 | The traceability table names what stays open. |
| L8 | README layout and status.md updated. |
