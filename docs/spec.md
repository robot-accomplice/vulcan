# Vulcan: Language Specification (Draft 0)

> **Name:** Vulcan. Source files end in `.vulcan` ([decision 5](decisions/0005-names-and-reproducibility.md)).
> Drafts before 2026-10-20 used the working name Keel.
> Status: brainstorming draft, 2026-09-28. Nothing in this document is implemented.

## 1. Purpose

Vulcan is a compiled, garbage-collected, statically typed language. Humans and LLMs use it to write
software *together*.

The thesis of Vulcan: most failures in human/AI co-development come from missing structure. The
model cannot see the architecture. It invents APIs, it duplicates code instead of reusing it, and it
masks errors. The human cannot review what the model wrote. Vulcan puts that structure in the
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

**How Vulcan makes validation shorter (hypotheses).** Two experiments with a model writing and
reviewing code in Go and in Vulcan (2026-09-29) suggested where the gain is, and where it is not.
Their data is lost, the Vulcan side had no compiler, and each had one writer. Thus the statements
below are hypotheses, not findings. A compiler-based experiment must test them (development plan,
milestone 4):

- **Vulcan takes the architecture rules out of review.** A rule that the compiler enforces (which
  leaf may use which, where the network and databases are reached, what a port promises) no longer
  has to be checked by a person on every change, and no later change can quietly break it. In Go
  the same rules hold only while each writer chooses to follow them. §5.4d states which rules the
  compiler proves and which paths around them it only reports.
- **Vulcan rejects rule-breaking changes before review.** A change that breaks an enforced rule does
  not build, so a model that iterates against the compiler fixes it before a person sees it.
- **Vulcan does not make a thorough reviewer read less code, and it does not reduce defects in
  logic that no rule covers.** Order-state logic, races, and design choices still need review.

If these hold, the gain is a smaller and more reliable review: a reviewer checks the logic and
trusts the compiler for the architecture. [Decision 6](decisions/0006-evidence-protocol.md) tests
the second hypothesis (H-B) in two steps: whether the rules help when Go's static analyzers enforce
them (Q1), and whether Vulcan does better on the rules that neither those analyzers nor a Go
contract generator can enforce (Q2). It measures, but does not test, the review benefit of the
first hypothesis. For the third hypothesis, its bound N2 only checks that Vulcan does not add
defects. Each threshold was fixed before any run.

### 1.2 Terms

This spec uses each term below with one meaning only. The compiler, the tools, and the diagnostics
use the same terms.

| Term | Meaning |
|---|---|
| **System** | The whole software product. The root node of the architecture tree. |
| **Container** | A node that runs as one process. Most systems have one container, so they compile to one binary. For more containers, the project chooses one binary with a mode per container, or one binary per container (§11). (The C4 meaning. It is not a Docker container.) |
| **Group** | An optional node inside a container that groups leaves, for zoom levels in diagrams. |
| **Composite** | A system, container, or group. A section of `system.vulcan`, with no code. |
| **Leaf** | A `component` node: one flat directory of code. |
| **Library** | A node that holds shared types and reusable code, with no ports and no topics. Its package-level variables cannot be reassigned after startup. It can use an I/O resource but never acquire one. |
| **Gateway** | The one leaf in a container that talks to a given external system. |
| **External system** | Software outside the system, such as Stripe. Declared with `external`. |
| **Person** | A human who uses the system. Declared with `person`. For diagrams. |
| **Port** | An interface with contracts, declared once by the leaf that requires it. A provider satisfies it with an ordinary type, named at the wire. Inside a container, a port call is an ordinary function call. |
| **Topic** | A publish channel, `topic[T]`, for asynchronous events between containers. |
| **Remote port** | A method port that crosses containers, declared `needs remote port`: a synchronous call. Events between containers use topics, which are not ports. |
| **Publisher / subscriber** | A leaf that declares `publishes X topic[T]` / a leaf that declares `subscribes X topic[T]` and handles it. |
| **Handler** | The function, marked `handle`, that processes the messages of one subscribed topic. |
| **Wire** | A line in `system.vulcan` that connects a required port or a subscribed topic to a provider. |
| **Transport** | A node kind that carries topic messages and remote calls between containers, named on a wire with `via`. |
| **Dead-letter topic** | Where a subscription sends a message that it cannot handle. |
| **Acknowledge** | Mark a message as handled. It occurs when its handler returns `nil`. |
| **Error** | An ordinary value that reports a failure the code expects and handles. |
| **Contract** | A `requires`, `ensures`, or `invariant` clause. |
| **Audit** | The `vulcan audit` report of each escape hatch, override, and unproved claim. |

## 2. Design principles

| Principle | Meaning in Vulcan |
|---|---|
| Compiled, GC | Native binaries and a tracing GC. No manual memory management and no borrow checker. |
| Go ergonomics | Function bodies are Go, unchanged. Vulcan adds the architecture, contracts, and tools around them (§4, §6). |
| Competitive performance | A goal, with no current test: resource use and speed competitive with Go, Java, C++, and Rust. Today only the comparison with Go is measured (§11.3). |
| Mechanism, not policy | The language gives mechanisms and overridable defaults. Operational policy (delivery, retries, queues, build output) is the developer's decision. A default that the language sets must be one that a setting can change. |
| Compiler first | If the compiler can check a rule, the compiler checks it. Each rule that depends on convention, review, or runtime is a candidate to move into the compiler. This is the most direct way to make tasks 2 and 3 of §1.1 shorter. |
| Rust-grade compiler intelligence | Diagnostics explain *why* and propose fixes. The tools find duplication, error masking, and common Go faults, as warnings. |
| Imposed structure | One project layout, one formatter, and one location for each kind of thing. |
| Architecture is first-class | Systems, components, and ports are language constructs. The compiler enforces them. |
| Self-documenting, self-diagramming | The tools generate docs and architecture diagrams from the source. Nobody maintains them by hand. |
| Contracts are code | Preconditions, postconditions, and invariants are part of declarations. Reviewers can audit them. |
| Tooling teaches | The tools write every output for an LLM reader and a human reader. |
| OO "light" | Structs, methods, interfaces, and generics. No inheritance and no classes. |

## 3. Evidence behind the design

This table shows how LLMs fail when they write applications with humans, and what Vulcan does about
each failure. Appendix A lists the sources.

| # | Observed failure | Vulcan's answer | Section |
|---|---|---|---|
| R1 | Agents get lost in large codebases. Quality drops as the context grows. | Leaves are small, self-contained units with explicit ports. `vulcan context` emits a minimal context pack. | §5, §10 |
| R2 | Duplication is up 81%. Reuse (cross-file calls) is down 35%. | The compiler finds near-duplicate code. Ports in each `component.vulcan` and all wires in `system.vulcan` make the available capabilities easy to find. | §9 |
| R3 | Error-masking constructs are up 47%. | The tools warn on each unhandled or masked error, with a fix-it, and the audit lists them. | §6, §9 |
| R4 | Models invent packages (about 20% of the time) and APIs. | The compiler serves an exact symbol index that tools can query. There are no implicit imports. | §10 |
| R5 | About 94% of compile errors in typed languages are type errors. | Go's static types. The compiler's type information works on incomplete programs, so tools can guide generation by type. | §6, §10 |
| R6 | AI code is harder to review. It "looks right but isn't". | Contracts in the code that reviewers can audit, and diffs at the architecture level. | §7, §9 |
| R7 | Developers feel faster than they measure. | Fast verification: incremental builds and per-leaf tests. | §8 |
| R8 | A new language has no training data, so LLM quality drops steeply. | Syntax close to Go, so the model can use what it knows. A primer (`vulcan guide`) with a hard budget of 4,000 tokens; the full spec is reference. Tools that instruct the model. | §4, §10 |

## 4. Syntax

**Function bodies are Go, unchanged (§6).** A Go programmer, or a model trained on Go, reads and
writes Vulcan code as Go. Vulcan's changes are all around the code: the architecture, contracts, and
tests. This spec must justify each change and list it in this table:

| Addition to Go | Reason |
|---|---|
| Declaration files: `system.vulcan` (containers, groups, `person`, `external`, `uses`, every wire, `via` with transport settings, `config { }`), `component.vulcan` (`needs port`, `needs remote port` with `timeout`, `publishes`, `subscribes`, `gateway for`, `entry for`), `library.vulcan`, `transport.vulcan`. They can use `flag("...")` and constant `time.Duration` expressions | Architecture as a first-class construct (§5) |
| A leaf never imports another leaf; it uses a port, a topic, or a library. A port call is an ordinary function call | Enforced, legible boundaries (§5.3, §5.4) |
| Library rules: no ports or topics, package-level `var` set once, no resource-acquiring calls | Libraries do not hide state or I/O decisions (§5.4a) |
| Only gateway and entry leaves reach a network function | Enforced network access (§5.4b, §5.4c) |
| `topic[T]`, `Publish`, and `idempotent handle` functions | Events between containers (§5.5) |
| `limit` bounds on variable-size fields | A transport can require bounded messages (§5.5.1) |
| `requires`, `ensures`, `invariant`, `pure`, and `result`, `old(x)`, `=>` inside contracts; only its own methods write the fields of a struct with an invariant | Contracts in the code (§7) |
| Functions named `setup` and `startup`, found by their names as Go finds `init`; `main` is generated | The architecture decides how a process starts (§6) |
| `*_test.vulcan` files; `vulcan.Bind` and `vulcan.Published` in Go tests | Testability by design (§8) |

All other syntax and behavior follow Go. Appendix B gives the grammar, and Appendix F shows how each
addition becomes Go.

### 4.1 Keywords

Vulcan reserves no word that Go does not. Each new word is a keyword only where the grammar expects
it: at the start of a declaration (`needs port`, `pure func`, `handle X(`), after a
field type (`limit`), after a signature or a struct (`requires`, `ensures`, `invariant`,
`timeout`), inside a contract (`result`, `old`), and inside a declaration file. Everywhere else it can be an identifier.

The new words: `system`, `container`, `group`, `component`, `library`, `transport`, `port`,
`remote`, `needs`, `requires`, `ensures`, `invariant`, `pure`, `result`, `old`, `uses`, `via`,
`topic`, `publishes`, `subscribes`, `handle`, `idempotent`, `limit`,
`gateway`, `entry`, `person`, `external`, `timeout`, `config`, `flag`, `default`, and `max`.

```go
port := 8080                   // an identifier
needs port Payments { ... }    // keywords: they start a declaration
```

`vulcan guide` prints the list.

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
| `system` | `system.vulcan` | containers, `person` and `external` declarations, all wires | no |
| `container` | a section of `system.vulcan` | groups and leaves | no |
| `group` | a section of `system.vulcan` | leaves; a label for diagrams only | no |
| `component` | `component.vulcan` in the leaf directory | the boundary of the leaf | yes, in the same directory |
| `library` | `library.vulcan` in the library directory | shared types and reusable code | yes, but no ports, topics, or mutable state (§5.4a) |

### 5.2 Layout

- **`system.vulcan`**, at the root, is the architecture file. It declares the containers and groups,
  the leaves in each, the people, the external systems, and **every wire**.
- **Each leaf is a directory** with one `component.vulcan` and its source files. `component.vulcan`
  declares the **boundary** of the leaf: each port that it needs, each topic that it publishes or
  subscribes to, and its `gateway for` or `entry for` line.
- **Directories mirror containers and leaves**: `api/orders/`, `worker/picking/`. Groups are only
  labels in `system.vulcan`, so they are not directories and not part of any path. The compiler
  checks that the directories and `system.vulcan` agree.
- Libraries are in `lib/`. Transports are in `transports/`, one directory each, with a
  `transport.vulcan` declaration.

Thus each thing has one location: the wires in `system.vulcan`, a boundary in its
`component.vulcan`, and code next to its boundary.

### 5.3 Declarations

**A port is declared once, by the leaf that requires it.** This is the Go idiom: the consumer
defines the interface where it uses it. A leaf declares its boundary in `component.vulcan`: each
port that it requires, with its methods and contracts.

```go
// api/orders/component.vulcan
import "shop/lib/money"

component Orders {
    // Checkout takes payment for a confirmed order.
    needs port Checkout {
        func Pay(customer money.CustomerID, amount money.Money) (money.Receipt, error)
            requires amount > 0 && amount < money.MaxOrder
            ensures  err == nil => result.Amount == amount
    }
}
```

**A provider satisfies a port with an ordinary type.** It needs no keyword and no declaration: any
exported type of a leaf whose method set matches the port can satisfy it, as with a Go interface.
The provider never names its consumers.

```go
// api/billing/payer.vulcan (ordinary code)
// Payer charges a customer's stored payment method.
type Payer struct { ... }

func NewPayer() *Payer { return &Payer{} }

func (p *Payer) Pay(customer money.CustomerID, amount money.Money) (money.Receipt, error) { ... }
```

If two consumers need different methods with the same name, the provider exports two types, for
example `Payer` and `SubscriptionPayer`, as in ordinary Go.

A consumer does not know which leaf serves it. A provider does not know which leaves use it. Only
`system.vulcan` knows both.

**Inside a leaf, each required port is a package-level name**, the name of the port itself. Code
calls it like a Go package: `Checkout.Pay(c, amt)`. The wiring binds the name at startup, and the
binding cannot change after that. A test binds fakes to the same names (§8). A leaf appears once in
the tree; reuse of code goes through libraries (§5.4a).

**A port call is an ordinary function call.** Vulcan does not invent a new way for components to
communicate. Inside one container, a port call runs on the caller's own task, usually the same
thread, with no queue, no copy to bytes, and no hidden goroutine. Arguments pass by value or by
reference, exactly as in Go. On the Go backend, a port call is a Go interface call: one dynamic
dispatch, because each leaf compiles alone against its own port declaration (§11.2). The generated
wrapper that checks the contract calls the provider directly (Appendix F). The LLVM backend can use
the fixed wiring to make it a direct call. What Vulcan adds at the boundary is **enforcement and
instrumentation of the contract**: the port signature, its `requires` and `ensures`, and the record
of the boundary in diagrams and in the audit. Topics and transports (§5.5) exist only for separate
processes.

**The architecture file wires the leaves.** A wire connects a required port to a providing type.
Inside a container section, a wire names leaves by name. A wire between containers names each leaf
by its path: `Container/Leaf`. Groups are not part of paths, so regrouping for a diagram never
changes a wire. Every wire reads `needer: supplier`: the left side needs what the right side
supplies, a method for a port and messages for a topic. `vulcan query` lists the subscribers of a
topic in one command.

