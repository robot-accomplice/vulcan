# Timing harness for rows 15 and 19

Code and config for the dry run of phase A and the timing campaign of
[plan-coverage.md](../../../docs/poc/plan-coverage.md), section 1, rule 6 (rule 7 for machines, rule 13
for amendments). No output is in the repository. Everything that carries an arm's direction goes to a raw
directory under `C:\Users\jonat\vulcan-timing\`, is sealed with `.harness/harness.exe seal`, and the raw
directory is deleted.

| File | What |
|---|---|
| `commands.json` | The commands per arm, the unit of each tool, the closed lists, the drop command, the tracer, the load gate, the warm-up constants. JSON with full-line `//` comments |
| `common.py` | Restore, program-file read, tool run, findings filter, load gate, plan, raw directory, seal, output guard, trace parser |
| `row15.py` | `setup`, `run`, `trace`, `confirm` |
| `row19.py` | `setup`, `run`, `confirm` |
| `reader.py` | The procedure reader's view of a sealed file: tool output and file counts, no time and no order |
| `commands.fake.json`, `changes.fake.json`, `requests.fake.json`, `fake/` | A FAKE arm set for `test_fake.py`. Never for a measurement |
| `test_fake.py` | 94 checks on the fake set |
| `changes.example.json`, `requests.example.json` | The format of the two files that the owner and the fresh sessions supply |

## What a command prints

Only these lines. None shows a time, a count or an order for one arm.

```
runs: <N over all arms>
faults: none            or, one line each, sorted and without repeats:   fault: <arm> <kind> [tool <name>]
row15_total_wall_s: <sum over all arms>   (row19.py: row19_total_wall_s; row15.py trace: trace_total_wall_s)
```

`emit()` in `common.py` stops the harness if a line has another shape. A fault line names the arm, the kind
and the tool, as the plan allows for the procedure reader. It has no run number, so it has no count.
`confirm` is the exception: it is an untimed check of the commands, and prints the exit status and the count
of findings of each tool (row 15) or one status code (row 19), and never a duration.

## Steps

Set the key first. It is read from the environment only and is never printed.

```bash
export VULCAN_SEAL_KEY=$(.harness/harness.exe newkey)      # keep this key: the sealed files need it
cd experiments/poc/timing
```

1. Edit `commands.json`: `"machine"` (machine 1 or 2, rule 7), `"drop"` and `"tracer"` (see below).
2. Put the 40 changes in `changes.json` (patches next to it) and the request paths in `requests.json`.
3. `python row15.py confirm --audit` and `python row19.py setup` then `python row19.py confirm`: check the
   commands work, untimed. `--audit` lists paths that a check wrote outside the closed list.
4. `python row15.py setup`: copies each base app to a work tree, runs one full check of the unchanged base,
   snapshots the closed list, and runs the check of every change on the unchanged tree for the baseline
   (status and findings per command). The baselines come after the snapshot, and the caches are restored at
   the end, so the snapshot is the state just after one full check.
5. `python row15.py run --label dry15-m1 --seed S --batch 1 --n 20`: one batch of 20 draws, each in the 3 arms
   in a random order. Same seed and batch give the same draws and orders, on both machines.
   `python row19.py run --label dry19-m1 --seed S --batch 1 --n 20` is the same for row 19.
