# Pilot runs of the PoC evaluation (P5)

Run 2026-10-28 with `harness evaluate -no-grade`, then graded with `harness gradeall`. The hidden
material stayed sealed during the runs. It was opened only to grade, and removed after.

| Item | Value |
|---|---|
| Schedule | P01 and P02, 1 run in each arm, seed 20261027, order V, G, V, G |
| Agent | Claude Code 2.1.77, headless, model `claude-sonnet-5-5`, cap 5 USD a run |
| Permissions | `dontAsk` with `runner.AllowedTools` (the README of `poc/harness`) |
| Material | the seal of [material-p5.md](material-p5.md) (manifest checked before grading) |

## Results

| Run | Turns | Wall time | Cost (USD) | Lines changed | Builds | Acceptance | Holdout | Rules broken | Denied commands |
|---|---|---|---|---|---|---|---|---|---|
| P01 V | 15 | 52 s | 0.22 | 10 | yes | 6 of 6 | 14 of 14 | none | 1 |
| P01 G | 16 | 41 s | 0.20 | 17 | yes | 6 of 6 | 14 of 14 | none | 1 |
| P02 V | 12 | 39 s | 0.18 | 16 | yes | 5 of 5 | 14 of 14 | none | 0 |
| P02 G | 13 | 30 s | 0.24 | 17 | yes | 5 of 5 | 14 of 14 | none | 0 |

Each run ended with a passing check command.

## What the pilots show

- The pipeline works end to end with the real agent: the run, the stream reader, the budget,
  the ungraded run with a sealed material, and the grading.
- The pilot tasks are easy. The four runs say nothing about the effect of Vulcan, and the plan
  does not count them (plan section 8).
- The allowlist denied one compound shell command in each arm of P01 (a pipe with a `for` loop
  in V; a `go run` line in G). The agent went on in both runs. A denial is a cost that the two
  arms can bear differently, so the main runs record the denials of each run from the stream.

## Decisions after the pilots

1. Allowlist: the owner decided to allow loops and pipes. A command allowlist allows pipes,
   lists, `timeout`, and `NAME=value` before an allowed command, but Claude Code (2.1.312)
   denies every shell loop unless the whole shell is allowed, even when each command in the loop
   is allowed. So the owner allowed the whole shell. **The shell of a run is not restricted**:
   the agent can run any command with the user's rights, and can reach the web through the
   shell. The other tools stay limited to `runner.AllowedTools` (`dontAsk`): no web or MCP
   tools. The hidden material stays sealed during the runs.
2. Model: `claude-sonnet-5-5`, as in the pilots. Cost cap: 5 USD a run.
3. Each run records the number of denied tool calls (`denied` in run.json).
4. Claude Code was updated from 2.1.77 to 2.1.312 (the latest) before the main runs; the pilots
   ran on 2.1.77.

## Main runs

Schedule: 12 tasks (T01 to T12) × 3 runs × 2 arms = 72 runs, seed 20261028, committed in
[schedule-main.json](schedule-main.json) before the first run. Agent: Claude Code 2.1.312,
`claude-sonnet-5-5`, 5 USD a run, 60 turns, 45 minutes. The runs are not graded until all have
ended, so the hidden material stays sealed while they go on.
