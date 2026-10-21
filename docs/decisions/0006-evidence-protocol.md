# Decision 6: evidence protocol

Status: accepted, 2026-10-21 (option B). Amended twice on the same day: by
[audit 3](../../experiments/audit-3.md) (A1 to A3, A5, A6) and by
[audit 4](../../experiments/audit-4.md), an independent audit (B1 to B4, H1, H2, H4, H6). See
"Amendments" at the end. Milestone 0, item 6 of the [development plan](../development-plan.md).
Answers [audit 2](../../experiments/audit-2.md), H10, and gives decision 4 its latency target.

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

- **Hidden rules work.** The writers did not see the rules, so the test measured what a writer
  does without being told. Keep this, but keep the rules out of the agents' reach (section 11).
- **A model checker is not a compiler.** Milestone 4 uses only the real compiler.
- **One run says nothing about variation.** The protocol needs several runs of each task.
- **Grading must be scripted where it can be.**

## Options

**A. A demonstration.** Cheap, and enough to show feasibility. It cannot support a comparison.

**B. A preregistered comparison (chosen).** Fix the tasks, measurements, statistics, roles, and
decision rule here, seal them before the first run, and report every run.

**C. An external study.** More credible, but slow and costly before the language has users. B
can be repeated externally later with the same files.

## Protocol

### 1. The question in three parts

A language must beat its own rules delivered as a linter, or it is not worth building. Thus the
protocol asks three questions, each a test in section 7:

- **T1. Do the rules help?** Go with the rules against Go without them.
- **T2. Does the language enforce rules that Go cannot?** Vulcan against Go with every rule that
  Go can express, on the rules that only Vulcan can enforce.
- **T3. Is the language at least as good on the rules both enforce?** Vulcan against Go with the
  rules, on the shared rules.

### 2. Three arms

- **Vulcan arm (V):** the milestone-3 application in Vulcan, strict mode on (decision 2).
- **Go-rules arm (R):** the same application in Go, with **every rule that Go code, comments, or
  configuration can express**: the guarantee matrix G1 to G9 (spec §5.4d) as
  `golang.org/x/tools/go/analysis` analyzers, the strict-mode checks (K0123 to K0125, K0210,
  K0211) as analyzers, a configuration file for gateways and entry leaves, and port and struct
  contracts as checked comments with a generator. The Go developer (section 3) signs that this
  coverage is complete: that no rule of Vulcan that Go can express is left out.
- **Go-plain arm (P):** the same Go code as R, with only `go vet`, `staticcheck`, and a dependency
  linter whose configuration is sealed with a statement of what it covers.
- **One local check command per arm.** Each arm has one command, named in every task brief, that
  runs its checkers, builds, and runs its tests: `vulcan check && vulcan test` for V, and
  `make check` for R and P. The agent can run it at any time, at the same cost. A change goes to
  review only after its arm's command passes. The harness logs each run of the command.
- **The same everywhere else:** the model and its version, the agent harness, the token and time
  budgets, the machine, and the task briefs, word for word except for the language name. Each arm
  gets documents of comparable size: V gets the primer and the spec; R and P get a rule document
  and a Go guide of the same total length, which the Go developer writes or checks.

The two Go arms share one codebase.

### 3. Roles

The owner builds the Vulcan arm, the compiler, and the harness, and makes **no** judgment that
affects a result. These roles are mandatory, and each is filled by someone other than the owner:

| Role | Duties |
|---|---|
| Go developer | Builds the Go version of the application, or reviews it and writes a timed list of changes that are applied before sealing. Signs the coverage of arm R. Certifies which rules Go cannot express (section 5). |
| Rule writer | Has not read the spec. Writes the hidden rules from a functional brief (section 5). Holds no other role. |
| Task reviewer | Reviews the task briefs and can veto any of them. Applies the filter of section 5. Labels the hidden rules with the Go developer. |
| Run adjudicator | Decides run exclusions and whether a compiler bug blocks a run (section 6). |
| Operator | Answers agents' questions during runs, only from an FAQ sealed before the runs. A question that the FAQ does not answer ends the run as "unanswered", for every arm alike. |
| Reviewer | Reviews accepted changes and times them (M6). Judges false positives (M8). |
| Second reviewer | Audits a random 30% sample for escaped defects (M7). |

