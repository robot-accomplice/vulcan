# Decision 6: evidence protocol

Status: accepted, 2026-10-21 (option B). Amended three times on the same day, by
[audit 3](../../experiments/audit-3.md), [audit 4](../../experiments/audit-4.md), and
[audit 5](../../experiments/audit-5.md). See "Amendments" at the end. Milestone 0, item 6 of the
[development plan](../development-plan.md). Answers [audit 2](../../experiments/audit-2.md), H10,
and gives decision 4 its latency target.

## Question

How does the project decide whether Vulcan earns continued work as a language? Which tasks,
measurements, statistics, roles, and thresholds, fixed before any result exists? What latency must
the compiler meet (decision 4, rule 4)?

## Current state

- Spec §1.1 states three hypotheses. Vulcan takes architecture rules out of review (H-A). It
  rejects rule-breaking changes before review (H-B). It does not make a thorough reviewer read
  less, and it does not reduce logic defects that no rule covers (H-C).
- Experiments 1 and 2 lost their data. Test 3 had one writer for each language, no compiler, and
  no finished grading. Audit 2 (H10) found that this cannot support the premise.

## Lessons from test 3

- **Hidden rules work**, if the agents cannot reach them (section 11).
- **A model checker is not a compiler.** Milestone 4 uses only the real compiler.
- **One run says nothing about variation.**
- **Grading must be scripted where it can be.**

## Options

**A. A demonstration.** Cheap; cannot support a comparison.

**B. A preregistered comparison (chosen).** Fix the tasks, measurements, statistics, roles, and
decision rule here, seal them before the first run, and report every run.

**C. An external study.** More credible, but slow and costly now. B can be repeated externally.

## Protocol

### 1. The question

A language must beat its own rules delivered the way Go teams already deliver rules: as static
analyzers and configuration. So the protocol asks two questions:

- **T1. Do the rules help?** Go with the rules as analyzers, against Go without them.
- **T2. Does the language help beyond the analyzers?** Vulcan against Go with the analyzers, on
  the rules that the analyzers cannot check.

What counts as "Go with the rules" decides the answer, so section 2 fixes it now.

### 2. Three arms

- **Vulcan arm (V):** the milestone-3 application in Vulcan, strict mode on (decision 2).
- **Go-rules arm (R):** the same application in Go, with **static analyzers and one configuration
  file, and nothing else**: `golang.org/x/tools/go/analysis` analyzers that read the unmodified Go
  source, and one configuration file (for example, which packages are gateways and entry points).
  No code generation, no comment-driven generator, no runtime checks, no changes to the Go
  toolchain. Inside these means, arm R covers every rule of Vulcan that it can: the Go developer
  writes or reviews the analyzers, within a cap of **15 working days**, and signs that no rule of
  Vulcan that static analysis of Go can check is left out.
- **Go-plain arm (P):** the same Go code as R, with the same rule document as R, but none of R's
  analyzers: only `go vet`, `staticcheck`, and a dependency linter whose sealed configuration says
  what it covers. P is "Go with the rules written down, but not enforced".
- **One local check command per arm**, named in every task brief: `vulcan check && vulcan test`
  for V, and `make check` for R and P. The agent can run it at any time. A change goes to review
  only after it passes. The harness logs each run of the command.
- **Documents:** V gets the release-1 primer and the release-1 parts of the spec (no deferred
  constructs). R and P get the same rule document, written by the Go developer, and a short guide
  to the application. R also gets the analyzers' documentation. The report gives each arm's
  document size.
- **The same everywhere else:** the model and its version, the harness, the token and time
  budgets, the machine, and the task briefs, word for word except for the language name.
- **Parity:** the Go developer reviews the V application as well as the Go one, and lists any
  difference in function or quality that would change a task's difficulty. Each item is fixed or
  logged before sealing.

**Caveat, stated in every report.** Arm R has no code generation, so rules that need generated
or runtime checks (contracts, for example) fall to Vulcan by definition. A Go code generator might
capture part of that benefit. Section 10 limits what such a result can decide.

