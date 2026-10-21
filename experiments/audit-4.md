# Audit 4: independent audit of the development plan

Date: 2026-10-21. Commit audited: `88e9dbd`. Read only. Paths are relative to `C:\code\vulcan`.

*Provenance: written by a freshly spawned agent with no access to the conversation that produced
the plan and decisions 1 to 6. It wrote its findings before it read `experiments/audit-3.md`. It is
independent of that conversation, not of the model family. Stored as returned; the five concrete
claims that were spot-checked (S08, the K0009 sentence of the plan, `README.md:100`, the spec word
count, and the escape clause of decision 6) all held.*

## Verdict

**The plan does not survive as written.** Its structure is good: it preregisters the evaluation, it has a cheap stop before the compiler, it counts inconclusive results as fails, and it freezes the compiler during the runs. But the evaluation still cannot reliably answer its own question. One outcome ("continue the language") can be produced by how the owner scopes the linter arm. The primary statistics break down on zero counts and on 4 to 6 task clusters. The independence safeguards have an escape clause that lets the owner fill every role. And the cheap milestone 0.5 tests "do the rules help", not "does the language beat the rules", so 24 weeks of compiler and application work are not gated on the question that decides "continue" versus "pivot". Four changes fix the blockers (see Minimum changes).

## Blockers

### B1. The linter arm is scoped so that "continue the language" can pass by construction

- The Go-rules arm gets only "the checks of the guarantee matrix ... that need no new syntax (G1 to G7)", plus contracts "where a generator can check them" (`docs/decisions/0006-evidence-protocol.md:55-59`; `docs/development-plan.md:86-87`).
- A hidden rule is labelled *enforced* when "a Vulcan diagnostic covers it" (`0006:91-93`). The primary outcomes use only that label (`0006:147-148`).
- So the primary rule set is chosen by Vulcan's coverage. Any enforced rule outside G1 to G7 favors Vulcan automatically: G8 port contracts, G9 one gateway per external (K0207), K0206, contracts, and the strict-mode checks if the analyzers implement only ordinary mode. Many of these need no syntax in Go. A `gateway for` list in a config file plus an analyzer gives G9. The "no new syntax" boundary is the owner's choice, not a property of the problem.
- The reverse case is invisible. A rule that the analyzers cover and Vulcan does not is labelled "not enforced" and drops out of the primary outcome.
- Result: Vulcan "passes against Go-rules" (`0006:184`) whenever the Go-rules agent breaks at least one rule outside the analyzers' scope. That measures coverage, not "a language versus a linter".
- **Remediation:** (a) The Go-rules arm implements every rule that can be expressed through Go code, comments, or a config file: G1 to G9, strict mode, and port and struct contracts as checked comments. The Go developer signs that this coverage is complete. (b) Label each hidden rule for each arm separately. Compute P1 and P2 on the rules that both arms enforce, and report coverage as a separate outcome. (c) "Continue" then needs a pass on the shared rule set, plus a written argument that each Vulcan-only rule cannot be expressed in Go.

### B2. The primary statistics cannot carry the decision

- **Interval against a point estimate.** P1 passes when "the upper bound of its interval is below that arm's rate" (`0006:152-154`). The comparator's sampling error is ignored. When the enforcing arm has zero violations at review (likely, because a change that does not build cannot reach review), its bootstrap interval is [0, 0]. P1 then passes after a single violation in the comparator. In milestone 0.5 that is one violation in 12 tempting Go-plain runs (4 tasks × 3 runs, `0006:215`).
- **P2 is near 100% by construction** for any arm whose checker blocks the build, if review requires a passing build (`0006:155-156`). It is undefined (0/0) when an arm shows no violation at all. Nothing says how 0/0 is scored. In milestone 0.5, 0/0 could either stop the project or be argued away.
- **Too few clusters.** The bootstrap "resampled by task" (`0006:146-147`) has 6 clusters in milestone 4 and 4 in milestone 0.5. A bootstrap over 4 clusters has a handful of distinct resamples, and its coverage is poor. More runs per task (`0006:107-109`) do not add clusters, so the "80% power" target may be unreachable at the 10-run cap. The plan does not say what happens then.
- **No minimum count.** Nothing sets a minimum number of enforced hidden rules or violation events. One or two rules could decide the project.
- **Remediation:** Compare the arms with an interval on the paired per-task difference or ratio, so that both arms' uncertainty counts. Add a pre-registered minimum: for example, at least 8 enforced rules and at least N violation events across arms, below which the result is "inconclusive" and therefore a fail. Define 0/0 for P2. Use an exact or permutation test on task clusters instead of a bootstrap over 4 to 6 clusters. Raise milestone 0.5 to at least 6 tempting tasks.

