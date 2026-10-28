# Data of round 1 (deepseek-v4-pro)

| File | What it holds |
|---|---|
| `runs.jsonl` | One record a run (`poc/harness/record`), graded, with the stream counts read again by `harness restream` |
| `runs.noscratch.jsonl` | The same, without the two rule breaks found only in `scratch/` (T02 G, runs 1 and 2) |
| `meta.json` | The labels of the hidden material: task kinds and targets, rule classes and methods. The rule texts are left out |
| `analysis.md` | `harness analyze -runs runs.jsonl -meta meta.json` |
| `analysis-noscratch.md` | The same on `runs.noscratch.jsonl` |
| `first-pass.json` | Per run: task, the message of the first passing check, messages, budget hit |

The streams and diffs of the runs are not here: they show the hidden tasks and rules. They stay
outside the repository with the sealed material. The schedule is
[schedule-main.json](../../schedule-main.json).
