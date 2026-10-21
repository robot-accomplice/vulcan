# Audit 5: independent re-audit of the development plan

Commit audited: `9386024`. Read only. Paths are relative to `C:\code\vulcan`.

*Provenance: written by a second freshly spawned agent with no access to the conversation that
produced the plan, or to the agent of audit 4. It wrote its findings before it read
`experiments/audit-4.md`, and did not read `experiments/audit-3.md`. Stored as returned. Two
concrete claims were spot-checked and held: the plan's reference to "decision 6, section 8" for
the latency targets, and the combination of "review only after the check passes" with the
10-event minimum that makes T3 inconclusive.*

## Verdict

**No. The plan does not survive as written.** Audit 4's amendments fixed the old failure, where "continue" could pass by construction. They created the reverse failure. "Continue the language" can now almost never be reached. "Narrow" keeps the language at any cost. The deciding test then depends on a set (U) that has no workable definition, and on temptation tasks that the owner writes. The structure is still good: preregistration, three arms, a cheap gate first, and sealing. The fixes are text changes to decision 6, and none of them needs new work.

## Blockers

### B1. "Continue" is unreachable by construction, and "Narrow" has no cost floor, so T2 alone decides

- A change goes to review only after its arm's check command passes (`docs/decisions/0006-evidence-protocol.md:65-68`).
- Set S holds the rules that both V and R enforce (`0006:116`). So in both arms, violations of S rules at review are zero by construction. The only exceptions are checker holes.
- A violation comparison needs at least 10 violation events at review across both arms. Below that it is inconclusive (`0006:169-171`), and inconclusive counts as fail (`0006:173`).
- T3 is a violation comparison on S (`0006:185-186`). It will be inconclusive, so it fails. Row 2, "Continue the language", needs T3 to pass (`0006:220`). Row 2 is therefore almost unreachable.
- Row 3, "Narrow", needs only T2. It applies when "T3 or a bound of N0 to N5 fails" (`0006:221`). A Vulcan arm that halves task success or doubles working time still narrows. It does not stop, and the language continues.
- Net effect: the realistic outcomes are "T2 passes → Narrow (the language continues, at any cost)" and "T2 fails → Pivot or Stop". The plan's own definitions do not hold. It says Narrow means "some cost fails" (`docs/development-plan.md:30-31`), and Continue means "no worse than Go with its rules" (`development-plan.md:28-29`).
- **Remediation:**
  - Exempt T3 from the minimum-event rule. Equal enforcement is a label-level fact, so test it at label level instead: each S rule has a seeded-violation case that V's check rejects. Or drop T3 and say why.
  - Make Narrow require N0 and N1 (or stated looser floors). When they fail, the row is Stop, or Pivot if T1 passes.

### B2. Set U, which decides language versus Go tool at both gates, has no workable definition

- Arm R holds every rule that "Go code, comments, or configuration can express". This includes contracts "as checked comments with a generator" (`0006:58-62`). A generator from comments can express any static rule that Vulcan checks: Vulcan itself is a generator from annotated Go-like source.
- U needs the Go developer's certification that a rule is "not expressible in Go code, comments, or configuration" (`0006:117-118`). Gate 2 of milestone 0.5 uses the same certification (`0006:247-249`). No criterion, effort bound, or tool limit makes this a decidable judgment. One person's unguided call can therefore set U to empty (pivot) or to large (build the compiler).
- The definition also contradicts itself. Milestone 0.5 counts violations of U rules in Go code (`0006:253-255`), and milestone 4 scores them through "hidden rule checks" (`0006:144`). Grading must be scripted where it can be (`0006:29`). A script that detects a rule's violation in Go code is a Go-expressible check, so that rule is not in U. U is then either empty, or it is graded by an unspecified non-script method.
- Gate 2 tests a different population from T2. In milestone 0.5, U comes "from the spec" (`0006:247`). In milestone 4, U comes from hidden rules written by someone who never read the spec (`0006:107-118`). Passing gate 2 does not predict that the rule writer's rules will put 4 or more rules in U. So the expensive milestones 1 to 3 are not gated on the quantity that decides milestone 4.
- **Remediation:**
  - Define R's means operationally: `go/analysis` analyzers plus one configuration file, with or without a contract generator, decided now, and a time cap for the Go developer.
  - Define U as "enforced by V and not by R's sealed analyzers", with a written reason for each rule. Report the cost of closing each gap separately.
  - State how U violations are graded. For example: a non-owner writes seeded-violation scripts against the declared architecture, or a blinded human grader scores them.
  - Run milestone 0.5's gate 2 on hidden rules from the same rule-writer process, so that it measures the same U as milestone 4.

