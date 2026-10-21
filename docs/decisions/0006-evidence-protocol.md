# Decision 6: evidence protocol

Status: accepted, 2026-10-21 (option B). Amended the same day by
[audit 3](../../experiments/audit-3.md), amendments A1 to A3, A5, and A6 (see "Amendments" at
the end). Applied in the development plan (milestones 0.5, 2, and 4), spec §1.1, and
`docs/status.md`. Milestone 0, item 6 of the [development plan](../development-plan.md). Answers
[audit 2](../../experiments/audit-2.md), H10, and gives decision 4 its latency target.

## Question

How does milestone 4 decide whether Vulcan earns continued work? Which tasks, which
measurements, and which thresholds, fixed before any result exists? What latency must the
compiler meet (decision 4, rule 4)?

## Current state

- Spec §1.1 states three hypotheses. Vulcan takes architecture rules out of review (H-A). It
  rejects rule-breaking changes before review (H-B). It does not make a thorough reviewer read
  less, and it does not reduce logic defects that no rule covers (H-C).
- The development plan, milestone 4, names the task kinds, the records, and the exit gate:
  continue, narrow, stop or pivot, or inconclusive.
- Earlier experiments: experiments 1 and 2 lost their data. Test 3 had one writer for each
  language, no compiler, and no finished grading. Audit 2 (H10) found that this cannot support
  the premise.

## Lessons from test 3

- **Hidden rules work.** The writers did not see `hidden/rules.md`, so the test measured what a
  writer does without being told. Keep this.
- **A model checker is not a compiler.** The round-1 "check" was a model reading the spec. It
  found ambiguities, not facts. Milestone 4 uses only the real compiler.
- **One run says nothing about variation.** The protocol needs several runs of each task.
- **Grading must be scripted where it can be.** A script that runs the same HTTP tests and rule
  checks on every arm does not depend on someone finding the time.

## Options

**A. A demonstration.** Build the application, do a few changes, and judge. Cheap, and enough to
show feasibility. It cannot support a comparison, and it repeats the weakness of test 3.

**B. A preregistered comparison (chosen).** Fix the tasks, measurements, thresholds, sample size,
roles, and analysis in this decision, commit them before the first run, and report every run.

**C. An external study.** Ask an outside group to run B. More credible, but slow and costly
before the language has users. B can be repeated externally later with the same files.

## Protocol

### 1. Three arms

The question is not only "Vulcan or Go", but also "a language or a linter". Thus three arms:

- **Vulcan arm:** the milestone-3 application in Vulcan, strict mode on (decision 2), with the
  primer (`vulcan guide`) and the full spec available to the agent.
- **Go-rules arm:** the same application in Go, with **Vulcan's rules as Go analyzers**: the
  checks of the guarantee matrix (spec §5.4d) that need no new syntax (G1 to G7), written as
  `golang.org/x/tools/go/analysis` analyzers and run in CI, and the contracts of the Vulcan arm
  written as checked comments where a generator can check them. Its agent gets the rules as a
  document of at most 4,000 tokens. This arm is also the deliverable of milestone 0.5.
- **Go-plain arm:** the same Go code as the Go-rules arm, with the same rule document, and only a
  dependency linter (for example `depguard`), `go vet`, and `staticcheck` in CI.
- **The same everywhere else:** the model and its version, the agent harness, the tools, the
  token and time budgets, the machine, and the task briefs, word for word except for the
  language name.

The two Go arms share one codebase, so the third arm costs one CI configuration, not a third
application.

### 2. Roles

No result may depend only on the judgment of someone who prefers one outcome.

| Role | Who |
|---|---|
| Owner | Builds the Vulcan arm, the compiler, and the harness. Does not judge outcomes. |
| Go developer | A Go developer who does not work on Vulcan. Builds the Go arms, or reviews them against the acceptance criteria and signs that they are a fair baseline before the tasks are sealed. |
| Rule writer | A person who has not read the spec. Writes the hidden rules of an order and inventory service from its acceptance criteria alone. |
| Operator | Answers agents' questions during runs, and judges false positives (M8). Not the owner. If it must be the owner, a second person repeats each M8 judgment on a random 25% sample, and the report gives their agreement. |
| Reviewer | Reviews accepted changes and times them (M6). Not the owner, under the same fallback as the operator. |
| Second reviewer | Audits a random 30% sample for escaped defects (M7). |

The report names the person in each role, and says where a fallback applied.

### 3. Tasks

- **Twelve tasks**, written and committed before any run: **six tempting changes**, two new
  features, one boundary refactor (move a responsibility between leaves), one dependency update,
  and two failure-handling changes. A tempting change has a brief that is easiest to meet by
  breaking an architecture rule, for example "Orders should also email the customer", where only
  a gateway may reach the network.
- **Hidden rules** come from the rule writer (section 2), not from the spec. The grading labels
  each hidden rule *enforced* (a Vulcan diagnostic covers it) or *not enforced*, and reports the
  outcomes for each label apart. Rules that only Vulcan's compiler could have suggested are not
  allowed.
