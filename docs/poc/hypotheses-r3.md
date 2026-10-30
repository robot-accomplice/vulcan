# Round 3: theories of what a language could do that a linter cannot

Status: accepted by the owner as the list of theories to test, 2026-10-30, with the amendments in
section 9. Nothing here is built yet. It answers the question that [plan-r3.md](plan-r3.md) leaves
open (its section 0 and its decision 3).

Every prior in this document is a **prediction by the session that wrote it**, to be measured. No
result is wanted. "No language feature beats a well-built linter here" is an acceptable end, and
section 7 says what follows from it.

## Summary

**Main finding of the analysis (not yet a measurement).** A language cannot differ from Go plus a
linter in *what it can reject*. Vulcan lowers to Go, so a check on Vulcan is a check on Go. An
analyzer can also read annotations, and can ban any construct, so it can hold code to any subset.
In the limit, a Go team can embed any language in Go. So what is left to test is narrower:

1. **The cost of a correct program:** the same guarantee in fewer tokens and fewer turns.
2. **The meaning of a body:** a meaning that Go does not give, which also makes proofs easier.
3. **The size of the holes:** a guarantee that L can state only through a subset with more holes.

| Verdict (section 1) | Theories | Count | Agent harms (N1 to N6) |
|---|---|---|---|
| Language-only | T8 | 1 | 2 (N1, N5) |
| Linter-partial | T5, T6, T7, T11, T13, T18, T20, T21, T24 | 9 | 0 |
| Linter-equal | T1, T2, T3, T4, T9, T10, T12, T14, T15, T16, T17, T19, T22, T23, T25, T26 | 16 | 4 (N2, N3, N4, N6) |

The one Language-only verdict is a close call, and it depends on a line that section 1 draws. Under
a looser line, the count is 0.

**Ranked shortlist** (section 6):

| Rank | Theory | First step | Why this rank |
|---|---|---|---|
| 1 | T5, with T4 inside it: static proof of contracts, and whether a stricter body meaning raises the proof rate | One verifier, two body meanings, on at least 60 clauses, with no agent runs | Class C is 0 of 7 today, T4 can move it, and one build separates the tool's effect from the language's |
| 2 | T21: the cost of a correct program, and N1, the cost of a new language | A generator for L, then a recount, then 20 ordinary tasks | A first count already shows a gap: L has 16.8% more source than V for the same behavior and contracts |
| 3 | T8: safer meanings of body code | A trap rewriter, run on the final workspaces of calibration runs | It is the one theory that survives the linter test, and its gate costs no new agent run |
| 4 | T7: illegal states that cannot be built | `enum`, `match`, and `nozero` in V, the best equal form in L, then form and hole counts | It is the largest form gap in rule classes C and D, but the most costly build here |

*Prediction:* the most likely end is "no measured gain of the language". I put about 20% on at least
one shortlisted theory giving a replicated gain over arm L.

**Measured for this document:**

| Measure | P | L | V | How |
|---|---|---|---|---|
| Source of Stockroom-lite, no tests, no generated code (pieces) | 22,691 | 25,797 | 22,093 | `primer_count.py` on the concatenated `.go` and `.vulcan` files |
| The same, in bytes | 78,504 | 89,807 | 78,605 | |
| Check after a one-file change ([parity-r2.md](../../experiments/poc/parity-r2.md)) | 0.9 s | 1.4 s | 4.6 s | Copied from that record |

Three more findings from the code:

- Neither checker emits a machine-applicable fix. No `SuggestedFix` exists in `poc/golint` or
  `internal`.
- **New hole H6:** a body-less Go function with an assembly file in a leaf passes `vulcan check`,
  `vulcan build`, and `vulcan-lint`. I ran it on scratch copies of both apps.
- **New hole H7:** `net/textproto.Dial` in a plain leaf passes both checks, because that function is
  in neither network list. A control call of `http.Get` in the same leaf is rejected by both.

Neither hole is fixed. Both are the same in L and V.

## 1. The linter test

### What arm L may use

- Analyzers on unmodified Go, with whole-program facts, call graphs, and structured comments.
- Subset rules: an analyzer can ban any construct, in any part.
- Code generators whose output is ordinary Go.
- Libraries, and existing Go tools (appendix A), at their real cost.

### What arm L may not use

A rewrite at build time that changes what a Go text means (`-toolexec` or `-overlay` that makes `+`
trap). Such a tool is a language with Go's grammar, and this document counts it on the language
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

"L cannot do it at any cost" would be an empty class, for the reason in the summary, so the form
question draws the line. Another reader could draw it elsewhere. T8 is the verdict most exposed to
that choice, followed by T7 and T18.

### What each kind of verdict can and cannot show

- A label gate (the sets S, G, U, N) can show a difference only for Language-only theories. For a
  Linter-partial theory, L rejects the seeded cases too, so U stays empty. These theories need the
  hole and form measures below.
- A Linter-equal theory is **not a language differentiator**, even when it is worth building. It
  moves a rule class in both arms. Its test asks whether the rule class moves, and whether U stays
  empty.

## 2. What the evidence says so far

| Fact | Source | What it means for these theories |
|---|---|---|
| Every check of the PoC is "Same in Go" | [desk-test-r2.md](desk-test-r2.md) | A theory must add something that the PoC does not have |
| 3 of 28 rules enforced, the same 3 in L and V. By class: A 1, B 2, C 0, D 0 | [labels-r2.md](../../experiments/poc/labels-r2.md) | Classes C and D have the most room |
| Of the other 25: 4 declarable, 5 run time only, 16 outside the model, the same in both arms | The same record | Class A: 6 outside. Class C: 3 run time, 4 outside. Class D: 1 run time, 6 outside |
| No rule break in product code in 36 runs of plain Go with prose rules | [results-p5.md](../../experiments/poc/results-p5.md) | Enforcement had nothing to win on those tasks and that model. Each test needs plan-r2's validity gate |
| V: 8 of 36 runs failed, against 1 of 36 for G, all stopped by a budget about half the planned size | The same record | Weak evidence of a cost of the language, and N1 and N3 must be measured |
| Function bodies are plain Go (spec §4) | [spec.md](../spec.md) | T5, T7, T8, T11, and T20 need a change to this promise. It is the main design cost of a language effect |
| The invariant of a type must hold for its zero value (K0506, VL0506) | Spec §7.2, `internal/check/contracts.go` | A rule such as "a price is more than 0" cannot be an invariant of the type today, in either arm |

