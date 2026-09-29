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
| **Library** | A node that holds shared types and reusable code, with no ports and no topics. Its package-level variables cannot be reassigned after startup. It can use an I/O resource but never acquire one. |
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
| **Transport** | A node kind that carries topic messages and remote calls between containers, named on a wire with `via`. |
| **Dead-letter topic** | Where a subscription sends a message that it cannot handle. |
| **Acknowledge** | Mark a message as handled. It occurs when its handler returns `nil`. |
| **Error** | An ordinary value that reports a failure the code expects and handles. |
| **Contract** | A `requires`, `ensures`, or `invariant` clause. |
| **Audit** | The `keel audit` report of each escape hatch, override, and unproved claim. |

## 2. Design principles

| Principle | Meaning in Keel |
|---|---|
| Compiled, GC | Native binaries and a tracing GC. No manual memory management and no borrow checker. |
| Go ergonomics | Function bodies are Go, unchanged. Keel adds the architecture, contracts, and tools around them (§4, §6). |
| Competitive performance | Resource use and speed must be competitive with Go, Java, C++, and Rust. §12.3 states the targets and the costs. |
| Compiler first | If the compiler can check a rule, the compiler checks it. Each rule that depends on convention, review, or runtime is a candidate to move into the compiler. This is the most direct way to make tasks 2 and 3 of §1.1 shorter. |
| Rust-grade compiler intelligence | Diagnostics explain *why* and propose fixes. The tools find duplication, error masking, and common Go faults, as warnings. |
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
| R3 | Error-masking constructs are up 47%. | The tools warn on each unhandled or masked error, with a fix-it, and the audit lists them. | §6, §10 |
| R4 | Models invent packages (about 20% of the time) and APIs. | The compiler serves an exact symbol index that tools can query. There are no implicit imports. | §11 |
| R5 | About 94% of compile errors in typed languages are type errors. | Go's static types. The compiler's type information works on incomplete programs, so tools can guide generation by type. | §6, §11 |
| R6 | AI code is harder to review. It "looks right but isn't". | Contracts in the code that reviewers can audit, and diffs at the architecture level. | §8, §10 |
| R7 | Developers feel faster than they measure. | Fast verification: incremental builds and per-leaf tests. | §9 |
| R8 | A new language has no training data, so LLM quality drops steeply. | Syntax close to Go, so the model can use what it knows. A primer (`keel guide`) with a hard budget of 4,000 tokens; the full spec is reference. Tools that instruct the model. | §4, §11 |

## 4. Syntax

**Function bodies are Go, unchanged (§6).** A Go programmer, or a model trained on Go, reads and
writes Keel code as Go. Keel's changes are all around the code: the architecture, contracts, and
tests. This spec must justify each change and list it in this table:

| Addition to Go | Reason |
|---|---|
| Declaration files: `system.keel` (containers, groups, `person`, `external`, `uses`, every wire, `via` with transport settings, `config { }`), `component.keel` (`needs port`, `needs remote port` with `timeout`, `publishes`, `subscribes`, `gateway for`, `entry for`), `library.keel`, `transport.keel`. They can use `flag("...")` and constant `time.Duration` expressions | Architecture as a first-class construct (§5) |
| A leaf never imports another leaf; it uses a port, a topic, or a library. A port call is an ordinary function call | Enforced, legible boundaries (§5.3, §5.4) |
| Library rules: no ports or topics, package-level `var` set once, no resource-acquiring calls | Libraries do not hide state or I/O decisions (§5.4a) |
| Only gateway and entry leaves import network packages | Enforced network access (§5.4b, §5.4c) |
| `topic[T]`, `Publish`, and `idempotent handle` functions | Events between containers (§5.5) |
| `limit` bounds on variable-size fields | A transport can require bounded messages (§5.5.1) |
| `requires`, `ensures`, `invariant`, `pure`, and `result`, `old(x)`, `=>` inside contracts; only its own methods write the fields of a struct with an invariant | Contracts in the code (§8) |
| `startup` functions; `main` is generated | The architecture decides how a process starts (§6) |
| `test` blocks, `fake X for Port`, `bind`, `*_test.keel` files | Testability by design (§9) |