## High

### H1. T1 and T2 measure how often temptations work, and the owner writes the temptations

- The enforcing arm has zero violations at review by construction (see B1). With the exact sign-flip test, a pass needs at least 5 tasks that all differ in one direction (1/2^5 = 0.031; 1/2^4 = 0.0625; `0006:164-167`). So T2 passes if and only if R's agent breaks a U rule in at least 5 of the tempting tasks, with at least 10 events.
- The owner writes all fourteen tasks (`0006:96-99`), yet "makes no judgment that affects a result" (`0006:78-79`). Nothing fixes which rules or sets the tempting tasks target. Eight tempting tasks split between U and S leave one set with 4 or fewer informative tasks, unless tasks tempt both. In milestone 0.5, 6 tempting tasks must serve T1 (5 or more needed) and gate 2's U count at once (`0006:250-256`).
- The benefit that H-A claims is not measured: that rules leave review, and that changes stay clean at merge. M1 requires the hidden rule checks to pass (`0006:144`), so M3, violations at merge (`0006:146`), is zero by construction. M6 decides nothing (`0006:197`).
- **Remediation:**
  - Preregister an allocation: at least 5 tempting tasks per tested set, each tagged with its target rules by the task reviewer from the sealed rule list.
  - Have a non-owner draft or approve the targets.
  - State in spec §1.1 that T2 tests H-B only, and that H-A's review benefit is not tested.

### H2. The statistics cannot give "Continue" a fair chance, and the power rule is incoherent

- Continue needs 8 conditions to pass together: T2, T3, and N0 to N5 (`0006:220`). Even at 80% power each, the joint chance is about 0.8^8 ≈ 0.17. Inconclusive counts as fail.
- Power is set by the number of runs per task, from a two-task pilot (`0006:102-103`, `0006:127-130`). The tests are over tasks (`0006:161-165`), so more runs do not add tasks. Two tasks cannot estimate between-task variance.
- N1 over 14 tasks: a one-sided 95% bound is about mean + 1.77·SD/√14 ≈ mean + 0.47·SD. To pass at a true difference of zero, the SD of the per-task log ratios must be 0.20 or less, against two different codebases and a language the model has never seen.
- "Each by the same test on paired per-task differences" (`0006:188`) cannot apply to N3 (a V-only rate, `0006:193`), N4 (one benchmark, `0006:194`), or N5 (fixed targets, `0006:195`).
- "α = 0.05" one-sided with "a 95% interval" (`0006:165`) is ambiguous between a one-sided 95% bound and a two-sided 95% bound.
- **Remediation:**
  - Size the number of tasks, not runs, by simulation over plausible task heterogeneity. Publish the expected P(Continue | real effect) before sealing, and revise the margins if it is low.
  - Define the tests for N3 to N5.
  - Say "one-sided 95% bound".

### H3. The owner still controls inputs that decide the result

The roles are mandatory now, but these inputs remain with the owner or have no assignee:

- the tasks (`0006:96`);
- arm R's analyzers, which the Go developer only signs for coverage, not for quality or diagnostics (`development-plan.md:95-96`);
- the hidden HTTP tests and the grading scripts (no author in `0006:230-231` or section 3);
- the V application;
- the conformance suite, which sets V's labels (`0006:114-115`).

Nothing orders the compiler tag freeze (`0006:132`) before the owner sees the hidden rules. The owner can add checks or conformance cases after reading them, and move rules from N to U or S.

