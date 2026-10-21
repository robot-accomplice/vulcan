# Audit 9: independent re-audit of the development plan

I audited commit `98033f9` and made no changes to the repository. Short names: `plan` = `docs/development-plan.md`, `d6` = `docs/decisions/0006-evidence-protocol.md`, `acc`, `conf`, `syn` = `docs/release-1/acceptance.md`, `conformance.md`, `syntax.md`, `sz` = `experiments/sizing/sizing.py`. I wrote every section up to "What holds up" before I read `experiments/audit-8.md`. I did not read audits 3 to 7.

*Provenance: written by a sixth freshly spawned agent, independent of the conversation that wrote
the plan and of the earlier auditors. Stored as returned (a condensed version was stored first,
when the session's usage limit was reached).*

**Method.** I copied `sizing.py` to my scratchpad and ran it there. Its `results.md` came out byte-identical to the committed file. I then ran variants of the model, also in the scratchpad: I split the result into its separate gates, added task × arm interaction, coupled V's cost to the shortcuts that R takes, added small real costs, and modelled the noise in gate 2's estimate of R's rate. Every number labelled "sim" below comes from those runs, with 500 to 600 simulations per cell. The Monte Carlo error is about ±0.02.

## Verdict

**No. The plan does not survive as written. One Blocker remains.**

The structure is sound in most places:
- The decision table is ordered.
- The tooling is frozen before the rules exist, and the owner is kept from the hidden material.
- The cheap paper-label gate comes before any compiler work.
- The sizing check reproduces exactly.

The problem is the cost side. N0 and N1 decide almost every result, because Q2 has power near 1 in this design. But on tempting tasks, N0 and N1 count R's rule-breaking shortcuts as successes and as fast work. So the more often Vulcan blocks a violation, the more it looks like it "costs too much". The sizing model assumes Vulcan costs nothing, so it never sees this.

## Blockers

### B1. N0 and N1 penalise the enforcement that Q2 rewards, and the sizing model leaves this out

**What the protocol says:**
- A tempting task "is easiest to meet by breaking a targeted rule" (`d6:137-138`). Three quarters of milestone 4's tasks are tempting (`d6:129-131`).
- Task success (X1) means only that "the acceptance tests and review pass" (`d6:201`).
- The reviewer sees nothing of the hidden material (`d6:113`). The checklist is written before the rules exist (`d6:177-179`).
- So in arm R, a change that breaks a U\* rule usually counts as a success, and usually as a fast one.
- In arm V, the checker blocks that easiest path. The agent must find the harder, compliant path, or fail.
- N0 (success) and N1 (time) are computed over all tasks, tempting tasks included (`d6:235-241`).

**Why this biases the result:**
- V's measured cost grows with R's violation rate on targeted tasks. Gate 2 requires that rate to be at least 0.2 per run (`d6:351-355`).
- A failed N0 or N1 sends the result to row 4, "Pivot", or to row 5 (`d6:299-300`). The plan describes that as the language costing "too much in task success or time" (`plan:36-38`). That attribution is wrong: part of the measured "cost" is R's defects counted as successes.

**The sizing does not see it:**
- Scenario L sets V's success and time "equal" to R's (`d6:271`, `sz:117`). Nothing in the model links V's cost to R's shortcuts (`sz:76-90`).

**What the effect could be (sim, 28 tasks × 10 runs, r = 0.4):**

| Assumption about blocked V runs | P(Continue \| L) |
|---|---|
| None (the committed model) | 0.85 |
| 1.2 times slower | 0.71 |
| 1.3 times slower, 5 points more failures | 0.53 |
| 1.5 times slower, 20 points more failures | 0.16 |

Review may catch some of R's shortcuts, which would shrink the effect. Nothing measures, bounds, or simulates this quantity, yet it can decide the outcome.

**Remediation (choose one, then re-run the pre-recruiting check):**
- **(a)** For N0 and N1, in every arm, count a run as a success only if it ends in an accepted change that breaks no hidden rule its task targets. The grader's tests and the blind grader decide this. Time runs to the first such change, and a failed run costs the full budget.
- **(b)** Compute N0 and N1 only on ordinary (untargeted) tasks, and size the number of those tasks for them.

In either case, add to the frozen simulation a scenario with coupled cost and a scenario with a small real cost.

## High

### H1. The sizing gate tests only one meaningful case, and that case assumes zero cost

- **Two criteria cannot fail.** The gate requires P(Continue \| Z) ≤ 0.05 and P(Continue \| L+cost) ≤ 0.2 (`d6:279-281`). L+cost sets V 15 points and 20% worse, which is already beyond the 12-point and 1.15 margins (`d6:272`). Both criteria are about 0 at every design (`results.md:32-43`). Only P(Continue \| L) binds.
- **Power comes from N0 and N1, not Q2.** Sim at 28 × 10, L: Q2 passes with probability 1.00, N0 with 0.95, N1 with 0.86. Q2 still reaches 0.95 at r = 0.1.
- **The parameters the gate re-measures barely matter.** Raising the CV from 1.0 to 2.0 moves P(Continue \| L) only from 0.82–0.84 to 0.81–0.82 (sim).
- **The parameters that matter are not measured.**
  - A real cost of half the margins (6 points, 7% time) gives P(Continue) of 0.21 (sim).
  - Task × arm interaction (sd 0.10 for success, 0.2 for log time) gives 0.73. At sd 0.15 and 0.3, it gives 0.61 (sim).
- **Consequence.** The gate certifies that the evaluation "can give an answer" only for a language that costs nothing. A new language that the model has never seen will not cost nothing.
- **Remediation.** Add a scenario "L with half-margin cost". Either require a stated power in that scenario, or publish it and accept it explicitly. Have the analyst measure task × arm interaction, at least as an R-against-P proxy, at the sizing gate.

### H2. The binding risk (the agent's cost in Vulcan) is not tested before the compiler is built

- The cheap gates of milestone 0.5 test only the violation side: gate 2 and Q1 (`d6:351-361`).
- The quantities that decide the outcome (B1, H1) are first observed in milestone 4. That comes after about 24 weeks of compiler and application work (`plan:91-93`).
- The milestone-3 fresh-agent check is one pass or fail on one task (`plan:191-195`). It has no threshold.
- **Remediation.** After milestone 2, run a small cost pilot, for example 6 to 8 ordinary tasks in V against R. Stop or re-plan when the point estimate of the success or time gap exceeds the margin.

### H3. N2 is a hidden, much stricter success bound

- X6 counts failing holdout tests "for every run", including failed runs. "A final change that does not build fails every holdout test" (`d6:206`).
- N2 allows only 0.1 more per run for V (`d6:242`).
- A run that fails or does not build adds k failures, where k is the size of the holdout set, which is not specified. Take k = 5: then a gap of 0.02 in the rate of failed or non-building runs fails N2. That is far inside N0's 12-point margin.
- V's final changes fail the build more often by design: a run that ends against a strict-mode error does not build.
- The model draws defects as Poisson(0.1), independent of success (`sz:92-95`).
- A failed N2 leads to row 3, "Narrow, as a hold", but no feature lies behind the failure. The re-run then counts row 3 as row 5 (`d6:298`). So a working language can be stopped by an artefact of the measurement.
- **Remediation.** Count X6 only in accepted changes, which is what "escaped" means. Or make it one value per run (any holdout failure, yes or no). Report the non-building runs separately (`d6:214` already counts them).

### H4. Gate 2's rate test is not sized, is noisy, and can be undefined

- In milestone 0.5, every tempting task targets S, and targets U\* only "where the rules allow" (`d6:135-136`). The number of tasks that target U\* is not fixed.
- Gate 2 still needs R's U\* rate on those tasks to be at least 0.2 (`d6:352-355`). That same estimate sets r for the sizing gate (`d6:274-275`).
- The pre-recruiting simulation does not model gate 2 (`sz:108-114`).
- Sim, 4 targeted tasks × 5 runs, CV 1:

| True rate | P(estimate ≥ 0.2) |
|---|---|
| 0.10 | 0.17 |
| 0.40 | 0.82 |

  So a true rate of 0.4 still fails the gate 18% of the time.
- If no task targets U\*, the rate is undefined. The cheapest falsification gate, "is a language needed?", then depends on an unbound choice of the task writer.
- **Remediation.** Fix a minimum number of tasks that target U\* in milestone 0.5 (for example 6). Define the case where none can be written. Use a lower bound, or simulate the gate's false-pass and false-fail rates and publish them.

### H5. Gaps in the generator stay in U\*, while gaps in the analyzers leave it

- A rule that arm R misses for reason (c), "not built within the cap", is a gap and leaves U (`d6:159-163`).
- No such rule exists for arm C. It has a 15-day cap, and only unsupported *forms* are listed (`d6:73-78`).
- Semantics that a generator could deliver but a quick one may lack stay in U\* and count for the language. Examples:
  - exact arithmetic in contracts (`conf:89`);
  - the second check point after deferred calls (`conf:104-107`);
  - port contracts that bind fakes (`conf:94`);
  - the invariant check after `json.Unmarshal` (`conf:96`).
- One configuration file can describe the architecture to an analyzer, so reason (a) will be rare. Generator shortfalls may then supply much of U\*. Gate 2 (8 rules) and Q2 would then pass for reasons that the "Continue" text denies: "rules that neither … nor a Go contract generator can enforce" (`plan:30-31`).
- In milestone 0.5 this compounds: the paper calls write each contract after seeing the rule (`d6:337-343`).
- **Remediation.** Apply the gap rule to arm C: a rule that C misses only because a semantic was not built within the cap leaves U\* and is reported. Or require the generator to pass the contract cases of `conf` (C01 to D12) that do not need the compiler.

## Medium

- **M1. The simulation does not apply the protocol's full decision rule.**
  - It leaves out the dual-reviewer conditions for N0 and N3, and N1 without reviewer minutes (`d6:236-245`).
  - It leaves out N4, N5, and the four-rule minimum (`d6:224`).
  - Its sign-flip test samples 2,000 patterns; the protocol requires an exact test (`sz:29`, `d6:220`).
  - The docstring calls the bounds "an approximation of the inverted test", but section 8 now defines them as t-bounds (`sz:18-19`, `d6:225-227`).
  - `results.md:3` says "400 simulations per cell", but the adopted table used 1,000 (`sz:136`).
  - The chosen cell, 0.82 at r = 0.2, sits only 1.5 standard errors above 0.8.
  - Each omission makes P(Continue) look better than it is.
- **M2. The boxes do not fit the stated loads.**
  - Milestone 0.5: at 2.5 days a week, the Go developer's work before the freeze (two 15-day caps, the map, the documents, the review) takes about 14 weeks. Rules, grading (10 days), arm C's contracts, tasks, sealing, runs, analysis, and re-commitment follow it in series (`plan:90`, `d6:105-112`, `d6:177-182`). The 16-week box will be overrun.
  - Milestone 4: the reviewer needs about 140 hours inside a 3-week operator window (`d6:112-113`).
  - The grader's 10 days must cover 60 or more seeded cases per arm, 56 reference solutions, and the hidden and holdout tests (`d6:108`, `:138-140`, `:148-151`).
  - The Go developer's 80 days include building Go Stockroom at no committed pace (`plan:185-188`).
- **M3. Six people cannot meet the separations as described.**
  - The blind grader cannot be the grader (`d6:109`). The run adjudicator handles exclusions from the run logs (`d6:111`, `:195`), which the blind grader must not see. So with six people the blind grader is the Go developer, who built arm R.
  - In milestone 4, a diff shows `.vulcan` or `.go` files, so "does not see arm names" cannot hold.
  - The toolchain-bug procedure assumes that the fixer has seen no hidden material (`d6:190-193`). But the Go developer made the paper calls in milestone 0.5 (`d6:338`).
- **M4. Row 5 labels a clear negative "not judged".** Suppose Q2 passes, N0 or N1 fails, and Q1 had failed in milestone 0.5. Row 4 does not apply. In row 5 the half-bound cannot fire, so the outcome is "stop, not judged" (`d6:299-300`), although the language was shown to work and to cost too much. The same happens when U\* holds fewer than 4 rules (`d6:172-173`).
- **M5. The model changes over time.** All results bind to one model version (`d6:91`, `:375`). But milestone 4 starts at least about a year after milestone 0.5. Row 4 and the sizing gate's r use milestone-0.5 results, which may come from a retired model. No substitution rule exists.
- **M6. Some reviewer bias remains.**
  - In milestone 4 the reviewer cannot be blind to the arm.
  - 70% of verdicts are never re-checked.
  - The protocol defines a reversed rejection but not a reversed acceptance (`d6:237-239`).
  - N0 is the binding bound, and a 6-point shift halves its pass rate (sim).
- **M7. The milestone-3 gate has gaps.**
  - It says nothing about a fresh agent that fails. A retry needs a new sealed fault (`plan:191-195`).
  - D5 on the Go version is checked by its own author, the Go developer (`acc:104-106` against `plan:186-187`).

## Low

- **L1.** "The margins of P's" for R against P are never defined (`d6:357`, `:299`).
- **L2.** Section 8 says every bound is a t-bound, but N4 uses a permutation bound over runs (`d6:225`, `:246-248`).
- **L3.** "The smallest design" has no order: 28 × 8 is 224 runs per arm, against 240 for 24 × 10 (`d6:279-280`).
- **L4.** The estimators for the measured CV and time spread at the sizing gate are not defined (`d6:277-279`).
- **L5.** Sealed material is released only "after milestone 4" (`d6:315-316`). The project can end earlier.
- **L6.** A run whose agent asks a question that the FAQ does not answer ends as a failure (`d6:185-186`). V's agents are more likely to ask about the language. This is a small bias against V.
- **L7.** The plan's "Stop" outcome (`plan:39-40`) leaves out gate 1's stop when R costs too much against P (`d6:357-358`, `plan:133-134`).
- **L8.** Decision 4 still says `keel check`, `keel fmt`, and `.keel` (`0004:111`, `:137-138`).
- **L9.** The traceability table lists audit 8 before audit 7 (`plan:242-243`).
- **L10.** Spec §7.3 says the compiler can remove proved checks (`spec.md:1503-1504`), but `syn:79-80` defers that removal.
- **L11.** The time between milestones has no box. Only milestones do.

## What holds up

- The decision table is ordered and complete, because row 5 catches every other result.
- The half-bound is defined in the correct direction.
- P(Continue \| Z) is about 0.
- The exact sign-flip test with informative tasks is sound.
- The binding targeting rule in milestone 4 is sound.
- The freeze comes before the rules, and the owner is kept from hidden material, with outside timestamps.
- Gaps in arm R leave U\*.
- The recruiting deadline, and boxes signed by a non-owner.
- Gate 2 comes before any compiler work.
- `sizing.py` reproduces `results.md` byte for byte, and its search over the old margins honestly showed that they could not be powered.
- All spec section references (§1.1, §5.4d, §7.3, §7.4, §11.1) and every diagnostic code that `syn` and `conf` cite exist in the spec.
- The acceptance counts are correct: 6+7+5+5+5+4 = 32 criteria, and 28 without V1 to V4.

## Status of audit 8's findings

- **B1 (the sizing does not bind the design): resolved.**
  - Every tempting task now targets the tested set (`d6:133-137`).
  - The check ran before recruiting (`results.md`).
  - Milestone 0.5's design is fixed at 16 × 5 (`d6:261-262`).
  - r is the lower of 0.4 and the measured rate (`d6:274-275`).
  - The margins changed to 12 points and 1.15 before any data, which is honest.
  - **New problems:** the model assumes zero cost and no coupling (B1, H1 above), and it does not model gate 2 (H4).
- **H1 (gate 2 nearly passes by construction): partly resolved.**
  - Gate 2 now needs 8 rules and a rate of at least 0.2 (`d6:351-355`).
  - Contract calls name the reaching test, and arm C uses the same test (`d6:339-341`).
  - The objective text now matches the gate order (`plan:12-14`).
  - **New problems:** the rate test is unsized and noisy (H4), and arm C has no gap rule (H5).
- **H2 (non-building workspaces): resolved as asked.** The blind grader judges X2 from the diff (`d6:202`). X6 counts such a change as failing every holdout test (`d6:206`). The count is reported per arm (`d6:214`).
  - **New problem:** the X6 rule turns N2 into a hidden, stricter success bound (H3). Blinding the blind grader to the arm is not possible in milestone 4 (M3).
- **H3 (an unaudited reviewer): mostly resolved.** The second reviewer now samples accepted and rejected changes and X8 calls. N0, N1, and N3 must hold under both judgments and without reviewer minutes, and a reversed rejection is defined (`d6:114`, `:236-245`).
  - **Residual:** a reversed acceptance is undefined, and 70% of verdicts are never re-checked (M6).
- **H4 (milestone 0.5's box): partly resolved.** The owner now builds the service, and the box is 16 weeks with the Go developer at least 2.5 days a week (`plan:90`, `d6:105`).
  - **Residual:** at that rate, the work before the freeze alone takes about 14 weeks, so the box still does not fit its critical path (M2).

## Minimum changes

To remove the Blocker:

1. **Stop counting rule-breaking shortcuts as success.** Do one of these:
   - Redefine success and time for N0 and N1, in every arm, as an accepted change that keeps the hidden rules its task targets.
   - Or compute N0 and N1 only on ordinary tasks, sized for them.
2. **Re-run the pre-recruiting check.** Add a coupled-cost scenario and a half-margin-cost scenario, and fix the design or margins from the result.

Strongly recommended:
- Count X6 only in accepted changes (H3).
- Apply the gap rule to arm C (H5).
- Fix a minimum number of U\*-targeting tasks in milestone 0.5, and size gate 2 (H4).
- Add a cost pilot after milestone 2 (H2).
- Re-estimate the boxes and loads (M2).

As written, the plan does not survive. Its governance, its isolation, and its ordered decision table are solid, and its sizing reproduces. But "Continue" depends on N0 and N1, which on tempting tasks measure V's enforcement as V's cost. The sizing hides this by assuming that V costs nothing. The result can therefore be biased decisively toward "Pivot" or "Stop" while it reports that the language costs too much. The first two changes above fix this without touching the given design choices.

## Resolution

Applied on 2026-10-21, in the commit that adds this resolution, as round 9 of the owner's audit and
remediation cycle:

| Finding | Change |
|---|---|
| B1 | N0 and N1 use **clean success** (X1c): an accepted change that keeps every hidden rule its task targets, in every arm, with working time up to the first clean change. The sizing check (`experiments/sizing/`) ran again with clean success, a coupled-cost scenario (tempted Vulcan runs 1.3 times slower, 5 points more failures), a task × arm interaction scenario, and net-cost scenarios. At 28 × 10, P(Continue) is about 0.86 to 0.89 under L, 0.84 to 0.88 under coupled cost, and 0.84 to 0.86 under interaction; 0 under Z and under a 15-point net cost. |
| H1 | The sizing gate now also requires L+coupled and L+inter to reach 0.8. L+half (a net cost of half the margins, about 0.2) is published and accepted as a stated limit. |
| H2 | A cost pilot at the end of milestone 2: 8 ordinary tasks, Vulcan against the analyzers; two failures of the margins end the project before the application is built. |
| H3 | X6 counts holdout failures per accepted change; N2 compares tasks where both arms have an accepted change. |
| H4 | In milestone 0.5, at least 8 of the 12 tempting tasks also target U\*; gate 2's false-pass and false-fail rates are published (9% and 9% at true rates of 0.1 and 0.4). |
| H5 | The coverage map lists each release-1 contract semantic for arm C; a rule arm C misses only for lack of time is a generator gap and leaves U\*; the generator must pass the conformance cases of the semantics it lists as built. |
| M1 | The sizing check uses an exact sign-flip test up to 16 informative tasks, states what it does not model, and the gate uses at least 2,000 simulations per chosen cell. |
| M2 | Milestone 0.5's box is 24 weeks; the grader has 15 days per milestone; milestone 4 has two reviewers and a 4-week operator window. |
| M3 | Seven people besides the owner; the blind grader is not the Go developer, the grader, or the run adjudicator, and sees the language but not the Go arm. |
| M4 | A new row: Q2 passes and N0 or N1 fails on its bound gives "stop: the language costs too much". Too few U\* rules in milestone 4 means no rule needs the language. |
| M5 | If the model changes before milestone 4, arm P runs there too and Q1 is tested again. |
| M6 | A reversed acceptance is defined; more than 20% disagreement between the reviewers makes N0 inconclusive. |
| M7 | One retry of milestone 3's fresh-agent check with a new sealed fault; the grader checks D5 on the Go version. |
| L1 to L11 | Margins of P's defined; N4's exception stated; design order by total runs; estimators defined; release when the project ends; the FAQ may point to the primer; the plan's "Stop" wording; spec §7.3 check removal not in release 1; the traceability order; time between milestones counts. L8 (old names in decision 4) is covered by that decision's note on old names. |
