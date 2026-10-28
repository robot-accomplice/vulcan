# The PoC harness

The harness runs, grades, and analyzes the evaluation of [docs/poc/plan.md](../../docs/poc/plan.md)
(sections 6 to 9). It is one command, `harness`, built from `poc/harness/cmd/harness`.

| Package | Job |
|---|---|
| `record` | The record of one run (`runs.jsonl`), and the task and rule metadata (`meta.json`) |
| `agentlog` | Reads the JSON stream of `claude -p --output-format stream-json --verbose`: turns (one per message: Claude Code writes one event per content block), tool calls, tokens, cost, and each run of the arm's check command with its Vulcan codes |
| `runner` | One run: a fresh copy of the arm's workspace with `RULES.md`, the agent in it, the budget (60 turns, 45 minutes, a cost cap), the stream, and the diff (M7) |
| `schedule` | Every task, arm, and run index, in a random order fixed by a seed |
| `grade` | Builds the final workspace, and runs the acceptance tests, the rule tests and detectors, and the holdout tests |
| `detect` | The library of the structural detectors: the final source with types, by unit, in both arms |
| `analysis` | The analysis script of round 1, frozen before the runs: per-task means, paired differences, bootstrap intervals, exact sign-flip p-values, and the verdict of section 9 |
| `analysis2` | The analysis script of round 2 ([plan-r2.md](../../docs/poc/plan-r2.md), section 3), frozen before its runs: arms P, L, and V; decision 6's violation tests and t-bounds; the validity gate and the verdicts |
| `synth` | Synthetic runs with a known effect, for round 1's dry run |
| `synth2` | Synthetic runs of round 2, one world per verdict, for its dry run |
| `measure` | Round 2's code measures (O7): lines in duplicated blocks, and dropped or masked errors |
| `provider` | The balance of the provider's account (DeepSeek, OpenRouter, Moonshot), for the spend of a batch |
| `seal` | Seals the hidden material (AES-256-GCM) and lists its SHA-256 manifest |
| `fakeagent` | Stands in for Claude Code in tests of the harness |
| `testdata/material` | Demo material in the format of the hidden material (not the hidden material) |

## The agent

Each run calls headless Claude Code in the run's workspace:

```
claude -p --output-format stream-json --verbose --include-partial-messages
       --setting-sources local --strict-mcp-config --mcp-config {"mcpServers":{}}
       --disallowedTools WebFetch,WebSearch --permission-mode dontAsk
       --allowedTools <runner.AllowedTools> --add-dir <scratch>
       [--resume SESSION] [--model M] [--max-budget-usd X] <prompt>
```

With `-base-url URL` and `-token-file F` (or `VULCAN_AGENT_TOKEN`), Claude Code talks to another
provider's Anthropic-compatible endpoint, and every model role is the `-model` of the run. The
cost that Claude Code reports, and so `--max-budget-usd`, uses Claude prices; the token counts
are exact.

No user or project settings load, so no one's CLAUDE.md or memories reach the agent; no MCP
server runs; the web is off. Go modules come from the local cache only (`GOPROXY=off`). The
`vulcan` and `fakepay` commands are first on the PATH. The prompt is `runner.Preamble` and the
task brief, the same in both arms except the language name. The runner stops a run at 60 turns
or 45 minutes; `--max-budget-usd` is the token cap.

The agent has only the tools of `runner.AllowedTools`, the same in both arms: the file tools,
the task tools of background commands, and the shell. Claude Code denies every other tool without
a prompt (`dontAsk`), so there are no web tools and no MCP tools.

Limit: the shell is not restricted. The agent can run any command with the user's rights on the
machine, and `curl` or `go` can reach the web; the briefs need neither. Claude Code allows shell
loops only when the whole shell is allowed, and the owner chose loops over a command allowlist
(experiments/poc/pilot-p5.md). Keep the hidden material sealed while the runs go on; it is opened
only to grade, and it is not on any path that the workspace names.

## Round 2

