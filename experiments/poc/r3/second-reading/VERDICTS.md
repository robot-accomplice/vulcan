# Verdicts on the theories

This is an independent reading of `THEORIES.md`. I did not read the earlier reading, other sessions' notes, or any barred path.

## Result

| Verdict | Count | Ids |
|---|---|---|
| Linter-equal | 22 | T1, T2, T3, T4, T6, T10, T11, T12, T14, T15, T16, T17, T19, T22, T23, T24, T25, T26, N2, N3, N4, N6 |
| Linter-partial | 6 | T7, T9, T13, T18, T20, T21 |
| Language-only | 4 | T5, T8, N1, N5 |

Eight verdicts depend most on how the test draws its line: T4, T5, T7, T8, T13, T18, T20, T21. Each entry gives the verdict under the other reading. The section "Where the test seems wrong or unclear" collects the causes.

## How I worked

**What exists.** V does not have most of these features. Its bodies are Go, unchanged (spec §6). It rejects topics and containers with K0009. So for most theories I compare two designs, and only the current arm code is fact.

**Rule for a ban.** The table says Language-only when L must ban a core Go construct in ordinary code. I count a ban against L when V keeps the construct, with a new meaning or a new form. I do not count it when V must reject the same programs. The test does not state this rule, so I flag each case where it decides the verdict.

**Evidence labels.**

- **Ran:** I ran a program. All programs are under `scratch/`.
- **Read:** I read code or the spec.
- **Doc:** I read the documentation of a tool, but did not run the tool.

**Limits of my scratch analyzers.** They are probes, not products. They show that a rule can be written on `go/analysis`. They do not show that a rule is complete. Each entry names the gaps that I saw.

**The snapshot.** `input/cmd/vulcan` does not build as delivered, because the root package that holds `Guide` is absent. I copied `input/` to `scratch/src` and added a one-line stub there. `input/` is unchanged. In the copy, all conformance cases pass except T05, which fails because it reads the stub guide. All test suites of arm L pass.

### Token measures

These are the measures for the Form question.

**Whole arms (Stockroom).**

| Measure | V | L | L/V |
|---|---|---|---|
| Go scanner tokens, no comments | 16,142 | 18,502 | 1.146 |
| cl100k tokens, with comments | 21,169 | 24,378 | 1.152 |
| cl100k, L without `lib/contract`, `arch/schema.go`, `arch/config.go` | 21,169 | 22,912 | 1.082 |
| cl100k, also with no `main.go` in L and no `system.vulcan` in V | 20,768 | 22,003 | 1.059 |
| Wiring only, scanner: `main.go` against `system.vulcan` | 242 | 640 | 2.64 |
| Primer, cl100k: `PRIMER.md` against `guide.md` | 2,651 | 2,370 | 0.89 |

cl100k is a proxy. It is not the tokenizer of the model under test.

**Per leaf, with its ports (cl100k).**

| Leaf | L/V |
|---|---|
| orders | 1.13 |
| billing | 1.25 |
| inventory | 1.12 |
| events | 1.06 |
| purchasing | 1.07 |
| web | 1.05 |

By scanner tokens, billing is 1.33 (910 against 685).

**Per site** (`scratch/probes/sites.py`). Pieces use the method of `primer_count.py`. **Every V form for a feature that V lacks is my own minimal guess at its syntax.** The ratios for those sites are therefore estimates.

| Site | Pieces L/V | cl100k L/V |
|---|---|---|
| `requires`, L statement (current arm) | 2.00 | 1.60 |
| `requires`, L comment | 1.75 | 1.20 |
| `ensures`, L statement (current arm) | 1.92 | 1.90 |
| `ensures`, L comment | 1.31 | 1.00 |
| Loop invariant, L comment | 1.27 | 1.00 |
| Invariant declaration (current arm) | 3.60 | 3.00 |
| Port call (current arm) | 1.25 | 1.33 |
| Port call with `ctx` | 1.50 | 1.67 |
| Sum declaration, by hand | 2.75 | 2.53 |
| Sum declaration, generator input | 1.00 | 1.00 |
| `match` against type switch | 1.36 | 1.37 |
| A mark on a declaration (`//vl:x`) | 1.67 to 2.00 | 1.60 to 1.80 |
| `a + b` against `checked.Add(a, b)` | 2.67 | 2.00 |
| Map range against sorted keys | 2.89 | 2.38 |
| `append` against a three-index slice | 2.33 | 1.75 |
| Labeled derived value | 1.67 | 1.33 |
| Effect inside a type | 1.88 | 2.00 |
| `iso` inside a type | 2.40 | 2.20 |

