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
| An unhandled error is a compile error. Postfix `?` propagates an error. | R3 (§7) |
| No `nil` or `null` value. Optional types `?T`, narrowed by control flow. | No null dereference can occur (§7a) |
| No exceptions of any kind. No `try`, `catch`, `throw`, or `recover()`. | Errors are values. Only a supervising task handles an abort (§7, §7.1) |
| Integer overflow aborts. `+% -% *%` wrap on purpose. | Go wraps without a signal (§7c) |
| A lock that an aborting task holds becomes poisoned. `Lock()` returns an error. | Go has no poisoning (§7.1) |
| `go f()` returns a task handle. The parent can observe the task's exit. | Supervision (§7.1) |
| `requires`, `ensures`, and `invariant` clauses | Contracts in the code (§8) |
| `pure` functions | Contracts can call them. The compiler verifies that they have no effects (§8.1) |
| `test` and `fake` blocks | Testability by design (§9) |
| `topic<T>` publish channels between containers | A publish never blocks and has zero or more subscribers. A `chan` is different (§5.5) |
| `remote port`, `idempotent` handlers, `via` wiring | Honest communication between containers (§5.5) |
| `gateway for` leaves; only gateways import network libraries | Enforced access to external systems (§5.4b) |
| `ignore err, "reason"` and `x ?? fallback` | Explicit discards and defaults that the audit lists (§7, §7a) |
| Sum types with an exhaustive `switch` | Typed errors, task exits, and state machines (§6) |
| Generics use `<T Bound>`, not `[T Bound]` | Familiar from the C family. An unbounded parameter needs no `any` (§6.1) |

All other syntax follows Go, unless a later section says otherwise. This includes `:=`, `func`,
structs, methods with receivers, `for`, `switch`, slices, maps, `defer`, packages as directories,
and gofmt-style formatting.

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

**Port compatibility.** When a parent wires a provided port to a required port, the compiler does
these checks:

1. **Signatures.** Each method of the required port must exist on the provided port, with an
   identical signature. The provided port can have more methods.
2. **Contracts (substitutability).** For each method:
   - The consumer's `requires` must imply the provider's `requires`. Thus the provider accepts all
     inputs that the consumer can send.
   - The provider's `ensures` must imply the consumer's `ensures`. Thus the provider promises at
     least what the consumer relies on.

   If the compiler cannot prove an implication, it adds a runtime check at the wire and lists the
   check in `keel audit`.

If two ports do not match, write an adapter. An adapter is an ordinary leaf that requires one port
and provides the other. Keel has no special adapter construct, because composites hold no code.

A composite wires its direct children and publishes some of their ports. Sometimes no sibling
satisfies a child's requirement. Then that requirement becomes a requirement of the composite, and
the composite's own parent satisfies it:

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

The compiler does the port-compatibility check at `Orders { Checkout: Billing.Payments }`.

The system adds the outer ring of C4: the people and the external systems that the system
interacts with.

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

Keel does not wire external systems with ports. The language does not control them, and they use
their own protocols. Instead, one **gateway leaf** in each container talks to each external system
(§5.4b). The `uses` line records the relationship, so that the relationship appears in diagrams.

### 5.4 Rules the compiler enforces

1. **Ports only.** Code in a leaf can use another node **only** through a port in its `requires`
   list, or through an imported library (§5.4a). Any other reference to another node is a compile
   error.
2. **Code only in leaves.** Composites hold no source files. Leaves hold no child nodes.
3. **Encapsulation at each level.** Outside its parent, a node does not exist. A composite exposes
   only the ports that it publishes with `provides X from Child`.
4. **Complete wiring.** Each composite wires each `requires` entry of each direct child exactly
   once. The wire goes to a port of a sibling or to a `requires` entry of the composite.
5. **Complete provision.** A leaf implements each port in its `provides` list. A composite gets
   each port in its `provides` list from a child.
