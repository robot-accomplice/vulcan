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

## 2. Design principles

| Principle | Meaning in Keel |
|---|---|
| Compiled, GC | Native binaries and a tracing GC. No manual memory management and no borrow checker. |
| Go ergonomics | Go is the syntax model. Keel changes Go only where Go is weak, for example generics and error enforcement. |
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
| `(T, error)` is a sum type: a value or an error, never both. An unhandled error is a compile error. Postfix `?` propagates an error. `none` means "no error", as it means "no value" for `?T`. | R3, and no zero value on the error path (§7) |
| No `nil` or `null` value. Optional types `?T`, narrowed by control flow. | No null dereference can occur (§7a) |
| No exceptions of any kind. No `try`, `catch`, `throw`, or `recover()`. | Errors are values. Only a supervising task handles an abort (§7, §7.1) |
| Integer overflow aborts. `+% -% *%` wrap on purpose. | Go wraps without a signal (§7c) |
| A lock that an aborting task holds becomes poisoned. `Lock()` returns an error. | Go has no poisoning (§7.1) |
| Values that `go` shares must be copies, channels, constants, or `guarded by` a mutex | Prevents data races such as concurrent map writes (§7.1) |
| `go` runs only inside a `tasks { }` scope (structured concurrency); `detach` for background tasks. A task exit is `Completed`, `Failed`, or `Aborted`. | Each error and abort has a known destination (§7.1) |
| `requires`, `ensures`, and `invariant` clauses | Contracts in the code (§8) |
| `pure` functions | Contracts can call them. The compiler verifies that they have no effects (§8.1) |
| `test` and `fake` blocks | Testability by design (§9) |
| `topic<T>` publish channels between containers | A publish never waits for a subscriber and has zero or more subscribers. A `chan` is different (§5.5) |
| `remote port`, `idempotent` handlers, `via` wiring | Honest communication between containers (§5.5) |
| `topic.Publish(msg)` and `handle` functions; a message is acknowledged when its handler returns `none` | No forgotten or early acknowledgement (§5.5.0) |
| `max` bounds on variable-size fields of topic messages | The compiler calculates the memory limit of each queue (§5.5.1) |
| `gateway for` leaves; only gateways import network libraries | Enforced access to external systems (§5.4b) |
| `foreign library` nodes are the only importers of Go packages | Keel guarantees hold outside a guarded boundary (§5.4c) |
| Only leaves call `acquires` functions; libraries use handles they are given | Each I/O decision is visible in a leaf (§5.4a) |
| `const` extends to composite values and is deeply immutable; libraries have no package-level `var` | Enforceable “no mutable state” in libraries (§5.4a) |
| `ignore err, "reason"` and `x ?? fallback` | Explicit discards and defaults that the audit lists (§7, §7a) |
| Sum types with an exhaustive `switch` | Typed errors, task exits, and state machines (§6) |
| Generics use `<T Bound>`, not `[T Bound]` | Familiar from the C family. An unbounded parameter needs no `any` (§6.1) |
| Declaration files (`system.keel`, `component.keel`, and so on) have their own syntax: `provides X from Y`, `parent.X`, `uses A -> B`, `layers a < b`, `via`, `flag("...")`, wire options (`capacity`, `whenFull`, `ttl`, `deadletters`, `deadletterCapacity`), `outbox` lines, method renames (`{ Pay = Charge }`), `budget`, `gateway for`, `client of`, `go "package"` | Architecture as a first-class construct. These files hold no Go-style code (§5) |
| Contract expressions add `result`, `old(x)`, `=>` (implies), `all x in xs: ...`, and `some x in xs: ...` | Contracts in the code (§8.1) |
| `unreachable` statement | Marks code that must never run; reaching it aborts (§7) |
| Methods on built-in types: `xs.At(i)`, `a.CheckedAdd(b)`, `CheckedSub`, `CheckedMul`, `CheckedConvert<T>` | Explicit handling of bounds and overflow (§7b, §7c) |
| A narrowing conversion (`int32(x)`) aborts when the value does not fit | Go truncates without a signal (§7c) |
| A single-value map read `m[k]` is `?V`. `m[k]++`, `m[k] += x`, and `v, ok := m[k]` work as in Go when `V` has a zero value | A missing key is never a silent zero (§7a) |
| A private function can omit its result types; the body alone determines them | Less noise, with no inference across call sites (§6) |
| `mu.LockForRepair()` and `mu.ClearPoison()` | Repair of poisoned data under the lock (§7.1) |
| Test files are `*_test.keel` in the leaf directory | Tests next to the code (§9) |

