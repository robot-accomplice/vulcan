# Review 8: spec.md and guide.md, read as a compiler implementer

Scope: `docs/spec.md` (1,621 lines) and `docs/guide.md` (151 lines), read in full. No other file was read.
Go checks ran with go1.27.2 in a scratch directory. "Verified" below means the claim was compiled or run.

Severity: **B** = blocker (an implementer cannot proceed without inventing semantics), **M** = major
(wrong, unsound, or uncompilable as written), **m** = minor.

---

## Blockers

### R8-01 (B) The B.1 newline rule breaks ordinary Go declarations that use the new words as names
- **Where:** App. B.1 lines 1253-1257; §4.1 lines 129-132.
- **Problem:** "Outside a function body, Keel does not insert [a semicolon] when the next line starts with `requires`, `ensures`, `invariant`, `timeout`, or `limit`." Struct field lists and `var`/`const` blocks at package level are outside a function body. So
  ```go
  type Client struct {
      Addr    string
      timeout time.Duration
  }
  var (
      max   = 10
      limit = 20
  )
  ```
  lexes as `Addr string timeout time.Duration` and `max = 10 limit = 20`, which are syntax errors.
- **Why it matters:** It contradicts §4.1 ("Everywhere else it can be an identifier") and §6 ("Each Go construct compiles in Keel"). `timeout` and `limit` are common field names.
- **Fix:** Restrict the rule to the exact positions: after a complete `Signature` (functions, methods, port methods), after the closing `}` of a package-level struct `TypeSpec`, and after the `Type` of a `FieldDecl` only when the next token after `limit` is not `:`/`=`/an identifier-start of a type. Better: require `limit` on the same line as its field and say so.