All other syntax and behavior follow Go.

### 4.1 Keywords

Keel reserves no word that Go does not. Each new word is a keyword only where the grammar expects
it: at the start of a declaration (`needs port`, `pure func`, `handle X(`, `test "..."`), after a
field type (`limit`), after a signature or a struct (`requires`, `ensures`, `invariant`,
`timeout`), inside a contract (`result`, `old`), inside a declaration file, and in a `bind` clause
(`fake`). Everywhere else it can be an identifier.

The new words: `system`, `container`, `group`, `component`, `library`, `transport`, `port`,
`remote`, `needs`, `requires`, `ensures`, `invariant`, `pure`, `result`, `old`, `uses`, `via`,
`topic`, `publishes`, `subscribes`, `handle`, `idempotent`, `test`, `fake`, `bind`, `limit`,
`gateway`, `entry`, `person`, `external`, `timeout`, `config`, `startup`, and `flag`.

```go
port := 8080                   // an identifier
needs port Payments { ... }    // keywords: they start a declaration
```

`keel guide` prints the list.

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
- Libraries are in `lib/`. Transports are in `transports/`, one directory each, with a
  `transport.keel` declaration.

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
        Web, Billing, Orders, LedgerMemory, SystemClock, StripeGateway, Metrics

        Orders.Checkout: Billing.NewPayer
        Billing.Ledger:  LedgerMemory.Store
        Web.Checkout:    Billing.NewPayer
        Billing.CardNetwork: StripeGateway.NewCards{ APIKey: flag("stripe-key") }
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
  startup flags, as for transports: `StripeGateway.NewCards{ APIKey: flag("stripe-key") }`, which fills the exported settings struct
  of `NewCards`.
- **A constructor can fail.** It can return `(T, error)`. A failure stops the process at startup
  with a report.
- **One instance for each constructor and settings value, in each process.** Two wires to the
  same constructor with the same settings share one instance.

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
   exactly once. A published topic is never wired by itself (§5.5).
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

