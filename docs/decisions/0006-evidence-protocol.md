# Decision 6: evidence protocol

Status: accepted, 2026-10-21 (option B), as the protocol of a development plan that is itself
still proposed. Amended by [audit 3](../../experiments/audit-3.md), then by the independent audits
[4](../../experiments/audit-4.md), [5](../../experiments/audit-5.md),
[6](../../experiments/audit-6.md), and [7](../../experiments/audit-7.md). See "Amendments" at the
end. Milestone 0, item 6 of the
[development plan](../development-plan.md).

## Question

How does the project decide whether Vulcan earns continued work as a language? Which tasks,
measurements, statistics, roles, and thresholds, fixed before any result exists? What latency must
the compiler meet (decision 4, rule 4)?

## Background

- Spec §1.1 states three hypotheses. Vulcan takes architecture rules out of review (H-A). It
  rejects rule-breaking changes before review (H-B). It does not make a thorough reviewer read
  less, and it does not reduce logic defects that no rule covers (H-C).
- Experiments 1 and 2 lost their data. Test 3 had one writer per language, no compiler, and no
  grading. Audit 2 (H10) found that this cannot support the premise.
- Lessons: hidden rules work if agents cannot reach them; a model checker is not a compiler; one
  run says nothing about variation; grading must be scripted where it can be.

## Options

**A. A demonstration.** Cheap; cannot support a comparison.
**B. A preregistered comparison (chosen).**
**C. An external study.** Too slow and costly now; B can be repeated externally.

## Protocol

### 1. The questions

A language must beat its own rules delivered the way Go teams deliver rules today: static
analyzers and configuration. And a language is a large investment compared with a code generator,
so for rules that need runtime checks, it must also beat a generator. Thus:

- **Q1. Do the rules help?** Go with the rules as static analyzers (arm R), against Go with the
  rules only written down (arm P).
- **Q2. Does the language help beyond what Go tools can do?** Vulcan (arm V) against arm R, on the
  rules that neither static analysis nor a contract generator can enforce (set U*, section 5).

Q2 shows that an effect exists on tasks built to tempt agents. Its size in ordinary work is
estimated from the untargeted tasks and reported (section 8), not tested.

### 2. Arms and check commands

| Arm | What it is | Check command, named in every task brief |
|---|---|---|
| V | The application in Vulcan, strict mode on (decision 2) | `vulcan check && vulcan test` |
| R | The same application in Go, with **static analyzers and one configuration file, and nothing else**: `golang.org/x/tools/go/analysis` analyzers that read unmodified Go, and one configuration file. No code generation, no runtime checks, no toolchain changes. | `go vet ./... && staticcheck ./... && go run ./tools/rules ./... && go test ./...` |
| P | The same Go code as R, with the same rule document, without R's analyzers | `go vet ./... && staticcheck ./... && go test ./...` |
| C | Not a run arm. A **labelling copy** of arm R's application that holds, as checked comments, exactly the contracts on which V's labels rely, plus a **contract generator** that turns those comments into runtime checks. Used only to label rules (section 5). | arm R's command, after running the generator |

- On a host that has the race detector, each `go test` and `vulcan test` adds `-race`.
- The agent can run its arm's command at any time. A change goes to review only after it passes.
  The harness logs each run of the command.
- **Configuration is not the agent's to change.** A change to the arm's check configuration
  (`strict_exempt`, `strict_interfaces`, `strict_shared`, `allow_*`, the analyzers, their
  configuration file, or the Makefile), or an in-source suppression (`//lint:ignore`, `//nolint`,
  or an ignore directive of the analyzers), fails the run, unless the task brief asks for it.
- **Arm R's coverage.** Before the freeze, the Go developer writes a **coverage map**: for each
  release-1 error code and each contract kind (`requires`, `ensures`, `invariant`), whether arm R
  enforces it by an analyzer, by the configuration file, or not at all. Each "not at all" entry
  has one of three reasons: (a) static analysis of Go cannot decide it; (b) it needs a runtime
  check; (c) it was not built within the cap. The run adjudicator reviews the map and each
  reason. Arm R is built to the map within **15 working days** of the
  Go developer's time, and is not changed after the freeze.
- **Arm C's generator** is built by the Go developer within **15 working days**, before the freeze.
  It supports every release-1 tier-1 contract form (`requires`, `ensures`, `old`, and invariants,
  release-1/syntax.md); each form it does not support is listed in the coverage map with its
  reason, and the run adjudicator reviews the list. Owner's decision (2026-10-21): arm R has no
  generator. Arm C exists so that the report can say which of Vulcan's advantages a generator
  would also deliver.
