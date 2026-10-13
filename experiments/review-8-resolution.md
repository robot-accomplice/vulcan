# Review 8: resolution

Each finding in `review-8.md` and what changed in `docs/spec.md` or `docs/guide.md`. Section
numbers are the new ones: §8 to §14 became §7 to §13 (R8-38).

| Finding | Resolution |
|---|---|
| R8-01 | B.1: the parser, not the lexer, ignores a line-end semicolon in three exact places where Go allows no identifier. `limit` stays on its field's line. |
| R8-02 | §5.5.1, App. C: setting tags `required`, `topic`, `call`, `size`, `secret`; type `transport.Route`; a `ProcessSettings` struct from `--transport.<name>.<field>`; `New` runs once per process; `Wire.Settings` carries each wire's settings. |
| R8-03 | §5.5.0: per-key order removed. The language promises no order; with concurrency 1 a handler sees the transport's delivery order. `Order` is `Unordered` or `Ordered`. |
| R8-04 | §7.1: a fixed proof rule (facts on the left, `=>` antecedents, unassigned `requires` conjuncts; linear integer arithmetic and nil comparison). K0501 and check removal use the same rule. |
| R8-05 | B.2: `system.keel` can import libraries; a value parses as a Go expression and is resolved later by the setting's type; scope of a value is stated. |
| R8-06 | App. C, F: new package `keel.dev/keel/rt` (NewTopic, NewSubscription, Handle, Startup, ServeMethod, Call, Fail), used only by generated code (K0118). `keel` types are aliases of `rt` types. Validation functions are generated in the library of the message type. |
| R8-07 | App. C: transport code uses `transport.Transport` and `transport.Guarantees`; `Order` is declared. The whole API was compiled with Go 1.27 (stub). |
| R8-08 | App. C: `Bind(t, port *P, impl any)`; the compiler checks the assignment. Compiled. |
| R8-09 | App. F: `&T{...}` lowers to `keel_checkPtrT(&T{...})`. |
| R8-10 | App. F: the flag is also set at a reachable end of the body, and after the return values are evaluated. |
| R8-11 | §7.2, App. F: one syntactic rule (receiver variable, not assigned). A call through another expression is checked, also when it aliases the receiver. |
| R8-12 | §7.2: a pointer-method call on a field and slicing an array field are K0508; a conversion to `T` is checked as a construction; a call that receives `*T` from outside its methods is followed by an invariant check; `keel audit` lists `reflect`/`unsafe`. |
| R8-13 | §7.1: indexes, dereferences, and divisors must be proved; type assertions are not allowed (K0512); `old(x)` uses `requires` facts; implicit `requires r != nil` on pointer receivers. |
| R8-14 | §7.1, App. F: exact arithmetic only in contract expressions. `pure` bodies keep Go semantics; a panic in one while a contract runs is a contract failure (Kind `panic`). |
| R8-15 | §7.1: `pure` writes only values it created, no `append` to a slice it did not create, no interface calls, function values only from parameters, and each call must pass `pure` functions or checked literals. |
| R8-16 | §5.5, §5.5.2: the serializable types are listed; `time.Time` and `time.Duration` are encoded as protobuf well-known types. |
| R8-17 | §5.5, §5.5.1, App. C: an invalid received message is dead-lettered at once (`transport.ErrInvalid`); `DeadLetter.Message` is not validated. |
| R8-18 | §5.5.1, B.3: `limit` takes one bound per variable-size part, outer first, map key before value. K0317. |
| R8-19 | §5.5.2: a `limit` change is breaking. `DeadLetter.Wire` has a fixed limit of 1024 (K0316), and `DeadLetter` types are not in the lock. |
| R8-20 | §5.4 rule 9: boundary types are built-in, standard, or from a library; topic and remote types must come from a library. Lock keys use the import path. Examples are qualified. |
| R8-21 | §5.4 rule 11, K0116: code cannot assign to or take the address of a port or topic name. |
| R8-22 | §6: "Keel never changes what a body means", and the checks that reject bodies are listed. |
| R8-23 | §5.4c: "reaches" is defined (calls, function values, static calls through imported packages, cached per module version). |
| R8-24 | §5.4c, App. D: the network rule is now about reaching network functions, not importing packages. `crypto/x509` is usable anywhere; a library can query through a `*sql.DB` it receives. |
| R8-25 | §5.5: the publish panic is stated as an overridable default; `maxWait` bounds a `block` wait; a publish waiting at shutdown panics; `Publish` gets the process context. |
| R8-26 | §5.5, App. C: `Publish` once per subscriber wire; `Subscribe` gets a `dead` callback, and the runtime builds the `DeadLetter[T]`. |
| R8-27 | §8: `keel.Bind` panics after `t.Parallel`, as `t.Setenv` does. "Separate processes" removed. |
| R8-28 | §8, App. C: `keel.Deliver(t, sub, msg)`. |
| R8-29 | §5.3: a wire that names a type provides `new(T)` and takes no settings (K0604). |
| R8-30 | §5.5.3, K0315: a remote method's last result is `error`. |
| R8-31 | §5.5.1: `direct` defaults (ttl, backoff 1 s doubling to 5 min with jitter, `maxWait`), queue location (publisher), process flags (`listen`, `peers`, `dir`), delivery and retry counting. §5.5.2: wire format is the protobuf binary encoding. |
| R8-32 | Primer rewritten: `StripeGateway`, the `Shipped` wire, `pure func`, qualified types, `Deliver`, handler `ctx`, network rule, `init`. About 2,200 tokens. |
| R8-33 | §5.5.0, B.3: optional `ctx` first parameter on handlers. §6: a four-step shutdown with `shutdown_timeout` (default 30 s) and exit codes. |
| R8-34 | App. F: `main` generates one wrapper per wire that calls the provider statically, so a port call is one dispatch; tests pay a second. |
| R8-35 | B.3: `=>` is a binary operator at a new lowest level, valid at every nesting level. |
| R8-36 | §7.1: rules for `err`, `result`, `result.N`, the guard (by the proof rule), and names `err`/`result` (K0511). |
| R8-37 | Examples qualified and given bodies (§5.3, §5.4a, §7.1). |
| R8-38 | Wrong references fixed; §8 to §14 renumbered to §7 to §13; README updated. |
| R8-39 | App. E: K0007, K0116 to K0119, K0301 extended, K0315 to K0318, K0511 to K0514, K0605. |
| R8-40 | §6, K0704: a race fails the test, as in Go, and is also reported. |
| R8-41 | §6: names `errcheck`, `nilaway`, and the race detector as established tools, pinned per Keel version. |
| R8-42 | §5.3, §5.5.1: "should" for addresses; a `secret` field accepts only a flag (K0605). |
| R8-43 | §5.5.2: the baseline is `topics.lock` at Git `HEAD`; `keel check` writes the working file. |
| R8-44 | §5.5.2: the lock example shows a nested type and a remote port. |
| R8-45 | §6: each port is bound as soon as its provider exists; a constructor's remote call can return `keel.Unavailable`. |
| R8-46 | §5.4 rule 12, K0117: no `init` in leaves; libraries can have it. Generated `init` is allowed. |
| R8-47 | §5.4 rule 4: a subscribed `DeadLetter[T]` topic needs at least one route. |
| R8-48 | §9: similarity is 1 − Levenshtein distance / longer length, over token sequences. |
| R8-49 | §5.5.1: `topic` and `call` defaults apply only to their wire kind; required settings cannot come from `config`. |
| R8-50 | §5.3: same settings means identical source text after `keel fmt`. |
| R8-51 | §5.3, K0602: different defaults or maxima for one flag are an error. |
| R8-52 | §5.3: no extra runtime check; a disproved implication is K0109, an unproved one is listed by `keel audit`. |
| R8-53 | §5.4b: a leaf can have any number of `gateway for` and `entry for` lines. |
| R8-54 | §7.1: the `old(x)` rule is in the body. |
| R8-55 | §2: new principle row "Mechanism, not policy". |
