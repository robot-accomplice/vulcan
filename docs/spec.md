# Keel — Language Specification (Draft 0)

> **Working name: Keel** — placeholder until a better name is chosen (Q1). Source extension: `.keel`.
> Status: brainstorming draft, 2026-09-28. Nothing here is implemented.

## 1. Purpose

Keel is a compiled, garbage-collected, statically typed language designed for software that humans and
LLMs write *together*. Its thesis: most failures in human/AI co-development come from missing structure —
the model can't see the architecture, invents APIs, duplicates instead of reusing, and masks errors; the
human can't review what the model produced. Keel makes that structure part of the language, so the
compiler can see it, enforce it, draw it, and teach it.

## 2. Design principles

| Principle | Meaning in Keel |
|---|---|
| Compiled, GC | Native binaries; tracing GC; no manual memory management, no borrow checker. |
| Go ergonomics | Go is the syntax model. Deviate only where Go is demonstrably weak (e.g. generics, error enforcement). |
| Rust-grade compiler intelligence | Diagnostics explain *why* and propose fixes; exhaustiveness checks; lints for duplication and error masking. |
| Imposed structure | One canonical project layout, one formatter, one place for each kind of thing. |
| Architecture is first-class | Systems, components and ports are language constructs. The compiler enforces them. |
| Self-documenting, self-diagramming | Docs and architecture diagrams are generated from source, never maintained beside it. |
| Contracts are code | Pre/postconditions and invariants live on declarations and are auditable. |
| Tooling teaches | Every tool output is designed to be read by an LLM as well as a human. |
| OO "light" | Structs, methods, interfaces, generics. No inheritance, no classes. |

## 3. Evidence behind the design

Research on how LLMs fail when writing applications with humans (sources in Appendix A):

| # | Observed failure | Keel's answer | Section |
|---|---|---|---|
| R1 | Agents get lost in large codebases; quality degrades as context grows | Components are small, self-contained units with explicit ports; `keel context` emits a minimal context pack | §5, §11 |
| R2 | Duplication up 81%, reuse (cross-file calls) down 35% | Compiler-level near-duplicate detection; ports make existing capabilities discoverable | §10 |
| R3 | Error-masking constructs up 47% | Errors must be handled explicitly; silent discard is a compile error | §7 |
| R4 | Hallucinated packages (~20%) and APIs | Exact, queryable symbol index served by the compiler; no implicit imports | §11 |
| R5 | ~94% of typed-language compile errors are type errors | Strong static types, inference that works on partial programs, enabling type-guided generation | §6 |
| R6 | AI code is harder to review; "looks right but isn't" | Embedded, auditable contracts; architecture-level diffs | §8, §10 |
| R7 | Perceived speed ≠ measured speed | Fast verification: incremental builds, per-component tests | §9 |
| R8 | New languages have no training data; LLM quality drops steeply | Go-shaped syntax for knowledge transfer; prompt-sized spec; tooling that instructs the model | §4, §11 |

## 4. Syntax

Go is the model. A Go programmer — or a model trained on Go — should read Keel on first sight.
Deviations must be justified in this spec and listed here:

| Deviation from Go | Why |
|---|---|
| `system`, `container`, `group`, `component`, `library`, `port`, `person`, `external`, `layer` declarations | Architecture as a first-class construct (§5) |
| Unhandled errors are compile errors; postfix `?` propagation | R3 (§7) |
| No `nil`/`null` value; optional types `?T` with flow narrowing | No null dereferences, ever (§7a) |
| No exceptions of any kind; no `try`/`catch`/`throw`/`recover()` | Errors are values; aborts are handled only by supervising tasks (§7, §7.1) |
| Integer overflow aborts; `+% -% *%` wrap explicitly | Go wraps silently (§7c) |
| A lock held by an aborting task is poisoned; `Lock()` returns an error | Go has no poisoning (§7.1) |
| `go f()` returns a task handle whose exit the parent can observe | Supervision (§7.1) |
| `requires` / `ensures` / `invariant` clauses | Embedded contracts (§8) |
| `pure` functions | Callable from contracts; effects verified absent (§8.1) |
| `test` and `fake` blocks | Inherent testability (§9) |
| `topic<T>` publish channels for crossing containers | Never blocks, 0..n readers; unlike `chan` (§5.5) |
| `remote port`, `idempotent` handlers, `via` wiring | Crossing containers honestly (§5.5) |
| `ignore err, "reason"`, `x ?? fallback` | Explicit, audited discards and defaults (§7, §7a) |
| Sum types with exhaustive `switch` | Typed errors, task exits, state machines (§6) |
| Generics use `<T Bound>`, not `[T Bound]` | C-family familiarity; unbounded params need no `any` (§6.1) |