All other syntax follows Go, unless a later section says otherwise. This includes `:=`, `func`,
structs, methods with receivers, `for`, `switch`, slices, maps, `defer`, packages as directories,
and gofmt-style formatting.

### 4.1 Keywords

Each reserved word breaks Go code that uses it as a name, and models write that code (R8). Thus
Keel reserves few words.

- **Reserved everywhere:** `none`, `handle`, `pure`, `tasks`, `detach`, `ignore`, `unreachable`.
- **Contextual:** all other new words. They are keywords only where they start a declaration, or
  inside a declaration file or a contract. Elsewhere they are ordinary identifiers. This applies to
  `system`, `container`, `group`, `component`, `library`, `foreign`, `port`, `remote`, `layer`,
  `layers`, `provides`, `requires`, `ensures`, `invariant`, `uses`, `via`, `in`, `out`, `topic`,
  `test`, `fake`, `result`, `old`, `all`, `some`, `max`, `guarded`, `acquires`, `idempotent`,
  `gateway`, `client`, `budget`, and `outbox`.

```go
port := 8080                   // an identifier
func read(in io.Reader)        // an identifier

port Payments { ... }          // a keyword: it starts a declaration
```

`keel guide` prints both lists.

## 5. Architecture model

The architecture is a tree that follows the C4 model. Code is only in small, flat leaves. Each node
above the leaves is only structure: children, ports, and wiring.

### 5.1 Hierarchy

```
system Shop                         C4 System, with the people and external systems around it
 ├ container Api                    C4 Container: a deployable or process
 │  ├ component Billing   (leaf)    C4 Component: one flat directory of code
 │  └ component Orders    (leaf)
 └ container Worker
    └ group Fulfilment              optional composite, for zoom levels inside a container
       ├ component Picking  (leaf)
       └ component Shipping (leaf)
```

A *leaf* is a `component` node. A *composite* is a `system`, `container`, or `group` node.

| Kind | Holds | Code? |
|---|---|---|
| `system` | containers, `person` and `external` declarations | no |
| `container` | groups and components | no |
| `group` | groups and components | no |
| `component` | source files and tests | yes, but no child nodes |
| `library` | shared types and reusable code | yes, but no ports, topics, or mutable state (§5.4a) |

### 5.2 Layout

The directory tree is the same as the architecture tree. Each node is a directory. Each directory
holds exactly one declaration file, and the name of the file is the kind of the node:
`system.keel`, `container.keel`, `group.keel`, `component.keel`, or `library.keel`.

- A composite directory holds no source files.
- A leaf directory and a library directory hold no child nodes.
- Libraries are in the `lib/` directory of the system.

Thus each thing has one location.

### 5.3 Declarations

A leaf declares the ports that it provides and the ports that it requires:

```go
// Billing charges customers and records the charge in the ledger.
component Billing {
    layer    app
    provides Payments
    requires Ledger, Clock
}
```

A port is an interface with contracts. **Each node owns the ports on its own boundary. It knows
nothing about the nodes on the other side.**

- The provider declares each port in its `provides` list: "what I offer".
- The consumer declares each port in its `requires` list: "what I need". This is the hexagonal
  style.

A provider does not know which nodes use it. A consumer does not know which node serves it. Only
the parent that wires the two nodes together knows both of them.

**Inside a leaf, each required port is a package-level name**, the name of the port itself. Code
calls it like a Go package: `Checkout.Pay(c, amt)`. The wiring binds the name at startup, and the
binding cannot change after that. A test binds fakes to the same names (§9). A leaf appears once in
the tree; reuse of code goes through libraries (§5.4a).

```go
// In Billing (provider): what Billing offers.
// Payments charges a customer's stored payment method.
port Payments {
    // Charge moves amount from the customer to the merchant.
    func Charge(customer CustomerID, amount Money) (Receipt, error)
        requires amount > 0
        ensures  result.Amount == amount
}

// In Orders (consumer): what Orders needs, with its own names and contract.
// Checkout takes payment for a confirmed order.
port Checkout {
    func Pay(customer CustomerID, amount Money) (Receipt, error)
        requires amount > 0 && amount < MaxOrder
}
```

**Port compatibility.** When a parent wires a provided port to a required port, the compiler does
these checks:

1. **Signatures.** Each method of the required port must exist on the provided port, with an
   identical signature. The provided port can have more methods. The wire can rename methods:
   `Orders { Checkout: Billing.Payments { Pay = Charge } }`. A rename changes only the name; the
   parameter and result types must still be identical.
