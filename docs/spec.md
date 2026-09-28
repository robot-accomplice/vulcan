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
| `system`, `container`, `group`, `component`, `port`, `person`, `external`, `layer` declarations | Architecture as a first-class construct (§5) |
| Unhandled errors are compile errors; postfix `?` propagation | R3 (§7) |
| No `nil`/`null` value; optional types `?T` with flow narrowing | No null dereferences, ever (§7a) |
| No `recover()`; no exceptions of any kind | Errors are values only (§7) |
| `requires` / `ensures` / `invariant` clauses | Embedded contracts (§8) |
| `test` and `fake` blocks | Inherent testability (§9) |
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

### 5.2 Layout

The directory tree mirrors the architecture tree. Each node is a directory holding exactly one
declaration file named for its kind: `system.keel`, `container.keel`, `group.keel`, or
`component.keel`. A composite directory contains no source files; a leaf directory contains no child
nodes. There is one place to look for everything.

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
    provides Payments from Billing      // re-exported; Billing itself stays hidden
    requires Ledger                     // passed up to the system

    Billing     { Ledger: parent.Ledger, Clock: SystemClock.Clock }
    Orders      { Checkout: Billing.Payments }   // compatibility checked here
    SystemClock {}
}
```

The system adds C4's outer ring — the people and external systems it interacts with:

```go
// Shop is the online store.
system Shop {
    layers domain < app < infra

    person   Customer "Buys goods through the web store"
    external Stripe   { provides CardNetwork }

    Api    { Ledger: Worker.Ledger, CardNetwork: Stripe.CardNetwork }
    Worker {}

    uses Customer -> Api "places orders over HTTPS"
}
```

### 5.4 Rules the compiler enforces

1. **Ports only.** Code in a leaf may use another node **only** through a port listed in its
   `requires`. Any other cross-node reference is a compile error.
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

### 5.5 Diagrams

`keel diagram --level context|container|component|code [--focus <node>]` renders each C4 level from
the declarations above. Because composites hold no code, a diagram at any zoom level shows where all
behavior lives — a parent cannot hide logic.

## 6. Type system

- **Strong and static.** No implicit conversions.
- **Inferred wherever possible.** Locals, closures, and generic instantiations are inferred.
  Port signatures and exported functions are explicit — they are the contract surface humans review.
  (Whether private function signatures may be inferred is Open Question Q3.)
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

**There are no exceptions, no `try`/`catch`, no `throw`, and no `recover()`.** An error is an ordinary
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
- **Aborts are not exceptions.** A broken invariant (contract violation, `unreachable`) terminates
  the task with a diagnostic report. Nothing can intercept an abort, so it can never be used for
  control flow. Supervising and restarting failed tasks is a runtime concern, not a language one.

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

## 8. Contracts

Contracts are clauses on declarations, not comments:

| Clause | Attaches to | Meaning |
|---|---|---|
| `requires expr` | functions, port methods | precondition, caller's obligation |
| `ensures expr` | functions, port methods | postcondition; `result` names the return value |
| `invariant expr` | structs | holds after every exported method returns |

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
| `keel doc` / `keel diagram` | Docs and C4 diagrams at every level — context, container, component, code — generated from source (§5.5). |
| `keel audit` / `keel archdiff` | Contract and architecture audit (§8, §10). |
| `keel fmt` | The one formatter. |

All commands have `--json` output. An LSP server and an MCP server expose the same capabilities to
editors and agents.

## 12. Runtime

- Native compilation, tracing GC. Backend choice is Open Question Q4.
- Concurrency model follows Go (goroutine-style tasks, channels) unless a later decision changes it.

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
| Q8 | Out-of-range indexing and integer overflow: abort (Go-like) or return `?T` / error? |
| Q3 | Inference boundary: may private function signatures omit types? |
| Q4 | Backend: LLVM, QBE, C, or transpile to Go for a fast first prototype. |
| Q5 | Compiler implementation language (bootstrap). |
| Q10 | Wiring across containers crosses a process boundary: how is the transport (RPC, queue) declared, and do contracts still hold across it? |
| Q7 | Contract language: which expressions are allowed in contracts (pure subset only?). |

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
