# Keel: Language Specification (Draft 0)

> **Working name: Keel.** The name is a placeholder (Q1). The source file extension is `.keel`.
> Status: brainstorming draft, 2026-09-28. Nothing in this document is implemented.

## 1. Purpose

Keel is a compiled, garbage-collected, statically typed language. Humans and LLMs use it to write
software *together*.

The thesis of Keel: most failures in human/AI co-development come from missing structure. The
model cannot see the architecture. It invents APIs, it duplicates code instead of reusing it, and it
masks errors. The human cannot review what the model wrote. Keel puts that structure in the
language. Then the compiler can see the structure, enforce it, draw it, and teach it.

### 1.1 Premise

The author presupposes that, in AI-augmented development, a developer spends most of the time on
three tasks:

1. The developer tells the model the requirements.
2. The developer reads the code that the model produced.
3. The developer validates the model's interpretation and implementation of the requirements, and
   corrects them.

In all three tasks, predictable structure, architectural legibility, and compiler guard rails help.
They make tasks 2 and 3 shorter, so that the developer can spend more time on task 1 and do it
faster. Each design decision in this spec serves that goal.

### 1.2 Terms

This spec uses each term below with one meaning only. The compiler, the tools, and the diagnostics
use the same terms.

| Term | Meaning |
|---|---|
| **System** | The whole software product. The root node of the architecture tree. |
| **Container** | A node that runs as one process. Most systems have one container, so they compile to one binary. For more containers, the project chooses one binary with a mode per container, or one binary per container (§12). (The C4 meaning. It is not a Docker container.) |
| **Group** | An optional node inside a container that groups leaves, for zoom levels in diagrams. |
| **Composite** | A system, container, or group. A section of `system.keel`, with no code. |
| **Leaf** | A `component` node: one flat directory of code. |
| **Library** | A node that holds shared types and reusable code, with no ports, no topics, and no mutable state. It can use an I/O resource but never acquire one. |
| **Foreign library** | The only kind of library that can import a Go package. It converts values at the boundary. |
| **Gateway** | The one leaf in a container that talks to a given external system. |
| **External system** | Software outside the system, such as Stripe. Declared with `external`. |
| **Person** | A human who uses the system. Declared with `person`. For diagrams. |
| **Port** | An interface with contracts, declared once by the leaf that requires it. A provider satisfies it with an ordinary type, named at the wire. Inside a container, a port call is an ordinary function call. |
| **Topic** | A publish channel, `topic[T]`, for asynchronous events between containers. |
| **Remote port** | A method port that crosses containers, declared `needs remote port`: a synchronous call. Events between containers use topics, which are not ports. |
| **Publisher / subscriber** | A leaf that declares `publishes X topic[T]` / a leaf that declares `subscribes X topic[T]` and handles it. |
| **Handler** | The function, marked `handle`, that processes the messages of one subscribed topic. |
| **Wire** | A line in `system.keel` that connects a required port or a subscribed topic to a provider. |
| **Transport** | A node kind that carries topic messages and remote calls between containers, named on a wire with `via`. With foreign libraries, the only kind of node that can import Go. |
| **Dead-letter topic** | Where a subscription sends a message that it cannot handle. |
| **Acknowledge** | Mark a message as handled. It occurs when its handler returns `nil`. |
| **Task** | A unit of concurrent work, started with `go` inside a `tasks { }` scope. |
| **Scope** | A `tasks { }` block. It owns the tasks that start in it and waits for them. |
| **Error** | An ordinary value that reports a failure the code expects and handles. |
| **Abort** | The stop of a task after a broken invariant (a bug) or an environment failure that the code cannot handle, such as a full disk under a publish. It is not an exception. |
| **Contract** | A `requires`, `ensures`, or `invariant` clause. |
| **Audit** | The `keel audit` report of each escape hatch, override, and unproved claim. |

## 2. Design principles

| Principle | Meaning in Keel |
|---|---|
| Compiled, GC | Native binaries and a tracing GC. No manual memory management and no borrow checker. |
| Go ergonomics | Go is the syntax model. Keel changes Go only where Go is weak, for example error enforcement and nil safety. |
| Competitive performance | Resource use and speed must be competitive with Go, Java, C++, and Rust. §12.3 states the targets and the costs. |
| Compiler first | If the compiler can check a rule, the compiler checks it. Each rule that depends on convention, review, or runtime is a candidate to move into the compiler. This is the most direct way to make tasks 2 and 3 of §1.1 shorter. |
| Rust-grade compiler intelligence | Diagnostics explain *why* and propose fixes. The compiler checks exhaustiveness and finds duplication and error masking. |
| Imposed structure | One project layout, one formatter, and one location for each kind of thing. |
| Architecture is first-class | Systems, components, and ports are language constructs. The compiler enforces them. |
| Self-documenting, self-diagramming | The tools generate docs and architecture diagrams from the source. Nobody maintains them by hand. |
| Contracts are code | Preconditions, postconditions, and invariants are part of declarations. Reviewers can audit them. |
| Tooling teaches | The tools write every output for an LLM reader and a human reader. |
| OO "light" | Structs, methods, interfaces, and generics. No inheritance and no classes. |

## 3. Evidence behind the design

This table shows how LLMs fail when they write applications with humans, and what Keel does about
each failure. Appendix A lists the sources.

| # | Observed failure | Keel's answer | Section |
|---|---|---|---|
| R1 | Agents get lost in large codebases. Quality drops as the context grows. | Leaves are small, self-contained units with explicit ports. `keel context` emits a minimal context pack. | §5, §11 |
| R2 | Duplication is up 81%. Reuse (cross-file calls) is down 35%. | The compiler finds near-duplicate code. Ports in each `component.keel` and all wires in `system.keel` make the available capabilities easy to find. | §10 |
| R3 | Error-masking constructs are up 47%. | The code must handle each error explicitly. A silent discard is a compile error. | §7 |
| R4 | Models invent packages (about 20% of the time) and APIs. | The compiler serves an exact symbol index that tools can query. There are no implicit imports. | §11 |
| R5 | About 94% of compile errors in typed languages are type errors. | Strong static types, and inference that works on incomplete programs, so tools can guide generation by type. | §6 |
| R6 | AI code is harder to review. It "looks right but isn't". | Contracts in the code that reviewers can audit, and diffs at the architecture level. | §8, §10 |
| R7 | Developers feel faster than they measure. | Fast verification: incremental builds and per-leaf tests. | §9 |
| R8 | A new language has no training data, so LLM quality drops steeply. | Syntax close to Go, so the model can use what it knows. A primer (`keel guide`) small enough for a prompt; the full spec is reference. Tools that instruct the model. | §4, §11 |

## 4. Syntax

Go is the model. A Go programmer, or a model trained on Go, must be able to read Keel at first
sight. This spec must justify each change from Go and list it in this table:

| Change from Go | Reason |
|---|---|
| `system`, `container`, `group`, `component`, `library`, `port`, `person`, `external` declarations | Architecture as a first-class construct (§5) |
| `(T, error)` keeps Go's spelling (`return v, nil`, `return x, err`) but is a sum type: a value or an error, never both. An unhandled error is a compile error. Postfix `?` propagates an error. | R3, and the value is never usable on the error path (§7) |
| `partial` functions return a value and an error together, as in Go | No data lost at Go boundaries such as `io.Reader` (§7) |
| `nil` and zero values as in Go, but each use that can panic on `nil` (a dereference, an interface method call, a function-value call, a map write, a channel close) needs a compile-time proof. Parameters and receivers are non-`nil` at entry; `nonnil` marks Go results and fields. `?T` for optional values of any type. | No null dereference can occur (§7a) |
| No exceptions of any kind. No `try`, `catch`, `throw`, or `recover()`. | Errors are values. Only a supervising task handles an abort (§7, §7.1) |
| Signed integer overflow aborts; unsigned arithmetic wraps as in Go. Signed code that wraps on purpose behaves differently from Go (warning K0701). | Signed overflow is almost always a bug (§7c) |
| `guarded by` on struct fields and `holds` on helpers, checked by the compiler | Fewer data races, with no ownership system (§7.1) |
| `go` runs only inside a `tasks { }` scope (structured concurrency); `detach` for background tasks. `t := go f()` is an expression that returns a task value; `t.Wait()` returns the sealed exit `Completed`, `Failed`, or `Aborted`. | Each error and abort has a known destination (§7.1) |
| `requires`, `ensures`, and `invariant` clauses | Contracts in the code (§8) |
| `pure` functions | Contracts can call them. The compiler verifies that they have no effects (§8.1) |
| `test` and `fake` blocks | Testability by design (§9) |
| `topic[T]` publish channels between containers | A publish does not wait for a subscriber to read, and a message has zero or more subscribers. A `chan` is different (§5.5) |
| `needs remote port` (synchronous methods with `timeout` and transport errors), `publishes` and `subscribes` topics, `idempotent`, `via` wiring | Honest communication between containers; the developer chooses a call or an event (§5.5, §5.5.3) |
| `topic.Publish(msg)` and `handle` functions; a message is acknowledged when its handler returns `nil` | No forgotten or early acknowledgement (§5.5.0) |
| `limit` bounds on variable-size fields | A transport can require bounded messages, so the compiler can calculate queue memory (§5.5.1) |
| `gateway for` and `entry for` leaves; only they call `acquires network` functions | Enforced network access (§5.4b, §5.4c) |
| `foreign library` and `transport` nodes are the only importers of Go packages; `acquires`, `acquires network`, and `nonnil` marks at the boundary | Keel guarantees hold outside a guarded boundary (§5.4c) |
| `entry for` leaves serve inbound traffic; `startup` functions; `main` is generated | Architecture as a first-class construct (§5.4b, §7.1) |
| Only leaves call `acquires` functions; libraries use handles they are given | Each I/O decision is visible in a leaf (§5.4a) |
| A library's package-level `var` is set once, at startup, and never assigned again | “No mutable state” in libraries (§5.4a) |
| `ignore err, "reason"` | Explicit discards that the audit lists (§7) |
| Sum types are `sealed interface` declarations, matched with Go's type switch, which must be exhaustive | Typed errors, task exits, and state machines (§6) |
| Declaration files (`system.keel` for the architecture, `component.keel` and `library.keel` for boundaries) have their own syntax: wires with paths (`Api/Orders.Checkout`), `uses A -> B`, `via`, `flag("...")`, transport settings (`via name{ ... }`), `gateway for`, `go "package"` | Architecture as a first-class construct. These files hold no Go-style code (§5) |
| Contract expressions add `result`, `old(x)`, and `=>` (implies) | Contracts in the code (§8.1) |
| `unreachable` statement | Marks code that must never run; reaching it aborts (§7) |
| `m.Get(k)` returns `?V`. Other map reads behave as in Go. | Explicit absence when the code wants it (§7a) |
| Test files are `*_test.keel` in the leaf directory | Tests next to the code (§9) |

