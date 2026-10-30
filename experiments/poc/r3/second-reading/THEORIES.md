# Theories to judge

Arms: L is Vulcan's model as a Go linter (`input/poc/golint`, `input/poc/app/golint`). V is Vulcan (`input/internal`, `input/cmd`, `input/poc/app/vulcan`, `input/docs/spec.md`).

## The test

### What arm L may use

- Analyzers on unmodified Go, with whole-program facts, call graphs, and structured comments.
- Subset rules: an analyzer can ban any construct, in any part.
- Code generators whose output is ordinary Go.
- Libraries, and existing Go tools, at their real cost.

### What arm L may not use

A rewrite at build time that changes what a Go text means (`-toolexec` or `-overlay` that makes `+`
trap). Such a tool is a language with Go's grammar, and count it on the language
side. A rewrite that only picks among behaviors that Go already allows, such as one legal goroutine
schedule in a test, is a test tool, and L may use it.

### The three questions

1. **Reach.** Does L reject the same programs?
2. **Soundness.** Does L have the same trusted base and the same holes? V trusts the same Go toolchain
   and imports the same Go packages, so a hole in a dependency is shared unless a theory says how V
   closes it.
3. **Form.** Does L's code stay ordinary Go at each place of use, at no more than about 1.25 times
   V's tokens? The factor is a judgment, set before any test of it, and the tests below measure the
   real ratio.

| Verdict | Meaning |
|---|---|
| **Linter-equal** | Yes to all three |
| **Linter-partial** | Yes to reach, with a named loss in soundness or form. The test then measures whether that loss matters |
| **Language-only** | L reaches it only by banning a core Go construct in ordinary code, or not at all without changing what the Go text means |

## Theories

Holes named below: H4, a dial inside an SDK. H6, a body-less Go function with an assembly file in a leaf passes both checks. H7, `net/textproto.Dial` in a plain leaf passes both checks.

### T1. A closed list of what code may use

An allowlist of packages and functions per kind of part, in place of the list of banned functions, closes the holes that come from an incomplete list (H6, H7).

### T2. Reach through dependencies

A check that follows calls into other modules closes hole H4, a dial inside an SDK.

### T3. Leaves in separate memory

A leaf in its own process or Wasm module cannot use `unsafe`, cgo, or a dependency to go around the rules, because the boundary is enforced at run time.

### T4. Static proof of contracts

A verifier that proves `requires`, `ensures`, and invariants from the body moves value rules from run time to check time.

### T5. A body meaning that proofs can use

With the same verifier, bodies with trapping integers and value semantics are proved more often than bodies with Go's meaning, because the proof has fewer cases.

### T6. The agent pays for the proof

Annotations were too costly for people, but an agent can write loop invariants and lemmas at a price worth paying, and a proof language built for this raises its success rate.

### T7. Illegal states that cannot be built

Sum types with exhaustive `match`, types with no zero value, and branded values cost few tokens in a language, so agents use them, and value rules hold by construction.

### T8. Safer meanings in bodies

Integer operations that trap, `append` that does not alias, and fixed map order remove defects that agents write, and that no local check can find.

### T9. Labels on values (secrets, personal data)

A type label on a value proves that it never reaches a log, a response, or an external.

### T10. Declared state machines

A declared table of legal transitions, checked at each write of a status field, rejects a write that does not follow from a known state.

### T11. Linear typestate and resources

A value that is used up by a transition, or a resource that must be closed, cannot be used twice or forgotten, by type.

### T12. No ambient authority

If code can reach the network, the files, or the clock only through a value that it was given, then a signature tells what a part can do, and rules on authority follow from the wiring.

### T13. Effects in function types

Effects in signatures, with inference and with polymorphism over function values, check each part by itself, and make what a function does visible in the signature.

### T14. Declared dependency boundaries

A table of which part may depend on which rejects each new wire that the team did not allow.

### T15. Declared ownership of data

A declaration of which part owns which table or file rejects a second writer, and moves G12 into the model.

### T16. Wire contracts in one compile unit

One compiler that sees both sides of a topic or remote port rejects an incompatible change before deployment.

### T17. Proved idempotency and ordering

An analysis of a handler's effects proves the `idempotent` mark, which the spec today only trusts, and rejects a handler that needs an order that the transport does not give.

### T18. Durable execution from the compiler

If the compiler lowers a handler so that it resumes after a crash, with each effect recorded once, then crash rules hold with no special code, and an agent need not reason about them.

### T19. Deterministic simulation

If time, randomness, map order, and goroutine schedules come from one seed under test, then an agent can replay a failure, and find concurrency faults that it cannot find today.

### T20. Race freedom by type

Types that say what may cross between goroutines reject data races at check time.

### T21. The cost of a correct program

With the same guarantee, a language lets an agent write less and read less, because the compiler generates what L must write out.

### T22. Machine-applicable fixes

A diagnostic with an exact edit reduces the turns from a rejection to a pass.

### T23. One way to write a thing, and a small primer

Fewer forms and one model make an agent more often right on the first attempt.

### T24. Typed holes

A check that accepts an unfinished program and answers with the type and the ports in scope at a hole reduces invented APIs (R4) and turns.

### T25. Local reasoning

If a leaf and its ports hold all that an edit needs, an agent reads fewer files and uses less context.

### T26. Tests derived from contracts

A tool that derives property tests, double delivery, and crash points from contracts turns many run-time contracts into failures that an agent sees before a change is done.

## Theories of harm

Judge which form carries the cost: Language-only if only V carries it, Linter-equal if the same feature costs the same in either form, Linter-partial if L carries part of it.

### N1

A new language costs success and turns, because the model has no training data on it

### N2

Annotations, colored functions, and solver timeouts cost more turns than they save

### N3

More rejections lead to longer runs, and runs that do not stop

### N4

Agents pass a strong check by weakening the thing checked: a contract removed, a marker added

### N5

Text that looks like Go and means something else misleads a model trained on Go

### N6

One central file (`system.vulcan`, or `arch.go` and `main.go`) causes conflicts between parallel agents