The reviewer is not blinded and makes decisive calls: review pass inside M1 (N0), and false positives (N3). The Go developer may hold every role except rule writer (`0006:91`), including both reviewer and second reviewer, so the 30% audit (M7, N2) can be self-audit. This also contradicts "at least three people besides the owner" (`0006:269-271`).

**Remediation:**
- Freeze and hash the compiler tag, the analyzers, and the V and Go applications before the hidden rules leave the rule writer.
- A non-owner writes the grading scripts and the hidden tests.
- The reviewer and the second reviewer are different people.
- Blind the reviewer to R versus P, which share one codebase.

### H4. The schedule contradicts itself, and milestone 0.5 is not cheap

- Recruiting is part of milestone 0.5 and has an 8-week deadline (`development-plan.md:93-94`). The milestone's box is 4 weeks, and an overrun of more than half (6 weeks) forces a re-plan (`development-plan.md:58-61`, `:65`). A slow but legal recruitment already counts as an overrun.
- In 4 weeks, milestone 0.5 must build analyzers for G1 to G9, including reach analysis into third-party code (§5.4d), the strict checks K0123 to K0125, K0210, and K0211, and a contract generator. It must also rebuild the shop, seal everything, and run 48 or more reviewed runs (`0006:58-62`, `:250-251`). This is most of milestone 2's analysis work.
- Milestone 4 has a 6-week box. It runs 14 × 3 × 5 to 10 = 210 to 420 main runs, plus 8 × 3 × 5 to 10 = 120 to 240 follow-on runs (`0006:100-101`). A volunteer reviews and times each accepted change. No token budget exists.
- **Remediation:**
  - Start the box after recruiting.
  - Re-box milestone 0.5, or cut R's scope in milestone 0.5 to the architecture analyzers.
  - Budget the volunteer hours and the tokens.

## Medium

- **M1. Milestone 0.5 terms are undefined.**
  - S is "enforced by both V and R" (`0006:116`), and no V exists in milestone 0.5.
  - The rule writer's brief is Stockroom's criteria (`0006:107-108`), not the shop's.
  - Gate 2 pivots without the N0 and N1 check that row 4 requires (`0006:222` against `:253-256`).
  - Spec-defined U can include deferred features that release 1 rejects.
  - **Fix:** define each for milestone 0.5.
- **M2. The arms' inputs are ambiguous.**
  - P gets "a rule document" (`0006:71-72`), but P is "Go without the rules" (`development-plan.md:159`).
  - P shares R's code (`0006:74`), so it may also hold R's contract comments and generated checks.
  - V gets the full spec, which includes deferred constructs that K0009 rejects (`docs/release-1/syntax.md:106-108`).
  - "Same total length" forces padded Go documents of about 25,000 words.
  - **Fix:** state exactly what each arm receives.
- **M3. "Unanswered" runs have no scoring.** `0006:87` says that such a run ends "for every arm alike", and `0006:262` lists them, but they count neither as failures nor as exclusions. V agents will likely ask more questions. **Fix:** count an unanswered run as a failure of its arm.
- **M4. V against R compares two codebases written by different people.**
  - Task difficulty and N4's benchmark (`0006:194`) mix the language with the implementation.
  - Only the Go side gets an outside review (`development-plan.md:147-148`).
  - **Fix:** have the Go developer also review the V application for parity.
- **M5. Three metrics are empty or tautological.**
  - M7's "failures of the hidden HTTP tests after acceptance" (`0006:150`) is zero, because acceptance requires those tests to pass (`0006:144`). N2 then rests on a 30% sample.
  - P2 for an enforcing arm is near 100% by construction (`0006:178`).
  - Row 3's "rules of set U that carried T2" (`0006:221`) is undefined.
- **M6. The owner judges every exit gate and every time-box overrun.**
  - The time-box unit, "weeks of the owner's work" (`development-plan.md:58`), is self-reported.
  - "A second overrun" does not say whether the re-plan sets a new box.
  - Milestone 2's gate, "accurately exposed" (`development-plan.md:140-141`), and milestone 3's "diagnoses one injected fault" (`:151-153`) have no criterion or judge. This is audit 4 M2, still open.