## 3. The theories

"Moves" names the rule classes that the theory would turn from "outside the model" or "run time"
into "rejected at check time": A boundaries, B effects, C values, D other. It says what I expect the
feature could reach, not what a test has shown.

Section 5 defines the stages that the tests name: **form**, **holes**, **labels**, and **runs**.

### Escape hatches

#### T1. A closed list of what code may use

- **Claim.** An allowlist of packages and functions per kind of part, in place of the list of banned
  functions, closes the holes that come from an incomplete list (H6, H7).
- **Linter test: Linter-equal.** Both arms use the same list form today, with the same gaps. An
  analyzer can use an allowlist and can read `Pass.OtherFiles` to reject assembly.
- **Moves.** B, for every effect that the list covers.
- **Cost.** Compiler: small, a few hundred lines in each checker. Agents: a rejection when a task
  needs a new package, and an edit to protected configuration.
- **Test.** *Measure:* attacker bypasses of "only gateways reach the network", before and after, in
  each arm (holes). *Refuted if* the count falls by no more than 1 in either arm, or the arms differ
  by more than 1 after the fix. *Size:* one attacker per arm, a fixed budget, repeated 3 times.
- **Prior (prediction).** 85% that the count falls and the arms stay equal. Both checkers read the
  same facts.

#### T2. Reach through dependencies

- **Claim.** A check that follows calls into other modules closes hole H4, a dial inside an SDK.
- **Linter test: Linter-equal.** The desk test says so, and `capslock` does this for Go. V imports
  the same modules. A language would be stronger only with its own package ecosystem, which spec §4
  and R8 reject. An allowlist of dependencies, or a table of trusted signatures, is available in
  both forms.
- **Moves.** B, for rules that an SDK could break.
- **Cost.** Compiler: facts over all dependencies. Departure 8 of the [linter README](../../poc/golint/README.md)
  expects the check to slow by an order of magnitude. Agents: slower check loop.
- **Test.** *Measure:* seeded dials in a dependency that each arm rejects, and check time. *Refuted
  if* either arm misses more than 10% of at least 20 seeds, or the arms differ by more than 1 seed.
- **Prior (prediction).** 80% that the arms stay equal. 50% that the slower loop is worth its cost.

#### T3. Leaves in separate memory

- **Claim.** A leaf in its own process or Wasm module cannot use `unsafe`, cgo, or a dependency to go
  around the rules, because the boundary is enforced at run time.
- **Linter test: Linter-equal.** A generator can do it from the manifest and `ports.go`. Service
  Weaver does this for Go. This is a deployment form, not a language property.
- **Moves.** B at run time, and part of A. Nothing new at check time. An OS sandbox could enforce part
  of G12 (no shared files) at run time.
- **Cost.** Compiler: a large change, as every port call becomes a copy and a call that can fail.
  Agents: partial failure at each boundary, and new latency.
- **Test.** *Measure:* attacker bypasses of memory sharing in each arm, and the latency of a port
  call. *Refuted if* V and L differ by more than 1 bypass. *Size:* as T1.
- **Prior (prediction).** 90% equal. 15% that a team would accept the cost for one process.

### Values

#### T4. Static proof of contracts

- **Claim.** A verifier that proves `requires`, `ensures`, and invariants from the body moves value
  rules from run time to check time.
- **Linter test: Linter-equal.** Arm L's contracts are Go expressions (`contract.Requires(cond)`) and
  V's bodies are Go, so both give the same proof obligations. Gobra shows the approach on Go. **This
  is not a language differentiator, even though it may be the most useful single feature here.** The
  spec's proof rule (§7.1) is not built in the PoC (status.md).
- **Moves.** C from run time to check time. Part of D, where a rule is a state or a sum.
- **Cost.** Compiler: an extractor, a verification-condition generator over `go/ssa`, and an SMT
  solver. Solver time joins the check loop. Agents: proof failures to read.
- **Test.** *Measure:* the share of clauses proved with no agent help, and class C rules rejected at
  check time on a fresh rule set (section 5). *Refuted if* fewer than 30% of clauses are proved, or no
  more than 1 more class C rule is rejected. *Size:* at least 60 clauses, and 28 rules. With 7 rules
  in a class the interval is wide (3 of 7 gives about 10% to 82%), so use 14 a class if a decision
  depends on the class share.
- **Prior (prediction).** 55%. Most service contracts here are simple linear facts on a few fields.
  Facts that rest on the database are the main risk, because no local proof reaches them.

#### T5. A body meaning that proofs can use

- **Claim.** With the same verifier, bodies with trapping integers and value semantics are proved
  more often than bodies with Go's meaning, because the proof has fewer cases.
- **Linter test: Linter-partial.** A concrete case: under wrapping, `Refunded <= Charged` after an
  addition needs a bound on the addition. Under trapping, the overflow path is a panic, so the case
  disappears. L can get the same proofs through a subset: value types in proved functions and a
  checked-arithmetic library on marked types. The loss is in form.
- **Moves.** C, beyond what T4 reaches.
- **Cost.** Compiler: T4, T8's lowering, and a restricted body mode. This breaks "bodies are plain
  Go" for code in that mode. Agents: new meaning to learn (N5), and some programs to rewrite.
- **Test.** *Measure:* the paired difference in the share of clauses proved, L against V, on one
  corpus and one solver. Also form on those clauses. *Refuted if* the interval lies within ±10 points.
  *Size:* 44 clauses detects a 20-point gap with 80% power (paired; assumes 25% and 5% discordant).
  Clauses of one function are not independent, so use 60 or more, and analyze by function as well.
- **Prior (prediction).** 25%. L can adopt most of the same subset. The vericoding benchmark (appendix
  A) shows large gaps in agent proof success between verification languages, which is the reason that
  this is above 10%, but those gaps have many causes.

