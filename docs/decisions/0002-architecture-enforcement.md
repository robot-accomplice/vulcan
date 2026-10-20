# Decision 2: architecture enforcement

Status: accepted, 2026-10-20 (option C). Applied in spec §5.4 rule 13, §5.4d, Appendices B, D,
and E, the primer, and the README. §5.4d numbers the paths 1 to 8: path 7 here (`linkname`)
became rule 13, so paths 8 and 9 here are paths 7 and 8 there. Milestone 0, item 2 of the
[development plan](../development-plan.md). Answers [audit 2](../../experiments/audit-2.md), H7.

## Question

Which architecture properties does the compiler prove, and what are the limits of each? Which
paths around them are rejected, which are reported, and which are out of scope? What may the docs
claim?

## Current rule

§5.4, §5.4a, §5.4b, and §5.4c define the rules. Each one is either a compile error or an entry
in `keel audit`. The summary text claims more than the rules give. For example, §5.4b says "the
compiler rejects network access in any other leaf, so the reviewer does not have to find it". But
the same section lets an entry leaf dial any host, and §5.4c does not follow function values.

## Paths around the rules

Each path below compiles today, and no error reports it.

1. **Function and interface values through ports.** Port signatures can carry function types and
   interface types. §5.4 rule 9 allows standard library types, such as `io.Reader`, and it does not
   forbid type literals, such as `func(OrderID)`. Leaf A passes a
   closure to leaf B. B then runs A's code with no wire from B to A, so the "no cycles" rule
   (rule 8) holds for wires but not for calls. A gateway can return a `func(host string, b []byte)
   error`. A domain leaf can then choose a network target through it. §5.4c does not follow a call
   through a function value beyond two conversion steps.
2. **Shared mutable state in libraries.** A library `var` cannot be reassigned, but it can change
   through a method or a reference (§5.4a rule 1). A `var Events = make(chan Event, 100)` or a
   `sync.Map` in a library is a channel between leaves that no port shows. `keel audit` lists it.
3. **Global state in imported Go packages.** Two leaves that import the same Go package share its
   package state: `os.Setenv`, `http.DefaultServeMux`, `expvar`, a metrics registry, a logger. The
   spec does not mention this path.
4. **Entry leaves can dial.** The compiler "does not tell serving from dialing" (§5.4b). An entry
   leaf can reach every network function.
5. **Gateway targets are not checked.** A gateway for Stripe can call any host (§5.4b rule 3).
6. **Escape hatches.** `syscall`, `unsafe`, `reflect` calls, and cgo can reach a resource that the
   analysis does not see (§5.4c, Limits). `keel audit` lists them.
7. **`//go:linkname`.** A leaf can call another leaf's unexported function with a `linkname`
   directive and `import _ "unsafe"`. No identifier names the other leaf, so rule 1 (K0101) does
   not see it. A test program confirmed this on Go 1.26.1. The spec does not mention it.
8. **Shared resources outside the program.** Two leaves can acquire the same file, environment
   variable, or table. Rule 2 of §5.4c makes each acquisition visible. It does not make leaves
   independent.
9. **Configuration overrides.** `allow_network` and `allow_acquiring` remove functions from the
   lists (§5.4c). `keel audit` lists each change.

## Guarantee matrix

**Proved** means that the compiler checks the property on every build, with no listed gap.
**Reported** means that `keel audit` lists each gap. **Out of scope** means that no static check
can decide it, and the docs must not claim it.