### B3. The independence safeguards are optional, and key judgments have no assigned judge

- **Escape clause.** "If they cannot be found, the report says so on its first page, and its decision is marked as not independently judged" (`0006:239-241`). The mark changes nothing in the decision rule. Operator and reviewer can fall back to the owner (`0006:78-79`). The second reviewer, who feeds gate N2, has no independence rule at all (`0006:80`). Audit 3 itself says B3 is "answered on paper only" (`experiments/audit-3.md:182-183`).
- **The rule writer has no fallback.** No text says what happens without one. Yet the plan names this person as the first step (`development-plan.md:197-199`).
- **Unassigned judgments.** The owner, by default, makes these calls:
  - writing the 12 task briefs, including choosing which changes are "tempting";
  - labelling hidden rules enforced or not;
  - writing the hidden-rule check scripts and the hidden HTTP tests;
  - writing the ≤4,000-token rule document for the Go arms (`0006:58-59`);
  - deciding infrastructure exclusions (`0006:190-192`);
  - deciding whether a compiler bug "blocks" a run (the 10% rule, `0006:113-116`);
  - judging "M14 shows no cost" (`0006:188`) and "most workarounds come from one feature" (`0006:186`), neither of which has a threshold;
  - judging every milestone exit gate.
- **Operator bias.** The operator answers agents' questions live (`0006:78`, `0006:111-112`). If the owner is the operator, the language's author unblocks the Vulcan agent. The fallback covers only M8 judgments, not this.
- **Remediation:** Make the Go developer and the rule writer mandatory non-owners. If they are not found, the evaluation cannot produce "continue" or "pivot". Add roles, each a non-owner, for: task author (or task reviewer with veto), rule labeller, and run adjudicator (exclusions and blocking bugs). Forbid the owner as operator in the Vulcan arm, or blind the operator's answers by having a fixed FAQ committed before the runs. Replace "M14 shows no cost" and "most workarounds" with numeric thresholds.

### B4. The cheap gate does not test the question that decides the expensive work

- The objective is to answer "at the lowest cost" (`development-plan.md:9-12`). Milestone 0.5 compares Go-rules with Go-plain only (`0006:213-220`). It can stop the project if the rules do not help. It cannot show that the language can beat the rules.
- Under the decision rule, "continue" requires Vulcan to beat Go-rules (`0006:184`). Milestone 0.5 already produces the evidence that predicts this: which enforced violations the analyzers missed. If they missed none, the best Vulcan can reach is "pivot", and the compiler (milestones 1 to 3, 24 weeks of box) is not needed to reach it.
- **Remediation:** Add a second exit condition to milestone 0.5. List each violation in either arm that the G1 to G7 analyzers did not stop, and classify it as "a Vulcan diagnostic would stop it, and no Go analyzer or config could". If that set is empty or below a pre-registered size, the plan pivots to the rules (as a Go tool) without building the compiler. Seal this classification rule with the milestone-0.5 files.

## High

### H1. When the Go-rules agent sees the analyzers is undefined, and it decides P1 and P2

- The analyzers are "run in CI" (`0006:57`), and Go-plain's linters are "in CI" too (`0006:61`). The Vulcan agent runs `vulcan check` locally on every build.
- If CI runs only when the agent submits for review, Go-rules violations reach review by definition (M2, `0006:126`), and Go-rules loses P1 to Vulcan and fails P1 against Go-plain. That could stop the project at milestone 0.5 because of tool placement, not the rules.
- This contradicts "the same everywhere else: ... the tools" (`0006:62-63`).
- **Remediation:** The analyzers run through the same local command the agent uses to build and test (for example a `make check` that the brief names), with the same invocation cost as `vulcan check`. State that review starts only after the arm's checks pass, or record when each arm's checker ran.

