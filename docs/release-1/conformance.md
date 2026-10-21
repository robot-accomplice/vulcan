# Release 1: first conformance cases

Status: accepted, 2026-10-21. Part of the milestone-1 scope of the
[development plan](../development-plan.md). The syntax and checks are in [syntax.md](syntax.md).

These are the first cases of the conformance suite. Each one becomes a test of the compiler when
milestone 1 or 2 implements its construct. The suite grows with each bug and each decision.

## Format

Each case is a directory `conformance/<id>/` in the compiler repository. It holds a small Vulcan
project and a file `want.txt`. Each line of `want.txt` is one expectation:

```
error  api/orders/orders.vulcan:12:5  K0101
warn   api/billing/ledger.vulcan:40:2  K0525
run    exit=0  stdout=testdata/out.txt
panic  kind=ensures  at=api/billing/ledger.vulcan:22:5
```

The runner builds the project with the compiler under test, and compares. **Positions are part of
the expectation**: a diagnostic at the wrong line or column fails the case. An `error` case expects
exactly the listed diagnostics, no more. A `run` case builds and runs the program, and compares
the exit code and the output.

Below, **Accept** means the project builds with no diagnostic except the listed ones.

## Lexing and positions (milestone 1)

| ID | Input | Expect |
|---|---|---|
| S01 | A method with three contract lines between its signature and a `{` on its own line | Accept (milestone 2), or K0009 at the first `requires` (milestone 1). A type error later in the body reports its true line and column. |
| S02 | `func F() requires { … }`: a result of a type named `requires` | Accept. Go meaning (Appendix B.1). |
| S03 | A struct field named `limit`, a local variable named `requires`, a parameter named `timeout` | Accept. |
| S04 | `limit` on the line after its field | K0001 at the `limit`. |
| S05 | A declared name `vulcan_x` | K0005 at the name. |
| S06 | A body with a Go type error, inside a file with contracts above it | K0008 at the true `.vulcan` position. |
| S07 | A panic in a body at a known line | The stack trace names the `.vulcan` file and line (`//line`). |
| S08 | Each deferred construct of syntax.md, one case each (topic, remote port, `via`, `limit`, `handle`, `pure`, a second container, `vulcan fmt`) | K0009 at the construct, naming it. |
| S09 | A Go 1.26 feature in a body, for example a range-over-func iterator | Accept, and the same output as the same body in Go. |

## Architecture and wiring (milestone 1)

| ID | Input | Expect |
|---|---|---|
| A01 | Leaf `Orders` imports the package of leaf `Billing` | K0101 at the import. |
| A02 | Leaf `Orders` names a type of `Billing` in its `component.vulcan` | K0110. |
| A03 | `Orders` needs port `Checkout`, and `system.vulcan` has no wire for it | K0104 at the port declaration. |
| A04 | Two wires for `Orders.Checkout` | K0105 at the second wire. |
| A05 | A wire names `Billing.NewPayr`, which does not exist | K0107 at the name. |
| A06 | The provider lacks one port method, or has it with a different parameter type | K0108, naming the method and both signatures. |
| A07 | `Pay` has a pointer receiver, and the constructor `NewPayer` returns `Payer`, not `*Payer` | K0108 (method sets, §5.3). A wire that names the type `Payer` instead provides `new(Payer)`, a `*Payer`, and is accepted. |
| A08 | Wires `A.P: B.New`, `B.Q: C.New`, `C.R: A.New` | K0111, listing the cycle. |
| A09 | A port method parameter of a type private to `Orders` | K0112, with the fix-it to move it to a library. |
| A10 | Code assigns to the port name `Checkout` | K0116. |
| A11 | A leaf declares `func init()` | K0117. |
| A12 | A package-level `var` in a leaf whose initializer calls a port | K0117. |
| A13 | `//go:linkname` in leaf `Orders` that names a function of `Billing` | K0122 at the directive. |
| A14 | A library imports a leaf | K0204. |
| A15 | A library `var` assigned in a function after startup | K0202 at the assignment. |
| A16 | Two leaves with one name in different groups | K0121. |
| A17 | A wire names a group | K0113. |