All other syntax follows Go, unless a later section says otherwise. This includes `:=`, `func`,
structs, methods with receivers, `for`, `switch`, slices, maps, `defer`, packages as directories,
and gofmt-style formatting.

### 4.1 Keywords

Each reserved word breaks Go code that uses it as a name, and models write that code (R8). Thus
Keel reserves few words.

- **Reserved everywhere:** none. Keel reserves no word that Go does not.
- **Contextual:** every new word. A new word is a keyword only where it starts a declaration or a
  statement (`tasks {`, `ignore err, "..."`, `detach f()`, `handle X(`, `pure func`), or inside a
  declaration file or a contract, or after a field type (`guarded by`, `limit`). Elsewhere they are
  ordinary identifiers. This applies to
  `system`, `container`, `group`, `component`, `library`, `foreign`, `port`, `remote`,
  `requires`, `ensures`, `invariant`, `uses`, `via`, `topic`,
  `transport`, `nonnil`, `config`, `startup`,
  `test`, `fake`, `result`, `old`, `all`, `some`, `limit`, `guarded`, `handle`, `pure`, `tasks`, `detach`,
  `ignore`, `unreachable`, `entry`, `needs`, `bind`, `holds`, `publishes`, `subscribes`, `acquires`, `idempotent`,
  `gateway`, `partial`, `sealed`, `person`, `external`, `by`, `timeout`, and `flag`.

```go
port := 8080                   // an identifier
func read(in io.Reader)        // an identifier

needs port Payments { ... }    // keywords: they start a declaration
```

`keel guide` prints both lists.

## 5. Architecture model

The architecture is a tree that follows the C4 model. Code is only in small, flat leaves. The
architecture of the whole system is in **one file**, and the boundary of each leaf is in **one file
next to its code**. Thus a reader finds any wire in one location, and any boundary in one location.

### 5.1 Hierarchy

```
system Shop                         C4 System, with the people and external systems around it
 ├ container Api                    C4 Container: one process
 │  ├ component Billing   (leaf)    C4 Component: one flat directory of code
 │  └ component Orders    (leaf)
 └ container Worker
    └ group Fulfilment              optional, for zoom levels in diagrams
       ├ component Picking  (leaf)
       └ component Shipping (leaf)
```

A *leaf* is a `component` node. A *composite* is a `system`, `container`, or `group`. A composite
has no directory of its own declarations and no code: it is a section of the architecture file.

| Kind | Declared in | Holds | Code? |
|---|---|---|---|
| `system` | `system.keel` | containers, `person` and `external` declarations, all wires | no |
| `container` | a section of `system.keel` | groups and leaves | no |
| `group` | a section of `system.keel` | leaves; a label for diagrams only | no |
| `component` | `component.keel` in the leaf directory | the boundary of the leaf | yes, in the same directory |
| `library` | `library.keel` in the library directory | shared types and reusable code | yes, but no ports, topics, or mutable state (§5.4a) |

### 5.2 Layout

- **`system.keel`**, at the root, is the architecture file. It declares the containers and groups,
  the leaves in each, the people, the external systems, and **every wire**.
- **Each leaf is a directory** with one `component.keel` and its source files. `component.keel`
  declares the **boundary** of the leaf: each port that it needs, each topic that it publishes or
  subscribes to, and its `gateway for` or `entry for` line.
- **Directories mirror containers and leaves**: `api/orders/`, `worker/picking/`. Groups are only
  labels in `system.keel`, so they are not directories and not part of any path. The compiler checks
  that the directories and `system.keel` agree.
- Libraries are in `lib/`.

Thus each thing has one location: the wires in `system.keel`, a boundary in its `component.keel`,
and code next to its boundary.

### 5.3 Declarations

**A port is declared once, by the leaf that requires it.** This is the Go idiom: the consumer
defines the interface where it uses it. A leaf declares its boundary in `component.keel`: each
port that it requires, with its methods and contracts.

```go
// api/orders/component.keel
component Orders {
    // Checkout takes payment for a confirmed order.
    needs port Checkout {
        func Pay(customer CustomerID, amount Money) (Receipt, error)
            requires amount > 0 && amount < MaxOrder
            ensures  result.Amount == amount
    }
}
```

**A provider satisfies a port with an ordinary type.** It needs no keyword and no declaration: any
exported type of a leaf whose method set matches the port can satisfy it, as with a Go interface.
The provider never names its consumers.

```go
// api/billing/payer.keel (ordinary code)
// Payer charges a customer's stored payment method.
type Payer struct { ... }

func NewPayer() *Payer { return &Payer{} }

func (p *Payer) Pay(customer CustomerID, amount Money) (Receipt, error) { ... }
```

If two consumers need different methods with the same name, the provider exports two types, for
example `Payer` and `SubscriptionPayer`, as in ordinary Go.

A consumer does not know which leaf serves it. A provider does not know which leaves use it. Only
`system.keel` knows both.

**Inside a leaf, each required port is a package-level name**, the name of the port itself. Code
calls it like a Go package: `Checkout.Pay(c, amt)`. The wiring binds the name at startup, and the
binding cannot change after that. A test binds fakes to the same names (§9). A leaf appears once in
the tree; reuse of code goes through libraries (§5.4a).

**A port call is an ordinary function call.** Keel does not invent a new way for components to
communicate. Inside one container, a port call runs on the caller's own task, usually the same
thread, with no queue, no copy to bytes, and no hidden goroutine. Arguments pass by value or by
reference, exactly as in Go. On the Go backend, a port call is a Go interface call: one dynamic
dispatch, because each leaf compiles alone against its own port declaration (§12.2). The LLVM
backend can use the fixed wiring to make it a direct call. What Keel adds at the boundary is **enforcement and
instrumentation of the contract**: the port signature, its `requires` and `ensures`, and the
record of the boundary in diagrams and in the audit. Topics and transports (§5.5) exist only for
separate processes.

**The architecture file wires the leaves.** A wire connects a required port to a providing type.
Inside a container section, a wire names leaves by name. A wire between containers names each leaf
by its path: `Container/Leaf`. Groups are not part of paths, so regrouping for a diagram never
changes a wire. Every wire reads `needer: supplier`: the left side needs what the right side
supplies, a method for a port and messages for a topic. `keel query` lists the subscribers of a
topic in one command.

```go
// system.keel
// Shop is the online store.
system Shop {
    person   Customer "Buys goods through the web store"
    external Stripe   "Card payment processor"

    container Api {
        Web, Billing, Orders, LedgerStore, SystemClock, StripeGateway, Metrics

        Orders.Checkout: Billing.NewPayer
        Billing.Ledger:  LedgerStore.Store
        Web.Checkout:    Billing.NewPayer
        Billing.CardNetwork: StripeGateway.NewCards{ apiKey: flag("stripe-key") }
        Billing.Clock:   SystemClock.Wall
    }

    container Worker {
        Alerts                           // subscribes DeadPlaced and DeadShipped (DeadLetter topics)
        Inventory
        group Fulfilment { Picking, Shipping }
    }

    container Analytics {
        Ingest
    }

    // Between containers: topics and remote method ports only (§5.5).
    Worker/Picking.Placed: Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }
    Api/Orders.Shipped: Worker/Picking.Shipped via direct{ deadletters: Worker/Alerts.DeadShipped }
    Analytics/Ingest.Placed:  Api/Orders.Placed  via direct{ deadletters: Worker/Alerts.DeadPlaced }
    Analytics/Ingest.Samples: Api/Metrics.Samples via kafka{ topic: "samples" }
    Api/Orders.Stock: Worker/Inventory.NewLevels via direct{}

    // Customer -> Api comes from the entry leaf Web; Api -> Stripe from the gateway (§5.4b).
}
```

The wire names an exported type or an exported constructor, for example `Billing.NewPayer`, which
returns `*Payer`. A wire follows Go's method-set rules: `Pay` has a pointer receiver, so only
`*Payer` satisfies the port, and the wire names a constructor that returns one.

- **Dependencies** come through the provider's own required ports, so a constructor needs no
  parameters for them.
- **Configuration** comes through one optional settings struct, filled at the wire from literals or
  startup flags, as for transports: `StripeGateway.NewCards{ apiKey: flag("stripe-key") }`, which fills the exported settings struct
  of `NewCards`.
- **A constructor can fail.** It can return `(T, error)`. A failure stops the process at startup
  with a report.
- **One instance for each wired constructor, in each process.** Two wires to the same constructor
  share one instance.

**Checks at each wire.**

1. **Method set.** The providing type must have each method of the port, with an identical
   signature. It can have more methods.
