# Review 9: resolution

Each finding in `review-9.md` and what changed in `docs/spec.md` or `docs/guide.md`. Go claims
were checked in a scratch module: the new `keel.RemoteError`/`keel.NewError` pair, the structured
`transport.Wire`, and the new `ensures` and invariant lowering (shadowed result, `r0` in the body,
and a mutex-guarded invariant under `go test -race`) all compile and pass.

| Finding | Resolution |
|---|---|
| R9-01 | §5.5.3, App. C: a callee error crosses as text and a code; the caller gets `*keel.RemoteError`; `keel.NewError(code, text)` keeps identity, so `errors.Is` matches. `error` is allowed as a remote method's last result. |
| R9-02 | §5.5, §5.5.2: a normative encoding table (sint64, uint64, bytes for `string`, wrapper messages for nested repeated or map values, UTC timestamps); map keys limited to integers, `bool`, `string`; topic message types are structs; recursive types are K0305; out-of-range integers make a message invalid. |
| R9-03 | §5.5.1, App. C: `transport.Wire` has `Kind`, `From`, `To`; the dead-letter hop is defined (process that holds the message, source transport, route as `To`, routes cleared); `Error` cut to 4,096 bytes; `Guarantees` adds `RequiresLimits` and `Calls` as a constant literal; `direct` has a path and needs no declaration. |
| R9-04 | §5.5.1: setting names are field names with a lower-case first letter, or a `name=` tag; enum values follow the same rule; `required` with `topic`/`call` means that wire kind only; all `direct` settings are topic settings. |
| R9-05 | B.1: after `Parameters`, `requires`/`ensures`/`timeout` starts a contract unless the next token is `{`, `;`, `.`, or `[`. |
| R9-06 | §7.1: `requires` facts in `ensures` only for parameters never assigned and never read through; call-site and return facts defined; atoms and ranges defined. |
| R9-07 | §7.1: exactness for integer `+ - *`, unary `-`, conversions, comparisons; `/` and `%` truncate; bitwise and shifts are Go; floats are Go; overflow into a `pure` parameter fails the contract; slices and shift counts must be proved (K0504). |
| R9-08 | §7.2: post-call checks for every value that holds `T` through a pointer, also inside methods; the "skip" needs no pointer passing; function literals count as outside unless called directly or deferred; `go` in a method is K0520. |
| R9-09 | §7.2, App. F: invariant and `ensures` are checked at each return before deferred calls; K0521 warns on non-deferred unlock. Verified race-free. |
| R9-10 | §7.1: `pure` writes only through paths rooted at a local it created, by value; loads, slice expressions, and call results cannot be written through. |
| R9-11 | App. F: each return becomes a block with typed `keel_t` temporaries; the added handler parameter is `keel_ctx`; every generated name starts with `keel_`/`Keel_`. Verified. |
| R9-12 | App. F: `Keel_invariant`, `Keel_checkT`, `Keel_checkPtrT` are exported; §5.3: a constructor's first result is an exported concrete type (K0604); a port contract names no identifier of its own leaf (K0120). |
| R9-13 | §5.4c: a conversion to an interface reaches the type's methods in the method set; a package-level interface variable reaches its initializer's methods; `(*http.Transport).RoundTrip` listed; §5.4b rule 3 no longer claims the diagram cannot be wrong. |
| R9-14 | App. D: `syscall` lists only the functions that meet the definition; §5.4c: `allow_network`/`allow_acquiring` remove entries, listed in `keel audit`; method name format given; §5.4a states the `(*http.Client).Do` exception. |
| R9-15 | §5.4 rule 9 reworded (`keel.DeadLetter[T]`, predeclared, `error`, time types); `rt` validates `DeadLetter` generically; examples use `cards.Receipt` and `events.StockChanged`; K0112 widened. |
| R9-16 | §6: new `setup` hook, run after construction and before serving; five start steps; K0601 covers it. |
| R9-17 | §5.5.2: the lock hashes the `pure` functions behind invariants and contracts; rename row is honest (compatible, data lost, K0319 warning); rows for adding/removing remote methods and `ctx`; ports keyed by consumer; a lock hash catches hand edits (K0321). |
| R9-18 | §7.1: implicit `requires r != nil` only where the method's own contract reads through the receiver; the invariant check skips a nil receiver. |
| R9-19 | B.1, B.2, B.3, §4.1: `invariant` in place 2; parentheses for a composite literal before the body; `ContractExpr` defined; `identifier {` in declaration files is always `Settings`; `default` and `max` in §4.1; the type-group restriction in B.3. |
| R9-20 | App. E: K0008, K0120, K0121, K0319, K0320, K0321, K0520, K0521 added; K0112, K0305, K0308, K0504, K0601, K0602 widened; K0318 has a body rule; K0113, K0502, K0509 cite the right sections. |
| R9-21 | §5.4 rule 12: a leaf's package-level initializers cannot reach a port call or a publish; §5.4a: "at startup" defined. |
| R9-22 | §8: unbound ports fail the test with a message; `keel test <library>`; `Published` panics under `t.Parallel`. |
| R9-23 | §6: panics in `setup`/`startup` are errors; exit 0 when all startups return `nil`; leaf settings come only from the container list; transports are created in step 1. |
| R9-24 | §5.3: leaf names are unique in the system (K0121). |
| R9-25 | §5.3: flag names that give one environment variable are K0602; each container reads only its own flags. |
| R9-26 | §6: K0702 is Keel's own; `keel test --race=false`, listed by `keel audit`. |
| R9-27 | §11.2: the fingerprint is the normalized token sequence; length pruning; in-leaf duplicates in the per-leaf build, all others and §5.4a/§5.4c in the holistic check; the §11.2 table updated. |
| R9-28 | Primer: `setup`, unique leaf names, K0511, the `requires`-fact condition, message type rules, remote error codes. About 2,400 tokens. |
| R9-29 | §7.1: invariant fields are unqualified and shadow; named results are in scope in `ensures`; §7.2: generic invariants checked per instantiation; "optional field" replaced by a pointer field with `P != nil =>`. |
