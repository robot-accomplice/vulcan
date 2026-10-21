# Decision 6: evidence protocol

Status: accepted, 2026-10-21 (option B), as the protocol of the development plan, which is
accepted. Amended by [audit 3](../../experiments/audit-3.md), then by the independent audits
[4](../../experiments/audit-4.md), [5](../../experiments/audit-5.md),
[6](../../experiments/audit-6.md), [7](../../experiments/audit-7.md),
[8](../../experiments/audit-8.md), and [9](../../experiments/audit-9.md). See "Amendments" at the
end. Milestone 0, item 6 of the [development plan](../development-plan.md).

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
  rules only written down (arm P). Tested in milestone 0.5 (section 13).
- **Q2. Does the language help beyond what Go tools can do?** Vulcan (arm V) against arm R, on the
  rules that neither static analysis nor a contract generator can enforce (set U\*, section 5).
  Tested in milestone 4.

Q2 shows that an effect exists on tasks built to tempt agents. Its size in ordinary work is
estimated from the untargeted tasks and reported (section 8), not tested.

### 2. Arms and check commands

| Arm | What it is | Runs in | Check command, named in every task brief |
|---|---|---|---|
| V | The application in Vulcan, strict mode on (decision 2) | Milestone 4 | `vulcan check && vulcan test` |
| R | The same application in Go, with **static analyzers and one configuration file, and nothing else**: `golang.org/x/tools/go/analysis` analyzers that read unmodified Go, and one configuration file. No code generation, no runtime checks, no toolchain changes. | Milestones 0.5 and 4 | `go vet ./... && staticcheck ./... && go run ./tools/rules ./... && go test ./...` |
| P | The same Go code as R, with the same rule document, without R's analyzers | Milestone 0.5 | `go vet ./... && staticcheck ./... && go test ./...` |
| C | Not a run arm. A **labelling copy** of arm R's application that holds, as checked comments, exactly the contracts on which V's labels rely, plus a **contract generator** that turns those comments into runtime checks. Used only to label rules (section 5). | Labelling only | arm R's command, after running the generator |

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
  check; (c) it was not built within the cap. The run adjudicator reviews the map and each reason;
  a reason (a) must name what the analysis would need to know and why Go source does not show it.
  Arm R is built to the map within **15 working days** of the Go developer's time, and is not
  changed after the freeze.
- **Arm C's generator** is built by the Go developer within **15 working days**, before the freeze.
  It supports every release-1 tier-1 contract form (`requires`, `ensures`, `old`, and invariants,
  release-1/syntax.md); each form it does not support is listed in the coverage map with its reason,
  and the run adjudicator reviews the list. The map also lists each contract semantic of release 1
  (exact arithmetic, `old`, the check after deferred calls, the unwinding check, port contracts that
  bind fakes, and the check after a call that receives the struct, spec §7) as built or not built,
  with reasons (a) to (c) as for arm R. The generator must pass the conformance cases of
  release-1/conformance.md for every semantic it lists as built. Owner's decision (2026-10-21): arm
  R has no generator. Arm C exists so that the report can say which of Vulcan's advantages a
  generator would also deliver.
- **Arm C's contracts.** In milestone 4, before the freeze, the Go developer writes every contract
  of the frozen Vulcan application as a checked comment in arm C's labelling copy. In milestone
  0.5, after labelling, the Go developer writes there every contract that the paper labels rely on
  (section 13). The run adjudicator checks the comments against the contracts one to one; a
  missing or different contract is fixed, not logged. Arms R and P never run the generator.
- **Tool bugs.** The owner fixes bugs in the compiler and runtime; the Go developer fixes bugs in
  the analyzers and the generator, but only under section 6's toolchain-bug procedure, never to
  change what arm R covers.
- **Documents:** V gets the release-1 primer and the release-1 parts of the spec. R and P get the
  same rule document, written by the Go developer, and a short guide to the application; R also
  gets the analyzers' documentation. The report gives each arm's document size.
- **The same everywhere else:** the model and its version, the harness, the token and time
  budgets, the machine, and the task briefs, word for word except for the language name.