### H2. The rule writer is primed with Vulcan's architecture

- The rule writer writes hidden rules "from its acceptance criteria alone" (`0006:77`).
- `docs/release-1/acceptance.md` describes the shape in Vulcan terms: "Entry leaf `Web`", "Gateway leaf `Store` (`gateway for Postgres`)", "The domain leaves need ports; only `Store` opens the database" (`acceptance.md:18-30`). It also includes the strict-mode criteria V1 to V4 (`acceptance.md:91-98`).
- A rule writer who reads this will restate Vulcan's rules. That inflates the "enforced" share and repeats the bias that amendment A2 was meant to remove.
- "Rules that only Vulcan's compiler could have suggested are not allowed" (`0006:93-94`) has no judge.
- **Remediation:** Give the rule writer a functional brief only: the F, D, S, and R criteria, and a neutral package diagram with no Vulcan terms. Seal the rules before anyone labels them. A non-owner applies the "only Vulcan could suggest" filter.

### H3. The premise's real claim is ungated, and task success is not an outcome

- The premise says structure makes review and validation shorter (`docs/spec.md:23-26`). It promises "a smaller and more reliable review" (`spec.md:46-47`).
- Review time M6 has no threshold (`0006:169-171`).
- The rate of accepted changes (M1) has no threshold either. A Vulcan arm that completes 50% of tasks can still "continue".
- N1 takes the median over accepted changes (M5 ends at "an accepted change", `0006:129`). P1 is "per accepted change" (`0006:152`). Both drop failed runs, which is survivorship bias.
- "Runs that fail ... count as failures of their arm" (`0006:191-192`) is attached to no gate.
- **Remediation:** Add a non-inferiority gate on task success (M1 rate). Define M5 for failed runs as censored at the budget. Either gate M6, or state in §1.1 that review time is not tested and remove the "shorter review" claim from the premise and the README (`README.md:24-26`).

### H4. The decision rule is not exhaustive, and its rows overlap

- **No row:** Vulcan passes against Go-plain, Go-rules does not pass against Go-plain, and Vulcan does not pass against Go-rules (`0006:182-188`).
- **No row:** Vulcan passes against Go-rules but not against Go-plain.
- **Overlap:** "Continue" (N1 to N5 pass) and "Narrow" ("or most workarounds come from one feature") both apply when all bounds pass and workarounds cluster.
- **Undefined terms:** "Pivot" does not require Go-rules to meet any non-inferiority bound against Go-plain. "Narrowed to what M14 and M7 support" (`0006:188`) is undefined.
- **A fifth outcome:** the plan says it "can end in four ways" (`development-plan.md:24`), but a second time-box overrun also stops the project (`development-plan.md:53-54`).
- **Remediation:** Write the rule as a complete truth table over (Vulcan vs Go-rules, Vulcan vs Go-plain, Go-rules vs Go-plain, all NI bounds pass) with one decision per cell. Give precedence where rows overlap. Add "stopped by schedule" as an explicit outcome with its own wording in the report.

### H5. The non-inferiority bounds are likely unreachable, so "narrow" is the default positive result

- N1 compares wall-clock time with a 10% margin. That time includes human review and the waiting time for the reviewer (`0006:129`, `0006:160-161`). With 12 tasks of very different sizes, and noise from reviewer queues, the interval will rarely fit inside 1.10.
- N2 is a ratio of rare events (`0006:162-164`). When the Go rate is 0, Vulcan must also be 0.
- An inconclusive bound counts as a fail of that bound (`0006:173-176`). The "Narrow" row then fires (`0006:186`).
- So the claim that "an unclear result never continues the project by default" (`0006:176`) is false. An unclear NI result continues the project as "narrow". With B1, the realistic outcomes are fixed largely by design.
- **Remediation:** Measure N1 as agent active time plus measured review minutes, not queue time. Use a difference margin on N2 (for example +1 defect per 10 changes) instead of a ratio. Run a pre-registered power check of N1 and N2 on the pilot. Make "narrow" require at least one feature-level pass, not merely a failed bound.

### H6. Hidden materials are stored where agents can read them