1. A library has **no ports and no topics**. A package-level `var` in a library is set once, at
   startup, and never assigned again; the compiler rejects each later assignment. The compiler does
   not track changes through a method or a reference (`hits.Add(1)`, `cache[k] = v`), so a library
   can still hold state that two leaves share. `keel audit` lists each library `var` whose value can
   change through a method or a reference, so that a reviewer can check it.

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
Billing.CardNetwork: StripeGateway.NewCards{ APIKey: flag("stripe-key") }
```

Rules:

1. A container that talks to an external system has exactly one gateway leaf for that system. The
   gateway declares `gateway for <External>`, and `<External>` must be an `external` declaration in
   the system.
2. Network access follows the single rule in §5.4c, rule 2. A gateway can use any Go package,
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

- An entry leaf declares `entry for <Person or External>`, and it can import network libraries to
  serve requests, under the one network rule (§5.4c, rule 2).
- The compiler cannot tell serving from dialing, so it does not try. `keel audit` lists each entry
  leaf and each gateway, and the network libraries that each imports.
- The compiler draws the `uses` relationship for each entry leaf, as for gateways.

Why a gateway (§1.1): each rule about an external system goes in one leaf. A reviewer reads all
traffic to that system in one location. Tests of other leaves fake a port with contracts, not HTTP.
The compiler rejects network access in any other leaf (§5.4c, rule 2), so the reviewer does not have to find it.

### 5.4c Go packages

Keel code is Go code, so a leaf or a library imports Go packages directly, as Go does. Two
architecture rules apply to these imports:

1. **Network access is decided per package.** A Go package whose dependencies include `net` or
   `os/exec` is a **network package**. **Only gateway leaves and entry leaves can import a network
   package.** This is the one rule for network access; other sections refer to it. It covers every
   path to the network in the imported code, including `http.DefaultClient`, and a database driver
   that dials a server. An embedded database driver (for example SQLite) does not import `net`, so
   the project chooses whether to model it as an `external` system. Limit: code that reaches the
   network through `syscall` directly is not detected; `keel audit` lists each import of
   `syscall`.
2. **Acquiring a resource is a leaf's decision.** Keel ships a list of the Go functions that acquire
   an I/O resource, for example `os.Open`, `os.Create`, and `sql.Open`. The project can add to the
   list in the `config { }` section of `system.keel`. Only leaves can call a listed function; a
   library cannot (§5.4a, rule 2). `keel audit` lists each call.

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

- An unbuffered send on a Go channel waits for a receiver. Each value goes to exactly one
  receiver.
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
  subscriber side. Each subscribed topic is wired exactly once, to one published topic of the same message type. A
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
  panic, and a received message that fails its invariant all go back to the transport, which
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
- The runtime runs each message in its own goroutine and recovers a panic in it (§6). By default, it runs the handler
  of one subscribed topic for one message at a time. A wire setting, for example
  `via direct{ concurrency: 8, deadletters: Worker/Alerts.DeadPlaced }`, allows more; shared state then needs a lock. At most
  `concurrency` handlers run at once for one subscribed topic; other messages wait in the
  transport's queue, which its capacity bounds. With concurrency
  above 1, the runtime keeps the order that the transport guarantees: if the transport keeps order
  per key, the runtime runs at most one handler at a time for each key. Thus the order at the wire
  is also the order at the handler.
- **The acknowledgement point is the return of the handler.** If the handler returns `nil`, the
  message is acknowledged. If it returns an error or panics, the message is not acknowledged, and
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

**The developer chooses.** Each wire names its transport and its settings. The `config { }` section of `system.keel` can
set a default transport and default settings. Addresses and credentials never appear in source;
they come from startup flags. `keel audit` lists the transport and
the settings of each wire.

**The built-in transport, `direct`.** Keel ships one transport, so that a small system with two
containers needs no broker. It sends from the publisher straight to each subscriber.

- **One queue for each subscriber.** A slow or failing subscriber fills only its own queue.
- **Settings:** `concurrency` (default 1), `capacity` (default 10,000), `whenFull` (`block`, `dropOldest`, `dropNewest`; default `block`),
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
- **A dead-letter route is a subscribed topic** of type `DeadLetter[T]` in some leaf. If its own
  handler fails, the message is retried with the source wire's settings, then counted and reported;
  there is no dead letter of a dead letter. The wire
  settings name it, and it can receive from any number of wires whose message type is `T`.
  Each dead letter carries the message, the last handler error or panic report, the attempt count,
  the wire, and the time. Thus "why did this message dead-letter?" has an answer without
  tracing.
- **Tracing** is off by default. The setting `trace: true` records each step of each message, and
  `keel trace <id>` shows them.

```go
Worker/Picking.Placed: Api/Orders.Placed via direct{
    capacity: 5_000, ttl: 24 * time.Hour, deadletters: Worker/Alerts.DeadPlaced }
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
| Add a field | Compatible. A message from an older publisher has the field's zero value; the invariant must accept it. |
| Remove a field | Compatible. An older subscriber reads the field's zero value; its invariant must accept it. |
| Rename a field | Breaking. Source has no field numbers, so a rename looks like a remove and an add. |
| Change an invariant (relax or tighten) | Breaking. Both sides check invariants, so a change in either direction rejects messages in one deploy order. |
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
        timeout 200 * time.Millisecond
}

