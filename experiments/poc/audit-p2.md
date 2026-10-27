# Audit of PoC phase P2 (independent)

Commit `ccffcce`, branch `main`. Auditor: an independent session that did not write the code.
Each reproduction ran with `vulcan.exe` built from this commit, in throwaway projects outside the
repository. The base project is conformance case C12 (`Web`, `Orders`, `Billing`, `lib/money`).
Unless stated otherwise, only `api/billing/billing.vulcan` or `api/orders/component.vulcan`
changes.

## Verdict

P2 is not ready for the agent runs. One Blocker and five High findings remain.

- The Blocker: the check after the body runs after a deferred `Unlock`. A correctly locked struct
  with an invariant (the shape of the Billing ledger) gets false contract failures under
  concurrency. This is a consequence of a listed deviation, but the deviation does not say it, and
  it changes the meaning of a valid program.
- Five High findings: valid Go input that fails to compile in generated code (`vulcan test` with a
  port contract, named port results, two `old()` in a port, a port parameter `w`, a shadowed
  receiver name), and three ways around the invariant or the contract rules that the primer says
  are enforced (elided literals and conversions, the shadowed receiver, calls with effects in
  contracts).

The conformance suite (108 cases) passes, and `gen_cases.py` regenerates it byte for byte. The
suite does not cover the shapes below.

Counts: 1 Blocker, 5 High, 3 Medium, 8 Low.

## Blocker

### B1. The invariant and `ensures` are read after a deferred unlock: false failures and data races

Location: `internal/lower/lower.go:509-527` (the body runs in a closure; the checks follow it).

Input:

```go
type Ledger struct {
	mu       sync.Mutex
	Charged  int
	Refunded int
}
	invariant Charged >= Refunded

func (l *Ledger) ChargeAndRefund(x int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Charged += x
	l.Refunded += x
}
```

`Pay` starts 8 goroutines that each call `l.ChargeAndRefund(1)` 200,000 times on one ledger.
Every write holds the lock. The invariant is true at every unlock.

Observed, in 3 of 3 runs:

```
panic: contract failure: Ledger invariant: Charged >= Refunded (api/billing/billing.vulcan:15:12)
		Charged = 1751
		Refunded = 4780
```

Cause: the generated method is `vulcan_r := func() {...body with defer l.mu.Unlock()...}();
l.Vulcan_invariant()`. The invariant reads `Charged`, then `Refunded`, without the lock, while
other goroutines write them. This is a data race. `vulcan test` with the race detector (on a
host that has it) also reports it and fails the test.

Why it matters: the plan's Billing leaf is "a ledger with an invariant" behind an HTTP API, so a
mutex with `defer Unlock` is the natural design. Arm V then gets panics that arm G never sees, or
race reports in CI. Status lists "one check point after the body" as a deviation, but not this
consequence. §7.2 ("Checks run before deferred calls") exists to prevent exactly this. Design rule
4 says a rule must never change what a body means. The deviation is acceptable for functions
without a deferred unlock, not for this case.

Fix: check at each `return` before the deferred calls, as Appendix F, step 3, already describes:
rewrite `return e0, e1` to a block with typed temporaries, the checks, and `return t0, t1`; check
the named results on a bare `return`; check at a reachable end of the body. Keep the closure only
for a function with a risky deferred call (§7.3 rule 3). Add a conformance case with a
mutex-guarded invariant under concurrent calls, run with `-race` where the host can.

## High

### H1. A port contract that uses a package not in the signatures breaks `vulcan test`

Location: `internal/gen/leaves.go:213` (`stubFile` calls `usedImports(p, pkg, sigTexts(pkg))`),
with `sigTexts` at `leaves.go:165-170`, which now adds the contract texts.

Input (`api/orders/component.vulcan`), and an empty test in `orders_test.vulcan`:

```
import (
    "math"
    "shop/lib/money"
)
component Orders {
    needs port Checkout {
        func Pay(amt money.Money) (money.Receipt, error)
            requires amt < math.MaxInt32
    }
}
```

Observed: `vulcan run` works. `vulcan test Orders` fails:

```
.vulcan/build/api/orders/vulcan_stub_test.go:6:2: error K0008: "math" imported and not used
exit=1
```

Why it matters: every test run of the leaf fails, with a position in a generated file that the
agent cannot edit. Any package used only in a port contract triggers it (`math`, `strings`,
`time`, a library constant). The plan counts a toolchain bug as a failure of arm V.

Fix: give the stub file only the imports of the signatures (keep a separate `sigTexts` without
contracts), or add `var _ = pkg.X` guards. Add a conformance case: a port contract with an import
used only there, plus a test in the leaf.