## Low

- `development-plan.md:137-138` points to "decision 6, section 8" for the latency targets. They are in section 9 (`0006:200-211`).
- `development-plan.md:25-26` says that "the first four" outcomes come from section 10. Section 10 has five rows, including "Stop: not judged" (`0006:219-223`).
- `docs/decisions/README.md:19` still says "amended by audit 3". It omits audit 4.
- `docs/release-1/acceptance.md:14-15` gives Go-plain "only a dependency linter". `0006:63-64` adds `go vet` and `staticcheck`.
- `docs/release-1/conformance.md:41`: range-over-func iterators came in Go 1.23, so they are not "a Go 1.26 feature". S10 is listed before S09 (`:39-41`).
- `docs/decisions/0004-compiler-structure.md:109-110` still says that milestone 2 measures "on the milestone-3 application". This contradicts `0006:209-210`.
- `syntax.md:57` and `development-plan.md:130`: `pure` is added in milestone 2 "only if the milestone-3 application needs them". That application does not exist yet (audit 4 M1, half unfixed).
- `acceptance.md:106` says that person-checked criteria are "none today". V1 and V2 need a person.

## What holds up

- Preregistration: sealed files, a committed seed, a scripted analysis, every exclusion listed (`0006:131`, `:228-240`, `:259-263`).
- An external timestamp and isolated workspaces (`0006:232-238`).
- The three-arm framing, and the principle that "a language must beat its own rules delivered as a linter" (`0006:45`).
- The permutation arithmetic is right: 1/32 < 0.05 < 1/16, so at least 5 informative tasks are needed (`0006:165-167`).
- The task count is right: 8 + 2 + 1 + 1 + 2 = 14.
- The criteria counts are right: 32 = 6 + 7 + 5 + 5 + 5 + 4, and 28 for the Go version.
- Every K-code used in the plan, decision 6, and release 1 exists in Appendix E. I checked them all.
- One local check command per arm. A rule writer who never read the spec, with a functional brief and rules sealed before labelling.
- A frozen toolchain, with one rerun of the full schedule for all arms.
- Mandatory roles with no escape clause, and a recruiting deadline.
- Release 1 is a real scope cut. It rejects deferred constructs with K0009 and deferred commands with exit 2 (`syntax.md:13-16`, `:101`).

## Status of audit 4's findings

- **B1 (linter arm scoped so that continue passes): resolved as written, with a new blocker.** Arm R now holds "every rule that Go code, comments, or configuration can express", signed by the Go developer (`0006:57-62`). Rules are labelled per arm (`0006:113-122`). But "tie on S" became T3 under the 10-event minimum, which makes Continue unreachable (my B1). "Expressible" has no definition (my B2).
- **B2 (statistics): mostly resolved.** The text now has paired per-task differences with an exact sign-flip test (`0006:161-167`), minimum counts (`0006:169-171`), P2 undefined below 10 events (`0006:178-179`), and 6 tempting tasks in milestone 0.5 (`0006:250`). Still open: power sized by runs from a two-task pilot (`0006:127-130`), and undefined tests for N3 to N5 (`0006:188`). The minimum-count rule causes my B1.
- **B3 (optional independence): partly resolved.** Roles are mandatory with no escape (`0006:91-92`, row 1 at `:219`). A task reviewer, a run adjudicator, and an FAQ-only operator now exist (`0006:85-87`). Still open: the owner writes the tasks (`0006:96`), and builds R's analyzers and, by default, the grading scripts. The compiler freeze is not ordered before the owner sees the rules. The reviewer and second reviewer can be one person. The owner judges the exit gates (my H3, M6).
- **B4 (cheap gate does not test language versus linter): partly resolved.** Gate 2 exists (`0006:253-256`, `development-plan.md:104-105`). Its U comes from the spec, not from the hidden-rule process. Its U cannot be graded by script, by definition. Its pivot skips the N0 and N1 check (my B2, M1).
- **H1 (tool placement): resolved** (`0006:65-68`). New problem: review after a passing check makes violations at review zero in the enforcing arm. That empties T3 and makes P2 tautological (my B1, M5).
- **H2 (primed rule writer): resolved for milestone 4** (`0006:107-110`). The filter has a judge (`0006:85`, `:111-112`). Not resolved for milestone 0.5, whose brief is Stockroom's (my M1).
- **H4 (decision rule): resolved formally.** The table is ordered with a catch-all (`0006:215-226`), and the plan lists six outcomes (`development-plan.md:23-37`). New problem: Narrow has no cost floor and is now the realistic best result (my B1). "Carried" is undefined (my M5). The plan's "first four" wording is wrong (Low).
- **H6 (hidden materials in the repository): resolved** (`0006:228-240`). Gap: the owner holds the private store and can see its contents before freezing the compiler (my H3).