// In a leaf of Api:
qty, err := Stock.Level(sku)    // err can also be Unavailable, Timeout, or RemotePanic
```

- **Transport errors.** Each method of a remote port can also return the errors `Unavailable`,
  `Timeout`, and `RemotePanic`, which `errors.Is` matches.
- **Each call has a timeout.** The `config { }` section of `system.keel` sets a default timeout. A method can override it
  with `timeout`. If no timeout resolves, the code does not compile.
- **A timeout means the outcome is unknown.** The call can have run or not. The runtime never
  retries a remote call. If the caller retries, the method must be safe to repeat.
- **Contracts are checked on both sides.** The caller checks `requires` before it sends. The callee
  checks it again when it receives. The callee checks `ensures` before it replies, and the caller
  checks it again when it receives the reply.
- **A first `context.Context` parameter carries the caller's deadline and cancellation** to the
  callee. It is exempt from the next rule: the transport carries only its deadline and cancellation.
- **Types must be serializable**, as for topic messages. The schema lock (§5.5.2) also covers the
  parameter and result types of remote methods, so the same evolution rules apply.
- **The wire names the transport** with `via`, as for topics. The built-in `direct` transport
  supports request and response:

  ```go
  Api/Orders.Stock: Worker/Inventory.NewLevels via direct{}
  ```

  The topic settings (`deadletters`, `retries`, `durable`, `whenFull`, `capacity`, `ttl`,
  `concurrency`) apply
  only to topic wires. A call wire accepts none of them.

- **A panic in the callee** is recovered by the runtime (§6); the caller receives `RemotePanic`.
- **The schema lock covers the contracts of remote ports.** A change to a remote port's contract is a
  breaking change, as for message invariants (§5.5.2).

**How to choose.** Use a remote method port when the caller needs an answer now. Use a topic when
the publisher must not depend on who reads, or when many subscribers react to one event. The
developer decides for each boundary.

### 5.6 Diagrams

`keel diagram --level context|container|component|code [--focus <node>]` draws each C4 level from
`system.keel` and the `component.keel` files. Composites hold no code. Thus a diagram at any zoom
level shows where all behavior is. A container or a group cannot hide logic.

## 6. Code: Go, unchanged

**Function bodies are Go.** Types, generics, errors, `nil`, zero values, integers, slices, maps,
goroutines, channels, locks, `panic`, and `recover` all have exactly Go's syntax and Go's behavior.
Each Go construct compiles in Keel and runs as it runs in Go. Keel adds no rule inside a function
body, except contracts (§8).

What Keel adds is around the code, not in it: the architecture (§5), contracts (§8), tests (§9),
and the tools (§10, §11).

**Checks on Go code are warnings.** The tools report common Go faults as warnings with fix-its:
an unhandled error, error masking, a possible `nil` dereference, a signed integer overflow risk,
and a shared field without a lock. A warning never stops a build. `keel audit` lists them.

**Where code starts.** The compiler generates `main` from `system.keel`; code never writes it. A
leaf can declare one `startup func(ctx context.Context) error`. The generated `main` runs all
startup functions at once, for example to start the server of an entry leaf. When any startup
function returns an error or panics, `main` cancels `ctx` for the others and the process exits with
a report. The process runs until a stop signal, or until a startup function fails. A container with
no subscription, no served remote port, and no entry leaf exits when all its startup functions have
finished.

**Panics in handlers and served calls.** The runtime recovers a panic in a topic handler (§5.5.0)
or in a served remote call (§5.5.3), and counts it as a failure of that message or call, with a
report. Elsewhere a panic behaves as in Go.

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
`requires slices.Every(xs, isPositive)`. Its runtime cost is O(n), and contracts run in every build.
`keel audit` lists each contract whose check costs more than O(1). When that cost is too high for
a hot path, the developer states the rule as a test instead of a contract.

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
- **Outer calls only.** The invariant is checked when the outermost method call on the value
  returns, exported or not. A nested call on the same value, for example `Transfer` that calls `Withdraw`, is
  not checked on its return. Thus a method can pass through a temporarily broken state. This is
  the rule of Eiffel.

```go
type Account struct { Balance int }
    invariant Balance >= 0