### H2. A shadowed receiver name skips the invariant check, or fails to compile

Location: `internal/lower/lower.go:535-568` (`receiverCalls` matches the receiver by its spelling).

Input 1:

```go
func (a *Account) Withdraw(x int) { a.Balance -= x }

func (a *Account) Spread(others []*Account, x int) {
	for _, a := range others {
		a.Withdraw(x)
	}
}
```

`me.Spread([]*Account{other}, 5)` with `other.Balance == 1`. Observed: `other balance -4`, exit 0.
No contract failure. The inner `a` is another account, so §7.2 requires the check at the return
of `Withdraw`. The lowering called the unchecked twin `vulcan_Withdraw`.

Input 2: the same, but the loop ranges over `[]*Log`, and `*Log` has its own `Deposit` method.
Observed:

```
api/billing/billing.vulcan:19:5: error K0008: a.port Deposit undefined (type *Log has no field or method port Deposit)
```

Valid Go fails to compile, with a message that names a "port" that does not exist.

Why it matters: input 1 is a silent way around the invariant. Input 2 is a toolchain failure on
valid Go. Reusing the receiver's name in a loop or closure is common.

Fix: decide "the receiver variable" by object, not by text. The lowering has no types, so either
resolve scopes in the AST (each `:=`, `range`, function literal parameter, and type switch that
declares the name hides it), or move the rewrite after type checking and compare
`types.Info.Uses[id]` with the receiver's object. Also treat a `range` that assigns the receiver
as an assignment.

### H3. The construction check misses elided literals, conversions, and aliases

Location: `internal/lower/lower.go:661-715` (`literals`; `checker` needs an explicit `lit.Type`).

Input:

```go
type Account struct{ Balance int }
	invariant Balance >= 0
type Raw struct{ Balance int }
type Alias = Account

xs := []Account{{Balance: -1}}
ps := []*Account{{Balance: -2}}
m := map[string]Account{"k": {Balance: -3}}
c := Account(Raw{Balance: -4})
al := Alias{Balance: -5}
```

Observed: all five values are built, and the program prints `-1 -2 -3 -4 -5` and exits 0.

Why it matters: §7.2 and the primer ("on each struct literal of the type") say construction is
checked. `[]T{{...}}` is the usual way to write seed data and table tests. The grader's seeded
violations decide the label "enforced by V"; a seed of this shape makes the label wrong.

Fix: run the literal check after type checking: for each `*ast.CompositeLit` whose type
(`info.TypeOf`) is an invariant type, or a pointer to one when the `&` is elided, wrap it. Wrap a
conversion `T(x)` whose target type is an invariant type. Resolve aliases with `types.Unalias`.

### H4. Contracts can call any function, with effects

Location: no check exists. `internal/lower/lower.go:429-496` lowers any call, and `checkNames`
(`lower.go:571-633`) looks only at names and type assertions.

Input:

```go
var count int
func bump() bool { count++; return true }

func Charge(x int) int
    requires bump() && x > 0
    ensures fmt.Sprintln("effect") != "" && os.Getenv("HOME") != "zz"
{
	return x
}
```

Observed: accepted. At runtime `count` is 1 after one call.

Why it matters: §7.1 says contracts are pure. Release 1 and the plan accept tier 1 only, with
`pure` and tier 2 deferred (K0009). The primer says "A contract is a side-effect-free Go
expression". Nothing enforces it, so a contract can call a port, a store, or `time.Now()`. A
rule that the primer states but the compiler does not enforce distorts the comparison.

Fix: in a contract, allow a call only to `len` (and `cap`) and to type conversions. Report any
other call as K0009 ("a call in a contract, tier 2, is not supported"), or K0503. Reject function
literals and channel receives too.

### H5. Port contract wrappers fail to compile for valid ports

Location: `internal/lower/port.go`.

1. Named results are dropped (`port.go:303-314` keeps only types). Input:

   ```
   func Pay(amt money.Money) (r money.Receipt, e error)
       ensures e == nil => r.Amount == amt
   ```

   Observed: `component.vulcan:6:24: error K0008: undefined: e` and `6:38: undefined: r`.

2. Each clause restarts the `old()` counter (`port.go:351`, `port.go:373`). Input: two `ensures`
   clauses that each use `old(...)`. Observed: `component.vulcan:6:34: error K0008: no new
   variables on left side of :=`.