2. **Contracts (substitutability).** For each method:
   - The consumer's `requires` must imply the provider's `requires`. Thus the provider accepts all
     inputs that the consumer can send.
   - The provider's `ensures` must imply the consumer's `ensures`. Thus the provider promises at
     least what the consumer relies on.

   If the compiler cannot prove an implication, it adds a runtime check at the wire and lists the
   check in `keel audit`.

If two ports differ in more than names, write an adapter. An adapter is an ordinary leaf that
requires one port and provides the other. Keel has no special adapter construct, because
composites hold no code. The near-duplicate check (§10) ignores port declarations: the two sides
of a boundary are separate contracts on purpose.

A composite wires its direct children and publishes some of their ports. Sometimes no sibling
satisfies a child's requirement. Then that requirement becomes a requirement of the composite, and
the composite's own parent satisfies it:

```go
// Api serves the public HTTP interface.
container Api {
    requires OrderFlow                  // Orders' remote port, passed up to the system (§5.5)

    Billing     { Ledger: LedgerStore.Ledger, Clock: SystemClock.Clock }
    Orders      { Checkout: Billing.Payments { Pay = Charge }, OrderFlow: parent.OrderFlow }
    LedgerStore {}
    SystemClock {}
}

// Worker picks and ships orders.
container Worker {
    provides Intake from Fulfilment     // re-exported; the Fulfilment group stays hidden

    Fulfilment {}
}
```

The compiler does the port-compatibility check, with the rename, at
`Orders { Checkout: Billing.Payments { Pay = Charge } }`.

The system adds the outer ring of C4: the people and the external systems that the system
interacts with.

```go
// Shop is the online store.
system Shop {
    layers domain < app < infra

    person   Customer "Buys goods through the web store"
    external Stripe   "Card payment processor"

    Worker { Intake.Placed:     Api.OrderFlow.Placed  via queue }   // topics are the only
    Api    { OrderFlow.Shipped: Worker.Intake.Shipped via queue }   // links between containers

    uses Customer -> Api "places orders over HTTPS"
    // Api -> Stripe comes from the gateway declaration (§5.4b).
}
```

Keel does not wire external systems with ports. The language does not control them, and they use
their own protocols. Instead, one **gateway leaf** in each container talks to each external system
(§5.4b). The compiler draws that
relationship in diagrams from the gateway declaration. A `uses` line records a relationship that
no declaration shows, such as a person who uses a container.

### 5.4 Rules the compiler enforces

1. **Ports only.** Code in a leaf can use another node **only** through a port in its `requires`
   list, or through an imported library (§5.4a). Any other reference to another node is a compile
   error.
2. **Code only in leaves.** Composites hold no source files. Leaves hold no child nodes.
3. **Encapsulation at each level.** Outside its parent, a node does not exist. A composite exposes
   only the ports that it publishes with `provides X from Child`.
4. **Complete wiring.** Each composite wires each `requires` entry and each `in` topic of each
   direct child exactly once. The wire goes to a port or a topic of a sibling, or to a `requires`
   entry of the composite. An `out` topic is never wired by itself (§5.5).
5. **Complete provision.** A leaf implements each port in its `provides` list. A composite gets
   each port in its `provides` list from a child.
6. **Compatible wiring.** Each wire passes the port-compatibility check in §5.3.
7. **Boundary ownership.** A node declares the ports that it provides and the ports that it
   requires. It cannot name a port that a sibling or any other node declares.
8. **Layers.** The `layers` order of the system constrains **source dependencies**: the libraries
   that a leaf imports, and the types in its port signatures. A leaf can use these only from its
   own layer or a lower layer. Layers do not constrain wires. Two wired leaves share no source
   code, because each owns its own ports, so a wire can connect leaves in any layers. Thus an `app`
   leaf can require a port that an `infra` adapter provides, and an `infra` HTTP handler can
   require a port that a `domain` leaf provides. But a port in an `app` leaf cannot mention an
   `infra` type such as `sql.Row`.
9. **No cycles.** At each level, the dependency graph of the siblings has no cycles.
10. **Private types.** A type is private to its leaf, unless the type is in a port signature.
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
external system into a normal Keel port. Other leaves require that port, and they contain no
network code.