- **Parity:** the Go developer reviews the V and Go applications for differences that would change
  a task's difficulty. Each is fixed or logged before the freeze. Arm C's contracts are not part of
  this review: they are always fixed (above).

### 3. Roles

The owner builds the Vulcan application, the compiler, the harness, and milestone 0.5's Go service
(which the Go developer reviews and signs for parity), and makes no judgment that affects a result.
The owner does not see task briefs, hidden rules, labels, targets, hidden tests, the FAQ, or run
transcripts until the runs end. These roles are mandatory, filled by people other than the owner:

| Role | Duties | Estimated load |
|---|---|---|
| Go developer | Reviews milestone 0.5's Go service; builds the Go version of Stockroom; writes the coverage map; builds arm R and arm C's generator, and writes arm C's contracts; for milestone 4, revises the coverage map, the configuration file, and arm C's contracts for Stockroom within 10 working days; writes the rule document and the review checklist; parity review; the person-checked acceptance criteria; signs the exit gates of milestones 1 and 2 | Up to 80 working days over the project, at least 2.5 days a week during milestone 0.5 |
| Rule writer | Has not read the spec. Writes the hidden rules from a functional brief (section 5). A different person for each rule set (milestone 0.5, milestone 4, and a re-run under row 3). Holds no other role. | 1 to 2 days per rule set |
| Task writer | Writes all task briefs and the FAQ from the sealed rules; tags targets; keeps the private store | 3 days per milestone |
| Grader | Writes seeded cases, hidden and holdout tests, grading scripts, and a reference solution that keeps every rule, for each task and arm; labels rules; checks acceptance criterion D5 on the Go version | 15 days per milestone |
| Blind grader | Judges, from the diff and the rule list, the rules that no hidden test can observe and every final change that does not build (section 7). Sees the diff, so the language is visible, but not run logs, other judgments, or which Go arm a diff comes from. Not the grader, the Go developer, or the run adjudicator. | About 2 days per milestone |
| Analyst | Writes the analysis script and the frozen sizing simulation, before the freeze. Not the grader. | 5 days |
| Run adjudicator | Exclusions, toolchain bugs, gap calls, coverage-map review, the equivalence of seeded cases across arms, arm C's contract check, replacements, milestone start dates, exit-gate and overrun signatures | 4 days per milestone |
| Operator | Runs the harness during runs, under the operator's own model account, and stores the logs in the private store; answers agents only from the sealed FAQ, which may point a Vulcan agent to a section of the primer | Present during runs: about 2 weeks in milestone 0.5, 4 weeks in milestone 4 |
| Reviewer | Reviews and times each change submitted for review, against the sealed checklist; judges every Vulcan diagnostic in the run logs (X8). Sees the task brief of the change, and nothing else of the hidden material. Blind to which Go arm a change comes from in milestone 0.5. In milestone 4 the role is held by two people, each assigned changes at random within each arm. | About 15 minutes per review: about 180 hours in total at the designs of section 9 |
| Second reviewer | Re-reviews a random 30% sample of accepted and of rejected changes from each arm, and re-judges a random 30% sample of X8 calls; reports agreement with the reviewer. Same blinding as the reviewer. | About 55 hours |

Holding several roles: the rule writer, the reviewer, and the second reviewer hold no other role;
the Go developer is neither the grader, the run adjudicator, nor the blind grader; the grader is
neither the run adjudicator, the analyst, nor the blind grader; the blind grader is not the run
adjudicator. Thus the evaluation needs **at least seven people besides the owner** (rule writer,
reviewer, second reviewer, Go developer, grader, run adjudicator, blind grader), a second reviewer
person in milestone 4, and another rule writer for each later rule set. **If a role is empty when
the files are sealed, the evaluation does not run** (row 1 of section 10). Each person commits to
their estimated load while recruiting, and again at the sizing gate (section 9). A person who leaves
is replaced by someone the run adjudicator approves, who has not seen material that the role must
not see; a rule writer cannot be replaced after the rules exist. The report names the holder of each
role.

### 4. Tasks