3. The wrapper's receiver is `w` (`port.go:400`). Input: `func Pay(w money.Money) (...)` with
   `requires w > 0`. Observed: `w redeclared in this block`, plus two more errors at
   `component.vulcan:6:187` and `10:7`, positions that do not exist in the source.

Why it matters: the primer says named results can be used in `ensures`. Two `old()` clauses on
one port method are a normal way to state that a stock level does not change on error. A task
that adds a port contract can then fail on valid input.

Fix: keep result names, and map each name to its `vulcan_rN`. Number `old()` per method, not per
clause. Name the receiver `vulcan_w`. Run K0511 on port contracts too.

## Medium

### M1. Field writes around the invariant: promoted fields, pointer methods on a field, closures

Location: `internal/check/contracts.go:72-131` (`fieldWrites`).

Input:

```go
type Counter struct{ N int }
func (c *Counter) Dec() { c.N-- }

type Account struct {
	Balance int
	C       Counter
}
	invariant Balance >= 0 && C.N >= 0

func (a *Account) Drainer() func() { return func() { a.Balance = -100 } }

type Holder struct{ Account }

var h Holder; h.Balance = -7      // promoted field write
var a Account; a.C.Dec()          // pointer method on a field
b := &Account{}; b.Drainer()()    // closure from a method
```

Observed: no diagnostic, and the program prints `-7 -1 -100`. `h.Account.Balance--` is caught
(K0507). The same promoted write across packages (`type Purse struct{ m.Wallet }`,
`pu.Cents = -9`) is also missed.

Why it matters: §7.2 forbids all three. The promoted write fails because `owner` checks
`sel.Recv()`, which is the outer type, not the type that declares the field. Embedding is common
in Go.

Fix: for a field selection, walk `sel.Index()` through the embedded fields and check each struct
on the path. Treat a call of a pointer-receiver method on a field (an implicit `&`) as K0508.
Treat a function literal inside a method as outside the method, unless the method calls it
directly or with `defer` (§7.2).

### M2. K0501 is reported for a valid call with a float argument

Location: `internal/check/contracts.go:305-312`, `338-341`. `ExactString()` turns `0.9` into
`9/10`, and the substituted text is then evaluated as untyped integer arithmetic.

Input:

```go
func Half(f float64) float64
    requires f / 2 > 0.4
{ return f / 2 }

Half(0.9)   // 0.45 > 0.4 at runtime
Half(1.0)   // 0.5 > 0.4 at runtime
```

Observed, for each call: `error K0501: this call breaks the contract of Half: requires f / 2 > 0.4
... is false for these arguments`.

Why it matters: a false compile error on a correct call. Prices, rates, and percentages are
floats in many services.

Fix: substitute a typed constant: `T(value)` with the parameter's type, written with
`constant.Value.String()` for floats, or evaluate with `go/constant` operations on the parameter
type. Skip a clause when a parameter's type is not a basic type.

### M3. Diagnostic positions inside contracts are wrong, and messages show generated names

Location: `internal/lower/lower.go:489` and `port.go:385`. The `/*line*/` directive gives the
start of the clause, but the text after it is `GoText`, not the source. `=>` becomes
`(!(...) || (...))`, and `result` becomes `vulcan_r0`, so each later column moves.

Input:

```go
func Get(x int) (n int, e error)
    ensures e == nil => n == "x"
```

Observed: `billing.vulcan:16:51: error K0008: invalid operation: vulcan_r0 == "x" (mismatched
types int and untyped string)`. The line is 33 characters long. `n` is at column 28.

The same happens for K0120: in P52's shape with `ensures err == nil => result.Amount <= Limit`,
the error is at column 66; `Limit` is at column 52.

Why it matters: the agent fixes code by position and by message. A column past the end of the
line and a name that does not exist in the source cost turns.