2. **One contract.** The contract of the port applies at every call through the port: the compiler
   checks `requires` before the call and `ensures` after it. If the providing method also declares
   its own contract, the compiler checks that the port's `requires` implies the method's `requires`,
   and that the method's `ensures` implies the port's `ensures`. If it cannot prove an implication,
   it adds a runtime check at the wire and lists it in `keel audit`.

If no type has the right methods, write an adapter: an ordinary leaf with a type that satisfies the
port, and that calls another port. Keel has no special adapter construct.

Keel does not wire external systems with ports. The language does not control them, and they use
their own protocols. Instead, one **gateway leaf** in each container talks to each external system
(§5.4b). The compiler draws each relationship with a person or an external system from the entry
and gateway declarations. A `uses` line is needed only for a relationship that no declaration
shows.

### 5.4 Rules the compiler enforces

1. **Ports only.** Code in a leaf can use another leaf **only** through a port that the leaf
   needs, a topic that it publishes or subscribes to, or an imported library (§5.4a). Any other reference to another leaf is a
   compile error.
2. **Code only in leaves and libraries.** `system.keel` holds no code. A leaf directory holds no
   child leaves.
3. **Container boundaries.** A wire between leaves in different containers must be a topic or a
   `needs remote port` (§5.5). A method port inside one container must not be `remote`, and a
   topic cannot connect two leaves of one container (§5.3: calls inside a container are ordinary). A group is only a label for diagrams: it
   adds no boundary.
4. **Complete wiring.** A `deadletters:` setting wires a `DeadLetter[T]` topic, and rule 3 does not
   apply to it. Otherwise, `system.keel` wires each required port and each subscribed topic of each leaf
   exactly once. An published topic is never wired by itself (§5.5).
5. **Satisfied ports.** Each wire names a providing type whose method set matches the port.
6. **Compatible wiring.** Each wire passes the port-compatibility check in §5.3.
7. **Boundary ownership.** A leaf declares each port that it requires, once, in its own
   `component.keel`. The code and the boundary of a leaf never name another leaf. Only
   `system.keel` names two leaves together, in a wire.
8. **No cycles.** The graph of wires between the leaves of one container has no cycles.
9. **Private types.** A type is private to its leaf. Only `system.keel` can name an exported type
    of a leaf, in a wire. A type in a port signature must come from a library (§5.4a). This costs a
    library for each shared type, which is the price of leaves that never import each other. The
    compiler reduces it: when a port signature uses a leaf-private type, the fix-it moves the type
    to a library (new or existing) and updates the imports.
10. **Leaf size (audit).** `keel audit` reports the size of each leaf: its `component.keel` and
    its non-test source, in non-blank, non-comment lines. A leaf over the project's
    `context_budget` (set in the `config { }` section of `system.keel`) gets a warning, with a hint
    that tells where to split it. It is not a compile error, so one more line never forces a
    split.

### 5.4a Libraries

Some types cross a wire. For example, `Money` is in the port signatures on both sides. Such a type
must be the *same* type on both sides, so it cannot belong to either leaf. It goes in a
**library**. A library is a node that holds shared types and reusable code. It is not part of the
architecture at runtime.

```go
// lib/money/library.keel
// Money represents amounts in minor currency units.
library Money {
}
```

```go
// api/billing/billing.keel
import "lib/money"
```

Rules:

1. A library has **no ports, no topics, and no mutable package-level state**. It is only code. It
   cannot hold state, so it cannot couple the leaves that import it. The compiler enforces this
   rule as follows:
   - A package-level `var` in a library is set once, at startup, and never assigned again. The
     compiler rejects each later assignment. It does not track changes through a reference, such
     as a write into a map that a `var` holds; the audit lists each library `var` of a map, slice,
     or pointer type, so that a reviewer can check it.

   ```go
   // lib/crc
   var table = buildTable()                         // set once, at startup
   var ErrNotFound = errors.New("not found")

   table = nil                                      // compile error: assigned after startup
   ```
2. **A library enables I/O. It does not decide on I/O.** A library can read, write, query, or
   send through a resource that its caller gives it. A library cannot *acquire* a resource: it cannot
   open a file, dial the network, open a database, or create a client. A call from a library to an
   acquiring function (§5.4c) is a compile error. Thus each I/O decision is in a leaf, where the
   diagram and the reviewer see it.

   ```go
   // lib/csvutil: allowed. The caller chose the file.
   func Parse(r Reader) ([]Row, error)

   // lib/orderlogic: compile error. The library chose to open a database.
   func Reprice(id OrderID) error {
       db := sql.Open(config.DSN)?
       ...
   }
   ```
3. A library can import other libraries. It cannot import leaves or composites. Library imports
   have no cycles.
4. Any leaf can import any library.
5. `keel diagram` shows libraries as shared dependencies. `keel audit` lists the leaves that use
   each library.

Libraries compile before the leaves that import them. Leaves compile against the library
declarations only (§12.2).

### 5.4b Gateways to external systems

A **gateway leaf** is the only leaf that talks to an external system. It turns the protocol of the
external system into an ordinary type that satisfies a port. Other leaves require that port, and
they contain no network code.

```go
// api/stripegateway/component.keel
// StripeGateway charges cards through Stripe.
component StripeGateway {
    gateway for Stripe
}
// api/stripegateway/cards.keel: type Cards, with a Charge method that calls Stripe

// api/billing/component.keel
component Billing {
    needs port CardNetwork { func Charge(...) (CardReceipt, error) }   // no network code
}

// system.keel
Billing.CardNetwork: StripeGateway.Cards
```

Rules:

1. A container that talks to an external system has exactly one gateway leaf for that system. The
   gateway declares `gateway for <External>`, and `<External>` must be an `external` declaration in
   the system.
2. Network access follows the single rule in §5.4c, rule 2. A gateway can use any foreign library,
   including an SDK for its external system or `net/http`.
3. The compiler generates the `uses` relationships in diagrams from the gateway declarations. Thus
   the diagram cannot be wrong.

**Entry leaves.** Traffic that comes **into** the system, for example HTTP requests from a `person`,
enters through an **entry leaf**. It is the mirror of a gateway:

```go
// api/web/component.keel
component Web {
    entry for Customer          // serves requests from the person Customer
    needs port Checkout { ... }
}
```

- An entry leaf declares `entry for <Person or External>`, and it can call `acquires network`
  functions to serve requests, under the one network rule (§5.4c, rule 2).
- The compiler cannot tell serving from dialing, so it does not try. `keel audit` lists each entry
  leaf and each gateway, and the `acquires network` calls in each.
- Each request that an entry leaf serves runs in its own implicit scope (§7.1).
- The compiler draws the `uses` relationship for each entry leaf, as for gateways.

Why a gateway (§1.1): each rule about an external system goes in one leaf. A reviewer reads all
traffic to that system in one location. Tests of other leaves fake a port with contracts, not HTTP.
The compiler rejects network access in any other leaf (§5.4c, rule 2), so the reviewer does not have to find it.

### 5.4c Foreign libraries (Go interop)

Keel uses Go packages, but only through a guarded boundary. A **foreign library** is the only node
that can import a Go package. It gives a Keel API to the rest of the program.

```go
// lib/stripeclient/library.keel
// StripeClient wraps the Stripe Go SDK.
foreign library StripeClient {
    go "github.com/stripe/stripe-go/v82"
}
```

**The boundary converts each crossing value:**

| From Go | To Keel |
|---|---|
| a value that can be `nil` | the same value; the proof rules of §7a apply, unless the foreign library marks it `nonnil` |
| a Go `error` | a Keel error |
| a panic | an abort of the calling task (§7.1) |
| a Keel function passed to Go as a callback (for example an `http.Handler`) | runs in a wrapper that applies Keel's abort rules before any Go frame sees the panic, so Go code such as `net/http` cannot recover a Keel abort |
| a pointer to a struct with an invariant, after Go code has used it (for example `json.Unmarshal`) | re-checked against its invariant when it returns to Keel |
| an integer | an integer; Keel arithmetic on it follows §7c |

**Rules:**

1. Only a foreign library or a transport (§5.5.1) can import a Go package. Leaves, gateways, and other libraries import the
   foreign library, never Go directly.
2. **Network access.** A foreign library marks each function that opens a network connection or
   listens on one with `acquires network`, for example `acquires network func Dial(...)`. **Only
   gateway leaves and entry leaves can call an `acquires network` function.** This is the one rule
   for network access; other sections refer to it. The compiler cannot tell a dial from a listen in
   Go code, so an entry leaf that dials is not a compile error; `keel audit` lists each
   `acquires network` call in an entry leaf. A database is always an `external` system, even an
   embedded one: its driver's open function is `acquires network`, so only its gateway can use it.
3. The Keel side of a foreign library has no mutable package-level state, the same as any library
   (§5.4a).
4. A foreign library marks each function that acquires an I/O resource with `acquires`, for example
   `acquires func Open(path string) (File, error)`. Only leaves can call an acquiring function
   (§5.4a, rule 2). The standard library ships with these marks. `keel audit` lists each
   acquiring function, so that a reviewer can check the marks of third-party foreign libraries.
5. Go code inside the boundary can break Keel guarantees. It can start unsupervised goroutines, use
   locks outside Keel's rules, and hold global state. `keel audit` lists each foreign library
   and these limits, so that a reviewer can see where the guarantees stop.
6. A foreign library marks each Go result **and each Go struct field** that is never `nil` with
   `nonnil`, for example `nonnil URL` on the field `http.Request.URL`. A `nonnil` value needs no
   proof, so Go code such as `r.URL.Query().Get("id")` works unchanged. If a `nonnil` result is `nil` at runtime, the boundary
   aborts. The standard library wrappers ship with these marks. `keel audit` lists each `nonnil`
   mark in a third-party foreign library, so that a reviewer can check it.