```go
// api/stripegw/component.keel
// StripeGateway charges cards through Stripe.
component StripeGateway {
    gateway  for Stripe
    provides CardNetwork
}

// api/billing/component.keel
component Billing {
    requires CardNetwork     // no network code in Billing
}
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
   locks that do not become poisoned, and hold global state. `keel audit` lists each foreign
   library and these limits, so that a reviewer can see where the guarantees stop.

**The standard library.** The Keel standard library is a set of foreign libraries over the Go
standard library, for example HTTP, JSON, SQL, and cryptography. It ships with the compiler.

**Versions.** A Keel project uses Go modules (`go.mod`) to pin the versions of Go packages.

### 5.5 Crossing containers

Containers are separate processes. Thus a **topic** is the only thing that crosses from one
container to another. A topic is a publish channel. It is like a Go channel, but it works across
the process boundary. Method ports never cross a container boundary.

A topic has its own type, `topic<T>`. It is not a Go `chan T`, because the two behave differently:

- A send on a Go channel blocks until a subscriber takes the value. Each value has exactly one
  subscriber.
- A publish on a topic never waits for a subscriber. It waits only when its own outbox is full (§5.5.1). Each message has zero, one, or many subscribers.

One keyword with two behaviors would mislead humans and models trained on Go (R8). A Go `chan`
keeps its Go behavior for concurrency inside one process.

```go
// In Orders, inside Api. Each side declares its own view of the topics.
remote port OrderFlow {
    Placed  out topic<OrderPlaced>    // Orders publishes
    Shipped in  topic<ShipmentSent>   // Orders subscribes
}

// In Picking, inside Worker.
remote port Intake {
    Placed  in  topic<OrderPlaced>
    Shipped out topic<ShipmentSent>
}