- The sealed tasks, hidden rules, hidden tests, and grading go in `experiments/m0.5/` and `experiments/m4/` in this repository (`0006:224-227`; `development-plan.md:146`).
- The Vulcan arm has "the full spec available" (`0006:54`). The milestone-3 agent gets "only the repository" (`development-plan.md:137-139`).
- If an agent's workspace or tools reach this repository, the hidden rules leak. Test 3's hidden rules are already public in `experiments/test3/hidden/`.
- Sealing by hashes in an owner-controlled git history proves little unless the hashes are published elsewhere.
- **Remediation:** Keep the sealed contents in a separate private store until the runs end. Commit only the hashes, and timestamp them externally (a signed tag pushed to a third party, or a public post). Give each agent a workspace that contains only its arm's code and its documents.

### H7. Feasibility: the time boxes cover the owner, not the bottleneck

- The boxes total 34 owner-weeks (`development-plan.md:56-62`).
- Milestone 2 alone is due in 10 weeks (`development-plan.md:113-124`). It covers: contracts with exact arithmetic and `old`, second check points and the closure lowering, reach analysis over all dependency modules (`spec.md` §5.4c; decision 4 open point), strict mode, guide, context, query, and audit, fuzzing, differential tests, and CI on four hosts (`decisions/0005-names-and-reproducibility.md`, item 6.2). That is not credible for one person without a record of comparable velocity.
- Milestone 4: 3 arms × (12 tasks + 6 follow-ons) × 5 to 10 runs is 270 to 540 runs, plus pilots and extensions. The reviewer reviews and times every accepted change (`0006:79`, `0006:140`), and the operator must be live during runs. At 15 minutes a review, that is 70 to 135 hours of a volunteer's reviewing. No milestone boxes this, and no budget covers tokens or model access.
- **What will most likely go wrong first:** recruiting the unpaid rule writer and Go developer. The plan already makes this step one (`development-plan.md:197-199`). Next is milestone 2's reach analysis through `net/http` and a Postgres driver (`decisions/0002-architecture-enforcement.md:200-202`).
- **Remediation:** Add a calendar deadline for recruiting, after which the plan reports "not independently judged" and stops before milestone 1 (see B3). Box the volunteer hours and the token budget. Size milestone 4 from the pilot before committing. Split milestone 2 into a contracts box and an architecture box.

## Medium

### M1. Latency and `pure` depend on an application that does not exist yet

Milestone 2 measures latency "on the milestone-3 application" (`0006:196-206`; `decisions/0004-compiler-structure.md:105-108`; `development-plan.md:123-124`). Release 1 adds `pure` "only if the milestone-3 application needs them" (`release-1/syntax.md:57`; `development-plan.md:115-116`). Decision 4 asks milestone 1 to choose the reach analysis on "the dependencies of the milestone-3 application" (`0004:134-136`). Stockroom is built in milestone 3. **Remediation:** Measure on a synthetic application of fixed size in milestone 2, and again on Stockroom at the milestone-3 exit. Decide `pure` from a written Stockroom design at milestone 1.

### M2. Several exit gates cannot be tested, and none names a judge

- Milestone 2: "known holes are either rejected or accurately exposed" and "No missing compiler feature is replaced with manual checking" (`development-plan.md:126-127`).
- Milestone 3: the fresh agent "diagnoses one injected fault". Nothing says who injects it, who judges the diagnosis, or what counts as success (`development-plan.md:136-139`).
- Milestone 0 is marked "met" by the owner (`development-plan.md:78-79`).
- **Remediation:** For each gate, state a script or a checklist, and name a non-owner who signs it.

### M3. Milestone 3's "all 32 criteria" cannot hold for the Go version

The criteria total 32 (F1-F6, D1-D7, S1-S5, R1-R5, T1-T5, V1-V4). V1 to V4 are "Vulcan arm only" (`acceptance.md:91-98`), so the Go version can meet 28 (`development-plan.md:136`). The acceptance file also says person-checked criteria are "none today" (`acceptance.md:106-107`), but V1 (reasons in a README), V2 ("each K0525 warning is reviewed"), and D5 ("each guard is written so that it cannot overflow", `acceptance.md:57`) need a person. T3 has no threshold (`acceptance.md:87`), so it passes by running. **Remediation:** Say "32 in the Vulcan version, 28 in the Go version". List the person-checked criteria and their checker.