- **Arm C's contracts.** In milestone 4, before the freeze, the Go developer writes every contract
  of the frozen Vulcan application as a checked comment in arm C's labelling copy. In milestone
  0.5, after labelling, the Go developer writes there every contract that the paper labels rely on
  (section 13). The run adjudicator checks the comments against the contracts one to one; a
  missing or different contract is fixed, not logged. Arms R and P never run the generator.
- **Documents:** V gets the release-1 primer and the release-1 parts of the spec. R and P get the
  same rule document, written by the Go developer, and a short guide to the application; R also
  gets the analyzers' documentation. The report gives each arm's document size.
- **The same everywhere else:** the model and its version, the harness, the token and time
  budgets, the machine, and the task briefs, word for word except for the language name.
- **Parity:** the Go developer reviews the V and Go applications for differences that would change
  a task's difficulty. Each is fixed or logged before the freeze. Arm C's contracts are not part of
  this review: they are always fixed (above).

### 3. Roles

The owner builds the Vulcan application, the compiler, and the harness, and makes no judgment that
affects a result. The owner does not see task briefs, hidden rules, labels, targets, hidden tests,
the FAQ, or run transcripts until the runs end. These roles are mandatory, filled by people other
than the owner:

| Role | Duties | Estimated load |
|---|---|---|
| Go developer | Builds or reviews the Go applications; writes the coverage map; builds arm R and arm C's generator, and writes arm C's contracts; for milestone 4, revises the coverage map, the configuration file, and arm C's contracts for Stockroom within 10 working days; writes the rule document and the review checklist; parity review; the person-checked acceptance criteria; signs the exit gates of milestones 1 and 2 | Up to 65 working days over the project |
| Rule writer | Has not read the spec. Writes the hidden rules from a functional brief (section 5). A different person for milestone 4 than for milestone 0.5. Holds no other role. | 1 to 2 days per milestone |
| Task writer | Writes all task briefs and the FAQ from the sealed rules; tags targets; keeps the private store | 3 days per milestone |
| Grader | Writes seeded cases, hidden and holdout tests, grading scripts, and a reference solution that keeps every rule, for each task and arm; labels rules | 10 days per milestone |
| Analyst | Writes the analysis script and the sizing simulation, before the freeze | 5 days |
| Run adjudicator | Exclusions, toolchain bugs, gap calls, coverage-map review, the equivalence of seeded cases across arms, arm C's contract check, replacements, milestone start dates, exit-gate and overrun signatures | 4 days per milestone |
| Operator | Runs the harness during runs, under the operator's own model account, and stores the logs in the private store; answers agents only from the sealed FAQ | Present during runs: about 1 week in milestone 0.5, 2 weeks in milestone 4 at the minimum design |
| Reviewer | Reviews and times each change submitted for review, against the sealed checklist; judges every Vulcan diagnostic in the run logs (X8). Sees the task brief of the change, and nothing else of the hidden material. Holds no other role. Blind to which Go arm a change comes from. | About 15 minutes per review: about 80 hours in total |
| Second reviewer | Re-reviews a random 30% sample of rejected changes from each arm, and reports agreement with the reviewer. Not the reviewer. | About 15 hours at the minimum design |

One person can hold several roles, except: the rule writer and the reviewer hold no other role;
and the Go developer is neither the grader nor the run adjudicator. Thus the evaluation needs **at
least four people besides the owner** (rule writer, reviewer, Go developer, and one or more for the
rest), and a second rule writer for milestone 4. **If a role is empty when the files are sealed,
the evaluation does not run** (row 1 of section 10). Each person commits to their estimated load
while recruiting, and again at the sizing gate (section 9). A person who leaves is replaced by
someone the run adjudicator approves, who has not seen material that the role must not see; a rule
writer cannot be replaced after the rules exist. The report names the holder of each role.

### 4. Tasks

- **Milestone 4:** fourteen tasks (more if section 9 requires), written by the task writer from the
  sealed rules: eight tempting changes and six ordinary ones (two features, one refactor, one
  dependency update, two failure-handling changes). A tempting change has a brief that is easiest
  to meet by breaking a hidden rule. Above fourteen, the same share of tempting tasks, rounded up.
- **Targets, preregistered:** the task writer tags each tempting task with the hidden rules it
  targets. For each of S and U\*, at least five tempting tasks, and at least half of them, target
  a rule in that set. One task can target both.
