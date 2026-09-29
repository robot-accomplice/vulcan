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
| **Container** | A node that runs as one process. The compiler builds each container into one executable binary. Most systems have one container, so they compile to one binary. (The C4 meaning. It is not a Docker container.) |
| **Group** | An optional node inside a container that groups leaves, for zoom levels in diagrams. |
| **Composite** | A system, container, or group. A section of `system.keel`, with no code. |
| **Leaf** | A `component` node: one flat directory of code. |
| **Library** | A node that holds shared types and reusable code, with no ports, no topics, and no mutable state. It can use an I/O handle but never acquire one. |
| **Foreign library** | The only kind of library that can import a Go package. It converts values at the boundary. |
| **Gateway** | The one leaf in a container that talks to a given external system. |
| **External system** | Software outside the system, such as Stripe. Declared with `external`. |
| **Person** | A human who uses the system. Declared with `person`. For diagrams. |
| **Port** | An interface with contracts, declared once by the leaf that requires it. A provider satisfies it with an ordinary type, named at the wire. Inside a container, a port call is an ordinary function call. |
| **Topic** | A publish channel, `topic[T]`, for asynchronous events between containers. |
| **Remote port** | A port that crosses containers. A remote method port is a synchronous call; a remote topic is an event. |
| **Publisher / subscriber** | The leaf that publishes on an `out` topic / the leaf that handles an `in` topic. |
| **Handler** | The function, marked `handle`, that processes the messages of one `in` topic. |
| **Wire** | A line in `system.keel` that connects a required port or an `in` topic to a provider. |
| **Transport** | The mechanism that carries topic messages between containers, named on a wire with `via`. |
| **Outbox / inbox** | The runtime queue at the publisher / at the subscriber. |
| **Dead-letter topic** | Where a subscription sends a message that it cannot handle. |
| **Acknowledge** | Mark a message as handled. It occurs when its handler returns `nil`. |
| **Task** | A unit of concurrent work, started with `go` inside a `tasks { }` scope. |
| **Scope** | A `tasks { }` block. It owns the tasks that start in it and waits for them. |
| **Error** | An ordinary value that reports a failure the code expects and handles. |
| **Abort** | The stop of a task after a broken invariant: a bug, not an expected failure. It is not an exception. |
| **Contract** | A `requires`, `ensures`, or `invariant` clause. |
| **Layer** | A named level (`domain < app < infra`) that constrains source dependencies. |
| **Audit** | The `keel audit` report of each escape hatch, override, and unproved claim. |

## 2. Design principles

| Principle | Meaning in Keel |
|---|---|
| Compiled, GC | Native binaries and a tracing GC. No manual memory management and no borrow checker. |
| Go ergonomics | Go is the syntax model. Keel changes Go only where Go is weak, for example generics and error enforcement. |
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
| R2 | Duplication is up 81%. Reuse (cross-file calls) is down 35%. | The compiler finds near-duplicate code. Ports make the available capabilities easy to find. | §10 |
| R3 | Error-masking constructs are up 47%. | The code must handle each error explicitly. A silent discard is a compile error. | §7 |
| R4 | Models invent packages (about 20% of the time) and APIs. | The compiler serves an exact symbol index that tools can query. There are no implicit imports. | §11 |
| R5 | About 94% of compile errors in typed languages are type errors. | Strong static types, and inference that works on incomplete programs, so tools can guide generation by type. | §6 |
| R6 | AI code is harder to review. It "looks right but isn't". | Contracts in the code that reviewers can audit, and diffs at the architecture level. | §8, §10 |
| R7 | Developers feel faster than they measure. | Fast verification: incremental builds and per-leaf tests. | §9 |
| R8 | A new language has no training data, so LLM quality drops steeply. | Syntax close to Go, so the model can use what it knows. A spec small enough for a prompt. Tools that instruct the model. | §4, §11 |

## 4. Syntax

Go is the model. A Go programmer, or a model trained on Go, must be able to read Keel at first
sight. This spec must justify each change from Go and list it in this table:

