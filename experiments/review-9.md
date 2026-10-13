# Review 9: spec.md and guide.md as an implementer's only source

Scope: `docs/spec.md` (Draft 0) and `docs/guide.md`, read in full. Nothing else under the repository
was read. The §13 open questions, style, and new features are out of scope.

Go claims were checked with Go 1.27.2 (module `go 1.24`) in
`/private/tmp/claude-501/-Users-jmachen-code-outrider-kopos/5545766a-ddc7-4e78-bac6-61990c472cca/scratchpad/review9/`:

- `c/`: the Appendix C API compiles as written. Generic aliases and inference for `Published`,
  `Deliver`, and `Bind` work. The Appendix C claim about `Bind` taking `any` is correct: a typed
  `impl P` fails with "type CheckoutOK ... does not match inferred type keel_Checkout for P".
- `f/a`, `f/b`: the Appendix F `ensures` lowering fails to compile (R9-11). `f/c` is the proposed fix
  and compiles.
- `f/leaf`: the Appendix F invariant lowering fails to compile across packages (R9-12).
- `f/inv`: the Appendix F invariant wrapper races with the method's own lock under `-race` (R9-09).

Severity: **blocker** = an implementer must invent semantics; **major** = wrong, unsound, or
uncompilable as written; **minor** = small gap or inconsistency.

| ID | Severity | Summary |
|---|---|---|
| R9-01 | blocker | The `error` result of a remote method has no wire representation |
| R9-02 | blocker | The proto3 mapping does not cover the serializable types it admits |
| R9-03 | blocker | The transport interface cannot route a message, and the dead-letter hop is undefined |
| R9-04 | major | Setting names: lower-case wire names vs exported Go fields; `dropOldest` cannot be exported |
| R9-05 | major | Contract keywords after a signature are ambiguous with Go's `Result` |
| R9-06 | major | The proof rule is unsound for field paths, and undefined at call sites and returns |
| R9-07 | major | "Exact" contract arithmetic is undefined for floats, `/`, `%`, `^`, shifts, mixed types |
| R9-08 | major | Invariants can be bypassed in several ways that the rules do not cover |
| R9-09 | major | Invariant and `ensures` checks run outside the method's lock: races and false failures |
| R9-10 | major | The `pure` "created value" rule is unsound under aliasing |
| R9-11 | major | The `ensures` lowering does not compile for common Go functions |
| R9-12 | major | Appendix F calls unexported names across packages |
| R9-13 | major | The network rule is unsound through interface conversions |
| R9-14 | major | `syscall` "every function" plus transitive reach rejects ordinary libraries, with no override |
| R9-15 | major | Rule 9 ("topic and remote types come from a library") contradicts the spec's own examples |
| R9-16 | major | A subscriber-only leaf has no place to initialize before its handlers run |
| R9-17 | major | The schema lock does not record what decides compatibility |
| R9-18 | minor | The implicit `requires r != nil` changes valid Go nil-receiver methods |
| R9-19 | minor | Smaller grammar gaps in Appendix B |
| R9-20 | minor | Diagnostic codes: missing, narrower than the rule, or miscited |
| R9-21 | minor | Startup-time state: the `init` ban is incomplete, and "set once" is undefined |
| R9-22 | minor | Leaf tests: unbound ports, no library tests, `Published` with parallel tests |
| R9-23 | minor | Startup and shutdown gaps |
| R9-24 | minor | Leaf names are not required to be unique across containers |
| R9-25 | minor | Flag name collisions, and per-container required flags |
| R9-26 | minor | §6 warning sources and the fixed race-detector policy |
| R9-27 | minor | Near-duplicate detection: "fingerprints", cost, and mode assignment |
| R9-28 | minor | The primer omits rules that reject ordinary Go |
| R9-29 | minor | Contract scope gaps: invariant field names, named results, generic invariants, "optional" |

---

## R9-01 (blocker): the `error` result of a remote method has no wire representation

**Location.** §5.5.3 (transport errors, "Types must be serializable"), §5.5 serializable types,
§5.5.2 wire format, Appendix C `Transport.Call`.

**Problem.** Every remote method must return `error` last (K0315). `error` is an interface, and
§5.5 makes an interface in a remote type a compile error (K0305). Only `context.Context` has an
exemption. §5.5.2 maps structs, slices, maps, `time.Time`, and `time.Duration` to proto3, but not
`error`. `Call` returns `([]byte, error)`, and the spec does not say if the callee's error travels in
the bytes or in the Go error. Nothing says what the caller receives for a callee's `ErrNotFound`:
the same sentinel, so that `errors.Is` matches; a new error with the same text; or a wrapped error.
Nothing says if the callee's `ensures` (checked on the callee and again on the caller) sees the
same `err == nil` value on both sides.

**Why it matters.** Every remote port in every multi-container system depends on this. The caller's
error handling (`errors.Is(err, stock.ErrUnknownSKU)`) works in-process and breaks silently when the
port becomes remote, unless the spec defines it.

