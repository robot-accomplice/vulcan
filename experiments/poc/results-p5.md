# Results of the PoC evaluation, round 1 (P5, deepseek-v4-pro)

Run 2026-10-28. This round compares arm V (Vulcan, with strict mode) with arm G (plain Go, with
the rules in prose in `RULES.md`), on 12 tasks of the Stockroom-lite application, 3 runs a task
and arm: 72 runs. The design is in [docs/poc/plan.md](../../docs/poc/plan.md), the hidden material
in [material-p5.md](material-p5.md), the pilots and the decisions before the round in
[pilot-p5.md](pilot-p5.md). The data is in [data/p5-deepseek](data/p5-deepseek).

## The answer

**No signal for Vulcan, and arm V did worse.** The frozen analysis (plan section 9) reads
"No signal". Neither arm broke a rule in the product code, so the round could not show the gain
that Vulcan is for. Arm V lost tasks instead: 8 of 36 runs failed in V, 1 of 36 in G. All 9
failures are runs that the turn budget stopped, and that budget was smaller than planned (below).

| Measure (V minus G, paired by task) | Value | 95% interval | p |
|---|---|---|---|
| Task success, all 12 tasks | −19.4 points | −38.9 to −2.8 | 0.12 |
| Task success, 8 tempting tasks | −25.0 points | −54.2 to −4.2 | 0.25 |
| Task success, 4 ordinary tasks | −8.3 points | −25.0 to 0.0 | 1.00 |
| Clean success, tempting tasks | −20.8 points | −50.0 to +4.2 | 0.38 |
| Runs that break a rule | 0 in V, 2 in G (both in `scratch/`, below) | | |
| Cost ratio V / G (input tokens, geometric mean) | 1.08 | | |

| Arm | Runs | Success | Rule breaks in product code | Stopped by the turn budget | Median first passing check (turn) |
|---|---|---|---|---|---|
| V | 36 | 28 | 0 | 32 | 10.5 |
| G | 36 | 35 | 0 | 11 | 13.0 |

The full frozen report is [analysis.md](data/p5-deepseek/analysis.md).

## What went wrong in the measurement

These are faults of the harness or of the setup, found after the runs. Each one is fixed or
listed for the next round. None of them was known when the runs started.

1. **The turn budget counted stream events, not turns.** Claude Code writes one stream event for
   each content block of a message (thinking, text, a tool call), and the stream reader counted
   each event as a turn. The runner stopped a run at "61 turns", which was about 26 messages of
   the model: less than half of the planned 60. Both arms ran under the same rule, so the
   comparison is fair as a comparison, but the budget was not the planned one. Both arms wrote
   about 2.4 events a message. G runs more often ended on their own before the budget; V runs
   went on working. So the budget stopped 32 V runs and 11 G runs. The 8 V failures
   are all stopped runs: 4 did not build when stopped, 2 (T03) had changed no line
   yet, 2 failed acceptance tests. The pilots (pilot-p5.md) were counted the same way.

   *Fix:* `agentlog` counts one turn per message id, with a test of the split stream
   (`TestSplitMessage`). `harness restream` read the 72 streams again; the turns and input tokens
   in the data are the corrected counts. The budget that stopped the runs cannot be corrected
   after the fact.

2. **The usage of a stopped run is not known.** A run that the runner stops has no final result
   event, and the events of a message carry its input tokens but not its output tokens. 41 runs
   (30 V, 11 G) have `usage_partial: true`: their output tokens and cost are unknown. The cost
   ratio above uses input tokens, which are known. The provider's balance after the round was
   17.91 USD.

3. **Scratch files counted as code.** The preamble asked the agent to keep scratch files in the
   current directory, so that runs would not share `/tmp`. The agent then wrote test servers,
   logs, and built binaries in `scratch/` in the workspace. The grader checks all Go code of the
   workspace, so the two rule breaks of arm G (T02, runs 1 and 2) are findings in test servers in
   `scratch/`, not in the product code. The diff size counted the binaries (up to 125,000 lines in a
   run of T02 G). The [sensitivity report](data/p5-deepseek/analysis-noscratch.md)
   does not count the breaks in `scratch/`: clean success, tempting tasks, is then −25.0 points.

4. **Runs still shared the user's temp directory.** Git Bash maps `/tmp` to the user's temp
   directory whatever `TMP` says. 5 runs (2 V, 3 G) wrote there; 3 of them wrote files of the
   same names (`fakepay.log`, `stockroom.log`), so a later run could have read an earlier run's
   log. All 5 runs succeeded. The files were moved out of the temp directory after the round.

## What the round does show

- **The tasks did not tempt this model to break the rules.** In arm G, with the rules only in
  prose, the model broke no rule in the product code in 36 runs. With no breaks to prevent,
  Vulcan's enforcement had nothing to win in this round. The tempting tasks were not tempting
  enough, or `RULES.md` was enough for this model.
- **Vulcan runs did not stop by themselves.** V runs reached their first passing check sooner
  (median turn 10.5 against 13.0), but then went on, and most met the budget; G runs more often
  ended on their own. A passing check is not a finished task: the base code passes the check,
  and both T03 runs of V that failed had a passing check and no change.
- **The pipeline works at scale.** 72 runs on a provider other than Anthropic, with no provider
  failure, the sealed material opened only to grade (its manifest checked against the committed
  hash), and the frozen analysis run unchanged.

## Limits

- One model (deepseek-v4-pro through an Anthropic-compatible endpoint). Claude Code did not know
  the model (`unrecognized_model`), and its cost figures use Claude's prices.
- 12 tasks, 3 runs each: the intervals are wide.
- The shell of a run was not restricted (pilot-p5.md, decision 1).
- Strict mode was added after the hidden material was written, by the owner's decision; arm V
  enforces 2 of the 16 rules (R03, R05).

## Next

The next round must answer the question this round could not: does Vulcan stop rule breaks that
plain Go does not, at a cost worth paying? The plan for it is to be written: the litmus test of
value against Go. Before it, the harness needs a budget in real turns, scratch files outside the
graded code, and a temp directory that no other run can see.