6. **Compatible wiring.** Each wire passes the port-compatibility check in §5.3.
7. **Boundary ownership.** A node declares the ports that it provides and the ports that it
   requires. It cannot name a port that a sibling or any other node declares.
8. **Layers.** Port dependencies obey the `layers` order of the system. A lower layer never
   requires a higher layer.
9. **No cycles.** At each level, the dependency graph of the siblings has no cycles.
10. **Private types.** A type is private to its leaf, unless the type is in a port signature.
11. **Leaf size budget (warning).** Each project sets one `context_budget` value in its config. If
    the `keel context` output of a leaf is larger than this budget, the compiler gives a warning.
    The hint in the warning tells where to split the leaf. Thus each unit stays small enough for a
    model to hold completely.

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
   cannot hold state, so it cannot couple the leaves that import it.
2. A library can import other libraries. It cannot import leaves or composites. Library imports
   have no cycles and obey `layers`.
3. A leaf can import a library in the same layer or a lower layer.
4. The leaf size budget (rule 11) also applies to libraries.
5. `keel diagram` shows libraries as shared dependencies. `keel audit` lists the leaves that use
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
2. Only a gateway leaf can import a raw network library, or a client library for an external
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

### 5.5 Crossing containers

Containers are separate processes. Thus a **topic** is the only thing that crosses from one
container to another. A topic is a publish channel. It is like a Go channel, but it works across
the process boundary. Method ports never cross a container boundary.

A topic has its own type, `topic<T>`. It is not a Go `chan T`, because the two behave differently:

- A send on a Go channel blocks until a subscriber takes the value. Each value has exactly one
  subscriber.
- A publish on a topic never blocks. Each message has zero, one, or many subscribers.

One keyword with two behaviors would mislead humans and models trained on Go (R8). A Go `chan`
keeps its Go behavior for concurrency inside one process.

```go
// In Orders (the consumer declares what it needs):
remote port Fulfilment {
    Orders  out topic<OrderPlaced>   // Orders publishes
    Shipped in  topic<ShipmentSent>  // Orders subscribes
}

// In Shop (the parent names the transport):
Api { Fulfilment: Worker.Intake via queue }
```

**Why publish: it is the loosest coupling possible.** The only job of the publisher is to publish.
The publisher does not know, and must not care, if a subscriber reads its messages. It does not know
how many subscribers there are, or when they read.

- **Publish, never wait.** A publish on a remote topic never blocks. Subscribers and the network
  cannot make it fail. The publish gives the message to a local outbox, and the runtime delivers the
  message from the outbox. Compare a Go channel: an unbuffered send blocks until a subscriber takes
  the value.
- **Zero, one, or many subscribers.** The parent can wire a topic to any number of subscribers,
  including zero. A change to the subscribers never changes the publisher.
- **No request/response.** A reply would bind the publisher to a subscriber. If a publisher must
  know the result, it subscribes to a separate event that the other side publishes. `Shipped` in
  the example above is such an event.
- **Ownership.** Each side owns only its end of each topic. Only message values cross, as copies.
  The two sides share no memory and no references.
- **`remote` is necessary to cross.** The compiler rejects a wire across containers if the consumer
  did not declare the port `remote`.
- **Contracts** on message types (struct `invariant`s) are checked when a message is published and
  again when it is received. Message types must be serializable. They cannot contain functions,
  topics, or pointers into the memory of the publisher.
