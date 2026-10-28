# Round 2, step 0: the desk test

Status: done 2026-10-28, by a session that did not build Vulcan. It answers the question of
[plan-r2.md](plan-r2.md), section 2: which of Vulcan's checks can a Go team not get from
`go/analysis` analyzers, build rules, and checks in the Go code?

## Summary

**Gate 0: every check is "Same in Go".** No check of the PoC compiler is "Harder in Go" or
"Not in Go". So round 2 cannot show a difference by enforcement, and the "V only" group of
plan-r2 section 4 is empty. The round should not run as planned.

The reason is simple. The PoC compiler is itself an analyzer of Go code. It turns `.vulcan`
files into Go, loads them with `go/packages`, and runs every rule on `go/types` data
(`internal/check`). The only inputs that Go code does not hold are the declaration files
(`system.vulcan`, `component.vulcan`, `library.vulcan`) and the contract clauses. A Go team
holds the same facts in Go: a manifest of the architecture, the consumer's interfaces, a
settings struct, and calls to a small contract package. Analyzers can make each of these
mandatory, so that an agent cannot leave one out without a diagnostic. Then each Vulcan rule
is the same AST or type walk, on the same facts.

In three places a Go team can do **more** than the PoC:

1. The PoC's network rule (K0205, and K0125, K0124, K0211 in strict mode) sees only direct,
   named uses in the project's own code. A call through an interface, or into another module,
   passes. A Go analyzer can use `go/callgraph` (CHA or VTA) over the whole program.
2. In Vulcan the agent must edit `system.vulcan` and `component.vulcan` to do ordinary work, and
   the same files say which leaf is a gateway. In Go the manifest can be configuration that the
   agent does not change (decision 6 already fails a run that changes the check configuration).
3. The PoC has holes with no diagnostic (section 4). The worst one: a code file whose name starts
   with `vulcan_` skips every code rule. I checked it on a copy of Stockroom-lite.

What is left of Vulcan's case is not a guarantee. It is how the rules come: one model, checks
that cover a whole family of rules without an analyzer for each one, declarations that the
language makes the agent write, and the primer. Those claims need their own test (section 6).

## 1. The Go baseline used below

The Go side is the Go app in `poc/app/go`, plus these conventions, each one checked:

- **Manifest.** A Go file (or a protected config file) that lists each package with its kind:
  leaf, gateway for an external, entry for a person or an external, library, or main. A Go file
  is type-checked, so a typo is a compile error.
- **Ports.** A leaf declares each dependency as an interface in its own `ports.go`, and gets it
  as a parameter of its constructor. A constructor takes only these interfaces and one settings
  struct. This is how `poc/app/go` is written today (`billing.NewLedger(card Card, store Store)`).
- **Contracts.** Calls into a small package: `contract.Requires(cond)` as the first statements,
  `defer contract.Ensures(...)` for postconditions, a method `invariant() bool` and a
  `defer x.check()` in each method. A port's contract is a wrapper type next to the port
  (`checkedCard`), and the constructor wraps each port that it gets, so fakes are checked too.
- **One analyzer driver** (`golint-rules`, a `multichecker`) that does not honor `//nolint`. It
  runs with the build tags of the real build, and rejects files that the build ignores in
  project packages (`Pass.IgnoredFiles`).

The analyzers that the table names:

| Name | What it inspects | What it rejects |
|---|---|---|
| A1 layout | The package list (`go/packages`) and the manifest | A package that is not in the manifest, a package inside a leaf, a package name that is not its directory, a gateway for an external that is not declared, two gateways for one external |
| A2 imports | The imports of each project package | A leaf or library that imports a leaf; an import of a forbidden package (`unsafe`, `syscall`, `C`) |
| A3 ports | `ports.go`, constructors, and struct fields of each leaf | An interface outside `ports.go`; a constructor parameter that is not a port of this leaf or the settings struct; a port signature with a type from another module, or (strict) a function, channel, or interface type; a write to a port field outside the constructor, or its address; `init` in a leaf |
| A4 main | The main package | Any call other than leaf constructors, the lifecycle calls, and the flag reader; a constructor called twice with other settings; a `nil` port; a settings value that is not a constant or a flag; a constant in a field tagged `secret`; a missing settings field (as `exhaustruct` does) |
| A5 effects | Each use of a function, by `types.Func.FullName`, or by call graph | A network function outside gateways and entry leaves; a dial in an entry leaf; an acquiring function in a library; `reflect` call functions; a global mutator |
| A6 library state | Package-level vars of libraries | A write after init; (strict) a type that holds a reference (`gochecknoglobals` is a stricter form) |
| A7 contracts | Calls of the contract package | A contract that is not first, or that calls a function other than `len`, `cap`, or a conversion; a call with constant arguments that makes a `Requires` false; a port wrapper that names a leaf's own identifiers |
| A8 invariants | Types with an `invariant` method | A method without the deferred check; a struct literal of the type outside its constructor; a write to a field, or its address, outside the type's methods; a zero value that breaks the invariant |

These are about 8 analyzers. The PoC's `internal/check` is about 2,400 lines; the analyzers would
be of the same order. That is a cost, not a missing guarantee.

## 2. The table

Every code that the PoC compiler can give: the K-codes in `internal/check` (`strict.go`
included), `internal/project`, `internal/driver`, `internal/lower`, `internal/decl`, and
`cmd/vulcan`. Spec Appendix E lists more codes; the PoC does not give them (K0009 rejects the
constructs), so they are not in the table. "Artifact" means that the check exists because
Vulcan has its own syntax: Go has no such mistake, or the Go compiler already rejects it.