```go
// system.vulcan
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
`*Payer` satisfies the port, and the wire names a constructor that returns one. A wire that names a
type `T`, for example `Billing.Clock: SystemClock.Wall`, provides `new(T)`, a `*T` with the zero
value. Such a wire takes no settings (K0604). A constructor's first result must be an exported
concrete type or a pointer to one, so that the generated `main` can name it (K0604).

**Leaf names are unique in the system**, also across containers (K0121). Thus a runtime name such
as `Orders.Placed`, and the leaf argument of each command, name one leaf.

- **Dependencies** come through the provider's own required ports, so a constructor needs no
  parameters for them.
- **Configuration** comes through one optional settings struct, filled at the wire from literals or
  startup flags, as for transports: `StripeGateway.NewCards{ APIKey: flag("stripe-key") }`, which fills the exported settings struct
  of `NewCards`. A leaf's own settings, for its `setup` and `startup` functions (§6), come only
  from the container list of `system.vulcan`: `Web{ Addr: flag("web-addr", default: ":8080") }`.
  They are separate from the settings of any constructor in the leaf.
- **A constructor can fail.** It can return `(T, error)`. A failure stops the process at startup
  with a report.
- **One instance for each constructor and settings value, in each process.** Two wires to the
  same constructor with the same settings share one instance. Two settings are the same when their
  source text is identical after `vulcan fmt`.
- **A secret stays out of source.** A settings field with the tag `vulcan:"secret"` accepts only a
  `flag`. A literal in it is a compile error (K0605).

**Flags.** `flag("name")` in a declaration file is a value that the process reads when it starts
(§6, step 1).

- **Source.** The value comes from `--name=value` on the command line. If that is absent, it comes
  from the environment variable with the same name in upper case, with `-` and `.` changed to `_`
  (`stripe-key` is `STRIPE_KEY`). Thus a credential can stay out of the command line. Two flag
  names that give the same environment variable are a compile error (K0602).
- **Per container.** A container reads and requires only the flags that its own wires, leaves, and
  transports use, also when one binary holds every container.
- **Type.** The type of the field that the flag fills is the type of the flag: a string, a
  number, a `bool`, or a `time.Duration`. Two uses of one flag name with different types, defaults,
  or maxima are a compile error (K0602).
- **Options.** `default: v` gives a value when neither source has one. A flag without a default
  is required. `max: n` bounds a size, as §5.5.1 requires for queue settings.
- **Audit.** `vulcan audit` lists each flag, with its type, its default, and each setting it fills.

**Checks at each wire.**

1. **Method set.** The providing type must have each method of the port, with an identical
   signature. It can have more methods.
2. **One contract.** The contract of the port applies at every call through the port: the compiler
   checks `requires` before the call and `ensures` after it. If the providing method also declares
   its own contract, the compiler checks that the port's `requires` implies the method's `requires`,
   and that the method's `ensures` implies the port's `ensures`. If it disproves an implication, it
   is a compile error (K0109). If it cannot prove one, `vulcan audit` lists it; no extra check is
   generated, because each side already checks its own clauses at each call.

If no type has the right methods, write an adapter: an ordinary leaf with a type that satisfies the
port, and that calls another port. Vulcan has no special adapter construct.

Vulcan does not wire external systems with ports. The language does not control them, and they use
their own protocols. Instead, one **gateway leaf** in each container talks to each external system
(§5.4b). The compiler draws each relationship with a person or an external system from the entry
and gateway declarations. A `uses` line is needed only for a relationship that no declaration
shows.

### 5.4 Rules the compiler enforces

1. **Ports only.** Code in a leaf can use another leaf **only** through a port that the leaf
   needs, a topic that it publishes or subscribes to, or an imported library (§5.4a). Any other reference to another leaf is a
   compile error.
2. **Code only in leaves and libraries.** `system.vulcan` holds no code. A leaf directory holds no
   child leaves.
3. **Container boundaries.** A wire between leaves in different containers must be a topic or a
   `needs remote port` (§5.5). A method port inside one container must not be `remote`, and a
   topic cannot connect two leaves of one container (§5.3: calls inside a container are ordinary). A group is only a label for diagrams: it
   adds no boundary.
4. **Complete wiring.** A `deadletters:` setting wires a `DeadLetter[T]` topic, and rule 3 does not
   apply to it. A subscribed `DeadLetter[T]` topic needs at least one `deadletters:` route.
   Otherwise, `system.vulcan` wires each required port and each subscribed topic of each leaf
   exactly once. A published topic is never wired by itself (§5.5).
5. **Satisfied ports.** Each wire names a providing type whose method set matches the port.
6. **Compatible wiring.** Each wire passes the port-compatibility check in §5.3.
7. **Boundary ownership.** A leaf declares each port that it requires, once, in its own
   `component.vulcan`. The code and the boundary of a leaf never name another leaf. Only
   `system.vulcan` names two leaves together, in a wire.
8. **No cycles.** The graph of wires between the leaves of one container has no cycles.
9. **Private types.** A type is private to its leaf. Only `system.vulcan` can name an exported type
    of a leaf, in a wire. A type that crosses a boundary (in a port signature, a topic message, or a
    remote method) must be a predeclared Go type, a type of the Go standard library, or a type from
    a library (§5.4a). In a topic message and in a remote signature, each named type must be
    declared in a library, or be `vulcan.DeadLetter[T]` for a library type `T`; predeclared types,
    `error`, `time.Time`, and `time.Duration` are also allowed. Thus each container compiles the
    same type (K0112). This costs a
    library for each shared type, which is the price of leaves that never import each other. The
    compiler reduces it: when a boundary uses a leaf-private type, the fix-it moves the type
    to a library (new or existing) and updates the imports.
10. **Leaf size (audit).** `vulcan audit` reports the size of each leaf: its `component.vulcan` and
    its non-test source, in non-blank, non-comment lines. A leaf over the project's `context_budget`
    (set in the `config { }` section of `system.vulcan`) gets a warning, with a hint that tells
    where to split it. It is not a compile error, so one more line never forces a split.
11. **Fixed names.** A port name and a topic name are bound once (§6). Code cannot assign to one
    or take its address; only `vulcan.Bind` in a test changes a port binding (K0116).
12. **No `init` in leaves.** A leaf cannot declare a Go `init` function, because it would run
    before the ports are bound. For the same reason, the initializer of a package-level variable
    in a leaf cannot reach a port call or a publish. Leaf setup goes in a constructor, in `setup`,
    or in `startup` (§6). A library can declare `init`, as in Go (K0117).
13. **No `linkname`.** A leaf or a library cannot contain a `//go:linkname` directive (K0122).
    The directive lets code call an unexported function of another package with no identifier
    that names it, so rule 1 could not see it. An imported Go package can still use it, as the
    standard library does.

### 5.4a Libraries

Some types cross a wire. For example, `Money` is in the port signatures on both sides. Such a type
must be the *same* type on both sides, so it cannot belong to either leaf. It goes in a
**library**. A library is a node that holds shared types and reusable code. It is not part of the
architecture at runtime.

A library is a Go package. Its directory holds one `library.vulcan`, which declares it, and ordinary
Go code files. Code imports it by its Go import path, as any Go package, and uses its package name
as the qualifier.

```go
// lib/money/library.vulcan
// Money represents amounts in minor currency units.
library money {
}

// lib/money/money.vulcan
package money

type Money int64

// api/billing/billing.vulcan
package billing

import "shop/lib/money"      // the module path from go.mod, then the directory

func total(xs []money.Money) money.Money { ... }
```

A `component.vulcan` imports libraries the same way, at its top, so its port signatures can use
their types:

```go
// api/orders/component.vulcan
import "shop/lib/money"

component Orders {
    needs port Checkout {
        func Pay(c money.CustomerID, amt money.Money) (money.Receipt, error)
    }
}
```

**Code files are Go files.** Each starts with a `package` clause that names its directory. A
function with a contract writes the contract lines between its signature and its body:

```go
func Withdraw(a *Account, amt Money) (Money, error)
    requires a != nil && amt > 0
    ensures  err != nil => a.Balance == old(a.Balance)
{
    ...
}
```

Rules:

1. A library has **no ports and no topics**. A package-level `var` in a library is set once, at
   startup, and never assigned again; the compiler rejects each later assignment. "At startup" means
   in the variable's initializer or in an `init` function of the same library, where each variable
   is assigned at most once. The compiler does not track changes through a method or a reference
   (`hits.Add(1)`, `cache[k] = v`), so a library can still hold state that two leaves share. `vulcan
   audit` lists each library `var` whose value can change through a method or a reference, so that a
   reviewer can check it (§5.4d, path 2). Strict mode rejects such a `var` (K0210).

   ```go
   // lib/crc
   var table = buildTable()                         // set once, at startup
   var ErrNotFound = errors.New("not found")

   table = nil                                      // compile error: assigned after startup
   ```
2. **A library enables I/O. It does not decide on I/O.** A library can read, write, or query
   through a handle that its caller gives it. One exception: `(*http.Client).Do` on a client that
   the caller gives is still a network function, because the request chooses the target.
   A library cannot *acquire* a resource: it cannot
   open a file, dial the network, open a database, or create a client. A library that reaches an
   acquiring function or a network function (§5.4c) is a compile error. Thus each I/O decision is in a leaf, where the
   diagram and the reviewer see it.

   ```go
   // lib/csvutil: allowed. The caller chose the file.
   func Parse(r io.Reader) ([]Row, error) { ... }

   // lib/orderlogic: compile error. The library chose to open a database.
   func Reprice(id OrderID) error {
       db, err := sql.Open("postgres", config.DSN)
       ...
   }
   ```
3. A library can import other libraries. It cannot import leaves or composites. Library imports
   have no cycles.
4. Any leaf can import any library.
5. `vulcan diagram` shows libraries as shared dependencies. `vulcan audit` lists the leaves that use
   each library.

Libraries compile before the leaves that import them. Leaves compile against the library
declarations only (§11.2).

### 5.4b Gateways to external systems

A **gateway leaf** is the only leaf that talks to an external system. It turns the protocol of the
external system into an ordinary type that satisfies a port. Other leaves require that port, and
they contain no network code.

```go
// api/stripegateway/component.vulcan
// StripeGateway charges cards through Stripe.
component StripeGateway {
    gateway for Stripe
}
// api/stripegateway/cards.vulcan: type Cards, with a Charge method that calls Stripe

// api/billing/component.vulcan
component Billing {
    needs port CardNetwork { func Charge(...) (cards.Receipt, error) } // no network code
}

// system.vulcan
Billing.CardNetwork: StripeGateway.NewCards{ APIKey: flag("stripe-key") }
```

Rules:

1. A container that talks to an external system has exactly one gateway leaf for that system. The
   gateway declares `gateway for <External>`, and `<External>` must be an `external` declaration in
   the system. One leaf can declare any number of `gateway for` and `entry for` lines, for
   example a Stripe gateway that also receives Stripe's webhooks (`entry for Stripe`).
2. Network access follows the single rule in §5.4c, rule 1. A gateway can use any Go package,
   including an SDK for its external system or `net/http`.
3. The compiler generates the `uses` relationships in diagrams from the gateway declarations, so
   nobody draws them by hand. It does not check which target a gateway reaches (§5.4d, G11).

**Entry leaves.** Traffic that comes **into** the system, for example HTTP requests from a `person`,
enters through an **entry leaf**. It is the mirror of a gateway:

```go
// api/web/component.vulcan
component Web {
    entry for Customer          // serves requests from the person Customer
    needs port Checkout { ... }
}
```

- An entry leaf declares `entry for <Person or External>`, and it can reach network functions to
  serve requests, under the one network rule (§5.4c, rule 1).
- In ordinary mode, the compiler does not tell serving from dialing. `vulcan audit` lists each
  entry leaf and each gateway, and the network functions that each reaches. In strict mode, an
  entry leaf can reach only the serve functions of Appendix D, and a leaf that dials must be a
  gateway (K0125, §5.4d).
- The compiler draws the `uses` relationship for each entry leaf, as for gateways.

Why a gateway (§1.1): each rule about an external system goes in one leaf. A reviewer reads all
traffic to that system in one location. Tests of other leaves fake a port with contracts, not HTTP.
The compiler rejects each static path to a network function from any other leaf (§5.4c, rule 1).
§5.4d lists the paths that this check does not cover, and strict mode rejects most of them.

### 5.4c Go packages

Vulcan code is Go code, so a leaf or a library imports Go packages directly, as Go does. Importing a
package is never an error. Two architecture rules apply to the functions that code **reaches**.

Code reaches a function when it calls the function, uses it as a value, or calls a function of an
imported package from which the function is reachable through static calls. Two more steps close
the paths through interfaces:

- **A conversion to an interface** of a value of type `T` reaches each method of `T` that is in the
  interface's method set. Thus `var d Dialer = &net.Dialer{}` reaches `(*net.Dialer).DialContext`.
- **A package-level variable of interface type**, when code reads it, reaches each method of the
  dynamic type of its initializer. Thus a read of `http.DefaultTransport` reaches
  `(*http.Transport).RoundTrip`.

A call through an interface value or a function value is not followed beyond these steps. The
compiler computes the reachability of each imported module once for each version, and caches it.
Each diagnostic shows the call path.

1. **Network access is decided by the network functions.** Appendix D lists the **network
   functions**: the Go functions that choose a network target or a process (dial, listen, send an
   HTTP request, open a database, run a command). **Only gateway leaves and entry leaves can reach
   a network function.** This is the one rule for network access; other sections refer to it. It
   covers an SDK that dials with `http.DefaultClient`, and a database driver opened with
   `sql.Open`. A package that imports `net` only for a type, for example `crypto/x509`, reaches no
   network function, so any leaf or library can use it. A library that receives a `*sql.DB` can
   query through it, because its caller chose the database. An embedded database driver (for
   example SQLite) reaches no network function, so the project chooses whether to model it as an
   `external` system.
2. **Acquiring a resource is a leaf's decision.** Appendix D also lists the **acquiring
   functions**: the Go functions that choose a file or another local resource by name. Only leaves
   can reach an acquiring function; a library cannot (§5.4a, rule 2).

**The lists are defaults.** The project can add to either list (`config { network: [...],
acquiring: [...] }`) and remove from it (`allow_network: [...]`, `allow_acquiring: [...]`).
`vulcan audit` lists each change. A method is written `(*net/http.Client).Do`, and a function
`os.Open`.

**Limits.** Code that uses `syscall`, `unsafe`, `reflect` calls, or cgo can reach a resource that
the analysis does not see. `vulcan audit` lists each such use, and each call to a listed function
(§5.4d, path 6). Strict mode rejects them in leaves and libraries (K0124).

**Versions.** A Vulcan project uses Go modules (`go.mod`) to pin the versions of Go packages.

### 5.4d Guarantees and modes

The rules of §5.4 to §5.4c prove some properties, and leave known paths around others. This
section lists both ([decision 2](decisions/0002-architecture-enforcement.md)). Docs, diagrams, and
tool output state an architecture property only as this section does.

**Paths around the rules.** Each path compiles in ordinary mode. `vulcan audit` lists each one that
it finds, in its section "Unchecked paths", and `vulcan check` prints their number.

1. A function value or an interface value crosses a port. The callee can then run the caller's
   code, and a gateway can hand out a function that dials any target (§5.4c follows a function
   value only in the two steps it names).
2. A library `var` changes through a method or a reference, so two leaves share it (§5.4a rule 1).
3. Two leaves share the package state of an imported Go package, for example the environment,
   `http.DefaultServeMux`, or a metrics registry.
4. An entry leaf dials, because ordinary mode does not tell serving from dialing (§5.4b).
5. A gateway connects to a target other than its external system (§5.4b rule 3).
6. Code uses `syscall`, `unsafe`, `reflect` calls, or cgo (§5.4c, Limits).
7. Two leaves acquire the same file, environment variable, or table.
8. `allow_network` or `allow_acquiring` removes a function from a list (§5.4c).

**Guarantees.** *Proved* means that the compiler checks the property on every build, with no
listed path around it. *Reported* means that `vulcan audit` lists each path. *Out of scope* means
that no static check can decide it.

| # | Property | Ordinary mode | Strict mode |
|---|---|---|---|
| G1 | A leaf's code and boundary never name another leaf (rules 1, 7, 13) | Proved | Proved |
| G2 | Each required port and subscribed topic is wired once, to a matching method set (rules 4, 5) | Proved | Proved |
| G3 | The wires of a container have no cycles (rule 8) | Proved for wires; calls can go back through path 1 | Proved for wires and calls (K0123) |
| G4 | Types that cross a boundary come from libraries or the standard library (rule 9) | Proved | Proved |
| G5 | Only gateway and entry leaves reach a network function (§5.4c rule 1) | Proved for static reach; paths 1, 4, 6, 8 reported | Paths 1, 4, 6 rejected (K0123, K0125, K0124); path 8 reported |
| G6 | Only leaves reach an acquiring function (§5.4c rule 2) | Proved for static reach; paths 6, 8 reported | Path 6 rejected (K0124); path 8 reported |
| G7 | Leaves share program memory only through ports and topics | Reported (paths 2, 3) | Path 2 rejected (K0210); known global mutators rejected (K0211); other package state reported |
| G8 | A port contract holds at every call through the port, fakes included (§5.3, §8) | Proved; unproved implications listed (K0109) | Same |
| G9 | Each external system has one gateway in each container (§5.4b rule 1) | Proved | Proved |
| G10 | Containers talk only through topics and remote ports (rule 3) | Proved for wires; paths 4, 5 reported | Path 4 rejected (K0125); path 5 out of scope |
| G11 | A gateway talks only to its own external system | Out of scope | Out of scope |
| G12 | Leaves share no resources outside the program (path 7) | Out of scope | Out of scope |

**Ordinary mode** is the default. It is every rule of §5.4 to §5.4c, with the paths reported.

**Strict mode.** `config { strict: true }` turns it on for the whole system. `strict_exempt:
["Web"]` keeps the named leaves in ordinary mode. `vulcan audit` lists each exempt leaf, and
`vulcan doc` shows the mode of each leaf. Strict mode adds these errors, in each leaf that is not
exempt and in each library:

1. **No behavior across a boundary (K0123).** A port signature cannot carry a function type, a
   channel type, or an interface type, other than `error`, `context.Context`, and the interfaces
   in `config { strict_interfaces: [...] }`. Thus a call through a port never runs the caller's
   code.
2. **No escape hatches (K0124).** Code cannot import `unsafe` or `syscall`, use cgo, or call
   `reflect.Value.Call`, `reflect.Value.CallSlice`, `reflect.Value.Method`, or
   `reflect.Value.MethodByName`. Imported Go packages are not checked for this; calls through
   `reflect` inside them stay reported.
3. **Entry leaves serve, gateways dial (K0125).** An entry leaf can reach only the serve
   functions of Appendix D. A leaf that reaches a dial function must be a gateway. One leaf can be
   both.
4. **Immutable library state (K0210).** The type of a library `var` holds no pointer, slice, map,
   channel, function, or interface. The exceptions are `error` values and the types in
   `config { strict_shared: [...] }`, for example `*regexp.Regexp`.
5. **No known global mutators (K0211).** Code cannot reach a global mutator of Appendix D, or a
   function in `config { mutators: [...] }`.
6. **One instance per constructor (K0607).** Two wires that name one constructor of the leaf with
   different settings are an error, not a warning: `main` would make a second instance, such as
   a second database, and nothing in the code would show it.

`vulcan audit` lists each entry of `strict_interfaces`, `strict_shared`, and `mutators`.

**What each mode lets the docs claim.**

- *Ordinary mode:* each dependency between leaves in the source goes through a declared port,
  topic, or library. Only gateway and entry leaves reach a network function through static calls.
  `vulcan audit` lists each path that these checks do not cover.