- **Reference solutions:** the grader writes, for each task and arm, a solution that keeps every
  hidden rule and passes the acceptance tests. A task without one in every arm is replaced before
  sealing.
- **Pilot:** two extra tasks test the harness. Reported, not counted.

### 5. Hidden rules, labels, and grading

- **Brief:** the rule writer gets only the functional, data-integrity, security, and resource
  criteria of the application and a neutral package diagram with no Vulcan terms, and is asked for
  at least **30 rules**. The rules are sealed before anyone labels them.
- **Seeded cases:** for each rule, the grader writes at least **two differently shaped** seeded
  changes that each break the rule and nothing else. Each shape is written once, as an
  arm-neutral description, and then translated into each arm. The run adjudicator checks that the
  translations make the same change.
- **Enforced:** an arm enforces a rule when its check command rejects every seeded case of the rule
  for that arm. One definition, in both milestones.
- **Sets:**
  - **S:** enforced by V and by R.
  - **U:** enforced by V and not by R.
  - **UC:** the rules of U that arm C enforces. A generator would deliver these.
  - **U\*:** U minus UC. Neither static analysis nor a generator enforces these.
  - **Gap:** a rule of U that the coverage map says arm R enforces, but arm R does not, or that
    arm R misses only for reason (c), "not built within the cap". The run adjudicator decides,
    and the rule is moved out of U and reported. Arm R is not patched. Thus U\* holds only rules
    that static analysis cannot decide or that need runtime checks, and that the generator does
    not cover.
  - **G** (enforced by R, not V) and **N** (neither) are reported.
- **Contract rules:** a rule of U is a contract rule when V rejects its seeded cases through a
  contract (spec §7). The grader records which.
- **Grading:** a violation is found by the grader's hidden tests, which may run the application,
  and by a blinded human grader for rules no test can observe. The method of each rule is sealed
  with it.
- **Integrity check:** immediately before and after the runs, the operator re-runs every seeded
  case on the frozen toolchain. Each must give the same result as at labelling.
- **Too few rules:** if S holds fewer than 4 rules after labelling, Q1 is inconclusive. If U\* holds
  fewer than 4, Q2 is inconclusive, and only arms R and P run, so that rows 4 and 5 (or gate 2 of
  section 13) can be decided. If both hold fewer than 4, the outcome is recorded without runs.

### 6. Order and runs

- **Order of work:** (1) the Go developer writes the coverage map, arm R, arm C, the rule document,
  and the review checklist; the analyst writes the analysis script and the simulation; the owner
  finishes the harness and the applications; (2) all of these are frozen and hashed; (3) the rule
  writer writes the rules; (4) the grader labels them and writes the tests; (5) the task writer
  writes tasks, targets, and the FAQ; (6) sealing (section 11); (7) the runs.
- **Runs:** 5 per task and arm (more if section 9 requires), from fresh workspaces, in a random
  interleaved order fixed by a committed seed.
- **Unanswered questions:** a run whose agent asks a question that the FAQ does not answer ends as
  a failure of its arm.
- **Rejected changes:** the agent gets the reviewer's comments and can revise and resubmit within
  the run's budget.
- **Toolchain bugs:** a run that a toolchain bug blocks ends as a failure of its arm; the run
  adjudicator decides. If more than 10% of an arm's runs end this way, the evaluation stops; a
  non-owner gives the owner a minimal reproduction with no hidden material; the owner fixes it
  under a new tag; the run adjudicator reviews the change; and the whole schedule runs again for
  every arm. The first schedule is reported. This can happen once; a second time ends the
  evaluation as inconclusive.
- **Exclusions:** a run counts unless its infrastructure failed, as the run adjudicator decides.

### 7. Measurements

| # | Measurement | Definition |
|---|---|---|
| X1 | Task success | A run ends in an accepted change: the acceptance tests and review pass |
| X2 | Violations | The number of hidden-rule violations in the run's final change: the change submitted for review, or, if none was, the workspace when the budget ran out. By set |
| X3 | Violations at merge | The same, in accepted changes only |
| X4 | Working time | Agent active time plus the reviewer's minutes; queue time excluded. A failed run counts as the full budget |
| X5 | Review time | The reviewer's minutes per review |
| X6 | Escaped defects | Failures of a holdout set of hidden tests in the run's final change, as for X2, for every run |
| X7 | Workarounds | Exemptions and allow-list entries in final changes (allowed only where a brief asks) |
| X8 | False positives | Vulcan diagnostics in the run logs that the reviewer judges wrong, per run |
| X9 | Agent cost | Tokens in and out, and check-command runs |
| X10 | Check latency | Section 12 |
| X11 | Runtime overhead | p99 latency and throughput of the application's benchmark |
| X12 | Maintenance | Hours spent on the compiler, runtime, analyzers, and generator |

