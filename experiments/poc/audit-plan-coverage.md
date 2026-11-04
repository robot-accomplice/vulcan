# Audit of the draft plan `plan-coverage.md`

Date: 2026-11-03. Object: [docs/poc/plan-coverage.md](../../docs/poc/plan-coverage.md), as it is in
the working tree on top of main commit `4752184` (the file is not committed).

Auditor: an independent session. It did not write the plan, the views draft, plan-r3, or
hypotheses-r3. No outcome is wanted. The audit looks for leans toward V as hard as for leans against V.

Read: the plan; the findings report, table "Where we are, by principle"; plan-r3 (with amendments 1
to 12); hypotheses-r3 (sections 1 to 9); the views draft; README ("Why Vulcan", "What Vulcan adds to
Go", "Evidence so far"); `docs/status.md` ("Open work"); spec §1.1, §2, §3, §5.5 (headings), §7 to
§11; desk-test-r2 (summary and section 6). Not read: anything under `C:\Users\jonat\vulcan-poc`, and
hidden harness material.

Severity: **High** means the plan cannot give a valid verdict on a row, or it leans materially.
**Medium** means a verdict is weaker, or a part is missing, but a fix inside the plan is simple.
**Low** means a gap in the record or the wording.

## 1. Summary

| Severity | Count |
|---|---|
| High | 12 |
| Medium | 15 |
| Low | 6 |

The plan has a good frame: three arms, an equal build, a parity gate, one verdict rule for each
direction, and sealed material. But it does not yet close the 14 rows by the owner's rule, for four
main reasons:

1. **No row has a terminal outcome.** "Unclear" does not close a row, and nothing stops a row from
   staying "Unclear". The run sizes make "Unclear" the likely result on task success (H1, H2).
2. **Several tests do not measure their claim.** Rows 2 (against P), 6, 7, 9 (against P) and 10 have
   tests whose result is fixed by construction, or that measure a different thing (H4 to H6, H10, M9).
3. **The plan replaces accepted rules without saying so.** Its verdict rule conflicts with the stop
   rules of hypotheses-r3, and campaign C2 conflicts with the gate of plan-r3 (H7, H8).
4. **The campaigns share runs with all new features on at once.** A verdict cannot be given to one
   row (H9).

Leans found, by direction:

| Lean | Direction | Finding |
|---|---|---|
| P has no architecture linter on the architecture row | Toward V and L, against P | H4 |
| The pilot gate selects tasks on P's failures | Toward V and L, against P | H3 |
| Harms N1 to N6 and NV1 to NV3 are not measured | Toward V | H12 |
| Seeded cases on a Linter-partial theory | Toward a null for V against L | H6 |
| Replication only for a Gain or a Loss | Toward a null in each direction | M1 |
| Old one-run results kept as closed | Against V (rows 1, 15) | M3 |
| P gets common analyzers that L and V do not get | Toward P on code quality | M4 |
| Duplicate seeds written from the spec's own definition | Toward V and L, against P | M10 |
| P gets no primer of equal size | Toward V and L, against P | M11 |
| L gets "the same output" for `context` and `query` | Toward a null for V against L | L3 |

## 2. Findings on the plan as a whole

### H1. High. No row has a terminal outcome, so a row can stay open forever

- **Location.** Section 0 ("An 'Unclear' verdict does not close it"); section 1, rule 4; views draft
  section 3 ("One more batch, sized before it is run, then read the same way").
- **Problem.** After the extra batch, the rule reads the result again. The result can be "Unclear"
  again. The plan sets no maximum size and no final reading. The hypotheses-r3 stop rules say the
  same: "unclear stays reported as unclear" (rank 1), "Report it" (ranks 2 to 4).
- **Effect.** The owner's rule needs each row to close. Under this plan, a row with a small true
  effect never closes. The completion rule then has no end. A reader could also be tempted to grow
  batches until an interval leaves the margin. That is optional stopping, and it leans toward
  whichever result appears first.
- **Fix.** For each test, set two sizes before any data: the first batch and a final maximum. At the
  final size, a result that is still "Unclear" is recorded as **"Tested, unresolved at N, interval
  [a, b]"**. Ask the owner, in the amendment, whether that entry counts as tested. Grow no batch past
  the final size.

### H2. High. The run sizes are too small to give "No measured gain" on task success

- **Location.** Section 6, table "Run counts". Rule 4 needs the whole 95% interval inside ±X.
- **Problem.** C1 and C2 give 60 runs an arm (20 tasks × 3 reps). The runs are clustered by task.
  Take a success rate near 0.8 and a margin of 12 points (plan-r2 verdict 3). With no pairing gain,
  the standard error of a difference is about sqrt(2 × 0.16 / 60) = 0.073. The interval half-width
  is about 14 points. That is wider than the margin, so the interval cannot lie inside ±12 even when
  the true difference is 0. C4 (30 runs an arm) and C5 (45 runs an arm) are worse. Pairing by task
  can narrow the interval, but the plan does not show that it is enough. At a success rate near 0.9
  the half-width is about 11 points, so the result depends on luck. Tokens are different: with the
  round 1 spread (SD of the task log ratio 0.26), 20 tasks give a half-width near 0.11 on the log
  ratio, inside the 0.87 bound (0.14). So tokens can close, and success mostly cannot.
- **Effect.** Rows 1, 3, 8, 10, 12 and 13 read task success. Most of them end "Unclear" by design.
  With H1, they stay open.
- **Fix.** Size each campaign for the equivalence outcome ("No measured gain"), not only for
  detection. Report the planned size per row with its expected half-width. Use paired analysis by
  task with a cluster bootstrap. Use 5 runs a task where hypotheses-r3 asks for 5 (rank 2, N1).
  Where the size is too costly, say before any run that success on that row can only give
  "Tested, unresolved" (H1).

### H7. High. The verdict rule replaces accepted stop rules without an amendment that says so

- **Location.** Section 1, rule 4 ("The rule of hypotheses-views section 3 applies"); rows 5, 6, 7,
  8, 10 and 11.
- **Problem.** Hypotheses-r3 is accepted. Its theories have their own measures and stop rules:
  - T8 (rank 3): counts on 108 workspaces (at least 10 supports, at most 1 refutes);
  - T16, T17, T20: "the arms differ by more than 1 seed";
  - T21 (rank 2): one-sided 95% bound above 1.05, and a 90% interval within 0.95 to 1.05;
  - T5 (rank 1): at least 10 points with the interval above 0;
  - T7 (rank 4): form ratio 1.25, bypasses within 1, and verdict 3 in runs.
  
  The plan says one rule (D, interval, margin X) applies to every row. It does not say which rule
  wins where they differ. The views draft is not accepted, so the plan imports a rule from a draft
  over an accepted document.
- **Effect.** Two readers can give two verdicts on the same data. A reader can pick the rule after
  the data. That is a forking path, in either direction.
- **Fix.** In the amendment, state one rule for each row and each column, with its margin, written
  down now. Either keep the hypotheses-r3 stop rules where they exist and use rule 4 only for rows
  that have none, or amend hypotheses-r3 by name. List each replaced rule.

### H8. High. Campaign C2 conflicts with the gate of plan-r3

- **Location.** Section 3 ("A fresh session writes the tasks from the blind rule set of plan-r3
  T1"); plan-r3 section 5 gate ("If not, no tasks are written and the round stops again").
- **Problem.** Plan-r3 forbids tasks if T1 has fewer than 8 rules in S and 4 in N. The plan writes
  C2's tasks from T1's rule set and does not say what happens if the gate fails. Rows 2 and 3 then
  have no path to close.
- **Effect.** Either the plan breaks an accepted gate, or rows 2 and 3 stay open with no rule. The
  first leans toward running tasks that the gate found not worth running. The second blocks
  completion.
- **Fix.** State the rule for both cases. Option A: C2 runs only if the plan-r3 gate passes, and if
  it fails, rows 2 and 3 are recorded as "Tested at the label stage: the checks enforce too few rules
  for a task test" (and the owner decides if that closes them). Option B: an amendment of plan-r3
  that lets C2 run on the rules that the checks enforce, whatever the gate says, with the reason.

### H9. High. Shared runs with all new features on cannot give a verdict to one row

- **Location.** Section 6 ("Campaigns share runs where they can. C1 serves rows 1, 8, 9, 11 and
  12. C2 serves rows 2, 3, 7, 11 and 14").
- **Problem.** Phase B adds T7 forms, compiler intelligence, `context` and `query`, the views tools
  and the crossing subset to V and L. The plan does not say which of them are on in each campaign.
  If all are on, V against L in C1 measures the whole set. A cost result on row 1 or row 8 can come
  from row 11's fixes or row 13's `context`. The primer ablation of row 12 is the only condition that
  removes one feature.
- **Effect.** A Gain or a Loss on rows 1, 8, 11, 12 and 13 cannot be given to its principle. A null
  can also hide a gain of one feature and a loss of another. This is a confound in each direction.
- **Fix.** For each campaign, list the features that are on in each arm. Give each row whose verdict
  is a task measure its own ablation condition (feature on and off, in the same arm), or run it in a
  campaign where only that feature differs. Pre-register which comparison closes which row.

### H12. High. The plan measures the claims and not the named harms

- **Location.** Section 2; hypotheses-r3 section 4 ("Each harm can cancel a gain, so a round must
  measure them with the same care"); views draft section 5 (NV1 to NV3).
- **Problem.** Only N1 has a home (row 1). N2 (annotation and solver turns), N3 (runs that do not
  stop), N4 (weakened contracts), N5 (Go-looking text with another meaning, T8's price), N6 (merge
  conflicts on the central file) and NV1 to NV3 are in no row and no campaign.
- **Effect.** A row can show "Gain measured" for V while the harm that cancels it is not measured.
  This leans toward V. It also breaks the equal-burden rule of the owner.
- **Fix.** Add a measure for each harm to the campaign that produces its data: N3 and N4 in C1 and
  C2 (budget hits, a scripted diff of contracts); N2 and N5 with rows 5 and 6; N6 needs parallel
  runs, or a stated reason why it stays out; NV1 to NV3 with row 14. Read each harm with the same
  rule and replication as a gain.

### M1. Medium. Replication applies to Gains and Losses only, and its form is chosen late

- **Location.** Section 1, rule 5; D5.
- **Problem.** "No measured gain" needs no replication. A null on one model and one corpus closes a
  row. The plan also lets a Gain be replicated by "a second model, or a second corpus", with no rule
  for which, and no rule for a replication that disagrees.
- **Effect.** The rule leans toward a null in each direction: a gain for V and a loss for V each
  need more evidence than "no difference". The free choice of model or corpus is a forking path.
- **Fix.** Say per row, before data, whether the replication is by model or by corpus. Define the
  verdict when the two disagree (for example "Unclear across replications"). Either replicate a null
  too, or write in the report on each null row that it rests on one model and one corpus.

### M2. Medium. The margins are not set, and phase A data come before them

- **Location.** Section 1, rule 4 ("a margin X set before any data"); section 6 ("their results can
  change what the campaigns need").
- **Problem.** The plan sets no margin for any row. Phase A produces data before the campaigns. The
  plan also lets phase A results change the campaigns, with no rule for how.
- **Effect.** Margins and campaign designs can be set after related data are seen. This can lean in
  either direction.
- **Fix.** Put each margin in the amendment that accepts the plan. State in advance what a phase A
  result can change (for example: drop a campaign condition only if its row is already closed), and
  record each change as an amendment with its reason.

### M3. Medium. Old entries stay closed on weaker evidence than the plan asks of new ones

- **Location.** Section 2 ("Row 2's L column is closed ... Row 15 ... stays as measured"); section
  4 ("not measured again unless the owner asks"); findings report rows 1, 2, 5 and 15.
- **Problem.**
  - Row 1 against P ("Worse once, tight budget, harness fault"): one model, a faulty budget, no
    replication. Under rule 5 it is not a Loss.
  - Row 15 (check time): one number per arm, no interval, no margin, no replication. The claim is
    "incremental builds and per-leaf tests"; the measure is one check after a one-file change.
  - Row 5 against L ("No difference at check time"): the class C rules are run-time rules by design,
    so 0 against 0 at check time does not measure contracts. It is a count that stands in for the
    claim, which the owner's rule does not accept.
  - Row 2 against L: a label count with no interval. It is a direct measure of the claim, so this
    one is the weakest concern.
- **Effect.** Rows 1 and 15 keep two results against V that new rows could not keep. That leans
  against V. Row 5's L column is closed by a proxy.
- **Fix.** Read row 1 against P from C1, which has a P arm anyway, with the same rule. Measure row
  15 again with a verdict rule (median of repeated checks, per-leaf test time, margin set now). Test
  row 5 against L with A1 in all three arms, at run time and test time. Read row 2 against L from
  T1 too.

### M4. Medium. The arms differ in tool set as well as in model

- **Location.** Section 1, rule 1; section 4; row 13.
- **Problem.**
  - P gets `staticcheck`, `errcheck`, `exhaustive`, `dupl`, `go test -race` and `gopls`. The plan
    does not say that L and V get them. L is Go and may use every Go tool (hypotheses-r3 section 1).
    V lowers to Go. If only P has them, the arms differ in two factors, and P's final code can score
    better on masked errors and duplicates because of the tools.
  - `gopls` helps an agent only through an editor protocol. Spec §10 defers V's LSP and MCP servers.
    The plan does not say how each arm's agent reaches its tools, or what each prompt says of them.
- **Effect.** Rows 11 and 13 lean toward P, or against P, by the tool access and not by the model.
- **Fix.** Give every arm the common analyzers, with L and V adding their own checks on top. Then
  the arm difference is the model or the language only. Give each arm's tools the same access path
  (a command line) and the same wording in the prompt. Record both.

### M13. Medium. The phase table does not match the builds that the tests need

- **Location.** Section 6, phases A and B.
- **Problem.**
  - Phase A lists A3, A5 and A6 and rows 7, 10 and 11, but these tests need the T7 forms, the
    crossing subset and the compiler intelligence of phase B.
  - Phase A lists shortlist ranks 1 and 3. Rank 1 needs a verifier, a trap mode in V and a checked
    library in L (hypotheses-r3 rank 1, build). Rank 3 needs a trap rewriter. Row 8 needs the L
    generator. None of these is in phase B, so none gets the fresh L builder and the parity audit of
    rules 2 and 3.
- **Effect.** The order cannot run as written. Builds without a parity audit can lean either way.
- **Fix.** Move A3, A5 and A6 after their builds. Add to phase B: the verifier and trap mode (rows
  5, 6), the checked library and analyzer for L (row 6), the trap rewriter (row 6), the L generator
  (row 8), the tempting tasks (rows 2, 3), the P fakes or P assertions (rows 5, 9), and the port of
  the C5 codebase (H11). Give each a parity audit.

### M14. Medium. The amendment does not list what it changes in plan-r3 and hypotheses-r3

- **Location.** Section 0 ("It changes nothing in plan-r3 or hypotheses-r3 until the owner accepts
  it by an amendment"); D1.
- **Problem.** Acceptance would change accepted documents in at least these places:
  - a P column in T1 and T2, where plan-r3 step 6 says "Arm P keeps the team's rules as prose";
  - new tools for P;
  - the verdict rule (H7);
  - C1 to C5 in place of, or beside, plan-r3 step 8 (plan-r2's about 370 runs) and the 180 runs of
    rank 2 (T21);
  - T29a, which the views draft puts in step 8 runs.
  
  The plan does not say whether C1 is plan-r3 step 8 or an extra campaign.
- **Effect.** Runs and costs can be counted twice, or a test can run under two plans with two rules.
- **Fix.** Add a table: each changed line of plan-r3 and hypotheses-r3, the old text, the new text.
  State how C1 to C5 relate to plan-r3 step 8 and to rank 2, and give one total of runs.

### M15. Medium. The list of principles misses some claims and merges others

- **Location.** Section 2; findings report table; README "What Vulcan adds to Go"; spec §2, §3.
- **Problem, missing.**
  - R5 (spec §3): type information on incomplete programs, so tools can guide generation by type.
    Theory T24 (typed holes) is about it. No row has it.
  - "Imposed structure" (spec §2): one layout, one formatter, one place for each thing. Theory T23
    is about it. No row has it.
  - The gap list and `vulcan audit` (README: "lists each path around them that it cannot check";
    spec §5.4d, §10). The soundness claim, which the "holes" stage of hypotheses-r3 measures, has no
    row. Row 13 names `context` and `query` but not `audit`.
  - Performance (spec §2, §11.3; decision 6, N4): "on the Go backend, Vulcan performs like Go, plus
    contract checks and port dispatch". No row measures the cost of port calls and contracts. It can
    be argued that this is outside an evaluation for AI-assisted work. If so, the plan must say so
    (section 8 does not).
  - Contracts that "reviewers can audit" (spec §2, R6): row 5 tests defects caught, not review.
- **Problem, merged.** Row 5 joins run-time contracts, exact arithmetic, static proof and review.
  Row 10 joins topics, remote ports, dead letters and `topics.lock`. Row 11 joins duplication, masked
  errors, races and machine-applicable fixes, with different predicted verdicts (T20
  Linter-partial, T22 Linter-equal). One verdict per row can hide a gain on one part and a loss on
  another.
- **Effect.** The evaluation can be "complete" while parts of the README's claims have no test.
- **Fix.** Ask the owner to decide on each missing claim: add a row, or record it as out of scope
  with the reason. Give a separate verdict for each part of rows 5, 10 and 11, and close the row
  only when each part has one.

### L1. Low. Many comparisons, and no plan for multiple tests

- **Location.** Section 1, rule 4.
- **Problem.** About 28 columns, each with several measures, are read at 95%. Chance alone gives
  some intervals outside a margin. Replication catches false Gains and Losses (rule 5), but not false
  nulls.
- **Fix.** Name one primary measure for each column. Report the others as secondary, and let no
  verdict read them.

### L2. Low. The unit of analysis is not stated

- **Location.** Section 6, run counts.
- **Problem.** Reps on one task are not independent. The counts multiply tasks by reps as if they
  were.
- **Fix.** State that each campaign is analyzed by task (paired, with a cluster bootstrap), as
  hypotheses-r3 T5 asks for clauses by function.

### L4. Low. The total cost is not stated

- **Location.** Section 6; D7.
- **Problem.** The estimates sum to 705 runs, plus the pilot, up to 705 runs of replication,
  reviewer sessions (C3), views questions (T28, 40 × 3 × conditions × arms) and the 20 tasks of T30's
  use measure. Round 1 had 72 runs. H2 says the sizes must grow. The plan gives no total.
- **Fix.** Give one total, with a low and a high bound, after the fix of H2 and M14.

### L5. Low. Decisions bundle other decisions

- **Location.** D3, D4, D6.
- **Problem.** D6 takes in the 5 decisions of the views draft section 7. D3's "common compatibility
  check" for P means a schema format (for example protobuf with `buf breaking`), which changes P's
  code. D4 needs a license check and a port (H11). The Z3 download (rank 1) is not in D1 to D7.
- **Fix.** List each bundled decision on its own line, and add the Z3 download.

## 3. Findings by row

### Row 1. Bodies are plain Go (against L)

- See M5 (double count with row 8) and H2 (size). Row 1 is N1 in other words: V against L on success,
  turns and tokens. The test is valid for N1 if the features of phase B are off (H9).

### Rows 2 and 3. The compiler enforces the architecture; rules leave review

#### H3. High. The pilot gate leans against P, and its fallback closes rows with no test

- **Location.** Section 3, "Pilot gate".
- **Problem.**
  1. The gate keeps tasks where P breaks rules in at least one third of runs. Selection on one
     arm's failures is not neutral. It is neutral between V and L, but it leans toward V and L
     against P. If the pilot runs count in the main data, P's rate is biased up. If they do not, P's
     main rate regresses toward its mean, and the size of the bias is unknown. The plan's sentence
     "so it does not lean toward V or L" is wrong for the P comparison.
  2. If the second pilot fails, the plan closes rows 2 and 3 as "No measured gain: the tasks did not
     tempt the model". No V or L run took place, so no difference D was measured. This is analysis
     standing in for a measurement, which the owner's rule does not accept, and it labels a null that
     no test found.
  3. The rewrite is "once", but the plan does not say who rewrites, or what the writer may see of
     the pilot.
- **Effect.** Rows 2 and 3 against P lean toward V and L. In the fallback, rows 2 and 3 are
  closed with a verdict that has no data behind it.
- **Fix.** Name the lean in the record. Do not count pilot runs or pilot tasks in the main data. Run
  the pilot in all three arms, and report each arm's rate, so that the main reading can adjust for
  the selection. Replace the fallback with an outcome named "Not testable on this model: no task
  tempted a break", and ask the owner whether it closes the row. Have a fresh session do the
  rewrite. It sees only P's rate per task.

#### H4. High. Row 2 against P gives P no architecture tool, so the result is fixed in advance

- **Location.** Row 2 ("P's best equal: the team's rules as prose plus P's tools"); section 4.
- **Problem.** P's tools (`go vet`, `staticcheck`, `errcheck`, `exhaustive`, `dupl`, `-race`) do not
  read architecture rules. So the "P column" of T1 and T2 rejects almost no seeded case, by
  construction. A Go team that wants its architecture enforced uses `internal/` packages, `depguard`
  (in golangci-lint) or `go-arch-lint`. The plan's own rule 2 says "Where a Go team has a common tool
  for the job, P uses that tool". Section 4 does not list these tools.
- **Effect.** The seeded-case result for row 2 against P is known before any run. It leans toward V
  and L. A result that cannot fail does not close the row.
- **Fix.** Give P a configured architecture linter (`depguard` or `go-arch-lint`, latest stable),
  set up from the prose rules by a fresh session, with the same budget as the L builder. Report P
  with and without it. Then "V and L against P" asks what Vulcan's model adds over an off-the-shelf
  architecture linter, which is the real alternative (D2's own reason).

#### M6. Medium. Rows 2 and 3 share one measure, and row 3's "review" part is not measured

- **Location.** Rows 2 and 3; campaign C2; spec §1.1.
- **Problem.**
  - Both rows close on "rule breaks in final code" from C2. One number closes two principles.
  - Row 3's claim is that rules "leave review". No measure in the plan reads review. C3 reviews only
    logic defects.
  - "Rule breaks fixed against the check during the run" has no meaning in P, which has no check.
    Its P value is 0 by construction.
  - The judge of a rule break sees the arm (V's `.vulcan` files). P is judged on prose rules, L and V
    partly by the check. The two methods differ.
  - The desk test (section "Summary", point 2) notes that V's agent must edit `system.vulcan` for
    ordinary work, while L's manifest can be protected. The plan does not say what each arm may edit
    in C2.
- **Effect.** Row 3 can close with no measure of its claim. The judge's method can lean either way.
- **Fix.** Give row 3 a review measure: for example, reviewers find seeded architecture breaks in
  diffs, with and without a passed check, in each arm, and the time and tokens they spend. Define
  "fixed against the check" for P as "broken in an intermediate state and not in the final state",
  judged the same way in all arms. Judge rule breaks with one procedure on the Go code that each arm
  builds (V's lowered Go). State which files each arm may edit, as F7 does.

### Row 4. No gain on reading or logic defects

#### M7. Medium. C3 cannot be blind to the arm, and its size is not set

- **Location.** Row 4; campaign C3 ("a sample of C2's diffs").
- **Problem.**
  - A reviewer sees `.vulcan` files and knows the arm.
  - H3 predicts no difference. To show "No measured gain", the interval must lie inside ±X, which
    needs more data than to detect a difference (H2). "A sample" has no size.
  - C2's diffs come from tasks built to tempt architecture breaks. They are not typical code for
    logic review.
  - The plan does not say how the logic defects are seeded into agent diffs, or who seeds them.
  - Reviewer sessions are agent sessions with a cost. The cost table says "no task runs" but gives
    no count.
- **Effect.** The verdict can lean with the reviewer's expectation of the arm. It is likely to end
  "Unclear" (H1).
- **Fix.** Accept that the arm is visible, and record it as a limit. Use one fixed reviewer prompt
  for all arms. Seed defects in the Go bodies only, by a fresh session, with the same seeds placed in
  equal code in each arm. Size C3 for the equivalence outcome. Use diffs from C1 (ordinary tasks) as
  well as C2. Count reviewer sessions in the cost.

### Row 5. Contracts (against P)

#### M8. Medium. The builds the row needs are missing, and T5 cannot close the P column

- **Location.** Row 5; section 6; findings report row 5.
- **Problem.**
  - The README claims "exact arithmetic in contract expressions". The PoC does not meet C08
    (`docs/status.md`). The plan has no build for it.
  - "Static proof" needs the verifier and the Z3 download (hypotheses-r3 rank 1). Neither is in phase
    B or in D1 to D7 (M13, L5).
  - The open entry is "Against plain Go". T5 compares V and L, and gives no P verdict. Yet the row
    closes only when "A1 and T5 give verdicts". T5's own stop rule can end "unclear", so the row can
    stay open for a test that does not bear on its open entry.
  - P's assertions and tests: the plan does not say who writes them, or to which statement of the
    contracts.
- **Effect.** The row can stay open because of T5. A1 can lean if P's assertions are written with a
  smaller budget or less information than V's 29 clauses.
- **Fix.** Close the P column on A1 alone. Report T5 as part of the V-against-L column (M3). Give the
  P form to a fresh builder with the same neutral statement of each contract that V and L had. Add
  exact arithmetic to the build, or record that the claim is narrowed to Go arithmetic.

### Row 6. Safer body meanings (T8)

#### H5. High. A2 uses a look that hypotheses-r3 bars from any stop rule, and L cannot be measured on it

- **Location.** Row 6, test A2 ("the trap rewriter on round 1's 72 final workspaces"); hypotheses-r3
  section 9, "A first look at T8" ("no stop rule reads it, whatever it shows, and rank 3's test still
  runs as written").
- **Problem.**
  1. The accepted text says the round 1 look is exploratory and decides nothing. Rank 3's test needs
     the 108 calibration workspaces of plan-r2, which come from agent runs (phase C), not phase A.
  2. Round 1's workspaces are arms G and V. No L workspace exists. L's form is "a checked-math
     library and an analyzer that requires it". Code that was written without the library either
     fails the analyzer at each operation, or has no marked types to guard. Neither gives a fair L
     number.
  3. T8's measure is a base rate (does a trap fire on a real fault). Rule 4's D against a margin is
     a different rule (H7).
- **Effect.** A2 cannot close row 6 by the accepted rules. Its V-against-L column has no valid L
  value.
- **Fix.** Close row 6 with rank 3's test as written, on the calibration workspaces, after phase C.
  For the L column, measure L where agents wrote code with the library: the runs of a campaign in
  arm L, with the same seeded overflow cases and the same rewriter on V. Report the round 1 look as
  exploratory only.

### Row 7. Illegal states (T7)

#### H6. High. Seeded cases cannot show a difference on a Linter-partial theory

- **Location.** Row 7 ("Test A3 on seeded cases, then campaign C2"); hypotheses-r3 section 1, "What
  each kind of verdict can and cannot show"; rank 4.
- **Problem.**
  - Hypotheses-r3 says a label gate shows a difference only for Language-only theories. For a
    Linter-partial theory, L rejects the seeded cases too. T7 is Linter-partial. A3 is a seeded-case
    test, so its V-against-L result is a null by construction.
  - Rank 4's accepted test is form on 20 instances, holes by an attacker 3 times, use without a
    prompt, and runs only on tasks where value rules can be broken. The plan drops form, holes and
    use.
  - C2's tasks tempt architecture breaks, not value-rule breaks.
- **Effect.** Row 7 against L would close as "No measured gain" by a test that could not find a
  gain. That leans against V. The C2 part does not measure the claim.
- **Fix.** Use rank 4's test: form, holes (3 attacker repeats an arm, the same budget), use, and runs
  on value-rule tasks. Keep A3 only for the P column, where P has `exhaustive` alone.

### Row 8. Less work for a correct program (T21)

#### M5. Medium. C1 has no "L with the generator" condition, and rows 1 and 8 share one measure

- **Location.** Row 8 (build: a generator for L); C1 conditions (P, L, V, L without primer, V without
  primer); hypotheses-r3 rank 2.
- **Problem.**
  - Rank 2 compares V, L, and L with the generator. C1 has no condition with the generator. If L in
    C1 has the generator, the plan loses the plain L condition that row 1 and N1 read.
  - Rows 1 and 8 both read success, turns and tokens of V against L in C1. One measure closes two
    rows.
  - Row 8 against P: the claim is "the same guarantee in fewer tokens". P has fewer guarantees, so a
    token difference against P does not measure the claim.
  - Rank 2 asks for 5 runs a task. C1 has 3.
- **Effect.** Row 8 cannot give rank 2's verdict. Rows 1 and 8 are not independent results.
- **Fix.** Add the condition "L with the generator" to C1. Close row 8 on V against L with the
  generator, and row 1 on V against plain L. For row 8 against P, define the guarantee P must
  meet (for example P with the same contracts as assertions, M8), or record the P column as "a cost
  comparison at unequal guarantees". Use 5 runs a task.

### Row 9. Tests with fakes bound to ports

#### M9. Medium. In P the measure has no meaning

- **Location.** Row 9, test A4 ("seeded fakes that drift from a port contract").
- **Problem.** P has no port contracts. A fake in P cannot drift from a contract that does not exist.
  P's value is 0 or undefined by construction. "Test defects in final code" (C1) has no definition and
  no labeler.
- **Effect.** The P column closes by a result that could not differ. It leans toward V and L.
- **Fix.** Give P its best equal: interfaces with the contracts as assertions in the real provider
  and a shared test suite that each fake must pass (a common Go practice). Seed the drift against
  that. Define a test defect and give its labeling to a fresh session.

### Row 10. Crossing processes

#### H10. High. The subset cannot show a compatibility break, and it drops a named part of the claim

- **Location.** Section 5 ("Two containers in one host, with an in-process transport"); row 10;
  campaign C4.
- **Problem.**
  1. If both containers build in one compile unit and the transport passes Go values, the Go
     compiler rejects most incompatible changes in every arm, P included. A compatibility break
     matters when the two sides build and deploy at different versions, and the message crosses as
     bytes. The subset has neither.
  2. The README names dead letters as part of the claim. The subset drops them and only records it.
     The owner's rule does not let a part that stands in for the claim close the row.
  3. T16 predicts Linter-equal, and its accepted rule is "differ by more than 1 seed" (H7).
  4. If L's lock check reuses V's evolution code, V against L is a null by construction (rule 7).
- **Effect.** A5 and C4 can give "No measured gain" because no arm can break compatibility in the
  setup.
- **Fix.** Serialize messages in the transport. Build the producer and the consumer as separate
  binaries, and test an old consumer against a new producer. Seed breaks that the Go compiler does
  not catch (a renamed field, a changed meaning, a removed optional field). Either build dead letters
  or narrow the row's claim by the owner's decision, in the record. State whether L shares V's
  evolution code.

### Row 11. Compiler intelligence

#### M10. Medium. Four claims in one row, and seeds may follow the spec's own definition

- **Location.** Row 11; spec §9 (duplicates: 90% similar, at least 50 tokens); hypotheses-r3 T22.
- **Problem.**
  - Duplication, masked errors, races and fixes have different tests (M15).
  - If the seed writer reads spec §9, seeds fit V's and L's detector, and `dupl` uses another
    algorithm. That leans toward V and L against P.
  - T22's accepted test is "turns from rejection to pass, with and without fixes". No campaign has a
    condition without fixes.
  - `go test -race` is listed for P, but `vulcan test` already runs the race detector (spec §10).
    This part cannot differ between V and P.
- **Effect.** The row's verdict mixes parts, and the duplicate part leans against P.
- **Fix.** Give each part its own verdict. Have a fresh writer seed duplicates from a neutral
  definition (for example from published clone classes, types 1 to 3) without the spec. Add a
  fixes-off condition in one arm of C2 for T22. Drop the race part from V against P, or state that
  it measures the same detector.

### Row 12. Tools teach the model

#### M11. Medium. The ablation mixes the primer with the language, and P has no equal

- **Location.** Row 12; C1 conditions.
- **Problem.**
  - V without its primer must write a language that the model has no training data for. Its result
    measures N1 as much as the primer. L without its primer still writes Go.
  - The claim (spec §10, R8) covers tools that teach: hints, `explain`, JSON output. The ablation
    removes only `guide`.
  - P has no primer. A Go team today often gives an agent a project file of conventions (for example
    an `AGENTS.md`). The P column has no defined measure.
  - The plan does not say what V without a primer gets instead (the spec, nothing, examples).
- **Effect.** The V-against-L contrast of the ablation is close to N1, not to the primer. The P
  column leans toward V and L if P gets nothing.
- **Fix.** Define what each "without primer" condition gets. Give P a conventions file of the same
  token budget, written by a fresh session from the same rules. Read the primer effect as the
  difference within each arm, and compare those differences. Name the tools that the claim covers,
  and ablate the hints too, or narrow the claim in the record.

### Row 13. Small context for the model

#### H11. High. The larger codebase needs a port that the plan does not build, and the port is a confound

- **Location.** Row 13; campaign C5; D4.
- **Problem.**
  - An open-source Go service has no `system.vulcan` and no `arch.go`. V needs a full port into
    leaves and ports. L needs its model declared. The plan lists no such build, no builder and no
    parity audit.
  - The port changes the structure of the code. If P keeps the original service, P differs in
    structure as well as in tools. If P gets the ported Go without the model, that is a new arm form
    that the plan does not define.
  - Who ports also matters: a port by the author of V can fit V's tools.
- **Effect.** C5 cannot run as written. If it runs, a gain for V or L can come from the port's
  structure and not from `context` or `query`.
- **Fix.** Add the port to phase B: a fresh session ports to V, another declares L, both from one
  structure plan written first, with a parity audit. Give P the same restructured Go without the
  model. Report P on the original layout as a second P condition if the owner wants it.

#### L3. Low. "The same output" for L makes V against L a null for the tool

- **Location.** Row 13 (L: "the same output from a `gopls`-based tool").
- **Problem.** If L's tool gives the same pack, V against L can differ only by the code form. This
  is fair, but the plan does not say so on this row, as rule 7 asks for the shared engine.
- **Fix.** Add the rule 7 note to row 13.

#### L6. Low. Invented APIs counted from two compilers

- **Location.** Row 13 (R4, invented APIs).
- **Problem.** V's diagnostics and `go build` messages differ. A count of `undefined:` lines can
  differ by the tool's wording.
- **Fix.** Count invented symbols on the Go build that each arm produces (V's lowered Go), with one
  script.

### Row 14. The architecture can be seen

#### M12. Medium. The views tests are not inside C2 and C5, and the views draft lets the owner stop early

- **Location.** Row 14 ("inside campaigns C2 and C5"); views draft sections 4 and 6.
- **Problem.**
  - T27 needs no agents. T28 has its own 40 questions with 3 conditions. T29b uses reviewer sessions.
    T30's use part needs 20 ordinary tasks with and without the document. T29a uses step 8 runs.
    None of these is a part of C2 or C5 as the plan defines them.
  - The views draft says the owner reads T27 and decides whether steps 3 and 4 go on. If the owner
    stops, row 14 stays open under the completion rule.
  - T29a has a prior on a rate, but no margin and no verdict rule.
  - "The views draft's tests give verdicts" does not say whether all of T27 to T30 and NV1 to NV3
    must give one.
- **Effect.** Row 14 has no defined path to close. Its runs and cost are not in section 6.
- **Fix.** List the views tests as their own campaign (C6), with sizes and cost. State which tests
  close row 14. Remove the option to stop after T27, or state that a stop leaves row 14 open. Give
  T29a a verdict rule.

### Row 15. Fast verification

- See M3.

## 4. Feasibility and cost

| Build | Size in the source documents | Concern |
|---|---|---|
| Crossing subset (row 10) | Spec §5.5 is the largest feature, milestone 5 | The subset as written is too small to test the claim (H10). A valid subset needs serialization and separate binaries, which costs more |
| `context` and `query` (row 13) | T25: "small, in both arms" | Feasible. The port of the C5 codebase is the large cost, and it is missing (H11) |
| Views tools (row 14) | About 300 lines a generator and 500 for ground truth (views draft) | Feasible. The question sets and seeded diffs are extra fresh sessions |
| T7 forms (row 7) | 1,000 to 2,000 lines in V (hypotheses-r3) | Feasible, but rank 4's test needs attacker sessions too (H6) |
| Verifier and trap mode (rows 5, 6) | Z3, an extractor per arm, a VC generator over `go/ssa` | Not listed in phase B, and the Z3 decision is missing (M8, M13) |
| L generator (row 8) | About 500 lines | Not listed in phase B (M13) |

Run counts: see H2, L2 and L4. The likely total after the fixes is well above 705 runs.

## 5. Consistency

| With | Conflict | Finding |
|---|---|---|
| Plan-r3, section 5 gate | C2's tasks when the gate fails | H8 |
| Plan-r3, step 6 | P keeps prose rules only; the plan adds a P column and P tools | M14 |
| Plan-r3, step 8 | C1 to C5 and step 8 overlap, with no total | M14 |
| Plan-r3, amendment 11 | V's agent cannot run `vulcan test` with a new finding; L's can. C2 must measure this cost the same way. The plan does not mention it | M6 |
| Hypotheses-r3, ranks 1 to 4 and T16, T17, T20, T22 | Different verdict rules | H7 |
| Hypotheses-r3, section 9 (T8 first look) | A2 uses the exploratory look as a closing test | H5 |
| Hypotheses-r3, section 1 (what a label gate shows) | A3 on a Linter-partial theory | H6 |
| Hypotheses-r3, section 4 (harms) | Harms not measured | H12 |
| Views draft, sections 4 and 6 | Tests outside C2 and C5; the option to stop after T27 | M12 |

No conflict was found with amendments 1 to 10 and 12 of plan-r3.

## 6. Coverage

See M15. The 15 rows cover the README's areas, but they miss R5, the imposed structure, the gap list
and `audit`, and performance (or the reason to leave it out). Rows 5, 10 and 11 each merge claims
that need their own verdicts.

## 7. Verdict

| Severity | Count | Ids |
|---|---|---|
| High | 12 | H1 to H12 |
| Medium | 15 | M1 to M15 |
| Low | 6 | L1 to L6 |

**The plan is not ready for the owner to accept.** By the owner's rule, it cannot yet close rows 2
(against P), 6, 7, 10, 13 and 14. It has no terminal outcome for any row (H1), and its sizes make
"Unclear" likely on task success (H2). It also replaces accepted rules without naming them (H7, H8).

Of the leans that this audit found, three lean toward V or L against P (H3, H4, M10), one leans toward
V (H12), and two lean toward a null for V against L (H6, L3). The remaining leans are toward a null in
each direction (M1), against V (M3), or depend on the tool set (M4, M11). No finding shows that the
author meant a lean. Most come from closing many rows with a few shared tests.

The fixes are within the plan's frame. A revised plan should:

1. Add a terminal outcome and final sizes (H1, H2).
2. Fix the tests that cannot fail or do not measure their claim (H4, H5, H6, H10, M9).
3. Give one verdict rule per column, named against the accepted documents (H7, H8, M14).
4. Pre-register which features are on in each campaign (H9).
5. Measure the harms (H12).
6. Add the missing builds and their parity audits (H11, M13).

After that, a short rerun of this audit should check the fixes before the owner decides D1.

## 8. Rerun on draft 2

Date: 2026-11-03. Object: draft 2 of `docs/poc/plan-coverage.md` (353 lines, in the working tree,
not committed). The same auditor, scale and constraints as above. Nothing under
`C:\Users\jonat\vulcan-poc` was read.

### 8.1 Result

| | Count |
|---|---|
| Draft 1 findings: Closed | 20 |
| Draft 1 findings: Partly closed | 12 |
| Draft 1 findings: Open | 1 (M11) |
| New findings: High | 4 |
| New findings: Medium | 11 |
| New findings: Low | 5 |

Each finding that is Partly closed or Open has its remaining problem stated as a new finding (R1 to
R20). So the new counts are the full list of open problems. Nothing is counted twice.

**Verdict: draft 2 is not ready for the owner to accept.** It is much better than draft 1. It closes
the structural gaps: the terminal outcome, one rule per column, the feature matrix, the harms, the
missing builds, the amendment table, and the missing rows. Four High findings remain. R3 leans toward
P. R2 and R4 lean toward V. R1 is a path to close a column with no information.

### 8.2 Status of each draft 1 finding

| Id | Status | Reason | Rest in |
|---|---|---|---|
| H1 | Partly closed | The terminal outcome exists. But the final size is not fixed: section 6 sizes "are estimates, to be sized by calibration", and D7 sets spend per campaign. No floor on the interval width exists | R1 |
| H2 | Partly closed | 5 runs a task and final sizes are set. The stated half-widths leave out the clustering of runs in tasks | R5 |
| H3 | Partly closed | Pilot in all arms, not counted, lean named, fresh rewriter. But the pilot may read P with its new linter. D1c recommends closure on an outcome with no measure of the claim | R9, R13 |
| H4 | Partly closed | P gets an architecture linter. But it is configured from the blind team rules, while L and V use frozen defaults | R3 |
| H5 | Partly closed | Rank 3 on calibration workspaces, L from its own runs. But calibration is 108 runs over three arms, and rank 3's thresholds are for 108 workspaces | R6 |
| H6 | Closed | Row 7 uses rank 4's test. A3 is for P only | |
| H7 | Partly closed | One rule a column, accepted rules kept, section 9 lists changes. But several accepted rules are one-sided or point rules, and two rules are said to come from hr3 when they do not | R2 |
| H8 | Closed | C2 runs only if the plan-r3 gate passes. The gate is unchanged | R13 for D1d |
| H9 | Partly closed | Section 5 lists the features per condition. But rows 11a and 11b read C1, where row 11's warnings are off | R8 |
| H10 | Closed | Serialized messages, separate binaries, seeds that the compiler cannot catch, dead letters or D9 | |
| H11 | Closed | The port is a phase B build, by fresh sessions, from one structure plan, with an audit | |
| H12 | Partly closed | Every harm has a campaign. N2 and N5 rest on runs that are optional. No rule says how a harm weighs against a gain | R15 |
| M1 | Closed | Nulls replicate, the method is named per row, disagreement is defined | R12 for T2 |
| M2 | Closed | Margins are in section 2. Rule 12 limits phase A changes | |
| M3 | Closed | Rows 1 (P), 5 (L) and 15 are measured again. Row 2 (L) reads T1 | |
| M4 | Partly closed | The same tools in every arm, the same path and words. How the Go tools run on V is not stated | R10 |
| M5 | Closed | L+gen condition. Rows 1 and 8 read different comparisons. P+ holds the guarantees | |
| M6 | Closed | C3a measures review. One judging procedure on lowered Go. Edit lists are stated | |
| M7 | Closed | Seeds in bodies, one prompt, C1 and C2 diffs, a size and a cost | |
| M8 | Closed | Rows 5a to 5d. P column on A1. D8 and D11 | R7 for 5c (P) |
| M9 | Closed | P's shared fake suite. A test defect has a labeler | |
| M10 | Partly closed | Split, neutral seeds, fixes-off condition. The race column against P cannot differ | R7 |
| M11 | Open | "Without" is defined, and P has a conventions file. The confound with N1 remains, and the new primary measure makes it a lean | R4 |
| M12 | Partly closed | C6, every theory, no stop after T27. T29a still has no verdict rule | R17 |
| M13 | Partly closed | Builds moved to phase B. But phase A still holds row 2 against P, which needs P's linter from phase B. P's verifier front end for 5c is in no phase | R11 |
| M14 | Closed | The table of changes, one total, and C1 and C2 replace step 8 and rank 2's runs | |
| M15 | Closed | Rows 5, 10 and 11 split. Rows 16 to 19. D10 | |
| L1 to L6 | Closed | Rule 7, rule 8, the code-form note, the cost total, separate decisions, one script | R20 for rule 7 |

### 8.3 New findings

#### R1. High. "Tested, unresolved" can close a column at any size

- **Location.** Rule 5 ("Terminal outcome"); section 6 ("Cost (estimates, to be sized by calibration
  before any spend)", "A smaller size means more columns end 'Tested, unresolved'"); D1b, D7.
- **Problem.** The final size triggers the terminal outcome. But section 6 lets calibration and the
  spend decision change the sizes, and no floor exists on the width of the interval. With D1b at
  "yes", a column closes at whatever size the funds allow. A column run at 20 runs, with an interval
  of about ±25 points, closes the same way as one at 200 runs. Calibration also sees the success rate
  of each arm, so a size chosen after it can depend on the differences between arms.
- **Effect.** Tests that say nothing can meet the completion rule. This leans toward an early end,
  not toward an arm. It goes against the owner's rule that a stand-in does not close an entry.
- **Fix.**
  1. Fix each final size in the amendment as a minimum. A smaller campaign gives no terminal
     outcome, and its column stays open.
  2. Let "Tested, unresolved" count only when the interval half-width at the final size is at most a
     stated value (for example 1.5 times the margin).
  3. Let calibration set sizes from the pooled variance only, not from the differences between arms.
     Record the inputs.

#### R2. High. The accepted rules that rule 5 keeps are one-sided or point rules, so some losses for V have no outcome

- **Location.** Section 2, the "Rule" column; rule 5; section 9 ("New rules ... the margins marked
  'Rule 5'").
- **Problem.**
  1. Several hr3 rules test only a gain for V. T23 (row 17): "Refuted if V is not better by a bound
     above 0". T24 (row 16): "Refuted if V's benefit does not beat L's by a bound above 0". T22 (row
     11d) reads a fall in turns and an arm difference, with no direction for a loss. Under these
     rules, V worse than L reads "refuted", the same as no difference.
  2. N1 (row 1 against L) and the "differ by more than 1 seed" rules (rows 10a, and 11c through T20)
     read point values with no interval. With a half-width near 11 points, a point rule reads noise.
  3. Rows 5b and 9 cite "hr3 T26 form: refuted if the arms differ by at most 1 seed of at least 20".
     T26's accepted rule is different ("no more than 10% more seeds are found, or the arms differ by
     more than 12 points"). The cited rule is new, and section 9 does not list it.
  4. No rule says how "refuted" or "supported" maps to the four outcomes of the findings report.
- **Effect.** A measured loss for V on rows 16, 17 and 11d cannot be recorded as "Loss measured".
  This leans toward V. Point rules can give any outcome by chance, in either direction.
- **Fix.** For each column, add a line that maps the accepted rule to Gain, Loss, No measured gain
  and Unclear. Add the mirror condition for a loss where the accepted rule has none. Read point rules
  on the 95% interval. List the rules of rows 5b and 9 as new rules in section 9. The amendment must
  name each of these changes to hr3.

#### R3. High. Row 2 against P: P's linter is configured from the test's own rules, and L and V are not

- **Location.** Row 2, P column ("configured from the prose rules by a fresh session"); plan-r3
  question QD ("do the frozen defaults and the checks enforce").
- **Problem.** T1 and T2 measure L and V with `defaults v1`, frozen before the blind rules exist. So
  L and V are out of sample. P's linter is configured from those same blind rules, so P is in sample.
  The arms get different information about the test.
- **Effect.** P is favored on the share of rules enforced. This is a material lean toward P, the
  reverse of H4.
- **Fix.** Pick one form and apply it to every arm.
  - Option A: P's linter gets a default configuration, written before the blind rules by a fresh
    session from published practice, as `defaults v1` was.
  - Option B: L and V also get a team rules file, written from the same prose rules by a fresh
    session with the same budget, with the rule kinds that exist.

  Report both if the owner wants. The verdict reads the matching pair.

#### R4. High. Row 12's primary measure leans toward a "Gain" for V by construction

- **Location.** Row 12 ("Primary: the change in success within each arm, then those changes
  compared"); the C1 conditions V−primer and L−primer.
- **Problem.** Without its primer, V's agent writes a language with no training data. L's agent still
  writes Go. So V's drop is large because V needs the primer to work at all (N1), not because its
  tools teach more. A larger drop in V reads as "Gain measured" under the difference of differences.
- **Effect.** Row 12 can record a gain for V that is the cost of the new language. This leans toward
  V, and it double counts against row 1.
- **Fix.** Make the primary measure the result with each arm's primer, against the margin, with the
  claim stated as "the primer and tools bring V to at least L's level". Keep the ablation as a
  secondary measure, reported within each arm. Or ablate only the hints and `explain`, and keep the
  primer in every condition, so that the need of the language stays fixed.

#### R5. Medium. The stated half-widths leave out the clustering, and some margins cannot be met at their sizes

- **Location.** Section 6, "Sizes"; section 2 margins "2 rules of 28" (row 2, P) and "2 of 20 seeds"
  (rows 10b, 11a, 11b).
- **Problem.**
  1. 100 runs a condition are 20 tasks × 5 runs. Rule 8 analyzes by task with a cluster bootstrap,
     so the effective sample is near 20 paired task values, not 100 runs. If runs of one task are
     correlated (for example ICC 0.3), the design effect is 1 + 4 × 0.3 = 2.2. The half-width at the
     first size is then near 16 points, not 11. Pairing may win some of it back. The plan states only
     the gain from pairing.
  2. Row 2 against P has 28 rules and a margin of 2 rules (about 7 points). The interval on a share
     of 28 rules is much wider. A blind rule set cannot grow without a new rule writer, so the final
     size has no meaning here.
  3. Seed margins need an interval method. With 20 paired seeds and 0 discordant, an exact upper
     bound is about 14%, above a margin of 10%.
- **Effect.** More columns end "Tested, unresolved" than the plan says. With R1, they may close with
  little information.
- **Fix.** State the half-widths with an ICC taken from round 1 data, and size for it. Name the
  interval method for paired seeds (an exact method on the discordant pairs). For label tests, state
  the final size as a second corpus, or set a margin that 28 rules can read.

#### R6. Medium. Row 6 has about 36 calibration workspaces an arm, and its rule is for 108

- **Location.** Row 6; section 5, "Calibration: P; L; V"; section 6 ("Calibration 108 runs, as
  plan-r2").
- **Problem.** Rank 3's thresholds (at least 10 of 108, at most 1 of 108) are for 108 workspaces. The
  plan reads them per arm, on about 36 workspaces each. The margin "5 of 108" for the difference
  also assumes 108 an arm.
- **Effect.** The rule cannot be read as written. At 36 workspaces, "at most 1" has an upper bound
  near 14%, not 4.3%.
- **Fix.** Run 108 calibration workspaces in each arm that row 6 reads, or rescale the thresholds to
  36 before any data, and list the change in section 9.

#### R7. Medium. Columns where P has no common equal are fixed by construction

- **Location.** Rows 5c (P), 11c (P), 16 (P), 17 (P), 18 (P).
- **Problem.**
  - 11c against P: P has no check-time race tool, so P rejects 0 at check time. The plan says so.
  - 17 against P: P has no checker, so "rejections before the first pass" is 0 or undefined.
  - 18 against P: P has no gap list, so every bypass is "not named".
  - 16 against P: the test is "with and without the command", and P has no command.
  - 5c against P: "P's assertions read as contracts by the same verifier" is a build for P that no
    Go team uses, and no phase lists it. With it, P is close to L. Without it, P proves 0.
- **Effect.** These columns close with a result known in advance. Each one leans toward V and L.
- **Fix.** For each such column, choose before any data:
  1. measure an outcome that exists in every arm (for example races, invented symbols, or bypasses
     in final code, found at test time), or
  2. record the column as "No common Go equal", and let the owner decide whether that closes it.

  Do not read a check-time count where P has no check.

#### R8. Medium. Rows 11a and 11b read final code in C1, where row 11's warnings are off

- **Location.** Section 5, C1 ("Base only, except the feature named by the condition"; "11a, 11b in
  final code"); section 6, phase B (row 11's warnings are a new build).
- **Problem.** "Base" is what each arm ships today. Row 11's warnings are not in it. So C1's count of
  duplicates and masked errors does not measure the warnings.
- **Effect.** The final-code part of 11a and 11b measures nothing about the claim.
- **Fix.** Read the final-code counts of 11a and 11b in C2, which has row 11's features on, against
  a warnings-off condition in the same arm. Or add warnings-on conditions to C1. State it in
  section 5.

#### R9. Medium. The pilot gate may read P with its new linter

- **Location.** Section 3 ("The gate reads P's break rate"); section 5, C2 ("P (with its linter)").
- **Problem.** If P in the pilot has its architecture linter, the gate reads a checked arm. P then
  seldom breaks a rule, and the gate can fail for that reason. The tasks are then judged "not
  tempting" when the check, not the task, stopped the break.
- **Effect.** Rows 2 and 3 can end "Not testable" because of the tool change of rule 2.
- **Fix.** Run the pilot gate on P with prose rules only (no linter). Add the same condition to C2 as
  P−linter. It gives the unchecked baseline that row 3 needs, and it is the arm of round 1.

#### R10. Medium. Rule 2 does not say how the Go tools run on V

- **Location.** Rule 2.
- **Problem.** V's code is `.vulcan` files. V lowers them to Go with generated wrappers and `main`.
  `staticcheck`, `errcheck` and `dupl` run on Go, so on V they read the lowered Go, with `//line`
  positions. Findings in generated code exist only in V. The plan does not say whether they are shown
  to the agent, filtered, or counted.
- **Effect.** Extra noise in V's tool output leans against V. A filter that removes too much leans
  toward V.
- **Fix.** State how each tool runs on V. Filter findings in generated code by one written rule.
  Report the count filtered in each arm, L's generator output included.

#### R11. Medium. Phase A holds a test that needs a phase B build

- **Location.** Section 6, phase A ("Plan-r3 steps 3 to 6, with a P column", rows 2, 15, 19); phase B
  ("P's architecture linter").
- **Problem.** Row 2 against P needs P's linter, which phase B builds. The verifier front end that
  reads P's assertions (row 5c, P) is in no phase.
- **Fix.** Move the P column of T1 and T2 to phase A', after the linter build. Add the 5c P build to
  phase B, or drop it under R7.

#### R12. Medium. Row 2 against L replicates on T2, which is not clean for the dependency check

- **Location.** Row 2, "Repl.: Corpus (T2)"; plan-r3 section 0, and section 5 ("not for the
  dependency check").
- **Problem.** The dependency check was designed after the round 2 diagnostic. So T2 is in sample for
  it. A replication on T2 can confirm a result that the design of the check brought.
- **Effect.** It leans toward V and L on the dependency rules.
- **Fix.** Replicate on T2 without the dependency check, as plan-r3 reports it. Or use a second fresh
  rule set.

#### R13. Medium. D1c and D1d recommend closure on outcomes that do not measure the claim

- **Location.** Section 3; section 10, D1c and D1d ("Yes").
- **Problem.** "Not testable on this model" and "Tested at the label stage" give no task result. Row
  3's claim is about review and fixes against the check, and a label count does not measure it. The
  owner's rule says that a count that stands in for the claim does not close an entry. The plan
  leaves the choice to the owner, which is correct, but it recommends "yes".
- **Effect.** The recommendation leans toward a fast close. It does not lean toward an arm.
- **Fix.** Keep the decisions. Recommend that these outcomes appear in the report as their own entry
  (for example "Not testable: no task tempted a break"), not under "No measured gain". Say in the
  recommendation that they do not measure the claim.

#### R14. Medium. The build effort of phase B is not estimated

- **Location.** Section 6, phase B; section 12.
- **Problem.** Phase B holds about 16 builds. Each needs fresh builders for two or three forms and a
  parity audit. Plan-r3's one build needed six audit reruns. The plan gives run costs, but no estimate
  of build sessions or time.
- **Effect.** The plan is likely to take much longer than its cost table suggests. A phase that stops
  part way can leave rows open.
- **Fix.** Give an estimate of sessions for phase B, with audit reruns. Order the builds by the rows
  they close, so that a stop leaves a known set of rows closed.

#### R15. Medium. Two harms rest on optional runs, and no rule weighs a harm against a gain

- **Location.** Section 4 (N2 in "C5c ... if the owner runs them"; N5 in "Row 6 and 5c runs");
  rule 10.
- **Problem.** Rank 1's runs happen only after a gain and an owner decision. So N2 and N5 can stay
  unmeasured. "A gain on a row is reported next to each harm" gives no rule for when a harm cancels a
  gain.
- **Effect.** A gain on row 5c or 6 can stand next to an unmeasured harm. This leans toward V.
- **Fix.** Measure N5 with rank 3's idiom seeds in every case. Measure N2 in C1 where annotations
  exist. State a rule: a gain on a row with a "Loss measured" harm in the same campaign is reported
  as "Gain with a measured harm", and the owner reads both.

#### R16. Low. C2v has no author, no pilot and no run condition

- **Location.** Section 5, C2v; row 7.
- **Problem.** Rank 4 runs only "if form or holes show a gap". C2v is in the cost table with no
  condition. Its tasks have no author and no tempt check, so it can end as round 1 did: no break in
  any arm.
- **Fix.** Name a fresh author, add a pilot gate like section 3, and run C2v only under rank 4's
  condition.

#### R17. Low. T29a still has no verdict rule

- **Location.** Row 14; views draft T29a (a prior on a rate only).
- **Fix.** Give T29a a measure and a margin in the views amendment, or say that row 14 closes without
  it.

#### R18. Low. Row 19 reads micro-benchmarks for an application claim

- **Location.** Row 19, A8.
- **Problem.** The claim is that an application "performs like Go, plus contract checks and port
  dispatch". A benchmark of port calls alone shows the overhead of one call. "P with the same work" is
  not defined.
- **Fix.** Add one application benchmark (a request path of the base apps). Define P's same work as
  the same code with assertions in place of contracts.

#### R19. Low. Wrong section number

- **Location.** Status paragraph ("Section 10 maps each finding").
- **Fix.** It is section 11.

#### R20. Low. Some columns name two tests and no primary one

- **Location.** Rows 7 (P: "A3 ... then C2v"), 10a ("Then C4"), 11a and 11b ("Then C1 counts").
- **Problem.** Rule 7 asks for one primary measure. Row 10a names "seeds rejected", so C4 is
  secondary there. Rows 7 (P), 11a and 11b do not say which test is primary.
- **Fix.** Name the primary test in each row.

### 8.4 The checks that the coordinator asked for

| Check | Result |
|---|---|
| Rule 5's terminal outcome | Present, but with no fixed final size and no floor on width (R1) |
| Sizes and half-widths of section 6 | The arithmetic matches (11 and 8 points at 100 and 200 runs, unpaired, p = 0.8). It leaves out clustering (R5). The cost totals add up (about 3,600 and 7,000) |
| The feature matrix of section 5 | Good structure. One gap (R8). P−linter is missing from C2 (R9) |
| Rows 16 to 19 | The right claims to add. The P columns of 16, 17 and 18 are fixed by construction (R7). Row 19 is thin (R18). The rules of 16 and 17 are one-sided (R2) |
| Rule 2, the same tools in every arm | Correct in principle. How they run on V is not defined (R10). It changes the pilot gate (R9) |
| The outcomes in section 3 that ask the owner | Correct to ask. The recommendations should not be "yes" without saying that the outcomes do not measure the claim (R13) |

### 8.5 New leans

| Lean | Direction | Finding |
|---|---|---|
| P's linter is configured from the blind rules; L and V use frozen defaults | Toward P | R3 |
| One-sided accepted rules have no Loss outcome | Toward V | R2 |
| Row 12's difference of differences | Toward V | R4 |
| Columns where P has no common equal | Toward V and L | R7 |
| Replication on T2 for the dependency check | Toward V and L | R12 |
| Gains next to unmeasured N2 and N5 | Toward V | R15 |
| Generated code under the common tools | Against V, or toward V if over-filtered | R10 |
| Closure at any size, and "yes" on D1c and D1d | Toward an early end, no arm | R1, R13 |

### 8.6 What a draft 3 needs before acceptance

1. Fix the final sizes as minimums, and set a width floor for the terminal outcome (R1, R5).
2. Map each accepted rule to the four outcomes, with a mirror loss condition (R2).
3. Give P, L and V the same information about the rules in row 2 (R3).
4. Change row 12's primary measure (R4).

The Medium findings can be fixed in the same draft. A check of the four High fixes alone is enough
before the owner decides D1a.

## 9. Second rerun on draft 2, after the owner's decision on D1b and D1c

Date: 2026-11-03. Object: `docs/poc/plan-coverage.md` as it is now (363 lines, not committed). The
coordinator changed section 0, rules 5 and 6, section 3, and sections 6, 8, 9 and 10, after the
owner's decision: "rows do not close in any of these circumstances". I read those sections in full,
and I checked by search that rows 2, 5b, 9, 12 and 17 and the calibration lines did not change. Same
scale and constraints as above.

### 9.1 Result

| | Count |
|---|---|
| Section 8 findings now Closed | R1, R13 |
| Section 8 findings still open (sections they are in did not change) | 18: R2 to R12, R14 to R20 |
| New findings: High | 3 (S1, S2, S4) |
| New findings: Medium | 4 (S3, S5, S6, S7) |
| New findings: Low | 1 (S8) |
| **All open findings** | **High 6, Medium 14, Low 6** |

The open High findings are R2, R3, R4, S1, S2 and S4.

**Verdict: the plan is not ready for the owner to accept.** The owner's decision removes the R1
problem: a column can no longer close with an interval that says nothing. But "a new test after each
Unclear final look, with no limit" brings a problem of its own. Repeated tests, chosen and designed
after each result, are optional stopping at the level of tests. The planned looks inside one test do
not protect against that (S1, S2). This is fixable inside the plan, by a method that is valid under
unlimited extension (S1, fix).

### 9.2 Status of the section 8 findings

| Id | Status | Reason |
|---|---|---|
| R1 | Closed | By the owner's decision, an unresolved result does not close a column, at any size. One part, sizes set after calibration, remains as S6 |
| R13 | Closed | D1c is decided: no closure. D1d no longer recommends closure on a label count. Its new form is S4 |
| R2 to R12, R14 to R20 | Open | The sections that hold them did not change |

### 9.3 Is the group sequential design valid?

The idea is right for one test: planned looks with O'Brien-Fleming spending let a reader look at an
interim result and go on, with the error rate held. As written, it is not yet enough:

- The verdict rule has four outcomes, read from an interval against a margin. Alpha spending is
  defined for a test of one hypothesis. Three outcomes are claims: "Gain measured" (above +X), "Loss
  measured" (below −X), and "No measured gain" (inside ±X, an equivalence claim). Each needs its own
  bound at each look. See S3.
- It holds the error rate inside one test. It says nothing about a series of tests on one column.
  See S1.
- Several columns read accepted hr3 rules that are counts, not intervals (rank 3: 10 of 108; T16:
  "more than 1 seed"). Alpha spending does not apply to them as written. See S3.

### 9.4 Is "a new test after Unclear" free of leans?

No, not as written. It allows three choices after the data, and each one can lean either way:

1. **Whether** the column gets a new test. D7 sets spend per test, so funds can go to the columns
   whose interval points in a preferred direction.
2. **How** the new test differs: "a larger size, a more sensitive primary measure, or another
   corpus".
3. **Whether** the new test pools the old data: "unless the amendment says so before its data". The
   amendment is written after the first test's data are known.

See S2.

### 9.5 New findings

#### S1. High. Repeated new tests until a verdict are optional stopping, and the error across the series is not controlled

- **Location.** Rule 5 ("Unclear at the final look ... A new test is designed"); rule 6 ("If the two
  disagree ... rule 5's next step applies"); section 6 ("this total is not an upper bound").
- **Problem.**
  1. Each new test spends a fresh 5% on a false claim. A column that runs k tests until one gives a
     verdict has an error rate near 1 − 0.95^k for a false Gain or Loss, not 5%.
  2. Take a true difference near the margin X. Then each test is close to a coin toss between a
     verdict and Unclear. Which verdict the column gets depends on when the series stops, not on the
     data.
  3. Replication repeats this. A disagreement keeps the column open, and a new test follows. The
     first pair that agrees closes the column. Earlier pairs that disagreed are reported, but they do
     not count. That is a selection of the agreeing pair.
- **Effect.** The verdicts that close columns are not at the stated error rate. The direction of the
  error is not fixed: it favors whichever outcome the data reach first. With S2, it can favor the
  outcome that someone wants.
- **Fix.** Control the error over the whole series of one column, not only inside one test:
  1. Use an anytime-valid method for each column: a confidence sequence (for example a mixture
     martingale or an e-value method). It stays valid when data are added without limit, and when a
     reader looks at any time. This fits the owner's rule, which has no upper bound. Or:
  2. Keep group sequential tests, but treat the series as one test: always pool every earlier
     test's data by a fixed rule, and spend one total alpha across the series with a spending
     function that has no end (for example a geometric spending of the remaining alpha).
  3. For replication, read the verdict on all agreeing and disagreeing data together, by the same
     rule. Do not keep only the pair that agrees.

  Write the method into rule 5 now, before any data.

#### S2. High. The next test of a column is chosen and designed after its data

- **Location.** Rule 5 ("a larger size ..., a more sensitive primary measure, or another corpus";
  "does not pool them unless the amendment says so before its data"); D7 ("spend per test").
- **Problem.** Three choices are made after the interval of the first test is known: whether the
  column gets a new test, which change the new test makes, and whether old data are pooled. "Before
  its data" means before the new test's data, not before the old test's data. A change of the primary
  measure after the data is also a change of what the column measures, and the new measure may stand
  in for the claim.
- **Effect.** Each choice can lean the final verdict in the direction that the person who makes it
  prefers. This can be toward V or against V. The owner's rule on equal burden is not kept.
- **Fix.**
  1. Write the next-test rule now, the same for every column: the same primary measure, a size
     computed from the pooled variance only, and pooling by the fixed rule of S1.
  2. Name now, per column, the one more sensitive measure or the other corpus that a new test may
     use, if any.
  3. Fund new tests per campaign, for every Unclear column of that campaign together, in a fixed
     order, without regard to the direction of each interval. Or let a session that does not see the
     interval direction propose the spend.

#### S3. Medium. The sequential design is not fitted to the four outcomes and to the count rules

- **Location.** Rule 5 ("Sizes, by a group sequential design"); section 6.
- **Problem.**
  - The plan does not say that each look reads a repeated confidence interval at the adjusted level
    (for two looks at half and full information, O'Brien-Fleming gives about z = 2.80 at the first
    look and 1.98 at the final look).
  - It does not say which outcomes may stop a test at the first look. Stopping early for "No
    measured gain" needs an equivalence bound (two one-sided tests) with its own spending.
  - It does not say how information is counted under the cluster bootstrap. The natural count is
    tasks (20 of 40).
  - The count rules of hr3 (rank 3, T16, T17, T20) are not interval rules, so alpha spending does
    not apply to them.
  - The stated half-widths in section 6 (11 and 8 points) are for unadjusted intervals. At the first
    look, the adjusted interval is wider, about 16 points before the clustering of R5.
- **Effect.** Two readers can apply the looks differently. The stated sizes give fewer verdicts than
  the plan suggests.
- **Fix.** State the boundaries, the interval method and the information count for each look. Say
  which outcomes can stop early. For the count rules, either convert them to intervals by amendment,
  or say that they have a single look. Restate the half-widths at the adjusted levels.

#### S4. High. D1d selects C2's rules on being enforced by V and L, which leans against P

- **Location.** Section 3 ("C2 then runs on the rules that the checks do enforce"); section 8 (the
  lean named as "Toward a task test that plan-r3 found not worth its cost"); D1d.
- **Problem.** If the plan-r3 gate fails, C2 uses only the rules that "the checks" enforce, which
  are L's and V's. Tasks then tempt breaks of rules that V and L reject by construction. P's linter
  may not cover those rules. The result says that V and L stop breaks of rules they stop. It does
  not show how much of a team's architecture that covers. Section 8 names a lean on cost, but not
  this lean on direction.
- **Effect.** Rows 2 and 3 against P lean toward V and L. A reader can take the result as general,
  when it holds only for a selected set of rules.
- **Fix.**
  1. Select the rules that any arm's check enforces, P's linter included (the union), and tempt
     each one.
  2. Report the result per rule and per arm, with the share of T1's rules that the set covers.
  3. Name the lean in section 8 as "toward V and L against P, by the selection of rules", beside
     the cost lean.

#### S5. Medium. D1c's "second model" picks a model by P's failures, and the lean is not named

- **Location.** Section 3 ("a third fresh task writer with another approach ..., or a second model,
  or both"); D1c ("Recommendation: both").
- **Problem.** A model chosen because it breaks rules where the first did not is a selection on P's
  failures, like the pilot gate. Prompts with "a deadline" or "a larger change" also move the tasks
  away from ordinary work. Section 8 lists neither lean.
- **Effect.** It leans toward V and L against P. Results then hold for a model and tasks chosen for
  temptation.
- **Fix.** Name the second model now (the replication model of D5), before any pilot, and use it in
  all arms. Name both leans in section 8. Report the main result per model.

#### S6. Medium. The looks are "set before any data", but the sizes come from calibration

- **Location.** Rule 5 ("planned looks, set in section 6 before any data"); section 6 ("estimates, to
  be sized by calibration before any spend").
- **Problem.** Calibration runs all three arms (row 6 reads them). If its success rates set the sizes,
  the looks are set after data that show arm differences. This is the remaining part of R1.
- **Fix.** Let calibration set only the pooled variance and the ICC. Record the inputs. Fix the looks
  in an amendment before any campaign data.

#### S7. Medium. The plan does not show how often a column will need a new test

- **Location.** Section 6 ("this total is not an upper bound"); D7.
- **Problem.** With a 12-point margin and a final half-width near 8 points (more with clustering
  and adjusted levels), any true success difference between about 4 and 20 points is likely to end
  Unclear. Round 1's difference was −19.4 points, inside that band. A label test cannot grow
  without a new rule writer, and each new corpus is a new set of fresh sessions. The plan gives the
  owner no estimate of how many columns will need new tests, or at what cost.
- **Effect.** The owner decides on spend with no view of the likely total. Completion can take far
  longer than the table shows.
- **Fix.** For each column, give the band of true differences that is likely to end Unclear at the
  final look, and the expected number of tests. Do this by simulation from round 1's data. Give the
  owner a range of total cost under two or three assumed effects.

#### S8. Low. Rule 6 does not say which test repeats after a disagreement

- **Location.** Rule 6.
- **Problem.** After "Unclear across replications", the plan does not say whether the next test
  repeats the first model or corpus, the second, or both.
- **Fix.** Say that both repeat, under the method of S1, so that neither side of the disagreement is
  chosen after the data.

### 9.6 New leans

| Lean | Direction | Finding |
|---|---|---|
| A series of new tests until a verdict | Toward whichever outcome comes first | S1 |
| The next test chosen and funded after its data | Toward the outcome that the chooser prefers, either arm | S2 |
| C2's rules selected on being enforced by V and L | Toward V and L against P | S4 |
| A second model chosen for its rule breaks | Toward V and L against P | S5 |

### 9.7 What a draft 3 needs before acceptance

1. An anytime-valid method, or one alpha spent over the whole series of a column (S1).
2. A next-test rule written now, the same for every column, with funding blind to direction (S2).
3. The union of enforced rules for C2 under D1d, and the lean named (S4).
4. The four High findings of section 8.6 that are still open: R2, R3, R4. R1 is closed.

A check of these six High fixes is enough before the owner decides D1a.

## 10. Third rerun, on draft 3

Object: draft 3 of `plan-coverage.md` (491 lines, working tree, not committed), and
`experiments/poc/sim/cs-sim.py` with its two output files `r10a.txt` and `r10b.txt`. I ran the
script, and I ran my own checks of the method (section 10.8). Not read: anything under
`C:\Users\jonat\vulcan-poc`, and hidden harness material.

### 10.1 Result

- **Of the 26 open findings:** 21 are Closed, 5 are Partly closed (R5, R17, S1, S2, S7), and 0 are
  Open.
- **New findings:** High 1 (W1), Medium 6 (W2 to W7), Low 2 (W8, W9).
- **Open after this rerun,** counting each partly closed finding once, by its residual: High 1 (W1,
  the residual of S1 and R5), Medium 6 (W2 to W7; W7 is the residual of S2, W2 of S7), Low 3 (W8,
  W9, R17).
- **Verdict: not ready to accept as written.** One High remains. The method of rule 5 is sound for
  one stream of units with a constant mean. Rule 7 feeds it a stream whose mean changes by batch,
  and my simulation shows false verdicts at 17% to 31%, not at most 5%. The fix is small: change the
  form of replication, define the estimand, and simulate again. After that fix, the plan is ready for
  the owner to decide D1a, with W2 to W9 fixed or accepted as named limits.

### 10.2 Status of each open finding

| Finding | Status | Reason |
|---|---|---|
| R2 (High) | Closed | Every column has the same four outcomes with ±X. Accepted rules set X and are reported as the hr3 reading. Section 9 lists each change, T23 and T24 included |
| R3 (High) | Closed | Option A: P's linter has a default configuration from published practice, frozen with `defaults v1` before any blind rule. Option B is secondary (D14) |
| R4 (High) | Closed | The primer stays in every condition. The primary is V against V−hints |
| R5 | Partly closed | The task is the unit, so runs of one task are inside the unit. But tasks, seeds and rules of one batch share one writer, and rule 7 changes the model or corpus by batch. The method assumes a constant mean across units. See W1 |
| R6 | Closed | Batches of 36 workspaces, read by rule 5. The new netting of faults against false alarms is W5 |
| R7 | Closed | Rows 11c, 16, 17 read an outcome that exists in every arm. Rows 5c and 18 against P go to the owner as D12, with a reason |
| R8 | Closed | Section 5: C2 has warnings on and off |
| R9 | Closed | Section 3 and C2's P−linter condition |
| R10 | Closed | Rule 2 |
| R11 | Closed | Phase A' |
| R12 | Closed | T2 is history |
| R14 | Closed | Build order and 65 to 115 sessions |
| R15 | Closed | Section 4 and rule 11 |
| R16 | Closed | Section 3 |
| R17 | Partly closed | T29a still has no measure and no margin. The plan now gates C6 on them, which is a justified sequence. The finding closes when the views amendment states them |
| R18 | Closed | A request path is the primary. Its sign is W3 |
| R19 | Closed | The status paragraph names section 11 |
| R20 | Closed | Rule 9 |
| S1 (High) | Partly closed | One confidence sequence per column replaces the series of tests. It is valid for one stream with a constant mean. With the alternation of rule 7, it is not (W1) |
| S2 (High) | Partly closed | The design is fixed, every batch pools, and funding is per campaign from a blind memo. Two paths remain for choices after data: D13 and the view of open intervals (W7) |
| S3 | Closed | Rule 5 replaces the group sequential design. Count rules become interval readings |
| S4 (High) | Closed | The union of enforced rules, P's linter included; per-rule results; the lean is named |
| S5 | Closed | The second model is D5's, named before any pilot, in every arm |
| S6 | Closed | Rule 14: calibration sets only the variance and the correlation, for cost |
| S7 | Partly closed | Section 6 simulates task success at X = 12 points. The columns with small margins, which drive C1's cost, are not simulated (W2) |
| S8 | Closed | Rule 7: no batch repeats after a result |

### 10.3 Is the confidence sequence valid as used?

I checked the code of `cs-sim.py` against Waudby-Smith and Ramdas (hedged capital, theorem 3).

- **The method, for one stream.** Valid.
  - λ at unit t uses the variance estimate from units 1 to t−1, so it is predictable.
  - μ̂ and σ̂² follow the paper's plug-in rule.
  - The truncations c/m and c/(1−m) with c = 1/2, and θ = 1/2, are as the paper states.
  - The grid has 2,000 points, a step of 0.001 on the D scale. That is fine for margins of 0.05 and
    more. For ln 1.05 on a range of ±ln 4, one grid step is about 3% of the margin. That is fine,
    too.
- **Batch-end reads.** Valid. The sequence holds at every time at once, so a read at a subset of
  times (batch ends) keeps the bound. Reading at every unit would also be valid.
- **The running intersection.** Valid, under the same assumption. The script keeps a point m alive
  until its capital first crosses 1/α, which is the intersection over all past times. It reports the
  hull of the live points, which is conservative.
- **The four outcomes.** Valid. With one sequence that covers the mean at all times with
  probability 0.95, the chance that any of the three verdicts is false is at most 5%. My check at a
  true difference exactly at +X gave 1 false verdict in 200 (section 10.8, case 1).
- **Clipping.** Valid for the mean of the clipped values. It changes the estimand, not the error
  rate. The estimand is then not the mean of D. See W6.
- **Pooled alternating replication.** Not valid. See W1.
- **An empty intersection.** Not defined. See W9.

**The assumption that rule 5 does not state.** Theorem 3 needs the conditional mean of each new
unit, given all past units, to be the same value m. Rule 5 says "the true mean", but rule 7 makes
the mean change with the batch: model A, then model B, then model A. Then the capital at the pooled
mean rises in each batch whose mean lies on the other side of it. It is not a supermartingale, so
Ville's inequality does not bound its running maximum.

### 10.4 Is the simulation sound?

- **The code matches the method.** See 10.3.
- **The table reproduces in part.** I ran the two command lines that the script header names, with
  5 runs a task. The rows at true differences 0, +4, +6, +7, +9 and −22 points (base 0.8) and at
  +15, +18, +21 and −10 points (base 0.6) match the table exactly. Two rows, +10 and +13 points at
  base 0.8, are not produced by those command lines. Their command is not recorded (W8).
- **"No simulation gave a false verdict"** is true for the stated cases. But no stated case tests
  where validity can fail: a true difference at ±X, or a mean that changes by batch. I ran both
  (10.8).
- **The data model is optimistic.** Each task has one success rate p, and the other condition has
  p + δ. So the task effect cancels in D, and there is no task-by-arm interaction. With an
  interaction of standard deviation 15 points, the batches at a true 0 rose from 4 (90th percentile
  5) to 5 (7). The effect is modest for this margin.
- **The batch medians are over the runs that reached a verdict.** For +13 points, "26 (27)" is the
  median of 7 runs in 200. A reader can take it for a typical count (W8).
- **"With 10 runs a task the median falls by only 0 to 2 batches."** On the medians, yes. But those
  medians are over the runs that reached a verdict, and 10 runs a task raises that share a lot: at
  +15 points (base 0.6), Gain rose from 82 to 155 of 200; at −10 points, No measured gain rose from
  41 to 94. The 90th percentiles fell by up to 7 batches. The script also never compares designs of
  equal cost (40 tasks × 5 runs against 20 × 10). So "more tasks, not more runs" may be right, but
  the simulation does not show it (W8).

### 10.5 New findings

#### W1. High. Rule 7's alternation by batch breaks the method of rule 5, and the pooled verdict has no fixed estimand

- **Location.** Rule 7 ("batches alternate between the main model and the second model ... in a
  fixed order"; also corpora and machines); rule 5 ("covers the true mean at every time at once");
  rule 6 (each batch from a fresh writer).
- **Problem.**
  1. Theorem 3 needs one conditional mean for every unit. Under rule 7, the mean of a model A batch
     and of a model B batch differ when the models differ. That is the case that replication exists
     to find.
  2. The pooled mean then depends on where the series stops. After an odd number of batches, it
     weights the first model more. So "the true mean" that the sequence covers is not defined.
  3. My check (10.8, case 2): model A at +22 points and model B at +2 points, so the pooled mean is
     +12 points, exactly at X. Every verdict is false. The sequence gave a false "No measured gain"
     in 35 of 200 runs (17.5%) with A first, and in 61 of 200 (30.5%) with B first. The nominal
     bound is 5%. The same stream with one constant mean gave 1 in 200.
  4. The limit note "differs by model" needs secondary verdicts "of opposite sign". In case 2, model
     A alone reads Gain and model B alone reads No measured gain. Those are not of opposite sign, so
     no limit is noted, and the pooled verdict is the false one.
  5. A random shift by writer is a weaker form of the same thing. With a shift of standard deviation
     6 points per batch, the false-verdict rate at X was 5 of 200 (2.5%). That is inside 5%, but
     above the 0.5% of a constant mean. This simulation does not show a failure from writers alone.
- **Effect.** The main fix for S1, one alpha per column, does not hold when the replication model
  or corpus changes the effect. The direction of the error follows the order of the batches, which
  is fixed in advance, so it does not lean to one arm by design. But the verdict changes with an
  arbitrary choice: which model goes first.
- **Fix.**
  1. Define the estimand in rule 5: the mean of D over tasks, with the two models (or corpora, or
     machines) at equal weight.
  2. Put both replication levels inside each unit, not in alternate batches. For models: run every
     task in every condition under both models, and let the unit's D be the mean of its two model
     differences. For corpora: each batch takes units from two writers, in a random order sealed
     before the batch. For machines: run each change or request on both machines, and average.
  3. Keep the sequence of each model, corpus or machine alone as secondary. Note "differs by model"
     when the two secondary verdicts differ, not only when their signs differ.
  4. State the assumption of theorem 3 in rule 5, and run the boundary and the alternation cases of
     10.8 in `cs-sim.py`, with their results in section 6.

#### W2. Medium. Section 6 understates C1's cost, because the small margins are not simulated

- **Location.** Section 6 ("At 4 to 20 batches a campaign ... 16,000 to 78,000"); rows 1 against L
  (X = 5 points), 8 against L (X = ln 1.05), N1.
- **Problem.** The simulation uses X = 12 points only. C1 holds columns with much smaller margins.
  My checks (10.8, cases 5 and 6), with the plan's batch and the same data model:
  - X = 5 points, true 0: median 12 batches (90th percentile 18). True +2 points: 19 (43).
  - X = ln 1.05 on log tokens clipped to ±ln 4, true 0, with a per-task standard deviation of D of
    0.32 (my assumption): median 30 batches (90th percentile 42).
  C1 runs all its conditions in every batch (rule 6), so it runs until its slowest column closes.
  That is 30 to 40 or more batches, at 900 runs each: about 27,000 to 38,000 runs for C1 alone, at a
  true difference of 0. A true difference near either margin makes it longer.
- **Effect.** The owner decides D1a and D7 on a cost range that is low for the largest campaign.
- **Fix.** Simulate each margin that the plan uses (5 points, ln 1.05, ln 1.1, ln 1.15, ln 1.2,
  ln 1.25, 10 and 14 points) with a stated variance per measure. Give the cost per campaign as set
  by its slowest column. Or let a closed column leave the campaign's conditions, if no other column
  needs them, and say so in rule 6.

#### W3. Medium. Rule 5 has no sign rule, and rows 18 and 19 read a loss for V as "Gain measured"

- **Location.** Rule 5 (outcomes); row 18 ("V minus L ... Clipped to 0 to 5"); row 19 ("V minus P");
  section 4.
- **Problem.**
  1. Rule 5 says "above +X" is a Gain, but not which sign of D is good for V. Most rows define D so
     that a positive value favors V (rows 8, 13a, 15). Rows 18 and 19 do not: more bypasses in V, or
     a slower V, gives a positive D. Read as written, that is "Gain measured".
  2. Row 18 clips a difference to 0 to 5. A difference can be negative. Clipping at 0 sets every
     session where V has fewer bypasses to 0. Then the mean leans toward more bypasses for V, and V
     cannot show a gain.
  3. Row 18's unit is "per attacker session", but each session attacks one arm. A difference per
     unit needs a pairing rule, and the row gives none. Its replication, "Repeat", is not a method of
     rule 7.
- **Effect.** Two rows can report the opposite of the result. Row 18 leans against V.
- **Fix.** Add to rule 5: "D is defined so that a positive value favors the first-named condition
  (V, or the condition with the feature)". Restate rows 18 and 19 to fit. In row 18, clip each arm's
  count to 0 to 5, then take the difference. Pair sessions by their order in a schedule fixed before
  the batch. Name its replication method, or add "Repeat" to rule 7.

#### W4. Medium. "No measured gain" also names a measured loss, and a measured harm, smaller than X

- **Location.** Rule 5 (outcomes); section 4 ("Loss measured means the harm is measured"); section 6
  (the row at −10 points reads "No measured gain").
- **Problem.** The third outcome holds when the interval lies within ±X, whatever its sign. An
  interval of −11 to −8 points is a loss that excludes 0, and it reads "No measured gain". For a
  harm column, a harm of 3 to 4 points against X = 5 reads as no harm. The name says less than the
  data, and it says it in one direction.
- **Effect.** A reader of the table sees "no gain" where a loss smaller than X was measured. This
  leans toward V on harms and on losses. On gains smaller than X, it leans against V, but the name
  does not mislead there.
- **Fix.** Rename the outcome "Within margin". Report its interval beside it, and add "(excludes 0)"
  when it does. The verdict does not change.

#### W5. Medium. Row 6 nets real faults against false alarms one for one, which hr3 rank 3 did not

- **Location.** Row 6 ("+1 if the rewriter shows a real fault, −1 if it raises a false alarm ... X 5
  points"); section 9 (hr3, rank 3).
- **Problem.** Rank 3 asked for at least 10 faults in 108 and at most 1 false alarm in 108: a false
  alarm cost about ten times a fault. The new unit weighs them the same. For example, 15 faults and
  9 false alarms in 108 give +5.6 points, which can read Gain. Rank 3 refutes that. Rule 11 forbids
  netting a harm against a gain, and this unit nets inside one measure.
- **Effect.** It leans toward V (the rewriter), against an accepted rule. Section 9 lists the change
  to rank 3, but not this change of weight, and section 8 does not name the lean.
- **Fix.** Give row 6 two columns, each read by rule 5: the rate of real faults shown, and the rate
  of false alarms (a harm). Or keep one unit with the weight of rank 3 written in it. Name the
  choice in section 9.

#### W6. Medium. The log measures have no value for a run that never reaches its end point

- **Location.** Rows 8, 13a ("tokens to a correct program", "to the first passing check"), 11d
  ("turns from a rejection to a pass"); N2; rule 5 (clipping).
- **Problem.** A run that fails, or hits its budget, has no "tokens to a correct program". If such
  runs are left out of the task's mean, the arm with more failures keeps only its easy runs. Then
  its mean falls, and it looks cheaper. If a task has no successful run in one condition, its D does
  not exist. Clipping at ±ln 4 then sets the tail of the runs that do not stop, which is N3's harm,
  to a fixed cap.
- **Effect.** The direction follows the arm with more failures. Round 1's failures were in V, so it
  leans toward V on rows 8 and 13a.
- **Fix.** State, for each such measure, the value of a run that does not reach the end point (for
  example the run's total tokens, or the clip value ln 4), and of a task with no success in a
  condition. Report the count of such runs per arm. Say that the estimand is the mean of the clipped
  value.

#### W7. Medium. Two paths remain for choices after the data (the residual of S2)

- **Location.** D13 ("Decide with D7, after calibration gives the variance"); rule 6 (the memo);
  rule 5.
- **Problem.**
  1. D13 cuts conditions, and D7 is per campaign batch. So a condition can be cut after some
     batches, when its columns' intervals are known. A cut leaves those columns open. That is a
     choice of which columns get a verdict, made after the data.
  2. The memo session does not see directions. But no rule seals the running interval of an open
     column from the owner or from the findings report. The owner decides on the next batch.
- **Effect.** The error rate of reached verdicts stays valid, because the sequence allows any stop.
  But the set of columns that reach a verdict can be chosen by direction, either way.
- **Fix.** Decide D13 once per campaign, before its first batch, from calibration's pooled values
  only. Seal the running interval of each open column until it closes or its campaign stops. Report
  only "open" in the findings report until then.

#### W8. Low. The simulation record cannot reproduce two rows, and its medians are conditional

- **Location.** Section 6, the table and "What it shows"; `cs-sim.py`, header.
- **Problem.**
  1. The rows at +10 and +13 points (base 0.8) do not come from the command lines in the script
     header. The output of the 5-run table is not saved, and only the 10-run outputs (`r10a.txt`,
     `r10b.txt`) are.
  2. "Batches to a verdict" is the median of the runs that reached a verdict within 30 batches. For
     +13 points, that is 7 runs of 200.
  3. The claim on 10 runs compares those conditional medians, and not designs of equal cost.
- **Fix.** Record every command line and save every output beside the script. Report the median
  over all runs, with "more than 30" as a value. Compare 40 × 5 with 20 × 10 at equal cost before
  the claim on tasks against runs.

#### W9. Low. An empty running intersection has no outcome

- **Location.** Rule 5; `cs-sim.py` (`'err'`).
- **Problem.** The running intersection can become empty when the assumption of a constant mean
  fails. The script counts this as `err`. The plan does not say what it means.
- **Fix.** Add to rule 5: an empty intersection is not a verdict. The column stays open, and the
  record names it as a failure of the method's assumption.

### 10.6 New leans

| Lean | Direction | Finding |
|---|---|---|
| The pooled verdict follows the order of the replication batches | Toward whichever level runs first; not toward an arm by design | W1 |
| Row 18 clips a difference at 0 | Against V | W3 |
| "No measured gain" for a measured loss or harm smaller than X | Toward V on losses and harms | W4 |
| Row 6 nets faults and false alarms one for one | Toward V | W5 |
| Runs with no end point left out of log measures | Toward the arm with more failures; on round 1, toward V | W6 |
| Conditions cut, and intervals seen, between batches | Either way, by the set of columns that reach a verdict | W7 |

The leans of section 8 of the plan are named correctly. I found no lean in draft 3 that the plan
states in a direction other than its true one.

### 10.7 What acceptance needs

1. W1: replication inside each unit, a defined estimand, and the assumption written in rule 5, with
   the boundary and alternation cases simulated.
2. The owner fixes W2 to W7, or accepts each as a named limit in section 8 or section 12.

The W1 fix is a change to rules 5 and 7 and to the script. A short check of that change is enough
before the owner decides D1a.

### 10.8 The checks that I ran

**Reproduction.** `python cs-sim.py 30 20 200 0.8 0,0.05,0.08,0.11,0.14,-0.22 5` and
`python cs-sim.py 30 20 200 0.6 0,0.17,0.21,0.26,-0.10 5`. The results are in 10.4.

**My checks.** A copy of the script's `verdict` function, unchanged except for a scale R (a unit's D
in [−R, R] maps to (D + R) / (2R)). The same data model: a task's p from a beta with mean 0.8 and
ICC 0.3, 5 runs a condition, 20 tasks a batch, at most 30 batches. Seed 20261104. For cases 1 to 3,
δ is solved so that the mean clipped difference equals the stated value.

| Case | Design | True value | Result in 200 runs (100 for case 5, 50 for case 6) |
|---|---|---|---|
| 1 | One constant mean | +12 points (= X) | 1 false verdict (0.5%); 199 open |
| 2a | Batches alternate, model A first: +22, then +2 points | Pooled +12 (= X) | 35 false "No measured gain" (17.5%) |
| 2b | Batches alternate, model B first: +2, then +22 points | Pooled +12 (= X) | 61 false "No measured gain" (30.5%) |
| 3 | Each batch shifted by N(0, 6 points), a writer effect | +12 (= X) | 5 false (2.5%) |
| 4 | Task-by-arm interaction, N(0, 15 points) | 0 | No measured gain, 200; median 5 batches (90th percentile 7) |
| 5 | X = 5 points, at most 100 batches | 0; +2 points | 12 (18); 19 (43), 1 open |
| 6 | Log tokens, D ~ N(0, 0.32) clipped to ±ln 4, X = ln 1.05, at most 300 batches | 0 | 30 (42) |

Cases 2a and 2b are the evidence for W1. Case 3 does not show a failure. Case 6 rests on my assumed
standard deviation, which calibration can replace.

## 11. Fourth rerun, on draft 4

Object: draft 4 of `plan-coverage.md` (601 lines, working tree, not committed), and the rewritten
`experiments/poc/sim/cs-sim.py` with its outputs in `sim/out/`. I ran the script in a scratch copy, so
that the files in `sim/out/` did not change. I also ran my own checks, which call the script's
functions (section 11.7). Not read: anything under `C:\Users\jonat\vulcan-poc`, and hidden harness
material.

### 11.1 Result

- **Of W1 to W9 and R17:** 7 are Closed (W1, W3, W4, W5, W6, W7, W8). 3 are Partly closed (W2, W9,
  R17). None is Open.
- **The residuals of section 10:** S1 and S2 close with W1 and W7. R5's residual, the writer of a
  batch, is now Y1. S7's residual is now Y2.
- **New findings:** High 1 (Y1), Medium 1 (Y2), Low 3 (Y3 to Y5).
- **Open after this rerun,** with each partly closed finding counted once by its residual: High 1
  (Y1), Medium 1 (Y2, which holds W2's residual), Low 4 (Y3, Y4 with W9's residual, Y5, R17).
- **Verdict: not ready to accept as written.** Draft 4's fix for models is valid, and my checks
  confirm it. But the same flaw remains one level up. Each task batch comes from one fresh writer.
  If writers differ in how much their tasks favor an arm, the units of one batch share that shift,
  and the bound fails. At a writer spread of 10 points, the false-verdict rate at the margin was
  18%. The fix is a change to how tasks are written and ordered (Y1). After that fix, and a short
  check of it, the plan is ready for the owner to decide D1a.

### 11.2 A correction to section 10.8, case 2

The coordinator is right. My case 2 asked for +22 points at a base rate of 0.8. There, the largest
reachable difference is the mean of 1 − p, about +20 points. My bisection stopped at its upper end, a
δ of 0.5, which gives about +18 points. So the pooled mean in cases 2a and 2b was about +10 points,
not +12. A pooled mean of +10 points lies within ±12. So the "No measured gain" verdicts in those
cases were **correct**, not false. **The 17.5% and 30.5% of section 10.8 are not false-verdict
rates, and I withdraw them.** Cases 1, 3, 4, 5 and 6 asked for reachable values, and they stand.

**Does W1 still hold?** I checked again at reachable values, with my own runs (11.7, cases A and B):

| Design | Models (base 0.6) | False verdicts in 200 |
|---|---|---|
| Alternating batches, 20 tasks × 5 runs (draft 3's batch) | +22 and +2 | A first 0; B first 4 (2%) |
| The same | +32 and −8 | A first 2 (1%); B first 6 (3%) |
| Alternating batches, 50 tasks × 2 runs (draft 4's batch) | +22 and +2 | A first 18 (9%); B first 30 (15%) |
| The same | +20 and +4 | A first 14 (7%); B first 21 (10.5%) |
| Both models inside each task, 50 × 2 | +22 and +2 | 5 (2.5%) |

So:
- At draft 3's own batch, I could not show a failure of alternation at reachable values. Section
  10's evidence for W1 was wrong at that batch.
- At draft 4's batch, alternation fails: my runs give 7% to 15%, and the plan's give 11% to 66%. The
  bound fails when one batch is large enough to reach a verdict on one model alone.
- W1's design point holds at any batch: under alternation, the pooled mean is not defined, and the
  assumption of theorem 3 does not hold. Draft 4's in-unit design answers it.

The plan's sentence in section 6, "The conclusion does not depend on that case", is true for the
plan's own cases. It is not true that draft 3's alternation failed at draft 3's batch. I suggest that
section 6 says so, so that the record does not claim more than the runs show.

### 11.3 Status of W1 to W9 and R17

| Finding | Status | Reason |
|---|---|---|
| W1 (High) | Closed | Rule 5 defines the estimand and states the assumption. Rule 7 puts both models inside each task. My check: 2.5% false at the margin with a 20-point gap between models. The same flaw by writer is Y1 |
| W2 | Partly closed | Every margin is simulated, and the cost is set by the slowest column. But the cost after columns close leaves out the columns that keep conditions running, and two columns are not simulated (Y2) |
| W3 | Closed | I checked the sign of every row and harm against rule 5. All are signed so that a positive value favors V or the feature. Rows 18 and 19 are correct now |
| W4 | Closed | "Within margin", with the interval and "(excludes 0)" |
| W5 | Closed | Rows 6a and 6b are separate, with no netting. The cost of 6b is in Y2 |
| W6 | Closed | A run that does not reach its end point counts its total cost. The lean is named |
| W7 | Closed | Conditions are fixed before a campaign's first batch, and open intervals are sealed. One gap is in Y5 |
| W8 | Closed | One command runs every scenario. My scratch copy gave the same bytes in all four files of `sim/out/`. Medians are over all simulations. The equal-cost comparison exists, with one gap (Y3) |
| W9 | Partly closed | An empty intersection is not a verdict. But "examined before any further batch" does not say what can follow (Y4) |
| R17 | Partly closed | Unchanged. C6 waits for T29a's measure and margin |

### 11.4 Is the new method text valid?

- **Estimand, sign rule, outcomes, empty intersection:** valid, and stated before any data.
- **The assumption:** stated correctly. But "The design keeps this" is true for models and machines,
  and not for writers (Y1).
- **Both models inside each task:** valid. The unit's D is the mean of the two model differences, so
  every unit has the same mean, whatever the gap between models.
- **Both machines inside each change:** valid, by the same argument.
- **Two writers in each corpus batch, in a sealed random order:** better than one, but not enough. It
  halves the writer share of each batch. The units of one batch still share two writer effects (Y1).
- **Clipping, and the end-point values of W6:** valid for the stated estimand.
- **Dropping a closed column's conditions (rule 6):** valid. The rule is fixed now, and the
  remaining columns keep their design.
- **The sealed interval:** valid as written. It does not change the bound. It removes the choice by
  direction.

### 11.5 Is the simulation sound?

- **Code.** The CS code is the same as in draft 3, now vectorized over simulations. `solve_delta`
  now refuses a target that it cannot reach within 0.002. That is the right guard, and it would have
  caught my case 2.
- **Reproduction.** All four output files reproduce byte for byte from `python cs-sim.py`.
- **Boundary cases.** At +X, 2 of 200 false (1%). At −X, 6 of 200 (3%). Both are within 5%.
- **Equal cost.** The three designs have equal runs a condition (200, 204 and 200). The result,
  more tasks with fewer runs, is in the direction that a task-by-arm interaction would make stronger.
  But the comparison counts runs only, not the tasks that must be written (Y3).
- **The data model** still has no writer effect, and no task-by-arm interaction. Section 6 says the
  second. It does not say the first, and the first is what breaks the bound (Y1).
- **The alternation numbers** are false-verdict rates against the equal-weight estimand. Some of
  them are verdicts at batch 1, where the data hold model A alone. The table is correct, because
  draft 4's estimand is the equal-weight mean.

### 11.6 New findings

#### Y1. High. One fresh writer per task batch breaks the method's assumption, as alternation did

- **Location.** Rule 6 ("New units come from a fresh writer who gets the same brief as the first");
  rule 5 ("Its assumption ... The design keeps this"); rule 7 ("Corpus: each batch takes half its
  units from each of two fresh writers"); section 6, batches.
- **Problem.**
  1. Every task batch, of every task column, comes from one fresh writer. Columns with Model
     replication have no second writer in the batch.
  2. Writers differ. One writer's tasks can favor an arm more than another's: for example, more
     tasks that touch the architecture, or tasks that are easier in plain Go. That is a
     writer-by-arm effect on D. All units of one batch share it.
  3. Then the conditional mean of a unit, given the earlier units of its batch, is not the
     estimand. The sequence counts 50 units a batch as 50 independent pieces of information, when
     the information on the writer effect is one piece a batch.
  4. My check (11.7, case C): one writer per batch of 50 tasks, both models inside each task, a
     true population difference exactly at +X, so every verdict is false:

     | Writer spread (standard deviation of the writer's shift of D) | False verdicts in 200 |
     |---|---|
     | 0 | 3 (1.5%) |
     | 10 points | 36 (18%) |
     | 15 points | 67 (33.5%) |
     | 20 points | 102 (51%) |

     Section 10.8, case 3, gave 2.5% at 6 points with batches of 20. The failure grows with the
     spread and with the batch size.
  5. Corpus columns (two writers a batch) have the same flaw, at about half the size. Seed and rule
     columns are corpus columns.
- **Effect.** The 5% bound of rule 5 does not hold for any task, seed or rule column when writers
  differ. The false verdicts go both ways (Gain and Within margin, in case C), so they do not lean
  to one arm by design. But each verdict rests on fewer writers than its interval says, and this is
  the main fix of S1.
- **Fix.** Choose one, and write it into rules 5 to 7 before any data:
  1. **A sealed pool written first.** Before a campaign's first batch, many fresh writers (for
     example 10 or more, each with at most 5% of the pool) write the whole pool of tasks for the
     campaign's batch cap. The pool is put in an order drawn at random and sealed. Each batch takes
     the next 50 tasks. Then the units are exchangeable draws from one fixed pool, and theorem 3
     holds for the pool mean. If the pool runs out, a new pool is a new estimand, and the record
     says so.
  2. **The writer as the unit.** The unit is one writer's batch, and its D is the batch mean. This
     is valid, but much slower: it gives one unit a batch.
  3. **Many writers a batch.** Each batch has many fresh writers, with a few tasks each, in a sealed
     random order. This reduces the flaw, but does not remove it. Size it by a simulation with a
     stated writer spread.

  In each case, state the estimand as the mean over the writer population (or the pool), and add a
  writer effect to `cs-sim.py` with the result in section 6. Calibration can estimate the writer
  spread if its tasks come from two or more writers.

#### Y2. Medium. The cost after columns close leaves out columns that keep conditions, and two columns are not simulated

- **Location.** Section 6, "Cost" ("row 8 against L reads only L+gen and V (400 runs a batch)"; C2's
  "Slowest column"; "Row 6b, 2 points: not simulated"; C7).
- **Problem.**
  1. In C1, row 1 against L (and N1) has X = 5 points: 7 batches at a true 0 and 15 (90th
     percentile 30) at half the margin. It keeps L and V. N3 and N4 read P against each checked arm,
     and N4 has X = 5 points. So after batch 6, C1 runs P, L, L+gen and V: about 800 runs a batch,
     not 400. That makes C1 about 16,400 to 30,000 runs, not 13,600 to 20,400.
  2. In C2, the slowest column is not row 3 at 12 points. N4 (5 points) and N3 (10 points) read C2
     too.
  3. Row 6b is not simulated. My check (11.7, case D): 36 units a batch, both models in each unit,
     X = 2 points. A true false-alarm rate of 0 closes in 10 batches; 0.5% in 14 (90th percentile
     17); 1% in 20 (37); 2%, at the margin, does not close in 60. Rank 3's own limit, 1 in 108, is
     about 0.9%. So calibration needs about 10 to 37 batches of 216 runs, which is about 2,200 to
     8,000 runs, not 430 to 1,300.
  4. C7 (N6, 5 points) is not simulated for chains. Its cost is borrowed from task success.
- **Effect.** The total of 27,000 to 55,000 is low, by roughly 10,000 to 20,000 runs on these items
  alone. The owner decides D1a and D7 on it.
- **Fix.** Give each campaign's cost as a sum over batches of the conditions that some open column
  still reads, by its slowest column of each condition. Simulate 6b with the rate model above, and
  C7 with a stated chain model. Restate the total.

#### Y3. Low. The equal-cost comparison counts runs, not the tasks that must be written

- **Location.** Section 6, "Equal cost" and the task batch.
- **Problem.** 50 new tasks a batch, over 13 to 30 batches in C1, is 650 to 1,500 tasks, each with a
  brief, tests and a grading rule, and a share of a parity check. The comparison and the cost table
  leave this out. With Y1, the writing plan also sets the validity.
- **Fix.** Add task writing to the cost, in sessions per task, and say how many writers the pool
  needs (Y1).

#### Y4. Low. "Examined before any further batch" leaves an open choice after data

- **Location.** Rule 5, "An empty intersection".
- **Problem.** The rule does not say what the examination can change. If it can change the design,
  it is a choice after data, against rule 6.
- **Fix.** Say what follows: the column stays open, its campaign continues unchanged for the other
  columns, and any change to this column is a new column with a new estimand, by an amendment.

#### Y5. Low. Three gaps in the wording

- **Location.** Rule 6 (sealing); rule 7 (secondary sequences); row 18; rule 5 ("the same burden
  ... in each direction"); rows 6a and 6b.
- **Problem.**
  1. Rule 6 seals the running interval of an open column. It does not say that the secondary
     sequences of each model, corpus or machine are sealed too. They show the direction.
  2. Row 18: "both models inside the pair" does not say how many sessions a pair has (two or four).
  3. Rows 6a and 6b are one-sided by nature: 6a cannot read a Loss, and 6b cannot read a Gain. Rule
     5's sentence "the same burden ... in each direction" does not hold for them. That is correct for
     these measures, but the plan should say so.
- **Fix.** Seal the secondary sequences with the primary. State the sessions of a pair in row 18.
  Add a note to rows 6a and 6b.

### 11.7 The checks that I ran

**Reproduction.** I copied `cs-sim.py` to a scratch folder and ran `python cs-sim.py`. All four
files in `out/` were byte-identical to `experiments/poc/sim/out/`.

**My checks.** They import `run_sequences`, `success_units`, `alternate_units` and `solve_delta` from
the plan's script, unchanged. Seeds 20261106 and 20261107. 200 simulations a row, except case D (100).
A cap of 30 batches, except case D (60).

| Case | Design | Result |
|---|---|---|
| A | Alternation against in-unit, base 0.6, at 20 × 5 and 50 × 2, models +22/+2 and +20/+4 | Section 11.2 |
| B | Alternation, 20 × 5, models +32/−8 | A first 2 false (1%); B first 6 (3%) |
| C | One writer per batch of 50 tasks × 2 runs × 2 models, base 0.8; the writer shifts both models' δ by N(0, sd) for the whole batch; δ solved so that the population difference, over the writer spread, is exactly +12 points | sd 0: 3 false; 0.10: 36; 0.15: 67; 0.20: 102 (of 200) |
| D | Row 6b: per unit, minus the mean of two Bernoulli false alarms (one a model), 36 units a batch, X = 2 points | Rate 0: 10 (10) batches; 0.005: 14 (17); 0.01: 20 (37); 0.02: more than 60 |

Case C rests on an assumed writer spread. Its size in real tasks is not known. Calibration, or a
small pilot with two writers on the same brief, can measure it.

## 12. Fifth rerun, on draft 5

Object: draft 5 of `plan-coverage.md` (656 lines, working tree, not committed), and
`experiments/poc/sim/cs-sim.py` with its six outputs in `sim/out/`. I ran the script in a scratch
copy, and ran one check of my own (section 12.7). Not read: anything under
`C:\Users\jonat\vulcan-poc`, and hidden harness material.

### 12.1 Result

- **Of Y1 to Y5, W2, W9 and R17:** 6 are Closed (Y1, Y3, Y4, Y5, W9, and W2 with Y2 except one
  gap). 2 are Partly closed: Y2 (row 18 is not costed, Z4) and R17 (unchanged).
- **New findings:** High 0, Medium 2 (Z1, Z2), Low 2 (Z3, Z4).
- **Open after this rerun,** with each partly closed finding counted once by its residual: High 0,
  Medium 2 (Z1, Z2), Low 3 (Z3, Z4 with Y2's residual, R17).
- **Verdict: no High finding remains. The plan is ready for the owner to decide D1a,** with Z1 and
  Z2 either fixed or accepted as named limits. Z1 is the decision that matters. The 5% bound now
  covers each column's pool, and not the writers in general. At the margin, a verdict can be wrong
  for writers in general in 28% to 48% of cases, at a writer spread of 10 to 20 points. Section 12 of
  the plan names this limit, and D15 puts it to the owner. The owner should decide D15 knowing
  that number.

### 12.2 Status

| Finding | Status | Reason |
|---|---|---|
| Y1 (High) | Closed | Units are draws with replacement from a sealed pool, so theorem 3 holds for the pool's mean (12.3). The writers scenario tests it fairly (12.4). The step from the pool to all writers is now a named limit and D15. Its size is Z1 |
| Y2 | Partly closed | The cost counts the conditions that an open column still reads, and row 6b and N6 are simulated. I checked the arithmetic (12.5). Row 18 is not simulated or costed (Z4) |
| Y3 | Closed | Pool writing is in the cost: 7 task pools × 20, about 10 seed pools × 20, 9 rule writers, and the checks, about 370 sessions. With a pool, the tasks to write no longer grow with the batches |
| Y4 | Closed | After an empty intersection, the campaign goes on unchanged. A change makes a new column by amendment before its data |
| Y5 | Closed | Rule 6 seals the secondary sequences. Row 18's pair is four sessions. Rows 6a and 6b carry the one-sided note |
| W2 | Closed | Through Y2. The remaining gap is Z4 |
| W9 | Closed | Through Y4 |
| R17 | Partly closed | Unchanged. C6 waits for T29a's measure and margin |

### 12.3 Do draws with replacement from a sealed pool satisfy theorem 3 for the pool mean?

Yes, under three conditions that the plan meets or that I add.

1. **The draw does not depend on the data.** The draw seed is fixed before the first batch. So the
   next unit is a uniform draw from the pool, independent of all past units. Its expected D is the
   mean, over the pool, of each unit's expected D. That is the same value at every step, which is
   the condition of theorem 3. Met.
2. **A unit drawn again is a new independent run.** The plan says "run again with fresh runs". Then
   two draws of one task are independent given the task, and the stream is i.i.d. from the pool's
   mixture. Met for agent runs, if no state carries over between runs (no cache, no workspace kept,
   no fixed sampling seed). The plan should say this in one line (Z2). For units with a fixed
   outcome, see Z2.
3. **The pool does not change after data.** The pool is sealed before the first batch, and "a fresh
   session checks the pool against its brief". The check is valid if it reads only the brief, not a
   run in any arm. The plan does not say that the check runs nothing. Section 3's pilot gate does
   read runs (Z3).

Two notes:
- With replacement, the pool never runs out, and the sequence can go on without limit. This is the
  right choice for the owner's rule. Draws without replacement would also be valid, but they end
  when the pool is used up.
- The estimand is a finite-pool mean. As the draws grow, the interval closes on that mean, whatever
  the writers in general would give. So every column closes in the end, unless its pool mean lies
  exactly at ±X. This is correct for the stated estimand, and it is why Z1 matters.

### 12.4 Does the `writers` scenario test it fairly?

Yes.
- **Pool design.** Each simulation draws a pool of 20 writers × 20 tasks, with one writer shift per
  writer. Units are drawn uniformly with replacement. Each draw has new binomial runs. The truth is
  the pool's mean of the expected difference, after clipping. That is exactly the estimand of rule 5.
- **One-writer design.** Each batch is one writer's tasks. The truth is the population difference,
  set exactly at +X, so every verdict is false. That is the right target for draft 4's design.
- **Results.** Against its own mean, the pool gives 0 to 2 false verdicts in 200 at every spread.
  The one-writer design gives 2% to 84%. My section 11 check, at base 0.8, gave 18% to 51% for the
  one-writer design. The plan's numbers at base 0.6 are higher, in the same direction.
- **The spread of the pool's mean** (1.7, 2.4 and 3.5 points at a writer spread of 10, 15 and 20
  points) is below spread / √20 (2.2, 3.4 and 4.5). Clipping at a base rate of 0.6 shrinks the
  shifts. That is consistent.
- **"28% to 48%"** is 57, 67 and 96 verdicts in 200. Each is right for its pool and wrong for a
  population at exactly +X. This is the size of Z1.
- **One limit.** The scenario has no task-by-arm interaction inside a writer. This does not change
  validity for the pool, and the plan names the assumption.

### 12.5 Is the recomputed cost right?

The arithmetic is right: C1 is 10,800 + 7 to 24 × 800, so 16,400 to 30,000. C2 is 9,600 + 1 to 9 ×
600, so 10,200 to 15,000. Calibration is 10 to 22 × 216, so 2,200 to 4,800. The sum of the rows is
36,600 to 69,600, which the plan gives as about 37,000 to 70,000. The rate scenario reproduces my
section 11 check of row 6b: 10, 13 (17), 22 (34) batches, and no close at 2%. Pool writing at about
370 sessions matches its parts (140 + 200 + 9 + checks).

Not counted: row 18's attacker sessions (Z4). Not simulated: C6, which the views draft owns.
Phase A' tests without agents cost no runs, but see Z2.

### 12.6 New findings

#### Z1. Medium. The bound covers each pool, and the verdict does not say how far the pool is from writers in general

- **Location.** Rule 5 (estimand); section 6 (`writers`, "28% to 48%"); section 12; D15.
- **Problem.**
  1. Rule 5's 5% is about the pool. A reader of the findings report will read "Gain measured" as a
     claim about Vulcan, not about 20 writers' tasks.
  2. At a writer spread of 10 to 20 points, a column at its margin reaches a verdict that is wrong
     for writers in general in 28% to 48% of simulations. The interval does not show this, because
     it narrows on the pool's mean without limit.
  3. D15 picks the writer count "with calibration's measured spread reported". Calibration runs
     all three arms, so if the count changes after calibration, it is a choice after data. That
     choice does not select a direction, but it should be fixed now.
- **Effect.** No lean to an arm. But the closed columns can claim more than the evidence holds, near
  their margins.
- **Fix.**
  1. Write "for this pool" in each verdict, and give the pool's writer count.
  2. Report beside each verdict a writer-level interval, with the writer as the unit: the mean of D
     for each of the 20 writers over its drawn units, and a 95% interval over the 20 writer means.
     It is secondary, and no verdict reads it. A reader then sees the step from the pool to writers
     in general.
  3. Fix D15's rule now: for example, "20 writers; if calibration's spread is above 10 points, 40
     writers, written before the campaign's first batch".

#### Z2. Medium. On columns with a fixed outcome per unit, repeated draws are copies, and the interval has no meaning

- **Location.** Rule 6 ("A unit drawn twice is run again with fresh runs"); rows 2, 5a, 5b, 9, 10a,
  10b, 11a, 11b, 11c (phase A', no agents); section 6, seeds and rules.
- **Problem.**
  1. On these columns, a unit's outcome is fixed. A check rejects a seeded case or does not. A rule
     is enforced or not. A second draw of the same unit gives the same D, so it is a copy and not a
     new independent run.
  2. The pool mean is then a known count. Evaluate each of the 200 seeds or 280 rules once, and the
     estimand is exact. A confidence sequence over draws with replacement adds sampling noise that
     the analysis made itself, and then narrows it away. The "95%" describes that noise and nothing
     else.
  3. Section 6 gives these columns 3 to 21 batches (60 to 420 draws). That cost buys nothing: one
     pass over the pool gives the answer.
  4. The only real uncertainty on these columns is the step from the pool to writers in general
     (Z1), and the plan does not bound it.
- **Effect.** No lean. But eight to ten columns would close on an interval that means less than it
  says, and they would spend batches on copies.
- **Fix.**
  1. For each column with a fixed outcome per unit, evaluate every pool unit once. Report the exact
     pool mean.
  2. Read the verdict on the writer-level interval of Z1 (2), here as the primary, because it is
     the only uncertainty that exists. Say whether it is a fixed-sample interval or a confidence
     sequence over pools.
  3. Or keep rule 5 as written, and state in section 12 that these verdicts are exact statements
     about the pool, with no sampling error.
  4. Add to rule 6: a repeated draw of an agent task carries no state from an earlier run.

#### Z3. Low. Section 3 still writes C2's tasks with one writer, and its pilot gate meets the pool undefined

- **Location.** Section 3 ("A fresh session writes the tasks"; "a second fresh session rewrites the
  tasks once. It sees only P's rate per task"); rule 6 (the pool); C2v.
- **Problem.**
  1. Section 3 has one task writer. Rule 6 has 20 writers for every task pool.
  2. The pilot reads runs. If its tasks come from the pool, then a gate failure leads to a rewrite
     after data, and the rewrite sees P's rate per task. Section 8 names this lean. But the plan does
     not say whether the whole pool is rewritten, by 20 writers again, and sealed again.
  3. C2v has the same gate.
- **Fix.** Restate section 3 with the pool: 20 writers write C2's pool, the pilot draws 10 tasks
  from it, and if the gate fails, 20 fresh writers write a new pool, which is a new estimand. Pilot
  tasks leave the pool. State the same for C2v.

#### Z4. Low. Row 18 is not simulated or costed, and one row of section 11 is out of date

- **Location.** Section 6, cost; row 18 (X 1 bypass, a batch of 3 pairs of 4 sessions); section 11,
  row W1.
- **Problem.**
  1. My check (12.7): with unnamed bypasses at a Poisson mean of 1 in both arms, row 18 closes in 13
     batches (90th percentile 15). With L at 1 and V at 0.5, it closes in 22 (29). With L at 2 and V
     at 1, a difference of exactly X, it does not close in 200. So row 18 needs about 160 to 350
     attacker sessions, or more near its margin. These are long sessions, and the cost leaves them
     out.
  2. Section 11's W1 row still says "two writers inside each batch in a sealed random order". Draft
     5 replaces that with the pool's fixed split of writers (rule 7).
- **Fix.** Add row 18 to the `rates` scenario and to the cost. Update the W1 row of section 11.

### 12.7 The checks that I ran

**Reproduction.** I copied `cs-sim.py` to a scratch folder and ran `python cs-sim.py`. All six files in `out/` were byte-identical to `experiments/poc/sim/out/`.

**My check (row 18).** It imports `run_sequences` and `summarize` from the plan's script, unchanged.
Seed 20261108, 100 simulations a row, a cap of 200 batches. A unit is a pair under two models. For
each arm and model, the count of unnamed bypasses is Poisson, clipped to 0 to 5. D is the mean over
the two models of L's count minus V's. X = 1 bypass, R = 5, 3 units a batch.

| L's mean | V's mean | Result |
|---|---|---|
| 1.0 | 1.0 | Within margin, 100; 13 (15) batches |
| 1.0 | 0.5 | Within margin, 100; 22 (29) batches |
| 2.0 | 1.0 (difference = X) | Within margin, 3; open, 97; more than 200 batches |

The Poisson means are my assumption. The holes stage of hr3 section 5 can replace them.

## 13. Sixth rerun, on draft 6

Object: draft 6 of `plan-coverage.md` (728 lines, working tree, not committed), and
`experiments/poc/sim/cs-sim.py` with its eight outputs in `sim/out/`. I ran the script in a scratch
copy, and ran one check of my own with two more seeds (section 13.8). Not read: anything under
`C:\Users\jonat\vulcan-poc`, and hidden harness material.

### 13.1 Result

- **Of Z1 to Z4 and R17:** 2 are Closed (Z3, Z4). 3 are Partly closed: Z1 (residual AA2, AA4), Z2
  (residual AA1, AA3) and R17 (unchanged).
- **New findings:** High 0, Medium 2 (AA1, AA2), Low 4 (AA3, AA4, AA5, AA6).
- **Open after this rerun,** with each partly closed finding counted once by its residuals: High 0,
  Medium 2 (AA1, AA2), Low 5 (AA3, AA4, AA5, AA6, R17).
- **Verdict: no High finding remains.** The plan is ready for the owner to decide D1a when AA1 is
  fixed, because without it four columns (rows 3 review, 4, 5d, and N5) have no unit, pool, batch or
  cost. AA2 is a matter of wording and reporting: the plan's "77% to 91%" comes from one seed, and
  my three seeds give 58% to 91%. The rest are Low.
- **Leans.** I found no new lean toward P, L or V in the draft 6 changes. The writer-level note
  and the writer unit apply the same condition in each direction. One limit is not symmetric in
  the test, not in the rule: the `fixed` scenario tests false verdicts at +X only (13.5).

### 13.2 Status

| Finding | Status | Reason |
|---|---|---|
| Z1 | Partly closed | Rule 5 writes "for this pool (N writers)" and gives the writer-level interval, as asked. D15 is a rule fixed before data, and it reads a spread, not a direction. Residuals: the interval's coverage given a verdict, and the size of the "77% to 91%" claim (AA2); D15's place in the order of work (AA4) |
| Z2 | Partly closed | Rule 6 makes the writer the unit for the ten no-agent rows. Theorem 3 holds for it (13.4). Residuals: the seeded agent columns lost their pools (AA1); T1 and the 29 existing clauses sit in a writer sequence that they were not drawn from (AA3) |
| Z3 | Closed | Section 3 has C2's pool by 20 writers, a pilot of 10 drawn tasks that leave the pool, and a new pool as a new estimand after a failed gate. C2v takes the same rules. Rule 6's pool check reads only the brief and the tasks. D15's interplay with the build order is new (AA4) |
| Z4 | Closed | `attackers` simulates row 18. The cost adds 12 to 34 batches × 12 sessions, about 150 to 410. Section 11's W1 row now names the fixed split of writers. Phase A' still calls row 18 "no agents" (AA5) |
| R17 | Partly closed | Unchanged. C6 waits for T29a's measure and margin |

### 13.3 Z1: rule 5, the writer-level interval and D15

1. **The wording.** Rule 5 states "for this pool (N writers)" on agent columns only, and no such
   words on fixed-outcome columns. That is right for each estimand.
2. **The writer-level interval is a sound secondary, but its "95%" does not hold where it is read.**
   - It is computed over the pool's 20 writers, each writer's mean over its drawn units, with
     equal weight per writer. This matches the pool's estimand, because each writer has 20 tasks.
   - It is read once, at the close. But the close is a stopping time of the sequence, and the
     sequence closes when the pool's mean is far from the margin. So, given a verdict, the writer
     means are a selected sample. The `writers` output shows it: the interval holds the population
     value in 187 to 194 of all 200 runs, but in only 44 of 57 runs with a verdict at a spread of 10
     points.
   - No verdict reads it, so the selection does not touch rule 5's bound. It does touch the note.
     See AA2.
3. **D15** is fixed before data. It reads a standard deviation of writer means, which has no sign,
   so it cannot select a direction. Its place in the order of work and its precision are open
   (AA4). Rule 14 does not yet allow it (AA4).
4. **Does `writers` support "77% to 91%"?** As one seed's result, yes: 44 of 57, 61 of 67 and 87 of
   96. "Shows the step" is counted as "holds the population value". That implies the note, so the
   count is a lower bound of the note's rate. In my check, the two counts were equal in every run.
   But the counts are small. Two other seeds give 28 of 48 (58%) and 30 of 40 (75%) at a spread of
   10 points (13.8). See AA2.

### 13.4 Z2: does theorem 3 hold for writers as units?

Yes, under the plan's own conditions:
- Each writer is a fresh session from one brief, with no state shared with other writers. So the
  writers' D values are independent, with one mean. That is the condition of theorem 3.
- Each seed is evaluated once in each condition. If the outcome is fixed, the writer's D is a
  function of the writer. If it is random (11c against P), it is a random function that is new for
  each writer. Both give independent units with one mean. Rule 6 says this correctly.
- The first read at 20 writers (rules: 10) is valid, because the sequence is valid at every time.
  Reading late only removes chances to stop.
- The check of each writer's seeds against the brief reads no arm output, so it is a fixed filter.
  The estimand is then "writers whose seeds pass the check". Rule 6 does not say what happens to a
  writer that fails the check. A replacement by a fresh writer is valid. Leaving the slot empty is
  also valid. The plan should name one (AA6).

**T1 as the first rule writer.** T1 was written for plan-r3 from round 2's brief (plan-r3 line
146), probably under an earlier model version, and plan-r3's gate reads its result before the fresh
writers exist. Its outcome does not change row 2's design, so this is not a choice after data. But
theorem 3 needs T1 to have the same mean as the fresh writers. That holds only if the fresh writers
get round 2's brief and T1's labeler procedure. The plan does not say so. Rule 7 also draws each
writer's corpus half "before it writes", which T1 cannot meet. Row 5c has the same problem with the
29 existing clauses of hr3 rank 1 step 1. See AA3.

**11c against P.** `go test -race -count=20` once per seed is a valid unit (above). Its detection
rate depends on the machine, `GOMAXPROCS` and load, and the plan does not fix them. Section 2 also
says that `go test -race` "runs in every arm", so V's and L's outcomes in that column are random
too, not only P's, as rule 6 says. See AA5.

**Is the list of fixed-outcome columns complete and correct?** For no-agent rows, yes: 2, 5a, 5b,
5c, 9, 10a, 10b, 11a, 11b and 11c. Rows 6a and 6b read a rewriter on agent workspaces, so they are
agent columns, as the plan has them. But the list leaves a gap on the agent side: draft 5's "about
10 seed pools" are gone, and the seeded agent columns C3a, C3b, C3c and N5 now have no rule. See
AA1.

### 13.5 Is the `fixed` scenario a fair test?

Mostly yes.
- **The truth.** `solve_delta_writers` sets the population mean of the clipped difference at
  exactly the target, over the writer spread. At true X every verdict is false, and the script
  counts both Gain and Within. That is the right target.
- **The reads.** `run_sequences(..., first=first)` reads at 20 writers, then every 10 (rules: at
  10). Writers at the close are 10 × batches. The plan's table converts this correctly.
- **The data model.** Each seed's outcomes in A and B are independent Bernoulli draws, given the
  seed's rate. A real check on a fixed seed is likely correlated across arms (an easy seed is caught
  in both). Independence gives more variance, so the writer counts are high, not low. This does not
  touch validity.
- **One direction only.** The scenario tests +X, not −X. The method is symmetric and base 0.5 is
  symmetric, so I expect the same at −X. But the burden of proof should be the same, and `success`
  tests both. Add −X (AA6).
- **The 40-seed result.** 40 seeds a writer save only about 10% of writers. The plan's reason is
  right: with a small variance, the plug-in bet hits the truncation of 1/2. So the cost of the
  fixed-outcome columns is set partly by the truncation, which is a choice, not a need. The plan
  could state this beside the cost.

### 13.6 Section 6 table and cost

- **Table.** Every entry matches `fixed.txt`: for example seeds at X 10 points, spread 20: 100
  (150) at 0, 210 (more than 400) at X / 2, 1 false. Row 10a: 4 / 1 / 4 false at true X. Rules: 1 /
  0 / 0.
- **Cost.** 8 seed rows × 90 to 210 = 720 to 1,680; row 10a 220 to 880; row 2 60 to 110. That is
  1,000 to 2,670 writers, plus one check per 10 writers, 100 to 267: about 1,100 to 2,940. The plan's
  "about 1,100 to 2,900" is right. The writers are per row, because the columns of one row share
  them. Row 5a at 12 points uses the 10-point figures, which overstates it a little.
- **Not counted.** A rule set needs a labeler who writes the shapes (plan-r3's T1 procedure). Row 9
  needs a labeler of test defects. At 60 to 110 rule writers, that is 60 to 110 more sessions (AA6).
- **Runs and review sessions.** The sum of the rows is 36,750 to 70,010. The plan gives about
  37,000 to 70,000. Right.

### 13.7 New findings

#### AA1. Medium. The seeded agent columns lost their pools, and rule 8 conflicts with rule 6

- **Location.** Rule 6 (two kinds of column); rule 8 ("Seeded tests and label tests use the writer
  as the unit"); rows 3 (review part, C3a), 4 (C3b), 5d (C3c); section 4, N5; section 6, batch
  table and cost ("Task pools", "Fixed-outcome writers").
- **Problem.**
  1. Draft 5 had "about 10 seed pools × 20" writers. Draft 6 replaces the seed pools with writer
     units on the no-agent rows only. C3a, C3b, C3c and N5 also read seeds, but through agent
     sessions (reviewers, or agent runs), so their outcome varies from run to run.
  2. Rule 6 makes them agent columns, with "20 writers of 20 tasks each". They have seeds, not
     tasks. No pool of seeds is named, and the cost's "7 task pools" does not include them.
  3. Rule 8 says that seeded tests use the writer as the unit. That puts them in the writer design,
     against rule 6.
  4. The batch table has a row for fixed-outcome seeds only. C3's batch of "about 300 sessions" has
     no stated number of seeds or writers.
- **Effect.** No lean. Four columns (rows 3 review, 4, 5d and N5) have no defined unit, estimand,
  batch or writer cost. As written, they cannot be run by the rules, and their bound is not
  defined.
- **Fix.**
  1. Choose one design for seeded agent columns. Either (a) a sealed pool of seeds by 20 writers,
     drawn with replacement, each draw with a fresh reviewer or agent run (the agent rule); or (b)
     the writer as the unit, with one fresh run per seed (the fixed-outcome rule, which is valid for
     random outcomes too, as rule 6 says).
  2. Restate rule 8 to match.
  3. Add the seed writers to the cost and the batch table.

#### AA2. Medium. The writer-level note is weaker than the plan says, and its rate comes from one seed

- **Location.** Rule 5 ("What a verdict states"); section 6, `writers` table and text ("77% to
  91%"); section 11, row Z1.
- **Problem.**
  1. The interval is called a 95% interval. It is read at the close, which is a stopping time, and
     given a verdict at the margin it held the population value in only 58% to 91% of runs over three
     seeds, the plan's and my two (13.8).
  2. The plan's "77% to 91%" comes from 57, 67 and 96 verdicts in one seed. Two more seeds give 58%
     and 75% at a spread of 10 points. Over the three seeds the rate is 70% at 10 points, 88% at 15
     and 89% at 20.
  3. A reader will take a verdict without the note as one that reaches writers in general. At the
     margin, a verdict that is wrong for the population and has no note occurs in 4% to 7% of runs
     (three-seed means), and in up to 10% in one seed at a spread of 10 points. That is near or above
     the 5% of rule 5.
  4. On a rate column near 0 (row 6b), most writer means are 0. The t-interval is then very
     narrow or of zero width, and the note does not appear, whatever the step.
- **Effect.** No lean: the note's condition is the same in each direction. But the note can show
  "step shown" more often than a reader expects.
- **Fix.**
  1. In rule 5, say that the interval is read at the close and that, given a verdict, its coverage
     is lower than 95%. Do not call the absence of the note evidence that the step is shown.
  2. In section 6, give the rate over several seeds, or with its binomial interval, and add the
     rate of "wrong for the population, no note".
  3. On rate columns, state that the writer-level interval is not informative near 0, or report the
     count of writers with a nonzero mean beside it.

#### AA3. Low. T1 and the 29 existing clauses are not draws from the writer sequence

- **Location.** Rule 6 ("T1 is its first writer"); row 2; row 5c (hr3 rank 1 step 1: "With the
  existing 29 clauses"); rule 7 (corpus half drawn "before it writes").
- **Problem.**
  1. T1 comes from round 2's brief, through plan-r3, probably under an earlier model version. The
     fresh writers' brief is not stated. If it differs, T1 has another mean, and theorem 3's
     condition fails at the first read, where T1 is 10% of the units.
  2. T1 was written before any corpus draw, so rule 7's split cannot apply to it.
  3. Row 5c's unit is a writer, but hr3 rank 1 step 1 uses 29 existing clauses plus one fresh
     writer. The 29 clauses have no writer from the brief. The plan does not say where they go.
- **Effect.** No lean. A small bias at the first reads, and a rule that one unit cannot meet.
- **Fix.** State that the fresh rule writers get round 2's brief and T1's labeler procedure, and
  draw T1's half by the same seed now. Or treat T1 as history, like T2. For row 5c, report the 29
  clauses as history, and start the writer sequence with fresh writers only.

#### AA4. Low. D15 meets pools that the build order writes before calibration, and rule 14 does not allow it

- **Location.** D15; section 6, build order steps 2 and 8 (C2's tasks and pilot, C2v's tasks);
  phase order (calibration in phase C); rule 14; section 3.
- **Problem.**
  1. C2's pool and pilot, and C2v's pool, are made in phase B. Calibration runs in phase C. D15 says
     that "every task pool of a campaign that has not started" gets 40 writers when the spread is
     above 10 points. C2 has not started, but its pool exists and its pilot has read runs. The plan
     does not say whether that pool is rewritten, or whether the new pool gets a new pilot.
  2. Section 3 fixes 20 writers for C2 and C2v, without D15's exception.
  3. Rule 14 says that calibration "may set only the pooled variance and the correlation". D15
     lets it set the writer count.
  4. D15 does not say at which calibration batch the spread is read. After 10 batches of 12 draws,
     each of 20 writers has about 6 units. Each writer mean then has a run and task noise of about
     the same size as the 10-point threshold, so the rule is close to a coin flip near 10 points.
- **Effect.** No lean: the spread has no sign. But an open choice remains after pilot data, and a
  rule with a weak input.
- **Fix.** Read D15's spread at a fixed calibration batch, and write C2's and C2v's pools after it;
  or state that pools written before D15 keep 20 writers. Add D15 to rule 14's exceptions. State the
  spread estimate's standard error at the read.

#### AA5. Low. Rows 15, 18 and 19 do not fit rule 6's agent-column rule, and two related gaps

- **Location.** Rule 6 ("agent runs, and the timing runs of rows 15 and 19"); rows 15, 18, 19;
  section 6, phase A' and the `margins` table; rule 6 and row 11c.
- **Problem.**
  1. Rule 6 says every agent column has a sealed pool by 20 writers of 20 tasks. Rows 15 and 19
     have changes and request paths, and row 18 has attacker sessions. No pool or writer is named
     for them, and the cost has none. Rule 5's "for this pool (N writers)" has no N there.
  2. The `margins` row for ln 1.20 (row 15) uses a batch of 50 units. The timing batch is 20. Row
     19's ln 1.1 at a batch of 20 is not simulated. So "every margin of the plan" is not quite true.
  3. Phase A' lists "row 18's holes" under "No agents". Row 18 uses attacker sessions.
  4. Row 11c against P: the race detector's result depends on the machine, `GOMAXPROCS` and load,
     which rule 6 does not fix. And `go test -race` runs in every arm, so V's and L's outcomes in
     that column are random too.
- **Effect.** No lean. Some columns have an unstated unit or a cost from the wrong batch.
- **Fix.** Name the unit of rows 15, 18 and 19 in rule 6 (for example: "a fixed set of changes or
  paths, each run again; no pool"), and drop "for this pool" there. Simulate ln 1.20 and ln 1.10 at
  a batch of 20. Move row 18 to phase C. Fix the machine and `GOMAXPROCS` for 11c.

#### AA6. Low. Small gaps in rule 6 and section 6

- **Location.** Rule 6; section 6, cost and `fixed` scenario.
- **Problem.**
  1. Rule 6 does not say what happens to a writer whose seeds fail the brief check.
  2. The cost does not count labelers: one per rule set (plan-r3's procedure), and row 9's labeler.
     That is about 60 to 110 more sessions.
  3. The `fixed` scenario tests false verdicts at +X only.
  4. Rule 6 writes "(Z3's `rlimit`)". Z3 there is the solver, but Z3 is also an audit finding,
     named in the same rule set. A reader can confuse the two.
- **Fix.** Name the replacement rule. Add the labelers to the cost. Add −X to `fixed`. Write "the Z3
  solver's `rlimit`".

### 13.8 The checks that I ran

**Reproduction.** I copied `cs-sim.py` to a scratch folder and ran `python cs-sim.py`. All eight files in `out/` were byte-identical to `experiments/poc/sim/out/`. The run took 65 minutes on this machine, with my check running beside it, not 15.

**My check (AA2).** It imports `writer_units`, `run_sequences`, `writer_interval` and
`solve_delta_writers` from the plan's script, unchanged. It runs the pool design at a population
difference of exactly +X = 12 points, base 0.6, 200 simulations, a cap of 30 batches. For each
verdict, it applies rule 5's exact note rule: the writer-level interval does not meet the verdict's
own condition. Seeds 20261109 and 20261110. The plan's own seed is 20261105, from `writers.txt`.

| Writer spread | Seed | Verdicts | Note appears | Holds the population value | Wrong for the population, no note (of 200) |
|---|---|---|---|---|---|
| 10 points | 20261105 (plan) | 57 | at least 44 (77%) | 44 | at most 13 (6.5%) |
| 10 points | 20261109 | 48 | 28 (58%) | 28 | 20 (10%) |
| 10 points | 20261110 | 40 | 30 (75%) | 30 | 10 (5%) |
| 15 points | 20261105 (plan) | 67 | at least 61 (91%) | 61 | at most 6 (3%) |
| 15 points | 20261109 | 79 | 69 (87%) | 69 | 10 (5%) |
| 15 points | 20261110 | 61 | 52 (85%) | 52 | 9 (4.5%) |
| 20 points | 20261105 (plan) | 96 | at least 87 (91%) | 87 | at most 9 (4.5%) |
| 20 points | 20261109 | 88 | 75 (85%) | 75 | 13 (6.5%) |
| 20 points | 20261110 | 103 | 93 (90%) | 93 | 10 (5%) |

In both of my seeds, "note appears" and "holds the population value" were equal in every run. So
the plan's measure is a fair proxy for the note. The spread of the rates between seeds is the
point of AA2.

**Arithmetic.** I summed the cost rows (13.6) and checked each `fixed` entry of section 6 against
`fixed.txt`, and each row 18 figure against `attackers.txt`.