### M4. Conformance contradicts itself and the syntax document

- S08 expects K0009 for `vulcan fmt` (`release-1/conformance.md:39`). `syntax.md:101` says a deferred command "prints that it is not in release 1, and exits 2".
- The plan says LLVM, self-hosting, and editor integrations are "rejected with K0009" (`development-plan.md:45-46`). These are not source constructs, and `syntax.md:101-102` treats commands and servers differently.
- D12 expects the output to name `pkg.F.func1` (`conformance.md:114`). The differential rule requires the same output as plain Go on every run where no contract fails (`conformance.md:154-158`). D12 must fail that rule.
- Decision 1's open point on design rule 4 is still open (`decisions/0001-contract-timing.md:193`). Yet milestone 0 is "done", and its gate says no decision weakens a promise (`development-plan.md:78-79`).
- **Remediation:** Fix S08. Correct the plan sentence. Exempt D12 from the differential rule by name. Close decision 1's open point with a written exception to design rule 4.

### M5. Milestone 0.5 leaves its own terms undefined

- "Enforced" means "a Vulcan diagnostic covers it" (`0006:92`), but no compiler exists in milestone 0.5. The section does not say whether a label comes from reading the spec. Decision 6 itself warns "a model checker is not a compiler" (`0006:30-31`).
- Section 9 does not say whether the inconclusive rule and its one extension apply (`0006:211-220`).
- Go-plain's "dependency linter (for example `depguard`)" (`0006:60-61`) does not say who configures it, or whether its config encodes the leaf import rules. If it does, Go-plain already enforces G1.
- Why milestone 0.5 builds contract comments (`development-plan.md:86-87`) when P1 and P2 measure only architecture violations is unexplained.
- **Remediation:** Label by analyzer coverage in milestone 0.5. Apply section 6's inconclusive rule. Seal the depguard config with an explicit statement of what it covers. Drop contracts from milestone 0.5.

### M6. The arms get unequal information

The Vulcan agent gets the primer and the full spec (`0006:53-54`), which is about 25,400 words. The Go arms get one rule document of at most 4,000 tokens (`0006:58-59`), whose author is unnamed. The checked-comment contract generator for Go-rules (`0006:57-58`) is scheduled in no milestone. **Remediation:** Give each arm documents of comparable size, written or checked by the Go developer. Schedule the generator in milestone 3, or drop contracts from Go-rules and say so in the report.

### M7. M14 is observational, small, and decides a row of the rule

M14 compares follow-on runs whose tempting change had a violation with runs that had none (`0006:138`). The Vulcan arm will rarely merge a violation, so the comparison comes mostly from the Go arms. It is confounded by run quality (agents that violate may also be weaker on that run), and there are only 6 follow-on tasks. Yet it can "narrow" any result (`0006:188`) with no threshold. **Remediation:** Make the follow-on an experiment. Start some follow-on runs from a seeded violating base and some from a clean base, assigned at random, and set a threshold.

### M8. The frozen-compiler rule breaks the design it protects

After a blocking bug, "every Vulcan run is repeated" but the Go runs are not (`0006:114-116`). That breaks the interleaved random order (`0006:110`). If the model service changed in the meantime, the arms run on different models. There is no cap on repeats. **Remediation:** Repeat the full interleaved block for all arms, record the model build per run, and cap the evaluation at one re-tag.

### M9. Contracts cannot earn their place

The primary outcomes cover architecture only (`0006:147-156`). Contracts can only cost time (N1), or pass N2 as "a finding beyond the hypotheses" (`0006:163-164`). "Narrow" keeps "the features that carry the passing outcomes" (`0006:186`), so contracts lose by construction under "narrow". **Remediation:** Either add a contract outcome (for example, hidden logic rules that a contract could catch, measured like P1), or state that release 1 does not test contracts and that "narrow" cannot drop them on this evidence.

### M10. Baseline ownership is soft

The Go developer "builds the Go version, or reviews it and signs" (`development-plan.md:133-134`; `0006:76`). In practice, the owner can write both baselines. **Remediation:** The Go developer builds the Go version, or the review includes a timed, written list of changes the developer would make, and those changes are applied before sealing.

