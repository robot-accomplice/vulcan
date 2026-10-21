# Decision 1: contract timing

Status: accepted, 2026-10-20 (option C, with rule 5). Applied in spec §7.3, Appendices C, E,
and F, and the primer. Milestone 0, item 1 of the
[development plan](../development-plan.md). Answers [audit 2](../../experiments/audit-2.md), H6
and M6.

## Question

When control goes back to the caller, what do `ensures` and a struct invariant guarantee? Cover
named results, deferred calls, deferred unlocking, panic, recovery, aliases, and concurrent access.

## Current rule

- `ensures` and the invariant are checked at each `return` and at a reachable end of the body,
  before the deferred calls run (§7.2, Appendix F step 3). Thus a method that unlocks with
  `defer mu.Unlock()` is checked while it holds the lock.
- A deferred call that changes a named result after the check is not checked. `keel audit` lists
  each such function.
- "A panic reaches no `return`", so a panic is not checked.
- The §7 table says that an invariant "is true after construction and after the outermost method
  call returns".

## Problems

A test program (Go 1.26.1) confirmed each case below. In each one, the caller gets control back
with no failed check.

1. **Recovery in the function itself.** A method breaks its invariant, panics, and recovers in a
   deferred function literal that sets `err`. Go returns normally from the method. No `return`
   statement ran, so no check ran. The value stays broken (`Balance` -450 with
   `invariant Balance >= 0`).
2. **A deferred change to a result.** The common close-error idiom
   `defer func() { if cerr := f.Close(); err == nil { err = cerr } }()` turns a success into a
   failure after the check. With `ensures err != nil => Balance == old(Balance)`, the caller gets an
   error and a changed balance. The contract is false, and nothing reports it. `keel audit` lists
   the function, but the guarantee in §7 does not mention the exception.
3. **Recovery in a caller.** A method breaks its invariant and panics. A caller recovers, as
   `net/http` does for each request. The value stays broken and later requests use it.
4. **A deferred write to a field.** A deferred function literal in a method can write a field
   after the invariant check. §7.2 counts a deferred literal as inside the method, so it is
   allowed.

## Constraints

- **Bodies stay Go** (design rule 4). A rule can reject a body. It must not change what a correct
  body does. A contract failure can panic, because it happens only when a contract is false.
- **No hidden fields.** A flag in each struct, to mark a value as broken, would change its size
  and layout for `unsafe`, `reflect`, and binary encodings.
- **Checks must not add data races.** A check that reads fields must run while the method holds
  its lock. With `defer mu.Unlock()`, that means before the deferred calls run. An invariant check
  at method entry would run before the body takes the lock, so it cannot be the answer.

## Options

**A. Keep the rule and state the exceptions.** Change only the text of §7. Cost: none. The four
holes stay open, and the §7 promise becomes "checked at each `return` statement".

**B. Reject the risky forms.** In a function with a contract, or a method of a struct with an
invariant, `recover` and deferred assignments to named results are compile errors. This is sound
for cases 1, 2, and 4, and simple to implement. It rejects common Go: the close-error idiom, error
wrapping in a `defer`, and recovery in servers. It does nothing for case 3.

**C. Check every normal return (recommended).** Keep the check before deferred calls, and add one
more check point after the deferred calls, generated only where a deferred call can change the
outcome. Close case 4 with one compile error. Report case 3 at the panic, and state the limit.

## Recommendation: option C

### Rules

1. **Return check (unchanged).** `ensures` and the invariant are checked at each `return` and at a
   reachable end of the body, before the deferred calls run.
2. **No deferred field writes.** In a method of a struct `T` with an invariant, a deferred call
   cannot write a field of `T`. This covers a deferred function literal and a deferred call of
   `keel_M`. A deferred call of a method of `T` through the receiver is an outer call, so it calls
   the checked `M` (§7.2). A violation is a compile error (new K0522). The fix-it moves the write
   before the `return`.
3. **Check after the deferred calls.** A function gets a second check point when its contract
   reads a named result and it has a **risky deferred call**:
   - a deferred function literal that assigns a named result, takes its address, or calls
     `recover`;
   - a deferred call of a named function that receives the address of a named result, or calls
     `recover` directly;
   - a deferred call through a function value or an interface value, because the compiler cannot
     see its body.

   At the second check point, the compiler checks `ensures` with the final values of the results.
   It does not read the state again. At the return check (rule 1), it saves by value each atom of
   an `ensures` conjunct that reads both a named result and other state. Examples are
   `a.Balance` and `len(a.items)`. The second check point uses these saved values. Thus it reads no
   field after a deferred unlock. A conjunct that passes state by reference to a `pure` function,
   for example `IsSorted(a.items)`, cannot be saved. If such a conjunct also reads a named result,
   and the function has a risky deferred call, that is a compile error (new K0523). The hint says
   to split the conjunct, or to assign the result before the `return`.
4. **Recovered panics.** When a function with a risky deferred call returns normally and no return
   check ran, a panic was recovered. The second check point then checks all of `ensures` and the
   invariant against the current state. No values were saved, so this check reads the state after
   all the deferred calls. If the function also unlocks in a deferred call, the check runs after
   the unlock. That is a warning (new K0524), and the race detector reports a real race in tests.
