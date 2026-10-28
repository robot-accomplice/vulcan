# vulcan-lint: Vulcan's model as a Go linter (arm L of round 2)

`vulcan-lint` is arm L of [plan-r2.md](../../docs/poc/plan-r2.md) (section 2, step 2 of section
8): Vulcan's model of architecture and contracts, enforced on plain Go by
`golang.org/x/tools/go/analysis` analyzers, with no new language. It is the analyzers A1 to A8 of
the [desk test](../../docs/poc/desk-test-r2.md) (section 1) in one `multichecker` command. It is
generic: it enforces the model (kinds, consumer-owned ports, gateways for effects, library state,
contracts, invariants, main-only wiring), not team rules. It was written before the hidden rules of
round 2 exist, by a session that did not build Vulcan, from the spec, the desk test, and the code.

```bash
go build -o vulcan-lint ./poc/golint/cmd/vulcan-lint   # or: harness tools [-linux]
cd poc/app/golint && go vet ./... && vulcan-lint ./... && go test ./...
```

Each diagnostic reads `file:line:col: VL0101 message`, and the message says how to fix it. The
command honors no `//nolint` or other suppression (a `multichecker` has none), runs on test files
too (`-test` is on by default), and rejects a project file that the build ignores
(`Pass.IgnoredFiles`, VL0901). It exits 3 when it reports a diagnostic, as every
`go/analysis` driver does.

| Path | What |
|---|---|
| `cmd/vulcan-lint` | The command |
| `model` | Reads the manifest, places a package in it, and the helpers of the analyzers |
| `layout`, `imports`, `ports`, `wiring`, `effects`, `libstate`, `contracts`, `invariants` | A1 to A8 |
| `conformance` | The port of the conformance cases (`gen_cases.py` writes `testdata/`), and checks of the workspace |
| `internal/lintest` | Builds the test modules |
| [`../app/golint`](../app/golint) | Arm L's workspace: Stockroom-lite with the manifest, ports, contracts, `PROTECTED`, and `PRIMER.md` |

## The model in Go

- **The manifest** is the Go package `arch` at the module root: `arch/schema.go` (the kinds, the
  same in every project) and `arch/arch.go` (`const Customer Person = ...`, `var Manifest =
  System{Strict: true, Parts: []Part{Main("cmd/stockroom"), Entry("api/web", Customer),
  Leaf("api/orders"), Gateway("api/store", Database), Library("lib/shop"), ...}}`). The build
  type-checks it (a name that is not declared is a compile error); the analyzers read its syntax.
  It also holds the linter's settings: `StrictExempt`, `StrictInterfaces`, `StrictShared`,
  `Mutators`, as `config` of `system.vulcan` does. `PROTECTED` lists `arch/*.go` and
  `lib/contract/*.go`.
- **Ports** are exported interfaces in the leaf's `ports.go`. A port with a contract has a
  wrapper `type checkedX struct{ X }` there, whose methods are the contract and one forwarding
  call. A **constructor** (an exported function that returns `T` or `(T, error)` of its leaf) takes
  only ports and, last, one settings struct, and stores a port that has a wrapper wrapped, so a
  fake in a test is checked too.
- **Main** calls only constructors, lifecycle methods (`func (ctx context.Context[, s Settings])
  error`), the standard library, and its own functions.
- **Contracts** are calls of the package `<module>/lib/contract`, which the workspace holds:
  `contract.Requires(cond)`, `x := contract.Old(v)`, `defer contract.Ensures(func() bool { return
  cond })`, and for a type with `func (x T) Invariant() bool`, `defer contract.Keep(recv)` in each
  pointer method and `contract.Valid(T{...})` on each value that the code builds. They are the first
  statements of a function.

## The analyzers and the codes

Each code keeps the digits of the Vulcan K-code that the desk test pairs it with (VL0101 is
K0101). The codes in VL09xx have no K-code: they check the Go conventions that stand in for
Vulcan's syntax, or a hole that Go closes.