## Low

- **L1. Stale names and text.**
  - Decisions 1 to 5 still use "keel" (12, 19, 18, 17, and 12 matching lines). Decision 4 rule 4 says `keel check` (`0004:105-108`). Decision 2 says `keel audit` throughout.
  - Decision 6 "Current state" still lists "inconclusive" as an outcome (`0006:20-21`).
  - `README.md:100` calls the plan a "Draft plan".
  - `docs/status.md:16` says the spec is "about 21,000 words". It is 25,408.
- **L2. H-C mapping.** `spec.md:47-51` says N1 (total time) tests H-C. H-C is about reviewer reading and logic defects (`spec.md:43-44`). Its reading part maps to M6, which has no gate.
- **L3. Suppressions that do not exist.** M9 counts "K0122 to K0125 or K0210 to K0211 suppressions" (`0006:133`). The spec has no suppression mechanism for these errors (no occurrence of "suppress" in `spec.md`).
- **L4. The milestone-5 entry depends on owner-written tasks.** The entry test needs a milestone-4 task that one container cannot meet (`development-plan.md:153-155`). The tasks are sealed in a one-container scope (`acceptance.md:109-113`), and their author is unassigned.
- **L5. Generalization.** All results bind to one model version (`0006:62`). A stop at milestone 0.5 may reflect that model's skill with a written rule document, not the value of enforcement. The report should say so.
- **L6. Status before audit.** The plan already says "accepted" (`development-plan.md:3`; `status.md:76`), while the owner treats it as accepted only if it survives this audit. Mark it "proposed" until then.

## What holds up

- Preregistration with sealed files, a committed seed, a scripted analysis, and a list of every excluded run (`0006:97-101`, `0006:110`, `0006:224-231`).
- Three arms and a named "pivot to the rules" outcome. The framing is right, even though B1 breaks its execution.
- Inconclusive primary outcomes count as fails (`0006:173-175`).
- The idea of a cheap, compiler-free test of the rules first (milestone 0.5), and time boxes with a re-plan trigger.
- Release 1 is a real scope cut. It rejects every unsupported construct with K0009 at a position instead of ignoring it (`syntax.md:11-18`).
- The conformance format pins positions and exact diagnostic sets (`conformance.md:21-25`). Every code that `syntax.md` and `conformance.md` use exists in Appendix E. I checked them all.
- Decision 4's staged compiler that reuses `go/types` is a sensible cost cut, backed by a prototype.
- The acceptance criteria are concrete and trace to real defects of test 3.
- The spec now states its limits honestly (§5.4d matrix, §7.3 limits, §11.3 "a goal, with no current test").

## Comparison with audit 3

**I agree with:**
- B1 (linter versus language) and B3 (judges with a stake). Both remain partly open. B3 is "answered on paper only", as audit 3 itself says.
- H1 (cost before the first stop). Milestone 0.5 answers it for "do the rules help".
- H3 (a frozen compiler).
- L1 (the closure changes the caller name).
- M1, M3, and M4 were fixed as described.

**Overstated or wrong:**
- "With A1 to A6, this audit finds no blocker" (`audit-3.md:153`) is overstated.
- A1 was applied in a narrowed form ("no new syntax, G1 to G7"). That form recreates B1 through coverage scope (my B1).
- A2 changed who writes the rules but not how they are labelled. The labels still follow Vulcan's coverage, so P2 still passes largely by construction (my B1, B2).
- A5 states that an inconclusive bound "leads to narrow". That makes "narrow" the default positive outcome, which contradicts decision 6's claim that an unclear result never continues the project (my H5).
- A11 pinned the caller-name limit with D12, but D12 now contradicts the differential-test rule, and design rule 4 is still breached (my M4).

**What audit 3 missed:**
- Tool placement, CI versus local (H1).
- The rule writer primed by `acceptance.md` (H2).
- Task success and review time not gated, and survivorship in N1 and P1 (H3).
- Gaps and overlaps in the decision rule, and the fifth outcome (H4).
- Interval against a point estimate, 0/0, and 4 to 6 bootstrap clusters (B2).
- Leakage of the sealed materials through the repository (H6).
- Volunteer workload and budget (H7).
- Latency and `pure` tied to an application that does not exist yet (M1).
- The 32-versus-28 criteria count and the person-checked criteria (M3).
- The S08 and `vulcan fmt` contradiction (M4).
- The "need no new syntax" gate not testing language versus linter (B4).
- The escape clause that lets the decision issue without independent roles (B3).