**Sites in the Stockroom bodies of arm L** (by type information, `scratch/probes/sites`): 23 integer arithmetic operations, 0 map ranges, 29 appends, 2 slice expressions, 1 `go` statement, 27 pointer parameters, 24 slice parameters. The app has 14 `requires`, 12 `ensures`, and 3 invariants. It is light on contracts.

### Tools I checked

| Tool | How | What I found |
|---|---|---|
| `golang.org/x/tools/go/analysis`, facts, `-fix`, `-json` | Ran, v0.52.0 | Facts cross modules. Fixes apply. |
| `golangci-lint` 2.11.4: `gochecksumtype`, `exhaustive` | Ran | Both report a missing case. Neither reports a nil sum value. |
| `golangci-lint` list: `depguard`, `forbidigo`, `exhaustruct`, `bodyclose`, `sqlclosecheck`, `rowserrcheck` | Listed as present, not run | They exist in this install. |
| Go `GOOS=wasip1` with `wazero` v1.12.0 | Ran | See T3. |
| `testing/synctest` (Go 1.27.3) | Ran, and doc | Fake clock works. The doc names no seeded schedule. |
| `gosim` (jellevandenhooff) | Installed, failed here | It stopped on a go-sqlite3 cgo stub. A list post reports trouble on Go 1.24 and 1.25. I removed the binary afterward. |
| `z3` | Ran | See T5. |
| Gobra | Doc, not run | Specs in `//@` comments, ghost code, loop invariants, `--overflow`. The book calls overflow checking experimental. |
| Temporal `workflowcheck` | Doc (search summary) | A `go/analysis` determinism check that follows calls. It has a `//workflowcheck:ignore` comment. |
| DBOS Transact | Doc (search summary) | A library with a Go version. |
| `go-flow-levee` | Doc | Taint analysis through `go vet`. It states false positives and false negatives. The latest version I saw is from 2020. |
| gVisor `checklocks` | Doc | Lock annotations on fields. It states limits on closures. |
| `buf breaking` | Doc, not installed | Compares a schema with a past version. |
| `pgregory.net/rapid` | Doc | Property tests, shrinking, state machines. |
| `sqlc`, `go-arch-lint`, Cadence `maprange` | Search summaries only | They exist. For `sqlc`, I saw only mirrors. Treat these as weak evidence. |

### Named holes, tested on both arms

| Hole | L today | V today | With the probe |
|---|---|---|---|
| H4, dial inside an SDK module | passes | passes | `reach` rejects it |
| H6, body-less function with a `.s` file in a leaf | passes, and builds | passes, and builds | `closedlist` rejects both the function and the file |
| H7, `net/textproto.Dial` in a plain leaf | passes | passes | `closedlist` rejects it |

As a control, `vulcan-lint` rejects `net.Dial` in the same leaf with VL0205.

## Theories

### T1. A closed list of what code may use

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** Ran `scratch/lab/cmd/closedlist`. Stockroom passes with a list of 24 standard packages for plain parts. H6 and H7 are rejected. By the same rule, an unlisted SDK is rejected, but I did not run that case.
- **Soundness: yes.** The list is the same data in either arm (`internal/check/network.go` and `poc/golint/effects/effects.go` already share one banned list). `pass.OtherFiles` shows `.s` files to the analyzer. A hole inside a listed package is shared.
- **Form: yes.** The code is unchanged. The list is a table.
- **Would change it:** a closed list that needs facts that `go/types` and `OtherFiles` do not give.

### T2. Reach through dependencies

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** Ran `scratch/lab/cmd/reach`. Object facts follow static calls through a separate module, and the report gives the path: `sdk.Ping -> net.Dial`. Stockroom gives no false report.
- **Soundness: yes.** Spec §5.4c follows static calls and two interface steps. A linter can use the same rule. My probe has only static calls and a short seed list. `go/callgraph` could add more. V today does not follow calls either (`network.go`, comment on `network`).
- **Form: yes.** No code changes.
- **Would change it:** a reach rule that needs code which analyzers cannot load. Then V could not load it either.