**The standard library.** The Keel standard library is a set of foreign libraries over the Go
standard library, for example HTTP, JSON, SQL, and cryptography. It ships with the compiler.

**Versions.** A Keel project uses Go modules (`go.mod`) to pin the versions of Go packages.

### 5.5 Crossing containers

Containers are separate processes. Two mechanisms cross from one container to another, and the
developer chooses one for each boundary:

- **A remote method port**, for a synchronous request and response (§5.5.3).
- **A topic**, for asynchronous events. A topic is a publish channel. It is like a Go channel, but
  it works across the process boundary.

The language supplies the **mechanism** of both. A **transport** carries them, and the transport
owns the operational **policy**: durability, retries, dead letters, capacity, full-queue behavior,
expiry, tracing, and delivery guarantees (§5.5.1). The compiler type-checks the settings of each
transport.

A topic has its own type, `topic[T]`. It is not a Go `chan T`, because the two behave differently:

- An unbuffered send on a Go channel waits for a receiver. Each value has exactly one
  subscriber.
- A publish on a topic does not wait for a subscriber to read the message. Each message has zero,
  one, or many subscribers.

One keyword with two behaviors would mislead humans and models trained on Go (R8). A Go `chan`
keeps its Go behavior for concurrency inside one process.

```go
// api/orders/component.keel. Topics are boundary entries, not ports.
component Orders {
    publishes  Placed  topic[OrderPlaced]
    subscribes Shipped topic[ShipmentSent]
}

// worker/picking/component.keel
component Picking {
    subscribes Placed  topic[OrderPlaced]
    publishes  Shipped topic[ShipmentSent]
}

// In system.keel. Each subscriber names its source; the wire names the transport and settings.
Worker/Picking.Placed: Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }
Analytics/Ingest.Placed:           Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }
Api/Orders.Shipped: Worker/Picking.Shipped via direct{ deadletters: Worker/Alerts.DeadShipped }
```

**Why publish: it is the loosest coupling possible.** The only job of the publisher is to publish.
The publisher does not know, and must not care, if a subscriber reads its messages. It does not know
how many subscribers there are, or when they read.

These rules are part of the language, for every transport:

- **Publish, do not wait for a reader.** A publish gives the message to the transport and returns.
  Subscribers cannot make a publish fail. What happens when a queue is full is the policy of the
  transport. **Under the default `block` policy of `direct`, a publish waits when a subscriber's
  queue is full, so a slow subscriber can slow the publisher.** A developer who wants the publisher
  never to wait chooses a drop policy for that wire.
- **Zero, one, or many subscribers.** `system.keel` wires topics one at a time, from the
  subscriber side. Each subscribed topic is wired exactly once, to one published topic of the same message type. An
  published topic is never wired by itself: its subscribers are the subscribed topics wired to it, zero or
  more. Thus a new subscriber adds one wire, and the wiring of the publisher does not change.
- **An unread topic causes a warning.** A topic with zero subscribers compiles, with a warning. A
  publisher can exist before its subscribers, but an unread topic must not go unnoticed.
- **A topic has no reply.** A reply would bind the publisher to a subscriber. For a request and
  response, use a remote method port (§5.5.3). If a publisher needs to know what happened later, it
  subscribes to a separate event that the other side publishes. `Shipped` in the example above is
  such an event.
- **A state change and a publish are not atomic.** In `orders.Save(o)?; Placed.Publish(m)`, the
  process can stop between the two lines, and the event is then lost. Keeping them together is the
  developer's decision: for example, the transactional outbox pattern (write the event to the same
  database transaction, and publish it from there). Keel does not hide this risk.
- **A read model is an alternative to a synchronous read.** The owner publishes change events. The
  reader subscribes, keeps a local copy, and answers queries from that copy. This is
  "event-carried state transfer". Reads can be slightly stale, and each reader stores its own copy.

  ```go
  // Worker owns stock and publishes each change.
  Stock.Publish(StockChanged{SKU: s, Qty: q})      // topic Stock carries StockChanged

  // Api keeps a read model, and answers GET /stock/42 from it.
  idempotent handle Stock(m StockChanged) error {
      stock.Set(m.SKU, m.Qty)
      return nil
  }
  ```

- **No raw network calls between containers.** A leaf cannot call another container over the
  network by hand, because only gateway and entry leaves can reach the network (§5.4c, rule 2). `keel audit`
  lists each `external` system with its gateway, so a reviewer can find an `external` that is
  really a container of the same system.
- **Ownership.** Each side owns only its end of each topic. Only message values cross, as copies.
  The two sides share no memory and no references.
- **Only topics and remote method ports cross.** The compiler rejects a wire across containers for
  a method port that is not declared `needs remote port`.
- **Contracts** on message types (struct `invariant`s) are checked when a message is published and
  again when it is received. Message types must be serializable. They cannot contain functions,
  topics, or pointers into the memory of the publisher.
- **A subscriber failure never reaches the code of the publisher.** A handler error, a handler
  abort, and a received message that fails its invariant all go back to the transport, which
  applies its policy (§5.5.1).
- **Idempotency is declared.** A transport states, in its interface, whether it can deliver a
  message more than once. If it can, a subscriber's handler must be marked `idempotent`. Without
  the marker, the code does not compile. The compiler cannot prove the claim, but the marker makes
  a hidden assumption visible. It appears in `keel context`, `keel doc`, and `keel audit`.

#### 5.5.0 Publishing and handling

**Publish** is a method on a published topic. It is not Go's `<-`, because a topic does not behave
like a `chan`:

```go
Placed.Publish(OrderPlaced{ID: id, Total: t})
```

**A handler** is a function bound to one subscribed topic with `handle`:

```go
idempotent handle Shipped(m ShipmentSent) error {
    orders.MarkShipped(m.OrderID)?
    return nil          // the message is acknowledged here
}
```

- Each subscribed topic of a leaf has exactly one handler. Code cannot iterate over a subscribed topic.
- The runtime runs each message in its own supervised task (§7.1). By default, it runs the handler
  of one subscribed topic for one message at a time. A wire setting, for example
  `via direct{ concurrency: 8 }`, allows more; shared state then needs a lock. At most
  `concurrency` handlers run at once for one subscribed topic; other messages wait in the
  transport's queue, which its capacity bounds. With concurrency
  above 1, the runtime keeps the order that the transport guarantees: if the transport keeps order
  per key, the runtime runs at most one handler at a time for each key. Thus the order at the wire
  is also the order at the handler.
- **The acknowledgement point is the return of the handler.** If the handler returns `nil`, the
  message is acknowledged. If it returns an error or aborts, the message is not acknowledged, and
  the transport applies its policy: for example, retry and then dead-letter, or discard. Thus code
  cannot forget to acknowledge a message, and it cannot acknowledge a message too early.

#### 5.5.1 Transports

A transport carries topic messages and remote calls between containers. Transports matter only for
a system with more than one container. Inside one container, leaves use method ports, and nothing
crosses a network.

**The transport interface.** The language defines one interface. A transport implements it and
declares:

- **A typed settings struct.** Each wire passes settings with `via name{ ... }`. The compiler
  type-checks the settings like any struct literal. A transport can make a field required, for
  example a dead-letter route or a capacity. A missing required setting is a compile error.
- **Its guarantees, as constants that the compiler reads.** For example, whether the transport can
  deliver a message more than once (which makes `idempotent` required), and whether it keeps order.
- **Its policy.** Durability, retries, dead letters, capacity, full-queue behavior, expiry,
  tracing, and delivery guarantees belong to the transport, not to the language.

**The developer chooses.** Each wire names its transport and its settings. The project config can
set a default transport and default settings. Addresses and credentials never appear in source;
they come from startup flags or from the deployment config. `keel audit` lists the transport and
the settings of each wire.

**The built-in transport, `direct`.** Keel ships one transport, so that a small system with two
containers needs no broker. It sends from the publisher straight to each subscriber.

- **One queue for each subscriber.** A slow or failing subscriber fills only its own queue.
- **Settings:** `capacity`, `whenFull` (`block`, `dropOldest`, `dropNewest`; default `block`),
  `ttl`, `retries` (default 5, with exponential backoff), `deadletters` (a route; required), and
  `durable` (default `true`, which keeps queues on
  disk so that no message is lost across a restart). A value can come from a startup flag:
  `capacity: flag("orders-cap", max: 10_000)`. A flag that sets a size must declare a maximum, and
  the compiler uses the maximum to calculate the memory limit.
- **Delivery:** at least once, unordered. Thus `idempotent` is required on its handlers.
- **Bounded memory:** `direct` requires topic message types to be bounded with `limit` fields (§6),
  so that the compiler can calculate the memory limit of each queue: capacity × maximum message
  size. `keel audit` shows each limit and the total.
- **Dead letters:** a message that fails all its retries, or expires, goes to the dead-letter
  route. A drop under `dropOldest` or `dropNewest` is counted, not dead-lettered.
- **A dead-letter route is a subscribed topic** of type `DeadLetter[T]` in some leaf. The wire
  settings name it, and it can receive from any number of wires whose message type is `T`.
  Each dead letter carries the message, the last handler error or abort report, the attempt count,
  the wire, and the time. Thus "why did this message dead-letter?" has an answer without
  tracing.
- **Tracing** is off by default. The setting `trace: true` records each step of each message, and
  `keel trace <id>` shows them.

```go
Worker/Picking.Placed: Api/Orders.Placed via direct{
    capacity: 5_000, ttl: "24h", deadletters: Worker/Alerts.DeadPlaced }
Analytics/Ingest.Samples: Api/Metrics.Samples via kafka{ topic: "samples" }
// startup: --transport.direct.peers=worker=10.0.0.7:7000
```