| Code | What the PoC really checks | Label | Go mechanism |
|---|---|---|---|
| K0001 | Syntax of declarations and contracts | Same (artifact) | The Go parser; the manifest is Go |
| K0002 | Code outside a leaf or library, a leaf inside a leaf, a `.go` file | Same | A1 |
| K0003 | Directories and `system.vulcan` disagree; library outside `lib/` | Same | A1 compares the manifest with the package list |
| K0004 | Package clause is not the directory; external test package | Same | A1; plain `go vet` style check of `Pkg.Name()` |
| K0005 | A name starts with `vulcan_` | Same (artifact) | Not needed |
| K0006 | Unknown `config` key | Same (artifact) | The manifest is a typed struct: the compiler |
| K0008 | Go syntax or type error | Same | It is the Go compiler |
| K0009 | Construct outside the PoC | Same (artifact) | Not needed |
| K0209 | Dependency not in `go.mod` or `go.sum` | Same | `-mod=readonly`, native Go |
| K0101 | A leaf imports another leaf (imports only) | Same | A2. In Go a package can name another package only through an import, so the import check is complete |
| K0104, K0105 | A port is wired never, or twice | Same | A port is a constructor parameter: the compiler wants one argument, and only one. A4 rejects `nil` |
| K0107 | A wire names a missing or unexported constructor or type | Same | The Go compiler, in `main` |
| K0108 | The supplier or a test fake does not have the port's methods | Same | The Go compiler: an argument to an interface parameter |
| K0110 | A port outside its leaf, twice, with an unexported method or a clashing name; a boundary or `system.vulcan` imports a leaf | Same | A3, A2; an unexported interface method cannot be met from another package, so the compiler rejects the wire |
| K0111 | A cycle in the wires | Same | Constructor injection cannot build a cycle without a write after construction, which A3 and A4 reject |
| K0112 | A port signature uses a leaf-private type or a type from another module | Same | Leaf-private: the compiler plus A2 (the provider cannot name it). Other module: A3 |
| K0113 | A wire names a group | Same (artifact) | Not needed |
| K0116 | Code assigns to a port or takes its address, except `vulcan.Bind` | Same | A3 on port fields |
| K0117 | `init` in a leaf; a package var initializer that reaches a port | Same | A3. With ports as fields, an initializer cannot reach a port |
| K0118 | Code imports the generated-code runtime | Same (artifact) | A2 |
| K0120 | A port contract names an identifier of its own leaf | Same | A7 on the wrapper |
| K0121 | Two leaves or nodes with one name | Same (artifact) | Import paths are unique; A1 for manifest names |
| K0122 | `//go:linkname` in a leaf or library | Same | A2 reads the comments of each file |
| K0123 | Strict: a function, channel, or interface type in a port signature | Same | A3, the same walk as `behavior()` in `strict.go` |
| K0124 | Strict: imports `unsafe`, `syscall`, cgo; calls a `reflect` call function | Same | A2 and A5 (`depguard` and `forbidigo` do it today) |
| K0125 | Strict: an entry leaf uses a dial function | Same | A5 with the manifest's kinds |
| K0201 | A library declares a port | Same (artifact) | A3: only leaves have `ports.go`; A4 wires only leaves |
| K0202 | A library var is assigned after startup | Same | A6 |
| K0203 | A library uses an acquiring function | Same | A5 |
| K0204 | A library imports a leaf; library import cycles | Same | A2; the Go compiler rejects cycles |
| K0205 | A non-gateway, non-entry leaf or a library uses a network function (direct uses only) | Same; Go can do more | A5. With `go/callgraph` it can follow interfaces and other modules, which the PoC does not |
| K0206 | `gateway for` or `entry for` names an undeclared node | Same | A1 on the manifest |
| K0207 | Two gateways for one external in one container | Same | A1 on the manifest |
| K0210 | Strict: a library var's type holds a reference | Same | A6, the same walk as `holdsReference()` |
| K0211 | Strict: a call to a known global mutator | Same | A5 with the same list |
| K0501 | A call with constant arguments makes a `requires` false | Same | A7, the same substitution and `types.Eval` |
| K0502, K0510, K0511 | `result`, `result.N`, and `err` are used right | Same (artifact) | Go names its results; no special names |
| K0503 | A contract calls a function other than `len`, `cap`, or a conversion; a function literal or a receive | Same | A7 on the arguments of the contract calls |
| K0506 | The zero value breaks an invariant (fields of basic types) | Same | A8, or one test: `var z T; z.invariant()` |
| K0507, K0508 | A field of a struct with an invariant is written, or its address taken, outside its methods | Same | A8, the same walk as `fieldWrites()` |
| K0509 | `old(x)` of a type that holds a reference | Same | A7 on the `old` copies |
| K0512, K0513, K0514 | A type assertion in a contract; an invariant on a non-struct; a contract with no body | Same (artifact) | A7, or not needed |
| K0601 | `setup` or `startup` has the wrong signature | Same (artifact) | A lifecycle interface: the compiler, in `main` |
| K0602 | A flag with two types or defaults; two flags with one environment variable | Same | A4, or a unit test of the flag table |
| K0603 | Settings for a leaf with none; a missing or unknown settings field | Same | Unknown: the compiler. Missing: A4 (`exhaustruct`) |
| K0604 | A constructor's shape: one settings struct, results `T` or `(T, error)` | Same | A3 |
| K0605 | A literal or flag default in a field tagged `secret` | Same | A4 |
| K0606 | A setting is not a constant of its field's type | Same | A4 |
| K0607 (W) | One constructor wired with two settings | Same | A4 |
| K0705 (W) | Tests ran without the race detector | Same | It is Go |
| K0706 | `go vet` finding | Same | It is `go vet` |

## 3. Details

### 3.1 Why "Same" holds for the architecture rules

The plan's "Harder" label needs a convention that the agent can break without a diagnostic. The
Go baseline has four conventions: the manifest, `ports.go`, the constructor shape, and the
contract calls. Each one is checked:

- **A new package that is not in the manifest** fails A1. In Vulcan, a new directory with code
  that has no `component.vulcan` fails K0002. Same.
- **A dependency that is not a port** (a `*sql.DB`, a function, a concrete type passed to a
  constructor) fails A3, because a constructor takes only its own ports and one settings struct.
  In Vulcan the constructor shape (K0604) and constant settings (K0606) give the same result.
- **An interface hidden outside `ports.go`** fails A3.
- **Code in `main` that goes around the leaves** fails A4. `main` is the one place where Go is
  more open than Vulcan, because Vulcan generates it. A4 closes it; `google/wire` is another way.
- **A suppression comment** has no effect, because the driver does not honor it. Decision 6 also
  fails a run that adds one.