| Change from Go | Reason |
|---|---|
| `system`, `container`, `group`, `component`, `library`, `port`, `person`, `external`, `layer` declarations | Architecture as a first-class construct (§5) |
| `(T, error)` keeps Go's spelling (`return v, nil`, `return x, err`) but is a sum type: a value or an error, never both. An unhandled error is a compile error. Postfix `?` propagates an error. | R3, and no zero value on the error path (§7) |
| `partial` functions return a value and an error together, as in Go | No data lost at Go boundaries such as `io.Reader` (§7) |
| Only `?T` and `error` can hold `nil` (the Swift model). Optional types `?T` are narrowed by control flow. | No null dereference can occur (§7a) |
| No exceptions of any kind. No `try`, `catch`, `throw`, or `recover()`. | Errors are values. Only a supervising task handles an abort (§7, §7.1) |
| Signed integer overflow aborts; unsigned arithmetic wraps as in Go. `+% -% *%` wrap signed values on purpose. Signed code that wraps on purpose behaves differently from Go (warning K0701). | Signed overflow is almost always a bug (§7c) |
| `guarded by` on struct fields, checked by the compiler | Fewer data races, with no ownership system (§7.1) |
| `go` runs only inside a `tasks { }` scope (structured concurrency); `detach` for background tasks. A task exit is `Completed`, `Failed`, or `Aborted`. | Each error and abort has a known destination (§7.1) |
| `requires`, `ensures`, and `invariant` clauses | Contracts in the code (§8) |
| `pure` functions | Contracts can call them. The compiler verifies that they have no effects (§8.1) |
| `test` and `fake` blocks | Testability by design (§9) |
| `topic[T]` publish channels between containers | A publish never waits for a subscriber and has zero or more subscribers. A `chan` is different (§5.5) |
| `remote port` (synchronous methods with `timeout` and transport errors, or topics), `idempotent`, `via` wiring | Honest communication between containers; the developer chooses a call or an event (§5.5, §5.5.3) |
| `topic.Publish(msg)` and `handle` functions; a message is acknowledged when its handler returns `nil` | No forgotten or early acknowledgement (§5.5.0) |
| `max` bounds on variable-size fields | A transport can require bounded messages, so the compiler can calculate queue memory (§5.5.1) |
| `gateway for` leaves; only gateways import network libraries | Enforced access to external systems (§5.4b) |
| `foreign library` nodes are the only importers of Go packages | Keel guarantees hold outside a guarded boundary (§5.4c) |
| Only leaves call `acquires` functions; libraries use handles they are given | Each I/O decision is visible in a leaf (§5.4a) |
| `const` extends to composite values and is deeply immutable; libraries have no package-level `var` | Enforceable “no mutable state” in libraries (§5.4a) |
| `ignore err, "reason"` and `x ?? fallback` | Explicit discards and defaults that the audit lists (§7, §7a) |
| Sum types with an exhaustive `switch` | Typed errors, task exits, and state machines (§6) |
| Declaration files (`system.keel` for the architecture, `component.keel` and `library.keel` for boundaries) have their own syntax: wires with paths (`Api/Orders.Checkout`), `uses A -> B`, `layers a < b`, `via`, `flag("...")`, transport settings (`via name{ ... }`), `budget`, `gateway for`, `client of`, `go "package"` | Architecture as a first-class construct. These files hold no Go-style code (§5) |
| Contract expressions add `result`, `old(x)`, `=>` (implies), `all x in xs: ...`, and `some x in xs: ...` | Contracts in the code (§8.1) |
| `unreachable` statement | Marks code that must never run; reaching it aborts (§7) |
| Methods on built-in types: `xs.At(i)`, `a.CheckedAdd(b)`, `CheckedSub`, `CheckedMul`, `CheckedConvert[T]` | Explicit handling of bounds and overflow (§7b, §7c) |
| A conversion to a signed type (`int32(x)`) aborts when the value does not fit; conversions to unsigned types truncate as in Go | Go truncates without a signal (§7c) |
| `m.Get(k)` returns `?V`. Other map reads behave as in Go. | Explicit absence when the code wants it (§7a) |
| A private function can omit its result types; the body alone determines them | Less noise, with no inference across call sites (§6) |
| Test files are `*_test.keel` in the leaf directory | Tests next to the code (§9) |

All other syntax follows Go, unless a later section says otherwise. This includes `:=`, `func`,
structs, methods with receivers, `for`, `switch`, slices, maps, `defer`, packages as directories,
and gofmt-style formatting.

### 4.1 Keywords

Each reserved word breaks Go code that uses it as a name, and models write that code (R8). Thus
Keel reserves few words.

- **Reserved everywhere:** `handle`, `pure`, `tasks`, `detach`, `ignore`, `unreachable`.
- **Contextual:** all other new words. They are keywords only where they start a declaration, or
  inside a declaration file or a contract. Elsewhere they are ordinary identifiers. This applies to
  `system`, `container`, `group`, `component`, `library`, `foreign`, `port`, `remote`, `layer`,
  `layers`, `requires`, `ensures`, `invariant`, `uses`, `via`, `in`, `out`, `topic`,
  `test`, `fake`, `result`, `old`, `all`, `some`, `max`, `guarded`, `acquires`, `idempotent`,
  `gateway`, `client`, `budget`, `partial`, `person`, `external`, `by`, `of`, `timeout`, `wire`, and
  `flag`.