## Startup and shutdown (milestone 1)

| ID | Input | Expect |
|---|---|---|
| W01 | Two leaves, a port between them, a library type in the signature, an entry leaf whose `startup` calls the port and returns | `run exit=0`, with the output of the call. The vertical slice of milestone 1. |
| W02 | A constructor returns `(T, error)` and fails | The process exits 1 before any `setup`, with a report that names the constructor. |
| W03 | A `setup` returns an error | Exit 1; no `startup` runs. |
| W04 | A `startup` blocks until its `ctx` is cancelled; the process gets a stop signal | `startup` sees the cancellation; exit 0 within `shutdown_timeout`. |
| W05 | A constructor calls its own port | Accept; the call reaches the provider (§6, step 2). |
| W06 | `setup` with the wrong signature | K0601. |
| W07 | A secret setting given a literal | K0605. |
| W08 | Two flags that map to one environment variable | K0602. |

## Contracts (milestone 2)

| ID | Input | Expect |
|---|---|---|
| C01 | `requires amt > 0`; a caller passes 0 at runtime | `panic kind=requires` at the call, with the value. |
| C02 | `requires amt > 0`; a caller passes the constant 0 | K0501 at the call. |
| C03 | `ensures err == nil => result.Amount == amt`; the body returns a wrong amount | `panic kind=ensures` at the `return`. |
| C04 | `ensures result.Amount == amt` in a function with an error result | K0502. |
| C05 | `requires xs[0] > 0` | K0504. |
| C06 | `requires len(xs) > 0 && xs[0] > 0` | Accept. |
| C07 | `old(xs)` with a slice `xs` | K0509. |
| C08 | Exact arithmetic: `ensures a.Balance == old(a.Balance) + x` where the body's `+=` wraps at `math.MaxInt64` | `panic kind=ensures`: the contract sees the true sum. |
| C09 | `invariant Lo < Hi` | K0506 (the zero value breaks it). |
| C10 | A field of a struct with an invariant written from a function that is not its method | K0507. |
| C11 | `&e.Amount` outside the methods | K0508. |
| C12 | `Transfer` calls `a.Withdraw` (no check) and `to.Deposit` (checked) | The invariant of `to` is checked at `Deposit`'s return; `a`'s only at `Transfer`'s return. |
| C13 | A port contract, and a fake bound with `vulcan.Bind` that breaks it | The test fails with `kind=ensures` at the port. |
| C14 | A provider whose own `requires` is stronger than the port's | K0109. |
| C15 | `json.Unmarshal(b, a)` where `a` points to a struct with an invariant, and the JSON breaks it | `panic kind=invariant` after the call (§7.2). |
| C16 | A slice field of a struct with an invariant over its elements, changed through a returned copy | Accept; `vulcan audit` lists the method that returns the field (a stated limit). |
| C17 | A type assertion in a contract | K0512. |

## Deferred calls and panics (milestone 2, decision 1)

| ID | Input | Expect |
|---|---|---|
| D01 | The close-error idiom changes `err` after `a.Balance` changed, with `ensures err != nil => a.Balance == old(a.Balance)` | `panic kind=ensures` from the second check point. |
| D02 | Error wrapping in a `defer`, guarded by `if err != nil` | Accept; on success `err` stays nil and no check fails. |
| D03 | A deferred literal in a method writes a field of its struct with an invariant | K0522. |
| D04 | A method breaks its invariant, panics, and recovers in a deferred literal that sets `err` | `panic kind=invariant` at the second check point. |
| D05 | The same, but the invariant holds after recovery | Accept; returns the recovered `err`; no panic. |
| D06 | A method breaks its invariant and panics; its caller recovers | The recovered value is a `*vulcan.ContractFailure` with `Cause` equal to the original panic value. |
| D07 | The same, but the invariant holds | The caller recovers the original panic value, unchanged. |
| D08 | `ensures err == nil => IsSorted(a.items)` with a risky deferred call | K0523 (when tier 2 exists), else K0009. |
| D09 | A recovered-panic check after `defer mu.Unlock()` | K0524. |
| D10 | `defer cancel()` from `context.WithTimeout` in a function with `ensures` | Accept; the second check point exists; `vulcan audit` lists it. |
| D11 | A method of a struct with an invariant that unlocks without `defer` | K0521. |
| D12 | A function with a risky deferred call whose body logs `runtime.FuncForPC` of `runtime.Caller(0)` | `run` output names `pkg.F.func1` and the right `.vulcan` line: the stated limit of §7.3. A change to the lowering that fixes it updates this case. |