### T3. Leaves in separate memory

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes.** Ran `scratch/wasm`. With `GOOS=wasip1`, a cgo file is excluded at build. `net.Dial` uses the fake network of `net/net_fake.go` and reaches no host. A read outside the mount fails. `unsafe` writes stay inside the module memory. A generator writes the port stubs from the port interfaces, and that output is ordinary Go.
- **Soundness: yes.** The boundary belongs to the host, not to the language. Both arms trust the same Wasm runtime or OS.
- **Form: yes.** The leaf text does not change. A port call stays an interface call.
- **Not tested:** I did not build the stubs across the boundary. A pointer across a port changes behavior in both arms, so both must ban it.
- **Would change it:** a case where the choice of `GOOS` counts as a change of meaning. I read it as a legal build target.

### T4. Static proof of contracts

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes.** The bodies are the same Go, so the same verifier gets the same result. Doc: Gobra verifies unmodified Go from `//@` comments. Read: the proof rule of spec §7.1 is linear arithmetic over atoms, and an analyzer can run it. `contracts.go` already does the constant case (VL0501).
- **Soundness: yes.** Both trust a solver. Both trust Go.
- **Form: yes in comment form, no in the current statement form.** cl100k: comment 1.00 to 1.20, statement 1.60 to 1.90.
- **Other reading:** if L must keep `contract.Requires` and `defer contract.Ensures`, the verdict is Linter-partial. If a contract that a proof does not close must still run, a generator must weave the check from the comment, and that is more tool work for L.
- **Would change it:** a run of Gobra on Stockroom bodies that fails for a cause that a language would remove (that is T5).

### T5. A body meaning that proofs can use

**Verdict: Language-only. Confidence: medium.**

- **Reach: no, without a ban or a change of meaning.** Ran `scratch/probes/t5.py` with one solver on 64-bit bodies:

| Property | Go wraps | Trap | Go, with a no-overflow obligation |
|---|---|---|---|
| `b>=0 => a+b>=a` | not proved | proved | not proved |
| midpoint in range | not proved | proved | not proved |
| `a,b>=0 => a+b>=0` | not proved | proved | not proved |
| `0<=b<=a => a-b>=0` | proved | proved | proved |

  A pointer write leaves `*q` unchanged only with the premise `p != q`. Value semantics needs no such premise. Stockroom has 27 pointer and 24 slice parameters. To get value semantics, L must ban them in ordinary code.
- **Soundness: no.** Gobra by default does not check overflow (Doc). That mode reaches the same proofs, but a wrap at run time then breaks a proved fact with no trap.
- **Form:** `checked.Add` costs 2.00 to 2.67 per operation.
- **Note:** "Trap" proves partial correctness. A program that traps still counts as proved. This is why it gets more proofs. L with the obligation gives a stronger result, and rejects more programs.
- **Other reading:** for the integer half alone, if an unsound "no overflow" assumption or `checked.Add` counts as reach, the verdict is Linter-partial. The value half stays Language-only.
- **Would change it:** a measured proof rate that shows no gain on real bodies. Or a verifier that handles aliasing on Go with no extra annotation.

### T6. The agent pays for the proof

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes.** A proof language can live in structured comments, which the test allows. Doc: Gobra has loop invariants and ghost code there.
- **Soundness: yes.** It is the same verifier and the same trust.
- **Form: yes.** cl100k 1.00 for a loop invariant, pieces 1.27. Lemmas and ghost code are the same text after a comment mark.
- **Limit:** I did not measure the success rate of an agent. The test does not ask for it. A gain that comes from body meaning belongs to T5.
- **Would change it:** a proof feature that a comment cannot hold, such as a proof term that the Go parser must read.

### T7. Illegal states that cannot be built

**Verdict: Linter-partial. Confidence: medium.**

- **Reach: yes.** Ran `gochecksumtype` and `exhaustive`: both report the missing case. Ran `scratch/lab/cmd/nozero`: it rejects `var`, `new`, a partial literal, `make` with a length, a map read, a named result, and a generic instantiation. Branded values are defined types plus the same analyzer.
- **Named loss, form:** a `match` is 1.36 to 1.37. A sum declared by hand is 2.53 to 2.75, and a generator brings it to 1.00.
- **Named loss, form and rule:** the Go idiom `return Email{}, err` is rejected by the rule (`nz.go:40`). L needs a pointer result or a special case for it. V must answer the same question, but it can add a form for it.
- **Named gap:** `gochecksumtype` does not report a nil sum value (`sum.go`, func `Nil`). L needs `nozero` on the interface as well.
- **Soundness: same.** `reflect` and JSON decoding make zero values in both arms. Spec §7.2 says that Vulcan "cannot block them all" today.
- **Other reading:** if a ban on `var`, `make`, and map reads for marked types counts as a ban of core constructs, the verdict is Language-only. I do not count it, because V with Go bodies must reject the same forms.
- **Would change it:** a test that shows the extra cost of `match` does not change what agents write. That moves it to Linter-equal.