Fix: emit a `/*line*/` directive before each identifier and operand that comes from the source
(the lowering already has each node's offset in the clause), or report type errors in a contract
at the clause start only. Map `vulcan_rN` and `vulcan_oldN` back to `result`, `err`, the named
result, and `old(x)` in messages.

## Low

### L1. K0509 gaps

- Port contracts are not checked. `ensures len(old(xs)) >= 0` on a port with `xs []int` is
  accepted (`port.go:371-377`; `pkg.Olds` is set only in `lower.go:474-477`).
- `holdsReference` (`contracts.go:153-171`) takes the underlying type of a type parameter, which
  is its constraint interface. So `old(*x)` with `x *T`, `T ~int`, is a false K0509.

Fix: record port `old()` variables too. Treat a type parameter as holding a reference only when
some type in its type set does.

### L2. K0502 misses `result` under `*`, a slice expression, or a composite literal

Location: `lower.go:587-630`. `walk` has no case for `StarExpr`, `SliceExpr`, `CompositeLit`, or
`KeyValueExpr`.

Input: `func Find(k int) (*int, error)` with `ensures *result > 0`, returning `nil, err`.
Observed: accepted; at runtime a nil dereference panic, not a contract failure.

Fix: walk every expression kind (use `ast.Inspect` with the guard state).

### L3. K0506 misses typed constants and named basic types

Location: `contracts.go:193-203`. The zero is `int64(0)`, not `Money(0)`, so comparing it with a
typed constant fails to type-check and the clause is skipped.

Input: `const MinQty money.Money = 1`, `type Line struct{ Qty money.Money }`,
`invariant Qty >= MinQty`. Observed: no K0506. `Lo < Hi` and `Name != ""` are reported.

Fix: write the zero as a conversion to the field's own type (`types.TypeString` with the package
qualifier), or evaluate with `go/constant` on the field type.

### L4. K0501 misses constant calls through the receiver in a method of an invariant type

Location: `contracts.go:296-300` with `lower.go:562-565`. The call becomes `a.vulcan_Withdraw(-5)`
before type checking, and `vulcan_Withdraw` is not in the callee table.

Input: `func (a *Account) Fix() { a.Withdraw(-5) }` with `requires x > 0` on `Withdraw`. Observed:
no K0501. The same call from `Pay` (`a.Withdraw(-1)`) is reported.

Fix: map each `vulcan_M` twin to the requires of `M` in `requiresOf`.

### L5. A literal that ends at the closing brace breaks the lowering

Location: `lower.go:338-359` (`apply`). Two insertions at one offset keep the order of creation,
so the body's closing text goes before the literal's `)`.

Input: `func Open(x int) *Account requires x >= 0` with the body `{ return &Account{Balance: x}}`.
Observed: `billing.vulcan:8:35: error K0001: missing ',' in argument list` (a position on the
invariant line) and a false `system.vulcan:9:34: error K0107: Billing.NewPayer does not exist`.
`gofmt` output always has a space here, so this is rare.

Fix: order insertions at one offset so that a literal's closing `)` comes before the body's
closing text, or add the literal edits first.

### L6. The primer has an example that the compiler rejects, and misses two rules

Location: `docs/poc/guide.md:98-107`, `117`, `123`.

- The example `func Withdraw(a *Account, amt int)` with `type Account ... invariant Balance >= 0`
  is a plain function. Any body that changes `a.Balance` gets K0507 (verified). Make it a method
  `func (a *Account) Withdraw(amt int) (int, error)`.
- A composite literal in a contract must be in parentheses: `result == money.Receipt{Amount: amt}`
  gives `K0001: expected declaration, found ensures` on the next line; `(money.Receipt{...})`
  works. Say so, as Go does for `if`.
- K0120 (a port contract names no identifier of its own leaf) is not mentioned.
- Lines 117 and 123 state rules that H4 and H3 show are not enforced. Fix the compiler, not the
  text.

### L7. The interface steps of §5.4c are not checked for K0205

Location: `internal/check/network.go:61-89`. Only direct uses of a listed function count.

Input, in the non-gateway leaf Billing: `http.DefaultTransport.RoundTrip(req)`, and
`var rt http.RoundTripper = &http.Transport{}; rt.RoundTrip(req)`. Observed: no K0205.
`http.DefaultClient.Do`, `srv.ListenAndServe`, and `f := http.Get` are reported.

The listed deviation is about calls into other modules. These two steps are about the project's
own code, so the deviation does not cover them. Agents rarely write this shape, so the impact is
low. Fix: on a conversion to an interface, and on a read of a listed package-level interface
variable (`http.DefaultTransport`), apply §5.4c's two steps.

### L8. Smaller runtime and report gaps

- An invariant failure names only the type and the clause (`Caller` is the type name,
  `lower.go:742`). A failure in a startup function goes through `runStep`'s `recover`
  (`vulcan/rt/program.go`), which drops the stack. So the report does not say which method call
  or literal broke the invariant. Fix: pass the method name, or the literal's position, to
  `Vulcan_invariant`, and print the stack of a recovered `*ContractFailure`.
- `--on-contract-failure exit` (with a space) fails with `unknown flag --on-contract-failure`;
  only `--on-contract-failure=exit` works. Other flags accept both forms.
- In a port wrapper, `old()` runs before `requires` when an `ensures` clause comes first
  (`port.go:386-393`). The function lowering puts all `requires` first (`lower.go:497-499`). Do
  the same in the wrapper.