One person can hold several roles, except the rule writer. **If any role is not filled when the
files are sealed, the evaluation does not run.** The report names the person in each role.

### 4. Tasks

- **Milestone 4:** fourteen tasks, written by the owner and approved by the task reviewer:
  **eight tempting changes**, two new features, one boundary refactor, one dependency update, and
  two failure-handling changes. A tempting change has a brief that is easiest to meet by breaking
  an architecture rule.
- **Follow-on tasks:** each tempting task has one follow-on task that starts from that run's
  accepted change. They measure what a violation costs later (M14, reported only).
- **Pilot:** two extra tasks, run first to test the harness and to estimate variation. Reported and
  not counted.

### 5. Hidden rules and their labels

- **The brief.** The rule writer gets only the functional, data-integrity, security, and resource
  criteria of the application (F, D, S, and R in `docs/release-1/acceptance.md`) and a neutral
  package diagram with no Vulcan terms. The rule writer gets no Vulcan document.
- **Sealing before labels.** The rules are sealed before anyone labels them.
- **Filter.** The task reviewer removes a rule that only someone who knows Vulcan's compiler could
  have written, and logs each removal with its reason.
- **Labels.** Each rule gets two labels, one for arm V and one for arm R: *enforced* (the arm's
  checkers reject a change that breaks it) or *not enforced*. For arm V, the label comes from the
  compiler's conformance suite; for arm R, from the analyzers. The rules then fall into four sets:
  - **S, shared:** enforced by both V and R.
  - **U, Vulcan-only:** enforced by V and not by R, *and* certified by the Go developer as not
    expressible in Go code, comments, or configuration, with a written reason for each.
  - **G, Go-only:** enforced by R and not by V. Reported.
  - **N, neither.** Reported.
  A rule that V enforces and R does not, but that the Go developer judges expressible in Go, is a
  gap in arm R. Arm R is fixed before sealing.

### 6. Runs

- **Fresh state:** each run starts from a fresh workspace (section 11), with a fresh agent context.
- **Runs per task:** at least 5 for each arm, at most 10. After the pilot, the number is set so
  that each test has 80% power at its threshold, and committed before the main runs. If the cap
  cannot reach that power, the report says so before the main runs, and the evaluation runs at the
  cap.
- **Order:** the runs of all arms interleave, in a random order fixed by a committed seed.
- **A frozen toolchain.** The main runs use one tag of the compiler, the analyzers, and the runtime,
  and record the model build of each run. A run that a compiler or analyzer bug blocks ends as a
  failure of its arm; the run adjudicator decides. If more than 10% of an arm's runs end this way,
  the evaluation stops, the bug is fixed under a new tag, and the whole interleaved schedule runs
  again for every arm. This can happen once; a second time ends the evaluation as inconclusive.
- **Exclusions:** a run counts unless its infrastructure failed (the harness, the model service, or
  the machine), as the run adjudicator decides. Each exclusion is listed.

### 7. Measurements

| # | Measurement | Definition |
|---|---|---|
| M1 | Task success | A run ends in an accepted change: the hidden HTTP tests, the hidden rule checks, and review pass |
| M2 | Violations at review | Hidden-rule violations in the change submitted for review, by rule set (S, U, G, N) |
| M3 | Violations at merge | Hidden-rule violations in the accepted change, by rule set |
| M4 | Violations stopped before review | Violations that appeared in a run of the arm's check command and were gone at review, divided by all that appeared, by rule set; defined only when at least 10 appeared in that arm |
| M5 | Working time | Agent active time plus the reviewer's measured minutes; queue time excluded. A failed run counts as the full budget |
| M6 | Review time | Minutes that the reviewer spends on each review |
| M7 | Escaped logic defects | Failures of the hidden HTTP tests after acceptance, and defects that the second reviewer finds in the 30% sample, per run |
| M8 | False positives | Diagnostics that the reviewer judges wrong, with the reason |
| M9 | Workarounds | Exemptions and allow-list entries that a run adds (`strict_exempt`, `strict_interfaces`, `strict_shared`, `allow_*`, analyzer exclusions) |
| M10 | Agent cost | Tokens in and out, and check-command runs |
| M11 | Check latency | Section 9 |
| M12 | Runtime overhead | p50 and p99 latency and throughput of the application's benchmark |
| M13 | Maintenance | Hours spent on the compiler, runtime, and analyzers during the evaluation |
| M14 | Cost of a violation | In follow-on tasks: working time and escaped defects after a violating base against a clean base. Reported only |