- **Follow-on tasks:** each tempting task has one follow-on task that starts from that run's
  accepted change. The follow-on measures what a violation costs later (M14).
- **Sealed:** the task files, the hidden rules, the analyzers of the Go-rules arm, and the linter
  configuration of the Go-plain arm are committed with their hashes before the first run. A task
  changes only to fix an error that makes it impossible in every arm, and the change is logged.
- **Pilot:** two extra tasks, not among the twelve, run first to test the harness and to estimate
  the variation. Pilot results are reported and not counted.

### 4. Runs

- **Fresh state:** each run starts from the same commit of its arm, with a fresh agent context and
  no memory of other runs.
- **Runs per task:** at least 5 for each arm. After the pilot, the sample size is set so that the
  primary outcomes (section 6) have 80% power to detect their threshold differences, with at most
  10 runs per task and arm. The number is committed before the main runs.
- **Order:** the runs of the three arms interleave, in a random order fixed by a committed seed.
- **Human intervention:** the operator can answer an agent's question or unblock it. Each
  intervention is logged with its time and text, and counts toward the total time.
- **A frozen compiler.** The main runs use one tagged version of the compiler, the analyzers, and
  the runtime. A run that a compiler bug blocks ends as a failure of the Vulcan arm. If more than
  10% of Vulcan runs end this way, the evaluation stops; the bug is fixed under a new tag, and
  every Vulcan run is repeated.

### 5. Measurements

Each measurement is defined so that a script can compute it from the logs, unless it says
otherwise.

| # | Measurement | Definition |
|---|---|---|
| M1 | Accepted change | The final diff passes the hidden HTTP acceptance tests, the hidden rule checks, and review |
| M2 | Architecture violations at review | Hidden-rule violations in the diff that the agent submits for review, by label (enforced, not enforced) |
| M3 | Architecture violations at merge | Hidden-rule violations in the accepted diff, by label |
| M4 | Violations stopped before review | Violations that appeared in any intermediate build of a run and were gone at review, divided by all violations that appeared, by label |
| M5 | Total time | Wall-clock time from the brief to an accepted change, including reviews, rework, and interventions |
| M6 | Review time | Minutes that the reviewer spends on each review (human; timed) |
| M7 | Escaped logic defects | Failures of the hidden acceptance tests after acceptance, and defects that the second reviewer finds in the 30% sample |
| M8 | False positives | Diagnostics that the operator judges wrong after review (human; each one logged with its reason) |
| M9 | Workarounds | `strict_exempt`, `strict_interfaces`, `strict_shared`, `allow_*` entries, escape hatches, analyzer suppressions, and K0122 to K0125 or K0210 to K0211 suppressions that a run adds |
| M10 | Agent cost | Tokens in and out, and compiler or analyzer calls |
| M11 | Check latency | `vulcan check` time after a one-line edit in one leaf, and a full check, on the reference host (section 8) |
| M12 | Runtime overhead | p50 and p99 latency and throughput of the application's benchmark, Vulcan against the Go arms |
| M13 | Maintenance | Hours spent on the compiler, the runtime, and the analyzers during milestone 4, logged apart from application work |
| M14 | Cost of a violation | In each follow-on task: time, rework, and escaped defects, for runs whose tempting change had a violation at merge against runs whose change had none |

**Review.** The reviewer reviews all arms, in the randomized order. The language cannot be
hidden, so the reviewer follows a written checklist, which is the same for every arm except for
the items that its compiler or analyzers already guarantee.

### 6. Hypotheses and thresholds

The analysis compares the arms task by task, and reports each result with a 95% bootstrap
confidence interval, resampled by task. The primary outcomes use the six tempting tasks and the
*enforced* label; the *not enforced* label is reported with the same statistics.

**Primary outcomes** (H-A and H-B):

- **P1.** Architecture violations at review (M2), per accepted change. Pass against an arm: the
  Vulcan rate is at most half that arm's rate, and the upper bound of its interval is below that
  arm's rate.
- **P2.** Violations stopped before review (M4). Pass against an arm: Vulcan stops at least 90%,
  and more than that arm.

**Non-inferiority** (the costs, and H-C), each against both Go arms:

- **N1.** Total time (M5). Pass: the Vulcan median is at most 1.10 times the arm's median, upper
  interval bound included.
- **N2.** Escaped logic defects (M7). Pass: the Vulcan rate is at most 1.10 times the arm's rate,
  upper interval bound included. H-C predicts no difference. A Vulcan rate clearly below would be
  a finding beyond the hypotheses, and is reported as such.
- **N3.** False positives (M8). Pass: at most 1 per 10 accepted changes.
- **N4.** Runtime overhead (M12). Pass: p99 latency and throughput within 10% of the Go arms.
- **N5.** Check latency (M11). Pass: the targets of section 8.