Everything else — `:=`, `func`, structs, methods with receivers, `for`, `switch`, slices, maps, `defer`,
packages-as-directories, gofmt-style formatting — follows Go unless a later section says otherwise.

## 5. Architecture model

The architecture is a tree that follows the C4 model. Code lives only in small, flat leaves. Every
node above the leaves is pure structure: children, ports, and wiring.

### 5.1 Hierarchy

```
system Shop                         C4 System    — plus the people and external systems around it
 ├ container Api                    C4 Container — a deployable or process
 │  ├ component Billing   (leaf)    C4 Component — one flat directory of code
 │  └ component Orders    (leaf)
 └ container Worker
    └ group Fulfilment              optional composite, for zoom levels inside a container
       ├ component Picking  (leaf)
       └ component Shipping (leaf)
```

| Kind | Holds | Code? |
|---|---|---|
| `system` | containers, `person` and `external` declarations | no |
| `container` | groups and components | no |
| `group` | groups and components | no |
| `component` | source files and tests | yes — and no child nodes |
| `library` | shared types and reusable code | yes — but no ports, topics, or mutable state (§5.4a) |

### 5.2 Layout

The directory tree mirrors the architecture tree. Each node is a directory holding exactly one
declaration file named for its kind: `system.keel`, `container.keel`, `group.keel`,
`component.keel`, or `library.keel`. A composite directory contains no source files; a leaf or
library directory contains no child nodes. Libraries live under the system's `lib/` directory.
There is one place to look for everything.

### 5.3 Declarations

A leaf declares what it provides and requires:

```go
// Billing charges customers and records the charge in the ledger.
component Billing {
    layer    app
    provides Payments
    requires Ledger, Clock
}
```

Ports are interfaces with contracts. **Every node owns the ports on its own boundary and knows
nothing about the nodes on the other side:**

- A port in `provides` is declared by the provider: "what I offer."
- A port in `requires` is declared by the consumer: "what I need." (Hexagonal style.)

A provider never learns who uses it; a consumer never learns who serves it. Only the parent that wires
them together knows both.

```go
// In Billing (provider): what Billing offers.
// Payments charges a customer's stored payment method.
port Payments {
    // Charge moves amount from the customer to the merchant.
    func Charge(customer CustomerID, amount Money) (Receipt, error)
        requires amount > 0
        ensures  result.Amount == amount
}

// In Orders (consumer): what Orders needs, in its own terms.
// Checkout takes payment for a confirmed order.
port Checkout {
    func Charge(customer CustomerID, amount Money) (Receipt, error)
        requires amount > 0 && amount < MaxOrder
}
```

**Port compatibility.** When a parent wires a provided port to a required port, the compiler checks:

1. **Signatures.** Every method of the required port exists on the provided port with an identical
   signature. (The provided port may have more methods.)
2. **Contracts (substitutability).** For each method, the consumer's `requires` implies the
   provider's `requires` (the provider accepts everything the consumer may send), and the provider's
   `ensures` implies the consumer's `ensures` (the provider promises at least what the consumer
   relies on). Where the compiler cannot prove an implication, it inserts a runtime check at the
   wiring point and lists it in `keel audit`.

When the ports don't match, the fix is an adapter: an ordinary leaf that requires one port and
provides the other. There is no special adapter construct, because composites hold no code.

A composite wires its direct children and publishes some of their ports. A child's requirement that
no sibling satisfies becomes the composite's own requirement, satisfied by *its* parent:

```go
// Api serves the public HTTP interface.
container Api {
    requires Fulfilment                 // Orders' remote port, passed up to the system (§5.5)

    Billing     { Ledger: LedgerStore.Ledger, Clock: SystemClock.Clock }
    Orders      { Checkout: Billing.Payments, Fulfilment: parent.Fulfilment }
    LedgerStore {}
    SystemClock {}
}

// Worker picks and ships orders.
container Worker {
    provides Intake from Fulfilment     // re-exported; the Fulfilment group stays hidden

    Fulfilment {}
}
```

`Orders { Checkout: Billing.Payments }` is where the port-compatibility check runs.

The system adds C4's outer ring — the people and external systems it interacts with:

```go
// Shop is the online store.
system Shop {
    layers domain < app < infra

    person   Customer "Buys goods through the web store"
    external Stripe   "Card payment processor"

    Api    { Fulfilment: Worker.Intake via queue }   // the only link between containers
    Worker {}

    uses Customer -> Api    "places orders over HTTPS"
    uses Api      -> Stripe "charges cards over HTTPS"
}
```