### T8. Safer meanings in bodies

**Verdict: Language-only. Confidence: high.**

- **Reach: no, without a ban.** A trapping `+` is the example of the barred rewrite in the test. L must ban `+` on integers, `range` on maps, and an aliasing `append`, and all three are core. V keeps those constructs with a new meaning.
- **Form of the rescue:** 2.00 to 2.67 for add, 2.38 to 2.89 for map range, 1.75 to 2.33 for append, at each site.
- **Whole-app cost is small here:** Stockroom has 23 integer operations, 0 map ranges, 29 appends. About 240 cl100k tokens of 24,378, about 1 percent, if each of those sites takes the longer form.
- **The other route:** Gobra with `--overflow` rejects a program that cannot prove no overflow. That rejects different programs, so it does not answer the theory.
- **Note:** this theory contradicts spec §6 and §7.4 as they stand today, and it creates the cost in N5.
- **Other reading:** if a ban plus a helper counts as reach, the verdict is Linter-partial with a large form loss per site and a small one per app.
- **Would change it:** a statement in the test that a ban is fine when an ordinary helper replaces the construct at low cost.

### T9. Labels on values (secrets, personal data)

**Verdict: Linter-partial. Confidence: medium.**

- **Reach: yes.** Ran `scratch/lab/cmd/labels` on a defined type with a label fact across packages. It reports a value that goes into `any`, a struct that holds it, a conversion, an index, a range, a compare, `len`, and a generic of another package. A label-polymorphic function works with an ordinary type parameter: `func Trim[S ~string](s S) S`.
- **Gap in my probe:** it misses `var x any = e` and return statements. These need one more case each. I did not add them.
- **Soundness: same.** A type rule holds on every path. I would not rely on taint tools for a proof, because `go-flow-levee` states false negatives. `unsafe` and `reflect` are banned in strict mode in both arms.
- **Named loss, form:** a derived value needs a conversion and a re-wrap, 1.33 to 1.67. A label needs a named type for each labeled value. A language could put a label on any type at the use.
- **Would change it:** a language design where the agent must also write the re-wrap. Then it is Linter-equal.

### T10. Declared state machines

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** Ran `scratch/lab/cmd/fsm` with the real table of Stockroom statuses. `orders` and `purchasing` pass. A mutant at `orders.go:97` is rejected: `illegal transition Pending -> Shipped`. The rule also rejects a write with no known state before it.
- **Soundness: yes.** The known state comes from the same guards in either arm. A status changed through an alias is a hole in both unless both ban it. My probe reads guards in order and is not a full flow analysis.
- **Form: yes.** The write `o.Status = shop.Shipped` is the same text. Only the table is new.
- **Would change it:** a V rule that gets its known state from a type that Go cannot hold. T11 shows that an analyzer can carry that state too.

### T11. Linear typestate and resources

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes.** Ran `scratch/lab/cmd/linear` on `go/cfg`. It rejects a forgotten value on one path, a double use, a use after close, and a capture by a closure. It accepts a `defer` close and a return that moves the value. Vet's `lostcancel`, and `bodyclose` and `sqlclosecheck` in `golangci-lint`, are the same kind of check in common use.
- **Soundness: yes.** A panic path, a store into a field, and a copy into `any` need the same rules in both arms. My probe covers only local variables and has no rule for a store into a field or `any`.
- **Form: yes at each use.** The code is the same. The mark on the declaration is 1.67 to 1.80.
- **Other reading:** if "by type" means that the Go compiler itself must reject it, L cannot do that, and the verdict is Language-only. The test asks only whether L rejects the same programs, so I do not take this reading.
- **Would change it:** a linear program that a flow check on a CFG cannot express.

### T12. No ambient authority

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** It is T1 with `time.Now`, `os.*`, and `net.*` off the list. `forbidigo` exists for this. Read: arm L already passes each port through a constructor (`PRIMER.md` section 2), and `ports.go` enforces it.
- **Soundness: yes.** A dependency that uses ambient authority is a shared hole until T2 closes it, in either arm.
- **Form: yes.** A port call is 1.25 to 1.33. The whole leaf is 1.05 to 1.25.
- **Would change it:** a V with its own standard library, so that the ambient names do not exist. Spec §5.4c says that code imports Go packages directly.