- *Strict mode:* each dependency between leaves, in the source and in calls, goes through a
  declared port, topic, or library. Only gateways dial, and only entry leaves serve. No leaf
  shares program memory with another leaf except through a port or a topic, apart from the
  imported-package state that `vulcan audit` lists. Exempt leaves are listed.
- *Neither mode* claims where a gateway connects (G11), or that leaves share no files, tables, or
  environment (G12).

### 5.5 Crossing containers

> **Deferred.** Release 1 does not support §5.5 to §5.5.3: it rejects topics, remote ports,
> transports, and a second container with K0009 ([release 1](release-1/syntax.md)). These
> sections are a design, not a commitment. The development plan, milestone 5, says when they can
> come back, and audit 2 (H8, H9, M7) lists known problems in them.

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
// api/orders/component.vulcan. Topics are boundary entries, not ports.
import "shop/lib/events"         // message types come from a library (§5.4, rule 9)

component Orders {
    publishes  Placed  topic[events.OrderPlaced]
    subscribes Shipped topic[events.ShipmentSent]
}

// worker/picking/component.vulcan
import "shop/lib/events"

component Picking {
    subscribes Placed  topic[events.OrderPlaced]
    publishes  Shipped topic[events.ShipmentSent]
}

// In system.vulcan. Each subscriber names its source; the wire names the transport and settings.
Worker/Picking.Placed: Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }
Analytics/Ingest.Placed:           Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }
Api/Orders.Shipped: Worker/Picking.Shipped via direct{ deadletters: Worker/Alerts.DeadShipped }
```

**Why publish: it is the loosest coupling possible.** The only job of the publisher is to publish.
The publisher does not know, and must not care, if a subscriber reads its messages. It does not know
how many subscribers there are, or when they read.

These rules are part of the language, for every transport:

- **Publish, do not wait for a reader.** `Placed.Publish(msg)` returns nothing. It gives the
  message to the transport, once for each subscriber wire, and returns. A message that breaks its
  invariant or a `limit` panics at the publish, as a failed contract does; a transport that cannot
  accept the message panics too. The panic is a default, not a fixed policy: the transport's
  settings decide when it cannot accept a message (for example `durable` and `whenFull` on
  `direct`), and a developer who wants a publish never to fail chooses a transport and settings
  that never refuse. Subscribers cannot make a publish fail. What happens when a queue is full is
  the policy of the
  transport. **Under the default `block` policy of `direct`, a publish waits when a subscriber's
  queue is full, so a slow subscriber can slow the publisher.** A developer who wants the publisher
  never to wait chooses a drop policy for that wire, or bounds the wait with `maxWait`. A publish
  that is still waiting when the process stops panics.
- **Zero, one, or many subscribers.** `system.vulcan` wires topics one at a time, from the
  subscriber side. Each subscribed topic is wired exactly once, to one published topic of the same message type. A
  published topic is never wired by itself: its subscribers are the subscribed topics wired to it, zero or
  more. Thus a new subscriber adds one wire, and the wiring of the publisher does not change.
- **An unread topic causes a warning.** A topic with zero subscribers compiles, with a warning. A
  publisher can exist before its subscribers, but an unread topic must not go unnoticed.
- **A topic has no reply.** A reply would bind the publisher to a subscriber. For a request and
  response, use a remote method port (§5.5.3). If a publisher needs to know what happened later, it
  subscribes to a separate event that the other side publishes. `Shipped` in the example above is
  such an event.
- **A state change and a publish are not atomic.** In `orders.Save(o); Placed.Publish(m)`, the
  process can stop between the two lines, and the event is then lost. Keeping them together is the
  developer's decision: for example, the transactional outbox pattern (write the event to the same
  database transaction, and publish it from there). Vulcan does not hide this risk.
- **A read model is an alternative to a synchronous read.** The owner publishes change events. The
  reader subscribes, keeps a local copy, and answers queries from that copy. This is
  "event-carried state transfer". Reads can be slightly stale, and each reader stores its own copy.

  ```go
  // Worker owns stock and publishes each change.
  Stock.Publish(events.StockChanged{SKU: s, Qty: q})   // topic Stock carries StockChanged

  // Api keeps a read model, and answers GET /stock/42 from it.
  idempotent handle Stock(m events.StockChanged) error {
      stock.Set(m.SKU, m.Qty)
      return nil
  }
  ```

- **No raw network calls between containers.** A leaf cannot call another container over the network
  by hand, because only gateway and entry leaves can reach the network (§5.4c, rule 1). `vulcan
  audit` lists each `external` system with its gateway, so a reviewer can find an `external` that is
  really a container of the same system.
- **Ownership.** Each side owns only its end of each topic. Only message values cross, as copies.
  The two sides share no memory and no references.
- **Only topics and remote method ports cross.** The compiler rejects a wire across containers for
  a method port that is not declared `needs remote port`.
- **Contracts** on message types (struct `invariant`s) are checked when a message is published and
  again when it is received. A topic message type is a struct. Message types must be
  serializable: they contain only `bool`, integers, floats, `string`, `[]byte`, `time.Time`,
  `time.Duration`, arrays, slices, maps whose keys are integers, `bool`, or `string`, and structs
  of these. A pointer, an interface, a function, a channel, a topic, another map key type, or a
  recursive type is a compile error (K0305). §5.5.2 gives the encoding of each type.
- **A subscriber failure never reaches the code of the publisher.** A handler error and a handler
  panic go back to the transport, which applies its policy (§5.5.1). A received message that fails
  its invariant or a `limit` cannot succeed on a retry, so it goes to the dead-letter route at
  once.
- **Idempotency is declared.** A transport states, in its interface, whether it can deliver a
  message more than once. If it can, a subscriber's handler must be marked `idempotent`. Without
  the marker, the code does not compile. The compiler cannot prove the claim, but the marker makes
  a hidden assumption visible. It appears in `vulcan context`, `vulcan doc`, and `vulcan audit`.

#### 5.5.0 Publishing and handling

**Publish** is a method on a published topic. It is not Go's `<-`, because a topic does not behave
like a `chan`:

```go
Placed.Publish(events.OrderPlaced{ID: id, Total: t})
```

**A handler** is a function bound to one subscribed topic with `handle`. It can take a
`context.Context` first, `handle Shipped(ctx context.Context, m events.ShipmentSent) error`; the
runtime cancels it when the process stops (§6).

```go
idempotent handle Shipped(m events.ShipmentSent) error {
    if err := orders.MarkShipped(m.OrderID); err != nil {
        return err      // not acknowledged: the transport retries
    }
    return nil          // the message is acknowledged here
}
```

- Each subscribed topic of a leaf has exactly one handler. Code cannot iterate over a subscribed topic.
- The runtime runs each message in its own goroutine and recovers a panic in it (§6). By default, it runs the handler
  of one subscribed topic for one message at a time. A wire setting, for example
  `via direct{ concurrency: 8, deadletters: Worker/Alerts.DeadPlaced }`, allows more; shared state then needs a lock. At most
  `concurrency` handlers run at once for one subscribed topic; other messages wait in the
  transport's queue, which its capacity bounds. The language promises no order. With concurrency
  1, a handler sees messages in the order in which the transport delivers them.
- **The acknowledgement point is the return of the handler.** If the handler returns `nil`, the
  message is acknowledged. If it returns an error or panics, the message is not acknowledged, and
  the transport applies its policy: for example, retry and then dead-letter, or discard. Thus code
  cannot forget to acknowledge a message, and it cannot acknowledge a message too early.
- **A contract failure in a handler is not retried by default.** A contract failure is a bug, so
  a retry usually fails again and repeats the effects that the handler made before the check.
  When a handler panics with a `*vulcan.ContractFailure`, the runtime sends the message to its
  dead-letter route at once, with the failure report in `DeadLetter.Error`. A transport setting,
  `retryContractFailures`, applies the normal retry policy instead. A dead-letter handler that
  fails a contract is counted and reported, with no retry (§7.4).
- **A delivery ID.** `vulcan.DeliveryID(ctx)` returns an ID of the message for one subscriber wire.
  It stays the same across redeliveries of that message, and it is different for each message. A
  transport that can redeliver must supply it. A handler uses it as an idempotency key, stored in
  the same transaction as its effects (§7.4).

#### 5.5.1 Transports

A transport carries topic messages and remote calls between containers. Transports matter only for
a system with more than one container. Inside one container, leaves use method ports, and nothing
crosses a network.

**The transport interface.** The language defines one interface. A transport implements it and
declares:

- **A typed settings struct.** Each wire passes settings with `via name{ ... }`. The compiler
  type-checks the settings like any struct literal. Tags on its fields tell the compiler more
  (Appendix C): `vulcan:"required"`, `vulcan:"topic"` (topic wires only), `vulcan:"call"` (call
  wires only), `vulcan:"size"` (needs `max` from a flag, and counts in the memory limit), and
  `vulcan:"secret"`. `required` together with `topic` or `call` means required on that kind of wire
  only. A field of type `transport.Route` is a dead-letter route. A missing required setting is a
  compile error.
- **Setting names.** A setting's name is the field's name with its first letter in lower case
  (`Capacity` is `capacity`), or the name in a `vulcan:"name=..."` tag. An enum value is written the
  same way: `dropOldest` is the exported constant `DropOldest` of the field's type.
- **Process settings.** A second struct holds the settings of the transport for the whole process,
  for example its listen address. Each field comes from the flag `--transport.<name>.<setting>`,
  with the same naming rule.
- **Its guarantees, as a constant composite literal that the compiler reads.** Whether the
  transport can deliver a message more than once (which makes `idempotent` required), whether it
  keeps order, whether it requires bounded messages (`limit`), and whether it supports calls.
- **Its policy.** Durability, retries, dead letters, capacity, full-queue behavior, expiry,
  tracing, and delivery guarantees belong to the transport, not to the language.

**The developer chooses.** Each wire names its transport and its settings. The `config { }` section
of `system.vulcan` can set a default transport and default settings. A default setting tagged
`topic` applies only to topic wires, and one tagged `call` only to call wires. A required setting
cannot come from `config`. Addresses and credentials should come from startup flags, and a `secret`
field accepts nothing else (§5.3). `vulcan audit` lists the transport and the settings of each wire.

**The built-in transport, `direct`.** Vulcan ships one transport, so that a small system with two
containers needs no broker. It sends from the publisher straight to each subscriber. Its package is
`github.com/robot-accomplice/vulcan/vulcan/transport/direct`, and it needs no `transport`
declaration. It redelivers, keeps no order, requires bounded messages, and supports calls.

- **One queue for each subscriber wire.** The queue is in the publisher's process, so a publish
  does not need the subscriber to be up. A slow or failing subscriber fills only its own queue.
- **Settings of a topic wire** (each tagged `topic`; a call wire takes no settings):
  `concurrency` (default 1), `capacity` (default 10,000, for each
  subscriber wire), `whenFull` (`block`, `dropOldest`, `dropNewest`; default `block`), `maxWait`
  (the longest a `block` publish waits; default none), `ttl` (default none: no expiry), `retries`
  (default 5), `backoff` (default 1 s) and `maxBackoff` (default 5 min) for the delay before each
  retry, which doubles from `backoff` up to `maxBackoff`, with random jitter of up to 10%,
  `deadletters` (a route; required), and `durable` (default `true`, which keeps queues on disk so
  that no message is lost across a restart), `retryContractFailures` (default `false`: a contract
  failure in a handler goes to the dead-letter route at once), and `trace` (default `false`). A
  value can come from
  a startup flag:
  `capacity: flag("orders-cap", max: 10_000)`. A flag that sets a size must declare a maximum, and
  the compiler uses the maximum to calculate the memory limit.
- **Process settings:** `--transport.direct.listen` (the address this container serves; default
  `:7400`), `--transport.direct.peers` (each other container's address,
  `worker=10.0.0.7:7400,api=10.0.0.6:7400`), and `--transport.direct.dir` (the directory of durable
  queues; default `./vulcan-queues`).
- **Delivery:** at least once, unordered. Thus `idempotent` is required on its handlers. The
  publisher sends each message to the subscriber's container over TCP, and removes it from the
  queue when the handler returns `nil`. While the subscriber's container cannot be reached, the
  message stays in the queue; only `ttl` ends its wait. `retries` counts handler failures only.
- **Bounded memory:** `direct` requires topic message types to be bounded with `limit` fields
  (Appendix B.3). A `limit` goes on a field, not on a named type: `ID OrderID limit 64`. It counts
  bytes for a string (Go's `len`) and elements for a slice or a map. The bound is a Go integer
  constant expression, as an array length is, so code and limit can share one constant:
  `Note string limit MaxNoteBytes`. A field with nested variable-size parts takes one bound for
  each part, outer first, and a map's key before its value: `Tags []string limit 10, 64` (10
  strings of at most 64 bytes), `Stock map[string]int limit 100, 32` (100 entries, keys of at most
  32 bytes). An element that is a struct uses its own field limits. The limits bound the memory,
  and the compiler calculates the memory limit of each queue: capacity × maximum message
  size. `vulcan audit` shows each limit and the total.
- **Dead letters:** a message that fails all its retries, or expires, goes to the dead-letter
  route. A drop under `dropOldest` or `dropNewest` is counted, not dead-lettered.
- **A dead-letter route is a subscribed topic** of type `DeadLetter[T]` in some leaf, declared as
  `subscribes DeadPlaced topic[vulcan.DeadLetter[events.OrderPlaced]]`. `vulcan.DeadLetter[T]` is a
  type of the runtime package `vulcan`, with the fields `Message T`, `Error string limit 4096`,
  `Attempts int`, `Wire string limit 1024`, and `Time time.Time`. The type is bounded, so the
  `direct` rule holds for it: `Message` has the limits of `T`. `Wire` holds the text of a wire, and
  a wire longer than 1,024 bytes is a compile error (K0316). The limit is fixed, not taken from the
  system, so that two builds agree on it. The runtime does not check `Message` against the invariant
  and limits of `T`, because a message that failed them must still reach its route. The wire
  settings name it, and it can receive from any number of wires whose message type is `T`. `Error`
  is cut to 4,096 bytes.
- **The dead-letter hop.** When a transport gives up on a message, it calls the runtime's `dead`
  function in the process that holds the message (for `direct`, the publisher's). That runtime
  builds the `DeadLetter[T]` and sends it through the source wire's transport, with a
  `transport.Wire` whose `To` is the route and whose settings are the source wire's settings with
  each route cleared. The hop may stay inside one container; rule 3 of §5.4 does not apply to it.
  A dead-letter handler must be `idempotent` when the source transport redelivers. If the
  dead-letter handler fails, the message is retried with these settings, then counted and
  reported; there is no dead letter of a dead letter.
  Each dead letter carries the message, the last handler error or panic report, the attempt count,
  the wire, and the time. Thus "why did this message dead-letter?" has an answer without
  tracing.
- **Tracing** is off by default. The setting `trace: true` records each step of each message, and
  `vulcan trace <id>` shows them.

```go
Worker/Picking.Placed: Api/Orders.Placed via direct{
    capacity: 5_000, ttl: 24 * time.Hour, deadletters: Worker/Alerts.DeadPlaced }