Other transports, for example Kafka, NATS, or SQS, are **transports**: a node kind of their own, not
a library. A transport is declared `transport kafka { go "..." }` and named in lower case at the
wire (`via kafka{ ... }`). It is part of the runtime, not of the architecture, and it is the second
kind of node that can import Go packages. It is exempt from the library rules (§5.4a, §5.4c): it
can acquire network resources, hold connections, and keep queues. No leaf or library can import it;
only a `via` in `system.keel` can name it. It implements the transport interface: `Publish`,
`Subscribe`, and `Call`, each with its typed settings. `keel audit` lists each transport.

#### 5.5.2 Message evolution

Containers do not deploy at the same moment. Thus two builds of one system run together, and their
message types can differ. The holistic check (§12.2) sees only one build. So the compiler also
compares each message type with the last committed schema, in `topics.lock`. It cannot see the
deployed build; the lock stands for it.

- **The compiler owns the wire format.** It serializes each message in a binary format. Each field
  has a number, and the wire carries the number, not the name. A field number is never used again.
- **A lock file records each message schema.** `topics.lock`, in the system directory, lists each
  topic message type with its field numbers, field types, and invariants. The compiler assigns
  the field numbers, generates the file, and the project commits it.
- **`keel check` compares the current types with the lock.** A compatible change passes, and the
  compiler updates the lock. A breaking change is a compile error.

| Change | Result |
|---|---|
| Add a field of type `?T` | Compatible. A message from an older publisher has `nil` in the field. |
| Remove a field of type `?T` | Compatible. An older subscriber reads `nil`. |
| Rename a field | Breaking. Source has no field numbers, so a rename looks like a remove and an add. |
| Change an invariant (relax or tighten) | Breaking. Both sides check invariants, so a change in either direction rejects messages in one deploy order. |
| Add a required field (not `?T`) | Breaking. |
| Remove a required field | Breaking. |
| Change the type of a field | Breaking. |

- **A breaking change needs a new message type.** For example, `OrderPlacedV2`. During the
  migration, the publisher publishes both types. When no subscriber reads the old type, the
  project removes it.
- **A subscriber ignores unknown fields.** Thus an older subscriber can read a message from a
  newer publisher.
- **Any deploy order works.** Because each change in the lock is compatible in both directions,
  the containers can deploy in any order.

#### 5.5.3 Remote method ports

A remote method port is a synchronous request and response between containers. It uses the same
syntax as a port inside a container. The `remote` marker adds what the network makes true:

```go
// In Api (the consumer declares what it needs):
needs remote port Stock {
    func Level(sku SKU) (int, error)
        timeout "200ms"
}

// In a leaf of Api:
qty := Stock.Level(sku)?        // the caller must handle Unavailable and Timeout
```

- **Transport errors are part of the error set.** Each method of a remote port can also return
  `Unavailable`, `Timeout`, and `RemoteAbort`. For a method that returns a sealed error type, the
  compiler adds them as variants, so a type switch must handle them. For a method that returns plain
  `error`, they are values that `errors.Is` matches.
- **Each call has a timeout.** The project config sets a default timeout. A method can override it
  with `timeout`. If no timeout resolves, the code does not compile.
- **A timeout means the outcome is unknown.** The call can have run or not. The runtime never
  retries a remote call. If the caller retries, the method should be `idempotent`.
- **Contracts are checked on both sides.** The caller checks `requires` before it sends. The callee
  checks it again when it receives. The callee checks `ensures` before it replies, and the caller
  checks it again when it receives the reply.
- **Types must be serializable**, as for topic messages. The schema lock (§5.5.2) also covers the
  parameter and result types of remote methods, so the same evolution rules apply.
- **The wire names the transport** with `via`, as for topics. The built-in `direct` transport
  supports request and response:

  ```go
  Api/Orders.Stock: Worker/Inventory.NewLevels via direct{}
  ```

  The topic settings (`deadletters`, `retries`, `durable`, `whenFull`, `capacity`, `ttl`) apply
  only to topic wires. A call wire accepts none of them.

- **The callee runs each call in its own implicit scope** (§7.1). If the callee aborts, the caller
  receives the error `RemoteAbort`, which is part of every remote error set with `Unavailable` and
  `Timeout`.
- **The schema lock covers the contracts of remote ports.** A change to a remote port's contract is a
  breaking change, as for message invariants (§5.5.2).

**How to choose.** Use a remote method port when the caller needs an answer now. Use a topic when
the publisher must not depend on who reads, or when many subscribers react to one event. The
developer decides for each boundary.

### 5.6 Diagrams

`keel diagram --level context|container|component|code [--focus <node>]` draws each C4 level from
`system.keel` and the `component.keel` files. Composites hold no code. Thus a diagram at any zoom
level shows where all behavior is. A container or a group cannot hide logic.

## 6. Type system

- **Strong and static.** There are no implicit conversions.
- **Inferred where possible.** The compiler infers the types of local variables, closures, and
  generic instantiations. Port signatures and exported functions are explicit, because humans
  review them as contracts.
  The tools always show the full inferred signature: in `keel context`, in `keel doc`, on hover,
  and in diffs. Thus a model or a reviewer never has to find a type by hand.
- **Inference works on incomplete programs.** Thus the tools can suggest completions that have
  valid types (R5).
- **Sum types are sealed interfaces.** A `sealed interface` lists its variants, and only those
  types implement it. Code matches a sum with Go's own type switch, and the compiler checks that
  the switch handles each variant, and also `case nil` unless the value is proved non-`nil` (§7a).
  Typed errors, task exits, and state machines use this one form.

  ```go
  type PayError sealed interface { Declined | Expired }

  switch e := err.(type) {
  case Declined: ...
  case Expired:  ...
  }                          // a missing case is a compile error
  ```
- **Bounded fields.** A string, slice, or map field can declare a size limit: `Note string limit 4096`,
  `Lines []Line limit 100`. A value over the limit fails the invariant of its struct. A transport
  can require bounded message types (§5.5.1).
- **`nil` and zero values as in Go.** The compiler proves that each dereference, interface call,
  and map write cannot reach `nil` (§7a).
- Methods attach to any defined type that is not a pointer or an interface, as in Go (for example
  `type Celsius float64`). Interfaces and ports are structural, as in Go.

### 6.1 Generics

Generics use the syntax of Go, unchanged: `[T constraint]`. Go generic code from a model's training
data compiles as it is.

```go
func Max[T cmp.Ordered](xs []T) ?T
type Cache[K comparable, V any] struct { ... }
func (c *Cache[K, V]) Get[R any](k K, f func(V) R) ?R   // a generic method, as in Go 1.27
```

- Usually the compiler infers the type arguments, as in Go.
- A method on a concrete type can declare its own type parameters, as Go 1.27 permits. A method of
  an interface or a port cannot, also as in Go.

## 7. Errors

**Keel has no exceptions, no `try`, no `catch`, and no `throw`. No function can catch an abort.
Recovery occurs only at task boundaries (§7.1).**

An error is an ordinary value. It moves only through return values. Thus the source shows each path
that an error can take.

A function that can fail declares the Go shape, `(T, error)`, and uses Go's spelling. In Keel the
shape is a **sum type**: the function returns a value **or** an error, never both. Thus on the error
path the value of `T` is never usable.

```go
func Charge(c CustomerID, a Money) (Receipt, error) {
    tok, err := vault.Lookup(c)
    if err != nil {
        return Receipt{}, err  // failure: the compiler discards the value slot
    }
    ...
    return r, nil              // success
}

r, err := Charge(c, a)
if err != nil {
    return Receipt{}, err      // r is not usable here
}
use(r)                         // r is a Receipt here: the check above proved it
```

- `return v, nil` returns success. `return x, err` with a non-`nil` error returns failure, and the
  compiler discards `x`. The slot is type-checked as in Go, and a literal in it is exempt from
  invariant checks. The caller can never read it.
- `nil` is the value of `error` that means "no error", as in Go. It keeps all its Go meanings
  (§7a). A function that returns only `error` ends with `return nil`
  on success.
- A shape with more values works the same way: `(A, B, error)`.
- **`partial` keeps Go's shape where both results matter.** Some functions return a value and an
  error together, for example `io.Reader`, which can return `(512, io.EOF)` for the last chunk. Such
  a function declares `partial`, and its caller can read the value even when `err != nil`, exactly
  as in Go:

  ```go
  func Read(p []byte) (n int, err error) partial

  n, err := r.Read(buf)
  process(buf[:n])           // allowed: Read is partial
  if err == io.EOF { ... }
  ```

  Foreign libraries mark each Go function that works this way; the standard library wrappers ship
  marked (§5.4c).
- At the call site, `r, err := f()` binds both names. The compiler narrows `r` by control flow, the
  same as a `?T` (§7a): the code can use `r` only where the flow proves that `err == nil`.

The code **must handle each error explicitly**:

- **An unused `error` result is a compile error**, whether or not the code uses the other results.
  `defer f.Close()`, `json.NewEncoder(w).Encode(v)`, and `v, _ := strconv.Atoi(s)` all need either
  handling or `ignore err, "reason"`; the fix-it inserts the `ignore`. A short list of calls whose
  errors are conventionally ignored (for example `fmt.Println` and writes to `strings.Builder`) is
  exempt; `keel guide` prints the list. `Publish` returns nothing: subscribers cannot make a publish
  fail, and if the transport itself cannot accept the message, the publishing task aborts.
- Postfix `?` returns the error to the caller: `r := Checkout.Pay(c, amt)?`. If the call
  succeeds, `?` gives its value. If the call fails, `?` returns the error from the enclosing
  function, which must also return `error`. It does nothing more. It does not unwind, catch, or
  jump. Keel uses `?` and not a `try` keyword, so that humans and models do not read it as a
  try/catch block.