### T13. Effects in function types

**Verdict: Linter-partial. Confidence: medium.**

- **Reach: yes for first-order code and parameters.** Ran `scratch/lab/cmd/effects`. Effects cross packages as facts, so each package is checked alone. A hidden `net` effect is rejected. A polymorphic `Map` with a pure function passes, and with a dialing function it is rejected.
- **Named loss, reach by rejection:** a Go function type holds no effect. `[]func()` in `b.go:22` is rejected, though `[]func() !net` is legal in a language with effects in types. L accepts it only with a comment that names a path inside a type, and I did not build that.
- **Named loss, form:** an effect inside a type is 1.88 to 2.00. A signature mark is 1.67 to 1.80. A mark on a polymorphic signature is 1.11 to 1.12.
- **Soundness: loss.** A function value that passes through `any` or a type parameter loses a comment, but not a type. L must ban those flows.
- **Other reading:** if only first-order effects and parameters matter, Linter-equal. If L must accept every program with higher-order effects without a comment path, Language-only.
- **Would change it:** a count of how often effectful function values sit in fields and containers in real code.

### T14. Declared dependency boundaries

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** Ran `scratch/lab/cmd/wires` on the real `main.go`. It rejected the wire `web -> events`, which I had left out of its table by mistake. So the check works on real wiring. `wiring.go` already restricts `main` to the form that this needs.
- **Soundness: yes.** Both arms read a closed form of wiring. Rule 1 of spec §5.4 and VL0101 close the import route.
- **Form: yes.** The table is data.
- **Would change it:** a route to a wire that goes around `main` in L. VL0907 and VL0908 reject those that I can name.

### T15. Declared ownership of data

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes.** Ran `scratch/lab/cmd/owners`. A constant write to another part's table is rejected. A query that is not a constant is rejected too, so the rule has no silent hole.
- **Soundness: same limit.** Spec §5.4d gives G12 as out of scope. A table name made at run time, or SQL inside an ORM, defeats both arms unless both ban it. If V adds a typed query form, L can match it with a generator such as `sqlc`.
- **Form: yes.** The call text is the same.
- **Would change it:** a V table type that makes the owner part of the type, so that no string is parsed. A generator can give L the same result, but I did not build one.

### T16. Wire contracts in one compile unit

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** A whole-module analyzer sees both sides, as `go/types` gives one type identity. Read: spec §5.5.2 says the compiler "cannot see the deployed build" and that the lock stands for it. So the mechanism is a lock file and a compare, and a generator can write that. `buf breaking` does this for Protobuf (Doc).
- **Soundness: yes.** Two repositories defeat both arms in the same way.
- **Form: yes.** `Publish` on a generic topic type is the same call.
- **Fact:** V rejects topics and remote ports today with K0009.
- **Would change it:** a wire check that needs code from two builds at once, with no lock.

### T17. Proved idempotency and ordering

**Verdict: Linter-equal. Confidence: low.**

- **Reach: same.** The analysis works on effects and call paths, and T2 and T13 show that facts carry those. A transport's `Guarantees` value is a constant that an analyzer can read.
- **Soundness: same.** Spec §5.5 says that "the compiler cannot prove the claim". I found nothing that makes the proof easier in V, unless T13's loss applies. In that case a function value in a handler weakens L.
- **Form: yes.** A mark is 1.44 to 1.50, on the declaration only.
- **Limit:** I built nothing here. The verdict compares two arms. It does not say that the proof is possible on either arm.
- **Would change it:** a working analysis in V that uses a fact that L cannot get.

### T18. Durable execution from the compiler

**Verdict: Linter-partial. Confidence: medium.**

- **Reach: yes.** Ran `scratch/probes/journal`. A wrapper that a generator can write from a port records each effect. The handler stays plain Go and runs again from its start. After a crash past a record, three charges stay three. The handler must be deterministic, and `workflowcheck` shows that an analyzer can check this (Doc). DBOS Transact and Temporal are libraries for the same job (Doc).
- **Soundness: same.** A crash between an effect and its record repeats the effect: four charges in my test. No lowering closes that window without an idempotency key at the other end.
- **Named loss, form:** the wrapper must know the invocation. So either each port call takes `ctx`, at 1.50 to 1.67, or `main` builds the leaf per delivery. A lowering can hide this.
- **Other reading:** if the theory needs a resume in the middle of a body with no run again, L needs a generator that rewrites bodies. By the spirit of the test, that is a language, so Language-only. If a leaf built per delivery is acceptable, Linter-equal.
- **Would change it:** a measure of the cost of `ctx` in real handlers, many of which have it already.