// In Shop. Each subscriber names its source, and the parent names the transport.
Worker    { Intake.Placed:     Api.OrderFlow.Placed  via queue }
Analytics { Ingest.Placed:     Api.OrderFlow.Placed  via queue }   // a second subscriber
Api       { OrderFlow.Shipped: Worker.Intake.Shipped via queue }
```

**Why publish: it is the loosest coupling possible.** The only job of the publisher is to publish.
The publisher does not know, and must not care, if a subscriber reads its messages. It does not know
how many subscribers there are, or when they read.

- **Publish, never wait for a subscriber.** A publish gives the message to a local outbox and
  returns. The runtime delivers the message from the outbox. Subscribers and the network cannot
  make a publish fail. Compare a Go channel: an unbuffered send blocks until a subscriber takes the
  value.
- **The only wait is a full outbox.** Under the default `block` policy (§5.5.1), a publish waits
  when its own outbox is full. A full outbox means overload. The runtime reports the depth and the
  time until full before that point. The publisher still knows nothing about its subscribers: it is
  coupled to them in time only, and only under overload.
- **Zero, one, or many subscribers.** The parent wires topics one at a time, from the subscriber
  side. Each `in` topic is wired exactly once, to one `out` topic of the same message type. An
  `out` topic is never wired by itself: its subscribers are the `in` topics wired to it, zero or
  more. Thus a new subscriber adds one wire, and the wiring of the publisher does not change.
- **Where the settings go.** The settings of one subscription (the inbox capacity, the `ttl`, and
  the dead-letter route and size) go on the wire of the subscriber. The settings of the outbox
  (its capacity and its `whenFull` policy) go on an `outbox` line in the parent, or come from the
  project default (§5.5.1).
- **No request/response.** A reply would bind the publisher to a subscriber. If a publisher must
  know the result, it subscribes to a separate event that the other side publishes. `Shipped` in
  the example above is such an event.
- **To read data that another container owns, keep a read model.** The owner publishes change
  events. The reader subscribes, keeps a local copy, and answers queries from that copy. This is
  "event-carried state transfer". Reads can be slightly stale, and each reader stores its own copy.

  ```go
  // Worker owns stock and publishes each change.
  StockChanged.Publish(StockChanged{SKU: s, Qty: q})

  // Api keeps a read model, and answers GET /stock/42 from it.
  idempotent handle StockChanged(m StockChanged) error {
      stock.Set(m.SKU, m.Qty)
      return none
  }
  ```

  A leaf cannot call another container over the network, because only gateways can import network
  libraries (§5.4b, §5.4c). `keel audit` lists each `external` system with its gateway, so a
  reviewer can find an `external` that is really a container of the same system.
- **Ownership.** Each side owns only its end of each topic. Only message values cross, as copies.
  The two sides share no memory and no references.
- **`remote` is necessary to cross.** The compiler rejects a wire across containers if the consumer
  did not declare the port `remote`.
- **Contracts** on message types (struct `invariant`s) are checked when a message is published and
  again when it is received. If a received message fails its invariant, the runtime sends it to
  the dead-letter topic with the report; the subscriber does not abort. Message types must be
  serializable. They cannot contain functions, topics, or pointers into the memory of the
  publisher.
- **Dead letters must have a route and a size.** A message that uses all its retries goes to the
  dead-letter topic of its subscription. The wire of each subscriber must give the dead-letter
  topic a route and a size:

  ```go
  Worker { Intake.Placed: Api.OrderFlow.Placed via queue,
           deadletters: Ops.Alerts, deadletterCapacity: 5_000 }             // fixed at compile time
  Worker { Intake.Placed: Api.OrderFlow.Placed via queue,
           deadletters: Ops.Alerts, deadletterCapacity: flag("dlq-size") }  // supplied at startup
  ```

  If the route or the size is missing, the code does not compile. If a runtime flag supplies the
  size, the process checks the flag at startup. The process does not start if the flag is missing
  or not valid. To have no route, write `ignore deadletters, "reason"`. `keel audit` records it,
  the same as `ignore err`.
- **The author chooses the delivery guarantee. The language does not.** The language lets the
  author declare the guarantee for each remote topic: at-most-once or at-least-once, and ordered or
  unordered. **The default is at-least-once and unordered.** No message is lost. A message can
  arrive twice, so subscribers must be idempotent.
- **An unread topic causes a warning.** A topic with zero subscribers compiles, with a warning. A
  publisher can exist before its subscribers, but an unread topic must not go unnoticed.
- **A subscriber abort never reaches the code of the publisher.** If a subscriber aborts while it
  handles a message, the publisher is not affected. The outbox records the event: the message, the
  subscriber, and the attempt. The runtime then retries the message and, at the end, sends it to
  the dead-letter topic, as the delivery guarantee of the topic specifies.
- **An outbox failure aborts.** The outbox itself can fail: it cannot store a write, or its store
  is corrupt. Then the publishing task aborts (§7.1). The code of the publisher still never sees an
  error.
- **The delivery guarantee sets the durability of the outbox.** The outbox of an at-least-once
  topic is on local disk, so undelivered messages survive a process restart. The outbox of an
  at-most-once topic can be in memory, because that guarantee already permits loss.
- **At-least-once holds only while the delivery mechanism is intact.** The language retries a
  message until the runtime delivers it or sends it to the dead-letter topic. The language does not
  promise that the delivery mechanism itself survives, for example after a disk is lost or corrupt.
  Protection against that is system design, for example persistent brokers or replication. It is
  outside the scope of the language.
- **Message tracing is off by default.** Turn it on for one topic or for the whole project. When
  tracing is on, the runtime gives each message an ID and records each step: published, delivered
  to a subscriber, aborted, retried, acknowledged, or sent to the dead-letter topic.
  `keel trace <id>` shows these steps for one message. Retention is bounded, the same as for all
  other runtime stores (§5.5.1). Developers know their own circumstances. The language does not use
  their disk and CPU for a scenario that it cannot predict.
- **Idempotency is declared.** A subscriber on an at-least-once topic must mark its handler
  `idempotent`. Without the marker, the code does not compile. The compiler cannot prove the claim,
  but the marker makes a hidden assumption visible. The marker appears in `keel context`, in
  `keel doc`, and in `keel audit`, where reviewers can check it.

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
    return none          // the message is acknowledged here
}
```

- Each `in` topic of a leaf has exactly one handler. Code cannot iterate over an `in` topic.
- The runtime runs each message in its own task and supervises that task (§7.1).
- **The acknowledgement point is the return of the handler:**

  | Handler outcome | At-least-once topic | At-most-once topic |
  |---|---|---|
  | Returns `none` | Acknowledged. | Acknowledged. |
  | Returns an error | Retried, then sent to the dead-letter topic. | Discarded and counted. |
  | Aborts | Retried, then sent to the dead-letter topic. | Discarded and counted. |

  Thus code cannot forget to acknowledge a message, and it cannot acknowledge a message too early.
- The runtime runs handlers of an unordered topic concurrently, up to a limit. The limit comes from
  the wire of the subscriber or from the project default. On an ordered topic, the runtime runs one
  handler at a time for each ordering key.

#### 5.5.1 Bounded buffers

Decoupling needs buffers. Each buffer must at some time refuse a message or discard one. The wiring
must state which, so that no runtime queue can grow in memory without a limit.