### 3. Roles

The owner builds the Vulcan arm, the compiler, and the harness, and makes **no** judgment that
affects a result. The owner does not see the hidden rules, the hidden tests, the task briefs, or the
labels until the runs end. These roles are mandatory, and each is filled by someone other than the
owner:

| Role | Duties |
|---|---|
| Go developer | Builds the Go application, or reviews it and writes a timed list of changes that are applied before sealing. Writes or reviews arm R's analyzers within the cap, and signs their coverage. Reviews the V application for parity. |
| Rule writer | Has not read the spec. Writes the hidden rules from a functional brief (section 5). Holds no other role. |
| Task writer | Writes all task briefs from the sealed rules, and assigns each tempting task its target rules (section 4). Keeps the private store (section 11). |
| Grader | Writes the hidden tests, the seeded-violation cases, and the grading scripts (section 5). |
| Run adjudicator | Decides run exclusions and whether a toolchain bug blocks a run. Signs the exit gates of milestones 0.5, 3, and 4, and each time-box overrun. |
| Operator | Answers agents' questions during runs, only from an FAQ sealed before the runs. |
| Reviewer | Reviews accepted changes, times them, and judges false positives. Does not know which Go arm a change comes from. |
| Second reviewer | Audits a random 30% sample for escaped defects. Not the reviewer. |

One person can hold several roles, except: the rule writer holds no other role, and the reviewer
and the second reviewer are different people. Thus the evaluation needs **at least three people
besides the owner**. **If any role is empty when the files are sealed, the evaluation does not
run.** The report names the person in each role.

### 4. Tasks

- **Milestone 4:** fourteen tasks, written by the task writer from the sealed rules: eight
  tempting changes, two new features, one boundary refactor, one dependency update, and two
  failure-handling changes. A tempting change has a brief that is easiest to meet by breaking a
  hidden rule.
- **Targets, preregistered.** The task writer tags each tempting task with the hidden rules it
  targets. At least **five** tempting tasks target a rule in set S, and at least **five** target a
  rule in set U (one task can target both). If the sealed rules do not allow this, the test that
  lacks five tasks is inconclusive, which counts as fail.
- **Follow-on tasks:** each tempting task has a follow-on task. Each follow-on run starts, by a
  committed random draw, from a base that contains a seeded violation of the targeted rule or from
  a clean base. M14 compares them. Reported only.
- **Pilot:** two extra tasks, run first to test the harness. Reported and not counted.

### 5. Hidden rules, labels, and grading

- **The brief.** The rule writer gets only the functional, data-integrity, security, and resource
  criteria of the application under test and a neutral package diagram with no Vulcan terms. For
  milestone 4, these are F, D, S, and R in `docs/release-1/acceptance.md`; for milestone 0.5, the
  same kinds of criteria for its small application, written by the grader.
- **Sealing before labels.** The rules are sealed before anyone labels them, and the toolchain is
  frozen before the rules leave the rule writer (section 6).
- **Seeded cases.** For each hidden rule, the grader writes at least one seeded-violation change
  for each arm: a small change that breaks the rule and nothing else.
- **Labels, by test.** A rule is *enforced* by an arm when that arm's check command rejects every
  seeded case of the rule. The grader runs the seeded cases against the frozen toolchain. The sets:
  - **S, shared:** enforced by V and by R.
  - **U, Vulcan-only:** enforced by V and not by R. For each U rule, the Go developer writes why
    static analysis of unmodified Go cannot check it, or what an analyzer for it would cost. A U
    rule that an analyzer could check within the 15-day cap is a gap in arm R: arm R is fixed and
    the toolchain re-frozen before sealing.
  - **G, Go-only:** enforced by R and not by V. Reported.
  - **N, neither.** Reported.

  A U rule is a **contract rule** when V rejects its seeded cases through a contract (spec §7),
  not through an architecture rule (spec §5.4). The grader records which, for each U rule.
