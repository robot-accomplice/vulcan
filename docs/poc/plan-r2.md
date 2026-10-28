# Round 2: Vulcan's objectives against plain Go

Status: draft for the owner's review, 2026-10-28 (second draft). Nothing here runs before the
owner accepts it. The first draft was replaced after the [desk test](desk-test-r2.md).

## 0. Why this draft

The [desk test](desk-test-r2.md) found that every check of the PoC compiler is "Same in Go": a
Go analyzer can enforce each one on the same facts. Decision 6 already says what that means:
when fewer than 4 rules need the language (set U\*), "no rule needs the language", and the
language's case is not judged by runs ([decision 6](../decisions/0006-evidence-protocol.md),
section 5). So round 2 does not try to show a rule that only Vulcan can enforce.

It tests what Vulcan was built for, **its objectives in spec §1.1 and §3**, against plain Go, and
asks of each one whether the *language* is needed or a *Go linter of Vulcan's model* gets it too.
That is decision 6's Q1 (do the rules help?) with arm R built as a generic linter of Vulcan's
model, plus Vulcan itself as a third arm, because the compiler exists. Decision 6 fixed the
consequences of each result before any of them exists: if the linter gets the gains, its row 4
applies (pivot to the rules, and continue as a Go tool).

**Stance.** The round measures; it does not seek an outcome. Every verdict, for Vulcan or against
it, is reached by the same tests, needs the same replication, and is reported with the same
effect sizes and intervals. No verdict carries a burden that another does not. Two inputs of the
design lean one way, and are named here so that a reader can weigh them: the desk test's session
was asked to argue the Go side as strongly as it could, so its finding that no check is "V only"
is a prediction, which the labels of step 3 measure (set U); and the tasks are chosen where arm P
breaks rules (section 4), which works against arms L and V.

Every measure below is computed by a script from the run's stream, final workspace, and the
graders' tests. No measure depends on a person's judgment, and every run can be graded again from
the sealed archive ([archive.md](../../experiments/poc/archive.md)).

## 1. The objectives, and what measures each one

Spec §1.1: the developer spends most time on telling the model the requirements (task 1), reading
the code (task 2), and validating and correcting it (task 3). Vulcan claims to make tasks 2 and 3
shorter. §3 lists eight failures it answers.

| # | Objective (spec) | Measure (scripted) | In the PoC? |
|---|---|---|---|
| O1 | **H-A: the rules leave review.** An enforced rule cannot be broken in a change that passes the check (§1.1) | Rule breaks in the final change (decision 6, X2), by rule set | Yes |
| O2 | **H-B: breaks are stopped before review.** The agent fixes them against the diagnostics (§1.1) | Breaks that appear at a check run and are gone by the end; check runs failed by a rule code before the first pass; recovery within two check runs (M5, M6) | Yes |
| O3 | **Task 2 is shorter: less to review** (§1.1, R6) | Review load: lines, files, and units changed (M7); the rules a reviewer must still check by hand (the rules that the arm does not enforce); reviewer cost (O3b below) | Yes |
| O3b | A reviewer finds the remaining faults with less work | An AI reviewer session per accepted change, fixed prompt and checklist: tokens and turns to a verdict, and its recall of the grader's known breaks | Yes |
| O4 | **Task 3 is shorter: fewer rounds of correction** | Review rejections before an accepted change; escaped defects (holdout failures per accepted change, X6) | Yes |
| O5 | **H-C: no gain on logic that no rule covers** (§1.1) | Class D breaks and holdout failures equal across arms (a control) | Yes |
| O6 | **R8: a new language costs little** (primer, Go syntax) | Task success and tokens on ordinary tasks (decision 6, N6) | Yes |
| O7 | **R2, R3, R4: less duplication, fewer masked errors, fewer invented APIs** | Clone lines added (a fixed clone detector), masked or dropped errors added (`errcheck` and a fixed pattern list), build failures with `undefined:` in check runs | **No:** the PoC has none of K0701, K0702, the duplication warning, or `query`. Measured in all arms, reported, not credited to Vulcan |
| O8 | **R1: small context** (`vulcan context`) | Input tokens to the first passing check after the first edit | **No:** `context` is deferred. Reported only |

O7 and O8 are measured so that a later round with those features has a baseline, and so that the
report does not credit Vulcan with them.

## 2. Arms