#### T6. The agent pays for the proof

- **Claim.** Annotations were too costly for people, but an agent can write loop invariants and lemmas
  at a price worth paying, and a proof language built for this raises its success rate.
- **Linter test: Linter-partial.** L can write `contract.Loop(cond)`, and lemmas as Go functions.
  Ghost state has no home in Go, except in comments that the Go compiler does not check.
- **Moves.** C and D, in code with loops.
- **Cost.** Compiler: T4 plus annotations, ghost state, and error messages for proofs. Agents: many
  turns per proof, and possible weakening of a contract to pass (N4).
- **Test.** *Measure:* proof completion within the turn budget, tokens per proved function, and
  contracts weakened. Paired L against V. *Refuted if* V's completion is not better than L's by more
  than 12 points, by the same bound as plan-r2 verdict 3, or tokens per proved function are more than
  3 times those of the same function with a run-time contract. *Size:* 20 functions, 3 runs an arm.
- **Prior (prediction).** 60% that agents can do it at this price. 15% that V beats L.

#### T7. Illegal states that cannot be built

- **Claim.** Sum types with exhaustive `match`, types with no zero value, and branded values cost few
  tokens in a language, so agents use them, and value rules hold by construction.
- **Linter test: Linter-partial.** L has sealed interfaces, `exhaustive`, `go-check-sumtype`, and
  branded types with an unexported field. L can ban each place that makes a zero value of a marked type
  (`var`, `new`, omitted fields, `make` with a length, a map miss, a closed channel, `Scan`). The
  losses: a sealed interface can be nil, a declaration costs more tokens, and the ban list is long.
  V inherits the same runtime, so V also needs the ban inside generated code, generics, and decoding.
- **Moves.** C and D. It also removes the K0506 limit, so `invariant Price > 0` becomes possible.
- **Cost.** Compiler: new syntax and lowering, about 1,000 to 2,000 lines, plus primer space. Agents:
  a new form for each state-like value, which works against R8.
- **Test.** *Measure:* form on 20 rule instances, attacker bypasses, and on runs, breaks of value rules.
  *Refuted if* L's form is within 1.25 times V's, or bypasses are within 1 of each other *and* runs show
  no difference by plan-r2's verdict 3. *Size:* form 20 instances; runs as plan-r2, section 7.
- **Prior (prediction).** 70% for a form gap above 1.25. 30% for fewer bypasses in V. 15% for a
  measured gain in runs, because round 1 suggests that a strong model breaks these rules rarely.

#### T8. Safer meanings in bodies

- **Claim.** Integer operations that trap, `append` that does not alias, and fixed map order remove
  defects that agents write, and that no local check can find.
- **Linter test: Language-only (close call).** Go defines these meanings. An analyzer can only ban
  them, and banning `+` on all integers is not ordinary Go. L can guard *marked* types through
  methods, which covers money, so the loss is smaller than it first looks. A deterministic test order
  is available to L, so only a production meaning counts here.
- **Moves.** C, as a run-time trap rather than a check-time rejection, unless T5 proves the trap away.
- **Cost.** Compiler: a rewrite in lowering, about 200 lines for arithmetic. Run time: small for
  arithmetic, but not measured. Agents: code that relies on wrapping, such as a hash, breaks (N5), and
  an explicit form is needed for it.
- **Test.** *Measure:* the base rate: re-run holdout tests on the final workspaces of calibration
  runs under the rewriter, and count runs where a trap fires or a result changes. *Refuted if* at most
  1 of 108 runs shows it (upper 95% bound about 4.3%). *Supported if* at least 10 of 108 do (lower
  bound about 5%). *Size:* the 108 calibration runs of plan-r2, section 7, which are needed anyway.
- **Prior (prediction).** 15%. Service code with small amounts seldom overflows, and the tasks may not
  reach these paths. The test has low reach on defects that the holdout tests do not cover.

#### T9. Labels on values (secrets, personal data)

- **Claim.** A type label on a value proves that it never reaches a log, a response, or an external.
- **Linter test: Linter-equal.** `Secret[T]` is a Go generic type, and taint analyzers such as
  `go-flow-levee` follow it. A language cannot close implicit flows in Go bodies either.
- **Moves.** B and D, for "X never reaches Y" rules.
- **Cost.** Compiler: facts and flow analysis, about 1,000 lines. Agents: wrap and unwrap at each
  boundary.