**Fix.** Add to §5.5.3: "The error result crosses as its `Error()` text and an optional code. On the
caller, a non-nil error is a `*keel.RemoteError{Text, Code}`. A library can declare a sentinel error
with `keel.Code("stock.unknown-sku")`; such an error crosses as its code, and `errors.Is` matches it
on the caller." Exempt `error` from K0305 in the last result position only. Add the encoding to
§5.5.2 and the type to Appendix C.

## R9-02 (blocker): the proto3 mapping does not cover the serializable types it admits

**Location.** §5.5 (serializable types), §5.5.2 ("The compiler owns the wire format").

**Problem.** §5.5 admits "arrays, slices, maps with keys of these scalar types, and structs of
these", nested freely. The §5.5.2 mapping ("a struct is a message, a slice is a repeated field, a
map is a map field") cannot encode several of them in proto3:

1. `[][]T`, `[]map[K]V`, `map[K][]V`, `map[K]map[K2]V`: proto3 has no repeated-of-repeated, and a
   map value cannot be a map or a repeated field. Wrapper messages are needed, and their shape is
   not defined.
2. Map keys of type `time.Time`, `time.Duration` (as a message), `float32`, or `float64`: proto3 map
   keys must be integral or string.
3. Integer encodings: `int`, `int8`, `int16`, `uint8`, ... have no proto3 type. The choice between
   `int64` (varint) and `sint64` (zigzag) changes the bytes. Go `int` is 32 bits on some targets.
4. A message type that is not a struct (`topic[events.OrderID]` with `type OrderID string`), arrays
   (`[4]T`: repeated, or fixed?), `[]byte` (bytes, or repeated uint32?), embedded fields (flattened,
   or a nested message?), unexported fields (do they cross?), recursive types (a `[]Node` inside
   `Node`, which also makes the §5.5.1 "maximum message size" infinite).
5. `string` maps to proto3 `string`, which must be valid UTF-8; Go strings need not be. `time.Time`
   as `Timestamp` drops the `Location` and the monotonic reading, so a received value differs from
   the sent one.

**Why it matters.** §5.5.2 promises "Any deploy order works" across builds. Builds made by two
compiler versions interoperate only if the mapping is fixed bit for bit. Today each implementation
must invent it.

**Fix.** Add a normative table to §5.5.2 that gives, for each admitted Go type, its proto3 type:
signed integers as `sint64` (or `int64`, chosen once), unsigned as `uint64`, `[]byte` as `bytes`,
`string` as `bytes` (or require UTF-8 and check it at publish), arrays as repeated with a length
check, and each nested repeated or map value as a generated wrapper message with field 1. Restrict
map keys to integers, `bool`, and `string` (new K0305 case). Require a topic message type to be a
struct. Define embedded fields as nested messages. State that unexported fields do not cross, and
that recursive message types are a compile error for transports that require limits. State that
`time.Time` arrives in UTC without a monotonic reading.

## R9-03 (blocker): the transport interface cannot route a message, and the dead-letter hop is undefined

**Location.** §5.5.1 (dead letters, process settings), Appendix C (`transport.Wire`, `Dead`,
`Guarantees`), §5.3 system example.

**Problem.**

1. `transport.Wire` holds `Name` (the wire text), `Container` ("the container at this end"), and
   `Settings`. A `direct` publisher must send to the subscriber's container and look it up in
   `--transport.direct.peers`; a caller must find the callee's container. Neither is in `Wire`
   except by parsing `Name`, which is source text.
2. The dead-letter hop has no defined path. The `direct` queue is "in the publisher's process", but
   `dead` is passed to `Subscribe`, which runs in the subscriber's process. Then "the runtime ...
   builds the `DeadLetter[T]` and publishes it on the wire's route". The route is not a wire in
   `system.keel`, so there is no `Wire` value, no transport, and no settings for that `Publish`. The
   example `Analytics/Ingest.Placed: Api/Orders.Placed via direct{ deadletters: Worker/Alerts.DeadPlaced }`
   needs a hop from Analytics (or Api) to Worker; `Api/Orders.Shipped: Worker/Picking.Shipped ...
   deadletters: Worker/Alerts.DeadShipped` needs an in-process hop inside Worker, which rule 3
   otherwise forbids for topics. "Retried with the source wire's settings" gives a hop with a
   `deadletters` setting that points to itself. A `Settings` struct may also have several
   `transport.Route` fields; `Dead` does not say which.
3. Whether a dead-letter handler must be `idempotent` depends on the hop's transport, which is not
   named. A `DeadLetter.Error` longer than 4,096 bytes: truncated, or a failed check?
4. `Guarantees` has `Redelivers` and `Order` only. §4, §5.5.1, and K0304 say "the transport requires"
   bounded messages, and §5.5.3 says `direct` "supports request and response"; neither can be
   declared, so the compiler must hard-code `direct`. §5.5.1 calls the guarantees "constants that
   the compiler reads", but Appendix C makes them a `var` of struct type.
5. The built-in `direct`: is a `transport direct { go "..." }` declaration needed, and what is its
   import path?

**Why it matters.** No one can write `direct`, or a second transport, without inventing routing and
the dead-letter path.