- **Grading.** A violation is found by the grader's hidden tests, which may run the application
  (HTTP calls, fault injection, data checks), and by a blinded human grader for rules no test can
  observe. These means are outside arm R's static analyzers, so a rule in U can still be graded.
  The grading method of each rule is sealed with the rule.

### 6. Runs

- **Freeze first.** The order is: freeze and hash the compiler tag, the analyzers, the runtime,
  and both applications; then the rule writer writes the rules; then the grader labels and writes
  the tests; then the task writer writes the tasks; then sealing (section 11).
- **Fresh state:** each run starts from a fresh workspace (section 11), with a fresh agent context.
- **Runs per task:** 5 for each arm. Section 8 sizes the number of tasks.
- **Order:** the runs of all arms interleave, in a random order fixed by a committed seed.
- **Unanswered questions:** a run whose agent asks a question that the FAQ does not answer ends as
  a failure of its arm.
- **A frozen toolchain.** The main runs use the frozen tag, and record the model build of each run.
  A run that a toolchain bug blocks ends as a failure of its arm; the run adjudicator decides. If
  more than 10% of an arm's runs end this way, the evaluation stops, the bug is fixed under a new
  tag, and the whole interleaved schedule runs again for every arm. This can happen once; a second
  time ends the evaluation as inconclusive.
- **Exclusions:** a run counts unless its infrastructure failed, as the run adjudicator decides.
  Each exclusion is listed.

### 7. Measurements

| # | Measurement | Definition |
|---|---|---|
| M1 | Task success | A run ends in an accepted change: the acceptance tests and review pass |
| M2 | Violations at review | Hidden-rule violations in the change submitted for review, by rule set (S, U, G, N), found by the grading of section 5 |
| M3 | Violations at merge | The same, in the accepted change |
| M4 | Violations stopped before review | Violations that a run of the arm's check command reported and that were gone at review, divided by all that it reported; defined only when at least 10 were reported in that arm |
| M5 | Working time | Agent active time plus the reviewer's measured minutes; queue time excluded. A failed run counts as the full budget |
| M6 | Review time | Minutes that the reviewer spends on each review |
| M7 | Escaped logic defects | Failures of a holdout set of hidden tests that acceptance does not use, plus defects that the second reviewer finds in the 30% sample, per run |
| M8 | False positives | Diagnostics that the reviewer judges wrong, with the reason |
| M9 | Workarounds | Exemptions and allow-list entries that a run adds |
| M10 | Agent cost | Tokens in and out, and check-command runs |
| M11 | Check latency | Section 9 |
| M12 | Runtime overhead | p99 latency and throughput of the application's benchmark |
| M13 | Maintenance | Hours spent on the compiler, runtime, and analyzers during the evaluation |
| M14 | Cost of a violation | Follow-on runs from a violating base against a clean base: working time and escaped defects. Reported only |

Acceptance does not require the hidden-rule checks, so a violation can reach merge (M3), and a
merged violation is visible to M14.

### 8. Statistics and thresholds

**Unit.** The task is the unit. For each task and arm, a rate is the mean over that arm's runs of
the task. A comparison of two arms uses the paired per-task difference.

**Test.** Each comparison is a one-sided exact sign-flip permutation test over the tasks (all
2^k sign patterns) at α = 0.05, and the result is reported with a **one-sided 95% bound** found by
inverting the test. With k tasks, the smallest p-value is 1/2^k, so a comparison needs at least 5
informative tasks.

**Minimum evidence.** A violation comparison is decided only when its set holds at least 4 rules,
at least 5 targeted tasks (section 4), and at least 10 violation events at review in the two arms
together. Below that, it is inconclusive.

**Inconclusive counts as fail**, for every test and every bound.

**P1(X better than Y) on a set:** the test of "X has fewer violations at review than Y" passes, and
the mean rate of X is at most half the mean rate of Y.

**Tests:**