- To discard an error, the code must give a reason. The audit records the reason:
  `ignore err, "cache miss is not an error here"`.
- A function or port method can return a sealed error type in place of `error`:
  `func Pay(...) (Receipt, PayError)`. A type switch on it must be exhaustive.
- A lint, on by default, marks each catch-all handler that returns a default value. This pattern
  is error masking (R3).
- **An abort is not an exception.** A broken invariant stops the task with a diagnostic report.
  Examples: a contract violation, `unreachable`, an overflow, or an index out of range. No function
  can catch an abort, so code can never use an abort for control flow. See §7.1.

### 7.1 Supervision

Recovery occurs only at **task boundaries**. This is the "let it crash" model of Erlang, in Go
syntax. Keel adds **structured concurrency**: each task has an owner in the source, so each abort
has a destination that the compiler knows.

- **A task starts inside a `tasks { }` scope.** A `go` statement outside a scope is a compile
  error. The diagnostic offers a fix-it that wraps the smallest block containing the `go` and every later
  statement that uses the same channels or `WaitGroup`, so the scope never waits before the code
  that feeds its tasks. If no such block exists, it offers no automatic fix. It can also offer
  `detach` for a background task:

  ```
  go process(it)
    error K0801: go outside a tasks scope
      fix-it: tasks { go process(it) }?
      or:     detach process(it), "reason"
  ```

  At the end of the scope, the function waits for all the tasks that started in it. A task value
  cannot leave its scope.
- The runtime discards the state of an aborted task. Its exit goes to the scope that owns it.
- The exit of a task is a sum type, typed from the result of the function: `Completed` (with the
  value, if any), `Failed` (with the error, if the function returns one), or `Aborted` (with the
  report). Code in the scope can observe an exit with a `switch`:

  ```go
  tasks {
      t := go price(order)                   // go as an expression returns a task value
      switch x := t.Wait().(type) {          // Wait returns the sealed Exit[Price]
      case Completed[Price]:
          use(x.Value)
      case Failed:
          log.Error(x.Err)
      case Aborted:
          log.Report(x.Report)   // the diagnostic: which invariant, where, with what values
      }
  }
  ```