**Fix.** Give `transport.Wire` structured ends:
`From, To transport.Endpoint` with `Endpoint{Container, Leaf, Name string}`, plus `Kind` (topic or
call). Define the dead-letter hop: "The runtime in the process that holds the failed message calls
the transport of the source wire with a synthetic `Wire` whose `To` is the route and whose
`Settings` are the source wire's settings with every `transport.Route` field cleared. A failure of
that hop is counted and reported." Name the route field used by `Dead`
(`Dead(ctx, route transport.Route, ...)`), and state that `Error` is truncated to 4,096 bytes. Add
`RequiresLimits bool` and `Calls bool` to `Guarantees`, require a composite literal of constants,
and change "constants" in §5.5.1 to "a constant composite literal". State that `direct` needs no
declaration and give its path.

## R9-04 (major): setting names do not map to Go fields

**Location.** §5.5.1 (`direct` settings), Appendix B.2 ("Scope of a value"), Appendix C
(`Settings` "with keel tags"), §5.3 leaf settings.

**Problem.** Leaf settings use Go field names (`APIKey`, `Addr`). Transport settings use lower-case
names (`capacity`, `whenFull`, `deadletters`) and lower-case process flags (`--transport.direct.listen`).
The generated `main` sets the transport's `Settings` from another package, so its fields must be
exported (`Capacity`). The mapping from `capacity` to `Capacity` is not stated (case fold? a tag?).
B.2 says a value can use "the exported constants of the setting's own type, without qualification
(`whenFull: dropOldest`)", but `dropOldest` starts with a lower-case letter and cannot be an exported
Go constant. The combination of tags `required` and `topic` on `deadletters` is not defined (is
`deadletters` required on the call wire `via direct{}`?). `maxWait`, `backoff`, `maxBackoff`, and
`trace` are in neither the topic list of §5.5.3 nor a call list.

**Why it matters.** Every transport and every wire needs this mapping. Two implementations will
choose differently.

**Fix.** Add to §5.5.1: "A setting name is the field's `keel:"name=..."` tag, or else the field name
with its first letter in lower case. An enum value is written as the constant's name with its first
letter in lower case (`dropOldest` for `DropOldest`). A process flag is `--transport.<name>.<setting>`
with the same rule." State: "`required` with `topic` means required on topic wires only." List each
`direct` setting as topic, call, or both.

## R9-05 (major): contract keywords after a signature are ambiguous with Go's `Result`

**Location.** Appendix B.3 (`FunctionDecl`, `MethodDecl`), B.2 `PortMethod`, B.1 rule 1, §4.1.

**Problem.** Go's `Signature = Parameters [ Result ]` and `Result = Parameters | Type`. For a
function with no results, `func (a *Account) Deposit(x int) requires x > 0 {` parses `requires` as a
result type name. The same holds for `ensures` and for `timeout` after a port method with no
parenthesized result. The productions give no rule to prefer the contract. §4.1 says these words are
keywords "after a signature", and B.1 says "no Go program changes its meaning", but
`type requires int; func F() requires { return 0 }` is valid Go and becomes ambiguous.

**Why it matters.** Every function without results that has a contract hits this. A parser that
follows Go's productions rejects the spec's own style (§7.2 `Transfer`-like methods with contracts).

**Fix.** Add to B.3: "After `Parameters`, an identifier `requires`, `ensures`, or `timeout` that is
followed by a token other than `{`, `;`, `)`, or `,` starts a contract and is not a `Result`." Keep
B.1's "no Go program changes its meaning" only if that rule is added, and give the one-line example.

## R9-06 (major): the proof rule is unsound for field paths, and undefined at call sites and returns

**Location.** §7.1 "The proof rule", §7.2 "How the compiler checks contracts", K0501, K0504.

**Problem.**

1. "Each conjunct of the function's `requires` whose variables the body never assigns" is a fact in
   `ensures`. A field write is not an assignment of the variable. With
   `requires s != nil && len(s.Items) > 0` and a body `s.Items = nil`, the rule keeps
   `len(s.Items) > 0` as a fact. Then `ensures s.Items[0] == x` passes K0504, and at runtime the
   contract indexes an empty slice: the "contract expression itself cannot fail" promise breaks. The
   same false fact can let the compiler "remove a check that it proves". A call that receives `s`
   can also change `s.Items`.
2. K0501 says the compiler proves a contract false at "a call, a literal, or a return", and §7.2 says
   it removes a check "at a call site where it proves the contract". The proof rule lists facts only
   from inside contracts. It does not say which facts exist at a call site (the argument expressions
   only? constants only? enclosing `if` conditions?) or at a return. Because K0501 is an error, two
   compilers that choose differently accept different programs, against "each compiler accepts the
   same contracts".
3. The atoms of the decided theory are not defined: field selectors (`result.Amount`), `old(...)`,
   named non-`int` integer types, unsigned types, and the range of a variable (its Go type's range,
   or unbounded).

