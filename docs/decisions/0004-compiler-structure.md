# Decision 4: compiler structure

Status: accepted, 2026-10-20 (option B). Applied in spec §10, §11, §11.1, §11.2, and
Appendix B.1. Milestone 0, item 4 of the
[development plan](../development-plan.md). Answers [audit 2](../../experiments/audit-2.md), H11.

## Question

How is the first compiler built? The spec requires a custom incremental query engine "from the
start" (§11.1) and its own IR for every backend (§11). The development plan proposes explicit
stages with reusable results, and an incremental engine only after measurement.

## Current rule

- §11.1: the compiler is written in Go, and later self-hosted. "The compiler is query-based and
  incremental from the start. It is a memoized query engine, not a batch pipeline. [...] Go has no
  equivalent of the Rust library `salsa`, so the project builds this engine itself."
- §11: the compiler lowers to its own IR, and each backend reads only the IR. The LLVM backend is
  deferred.
- §11.2: two modes. The per-leaf build compiles each leaf alone, in parallel, with a cache keyed
  by a hash of the leaf and the declarations it depends on. The holistic check reads all
  declarations, and uses fingerprints from the leaf jobs. "The query engine (§11.1) makes the
  holistic check incremental."

## What the first compiler must do

From decisions 1 to 3 and §5 to §11: parse declaration files and code files; type-check Go bodies
with the Keel additions; check contracts (the proof rule of §7.1, decision 1); check the
architecture rules and strict mode (decision 2); compute reach over all dependency modules
(§5.4c); lower to Go with `//line` directives; and serve `check`, `build`, `test`, `guide`,
`context`, `query`, and JSON diagnostics. The development plan defers the LSP server, LLVM, and
self-hosting.

## Findings

1. **A query engine means a new type checker.** `go/types` checks one package at a time, as a
   batch. A query such as "the type of X", answered incrementally after each edit, needs a type
   checker built for queries. That is a rewrite of Go's type checker, on top of the engine itself.
   The spec does not count this cost.
2. **The structure of Vulcan already gives most of the incrementality.** A leaf compiles against
   its own declarations (§11.2), and `context_budget` keeps leaves small. A per-package cache,
   keyed by content hashes as Go's build cache is, rebuilds only the changed leaf. The Go toolchain
   then caches each generated package.
3. **The Go front end can be reused.** A prototype (Go 1.26.1) turned the Keel-only text of test
   3's `lib/shop` and `api/billing` into spaces, then parsed and type-checked them with the stock
   `go/parser` and `go/types`. It used a port stub generated from `component.keel`. Both packages
   checked with no errors, and each position pointed to the right line and column of the `.keel`
   file. One detail was needed: when contracts follow a signature, Go inserts a semicolon at the
   end of the signature line (Appendix B.1), so the body's `{` moves to the end of that line.
   Lines and the columns of every other token stay the same.
4. **A second IR has no consumer yet.** The only backend in scope is Go source. An IR that no
   second backend reads cannot be tested for its purpose, and it adds a layer to every feature.

## Options

**A. Query engine from the start (the current spec).** Build a memoized query engine and a type
checker for queries. Best latency for an editor. Cost: the largest part of the compiler, before any
language feature is shown to work. It cannot reuse `go/types`.

**B. Staged compiler with per-package caches (recommended).** Explicit stages, each with typed
input and output, cached per package by content hash. Reuse `go/parser` and `go/types` through
blanking (finding 3). Lower from the typed Go syntax tree to Go source. Add an incremental engine
for a stage only when a measurement shows the need.

**C. A plain translator to Go, with checks as `go vet` analyzers.** Translate `.keel` files to
`.go` files once, and run every Keel check as a `golang.org/x/tools/go/analysis` analyzer on the
result. This reuses most tooling. Declaration files are not Go, diagnostics point to generated
files unless each tool maps them back, and the holistic check does not fit the analyzer model
well.

## Recommendation: option B

### Stages

Each stage reads only the outputs of earlier stages, so each can be tested alone. Stages 1 to 4
run per package, in parallel. Stages 5 and 6 run once per build.

