# Round 2: the sandbox pilot with the scripted agent

Run 2026-10-28, step 1 of [plan-r2.md](../../docs/poc/plan-r2.md). It checks the harness of round
2 end to end before any model runs: the WSL sandbox (F4), the review loop (F9), and grading,
with the scripted agent `poc/harness/fakeagent` on the demo material
(`poc/harness/testdata/material`). No model and no hidden material took part.

## Tools of the sandbox

| Tool | Version | Source | Check |
|---|---|---|---|
| Go | 1.27.3 linux/amd64 | `go.dev/dl/go1.27.3.linux-amd64.tar.gz` | SHA-256 `65efbc80fdc1fa2cd2631cad5b481777a68e8725f34e89f50fc6ac714e379b40`, from go.dev's release list |
| Claude Code | 2.1.312, linux-x64 | `downloads.claude.ai/claude-code-releases/2.1.312/linux-x64/claude` | SHA-256 `4f4178ce41d83e23341ebfbaecdbcf8ea6bf7baea0690025abe49eeec3a5d01d`, from the release's `manifest.json` |
| vulcan, fakepay, fakeagent, the runtime module | this repository | `harness tools -linux` | built from source |

They live in `.harness/tools-linux` (not committed). The WSL distribution is Ubuntu 22.04.

## What ran

`harness evaluate -agent fakeagent -sandbox Ubuntu-22.04 -review` on one demo task, arms P and V,
one run each. The scripted reviewer rejects the first review and accepts the rest.

| Arm | Turns | Checks | Reviews | Builds | Acceptance | Holdout | Rules broken |
|---|---|---|---|---|---|---|---|
| P | 2 | 2, both pass | reject, accept, then the second review: accept | yes | 2 of 2 | 1 of 1 | none |
| V | 1 | 1, passes | accept, then the second review: accept | yes | 2 of 2 | 1 of 1 | none |

In arm P the rejection went back to the agent in the same session (`--resume`), and its tokens
add up over the two rounds. No process of a run was left in WSL after it ended, and the real WSL
`/tmp` holds nothing from the runs. `go test ./poc/harness/runner` with `VULCAN_TEST_WSL` also
checks that a run sees no Windows drive and no home, that its tools are read-only, and that the
provider's key is in no file.

## Faults the pilot found, fixed before this record

1. `vulcan` could not find its runtime module in the sandbox: `harness tools -linux` now copies
   it, and the sandbox sets `VULCAN_RUNTIME`.
2. The grader built from the agent's `.vulcan/` output, which names sandbox paths: it now removes
   that generated directory and builds from the source.
3. A relative tool directory broke the grader's build (a command's relative path is resolved
   against its working directory): the tool directory is now absolute, and the build log keeps
   the error.
4. `wsl.exe --` passes its arguments through a shell, which loses quoting: the sandbox now uses
   `wsl.exe --exec`.
5. The scripted agent named its messages by process id, which is the same in every PID namespace:
   it now names them by time. Real Claude Code messages have unique ids.

## Not yet checked

A run of a real model in the sandbox: its first one is the pilot of step 7, which also prices a
run. Arm L cannot run before its template exists (step 2).