External systems are not wired with ports: they are outside the language's control and speak their
own protocols. A leaf reaches one through an ordinary client library, and `uses` records the
relationship so it appears in diagrams.

### 5.4 Rules the compiler enforces

1. **Ports only.** Code in a leaf may use another node **only** through a port listed in its
   `requires`, or by importing a library (§5.4a). Any other cross-node reference is a compile error.
2. **Code only in leaves.** Composites hold no source files; leaves hold no child nodes.
3. **Encapsulation at every level.** Outside its parent, a node does not exist. A composite exposes
   only the ports it publishes with `provides X from Child`.
4. **Complete wiring.** Each composite wires every `requires` of every direct child exactly once —
   to a sibling's port, or to one of the composite's own `requires`.
5. **Complete provision.** A leaf implements every port it `provides`; a composite sources every
   port it `provides` from a child.
6. **Compatible wiring.** Every wire satisfies the port-compatibility check in §5.3.
7. **Boundary ownership.** A node declares the ports it provides and the ports it requires. It may
   not name a port declared by a sibling or by any other node.
8. **Layers.** Port dependencies respect the system's `layers` order (lower never requires higher).
9. **Acyclic.** The dependency graph among siblings is acyclic at every level.
10. **Private types.** Types are private to their leaf unless they appear in a port signature.
11. **Leaf size budget (warning).** A leaf whose `keel context` output exceeds the project's
   `context_budget` (one value in project config) gets a warning whose hint suggests where to split
   it. This keeps every unit small enough for a model to hold whole.

### 5.4a Libraries

Types shared across a wire — `Money` in a port signature on both sides — must be the *same* type,
so they cannot belong to either leaf. They live in a **library**: a node that holds shared types
and reusable code but takes no part in the running architecture.

```go
// lib/money/library.keel
// Money represents amounts in minor currency units.
library Money {
    layer domain
}
```

```go
// api/billing/billing.keel
import "lib/money"
```

Rules:

1. A library has **no ports, no topics, and no mutable package-level state**. It is code, not a
   participant: it cannot hold state that couples the leaves that import it.
2. A library may import other libraries, never components or composites. Library imports are
   acyclic and respect `layers`.
3. Any leaf may import a library in the same or a lower layer.
4. The leaf size budget (rule 11) applies to libraries too.
5. `keel diagram` shows libraries as shared dependencies, and `keel audit` lists which leaves use
   each one.

Libraries compile before the leaves that import them, and leaves compile against library
declarations only (§12.2).

### 5.5 Crossing containers

Containers are separate processes, so the only thing that crosses between them is a **topic**: a
publish channel, modeled on Go's channels but extended across the process boundary. Method ports
never cross a container boundary.

A topic is a distinct type, `topic<T>`, not Go's `chan T`, because it behaves differently: a Go
channel send blocks until a reader takes the value and each value has exactly one reader; a topic
publish never blocks and has zero, one, or many readers. One keyword with two behaviors would
mislead humans and models trained on Go (R8). Go's `chan` keeps Go semantics for in-process
concurrency.

```go
// In Orders (the consumer declares what it needs):
remote port Fulfilment {
    Orders  out topic<OrderPlaced>   // Orders publishes
    Shipped in  topic<ShipmentSent>  // Orders subscribes
}

// In Shop (the parent names the transport):
Api { Fulfilment: Worker.Intake via queue }
```

**Why publish: it is the loosest possible coupling.** The sender's only job is to publish. It
does not know, and must not care, whether anything reads its messages, how many readers there are,
or when they read.

- **Publish, never wait.** A send on a remote topic never blocks and never fails because of
  readers or the network. It hands the message to a local outbox owned by the runtime, which
  delivers it. (Contrast Go's in-process channel, where an unbuffered send blocks until a reader
  takes the value.)
- **Zero, one, or many readers.** The parent may wire a published topic to any number of
  subscribers, including none. Adding or removing a subscriber never changes the publisher.
- **No request/response.** A reply would bind the sender to a reader. When a sender needs to know
  what happened, it subscribes to a separate event that the other side publishes (as `Shipped`
  does above).
- **Ownership.** Each side owns only its end of each topic. Only message values cross, by copy;
  no memory or references are shared.
- **`remote` is required to cross.** The compiler rejects wiring a port across containers unless the
  consumer declared it `remote`.
- **Contracts** on message types (struct `invariant`s) are checked at send and again at receipt.
  Message types must be serializable: no functions, topics, or pointers into the sender's memory.