Analytics/Ingest.Samples: Api/Metrics.Samples via kafka{ topic: "samples" }
// startup: --transport.direct.peers=worker=10.0.0.7:7400
```

Other transports, for example Kafka, NATS, or SQS, are **transports**: a node kind of their own, not
a library. A transport is declared `transport kafka { go "..." }` and named in lower case at the
wire (`via kafka{ ... }`). It is part of the runtime, not of the architecture, and it is the second
kind of node that can import Go packages. It is exempt from the library rules (§5.4a, §5.4c): it
can acquire network resources, hold connections, and keep queues. No leaf or library can import it;
only a `via` in `system.vulcan` can name it. It implements the transport interface of Appendix C.
`vulcan audit` lists each transport.

#### 5.5.2 Message evolution

Containers do not deploy at the same moment. Thus two builds of one system run together, and their
message types can differ. The holistic check (§11.2) sees only one build. So the compiler also
compares each message type with the last committed schema, in `topics.lock`. It cannot see the
deployed build; the lock stands for it.

- **The compiler owns the wire format.** Vulcan does not invent one: it uses the binary encoding of
  Protocol Buffers (proto3), with the field numbers from the lock. The wire carries the number, not
  the name. A field number is never used again. Each type has one encoding, so two compilers
  produce the same bytes:

  | Go type | proto3 encoding |
  |---|---|
  | `bool` | `bool` |
  | signed integers (`int`, `int8` … `int64`) | `sint64` (zigzag) |
  | unsigned integers (`uint`, `uint8` … `uint64`, `uintptr` not allowed) | `uint64` |
  | `float32`, `float64` | `float`, `double` |
  | `string` | `bytes` (Go strings need not be UTF-8) |
  | `[]byte` | `bytes` |
  | `time.Time` | `google.protobuf.Timestamp`; it arrives in UTC, with no monotonic reading |
  | `time.Duration` | `google.protobuf.Duration` |
  | struct | a message; every field crosses, exported or not; an embedded field is a nested message |
  | slice, array | a repeated field; an array's length is checked at receipt |
  | map | a map field |
  | a slice or a map as the element of a slice, or as a map value | a generated wrapper message whose field 1 holds it |

  A received integer that does not fit its Go type (an `int` on a 32-bit target) makes the
  message invalid, as a broken `limit` does.
- **A lock file records each schema.** `topics.lock`, in the system directory, lists each
  topic message type, each type that it contains, and each remote port, with field numbers, field
  types, limits, and invariants. The compiler assigns
  the field numbers, generates the file, and the project commits it. No one edits it by hand. It is a
  text file, so a diff shows each schema change in review:

  ```
  # topics.lock: generated by vulcan check. Commit it. Do not edit it.
  type shop/lib/events.Line
    1 SKU  string  limit 32
    2 Qty  int
  type shop/lib/events.OrderShipped
    1 Order  shop/lib/events.OrderID  limit 64
    2 Note   string                   limit 500
    3 Lines  []shop/lib/events.Line   limit 100
    invariant Order != ""
    retired 4
  retired type shop/lib/events.OrderPlacedV1
  port Api/Orders.Stock
    func Level(sku shop/lib/stock.SKU) (int, error)
      requires sku != ""
  pure shop/lib/stock.Valid  sha256:9f2c…
  hash sha256:41d0…
  ```

  The last line is a hash of the lines before it, so a hand edit is a compile error (K0321). When
  no lock exists, `vulcan check` creates one. A type is named by its Go import path. Types and ports
  are in name order, and fields are in
  number order. A field shows its number, name, type,
  and limit. A limit shows its value, so a named constant appears as its number. An
  invariant appears in the compiler's canonical form, so a change in formatting is not a change in
  the invariant. A `pure` line records a hash of the canonical body of each `pure` function that an
  invariant or a remote contract calls, directly or through other `pure` functions; a new hash is a
  change of that invariant or contract. A remote port is named by its consumer. `retired` lists the
  numbers of removed fields, which are never used again. A
  removed message type stays as a `retired type`, and its name cannot be used again, because a
  deployed build can still send it.
- **`vulcan check` compares the current types with the lock in the last commit.** It reads
  `topics.lock` from Git `HEAD`, or from the disk when the project is not in Git. Thus two local
  edits to one new field are one change. A compatible change passes, and the compiler writes the
  new lock to the disk. A breaking change is a compile error. `DeadLetter` types are not in the
  lock; their `Message` type is.

| Change | Result |
|---|---|
| Add a field | Compatible. A message from an older publisher has the field's zero value; the invariant must accept it. |
| Remove a field | Compatible. An older subscriber reads the field's zero value; its invariant must accept it. |
| Rename a field | Compatible to the compiler, which sees a remove and an add; the old field's data is lost. A change that removes one field and adds another of the same type gets a warning (K0319). To keep the data, use a new type. |
| Change an invariant (relax or tighten) | Breaking. Both sides check invariants, so a change in either direction rejects messages in one deploy order. |
| Change the type of a field | Breaking. |
| Change a `limit` (raise or lower) | Breaking. Both sides check limits, so a change in either direction rejects messages in one deploy order. |
| Change a remote method's signature or contract | Breaking. |
| Add a method to a remote port | Compatible. A call that reaches a callee build without the method returns `vulcan.Unavailable`, so deploy the provider first. |
| Remove a method from a remote port | Compatible. |
| Add or remove a first `context.Context` parameter | Compatible: it does not cross as a value. |

- **A breaking change needs a new message type.** For example, `OrderPlacedV2`. During the
  migration, the publisher publishes both types. When no subscriber reads the old type, the
  project removes it.
- **A subscriber ignores unknown fields.** Thus an older subscriber can read a message from a
  newer publisher.
- **Any deploy order works.** Because each change in the lock is compatible in both directions,
  the containers can deploy in any order. The one exception is a new remote method, which returns
  `vulcan.Unavailable` until its provider deploys.

#### 5.5.3 Remote method ports

A remote method port is a synchronous request and response between containers. It uses the same
syntax as a port inside a container. The `remote` marker adds what the network makes true:

```go
// In Api (the consumer declares what it needs):
needs remote port Stock {
    func Level(sku stock.SKU) (int, error)
        timeout 200 * time.Millisecond
}

// In a leaf of Api:
qty, err := Stock.Level(sku)    // err can also be Unavailable, Timeout, or RemotePanic
```

- **Transport errors.** The last result of each method of a remote port must be `error` (K0315).
  It can also be `vulcan.Unavailable`, `vulcan.Timeout`, or `vulcan.RemotePanic`, which `errors.Is`
  matches.
- **Errors cross as text and a code.** A non-`nil` error from the callee reaches the caller as a
  `*vulcan.RemoteError` with its `Error()` text and its code. A library declares an error that keeps
  its identity across containers with `vulcan.NewError("stock.unknown-sku", "unknown SKU")`. Such an
  error crosses as its code, and on the caller `errors.Is(err, stock.ErrUnknownSKU)` matches it.
  Any other error crosses as text only. `error` is allowed here although §5.5 forbids interfaces in
  messages.
- **Each call has a timeout.** The `config { }` section of `system.vulcan` sets a default timeout. A
  method of a remote port can override it with `timeout`; `timeout` on a method of a port that is
  not remote is a compile error (K0318). If no timeout resolves, the code does not compile.
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

  The settings of `direct` are tagged `topic`, so a call wire accepts none of them (K0308).

- **A panic in the callee** is recovered by the runtime (§6); the caller receives `RemotePanic`.
  When the callee fails a contract, `RemotePanic` carries the failure report. When the caller's
  check of the reply fails, the caller panics. In both cases the outcome is unknown, as after a
  timeout: the callee's effects can have happened (§7.4).
- **The schema lock covers the contracts of remote ports.** A change to a remote port's contract is a
  breaking change, as for message invariants (§5.5.2).

**How to choose.** Use a remote method port when the caller needs an answer now. Use a topic when
the publisher must not depend on who reads, or when many subscribers react to one event. The
developer decides for each boundary.

### 5.6 Diagrams

`vulcan diagram --level context|container|component|code [--focus <node>]` draws each C4 level from
`system.vulcan` and the `component.vulcan` files. Composites hold no code. Thus a diagram at any
zoom level shows where all behavior is. A container or a group cannot hide logic.

## 6. Code: Go, unchanged

**Function bodies are Go.** Types, generics, errors, `nil`, zero values, integers, slices, maps,
goroutines, channels, locks, `panic`, and `recover` all have exactly Go's syntax and Go's behavior.
Each Go construct compiles in Vulcan and runs as it runs in Go. Vulcan never changes what a body
means. It adds checks that reject some bodies: contracts (§7), the boundary rules (§5.4: a body
cannot refer to another leaf, or assign a port name), the library rules (§5.4a: no assignment to a
library `var` after startup), the rules on reached functions (§5.4c), and the invariant rules
(§7.2: no field write from outside a struct's methods).

What Vulcan adds is around the code, not in it: the architecture (§5), contracts (§7), tests (§8),
and the tools (§9, §10).

**Checks on Go code are warnings.** The tools report common Go faults as warnings with fix-its. A
warning never stops a build, and `vulcan audit` lists them. Three warnings follow established Go
analysis tools: `errcheck` (K0701), Uber's `nilaway` (K0703), and Go's race detector (K0704).
Error masking (K0702) is Vulcan's own, and this section defines it.

- an unhandled error: an `error` result that the code drops without assigning it (as `errcheck`
  reports). An explicit `_ = f()` is a visible, deliberate discard: it is not a warning, and
  `vulcan audit` lists each one.
- error masking: a branch that receives an error and never reads the error value. Any read
  handles it: a return, a wrap, a log, an inspection (`errors.Is`), or a pass to another call, for
  example into an HTTP response. Whether a response should show an error is a question for review,
  not a check.
- a possible `nil` dereference (as `nilaway` reports)
- a data race, as the race detector reports. `vulcan test` runs every test with the race detector by
  default; `vulcan test --race=false` turns it off, and `vulcan audit` lists each such run. A race
  fails the test, as in Go, and `vulcan audit` also lists it. Vulcan does no static race analysis.
  On a host where Go cannot run the race detector (on Windows, it needs cgo and a C compiler),
  `vulcan test` runs without it, warns (K0705), and `vulcan audit` lists the run. A project that
  relies on race detection runs it in CI on a host that has it.

Each Vulcan version pins the version of each analysis tool, so a warning depends only on the code
and the Vulcan version.

**The runtime package.** Code imports the runtime package as
`github.com/robot-accomplice/vulcan/vulcan`, which provides `vulcan.DeadLetter`, `vulcan.Bind`, and
`vulcan.Published`. Appendix C gives its API. It holds only types and test helpers, and it reaches
no network function. The transports, `direct` included, are separate packages that only the
generated `main` imports (§5.5.1). Thus the network rule (§5.4c) has no exception for the runtime.
`vulcan check` needs `go.sum`, as `go build` does, to resolve the Go dependencies that the rules of
§5.4c read.

**Modules.** The `go` command cannot see imports in `.vulcan` files, so `go mod tidy` would remove
each dependency that only Vulcan code uses. `vulcan mod <args>` runs `go mod <args>` over the Go
code that the compiler generates, with the project's `go.mod` and `go.sum`. Thus `vulcan mod tidy`
adds each dependency that Vulcan code imports, and writes `go.sum`.

**Go versions and pinning** ([decision 5](decisions/0005-names-and-reproducibility.md)).

- **Supported Go versions.** Each Vulcan release supports the Go releases that the Go project
  supports on the day of the release: the newest two. The first release supports Go 1.26 and the
  newest release after it. The release notes name the range.
- **Go syntax follows the compiler.** A body can use the Go syntax of the `go/types` that the
  Vulcan release is built with (§11.1). A new Go feature works in bodies once a Vulcan release is
  built with a Go version that has it.
- **One `go.mod` pins everything.** The `go` line sets the language version, the `toolchain` line
  the exact Go toolchain, and a `tool` line the exact compiler. The analysis tools are module
  dependencies of the compiler, so the same line pins them.

  ```
  module shop

  go 1.26
  toolchain go1.26.1

  tool github.com/robot-accomplice/vulcan/cmd/vulcan

  require (
      github.com/robot-accomplice/vulcan v0.1.0
      github.com/robot-accomplice/vulcan/vulcan v0.1.0
  )
  ```

  `go tool vulcan build` runs the pinned compiler. The runtime is its own module, so a program's
  `go.sum` holds the runtime and its dependencies, not the compiler's.

**Where code starts.** The compiler generates `main` from `system.vulcan`; code never writes it. A
leaf can define two plain Go functions, which Vulcan finds by their names, as Go finds `main` and
`init`:

- `setup`, for work that must finish before the leaf receives anything: open a database, read
  settings. A leaf that no wire constructs, for example a subscriber, has no constructor, so this
  is its place for setup.
- `startup`, for long-running work after serving starts, for example the server of an entry leaf.

Each has the signature `func(ctx context.Context) error`, or `func(ctx context.Context, s Settings)
error` for a leaf with settings (§5.3), where `Settings` is any struct type of the leaf. Another
signature is a compile error (K0601). Both are ordinary functions, so a test of the leaf can call
them.

The generated `main` starts a container in five steps:

1. **Read the flags and create the transports** (§5.3). A missing or invalid flag stops the
   process with a report that lists each one.
2. **Construct and bind.** `main` calls the constructor of each wire, a provider before each
   leaf that needs it, and binds each port as soon as its provider exists. The wiring has no
   cycles (§5.4, rule 8), so this order exists. A constructor can call its own ports, because they
   are already bound. A remote port is bound to a client, but the other container may not serve
   it yet, so a call can return `vulcan.Unavailable`. A failed constructor stops the process with a
   report.
3. **Set up.** `main` runs every `setup` function, all at once, and waits for them. A failed
   `setup` stops the process with a report.
4. **Serve.** `main` starts each subscription and each served remote port.
5. **Run the startup functions**, all at once, for example to start the server of an entry leaf.

Steps 1 to 4 finish before any startup function runs. Thus an entry leaf never serves a request
before the ports it calls are bound, and a handler never runs before its leaf's `setup` returns.
A resource that a provider needs, for example a database connection, is opened in its
constructor; a resource that a subscriber needs is opened in `setup`. The runtime recovers a panic
in `setup` or `startup` and treats it as an error. The process runs until a stop signal, or until
a startup function returns an error. A container with no subscription, no served remote port, and
no entry leaf exits with code 0 when all its startup functions have returned `nil`.

`main` stops a container in four steps:

1. **Stop new work.** Cancel the `ctx` of the startup functions, so that an entry leaf stops
   accepting requests. Stop each subscription and each served remote port from taking new
   messages and calls.
2. **Finish work in progress.** Wait for running handlers, served calls, and startup functions,
   for at most `shutdown_timeout` (a `config` key; default 30 s).
3. **Cancel the rest.** Cancel the `ctx` of each handler and served call that still runs. A
   message whose handler did not return `nil` is not acknowledged, so the transport delivers it
   again later.
4. **Close.** Close each transport, then exit. The exit code is 0 after a stop signal, and 1 after
   a failed startup function, with a report. A process with `on_contract_failure: exit` ends with
   exit code 3 at a contract failure, without these steps (§7.4).

**Panics in handlers and served calls.** The runtime recovers a panic in a topic handler (§5.5.0)
or in a served remote call (§5.5.3), and counts it as a failure of that message or call, with a
report. Elsewhere a panic behaves as in Go.

## 7. Contracts

A contract is a clause on a declaration. It is not a comment.

| Clause | Attaches to | Meaning |
|---|---|---|
| `requires expr` | functions, port methods | Precondition: the obligation of the caller |
| `ensures expr` | functions, port methods | Postcondition, checked at every return, and again after the deferred calls when one of them can change a result (§7.3). `err` and `result` name the results (§7.1). Thus a failure rule is a contract too: `ensures err != nil => a.Balance == old(a.Balance)`. |
| `invariant expr` | structs | Is true after construction and when an outer method call returns normally, exported or not. A panic that leaves an outer call with the invariant false becomes a contract failure (§7.2, §7.3). |

### 7.1 What a contract can contain

Contract expressions are **pure**. They never do I/O, call ports, use channels or topics, change
data, or read the clock or a random source. A contract expression itself also cannot fail:

- **Exact integers.** Integer `+`, `-`, `*`, unary `-`, conversions between integer types, and
  comparisons give the mathematical result. They cannot overflow. `/` and `%` truncate toward
  zero, as in Go. Bitwise operators and shifts use Go semantics on the operand's type. Float
  arithmetic is Go arithmetic. Operand types follow Go's rules. When an exact value is passed to a
  `pure` function and does not fit the parameter's type, the contract fails.
- An index, a slice expression, a pointer dereference, a divisor, and a shift count must be proved
  safe by the contract itself, for example `len(xs) > 0 && xs[0] > 0`,
  `a != nil => a.Balance >= 0`, or `d != 0 && n / d > 1`. `requires xs[0] > 0` alone is a compile
  error (K0504).
- A type assertion is not allowed in a contract (K0512).
- **Names.** In an invariant, the struct's fields are in scope without qualification, and they
  shadow package-level names. In `ensures`, named results are in scope by their names.

**The proof rule.** The compiler proves a fact only from these facts:

1. each conjunct to the left in a chain of `&&`, and the antecedent of each enclosing `=>`;
2. in an `ensures` or an `old(x)`, each conjunct of the function's `requires` that reads only
   parameters that the body never assigns, and reads no field, element, or pointee of them. The
   value of a pointer parameter (`a != nil`) and the length of a slice or string parameter
   (`len(xs) > 0`) qualify; `a.Balance > 0` does not, because the body can change it;
3. at a call, the callee's `requires` with each parameter replaced by its argument, where an
   argument that is not a constant is an unknown value; at a return, the `ensures` with the
   returned values. No fact comes from the caller's body.

The atoms are integer variables, lengths, constants, field selectors, `old(…)`, `result`, and
`nil` comparisons. Each integer atom ranges over its Go type, so an unsigned atom is at least 0.
With these facts, the compiler decides linear integer arithmetic over the atoms. It uses nothing
else, so each compiler accepts the same contracts. The same rule decides when a contract is
proved false (K0501) and when a check can be removed; everything else is checked at runtime.

**Nil receivers.** A method with a pointer receiver whose own contract reads through the receiver
has an implicit `requires r != nil`, checked like any `requires`. The invariant check skips a nil
receiver. Thus a Go method that handles a nil receiver keeps its meaning.

**`err` and `result`.** `err` names the last result when its type is `error`. `result` names the
first other result, and `result.0`, `result.1`, … name each other result in order. Plain `result`
in a function with two or more other results is a compile error (K0510). In a function with an
`error` result, `result` can appear only where `err == nil` is a fact by the proof rule, for
example `err == nil => result.Amount == amt` (K0502). A function with a contract cannot name a
parameter or a result `err` or `result` (K0511).

**`old(x)`** is the value of `x` when the function starts. The compiler evaluates it at entry, by
Go assignment. Thus the type of `x` must hold no reference (no pointer, slice, map, channel,
function, or interface): a copy of a reference would show the new data, not the old (K0509). Write
`old(len(xs))`, not `old(xs)`. Because `old(x)` runs at entry, the facts that make it safe must
come from the function's `requires`.

Thus checking a contract never changes what a correct program does. Within that rule, Vulcan
permits two tiers:

| Tier | Adds | Example |
|---|---|---|
| 1 | Comparisons, arithmetic, `&&` `\|\|` `!`, `=>` (implies), field access, `len`, `result`, `old(x)` (the value on entry) | `ensures len(result) == old(len(xs)) + 1` |
| 2 | Calls to functions marked `pure`. The compiler verifies that these functions have no effects. Their bodies are Go and run as Go: Go arithmetic, and a panic as in Go. A panic while a contract runs is a failure of that contract. A `pure` function can still loop forever; the compiler does not prove termination. | `requires IsSorted(a) && IsSorted(b)` |

```go
pure func IsSorted[T cmp.Ordered](xs []T) bool {
    for i := 1; i < len(xs); i++ {
        if xs[i-1] > xs[i] { return false }
    }
    return true
}