- **A failure cancels the scope, by default.** A scope can name a `context.Context`:
  `tasks ctx { go process(ctx, it) }`. When a task in the scope fails or aborts, the scope cancels
  `ctx`, so the other tasks can stop early (the semantics of Go's `errgroup.WithContext`). A task whose
  failure must not cancel the others handles its own error and returns `nil`. A remote method whose first
  parameter is a `context.Context` passes the caller's deadline and cancellation to the callee.
- **No error and no abort is silent.** At the end of the scope, the compiler and the runtime
  handle each exit that the code did not observe:
  - Each unobserved `Failed` exit becomes part of the error of the scope, joined as with Go's
    `errors.Join`, so no failure is lost. A `tasks { }` scope that can
    produce an error is an expression of type `error`, and the code must handle it like any error
    (§7), for example with `tasks { ... }?`.
  - An unobserved `Aborted` exit aborts the function that contains the scope, at the end of the
    scope. Its own owner then receives it. The rule is deterministic: it does not depend on the GC.

  ```go
  func handleBatch(items []Item) error {
      tasks {
          for _, it := range items {
              go process(it)            // process returns error
          }
      }?                                // waits here; returns all Failed errors, joined
      return nil
  }
  ```

- **To ignore an error on purpose**, write `ignore go f(), "reason"`. `keel audit` lists it.
- **A long-lived background task** uses `detach f(), "reason"` in place of a scope. The process
  supervisor owns it. An abort of a detached task stops the process with the report. `keel audit`
  lists each `detach`.
- **Where code starts.** The compiler generates `main` from `system.keel`; code never writes it. A
  leaf can declare one `startup func() error`. The generated `main` runs each startup function as
  a supervised task, for example to start the server of an entry leaf.
- **Implicit scopes contain aborts only.** Each startup function, each topic handler (§5.5.0), and each
  request that an entry leaf serves (§5.4b) run in their own scope. For a request, an abort ends that request
  with an error response and a report, and the process keeps running. Thus one bad request cannot
  stop a server. An implicit scope does **not** permit a bare `go`: that is still error K0801. Thus
  a `go` in a handler never makes the response wait without a visible `tasks` scope. Use `detach`
  or a topic for work that must outlive the request.
- The runtime records each abort report as an RCA trail: the task, the owning scope, the
  invariant, the source position, and the values. For a value that must not appear in
  records, the standard library type `secret.String` prints as `[secret]`.

**Aborts under a lock.** Locks behave exactly as in Go: `mu.Lock()` and `mu.Unlock()`, with no
error and no owner. An abort during an update can leave shared data half-changed. Thus, when the
compiler can see that a lock is held at an abort point (the `Lock` in the same function, or a
helper marked `holds`), it makes that abort **process-level**: the deferred calls run, as in Go,
and then the process stops with a report. The deployment restarts it. When the compiler cannot see
a held lock, the task aborts as usual.

- **Deferred calls run when a task aborts**, in reverse order, as in Go. Thus files and connections
  close. An abort inside a deferred call stops the remaining deferred calls, and the runtime adds it
  to the report.
- Limit: this protects only data that a lock guards. A change to shared data without a lock is a
  data race, and a data race is already a bug. The data-race tools below help find them.

**Data races.** Keel does not prevent all data races; neither does Go. Preventing all of them
needs an ownership system, which would be a large break from Go. Keel gives three tools:

- **`guarded by`** is an optional annotation on a struct field. The compiler checks that each
  access to the field holds the named `sync.Mutex` or `sync.RWMutex`. `RLock` permits reads;
  `Lock` permits reads and writes. A helper that its callers call with the lock held declares
  `holds mu`: `func (c *Cache) evictLocked() holds mu`. The compiler then checks the callers, not
  the helper.

  ```go
  type Cache struct {
      mu sync.Mutex
      m  map[string]int guarded by mu   // each access must hold mu
  }
  ```

- **One handler at a time by default.** The runtime runs the handlers of one subscribed topic one message
  at a time, so state that only one handler touches needs no lock (§5.5.0). A wire setting opts into
  concurrency.
- **The race detector** runs in test builds.

**Fault classes.** Not every fault can stay inside one task:

| Class | Faults | Result |
|---|---|---|
| Task-level | Contract violations, overflow, bounds, and panics in Go code inside a foreign library, when the task holds no lock | The task aborts. The parent receives `Aborted` (§7.1). On the Go backend, the runtime wraps each task in `recover` to do this. |
| Process-level | Any abort while the task holds a lock; stack exhaustion; out of memory; deadlock of all tasks | The process exits with a report. The deployment restarts it. No parent receives `Aborted`. |

## 7a. Null safety

Keel keeps Go's `nil` and Go's zero values, unchanged. A pointer, an interface, a map, a slice, a
channel, or a function can be `nil`, and `var p *Card` is `nil`, exactly as in Go. What Keel adds
is a **compile-time proof**: the compiler rejects each use that can reach `nil` and would panic in
Go. Thus a null-pointer failure cannot occur, and Go code keeps its types and its zero values.
(Uber's NilAway checks Go code in a similar way.)

**Uses that need a proof.** These panic in Go on `nil`, so the compiler must prove that the value
is not `nil` at that point:

- a dereference or a field access through a pointer: `p.Number`, `*p`
- a method call on an interface value
- a call of a function value
- a write to a map: `m[k] = v`
- a close of a channel

Reads from a `nil` map or slice, `len`, `range`, and `append` are safe in Go, so they need no proof.
A send on a `nil` channel blocks forever in Go, and code uses it on purpose to turn off a `select`
case, so it needs no proof either.

**What proves a value is not `nil`:**

- **a parameter or a receiver.** Each is non-`nil` at entry, and each call site must prove the
  arguments that it passes (the NilAway model). A parameter that accepts `nil` declares `?T`, for
  example `func Find(c ?*Card)`. Thus a method body can use its receiver and parameters directly.
- **a bound port name** (§5.3): the wiring sets it at startup.
- **a `nonnil` field of a Go struct**, marked by its foreign library (§5.4c).

- a check narrowed by control flow: `if p != nil { ... p.Number ... }`, or an early `return`
- an allocation: `&Card{}`, `new(Card)`, `make(map[K]V)`
- a result of a function marked `nonnil`, which the compiler checks in the body of that function:
  `nonnil func NewCard() *Card`

```go
var p *Card                // nil, as in Go
use(p.Number)              // compile error: p may be nil
if p != nil {
    use(p.Number)          // proved by the check
}
c := NewCard()             // NewCard is nonnil
use(c.Number)              // proved
```

- **Only a stable local narrows.** A check proves a local variable that no code assigns between
  the check and the use, and that no closure or task captures. A field does not narrow, because a
  call or another task can change it. The fix-it copies the value into a local first:

  ```go
  if s.user != nil {
      notify()
      send(s.user.Email)       // compile error: a field does not narrow
  }
  // fix-it:
  u := s.user
  if u != nil {
      notify()
      send(u.Email)            // proved
  }
  ```

- When the compiler cannot prove a use, the diagnostic shows these patterns, not only
  "may be nil".

**Optional values.** `?T` means "a `T` or absent", for every type `T`. For a type that cannot be
`nil` (`?int`), absent is written `nil`. For a type that can (`?*User`), `?T` is a separate wrapper,
so a stored `nil` pointer and "absent" stay different. The same proof rules apply before the value
is used. Thus generic code such as `?R` or `slices.At(xs, i)` works for every element type.

- An API that "might not find" a value of such a type returns `?T`. Examples: `Find`, a search,
  and a parse that has no error.

**Maps behave as in Go.** `m[k]` gives the zero value when the key is missing, and `v, ok := m[k]`
and `m[k]++` work. When the code wants absence to be explicit, `m.Get(k)` returns `?V` (or a
possibly-`nil` pointer, for a pointer value type):

```go
n := counts[word]              // 0 when the key is missing, as in Go
v, ok := counts[word]          // the Go comma-ok form
u := users.Get(id)             // ?User
if u == nil { ... }
```

**At the foreign boundary**, a Go value that can be `nil` stays a possibly-`nil` value, and the
same proof rules apply. A foreign library can mark results `nonnil` (§5.4c).

## 7b. Bounds

The compiler tries to prove that each index `xs[i]` is in range. A proof comes from one of these:

- the loop: `for i := range xs`
- an earlier check of both bounds, narrowed by control flow:
  `if 0 <= i && i < len(xs) { ... xs[i] ... }`
- a contract: `requires 0 <= i && i < len(xs)`

A proof needs both bounds, because `int` is signed. Narrowing follows the stable-local rule of §7a:
`i` and `xs` must be stable locals between the check and the use.

| Outcome | Result |
|---|---|
| The index is proved in range | No Keel runtime check. On the LLVM backend, the check is removed. On the Go backend, Go still emits its own check; the proof gives the compile-time guarantee only. |
| The index is proved out of range | Compile error. |
| The compiler cannot decide | **Warning.** The hint shows the three proof patterns above and `slices.At(xs, i)` from the standard library, which returns `?T`. The runtime checks the index, and `keel audit` lists it. |
| The index is out of range at runtime | Abort, as in Go. Only a supervising parent task observes it (§7.1). |

## 7c. Integer overflow

**Unsigned integers behave as in Go. Signed integers abort on overflow.**

Almost all deliberate wrapping in Go code (hashes, checksums, binary encoders, random number
generators) uses unsigned types. Signed overflow is almost always a bug; C makes it undefined
behavior for this reason. Thus Keel keeps Go's behavior where code relies on it, and adds safety
where an overflow is a bug.

| Operation | On overflow |
|---|---|
| `+ - *`, `<<`, and unary `-` on an unsigned type (`uint`, `uint8` to `uint64`, `byte`, `uintptr`) | Wraps, as in Go. |
| A conversion to an unsigned type (`byte(x)`, `uint32(x)`) | Truncates, as in Go. |
| `+ - * / <<` and unary `-` on a signed type (`int`, `int8` to `int64`), for example `math.MinInt64 / -1` | Abort. |
| A conversion to a signed type (`int32(x)`) | Truncates, as in Go. `checked.Convert[T]` detects a value that does not fit. |
| `checked.Add(a, b)`, `checked.Sub`, `checked.Mul`, `checked.Convert[T](x)` (standard library) | Returns `?T`, which is `nil` on overflow. Use these when the code handles the overflow itself. |
| Constant expressions | Compile error, as in Go. |

Division by zero aborts, as in Go.

**Where Keel accepts Go code and runs it differently.** Keel's rule is to reject Go code, not to
change its behavior. These are the deliberate exceptions, each listed in §4:

1. Signed arithmetic that overflows aborts; Go wraps. The compiler warns (K0701) where signed code
   looks like deliberate wrapping, but the warning cannot find every case.
2. An abort where the compiler can see a held lock stops the process, not only the task (§7.1).
   Thus a signed overflow under a visible lock stops the process.

```
h = h*31 + int(c)          // h is int
  warning K0701: signed arithmetic that looks like deliberate wrapping
    fix-it: use an unsigned type, or wrapping.Mul and wrapping.Add from the standard library
```

## 8. Contracts

A contract is a clause on a declaration. It is not a comment.

| Clause | Attaches to | Meaning |
|---|---|---|
| `requires expr` | functions, port methods | Precondition: the obligation of the caller |
| `ensures expr` | functions, port methods | Postcondition. For a function that can fail, it applies only when the function succeeds, and `result` is the success value. With more than one success value, `result.0`, `result.1`, and so on name them. |
| `invariant expr` | structs | Is true after construction and after the outermost exported method call returns (§8.2) |

### 8.1 What a contract can contain

Contract expressions are **pure**. They never do I/O, call ports, use channels or topics, change
data, or read the clock or a random source. Contract expressions also cannot fail:

- Arithmetic in a contract is exact. It cannot overflow.
- An index in a contract must be proved in range by the contract itself, for example
  `len(xs) > 0 && xs[0] > 0`. `requires xs[0] > 0` alone is a compile error.

Thus checking a contract never changes what a correct program does. Within that rule, Keel
permits two tiers:

| Tier | Adds | Example |
|---|---|---|
| 1 | Comparisons, arithmetic, `&&` `\|\|` `!`, `=>` (implies), field access, `len`, `result`, `old(x)` (the value on entry) | `ensures acct.Balance == old(acct.Balance) - amt` |
| 2 | Calls to functions marked `pure`. The compiler verifies that these functions have no effects, and their bodies follow the same rules as contract expressions: exact arithmetic and proved indexes. A `pure` function can still loop forever; the compiler does not prove termination. | `requires IsSorted(a) && IsSorted(b)` |

```go
pure func IsSorted[T cmp.Ordered](xs []T) bool { ... }

func Merge(a, b []int) []int
    requires IsSorted(a) && IsSorted(b)
    ensures  IsSorted(result) && len(result) == len(a) + len(b)
```

A rule over a collection uses a `pure` predicate from the standard library, for example
`requires slices.All(xs, isPositive)`. Its runtime cost is O(n), and `keel audit` lists each such
contract.

### 8.2 Struct invariants

An invariant protects a struct only if no code can go around it. Thus:

- **Only methods write fields.** The fields of a struct with an invariant are read-only outside
  its own methods. A write from anywhere else is a compile error.
- **Construction is checked.** A struct literal is checked where the code creates it. If the
  compiler can prove the violation, it is a compile error; if not, the check runs at runtime.
- **The zero value must satisfy the invariant.** Go creates zero values in many places (`var`,
  `new`, `make`, arrays, missing map keys, closed channels, generic `var zero T`), so Keel cannot
  block them all. Thus an invariant that the zero value breaks is a compile error on the invariant
  itself: write `invariant Lo <= Hi`, not `Lo < Hi`, or make the field optional.
- **Outer calls only.** The invariant is checked when the outermost exported method call on the
  value returns. A nested call on the same value, for example `Transfer` that calls `Withdraw`, is
  not checked on its return. Thus a method can pass through a temporarily broken state. This is
  the rule of Eiffel.

```go
type Account struct { Balance int }
    invariant Balance >= 0

a := Account{Balance: -5}   // checked here
a.Balance = -1              // compile error: write outside a method

func (a *Account) Transfer(to *Account, x int) {
    a.Withdraw(x)           // nested call: no check on its return
    to.Deposit(x)
}                           // checked here
```

How the compiler checks contracts:

- **Statically**, when the compiler can prove or disprove a contract.
- **At runtime** in every build, including release builds. Contracts cannot be turned off, so a
  proof that depends on a contract (§7b) always holds.
- **Port contracts bind each implementation**, including test fakes. If a fake violates the
  contract of its port, the test fails.

How reviewers audit contracts:

- `keel audit` lists each contract. For each contract, it shows if the compiler proved it and if
  tests exercised it.
- Contracts appear in the generated docs and diagrams.
- The audit report lists each `ignore` of an error and each contract that a leaf turned off.

## 9. Testing

- `test` blocks are next to the code that they test, in `*_test.keel` files in the leaf directory.
- The design makes each leaf testable in isolation: its only dependencies are the ports that it
  `needs` and the topics that it subscribes to.
- A fake is declared in a test file with one form: `fake CheckoutOK for Checkout { ... }`. Each
  fake, written or generated, must meet the contract of its port.
- In a test, port names are bound for each test separately. Tests that run in parallel run in
  separate processes, so they never share a fake.
- `keel fake <Port>` writes the skeleton of a fake, with one method for each method of the port.
  The developer fills in the bodies.
- A `test` block binds fakes to the required ports of the leaf with a `bind` clause:

  ```go
  test "pays for an order" {
      bind { Checkout: fake CheckoutOK }
      ...
  }
  ```
- `keel test Billing` builds and runs the tests of one leaf. It does not build the rest of the
  system.

## 10. Compiler intelligence

In addition to type checks, the compiler reports these problems as diagnostics with fix-its:

- **Near-duplicate code** in the project (R2). The fix depends on where the duplicates are:
  - Inside one leaf: the diagnostic points to the existing function to reuse.
  - In different leaves: a leaf cannot call another leaf's function (§5.4, rule 1). So the fix-it
    is **extract to library**. The compiler moves the function, and the private types that it
    needs, into a new or existing library, and updates both leaves to import it.
  - If the move is impossible, the diagnostic says why. For example, the function calls an
    `acquires` function (§5.4a, rule 2) or uses a port. The hint then suggests how to extract only
    the pure part.

  ```
  warning K0412: validateAddress in Orders and Shipping are 94% similar
    fix-it: extract to lib/address (moves validateAddress, type Address)

  warning K0412: syncStock in Api/Stock and Worker/Inventory are 91% similar
    cannot extract: syncStock calls acquires func sql.Open
    hint: extract the pure part (diffStock) to a library
  ```
- **Error masking** patterns (R3).
- **Architecture violations** (§5.4). The diagnostic gives the exact dependency edge that broke the
  rule.
- **Architecture diffs.** `keel archdiff main..HEAD` reports added and removed ports,
  dependencies, and contracts. Reviewers use this report for R6.
- Dead code and unused ports.

Each diagnostic has:

- a stable code
- a message for humans
- a *hint* that tells how to fix the problem in idiomatic Keel
- a machine-readable form (JSON)

## 11. Tooling that instructs the model

The tools assume that one of their users is an LLM (R8):

| Command | Purpose |
|---|---|
| `keel guide` | Prints a short language primer, small enough for a prompt, and the conventions of the project. The spec stays small, so that this primer can be complete. |
| `keel context <component>` | Emits the minimum context to work on one leaf: its `component.keel`, the wires in `system.keel` that connect it, and the method sets of the types on the other end of each wire (R1). |
| `keel query <symbol>` | Looks up a symbol exactly in the symbol index of the compiler. This prevents invented APIs (R4). |
| `keel explain <code>` | Gives a long explanation of a diagnostic, with examples. |
| `keel doc` / `keel diagram` | Generates docs, and C4 diagrams at each level (context, container, component, code), from the source (§5.6). |
| `keel audit` / `keel archdiff` | Audits contracts and architecture (§8, §10). |
| `keel fmt` | The one formatter. |
| `keel check` / `keel build` | The holistic boundary check, and the parallel build of each leaf (§12.2). |

Each command has `--json` output. An LSP server and an MCP server give the same functions to
editors and agents.

## 12. Runtime

- Native compilation and a tracing GC.
- **Backend: Go source first, LLVM later.** The compiler lowers the code to its own intermediate
  representation (IR). The backends read only this IR, never the syntax tree. Thus a second
  backend needs no change to the front end.
  - *Go backend (first).* Keel gets the GC, goroutine scheduler, channels, and cross-compilation of
    Go at no cost. The semantics of Keel map closely onto Go. The emitted code has `//line`
    directives. Thus the stack traces, profilers, and debuggers of Go report positions in `.keel`
    files, not in the generated Go.
  - *LLVM backend (deferred).* The LLVM backend needs a GC and a scheduler of Keel's own. That work
    takes months, and it blocks nothing in the language design, so it is deferred. The work starts
    when the IR is stable. "Stable" means that the self-hosted compiler (§12.1) builds on the IR
    with no change across releases.
- The concurrency model follows Go (tasks like goroutines, and channels), with the structured
  scopes of §7.1.

- **Build output is the developer's decision.** A system with one container compiles to one binary.
  For a system with more than one container, the project config chooses the output. The compiler
  supports both forms:
  - one binary for the system, with one mode for each container (`shop api`, `shop worker`)
  - one binary for each container (`shop-api`, `shop-worker`)

### 12.1 Compiler implementation

- **The compiler is written in Go, and later self-hosted in Keel.** The semantics of Keel are close
  to those of Go, so a tool can translate most of the compiler source automatically. Go took the
  same path from C at version 1.5. The self-hosted compiler becomes the first large Keel program,
  and it tests §5 in real use.
- **The compiler is query-based and incremental from the start.** It is a memoized query engine,
  not a batch pipeline. Example queries: "type of X", "ports of node Y", "context pack for Z". After
  each edit, the LSP and the tools in §11 calculate again only what changed. Go has no equivalent
  of the Rust library `salsa`, so the project builds this engine itself.

### 12.2 Compilation modes

Compilation has two separate modes. They run in parallel.

| | **Holistic check** (`keel check`) | **Per-leaf build** (`keel build`) |
|---|---|---|
| Purpose | Enforce boundaries. Find errors that need global context. | Compile code fast. |
| Input | All declaration files, all ports, and all signatures. Bodies only for global analyses. | One leaf directory, its `component.keel`, and the declarations of the libraries that it imports. |
| Finds | All of §5.4 (ports only, wiring, cycles, port compatibility), the topic and transport rules (§5.5 to §5.5.3), near-duplicate code in different leaves (§10), architecture diffs, and the project audit. | Type, error-handling, null, bounds, overflow, and contract errors inside the leaf. |
| Parallelism | One pass over the whole tree. | One job for each leaf. All the jobs run in parallel. |

- **Why a leaf can compile alone.** A leaf declares each port that it requires (§5.3). Thus it
  compiles against its own declarations, never against the code of another leaf. A change inside a
  leaf rebuilds only that leaf.
- **Both modes must pass before linking.** The compiler makes a binary only when the holistic check
  passes and all leaves compile. Neither mode accepts the results of the other without a check.
- **Both modes run on each build.** `keel build` starts the holistic check at the same time as the
  per-leaf jobs. The architecture part of the holistic check reads only declarations, so it finishes
  early. The near-duplicate part needs the fingerprints of every leaf, so it finishes after the
  slowest leaf job.
- **Caching.** The compiler caches the result of each leaf. The cache key is a hash of the
  contents of the leaf directory and the declarations that the leaf depends on. The query engine
  (§12.1) makes the holistic check incremental.

**Why this is feasible.** Go already compiles packages in parallel. Each package compiles against
the export data of its dependencies, which contains declarations only. The design of Keel removes
the usual obstacles:

- Consumer-owned ports (§5.3): a leaf compiles against its own declarations, not against the code
  of providers.
- No cross-leaf inference: every function declares its types (§6).
- Global analyses of bodies use output from the per-leaf jobs. Each job emits normalized
  fingerprints of its functions. The holistic check compares the fingerprints to find
  near-duplicates, and it does not read the bodies again.

On the Go backend, each leaf becomes one Go package, and the generated wiring becomes `main`. Thus
the Go toolchain adds its own parallelism. Shared types are in libraries (§5.4a), which compile
first. Then the leaves compile in parallel against the library declarations.

### 12.3 Performance

Keel must be competitive with Go, Java, C++, and Rust in speed and in resource use.

**How the project measures it.** A benchmark suite runs in CI. It covers CPU-bound work,
allocation-heavy work, concurrency, and a service with topics. Each benchmark has a target
relative to Go, Java, C++, and Rust. Once the numeric targets exist (open question Q22), a change that misses a
target fails CI.

**What each safety feature costs, and how Keel reduces the cost:**

| Feature | Runtime cost | Reduction |
|---|---|---|
| Overflow checks (§7c) | About one branch for each signed arithmetic operation; unsigned arithmetic has no check | Unsigned types or the `wrapping` package for deliberate wrapping; the LLVM backend can remove checks that a range proof covers |
| Bounds checks (§7b) | One compare for each unproved index | A proved index needs no Keel check; on LLVM, no check at all |
| Contracts (§8) | The cost of each expression, in release builds | The audit lists each contract with a collection-wide cost; the compiler removes checks that it proves |
| Garbage collection | GC pauses and memory overhead | On the Go backend, Go's GC; the LLVM backend needs Keel's own GC |
| Durable topics (§5.5.1) | A disk write for each message, on transports that keep queues on disk | The developer chooses the transport and its `durable` setting |
| Task scopes (§7.1) | A wait at the end of each scope | The same cost as a Go `sync.WaitGroup` |
| Foreign boundary (§5.4c) | A conversion for each value that crosses | Only at foreign libraries, not at every call |
| Port calls (§5.3) | An interface dispatch and the port's `requires` and `ensures` at each call | The compiler removes checks that it proves; the LLVM backend can make the call direct |
| Tasks (§7.1, §5.5.0) | A `recover` frame and an exit value for each task, and one task for each message | The same order of cost as a goroutine |
| Optionals (§7a) | `?T` of a non-pointer type holds a flag next to the value | None; pointers use their own `nil` |

**Backends.** On the Go backend, Keel performs like Go, plus the checks above. To close the gap
with C++ and Rust, the LLVM backend (§12) can use proofs to remove checks and can control memory
layout directly.

## 13. Non-goals (for now)

- Manual memory management and ownership types. The GC principle excludes them.
- Inheritance and class hierarchies. The "OO light" principle excludes them.
- Macros and metaprogramming. Each macro makes code harder to navigate and harder for a model to
  predict.
- A general system of effects or capabilities. It is deferred, because it
  conflicts with Go ergonomics. Examine it again when components and ports are proven.

## 14. Open questions

| # | Question |
|---|---|
| Q1 | The final name and file extension. |
| Q22 | Numeric performance targets for the benchmark suite (§12.3): for each benchmark, the accepted ratio of speed and memory against Go, Java, C++, and Rust. |

## Appendix A: Sources

- Sonar, *Your AI agent doesn't know your codebase's context or constraints*: https://www.sonarsource.com/blog/your-ai-agent-doesnt-know-your-codebases-context-or-constraints/
- Packmind, *Context engineering for large codebases* (cites Chroma 2025, 18 models): https://packmind.com/context-engineering-ai-coding/context-engineering-large-codebases/
- *Codified Context: Infrastructure for AI Agents in a Complex Codebase*: https://arxiv.org/html/2602.20478v1
- GitClear, *The Maintainability Gap* (2026): https://www.gitclear.com/the_ai_code_quality_maintainability_gap
- *Package Hallucinations* (USENIX ;login:): https://www.usenix.org/publications/loginonline/we-have-package-you-comprehensive-analysis-package-hallucinations-code
- *On Mitigating Code LLM Hallucinations with API Documentation*: https://arxiv.org/abs/2407.09726
- *Type-Constrained Code Generation with Language Models*: https://arxiv.org/pdf/2504.09246
- METR, developer productivity study and 2026 design update: https://metr.org/blog/2026-02-24-uplift-update/
- *"An Endless Stream of AI Slop"*: https://arxiv.org/pdf/2603.27249
- Sonar, *The AI trust gap*: https://www.sonarsource.com/blog/ai-coding-trust-gap/
- *No Resource, No Benchmarks, No Problem?* (no-resource languages): https://arxiv.org/html/2606.16827v1
- *A Survey on LLM-based Code Generation for Low-Resource and Domain-Specific Languages* (ACM TOSEM): https://dl.acm.org/doi/10.1145/3770084
