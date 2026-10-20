# Decision 5: names and reproducibility

Status: accepted, 2026-10-20. The owner chose `.vulcan`, `vulcan`, and the runtime module at
`github.com/robot-accomplice/vulcan/vulcan`, and accepted items 5 and 6. Applied in the rename of
the spec and the primer, §6, §11, §13 (Q1), and Appendix E (K0705). Milestone 0, item 5 of the
[development plan](../development-plan.md). Answers Q1 (§13) and audit 2, L3.

## Question

What are the source file extension, the command name, and the module paths of the runtime? Which
Go versions and host platforms does the first release support? How does a project pin all of
them, so that two builds of one commit agree?

## Current rule

- The language is Vulcan (decided 2026-10-19). The spec and the primer still use the working
  name Keel: the extension `.keel` (64 uses in the spec), the command `keel` (72), the runtime
  path `keel.dev/keel` (9, an invented domain), the generated prefixes `keel_` and `Keel_`
  (37), and the queue directory `keel-queues`.
- The spec names no Go version. The test 3 fixtures say `go 1.22`. The local toolchain is Go
  1.26.1.
- "Each Keel version pins the version of each analysis tool" (§6).
- `keel test` runs the race detector by default (§6). On this Windows host, `go test -race`
  fails: "-race requires cgo".
- The repository is `github.com/robot-accomplice/vulcan`.

## Facts checked

- No active programming language is called Vulcan. The closest are proto-vulcan, a small
  logic language embedded in Rust, and old languages of the same name from the 1970s and 1980s.
- `.vul` is the extension of Vulpin, a small scripting language. `.v` belongs to Verilog, Coq,
  and the V language.
- An npm package, `vulcan-ui`, installs a command named `vulcan`.
- Go 1.24 added the `tool` directive to `go.mod`, so a module can pin an executable tool and run
  it with `go tool <name>`.
- Go supports each major release until two newer ones exist. Go 1.26 shipped in February 2026.
  Go 1.27 was due in August 2026; this decision does not rely on its exact date.

## Decisions to make

### 1. The source file extension

| Option | For | Against |
|---|---|---|
| **`.vulcan` (recommended)** | No known use. Clear in a file list and in a search. | Longer than most extensions. Tools do not care. |
| `.vn` | Short. No known use found. | Easy to mistake; says nothing. |
| `.vul` | Short. | Used by Vulpin. |

The declaration files become `system.vulcan`, `component.vulcan`, and `library.vulcan`. Test files
are `*_test.vulcan`. `topics.lock` keeps its name, because it names what it holds.

### 2. The command name

| Option | For | Against |
|---|---|---|
| **`vulcan` (recommended)** | Matches the language. Projects run it as `go tool vulcan` (item 5), so a different `vulcan` on `PATH` does not matter to the build. | `vulcan-ui` on npm installs a command with the same name. |
| `vlc` | Short. | VLC media player. |
| `vk` | Short. | Says nothing; near Vulkan. |

### 3. Module paths

The runtime must be a real module, so that `go mod download` works. It should not pull in the
compiler's dependencies.

| Option | For | Against |
|---|---|---|
| **GitHub paths (recommended)** | Work today. The owner controls them. | A move to another host changes every import. |
| A vanity domain, for example `vulcan-lang.org/vulcan` | Independent of the host. | Needs a domain the owner holds, and a server for the `go-import` tag. Changing paths after release breaks every user. |

With GitHub paths:

| Today | Recommended |
|---|---|
| (compiler, not named) | module `github.com/robot-accomplice/vulcan`, command at `cmd/vulcan` |
| `keel.dev/keel` | module and package `github.com/robot-accomplice/vulcan/vulcan`; code writes `vulcan.Bind`, `vulcan.DeliveryID` |
| `keel.dev/keel/rt` | `github.com/robot-accomplice/vulcan/vulcan/rt` |
| `keel.dev/keel/transport` | `github.com/robot-accomplice/vulcan/vulcan/transport` |
| `keel.dev/keel/transport/direct` | `github.com/robot-accomplice/vulcan/vulcan/transport/direct` |