## Minimum changes

For the plan to survive, these must change before milestone 0.5 seals anything:

1. **B1.** Widen the Go-rules arm to every rule that Go code, comments, or config can express, signed by the Go developer. Label rules per arm, and compute P1 and P2 on the rules both arms enforce.
2. **B2.** Replace "upper bound below the other arm's point rate" with an interval on the paired difference. Use a cluster-appropriate test. Set minimum counts of enforced rules and violation events, define 0/0, and raise milestone 0.5 to at least 6 tempting tasks.
3. **B3.** Make a non-owner Go developer, rule writer, task reviewer, and run adjudicator mandatory. Without them, the evaluation cannot yield "continue" or "pivot". Bar the owner as operator for the Vulcan arm.
4. **B4.** Add a milestone-0.5 gate on the Vulcan-only remainder. If no stopped violation needs the language, pivot without building the compiler.
5. **H1, H2, H4, H6.** These are cheap and should come in the same change:
   - make the analyzers run locally like `vulcan check`;
   - give the rule writer a brief free of Vulcan terms;
   - rewrite the decision rule as a complete table;
   - keep the sealed contents out of the agents' repository, with externally timestamped hashes.

H3, H5, and H7 should follow before milestone 4 is sealed.

## Resolution

On 2026-10-21 the owner had the minimum changes applied, plus the cheap fixes of M4 and L1, in the
commit that adds this report. The plan went back to **proposed**, for a new independent audit.

| Finding | Change |
|---|---|
| B1 | Decision 6: arm R holds every rule that Go can express, signed by the Go developer; rules are labelled per arm into sets S, U, G, N; "continue" needs a win on U (T2) and a tie on S (T3). |
| B2 | Decision 6: exact sign-flip permutation tests on paired per-task differences; at least 8 rules (S) or 4 (U) and 10 events, else inconclusive, which counts as fail; P2 undefined below 10 events; 8 tempting tasks in milestone 4, 6 in milestone 0.5. |
| B3 | Decision 6: mandatory non-owner roles, including a task reviewer and a run adjudicator; no escape clause (row 1 of the decision rule stops the project); an operator who answers only from a sealed FAQ. The plan adds an 8-week recruiting deadline. |
| B4 | Decision 6, section 12, and milestone 0.5: gate 2 pivots to the rules without a compiler when fewer than 4 rules need a language, or their violations are fewer than 5. |
| H1 | Decision 6: one local check command per arm; review only after it passes. |
| H2 | Decision 6: the rule writer gets only the F, D, S, and R criteria and a neutral diagram; rules are sealed before labelling. |
| H4 | Decision 6, section 10: an ordered table that covers every case, with "not judged" and "stopped by schedule"; the plan lists six outcomes. |
| H6 | Decision 6, section 11: a private store, an outside timestamp, isolated workspaces. |
| M3 (cheap part) | The plan's milestone-3 gate: 32 criteria for Vulcan, 28 for Go. |
| M4 | S08 covers source constructs only; new S10 covers deferred commands; the plan's K0009 sentence is corrected; D12 is exempt from the differential rule; decision 1 closes its open point with a written exception to design rule 4. |
| L1 | A note on old names in decisions 1 to 4; decision 6's current-state text; the README's "Draft plan"; the spec word count. |

Also answered in passing: H3 in part (task success is bound N0; failed runs count at the full
budget), H5 in part (N1 excludes queue time; N2 uses a difference margin), M1 (latency on a
synthetic application in milestone 2), L2 (spec §1.1 maps the hypotheses to T1 to T3 and N2).

Still open, before milestone 4 is sealed: the rest of H3 (review time is measured, not gated), H5
(a power check of N1 and N2 on the pilot), H7 (budgets for volunteer hours and tokens; a split of
milestone 2), M2 (named judges for every exit gate), M5 to M10, and L3 to L6.