- **Sizes:** fixed by section 9: in milestone 0.5, **16 tasks**, 12 of them tempting; in milestone
  4, the design that the sizing gate chooses, three quarters of the tasks tempting. The rest are
  ordinary tasks of the kinds the plan names (features, a refactor, a dependency update,
  failure-handling changes).
- **Targets, preregistered and binding:** the task writer tags each tempting task with the hidden
  rules it targets. In milestone 4, **every** tempting task targets at least one rule of U\*, and
  at least one rule of S where the rules allow. In milestone 0.5, **every** tempting task targets
  at least one rule of S, and **at least 8** of the 12 also target a rule of U\*. If the rules do
  not allow 8, gate 2 of section 13 does not pass. A tempting change
  has a brief that is easiest to meet by breaking a targeted rule.
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
  - **Generator gap:** a rule of U that arm C misses only because a contract semantic was not
    built within the generator's cap (reason (c) in the coverage map). The run adjudicator decides;
    the rule is moved out of U\* and reported.
  - **G** (enforced by R, not V) and **N** (neither) are reported.
- **Contract rules:** a rule of U is a contract rule when V rejects its seeded cases through a
  contract (spec §7). The grader records which.
- **Grading:** a violation is found by the grader's hidden tests, which may run the application,
  and by the blind grader for rules that no test can observe. The method of each rule is sealed
  with it.
- **Integrity check:** immediately before and after the runs, the operator re-runs every seeded
  case on the frozen toolchain. Each must give the same result as at labelling.
- **Too few rules:** in milestone 4, if U\* holds fewer than 4 rules after labelling, no rule needs
  the language, and the outcome is recorded without runs: row 4 of section 10 if its conditions
  hold, otherwise **stop, not judged**.

### 6. Order and runs

- **Order of work:** (1) the Go developer writes the coverage map, arm R, arm C, the rule document,
  and the review checklist; the analyst writes the analysis script and the simulation; the owner
  finishes the harness and the applications; (2) all of these are frozen and hashed; (3) the rule
  writer writes the rules; (4) the grader labels them and writes the tests; (5) the task writer
  writes tasks, targets, and the FAQ, at the sizes of section 9; (6) sealing (section 11); (7) the
  runs.
- **Runs:** at the sizes of section 9, from fresh workspaces, in a random interleaved order fixed by
  a committed seed.
- **Unanswered questions:** a run whose agent asks a question that the FAQ does not answer ends as
  a failure of its arm.
- **Rejected changes:** the agent gets the reviewer's comments and can revise and resubmit within
  the run's budget.
- **Toolchain bugs:** a run that a toolchain bug blocks ends as a failure of its arm; the run
  adjudicator decides. If more than 10% of an arm's runs end this way, the evaluation stops; a
  non-owner gives the person who fixes the bug (section 2) a minimal reproduction with no hidden
  material; the bug is fixed under a new tag; the run adjudicator reviews the change; and the whole
  schedule runs again for every arm. The first schedule is reported. This can happen once; a
  second time ends the evaluation as inconclusive.
- **Exclusions:** a run counts unless its infrastructure failed, as the run adjudicator decides.

### 7. Measurements

| # | Measurement | Definition |
|---|---|---|
| X1 | Task success | A run ends in an accepted change: the acceptance tests and review pass. Reported |
| X1c | Clean success | A run ends in an accepted change that keeps every hidden rule its task targets, as the grader's tests and the blind grader find. On an ordinary task, the same as X1. Used by N0 and N1 in every arm |
| X2 | Violations | The number of hidden-rule violations in the run's final change: the change submitted for review, or, if none was, the workspace when the budget ran out. By set. A final change that does not build is judged by the blind grader from its diff, in every arm alike |
| X3 | Violations at merge | The same, in accepted changes only |
| X4 | Working time | Agent active time plus the reviewer's minutes, up to the first clean accepted change; queue time excluded. A run without a clean success counts as the full budget |
| X5 | Review time | The reviewer's minutes per review |
| X6 | Escaped defects | Failures of a holdout set of hidden tests, per accepted change |
| X7 | Workarounds | Exemptions and allow-list entries in final changes (allowed only where a brief asks) |
| X8 | False positives | Vulcan diagnostics in the run logs that the reviewer judges wrong, per run |
| X9 | Agent cost | Tokens in and out, and check-command runs |
| X10 | Check latency | Section 12 |
| X11 | Runtime overhead | p99 latency and throughput of the application's benchmark |
| X12 | Maintenance | Hours spent on the compiler, runtime, analyzers, and generator |