The runtime is its own module, so a program's `go.sum` holds only the runtime and its
dependencies. A vanity path can still be chosen before the first tagged release. After that, the
path is fixed.

### 4. Names in generated code and the runtime

- The reserved prefixes `keel_` and `Keel_` become `vulcan_` and `Vulcan_` (Appendix B.1, F).
- The default queue directory becomes `./vulcan-queues` (§5.5.1).
- `keel.ContractFailure`, `keel.DeadLetter`, and the other runtime names keep their names, with
  the qualifier `vulcan`.
- The experiment files keep their old names. They are records.

### 5. Go versions and pinning

1. **Supported Go versions.** Each Vulcan release supports the Go releases that the Go project
   supports on the day of the release: the newest two. The first release supports Go 1.26 and the
   newest release after it. A Vulcan release names this range in its notes.
2. **Go syntax follows the compiler's `go/types`.** Bodies are Go (§6). The Go syntax that a body
   can use is the syntax of the `go/types` that the Vulcan release is built with (decision 4).
   Thus a new Go feature, for example generic methods, works in bodies once a Vulcan release is
   built with a Go version that has it.
3. **A project pins everything in `go.mod`.** The `go` line sets the language version, the
   `toolchain` line the exact Go toolchain, and a `tool` line the exact Vulcan compiler:

   ```
   module shop

   go 1.26
   toolchain go1.26.1

   tool github.com/robot-accomplice/vulcan/cmd/vulcan

   require (
       github.com/robot-accomplice/vulcan v0.1.0
       github.com/robot-accomplice/vulcan/vulcan v0.1.0
   )
   ```

   `go tool vulcan build` then runs the pinned compiler. The analysis tools of §6 are Go module
   dependencies of the compiler, so the same line pins them.
4. **Builds are reproducible.** For one commit, one Vulcan version, and one toolchain, the
   generated Go is the same bytes: the compiler sorts every generated list, writes no time or
   path, and builds with `-trimpath`. Go records the module versions in the binary
   (`debug.BuildInfo`), so a binary names the Vulcan version that built it.

### 6. Host platforms

1. **The compiler is tested on four hosts:** `linux/amd64`, `linux/arm64`, `darwin/arm64`, and
   `windows/amd64`. Other hosts that Go supports may work, and are not tested.
2. **Targets are Go's targets.** A program can be built for any target of the pinned Go
   toolchain. CI runs the conformance suite (development plan, milestone 2) on the four tested
   hosts.
3. **The race detector where the host has it.** `go test -race` needs cgo on some hosts, for
   example Windows. Where the host cannot run it, `vulcan test` runs without it, prints a warning
   (new K0705), and `vulcan audit` lists the run, as for `--race=false` (§6). A project that
   relies on race detection runs it in CI on a host that has it. The development plan already
   requires this.

## Recommendation

`.vulcan`, `vulcan`, GitHub module paths with a separate runtime module, `vulcan_` prefixes, the
Go version policy of item 5, and the four hosts of item 6.

## Spec changes on acceptance

One change renames Keel to Vulcan in `docs/spec.md` and `docs/guide.md`: the name, the extension,
the command, the module paths, the prefixes, and the queue directory. In the same change:

- Q1 (§13) is closed, and points to this decision.
- §6: the Go version policy, `go.mod` pinning, and the race-detector rule (K0705).
- §11: the tested hosts and reproducible builds.
- Appendix E: K0705 (warning).
- The README and `docs/status.md` drop the note that the spec uses the working name.
- The experiments keep their names.

## Open points

- **A vanity path.** Choose it before the first tagged release, or keep the GitHub paths for good.
- **`vulcan/vulcan` reads twice.** The repeated element is the cost of a runtime package named
  `vulcan` in a separate module. The alternative, a runtime at the module root and the compiler
  in a second module, gives the shorter `github.com/robot-accomplice/vulcan` to user code instead.
  Both work.
