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