- **A file hidden by a build tag** fails the driver, which rejects ignored files in project
  packages.

The agent can still change the manifest. In Vulcan it can still change `system.vulcan` and
`component.vulcan`: it can add an `external` and a `gateway for` it, and the leaf can then dial
any target, with no diagnostic (K0206 checks only that the external is declared; G11 is out of
scope by the spec). The difference favors Go: a Go team can make the manifest configuration that
the agent may not change, while a Vulcan agent must edit these files to add a wire.

### 3.2 The rules that look like Vulcan's own

- **No behavior across a port (K0123), immutable library state (K0210).** Both are type walks
  over declarations that Go code holds: the port interfaces and the library vars. The PoC's
  `behavior()` and `holdsReference()` work on `go/types` and port to an analyzer as they are.
- **Ports bound once (K0116, K0117).** In Vulcan a port is a package-level name, so these rules
  are needed. In idiomatic Go a port is a field set by the constructor, and A3 rejects any other
  write. The test fake goes through the same constructor, so it is wrapped and checked too.
- **Wiring (K0104, K0105, K0108, K0111).** In Go, the compiler does most of it: a missing
  argument, an extra argument, and a wrong method set are type errors. Cycles cannot be built.
- **Every port wired, startup order, shutdown.** Go needs a hand-written or `wire`-generated
  `main` for this. That is code to write, but not a missing check.

### 3.3 Contracts

Vulcan's contracts (`requires`, `ensures`, `invariant`) run in every build. Go asserts also run
in every build: there is no switch to remove them unless someone writes one, and A7 can reject a
contract call under a condition. What the PoC adds, and what Go does:

| PoC behavior | Go |
|---|---|
| `requires` is checked at entry | `contract.Requires` as the first statements; A7 checks the place |
| `ensures` is checked at each return, before deferred calls | `defer contract.Ensures(...)` with named results. It runs after the other deferred calls, not before. This is a small difference, and the PoC itself does not build the second check point of §7.3 |
| `old(x)` | A local copy at entry; A7 rejects a copy of a reference type (K0509) |
| Invariants after each outer method call and each struct literal | A8: a deferred check in each method; no literal outside the constructor |
| Port contracts checked for every provider and every fake | A wrapper next to the port, applied in the constructor; A3 checks that each port field is wrapped |
| Exact integer arithmetic | Not in the PoC either: status.md says C08 is not met. Contract arithmetic is Go arithmetic in both |
| K0501, K0506 at compile time | A7 and A8, the same constant evaluation |

An agent can weaken a contract in both languages: delete the clause, or delete the assert. A
grader sees both in the diff. A contract generator from comments, such as `gocontracts`, would
make the Go form shorter, but it is not needed for the guarantee.

### 3.4 The generated startup and wiring

Vulcan generates `main` from `system.vulcan`: construction in order, flags, settings, `setup`,
`startup`, shutdown. Go can get the same in two ways: `google/wire` (generated injector, with
errors for a missing or duplicate provider and for cycles), or a hand-written `main` that A4
holds to a fixed shape. `poc/app/go/cmd/stockroom` is such a `main` today, 100 lines. What
Vulcan gives is less code to write and one file that shows every wire. That is a cost and
readability claim, not a guarantee.

### 3.5 The primer (`vulcan guide`)

The primer ([guide.md](guide.md), 181 lines) teaches one model: leaves, ports, wires, gateways,
contracts, and the commands. Go can give an agent the same text as `RULES.md` or `AGENTS.md`,
plus the analyzers' documentation and messages with the same fix hints. Arm GL of plan-r2 has
this already. Whether one model that the language enforces is easier for an agent to follow than
the same rules as a list is a real question, but no compile check answers it. It needs its own
test (section 6).

## 4. Holes in the PoC that Go would not have

These are paths around Vulcan's own checks, with no diagnostic. They matter for round 2,
because an arm V agent can take them, and because they make "V only" even less likely.