- **Dead letters must be routed and sized.** A message that runs out of retries goes to its
  topic's dead-letter topic. The wiring parent must route it, and must give it a size:

  ```go
  Api { Fulfilment: Worker.Intake via queue,
        deadletters: Ops.Alerts, deadletterCapacity: 5_000 }             // fixed at compile time
  Api { Fulfilment: Worker.Intake via queue,
        deadletters: Ops.Alerts, deadletterCapacity: flag("dlq-size") }  // supplied at startup
  ```

  Missing either is a compile error. A size bound to a runtime flag is checked at startup: the
  process refuses to start if the flag is absent or invalid. Opting out of routing requires
  `ignore deadletters, "reason"`, recorded in `keel audit` like `ignore err`.
- **Delivery guarantees are the author's decision, not the language's.** The language supplies the
  means to declare them per remote topic (at-most-once or at-least-once; ordered or unordered).
  **Default: at-least-once, unordered** — nothing is lost, and because a message may arrive twice,
  subscribers are written to be idempotent (handling a message twice has the same effect as once).
- **Unread topics warn.** A published topic wired to zero subscribers compiles, with a warning:
  publishers may exist before their readers, but an unread topic should never go unnoticed.
- **Subscriber aborts never reach the publisher's code.** When a subscriber aborts while handling a
  message, the publisher is unaffected. The outbox records the event — which message, which
  subscriber, which attempt — and the message is retried, then dead-lettered, under the topic's
  delivery guarantee.
- **An outbox failure aborts.** If the outbox itself fails — a write cannot be stored, the store is
  corrupt — the publishing task aborts (§7.1). The publisher's code still never sees an error.
- **Outbox durability follows the delivery guarantee.** An at-least-once topic's outbox is on
  local disk, so undelivered messages survive a process restart. An at-most-once topic's outbox
  may be in memory, since loss is already allowed.
- **At-least-once holds while the delivery mechanism is intact.** The language retries a message
  until it is delivered or dead-lettered. It does not promise survival of the delivery mechanism
  itself (a lost or corrupt disk). Making delivery survive that — persistent brokers, replication,
  and so on — is system design outside the language's purview.
- **Message tracing is off by default; opt in per topic or project-wide.** When on, the runtime
  gives each message an ID and records its lifecycle (published, delivered to X, aborted, retried,
  acknowledged or dead-lettered), and `keel trace <id>` reconstructs it. Retention is bounded, like
  every other runtime store (§5.5.1). Developers know their circumstances; the language does not
  spend their disk and CPU on a scenario it cannot foresee.
- **Idempotency is declared.** A subscriber on an at-least-once topic must mark its handler
  `idempotent`; without it, the code does not compile. The compiler cannot prove the claim, but the
  marker turns an invisible assumption into a written one: it appears in `keel context`, in
  `keel doc`, and in `keel audit`, where reviewers can check it.

#### 5.5.1 Bounded buffers

Decoupling requires buffering, and every buffer must eventually refuse or forget. Keel makes the
wiring choose which, in writing, so no runtime queue can grow memory without limit.

- **Every runtime queue has a capacity** — outbox, retry queue, subscriber inbox, and dead-letter
  topic. A capacity comes from the wire, from the one central project default, or from a startup
  flag (`flag("name")`, checked at process start as in §5.5). A remote topic whose queues do not
  all resolve to a capacity is a compile error.
- **The full-queue policy comes from one project-wide default unless a wire overrides it.** The
  default ships as `block` — nothing is lost — and a project may change it in its config. A wire
  overrides it where the message calls for something else. `keel audit` shows, per topic, whether
  the policy is the project default or an override.

  | `whenFull` | Effect | Typical use |
  |---|---|---|
  | `block` | Publisher waits for room (coupled to slow readers in time only) | Business events — losing one is unacceptable |
  | `dropOldest` / `dropNewest` | Message discarded, counted, dead-lettered; listed in `keel audit` | Telemetry — stale is fine |
  | `abort` | The task aborts (§7.1) | When neither waiting nor loss is acceptable |

  ```go
  Api { Fulfilment: Worker.Intake via queue, capacity: 10_000, whenFull: block, ttl: 24h,
        deadletters: Ops.Alerts, deadletterCapacity: flag("dlq-size") }
  Api { Metrics:    Obs.Ingest    via queue, capacity: 1_000,  whenFull: dropOldest, ttl: 5m,
        deadletters: Ops.Discard, deadletterCapacity: 100 }
  ```