func Merge(a, b []int) []int
    requires IsSorted(a) && IsSorted(b)
    ensures  IsSorted(result) && len(result) == len(a) + len(b)
{
    ...
}
```

**What a `pure` function can do** (K0505). It can assign to its local variables and its results.
It can write through a path only when the path's root is a local variable that a literal, `make`,
`new`, or `append` to such a variable initialized, and each step of the path is an index or a
field of a value that the root holds by value. It cannot write through a pointer, a slice, or a
map that it loaded from such a value, or that a slice expression or a call returned. Thus
`c := []*T{param}; c[0].X = 1` is an error. It cannot `append` to a slice that it did not create
by these rules. It cannot read a package-level `var`, use a channel, a `select`, a goroutine, `defer`, or
`recover`, call a method through an interface, call a port, or use a topic. It can call only
`pure` functions. It can call a function value only if the value is one of its parameters; then
each call of the `pure` function must pass, for that parameter, a `pure` function or a function
literal that passes these same rules.

A rule over a collection uses a `pure` predicate, for example `requires vulcan.All(xs, isPositive)`
(Appendix C). Its runtime cost is O(n), and contracts run in every build.
`vulcan audit` lists each contract whose check costs more than O(1). When that cost is too high for
a hot path, the developer states the rule as a test instead of a contract.

### 7.2 Struct invariants

An invariant protects a struct only if no code can go around it. Thus:

- **Only methods write fields.** The fields of a struct with an invariant are read-only outside
  its own methods. A write from anywhere else is a compile error.
- **No field addresses outside methods.** Code outside the struct's own methods cannot take the
  address of a field (`&e.Amount`), because a write through that pointer would go around the
  invariant. It is a compile error. The fix-it reads into local variables, then builds the value
  with a literal, which is checked. A call of a pointer-receiver method on a field, and the
  slicing of an array field, take the field's address too. A pointer to the whole struct is
  allowed: a write through it, `p.Amount = 1`, is a field write, and the first rule covers it.
- **Calls that receive the struct are checked.** When code passes a pointer to the struct, or a
  value that holds the struct through a pointer (`*[]T`, `*[N]T`, `*S` with a `T` field, a map of
  `T` or `*T`), to a function that is not a method of `T`, the compiler checks the invariant of each
  `T` that it can reach when that call returns. This applies also inside a method of `T`, for
  example `json.Unmarshal(b, a)`. `vulcan audit` lists each such check with its cost, and each use
  of `reflect` or `unsafe` on such a struct.
- **Function literals and goroutines.** A function literal inside a method counts as outside the
  method for these rules, unless the method calls it directly or with `defer`. A deferred call
  still cannot write a field (§7.3, rule 2). A `go` statement
  in a method runs after the method returns, so the invariant check does not cover it; it is a
  warning (K0520).
- **Elements are a limit.** A slice or a map field shares its elements with each copy of it. Code
  that holds a copy can change the elements without a method, and the compiler does not track
  copies. Thus an invariant over the elements holds only while no copy leaves the struct's
  methods. `vulcan audit` lists each method that returns a slice or a map field of a struct with an
  invariant.
- **Construction is checked.** A struct literal, and a conversion to the struct type, are checked
  where the code creates them. If the compiler can prove the violation, it is a compile error; if
  not, the check runs at runtime.
- **The zero value must satisfy the invariant.** Go creates zero values in many places (`var`,
  `new`, `make`, arrays, missing map keys, closed channels, generic `var zero T`), so Vulcan cannot
  block them all. Thus an invariant that the zero value breaks is a compile error on the invariant
  itself: write `invariant Lo <= Hi`, not `Lo < Hi`, or use a pointer field and write
  `P != nil => ...`. For a generic struct, the compiler checks a tier-1 invariant for each
  instantiation at compile time, and a tier-2 invariant at the first construction of each
  instantiation.
- **Outer calls only.** The invariant is checked when a method call returns, exported or not,
  except for a call inside a method whose receiver expression is that method's own receiver
  variable, not assigned in the method. Thus `Transfer` that calls `a.Withdraw` is not checked on
  that return, and a method can pass through a temporarily broken state. This is the rule of
  Eiffel for unqualified calls. A call through any other expression is checked, even when it
  refers to the same value: in `a.Transfer(a, x)`, `to.Deposit(x)` checks `a`. The rule is decided
  from the source text, so it needs no alias analysis. A method that writes no field, calls no
  method that does, and passes no pointer to `T` to a function, skips the check: the invariant held
  when it started, so the result is the same.
- **Checks run before deferred calls.** The invariant and `ensures` are checked at each return,
  before the method's deferred calls run. Thus a method that unlocks with `defer mu.Unlock()` is
  checked while it still holds the lock. A method of such a struct that unlocks other than by
  `defer` gets a warning (K0521), because its check then runs after the unlock. §7.3 covers what
  the deferred calls and a panic can change after this check.

```go
type Account struct { Balance int }
    invariant Balance >= 0

a := Account{Balance: -5}   // compile error: Balance >= 0 is false
a.Balance = -1              // compile error: write outside a method

func (a *Account) Transfer(to *Account, x int) {
    a.Withdraw(x)           // call through the receiver: no check on its return
    to.Deposit(x)           // call through another expression: checks to
}                           // checked here
```

### 7.3 Deferred calls and panics

Go runs deferred calls after a `return`, and they can change the named results. A deferred
`recover` turns a panic into a normal return that passes no `return` statement. A panic can leave
a method halfway through a change. These rules keep the checks of §7.1 and §7.2 valid on each of
these paths ([decision 1](decisions/0001-contract-timing.md)).

1. **Return check.** `ensures` and the invariant are checked at each `return` and at a reachable
   end of the body, before the deferred calls run (§7.2).
2. **No deferred field writes.** In a method of a struct `T` with an invariant, a deferred call
   cannot write a field of `T` (K0522). This covers a deferred function literal and a deferred call
   of a method of `T` that skips its check. A deferred call of a method of `T` through the receiver
   is an outer call, so its own check runs. The fix-it moves the write before the `return`.
3. **Check after the deferred calls.** A *risky deferred call* is one of these:
   - a deferred function literal that assigns a named result, takes its address, or calls
     `recover`;
   - a deferred call of a named function that receives the address of a named result, or that
     calls `recover` directly;
   - a deferred call through a function value or an interface value.

   A function whose `ensures` reads a named result, and that has a risky deferred call, checks
   `ensures` again after its deferred calls, with the final results. The second check does not
   read the state again. At the return check, the compiler saves by value each atom of a conjunct
   that reads a named result and other state, for example `a.Balance` or `len(a.items)`. The
   second check uses the saved values, so it reads no field after a deferred unlock. A conjunct
   that reads a named result and passes state by reference to a `pure` function, for example
   `err == nil => IsSorted(a.items)`, cannot be saved. In a function with a risky deferred call,
   it is a compile error (K0523).
4. **Recovered panics.** When a function with a risky deferred call returns normally and no return
   check ran, a deferred call recovered a panic. The function then checks all of its `ensures`, and
   its receiver's invariant, against the state after its deferred calls. If the function also
   unlocks in a deferred call, this check runs after the unlock: a warning (K0524).
5. **Panics that leave a method.** No result reaches the caller, so `ensures` is not checked. When
   a panic leaves an outer method call of `T` before a return check ran, the method checks the
   invariant. If the invariant is false, the panic continues as a `*vulcan.ContractFailure` whose
   `Cause` is the original panic value (Appendix C). The panic value changes only when the
   invariant is false. This check runs after the deferred calls, so it can read the fields after a
   deferred unlock. It runs only while a panic leaves the method, so it has no warning.
6. **Data races.** The checks assume a program without data races. Vulcan does no static race
   analysis (§6); `vulcan test` runs the race detector.

Thus:

- When a function with `ensures` returns normally, its results satisfy `ensures`, also after its
  deferred calls and after a recovered panic. The state that the conjuncts read is the state at
  the `return`, or, after a recovered panic, the state after the deferred calls.
- When an outer method call on a struct with an invariant returns normally, the value satisfies
  the invariant. When a panic leaves the call with the invariant false, the panic is a contract
  failure.
- **Limit.** A caller that recovers that contract failure, as `net/http` does for each request,
  can still use the broken value. Vulcan reports the break; it does not prevent the later use.
  Elements of slice and map fields stay a limit (§7.2).
- **Limit: the caller's name.** In a function with a risky deferred call, the body runs in a
  generated closure (Appendix F). `runtime.Caller` and `runtime.FuncForPC` then name the closure,
  for example `pkg.F.func1` instead of `pkg.F`, and code that walks the stack sees one more frame.
  The line stays right. Go gives no way to hide the frame, so this is a stated difference from
  Go, pinned by a conformance case.

```go
func (a *Account) Deposit(f *os.File, x int) (err error)
    ensures err != nil => a.Balance == old(a.Balance)
{
    defer func() {                       // risky: assigns err
        if cerr := f.Close(); err == nil {
            err = cerr
        }
    }()
    a.Balance += x
    return nil                           // return check passes; Balance is saved
}                                        // second check: err != nil and Balance changed: fails
```

How the compiler checks contracts:

- **Where.** A function checks its `requires` at entry and its `ensures` at each return. The
  compiler can remove a check at a call site where it proves the contract (not in release 1, which
  runs every check).
- **Statically**, when the proof rule of §7.1 proves or disproves a contract.
- **At runtime** in every build, including release builds. Contracts cannot be turned off. A contract
  that fails at runtime panics with a `*vulcan.ContractFailure` (Appendix C), which reports the
  contract, its source position, and the values.
- **Port contracts bind each implementation**, including test fakes. If a fake violates the
  contract of its port, the test fails.

How reviewers audit contracts:

- `vulcan audit` lists each contract. For each contract, it shows if the compiler proved it and if
  tests exercised it. It also lists each function with a check after its deferred calls (§7.3), and
  each K0524 site.
- Contracts appear in the generated docs and diagrams.

### 7.4 Contract failures and effects

A contract failure stops the code that broke the contract. It does not undo what that code
already did ([decision 3](decisions/0003-contract-failure.md)).

1. **Nothing is undone.** A contract failure is a Go panic. Each frame runs its deferred calls, as
   in Go, so a deferred `tx.Rollback()` runs. Vulcan undoes nothing else. An effect that happened
   before the failed check stays: a committed write, a sent request, a published message.
2. **`requires` prevents; `ensures` and the invariant detect.** Only `requires` runs before the
   body's effects. A rule that must stop an effect goes in the `requires` of the function that
   makes the effect, or in a check in the body before it. An *effect* is a call that reaches a
   network or acquiring function (Appendix D), a port call, or a publish. `vulcan audit` lists each
   `ensures` and each invariant of a function that reaches an effect before its return check, as
   "detects after effects".
3. **An effect in a `return` (K0525).** In a function with `ensures`, or a method of a struct with
   an invariant, a `return` whose expressions reach an effect gets a warning, because Go runs the
   effect before the check. In `return n, tx.Commit()`, a failing `ensures` leaves the data
   committed. Check first, so that a failure panics before the commit and the deferred rollback
   runs:

   ```go
   func (s *Store) Refund(tx *sql.Tx, id OrderID, amt Money) (err error)
       requires amt > 0
   {
       defer tx.Rollback()
       a, err := s.load(tx, id)
       if err != nil { return err }
       if err := a.refund(amt); err != nil { return err }   // a method with ensures: checked here
       if err := s.save(tx, a); err != nil { return err }
       return tx.Commit()                                    // the last step; no check follows it
   }
   ```

   The warning stays when the developer accepts the order, so that a reviewer sees it.
4. **Handlers and remote calls.** A contract failure in a handler goes to the dead-letter route at
   once, unless the wire sets `retryContractFailures` (§5.5.0). A remote contract failure leaves
   the outcome unknown (§5.5.3).
5. **Fail-stop on request.** The process setting `on_contract_failure` takes `panic` (the default)
   or `exit`. With `exit`, the runtime writes the failure report to standard error and ends the
   process with exit code 3. It runs no deferred calls, and no caller can recover the failure. The
   `config` key sets the value for the system, and the flag `--on-contract-failure` sets it for
   one process. `vulcan test` always uses `panic`, so that tests can check failures. With `exit`,
   the limit of §7.3 does not apply: no later request can use a broken value.

**Responsibilities of the application.** Vulcan does not enforce these:

- **One transaction for one change.** Write a state change, its audit line, and each event to
  publish (an outbox row) in one database transaction. Publish from the outbox after the commit.
- **Check before the commit.** Run each check that guards a write before `Commit`, so that a
  deferred rollback undoes the work on a failure.
- **Idempotency keys at each retryable entry.** An HTTP `POST`, a remote method that a caller can
  retry, and each handler take a key: a client key, an operation ID in the arguments, or
  `vulcan.DeliveryID(ctx)`. Store the key and the outcome with the change, in the same transaction.
- **Recover the outcome on a retry.** A retry with a known key returns the stored outcome, and
  does not run the change again. This covers a reply lost after a commit.
- **Overflow-safe arithmetic before a write.** Body arithmetic is Go arithmetic and can wrap;
  exact arithmetic exists only in contract expressions (§7.1). Write a guard that cannot overflow
  (`amount > charged - refunded`, not `refunded + amount > charged`), in `requires` or before the
  write.
- **Choose a process policy.** A container that holds money, or other state that must stay
  consistent, should consider `on_contract_failure: exit`.

## 8. Testing

Tests are Go tests: `func TestX(t *testing.T)` in `*_test.vulcan` files in the leaf directory.

- The design makes each leaf testable in isolation: its only dependencies are the ports that it
  `needs` and the topics that it subscribes to.
- **A fake is an ordinary type** that satisfies the port. It must meet the contract of the port,
  which is checked at each call, as for the real provider.
- `vulcan.Bind(t, &Checkout, CheckoutOK{})` binds a fake to a port name for one test, and restores
  the wiring when the test ends. A second `vulcan.Bind` on the same port in one test replaces the
  first. When the test ends, the bindings are undone in reverse order, so the wiring is as it was
  before the test. A port name is shared by the whole test binary, so `vulcan.Bind` panics in a test
  that called `t.Parallel`, or whose parent did, as Go's `t.Setenv` does.
- `vulcan.Published(t, Placed)` returns the messages that the test published on a topic. It panics
  under `t.Parallel`, as `vulcan.Bind` does.
- **An unbound port fails the test.** In a test binary, each port that no `vulcan.Bind` binds is
  bound to a stub that fails the test with "port Checkout is not bound; use vulcan.Bind".
- `vulcan.Deliver(t, Shipped, msg)` calls the handler of a subscribed topic with one message, and
  returns its error. It checks the message's invariant and limits first, as a receipt does.
- `vulcan fake <Port>` writes the skeleton of a fake type, with one method for each method of the
  port.

```go
func TestPaysForAnOrder(t *testing.T) {
    vulcan.Bind(t, &Checkout, CheckoutOK{})
    if err := Place(order); err != nil {
        t.Fatal(err)
    }
    if got := vulcan.Published(t, Placed); len(got) != 1 {
        t.Fatalf("published %d, want 1", len(got))
    }
}
```

- `vulcan test Billing` builds and runs the tests of one leaf, and `vulcan test money` the tests of
  one library. It does not build the rest of the system.

## 9. Compiler intelligence

In addition to type checks, the compiler reports these problems as diagnostics with fix-its:

- **Near-duplicate code** in the project (R2). Two functions or methods are near-duplicates when their
  token sequences, with names and literals replaced by placeholders, are at least 90% similar, and
  the smaller has at least 50 tokens. Similarity is 1 − d / n, where d is the Levenshtein distance
  between the two token sequences and n is the length of the longer one. A function literal counts as part of the function around it.
  `config { duplicate_similarity: 90, duplicate_min_tokens: 50 }` changes these values.
  The fix depends on where the duplicates are:
  - Inside one leaf: the diagnostic points to the existing function to reuse.
  - In different leaves: a leaf cannot call another leaf's function (§5.4, rule 1). So the fix-it
    is **extract to library**. The compiler moves the function, and the private types that it
    needs, into a new or existing library, and updates both leaves to import it.
  - If the move is impossible, the diagnostic says why. For example, the function calls an
    acquiring function (§5.4c, rule 2) or uses a port. The hint then suggests how to extract only
    the pure part.

  ```
  warning K0412: validateAddress in Orders and Shipping are 94% similar
    fix-it: extract to lib/address (moves validateAddress, type Address)

  warning K0412: syncStock in Api/Orders and Worker/Inventory are 91% similar
    cannot extract: syncStock calls the acquiring function os.Open
    hint: extract the pure part (diffStock) to a library
  ```
- **Error masking** patterns (R3).
- **Architecture violations** (§5.4). The diagnostic gives the exact dependency edge that broke the
  rule.
- **Architecture diffs.** `vulcan archdiff main..HEAD` reports added and removed ports,
  dependencies, and contracts. Reviewers use this report for R6.
- Dead code and unused ports.

Each diagnostic has:

- a stable code (Appendix E)
- a message for humans
- a *hint* that tells how to fix the problem in idiomatic Vulcan
- a machine-readable form (JSON)

## 10. Tooling that instructs the model

The tools assume that one of their users is an LLM (R8):

| Command | Purpose |
|---|---|
| `vulcan guide` | Prints the language primer (`docs/guide.md`) and the conventions of the project. **The primer has a hard budget of 4,000 tokens**, and it must stay complete: a model that reads only the primer must be able to write correct Vulcan. A language change that does not fit the budget must be cut or simplified. |
| `vulcan context <component>` | Emits the minimum context to work on one leaf: its `component.vulcan`, the wires in `system.vulcan` that connect it, and the method sets of the types on the other end of each wire (R1). |
| `vulcan query <symbol>` | Looks up a symbol exactly in the symbol index of the compiler. This prevents invented APIs (R4). |
| `vulcan explain <code>` | Gives a long explanation of a diagnostic, with examples. |
| `vulcan doc` / `vulcan diagram` | Generates docs, and C4 diagrams at each level (context, container, component, code), from the source (§5.6). |
| `vulcan audit` / `vulcan archdiff` | Audits contracts and architecture (§7, §9), with the unchecked paths and the strict-mode exemptions of §5.4d, and the checks that detect after effects (§7.4). |
| `vulcan fmt` | The one formatter. |
| `vulcan test <leaf>` | Builds and runs the tests of one leaf (§8), with the race detector (§6). |
| `vulcan mod <args>` | Runs `go mod <args>` over the generated Go, so `vulcan mod tidy` sees imports in `.vulcan` files (§6). |
| `vulcan fake <Port>` | Writes the skeleton of a fake type (§8). |
| `vulcan trace <id>` | Shows the steps of one traced message (§5.5.1). |
| `vulcan check` / `vulcan build` | The holistic boundary check, and the parallel build of each leaf (§11.2). |

Each command has `--json` output. An LSP server and an MCP server will give the same functions
to editors and agents. They are deferred until the compiler's stages are measured (§11.1), and
they reuse stages 1 to 5.

## 11. Runtime

- Native compilation and a tracing GC.
- **Backend: Go source first, LLVM later.** The first compiler lowers the typed Go syntax tree
  directly to Go source (§11.1, stage 6). It has no intermediate representation (IR) of its own.
  An IR comes with a second backend, which is its first test.
  - *Go backend (first).* Vulcan gets the GC, goroutine scheduler, channels, and cross-compilation
    of Go at no cost. The semantics of Vulcan map closely onto Go. The emitted code has `//line`
    directives. Thus the stack traces, profilers, and debuggers of Go report positions in `.vulcan`
    files, not in the generated Go.
  - *LLVM backend (deferred).* The LLVM backend needs a GC and a scheduler of Vulcan's own. That
    work takes months, and it blocks nothing in the language design, so it is deferred. It starts
    only after the language earns continued work (development plan, milestone 4), and it brings the
    IR.