The report gives, for each arm, the number of runs whose final change does not build.

### 8. Statistics

- **Unit:** the task. A task's rate for an arm is the mean of the measurement over that arm's runs
  of the task. A comparison uses the paired per-task difference.
- **Violation tests:** a one-sided exact sign-flip permutation test over the tasks at α = 0.05. A
  violation test on a set uses only the tempting tasks that target that set. A task is
  **informative** when its paired difference is not zero; tasks with a zero difference are left
  out, as in a sign test. A test needs at least 5 informative tasks, at least 4 rules in its set,
  and at least 10 violations in the two arms together; otherwise it is inconclusive.
- **Bounds:** every one-sided 95% bound below is a t-bound over tasks, all tasks included: the mean
  of the per-task values plus t(0.95, k−1) times their standard deviation over the square root of
  the number of tasks k. The sizing simulation uses the same bounds.
- **Inconclusive counts as fail**, for every test and bound, unless a row says otherwise.
- **P1(X better than Y) on a set:** the violation test that X has fewer violations than Y passes,
  and the mean rate of X is at most half that of Y.
- **Half-bound:** X keeps more than half of Y's violations when the one-sided 95% upper bound of the
  per-task value (0.5·Y − X), over the tasks of the violation test, is below zero.
- **Q1 passes** (milestone 0.5) when P1(R better than P) on S passes.
- **Q2 passes** (milestone 4) when P1(V better than R) on U\* passes.
- **Non-inferiority bounds, V against R (milestone 4):**
  - **N0. Clean success (X1c):** the bound of (R minus V) is at most **12 percentage points**. N0 is
    computed with the reviewer's verdicts, and again with each verdict that the second reviewer
    disputed replaced by the second reviewer's; a reversed rejection counts as acceptance at the
    first disputed submission, with the time up to that point, and a reversed acceptance counts as
    a run without a success, at the full budget. N0 passes only if both pass. If the two reviewers
    disagree on more than 20% of the sampled verdicts, N0 is inconclusive.
  - **N1. Working time (X4):** the bound of the per-task log ratio V/R is at most log **1.15**. It
    is computed with and without the reviewer's minutes; N1 passes only if both pass.
  - **N2. Escaped defects (X6):** over the tasks where both arms have an accepted change, the bound
    of (V minus R) of the mean failures per accepted change is at most 0.1.
  - **N3. False positives (X8):** the bound of the per-task share of V's runs with a false positive
    is at most 0.1, computed with the reviewer's calls and again with the second reviewer's calls
    where they differ; N3 passes only if both pass.
  - **N4. Runtime overhead (X11):** ten alternating benchmark runs of each arm; the one-sided 95%
    bound (a permutation over runs, the one bound that is not a t-bound over tasks) of V's p99
    latency is at most 1.10 times R's, and of V's
    throughput at least 0.90 times R's.
  - **N5. Check latency:** the targets of section 12.
- **Reported, not tested:** violations on untargeted tasks, by set (the estimate of practical
  benefit); X3, X5, X7, X9, X12; sets UC, G, N, and gaps; the second reviewer's agreement.

### 9. Sizing

**Before recruiting** (done): `experiments/sizing/sizing.py` simulates both milestones with the
tests and bounds of section 8, clean success for N0 and N1, escaped defects per accepted change,
and the parameters listed in its docstring (task heterogeneity CV 1.0, run time spread 0.5, a run
without a clean success costs three times the median, base success 0.7, a small task × arm
interaction, defects 0.1 per accepted change, Vulcan false positives in 2% of runs). Its results
are in `experiments/sizing/results.md`. It does not model reviewer disagreement, N4, or N5, each of
which would lower P(Continue) somewhat.

