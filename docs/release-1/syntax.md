# Release 1: supported syntax

Status: accepted, 2026-10-21. Appendix E has K0009. Part of the milestone-1 scope of the
[development plan](../development-plan.md).

Release 1 is the compiler that milestones 1 and 2 build, and that milestones 3 and 4 use. It
supports one process with several leaves, consumer-owned ports, libraries, explicit wiring,
gateways and entry leaves, contracts, and generated startup and shutdown. It does not support
topics, remote ports, transports, or more than one container.

## The rule for everything else

Each construct of the spec that this document does not list as supported is **rejected**, with a
new error (K0009): "*construct* is not supported in release 1". The message names the construct
and the spec section. Nothing is silently ignored. The compiler parses the whole grammar of
Appendix B, so that it can give K0009 at the right position, not a syntax error.

The **Milestone** column says when a construct starts to work. Until then, K0009 rejects it.

## Declaration files (Appendix B.2)

| Construct | Milestone | Notes |
|---|---|---|
| `system` with one `container` | 1 | |
| A second `container` | Deferred | K0009. Needs topics or remote ports (§5.5). |
| `group` | 1 | A label only. |
| Leaf list, with leaf settings `Web{ Addr: flag(...) }` | 1 | |
| `person`, `external` | 1 | Needed by `entry for` and `gateway for`. |
| `uses` | 1 | Parsed and resolved. Diagrams are deferred. |
| A wire `Leaf.Port: Leaf.Constructor` or `: Leaf.Type` | 1 | |
| Constructor settings `Leaf.NewX{ ... }` | 1 | |
| `flag("name", default: ..., max: ...)` | 1 | |
| An endpoint with a container, `Api/Orders.Stock` | Deferred | K0009. |
| `via` on a wire | Deferred | K0009. |
| `config` keys `context_budget`, `shutdown_timeout`, `on_contract_failure` | 1 | `on_contract_failure: exit` works from milestone 2. |
| `config` keys `network`, `acquiring`, `allow_network`, `allow_acquiring`, `strict`, `strict_exempt`, `strict_interfaces`, `strict_shared`, `mutators` | 2 | |
| `config` keys `timeout`, `transport`, `output` | Deferred | K0009. Cross-container only. |
| `config` keys `duplicate_similarity`, `duplicate_min_tokens` | Deferred | K0009. Near-duplicate detection (§9) is deferred. |
| `component` with `needs port` | 1 | |
| Port method contracts (`requires`, `ensures`) | 2 | |
| `needs remote port`, `timeout` | Deferred | K0009. |
| `publishes`, `subscribes`, `topic[T]` | Deferred | K0009. |
| `gateway for`, `entry for` | 1 | The network rule that uses them is milestone 2. |
| `library` | 1 | |
| `transport` files | Deferred | K0009. |

## Code files (Appendix B.3)

| Construct | Milestone | Notes |
|---|---|---|
| Go bodies, types, generics, and all Go statements | 1 | The Go syntax of the pinned toolchain's `go/types` (decision 5). Go 1.26 or later. |
| `setup` and `startup` functions | 1 | §6. |
| `requires`, `ensures` on functions and methods | 2 | |
| `err`, `result`, `result.N`, `old(x)`, `=>` in contracts | 2 | |
| `invariant` on a struct | 2 | |
| Tier-1 contract expressions | 2 | Comparisons, arithmetic with exact integers, `&&`, `\|\|`, `!`, `=>`, field access, `len`. |
| `pure` functions, and calls to them in contracts (tier 2) | Deferred | K0009. Added in milestone 2 only if the milestone-3 application needs them. |
| `vulcan.All`, `vulcan.Any` | Deferred | K0009, with `pure`. |
| `limit` on a field | Deferred | K0009. Only message types use it. |
| `handle`, `idempotent` | Deferred | K0009. |
| `//go:linkname` in a leaf or a library | 1 | Rejected with K0122 (§5.4 rule 13). |

## Rules and checks by milestone

| Area | Milestone 1 | Milestone 2 |
|---|---|---|
| Layout and names (§5.2, App. B) | K0001 to K0008 | |
| Architecture (§5.4) | Rules 1, 2, 4, 5, 7, 8, 9, 11, 12, 13, and the wire checks (K0002, K0101, K0104, K0105, K0107, K0108, K0110 to K0113, K0116 to K0119, K0121, K0122) | Rule 6 (K0109), rule 10 (K0114), port contracts (K0120), dead code (K0115) |
| Libraries and modules (§5.4a, §6) | K0201, K0202, K0204, K0209 | K0203 |
| Network and acquiring (§5.4c) | | K0205, and the reach analysis of Appendix D |
| Gateways and entry leaves (§5.4b) | K0206, K0207 | |
| Strict mode (§5.4d) | | K0123, K0124, K0125, K0210, K0211 |
| Contracts (§7.1, §7.2) | | K0501 to K0514, K0520, K0521; the proof rule; runtime checks; no check removal |
| Deferred calls and panics (§7.3) | | K0522 to K0524, the second check point, the unwinding check |
| Effects (§7.4) | | K0525, the "detects after effects" list, `on_contract_failure` |
| Startup (§6) | K0601 to K0605 | |
| Go warnings (§6) | K0701, K0704, K0705 | K0702, K0703 |

The proof rule of §7.1 is needed in milestone 2, because K0502 and K0504 depend on it. The
removal of proved checks is deferred: every check runs.

## Runtime API (Appendix C)

Release 1's package `github.com/robot-accomplice/vulcan/vulcan` declares only:
`ContractFailure`, `Value`, `Bind`, and, from milestone 2, the contract-failure behavior of
§7.4. A use of any other name of Appendix C (`Topic`, `Subscription`, `DeadLetter`, `Published`,
`Deliver`, `DeliveryID`, `RemoteError`, `NewError`, `Unavailable`, `Timeout`, `RemotePanic`,
`All`, `Any`) is K0009, not Go's "undefined".

## Commands (§10)

| Command | Milestone | Notes |
|---|---|---|
| `vulcan check`, `vulcan build` | 1 | |
| `vulcan test` | 1 | With the race detector where the host has it (K0705). |
| `vulcan mod` | 1 | |
| `vulcan guide` | 2 | The primer, cut to release 1 (see below). |
| `vulcan context`, `vulcan query` | 2 | |
| `vulcan audit` | 2 | The sections that decisions 1 to 3 define. |
| `--json` on each command above | 1 for check, build, test; 2 for the rest | |
| `vulcan fmt`, `explain`, `doc`, `diagram`, `archdiff`, `fake`, `trace` | Deferred | The command prints that it is not in release 1, and exits 2. |
| LSP and MCP servers | Deferred | Decision 4. |

## The primer for release 1

`vulcan guide` must teach only what release 1 accepts, so that a model does not write deferred
constructs. Release 1 ships the primer with its sections on topics, remote ports, and transports
removed, and one line that lists what is deferred. The full primer stays in `docs/guide.md`.

## Spec change

Appendix E has K0009: "A construct that this release does not support; the message names the
construct, the release, and the spec section".