5. **Panics that leave the function.** No result reaches the caller, so `ensures` is not checked.
   When a panic leaves an outer method of `T` and no return check ran, the generated code checks
   the invariant. If the invariant is false, the panic continues as a `*keel.ContractFailure`. The
   failure names the invariant, the values, and the original panic value, which `Unwrap` and
   `errors.As` return. The original panic value changes only when the invariant is false. This
   check can run after a deferred unlock. It runs only on a panic path, so it has no warning:
   K0524 would otherwise fire on almost every method that locks. **Limit:** a caller that
   recovers this panic, for example `net/http`, can still use the broken value. Vulcan reports the
   break. It does not prevent the later use.
6. **Concurrent access.** The guarantees hold for programs without data races. A check runs where
   the method returns, under the locks it then holds. Vulcan does no static race analysis (§6). The
   race detector runs in `keel test`. K0521 stays.
7. **Aliases.** No change. §7.2 checks calls that receive `T`. Elements of slice and map fields
   stay a stated limit.

### The guarantee, as §7 would state it

> When a function with `ensures` returns normally, its results satisfy `ensures`, including after
> its deferred calls and after a recovered panic. State that the conjuncts read is as it was at the
> `return`, or, after a recovered panic, as it is after the deferred calls.
>
> When an outer method call on a struct with an invariant returns normally, the value satisfies
> the invariant. When a panic leaves the call and the value breaks the invariant, the panic becomes
> a contract failure. This holds for programs without data races. It does not cover elements that
> a slice or map field shares with a copy (§7.2).

### Lowering (Appendix F)

Only a function with a risky deferred call changes. Its body runs in a generated closure, and
the generated function checks after the closure returns:

```go
func (a *Account) Withdraw(x int) (err error) {
    // requires checks
    var keel_checked bool
    var keel_s0 int                      // saved a.Balance
    err = func() (err error) {
        // the body, unchanged, with its defers; each return check also
        // sets keel_checked = true and keel_s0 = a.Balance
    }()
    if keel_checked {
        // ensures with the final err and keel_s0 in place of a.Balance
    } else {
        // recovered panic: ensures and invariant on the current state
    }
    return err
}
```

`recover` in the body's deferred calls still runs in a function that the deferred call made
directly, so its behavior does not change. The test program confirmed this. A panic that leaves
the closure leaves the generated function too, so `keel_checked` is read only on a normal return.
For rule 5, each outer method of `T` with an invariant gets one generated deferred call that runs
last. It does nothing when the return check ran.

Cost: one closure call and one flag, only in functions with a risky deferred call. Milestone 2
measures it.

**Limit (added after audit 3, L1).** The closure changes what `runtime.Caller` and
`runtime.FuncForPC` report: a body that names its caller sees `pkg.F.func1` instead of `pkg.F`,
and code that walks the stack sees one more frame. The line stays right through `//line`. Go gives
user code no way to hide a frame. A lowering without the closure would need one generated
deferred call that tells a recovered panic from a running one, which Go cannot do without calling
`recover` itself. Spec §7.3 states the limit, and conformance case D12 pins it.

## Examples

| Code | Result |
|---|---|
| Close-error idiom in a function with `ensures err != nil => Balance == old(Balance)` | Accepted. The second check uses the final `err` and the saved `Balance`, and fails when the close turns success into failure after a change. |
| `defer func() { if err != nil { err = fmt.Errorf("withdraw: %w", err) } }()` with `ensures err == nil => ...` | Accepted. The wrap runs only on failure, so `err` stays nil on success and the check passes. Without the `if`, `fmt.Errorf` returns a non-nil error, and the check reports it. |
| `defer func() { a.busy = false }()` in a method of `Account`, which has an invariant | Rejected: K0522. |
| `defer a.reset()` in a method of `Account` | Accepted. It calls the checked `reset`. |
| Self-recovery that sets `err` after `Balance` went negative | Accepted. The second check point finds `Balance >= 0` false and fails. |
| `defer cancel()` from `context.WithTimeout` | Accepted. It is a risky deferred call, so the function gets the second check point, at the cost above. |
| `ensures err == nil => IsSorted(a.items)` with a deferred literal that assigns `err` | Rejected: K0523. |
| A panic that leaves `Withdraw` after `Balance` went negative | The panic continues as a contract failure that wraps the original value. |

## Spec changes on acceptance

- §7 table, `invariant` row, and §7.2: state the guarantee above, and replace "Checks run before
  deferred calls" with rules 1 to 6.
- Appendix F step 3 and "Outer calls": the second check point, the saved values, and the
  unwinding check.
- Appendix E: K0522 (error), K0523 (error), K0524 (warning).
- §10 `keel audit`: drop the list of deferred result changes. Add each function with a second
  check point and each K0524 site.
- `docs/guide.md`: one sentence on deferred calls and recovery in functions with contracts.

## Open points

- Rule 5 changes the panic value that a recovering caller sees, but only when the invariant is
  false. Confirm that this is acceptable under design rule 4.
- Treating each deferred call through a function value as risky is safe, but it adds the closure
  to many functions (`defer cancel()`). A later refinement can exclude function values whose
  source is known, such as the result of `context.WithTimeout`.