| Scenario | Violations per run on targeted tasks (R, V) | V against R |
|---|---|---|
| Z: no effect | r, r | equal |
| L: large effect | r, r/8 | equal |
| L+coupled | r, r/8 | a tempted V run (as often as R violates) is 1.3 times slower and fails 5 points more |
| L+inter | r, r/8 | task × arm interaction sd 0.10 (success) and 0.20 (log time) |
| L+half | r, r/8 | a net cost of half the margins: clean success 6 points lower, 7% slower |
| L+cost | r, r/8 | a net cost of 15 points and 20% |

Net costs are stated against R's clean success, so they are costs in the quantity that N0 and N1
measure. Here r is 0.4 before recruiting, and at the sizing gate the lower of 0.4 and R's measured
U\* rate on targeted tasks in milestone 0.5. Results before recruiting:

- **Milestone 0.5:** 16 tasks, 12 tempting, 5 runs per task and arm: P(Q1 passes \| L) about 0.93.
  Gate 2's rate test with 8 tasks targeting U\*: a true rate of 0.4 passes 91% of the time, a true
  rate of 0.1 passes 9%.
- **Milestone 4:** at 28 tasks (21 tempting) × 10 runs, P(Continue) is about 0.86 to 0.89 under L,
  0.84 to 0.88 under L+coupled, 0.84 to 0.86 under L+inter, about 0.2 under L+half, and 0 under Z
  and L+cost, for r of 0.4 and 0.2.

**Accepted limit:** a language with a real net cost of half the margins continues only about one
time in five. The margins are the costs the owner accepts, and a bound must exclude them; a smaller
real cost cannot be told apart from one at the margin at this size.

**Sizing gate (end of milestone 0.5).** The analyst runs the frozen simulation (at least 2,000
simulations per chosen cell) with r as above, the task heterogeneity measured in milestone 0.5 (the
coefficient of variation of R's and P's per-task rates on targeted tasks, by the method of moments
with the Poisson part removed, at least 1.0), and the run time spread (the pooled within-task
standard deviation of log working time of clean runs, at least 0.5). It chooses, among 24, 28, and
32 tasks (three quarters tempting) and 8 or 10 runs, the design with the fewest total runs (fewer
tasks on a tie) that gives P(Continue \| Z) ≤ 0.05; P(Continue) ≥ 0.8 under each of L, L+coupled,
and L+inter; and P(Continue \| L+cost) ≤ 0.2. If none does, the project ends as **stop, not
judged**, before any compiler work. The report publishes every probability, including those under
L+half, and P(Stop \| L) and P(Pivot \| L).

**Re-commitment.** The simulation also gives the volunteer hours of each role and the tokens at the
chosen design. The sizing gate passes only when each role holder commits in writing to the sized
load, and the budget is committed. Milestone 4's time box is 8 weeks times the ratio of the sized
number of runs to 560 (28 tasks × 10 runs × 2 arms), rounded up to whole weeks, and never less
than 8 weeks.

**Model change.** Milestone 4 uses the model version of milestone 0.5 if it is still available. If
not, arm P runs in milestone 4 as well, Q1 is tested again there, and row 4 uses that result.

### 10. The decision rule (milestone 4)

Apply the rows in order. The first row that matches decides.

| # | Condition | Outcome |
|---|---|---|
| 1 | A role is empty, the recruiting deadline passed, the budget would be exceeded, the sizing gate failed, the integrity check failed, or the evaluation ended inconclusive under section 6 | **Stop: not judged.** |
| 2 | Q2 passes, and N0 to N5 pass | **Continue the language.** |
| 3 | Q2 passes, N0 and N1 pass, and a bound of N2 to N5 fails | **Narrow, as a hold.** A new decision changes or drops the feature behind the failing bound. Then milestone 4 runs once more, at the sized design, with a committed budget and a time box equal to milestone 4's, a new rule writer recruited for it, new rules, and new tasks. Its result is decided by rows 1 to 6 again, except that row 3 then counts as row 6. Milestone 5 and new features wait. |
| 4 | Q1 passed in milestone 0.5, and R's clean success and working time were within the margins of P's there (at most 12 points lower, at most 1.15 times slower), as point estimates | **Pivot to the rules.** Continue as a Go tool: arm R's analyzers, plus arm C's generator when Q2 would pass on U (with UC included). |
| 5 | Q2 passes, and N0 or N1 fails on its bound (not by being inconclusive) | **Stop: the language costs too much.** |
| 6 | Any other result | **Stop** when Q2's half-bound shows that V keeps more than half of R's U\* violations; otherwise **stop, not judged**. |