| # | Property | Status in ordinary mode | Gaps | Strict mode |
|---|---|---|---|---|
| G1 | A leaf's code and boundary never name another leaf (rules 1, 7) | Proved, after K0122 | Path 7, closed in both modes by K0122 | Proved |
| G2 | Each required port and subscribed topic is wired once (rule 4); the method sets match (rule 5) | Proved | None | Proved |
| G3 | The wires of a container have no cycles (rule 8) | Proved for wires | Path 1: calls can go against a wire through function and interface values | Proved for calls too (K0123) |
| G4 | Types that cross a boundary come from libraries or the standard library (rule 9) | Proved | None | Proved |
| G5 | Only gateway and entry leaves reach a network function (§5.4c rule 1) | Proved for static reach | Paths 1, 4, 6, 9 | Paths 1 and 6 rejected (K0123, K0124); entry leaves cannot dial (K0125); path 9 still allowed and reported |
| G6 | Only leaves reach an acquiring function (§5.4c rule 2) | Proved for static reach | Paths 6, 9 | Path 6 rejected (K0124); path 9 allowed and reported |
| G7 | Leaves share no state in program memory except through ports and topics | Reported | Paths 2, 3 | Path 2 rejected (K0210); path 3 reported, with a list of known mutators rejected (K0211) |
| G8 | Port contracts hold at every call through the port, fakes included (§5.3, §8) | Proved, under decision 1 | Unproved implications (K0109) are listed | Same |
| G9 | Each external system has one gateway in each container (§5.4b rule 1) | Proved | None | Proved |
| G10 | Containers talk only through topics and remote ports (rule 3) | Proved for wires | Paths 4, 5: a gateway or an entry leaf can call another container by hand | Path 4 rejected (K0125); path 5 out of scope |
| G11 | A gateway talks only to its declared external system | Out of scope | Path 5. The target is a runtime value, so no static check can decide it | Out of scope |
| G12 | Leaves do not share resources outside the program | Out of scope | Path 8. The compiler makes each acquisition visible, not exclusive | Out of scope |

## Options

**A. Ordinary mode only, with honest claims.** Add K0122 (`linkname`). Rewrite the summary text of
§1.1, §5.4b, and the README to match the matrix. Report every gap in `keel audit`. Cost: low. The
language promises less.

**B. Strict mode only.** Reject every gap that a static check can find. This removes callbacks
across ports, mutable library state, and direct `reflect` calls from every project. These are
ordinary Go, so the rule breaks the "Go ergonomics" principle for projects that do not need the
stronger promise.

**C. Ordinary mode by default, and strict mode on request (recommended).** Ordinary mode is option
A. Strict mode turns each gap that a static check can find into an error. A project chooses the
mode, and it can exempt single leaves. This follows design rule 1 (mechanism, default, override).

## Recommendation: option C

### Rules in both modes

1. **No `linkname`.** A leaf or a library that contains a `//go:linkname` directive is a compile
   error (K0122). An imported Go package can still use it, as the standard library does.
2. **The summary claims follow the matrix.** A doc, a diagram, or a tool output that states an
   architecture property states it as the matrix does. `keel diagram` draws only wires as
   dependencies. `keel doc` names each reported gap of a leaf.
3. **One report of the gaps.** `keel audit` has one section, "Unchecked paths", with one entry for
   each gap of paths 1 to 9 that the compiler finds. `keel check` prints the number of entries.
   `--json` gives each entry with its path number, its location, and its leaf.

### Strict mode

`config { strict: true }` in `system.keel` turns it on for the whole system. `strict_exempt:
["Web", "Admin"]` keeps the named leaves in ordinary mode. `keel audit` lists each exempt leaf, and
`keel doc` shows the mode of each leaf. Strict mode adds these errors:

1. **K0123: behavior across a boundary.** A port signature cannot carry a function type, a
   channel type, or an interface type. The exceptions are `error` and `context.Context`, and the
   interfaces named in `config { strict_interfaces: [...] }`, which `keel audit` lists. Thus a call
   through a port never runs the caller's code, and G3 holds for calls.
2. **K0124: escape hatches.** Code in a leaf that is not exempt, or in a library, cannot import
   `unsafe` or `syscall`, use cgo, or call `reflect.Value.Call`, `reflect.Value.CallSlice`,
   `reflect.Value.Method`, or `reflect.Value.MethodByName`. Imported Go packages are not checked
   for this. Their reach is still computed, and calls through `reflect` inside them are reported.
3. **K0125: entry leaves serve, gateways dial.** Appendix D splits the network functions into
   *serve* functions (`Listen*`, `Serve*`, `ListenAndServe*`, `Accept`) and *dial* functions (all
   others). An entry leaf can reach a serve function. A leaf that reaches a dial function must be a
   gateway. A leaf can be both: `gateway for Stripe` and `entry for Stripe` in one leaf.