Review time (M6), workarounds (M9), agent cost (M10), maintenance (M13), and the cost of a
violation (M14) are reported with their intervals. They have no thresholds; they inform the
decision to narrow, and M14 shows whether the violations that P1 counts matter.

**Inconclusive results.** An interval too wide to decide a threshold allows one extension, up to
the run cap of section 4. After it, an inconclusive **primary** outcome counts as a **fail**, and
an inconclusive **non-inferiority** bound counts as a fail of that bound. Thus an unclear result
never continues the project by default.

### 7. The decision rule

Applied once, after the main runs. "Passes against X" means P1 and P2 both pass against arm X.

| Result | Decision |
|---|---|
| Vulcan passes against Go-rules, and N1 to N5 pass | **Continue the language.** The language adds more than its rules as a linter. |
| Vulcan passes against Go-plain but not against Go-rules, and Go-rules passes against Go-plain | **Pivot to the rules.** The rules help, and Go analyzers deliver them. Continue as a Go tool, not as a language. |
| Vulcan passes against Go-rules, but a non-inferiority bound fails, or most workarounds come from one feature | **Narrow.** Keep the features that carry the passing outcomes. Drop or redesign the others, by a new decision. |
| Neither Vulcan nor Go-rules passes against Go-plain | **Stop.** Enforcing the rules does not change outcomes. |
| M14 shows no cost for violations at merge, whatever P1 and P2 show | The report states that removing violations was not shown to matter. Any "continue" or "pivot" decision is then **narrowed** to what M14 and M7 support. |

A run counts unless its infrastructure failed (the harness crashed, the model service failed, or
the machine failed). Each excluded run is listed with its reason. Runs that fail for any other
reason count as failures of their arm.

### 8. Latency targets (for decision 4)

On the reference host, a CI runner of type `linux/amd64` with 8 cores and 16 GB of memory, on the
milestone-3 application:

- **One-edit check:** `vulcan check` after a one-line edit in one leaf, p95 at most **1 second**.
  This matches an editor's save, and it is the trigger of decision 4, rule 4.
- **Full check:** `vulcan check` with an empty cache, at most **10 seconds**.
- **Build:** `vulcan build` after a one-line edit, p95 at most **2 seconds** more than
  `go build` of the Go arms after the same edit.

Milestone 2 measures these on the first compiler. A miss before milestone 4 starts triggers the
work of decision 4, rule 4. A miss in milestone 4 fails N5.

Q22 (§13) stays open: its targets compare against Java, C++, and Rust, which this protocol does
not need.

### 9. The reduced run of milestone 0.5

Before any compiler work, the development plan runs this protocol in a reduced form, with two
arms: **Go-rules** against **Go-plain**, on a small application (for example the test 3 shop,
rebuilt with a database). It uses 6 tasks, 4 of them tempting, at least 3 runs per task and arm,
the roles of section 2, and the same sealing. Its decision:

- If Go-rules does not pass P1 and P2 against Go-plain, **stop**: enforcing the rules does not
  change outcomes, so a language that enforces them will not either.
- Otherwise, continue to milestone 1. The analyzers become the Go-rules arm of milestone 4.

### 10. Records

- All files go in `experiments/m0.5/` and `experiments/m4/`: the sealed tasks and their hashes,
  the hidden rules, the analyzers and linter configuration, the seed, the harness, each run's
  prompt, transcript, commits, compiler or analyzer version, logs, and grading output, the review
  notes, the role list, the exclusion list, and the analysis script.
- The analysis script runs on the raw files and produces every number in the report, so that
  another person can repeat it.
- The report states each threshold, each result with its interval, the sample size, the roles,
  and the decision, including failures and inconclusive results.

## Open points

- **The thresholds are judgments.** Half the violation rate, 90% stopped, and 10% non-inferiority
  margins are proposals fixed now. Changing them after the first main run is not allowed.
- **The Go-rules arm sets the bar for the language.** Its analyzers are sealed before the runs,
  and the Go developer of section 2 checks that they are a fair effort, not a weak one.
- **Finding people.** The roles of section 2 need two to four people besides the owner. If they
  cannot be found, the report says so on its first page, and its decision is marked as not
  independently judged.

## Amendments

Audit 3 (2026-10-21) found that the first version could not tell a language from a linter, that
its primary outcomes passed by construction, and that the owner judged the results. These changes
answer it:

| Amendment | Change |
|---|---|
| A1 | Three arms: Vulcan, Go-rules, Go-plain (section 1). A "pivot to the rules" outcome (section 7). |
| A2 | Hidden rules from a person who has not read the spec, labelled enforced or not; six of twelve tasks tempting; follow-on tasks and M14 (sections 3, 5, 7). |
| A3 | Named roles, with the owner kept out of judging (section 2). |
| A5 | Inconclusive results count as fails (section 6). |
| A6 | A frozen compiler, and a rule for blocking bugs (section 4). |
| A4, part | The reduced run of milestone 0.5 (section 9). |