Every result matches exactly one row. Row 4 uses point estimates, not bounds, because a pivot
keeps Go and commits to no new language; rows 2 and 3 commit to a language and need bounds.

### 11. Sealing and isolation

- **Private store:** kept by the task writer, outside this repository and out of the owner's reach.
  It holds the task briefs, rules, labels, seeded cases, hidden and holdout tests, FAQ, seed, and
  grading scripts until milestone 4 ends (milestone 0.5's material too). The repository holds only
  their SHA-256 hashes, and the hashes of the frozen toolchain, applications, harness, analysis
  script, and simulation.
- **Outside timestamp:** the commit of the hashes gets a signed tag pushed to a third party, and the
  hashes are published where the owner cannot change them later.
- **Workspaces:** each run gets a fresh workspace with only its arm's application and documents.
- **Release:** when the project ends, or after milestone 4, the sealed contents are copied into
  `experiments/m0.5/` and `experiments/m4/`, and checked against the hashes.

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
database), built by the owner and reviewed by the Go developer:

- **Arms:** R and P run; C labels. There is no compiler, so "V enforces a rule" is a paper call:
  the grader and the Go developer write down the release-1 diagnostic (spec Appendix E) or the
  release-1 contract (spec §7) that would reject each seeded case. For a contract, the call also
  names the test of the application that reaches the contract; arm C's label for that rule uses the
  same test, so a contract that no test reaches counts in neither. The run adjudicator reviews each
  call. The Go developer writes each such contract as a checked comment in arm C's labelling copy
  (section 2). The sets of section 5 then follow by the same definitions. **Limit:** paper calls
  assume a correct compiler. The report says so.
- **Rules, tasks, roles, order, sealing:** as in sections 3 to 6 and 11, with its own rule writer,
  at the size of section 9 (16 tasks, 12 tempting, 5 runs). Its budget is committed while
  recruiting.

The gates, in order:

- **Gate 2 first (is a language needed?):** count the U\* violations (X2) in both arms, over all
  tasks. If U\* holds at least **8 rules**, they show at least **5 violations**, and R's U\*
  violation rate on the tasks that target U\* is at least **0.2 per run** (the lower rate that
  section 9 sizes for), go to gate 3, whatever Q1 shows: a language may be needed even if the
  statically checkable rules did not matter. Section 9 publishes this test's false-pass and
  false-fail rates.
- **Gate 1 (do the rules help?):** otherwise, if Q1 passes, **pivot to the rules** (arm R's
  analyzers, plus arm C's generator for the rules of UC), when R's success and working time are
  within the margins of P's as point estimates; if they are not, **stop**. If Q1 fails and its
  half-bound shows that R keeps more than half of P's violations, **stop**: enforcing the rules
  does not change outcomes. If Q1 fails in any other way, or is inconclusive, **stop, not judged**.
- **Gate 3 (can milestone 4 answer?):** the sizing gate of section 9, with re-commitment.
- Otherwise continue to milestone 1.

**Cost pilot (end of milestone 2).** Before milestone 3, the agents' cost in Vulcan is checked
cheaply: 8 ordinary tasks on a small application, written by the task writer and sealed apart from
milestone 4's material, 5 runs each in arms V and R. If V's clean success is more than 12 points
below R's, or V is more than 1.15 times slower, as point estimates, milestone 2 re-plans once (a
note in `docs/status.md`, signed by the run adjudicator), and the pilot runs again with 8 new
tasks. If it fails again, the project ends by row 4 of section 10 if its conditions hold, and
otherwise as **stop: the language costs too much**.


### 14. Records