- **Messages expire.** Every topic has a time to live (`ttl`), resolved like capacity. An expired
  message is dead-lettered, so a dead subscriber cannot pin memory.
- **The dead-letter topic overflows to `abort`.** It cannot `block` — that would stall the
  pipeline on its own error path — and it cannot drop, because dropping a dead letter is a silent
  failure. A full dead-letter topic is loud.
- **Limits are visible before they are hit.** The runtime reports each queue's depth, high-water
  mark, and a time-to-full estimate. `keel audit` lists every topic's capacity, policy, and TTL,
  so a reviewer sees the memory bound of the whole system in one place.

### 5.6 Diagrams

`keel diagram --level context|container|component|code [--focus <node>]` renders each C4 level from
the declarations above. Because composites hold no code, a diagram at any zoom level shows where all
behavior lives — a parent cannot hide logic.

## 6. Type system

- **Strong and static.** No implicit conversions.
- **Inferred wherever possible.** Locals, closures, and generic instantiations are inferred.
  Port signatures and exported functions are explicit — they are the contract surface humans review.
- **Private signatures may omit types only when inference is safe.** A private function (not
  exported, not in a port) may leave out parameter or return types when all of these hold:
  1. The type is inferred from the function's **own body only** — never from its call sites, so
     editing a caller can never change a function's type.
  2. The body determines **exactly one** type. If a field access or method call fits more than one
     type, the compiler reports an error with a fix-it that inserts the annotation.
  3. The function is not recursive; recursive functions annotate their return type.

  Tooling always shows the full inferred signature — `keel context`, `keel doc`, hover, and diffs
  — so a model or reviewer never has to reconstruct a type by hand.
- **Inference must work on incomplete programs**, so tooling can offer type-valid completions (R5).
- **Sum types** (tagged unions) with exhaustive `switch`. Needed for typed errors and for state
  machines; a deliberate addition over Go.
- **No nil.** No type, including pointers, interfaces, slices and maps, has a null value (§7a).
- Methods attach to structs as in Go. Interfaces are structural as in Go; ports are nominal.

### 6.1 Generics

Angle brackets, with bounds written like Go parameters (name, then type — no colon):

```go
func Max<T Ordered>(xs []T) ?T
type Cache<K Hashable, V> struct { ... }
func (c *Cache<K, V>) Get<R>(k K, f func(V) R) ?R   // methods may declare their own type params
```

- An unbounded parameter needs no bound (`V`, not `V any`).
- Several bounds combine with `&`: `<T Ordered & Hashable>`.
- Type arguments are normally inferred; explicit instantiation is the exception.
- Methods on concrete types may declare their own type parameters (as Go 1.27 now allows).
  Interface and port methods may not, because a generic method cannot be dispatched dynamically
  without knowing every instantiation in advance.

**Parsing.** In declarations and type positions `<` is unambiguous. In expressions, `a < b > (c)` could
be two comparisons or a generic call. Keel uses C#'s rule, which needs lookahead but no symbol
table: if `<...>` parses as a type-argument list **and** the next token is one of `( ) ] } , ; . {`
or end of line, it is a type-argument list; otherwise `<` is less-than. `>>` is split into two `>` when
it closes nested type arguments (as in C#, Java and C++11). The formatter rewrites any remaining
ambiguous expression with parentheses, so humans and models never see one.

## 7. Errors

**There are no exceptions, no `try`/`catch`, and no `throw`. No function can catch an abort; recovery happens only at task boundaries (§7.1).** An error is an ordinary
value that travels only through return values, so every path an error can take is visible in the source.

Errors are returned in Go's `(T, error)` shape, but handling is **required and explicit**:

- Using `T` from a call without handling its error is a compile error.
- Postfix `?` returns the error to the caller: `r := billing.Charge(c, amt)?`. It is shorthand for
  `if err != ok { return err }` and nothing else — it does not unwind, catch, or jump.
  (`?` rather than a `try` keyword, so neither humans nor models read it as a try/catch block.)
- Discarding an error requires a stated reason, which is recorded for audit:
  `ignore err, "cache miss is not an error here"`.
- Ports declare which error kinds a function may return; `switch` over them is exhaustive.
- Lint (on by default): a catch-all handler that substitutes a default value is flagged as error
  masking (R3).
- **Aborts are not exceptions.** A broken invariant (contract violation, `unreachable`, overflow,
  out-of-range index) terminates the task with a diagnostic report. No function can catch an abort,
  so it can never be used for control flow. See §7.1.

### 7.1 Supervision

Recovery happens only at **task boundaries** — Erlang's "let it crash" model in Go syntax.

- An aborted task's state is discarded. Its **parent** — the task that started it — receives the
  abort as an ordinary value and decides what happens next: restart, fail that request, or report.
- Starting a task returns a handle; the handle's exit is a sum type the parent `switch`es on:

  ```go
  t := go handle(req)
  switch exit := <-t.Exit {
  case Completed:
      ...
  case Aborted:
      log.Report(exit.Report)   // the diagnostic: which invariant, where, with what values
      respond(req, Unavailable)
  }
  ```

- **No abort is ever silent.** If the parent does not observe a child's exit (`go f()` with the
  handle discarded), an abort propagates to the parent, then to its parent, up to `main`, where the
  process exits with the report. Supervision is opt-in; crashing loudly is the default.