- **Each runtime queue has a capacity.** The runtime queues are the outbox, the retry queue, the
  subscriber inbox, and the dead-letter topic. A capacity comes from the wire, from the one project
  default, or from a startup flag (`flag("name")`, checked at startup as in §5.5). If a queue of a
  remote topic has no capacity from one of these sources, the code does not compile.
- **The full-queue policy comes from one project default, unless an `outbox` line overrides it.**
  The default is `block`, which never loses a message. A project can change the default in its
  config. An `outbox` line can override the default when the message needs a different policy. For each topic, `keel audit`
  shows if the policy is the project default or an override.

  | `whenFull` | Effect | Typical use |
  |---|---|---|
  | `block` | The publisher waits for space. It is coupled to slow subscribers in time only. | Business events: the loss of one is not acceptable |
  | `dropOldest` / `dropNewest` | The runtime discards a message and counts it. A drop is not a failure, so the message does not go to the dead-letter topic. `keel audit` lists the policy. | Telemetry: old data is acceptable |
  | `abort` | The task aborts (§7.1). | Waiting is not acceptable and loss is not acceptable |

  ```go
  outbox Api.OrderFlow.Placed { capacity: 10_000, whenFull: block }          // publisher side
  Worker { Intake.Placed: Api.OrderFlow.Placed via queue, capacity: 5_000, ttl: 24h,
           deadletters: Ops.Alerts, deadletterCapacity: flag("dlq-size") }  // subscriber side

  outbox Api.Metrics.Samples { capacity: 1_000, whenFull: dropOldest }
  Obs    { Ingest.Samples: Api.Metrics.Samples via queue, capacity: 1_000, ttl: 5m,
           deadletters: Ops.Discard, deadletterCapacity: 100 }
  ```

- **Messages expire.** Each topic has a time to live (`ttl`). The `ttl` comes from the same sources
  as a capacity. The runtime sends an expired message to the dead-letter topic. Thus a dead
  subscriber cannot hold memory without a limit.
- **A full dead-letter topic aborts the subscriber process.** A dead-letter topic belongs to one
  subscription (§5.5). When it is full, the container process of that subscriber aborts with a
  report. The publisher is never affected. At-least-once messages stay in the durable outbox of the
  publisher, so no message is lost, and delivery continues when the subscriber restarts. The
  dead-letter topic cannot use `block`, because that would stop the pipeline on its own error path.
  It cannot discard messages, because a discarded dead letter is a silent failure.
- **Limits are visible before the system reaches them.** For each queue, the runtime reports the
  depth, the highest depth so far, and an estimate of the time until the queue is full.
  `keel audit` lists the capacity, the policy, and the `ttl` of each topic. Thus a reviewer can see
  the memory limit of the whole system in one location.
- **Message types are bounded.** A capacity counts messages, so it bounds memory only if each
  message has a maximum size. Each variable-size field of a topic message type declares a maximum:

  ```go
  type OrderPlaced struct {
      ID    string max 64
      Lines []Line max 100
      Note  string max 4KB
  }
  ```

  A field without a maximum is a compile error in a topic message type. The compiler calculates the
  maximum encoded size of each message type. For each queue, it calculates the memory limit:
  capacity × maximum message size. `keel audit` shows each limit and the total for the system. A
  value over a maximum fails the invariant of the message at publish, and the publishing task
  aborts, as for any contract violation.

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
| Add a field of type `?T` | Compatible. A message from an older publisher has `none` in the field. |
| Remove a field of type `?T` | Compatible. An older subscriber reads `none`. |
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

### 5.6 Diagrams

`keel diagram --level context|container|component|code [--focus <node>]` draws each C4 level from
the declarations above. Composites hold no code. Thus a diagram at any zoom level shows where all
behavior is. A parent cannot hide logic.

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
- **No nil.** No type has a null value. This includes pointers, interfaces, slices, and maps (§7a).
- Methods attach to structs, as in Go. Interfaces are structural, as in Go. Ports are nominal.

### 6.1 Generics

Generics use angle brackets. A bound follows its parameter, as in a Go parameter list: the name,
then the type, with no colon.

```go
func Max<T Ordered>(xs []T) ?T
type Cache<K Hashable, V> struct { ... }
func (c *Cache<K, V>) Get<R>(k K, f func(V) R) ?R   // methods may declare their own type params
```