### 8. Statistics

- **Unit:** the task. A task's rate for an arm is the mean of the measurement over that arm's runs
  of the task. A comparison uses the paired per-task difference.
- **Which tasks:** a violation test on a set uses only the tempting tasks that target that set.
  The bounds N0 to N2 use all tasks.
- **Test:** a one-sided exact sign-flip permutation test over the tasks at α = 0.05, reported with a
  one-sided 95% bound found by inverting the test. A task is **informative** when its paired
  difference is not zero; tasks with a zero difference are left out of the test, as in a sign
  test. A test needs at least 5 informative tasks.
- **Minimum evidence:** a violation test needs at least 4 rules in its set, 5 targeted tasks, and
  10 violations in the two arms together. Below that, it is inconclusive.
- **Inconclusive counts as fail**, for every test and bound, unless a row says otherwise.
- **P1(X better than Y) on a set:** the test that X has fewer violations than Y passes, and the mean
  rate of X is at most half that of Y.
- **Q1 passes** when P1(R better than P) on S passes.
- **Q2 passes** when P1(V better than R) on U\* passes.
- **Non-inferiority bounds, V against R:**
  - **N0. Task success (X1):** the one-sided 95% bound of (R minus V) is at most the success margin
    (section 9; 10 percentage points unless revised there). N0 is computed twice: with the
    reviewer's verdicts, and with every rejection that the second reviewer disputed reversed. If
    the two disagree, N0 is inconclusive.
  - **N1. Working time (X4):** the one-sided 95% bound of the mean per-task log ratio V/R is at
    most the log of the time margin (section 9; 1.10 unless revised there).
  - **N2. Escaped defects (X6):** the one-sided 95% bound of (V minus R) is at most 0.1 per run.
  - **N3. False positives (X8):** over tasks, the one-sided 95% sign-flip bound of the mean per-task
    share of V's runs with a false positive is at most 0.1.
  - **N4. Runtime overhead (X11):** ten alternating benchmark runs of each arm; the one-sided 95%
    bound (permutation over runs) of V's p99 latency is at most 1.10 times R's, and of V's
    throughput at least 0.90 times R's.
  - **N5. Check latency:** the targets of section 12.
- **Reported, not tested:** violations on untargeted tasks, by set (the estimate of practical
  benefit); X3, X5, X7, X9, X12; sets UC, G, N, and gaps; the second reviewer's agreement.

### 9. Sizing gate (end of milestone 0.5)

Before milestone 1 starts, the analyst runs the frozen sizing simulation, with the task-to-task and
run-to-run variation measured in milestone 0.5. It estimates the probability of each outcome row of
section 10 under four fixed scenarios:

| Scenario | Violations per run on targeted tasks (R, V) | V against R: success, time |
|---|---|---|
| Z: no effect | 0.4, 0.4 | equal |
| M: moderate effect | 0.4, 0.2 | equal |
| L: large effect | 0.4, 0.05 | equal |
| L+cost: large effect, real cost | 0.4, 0.05 | 15 points lower, 20% slower |

**Fixed parameters,** frozen with the simulation: base task success 0.7 in every arm; working
time log-normal with the same median in every arm, its spread taken from milestone 0.5; escaped
defects 0.1 per run in every arm; Vulcan false positives in 2% of runs; N4 and N5 assumed to pass;
task-to-task variation of violation rates gamma-distributed with the coefficient of variation
measured in milestone 0.5 (1.0 if it measured less). These parameters exist for sizing only; the
evaluation measures the real values.

The design passes when P(Continue \| Z) ≤ 0.05, P(Continue \| L) ≥ 0.8, and
P(Continue \| L+cost) ≤ 0.2. The analyst searches, in this order: tasks from 14 to 24, then runs per
task from 5 to 10. If no design passes, the success and time margins may be widened once, to at most
12 points and 1.15, which stays below the cost of scenario L+cost, with a written reason signed by
the run adjudicator. If no design passes then, the project ends as **stop, not judged**, before any
compiler work. The report publishes every probability, including P(Stop \| L) and P(Pivot \| L).