| # | Path | How I found it | Effect |
|---|---|---|---|
| H1 | A code file named `vulcan_*.vulcan` skips every code rule | Run on a copy of Stockroom-lite: `api/orders/vulcan_x.vulcan` imports `api/billing`, imports `unsafe`, declares `init`, calls `os.Setenv` and `http.Get`. `vulcan check` exits 0. The same file as `x.vulcan` gives K0124, K0101, K0117, K0211, K0205 | `Checker.Generated` treats each file whose name starts with `vulcan_` as generated (`internal/check/load.go:221`). K0005 reserves the prefix for identifiers, not for file names |
| H2 | `strict_exempt: ["Orders"]` in `config` turns strict mode off for a leaf | Run: with it, `import "unsafe"` in Orders passes | The PoC has no `vulcan audit`, so nothing lists the exempt leaf. Decision 6 fails such a run, but only if the grader looks |
| H3 | A network call through an interface | Run: `http.DefaultTransport.RoundTrip(req)`, and `http.DefaultClient` stored in a var of a local interface type and called, both pass in a non-gateway leaf | K0205 matches only named uses of listed functions (`network.go`); status.md and audit P2 (L7) say so |
| H4 | A network call inside another module | From the code and status.md (N02 is not met) | An SDK that dials is not followed |
| H5 | A new `external` and `gateway for` it | From the code (K0206, K0207) | Any leaf can become a gateway by editing two declaration files |

H1 should be fixed before any run of arm V: reserve the file name prefix too. A Go analyzer has
none of H1 to H4, because it reads every file that the build compiles and can use a call graph.

## 5. Gate 0 verdict

**Every check is "Same in Go".** The list of "Harder" and "Not in Go" checks is empty, so no rule
can be enforced by V and not by GL, and plan-r2 section 4 cannot reach 6 "V only" rules. Following
the plan, round 2 does not run as planned, and the owner decides what Vulcan's case rests on.

Rules a real team would write, and where they fall. None of them is "V only":

| Rule in plain words | Group |
|---|---|
| Only the payments part talks to the card processor | Both (K0205 / A5); Go with a call graph is stronger |
| The orders part never imports the billing part; it goes through an interface it owns | Both (K0101 / A2) |
| Shared types live in `lib/shop`; no package-private type crosses a boundary | Both (K0112 / A3, compiler) |
| A dependency is never a callback or a channel | Both (K0123 / A3) |
| Shared library code holds no mutable global state | Both (K0202, K0210 / A6) |
| Nobody changes the environment or the default logger outside `main` | Both (K0211 / A5) |
| No `unsafe`, no `reflect` calls in product code | Both (K0124 / A2, A5) |
| Reserved stock never exceeds stock on hand, after every operation | Both (invariant / A8 and asserts) |
| A refund is more than 0 and never more than the amount not yet refunded | Both (`requires` / asserts) |
| The card processor's key is never a literal in the code | Both (K0605 / A4) |
| The payments part calls only the card processor's host | Neither (G11) |
| Two parts never write the same table | Neither (G12) |
| Errors are wrapped with the order id | Neither (prose) |

The labeler of step 3 should still label each rule, but should expect the "V only" column to stay
empty for anything that a compile check decides.

## 6. What a test of Vulcan's case could measure instead

This is for the owner's decision, not a plan. Vulcan's case, after this desk test, rests on
claims that are not about which checks exist:

1. **Coverage without per-rule work.** Vulcan's checks are generic: they enforce a family of
   rules that nobody wrote an analyzer for. Arm GL of plan-r2 gets one analyzer per rule, written
   from the rules. A fairer test of this claim gives GL a *generic* Go linter of the same model
   (A1 to A8 above, written before the rules), and then asks whether rules that the team did not
   foresee are kept. But if that linter does as well, the result is "build the linter", as
   plan-r2 section 1 says.
2. **The primer and one model.** Does an agent that learns one enforced model make fewer
   mistakes, or reach a passing check sooner, than one that reads the same rules as a list? This
   can be tested with the same linter in both arms and only the documents changed.
3. **Less code.** Generated `main`, wrappers, and contract checks are code that the Go agent
   writes and the Vulcan agent does not. This is a cost measure: tokens and turns on ordinary
   tasks.

Each of these is cheaper to test against a Go linter of Vulcan's model than against the
language, and a Go linter would be the product if they hold.