a := Account{Balance: -5}   // compile error: Balance >= 0 is false
a.Balance = -1              // compile error: write outside a method

func (a *Account) Transfer(to *Account, x int) {
    a.Withdraw(x)           // nested call: no check on its return
    to.Deposit(x)
}                           // checked here
```

How the compiler checks contracts:

- **Where.** A function checks its `requires` at entry and its `ensures` at each return. The
  compiler can remove a check at a call site where it proves the contract.

- **Statically**, when the compiler can prove or disprove a contract.
- **At runtime** in every build, including release builds. Contracts cannot be turned off. A contract
  that fails at runtime panics, with a report of the contract, its source position, and the values.
- **Port contracts bind each implementation**, including test fakes. If a fake violates the
  contract of its port, the test fails.

How reviewers audit contracts:

- `keel audit` lists each contract. For each contract, it shows if the compiler proved it and if
  tests exercised it.
- Contracts appear in the generated docs and diagrams.

## 9. Testing

- `test` blocks are next to the code that they test, in `*_test.keel` files in the leaf directory.
- The design makes each leaf testable in isolation: its only dependencies are the ports that it
  `needs` and the topics that it subscribes to.
- A fake is declared in a test file with one form: `fake CheckoutOK for Checkout { ... }`. Each
  fake must meet the contract of its port.
- In a test, port names are bound for each test separately. Tests that run in parallel run in
  separate processes, so they never share a fake.
- In a test, each published topic records its messages, and the test can read them.
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

  warning K0412: syncStock in Api/Orders and Worker/Inventory are 91% similar
    cannot extract: syncStock calls acquires func os.Open
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
| `keel guide` | Prints the language primer (`docs/guide.md`) and the conventions of the project. **The primer has a hard budget of 4,000 tokens**, and it must stay complete: a model that reads only the primer must be able to write correct Keel. A language change that does not fit the budget must be cut or simplified. |
| `keel context <component>` | Emits the minimum context to work on one leaf: its `component.keel`, the wires in `system.keel` that connect it, and the method sets of the types on the other end of each wire (R1). |
| `keel query <symbol>` | Looks up a symbol exactly in the symbol index of the compiler. This prevents invented APIs (R4). |
| `keel explain <code>` | Gives a long explanation of a diagnostic, with examples. |
| `keel doc` / `keel diagram` | Generates docs, and C4 diagrams at each level (context, container, component, code), from the source (§5.6). |
| `keel audit` / `keel archdiff` | Audits contracts and architecture (§8, §10). |
| `keel fmt` | The one formatter. |
| `keel test <leaf>` | Builds and runs the tests of one leaf (§9). |
| `keel fake <Port>` | Writes the skeleton of a fake (§9). |
| `keel trace <id>` | Shows the steps of one traced message (§5.5.1). |
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
- The concurrency model follows Go (tasks like goroutines, and channels), unchanged.

- **Build output is the developer's decision.** A system with one container compiles to one binary.
  For a system with more than one container, the `config { }` section of `system.keel` chooses the output. The compiler
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
| Contracts (§8) | The cost of each expression, in release builds | The audit lists each contract with a collection-wide cost; the compiler removes checks that it proves |
| Garbage collection | GC pauses and memory overhead | On the Go backend, Go's GC; the LLVM backend needs Keel's own GC |
| Durable topics (§5.5.1) | A disk write for each message, on transports that keep queues on disk | The developer chooses the transport and its `durable` setting |
| Port calls (§5.3) | An interface dispatch and the port's `requires` and `ensures` at each call | The compiler removes checks that it proves; the LLVM backend can make the call direct |

**Backends.** On the Go backend, Keel performs like Go, plus contract checks and port dispatch. To close the gap
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