6. `python row15.py trace --label trace-m1 --all` (phase A': every change in every arm) or `--n 20 --seed S`
   (the dry run's 20 draws). Machine 1 only.

Run the same on machine 2 with its own `commands.json` `"machine"` and the same seed.

One run of row 15 is steps (1) restore the closed list by `robocopy /MIR` from the snapshot, (2) the drop
command, (3) read every program file, (4) `git apply` of the arm's patch, (5) the check. The check is the
tools of the arm run one after the other in the order of `commands.json`; each tool's exit status and output
are kept. The change is reverted with `git apply -R` and the tree is compared by hash with the pristine copy
(a mismatch is a fault).

A run is clean when each tool has the exit status and the findings of the same command on the unchanged base.
Findings are the output lines with line and column numbers removed (a one-file change shifts them), after the
one filter of rule 2 (a line about a file that starts with `// Code generated ... DO NOT EDIT.` is dropped).

## Load gate

Before step 1 of each timed run, each trace run and each row-19 run, and outside the timed interval,
`common.load_gate` samples the machine for `sample_s` (3 s). The run starts only when the total CPU use is at
most `cpu_max_pct` (25) and the free memory is at least `mem_free_min_gb` (8, in GiB, as psutil `available`).
It samples again until `wait_max_s` (120 s) has passed. The limits are in the `load_gate` object of the config.
The record of each run keeps `load_gate`: `passed`, `waited_s` and every sample (`cpu_pct`, `free_gb`), and is
sealed with the run.

A run that never passes is not run and is a fault (`fault: <arm> load-gate`). Its slot still takes a run
number and a record (`"skipped": "load-gate"`, no tools, no time), so **`runs:` counts the planned slots, run
or skipped**, as it already counted a row-19 run that failed. The number of skipped runs is the number of
`load-gate` faults of the raw (the printed fault line has no count).

## Reader view

```bash
python reader.py --sealed <label>.sealed --out <empty dir> [--config commands.json]
```

The sealed file is opened (key from `VULCAN_SEAL_KEY`) into a temporary directory that is deleted at the end.
Only a whitelist of fields is written to `--out`: per run its label, `run`, `arm`, `change` (row 15) or `path`
(row 19), each tool's `name`, `status` and kept `output`, and the `counts` (`files_opened`,
`cache_files_opened`) of a trace run; and in `index.json` the arm list of the plan and the faults of the raw
(`faults.json`, written by the seal when there is one). No time, draw, position, seed, load sample, latency
figure, unsettled flag or manifest time is copied. `run` is the number in the sorted list (arm, then change
id, then a hash of the run's own content), not the harness's run number, because that number is the run order.
The output is one `run-NNNNN.json` per run and `index.json`.

## The commands per arm and the unit of each tool

| Arm | Tool | Command | Unit |
|---|---|---|---|
| P, L | go vet | `go vet {pkgs}` | package |
| P, L | staticcheck | `staticcheck {pkgs}` | package |
| P, L | errcheck | `errcheck {pkgs}` | package |
| P, L | exhaustive | `exhaustive {pkgs}` | package |
| P, L | dupl | `dupl .` | module (it compares files with each other) |
| P, L | gopls check | `gopls check {files}` | file |
| L | vulcan-lint | `vulcan-lint ./...` | whole program |
| V | vulcan check | `vulcan check .` | whole program |

`{pkgs}` is the package directory of each changed file. In a full check it is `./...`.

## Caches and program files (closed lists)

Every arm has `GOCACHE`, `GOMODCACHE`, `LOCALAPPDATA` (staticcheck, gopls and gopls' goimports index write
under it on Windows) and `XDG_CACHE_HOME` under its own `arms\<arm>\cache\`. V adds `<tree>\.vulcan`
(its lowered Go and its built program). L has no cache of its own: vulcan-lint is a `go/analysis`
multichecker that uses the Go build cache. P has no architecture linter yet. Program files: the Go
toolchain (`bin`, `pkg`, `src`, `lib`, `VERSION`, `go.env`), the tool executables, and for V the runtime
module `vulcan\` that `vulcan check` reads. The parity audit sets the final lists; a path outside them is a
finding.

## What still needs the owner

- **Admin and EULA.** The drop (RAMMap64 `-Et`, empties the standby list) and Process Monitor need an
  administrator shell, and the Sysinternals EULA must be accepted once by hand. The harness never passes
  `-accepteula`. The intended commands are in `commands.json` as comments. Until set, the defaults fail
  loudly: a run stops at its first drop.
- **The 40 changes** (`changes.json` and patches, rule: no change of a signature or an export, and each
  builds and ends clean in every arm) and **the request paths** (`requests.json`), from the fresh sessions
  of the plan.
- **The machine names** and the same hardware and system for the whole column.
- **Decisions the plan does not settle** are listed in the report of the builder: the unit of vulcan-lint,
  V's `go vet ./...`, the definition of a read for Procmon, and "one command" for the check.

## Not built

- The audit of **program files loaded** (it needs the tracer; the closed list of program files is by
  directory).
- The row-19 app for P with assertions (phase A'); the dry run uses the base code.