| Arm | What the agent gets | Check command |
|---|---|---|
| **P** (plain Go) | The Go app and `RULES.md` | `go vet ./... && go test ./...` |
| **L** (Vulcan's model as a Go linter) | The Go app, `RULES.md`, the manifest of the architecture (packages and their kinds), `vulcan-lint` (the analyzers A1 to A8 of the desk test, section 1), contracts as calls to a small `contract` package, and a primer of the model in Go terms with the same token budget as Vulcan's | `go vet ./... && vulcan-lint ./... && go test ./...` |
| **V** (Vulcan) | Vulcan in strict mode, its primer, `RULES.md` | `vulcan check && vulcan test` |

- **Arm L is generic, not written to the rules.** A fresh session writes `vulcan-lint` and the
  Go primer from the spec and the desk test **before the rules exist**, as Vulcan's checks were.
  It must pass a port of the conformance cases that the desk test maps to its analyzers. This is
  the fair rival: the same model of architecture, without a new language.
- **Same contracts.** L holds the same contracts as V, in Go, so that both start with the same
  checks in the code.
- **No arm may change its check configuration** (decision 6, section 2): a change to the
  manifest's kinds, the linter, `strict_exempt`, `strict_interfaces`, `strict_shared`, or an
  in-source suppression, fails the run unless the brief asks for it. This closes hole H2 of the
  desk test in both arms. H1 is fixed in the compiler (K0005 now covers file names).
- Everything else is the same in the three arms: model, harness, budget, briefs, README.

## 3. Questions and how each is decided

The unit is the task; differences are paired by task. Tests and bounds are those of decision 6,
section 8 (one-sided exact sign-flip test at α = 0.05 over informative tasks; one-sided 95%
t-bounds over tasks).

| Q | Question | Test |
|---|---|---|
| QA | Do enforced rules keep the agent within them? (O1) | P1(L better than P) on rule set S (rules that both L and V enforce) |
| QV | Does the language add to the linter? (O1 to O4) | V against L on S: two one-sided bounds, equivalence within 12 points of clean success |
| QR | Is review shorter? (O3, O4) | Review load and reviewer tokens, L/P and V/L: one-sided 95% bound of the per-task log ratio |
| QC | What does each approach cost? (O6) | Decision 6's N0, N1 (tokens and turns in place of working time), N2, N6: L against P, and V against L |
| — | Controls (O5) and the reported measures (O7, O8) | Reported with intervals, not tested |

**Validity gate.** Before any verdict: on tempting tasks, P breaks a targeted rule in at least
30% of its runs; and P succeeds on at least 70% of its ordinary-task runs. If not, the round is
**not informative**, as round 1 was, and no claim is made. The success condition reads P only:
it asks whether the setup is too hard, and an arm that does worse than P is a result. (The first
draft put it on every arm; the dry run of F10 showed that this hides a worse arm V as "not
informative" in a quarter of the cases.)

**Verdicts, applied in order** (written before any run):

| # | Condition | Verdict |
|---|---|---|
| 1 | Validity gate fails, the cost gate (section 6) stops the round, or a toolchain fault blocks more than 10% of an arm's runs | **Not informative** |
| 2 | QA passes, L is within the margins of P (N0, N1, N6), and V is not better than L (QV: equivalent, worse, or unclear) | **The model helps; no measured gain of the language over the linter.** Decision 6 row 4: pivot to the rules, as a Go tool |
| 3 | QA passes, and V is better than L on clean success by more than 12 points with the bound above 0, or on review (QR) by a ratio bound below 0.87 | **The model helps, and the language adds a measured gain over the linter** |
| 4 | QA passes, but L fails N0, N1, or N6 against P | **The model helps, at a cost too high.** Find the cost in the check loop (O2) and the primer (O6) |
| 5 | QA fails, and L's rate on S is not lower than P's | **No measured help from the model for this agent.** The round supports neither the linter nor the language |
| 6 | Anything else | **Unclear**: more runs on the same tasks, decided and sized before they run |

Every verdict is reported with every effect size and interval of this section, whichever row it
is. Every verdict holds only if the replication (section 5) gives the same row; a verdict that the
second model does not repeat is reported as such, and leads to no decision. Whatever the verdict,
the measures that explain it (the check loop O2, the costs O6, the reported O7 and O8) are reported
for every arm.

## 4. Rules and tasks

- **Rules:** a fresh rule-writer session with no Vulcan material, from the functional brief, at
  least 24 rules in the four classes. A labeler session then labels each rule by running the
  seeded cases: S (L and V enforce it), G (L only), U (V only; the desk test predicts it empty,
  and the labels measure it), N (neither).
  At least 8 rules in S and 4 in N, or the round stops before tasks are written.
- **Tasks:** a fresh task-writer session writes 36 tempting and 10 ordinary tasks, with briefs in
  a user's words, a larger application (Stockroom-lite with two more services), and chains of 3
  changes on the same code. Each tempting task targets at least one rule of S.
- **Calibration:** each tempting task runs 3 times in arm P only, on its own seed. A task is kept
  if P broke a targeted rule in at least 1 run. The first 24 kept tasks by a committed random
  order go to the main runs, with 8 ordinary tasks. Calibration runs are not counted. Selecting on
  P's breaks lowers P's rate in the main runs (regression to the mean), which works against L and
  V, never for them.

## 5. Repeatability

- **Pinned:** the model and provider, the Claude Code version, the toolchain, the harness, the
  hidden material, and the analysis script, each by version or hash, committed before the runs.
- **Seeds:** the schedule and the task choice come from committed seeds.
- **Scripted grading** only: tests, detectors, and the reviewer's fixed prompt. The reviewer
  session is run twice on each change; the report gives the agreement of the two.
- **Test-retest:** 10% of the main slots, chosen by seed, run a second time. The report gives the
  agreement of clean success between the two runs of a slot, so a reader can see how much one run
  can vary.
- **Replication:** the main schedule runs again on a second model (Kimi through OpenRouter). A
  verdict holds only if the second model gives the same row. The models are reported
  apart, never pooled.
- **Archive:** each round's runs are sealed, hashed, and kept off the machine before any working
  copy is deleted ([archive.md](../../experiments/poc/archive.md)). All scripts are in this
  repository, so a third party with the seal key can grade every run again.

## 6. Harness fixes before any run

| # | Fault (round 1) | Fix |
|---|---|---|
| F1 | The turn budget counted stream events | Done: one turn per message. Budget: 60 messages |
| F2 | Stopped runs have no output tokens or cost | `--include-partial-messages`, output tokens from each message's last usage, cost from the provider's price table |
| F3 | Scratch files were graded as code | A scratch directory outside the workspace (`--add-dir`); grading and diff stats read only product source |
| F4 | Runs shared `/tmp` | Each run in WSL in its own user, mount, and PID namespace (`unshare`, no root needed; `bwrap` is not installed): own `/tmp`, the user's home and `/mnt` (the Windows drives) hidden, tools read-only, modules from a read-only local proxy; the race detector on, as the plan wanted |
| F5 | A pass on unchanged code looked like progress | Measures start at the first edit |
| F6 | V runs went on after the task was done | Every preamble: "When the check passes and the brief is met, stop." |
| F7 | A changed check configuration was not caught | The grader fails a run that changes the arm's check configuration (section 2) |
| F8 | Spend per round unknown | The provider's balance before and after each batch |
| F9 | No reviewer | The reviewer session of O3b, with a fixed prompt, checklist, and model |
| F10 | Two arms only | The analysis grows to three arms and the verdicts of section 3 (`analysis2`); it reads synthetic data sets right first (`synth2`): over 200 seeds each, at the size of section 7, the expected verdict comes in at least 80% of cases, and "no effect" gives verdict 2 or 3 in at most 5% |

## 7. Size and cost

| Stage | Runs |
|---|---|
| Pilot: 2 tasks × 3 arms × 1 run, to check F1 to F10 and to price a run | 6 |
| Calibration: 36 tempting tasks × 3 runs, arm P | 108 |
| Main: 24 tempting tasks × 3 arms × 3 runs, and 8 ordinary tasks × 3 arms × 5 runs, plus 10% test-retest | about 370 |
| Replication on a second model | about 370 |

The ordinary tasks get 5 runs, as decision 6's cost pilot has: with 3, the point estimate of N6
alone fails a cost-free linter in about one case in eight (dry run of F10).

Dry run of F10 at this size (200 seeds of each synthetic world, `analysis2` test): no effect gives
verdict 2 or 3 in 0% of cases; "the model helps, the language does not" gives verdict 2 in 88%;
"V worse than L" gives verdict 2 in 92%; "the language adds" gives verdict 3 in 86%; a costly
linter gives verdict 4 in 95%; untempting tasks give verdict 1 in 100%. When V and L are equal,
verdict 3 comes in 4.5% of cases: a false "the language adds", at the 5% level.

Before the main runs, `experiments/sizing/sizing.py`, adapted to three arms and these tests, is run with the
calibration's measured rates. The main runs start only if P(verdict 2 or 3 \| a large effect) is
at least 0.8 and P(verdict 2 or 3 \| no effect) at most 0.05.

**Cost gate.** The pilot prices a run. If the main runs and the replication would cost more than
the funds left, they shrink in this order before they start (a verdict without its replication is
reported as not yet replicated): the replication waits; then 2 runs
per task and arm; then 24 tasks. No arm is ever dropped. The order is fixed now, so that the size
is not chosen after seeing data.

## 8. Work order

| Step | Who | Gate |
|---|---|---|
| 1. Harness fixes F1 to F10, the three-arm analysis, its dry run | Builder | Tests pass; synthetic data read right |
| 2. `vulcan-lint`, the manifest, the `contract` package, the Go primer | A fresh session, from the spec and the desk test, before the rules | It passes the ported conformance cases |
| 3. Rules, then labels from the seeded cases | Fresh rule-writer and labeler sessions | At least 8 rules in S and 4 in N |
| 4. Tasks, reference solutions in three arms | A fresh task-writer session | Each reference passes in all arms |
| 5. Independent audit of steps 2 to 4 | A fresh session | No open High finding |
| 6. Seal, commit hashes, seeds, and schedules | Builder | |
| 7. Pilot, calibration, sizing | Builder | F1 to F10 work; 24 tasks kept; sizing passes; cost gate |
| 8. Main runs, grading, frozen analysis, replication | Builder | Validity gate |
| 9. Results in `experiments/poc`, summary in the README, archive | Builder | |

The builder sees ids, counts, and pass or fail of the hidden material, never its text.