**Fix.** (1) Change the rule to: "a conjunct of `requires` is a fact in `ensures` only if every
variable, field path, and dereference in it is not written in the body, not passed by pointer or
reference to a call, and not reachable from a parameter that the body passes to a call; otherwise
the conjunct is a fact only inside `old(...)`." (2) Define call-site facts as "the callee's
`requires` with each parameter replaced by the argument expression, where an argument that is not a
constant is an unknown atom; no facts from the caller's body". The same rule at a return uses the
returned expressions. (3) List the atoms and state that each variable ranges over its Go type.

## R9-07 (major): "exact" contract arithmetic is undefined at its edges

**Location.** §7.1 ("Arithmetic in a contract expression is exact"), Appendix F item 5.

**Problem.** §7.1 says all contract arithmetic is exact. Appendix F item 5 speaks of integers only.
Undefined:

1. Floats. If exact, `ensures result == a + b` for `float64` fails for most inputs, because the body
   rounds and the contract does not.
2. Integer `/` and `%`: the "mathematical result" of `-7 / 2` is −3.5. Go truncates.
3. `^x` on unsigned types: Go `^uint8(0)` is 255; the exact (big-integer) complement is −1. Also
   `&^`, and `&`, `|` on negative values.
4. Shifts: a negative shift count panics in Go and is not in K0504's list of things to prove safe.
   Slicing `xs[i:j]` is also not listed.
5. Mixed types: Go rejects `x > len(s)` for `x int64`. Does "a comparison uses the mathematical
   value" relax the type rules?
6. An exact value passed to a `pure` function parameter (`IsSmall(x + 1)` where `x` is `math.MaxInt64`):
   an error, a wrap, or a contract failure?

**Fix.** State: "Exactness applies to integer `+`, `-`, `*`, unary `-`, conversions between integer
types, and comparisons. `/` and `%` truncate toward zero, as in Go. Bitwise operators and shifts use
Go semantics on the operand's type, and a shift count must be proved non-negative (K0504). Float
arithmetic is Go arithmetic. Operand types follow Go's rules. A value outside the parameter type of a
`pure` call is a contract failure." Add slicing and shifts to K0504.

## R9-08 (major): invariants can be bypassed in ways the rules do not cover

**Location.** §7.2 bullets "Only methods write fields", "Calls that receive the struct are checked",
"Outer calls only"; Appendix F "Struct invariants".

**Problem.** §7.2 opens: "An invariant protects a struct only if no code can go around it." These
paths go around it with no compile error, no runtime check, and no audit line:

1. A container of `T` passed to a function: `json.Unmarshal(data, &accounts)` with
   `accounts []Account`, `&arr` with `[3]Account`, `&outer` where `Outer` has an `Account` field, or
   a `map[string]*Account`. The check runs only after a call that receives a `*T`.
2. Inside a method: `func (a *Account) Load(b []byte) error { return json.Unmarshal(b, a) }`. The
   method writes no field and calls no method that does, so "skips the check"; and "Calls that
   receive *T" applies only "outside the methods of T".
3. A closure created in a method and called later from outside:
   `func (a *Account) Setter() func(int) { return func(v int) { a.Balance = v } }`. The write is
   textually inside a method, so K0507 allows it.
4. A goroutine started in a method (`go a.drain()`, or `go a.Withdraw(x)`, which is a call through
   the receiver and so not checked) that writes after the outer method returned.

**Fix.** (1) Run the post-call check for every argument whose type contains `T` by value or by
pointer (`*[]T`, `*[N]T`, `*S` with a `T` field, maps of `T` or `*T`), and list the containers with
reference elements in `keel audit`, as for slice fields. (2) Make the "writes no field" skip also
require that the method passes no `*T` or the receiver to a non-method call. (3) Treat a function
literal in a method as outside the method for K0507, unless it is called directly in the method.
(4) Treat a `go` statement in a method as outside the method for the outer-call rule. If any of
these is not wanted, change the opening sentence of §7.2 to state the limits.

## R9-09 (major): invariant and `ensures` checks run outside the method's lock

**Location.** Appendix F "Outer calls" (wrapper `M` calls `keel_M`, then checks) and "Contracts on
functions" item 3 (`ensures` checked "after the function's own deferred calls").

**Problem.** A Go type with a mutex releases it in a `defer` or before return. The wrapper checks
the invariant after `keel_M` returns, so the lock is released. A second goroutine can be inside
`keel_M` with a temporarily broken state. The check then reads fields without the lock: it is a data
race, and it can panic on a correct program. `ensures` has the same problem, because it runs after
`defer mu.Unlock()`. Verified: `f/inv/inv_test.go` (an `Account` with `Reserved <= Balance` and a
`Move` that breaks and restores it under its lock) reports `WARNING: DATA RACE` in
`(*Account).keel_invariant` under `go test -race`. §6 runs every test with the race detector, so
these generated races fail the user's tests.

**Fix.** Check the invariant and `ensures` at each `return` before the function's deferred calls run
(the lowering in R9-11 can set results and check before `return`). This covers the
`defer mu.Unlock()` pattern. Add to §7.2: "A method that releases a lock before it returns gets its
check after the release; the compiler warns when a struct with an invariant has a `sync.Mutex` or
`sync.RWMutex` field and a method that unlocks other than by `defer`."

## R9-10 (major): the `pure` "created value" rule is unsound under aliasing

