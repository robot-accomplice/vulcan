# The PoC harness

The harness runs, grades, and analyzes the evaluation of [docs/poc/plan.md](../../docs/poc/plan.md)
(sections 6 to 9). It is one command, `harness`, built from `poc/harness/cmd/harness`.

| Package | Job |
|---|---|
| `record` | The record of one run (`runs.jsonl`), and the task and rule metadata (`meta.json`) |
| `agentlog` | Reads the JSON stream of `claude -p --output-format stream-json --verbose`: turns, tool calls, tokens, cost, and each run of the arm's check command with its Vulcan codes |
| `runner` | One run: a fresh copy of the arm's workspace with `RULES.md`, the agent in it, the budget (60 turns, 45 minutes, a cost cap), the stream, and the diff (M7) |
| `schedule` | Every task, arm, and run index, in a random order fixed by a seed |
| `grade` | Builds the final workspace, and runs the acceptance tests, the rule tests and detectors, and the holdout tests |
| `detect` | The library of the structural detectors: the final source with types, by unit, in both arms |
| `analysis` | The analysis script, frozen before the runs: per-task means, paired differences, bootstrap intervals, exact sign-flip p-values, and the verdict of section 9 |
| `synth` | Synthetic runs with a known effect, for the dry run |
| `seal` | Seals the hidden material (AES-256-GCM) and lists its SHA-256 manifest |
| `fakeagent` | Stands in for Claude Code in tests of the harness |
| `testdata/material` | Demo material in the format of the hidden material (not the hidden material) |

## The agent

Each run calls headless Claude Code in the run's workspace:

```
claude -p --output-format stream-json --verbose --setting-sources local --strict-mcp-config
       --mcp-config {"mcpServers":{}} --disallowedTools WebFetch,WebSearch
       --permission-mode bypassPermissions [--model M] [--max-budget-usd X] <prompt>
```

No user or project settings load, so no one's CLAUDE.md or memories reach the agent; no MCP
server runs; the web is off. Go modules come from the local cache only (`GOPROXY=off`). The
`vulcan` and `fakepay` commands are first on the PATH. The prompt is `runner.Preamble` and the
task brief, the same in both arms except the language name. The runner stops a run at 60 turns
or 45 minutes; `--max-budget-usd` is the token cap.

Limit: the agent runs with the user's rights on the machine. Keep the hidden material sealed
while the runs go on; it is opened only to grade.

## The hidden material

```
meta.json               tasks (id, kind, targets) and rules (id, class, method, labels)
RULES.md                the team rules, copied into each workspace
tasks/<id>/brief.md     the brief
tasks/<id>/accept/      acceptance tests (a Go test package using poc/stockroomtest)
rules/<id>/             a rule test (Go test package) or a detector (main package using detect)
holdout/                holdout tests (a Go test package)
```

A detector exits 1 when it finds a violation, 0 when it finds none, and 2 when it cannot read
the source.

## Commands

```bash
harness tools                                  # build vulcan and fakepay into .harness/tools
harness newkey                                 # a key for VULCAN_SEAL_KEY
harness manifest -dir MATERIAL > hashes.txt    # commit before the runs
harness seal -dir MATERIAL -out material.sealed
harness schedule -tasks T01,...,T12 -runs 3 -seed SEED > schedule.json
harness open -in material.sealed -dir MATERIAL # only to grade
harness evaluate -schedule schedule.json -material MATERIAL -out RUNS -model MODEL -budget-usd X
harness analyze -runs RUNS/runs.jsonl -meta MATERIAL/meta.json > report.md
```

`evaluate` runs and grades each slot that `runs.jsonl` does not hold yet, so an interrupted
evaluation resumes. A run blocked by a compiler bug is marked by hand (`toolchain_bug`), and a run
whose infrastructure failed is marked `excluded`.

## Checks

- `go test ./poc/harness/...`: the stream parser, the diff, sealing, and the analysis on four
  synthetic scenarios (promising, no signal, costly, a tool may do it), each read right.
- The dry run: `harness synth -scenario promising -out DIR`, then `harness analyze`.
- The pipeline: `harness evaluate` with `-agent fakeagent` on `testdata/material`. With
  `FAKEAGENT_BREAK=1` the fake agent opens a database in orders: the detector flags it in both
  arms, and arm V's check fails with K0205.