- **T1 passes** when P1(R better than P) on S passes.
- **T2 passes** when P1(V better than R) on U passes.

**Label check.** Before the runs, every seeded case of every S rule is rejected by both V and R,
and of every U rule by V. This is how "both arms enforce S" is shown; there is no run-based test of
it, because neither arm can show violations of S at review.

**Non-inferiority bounds, V against R:**

- **N0. Task success (M1):** the one-sided 95% bound of (R rate minus V rate) is at most 10
  percentage points.
- **N1. Working time (M5):** the one-sided 95% bound of the mean per-task log ratio V/R is at most
  log 1.10.
- **N2. Escaped defects (M7):** the one-sided 95% bound of (V minus R) is at most 0.1 defects per
  run.
- **N3. False positives (M8), V only:** the one-sided 95% exact binomial upper bound of the rate
  per accepted change is at most 0.1.
- **N4. Runtime overhead (M12):** ten alternating benchmark runs of each arm; the one-sided 95%
  bound (permutation over runs) of V's p99 latency is at most 1.10 times R's, and of V's throughput
  at least 0.90 times R's.
- **N5. Check latency:** the targets of section 9, each on its stated statistic.

M6, M9, M10, M13, M14, and the sets G and N are reported and decide nothing.

**Sizing before sealing.** Before the files are sealed, a simulation over plausible task-to-task
variation (taken from the pilot and from milestone 0.5) gives, for the planned number of tasks,
the probability of each outcome row of section 10 under three scenarios: no effect, a moderate
effect, and a large effect. The report publishes these numbers. If P(Continue | large effect) is
below 0.5, the number of tasks is raised (to at most 24) before sealing, with the same split of
tempting tasks. The same simulation gives the expected volunteer hours and tokens; the budget for
both is committed before sealing, and an evaluation that would exceed it does not start.

### 9. Latency targets (for decision 4)

On a CI runner of type `linux/amd64` with 8 cores and 16 GB of memory:

- **One-edit check:** `vulcan check` after a one-line edit in one leaf, p95 over 100 edits at most
  1 second.
- **Full check:** `vulcan check` with an empty cache, median of 5 runs at most 10 seconds.
- **Build:** `vulcan build` after a one-line edit, p95 over 100 edits at most 2 seconds more than
  `go build` of arm R after the same edit.

Milestone 2 measures these on a synthetic application of fixed size, committed with the compiler.
Milestone 3 measures them again on the application. A miss before milestone 4 starts the work of
decision 4, rule 4. A miss in milestone 4 fails N5.

### 10. The decision rule

Apply the rows in order. The first row that matches decides.