- The concurrency model follows Go (tasks like goroutines, and channels), unchanged.

- **Build output is the developer's decision.** A system with one container compiles to one binary.
  For a system with more than one container, the `config { }` section of `system.vulcan` chooses the
  output. The compiler supports both forms:
  - one binary for the system, with one mode for each container (`shop api`, `shop worker`)
  - one binary for each container (`shop-api`, `shop-worker`)
- **Tested hosts.** The compiler is tested on `linux/amd64`, `linux/arm64`, `darwin/arm64`, and
  `windows/amd64`. Other hosts that Go supports may work, and are not tested. A program can be
  built for each target of the pinned Go toolchain.
- **Reproducible builds.** For one commit, one Vulcan version, and one Go toolchain, the generated
  Go is the same bytes: the compiler sorts every generated list, writes no time and no local
  path, and builds with `-trimpath`. Go records the module versions in the binary
  (`debug.BuildInfo`), so each binary names the Vulcan version that built it.

### 11.1 Compiler implementation

The compiler is written in Go. It is a staged compiler with a cache for each package
([decision 4](decisions/0004-compiler-structure.md)). It reuses `go/parser` and `go/types`, so
Vulcan bodies get Go's own type checker.

| # | Stage | Input | Output |
|---|---|---|---|
| 1 | Parse | `.vulcan` code files; declaration files | Blanked Go source with the same positions; a side table of contracts, `limit`s, `pure` and `handle` marks; declaration syntax trees |
| 2 | Stubs | Declaration syntax trees | Go declarations for each port, topic, and handler, in reserved `vulcan_` names (Appendix F) |
| 3 | Type check | Stages 1 and 2, and the export data of imported packages | `go/types` package and info; type-checked contract expressions |
| 4 | Leaf checks | Stage 3 | Diagnostics of the leaf (§6, §7); a summary of the package: declarations, reach facts, fingerprints, effects |
| 5 | Holistic check | Declaration syntax trees and every stage-4 summary | Diagnostics of §5.4 to §5.5.2, strict mode, near-duplicates across leaves, `topics.lock`, the audit |
| 6 | Lower | Stages 3 and 4, when all checks pass | Go source with `//line` directives and the generated `main`; then `go build` |

Stages 1 to 4 run for each package, in parallel: they are the per-leaf build of §11.2. Stage 5 is
the holistic check. Each stage reads only the outputs of earlier stages, so each can be tested
alone.

- **Blanking.** Stage 1 is a Vulcan lexer. It writes Go source in which each Vulcan-only token
  becomes spaces, so each position in the Go source is a position in the `.vulcan` file
  (Appendix B.1). A handler becomes Go text: `idempotent` becomes spaces, and `handle` becomes
  `func` and two spaces. Stage 1 then renames the function in the syntax tree to
  `vulcan_handle_<Name>`, which moves no position. Each contract expression is parsed with
  `go/parser.ParseExpr` at its own offset.
- **Caches.** The key of a result of stages 1 to 4 is a hash of the package's files, the Vulcan
  version, and the stage-4 summaries of the packages that it imports. Stage 5 caches the reach
  facts of each dependency module for each version (§5.4c). Stage 6 uses Go's build cache.
- **Incremental computation only when measured.** The project measures the time of `vulcan check`
  after a one-line edit in one leaf, and of a full check, first on a synthetic application of fixed
  size, then on a real application (decision 6, section 12). When the 95th
  percentile of a one-edit check exceeds the project's target, the slowest stage gets incremental
  computation, by a new decision.
- **Self-hosting is not planned.** It stays possible after the language earns continued work.

### 11.2 Compilation modes

Compilation has two separate modes. They run in parallel.

| | **Holistic check** (`vulcan check`) | **Per-leaf build** (`vulcan build`) |
|---|---|---|
| Purpose | Enforce boundaries. Find errors that need global context. | Compile code fast. |
| Input | All declaration files, all ports, and all signatures. Bodies only for global analyses. | One leaf directory, its `component.vulcan`, and the declarations of the libraries that it imports. |
| Finds | All of §5.4 (ports only, wiring, cycles, port compatibility), the library and reach rules (§5.4a, §5.4c), the topic and transport rules (§5.5 to §5.5.3), near-duplicate code across leaves and libraries (§9), architecture diffs, and the project audit. | Type errors and contract errors inside the leaf, near-duplicates inside the leaf, and the warnings on Go code (§6). |
| Parallelism | One pass over the whole tree. | One job for each leaf. All the jobs run in parallel. |

- **Why a leaf can compile alone.** A leaf declares each port that it requires (§5.3). Thus it
  compiles against its own declarations, never against the code of another leaf. A change inside a
  leaf rebuilds only that leaf.
- **Both modes must pass before linking.** The compiler makes a binary only when the holistic check
  passes and all leaves compile. Neither mode accepts the results of the other without a check.
- **Both modes run on each build.** `vulcan build` starts the holistic check at the same time as the
  per-leaf jobs. The architecture part of the holistic check reads only declarations, so it finishes
  early. The near-duplicate part needs the fingerprints of every leaf, so it finishes after the
  slowest leaf job.
- **Caching.** The compiler caches the result of each leaf. The cache key is a hash of the
  contents of the leaf directory and the declarations that the leaf depends on (§11.1). The
  holistic check reads the cached summaries of the leaves, so it reads no unchanged body again.
  §11.1 says when it becomes incremental.

**Why this is feasible.** Go already compiles packages in parallel. Each package compiles against
the export data of its dependencies, which contains declarations only. The design of Vulcan removes
the usual obstacles:

- Consumer-owned ports (§5.3): a leaf compiles against its own declarations, not against the code
  of providers.
- No cross-leaf inference: as in Go, every function declares its types.
- Global analyses of bodies use output from the per-leaf jobs. Each job emits the normalized token
  sequence of each function (its fingerprint). The holistic check compares them to find
  near-duplicates, and it does not read the bodies again. It skips each pair whose lengths differ
  by more than `100 - duplicate_similarity` percent, which cannot reach the threshold.
  Duplicates inside one leaf are found by the per-leaf job; all other pairs, and the rules of
  §5.4a and §5.4c, belong to the holistic check.

On the Go backend, each leaf becomes one Go package, and the generated wiring becomes `main`. Thus
the Go toolchain adds its own parallelism. Shared types are in libraries (§5.4a), which compile
first. Then the leaves compile in parallel against the library declarations.

### 11.3 Performance

**A goal, with no current test.** Vulcan aims to be competitive with Go, Java, C++, and Rust in
speed and in resource use. Today only the comparison with Go is measured (decision 6, N4). The
comparison with Java, C++, and Rust needs Q22's targets and, for C++ and Rust, the deferred LLVM
backend. Until then, no document may claim it.

**How the project measures it.** A benchmark suite runs in CI. It covers CPU-bound work,
allocation-heavy work, concurrency, and a service with topics. Each benchmark has a target
relative to Go, Java, C++, and Rust. Once the numeric targets exist (open question Q22), a change that misses a
target fails CI.

**What each safety feature costs, and how Vulcan reduces the cost:**

| Feature | Runtime cost | Reduction |
|---|---|---|
| Contracts (§7) | The cost of each expression, in release builds | The audit lists each contract with a collection-wide cost; the compiler removes checks that it proves |
| Garbage collection | GC pauses and memory overhead | On the Go backend, Go's GC; the LLVM backend needs Vulcan's own GC |
| Durable topics (§5.5.1) | A disk write for each message, on transports that keep queues on disk | The developer chooses the transport and its `durable` setting |
| Port calls (§5.3) | An interface dispatch and the port's `requires` and `ensures` at each call | The compiler removes checks that it proves; the LLVM backend can make the call direct |

**Backends.** On the Go backend, Vulcan performs like Go, plus contract checks and port dispatch. To
close the gap with C++ and Rust, the LLVM backend (§11) can use proofs to remove checks and can
control memory layout directly.

## 12. Non-goals (for now)

- Manual memory management and ownership types. The GC principle excludes them.
- Inheritance and class hierarchies. The "OO light" principle excludes them.
- Macros and metaprogramming. Each macro makes code harder to navigate and harder for a model to
  predict.
- A general system of effects or capabilities. It is deferred, because it
  conflicts with Go ergonomics. Examine it again when components and ports are proven.

## 13. Open questions

| # | Question |
|---|---|
| Q1 | Closed. The name is Vulcan, the extension `.vulcan`, the command `vulcan`, and the runtime module `github.com/robot-accomplice/vulcan/vulcan` ([decision 5](decisions/0005-names-and-reproducibility.md)). |
| Q22 | Numeric performance targets for the benchmark suite (§11.3): for each benchmark, the accepted ratio of speed and memory against Go, Java, C++, and Rust. |

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

## Appendix B: Grammar

The notation is the EBNF of the Go specification. A production that this appendix does not define
(`Expression`, `Type`, `Signature`, `ImportDecl`, `string_lit`, and others) is Go's production.

### B.1 Lexical additions

- **Contextual words.** The words of §4.1 are identifiers to the lexer. The parser treats one as a
  keyword only in the positions that the productions below show.
- **`=>`** is a new token. It is valid only in a contract expression.
- **`result.N`.** In a contract expression, `result` followed by `.` and a decimal integer is a
  selector, not `result` followed by a float literal.
- **Line ends before a contract.** Go's lexer inserts a semicolon at the end of some lines, and
  Vulcan's lexer does the same. The parser then ignores that semicolon in exactly three places:
  1. after the `Signature` of a package-level function or method, or of a port method, when the
     next token is `requires`, `ensures`, or `timeout`;
  2. after a contract clause, when the next token is `requires`, `ensures`, `invariant`,
     `timeout`, or `{`;
  3. after the type of a package-level `type` declaration that is not in a parenthesized group,
     when the next token is `invariant`.

  In each place, Go allows no identifier and no `{`, so no Go program changes its meaning. A field
  named `timeout` or `limit`, and a local variable named `requires`, are ordinary identifiers.
  `limit` must be on the same line as its field.

  *For implementers:* the blanking lexer of §11.1 applies this rule when it writes Go source.
  When contract lines follow a signature, it turns them into spaces and moves the body's `{` to
  the end of the signature line. Lines, and the columns of every other token, stay the same.
- **A contract after `Parameters`.** In Go, a function with no results can be followed by a result
  type, so `func F(x int) requires x > 0` could read `requires` as a type. In Vulcan, after
  `Parameters`, the identifier `requires`, `ensures`, or `timeout` starts a contract, unless the
  next token is `{`, `;`, `.`, or `[`. Thus `func F() requires { … }` keeps its Go meaning (a
  result of type `requires`), and `func F(x int) requires x > 0 { … }` is a contract.
- **A composite literal before the body.** A contract that ends with a composite literal needs
  parentheses before the body's `{`, as an `if` condition does in Go: `requires p != (T{})`.
- **Reserved names.** Identifiers that start with `vulcan_` or `Vulcan_` are reserved for generated
  code (Appendix F). A declaration of one is a compile error (K0005).

### B.2 Declaration files

```ebnf
SystemFile    = { ImportDecl ";" } "system" identifier "{" { SystemItem ";" } "}" .
SystemItem    = Container | Person | External | Uses | Wire | Config .
Container     = "container" identifier "{" { ContainerItem ";" } "}" .
ContainerItem = LeafList | Group | Wire .
Group         = "group" identifier "{" { LeafList ";" | Group ";" } "}" .
LeafList      = LeafEntry { "," LeafEntry } .
LeafEntry     = identifier [ Settings ] .
Person        = "person" identifier string_lit .
External      = "external" identifier string_lit .
Uses          = "uses" NodeRef NodeRef string_lit .
NodeRef       = identifier [ "/" identifier ] .
Wire          = Endpoint ":" Supplier [ "via" identifier Settings ] .
Endpoint      = [ identifier "/" ] identifier "." identifier .
Supplier      = Endpoint [ Settings ] .
Settings      = "{" [ Setting { "," Setting } [ "," ] ] "}" .
Setting       = identifier ":" Value .
Value         = Flag | List | identifier Settings | Expression .
List          = "[" [ Value { "," Value } [ "," ] ] "]" .
Flag          = "flag" "(" string_lit { "," FlagOption } ")" .
FlagOption    = ( "default" | "max" ) ":" Expression .
Config        = "config" Settings .

ComponentFile = { ImportDecl ";" } "component" identifier "{" { BoundaryItem ";" } "}" .
BoundaryItem  = Port | Topic | "gateway" "for" identifier | "entry" "for" identifier .
Port          = "needs" [ "remote" ] "port" identifier "{" { PortMethod ";" } "}" .
PortMethod    = "func" identifier Signature { Contract } [ "timeout" Expression ] .
Topic         = ( "publishes" | "subscribes" ) identifier "topic" "[" Type "]" .

LibraryFile   = { ImportDecl ";" } "library" identifier "{" "}" .
TransportFile = "transport" identifier "{" "go" string_lit "}" .
```

- **Parse, then resolve.** In a declaration file, an identifier followed by `{` always starts
  `identifier Settings`, never a Go composite literal; thus a `flag` can appear inside it
  (`transport: direct{ capacity: flag("cap", max: 10_000) }`). Any other `Value` parses as a Go
  expression, with no type information: `Worker/Alerts.DeadPlaced` parses as a division, and
  `Alerts.DeadPlaced` and `time.Hour` as selectors. A later pass resolves each value by the type of
  its setting: a setting of type `transport.Route` reads the expression as an endpoint.
- **Scope of a value.** A value can use the package `time` (no import needed), the libraries that
  the file imports, the leaves of the system (in an endpoint), and the exported constants of the
  setting's own type, without qualification (`whenFull: dropOldest`).