**Location.** §7.1 "What a `pure` function can do" (K0505).

**Problem.** A `pure` function "can write through an index, a field, or a pointer only into a value
that it created (with a literal, `make`, `new`, or `append` to such a value)". The written object and
the created root are different things:

- `c := []*Account{param}; c[0].Balance = 0`: the root `c` is created; the write goes into `*param`.
- `s := S{P: param}; s.P.Balance = 0`; `m := map[string]*T{"a": param}; m["a"].X = 1`.
- `ys := param[:0]; ys = append(ys, 1)`: is `param[:0]` "created"? `append` writes into the
  caller's backing array.
- `ys := id(param); ys[0] = 1`, where `id` is `pure` and returns its argument.

Each passes a plain reading of the rule and changes the caller's data while a contract runs. That
breaks "checking a contract never changes what a correct program does".

**Fix.** Define it by type and provenance, decidable from source: "A `pure` function can write only
through a path whose root is a local variable initialized by a literal, `make`, `new`, or `append`
to such a variable, and every step of the path is an index or field of a value that the same root
holds by value. It cannot write through a pointer, slice, or map that it loaded from such a value, or
that a slice expression or a call returned."

## R9-11 (major): the `ensures` lowering does not compile for common Go functions

**Location.** Appendix F "Contracts on functions" item 3; §B.1 reserved names.

**Problem.** Each `return e0, e1` becomes `r0, r1 = e0, e1; keel_returned = true; return`.

1. A bare `return` is illegal where a named result is shadowed. User code
   `func F(x int) (n int, e error) ensures n >= 0 { if x > 0 { n := x; return n, nil }; return 0, nil }`
   is valid Go. Its lowering fails: `result parameter n not in scope at return` (`f/a/a.go`).
2. The compiler "gives the results names". The example names are `r0, r1`, not `keel_`-prefixed. A
   body that declares `r0` at its top level fails: `no new variables on left side of :=`
   (`f/b/b.go`).
3. Same root cause: the handler lowering adds a parameter `ctx` to a handler declared without one
   (`func keel_handle_Shipped(ctx context.Context, ...)`). A package-level `ctx` that the body uses
   is then shadowed, and the body's meaning changes (§2 "function bodies are Go, unchanged").

**Fix.** Lower each return to a block with typed temporaries and an explicit return:
`{ var keel_t0 T0; var keel_t1 T1; keel_t0, keel_t1 = e0, e1; keel_returned = true; return keel_t0, keel_t1 }`
(`f/c/c.go` compiles). Name added results `keel_r0, ...`, and name an added handler parameter
`keel_ctx`. State in Appendix F that every generated identifier starts with `keel_`.

## R9-12 (major): Appendix F calls unexported names across packages

**Location.** Appendix F "Struct invariants", "Ports"; §5.3 wires.

**Problem.**

1. `func (v *T) keel_invariant()` and `keel_checkT(...)` are unexported. Invariants are most useful on
   library types that leaves create and pass (`json.Unmarshal(data, &acct)` in a leaf, `events.X{...}`
   literals in a leaf). The check is emitted in the leaf's package and cannot call them:
   `cannot refer to unexported method keel_invariant`, `undefined: lib.keel_checkAccount`
   (`f/leaf/leaf.go`). The topic validator is already exported (`Keel_validateOrderPlaced`), so the
   pattern is known but not applied.
2. The port wrapper in `main` has a field `impl *billing.Payer`. A wire may name "an exported
   constructor"; its result can be an unexported type (`*payer`) or an interface, which `main`
   cannot name. With an interface, the call is not "a static call", and §5.3's "one dynamic
   dispatch" is false.
3. A port contract in `component.keel` may call a `pure` function. If that function is in the
   consumer leaf and unexported, the wrapper in `main` cannot call it. The scope of names in
   `component.keel` is not stated.

**Fix.** (1) Generate `Keel_invariant` and `Keel_checkT` (exported, reserved by B.1). (2) Require a
constructor's first result to be an exported concrete type (new K0604 case), or generate the wrapper
in the provider's package. (3) State that a port contract can name only predeclared, standard
library, and library identifiers (`pure` functions included), never the leaf's own.

## R9-13 (major): the network rule is unsound through interface conversions

**Location.** §5.4c (reachability, rule 1), §5.4b rule 3 ("the diagram cannot be wrong"), §5.4b
"The compiler rejects network access in any other leaf ... so the reviewer does not have to find it".

**Problem.** The analysis "does not follow a call through an interface value or a function value,
because the code that made the value chose its target". Making a value is not "reaching" a function,
so no one is charged:

- `var d interface{ DialContext(context.Context, string, string) (net.Conn, error) } = &net.Dialer{}; d.DialContext(ctx, "tcp", addr)`
  in any leaf reaches no listed function.
- `http.DefaultTransport.RoundTrip(req)`: `DefaultTransport` is declared
  `var DefaultTransport RoundTripper = &Transport{...}` (checked with `go doc`). The standard library
  made the value, so no leaf is charged, and `(*Transport).RoundTrip` is not in Appendix D.