Round 2 ([plan-r2.md](../../docs/poc/plan-r2.md)) has three arms: P (plain Go, `poc/app/go`), L
(Go with Vulcan's model as a linter and primer, `poc/app/golint`, or `-template-l`), and V. Its
fixes to round 1 (plan-r2, section 6):

| # | What | Where |
|---|---|---|
| F1 | One turn per message | `agentlog` |
| F2 | Output tokens of a stopped run from `--include-partial-messages`; cost at the provider's prices with `-prices FILE` (`{"model": {"input": …, "cached_input": …, "output": …}}`, USD per million tokens) | `agentlog`, `runner.Price` |
| F3 | Scratch files in `<run>/scratch`, outside the workspace (`--add-dir`, named in the preamble); the diff counts only product source (no new top-level names, no binary files); detectors read only `api/`, `lib/`, `cmd/`; leftovers are listed as `stray_files` | `runner`, `detect.Product` |
| F4 | `-sandbox DISTRO`: each agent round runs in WSL in its own user, mount, and PID namespace (`unshare --map-root-user`, no root needed): its own `/tmp`; the user's home, `/mnt`, `/var/tmp`, and `/dev/shm` hidden; the run's directory at `/tmp/run`; the Linux tools (`-linux-tools`, default `.harness/tools-linux`: `bin/` with claude, vulcan, fakepay, vulcan-lint, and `go/`, a Go toolchain) read-only; Go modules from a read-only file proxy of the host's module cache. The provider's key reaches the namespace through `WSLENV`, never through a file | `runner/sandbox.go` |
| F5 | `first_edit_turn`, `first_pass_turn` (the first passing check after the first edit), `tokens_in_at_first_pass` | `agentlog` |
| F6 | The preamble ends: "When the check passes and the request is met, stop." | `runner.Preamble` |
| F7 | `config_changes`: a changed file that the template's `PROTECTED` file lists, a changed `config` block of `system.vulcan`, or new in-source suppressions (`//nolint`, `//lint:ignore`, `//vulcan-lint:ignore`). Such a run is no clean success unless its task has `allow_config` | `runner/config.go` |
| F8 | The provider's balance before a batch, every 10 runs, and at the end, in `balance.jsonl` | `provider`, `evaluate` |
| F9 | `-review`: after each agent round, a reviewer session (read-only tools, `runner.ReviewPrompt`, the brief, `RULES.md`, the diff) gives a verdict; a rejection goes back to the agent in the same session (`--resume`), up to `-resubmits` times within the run's budget; a second review of the final change measures the reviewer's agreement | `runner/review.go` |
| F10 | `harness analyze2`, and its dry run: `go test ./poc/harness/analysis2` reads each world of `synth2` over 200 seeds | `analysis2`, `synth2` |

Arm L's linter, written in step 2 of plan-r2, must print a code `VL` and four digits in each
diagnostic, as Vulcan prints K-codes, so that the check loop (O2) reads both arms alike; and arm
L's template must list its configuration (the manifest, the linter's settings) in `PROTECTED`.

The sandbox needs two things that `harness tools -linux` does not fetch: Claude Code for Linux
in `bin/claude`, and a Go toolchain in `go/`, of at least the version of `go.mod`.

```bash
harness tools -linux                           # vulcan, fakepay, fakeagent for the sandbox
harness schedule -tasks T01,...,T24 -ord-tasks T25,...,T32 -arms P,L,V -runs 3 -ord-runs 5 -retest 0.1 -seed SEED > schedule.json
harness evaluate -schedule schedule.json -material PUBLIC -no-grade -out RUNS -model M -base-url URL -token-file F     -sandbox Ubuntu-22.04 -review -prices prices.json
harness gradeall -material MATERIAL -out RUNS
harness analyze2 -runs RUNS/runs.jsonl -meta MATERIAL/meta.json > report.md
```

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
harness gradeall -material MATERIAL -out RUNS    # grade runs made with evaluate -no-grade
harness restream -out RUNS                      # read each run's stream again (a fixed reader)
harness analyze -runs RUNS/runs.jsonl -meta MATERIAL/meta.json > report.md
```

`evaluate` runs and grades each slot that `runs.jsonl` does not hold yet, so an interrupted
evaluation resumes. A run blocked by a compiler bug is marked by hand (`toolchain_bug`), and a run
whose infrastructure failed is marked `excluded`. A run that the runner stops has no final result
in its stream, so its output tokens and cost are not known (`usage_partial`).

## Checks

- `go test ./poc/harness/...`: the stream parser, the diff, sealing, and the analysis on four
  synthetic scenarios (promising, no signal, costly, a tool may do it), each read right; for
  round 2, the review loop with the scripted agent, the configuration checks, the code measures,
  and the dry run of `analysis2`.
- `VULCAN_TEST_WSL=Ubuntu-22.04 go test ./poc/harness/runner`: the sandbox on a real WSL
  distribution: no Windows drive, no home, read-only tools, a private `/tmp`, and no key on disk.
- The dry run: `harness synth -scenario promising -out DIR`, then `harness analyze`.
- The pipeline: `harness evaluate` with `-agent fakeagent` on `testdata/material`. With
  `FAKEAGENT_BREAK=1` the fake agent opens a database in orders: the detector flags it in both
  arms, and arm V's check fails with K0205.