### 8. Statistics and thresholds

**Unit.** The task is the unit. For each task and arm, the rate of a measurement is its mean over
that arm's runs of the task. A comparison of two arms uses the **paired per-task difference**.

**Test.** Each comparison is a one-sided **exact sign-flip permutation test** over the tasks (all
2^k sign patterns), at α = 0.05, with a 95% interval for the mean paired difference found by
inverting the test. With k tasks, the smallest p-value is 1/2^k, so a comparison needs at least 5
informative tasks; the tempting tasks give 8.

**Minimum evidence.** A violation comparison is decided only when its rule set holds at least
**8 rules** (S) or **4 rules** (U), and both arms together show at least **10 violation events**
at review. Below that, the comparison is **inconclusive**.

**Inconclusive counts as fail**, for every test and every bound. There is no extension.

**P1(X better than Y) on a rule set:** the test of "X has fewer violations at review than Y"
passes, and the mean rate of X is at most half the mean rate of Y.

**P2(X):** X stops at least 90% of the violations that appeared (M4). When M4 is not defined for X
(fewer than 10 appeared), P2 is not used, and the test relies on P1.

**The tests of section 1:**

- **T1 passes** when P1(R better than P) on S passes, and P2(R) passes where defined.
- **T2 passes** when P1(V better than R) on U passes.
- **T3 passes** when V is no worse than R on S: the upper bound of the interval of
  (V rate minus R rate) is at most 0.05 violations per run.

**Non-inferiority bounds**, V against R, each by the same test on paired per-task differences:

- **N0. Task success (M1):** V's success rate is at most 10 percentage points below R's.
- **N1. Working time (M5):** the upper bound of the mean per-task log ratio V/R is at most log 1.10.
- **N2. Escaped defects (M7):** the upper bound of (V minus R) is at most 0.1 defects per run.
- **N3. False positives (M8):** at most 1 per 10 accepted changes in V.
- **N4. Runtime overhead (M12):** p99 latency and throughput within 10% of R.
- **N5. Check latency:** the targets of section 9.

M2 on the sets G and N, and M6, M9, M10, M13, and M14 are reported with intervals and decide
nothing.

### 9. Latency targets (for decision 4)

On a CI runner of type `linux/amd64` with 8 cores and 16 GB of memory:

- **One-edit check:** `vulcan check` after a one-line edit in one leaf, p95 at most 1 second.
- **Full check:** `vulcan check` with an empty cache, at most 10 seconds.
- **Build:** `vulcan build` after a one-line edit, p95 at most 2 seconds more than `go build` of
  arm R after the same edit.

Milestone 2 measures these on a synthetic application of fixed size, committed with the compiler,
and milestone 3 measures them again on the application. A miss before milestone 4 starts the work
of decision 4, rule 4. A miss in milestone 4 fails N5.

### 10. The decision rule

Apply the rows in order. The first row that matches decides.

| # | Condition | Outcome |
|---|---|---|
| 1 | A mandatory role is empty, or the evaluation ended as inconclusive under section 6 | **Stop: not judged.** The project does not continue on this evidence. |
| 2 | T2 passes, T3 passes, and N0 to N5 pass | **Continue the language.** |
| 3 | T2 passes, and T3 or a bound of N0 to N5 fails | **Narrow.** Keep the language features that enforce the rules of set U that carried T2. A new decision drops or redesigns the rest. |
| 4 | T2 fails, T1 passes, and R meets N0 and N1 against P | **Pivot to the rules.** Continue as a Go tool (the analyzers of arm R), not as a language. |
| 5 | Any other result | **Stop.** |