- Every abort report is recorded (RCA trail: task, parent, invariant, source position, values).

**Poisoned locks.** Tasks share memory, as in Go, so an abort mid-update could leave shared data
half-changed after the task's own state is gone. Keel follows Rust:

- A mutex (or read-write mutex) held by a task when it aborts becomes **poisoned**.
- Every later `Lock()` returns an error, `Poisoned`, which must be handled like any error (§7):

  ```go
  mu.Lock()?                      // propagates Poisoned to the caller
  ```

- A task that repairs or rebuilds the data may call `mu.ClearPoison()`. Each call is listed in
  `keel audit`.
- Limit: poisoning protects only data guarded by a lock. Shared data mutated without a lock is a
  data race — already a bug — and the race detector runs in test builds to find it.

## 7a. Null safety

Keel has **no null value**. `nil` does not exist. No expression can dereference nothing, so a
null-pointer failure cannot occur.

- Every type has a well-defined zero value (as in Go) **or** must be initialized before use; pointers,
  interfaces, and function values must be initialized.
- Absence is a type, not a value: `?T` is "a `T` or `none`".
- A `?T` cannot be used as a `T`. The compiler narrows it by control flow:

  ```go
  u := users.Find(id)        // u is ?User
  if u == none {
      return NotFound
  }
  send(u.Email)              // u is User here — the check above proved it
  ```

- `x ?? fallback` supplies an explicit default. It is allowed because the fallback is written at
  the call site, but it appears in `keel audit` like `ignore`.
- APIs that "might not find" something return `?T`: map lookup, `Find`, search, parsing without error.
- Values crossing a foreign-function boundary arrive as `?T` until checked.
- Diagnostic hint on misuse: the compiler shows the narrowing pattern above, not just "type mismatch".

## 7b. Bounds

The compiler tries to prove every index `xs[i]` is in range. Proof comes from:

- the loop: `for i := range xs`
- a prior check, narrowed by control flow: `if i < len(xs) { ... xs[i] ... }`
- a contract: `requires i < len(xs)`

| Outcome | Result |
|---|---|
| Proved in range | No check emitted at runtime. |
| Proved out of range | Compile error. |
| Cannot be decided at compile time | **Warning**, with a hint showing the three proof patterns above and `xs.At(i)`, which returns `?T`. The index is bounds-checked at runtime and listed in `keel audit`. |
| Runtime index out of range | Abort, as in Go. Only a supervising parent task observes it (§7.1). |

## 7c. Integer overflow

Integers never wrap silently.