The report states each threshold, each result with its bound and p-value, the sizing results, the
budget, the roles and their holders, every excluded or unanswered run, the runs whose final change
does not build, the coverage map, the sets with their sizes, the caveat that arm R has no
generator, and the outcome row. The analysis script runs on the released raw files and produces
every number.

## Open points

- **The thresholds are judgments,** fixed now. Nothing may change them after recruiting starts.
- **One model.** All results bind to one model version. The report says so.

## Amendments

| Source | Change |
|---|---|
| Audit 3 | Three arms, a pivot outcome, independent rule writer, named roles, inconclusive counts as fail, a frozen toolchain. |
| Audit 4 | Per-arm labels, exact permutation tests on paired per-task differences, minimum counts, mandatory roles, one local check command per arm, a functional brief, a complete decision table, sealing and isolation, a cheap gate on whether a language is needed. |
| Audit 5 | Arm R limited to static analyzers and one configuration file (owner's decision); seeded-case labels; U graded by runtime tests and a blinded grader; a non-owner task writer with preregistered targets; the owner kept from hidden material; freeze before rules; sizing by tasks. |
| Audit 6 | Contracts count in milestone 0.5; one "enforced" definition; arm C and U\*; a sizing gate before any compiler work; written-out check commands; a coverage map; an isolated reviewer; defined statistics; "narrow" as a hold; loads, commitments, and budgets. |
| Audit 7 | "Informative" defined; gate 2 first; arm C's contracts written one to one; seeded shapes translated; role separations; re-commitment and a scaled box; rejections audited; a defined re-run for "narrow"; cap gaps leave U\*; fixed sizing parameters; reference solutions. |
| Audit 8, B1 | Every tempting task targets the tested set (binding); a sizing check run before recruiting (`experiments/sizing/`), which fixed milestone 0.5 at 16 × 5 and milestone 4 at 24 to 32 tasks × 8 or 10 runs; margins of 12 points and 1.15 with no widening, since 10 points and 1.10 could not reach 0.8 at any design; r is the lower of 0.4 and the measured rate. |
| Audit 8, H1 | Gate 2 needs 8 rules and R's measured U\* rate of at least 0.2; paper contract calls name the reaching test, and arm C uses the same test. |
| Audit 8, H2 | Non-building final changes are judged by a blind grader for X2, fail every holdout test for X6, and are counted per arm. |
| Audit 8, H3 | The second reviewer samples accepted and rejected changes and X8 calls; N0, N1, and N3 must hold under both reviewers' judgments and without reviewer minutes; reversal defined. |
| Audit 8, H4 | The owner builds milestone 0.5's service; the Go developer commits at least 2.5 days a week; the plan's box follows. |
| Audit 9, B1 | N0 and N1 use clean success (X1c): an accepted change that keeps its targeted rules, in every arm; time runs to the first clean change. The sizing check ran again with coupled-cost, interaction, and net-cost scenarios; 28 × 10 still passes L, L+coupled, and L+inter at about 0.84 or more. |
| Audit 9, H1 to H5 | The sizing gate also requires L+coupled and L+inter, and publishes L+half as an accepted limit; a cost pilot at the end of milestone 2; X6 per accepted change; 8 U\*-targeting tasks in milestone 0.5 and published error rates for gate 2; a generator-gap rule and conformance cases for arm C. |
| Audit 9, M1 to M7, Lows | Exact sign-flip in the sizing check up to 16 tasks, and its limits stated; loads and boxes; seven people and the blind grader's separations; a "costs too much" row; the model-change rule; reversed acceptances and a disagreement limit; the milestone-3 retry and D5 checker; defined margins of P's, N4's exception, design order, estimators, release when the project ends, the FAQ pointer. |
| Audit 8, M1 to M9 | The grader is neither adjudicator, analyst, nor blind grader; the Go developer builds Go Stockroom; bounds are t-bounds over all tasks, and the half-bound is defined; a named blind grader; the second reviewer's blinding; U\* ≥ 8 at milestone 0.5; row 5 says "stop" only when the half-bound rules out the effect; gate 2 counts X2. Arm P runs only in milestone 0.5, and row 4 uses its result, which cuts milestone 4 to two arms. |
