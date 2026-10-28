# Round 2: the litmus test of Vulcan against Go

Status: draft for the owner's review, 2026-10-28. Nothing here runs before the owner accepts it.

Round 1 ([results-p5.md](../../experiments/poc/results-p5.md)) could not answer its question.
Neither arm broke a rule in the product code, so there was nothing for Vulcan to prevent. Arm V
enforced 2 of the 16 rules, and an analyzer could enforce both, so a win would not have shown
that Vulcan differs from Go with good tools. A harness fault cut the turn budget to about 26
turns. This plan is built so that round 2 gives a clear answer, yes or no, and says in advance
which results mean which.

## 1. The questions

1. **Value:** when an AI agent changes an application, does Vulcan keep the team's rules more
   often than plain Go, at an acceptable cost?
2. **Difference:** does Vulcan do better than Go *with the best tools a Go team could add*:
   linters for every rule that a linter can decide, and the same contracts written as checks in
   Go?

Question 2 is the litmus test. If Go with linters gets the same result, Vulcan adds a language
where a linter would do, and the honest conclusion is to build the linter.

## 2. Step 0: the desk test (no runs)

Before any run, a fresh session that has written none of Vulcan answers one question in writing:
**which of Vulcan's checks can a Go team not get from `go/analysis` analyzers, build rules, and
in-code checks?** It goes through each diagnostic that the PoC compiler gives in strict mode
(architecture, effects, contracts, startup) and labels it:

| Label | Meaning |
|---|---|
| **Same in Go** | An analyzer, or an assert in the Go code, gives the same guarantee on the same code |
| **Harder in Go** | Possible in Go, but only with a convention that the agent can break without a diagnostic (for example a comment that marks a gateway) |
| **Not in Go** | No Go tool can check it from the code |

**Gate 0.** If every check is "Same in Go", round 2 cannot show a difference by enforcement. Then
it does not run as planned: the result is written down, and the owner decides whether Vulcan's
case rests on something else (the primer, the architecture that the code shows), which needs its
own test. This step costs no runs and may be the most useful result of the plan.

## 3. Arms

| Arm | What the agent gets | Check command |
|---|---|---|
| **G** | Plain Go and `RULES.md`, as in round 1 | `go vet ./... && go test ./...` |
| **GL** | Plain Go, `RULES.md`, a `golint-rules` command with one analyzer for each rule that a linter can decide, and the app's contracts written as Go checks that run in every test | `go vet ./... && golint-rules ./... && go test ./...` |
| **V** | Vulcan in strict mode, the primer, `RULES.md` | `vulcan check && vulcan test` |

The analyzers of GL are written by a fresh session that gets the rules in prose and the Go app,
and nothing of Vulcan. It also gets the seeded violations of each rule in arm G, and an analyzer
counts only if it rejects all of them and accepts the reference solutions. That makes GL the best
Go baseline we can build, not a weak one. The contracts in GL are the same as in V, written as Go
code that panics or returns an error, so that both arms start with the same checks in the code.

Everything else is the same in the three arms: model, harness, budget, briefs, and README.

## 4. Rules

A fresh rule-writer session writes 18 to 24 rules, in the four classes of round 1, then a labeler
session labels each one before any task exists. Labels: enforced by V, enforced by GL, in the
prose only. The rule set must have at least 6 rules in each of these groups:

| Group | Enforced by | Purpose |
|---|---|---|
| **Both** | V and GL | Vulcan should equal GL here, and both should beat G |
| **V only** | V, not GL (from step 0: "Harder" or "Not in Go") | The difference under test |
| **Neither** | prose only | The control: no arm should beat G |

If the rule-writer cannot reach 6 rules of "V only" that a real team would write, that is a
result in itself (gate 0 again), and the round stops there.

## 5. Tasks

Round 1's tasks did not tempt the model. Round 2 makes the temptation real and checks it before
the main runs.

- **More pressure:** a larger application (Stockroom-lite with two more services), briefs that
  ask for the feature in the words of a user, with the shortcut the natural way to do it, and
  some tasks in a chain of 3 changes on the same code, where a break in step 1 makes step 3
  easier.
- **Calibration:** a task-writer session writes 36 tempting tasks and 10 ordinary ones. Each
  tempting task runs 3 times in arm G only, with a separate seed. A tempting task is kept if G
  broke a targeted rule in at least 1 of 3 runs. The first 24 kept tasks, by a committed random
  order, go into the main runs. The calibration runs are not counted in the result.
  Choosing tasks where G broke rules lets G's rate fall in the main runs (regression to the
  mean). That works against V, so it does not make a V win look larger.
- **Ordinary tasks:** 8, to measure the cost of each arm on normal work.

## 6. Harness fixes before any run

Round 1 found these faults. Each fix comes with a test, and the pilot (section 8) checks them.