- **A value is a constant.** An expression value is a Go constant of its field's type, for example
  `5 * time.Second`, `"blue"`, or `money.MaxOrder`. A function literal, a call other than a
  conversion, or a variable is a compile error (K0606), because `system.vulcan` holds no code
  (§5.4, rule 2). A flag default follows the same rule.
- **Every field has a value.** Each exported field of a settings struct gets a value or a flag in
  `system.vulcan`; a field with none is a compile error (K0603).
- **Names.** A `library` name is its Go package name. A `transport` name is lower case. A
  `component` name matches its directory name, ignoring case (`Orders` in `api/orders/`).
- **`Uses`** draws an arrow from the first node to the second, for a relationship that no
  declaration shows (§5.3).

**Config keys.** `config { }` accepts these keys and no others (K0006):

| Key | Value | Default | Use |
|---|---|---|---|
| `context_budget` | lines | none: no leaf-size warning | §5.4, rule 10 |
| `timeout` | `time.Duration` | none | Default timeout of a remote method (§5.5.3) |
| `shutdown_timeout` | `time.Duration` | 30 s | §6 |
| `transport` | a transport with settings, `direct{ capacity: 5_000 }` | none | Transport of a wire across containers that has no `via`; a wire's own settings override these (§5.5.1) |
| `output` | `"system"` or `"containers"` | `"system"` | One binary for the system, or one for each container (§11) |
| `network` | a list of full function names | empty | Additions to the network functions (Appendix D) |
| `acquiring` | a list of full function names | empty | Additions to the acquiring functions (Appendix D) |
| `allow_network` | a list of full function names | empty | Removals from the network functions (§5.4c) |
| `allow_acquiring` | a list of full function names | empty | Removals from the acquiring functions (§5.4c) |
| `strict` | `bool` | `false` | Strict mode (§5.4d) |
| `strict_exempt` | a list of leaf names | empty | Leaves that stay in ordinary mode (§5.4d) |
| `strict_interfaces` | a list of full interface type names | empty | Interfaces that a port signature can carry in strict mode (§5.4d, K0123) |
| `strict_shared` | a list of full type names | empty | Types that a library `var` can have in strict mode (§5.4d, K0210) |
| `mutators` | a list of full function names | empty | Additions to the global mutators (Appendix D, K0211) |
| `duplicate_similarity` | percent | 90 | §9 |
| `duplicate_min_tokens` | tokens | 50 | §9 |
| `on_contract_failure` | `panic` or `exit` | `panic` | What a contract failure does to the process; the flag `--on-contract-failure` overrides it (§7.4) |

### B.3 Additions to Go source

Code files and test files follow Go's grammar, with these changed productions:

```ebnf
FunctionDecl  = [ "pure" ] "func" FunctionName [ TypeParameters ] Signature
                { Contract } [ FunctionBody ] .
MethodDecl    = [ "pure" ] "func" Receiver MethodName Signature { Contract } [ FunctionBody ] .
Contract      = ( "requires" | "ensures" ) ContractExpr .
TypeDef       = identifier [ TypeParameters ] Type { "invariant" ContractExpr } .
FieldDecl     = ( IdentifierList Type | EmbeddedField ) [ Limit ] [ Tag ] .
Limit         = "limit" Expression { "," Expression } .
TopLevelDecl  = Declaration | FunctionDecl | MethodDecl | Handler .
Handler       = [ "idempotent" ] "handle" identifier
                "(" [ identifier Type "," ] identifier Type ")" "error" FunctionBody .
ContractExpr  = Expression .   // with "=>" and the names below
```

- **Contract expressions** use Go's `Expression` grammar with one more binary operator, `=>`. It
  has a new lowest precedence level, below `||`, and it groups to the right. It is valid at every
  level of a contract expression, also inside parentheses and call arguments:
  `a && (b => c)`.
- **In a contract expression**, these names are also valid: `err`, `result`, `result.N`, and
  `old(Expression)`.
- **`invariant`** is valid only on a struct type declared at package level, in a `type`
  declaration that is not a parenthesized group (K0513).
- **Each bound of `limit`** is an integer constant expression, one for each variable-size part of
  the field's type (§5.5.1).
- **A handler's name** is the name of a subscribed topic of its leaf. Its last parameter's type is
  the topic's message type. A first parameter, if present, is a `context.Context`.
- **A function declaration without a body** is valid only in Go's own cases (assembly). A contract
  needs a body.

## Appendix C: Runtime API

The runtime has three packages. A Go check compiled each signature below (Go 1.24, for generic
type aliases).

| Package | Imported by |
|---|---|
| `github.com/robot-accomplice/vulcan/vulcan` | code and tests |
| `github.com/robot-accomplice/vulcan/vulcan/rt` | generated code only (Appendix F); code that imports it is a compile error (K0118) |
| `github.com/robot-accomplice/vulcan/vulcan/transport` | transports and the generated `main` only (K0208) |

None of them reaches a network function, except through a transport that `main` gives to `rt`
(§6).

**`github.com/robot-accomplice/vulcan/vulcan`.** The types are aliases of the types in `rt`, so
generated code and code share them.

```go
package vulcan

// Topics (§5.5). Each topic of a leaf is a package-level name of type *Topic[T] or
// *Subscription[T], which the generated code creates.
type Topic[T any] = rt.Topic[T]               // func (t *Topic[T]) Publish(msg T)
type Subscription[T any] = rt.Subscription[T] // no methods for code

// Dead letters (§5.5.1). The runtime checks Error and Wire, not Message.
type DeadLetter[T any] = rt.DeadLetter[T]     // Message T; Error string (limit 4096);
                                              // Attempts int; Wire string (limit 1024); Time time.Time

// Errors of remote method ports (§5.5.3). Match them with errors.Is.
var (
    Unavailable error   // the transport could not reach the callee
    Timeout     error   // the call did not finish in time; the outcome is unknown
    RemotePanic error   // the callee panicked; the error message holds the panic report
)

// A callee's error on the caller (§5.5.3). Its Is method matches an error from NewError with
// the same code.
type RemoteError = rt.RemoteError             // Code, Text string
func NewError(code, text string) error        // an error that keeps its identity across containers

// Contract failures (§7). Kind is "requires", "ensures", "invariant", "limit", or "panic"
// (a panic in a pure function while a contract runs). Cause is the panic value that the failure
// replaced (§7.3, rule 5), or nil; Unwrap returns Cause when it is an error.
type ContractFailure = rt.ContractFailure     // Kind, Contract, Position, Caller string; Values []Value; Cause any
type Value = rt.Value                         // Name, Text string

// The ID of the message that a handler runs for, stable across redeliveries (§5.5.0). It returns
// "" outside a handler.
func DeliveryID(ctx context.Context) string

// Pure helpers for contracts (§7.1).
func All[T any](xs []T, p func(T) bool) bool
func Any[T any](xs []T, p func(T) bool) bool

// Tests (§8). Each panics outside a test of the leaf that owns the name.
func Bind[P any](t testing.TB, port *P, impl any)
func Published[T any](t testing.TB, topic *Topic[T]) []T
func Deliver[T any](t testing.TB, s *Subscription[T], msg T) error
```

`Bind` takes `impl` as `any`, because Go's type inference does not convert a concrete type to the
port's interface type. The compiler checks `impl` as an assignment to the port (K0108), and wraps
it so that the port's contract is checked (Appendix F).

**`github.com/robot-accomplice/vulcan/vulcan/rt`.** The generated code uses it to create topics,
register handlers, and report contract failures:

```go
package rt

func NewTopic[T any](name string, validate func(*T) error) *Topic[T]
func NewSubscription[T any](name string, validate func(*T) error) *Subscription[T]
func Handle[T any](s *Subscription[T], h func(ctx context.Context, m T) error)
func Startup(leaf string, f func(ctx context.Context) error)
type Method func(ctx context.Context, req []byte) ([]byte, error)
func ServeMethod(wire, method string, m Method)
func Call(ctx context.Context, wire, method string, req []byte) ([]byte, error)
func Fail(kind, contract, position, caller string, values ...Value) // panics with *ContractFailure
```

**`github.com/robot-accomplice/vulcan/vulcan/transport`.** A transport is a Go package that the
declaration `transport x { go "<path>" }` names. It exports:

```go
type Settings struct { ... }          // the fields that `via x{ ... }` sets, with vulcan tags
type ProcessSettings struct { ... }   // filled from --transport.x.<field> flags
func New(p ProcessSettings) (transport.Transport, error)   // called once for each process
var Guarantees = transport.Guarantees{ ... }
```

The tags on `Settings` fields are `vulcan:"required"`, `vulcan:"topic"`, `vulcan:"call"`,
`vulcan:"size"`, and `vulcan:"secret"` (§5.5.1). The shared types:

```go
package transport

type Route struct{ Container, Leaf, Topic string }   // a dead-letter route

var ErrInvalid error   // a received message failed its invariant or limits

type Order int
const (
    Unordered Order = iota
    Ordered
)
type Guarantees struct {      // a transport declares it as a constant composite literal
    Redelivers     bool    // can deliver a message more than once: handlers must be idempotent
    Order          Order
    RequiresLimits bool    // message types need limits (K0304)
    Calls          bool    // supports remote method ports
}

type Endpoint struct{ Container, Leaf, Name string }   // Name: the port or topic
type Kind int
const (
    TopicWire Kind = iota
    CallWire
)
type Wire struct {
    Name     string     // the text of the wire in system.vulcan
    Kind     Kind
    From, To Endpoint   // From: the publisher or the caller; To: the subscriber or the callee
    Settings any        // this wire's Settings value, after the config defaults
}

// Dead hands a message that the transport gives up on back to the runtime, which builds the
// DeadLetter[T] and publishes it on the wire's route.
type Dead func(ctx context.Context, msg []byte, lastErr string, attempts int) error

// Messages and calls are bytes in the wire format of §5.5.2.
type Transport interface {
    Publish(ctx context.Context, w Wire, msg []byte) error
    Subscribe(ctx context.Context, w Wire,
        deliver func(ctx context.Context, msg []byte) error, dead Dead) error
    Call(ctx context.Context, w Wire, method string, req []byte) ([]byte, error)
    Serve(ctx context.Context, w Wire,
        handle func(ctx context.Context, method string, req []byte) ([]byte, error)) error
    Close() error
}
```

- The runtime calls `Publish` once for each subscriber wire of the topic. It returns an error
  only when the transport cannot accept the message, and the runtime turns that error into a panic
  at the publish (§5.5). The `ctx` is the process context, which is cancelled when the process
  stops.
- `deliver` returns `nil` when the handler acknowledged the message. On an error, the transport
  applies its policy. A message that fails its invariant or limits at receipt makes `deliver`
  return an error that wraps `transport.ErrInvalid`; the transport then calls `dead` at once,
  without retries.
- `Call` returns `vulcan.Unavailable` or `vulcan.Timeout`, wrapped or not. The runtime turns a
  callee panic into `vulcan.RemotePanic`.

## Appendix D: Network and acquiring functions

A function **acquires** a resource when it chooses an I/O target by name (a path, an address, a
URL, or a command) or creates a handle to one. A function that reads or writes through a handle
that it receives does not acquire. §5.4c defines "reaches".

**Network functions.** Only gateway and entry leaves can reach them (§5.4c, rule 1):

| Package | Functions |
|---|---|
| `net` | `Dial`, `DialTimeout`, `DialTCP`, `DialUDP`, `DialIP`, `DialUnix`, `Listen`, `ListenPacket`, `ListenTCP`, `ListenUDP`, `ListenIP`, `ListenUnix`, `ListenUnixgram`, `ListenMulticastUDP`, `FileConn`, `FileListener`, `FilePacketConn`, `(*Dialer).Dial`, `(*Dialer).DialContext`, `(*ListenConfig).Listen`, `(*ListenConfig).ListenPacket`, `Lookup*`, `(*Resolver).Lookup*` |
| `crypto/tls` | `Dial`, `DialWithDialer`, `Listen`, `(*Dialer).Dial`, `(*Dialer).DialContext` |
| `net/http` | `Get`, `Head`, `Post`, `PostForm`, `(*Client).Do`, `(*Transport).RoundTrip`, `(*Client).Get`, `(*Client).Head`, `(*Client).Post`, `(*Client).PostForm`, `ListenAndServe`, `ListenAndServeTLS`, `Serve`, `ServeTLS`, `(*Server).ListenAndServe`, `(*Server).ListenAndServeTLS`, `(*Server).Serve`, `(*Server).ServeTLS` |
| `net/smtp` | `Dial`, `SendMail` |
| `net/rpc` | `Dial`, `DialHTTP`, `DialHTTPPath` |
| `database/sql` | `Open`, `OpenDB` |
| `os/exec` | `Command`, `CommandContext` |
| `os` | `StartProcess` |
| `syscall` | `Socket`, `Connect`, `Bind`, `Listen`, `Accept`, `Exec`, `ForkExec`, `StartProcess`, and their variants on each target |

**Serve and dial.** Strict mode (§5.4d, K0125) splits the network functions. The *serve*
functions are `net.Listen`, `net.ListenPacket`, `net.ListenTCP`, `net.ListenUDP`, `net.ListenIP`,
`net.ListenUnix`, `net.ListenUnixgram`, `net.ListenMulticastUDP`, `net.FileListener`,
`(*net.ListenConfig).Listen`, `(*net.ListenConfig).ListenPacket`, `crypto/tls.Listen`, the
`ListenAndServe`, `ListenAndServeTLS`, `Serve`, and `ServeTLS` functions and methods of `net/http`,
and `syscall.Bind`, `syscall.Listen`, and `syscall.Accept`. Every other network function is a
*dial* function.

**Acquiring functions.** Only leaves can reach them (§5.4c, rule 2):

| Package | Functions |
|---|---|
| `os` | `Open`, `OpenFile`, `Create`, `CreateTemp`, `MkdirTemp`, `OpenRoot`, `OpenInRoot`, `NewFile`, `Pipe`, `DirFS`, `ReadFile`, `WriteFile`, `ReadDir`, `Remove`, `RemoveAll`, `Rename`, `Mkdir`, `MkdirAll`, `Truncate`, `Chmod`, `Chown`, `Lchown`, `Chtimes`, `Link`, `Symlink`, `Readlink`, `Stat`, `Lstat`, `Chdir`, `CopyFS` |
| `io/ioutil` | `ReadFile`, `WriteFile`, `ReadDir`, `TempFile`, `TempDir` |
| `path/filepath` | `Walk`, `WalkDir`, `Glob`, `EvalSymlinks` |
| `plugin` | `Open` |
| `syscall` | `Open`, `Openat`, `Creat`, `Mkdir`, `Mkdirat`, `Rmdir`, `Unlink`, `Unlinkat`, `Rename`, `Renameat`, `Link`, `Symlink`, `Chdir`, `Chroot`, `Mount`, and their variants on each target. `vulcan audit` lists every other use of `syscall` (§5.4c). |

A project adds a function with its full name in `config { network: [...] }` or
`config { acquiring: ["example.com/blob.Open"] }`, and removes one with `allow_network` or
`allow_acquiring` (§5.4c). `vulcan audit` lists each change and each call to a listed function.

**Global mutators.** In strict mode, only exempt leaves can reach them (§5.4d, K0211). They change
package state of the standard library that every leaf in the process shares:

| Package | Functions |
|---|---|
| `os` | `Setenv`, `Unsetenv`, `Clearenv`, `Chdir` |
| `net/http` | `Handle`, `HandleFunc` |
| `expvar` | `Publish`, `NewInt`, `NewFloat`, `NewMap`, `NewString` |
| `flag` | `Parse`, `Set`, and each function that defines a flag on `flag.CommandLine` |
| `log` | `SetOutput`, `SetFlags`, `SetPrefix` |
| `log/slog` | `SetDefault`, `SetLogLoggerLevel` |
| `runtime/debug` | `SetGCPercent`, `SetMemoryLimit`, `SetMaxThreads`, `SetMaxStack` |

A project adds a function with `config { mutators: [...] }`.

## Appendix E: Diagnostics

Each diagnostic has a stable code (§9). A code is never reused for a different problem. **E** is
a compile error. **W** is a warning, which never stops a build (§6). `vulcan explain <code>` gives
the long form.