```go
port := 8080                   // an identifier
func read(in io.Reader)        // an identifier

port Payments { ... }          // a keyword: it starts a declaration
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
  the leaves in each, the layers, the people, the external systems, and **every wire**.
- **Each leaf is a directory** with one `component.keel` and its source files. `component.keel`
  declares the **boundary** of the leaf: its layer, and the full declaration of each port that it
  requires.
- **Directories mirror the tree**: `api/orders/`, `worker/fulfilment/picking/`. The compiler checks
  that the directories and `system.keel` agree.
- Libraries are in `lib/`.

Thus each thing has one location: the wires in `system.keel`, a boundary in its `component.keel`,
and code next to its boundary.

### 5.3 Declarations

**A port is declared once, by the leaf that requires it.** This is the Go idiom: the consumer
defines the interface where it uses it. A leaf declares its boundary in `component.keel`: its
layer, and each port that it requires, with its methods and contracts.

```go
// api/orders/component.keel
component Orders {
    layer app

    // Checkout takes payment for a confirmed order.
    requires port Checkout {
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
reference, exactly as in Go. Because the wiring is fixed at compile time, the compiler can compile
each port call to a direct call. What Keel adds at the boundary is **enforcement and
instrumentation of the contract**: the port signature, its `requires` and `ensures`, and the
record of the boundary in diagrams and in the audit. Topics and transports (§5.5) exist only for
separate processes.

**The architecture file wires the leaves.** A wire connects a required port to a providing type.
Inside a container section, a wire names leaves by name. A wire between containers names each leaf
by its path: `Container/Leaf`, or `Container/Group/Leaf`.

```go
// system.keel
// Shop is the online store.
system Shop {
    layers domain < app < infra

    person   Customer "Buys goods through the web store"
    external Stripe   "Card payment processor"

    container Api {
        Billing, Orders, LedgerStore, SystemClock, StripeGateway

        Orders.Checkout: Billing.Payer
        Billing.Ledger:  LedgerStore.Store
        Billing.Clock:   SystemClock.Wall
    }

    container Worker {
        group Fulfilment { Picking, Shipping }
    }

    // Between containers: remote ports only (§5.5).
    Worker/Fulfilment/Picking.Intake.Placed: Api/Orders.OrderFlow.Placed via direct{ deadletters: Ops.Alerts }
    Api/Orders.OrderFlow.Shipped: Worker/Fulfilment/Picking.Intake.Shipped via direct{ deadletters: Ops.Alerts }

    uses Customer -> Api "places orders over HTTPS"
    // Api -> Stripe comes from the gateway declaration (§5.4b).
}
```

The wire names an exported type or an exported constructor with no parameters, for example
`Billing.NewPayer`. The provider needs no parameters, because it reaches its own dependencies
through its own required ports.

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
(§5.4b). The compiler draws that relationship in diagrams from the gateway declaration. A `uses`
line records a relationship that no declaration shows, such as a person who uses a container.

### 5.4 Rules the compiler enforces

1. **Ports only.** Code in a leaf can use another leaf **only** through a port that the leaf
   requires, or through an imported library (§5.4a). Any other reference to another leaf is a
   compile error.
2. **Code only in leaves and libraries.** `system.keel` holds no code. A leaf directory holds no
   child leaves.
3. **Container boundaries.** A wire between leaves in different containers must use a `remote`
   port (§5.5). A wire inside one container must not. A group is only a label for diagrams: it
   adds no boundary.
4. **Complete wiring.** `system.keel` wires each required port and each `in` topic of each leaf
   exactly once. An `out` topic is never wired by itself (§5.5).
5. **Satisfied ports.** Each wire names a providing type whose method set matches the port.
6. **Compatible wiring.** Each wire passes the port-compatibility check in §5.3.
7. **Boundary ownership.** A leaf declares each port that it requires, once, in its own
   `component.keel`. The code and the boundary of a leaf never name another leaf. Only
   `system.keel` names two leaves together, in a wire.
8. **Layers.** The `layers` order of the system constrains **source dependencies**: the libraries
   that a leaf imports, and the types in its port signatures. A leaf can use these only from its
   own layer or a lower layer. Layers do not constrain wires. Two wired leaves share no source
   code, because each owns its own ports, so a wire can connect leaves in any layers. Thus an `app`
   leaf can require a port that an `infra` adapter provides, and an `infra` HTTP handler can
   require a port that a `domain` leaf provides. But a port in an `app` leaf cannot mention an
   `infra` type such as `sql.Row`.
9. **No cycles.** The graph of wires between the leaves of one container has no cycles.
10. **Private types.** A type is private to its leaf. Only `system.keel` can name an exported type
    of a leaf, in a wire. A type in a port signature must come from a library (§5.4a).
11. **Leaf size budget.** The budget measures what a model reads to work on a leaf: all non-test
    source of the leaf plus its `keel context` output, in tokens. Each project sets one
    `context_budget` value in its config. A leaf over the budget is a compile error. The hint tells
    where to split the leaf. A leaf can raise its own limit with a reason:
    `budget 12k, "generated parser tables"`. `keel audit` lists each override. Thus each unit stays
    small enough for a model to hold completely.

### 5.4a Libraries

Some types cross a wire. For example, `Money` is in the port signatures on both sides. Such a type
must be the *same* type on both sides, so it cannot belong to either leaf. It goes in a
**library**. A library is a node that holds shared types and reusable code. It is not part of the
architecture at runtime.

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

1. A library has **no ports, no topics, and no mutable package-level state**. It is only code. It
   cannot hold state, so it cannot couple the leaves that import it. The compiler enforces this
   rule as follows:
   - A library cannot declare a package-level `var`.
   - A library can declare a package-level `const` of any type, including arrays, maps, structs,
     and slices. A `const` is **deeply immutable**: the compiler proves that no code can change it,
     or get a mutable reference to it or to any value inside it.
   - A `pure` function (§8.1) can compute a `const` once, at startup.

   ```go
   // lib/crc
   const table = buildTable()                       // pure function, runs once
   const names = map[Status]string{Paid: "paid"}

   var cache = map[string]int{}                     // compile error in a library
   names[Paid] = "x"                                // compile error: names is const
   ```
2. **A library enables I/O. It does not decide on I/O.** A library can read, write, query, or
   send through a handle that its caller gives it. A library cannot *acquire* a handle: it cannot
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
   have no cycles and obey `layers`.
4. A leaf can import a library in the same layer or a lower layer.
5. The leaf size budget (rule 11) also applies to libraries.
6. `keel diagram` shows libraries as shared dependencies. `keel audit` lists the leaves that use
   each library.

Libraries compile before the leaves that import them. Leaves compile against the library
declarations only (§12.2).

### 5.4b Gateways to external systems

A **gateway leaf** is the only leaf that talks to an external system. It turns the protocol of the
external system into an ordinary type that satisfies a port. Other leaves require that port, and
they contain no network code.

```go
// api/stripegw/component.keel
// StripeGateway charges cards through Stripe.
component StripeGateway {
    gateway for Stripe
}
// api/stripegw/cards.keel: type Cards, with a Charge method that calls Stripe

// api/billing/component.keel
component Billing {
    requires port CardNetwork { func Charge(...) (CardReceipt, error) }   // no network code
}

// system.keel
Billing.CardNetwork: StripeGateway.Cards
```

Rules:

1. A container that talks to an external system has exactly one gateway leaf for that system. The
   gateway declares `gateway for <External>`, and `<External>` must be an `external` declaration in
   the system.
2. Only a gateway leaf can import a network library (§5.4c), or a client library for an external
   system. Any other leaf that imports one causes a compile error. The compiler cannot read a URL
   string at compile time, so this import rule is the only way to enforce which leaves reach the
   network.
3. A client library declares the external system that it serves: `library StripeClient { client
   of Stripe }`. A gateway can import only the client libraries of its own external system.
4. The compiler generates the `uses` relationships in diagrams from the gateway declarations. Thus
   the diagram cannot be wrong.

Why a gateway (§1.1): each rule about an external system goes in one leaf. A reviewer reads all
traffic to that system in one location. Tests of other leaves fake a port with contracts, not HTTP.
The compiler rejects network code in any other leaf, so the reviewer does not have to find it.

### 5.4c Foreign libraries (Go interop)

Keel uses Go packages, but only through a guarded boundary. A **foreign library** is the only node
that can import a Go package. It gives a Keel API to the rest of the program.

```go
// lib/stripeclient/library.keel
// StripeClient wraps the Stripe Go SDK.
foreign library StripeClient {
    client of Stripe
    go "github.com/stripe/stripe-go/v82"
}
```

**The boundary converts each crossing value:**

| From Go | To Keel |
|---|---|
| a value that can be `nil` | `?T` |
| a Go `error` | a Keel error |
| a panic | an abort of the calling task (§7.1) |
| an integer | an integer; Keel arithmetic on it follows §7c |

**Rules:**

1. Only a foreign library can import a Go package. Leaves, gateways, and other libraries import the
   foreign library, never Go directly.
2. The compiler reads the Go dependency graph of each foreign library. If the graph includes a Go
   network package, the foreign library is a **network library**. Only a gateway (§5.4b), or a
   client library that a gateway imports, can import a network library. Thus the gateway rule is
   enforced through Go dependencies too.
3. The Keel side of a foreign library has no mutable package-level state, the same as any library
   (§5.4a).
4. A foreign library marks each function that acquires an I/O handle with `acquires`, for example
   `acquires func Open(path string) (File, error)`. Only leaves can call an acquiring function
   (§5.4a, rule 2). The standard library ships with these marks. `keel audit` lists each
   acquiring function, so that a reviewer can check the marks of third-party foreign libraries.
5. Go code inside the boundary can break Keel guarantees. It can start unsupervised goroutines, use
   locks outside Keel's rules, and hold global state. `keel audit` lists each foreign
   library and these limits, so that a reviewer can see where the guarantees stop.

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

- A send on a Go channel blocks until a subscriber takes the value. Each value has exactly one
  subscriber.
- A publish on a topic never waits for a subscriber. Each message has zero, one, or many
  subscribers.

One keyword with two behaviors would mislead humans and models trained on Go (R8). A Go `chan`
keeps its Go behavior for concurrency inside one process.

```go
// In Orders, inside Api. Each side declares its own view of the topics.
remote port OrderFlow {
    Placed  out topic[OrderPlaced]    // Orders publishes
    Shipped in  topic[ShipmentSent]   // Orders subscribes
}

// In Picking, inside Worker.
remote port Intake {
    Placed  in  topic[OrderPlaced]
    Shipped out topic[ShipmentSent]
}

// In system.keel. Each subscriber names its source; the wire names the transport and settings.
Worker/Fulfilment/Picking.Intake.Placed: Api/Orders.OrderFlow.Placed via direct{ deadletters: Ops.Alerts }
Analytics/Ingest.Feed.Placed:            Api/Orders.OrderFlow.Placed via direct{ deadletters: Ops.Alerts }
Api/Orders.OrderFlow.Shipped: Worker/Fulfilment/Picking.Intake.Shipped via direct{ deadletters: Ops.Alerts }
```

**Why publish: it is the loosest coupling possible.** The only job of the publisher is to publish.
The publisher does not know, and must not care, if a subscriber reads its messages. It does not know
how many subscribers there are, or when they read.

These rules are part of the language, for every transport:

- **Publish, never wait for a subscriber.** A publish gives the message to the transport and
  returns. Subscribers cannot make a publish fail. What happens when the transport is full is the
  policy of the transport.
- **Zero, one, or many subscribers.** `system.keel` wires topics one at a time, from the
  subscriber side. Each `in` topic is wired exactly once, to one `out` topic of the same message type. An
  `out` topic is never wired by itself: its subscribers are the `in` topics wired to it, zero or
  more. Thus a new subscriber adds one wire, and the wiring of the publisher does not change.
- **An unread topic causes a warning.** A topic with zero subscribers compiles, with a warning. A
  publisher can exist before its subscribers, but an unread topic must not go unnoticed.
- **A topic has no reply.** A reply would bind the publisher to a subscriber. For a request and
  response, use a remote method port (§5.5.3). If a publisher needs to know what happened later, it
  subscribes to a separate event that the other side publishes. `Shipped` in the example above is
  such an event.
- **A read model is an alternative to a synchronous read.** The owner publishes change events. The
  reader subscribes, keeps a local copy, and answers queries from that copy. This is
  "event-carried state transfer". Reads can be slightly stale, and each reader stores its own copy.

  ```go
  // Worker owns stock and publishes each change.
  StockChanged.Publish(StockChanged{SKU: s, Qty: q})

  // Api keeps a read model, and answers GET /stock/42 from it.
  idempotent handle StockChanged(m StockChanged) error {
      stock.Set(m.SKU, m.Qty)
      return nil
  }
  ```

- **No raw network calls between containers.** A leaf cannot call another container over the
  network by hand, because only gateways can import network libraries (§5.4b, §5.4c). `keel audit`
  lists each `external` system with its gateway, so a reviewer can find an `external` that is
  really a container of the same system.
- **Ownership.** Each side owns only its end of each topic. Only message values cross, as copies.
  The two sides share no memory and no references.
- **`remote` is necessary to cross.** The compiler rejects a wire across containers if the consumer
  did not declare the port `remote`.
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

**Publish** is a method on an `out` topic. It is not Go's `<-`, because a topic does not behave
like a `chan`:

```go
Placed.Publish(OrderPlaced{ID: id, Total: t})
```

**A handler** is a function bound to one `in` topic with `handle`:

```go
idempotent handle Shipped(m ShipmentSent) error {
    orders.MarkShipped(m.OrderID)?
    return nil          // the message is acknowledged here
}
```

- Each `in` topic of a leaf has exactly one handler. Code cannot iterate over an `in` topic.
- The runtime runs each message in its own supervised task (§7.1). By default, it runs the handler
  of one `in` topic for one message at a time. A wire setting, for example
  `via direct{ concurrency: 8 }`, allows more; shared state then needs a lock.
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

- **One queue for each subscriber.** A slow or failing subscriber fills only its own queue. It can
  never block the publisher for other subscribers.
- **Settings:** `capacity`, `whenFull` (`block`, `dropOldest`, `dropNewest`; default `block`),
  `ttl`, `deadletters` (a route; required), and `durable` (default `true`, which keeps queues on
  disk so that no message is lost across a restart). A value can come from a startup flag:
  `capacity: flag("orders-cap")`.
- **Delivery:** at least once, unordered. Thus `idempotent` is required on its handlers.
- **Bounded memory:** `direct` requires topic message types to be bounded with `max` fields (§6),
  so that the compiler can calculate the memory limit of each queue: capacity × maximum message
  size. `keel audit` shows each limit and the total.
- **Dead letters:** a message that fails all its retries, or expires, goes to the dead-letter
  route. A drop under `dropOldest` or `dropNewest` is counted, not dead-lettered.
- **Tracing** is off by default. The setting `trace: true` records each step of each message, and
  `keel trace <id>` shows them.

```go
Worker/Fulfilment/Picking.Intake.Placed: Api/Orders.OrderFlow.Placed via direct{
    capacity: 5_000, ttl: 24h, deadletters: Ops.Alerts }
Obs/Ingest.Feed.Samples: Api/Metrics.Telemetry.Samples via kafka{ topic: "samples" }
// startup: --transport.direct.peers=worker=10.0.0.7:7000
```

Other transports, for example Kafka, NATS, or SQS, are foreign libraries (§5.4c) that implement the
interface, each with its own settings and guarantees.

#### 5.5.2 Message evolution

Containers do not deploy at the same moment. Thus two builds of one system run together, and their
message types can differ. The holistic check (§12.2) sees only one build. So the compiler also
compares each message type with the version that is already deployed.

- **The compiler owns the wire format.** It serializes each message in a binary format. Each field
  has a number, and the wire carries the number, not the name. A field number is never used again.
- **A lock file records each message schema.** `topics.lock`, in the system directory, lists each
  topic message type with its field numbers, field types, and invariants. The compiler generates
  the file, and the project commits it.
- **`keel check` compares the current types with the lock.** A compatible change passes, and the
  compiler updates the lock. A breaking change is a compile error.

| Change | Result |
|---|---|
| Add a field of type `?T` | Compatible. A message from an older publisher has `nil` in the field. |
| Remove a field of type `?T` | Compatible. An older subscriber reads `nil`. |
| Rename a field | Compatible. The wire carries the number, not the name. |
| Relax an invariant | Compatible. |
| Add a required field (not `?T`) | Breaking. |
| Remove a required field | Breaking. |
| Change the type of a field | Breaking. |
| Tighten an invariant | Breaking. |

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
remote port Stock {
    func Level(sku SKU) (int, error)
        timeout 200ms
}

// In a leaf of Api:
qty := Stock.Level(sku)?        // the caller must handle Unavailable and Timeout
```

- **Transport errors are part of the error set.** Each method of a remote port can also return
  `Unavailable` and `Timeout`. A `switch` on its errors must handle them.
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
  supports request and response.

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
- **A private function can omit its result types, never its parameter types.** A private function
  is a function that is not exported and is not in a port. The compiler infers its result types
  from the **body only**, never from its call sites. Parameters always have declared types, so the
  inference depends only on them: a change to a caller, or a new unrelated type, can never change
  or break the inferred type. A recursive function must declare its result types.

  ```go
  func area(s Rect) { return s.W * s.H }   // result int is inferred
  func area(s) { ... }                      // compile error: a parameter needs a type
  ```

  The tools always show the full inferred signature: in `keel context`, in `keel doc`, on hover,
  and in diffs. Thus a model or a reviewer never has to find a type by hand.
- **Inference works on incomplete programs.** Thus the tools can suggest completions that have
  valid types (R5).
- **Sum types** (tagged unions), with an exhaustive `switch`. Typed errors and state machines need
  them. They are a deliberate addition to Go.
- **`nil` only in `?T` and `error`.** A plain type, including a pointer, an interface, a slice, or a
  map, can never be `nil` (§7a).
- Methods attach to structs, as in Go. Interfaces are structural, as in Go. Ports are nominal.

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
shape is a **sum type**: the function returns a value **or** an error, never both. Thus the error
path needs no usable value of `T`, and no zero value.

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
  compiler discards `x`, so any expression is accepted there, for example `Receipt{}` or `nil`.
  The caller can never read it.
- `nil` is the value of `error` that means "no error", as in Go. It is also the absent value of a
  `?T` (§7a), and it is nothing else. A function that returns only `error` ends with `return nil`
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

- If the code uses the value of a call and does not handle the error of the call, the code does
  not compile.
- Postfix `?` returns the error to the caller: `r := Checkout.Pay(c, amt)?`. If the call
  succeeds, `?` gives its value. If the call fails, `?` returns the error from the enclosing
  function, which must also return `error`. It does nothing more. It does not unwind, catch, or
  jump. Keel uses `?` and not a `try` keyword, so that humans and models do not read it as a
  try/catch block.
- To discard an error, the code must give a reason. The audit records the reason:
  `ignore err, "cache miss is not an error here"`.
- A port declares the kinds of error that each of its functions can return. A `switch` on these
  kinds must be exhaustive.
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
  error. The diagnostic has an automatic fix-it that wraps the code in a scope, or offers `detach`
  for a background task:

  ```
  go process(it)
    error K0801: go outside a tasks scope
      fix-it: tasks { go process(it) }?
      or:     detach process(it), "reason"
  ```

  At the end of the scope, the function waits for all the tasks that started in it. A handle
  cannot leave its scope.
- The runtime discards the state of an aborted task. Its exit goes to the scope that owns it.
- The exit of a task is a sum type, typed from the result of the function: `Completed` (with the
  value, if any), `Failed` (with the error, if the function returns one), or `Aborted` (with the
  report). Code in the scope can observe an exit with a `switch`:

  ```go
  tasks {
      t := go price(order)
      switch exit := <-t.Exit {
      case Completed:
          ...
      case Failed:
          log.Error(exit.Err)
      case Aborted:
          log.Report(exit.Report)   // the diagnostic: which invariant, where, with what values
      }
  }
  ```

- **No error and no abort is silent.** At the end of the scope, the compiler and the runtime
  handle each exit that the code did not observe:
  - An unobserved `Failed` exit becomes the error of the scope. A `tasks { }` scope that can
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
      }?                                // waits here; the first Failed error returns
      return nil
  }
  ```

- **To ignore an error on purpose**, write `ignore go f(), "reason"`. `keel audit` lists it.
- **A long-lived background task** uses `detach f(), "reason"` in place of a scope. The process
  supervisor owns it. An abort of a detached task stops the process with the report. `keel audit`
  lists each `detach`.
- **Implicit scopes contain aborts only.** The body of `main`, each topic handler (§5.5.0), and each
  request at an entry point (Q21) run in their own scope. For a request, an abort ends that request
  with an error response and a report, and the process keeps running. Thus one bad request cannot
  stop a server. An implicit scope does **not** permit a bare `go`: that is still error K0801. Thus
  a `go` in a handler never makes the response wait without a visible `tasks` scope. Use `detach`
  or a topic for work that must outlive the request.
- The runtime records each abort report as an RCA trail: the task, the owning scope, the
  invariant, the source position, and the values.

**Aborts under a lock.** Locks behave exactly as in Go: `mu.Lock()` and `mu.Unlock()`, with no
error. An abort during an update can leave shared data half-changed. Thus, if a task aborts while
it holds a lock, the abort is **process-level**: the process stops with a report, and the
deployment restarts it. No other task can see the half-changed data.

- **Deferred calls run when a task aborts**, in reverse order, as in Go. Thus files and
  connections close. An abort inside a deferred call stops the remaining deferred calls, and the
  runtime adds it to the report.
- Limit: this protects only data that a lock guards. A change to shared data without a lock is a
  data race, and a data race is already a bug. The sharing rules below and the race detector in
  test builds find data races.

**Data races.** Keel does not prevent all data races; neither does Go. Preventing all of them
needs an ownership system, which would be a large break from Go. Keel gives three tools:

- **`guarded by`** is an optional annotation on a struct field. The compiler checks that each
  access to the field holds the named mutex.

  ```go
  type Cache struct {
      mu sync.Mutex
      m  map[string]int guarded by mu   // each access must hold mu
  }
  ```

- **One handler at a time by default.** The runtime runs the handlers of one `in` topic one message
  at a time, so state that only one handler touches needs no lock (§5.5.0). A wire setting opts into
  concurrency.
- **The race detector** runs in test builds.

**Fault classes.** Not every fault can stay inside one task:

| Class | Faults | Result |
|---|---|---|
| Task-level | Contract violations, overflow, bounds, and panics in Go code inside a foreign library, when the task holds no lock | The task aborts. The parent receives `Aborted` (§7.1). On the Go backend, the runtime wraps each task in `recover` to do this. |
| Process-level | Any abort while the task holds a lock; stack exhaustion; out of memory; deadlock of all tasks | The process exits with a report. The deployment restarts it. No parent receives `Aborted`. |

## 7a. Null safety

Keel keeps Go's word `nil`, but only two types can hold it: an optional `?T` and `error`. This is
the model of Swift. A plain `T`, including a pointer, an interface, a slice, or a map, can never be
`nil`. No expression can dereference nothing, so a null-pointer failure cannot occur.

- Each type has a zero value, as in Go, **or** the code must initialize it before use. The code
  must initialize pointers, interfaces, and function values.
- Absence is a type, not a value. `?T` means "a `T` or `nil`".
- The code cannot use a `?T` as a `T`. The compiler narrows the type by control flow:

  ```go
  u := users.Find(id)        // u is ?User
  if u == nil {
      return NotFound
  }
  send(u.Email)              // u is User here: the check above proved it
  ```

- **Only a stable local narrows.** Narrowing applies to a local variable that no code assigns
  between the check and the use, and that no closure or task captures. A field, a pointer target,
  or shared data does not narrow, because a call or another task can change it. The fix-it copies
  the value into a local first:

  ```go
  if s.user != nil {
      notify()
      send(s.user.Email)       // compile error: a field does not narrow
  }
  // fix-it:
  u := s.user
  if u != nil {
      notify()
      send(u.Email)            // u is User here
  }
  ```

- `x ?? fallback` gives an explicit default. Keel permits it, because the code states the default
  where it is used. But `keel audit` lists each `??`, the same as `ignore`.
- An API that "might not find" a value returns `?T`. Examples: `Find`, a search, and a parse that
  has no error.
- **Maps behave as in Go.** `m[k]` gives the zero value when the key is missing, and
  `v, ok := m[k]` and `m[k]++` work. When the code wants absence to be explicit, `m.Get(k)` returns
  `?V`:

  ```go
  n := counts[word]              // 0 when the key is missing, as in Go
  v, ok := counts[word]          // the Go comma-ok form
  u := users.Get(id)             // ?User
  if u == nil { ... }
  ```

  For a value type with no zero value (Q20), `m[k]` is a compile error, and the code uses
  `v, ok := m[k]` with narrowing, or `m.Get(k)`.
- A value from Go that can be `nil` has the type `?T` at the foreign-library boundary (§5.4c).
- When code uses a `?T` incorrectly, the diagnostic hint shows the narrowing pattern above. It does
  not only say "type mismatch".

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
| The compiler cannot decide | **Warning.** The hint shows the three proof patterns above and `xs.At(i)`, which returns `?T`. The runtime checks the index, and `keel audit` lists it. |
| The index is out of range at runtime | Abort, as in Go. Only a supervising parent task observes it (§7.1). |

## 7c. Integer overflow

**Unsigned integers behave as in Go. Signed integers abort on overflow.**

Almost all deliberate wrapping in Go code (hashes, checksums, binary encoders, random number
generators) uses unsigned types. Signed overflow is almost always a bug; C makes it undefined
behavior for this reason. Thus Keel keeps Go's behavior where code relies on it, and adds safety
where an overflow is a bug.

| Operation | On overflow |
|---|---|
| `+ - *` and unary `-` on an unsigned type (`uint8` to `uint64`, `byte`, `uintptr`) | Wraps, as in Go. |
| A conversion to an unsigned type (`byte(x)`, `uint32(x)`) | Truncates, as in Go. |
| `+ - *` and unary `-` on a signed type (`int`, `int8` to `int64`) | Abort. |
| A conversion to a signed type (`int32(x)`) that does not fit | Abort. |
| `+% -% *%` on a signed type | Wraps (two's complement), for deliberate signed wrapping. |
| `a.CheckedAdd(b)`, `CheckedSub`, `CheckedMul`, `CheckedConvert[T]` | Returns `?T`, which is `nil` on overflow. Use these when the code handles the overflow itself. |
| Constant expressions | Compile error, as in Go. |

Division by zero aborts, as in Go.

**One case still differs from Go:** signed arithmetic that wraps on purpose, for example a hash
that uses `int`. The compiler warns where signed code looks like deliberate wrapping, and offers the
wrapping operators. The warning cannot find every case, so this difference is listed in the
changes from Go (§4).

```
h = h*31 + int(c)          // h is int
  warning K0701: signed arithmetic that looks like deliberate wrapping
    fix-it: h = h *% 31 +% int(c), or use an unsigned type
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

Thus a correct program does the same thing when its checks are on and when they are off. Within
that rule, Keel permits three tiers:

| Tier | Adds | Example |
|---|---|---|
| 1 | Comparisons, arithmetic, `&&` `\|\|` `!`, `=>` (implies), field access, `len`, `result`, `old(x)` (the value on entry) | `ensures acct.Balance == old(acct.Balance) - amt` |
| 2 | Calls to functions marked `pure`. The compiler verifies that these functions have no effects. | `requires IsSorted(a) && IsSorted(b)` |
| 3 | Quantifiers over collections: `all x in xs: …` and `some x in xs: …` | `requires all x in xs: !IsNaN(x)` |

```go
pure func IsSorted[T cmp.Ordered](xs []T) bool { ... }

func Merge(a, b []int) []int
    requires IsSorted(a) && IsSorted(b)
    ensures  IsSorted(result) && len(result) == len(a) + len(b)
```

A runtime check of a quantifier costs O(n). `keel audit` lists each quantifier that release builds
still check. A leaf can turn these checks off.

### 8.2 Struct invariants

An invariant protects a struct only if no code can go around it. Thus:

- **Only methods write fields.** The fields of a struct with an invariant are read-only outside
  its own methods. A write from anywhere else is a compile error.
- **Construction is checked.** A struct literal is checked where the code creates it. If the
  compiler can prove the violation, it is a compile error; if not, the check runs at runtime.
- **The zero value must satisfy the invariant.** If it does not, the type has no zero value
  (see Q20).
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
- **At runtime** in debug builds and test builds. Release builds keep the checks by default. A leaf
  can turn them off, with one exception: a **load-bearing** contract stays on. A contract is
  load-bearing when a proof depends on it, for example a bounds proof (§7b) or a narrowing.
  `keel audit` lists each load-bearing contract.
- **Port contracts bind each implementation**, including test fakes. If a fake violates the
  contract of its port, the test fails.

How reviewers audit contracts:

- `keel audit` lists each contract. For each contract, it shows if the compiler proved it and if
  tests exercised it.
- Contracts appear in the generated docs and diagrams.
- The audit report lists each `ignore` of an error and each contract that a leaf turned off.

## 9. Testing

- `test` blocks are next to the code that they test, in `*_test.keel` files in the leaf directory.
- The design makes each leaf testable in isolation: its only dependencies are the ports in its
  `requires` list.
- A `fake` block implements a port for tests. Each fake, written or generated, must meet the
  contract of its port.
- The compiler generates a default fake only when it can build values that meet each `ensures`:
  a port with no `ensures`, or with simple equalities that it can solve, such as
  `result.Amount == amount`. Otherwise it reports which clause prevents it, and the developer
  writes the fake.

  ```
  K0520: cannot generate a fake for Sorter.Merge
    clause: ensures IsSorted(result)
    hint: write fake Sorter { ... }
  ```

- A `test` block binds fakes to the required ports of the leaf with a `wire` clause:

  ```go
  test "pays for an order" {
      wire { Checkout: fake CheckoutOK }
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

  warning K0412: syncStock in Api and Worker are 91% similar
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
| `keel context <component>` | Emits the minimum context to work on one leaf: its `component.keel`, the wires in `system.keel` that connect it, and the signatures of the ports on the other end of each wire (R1). |
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
- The concurrency model follows Go (tasks like goroutines, and channels), unless a later decision
  changes it.

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
| Input | All declaration files, all ports, and all signatures. Bodies only for global analyses. | One leaf directory and the port declarations that the leaf owns. |
| Finds | All of §5.4 (ports only, wiring, layers, cycles, port compatibility), the topic and transport rules (§5.5 to §5.5.3), near-duplicate code in different leaves (§10), architecture diffs, and the project audit. | Type, error-handling, null, bounds, overflow, and contract errors inside the leaf. |
| Parallelism | One pass over the whole tree. | One job for each leaf. All the jobs run in parallel. |

- **Why a leaf can compile alone.** A leaf declares each port that it requires (§5.3). Thus it
  compiles against its own declarations, never against the code of another leaf. A change inside a
  leaf rebuilds only that leaf.
- **Both modes must pass before linking.** The compiler makes a binary only when the holistic check
  passes and all leaves compile. Neither mode accepts the results of the other without a check.
- **Both modes run on each build.** `keel build` starts the holistic check at the same time as the
  per-leaf jobs. The holistic check reads mostly declarations. Thus it is fast, and it is almost
  never the slowest job.
- **Caching.** The compiler caches the result of each leaf. The cache key is a hash of the
  contents of the leaf directory and the declarations that the leaf depends on. The query engine
  (§12.1) makes the holistic check incremental.

**Why this is feasible.** Go already compiles packages in parallel. Each package compiles against
the export data of its dependencies, which contains declarations only. The design of Keel removes
the usual obstacles:

- Consumer-owned ports (§5.3): a leaf compiles against its own declarations, not against the code
  of providers.
- Inference from the body only (§6): the compiler never infers a type from a call site in another
  leaf.
- The leaf size budget (§5.4, rule 11) limits the largest job, and thus the critical path.
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
relative to Go, Java, C++, and Rust. A change that misses a target fails CI. The numeric targets
are open question Q22.

**What each safety feature costs, and how Keel reduces the cost:**

| Feature | Runtime cost | Reduction |
|---|---|---|
| Overflow checks (§7c) | About one branch for each signed arithmetic operation; unsigned arithmetic has no check | `+%` for deliberate wrapping; the LLVM backend can remove checks that a range proof covers |
| Bounds checks (§7b) | One compare for each unproved index | A proved index needs no Keel check; on LLVM, no check at all |
| Contracts (§8) | The cost of each expression, in release builds | A leaf can turn off contracts that are not load-bearing; quantifier cost is listed in the audit |
| Garbage collection | GC pauses and memory overhead | On the Go backend, Go's GC; the LLVM backend needs Keel's own GC |
| Durable topics (§5.5.1) | A disk write for each message, on transports that keep queues on disk | The developer chooses the transport and its `durable` setting |
| Task scopes (§7.1) | A wait at the end of each scope | The same cost as a Go `sync.WaitGroup` |
| Foreign boundary (§5.4c) | A conversion for each value that crosses | Only at foreign libraries, not at every call |

**Backends.** On the Go backend, Keel performs like Go, plus the checks above. To close the gap
with C++ and Rust, the LLVM backend (§12) can use proofs to remove checks and can control memory
layout directly.

## 13. Non-goals (for now)

- Manual memory management and ownership types. The GC principle excludes them.
- Inheritance and class hierarchies. The "OO light" principle excludes them.
- Macros and metaprogramming. Each macro makes code harder to navigate and harder for a model to
  predict.
- A system of effects or capabilities (architecture option C). It is deferred, because it
  conflicts with Go ergonomics. Examine it again when components and ports are proven.

## 14. Open questions

| # | Question |
|---|---|
| Q1 | The final name and file extension. |
| Q20 | Because plain types cannot be `nil`, some types have no zero value (types that hold pointers, interfaces, or functions). What happens to `make([]*T, n)`, a generic `var x T`, and a partial struct literal? |
| Q21 | Inbound traffic. A leaf that serves HTTP to a `person` (for example, Api serving Customer) needs a network library, but it is not a gateway to an external system. How does Keel declare and enforce an entry point? |
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