| # | Fault in round 1 | Fix |
|---|---|---|
| F1 | The turn budget counted stream events | Done: one turn per message (`agentlog`, `TestSplitMessage`). Budget: 60 messages |
| F2 | A stopped run has no output tokens or cost | Run with `--include-partial-messages` and read the output tokens from each message's final usage; compute the cost from a price table for the provider's model, not Claude's prices |
| F3 | Scratch files in the workspace were graded and counted as diff | Give the agent a scratch directory outside the workspace (`--add-dir`); the grader and the detectors read only the product packages; the diff counts only source files |
| F4 | Runs shared the user's `/tmp` | Run each agent in WSL (Ubuntu) under `bwrap`: its own `/tmp` and home, only the workspace and scratch writable. On Linux the check commands also add the race detector, which the plan wanted from the start |
| F5 | A passing check on unchanged code looked like progress | Record the first passing check *after the first edit* |
| F6 | V runs went on after the task was done | Both arms' preamble: "When the check passes and the brief is met, stop." Same words in all arms |
| F7 | Provider errors | Done: the run is kept apart and run again |
| F8 | The spend was not known per round | Record the provider's balance before and after each batch of runs |

The analysis grows to three arms (V−G, V−GL, GL−G, each paired by task) and the verdict table of
section 7. It is frozen, and it reads synthetic data sets right before the runs, including one
where GL equals V and one where V helps only on "V only" rules.

## 7. Reading the result (written now, before any run)

Primary measure: **clean success on tempting tasks** (the acceptance tests pass and no targeted
rule is broken), per task, paired across arms. Intervals: bootstrap 95% over tasks. "Above" means
the lower end of the interval is above 0.

**Validity gate.** The round counts only if (a) G breaks a targeted rule in at least 30% of its
runs on tempting tasks, and (b) each arm succeeds on at least 70% of ordinary-task runs. If (a)
fails, the tasks did not tempt; if (b) fails, the budget or the setup was too hard. In both cases
the result is "not informative", as in round 1, and no claim is made.

| Verdict | Condition |
|---|---|
| **Vulcan adds value and differs from Go** | V−G at least +15 points and above 0; V−GL at least +10 points and above 0; the gain over GL comes from the "V only" rules; no-harm holds |
| **Go with tools is enough** | V−G at least +15 and above 0, but V−GL below +5 or its interval includes 0, and GL−G at least +10. Build the linter, not the language |
| **No value** | V−G below +5, or the upper end of V−G's interval below +15 |
| **Harmful** | Task success on ordinary tasks, V−G, has its interval below −10, or the cost ratio V/G is above 1.5 |
| **Unclear** | Anything else: more runs on the same tasks, decided before they run |

No-harm: ordinary-task success V−G is at least −10 points, and the cost ratio V/G (input and
output tokens, geometric mean) is at most 1.5.

**Replication.** A "differs from Go" verdict counts only if a second model (Kimi through
OpenRouter) gives V−GL with the same sign on the same tasks. The two models are reported apart,
never pooled.

## 8. Size and cost

| Stage | Runs |
|---|---|
| Pilot: 2 tasks × 3 arms × 1 run, to check F1 to F8 | 6 |
| Calibration: 36 tempting tasks × 3 runs, arm G | 108 |
| Main: 32 tasks (24 tempting, 8 ordinary) × 3 arms × 3 runs | 288 |
| Replicate on a second model, only if the main result is positive | 288 |

Why this size: with 12 tasks, round 1's intervals were about 40 points wide. With 24 tempting
tasks the width is about 25 points, enough to tell +15 from 0, and the sign-flip test can reach
p below 0.01.

Cost: round 1's spend per run is not known exactly (F2, F8). The pilot measures it. **Cost gate:**
if the main runs would cost more than the funds left, the round shrinks in this order before it
starts: first the replicate waits; then 2 runs per task and arm instead of 3; then 18 tempting
tasks. It never drops an arm. The order is fixed now so that the size is not chosen after seeing
data.

## 9. Work order and who does it

| Step | Who | Gate |
|---|---|---|
| 1. Step 0, the desk test | A fresh session | Gate 0 |
| 2. Harness fixes F2 to F8, three-arm analysis, dry run | This project's builder | Tests pass, synthetic scenarios read right |
| 3. Rules and labels | Fresh rule-writer and labeler sessions | At least 6 rules in each group |
| 4. GL analyzers and contracts in Go | A fresh session with no Vulcan material | Reject every seeded violation, accept the reference solutions |
| 5. Tasks, reference solutions in three arms | A fresh task-writer session | Each reference passes in all arms |
| 6. Independent audit of steps 3 to 5 | A fresh session | No open High finding |
| 7. Seal, commit hashes and schedules | Builder | |
| 8. Pilot, then calibration | Builder | F1 to F8 work; at least 24 tempting tasks kept |
| 9. Main runs, grading, frozen analysis | Builder | Validity gate |
| 10. Results in `experiments/poc`, summary in the README | Builder | |

The builder sees ids, counts, and pass or fail of the hidden material, never its text, as in
round 1.