Every combination of the three tests and the bounds matches exactly one row. A second time-box
overrun of a milestone (development plan) also ends the project, as **stopped by schedule**.

### 11. Sealing and isolation

- **A private store.** Task briefs, hidden rules, hidden tests, labels, FAQ, seed, analyzer and
  linter configurations, and grading scripts are kept in a private store outside this repository
  until the runs end. The repository holds only their SHA-256 hashes.
- **An outside timestamp.** The commit of the hashes gets a signed tag that is pushed to a third
  party, and the hashes are also published where the owner cannot change them later (for example,
  a public post).
- **Agent workspaces.** Each run gets a fresh workspace that holds only its arm's application code
  and its arm's documents. It has no access to this repository's `experiments/`, to the private
  store, or to the other arms.
- **Release after the runs.** After the runs, the sealed contents are copied into
  `experiments/m0.5/` or `experiments/m4/`, and their hashes are checked against the tag.

### 12. The reduced run of milestone 0.5

Before any compiler work, the development plan runs this protocol in a reduced form, on a small
application (for example the test 3 shop, rebuilt with a database):

- **Arms:** R and P only. There is no compiler, so set U is defined from the spec: a rule that the
  spec says Vulcan enforces, and that the Go developer certifies Go cannot express. The task
  reviewer and the Go developer make this call together, and record each reason.
- **Tasks:** eight, **six of them tempting**; at least 3 runs per task and arm. All roles of
  section 3. All sealing of section 11.
- **Gate 1 (do the rules help?):** if T1 fails, **stop**.
- **Gate 2 (is a language needed?):** count the violations at review, in both arms, of rules in set
  U. If set U holds fewer than 4 rules, or these violations number fewer than 5 events, **pivot to
  the rules** without building the compiler: the analyzers already deliver everything that Go can
  express, and there is too little left that only a language could enforce.
- Otherwise continue to milestone 1. Arm R's analyzers become arm R of milestone 4.

### 13. Records

The report states each threshold, each result with its interval and p-value, the sample size, the
roles and their holders, every excluded or unanswered run, and the outcome row. The analysis script
runs on the released raw files and produces every number, so that another person can repeat it.

## Open points

- **The thresholds are judgments.** Margins and minimums are fixed now; changing them after the
  first main run is not allowed.
- **Finding people.** Section 3 needs at least three people besides the owner (the rule writer,
  the Go developer, and one more for the other roles). Without them, row 1 applies and the project
  stops. The development plan sets a recruiting deadline.
- **One model.** All results bind to one model version. The report says so.

## Amendments

| Source | Change |
|---|---|
| Audit 3, A1 | Three arms, and a "pivot to the rules" outcome. |
| Audit 3, A2 | Hidden rules from a person who has not read the spec; follow-on tasks. |
| Audit 3, A3 | Named roles. |
| Audit 3, A5 | Inconclusive counts as fail. |
| Audit 3, A6 | A frozen toolchain. |
| Audit 4, B1 | Arm R holds every rule that Go can express; rules are labelled per arm into sets S, U, G, and N; the language must win on U (T2) and tie on S (T3). |
| Audit 4, B2 | Paired per-task differences with an exact permutation test; minimum rule and event counts; P2 undefined below 10 events; eight tempting tasks in milestone 4 and six in milestone 0.5. |
| Audit 4, B3 | Mandatory non-owner roles with no escape clause; a task reviewer and a run adjudicator; an operator who answers only from a sealed FAQ. |
| Audit 4, B4 | Gate 2 of milestone 0.5: pivot without a compiler when too little needs a language. |
| Audit 4, H1 | One local check command per arm; review only after it passes. |
| Audit 4, H2 | A functional brief for the rule writer, sealed before labelling. |
| Audit 4, H4 | A complete, ordered decision table, with "not judged" and "stopped by schedule". |
| Audit 4, H6 | A private store, an outside timestamp, and isolated workspaces. |