Neither is in the §5.4c "Limits" list, so `keel audit` does not show it. The section claims the rule
covers SDKs; any SDK that holds its client as an interface escapes.

**Fix.** Add to §5.4c: "Converting a value of type `T` to an interface type reaches each method of
`T` in the interface's method set. Reading a package-level variable of interface type reaches each
method of the dynamic type of its initializer." Add `(*http.Transport).RoundTrip` to Appendix D.
Change §5.4b rule 3 to "the compiler generates the `uses` relationships from the declarations; it
does not check which target a gateway reaches".

## R9-14 (major): `syscall` "every function" plus transitive reach rejects ordinary libraries

**Location.** Appendix D (acquiring functions: `syscall` every function), §5.4c rule 2, §5.4a rule 2,
§2 "Mechanism, not policy".

**Problem.** A library may not reach an acquiring function, reach is transitive through static
calls, and every `syscall` function is acquiring. In the standard library (checked in GOROOT):
`os.Getenv` calls `syscall.Getenv` (`os/env.go`), `os.Exit` calls `syscall.Exit` (`os/proc.go`, so
also `log.Fatal`), and `time.LoadLocation` calls `syscall.Open` (`time/sys_unix.go`). A library that
calls any of them is a compile error (K0203). `syscall.Getenv` and `syscall.Exit` do not "choose an
I/O target by name", so they do not meet Appendix D's own definition of "acquire". `config` can only
add to the lists, never remove from them, so the project cannot correct this. That breaks §2: "A
default that the language sets must be one that a setting can change." Related contradiction: §5.4a
says a library can "send through a resource that its caller gives it", but `(*http.Client).Do` on a
caller's client is a network function (K0205).

**Fix.** Replace "every function" for `syscall` with the functions that meet the definition
(`Open`, `Openat`, `Socket`, `Connect`, `Bind`, `Exec`, `ForkExec`, ...), and keep the audit line
for every `syscall` use. Add `config { allow_network: [...], allow_acquiring: [...] }` to remove
entries, listed by `keel audit`. Define the format of a method name in the lists (`(*net/http.Client).Do`).
Change the §5.4a sentence to "read, write, or query through a handle", and name `(*http.Client).Do`
as the exception.

## R9-15 (major): rule 9 contradicts the spec's own examples

**Location.** §5.4 rule 9, §5.5.3 example, §5.5.1 `DeadLetter`, §5.4b example, §5.5 read-model
example, Appendix F "Topics", guide §2.

**Problem.** Rule 9: "A topic message type and a remote type must come from a library". The guide
repeats it. But:

- `needs remote port Stock { func Level(sku stock.SKU) (int, error) ... }`: `int` and `error` are not
  from a library.
- `subscribes DeadPlaced topic[keel.DeadLetter[events.OrderPlaced]]`: `keel.DeadLetter` is from the
  runtime package. Appendix F generates "its validation function ... in that library", which does
  not exist for `DeadLetter[T]`.
- §5.4b `needs port CardNetwork { func Charge(...) (CardReceipt, error) }`: an unqualified, so
  leaf-private, type in a port signature (K0112).
- §5.5 `Stock.Publish(StockChanged{...})`: unqualified message type.

**Fix.** Reword rule 9: "A named type in a topic message or a remote signature must be declared in a
library, or be `keel.DeadLetter[T]` for a library type `T`; predeclared types, `error`, and
`time.Time`/`time.Duration` are allowed." State that the validator for `DeadLetter[T]` is generic in
`rt`. Qualify the example types (`billing.CardReceipt` becomes `cards.Receipt` from a library;
`events.StockChanged`).

## R9-16 (major): a subscriber-only leaf has no place to initialize before its handlers run

**Location.** §6 startup steps 2 to 4, §5.4 rule 12.

**Problem.** Setup goes "in a constructor or in `startup`", and `init` is banned. A leaf that only
subscribes (Picking, Alerts) is constructed by no wire, so it has no constructor. Step 3 starts its
subscriptions, and step 4 runs its `startup` after that. So its handlers can run before its
`startup` has opened its database or stored its settings. The spec states the guarantee only for
entry leaves ("never serves a request before the ports it calls are bound").

**Why it matters.** Every subscriber that holds a resource races its own setup. A durable `direct`
queue that holds messages from before a restart delivers them at once.

**Fix.** Split the hook: "A leaf can define `func setup(ctx context.Context[, s Settings]) error`.
`main` runs every `setup` after step 2 and before step 3. `startup` keeps its meaning (long-running
work after serving starts)." Or start each leaf's subscriptions only after its `startup` returns or
calls `keel.Ready(ctx)`. Add the hook to K0601.

## R9-17 (major): the schema lock does not record what decides compatibility

**Location.** §5.5.2 (lock contents, change table, "Any deploy order works"), §5.5.3.

**Problem.**

1. Invariants and remote contracts are locked "in the compiler's canonical form". A tier-2 invariant
   `invariant Valid(o)` keeps the same text when the body of the `pure` function `Valid` changes. That
   is a breaking change ("Change an invariant ... Breaking") that the lock cannot see.