### T19. Deterministic simulation

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes, by the test's own rule.** A rewrite that picks among legal behaviors in a test is a test tool. A clock value, a map order, a random stream, and a schedule are all legal behaviors.
- **Evidence:** Ran `testing/synctest`: a 48-hour sleep passes at once under a fake clock. It gives no seeded schedule. `gosim` is the tool that goes further, and it **did not run here**. It stopped on a cgo stub for go-sqlite3.
- **Soundness: yes.** V lowers to Go and runs on Go's scheduler. A V simulator must replace the same runtime.
- **Form: yes.** The code does not change.
- **Other reading:** if a translation of all Go code for a test goes past "picks among behaviors", L loses it. V's lowering would then be the same work, on the language side. So the verdict would be Language-only in the letter, with no gain in the substance.
- **Would change it:** a working `gosim` on Go 1.27 that raises the confidence, or a fault that only a lowering can expose.

### T20. Race freedom by type

**Verdict: Linter-partial. Confidence: low.**

- **Reach: yes, on the rule that I built.** Ran `scratch/lab/cmd/send`. It rejects a captured variable that the function writes, a pointer on a channel, a pointer as a `go` argument, and a captured slice. It accepts a value of a type marked `//vl:send`. `checklocks` shows that lock annotations work as an analyzer (Doc).
- **Limit of the probe:** it trusts the mark, and it does not check what a method does inside. Moved ownership needs T11. A real rule needs both, plus a check on the guarded fields.
- **Named loss, form:** a capability inside a type, such as `chan iso *Buf`, is 2.20 to 2.40. A mark is 1.62 to 1.80.
- **Named loss, soundness:** a type parameter holds no capability, except by a comment. Both arms need a table for goroutines that start inside dependencies, such as `net/http` handlers.
- **Other reading:** if rejecting a `go` statement that captures shared data counts as a ban of a core construct, the verdict is Language-only. I do not count it, because V with Go bodies must reject the same programs. If V changes the goroutine model, L cannot follow, and Language-only holds too.
- **Would change it:** a concrete V design. This theory names no design, which is why my confidence is low.

### T21. The cost of a correct program

**Verdict: Linter-partial. Confidence: medium.**

- **Reach: yes, as far as I checked.** Arm L states the same rules, and its 65 conformance cases carry the same ids as V cases. V has 145 cases, so L's suite does not cover all of V's. I did not do a full parity review. Part of V's suite is for constructs that L has in another form.
- **Soundness: yes, as far as I checked.** I found no hole in one arm only. H4, H6, and H7 pass in both.
- **Named loss, form:** the whole arm is 1.146 to 1.152, and 1.06 to 1.08 without the fixed files or with `main` generated. So at the whole-app level L passes. At each site it does not: an invariant declaration is 3.00 to 3.60, and an `ensures` statement is 1.90. Wiring is 2.64 by scanner tokens. Billing, the leaf with the densest contracts, is 1.33 by scanner.
- **The trend:** Stockroom has 29 contract clauses in about 16,000 tokens. The ratio rises with contract density, so a contract-heavy program costs L more.
- **Generators reduce it:** `main` and the port wrappers can be generated, and contracts can be in comments. I did not build this. It removes most of the loss by my estimate.
- **Other reading:** if the factor applies to the whole program, Linter-equal.
- **Would change it:** a generator for L that brings every site under 1.25. Then Linter-equal.

### T22. Machine-applicable fixes

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** Ran `scratch/lab/cmd/fixkeep`. `-json` gives the exact edit with file and offset. `-fix` applies it, and the check then passes.
- **Soundness: yes.** No trust issue.
- **Form: yes.** No code cost.
- **Fact:** today neither arm has exact edits. V has a `Fix` text field (`diag.go`), with one use that I found (`check/wiring.go:349`). L has no `SuggestedFix`. The framework gives L this for free.
- **Would change it:** a fix that needs a rewrite across files that the framework cannot express. `TextEdit` holds edits in any file.