**Re-commitment.** The simulation also gives the volunteer hours of each role and the tokens at the
chosen design. Gate 3 passes only when each role holder commits in writing to the sized load, and
the budget is committed. Milestone 4's time box is scaled by the ratio of the sized number of runs
to 210, rounded up to whole weeks.

### 10. The decision rule (milestone 4)

Apply the rows in order. The first row that matches decides.

| # | Condition | Outcome |
|---|---|---|
| 1 | A role is empty, the recruiting deadline passed, the budget would be exceeded, the sizing gate failed, the integrity check failed, or the evaluation ended inconclusive under section 6 | **Stop: not judged.** |
| 2 | Q2 passes, and N0 to N5 pass | **Continue the language.** |
| 3 | Q2 passes, N0 and N1 pass, and a bound of N2 to N5 fails | **Narrow, as a hold.** A new decision changes or drops the feature behind the failing bound. Then milestone 4 runs once more, at the sized design, with a committed budget, a new rule writer, new rules, and new tasks. Its result is decided by rows 1 to 5 again, except that row 3 then counts as row 5. Milestone 5 and new features wait. |
| 4 | Q1 passes, and R's success and working time are within the margins of P's, as point estimates | **Pivot to the rules.** Continue as a Go tool: arm R's analyzers, plus arm C's generator when Q2 would pass on U (with UC included). |
| 5 | Any other result | **Stop.** |

Every result matches exactly one row. Row 4 uses point estimates, not bounds, because a pivot
keeps Go and commits to no new language; rows 2 and 3 commit to a language and need bounds. A
second time-box overrun of a milestone ends the project as **stopped by schedule** (development
plan).

### 11. Sealing and isolation