4. **K0210: immutable library state.** The type of a library `var` must hold no mutable
   reference: no pointer, slice, map, channel, function, or interface. The exceptions are `error`
   values and the types in `config { strict_shared: [...] }`, for example `*regexp.Regexp`, which
   `keel audit` lists. A `const`-like table goes in an array or a struct.
5. **K0211: known global mutators.** Code in a leaf that is not exempt, or in a library, cannot
   reach a function on the Keel list of global mutators of the standard library. The list includes
   `os.Setenv`, `os.Unsetenv`, `os.Clearenv`, `http.Handle`, `http.HandleFunc`, `expvar.Publish`,
   `flag.Parse`, `log.SetOutput`, `log.SetFlags`, `log.SetPrefix`, `log.SetDefault`, and
   `slog.SetDefault`. A project adds others with `config { mutators: [...] }`. Global state in
   other imported packages stays reported (path 3), because the compiler cannot decide which
   writes are shared.

### What each mode lets the docs claim

- **Ordinary mode:** "Each dependency between leaves in the source goes through a declared port,
  topic, or library. Only gateway and entry leaves reach a network function through static calls.
  `keel audit` lists each path that these checks do not cover."
- **Strict mode:** "Each dependency between leaves, in the source and in calls, goes through a
  declared port, topic, or library. Only gateways dial, and only entry leaves serve. No leaf
  shares program memory with another leaf except through a port or a topic, apart from the
  imported-package state that `keel audit` lists. Exempt leaves are listed."
- **Neither mode** claims where a gateway connects (G11), or that leaves share no files, tables,
  or environment (G12).

## Examples

| Code | Ordinary | Strict |
|---|---|---|
| `//go:linkname charge shop/api/billing.charge` in leaf Orders | K0122 | K0122 |
| `needs port Hooks { func OnPaid(f func(OrderID)) }` | Reported (path 1) | K0123 |
| `needs port Files { func Open(name string) (io.ReadCloser, error) }` | Reported (path 1) | K0123, unless `io.ReadCloser` is in `strict_interfaces` |
| `var Events = make(chan Event, 100)` in a library | Reported (path 2) | K0210 |
| `var Pattern = regexp.MustCompile("...")` in a library | Reported (path 2) | K0210, unless `*regexp.Regexp` is in `strict_shared` |
| Entry leaf Web calls `http.Get` | Allowed, listed with its network functions | K0125, unless Web is also a gateway |
| Leaf Billing calls `os.Setenv` | Reported (path 3) | K0211 |
| Leaf Billing imports `unsafe` | Reported (path 6) | K0124 |
| Gateway StripeGateway calls `http.Post("https://example.org", ...)` | Allowed (G11 out of scope) | Allowed |

## Spec changes on acceptance

- §1.1 and §5.4b: replace the summary claims with the claims of the matrix. Remove "so the
  reviewer does not have to find it".
- §5.4 (new rule 13): no `linkname` (K0122). New §5.4d: the guarantee matrix, the two modes, and
  the strict rules.
- §5.4a rule 1, §5.4b, §5.4c: point each listed gap to its path in §5.4d.
- Appendix B: the config keys `strict`, `strict_exempt`, `strict_interfaces`, `strict_shared`,
  and `mutators`.
- Appendix D: split the network functions into serve and dial, and add the list of global
  mutators.
- Appendix E: K0122 to K0125, K0210, K0211.
- §10 `keel audit`: the "Unchecked paths" section.
- `docs/guide.md`: one sentence on the two modes, and the `linkname` rule.
- README: the claim of "What Vulcan adds to Go", architecture row.

## Open points

- **The strict rules have a cost that only real code shows.** K0123 and K0210 may reject too
  much. The application of milestone 3 should run in strict mode, and record each exemption and
  each `strict_interfaces` or `strict_shared` entry that it needs.
- **Path 3 needs implementation evidence.** The compiler could compute, for each imported
  function, whether it writes package state of another package. Whether that report is precise
  enough to be useful is a question for milestone 2.
- **Serve functions must not reach dial functions.** K0125 assumes that, for example,
  `(*http.Server).Serve` reaches no dial function through static calls. Milestone 2 must check
  this on the real call graph of `net/http`.
- **Per-container strict mode.** Strict mode could be set per container instead of per system.
  This draft keeps one switch and per-leaf exemptions, which is simpler, and can grow later.
