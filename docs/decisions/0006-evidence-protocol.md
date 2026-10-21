# Decision 6: evidence protocol

Status: accepted, 2026-10-21 (option B, with the thresholds as written). Applied in the
development plan (milestones 2 and 4), spec §1.1, and `docs/status.md`. Milestone 0, item 6 of the
[development plan](../development-plan.md). Answers [audit 2](../../experiments/audit-2.md), H10,
and gives decision 4 its latency target.

## Question

How does milestone 4 decide whether Vulcan earns continued work? Which tasks, which
measurements, and which thresholds, fixed before any result exists? What latency must the
compiler meet (decision 4, rule 4)?

## Current state

- Spec §1.1 states three hypotheses. Vulcan takes architecture rules out of review (H-A). It
  rejects rule-breaking changes before review (H-B). It does not make a thorough reviewer read
  less, and it does not reduce logic defects that no rule covers (H-C).
- The development plan, milestone 4, names the task kinds, the records, and the exit gate:
  continue, narrow, stop or pivot, or inconclusive. It leaves the thresholds to milestone 0.
- Earlier experiments: experiments 1 and 2 lost their data. Test 3 had one writer for each
  language, no compiler, and no finished grading. Audit 2 (H10) found that this cannot support
  the premise.

## Lessons from test 3

- **Hidden rules work.** The writers did not see `hidden/rules.md`, so the test measured what a
  writer does without being told. Keep this.
- **A model checker is not a compiler.** The round-1 "check" was a model reading the spec. It
  found ambiguities, not facts. Milestone 4 uses only the real compiler.
- **One run says nothing about variation.** The protocol needs several runs of each task.
- **Grading must be scripted where it can be.** The grading of test 3 was never done. A script
  that runs the same HTTP tests and rule checks on both arms does not depend on someone finding
  the time.

## Options

**A. A demonstration.** Build the application, do a few changes, and judge. Cheap, and enough to
show feasibility. It cannot support a comparison, and it repeats the weakness of test 3.

**B. A preregistered comparison (recommended).** Fix the tasks, measurements, thresholds,
sample size, and analysis in this decision, commit them before the first run, and report every
run.

**C. An external study.** Ask an outside group to run B. More credible, but slow and costly
before the language has users. B can be repeated externally later with the same files.

## Recommendation: option B

### 1. The two arms

- **Vulcan arm:** the milestone-3 application in Vulcan, strict mode on (decision 2), with the
  primer (`vulcan guide`) and the full spec available to the agent.