- **Private store:** kept by the task writer, outside this repository and out of the owner's reach.
  It holds the task briefs, rules, labels, seeded cases, hidden and holdout tests, FAQ, seed, and
  grading scripts until milestone 4 ends (milestone 0.5's material too). The repository holds only
  their SHA-256 hashes, and the hashes of the frozen toolchain, applications, harness, analysis
  script, and simulation.
- **Outside timestamp:** the commit of the hashes gets a signed tag pushed to a third party, and the
  hashes are published where the owner cannot change them later.
- **Workspaces:** each run gets a fresh workspace with only its arm's application and documents.
- **Release:** after milestone 4, the sealed contents are copied into `experiments/m0.5/` and
  `experiments/m4/`, and checked against the hashes.

### 12. Latency targets (for decision 4)

On a CI runner of type `linux/amd64` with 8 cores and 16 GB of memory:

- **One-edit check:** `vulcan check` after a one-line edit in one leaf, p95 over 100 edits at most
  1 second.
- **Full check:** `vulcan check` with an empty cache, median of 5 runs at most 10 seconds.
- **Build:** `vulcan build` after a one-line edit, p95 over 100 edits at most 2 seconds more than
  `go build` of arm R.

Milestone 2 measures these on a synthetic application of fixed size, committed with the compiler;
milestone 3 again on the application. A miss before milestone 4 starts the work of decision 4,
rule 4. A miss in milestone 4 fails N5.

### 13. Milestone 0.5

Before any compiler work, on a small application (for example the test 3 shop, rebuilt with a
database), built by the Go developer:

- **Arms:** R and P run; C labels. There is no compiler, so "V enforces a rule" means: the grader
  and the Go developer write down the release-1 diagnostic (spec Appendix E) or the release-1
  contract (spec §7) that would reject each seeded case. The run adjudicator reviews each call.
  The Go developer writes each such contract as a checked comment in arm C's labelling copy
  (section 2). The sets of section 5 then follow by the same definitions. **Limit:** these are
  paper calls; they assume a correct compiler and tests that reach each contract. The report says
  so.
- **Rules, tasks, roles, order, sealing:** as in sections 3 to 6 and 11, with its own rule writer.
  Three quarters of the tasks are tempting, with the targets of section 4. Five runs per task and
  arm. Its budget is committed while recruiting.
- **Size, before the runs:** the analyst runs the frozen simulation with its fixed parameters
  (section 9) for arms R and P under scenario L (P 0.4 violations per run, R 0.05). The number of
  tasks is the smallest from 8 to 16 that gives P(Q1 passes \| L) ≥ 0.8. If 16 tasks do not reach
  it, runs per task rise up to 10. If that is still not enough, the outcome is **stop, not
  judged**.

The gates, in order:

- **Gate 2 first (is a language needed?):** count the U\* violations at review in both arms, over
  all tasks. If U\* holds at least 4 rules and they show at least 5 violations, go to gate 3,
  whatever Q1 shows: a language may be needed even if the statically checkable rules did not
  matter.
- **Gate 1 (do the rules help?):** otherwise, if Q1 passes, **pivot to the rules** (arm R's
  analyzers, plus arm C's generator for the rules of UC), when R's success and working time are
  within the margins of P's as point estimates; if they are not, **stop**. If Q1 fails and its
  bound shows that R removes less than half of P's violations, **stop**: enforcing the rules does
  not change outcomes. If Q1 fails in any other way, or is inconclusive, **stop, not judged**.
- **Gate 3 (can milestone 4 answer?):** the sizing gate of section 9, with re-commitment.
- Otherwise continue to milestone 1.

### 14. Records

The report states each threshold, each result with its bound and p-value, the sizing results, the
budget, the roles and their holders, every excluded or unanswered run, the coverage map, the sets
with their sizes, the caveat that arm R has no generator, and the outcome row. The analysis script
runs on the released raw files and produces every number.

## Open points

- **The thresholds are judgments,** fixed now. Only section 9 may change the two margins, once.
- **One model.** All results bind to one model version. The report says so.

## Amendments

| Source | Change |
|---|---|
| Audit 3 | Three arms, a pivot outcome, independent rule writer, named roles, inconclusive counts as fail, a frozen toolchain. |
| Audit 4 | Per-arm labels, exact permutation tests on paired per-task differences, minimum counts, mandatory roles, one local check command per arm, a functional brief, a complete decision table, sealing and isolation, a cheap gate on whether a language is needed. |
| Audit 5 | Arm R limited to static analyzers and one configuration file (owner's decision); seeded-case labels; U graded by runtime tests and a blinded grader; a non-owner task writer with preregistered targets; the owner kept from hidden material; freeze before rules; sizing by tasks. |
| Audit 6, B1 | Milestone 0.5 counts release-1 contracts as Vulcan enforcement; one "enforced" definition; arm C labels the rules a generator would deliver, and Q2 uses U\* without them. |
| Audit 6, B2 | A sizing gate at the end of milestone 0.5, before any compiler work, with four fixed scenarios, a search over tasks and runs, one bounded widening of the margins, and "stop, not judged" if no design passes. |
| Audit 6, H1 | Defined check commands for every arm; P has no dependency linter. |
| Audit 6, H2 | A coverage map before the freeze; no patch of arm R after it; the run adjudicator decides gaps. |
| Audit 6, H3 | The reviewer holds no other role, sees only the brief, and follows a sealed checklist. |
| Audit 6, H4 | Rates, the run without review, task selection, and the split above fourteen are defined; the analysis script, simulation, and harness are frozen and hashed; a non-owner analyst. |
| Audit 6, H5 | Q2 is stated as existence; untargeted-task rates are reported as the estimate of benefit. |
| Audit 6, H6 | "Narrow" is a hold; the codegen question is answered by arm C before the runs, not after. |
| Audit 6, H7 | Estimated load per role, commitments while recruiting, budgets for both milestones; follow-on tasks removed. |
| Audit 7, B1 | "Informative" defined; milestone 0.5 sized before its runs (P(Q1 passes \| L) ≥ 0.8); gate 2 checked first, so evidence for a language goes to gate 3 whatever Q1 shows; a plain "stop" only when the bound rules out a meaningful effect. |
| Audit 7, B2 | Arm C's labelling copy holds, one to one, the contracts that V's labels rely on, written by the Go developer and checked by the run adjudicator; the generator covers the tier-1 forms or lists each gap. |
| Audit 7, H1 to H6 | Seeded shapes written once and translated; the Go developer is neither grader nor adjudicator; re-commitment and a scaled box at the sizing gate; the second reviewer audits rejections, and N0 must hold either way; a defined re-measurement for "narrow"; cap gaps leave U\*; sizing target 0.8 and a widening that stops below L+cost. |
| Audit 7, M1 to M10 | Fixed sizing parameters and target shares; Stockroom's arm R and C revision assigned; R and P run when only U\* is too small; reference solutions; X6 for every run; N3 over tasks; replacements; in-source suppression; the operator's account and log storage. |
| Audit 6, M1 to M11 | Integrity check with two seeded shapes; milestone-2 hole cap and re-plan cap (plan); a 30-rule brief; milestone 0.5's extension and "not judged"; material kept private until milestone 4 ends and a second rule writer; the bug-fix procedure; row 1 aligned; one definition; X8 judged from logs; every duty assigned. |