2. "Rename a field | Breaking. ... a rename looks like a remove and an add." Remove and add are each
   "Compatible", so the compiler sees a compatible change and cannot raise K0311. The row promises an
   error that cannot be detected.
3. Adding or removing a method of a remote port, and adding or removing a `context.Context` first
   parameter, are not in the table.
4. The example names the remote port `port Worker/Inventory.Stock`. Inventory supplies `NewLevels`;
   the port `Stock` is declared by `Api/Orders`.

**Fix.** (1) Lock the transitive closure of `pure` functions that an invariant or a remote contract
calls, by a hash of their canonical bodies, and treat a hash change as a breaking change.
(2) Change the rename row to "Compatible to the compiler; the old field's data is lost. Use a new
type to keep it." Or detect a remove and an add of the same type in one change and warn. (3) Add
rows: adding a method is compatible only after every provider has deployed (state the order), and
removing one is compatible. (4) Key a remote port by its consumer, `port Api/Orders.Stock`.

## R9-18 (minor): the implicit `requires r != nil` changes valid Go nil-receiver methods

**Location.** §7.1 "The proof rule" (implicit `requires r != nil`).

**Problem.** "A method with a pointer receiver, on a type with an invariant or with a contract of its
own" gets `requires r != nil`. Go code often handles a nil receiver
(`func (l *List) Len() int { if l == nil { return 0 }; ... }`). Adding an invariant to `List` makes
that valid call panic. This changes what a body means (§2, §6). The phrase "with a contract of its
own" is also ambiguous: the type's, or the method's?

**Fix.** Apply the implicit `requires` only where a contract of that method, or the invariant check,
dereferences the receiver. Skip the invariant check when the receiver is nil. Reword to "a method
that has its own contract, or a method of a type with an invariant".

## R9-19 (minor): smaller grammar gaps in Appendix B

**Location.** Appendix B.1, B.2, B.3; §4.1.

1. Two `invariant` clauses on separate lines do not parse. B.1 place 3 covers the first `invariant`
   only, and place 2 lists `requires`, `ensures`, `timeout`, `{` but not `invariant`. Add `invariant`
   to place 2.
2. A composite literal in a contract before the body brace (`requires p != T{} {`) has Go's `if`
   ambiguity. State that it needs parentheses, as in Go.
3. `ContractExpr` is used but not defined. B says undefined productions are Go's; it is not. Add
   `ContractExpr = Expression .` with the `=>` and name rules.
4. `config { transport: direct{ capacity: flag("cap", max: 10) } }`: in `config`, `Value` is a Go
   `Expression`, so `flag(..., max: ...)` inside the literal is not valid Go syntax. Parse nested
   settings with `Settings`, not `Expression`.
5. `max` (in `FlagOption`) is a contextual keyword but is not in the §4.1 list.
6. B.3 allows `invariant` inside a `type ( )` group on one line; K0513 forbids it; §7.2 does not say
   so. Put the restriction in B.3.

## R9-20 (minor): diagnostic codes are missing, narrower than the rule, or miscited

**Location.** Appendix E against the body.

- No code: a topic message or remote type not from a library (rule 9; K0112 covers port signatures
  only); a `call` setting on a topic wire (K0308 covers the reverse); a required setting given in
  `config` (§5.5.1); a hand-edited or missing `topics.lock`; Go syntax and type errors in bodies (§9
  says each diagnostic has a stable code).
- Narrower than the rule: K0602 says "two types"; §5.3 also forbids different defaults or maxima.
  K0305 omits "an interface", which §5.5 lists.
- No body rule: K0318 (`timeout` on a non-remote port). §5.5.3 only says a method "can override" the
  default.
- Miscited: K0509 cites App. F; the rule is in §7.1. K0502 cites §7; the rule is in §7.1. K0113 cites
  §5.2; groups and paths are in §5.3.

**Fix.** Add the codes, widen the descriptions, add one sentence for K0318 in §5.5.3, and correct the
section references.

## R9-21 (minor): startup-time state: the `init` ban is incomplete, and "set once" is undefined

**Location.** §5.4 rule 12, §5.4a rule 1 (K0202).

**Problem.** `init` is banned in leaves "because it would run before the ports are bound". A leaf's
package-level `var x = build()` runs at the same time, and can call a port or publish. Separately, a
library `var` is "set once, at startup": it is not stated if assignments in a library `init` count,
or how many.

**Fix.** Extend K0117: "A package-level variable initializer in a leaf cannot reach a port call or a
publish." Define: "At startup means in the variable's initializer or in an `init` function of the same
library. Each variable is assigned at most once there."

## R9-22 (minor): leaf tests

**Location.** §8, Appendix F "The zero value".

**Problem.** In a `keel test` of one leaf, no `main` binds ports. An unbound port is a nil interface,
so a test that forgets `keel.Bind` panics with a nil dereference. Libraries have no test command, but
Appendix F relies on "`keel test` of the leaf" to report a library's tier-2 zero-value failure.
`keel.Published` returns "the messages that the test published", which cannot be attributed under
`t.Parallel`.