### R8-02 (B) One `Settings` struct per transport cannot express the settings rules of §5.5.1 and §5.5.3
- **Where:** App. C lines 1398-1407; §5.5.1 lines 688-690, 705-710; §5.5.3 lines 837-839; App. B.2 lines 1299-1301; K0307, K0308, K0309.
- **Problem:** The spec needs per-setting metadata that the App. C interface has no way to declare:
  1. Topic-only versus call-only settings (K0308). `deadletters` is "required" (`keel:"required"`) on topic wires but must be absent on call wires (`via direct{}`), with one struct.
  2. Which fields are "routes" (B.2 says the setting's type decides whether `Worker/Alerts.DeadPlaced` parses as an endpoint or a division). No route type exists in App. C.
  3. Which fields are "sizes" that need `max:` (K0309) and feed the memory calculation.
  4. `func New(s Settings)` takes one value, but settings differ per wire. Whether `New` runs once per wire or once per process is not stated, and `Wire` carries no settings.
- **Why it matters:** K0307, K0308, K0309, the endpoint parse rule, and the memory bound are unimplementable from the published API.
- **Fix:** Define in `keel.dev/keel/transport` a `Route` type and struct tags `keel:"required"`, `keel:"topic"`, `keel:"call"`, `keel:"size"`; state "`New` is called once per wire with that wire's merged settings" (or give `Publish`/`Subscribe` a per-wire `Settings` argument).

### R8-03 (B) Per-key ordering under `concurrency` has no key
- **Where:** §5.5.0 lines 666-673; App. C lines 1413-1416, 1425.
- **Problem:** "If the transport keeps order per key, the runtime runs at most one handler at a time for each key." The language has no syntax that names a message key, and `deliver func(ctx, msg []byte) error` carries no key. `Order` (`PerKey`) is also undefined in App. C (see R8-07).
- **Why it matters:** The runtime cannot implement the stated guarantee, and a transport cannot partition by key.
- **Fix:** Either remove per-key ordering from §5.5.0 and `Guarantees`, or define how a message type names its key (for example, a field tag `keel:"key"`) and add the key to `deliver`.

### R8-04 (B) The strength of the static prover decides which programs compile, and it is not specified
- **Where:** §8.1 lines 942-943 (indexes "proved in range by the contract itself"), line 951 (pure bodies need "proved indexes"), §8 line 1012, K0501, K0504, K0505; §8.2 line 982-983.
- **Problem:** K0504 rejects unproved indexes and K0501 rejects contracts the compiler "proves" fail. Whether a program compiles therefore depends on the prover. `pure func IsSorted` (line 954) cannot be written without proving `xs[i+1]` in range inside a loop, which needs loop-invariant reasoning. No decision procedure, logic fragment, or minimum capability is given.
- **Why it matters:** Two conforming compilers accept different programs; the spec's own `IsSorted` example may be uncompilable.
- **Fix:** Specify the proof rule exactly (for example: "an index `e[i]` is proved when `0 <= i && i < len(e)` follows by linear arithmetic over the enclosing `&&`/`=>` guards and `for i := 0; i < len(e); i++` loop headers"), and say K0501 fires only for that same fragment; everything else is a runtime check.

### R8-05 (B) Name resolution and parsing in declaration files are undefined
- **Where:** App. B.2 lines 1263-1264, 1267-1287, 1299-1301; §5.5.1 line 705 (`whenFull` values `block`, `dropOldest`, `dropNewest`).
- **Problem:**
  1. `SystemFile` has no `ImportDecl`, and only `time` is visible. Bare identifiers such as `dropOldest`, or a library constant used as a setting, have no scope.
  2. `Value = ... | identifier Settings | ... | Expression`: `direct{ capacity: 5_000 }` matches both `identifier Settings` and a Go composite literal. EBNF alternation is not ordered.
  3. "The type of the setting decides" makes the parser depend on type information from a transport's Go package, so `system.keel` cannot be parsed before Go packages are loaded and type-checked.
- **Why it matters:** An implementer cannot write a parser or a resolver for `system.keel` without inventing the scope rules.
- **Fix:** Parse one superset `Value` (endpoint-or-expression) and resolve it in a later pass; state the scope of a setting value (the fields' enum constants of the transport package, `time`, and imports allowed at the top of `system.keel`), and add `{ ImportDecl ";" }` to `SystemFile` if library constants are allowed.

---

## Major

### R8-06 (M) Appendix F's lowering cannot compile against Appendix C's runtime API
- **Where:** App. F lines 1569-1575, 1618-1620; App. C lines 1347, 1353-1357.
- **Problem:** `keel.Topic` and `keel.Subscription` are opaque structs in another package, with no exported constructor; `keel_newTopic[...]` and `keel_Register` "give ... to the runtime" through an API that App. C does not have. App. C says it is "all of the runtime that code and tests can see", and generated code is code. `keel_validate` is "a generated method" on each message type, but `keel.DeadLetter[T]` is in package `keel`, so no method can be added to it from outside, and its `Wire` limit is per-system.
- **Fix:** Add an internal package (for example `keel.dev/keel/rt`) to App. C with `NewTopic[T]`, `NewSubscription[T]`, `Register`, and a validator hook that takes a function value, and say that only generated code imports it.

### R8-07 (M) The App. C transport code does not type-check (verified)
- **Where:** App. C lines 1402-1416.
- **Problem:** `func New(s Settings) (keel.Transport, error)` and `keel.Guarantees{...}` name package `keel`, but line 1407 puts the types in `keel.dev/keel/transport`. `Order` is never declared. `go build` reports `undefined: keel.Transport`, `undefined: keel.Guarantees`, `undefined: Order`.
- **Fix:** Write `transport.Transport` and `transport.Guarantees`, and add `type Order int; const (None Order = iota; PerKey; Total)` (or drop `PerKey` per R8-03).

### R8-08 (M) The documented `keel.Bind` call fails Go type inference (verified)
- **Where:** App. C line 1391; §9 line 1032; guide line 139.
- **Problem:** With `func Bind[P any](t testing.TB, port *P, impl P)`, the call `keel.Bind(t, &Checkout, CheckoutOK{})` fails: `type CheckoutOK of CheckoutOK{} does not match inferred type keel_Checkout for P`. Line 1395 says the compiler rewrites the call, but test files are "Go tests" (§9), and gopls/`go vet` on the generated or source code reject the published signature.
- **Fix:** Declare `func Bind[P any](t testing.TB, port *P, impl any)` (the compiler checks assignability), or document the call as `keel.Bind[keel_Checkout]`-free by generating a per-port `BindCheckout(t, impl)`.

### R8-09 (M) `&T{...}` lowering produces a type error (verified)
- **Where:** App. F lines 1606-1607.
- **Problem:** "`T{...}` and `&T{...}` become `keel_checkT(T{...})`, which ... returns the value." For `&T{...}` that returns `T` where `*T` is required: `cannot use keel_checkAccount(Account{…}) (value of struct type Account) as *Account value`. Taking `&` of the call result is also illegal (not addressable).
- **Fix:** Lower `&T{...}` to `keel_checkPtrT(&T{...})`, which checks `*p` and returns `p`.

### R8-10 (M) `ensures` is never checked on a function that ends without a `return` statement (verified)
- **Where:** App. F lines 1585-1590; §8 table line 933 ("checked at every return").
- **Problem:** The flag `keel_returned` is set only "at each `return`". A function with no results that falls off the end of its body (for example `Deposit(a *Account, x int)` with `ensures a.Balance == old(a.Balance) + x`) never sets it, so the check is skipped. A scratch lowering with a deliberate bug printed `ensures SKIPPED (no return stmt)`.
- **Fix:** Also set `keel_returned = true` at the end of the body, and set it after the return expressions are evaluated (`r0, r1 = e0, e1; keel_returned = true; return`), so a panic inside a return expression does not trigger the check.

### R8-11 (M) "Outer calls only" is defined two different ways, and the App. F version is unsound under aliasing
- **Where:** §8.2 lines 988-992 ("a nested call on the same value"); App. F lines 1600-1605 ("a call through the receiver variable").
- **Problem:** "Same value" needs alias analysis; "receiver variable" is syntactic. They differ when `to == a` in `Transfer(to *Account, x int)`: `to.Deposit(x)` calls the checking wrapper while `a` may be in a broken intermediate state, which gives a spurious panic. They also differ when the receiver is copied (`b := a; b.Withdraw(x)`) or reassigned.
- **Fix:** Make §8.2 state the syntactic rule of App. F explicitly ("a call whose receiver expression is the receiver identifier, unreassigned"), and say that a call on an alias of the receiver is checked.

### R8-12 (M) Three ordinary Go paths bypass struct invariants and nothing reports them (verified)
- **Where:** §8.2 lines 970-976, 982-983; K0507, K0508.
- **Problem:**
  1. Implicit address-of: `b.In.Set(-1)`, where `In` is a field and `Set` has a pointer receiver, takes `&b.In` outside `Box`'s methods. The scratch program wrote `-1`.
  2. Conversion: `Account(struct{ Balance int }{-5})` creates an `Account` without a `T{...}` literal. The scratch program printed `-5`.
  3. Reflection decoding: `json.Unmarshal(data, &acct)` writes fields through a whole-struct pointer, which §8.2 allows.
- **Fix:** Treat an implicit `&` of a field (method call with pointer receiver, slicing an array field) as K0508; treat a conversion to `T` as a construction and lower it through `keel_checkT`; have `keel audit` list each `&T` passed to a function outside the package (or each reflect/encoding call on `*T`).

### R8-13 (M) Contract expressions can fail, which §8.1 says is impossible
- **Where:** §8.1 lines 938-945; App. F lines 1581-1584.
- **Problem:** §8.1 rules out only overflow and out-of-range indexes. A contract can still panic through a `nil` dereference (`a.Balance` with `a == nil`), division or modulo by zero, a map or slice through a nil pointer, a type assertion, or an integer conversion. `old(a.Balance)` is evaluated at entry without its guard, so `ensures err == nil => a.Balance == old(a.Balance)` dereferences `a` even on a path where the guard is false.
- **Fix:** Extend the "proved" requirement to pointer dereference and divisors (or define these as contract failures with a distinct `Kind`), and evaluate `old(e)` only when `e` cannot fail, else reject it.

### R8-14 (M) Exact arithmetic changes the meaning of `pure` function bodies, and the result type is undefined
- **Where:** §8.1 line 951; App. F lines 1592-1594; §2 "Function bodies are Go, unchanged"; §6 lines 857-860.
- **Problem:** A `pure` function is an ordinary function that ordinary code also calls. Its integer arithmetic "gives the mathematical result", so a hash or checksum written with wrap-around gives a different value than in Go. When the exact value does not fit the declared result type (`pure func Sum(xs []int) int`), the spec does not say whether it panics, truncates, or fails. Floats, conversions (`int32(x)`), and shifts are not addressed.
- **Why it matters:** It is a semantic change inside a function body, which §2 and §6 rule out.
- **Fix:** Apply exact arithmetic only to the contract expression itself, keep Go semantics in `pure` bodies, and state that an overflow in a contract-called `pure` function is a contract failure (or restrict exactness to tier 1).

### R8-15 (M) The purity check is unsound and undefined for function values
- **Where:** §8.1 line 951, lines 961-962; App. F lines 1612-1616; App. C lines 1386-1388.
- **Problem:** A `pure` function "can assign only to its local variables and its results". `ys := xs; ys[0] = 1` and `append(xs, v)` with spare capacity write through a local into the caller's array. `keel.All(xs, p)` is listed as pure, but it calls `p`, a `func(T) bool` with no purity in its type; a closure that mutates state passes. Calls through interface methods have the same gap.
- **Fix:** Forbid index/field assignment through any local that is not freshly allocated in the function, forbid `append` to a parameter-derived slice, and require that a function-valued argument to a `pure` function be a named `pure` function or a literal whose body passes the same check.

### R8-16 (M) The runtime's own `DeadLetter[T]` is not serializable under the spec's rule (verified)
- **Where:** §5.5.1 lines 721-726; App. C lines 1360-1366; §5.5 lines 634-636; K0305.
- **Problem:** Messages "cannot contain ... pointers"; K0305 rejects "a pointer". `DeadLetter.Time` is `time.Time`, whose fields are `wall uint64`, `ext int64`, `loc *time.Location`, and are unexported, so the compiler-owned wire format cannot encode it by field.
- **Fix:** State that `time.Time` (and `time.Duration`) are encoded as built-in scalars by the wire format, or change `Time` to `int64` Unix nanoseconds.

### R8-17 (M) A message that fails its invariant or limit cannot be dead-lettered
- **Where:** §5.5 lines 637-639; §5.5.1 lines 719-730; App. F lines 1574-1575.
- **Problem:** A received message that fails its invariant "goes back to the transport", which retries and then dead-letters it. The dead letter is `DeadLetter[T]` with `Message T`, and `keel_validate` runs "at each publish and at each receipt", including the invariant and limits of `T`. The invalid message fails validation again inside its dead letter, so it can never be delivered; retrying it is also pointless.
- **Fix:** Skip `T`'s validation for `DeadLetter[T].Message`, and dead-letter a message that fails validation at receipt at once, without retries.

### R8-18 (M) `limit` cannot bound nested variable-size data, so common message types can never use `direct`
- **Where:** §5.5.1 lines 712-718; App. B.3 line 1329; guide lines 121-124; K0304.
- **Problem:** A `limit` goes "on a field, not on a named type" and counts elements of a slice or map. A `[]string` field, a `map[string]int` field (keys), or a `[]OrderID` field has elements whose size no syntax can bound. The memory limit "capacity × maximum message size" is then infinite, and K0304 rejects the type.
- **Fix:** Define how element sizes are bounded (for example `Tags []string limit 10, 64` for count then element bytes, and `limit 10, 64, 0` for maps), or allow `limit` on a named string type.

### R8-19 (M) The message-evolution rules do not cover a change of `limit`
- **Where:** §5.5.2 table lines 785-791; lock format lines 766-777; guide line 125-126; §5.5.1 lines 725-726.
- **Problem:** The lock records each limit, and both sides validate limits, but the table does not say whether raising or lowering a limit is compatible. Lowering it rejects messages from an older publisher; raising it makes an older subscriber reject messages. `DeadLetter.Wire`'s limit is "the longest wire in the system", so adding a longer wire changes a locked limit.
- **Fix:** Add a row: "Change a limit: Breaking (both sides check it)", and state that `DeadLetter` types are not in the lock, or that their `Wire` limit is a fixed constant.

### R8-20 (M) Where topic message types come from, and how their identity is decided, is not specified
- **Where:** §5.4 rule 9 lines 358-362; §5.5 lines 566-576; lock example line 768; K0306; guide lines 44-45.
- **Problem:** Rule 9 requires library types only for port signatures. Topic message types in every example are unqualified (`topic[OrderPlaced]`), so they look leaf-private. Two leaf-private types in two containers are never identical, so K0306 would reject every topic wire. The lock names a type `shop.OrderShipped` (the system name, not a Go import path). A message type from a non-Keel Go package cannot receive the generated `keel_validate` method. Also, "must come from a library" literally excludes `int`, `error`, `string`, and `time.Time`.
- **Fix:** Extend rule 9 to topic message types and remote-port types ("declared in a library, or predeclared, or in the standard-library allow list"), qualify the examples, and key the lock by import path (`shop/lib/orders.OrderShipped`).

### R8-21 (M) Nothing prevents leaf code from rebinding a port name
- **Where:** §5.3 lines 230-232 ("the binding cannot change after that"); App. F line 1552; App. E.
- **Problem:** The port name lowers to `var Checkout keel_Checkout`, a mutable package variable in the leaf's own package. `Checkout = nil` or `Checkout = myFake{}` in a leaf is legal Go. The body claims immutability, but no rule, check, or diagnostic enforces it, which also contradicts "compiler first".
- **Fix:** Add a rule and a code (for example K0116, "a port or topic name is assigned or its address is taken outside `keel.Bind`").

### R8-22 (M) "Keel adds no rule inside a function body, except contracts" is false
- **Where:** §6 lines 859-860; §2 "Go ergonomics"; vs §5.4a rules 1-2 (lines 427-455), §5.4 rule 1, R8-21.
- **Problem:** K0202 (assignment to a library `var` inside any body), K0203 (a call to `os.Open` inside a library body), and K0101 are rules about statements inside function bodies, and they are not contracts.
- **Fix:** Restate §6 as "Keel adds no rule that changes what a body means; it adds checks that reject some bodies (contracts §8, library rules §5.4a, boundary rules §5.4)", and list them.

### R8-23 (M) The acquiring-function rule does not say direct or transitive
- **Where:** §5.4a rule 2 lines 440-443; §5.4c rule 2 lines 534-537; App. D lines 1441-1443.
- **Problem:** "A call from a library to an acquiring function is a compile error." If the rule is direct only, a library calls any third-party helper that calls `os.ReadFile` (or takes `f := os.Open` as a value) and the rule means nothing. If transitive, it needs a whole-program call graph through Go dependencies and interface dispatch, which the per-leaf build (§12.2) does not have.
- **Fix:** State "direct calls and function values only, in Keel code", and have `keel audit` list each imported non-Keel package whose code calls a listed function.

### R8-24 (M) The network-package rule forbids libraries from using common non-network packages (verified)
- **Where:** §5.4c rule 1 lines 524-533; §5.4a rule 2 lines 440-441.
- **Problem:** `go list -deps` shows that `crypto/x509`, `crypto/tls`, `mime/multipart`, `net/mail`, `log/syslog`, and `expvar` all depend on `net`. So a library or an ordinary leaf cannot parse a certificate or a multipart body. §5.4a rule 2 also says a library can "query ... through a resource that its caller gives it", but a `*sql.DB` parameter needs `database/sql`, a network package that a library cannot import.
- **Fix:** Classify by symbol use (calls into `net.Dial*`, `net.Listen*`, `exec.Command`, `sql.Open`) rather than by transitive import, or state the consequence and remove "query" from §5.4a rule 2.

### R8-25 (M) A publish failure in the transport is a panic, which takes the policy from the developer
- **Where:** §5.5 lines 590-596; App. C lines 1356, 1432-1433; §5.5 lines 550-552 ("the transport owns the operational policy").
- **Problem:** "A transport that cannot accept the message panics too." When a broker is down, every `Publish` in an HTTP entry leaf panics, and the developer's only mechanism is `recover`. `Publish` also has no `context.Context`, so under the default `block` policy a publish can wait without limit, and the runtime has no `ctx` to pass to `Transport.Publish`.
- **Why it matters:** "Mechanism, not policy" (§5.5) and "Go ergonomics" (errors are values).
- **Fix:** State which `ctx` the runtime passes and what bounds a `block` wait, and either give a mechanism to observe the failure (for example a per-wire `onPublishError` setting) or justify the panic in §5.5 as an overridable default.

### R8-26 (M) `Publish` semantics per wire versus per topic are undefined
- **Where:** App. C lines 1417-1426; §5.5 lines 597-600; §5.5.1 line 740.
- **Problem:** `Publish(ctx, w Wire, msg)` takes one `Wire`, and a wire is a subscriber's line. For a topic with two subscriber wires on `kafka{ topic: "samples" }`, the runtime either calls `Publish` twice (duplicate records in one Kafka topic) or once (then which `Wire`?). How a non-`direct` transport produces a `DeadLetter[T]` in the compiler's wire format is also not given.
- **Fix:** State "the runtime calls `Publish` once per subscriber wire", and add an encoder for dead letters to `keel.dev/keel/transport`, or give the transport a `DeadLetter(ctx, w, msg, err, attempts)` callback.

### R8-27 (M) Parallel tests "run in separate processes" with no mechanism
- **Where:** §9 line 1035; §9 line 1026 ("Tests are Go tests").
- **Problem:** In Go, `t.Parallel()` runs tests as goroutines in one process, so `keel.Bind` on a package-level port name races. The spec states a different behavior but not how the test runner achieves it (one process per test? per `t.Parallel` call? subtests?).
- **Fix:** Specify: "`keel test` runs each top-level test that calls `t.Parallel` in its own process (`-test.run '^Name$'`); `t.Parallel` in a subtest is a compile error", or make `keel.Bind` panic when called after `t.Parallel`.

### R8-28 (M) A handler cannot be called from a test
- **Where:** App. F lines 1571-1573; §9 lines 1028-1029.
- **Problem:** `handle Shipped(...)` lowers to `keel_handle_Shipped`, and `Shipped` is a `*keel.Subscription` "with no methods for code". §9 says each leaf is testable in isolation, but App. C offers no way to deliver a message to a handler in a test.
- **Fix:** Add `func Deliver[T any](t testing.TB, s *Subscription[T], msg T) error` to App. C.

### R8-29 (M) A wire that names a type, not a constructor, is undefined
- **Where:** §5.3 lines 263, 266, 290-292.
- **Problem:** "The wire names an exported type or an exported constructor." `Billing.Ledger: LedgerMemory.Store` names a type. The spec does not say whether the provider is the zero `Store` or `&Store{}`, which decides the method set; or whether `Store{ ... }` settings fill the type itself.
- **Fix:** State "a wire that names type `T` provides `new(T)`; settings on it are a compile error", or allow only constructors.

### R8-30 (M) A remote method without an `error` result has no defined behavior
- **Where:** §5.5.3 lines 817-818; App. E.
- **Problem:** Each remote method "can also return" `Unavailable`, `Timeout`, `RemotePanic`. A `needs remote port` method declared `func Count() int` has no way to return them, and no rule forbids it.
- **Fix:** Require the last result of a remote method to be `error`, with a new code (for example K0315).

### R8-31 (M) The built-in `direct` transport is not specified enough to build
- **Where:** §5.5.1 lines 697-711, 741.
- **Problem:** Missing: how a transport declares its own startup flags (`--transport.direct.peers=worker=10.0.0.7:7000` is not a `flag("...")` in source, and its value syntax is not given); where `durable` queues are stored, and on which side (publisher or subscriber); the default `ttl`; the backoff base and cap; and whether the `capacity` bound is per subscriber wire.
- **Fix:** Add a `direct` subsection with these values, and a rule that a transport's process-level flags are `--transport.<name>.<field>` from a second `ProcessSettings` struct.

### R8-32 (M) The primer teaches code that the spec rejects, and omits syntax it requires
- **Where:** guide lines 73-75, 44-45, 56-69, 97-100; §11 line 1100 ("a model that reads only the primer must be able to write correct Keel").
- **Problem:**
  1. `Billing.Cards: Stripe.NewCards{ ... }` wires to `Stripe`, which the same example declares as `external`, not a leaf (K0107).
  2. Orders `subscribes Shipped`, but the guide's `system.keel` never wires it (K0104).
  3. Contracts "may call only `pure` functions", but the primer never shows the `pure func` declaration.
  4. `SKU` (line 127) and `OrderPlaced` are unqualified, against rule 9 (see R8-20).
- **Fix:** Use `StripeGateway.NewCards`, add `Api/Orders.Shipped: Worker/Picking.Shipped via direct{...}`, add one `pure func` line, and qualify the types.

### R8-33 (M) Handlers have no `context.Context`, and shutdown of in-flight work is unspecified
- **Where:** App. B.3 line 1331; §6 lines 917-920; App. C line 1425.
- **Problem:** The `Handler` production fixes the signature to one parameter, so a handler cannot receive the `ctx` that `deliver` gets, cannot pass a deadline to a remote port (§5.5.3 line 826), and cannot see cancellation. On a stop signal, the order of "cancel `ctx`", "stop subscriptions", "wait for handlers", and `Transport.Close` is not given, nor the exit code.
- **Fix:** Allow `handle X(ctx context.Context, m T) error` as a second form, and add a numbered shutdown sequence to §6.

### R8-34 (M) A port call is two dynamic dispatches, not one
- **Where:** §5.3 lines 238-240; §12.3 line 1204; App. F lines 1551-1563.
- **Problem:** `Checkout` holds `keel_CheckoutChecked{impl}` in a `keel_Checkout` interface, and the wrapper calls `w.impl.Pay` through a second interface. §5.3 and §12.3 claim "one dynamic dispatch".
- **Fix:** Make the wrapper generic over the concrete provider type in generated `main` (`keel_CheckoutChecked[*billing.Payer]`), or correct the cost to two dispatches.

---

## Minor

### R8-35 (m) `=>` cannot appear inside parentheses or call arguments
- **Where:** App. B.3 lines 1332-1335.
- **Problem:** `ContractExpr = Expression [ "=>" ContractExpr ]`, and Go's `Expression` contains `"(" Expression ")"`, not `ContractExpr`. So `a && (b => c)` and `keel.All(xs, ...)` with an implication do not parse.
- **Fix:** Define a contract-expression grammar in which `PrimaryExpr` parentheses contain `ContractExpr`.

### R8-36 (m) The `result` guard and naming rules are under-specified
- **Where:** §8 table line 933; App. B.3 lines 1336-1337; K0502, K0510.
- **Problem:** "Guarded by `err == nil =>`": is `err == nil && result > 0` or `err != nil || result > 0` accepted? Is plain `result` legal when there are two non-error results? What happens with named results, a parameter named `err` or `result`, or an `error` that is not the last result?
- **Fix:** State: the guard is the syntactic antecedent `err == nil` of an enclosing `=>` or a conjunct to the left in `&&`; plain `result` with two or more results is K0510; `err`/`result` shadow nothing and a parameter with those names is an error.

### R8-37 (m) Examples contradict the rules they illustrate
- **Where:** §5.3 lines 203-205, 216-221 (unqualified `CustomerID`, `Money`, `Receipt`, `MaxOrder` in a port, against rule 9 and the missing scope of `MaxOrder`); §8.1 lines 956-958 (`Merge` has a contract and no body, against B.3 line 1342-1343); §5.4a line 448 (`Parse` without a body).
- **Fix:** Qualify the types (`money.Money`) and give each example a `{ ... }` body.

### R8-38 (m) Broken and missing cross-references
- **Where:** line 712 "bounded with `limit` fields (§6)": §6 says nothing about `limit` (should be App. B.3 / this section). Line 1179 "every function declares its types (§6)": §6 has no such rule. Section 7 does not exist (§6 is followed by §8).
- **Fix:** Point line 712 to App. B.3, line 1179 to §5.3/§12.2 (or remove it), and renumber §8-§14 or add a §7.

### R8-39 (m) Rules in the body with no diagnostic code in Appendix E
- **Where:** App. E.
- **Problem:** No code exists for: `invariant` on a non-struct or non-package-level type (B.3 line 1338); a function with a contract and no body (B.3 line 1342); a handler whose parameter type is not the topic's type (B.3 line 1340); `limit` on a fixed-size field or with a non-constant bound (B.3 line 1339); `timeout` on a non-remote port method (the `PortMethod` production allows it); a `uses` line naming an unknown node; a transport name not in lower case (B.2 line 1302); `gateway for` naming a person; port-name assignment (R8-21); a remote method without `error` (R8-30).
- **Fix:** Add codes for each, or state that they are K0001.

### R8-40 (m) K0704 is a warning, but Go's race detector fails the test
- **Where:** §6 lines 877-878; K0704 line 1537.
- **Problem:** Under `-race`, Go marks the test failed ("race detected during execution of test"). "A warning never stops a build" does not say whether `keel test` passes.
- **Fix:** State that a detected race fails the test (as in Go) and K0704 is reported as well.

### R8-41 (m) The analysis tools are not "Go's own"
- **Where:** §6 lines 866-867.
- **Problem:** `errcheck` and `nilaway` are third-party tools, so "the ones that Go's own analysis tools define" is false.
- **Fix:** Name the tools and their pinned versions in an appendix.

### R8-42 (m) "Addresses and credentials never appear in source" is false and unenforced
- **Where:** §5.5.1 lines 697-698; §5.3 line 299 (`default: ":8080"`).
- **Problem:** The spec's own example puts an address in source as a flag default, and nothing rejects `APIKey: "sk_live_..."`. This is a claimed rule with no check, against "compiler first".
- **Fix:** Reword as a recommendation, or reject string literals in fields tagged `keel:"secret"`.

### R8-43 (m) The baseline of the lock check is ambiguous
- **Where:** §5.5.2 lines 755-757 ("the last committed schema") vs lines 782-783 (`keel check` "updates the lock").
- **Problem:** If `keel check` rewrites `topics.lock` on every run, the next run compares against the working file, not the committed one. Two compatible local edits (add field, then change its type) become "breaking" before anything ships, or a breaking change passes if compared to the working file.
- **Fix:** State that the comparison baseline is the lock at `HEAD` (or the file on disk), and which one `keel check` rewrites.

### R8-44 (m) The lock format covers neither remote ports nor nested types
- **Where:** §5.5.2 lines 766-777; §5.5.3 lines 828-829, 842-843.
- **Problem:** Remote-port parameter types, result types, and contracts are "in the lock", and `[]Line` refers to a nested type, but the example and format show only flat message types.
- **Fix:** Add `port` and nested-`type` entries to the format example.

### R8-45 (m) Port binding happens in step 2 and step 3
- **Where:** §6 lines 906-911.
- **Problem:** Step 2 says a constructor's ports "are already constructed and bound"; step 3 says "`main` binds every port". Also, a constructor that calls a `remote` port in step 2 runs before the other container serves it (step 3 there).
- **Fix:** Say "each port is bound as soon as its provider is constructed", and that a constructor's remote call can return `keel.Unavailable`.

### R8-46 (m) Package `init` in leaves is not addressed
- **Where:** guide lines 27-28 ("A leaf's code runs only from a handler, a port ..., an entry leaf's server, or ... `startup`"); App. F lines 1608-1610 (uses `init`).
- **Problem:** Constructors and Go `init` functions also run leaf code, before ports are bound. The spec neither allows nor forbids `init` in leaves.
- **Fix:** State whether `init` is allowed in leaves (and that ports are nil there), and list constructors in the guide sentence.

### R8-47 (m) A `DeadLetter` topic with no `deadletters` wire is not covered by rule 4
- **Where:** §5.4 rule 4 lines 349-351.
- **Problem:** Rule 4 exempts dead-letter topics from "exactly once" but does not say whether zero routes is K0104.
- **Fix:** Add: "A subscribed `DeadLetter[T]` topic needs at least one `deadletters` route (K0104)."

### R8-48 (m) The near-duplicate similarity metric is undefined
- **Where:** §10 lines 1059-1062; §6 lines 880-881 (warnings depend only on code and version).
- **Problem:** "At least 90% similar" over token sequences can mean edit distance, LCS ratio, or shingle Jaccard. Results differ.
- **Fix:** Name the metric (for example "1 - Levenshtein(a, b) / max(len a, len b)").

### R8-49 (m) The `config` default transport cannot apply to call wires or supply routes
- **Where:** App. B.2 line 1313; §5.5.3 lines 837-839.
- **Problem:** `transport: direct{ capacity: 5_000 }` contains a topic setting, which is K0308 on a call wire, and `deadletters` (required) cannot be a useful default.
- **Fix:** State that topic-only defaults apply only to topic wires and that required settings cannot come from `config`.

### R8-50 (m) "Same settings value" for instance sharing is undefined
- **Where:** §5.3 lines 303-304.
- **Problem:** Equality is not defined for settings filled by flags (compile-time text or runtime value?) or for settings structs that contain slices or maps (not comparable in Go).
- **Fix:** Define equality as identical settings source text after `keel fmt`.

### R8-51 (m) Two uses of one flag with different `default:` or `max:` are not covered
- **Where:** §5.3 lines 312-316; K0602.
- **Fix:** Make different defaults or maxima for one flag name an error under K0602.

### R8-52 (m) "A runtime check at the wire" for an unproved implication is undefined
- **Where:** §5.3 lines 324-327.
- **Problem:** The implications "port `requires` ⇒ method `requires`" and "method `ensures` ⇒ port `ensures`" cannot be checked as such at runtime; each side's clauses are already checked at each call.
- **Fix:** Say that nothing extra is generated, and that `keel audit` lists the unproved implication.

### R8-53 (m) Gateway and entry roles: number per leaf is unspecified
- **Where:** §5.2 line 182 ("its `gateway for` or `entry for` line"); App. B.2 line 1290.
- **Problem:** The grammar allows several `gateway for` and `entry for` items in one leaf. The body implies one. A Stripe gateway that also receives Stripe webhooks needs both.
- **Fix:** State the allowed combinations.

### R8-54 (m) The `old(x)` restriction is only in the lowering appendix
- **Where:** K0509 line 1526 cites App. F; §8 table line 950 does not mention it.
- **Fix:** Move the rule "`old(x)` needs a type with no references" into §8.1.

### R8-55 (m) "Mechanism, not policy" is used but not stated in §2
- **Where:** §2 table lines 76-88; §5.5 lines 550-552.
- **Problem:** The principle drives §5.5 and §5.5.1 but is not in the principles table, so the spec gives no general rule for when the language may set a policy (as in R8-25).
- **Fix:** Add a row: "Mechanism, not policy: the language gives mechanisms and overridable defaults; operational policy is the developer's."