| Analyzer | Code | K-code | What it rejects |
|---|---|---|---|
| A1 `layout` | VL0002 | K0002 | A package that the manifest does not list, or a package (or part) inside a leaf |
| | VL0003 | K0003 | A part with no package; a library outside `lib/`, or another kind in it; `Main` not `package main`, or `package main` not `Main`; the manifest as a part |
| | VL0004 | K0004 | A package name that is not its directory; an external test package |
| | VL0110 | K0110 | The manifest imports a package (as `system.vulcan` importing a leaf) |
| | VL0121 | K0121 | A directory listed twice |
| | VL0206 | K0206 | A gateway or an entry for a name that the manifest does not declare (in practice a compile error first) |
| | VL0207 | K0207 | Two gateways for one external |
| | VL0901 | — | A file of a project package that the build ignores |
| | VL0902 | — | A manifest that is missing or not in its literal form; `StrictExempt` naming no leaf |
| A2 `imports` | VL0101 | K0101 | A leaf imports another leaf |
| | VL0118 | K0118 | Code imports the manifest (Vulcan: the generated-code runtime) |
| | VL0122 | K0122 | `//go:linkname` in a leaf or a library |
| | VL0124 | K0124 | Strict: `unsafe`, `syscall`, or cgo in a leaf or a library |
| | VL0204 | K0204 | A library imports a leaf |
| A3 `ports` | VL0104 | K0104 | A port that no constructor takes |
| | VL0110 | K0110 | Anything but ports and wrappers in `ports.go`; an unexported port or port method |
| | VL0112 | K0112 | A port signature with a type of the leaf, of a non-library package, or of another module |
| | VL0116 | K0116 | A write to a port field, or its address, outside a constructor; a package-level variable that holds a port |
| | VL0117 | K0117 | `init` in a leaf |
| | VL0120 | K0120 | A port contract that names a package-level name of its own leaf |
| | VL0123 | K0123 | Strict: a port method that carries a function, channel, or interface (but `error`, `context.Context`, `StrictInterfaces`) |
| | VL0201 | K0201 | A library with a `ports.go` |
| | VL0604 | K0604 | A constructor that takes anything but ports and one last exported settings struct, or returns an unexported type |
| | VL0906 | — | A wrapper method that does more than its contract and one forwarding call, or is not a port method; a port with a wrapper stored unwrapped |
| A4 `wiring` | VL0104 | K0104 | `nil` for a port in main |
| | VL0603 | K0603 | A keyed settings literal in main without every field |
| | VL0605 | K0605 | A constant for a field tagged `vulcan:"secret"` |
| | VL0606 | K0606 | A setting that is not a constant or a flag value (a local variable or `m["key"]`); settings that are not a literal in the call |
| | VL0907 | — | Main calls a function of the module that is not a constructor or a lifecycle method of a leaf |
| | VL0908 | — | Main builds a leaf value (not settings) with a literal or `new` |
| A5 `effects` | VL0124 | K0124 | Strict: a `reflect` call function |
| | VL0125 | K0125 | Strict: an entry leaf dials |
| | VL0203 | K0203 | A library uses a function that chooses a file or a resource by name |
| | VL0205 | K0205 | A network function, `http.DefaultClient`, `http.DefaultTransport`, `net.DefaultResolver`, a literal of `http.Client`, `http.Transport`, `net.Dialer`, `tls.Dialer`, `net.Resolver`, or `RoundTripper.RoundTrip`, outside a gateway or an entry leaf (main included) |
| | VL0211 | K0211 | Strict: a global mutator (the PoC's list and `Mutators`) outside main |
| A6 `libstate` | VL0202 | K0202 | A library variable (or a field or an element of it) assigned outside an `init` of its library |
| | VL0210 | K0210 | Strict: a library variable whose type holds a reference (but `error`, `StrictShared`) |
| A7 `contracts` | VL0501 | K0501 | A call whose constant arguments make a `contract.Requires` of the callee false, in any package of the module |
| | VL0503 | K0503 | A contract that calls a function other than `len`, `cap`, or a conversion, holds a function literal or a receive; an `Ensures` that is not `func() bool { return cond }` |
| | VL0509 | K0509 | `contract.Old` of a value that holds a reference |
| | VL0512 | K0512 | A type assertion in a contract |
| | VL0903 | — | A contract call that is not among the first statements of a function |
| A8 `invariants` | VL0503 | K0503 | An `Invariant` body that is not one pure `return` |
| | VL0506 | K0506 | The zero value breaks the invariant (fields of basic types) |
| | VL0507 | K0507 | A field of an invariant type written outside its methods (also through embedded fields and closures) |
| | VL0508 | K0508 | Its address, or a pointer method called on it, outside its methods |
| | VL0513 | K0513 | `Invariant` on a non-struct type, or with a pointer receiver |
| | VL0904 | — | A pointer method of an invariant type without `defer contract.Keep(recv)` first; a `Keep` on anything else |
| | VL0905 | — | A non-empty literal or a conversion of an invariant type that is not the argument of `contract.Valid` |

The K-codes that the desk test labels "Same (artifact)" or that Go's compiler or `go vet` already
decide have no VL code: K0001, K0005, K0006, K0008, K0009, K0105, K0107, K0108, K0111, K0113,
K0209, K0502, K0510, K0511, K0514, K0601, K0705, K0706.

## The conformance port

`conformance/testdata` holds the Go form of 61 cases of `conformance/*.txtar`, and of the base
system (`accept-base`), each as an overlay of `base.txtar` (the Go form of the cases' base system). Each
gives exactly the listed `VL` diagnostics at the listed positions, or none (`accept`), through the
real command. All pass.

| Analyzer | Cases |
|---|---|
| A1 | P10, P11, P12, P15, P33, P40, A16 |
| A2 | A01, A02, A13, A14, N07, P38 |
| A3 | A03 (no provider: `nil` in main, VL0104), A09, A10, A10b, A10c, A11, N05, N06, N06b, P18, P24, P36, P41, P52 |
| A4 | P27, P32, W07 |
| A5 | N01, N01b, N03, N04, N08, N10, N11, N12, N13 |
| A6 | A15, A15b, A15c, A15d, A15e, N09, N09b |
| A7 | C02, C06, C07, C17, H4, H4b, L4, M2 |
| A8 | C09, C10, C11, L3, M1a, M1b, M1c |

Two cases map to another code in Go: P11 (a leaf that `system.vulcan` does not list, K0003) is
VL0002, because Go has no component file, so an unlisted package is the same mistake as P10; A02
(a boundary names a type of another leaf, K0110) gives VL0101 for the import and VL0112 for the
type in the port signature.

Not ported, with the reason:

| Cases | Reason |
|---|---|
| A04, A05, A06, A06b, A07, P34, P35, P37 | The Go compiler decides them: a constructor takes each port once, a missing constructor or method is a type error (K0105, K0107, K0108, K0110) |
| P14, P22, P23 | Compile errors in Go: the manifest's names are typed constants, and a setting is a typed Go value (K0206, K0606) |
| A08 | Constructor injection cannot build a cycle (K0111) |
| A12, A12b, A12c | A port is a field set by the constructor, so no package-level initializer can reach one (K0117) |
| P42, W08 | Main's flag table is hand-written Go; no generic rule reads it (K0602) |
| V02 | A warning (K0607); a `go/analysis` driver has no warning level, and an error would be stricter than Vulcan |
| V01 | `go vet` itself (K0706) |
| C04, L2, P50, P51, W06, P43, A17, P17, P29, P13, P31, M3, S04 to S08f | Vulcan syntax or an artifact of the language (K0502, K0510, K0511, K0601, K0604 on a type wire, K0113, K0118, K0002 on a `.go` file, K0006, K0209, K0008, K0005, K0009) |
| C12 | Vulcan does not check an invariant on a call through the method's own receiver; `defer contract.Keep` checks at every return (departure 6) |
| B1, C01, C03, C12b, C13, C18, E04, H1, H2, H2b, H3a to H3e, H5, L5, P01, P02, P16, P20, P21, P25, P26, P28, P30, P39, P53, S01 to S03, S07, S09, S10, T05, W01 to W03, W05, A07b | Run, test, or command cases: they test the generated program or the `vulcan` command, not a check. The contract package has its own runtime test (`TestContractRuntime`: the Go forms of C01, C03, C18, C12b, and a body's panic that a check does not hide) |

The analyzer tests (`<analyzer>/testdata/*.txtar`, with `analysistest`) cover the VL09xx codes and
the paths that no conformance case reaches. `TestWorkspace` checks that `vulcan-lint ./...` is
clean on `poc/app/golint`, and `TestReadme` that its README is arm P's but for the check command
and the line of the primer.

## The primer

`poc/app/golint/PRIMER.md` teaches the same model as Vulcan's primer (`vulcan guide`,
`docs/poc/guide.md`) in Go terms: the manifest and kinds, ports and wrappers, constructors and
main, strict mode, contracts and invariants, tests, and each VL code with its fix. Its budget is
Vulcan's, at most 4,000 tokens. No Claude tokenizer runs offline here, so both are measured the
same way, and the estimate (bytes / 4) agrees with the ~2,400 tokens that `docs/status.md` gives
for Vulcan's primer:

| Primer | Bytes | Words | Estimated tokens (bytes / 4) |
|---|---|---|---|
| `docs/poc/guide.md` (`vulcan guide`) | 9,434 | 1,455 | ~2,360 |
| `poc/app/golint/PRIMER.md` | 11,597 | 2,021 | ~2,900 |

The Go primer is longer by its table of codes (about 3,900 bytes). It is within the budget by a
margin of about a quarter.

## The workspace

`poc/app/golint` is `poc/app/go` with: `arch/` (the manifest), `lib/contract`, a `ports.go` in
each leaf with ports (billing, inventory, orders, web), the wrappers of the ports with contracts,
the contracts, a `vulcan:"secret"` tag on the card processor's key, `contract.Valid` on the five
`shop.Stock` values that the store builds, `PROTECTED`, and `PRIMER.md`. The contracts are those of
`poc/app/vulcan`, one for one: 22 clauses in both (11 requires, 9 ensures, 2 invariants), each
`err == nil => X` written as `err != nil || X` and `old(x)` as `contract.Old`. The acceptance
suite passes against it (`STOCKROOM_BIN=<its binary> go test ./poc/acceptance/...`), and it passes
its own check command.

## Departures from the desk test

The desk test designed A1 to A8 on paper. Where the code showed the design wrong, or left a choice
open, the linter follows the code:

1. **Interfaces outside `ports.go` are allowed** (the desk test rejects them). Stockroom's store
   has a private `scanner` interface, as Vulcan allows interfaces in code; what makes a port is a
   constructor parameter, and a constructor takes only interfaces of `ports.go` (VL0604).
2. **A port contract is a wrapper of a fixed form**, `type checkedX struct{ X }` in `ports.go`,
   with each method its contract and one forwarding call (VL0906), so that a reviewer and the
   linter find it. The desk test names a wrapper but not its form.
3. **Constructors and lifecycle methods are found by their signatures**, since Go has no
   declaration of them: an exported function that returns `T` or `(T, error)` of its leaf; a method
   `(ctx context.Context[, s Settings]) error`.
4. **Values with an invariant are checked where they are built** (`contract.Valid`, VL0905), as
   Vulcan checks each literal and conversion, rather than "no literal outside the constructor": the
   store builds `shop.Stock` values, a library type, in five places. An empty literal needs no
   check: VL0506 makes the zero value valid.
5. **Main.** A4 does not check main's flag table (K0602) or the default of a secret flag (only a
   constant in a secret field, VL0605), and does not report one constructor called twice (K0607 is
   a warning). Main is outside strict mode, as Vulcan's generated main is: it may import `syscall`
   and set process-wide state. The desk test's "a settings value that is not a constant or a flag"
   is read as: a constant, a local variable, or an index with a constant key.
6. **`defer contract.Keep` checks the invariant at every return of a pointer method**, also of a
   call through the method's own receiver, where Vulcan checks only outer calls (§7.2). A method
   cannot pass through a broken state by calling another method of its receiver.
7. **`Ensures` runs after the calls deferred after it** (the desk test notes this), reads the
   parameters at return rather than at entry, and checks nothing when the function panics, so that
   it does not hide the panic.
8. **No call graph.** The desk test says a Go analyzer can follow calls through interfaces and into
   other modules (`go/callgraph`, or facts). Both need every dependency loaded from source,
   `modernc.org/sqlite` and its libc included, at every check; the check loop would slow by an
   order of magnitude. A5 matches direct uses, as the PoC does, and closes hole H3 instead by
   rejecting the network values and types of the standard library and `RoundTripper.RoundTrip`.
   Hole H4 (a dial inside another module) stays open, as in the PoC.
9. **K0120 is in A3**, which knows the wrappers, not in A7.
10. **K0206 is a compile error** in Go: `Gateway(dir, x)` takes an `External`. VL0206 stays for a
    manifest that does not compile.
11. **`StrictExempt` names directories** (`"api/orders"`), as the manifest does; Vulcan names leaves.
12. A **file that the build ignores** is reported at the package clause of the package's first
    file, since a diagnostic needs a position in a file that the driver parsed.