## The listed deviations

| Deviation | Behaves as stated | Acceptable for the PoC |
|---|---|---|
| One check point, after the body and its deferred calls | Yes (closure; P53 and a test with a deferred doubling of a named result pass) | Only without a deferred unlock. With one, it gives false failures and races: B1 |
| Go arithmetic in contracts | Yes | Yes. Money is `int64` cents; overflow is unlikely in the tasks |
| No proof rule (no K0504; K0502 is syntactic) | Yes. A contract that dereferences nil panics as Go does | Yes. The primer's advice to guard on the left is correct |
| No unwinding check (§7.3 rule 5) | Not tested by panic; code has none | Yes |
| No check after a call that receives the struct | Yes (code has none) | Only if no type with an invariant is decoded from JSON in Web. If the plan puts an invariant on a request type, `json.Unmarshal` goes around it silently |
| K0205 and K0203 do not follow calls into other modules | Yes | Yes. The structural detectors use the same direct view. The interface steps are a separate gap: L7 |

## What is done well

- The lowering keeps the meaning of the bodies tested: deferred `recover` that sets named
  results, `goto` and labels, generic functions and methods, named results shadowed in an inner
  block, value receivers, and unnamed receivers all compile and give Go's results.
- Literals of a library type with an invariant are checked in another package, also with an
  import alias, and K0507 and K0508 work across packages.
- Errors in a method body that has an unchecked twin are reported once, not twice.
- `on_contract_failure: exit`, and `--on-contract-failure=exit|panic`, exit with code 3 and skip
  deferred calls as §7.4 says. `vulcan test` keeps `panic`.
- `vulcan.Bind` binds a fake through the contract wrapper, and a nested `Bind` does not wrap
  twice.
- K0513, K0514, K0510, K0511, K0512 behave as specified in the cases tried.
- The network and acquiring lists match Appendix D.
- `gen_cases.py` regenerates all 108 archives with no difference, and the suite passes.

## Resolution (2026-10-27)

Each fix has a conformance case; the suite has 129 cases, and all pass.

| Finding | Status | Change | Cases |
|---|---|---|---|
| B1 | Fixed | `ensures` and the invariant are checked at each `return` (typed temporaries, then the checks, then `return`), at a bare `return` on the named results, and at a reachable end of the body, before the deferred calls run. The closure is gone. The second check point of §7.3 is not built; `docs/status.md` says so | B1 (8 goroutines, 160,000 locked calls) |
| H1 | Fixed | The test stubs import only what the port signatures use | H1 |
| H2 | Fixed | The compiler type-checks the code once without contracts, and the lowering finds the receiver by its object; a `range`, an assignment, or `&` of the receiver turns the rewrite off | H2, H2b |
| H3 | Fixed | Literals are found by type: explicit, elided (`[]T{{...}}`, `[]*T{{...}}`, map values), conversions `T(x)`, and aliases | H3a to H3e |
| H4 | Fixed | K0503: a contract and an `old()` argument call only `len`, `cap`, and conversions, and hold no function literal and no receive | H4, near miss H4b |
| H5 | Fixed | Port wrappers keep named results, number `old()` per method, and name the receiver `vulcan_w`; K0511 runs on port contracts | H5 |
| M1 | Fixed | K0507 and K0508 follow embedded fields, treat a pointer method called on a field as K0508, and treat a function literal that a method does not call as outside the method | M1a, M1b, M1c |
| M2 | Fixed | K0501 substitutes typed constants (`float64(0.9)`) | M2 |
| M3 | Fixed | A diagnostic inside a contract is reported at the clause start, and generated names are written as source names | M3, P52 |
| L1 | Fixed | Port `old()` variables are checked for K0509; a type parameter is not a reference | |
| L2 | Fixed | K0502 walks every kind of expression | L2 |
| L3 | Fixed | The zero value is a conversion to the field's own type | L3 |
| L4 | Fixed | A method's twin has the method's `requires` for K0501 | L4 |
| L5 | Fixed | Edits at one offset are ordered: outer openings first, inner closings first | L5 |
| L6 | Fixed | The primer's `Withdraw` is a method; it says that a composite literal in a contract needs parentheses, and states K0120 and K0503 | |
| L7 | Recorded | Not fixed; `docs/status.md` lists it | |
| L8 | Fixed | An invariant failure names the method call or the position of the value that broke it; a panic in setup or startup prints its stack; `--on-contract-failure exit` works with a space; a port wrapper runs `requires` before `old()` | C12b, C18 |