### T23. One way to write a thing, and a small primer

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes.** Subset rules can ban the other forms, in the same way in each arm. Bodies are Go in both (spec §6).
- **Soundness: yes.** Nothing depends on it.
- **Form: yes.** The primer of L is smaller: 2,370 against 2,651 cl100k tokens, and 5.6 percent fewer pieces.
- **Limit:** L adds rules of pure form, such as VL0903 and VL0904 (the first statements, `Keep` in each pointer method). V states these in syntax. They are in the primer count already. I did not measure first-attempt success.
- **Would change it:** a measured first-attempt rate that differs between the two arms for a cause in the form. A V that also adopts T7 or T8 would need a new primer count.

### T24. Typed holes

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** Ran `scratch/lab/cmd/hole` on `orders.go` with `_hole` in place of an argument. It tolerated the one type error. It answered with the expected type (`[]shop.Line`), the names in scope, and each port on the receiver with its methods.
- **Soundness: yes.** V uses the same `go/types` (spec §11.1).
- **Form: yes.** A hole is a name.
- **Would change it:** nothing that I can name for the mechanism. The effect on R4 and on turns is not measured, and the test does not ask for it.

### T25. Local reasoning

**Verdict: Linter-equal. Confidence: medium.**

- **Reach: yes.** The same boundary rules hold, so the same facts stay local: `ports.go` holds what `component.vulcan` holds.
- **Soundness: yes.**
- **Form: yes.** Leaf with ports: 1.05 to 1.25 by cl100k.
- **Limit:** `main.go` is larger than `system.vulcan` by 2.64 times. A tool can cut the wires of one leaf from it. `vulcan context` is itself not in this release (`deferred` in `cmd/vulcan/main.go`), so both arms need a tool. I built neither.
- **Would change it:** a measured count of files read that differs because `main.go` or the field form of ports misleads an agent.

### T26. Tests derived from contracts

**Verdict: Linter-equal. Confidence: high.**

- **Reach: yes.** Ran `scratch/probes/derive/gen`. It reads `contract.Requires` from plain Go and writes a `testing/quick` test as ordinary Go. The test finds a seeded defect in `Clamp`. `rapid` adds shrinking and state machines (Doc). A wrapper from T18 gives double delivery and crash points.
- **Soundness: yes.** A contract statement is structured text. `model.PrologueOf` in `poc/golint/model` already reads it.
- **Form: yes.** The test is generated.
- **Limit:** the probe handles integers only.
- **Would change it:** a contract form that V can read but that a generator cannot read from L. I found none.

## Theories of harm

### N1

**Verdict: Language-only. Confidence: medium.**

Only V has a new syntax, new declaration files, and a non-Go extension. Spec R8 states this risk. L keeps Go text, and models know interfaces and constructors. L has its own unknown rules and library, but those are not a new language, and V has the same rules.

- **Other reading:** if "new language" includes any new model to learn, L carries part of it, and the verdict is Linter-partial. Evidence for this: the two primers have about the same size.
- **Limit:** I did not measure the cost. V's bodies are Go, so the cost can be small.
- **Would change it:** a run that shows no gap between the arms from syntax, which would remove the harm and not move it.

### N2

**Verdict: Linter-equal. Confidence: medium.**

A proof, an annotation, or an effect mark costs the same work in a comment or in a clause. cl100k shows 1.00 to 1.20 for clauses in comments. A solver timeout does not know which arm called it.

- **Note:** the marks that L needs on declarations cost a little more. See the table.
- **Would change it:** a V that infers what L must write. T13 and T21 name the sites.

### N3

**Verdict: Linter-equal. Confidence: medium.**

The same rules reject the same programs, so the same pressure applies. Each arm also has its own extra rejections. L has rules of form (VL0901 to VL0908). V has syntax and blanking errors in a form that the model has not seen.

- **Would change it:** a measure of rejections and of runs that do not stop, per arm. I did not run agents.

### N4

**Verdict: Linter-equal. Confidence: medium.**

An agent can remove a `requires` clause or a `contract.Requires` call with the same ease. It can add a marker in either form, or edit `system.vulcan` or `arch/config.go`. Read: `vulcan-lint` honors no suppression comment, and `poc/app/golint/PROTECTED` lists protected files.

- **Other reading:** `go build` in L succeeds without the check, while `vulcan build` refuses. If nothing binds the check to the build, L carries more of this harm, and the header of the harms section has no slot for it. See the section below.
- **Would change it:** a measure of how often agents weaken checks, per arm.

### N5

**Verdict: Language-only. Confidence: high.**

