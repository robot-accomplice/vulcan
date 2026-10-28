# Run data: where it is kept, and for how long

The raw data of the runs cannot be in this repository. The streams and diffs show the hidden
tasks and rules. They are kept sealed (AES-256-GCM, `harness seal`, the key of the hidden
material) on a second disk, and their hashes are here, so that any later analysis can show it
read the same data.

## Copies

| Copy | Where | What |
|---|---|---|
| Working copy | `C:\Users\jonat\vulcan-poc\runs` (system disk) | Everything, unsealed, with built binaries (3.5 GB) |
| Archive | `D:\vulcan-poc-archive\2026-10-28` (second physical disk) | Each run set sealed, its manifest, `material.sealed`, and `SHA256SUMS` |

The archive leaves out only what can be built again: each run's `bin/` (tool copies), `tmp/`,
the grader's `prepared/`, and `.exe` files. It keeps each run's `run.json`, `stream.jsonl`,
`diff.txt`, `grade.log`, `agent.err`, `base/`, the final `workspace/` source, and each set's
`runs.jsonl`, logs, and `meta.json`. Each archive was opened again after sealing and matched its
manifest.

| Sealed file | SHA-256 |
|---|---|
| `runs-main.sealed` (round 1, 72 runs) | `f3c2ed827b5f934c151e5074012d2736e0694175c3448fa4808b6604c606f77c` |
| `runs-main-sonnet-stopped.sealed` (34 run folders; the start that was stopped, not graded) | `7b1b5fc7dbc58902da780ab599385a921cb29fd89a0168d115a853b322f882ae` |
| `runs-pilot.sealed` (Sonnet pilots) | `43fa76b75ad523a17aeb9409de91fc531a204f2c45f315086542d89dda1a4b77` |
| `runs-pilot-deepseek.sealed` | `e4b01f0dd5dadc321c22c783fba4a308fc3889a56a18ed82c7cbed9618605a30` |
| `runs-pilot-kimi-moonshot-overloaded.sealed` | `dd045bca7b4f4b2119f0cf6f2c3e405f429d255eec60ebefe81617728d1024d6` |
| `material.sealed` (the hidden material) | `d89b3e08473e9fe28229052308a6022d31f6138e1c5da43a6a99d11347cb2bcd` |

To open one: `VULCAN_SEAL_KEY=… harness open -in runs-main.sealed -dir DIR`, then
`harness manifest -dir DIR` and compare with `runs-main.manifest.txt`.

**The seal key is the single point of failure.** Without it no archive can be read. It must have
a copy outside this machine (a password manager), never in this repository.

## Retention

1. The sealed archives are kept until the research ends and its results are published or
   dropped. They are small (31 MB).
2. The working copy is kept until each item below is done or dropped with a reason written here.
   Only then may it be deleted; the archive stays.
3. Each new round adds its own dated folder to the archive, its hashes to this file, and its
   open items below, before any working copy is deleted.

## Not yet taken from round 1

| Item | Data | Who may do it |
|---|---|---|
| M9: false positives of Vulcan diagnostics (plan section 8), by a judge session | V streams, check outputs | A fresh session (it reads task text) |
| Why V runs did not stop after the task was done: what they did in the turns after the first passing check | V streams | A fresh session |
| Did any run read another run's files in the shared `/tmp` (fault 4) | Streams of the 5 runs, `temp-leftovers/` | Builder (file names only) |
| Use of the primer: when and how often V runs called `vulcan guide`, and what they did next | V streams | Builder (counts) |
| The stopped Sonnet runs (34 folders): grade them as exploratory data, never counted, for a first look at a second model | `runs-main-sonnet-stopped` | Builder |
| Time to the first edit, and turns spent reading, by arm | All streams | Builder (counts) |