## Minimum changes

1. Fix T3: make it a label-level check, or exempt it from the event minimum. Make Narrow require N0 and N1.
2. Define arm R's permitted means and U operationally, with a grading method for U. Run milestone 0.5's gate 2 on hidden rules from the same process as milestone 4.
3. Preregister how tempting tasks are allocated to rule sets, under a non-owner. Freeze and hash the compiler, the analyzers, and both applications before any hidden material reaches the owner. Assign the grading scripts and hidden tests to a non-owner. Separate the reviewer from the second reviewer.
4. Replace run-based power sizing with task-based simulation. Publish P(Continue | real effect). Define the tests for N3 to N5.
5. Fix the conflict between milestone 0.5's box and the recruiting deadline, and re-box or rescope milestone 0.5.

## Resolution

On 2026-10-21 the owner decided the open question of B2: arm R may use only static analyzers and
one configuration file, with no code generation, within a 15-day cap. The minimum changes and most
Medium and Low items were then applied, in the commit that adds this report:

| Finding | Change |
|---|---|
| B1 | T3 is gone. A label check with seeded cases shows that both arms enforce set S. "Narrow" needs N0 and N1, so a costly language stops or pivots. |
| B2 | Arm R is static analyzers and one configuration file. Sets are labelled by seeded cases. U rules are graded by runtime tests and a blinded grader, which are outside arm R's means. Milestone 0.5 uses the same rule-writer process. When contract rules carry T2, the outcome is at most "narrow", and the report names a Go code generator as the next comparison. |
| H1 | A non-owner task writer writes every task and preregisters targets, at least 5 per tested set. Acceptance no longer requires the hidden-rule checks, so merged violations count. Spec §1.1 says that T2 tests H-B, and that H-A's review benefit is measured, not tested. |
| H2 | Tasks, not runs, are sized by a simulation that publishes outcome probabilities before sealing. N3 to N5 have defined tests. Bounds are one-sided 95%. |
| H3 | The owner never sees hidden material before the runs end. The toolchain and both applications are frozen first. A non-owner grader writes tests and grading. Reviewer and second reviewer differ, and the reviewer is blind to the Go arm. |
| H4 | Recruiting has its own 8-week deadline before milestone 0.5. Milestone 0.5's box is 6 weeks from the day the roles are filled. A committed budget for volunteer hours and tokens. |
| M1 | Milestone 0.5's sets, brief, and pivot condition are defined. |
| M2 | Each arm's documents are stated. Arm P is "rules written down, not enforced". |
| M3 | Unanswered runs count as failures. |
| M4 | The Go developer reviews the Vulcan version for parity. |
| M5 | M7 uses a holdout test set. Row 3 names the features it keeps. P2 is gone. |
| M6 | Time boxes are calendar weeks with one re-plan. The run adjudicator signs the exit gates of milestones 0.5, 3, and 4 and each overrun. Milestone 2's gate is a script plus a non-owner's signature; milestone 3's fault is sealed by the grader. |
| Lows | Section references, the outcome count, the decisions index, the Go-plain wording, S09 and S10, decision 4's latency, `pure` from a written design, and the person-checked criteria. |