## Effects (milestone 2, decision 3)

| ID | Input | Expect |
|---|---|---|
| E01 | `return n, tx.Commit()` in a function with `ensures` | K0525 at the `return`. |
| E02 | E01 with a failing `ensures`, run against a test database | The row is committed; the panic follows the commit (no rollback). |
| E03 | Check, then `return tx.Commit()`, with `defer tx.Rollback()` and a failing check | No row is committed; the panic precedes the commit. |
| E04 | `on_contract_failure: exit`; a contract fails inside an HTTP handler | The process exits 3; `net/http` does not recover it. |
| E05 | E04 with the default `panic` | `net/http` recovers; the next request is served. |
| E06 | A function with `ensures` that calls a gateway port before its return | `vulcan audit` lists it as "detects after effects". |

## Network, libraries, and strict mode (milestone 2, decision 2)

| ID | Input | Expect |
|---|---|---|
| N01 | A leaf that is not a gateway or an entry leaf calls `http.Get` | K0205, with the call path. |
| N02 | The same through a third-party SDK that reaches `(*http.Client).Do` | K0205, with the full path into the SDK. |
| N03 | A library calls `os.Open` | K0203. |
| N04 | A library queries through a `*sql.DB` it receives | Accept. |
| N05 | Strict: a port method takes a `func(OrderID)` | K0123. |
| N06 | Strict: a port method returns `io.ReadCloser`; then the same with `strict_interfaces: ["io.ReadCloser"]` | K0123; then accept, and `vulcan audit` lists the entry. |
| N07 | Strict: a leaf imports `unsafe` | K0124. |
| N08 | Strict: an entry leaf calls `http.Get` | K0125. |
| N09 | Strict: a library `var Events = make(chan Event)` | K0210. |
| N10 | Strict: a leaf calls `os.Setenv` | K0211. |
| N11 | Strict: N05 to N10 in a leaf listed in `strict_exempt` | Accept; each appears in "Unchecked paths". |
| N12 | Ordinary mode: N05 to N10 | Accept; each appears in "Unchecked paths", and `vulcan check` prints the count. |

## Tools (milestone 2)

| ID | Input | Expect |
|---|---|---|
| T01 | `vulcan check --json` on A01 | One JSON object with code, position, message, and hint, in a stable order. |
| T02 | Two runs of `vulcan build` on one commit | Byte-identical generated Go (decision 5). |
| T03 | `vulcan context Orders` | The leaf's `component.vulcan`, its wires, and the method sets of the providers on the other end; nothing from other leaves' bodies. |
| T04 | `vulcan query Billing.NewPayer` | The compiler's symbol, with its signature and position. A misspelled name returns no match, not a guess. |
| T05 | `vulcan guide` | At most 4,000 tokens; no section on a deferred construct. |

## Differential tests

For each case that accepts and runs, the suite also builds the same bodies as plain Go, with the
Vulcan additions removed, and compares the output. They must agree on every run where no
contract fails, because a rule may reject a body but never change what it means (design rule 4).

## Spec gaps these cases expose

Writing the cases showed no new gap. Two points need care in the implementation:

- S01 and S06 depend on the blanking lexer keeping columns (decision 4). The prototype kept them
  for every token except a moved `{`.
- D06 and D07 depend on the generated deferred call running last (Appendix F, "Outer calls").