- **Go arm:** a functionally equivalent Go application with the same packages, the same HTTP
  API, and the same tests. It gets "reasonable architecture tooling": the same architecture rules
  written as a document of at most 4,000 tokens (the primer's budget), and the import rules
  enforced in CI by a dependency linter, for example `depguard`. Its `go vet` and `staticcheck`
  run in CI.
- **The same everywhere else:** the model and its version, the agent harness, the tools, the
  token and time budgets, the machine, and the task briefs, word for word except for the
  language name.

### 2. Tasks

- **Twelve tasks**, written and committed before any run: three new features, two boundary
  refactors (move a responsibility between leaves), two dependency updates, two failure-handling
  changes, and three **tempting changes**. A tempting change has a brief that is easiest to meet
  by breaking an architecture rule, for example "Orders should also email the customer", where
  only a gateway may reach the network.
- **Hidden rules:** each task has rules that the agent does not see, in the style of test 3. They
  are the same for both arms.
- **Sealed:** the task files are committed with their hash before the first run. A task changes
  only to fix an error that makes it impossible in both arms, and the change is logged.
- **Pilot:** two extra tasks, not among the twelve, run first to test the harness and to estimate
  the variation. Pilot results are reported and not counted.

### 3. Runs

- **Fresh state:** each run starts from the same commit of its arm, with a fresh agent context and
  no memory of other runs.
- **Runs per task:** at least 5 for each arm. After the pilot, the sample size is set so that the
  primary outcome (section 5) has 80% power to detect the threshold difference, with at most 10
  runs per task and arm. The number is committed before the main runs.
- **Order:** the runs of the two arms alternate, in a random order fixed by a committed seed.
- **Human intervention:** an operator can answer an agent's question or unblock it. Each
  intervention is logged with its time and text, and counts toward the total time.

### 4. Measurements

Each measurement is defined so that a script can compute it from the logs, unless it says
otherwise.

| # | Measurement | Definition |
|---|---|---|
| M1 | Accepted change | The final diff passes the hidden HTTP acceptance tests, the hidden rule checks, and review |
| M2 | Architecture violations at review | Hidden-rule violations in the diff that the agent submits for review |
| M3 | Architecture violations at merge | Hidden-rule violations in the accepted diff |
| M4 | Violations stopped before review | Violations that appeared in any intermediate build of a run and were gone at review, divided by all violations that appeared |
| M5 | Total time | Wall-clock time from the brief to an accepted change, including reviews, rework, and interventions |
| M6 | Review time | Minutes that the reviewer spends on each review (human; timed) |
| M7 | Escaped logic defects | Failures of the hidden acceptance tests after acceptance, and defects that a second reviewer finds in a later audit of a random 30% sample |
| M8 | False positives | Diagnostics that the operator judges wrong after review (human; each one logged with its reason) |
| M9 | Workarounds | `strict_exempt`, `strict_interfaces`, `strict_shared`, `allow_*` entries, escape hatches, and K0122 to K0125 or K0210 to K0211 suppressions that a run adds |
| M10 | Agent cost | Tokens in and out, and compiler calls |
| M11 | Check latency | `vulcan check` time after a one-line edit in one leaf, and a full check, on the reference host (section 7) |
| M12 | Runtime overhead | p50 and p99 latency and throughput of the application's benchmark, Vulcan against the Go arm |
| M13 | Maintenance | Hours spent on the compiler and the runtime during milestone 4, logged apart from application work |

**Review.** One reviewer reviews both arms, in the randomized order. The language cannot be
hidden, so the reviewer follows a written checklist, which is the same for both arms except for
the items that a compiler or linter already guarantees. The reviewer records time and findings for
each review.

### 5. Hypotheses and thresholds

The analysis compares the arms task by task, and reports each result with a 95% bootstrap
confidence interval, resampled by task.

**Primary outcomes** (H-A and H-B):

- **P1.** Architecture violations at review (M2), per accepted change. Pass: the Vulcan rate is at
  most half the Go rate, and the upper bound of its interval is below the Go rate.
- **P2.** Violations stopped before review (M4). Pass: Vulcan stops at least 90%, and more than
  the Go arm with its linter.

**Non-inferiority** (the costs, and H-C):

- **N1.** Total time (M5). Pass: the Vulcan median is at most 1.10 times the Go median, upper
  interval bound included.
- **N2.** Escaped logic defects (M7). Pass: the Vulcan rate is at most 1.10 times the Go rate,
  upper interval bound included. H-C predicts no difference. A Vulcan rate clearly below Go would
  be a finding beyond the hypotheses, and is reported as such.
- **N3.** False positives (M8). Pass: at most 1 per 10 accepted changes.
- **N4.** Runtime overhead (M12). Pass: p99 latency and throughput within 10% of the Go arm.
- **N5.** Check latency (M11). Pass: the targets of section 7.

Review time (M6), workarounds (M9), agent cost (M10), and maintenance (M13) are reported with
their intervals. They have no thresholds, because the hypotheses make no claim about them; they
inform the decision to narrow.

### 6. The decision rule

Applied once, after the main runs, by the rule below:

| Result | Decision |
|---|---|
| P1 and P2 pass, and N1 to N5 pass | **Continue.** |
| P1 or P2 passes, and some feature fails a non-inferiority bound, or most workarounds come from one feature | **Narrow.** Keep the features that carry the passing outcome. Drop or redesign the others, by a new decision. |
| P1 and P2 fail, and the Go arm's intervals overlap Vulcan's or are better | **Stop or pivot.** Ordinary Go with a linter gets the same result. |
| An interval is too wide to pass or fail a threshold | **Inconclusive** for that outcome. One extension is allowed: up to the run cap of section 3. After it, the result stays inconclusive and is reported so. |

A run counts unless its infrastructure failed (the harness crashed, the model service failed, or
the machine failed). Each excluded run is listed with its reason. Runs that fail for any other
reason count as failures of their arm.

### 7. Latency targets (for decision 4)

On the reference host, a CI runner of type `linux/amd64` with 8 cores and 16 GB of memory, on the
milestone-3 application:

- **One-edit check:** `vulcan check` after a one-line edit in one leaf, p95 at most **1 second**.
  This matches an editor's save, and it is the trigger of decision 4, rule 4.
- **Full check:** `vulcan check` with an empty cache, at most **10 seconds**.
- **Build:** `vulcan build` after a one-line edit, p95 at most **2 seconds** more than
  `go build` of the Go arm after the same edit.

Milestone 2 measures these on the first compiler. A miss before milestone 4 starts triggers the
work of decision 4, rule 4. A miss in milestone 4 fails N5.

Q22 (§13) stays open: its targets compare against Java, C++, and Rust, which this protocol does
not need.

### 8. Records

- All files go in `experiments/m4/`: the sealed tasks and their hashes, the hidden rules, the seed,
  the harness, each run's prompt, transcript, commits, compiler version, logs, and grading output,
  the review notes, the exclusion list, and the analysis script.
- The analysis script runs on the raw files and produces every number in the report, so that
  another person can repeat it.
- The report states each threshold, each result with its interval, the sample size, and the
  decision, including failures and inconclusive results.

## Spec and plan changes on acceptance

- Development plan, milestone 4: point to this decision for tasks, measurements, thresholds, and
  the decision rule. Milestone 2: the latency targets of section 7.
- Spec §1.1: the hypotheses name the outcomes (P1, P2, N1, N2) that test them.
- `docs/status.md`: the milestone-0 packet is complete.

## Open points

- **The thresholds are judgments.** Half the violation rate, 90% stopped, and 10% non-inferiority
  margins are this draft's proposals. Changing them after the first main run is not allowed.
- **One reviewer is a limit.** A second reviewer for the defect audit (M7) is planned. More would
  reduce the effect of one person's habits.
- **The Go arm's linter sets the bar.** A stronger Go baseline makes P1 and P2 harder to pass. The
  linter configuration is committed with the tasks, so it cannot be weakened later.
