# Decisions

Each file records one design decision: the question, the current rule, the options, and a
recommendation. A decision is **proposed** until the owner accepts it. On acceptance, the same
change updates `docs/spec.md` and `docs/guide.md`.

## Milestone 0 decision packet

The [development plan](../development-plan.md) lists six decisions that come before any compiler
code.

| # | Decision | Status |
|---|---|---|
| 1 | [Contract timing](0001-contract-timing.md): deferred calls, recovery, panics, concurrency | Accepted |
| 2 | [Architecture enforcement](0002-architecture-enforcement.md): the guarantee matrix, and strict and ordinary modes | Accepted |
| 3 | [Contract failure](0003-contract-failure.md): what a panic does to external effects | Accepted |
| 4 | [Compiler structure](0004-compiler-structure.md): staged compiler or incremental query engine | Accepted |
| 5 | [Names and reproducibility](0005-names-and-reproducibility.md): extension, command, module path, Go version, platforms | Accepted |
| 6 | [Evidence protocol](0006-evidence-protocol.md): tasks, measurements, and thresholds for milestone 4 | Accepted; amended by audits 3 to 9 |