- A parameter without a bound needs no bound: write `V`, not `V any`.
- `&` joins two or more bounds: `<T Ordered & Hashable>`.
- Usually the compiler infers the type arguments. Explicit instantiation is rare.
- A method on a concrete type can declare its own type parameters, as Go 1.27 now permits. A method
  of an interface or a port cannot. Dynamic dispatch of a generic method would need to know each
  instantiation in advance.

**Parsing.** In declarations and in type positions, `<` has only one meaning. In expressions,
`a < b > (c)` can be two comparisons or a generic call. Keel uses the rule of C#. The rule needs
lookahead, but it does not need a symbol table:

- If `<...>` parses as a type-argument list, **and** the next token is one of `( ) ] } , ; . {` or
  the end of the line, then `<...>` is a type-argument list.
- If not, `<` means "less than".

When `>>` closes nested type arguments, the parser splits it into two `>` tokens, as in C#, Java,
and C++11. The formatter adds parentheses to each expression that is still ambiguous. Thus humans
and models never see an ambiguous expression.

## 7. Errors

**Keel has no exceptions, no `try`, no `catch`, and no `throw`. No function can catch an abort.
Recovery occurs only at task boundaries (§7.1).**

An error is an ordinary value. It moves only through return values. Thus the source shows each path
that an error can take.

A function that can fail declares the Go shape, `(T, error)`. In Keel this shape is a **sum type**:
the function returns a value **or** an error, never both. Thus the error path needs no value of
`T`, and no zero value.

```go
func Charge(c CustomerID, a Money) (Receipt, error) {
    tok := vault.Lookup(c)?   // on failure, returns the error; no Receipt needed
    ...
    return receipt            // success: the value only
}

r, err := Charge(c, a)
if err != none {
    return err                // r is not usable here
}
use(r)                        // r is a Receipt here: the check above proved it
```

- `return v` returns success. `return err` returns failure. `return v, err` is a compile error,
  because the result cannot be both.
- `none` is the value of `error` that means "no error", the same word that means "no value" for
  `?T` (§7a). It replaces Go's `nil`: write `if err != none`. A function that returns only `error`
  ends with `return none` on success. `ok` is not reserved, so Go's comma-ok form still works.
- A shape with more values works the same way: `(A, B, error)` returns `a, b` or `err`.
- At the call site, `r, err := f()` binds both names. The compiler narrows `r` by control flow, the
  same as a `?T` (§7a): the code can use `r` only where the flow proves that `err == none`.

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
  error. At the end of the scope, the function waits for all the tasks that started in it. A
  handle cannot leave its scope.
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
      return none
  }
  ```

- **To ignore an error on purpose**, write `ignore go f(), "reason"`. `keel audit` lists it.
- **A long-lived background task** uses `detach f(), "reason"` in place of a scope. The process
  supervisor owns it. An abort of a detached task stops the process with the report. `keel audit`
  lists each `detach`.
- **Implicit scopes.** The body of `main`, each topic handler (§5.5.0), and each request at an
  entry point (Q21) run in their own scope. For a request, an abort ends that request with an error
  response and a report. The process keeps running. Thus one bad request cannot stop a server.
- The runtime records each abort report as an RCA trail: the task, the owning scope, the
  invariant, the source position, and the values.

**Poisoned locks.** Tasks share memory, as in Go. An abort during an update can leave shared data
half-changed after the state of the task is gone. Keel does what Rust does:

- If a task holds a mutex (or a read-write mutex) when it aborts, the mutex becomes **poisoned**.
- Each later `Lock()` returns an error, `Poisoned`, and does **not** hold the lock. The code must
  handle this error like all other errors (§7). Because the lock is not held, `mu.Lock()?` can
  never leave a lock held:

  ```go
  mu.Lock()?                      // propagates Poisoned to the caller; the lock is not held
  ```

- To repair the data, a task calls `mu.LockForRepair()`, which takes the lock even when it is
  poisoned. The compiler permits `mu.ClearPoison()` only between `LockForRepair()` and `Unlock()`
  in the same function. Thus no repair runs without the lock. `keel audit` lists each repair.

  ```go
  mu.LockForRepair()
  rebuild(&shared)
  mu.ClearPoison()
  mu.Unlock()
  ```

- **Deferred calls run when a task aborts**, in reverse order, as in Go. Thus files and
  connections close. If a deferred `Unlock()` runs, it releases the lock, but the lock stays
  poisoned, because the data may still be half-changed. An abort inside a deferred call stops the
  remaining deferred calls, and the runtime adds it to the report.
- Limit: poisoning protects only data that a lock guards. A change to shared data without a lock
  is a data race, and a data race is already a bug. The race detector runs in test builds to find
  data races.

**Sharing between tasks.** The compiler rejects the most common data races before they run. When
`go` captures or passes a value, the value must be one of these:

- a copy (a value type with no references inside it)
- a `chan`
- a `const` (§5.4a)
- a field of a struct that declares it `guarded by` a mutex in the same struct

```go
type Counts struct {
    mu   Mutex
    byID map[string]int guarded by mu
}

