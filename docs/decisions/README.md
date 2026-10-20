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
| 2 | Architecture enforcement: the guarantee matrix, and strict and ordinary modes | Not started |
| 3 | Contract failure: what a panic does to external effects | Not started |
| 4 | Compiler structure: staged compiler or incremental query engine | Not started |
| 5 | Names and reproducibility: extension, command, module path, Go version, platforms | Not started |
| 6 | Evidence protocol: tasks, measurements, and thresholds for milestone 4 | Not started |