| # | Condition | Outcome |
|---|---|---|
| 1 | A mandatory role is empty, the label check failed, or the evaluation ended as inconclusive under section 6 | **Stop: not judged.** |
| 2 | T2 passes, N0 to N5 pass, and no more than half of U's violation events come from contract rules | **Continue the language.** |
| 3 | T2 passes, N0 and N1 pass, and either a bound of N2 to N5 fails, or more than half of U's violation events come from contract rules | **Narrow.** Keep the language features that enforce the U rules with violations in arm R; a new decision drops or redesigns the rest. When contract rules carried T2, the report says that a Go code generator might deliver the same, and names that comparison as the next step. |
| 4 | T1 passes, and R meets N0 and N1 against P | **Pivot to the rules.** Continue as a Go tool (arm R's analyzers), not as a language. |
| 5 | Any other result | **Stop.** |

Every result matches exactly one row. Rows 2 and 3 need N0 and N1: a language that costs more
than 10% in success or time does not continue, whatever T2 shows. A second time-box overrun of a
milestone (development plan) ends the project as **stopped by schedule**.

### 11. Sealing and isolation

- **A private store,** kept by the task writer, outside this repository and out of the owner's
  reach, holds the task briefs, hidden rules, labels, seeded cases, hidden and holdout tests, FAQ,
  seed, and grading scripts until the runs end. The repository holds only their SHA-256 hashes.
- **An outside timestamp:** the commit of the hashes gets a signed tag pushed to a third party, and
  the hashes are published where the owner cannot change them later.
- **Agent workspaces:** each run gets a fresh workspace with only its arm's application and
  documents, and no access to this repository's `experiments/`, the private store, or other arms.
- **Release after the runs:** the sealed contents are copied into `experiments/m0.5/` or
  `experiments/m4/`, and checked against the hashes.

### 12. The reduced run of milestone 0.5

Before any compiler work, the development plan runs this protocol in a reduced form on a small
application (for example the test 3 shop, rebuilt with a database):

- **Arms:** R and P. There is no compiler, so set U is labelled from the spec: a rule is in U when
  arm R's analyzers reject none of its seeded cases, and the grader and the Go developer find a
  diagnostic in spec Appendix E, of a construct that release 1 supports, that would reject them.
  Each such call is recorded with its reason. Set S is the rules that arm R enforces.
- **Rules:** from the rule writer, by the same process as section 5.
- **Tasks:** eight, six of them tempting, written and targeted by the task writer: at least five
  target S rules, and as many as the rules allow target U rules. At least 5 runs per task and arm.
  All roles of section 3, the freeze order of section 6, and the sealing of section 11.
- **Gate 1 (do the rules help?):** if T1 fails, **stop**.
- **Gate 2 (is a language needed?):** if set U holds fewer than 4 rules, or the U rules show fewer
  than 5 violation events at review in the two arms together, **pivot to the rules** without
  building the compiler, provided that R meets N0 and N1 against P. If R fails N0 or N1, **stop**.
- Otherwise continue to milestone 1. Arm R's analyzers become arm R of milestone 4.
- Inconclusive counts as fail here too.

### 13. Records

The report states each threshold, each result with its bound and p-value, the sizing simulation,
the budget, the roles and their holders, every excluded or unanswered run, the caveat of
section 2, and the outcome row. The analysis script runs on the released raw files and produces
every number.

## Open points

- **The thresholds are judgments.** They are fixed now; changing them after the first main run is
  not allowed.
- **Finding people.** At least three people besides the owner. Without them, row 1 applies.
- **One model.** All results bind to one model version. The report says so.

## Amendments

| Source | Change |
|---|---|
| Audit 3 | Three arms; a pivot outcome; hidden rules from a person who has not read the spec; named roles; inconclusive counts as fail; a frozen toolchain. |
| Audit 4 | Rules labelled per arm; exact permutation tests on paired per-task differences; minimum counts; mandatory roles with no escape clause; one local check command per arm; a functional brief; a complete decision table; a private store, an outside timestamp, isolated workspaces; gate 2 of milestone 0.5. |
| Audit 5, B1 | T3 replaced by a label check with seeded cases; "narrow" needs N0 and N1. |
| Audit 5, B2 | Arm R limited to static analyzers and one configuration file, with a 15-day cap (owner's decision, 2026-10-21); sets labelled by seeded cases; U graded by runtime tests and a blinded grader; milestone 0.5 uses the same rule process. |
| Audit 5, H1 | A non-owner task writer writes all tasks and preregisters targets: at least 5 per tested set. Acceptance no longer requires the hidden-rule checks, so merge violations are visible. |
| Audit 5, H2 | Sizing by tasks through simulation, with published outcome probabilities; defined tests for N3 to N5; one-sided 95% bounds. |
| Audit 5, H3 | The owner never sees hidden material before the runs end; the toolchain and applications are frozen first; a non-owner grader; reviewer and second reviewer differ; the reviewer is blind to the Go arm. |
| Audit 5, H4 | A budget for volunteer hours and tokens, committed before sealing. Milestone 0.5's box is set in the plan. |
| Audit 5, M1 to M5 | Milestone 0.5 terms defined; each arm's documents stated; unanswered runs count as failures; parity review of the V application; a holdout test set for M7. |