counts := map[string]int{}
go worker(counts)          // compile error: map shared with a task without a lock
  // hint: pass a copy, send it on a chan, or guard it with a Mutex
```

The compiler also checks each access to a `guarded by` field: the function must hold the mutex at
that point.

**Fault classes.** Not every fault can stay inside one task:

| Class | Faults | Result |
|---|---|---|
| Task-level | Contract violations, overflow, bounds, poisoned data, and panics in Go code inside a foreign library | The task aborts. The parent receives `Aborted` (§7.1). On the Go backend, the runtime wraps each task in `recover` to do this. |
| Process-level | Stack exhaustion, out of memory, deadlock of all tasks | The process exits with a report. The deployment restarts it. No parent receives `Aborted`. |

The rule on sharing between tasks removes concurrent map writes, the most common cause of a
process-level fault in Go programs.

## 7a. Null safety

Keel has **no null value**. `nil` does not exist. No expression can dereference nothing, so a
null-pointer failure cannot occur.

- Each type has a zero value, as in Go, **or** the code must initialize it before use. The code
  must initialize pointers, interfaces, and function values.
- Absence is a type, not a value. `?T` means "a `T` or `none`".
- The code cannot use a `?T` as a `T`. The compiler narrows the type by control flow:

  ```go
  u := users.Find(id)        // u is ?User
  if u == none {
      return NotFound
  }
  send(u.Email)              // u is User here: the check above proved it
  ```

- **Only a stable local narrows.** Narrowing applies to a local variable that no code assigns
  between the check and the use, and that no closure or task captures. A field, a pointer target,
  or shared data does not narrow, because a call or another task can change it. The fix-it copies
  the value into a local first:

  ```go
  if s.user != none {
      notify()
      send(s.user.Email)       // compile error: a field does not narrow
  }
  // fix-it:
  u := s.user
  if u != none {
      notify()
      send(u.Email)            // u is User here
  }
  ```

- `x ?? fallback` gives an explicit default. Keel permits it, because the code states the default
  where it is used. But `keel audit` lists each `??`, the same as `ignore`.
- An API that "might not find" a value returns `?T`. Examples: `Find`, a search, and a parse that
  has no error.
- **Maps keep the Go idioms where the value type has a zero value:**

  ```go
  counts[word]++                 // works: int has a zero value
  v, ok := counts[word]          // Go comma-ok form: v is 0 when the key is missing
  u := users[id]                 // a single-value read is ?User: a missing key is never a silent zero
  if u == none { ... }
  ```

  For a value type with no zero value (Q20), only the `?V` read exists.
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

An integer never wraps without a signal.

| Operation | On overflow |
|---|---|
| `+ - *`, unary `-`, narrowing conversion (`int32(x)`) | Abort. |
| `+% -% *%` | Wraps (two's complement). Use these for hashes, checksums, and other deliberate modular arithmetic. |
| `a.CheckedAdd(b)`, `CheckedSub`, `CheckedMul`, `CheckedConvert<T>` | Returns `?T`, which is `none` on overflow. Use these when the code handles the overflow itself. |
| Constant expressions | Compile error, as in Go. |

Division by zero aborts, as in Go.

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
pure func IsSorted<T Ordered>(xs []T) bool { ... }

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
| `keel context <component>` | Emits the minimum context to work on one leaf: its declaration, its ports and contracts, and the signatures of what it requires (R1). |
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
| Finds | All of §5.4 (ports only, wiring, layers, cycles, port compatibility), the topic rules (§5.5 to §5.5.1), near-duplicate code in different leaves (§10), architecture diffs, and the project audit. | Type, error-handling, null, bounds, overflow, and contract errors inside the leaf. |
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
| Q20 | Without `nil`, some types have no zero value (types that hold pointers, interfaces, or functions). What happens to `make([]*T, n)`, a generic `var x T`, and a partial struct literal? |
| Q21 | Inbound traffic. A leaf that serves HTTP to a `person` (for example, Api serving Customer) needs a network library, but it is not a gateway to an external system. How does Keel declare and enforce an entry point? |

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