- **Dead letters must have a route and a size.** A message that uses all its retries goes to the
  dead-letter topic of its topic. The parent that wires the topic must give the dead-letter topic a
  route and a size:

  ```go
  Api { Fulfilment: Worker.Intake via queue,
        deadletters: Ops.Alerts, deadletterCapacity: 5_000 }             // fixed at compile time
  Api { Fulfilment: Worker.Intake via queue,
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

#### 5.5.1 Bounded buffers

Decoupling needs buffers. Each buffer must at some time refuse a message or discard one. The wiring
must state which, so that no runtime queue can grow in memory without a limit.

- **Each runtime queue has a capacity.** The runtime queues are the outbox, the retry queue, the
  subscriber inbox, and the dead-letter topic. A capacity comes from the wire, from the one project
  default, or from a startup flag (`flag("name")`, checked at startup as in §5.5). If a queue of a
  remote topic has no capacity from one of these sources, the code does not compile.
- **The full-queue policy comes from one project default, unless a wire overrides it.** The default
  is `block`, which never loses a message. A project can change the default in its config. A wire
  can override the default when the message needs a different policy. For each topic, `keel audit`
  shows if the policy is the project default or an override.

  | `whenFull` | Effect | Typical use |
  |---|---|---|
  | `block` | The publisher waits for space. It is coupled to slow subscribers in time only. | Business events: the loss of one is not acceptable |
  | `dropOldest` / `dropNewest` | The runtime discards a message, counts it, and sends it to the dead-letter topic. `keel audit` lists the policy. | Telemetry: old data is acceptable |
  | `abort` | The task aborts (§7.1). | Waiting is not acceptable and loss is not acceptable |

  ```go
  Api { Fulfilment: Worker.Intake via queue, capacity: 10_000, whenFull: block, ttl: 24h,
        deadletters: Ops.Alerts, deadletterCapacity: flag("dlq-size") }
  Api { Metrics:    Obs.Ingest    via queue, capacity: 1_000,  whenFull: dropOldest, ttl: 5m,
        deadletters: Ops.Discard, deadletterCapacity: 100 }
  ```

- **Messages expire.** Each topic has a time to live (`ttl`). The `ttl` comes from the same sources
  as a capacity. The runtime sends an expired message to the dead-letter topic. Thus a dead
  subscriber cannot hold memory without a limit.
- **A full dead-letter topic aborts.** It cannot use `block`, because that would stop the pipeline
  on its own error path. It cannot discard messages, because a discarded dead letter is a silent
  failure. Thus a full dead-letter topic causes a loud failure.
- **Limits are visible before the system reaches them.** For each queue, the runtime reports the
  depth, the highest depth so far, and an estimate of the time until the queue is full.
  `keel audit` lists the capacity, the policy, and the `ttl` of each topic. Thus a reviewer can see
  the memory limit of the whole system in one location.

### 5.6 Diagrams

`keel diagram --level context|container|component|code [--focus <node>]` draws each C4 level from
the declarations above. Composites hold no code. Thus a diagram at any zoom level shows where all
behavior is. A parent cannot hide logic.

## 6. Type system

- **Strong and static.** There are no implicit conversions.
- **Inferred where possible.** The compiler infers the types of local variables, closures, and
  generic instantiations. Port signatures and exported functions are explicit, because humans
  review them as contracts.
- **A private signature can omit types only when inference is safe.** A private function is a
  function that is not exported and is not in a port. It can omit parameter types or return types
  only when all these conditions are true:
  1. The compiler infers the type from the **body of the function only**, never from its call
     sites. Thus a change to a caller can never change the type of a function.
  2. The body gives **exactly one** type. If a field access or a method call fits more than one
     type, the compiler reports an error. The error has a fix-it that adds the annotation.
  3. The function is not recursive. A recursive function must annotate its return type.

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

A function returns an error in the Go shape, `(T, error)`. But the code **must handle each error
explicitly**:

- If the code uses `T` from a call and does not handle the error of the call, the code does not
  compile.
- Postfix `?` returns the error to the caller: `r := billing.Charge(c, amt)?`. It is a short form
  of `if err != ok { return err }` and it does nothing more. It does not unwind, catch, or jump.
  Keel uses `?` and not a `try` keyword, so that humans and models do not read it as a try/catch
  block.
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
syntax.

- The runtime discards the state of an aborted task. The **parent** of the task, which is the task
  that started it, receives the abort as an ordinary value. The parent then decides the next step:
  restart the task, fail the request, or report the abort.
- When a task starts, it returns a handle. The exit of the task is a sum type. The parent uses a
  `switch` on it:

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

- **No abort is silent.** Sometimes a parent does not observe the exit of a child, for example
  after `go f()` discards the handle. Then an abort of the child goes to the parent, then to the
  parent of the parent, and up to `main`. At `main`, the process stops with the report.
  Supervision is optional. A loud stop is the default.
- The runtime records each abort report as an RCA trail: the task, the parent, the invariant, the
  source position, and the values.

**Poisoned locks.** Tasks share memory, as in Go. An abort during an update can leave shared data
half-changed after the state of the task is gone. Keel does what Rust does:

- If a task holds a mutex (or a read-write mutex) when it aborts, the mutex becomes **poisoned**.
- Each later `Lock()` returns an error, `Poisoned`. The code must handle this error like all other
  errors (§7):

  ```go
  mu.Lock()?                      // propagates Poisoned to the caller
  ```

- A task that repairs or rebuilds the data can call `mu.ClearPoison()`. `keel audit` lists each of
  these calls.
- Limit: poisoning protects only data that a lock guards. A change to shared data without a lock
  is a data race, and a data race is already a bug. The race detector runs in test builds to find
  data races.

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

- `x ?? fallback` gives an explicit default. Keel permits it, because the code states the default
  where it is used. But `keel audit` lists each `??`, the same as `ignore`.
- An API that "might not find" a value returns `?T`. Examples: a map lookup, `Find`, a search, and
  a parse that has no error.
- A value from a foreign function has the type `?T` until the code checks it.
- When code uses a `?T` incorrectly, the diagnostic hint shows the narrowing pattern above. It does
  not only say "type mismatch".

## 7b. Bounds

The compiler tries to prove that each index `xs[i]` is in range. A proof comes from one of these:

- the loop: `for i := range xs`
- an earlier check, narrowed by control flow: `if i < len(xs) { ... xs[i] ... }`
- a contract: `requires i < len(xs)`

| Outcome | Result |
|---|---|
| The index is proved in range | The compiler emits no runtime check. |
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
| `ensures expr` | functions, port methods | Postcondition: `result` is the return value |
| `invariant expr` | structs | Is true after each exported method returns |

### 8.1 What a contract can contain

Contract expressions are **pure**. They never do I/O, call ports, use channels or topics, change
data, or read the clock or a random source. Thus a program does the same thing when its checks are
on and when they are off. Within that rule, Keel permits three tiers:

| Tier | Adds | Example |
|---|---|---|
| 1 | Comparisons, arithmetic, `&&` `\|\|` `!`, `=>` (implies), field access, `len`, `result`, `old(x)` (the value on entry) | `ensures err == ok => acct.Balance == old(acct.Balance) - amt` |
| 2 | Calls to functions marked `pure`. The compiler verifies that these functions have no effects. | `requires IsSorted(a) && IsSorted(b)` |
| 3 | Quantifiers over collections: `all x in xs: …` and `any x in xs: …` | `requires all x in xs: !IsNaN(x)` |

```go
pure func IsSorted<T Ordered>(xs []T) bool { ... }

func Merge(a, b []int) []int
    requires IsSorted(a) && IsSorted(b)
    ensures  IsSorted(result) && len(result) == len(a) + len(b)
```

A runtime check of a quantifier costs O(n). `keel audit` lists each quantifier that release builds
still check. A leaf can turn these checks off.

How the compiler checks contracts:

- **Statically**, when the compiler can prove or disprove a contract.
- **At runtime** in debug builds and test builds. Release builds keep the checks by default. A leaf
  can turn them off.
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
- A `fake` block implements a port for tests. The compiler can generate a default fake from the
  port.
- `keel test Billing` builds and runs the tests of one leaf. It does not build the rest of the
  system.

## 10. Compiler intelligence

In addition to type checks, the compiler reports these problems as diagnostics with fix-its:

- **Near-duplicate code** in the project (R2). The diagnostic points to the existing function to
  reuse.
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