| Code | | Problem | Section |
|---|---|---|---|
| **K00** | | **Files and syntax** | |
| K0001 | E | Syntax error in a declaration file or a contract | App. B |
| K0008 | E | A Go syntax or type error in a body; the message is Go's | §6 |
| K0002 | E | A code file outside a leaf or a library, or a leaf inside a leaf | §5.4 r2 |
| K0003 | E | Directories and `system.vulcan` disagree | §5.2 |
| K0004 | E | A `package` clause does not name its directory, or a `library` name is not its package name | §5.4a, App. B |
| K0005 | E | A declared name, or the name of a code file, starts with `vulcan_` or `Vulcan_` | App. B |
| K0006 | E | Unknown key in `config { }` | App. B |
| K0007 | E | A `transport` name is not lower case | App. B |
| K0009 | E | A construct that this release does not support; the message names the construct, the release, and the spec section | [release 1](release-1/syntax.md) |
| **K01** | | **Architecture** | |
| K0101 | E | A leaf refers to another leaf other than through a port, a topic, or a library | §5.4 r1 |
| K0102 | E | A method port wired across containers is not `remote`, or a `remote` port is wired inside one container | §5.4 r3 |
| K0103 | E | A topic wire connects two leaves of one container | §5.4 r3 |
| K0104 | E | A required port or a subscribed topic is not wired | §5.4 r4 |
| K0105 | E | A required port or a subscribed topic is wired more than once | §5.4 r4 |
| K0106 | E | A published topic is wired by itself | §5.4 r4 |
| K0107 | E | A wire names a type or a constructor that does not exist or is not exported | §5.3 |
| K0108 | E | The method set of the supplier does not match the port | §5.4 r5 |
| K0109 | E | The provider's own contract disproves the port's contract | §5.3 |
| K0110 | E | A port is declared outside the leaf that requires it, or twice, or with the name of a declaration of its leaf; a port method is not exported; or a boundary, or `system.vulcan`, names another leaf or imports something other than a library | §5.4 r7 |
| K0111 | E | A cycle in the wires of one container | §5.4 r8 |
| K0112 | E | A boundary uses a leaf-private type, or a topic message or remote signature uses a named type that is not from a library (fix-it: move it to a library) | §5.4 r9 |
| K0113 | E | A wire names a group | §5.3 |
| K0114 | W | A leaf is larger than `context_budget` | §5.4 r10 |
| K0115 | W | Dead code, or a port that no code calls | §9 |
| K0116 | E | Code assigns to a port or a topic name, or takes its address, other than in `vulcan.Bind` | §5.4 r11 |
| K0117 | E | A leaf declares an `init` function | §5.4 r12 |
| K0118 | E | Code imports `github.com/robot-accomplice/vulcan/vulcan/rt` | App. C |
| K0119 | E | A `uses` line names an unknown node | App. B |
| K0120 | E | A port contract names an identifier of its own leaf | App. F |
| K0121 | E | Two leaves in the system have the same name | §5.3 |
| K0122 | E | A leaf or a library contains a `//go:linkname` directive | §5.4 r13 |
| K0123 | E | Strict mode: a port signature carries a function, a channel, or an interface type that is not allowed | §5.4d |
| K0124 | E | Strict mode: code imports `unsafe` or `syscall`, uses cgo, or calls a `reflect` call function | §5.4d |
| K0125 | E | Strict mode: an entry leaf reaches a dial function, or a leaf that is not a gateway dials | §5.4d |
| **K02** | | **Libraries, packages, and resources** | |
| K0201 | E | A library declares a port or a topic | §5.4a r1 |
| K0202 | E | A library `var` is assigned after startup | §5.4a r1 |
| K0203 | E | A library reaches an acquiring function | §5.4a r2, App. D |
| K0204 | E | A library imports a leaf, or library imports form a cycle | §5.4a r3 |
| K0205 | E | A leaf that is not a gateway or an entry leaf, or a library, reaches a network function | §5.4c r1 |
| K0206 | E | `gateway for` names something that is not an `external`, or `entry for` names something that is not a `person` or an `external` | §5.4b |
| K0207 | E | Two gateway leaves for one external system in one container | §5.4b |
| K0208 | E | A leaf or a library imports a transport package | §5.5.1 |
| K0209 | E | A dependency is not in `go.mod` or `go.sum` (fix-it: `vulcan mod tidy`) | §6 |
| K0210 | E | Strict mode: the type of a library `var` holds a mutable reference | §5.4d |
| K0211 | E | Strict mode: code reaches a global mutator | §5.4d, App. D |
| **K03** | | **Topics, transports, and remote ports** | |
| K0301 | E | A subscribed topic has no handler or more than one, a handler has no subscribed topic, or a handler's parameters do not match | §5.5.0 |
| K0302 | E | A handler is not `idempotent`, and its transport can redeliver | §5.5 |
| K0303 | W | A published topic has no subscriber | §5.5 |
| K0304 | E | A variable-size field of a message type has no `limit`, and the transport requires one | §5.5.1 |
| K0305 | E | A message or remote type is not serializable (a pointer, an interface other than a remote method's `error`, a function, a channel, a topic, another map key type, or a recursive type), or a topic message type is not a struct | §5.5 |
| K0306 | E | The two ends of a topic wire have different message types | §5.5 |
| K0307 | E | A transport setting is missing, unknown, or of the wrong type | §5.5.1 |
| K0308 | E | A topic setting on a call wire, or a call setting on a topic wire | §5.5.1 |
| K0309 | E | A size setting from a flag has no `max` | §5.5.1 |
| K0310 | E | A remote method has no timeout, and `config` sets no default | §5.5.3 |
| K0311 | E | A breaking change to a locked message type or remote port (fix-it: a new type) | §5.5.2 |
| K0312 | E | A retired type name is used again | §5.5.2 |
| K0313 | E | A `deadletters` route is not a subscribed `DeadLetter[T]` topic with the wire's `T` | §5.5.1 |
| K0314 | E | A wire across containers has no `via`, and `config` sets no `transport` | §5.5.1 |
| K0315 | E | The last result of a remote method is not `error` | §5.5.3 |
| K0316 | E | A wire is longer than 1,024 bytes, the limit of `DeadLetter.Wire` | §5.5.1 |
| K0317 | E | A `limit` on a fixed-size field, a bound that is not an integer constant, or the wrong number of bounds | §5.5.1 |
| K0318 | E | `timeout` on a method of a port that is not `remote` | §5.5.3 |
| K0319 | W | One lock change removes a field and adds another of the same type (a rename loses data) | §5.5.2 |
| K0320 | E | A required setting is given in `config` | §5.5.1 |
| K0321 | E | `topics.lock` was edited by hand (its hash does not match) | §5.5.2 |
| **K04** | | **Compiler intelligence** | |
| K0412 | W | Near-duplicate code (fix-it: reuse, or extract to a library) | §9 |
| **K05** | | **Contracts** | |
| K0501 | E | The compiler proves that a contract fails (a call, a literal, or a return) | §7 |
| K0502 | E | `result` is used without an `err == nil =>` guard | §7.1 |
| K0503 | E | A contract calls a function that is not `pure`, or has an effect | §7.1 |
| K0504 | E | An index, a slice expression, a pointer dereference, a divisor, or a shift count in a contract is not proved safe | §7.1 |
| K0505 | E | A `pure` function has an effect, or calls a function that is not `pure` | §7.1 |
| K0506 | E | The zero value breaks an invariant | §7.2 |
| K0507 | E | A field of a struct with an invariant is written outside its methods | §7.2 |
| K0508 | E | The address of such a field is taken outside its methods | §7.2 |
| K0509 | E | `old(x)`, where the type of `x` holds a reference | §7.1 |
| K0510 | E | `result.N` is out of range, plain `result` with two or more other results, or `result` in a function with no other result | §7.1 |
| K0511 | E | A function with a contract names a parameter or a result `err` or `result` | §7.1 |
| K0512 | E | A type assertion in a contract | §7.1 |
| K0513 | E | An `invariant` on a type that is not a struct declared at package level outside a `type ( )` group | §7.2, App. B |
| K0514 | E | A function with a contract has no body | App. B |
| K0520 | W | A `go` statement in a method of a struct with an invariant | §7.2 |
| K0521 | W | A method of a struct with an invariant unlocks a mutex other than by `defer` | §7.2 |
| K0522 | E | A deferred call in a method of a struct with an invariant writes a field of that struct | §7.3 |
| K0523 | E | An `ensures` conjunct reads a named result and passes state by reference to a `pure` function, in a function with a risky deferred call | §7.3 |
| K0524 | W | The check after a recovered panic runs after a deferred unlock | §7.3 |
| K0525 | W | A `return` reaches an effect in a function with `ensures` or in a method of a struct with an invariant; the effect runs before the check | §7.4 |
| **K06** | | **Startup, settings, and flags** | |
| K0601 | E | A function named `setup` or `startup` has the wrong signature | §6 |
| K0602 | E | One flag name is used with two types, defaults, or maxima, or two flag names give one environment variable | §5.3 |
| K0603 | E | Settings for a leaf whose `startup` takes none, or a missing settings value | §5.3 |
| K0604 | E | A constructor has parameters other than one settings struct, or results other than `T` or `(T, error)`; or a wire that names a type has settings | §5.3 |
| K0605 | E | A literal, or a flag default, in a settings field tagged `vulcan:"secret"` | §5.3 |
| K0606 | E | A settings value, a flag default, or a `config` value is not a constant of its field's type; a value can use only constants, the package `time`, and the libraries that `system.vulcan` imports | §5.3, App. B.2 |
| K0607 | W | One constructor is wired with two different settings, so `main` makes two instances; an error in strict mode, unless the supplier leaf is exempt | §5.3, §5.4d |
| **K07** | | **Warnings on Go code** | |
| K0701 | W | Unhandled error | §6 |
| K0702 | W | Error masking | §6 |
| K0703 | W | Possible `nil` dereference | §6 |
| K0704 | W | Data race (from `vulcan test`; the test also fails, as in Go) | §6 |
| K0705 | W | `vulcan test` ran without the race detector, because the host cannot run it | §6 |
| K0706 | E | `go vet` reports a finding in the code (the PoC gives both arms of its evaluation the same Go analysis) | §6 |

## Appendix F: Lowering to Go

The Go backend (§11) turns each Vulcan construct into Go as follows. The generated code has
`//line` directives, so errors and stack traces point to `.vulcan` files. Names that start with
`vulcan_` are generated.

**Leaves and libraries.** Each leaf and each library is one Go package. Each `.vulcan` code file
becomes one `.go` file with the Vulcan additions removed or replaced as below.

**Ports.** For `needs port Checkout { ... }` in leaf `orders`, the compiler generates, in the
package `orders`:

```go
type vulcan_Checkout interface { Pay(c money.CustomerID, amt money.Money) (money.Receipt, error) }
var Checkout vulcan_Checkout            // bound by main, or by vulcan.Bind in a test
```

For each wire, the generated `main` holds a wrapper for that port and that provider type. The
wrapper checks the port's contract and calls the provider's method directly:

```go
type vulcan_wire3 struct{ impl *billing.Payer }   // Orders.Checkout: Billing.NewPayer
func (w vulcan_wire3) Pay(c money.CustomerID, amt money.Money) (money.Receipt, error) {
    if !(amt > 0) { rt.Fail("requires", ...) }
    r, err := w.impl.Pay(c, amt)                // a static call
    if err == nil && !(r.Amount == amt) { rt.Fail("ensures", ...) }
    return r, err
}
// in main: orders.Checkout = vulcan_wire3{impl: billing.NewPayer()}
```

Thus a port call is one dynamic dispatch, on `Checkout` (§5.3). In a test, the compiler rewrites
`vulcan.Bind(t, &Checkout, f)` to bind a wrapper generated in `orders` whose `impl` has the type
`vulcan_Checkout`; that costs a second dispatch in tests only. Thus a fake meets the same contract
as the real provider (§8). A remote port binds a client wrapper that encodes each call and calls
`rt.Call`. The callee's container registers the provider's methods with `rt.ServeMethod`.

**Topics.** `publishes Placed topic[events.OrderPlaced]` becomes
`var Placed = rt.NewTopic[events.OrderPlaced]("Orders.Placed", events.Vulcan_validateOrderPlaced)`,
of type `*vulcan.Topic[events.OrderPlaced]`. `subscribes` becomes `rt.NewSubscription` in the same
way. A message type is in a library (§5.4, rule 9), so its validation function, which checks its
`limit`s and its invariant, is generated in that library. For `vulcan.DeadLetter[T]`, `rt` has one
generic validation function, which checks `Error` and `Wire` only. A handler
`handle Shipped(m events.ShipmentSent) error { ... }` becomes
`func vulcan_handle_Shipped(vulcan_ctx context.Context, m events.ShipmentSent) error { ... }`; the
added parameter has a reserved name, so it hides no name that the body uses. The leaf's generated
registration calls `rt.Handle(Shipped, vulcan_handle_Shipped)`.

**Every generated identifier starts with `vulcan_` or `Vulcan_`** (Appendix B.1), so it cannot
collide with a name in the code.

**Contracts on functions.**

1. `requires` becomes a check at entry: `if !(expr) { rt.Fail(...) }`. `rt.Fail` panics with a
   `*vulcan.ContractFailure` (Appendix C).
2. `old(x)` becomes a local variable `vulcan_old0`, set at entry by Go assignment (§7.1).
3. `ensures` is checked at each `return`, before the function's deferred calls run (§7.2). Each
   `return e0, e1` becomes a block with typed temporaries:

   ```go
   {
       var vulcan_t0 T0
       var vulcan_t1 T1
       vulcan_t0, vulcan_t1 = e0, e1
       if !(ensures with result = vulcan_t0, err = vulcan_t1) { rt.Fail("ensures", ...) }
       return vulcan_t0, vulcan_t1
   }
   ```

   A bare `return` checks the named results. If the end of the body can be reached, the checks
   run there too. This form compiles where a named result is shadowed, and it adds no name to the
   body's scope.

   A function with a risky deferred call (§7.3, rule 3) runs its body in a generated closure, and
   checks again after the closure returns. Each return check in the body also sets `vulcan_checked`
   and saves the state atoms of §7.3, rule 3:

   ```go
   func (a *Account) Deposit(f *os.File, x int) (err error) {
       var vulcan_checked bool
       var vulcan_s0 int                          // a.Balance, saved at the return check
       vulcan_old0 := a.Balance
       err = func() (err error) {
           // the body, unchanged, with its deferred calls
       }()
       if vulcan_checked {
           if !(!(err != nil) || vulcan_s0 == vulcan_old0) { rt.Fail("ensures", ...) }
       } else {
           // a recovered panic: ensures and the invariant on the current state (§7.3, rule 4)
       }
       return err
   }
   ```

   A deferred `recover` in the body is still called directly by a deferred function, so it works as
   in Go. A panic that leaves the closure leaves the generated function too.
4. `a => b` becomes `!a || b`.
5. **Exact arithmetic.** Integer arithmetic in a contract expression gives the mathematical
   result (§7.1). The compiler uses machine integers with an overflow check, and changes to
   `math/big` only when the check fails. Thus the usual cost is one branch. The body of a `pure`
   function keeps Go arithmetic.
6. A panic in a `pure` function while a contract runs is recovered and reported with
   `rt.Fail("panic", ...)`.
7. The compiler removes a check that it proves, and `vulcan audit` shows each removal.

**Struct invariants.** For a struct type `T` with an invariant, the compiler generates, in the
package of `T`, the exported method `func (v *T) Vulcan_invariant()`, which does nothing for a nil
`v` and calls `rt.Fail` when the invariant is false. It also generates `Vulcan_checkT` and
`Vulcan_checkPtrT`. They are exported, so a leaf that uses a library type can call them.

- **Outer calls.** Each method `M` is generated twice: `M`, which checks the invariant at each
  `return` and at a reachable end of its body, before its deferred calls run; and `vulcan_M`, the
  same body without these checks. `M` also gets one generated deferred call, registered first so
  that it runs last. If a panic leaves `M` before a return check ran, it checks the invariant, and
  when the invariant is false it panics again with a `*vulcan.ContractFailure` whose `Cause` is the
  original value (§7.3, rule 5). Inside a method of `T`, a call whose receiver expression is the
  method's own receiver variable, not assigned in the method, calls `vulcan_M`. Every other call,
  for example `to.Deposit(x)`, calls `M` (§7.2). A method that writes no field, calls no method that
  does, and passes no pointer to `T` to a function has no check.
- **Literals and conversions.** `T{...}` and a conversion `T(x)` become `Vulcan_checkT(...)`, which
  checks the invariant and returns the value. `&T{...}` becomes `Vulcan_checkPtrT(&T{...})`, which
  checks `*p` and returns the pointer `p`.
- **Calls that receive `T`.** A call that passes a pointer to `T`, or a value that holds `T`
  through a pointer, to a function that is not a method of `T` is followed by a check of each `T`
  that it can reach (§7.2).
- **The zero value.** The compiler evaluates a tier-1 invariant on the zero value at compile time
  (K0506). A tier-2 invariant runs in a generated `init` of the library or leaf, so a failure
  stops the process before `main` starts, and `vulcan test` of that leaf or library reports it.
  (§5.4, rule 12 forbids `init` that code declares, not generated `init`.) For a generic struct,
  see §7.2.

**Port contracts.** The wrapper of a port is generated in `main` (see Ports). Thus a port contract
in `component.vulcan` can name only predeclared identifiers, the standard library, and libraries,
including their `pure` functions; a name of the leaf itself is a compile error (K0120).

**`pure` functions.** The compiler checks the body of a `pure` function with the rules of §7.1
(K0505). Vulcan ships the list of `pure` functions of the Go standard library, for example `len`,
`strings.HasPrefix`, `slices.Contains`, `slices.IsSorted`, and `math.Abs`. `vulcan query --pure`
prints it.

**`startup` and `main`.** The generated `main` follows the steps of §6. Each leaf gets a generated
registration function, which gives its handlers, its served ports, and its `startup` function to
the runtime through `rt`. The order of constructor calls is a topological order of the wires of
the container.