| # | Stage | Input | Output |
|---|---|---|---|
| 1 | Parse | `.keel` code files; declaration files | Blanked Go source with the same positions; a side table of contracts, `limit`s, `pure` and `handle` marks; declaration syntax trees |
| 2 | Stubs | Declaration syntax trees | Go declarations for each port, topic, and handler, in reserved `keel_` names (Appendix F) |
| 3 | Type check | Stages 1 and 2, and the export data of imported packages | `go/types` package and info; type-checked contract expressions |
| 4 | Leaf checks | Stage 3 | Diagnostics of the leaf (§7, decisions 1 and 3, the Go warnings of §6); a summary of the package: declarations, reach facts, fingerprints, effects |
| 5 | Holistic check | Declaration syntax trees and every stage-4 summary | Diagnostics of §5.4 to §5.5.2, strict mode, near-duplicates across leaves, `topics.lock`, the audit |
| 6 | Lower | Stages 3 and 4, when all checks pass | Go source with `//line` directives and the generated `main`; then `go build` |

### Rules

1. **Blanking, not a forked parser.** Stage 1 is a Keel lexer that knows the Keel additions
   (Appendix B.1 and B.3). It writes Go source in which each Keel-only token becomes spaces, and
   it applies the semicolon rule of Appendix B.1 by moving the body's `{` up to the signature
   line. It parses each contract expression with `go/parser.ParseExpr` at its original offset.
   Thus `go/types` reports errors at positions in the `.keel` files. A handler is the one
   construct that becomes Go text, not spaces: `idempotent` becomes spaces, and `handle` becomes
   `func` plus two spaces. After parsing, stage 1 renames the function in the syntax tree to
   `keel_handle_<Name>`, which moves no position, so it does not collide with the subscription
   name that tests pass to `keel.Deliver`. A Keel construct that cannot be blanked or renamed
   this way needs a change to this decision.
2. **Caches by content hash.** The key of a stage-1 to stage-4 result is a hash of the package's
   files, the Keel version, and the stage-4 summaries of the packages it imports. Stage 5 caches
   the reach facts of each dependency module by module version (§5.4c). Stage 6 uses Go's build
   cache.
3. **No separate IR in the first release.** Stage 6 rewrites the typed Go syntax tree. An IR is
   added when a second backend is a real requirement, with that backend as its first test.
4. **An incremental engine only when measured.** Milestone 2 measures, on the milestone-3
   application: the time of `keel check` after a one-line edit in one leaf, and of a full check.
   If the 95th percentile of a one-edit check exceeds the target that decision 6 sets, the project
   adds incremental computation to the slowest stage, and records that in a new decision.
5. **Self-hosting is not planned.** It stays a possibility after the language earns continued
   work (development plan, milestone 4). The spec stops presenting it as part of the design.

### What this changes in the guarantees

Nothing. The stages compute the same diagnostics as the spec defines. The two modes of §11.2
stay: stages 1 to 4 are the per-leaf build, and stage 5 is the holistic check.

## Spec changes on acceptance

- §11, backend bullet: the Go backend lowers from the typed Go syntax tree. An IR comes with a
  second backend.
- §11.1: replace "query-based and incremental from the start" with the six stages, the cache
  rule, and the measurement rule. Replace the self-hosting bullet with rule 5.
- §11.2: "The query engine (§11.1) makes the holistic check incremental" becomes "stage 5 reads
  cached summaries; §11.1 says when it becomes incremental".
- §10: the LSP and MCP servers are deferred until after milestone 2. They reuse stages 1 to 5.
- Appendix B.1: one sentence on the blanking form of the semicolon rule, for implementers.

## Open points

- **The latency target.** Rule 4 needs a number. Decision 6 (evidence protocol) sets it with the
  other thresholds.
- **Blanking and `gofmt`-like tools.** Blanked source is not meant for people. `keel fmt` formats
  the `.keel` file itself, so it needs its own printer for the Keel additions.
- **Reach analysis.** Stage 5 can use `golang.org/x/tools/go/ssa` and its call graph, or a smaller
  analysis over `go/types`. Milestone 1 should choose by trying both on the dependencies of the
  milestone-3 application.