**Fix.** Bind each unbound port in a test binary to a stub that fails the test with "port Checkout is
not bound; use keel.Bind". Add `keel test <library>`. Make `Published` panic under `t.Parallel`, as
`Bind` does.

## R9-23 (minor): startup and shutdown gaps

**Location.** §6.

**Problem.** "The process runs until ... a startup function returns an error or panics" conflicts with
"Elsewhere a panic behaves as in Go" (the process dies with no shutdown). The exit code of a
container that exits because all startup functions finished is not given. A leaf that a wire
constructs with settings and that also has `startup(ctx, s Settings)`: which settings does `startup`
get, and what if two wires construct it with different settings? When transports are created
(constructors may publish or call a remote port in step 2) is not stated.

**Fix.** Recover a panic in a startup function and treat it as an error, or delete "or panics". Exit
0 when all startup functions return `nil`. State that container-list settings are the only source
for `startup`. Create transports in step 1.

## R9-24 (minor): leaf names are not unique across containers

**Location.** §5.3, Appendix F (`rt.NewTopic("Orders.Placed", ...)`), §10.

**Problem.** Wires across containers use `Container/Leaf`, so two containers may each have a leaf
`Orders`. Then the runtime topic name `"Orders.Placed"`, `keel test Billing`, `keel context`, and
`keel fake` are ambiguous.

**Fix.** Either require leaf names to be unique in the system (new K01 code), or use
`Container/Leaf` in runtime names and accept it in each command.

## R9-25 (minor): flag name collisions, and per-container required flags

**Location.** §5.3 "Flags".

**Problem.** `stripe-key`, `stripe.key`, and `stripe_key` all read `STRIPE_KEY`. With one binary for
the system, it is not stated if a required flag that only the Worker wires use is required when the
binary runs as `shop api`.

**Fix.** Make two flag names that map to one environment variable a compile error (K0602). State
that a container reads and requires only the flags that its own wires, leaves, and transports use.

## R9-26 (minor): §6 warning sources and the fixed race-detector policy

**Location.** §6 "Checks on Go code are warnings", §10.

**Problem.** "The warnings are the ones that these established Go analysis tools define: errcheck,
nilaway, and Go's race detector". Error masking (K0702) is defined by none of them. `keel test` runs
every test with the race detector and offers no setting. The race detector needs cgo on some targets
and slows tests 2 to 20 times; that is operational policy fixed by the language (§2 "Mechanism, not
policy").

**Fix.** Say "K0701, K0703, and K0704 follow these tools; K0702 is Keel's own and defined here".
Add `keel test --race=false` (default on), and list its use in `keel audit`.

## R9-27 (minor): near-duplicate detection: "fingerprints", cost, and mode assignment

**Location.** §9, §11.2.

**Problem.** §9 defines similarity by Levenshtein distance between whole token sequences. §11.2 says
the holistic check compares "normalized fingerprints" and does not read bodies again; a fingerprint
(a hash) cannot give a Levenshtein distance. All-pairs comparison is quadratic in the number of
functions. Duplicates inside one leaf, between a leaf and a library, and between libraries are
assigned to no mode. The network and library rules (§5.4c, §5.4a) are also in neither "Finds" row.

**Fix.** Define the fingerprint as the normalized token sequence. Permit pruning pairs whose lengths
differ by more than `100 - duplicate_similarity` percent, which cannot reach the threshold. Assign
within-leaf duplicates to the per-leaf build, and all other pairs and the §5.4a and §5.4c rules to
the holistic check.

## R9-28 (minor): the primer omits rules that reject ordinary Go

**Location.** guide.md; §10 ("a model that reads only the primer must be able to write correct
Keel").

**Problem.** A model that follows only the primer hits compile errors it cannot predict:

- K0511: a function with a contract cannot name a result `err`. Go's `(n int, err error)` is common.
- K0305: a message type cannot hold a pointer, an interface, or a nested map (for example `*time.Time`
  for an optional field).
- §7.1: a `requires` conjunct is a fact in `ensures` only if the body never assigns its variables.
  The primer says "from `requires`" without the condition.
- §6: setup of a subscriber leaf (R9-16) and the order of startup against handlers.

The primer is about 8,400 characters, well under its 4,000-token budget, so there is space.

**Fix.** Add one line each for K0511, K0305, and the `requires`-fact condition.

## R9-29 (minor): contract scope gaps

**Location.** §7.1, §7.2, Appendix F "The zero value".

**Problem.** Not stated: that an invariant names fields unqualified (`invariant Balance >= 0`), and
what happens when a field name shadows a package-level name; whether `ensures` can name a named result
(`n`), and whether `err` still names a last result that the user named `e`; how the zero-value check
works for a generic struct (`Range[T]`), since a generated `init` cannot instantiate it without type
arguments; what "or make the field optional" means (§7.2), since Keel has no optional fields.

**Fix.** Add to §7.1: "In an invariant, the struct's fields are in scope unqualified and shadow
package-level names. In `ensures`, named results are in scope; `err` and `result` refer by position."
For generic invariants: check the zero value for each instantiation at compile time when tier 1, and
at the first construction when tier 2. Replace "make the field optional" with "use a pointer field
and write `P != nil => ...`".