Go text in L means what Go says. The harm exists only where V text looks like Go and means something else. Today that is small: `Checkout.Pay(c, amt)` reads as a package call but is a port call. `setup` and `startup` run by name. A risky deferred call adds a stack frame (spec §7.3). T5 and T8 would make the harm large.

- **Would change it:** a linter arm that uses the barred rewrite. Then that arm is a language by the test.

### N6

**Verdict: Linter-equal. Confidence: medium.**

Ran `git merge-file` with two parallel changes that each add one leaf and one wire. `system.vulcan` gives two conflict regions in one file. `arch.go` and `main.go` give one region each in two files. Both arms conflict.

- **Note:** L has a real extra cost here. Two new ports on one leaf change the same line, `orders.NewService(catalog, ledger, db, hub)`. V adds separate wire lines, and could place them apart. `main.go` is also 2.64 times the size. A generated `main` removes both costs.
- **Would change it:** a run of parallel agents that counts conflicts per arm. The header has no slot for L carrying more than V.

## Where the test seems wrong or unclear

I gave each verdict by the test as written, with my reading stated. These are the points where the test did not decide.

1. **Most of V does not exist.** "Fewer tokens than V" compares L with a guess at V's syntax. All the site ratios for T7, T8, T9, T11, T13, and T20 depend on my guess. A designer could move each one.
2. **"Place of use" against the whole program.** T21 passes as a whole (1.15) and fails at each site (3.00 for an invariant). T8's helpers cost 1.75 to 2.38 at each site and about 1 percent of the app. The test says "at each place of use", so I used the site, but I doubt that a site ratio on a rare construct predicts cost. Is a declaration a place of use? I treated a mark on a declaration as a small cost, not as a failure.
3. **A comment mark at a site.** A fixed prefix such as `//@` makes a short clause look worse under pieces (1.75) than under cl100k (1.20). The test does not name a tokenizer.
4. **A ban that V also needs.** The table counts a ban against L only. For T7 and T20, V with Go bodies must reject the same forms. I did not count such a ban, and this choice decides both verdicts. For a fair test, the question should be "does L ban what V keeps?"
5. **A generator against a rewrite.** Generators are allowed and rewrites are not, but V itself is a generator from Go-like text. A tool that reads a Go body and writes a new body sits on the line. T18 and T19 depend on this. The test's own example covers only a schedule in a test.
6. **"Reject the same programs" does not fit every theory.** T5, T6, T21 to T26, and the harms are about cost or proof rate, not about rejection. I read Reach as "delivers the same result". I did not measure any agent's success rate, so these verdicts say only which arm can carry the feature.
7. **The check is not bound to the build in L.** `go build` works with no linter. `vulcan build` refuses. The test does not say whether L may count on a gate in CI, or on a build wrapper that only refuses and rewrites nothing. I assumed a gate. Without one, every Linter-equal verdict has a named soundness loss and becomes Linter-partial. A gate is cheap, so I do not think this is the right reading, but it is the largest single assumption in this file.
8. **The harm header has no slot for "L carries more".** N4 and N6 both have a part where L is worse. The options are only Language-only, Linter-equal, and Linter-partial.
9. **A verdict does not say that a feature works.** T17 is Linter-equal because both arms face the same open problem. Read it as "no gap", not as "possible".

## Sources

- [Gobra overflow checking](https://viperproject.github.io/gobra-book/01/overflow.html), [Gobra basic specifications](https://viperproject.github.io/gobra-book/basic-specs.html), [Gobra tutorial](https://github.com/viperproject/gobra/blob/master/docs/tutorial.md)
- [testing/synctest](https://pkg.go.dev/testing/synctest), [gosim announcement](https://groups.google.com/g/golang-nuts/c/XBB7Nzvqvhs)
- [Temporal workflowcheck](https://pkg.go.dev/go.temporal.io/sdk/contrib/tools/workflowcheck), [DBOS Transact](https://dbos.dev/dbos-transact)
- [go-flow-levee](https://pkg.go.dev/github.com/google/go-flow-levee@v0.1.4), [gVisor checklocks](https://pkg.go.dev/gvisor.dev/gvisor/tools/checklocks)
- [buf breaking](https://buf.build/docs/breaking/), [rapid](https://pkg.go.dev/pgregory.net/rapid)
- [go-arch-lint](https://beta.pkg.go.dev/github.com/fe3dback/go-arch-lint@v1.7.0-rc1), [Cadence maprange](https://pkg.go.dev/github.com/onflow/cadence/tools/maprange)