| Operation | On overflow |
|---|---|
| `+ - *`, unary `-`, narrowing conversion (`int32(x)`) | Abort. |
| `+% -% *%` | Wraps (two's complement). For hashes, checksums, and other deliberate modular arithmetic. |
| `a.CheckedAdd(b)`, `CheckedSub`, `CheckedMul`, `CheckedConvert<T>` | Returns `?T`; `none` on overflow. For code that handles overflow itself. |
| Constant expressions | Compile error, as in Go. |

Division by zero aborts, as in Go.

## 8. Contracts

Contracts are clauses on declarations, not comments:

| Clause | Attaches to | Meaning |
|---|---|---|
| `requires expr` | functions, port methods | precondition, caller's obligation |
| `ensures expr` | functions, port methods | postcondition; `result` names the return value |
| `invariant expr` | structs | holds after every exported method returns |

### 8.1 What a contract may contain

Contract expressions are **pure**: they never do I/O, call ports, touch channels or topics, mutate
anything, or read the clock or randomness. So enabling or disabling checks can never change what a
program does. Within that rule, three tiers are allowed:

| Tier | Adds | Example |
|---|---|---|
| 1 | Comparisons, arithmetic, `&&` `\|\|` `!`, `=>` (implies), field access, `len`, `result`, `old(x)` (value on entry) | `ensures err == ok => acct.Balance == old(acct.Balance) - amt` |
| 2 | Calls to functions marked `pure` — the compiler verifies they have no effects | `requires IsSorted(a) && IsSorted(b)` |
| 3 | Quantifiers over collections: `all x in xs: …`, `any x in xs: …` | `requires all x in xs: !IsNaN(x)` |

```go
pure func IsSorted<T Ordered>(xs []T) bool { ... }

func Merge(a, b []int) []int
    requires IsSorted(a) && IsSorted(b)
    ensures  IsSorted(result) && len(result) == len(a) + len(b)
```

Quantifiers cost O(n) when checked at runtime; `keel audit` lists each quantifier still checked in
release builds, and a component may opt out of them.

Checking:
- **Statically** where the compiler can prove or refute them.
- **At runtime** in debug and test builds; release builds keep them by default, with per-component opt-out.
- **Port contracts bind every implementation**, including test fakes — a fake that violates its port's
  contract fails the test.

Auditability:
- `keel audit` lists every contract, whether it was proved statically, and whether tests exercised it.
- Contracts appear in generated docs and diagrams.
- Every `ignore` of an error and every contract opt-out appears in the audit report.

## 9. Testing

- `test` blocks live beside the code they test, in `*_test.keel` files in the component directory.
- A component is testable in isolation by construction: its only dependencies are its `requires` ports.
- `fake` blocks implement a port for tests; the compiler can generate a default fake from the port.
- `keel test Billing` builds and runs one component's tests without building the rest of the system.

## 10. Compiler intelligence

Beyond type checking, the compiler reports (as diagnostics with fix-its):

- **Near-duplicate code** across the project (R2), pointing to the existing function to reuse.
- **Error masking** patterns (R3).
- **Architecture violations** (§5.4) with the exact dependency edge that broke the rule.
- **Architecture diffs**: `keel archdiff main..HEAD` reports added/removed ports, dependencies and
  contracts — the review surface for R6.
- Dead code and unused ports.

Every diagnostic has a stable code, a human message, a *hint* explaining how to fix it idiomatically,
and a machine-readable form (JSON).

## 11. Tooling that instructs the model

Tooling assumes one of its users is an LLM (R8):

| Command | Purpose |
|---|---|
| `keel guide` | Prints a compact, prompt-sized language primer and project conventions. The whole spec is kept small enough for this to be complete. |
| `keel context <component>` | Emits the minimal context for working on one component: its declaration, its ports and contracts, and signatures of what it requires (R1). |
| `keel query <symbol>` | Exact lookup against the compiler's symbol index; the cure for hallucinated APIs (R4). |
| `keel explain <code>` | Long-form explanation of a diagnostic, with examples. |
| `keel doc` / `keel diagram` | Docs and C4 diagrams at every level — context, container, component, code — generated from source (§5.6). |
| `keel audit` / `keel archdiff` | Contract and architecture audit (§8, §10). |
| `keel fmt` | The one formatter. |
| `keel check` / `keel build` | Holistic boundary check; parallel per-component build (§12.2). |

All commands have `--json` output. An LSP server and an MCP server expose the same capabilities to
editors and agents.

## 12. Runtime

- Native compilation, tracing GC.
- **Backend: emit Go source first, LLVM later.** The compiler lowers to its own intermediate
  representation (IR), and backends consume only that IR — never the syntax tree — so a second
  backend needs no front-end changes.
  - *Go backend (first).* Keel gets Go's GC, goroutine scheduler, channels, and cross-compilation
    for free, and Keel's semantics map onto Go closely. Emitted code carries `//line` directives, so
    Go's stack traces, profilers, and debuggers report `.keel` file positions, not generated Go.
  - *LLVM backend (deferred).* Deferred because it requires Keel's own GC and scheduler, which are
    months of work that block nothing in the language design. Work starts when the IR is stable
    — the self-hosted compiler (§12.1) builds on it unchanged across releases.
- Concurrency model follows Go (goroutine-style tasks, channels) unless a later decision changes it.

### 12.1 Compiler implementation

- **Written in Go**, then **self-hosted in Keel**. Keel's semantics sit close to Go's, so the
  compiler's source can be translated largely mechanically — the path Go itself took from C at 1.5.
  The self-hosted compiler becomes Keel's first large program and the proving ground for §5.
- **Query-based and incremental from the start.** The compiler is a memoized query engine ("type of
  X", "ports of node Y", "context pack for Z") rather than a batch pipeline, so the LSP and the
  tooling in §11 answer after each edit by recomputing only what changed. Go has no equivalent of
  Rust's `salsa`, so this engine is built in-house.

### 12.2 Compilation modes

Compilation has two separate modes that run in parallel with each other.

| | **Holistic check** (`keel check`) | **Per-component build** (`keel build`) |
|---|---|---|
| Purpose | Enforce boundaries; find errors that need global context | Compile code fast |
| Input | Every declaration file, every port, every signature; bodies only for global analyses | One leaf directory plus the port declarations it owns |
| Finds | All of §5.4 (ports-only, wiring, layers, cycles, port compatibility), topic rules (§5.5–5.5.1), near-duplicate code across leaves (§10), architecture diffs, the project-wide audit | Type, error-handling, null, bounds, overflow, and contract errors inside the leaf |
| Parallelism | One pass over the whole tree | One job per leaf, all leaves in parallel |

- **Why per-leaf compilation is possible.** A leaf declares every port it requires (§5.3), so it
  compiles against its own declarations — never against another leaf's code. A change inside one
  leaf rebuilds that leaf only.
- **Both must pass before linking.** A binary is produced only when the holistic check passes and
  every leaf has compiled. Neither mode trusts the other's results implicitly.
- **Both run on every build.** `keel build` starts the holistic check alongside the per-leaf jobs;
  the holistic check reads mostly declarations, so it is fast and rarely the slowest job.
- **Caching.** Each leaf's result is cached, keyed by a hash of its directory contents and the
  declarations it depends on. The holistic check is incremental through the query engine (§12.1).

**Why this is feasible.** Go already compiles packages in parallel against dependencies' export
data (declarations only). Keel removes the usual obstacles by design:

- Consumer-owned ports (§5.3): a leaf compiles against its own declarations, not providers' code.
- Body-only inference (§6): no type is inferred from another leaf's call sites.
- The leaf size budget (§5.4 rule 11) bounds the largest job, and so the critical path.
- Global body analyses use per-leaf output: each leaf job emits normalized function fingerprints,
  and the holistic check compares fingerprints for near-duplicates without re-reading bodies.

On the Go backend each leaf becomes one Go package and the generated wiring becomes `main`, so Go's
toolchain adds its own parallelism. Shared types live in libraries (§5.4a), which compile first;
leaves then compile in parallel against library declarations.

## 13. Non-goals (for now)

- Manual memory management or ownership types — excluded by the GC principle.
- Inheritance and class hierarchies — excluded by "OO light".
- Macros / metaprogramming — each one makes code less navigable and less predictable for a model.
- Effects/capabilities system (architecture option C) — deferred because it conflicts with Go
  ergonomics; revisit once components and ports are proven.

## 14. Open questions

| # | Question |
|---|---|
| Q1 | Final name and file extension. |
| Q14 | *Parked.* Should the compiler enforce which leaves may call an `external` system? Parked because it depends on how external client libraries and foreign code are modeled, which follows the core port model; nothing else in the spec depends on it. |

## Appendix A — Sources

- Sonar, *Your AI agent doesn't know your codebase's context or constraints* — https://www.sonarsource.com/blog/your-ai-agent-doesnt-know-your-codebases-context-or-constraints/
- Packmind, *Context engineering for large codebases* (cites Chroma 2025, 18 models) — https://packmind.com/context-engineering-ai-coding/context-engineering-large-codebases/
- *Codified Context: Infrastructure for AI Agents in a Complex Codebase* — https://arxiv.org/html/2602.20478v1
- GitClear, *The Maintainability Gap* (2026) — https://www.gitclear.com/the_ai_code_quality_maintainability_gap
- *Package Hallucinations* (USENIX ;login:) — https://www.usenix.org/publications/loginonline/we-have-package-you-comprehensive-analysis-package-hallucinations-code
- *On Mitigating Code LLM Hallucinations with API Documentation* — https://arxiv.org/abs/2407.09726
- *Type-Constrained Code Generation with Language Models* — https://arxiv.org/pdf/2504.09246
- METR, developer productivity study and 2026 design update — https://metr.org/blog/2026-02-24-uplift-update/
- *"An Endless Stream of AI Slop"* — https://arxiv.org/pdf/2603.27249
- Sonar, *The AI trust gap* — https://www.sonarsource.com/blog/ai-coding-trust-gap/
- *No Resource, No Benchmarks, No Problem?* (no-resource languages) — https://arxiv.org/html/2606.16827v1
- *A Survey on LLM-based Code Generation for Low-Resource and Domain-Specific Languages* (ACM TOSEM) — https://dl.acm.org/doi/10.1145/3770084