- **Test.** *Measure:* labels (seeded leaks), and bypasses. *Refuted if* V and L differ by at least 4
  rules (U at or above decision 6's limit), or by more than 1 bypass. *Size:* 28 rules, or a focused
  set of at least 10 flow rules.
- **Prior (prediction).** 85% equal.

### State

#### T10. Declared state machines

- **Claim.** A declared table of legal transitions, checked at each write of a status field, rejects
  a write that does not follow from a known state.
- **Linter test: Linter-equal.** The analysis is a flow check over `go/ssa` or the same AST, on a
  declaration that can be a Go table in the manifest.
- **Moves.** C and D, for life-cycle rules.
- **Cost.** Compiler: about 500 lines in each checker. Agents: one table, and a guard before each
  write. Persisted state means that a guard is the only fact, so the check stays local.
- **Test.** *Measure:* labels, and false rejections on the base applications. *Refuted if* at most 1
  more rule moves, or U reaches 4. *Size:* 28 rules, and each machine in Stockroom-lite.
- **Prior (prediction).** 70% that it moves rules and stays equal.

#### T11. Linear typestate and resources

- **Claim.** A value that is used up by a transition, or a resource that must be closed, cannot be
  used twice or forgotten, by type.
- **Linter test: Linter-partial.** Go can give each state its own type. Go has no move, so
  `d.Pay(); d.Pay()` compiles. An analyzer can reject a second use of a local (`sqlclosecheck` does a
  part of this), but fields, maps, and closures are holes unless banned. A language has the same
  holes unless it adds moves to all of Go's bodies.
- **Moves.** D, and part of C. A service reads most state from a database, so a typestate lasts for one
  request. This limits what it can reach.
- **Cost.** Compiler: a move checker, more than 2,000 lines, and a change to the body language. Agents:
  a borrow-checker style of error, a known cost in Rust, and `.clone()`-like ways around it.
- **Test.** *Measure:* bypasses, form, and the turns to fix a rejection. *Refuted if* the bypass counts
  are within 1, or V's turns to fix are more than 1.5 times L's. *Size:* as T7.
- **Prior (prediction).** 20%.

### Capabilities and effects

#### T12. No ambient authority

- **Claim.** If code can reach the network, the files, or the clock only through a value that it was
  given, then a signature tells what a part can do, and rules on authority follow from the wiring.
- **Linter test: Linter-equal.** A port given to a constructor is already a capability. An analyzer can
  ban `net`, `os`, and `time.Now` outside gateways. An attenuated value (a dialer for one host) is a
  library type, and it reaches G11 at run time. Neither form closes dependencies: T2 is the same in
  both.
- **Moves.** B. Part of G11, at run time.
- **Cost.** Compiler: rules only. Plan-r3's `only-in` kind covers most of it. Agents: a `Clock` port
  and similar, in more constructors.
- **Test.** *Measure:* labels and bypasses. *Refuted if* U reaches 4, or the bypass counts differ by
  more than 1. *Size:* 28 rules.
- **Prior (prediction).** 85% equal.

#### T13. Effects in function types

- **Claim.** Effects in signatures, with inference and with polymorphism over function values, check
  each part by itself, and make what a function does visible in the signature.
- **Linter test: Linter-partial.** Analyzer facts give the same reach for direct calls, and strict
  mode bans function values across ports. Polymorphism and interface methods need comments in L, which
  the Go compiler does not check. The spec lists this as a non-goal (§12) because of its conflict with
  Go ergonomics.
- **Moves.** B. It gives no more rejections than a whole-program check, but a local and readable form.
- **Cost.** Compiler: a type system change, with new syntax across signatures. Agents: "colored"
  functions: one new effect can change many signatures (N2).
- **Test.** *Measure:* lines touched and turns when a leaf function gains an effect, and input tokens
  to the first pass. *Refuted if* V's cost is not lower than L's by a bound below 0.87, as in plan-r2
  verdict 3. *Size:* 20 ordinary tasks that add an effect.
- **Prior (prediction).** 10%. I expect a cost, not a gain.

### Boundaries

#### T14. Declared dependency boundaries

- **Claim.** A table of which part may depend on which rejects each new wire that the team did not
  allow.
- **Linter test: Linter-equal.** It is a check on the manifest in L and on `system.vulcan` in V. Plan-r3
  section 3 already plans it in both. It is not a language differentiator.
- **Moves.** A. All 6 class A rules that are outside the model are of this form, according to the
  diagnostic.
- **Cost.** Compiler: small. Agents: a protected file to change for a legitimate wire, which plan-r3
  measures.
- **Test.** Plan-r3's T1 and N measures. *Refuted if* class A does not move, or U reaches 4.
- **Prior (prediction).** 90% that class A moves and U stays empty.

#### T15. Declared ownership of data

- **Claim.** A declaration of which part owns which table or file rejects a second writer, and moves
  G12 into the model.
- **Linter test: Linter-equal.** With `sqlc` or a typed query layer, the tables are known at check time.
  A language could have typed queries in its syntax, at the same reach. Dynamic SQL is a hole in both.
- **Moves.** A and D.
- **Cost.** Compiler: a query reader for one SQL dialect, about 1,000 lines, or a generator. Agents:
  a query layer in place of raw strings.
- **Test.** *Measure:* seeded second writers, by constant SQL and by built SQL. *Refuted if* the arms
  differ by more than 1 seed, or fewer than 80% of constant-SQL seeds are rejected. *Size:* at least
  20 seeds.
- **Prior (prediction).** 80% equal.

### Across services

The topics and remote ports of §5.5 are deferred. T16 to T18 need that work first, or a Go equivalent.

#### T16. Wire contracts in one compile unit

- **Claim.** One compiler that sees both sides of a topic or remote port rejects an incompatible
  change before deployment.
- **Linter test: Linter-equal.** `buf breaking` does it for protobuf, and `topics.lock` is a file that
  a Go tool can check. No compiler sees what is deployed, in either form, so version skew is a shared
  limit.
- **Moves.** D, and part of C, for invariants on messages.
- **Cost.** Compiler: §5.5.2 is written, but not built. Agents: small.
- **Test.** *Measure:* seeded incompatible changes rejected, across at least 20 seeds. *Refuted if*
  the arms differ by more than 1 seed.
- **Prior (prediction).** 90% equal.

#### T17. Proved idempotency and ordering

- **Claim.** An analysis of a handler's effects proves the `idempotent` mark, which the spec today
  only trusts, and rejects a handler that needs an order that the transport does not give.
- **Linter test: Linter-equal.** The effects are the same Go calls in both forms. The real proof needs
  a known write form, such as a keyed upsert in one transaction, and a linter can require that form.
- **Moves.** D. Without a known write form, only T26's run-time test reaches it.
- **Cost.** Compiler: T15, plus an effect analysis. Agents: one handler shape. A rule cannot make
  exactly-once delivery exist: it can only hold effects to at-least-once with idempotent handling.
- **Test.** *Measure:* seeded non-idempotent handlers rejected, and false rejections. *Refuted if* the
  arms differ by more than 1 seed, or fewer than 70% are rejected. *Size:* at least 20 seeds.
- **Prior (prediction).** 80% equal. 40% that the check is useful enough to keep.

#### T18. Durable execution from the compiler

- **Claim.** If the compiler lowers a handler so that it resumes after a crash, with each effect
  recorded once, then crash rules hold with no special code, and an agent need not reason about them.
- **Linter test: Linter-partial (close call).** Temporal and similar libraries give the guarantee in
  Go, with a determinism check. The loss is in form: a split into workflow and activities, and
  `workflow.Now` in place of `time.Now`. A rewriter that hides this (`coroutine`) changes what the
  Go text means, so it counts as a language.
- **Moves.** D, at run time. A check can reject effects outside a durable step.
- **Cost.** Compiler: very large, with a runtime and storage. Agents: state that must be serializable,
  and a new failure model. It conflicts with spec §5.5's "mechanism, not policy".
- **Test.** *Measure:* crash-at-each-effect tests passed, and form. *Refuted if* the pass rates are
  within 12 points, and form is within 1.25. *Size:* 10 crash-sensitive tasks, 3 runs an arm.
- **Prior (prediction).** 15% that it is worth its build. High cost, deferred dependency.

### Concurrency and determinism

#### T19. Deterministic simulation

- **Claim.** If time, randomness, map order, and goroutine schedules come from one seed under test,
  then an agent can replay a failure, and find concurrency faults that it cannot find today.
- **Linter test: Linter-equal.** A fixed schedule is one that Go already allows. `gosim` and
  `testing/synctest` show the approach, and both arms lower to the same Go.
- **Moves.** D, from "flaky test" to "replayable test". No check-time rejection.
- **Cost.** Build: large if done from nothing, small if an existing tool fits. Agents: a seed to read.
- **Test.** *Measure:* seeded races and ordering faults found and fixed, with and without the tool, in
  both arms. *Refuted if* the arms differ by more than 12 points. *Size:* 12 seeded faults, 3 runs an arm.
- **Prior (prediction).** 80% equal. 40% that the tool helps either arm.

#### T20. Race freedom by type

- **Claim.** Types that say what may cross between goroutines reject data races at check time.
- **Linter test: Linter-partial.** An analyzer can apply `holdsReference()` (K0210) to `go` statements
  and channel element types, with an allowlist of safe types such as `*sql.DB`. Both need that list.
  A language adds ownership transfer, which accepts more programs, but it needs moves in bodies (T11).
- **Moves.** D.
- **Cost.** Compiler: from a simple walk to a region system. Agents: rejected ordinary shapes, such as
  a shared store behind a lock, unless a `Guarded[T]` form is learned.
- **Test.** *Measure:* attacker bypasses, seeded races rejected, and false rejections on base apps.
  *Refuted if* bypass counts are within 1, and false rejections are not lower in V. *Size:* at least 20
  seeds.
- **Prior (prediction).** 20%. Agents appear to make few such faults in this kind of service, but no
  measurement here isolates them.

### Agent economics

#### T21. The cost of a correct program

- **Claim.** With the same guarantee, a language lets an agent write less and read less, because the
  compiler generates what L must write out.
- **Linter test: Linter-partial.** L's source is 16.8% larger in pieces than V's (25,797 against
  22,093), with the same 29 clauses and the same behavior. Part of this is `checkedX` wrappers,
  `contract.Valid` sites, `defer contract.Keep`, and a hand-written `main`. A generator can remove the
  wrappers and most of `main`. Then the source of truth is a comment or a second file.
- **Moves.** No class. It is a cost theory.
- **Cost.** Compiler: exists. Generator for L: about 500 lines. Agents: V needs a new syntax around Go
  (N1), and that may cost more than this saves.
- **Test.** *Measure:* pieces after the generator, then input tokens and turns on ordinary tasks,
  paired per task: V, L, and L with the generator. *Refuted if* the ratio of L with the generator to V
  has a one-sided 95% bound below 1.05, or if V's cost is not lower at all. *Size:* with the round 1
  spread (SD of the task log ratio 0.26, from runs cut by a faulty budget), 18 tasks detect 17%, and 47
  detect 10%. Plan-r2's 8 ordinary tasks detect only about 26%.
- **Prior (prediction).** 30% for a measured gain over L with the generator. Round 1's cost ratio was
  1.08 against V, on a tight budget.

#### T22. Machine-applicable fixes

- **Claim.** A diagnostic with an exact edit reduces the turns from a rejection to a pass.
- **Linter test: Linter-equal.** `go/analysis` has `SuggestedFix`. Neither checker uses it today.
- **Moves.** No class. It is a cost theory.
- **Cost.** Build: about 10 to 20 lines per fix, with tests. Agents: none, but a fix that is applied
  without thought can hide the rule's reason.
- **Test.** *Measure:* M5 and M6 of plan-r2 (recovery within two check runs), with and without fixes,
  in both arms. *Refuted if* the turns from rejection to pass do not fall by at least 20%, or the arms
  differ by more than 12 points. *Size:* the tempting tasks of plan-r2.
- **Prior (prediction).** 45% that it helps, 90% equal. Current messages already say how to fix.

#### T23. One way to write a thing, and a small primer

- **Claim.** Fewer forms and one model make an agent more often right on the first attempt.
- **Linter test: Linter-equal.** The primers are within 3% ([linter README](../../poc/golint/README.md)),
  and a linter can ban the other forms. The model knows Go's forms, so a ban may cost rejections, and a
  new form costs learning. Those two costs are what the test compares.
- **Moves.** No class.
- **Cost.** Primer space, in both arms.
- **Test.** *Measure:* rejections before the first pass, and first-attempt pass rate. *Refuted if* V
  is not better by a bound above 0, with at least 12 points. *Size:* plan-r2, section 7.
- **Prior (prediction).** 15%. Measured in the same runs as T21.

#### T24. Typed holes

- **Claim.** A check that accepts an unfinished program and answers with the type and the ports in
  scope at a hole reduces invented APIs (R4) and turns.
- **Linter test: Linter-partial.** `go/types` continues after errors, and `gopls` gives much of this. A
  generic `hole[T]()` works where `T` is written out. The loss is that Go does not infer it from
  context. The research on typed context (appendix A) is on small programs.
- **Moves.** No class. It is a cost theory.
- **Cost.** Compiler: bidirectional checks, and a server. Agents: one new command to use.
- **Test.** *Measure:* `undefined:` build failures, and turns, with and without the hole command, in
  both arms. *Refuted if* V's benefit does not beat L's by a bound above 0. *Size:* 20 ordinary tasks.
- **Prior (prediction).** 10% for a V gain over L. 35% that it helps either arm.

#### T25. Local reasoning

- **Claim.** If a leaf and its ports hold all that an edit needs, an agent reads fewer files and uses
  less context.
- **Linter test: Linter-equal.** `ports.go` holds the same facts. A `context` command is a small
  tool in either form.
- **Moves.** No class.
- **Cost.** Build: small, in both arms. Neither has it today (O8 of plan-r2).
- **Test.** *Measure:* input tokens to the first passing check after the first edit, with and without
  the command. *Refuted if* the arms differ by a ratio bound below 0.87 either way. *Size:* 20 ordinary
  tasks, and a larger application than Stockroom-lite.
- **Prior (prediction).** 85% equal. 35% that it helps either arm at this application size.

#### T26. Tests derived from contracts

- **Claim.** A tool that derives property tests, double delivery, and crash points from contracts
  turns many run-time contracts into failures that an agent sees before a change is done.
- **Linter test: Linter-equal.** L's contracts are Go expressions, and a generator can read them.
- **Moves.** C and D, to test time, not check time.
- **Cost.** Build: about 1,000 lines, in both arms. Agents: more failures to work through, and slower
  checks.
- **Test.** *Measure:* seeded value and retry faults found before a change passes. *Refuted if* no more
  than 10% more seeds are found, or the arms differ by more than 12 points. *Size:* at least 20 seeds.
- **Prior (prediction).** 60% that it helps both arms. 90% equal.

## 4. Theories of harm

Each harm can cancel a gain, so a round must measure them with the same care. Their verdicts say
which form carries the cost.

| # | Claim | Verdict and reason | Moves | Cost | Test: measure, and what refutes it | Prior (prediction) |
|---|---|---|---|---|---|---|
| N1 | A new language costs success and turns, because the model has no training data on it | **Language-only.** L has no such cost | None | Primer, and failed runs | Ordinary tasks, V against L: N6 and cost ratio. Refuted if V is within 5% on tokens and 5 points on success. 20 tasks, 5 runs | 55% that a cost is measured. Round 1 is weak evidence: −19.4 points, under a faulty budget |
| N2 | Annotations, colored functions, and solver timeouts cost more turns than they save | **Linter-equal.** The same features cost the same in either form | C, B at a price | Turns per feature | Turns with and without the feature, in each arm. Refuted if the turns rise by less than 10% | 60% |
| N3 | More rejections lead to longer runs, and runs that do not stop | **Linter-equal.** Any strong check can do this. Round 1 saw it in V only, with no linter arm | None | Budget hits | Budget hits, turns after the first pass, by arm. Refuted if arms with checks are within 10 points of P | 50% |
| N4 | Agents pass a strong check by weakening the thing checked: a contract removed, a marker added | **Linter-equal.** Both arms can edit their contracts. Decision 6 fails a changed configuration only | All, against | Silent loss of a rule | A scripted diff of contracts and lists against the base. Refuted if under 5% of runs weaken one | 50% |
| N5 | Text that looks like Go and means something else misleads a model trained on Go | **Language-only.** It is T8's and T5's price | None | Silent faults | Seeded Go idioms that rely on wrapping or aliasing: agent faults in V's stricter mode against L. Refuted if the rates are within 12 points | 40% |
| N6 | One central file (`system.vulcan`, or `arch.go` and `main.go`) causes conflicts between parallel agents | **Linter-equal.** Both forms centralize wiring | None | Merge failures | Merge failures in chains of changes, across parallel runs. Refuted if under 5% conflict | 30% |

## 5. Test stages

All stages reuse plan-r2 and decision 6. Every verdict, for or against the language, gets the same
tests, the same replication, and the same reporting.

| Stage | How it runs | What it decides |
|---|---|---|
| **Build of L** | A fresh session gets the same feature brief and budget as V's builder, is told to match it, and writes a coverage map with decision 6's reasons (a), (b), (c) | Reason (c) leaves U as a gap (decision 6, section 5). No theory wins because L ran out of budget |
| **Form** | A fresh writer, given a neutral statement of each rule instance, writes it in each arm. `primer_count.py` counts pieces. Both forms are reviewed for parity | The ratio of L's form to V's |
| **Holes** | A fresh attacker session per arm reads the checker and tries, within a fixed budget, to write programs that break the claimed guarantee and pass the check. Each bypass runs. Budget and prompt are the same for each arm | A count of working bypasses per arm. It is a measure of effort, so repeat it at least 3 times, and report the spread |
| **Labels** | A fresh rule writer and labeler, as in plan-r2 section 4. At least 28 rules, 7 a class, 2 shapes each. Sets S, G, U, N | U below 4 means "no rule needs the language" (decision 6) |
| **Runs** | Plan-r2 sections 2, 3, 5, 6, and 7, with its validity gate, verdicts, and replication on a second model. Its dry run gives verdict 3 in 86% of cases when the language adds a large effect, and in 4.5% when it adds none | Clean success, cost, and review |

**Which rule set.** Round 2's sealed set is not a clean test of features that come after its summary,
as plan-r3 section 0 says. Its T1 set is clean only for features that are frozen before it is written.
A feature that misses that freeze needs a new rule writer. This document's author read the class counts
and the diagnostic's summary, and no rule text.

**Inputs that lean one way**, named so that a reader can weigh them:

- Arm L is allowed every Go tool that exists, with no price for adopting it. This favors L.
- The form factor of 1.25 is a judgment. A smaller factor would turn some Linter-equal verdicts into
  Linter-partial, and a larger one the reverse.
- Theories come from one session. A second session should repeat the verdicts before a build starts,
  with no sight of this document's verdict column.
- The shortlist favors theories that a cheap gate can test. A theory with a slow test could rank lower
  than its value.

## 6. Ranked shortlist

Each stop rule has three outcomes with the same burden. "Replicated" means that a second corpus or
second model gives the same outcome.

### Rank 1: T5, with T4 inside it. Static proof, and whether a stricter body meaning helps

**Why first.** Class C is 0 of 7 today, and T4 has the best chance of moving it. T5 is the most direct
test of a language effect. Both come from one build, so the result separates the tool from the
language.

**Build.**

1. A verifier, `poc/prove`: one extractor per arm into one form, verification conditions over `go/ssa`,
   and Z3. Scope: integer and length facts on straight-line code, branches, callee contracts, and port
   contracts. Z3 is not on this machine, so its download needs the owner's yes.
2. A flag in V's lowering, `semantics: trap`, with trapping integers. The verifier reads this flag.
3. A fresh L builder gives L its best equal: checked arithmetic on marked types, with a subset rule.

**Test.**

1. *No agents.* A fresh writer states contracts in neutral form on Stockroom-lite's two new services.
   With the existing 29 clauses, this makes at least 60. Report the share proved per arm, paired.
2. *Labels.* A fresh rule set. Report class C and D rules that each arm rejects at check time.
3. *Runs,* only if step 1 or 2 shows a gain, and the owner decides to spend. Measure success, turns,
   tokens, solver time, weakened contracts (N4), and proof failures (N2).

**Stop rule.**

| Outcome | Condition (replicated on a second corpus writer) | Next |
|---|---|---|
| T4 supported | At least 30% of clauses proved, and at least 2 more class C or D rules rejected | Keep the verifier, in both forms |
| T4 not supported | Under 30% proved, or at most 1 more rule rejected | Stop the proof work. Contracts stay at run time |
| T5 supported | V's share exceeds L's by at least 10 points, with the 95% interval above 0 | Go on to runs of the stricter mode, and count N5 |
| T5 no measured gain | The 95% interval lies within ±10 points | Stop the stricter mode. No language case from proofs |
| T5 unclear | Anything else | One more corpus, sized before it is written. Then read it as above; unclear stays reported as unclear |

### Rank 2: T21. The cost of a correct program, and N1

**Why second.** It already has a first measure, costs little, and has a real chance of a result in
either direction. It also measures N1, which any language case must overcome.

**Build.** A generator for L: port wrappers and `main` from the manifest and `ports.go`. Update the
primer within the budget. A fresh session writes it, and a parity audit reads it.

**Test.**

1. Recount pieces for V, L, and L with the generator, counting what an agent must read and write.
2. Grow the ordinary tasks of plan-r3 step 8 from 8 to 20. This adds about 180 runs (12 tasks, 3 arms,
   5 runs). Measure input tokens, turns, and success, paired per task. A pilot replaces the spread that
   came from round 1's faulty budget.

**Stop rule** (replicated on a second model).

| Outcome | Condition | Next |
|---|---|---|
| Gain for V | L with the generator costs more than V, with a one-sided 95% bound above 1.05 | A language or declaration form has a measured case on cost. Go to section 7, branch B |
| No measured gain | The 90% interval of the ratio lies within 0.95 to 1.05 | Cost gives no case for the language. Use the generator |
| Gain for L | V costs more than L with the generator, with a one-sided 95% bound above 1.05 | N1 outweighs the saved syntax. Report it as a result against the language |
| Unclear | Anything else | Report the interval. More runs only if sized before they start |

### Rank 3: T8. Safer meanings in bodies

**Why third.** It is the only theory that the linter test does not remove, so testing it closes that
question. Its gate adds no agent run.

**Build.** A research rewriter that adds a trap to integer `+`, `-`, `*`, and narrowing conversions in
any Go workspace, about 200 lines. A second mode checks `append` aliasing at run time.

**Test.** After plan-r3 or plan-r2 calibration, re-run the holdout tests on the 108 final workspaces, plain
and rewritten. Count the runs where a trap fires or a test result changes. Classify each as a real
fault or a false alarm, with a fresh session. Also run the rewriter on the base applications, and on
idioms that rely on wrapping (N5). The tasks were not written for this class, so the count is a floor
for what agents write, and holdout tests can miss a fault.

**Stop rule.**

| Outcome | Condition | Next |
|---|---|---|
| Supported | At least 10 of 108 runs show a real fault (lower 95% bound about 5%) | Run a second model. If it repeats, T8 joins the stricter mode of rank 1 |
| No measured gain | At most 1 of 108 runs (upper 95% bound about 4.3%) | Stop. Section 1's only Language-only theory has no measured base rate here |
| Unclear | 2 to 9 of 108 | Report the count, and repeat on a set of tasks written for no feature, sized before it runs |

### Rank 4: T7. Illegal states that cannot be built

**Why fourth.** It has the largest form gap in classes C and D, and an agent may use a cheap form when it
will not use a costly one. The build is larger and the prior of a gain in runs is low.

**Build.** In V: `enum` with payloads, `match` with no default, and `nozero` types whose invariant need
not hold at zero. In L, by a fresh builder: sealed interfaces, `exhaustive`, `go-check-sumtype`, and an
analyzer of zero-value sites. Primers stay within 5% of each other.

**Test.**

1. *Form,* on 20 neutral rule instances from a fresh writer.
2. *Holes:* an attacker per arm, repeated 3 times.
3. *Use:* in ordinary tasks, how often an agent chooses the new form with no prompt.
4. *Runs,* only if form or holes show a gap, on tasks where value rules can be broken.

**Stop rule.**

| Outcome | Condition (replicated) | Next |
|---|---|---|
| Supported | Plan-r2 verdict 3 on value rules: V better than L by more than 12 points, with the bound above 0 | Keep the feature in the language |
| No measured gain | Form ratio under 1.25 and bypass counts within 1, or verdict 2 in runs | Keep L's analyzers, if they move rules. No language case |
| Unclear | Anything else | Report it. Runs on more tasks only if sized before they start |

### Worth building, but not differentiators

These are Linter-equal. Each could move rules in both arms, so each is a matter for plan-r3's rule kinds,
and the owner's decision, not for the question of the language.

| Theory | Class | Fit |
|---|---|---|
| T14, declared boundaries | A | In plan-r3 section 3 |
| T1, closed lists, and the fix of H6 and H7 | B | Before any blind test: the same fix in both checkers |
| T10, state machines | C, D | A new rule kind in plan-r3, if it joins before the freeze |
| T4, static proof | C | Rank 1 builds it |
| T26, tests from contracts | C, D | After rank 1, as they use one extractor |
| T22, fixes in diagnostics | Cost | Small, and independent of the rest |

**A note on order.** Rank 3's gate and rank 1's first step need no agent run. So they can answer before
plan-r3 step 8, and the owner can decide what joins that round with their results in hand.

## 7. If no theory survives

These branches are fixed now, before any result, and each one states what it would give up. They extend
decision 6's row 4.

| Branch | Result | What it implies |
|---|---|---|
| **A. Nothing survives** | No rank gives a replicated language gain over L | Keep Vulcan as a ruleset and a linter for Go: `vulcan-lint`, `vulcan-lint init`, the rules file, the primer, and any verifier or generator that showed a value. Stop work on the compiler and on the language sections of the spec. Keep them archived with the evidence. Update the README's conclusion. The claim becomes "an enforced model for agents working in Go", and needs its own test (QA of plan-r2). If QA also fails, no part of the model has a measured value for this agent, and that should be stated too |
| **B. Only form survives** (T21, T7) | A gain in cost, with no gain in guarantees | The gain may come from the declarations, not from a language. Consider a small declaration language for ports, contracts, and sums that generates plain Go, with bodies in plain `.go` files. This is close to what Vulcan is now, with a smaller claim and less to teach |
| **C. Only a meaning survives** (T5, T8) | A gain from body meaning, with no need for syntax | Consider a Go dialect: the same grammar, a build step that changes the meaning, and a marked mode per part. N5 is its main risk, and the replication must include that measure |
| **D. Guarantees survive** (T7, T11, T20, by holes and runs) | A real gap in holes that matters in runs | The case for a language with its own body rules stands. "Bodies are plain Go" must change, and decision 6's milestones apply. N1 becomes the main risk |

Branch A is the outcome that I predict, at about 80% with B, C, and D sharing the rest. This does not
weigh for or against it: the point of the stop rules is that each branch is reached by the same burden.

## 8. Considered and set aside

They got a first look only, so they are not in the counts.

| Idea | Why it is not a full theory here |
|---|---|
| Session types across services | Scribble generates Go APIs for them. It needs T11's linear use, and §5.5 is deferred |
| Constrained decoding: a grammar and types that bind the model's output | The harness uses a hosted model, through an agent, with no control of sampling. So it cannot be tested here |
| Content-addressed code, as in Unison | It gives no rule enforcement, and it costs the entire Go toolchain. N6 tests the problem that it would address |
| Resource bounds in types (no N+1 queries, bounded memory) | A syntax check for a query in a loop is a linter rule. Real bounds are a research project |
| Termination and "no panic" proofs | Follows from T4 where the facts are linear. No separate language point |
| Rules in a query language (Datalog over facts) | It is a form of plan-r3's rules file. Semgrep and CodeQL show it in Go |
| The check as a gate on the build | `go build` works without `vulcan-lint`, but a harness and a CI gate both run it. It is the same in practice under decision 6's named check command |

## 9. Amendments at acceptance

**A second reading of the verdicts.** Every rank rests on the verdicts of section 3 and section 4.
Before any feature here is built, a fresh session judges each theory with the rules of section 1. It
sees the claims and the code, not this document's verdicts, priors, or ranks, and not the results of
round 2. Each disagreement is recorded here, with both reasons. The owner decides what a disagreement
changes. A disagreement is not settled by the first author's reading or by the second's.

**A first look at T8.** Rank 3's gate needs calibration runs that do not exist yet. Round 1's 72 final
workspaces do exist, so the rewriter can run on them first. This look is exploratory. Its app is
smaller, its arms are G and V, its turn budget was cut by a fault, and it has too few runs for the
stop rule. So no stop rule reads it, whatever it shows, and rank 3's test still runs as written. Round
1's cases are reported with their count and their classification by a fresh session.

**Holes and rule kinds.** The fix of H6 and H7, and T10 as a rule kind, join [plan-r3.md](plan-r3.md)
before its freeze (its section 9).

## Appendix A: outside sources

These come from general knowledge. I did not check them for this document, so check each one before
relying on it.

| Source | Used for |
|---|---|
| Gobra, a verifier for Go with annotations in comments | T4, T6 |
| `capslock`, a capability analysis of Go programs and their dependencies | T2 |
| `nilaway`, `exhaustive`, `go-check-sumtype`, `sqlclosecheck` | T7, T11 |
| `go-flow-levee`, taint analysis for Go | T9 |
| Service Weaver, Go components as generated services | T3 |
| Temporal's Go SDK and its workflow check; `coroutine` of dispatchrun, a Go compiler for durable coroutines | T18 |
| `gosim`, and Go's `testing/synctest` | T19 |
| `buf breaking`, `sqlc` | T15, T16 |
| Scribble-Go, session types for Go (Castro and others, POPL 2019) | Section 8 |
| "A benchmark for vericoding" (2025). From memory: success of about 82% in Dafny, 44% in Verus, and 27% in Lean, on different task sets | T5, T6 |
| Mündler and others, type-constrained code generation (PLDI 2025) | Section 8 |
| Blinn and others, typed holes as context for models (OOPSLA 2024) | T24 |

## Appendix B: how the numbers here were made

- **Source size:** every `.go` and `.vulcan` file under `poc/app/go`, `poc/app/golint`, and `poc/app/vulcan`,
  with no test files and nothing under `.vulcan/`, concatenated and counted with `poc/golint/primer_count.py`.
  Pieces are a proxy for tokens. The count is of source, not of what a model reads per task.
- **Holes H6 and H7:** on scratch copies, at commit `08099e0`. Added to `api/inventory` in each arm:
  `net/textproto.Dial` in an exported function, and `func holeB() int` with `h_b_amd64.s`. `vulcan-lint ./...`
  and `vulcan check` exit 0, `go build` and `vulcan build` succeed. A control with `http.Get` gives VL0205
  and K0205.
- **Task spread:** the SD of the per-task log ratio of `tokens_in`, V against G, in
  `experiments/poc/data/p5-deepseek/runs.jsonl`: 0.264 over 12 tasks. Tasks needed: `((1.645 + 0.84) ×
  SD / ln(1 + effect))²`.
- **Clause count:** McNemar's test, one-sided α 0.05, power 0.8, with discordant shares of 25% and 5%.
- **Bounds on 108 runs:** exact binomial, one-sided 95%.
