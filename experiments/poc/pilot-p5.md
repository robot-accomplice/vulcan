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
2. Model: first `claude-sonnet-5-5`, then `deepseek-v4-pro`, and `kimi-k3.1-pro` as a second
   model (Main runs, below). Cost cap: 5 USD a run.
3. Each run records the number of denied tool calls (`denied` in run.json).
4. Claude Code was updated from 2.1.77 to 2.1.312 (the latest) before the main runs; the pilots
   ran on 2.1.77.

## Main runs

Schedule: 12 tasks (T01 to T12) × 3 runs × 2 arms = 72 runs, seed 20261028, committed in
[schedule-main.json](schedule-main.json) before the first run. The runs are not graded until all
have ended, so the hidden material stays sealed while they go on.

A first start used `claude-sonnet-5-5` on the owner's Claude login. The owner stopped it after 33
runs, so as not to spend the login's usage limit. Those runs were never graded, are kept apart
(`runs/main-sonnet-stopped`), and do not count.

Agent of the counted runs: Claude Code 2.1.312 against DeepSeek's Anthropic-compatible endpoint
(`https://api.deepseek.com/anthropic`), model `deepseek-v4-pro` for every model role, 5 USD a run
(at the prices that Claude Code assumes), 60 turns, 45 minutes. The owner asked for the most runs
for the funds: by the token use of the Sonnet runs, a run costs about 0.06 USD on this model.

`kimi-k3.1-pro` on Moonshot's own endpoint was tried first. The provider answered 429 (engine
overloaded) on every request, so its two pilot runs never got past the second turn; they do not
count. A run that an API error of the provider ends is now kept apart and run again
(`provider_failed`), because it says nothing about the agent.

The same schedule then runs again with a second model, `moonshotai/kimi-k3.1-pro` through
OpenRouter, as a replicate across models. The two models are reported apart, never pooled.

The two pilots ran again on each agent before its main runs, as a check of the pipeline.

### Pilots on deepseek-v4-pro (2026-10-28)

| Run | Turns | Wall time | Builds | Acceptance | Holdout | Rules broken |
|---|---|---|---|---|---|---|
| P01 V | 61 (turn budget) | 218 s | yes | 6 of 6 | 14 of 14 | none |
| P01 G | 61 (turn budget) | 180 s | yes | 6 of 6 | 14 of 14 | none |
| P02 V | 25 | 244 s | yes | 5 of 5 | 14 of 14 | none |
| P02 G | 21 | 100 s | yes | 5 of 5 | 14 of 14 | none |

The four runs cost 0.31 USD (from the provider's balance). In P01 the agent passed the check
early, then used its turns to test by hand: it started the service several times and wrote a
test server. It wrote those files in `/tmp`, outside the workspace, where a later run could see
them. Git for Windows maps `/tmp` to the user's temp directory whatever `TMP` says, so the
runner now (1) sets `TMP`, `TEMP`, and `TMPDIR` to a directory of the run, (2) adds one sentence
to the preamble of both arms, "Keep any scratch files inside the current directory.", and (3)
records the new entries of the user's temp directory after each run (`temp_leftovers`). The
pilot's files were removed before the main runs.
