# Audit of PoC phase P1 (commit 6b6db77)

Independent audit of the Vulcan PoC compiler, phase P1, against docs/spec.md, docs/release-1/,
and docs/poc/plan.md (sections 10 and 12). The auditor did not write the code.

Method: I read the spec sections in scope and all of cmd/vulcan, internal/*, vulcan/, and
conformance/. I built the compiler (`go build -o vulcan.exe ./cmd/vulcan`) and ran small
projects outside the repository (in C:\tmp\vaudit, each one a copy of conformance case W01 with
one change). I also fuzzed `blank.File` and the three `decl.Parse*` functions for 6 minutes, on
a copy of the code outside the repository. The conformance suite and the unit tests pass, and
`go vet` is clean.

## Verdict

P1 is a sound skeleton, but it is not ready to carry the measurement. The happy path works,
positions in code files are right, and the main rules have a first case each. But valid input
can fail in generated code (leaf settings that use `time`), the blanker rejects valid Go that
uses the identifier `limit`, and the K0202 and K0117 rules have simple bypasses. Several
ordinary mistakes give diagnostics that point into `.vulcan/build/`, a directory that the agent
must not edit. Three truncated inputs crash the compiler. Fix the High findings before P3 starts,
and add a conformance case for each bypass.

Count: 0 Blocker, 4 High, 9 Medium, 10 Low.

## Blocker

None.

## High

### H1. Leaf settings that use `time` do not compile

Location: internal/link/link.go:121-122 and 134-151 (the registration file imports only the
aliases and system.vulcan's imports), link.go:55 (values are pasted as text).

Reproduction (W01, with Web settings):

```
system.vulcan:   Web{ Wait: 2 * time.Second }, Orders, Billing
api/web/web.vulcan:
    type Settings struct { Wait time.Duration }
    func startup(ctx context.Context, s Settings) error { ... }
```

Observed: `vulcan run` prints
`.vulcan/build/api/web/vulcan_register.go:12:42: error K0008: undefined: time`, exit 1.

Appendix B.2 says a value can use the package `time` with no import. The generated `main` always
imports `time`, so constructor settings work, but `vulcan_register.go` does not. A duration in a
leaf setting (an HTTP timeout for `Web`, a client timeout for `Payments`) is likely in
Stockroom-lite.

Why it matters: valid input fails with a K0008 in a generated file. An agent cannot fix it in
its own code, so the run burns turns or is logged as a toolchain bug (M10).

Fix: always import `time` in the registration file (with `var _ = time.Second`, as main does),
and add a conformance case with a duration leaf setting.

### H2. The blanker treats the identifier `limit` in a struct type as a `limit` clause

Location: internal/blank/blank.go:271-293 (`limits`), the test at 281-285.

Reproduction (added to api/billing/billing.vulcan):

```go
const limit = 4

type pager struct {
	fetch func(limit int) []int
	buf   [limit]byte
}
```

Observed: `vulcan check` gives K0009 "limit on a field is not supported" at 19:13 and 20:9, then
13 K0001 syntax errors, because the blanker turned the rest of each field into spaces. This is
valid Go.

The heuristic only skips `limit` when it follows `;`, `{`, or `,`. After `(`, `[`, `*`, `]`, or a
field name it blanks the text. A parameter named `limit` is common in paging code, and Stockroom
lists orders and stock.

Why it matters: §6 says each Go construct compiles in Vulcan, and design rule 4 says Vulcan never
changes what a body means. Here the compiler changes the text and then reports a cascade of
false errors (M9 false positives). The same body passes in arm G.

Fix: recognize `limit` only in the position of Appendix B.3: after a complete field type at
depth 0 of the struct's braces, on the same line, and before the tag. The simplest correct
method is to parse each struct field list: find the field type with `go/parser.ParseExpr` on the
text up to `limit`, and treat `limit` as a clause only when that parse succeeds and `limit` is at
the field's top level. Add S03-style accept cases for `func(limit int)` and `[limit]T` fields.

### H3. K0202 (library var assigned after startup) has simple bypasses

Location: internal/check/rules.go:198-249 (`libVarAssigns`); only `FuncDecl` bodies are
inspected (208), and a selector on the left side is resolved to its last name only (231-232).

Reproduction (lib/money declares `var Default = Config{Fee: 1}`, `var Rates = [2]Money{1, 2}`,
and `var Limit Money = 100`; api/billing/billing.vulcan has):

```go
func tweak() {
	money.Default.Fee = 99         // a field of a library var
	money.Rates[0] = 7             // an element of a library array var
}
var _ = func() bool { money.Limit = 1; return true }()   // package-level initializer
var reset = func() { money.Limit = 0 }                   // package-level func literal
func again() { money.Limit = 5 }
```

Observed: one diagnostic only, K0202 at the line of `again`. The other four writes compile.

§5.4a rule 1 excludes changes "through a method or a reference" (`hits.Add(1)`,
`cache[k] = v`). A field or an array element of a struct-valued or array-valued var is part of
the variable itself, so it is an assignment, not a reference. Writes in package-level function
literals are plain assignments.

Why it matters: shared library state is a class A or B temptation ("put the flag in a shared
package"). If the hidden rules include one, the "enforced by V" label of plan section 4 will be
wrong for some seeded violations, or the grader will find violations that the compiler missed.
Either way the H-A comparison is distorted.

Fix: inspect every `ast.Inspect` root, not only function declarations (include `GenDecl` value
expressions). For each left side, walk down `SelectorExpr` (field access on a value, not through
a pointer), `IndexExpr` on an array (not a slice or map), and `ParenExpr` to the root
identifier, and flag it when the root is a library var. Also cover `RangeStmt` with `Tok ==
ASSIGN` and `&lib.X` passed to a function, or list `&lib.X` as path 2 in the audit. Add a case for
each shape.

### H4. Ordinary wiring and settings mistakes are reported in generated files

Location: internal/link/link.go:55 (value text with no position), 206 (flag default), and
internal/check/wiring.go:205-211 (the settings parameter type is not checked for export);
link.go:217-223 (import names).

Reproductions:

1. `Web{ Wait: "soon" }` for a `time.Duration` field gives
   `.vulcan/build/api/web/vulcan_register.go:12:37: error K0008: cannot use ("soon") ...`.
2. `flag("wait", default: "5s")` for a `time.Duration` field gives
   `.vulcan/build/vulcan_main/main.go:24:60: error K0008: cannot convert ("5s") ...`. The flag's
   command-line form is the string "5s", so this mistake is natural.
3. A constructor `func NewPayer(o opts) *Payer` with an unexported settings type is accepted by
   the checker, then gives `.vulcan/build/vulcan_main/main.go:38:44: error K0008: name opts not
   exported by package billing`. §5.3 says the settings struct is exported, and K0604 covers a
   bad constructor signature.
4. `import "time"` at the top of system.vulcan (harmless, since `time` needs no import) with a
   value `2 * time.Second` gives three K0008 errors in main.go: "time redeclared", "other
   declaration of time" (at the broken path `.vulcan/build/\tvulcan_main/main.go:8:2`), and
   "imported and not used".

docs/status.md lists case 1 as a known limit ("reported in the generated main"). Cases 2 to 4
are not listed, and case 1 also happens in `vulcan_register.go`.

Why it matters: M5 counts K-codes against Go errors, and M6 measures recovery from diagnostics. A
K0008 in a generated file tells the agent to look at code it must not edit. This raises the cost
of arm V for reasons that have nothing to do with the rules.

Fix: type-check each value against its field in the checker. Build a small `go/types` check
(`types.Eval` of `T(value)` with the field type `T`, in a scope with `time`, the system imports,
and the field type's package), and report K0603 (or a new code) at the value's position in
system.vulcan. Check the flag default the same way. Reject an unexported settings type with
K0604. Skip a system.vulcan import whose name is already imported by main, or give the main's
own imports reserved aliases (`vulcan_time`). If a mapping still fails, map any position in
`.vulcan/build/vulcan_main` or `vulcan_register.go` to the wire's position in system.vulcan.

## Medium

### M1. Three truncated inputs crash the compiler

Location: internal/blank/blank.go:132-134 and 139-142 (`toks[j]` after `Match` returns the EOF
index), internal/lex/lex.go:116-117 and 153-156 (`SigEnd` returns `len(toks)`, then `Clauses`
reads past the end), internal/decl/parse.go:498 (`publishes`), parse.go:406 (end offset from the
replacement rune).

Reproductions, each with `vulcan check`:

- api/billing/extra.vulcan = `package billing\n\nfunc (` : `panic: runtime error: index out of
  range [6] with length 6` in `blank.funcDecl`.
- the same with `func F[` : index out of range in `blank.funcDecl`.
- the fuzzer: `func A()(0000` : index out of range in `lex.Clauses`.
- component.vulcan = `component Billing {\n    publishes P topic[` : index out of range in
  `decl.(*p).tok`, re-panicked through `decl.run`.
- the fuzzer: a system.vulcan value that ends in an invalid UTF-8 byte
  (`system A0{A0.A:A0.A00{A0:00000\x97`) : slice bounds out of range in `decl.expr`. The scanner
  gives such a byte the 3-byte literal U+FFFD, so `Tok.End` (lex.go:50-51) is past the source.

After guards for these five, 150 more seconds of fuzzing found no crash.

Why it matters: a file that the agent's edit left incomplete gives a Go stack trace and no
K-code. The run log then shows a toolchain bug (M10), not a diagnostic.

Fix: clamp each `Match(...)+1` to `len(toks)-1` in blank.go, lex.go, and parse.go; compute
`End` from the source (`off + utf8` width of the source bytes, or the next token's start), not
from `len(lit)`. Keep a fuzz test for `blank.File` and the parsers in the repository.

### M2. K0117 misses a port call reached through a function, and flags one that is never run

Location: internal/check/rules.go:139-155.

Reproduction (api/orders/orders.vulcan):

```go
func warm() money.Receipt { r, _ := Checkout.Pay(1); return r }
var first = warm()                                   // not reported
var later = func() { _, _ = Checkout.Pay(2) }        // reported as K0117
```

Observed: K0117 at the `later` line only. Without `later`, `vulcan check` passes and the program
panics at startup: `nil pointer dereference ... shop/api/orders.warm ... orders.vulcan:17 ...
orders.init()`.

§5.4 rule 12 says the initializer "cannot reach a port call or a publish". The check looks only
for the port identifier in the initializer's syntax. It misses calls through functions of the
leaf, and it flags a function literal that the initializer stores and does not call.

Why it matters: the miss turns a compile error into a runtime panic. The false positive rejects
a common pattern (a package-level handler variable). Both distort M5 and M9.

Fix: build the static call graph of the leaf's package-level functions and methods (direct calls
only, as §5.4c does), and report a package-level initializer that reaches a port use. Do not
descend into a `FuncLit` that is not called in the initializer.

### M3. A missing leaf or constructor setting is silently zero

Location: internal/check/wiring.go:532 and 539 (`given` is filled and never read).

Reproduction: `type Settings struct { Wait time.Duration }` with
`func startup(ctx context.Context, s Settings) error`, and system.vulcan lists `Web` with no
settings. Observed: accepted; `vulcan run` prints `paid 42 0s`.

Appendix E defines K0603 as "Settings for a leaf whose startup takes none, or a missing settings
value". The second half is not implemented, and no conformance case covers K0603.

Why it matters: a forgotten setting (an address, a DSN) gives a running program with a zero
value, which the acceptance tests then catch as a logic defect. This is exactly the kind of error
that Vulcan claims to catch at compile time.

Fix: after the loop, report K0603 at the settings (or the leaf entry) for each exported field
that is not in `given`. If optional fields are wanted, decide how they are marked (a tag, or a
pointer type) and write it in the spec. Add a case.

### M4. A failed startup function can hang the process with no report

Location: vulcan/rt/program.go:156-186.

Reproduction: `config { shutdown_timeout: 2 * time.Second }`; Web's `startup` returns
`errors.New("boom")`; Billing's `startup` is `select {}` and ignores `ctx`. Observed: the program
prints nothing and runs until killed (15 s in the test, `timeout` exit 124).

When the first startup fails, `Main` cancels the others and then waits for them with no
deadline. The deadline exists only on the stop-signal path (168-175). The report is printed only
after every startup has returned. §6 says the stop steps wait "for at most `shutdown_timeout`",
and a failed startup function gives exit 1 with a report.

Why it matters: a startup that fails (a port in use, a bad flag value) while another startup
ignores `ctx` makes the agent's `vulcan run` hang until the 45-minute budget ends, with no
output to act on.

Fix: print the failure report at once, then use the same deadline loop as the signal path, and
exit 1 at the deadline. Add a case (it does not need a signal, so it runs on Windows).

### M5. Non-.vulcan files of a leaf are ignored or lost

Location: internal/project/project.go:149 (only `.vulcan` files are read), internal/gen/leaves.go
87-98 (only code files are written to the build directory).

Reproductions:

- api/billing/helper.go holds `func fee() int`; billing.vulcan uses `fee()`. Observed:
  `K0008: undefined: fee`, with no hint that helper.go was skipped.
- api/billing/testdata/x.txt, and a test that reads `testdata/x.txt`. Observed: `vulcan test`
  fails with `open testdata/x.txt: The system cannot find the path specified`, because the test
  runs in `.vulcan/build/api/billing`.

`//go:embed` fails the same way. §6 says each Go construct compiles in Vulcan. Using `testdata/`
and embedded files is ordinary Go practice.

Why it matters: an agent trained on Go will create a `.go` file or a `testdata/` fixture. In arm
V this gives confusing errors that arm G never sees, so it raises V's cost for reasons that are
not part of the hypothesis.

Fix: report a `.go` file in a leaf or library directory with K0002 and a hint ("rename it to
.vulcan"). Copy (or link) `testdata/` and every non-Go file of each package directory into the
build directory. Add a case for each.

### M6. Module dependencies are resolved silently in the build directory

Location: internal/check/load.go:34 (`GOFLAGS=-mod=mod`), cmd/vulcan/main.go:33 (`mod` is a
deferred command).

Reproduction: billing.vulcan imports `golang.org/x/sync/errgroup`, which the project's go.mod
does not require. Observed: `vulcan check` passes. The build directory's go.mod gains
`golang.org/x/sync v0.24.0` (the newest version, fetched from the network); the project's go.mod
does not change. With `GOPROXY=off`, the same input gives
`K0008: could not import golang.org/x/sync/errgroup (invalid package name: "")`.

docs/release-1/syntax.md puts `vulcan mod` and K0209 in milestone 1. The plan's section 10 does
not list `vulcan mod`, and does not say that it is deferred. With `-mod=mod`, K0209 can never
fire, and the version that a clean build uses can change from day to day (against §11
"Reproducible builds" and decision 5).

Why it matters: Stockroom-lite needs a pure-Go SQLite driver. An agent that adds a dependency in
arm V gets either a silent float (with a proxy cache) or a meaningless message (without one). In
arm G, `go mod tidy` works.

Fix: use `-mod=readonly` in the build directory, report a missing requirement as K0209 at the
import with the fix-it, and implement `vulcan mod tidy` (run `go mod tidy` in the build
directory, then copy go.mod and go.sum back). If `vulcan mod` stays deferred, write that in plan
section 10 and in the primer.

### M7. system.vulcan can hold code

Location: internal/decl/parse.go:552-581 (any Go expression is a value), internal/link/link.go:55.

Reproduction:

```
Orders.Checkout: Billing.NewPayer{ Fee: func() int64 { println("code in system.vulcan"); return int64(len(os.Args)) }() }
```

Observed: accepted; `vulcan run` prints `code in system.vulcan`, then `paid 43`. The literal uses
`os`, which only the generated main imports.

§5.4 rule 2 says system.vulcan holds no code, and Appendix B.2 limits the scope of a value to
`time`, the file's imports, the leaves (in an endpoint), and the setting type's constants. A
system.vulcan import of a leaf package is also not rejected, so a value can name a leaf's exported
var.

Why it matters: an agent can move logic into the architecture file, where no leaf rule applies.
That is a boundary bypass that no grader of arm V expects.

Fix: allow only constant expressions, `time` selectors, conversions, and selectors on imported
libraries; reject function literals and calls other than conversions. Reject a system.vulcan
import that names a leaf (K0110 or K0101). Resolve names against an explicit scope instead of the
generated file's imports.

### M8. Port declaration errors have wrong positions and leak generated names

Location: internal/gen/leaves.go:175-181.

Reproductions:

- Leaf Orders declares a function `Checkout` and a port `Checkout`. Observed:
  `api/orders/component.vulcan:4:20: K0008 Checkout redeclared`. The port name is at 4:16. The
  `/*line*/` directive is placed before `var `, so every position on that line is 4 columns late.
  The run also reports `.vulcan/build/api/orders/vulcan_stub_test.go:10:2: cannot assign to
  Checkout`.
- Two `needs port Checkout` blocks in one component. Observed: seven K0008 errors, four of them in
  `.vulcan/build/...` files, and `port Checkout redeclared` at 6:6, which is the closing `}` of
  the first port. The `type vulcan_X interface` line has no directive, so it takes the position of
  the previous directive's lines.

Why it matters: §5.4 rule 7 says a leaf declares each port once. That rule has no diagnostic of
its own, and the Go fallback is noisy and points at generated files (M5, M6 of the plan).

Fix: report a duplicate port name in a component (K0110 or a new code) and a port name that
clashes with a package-level name of the leaf, before type checking. Put the directive right
before the identifier (`var /*line f:L:C*/Checkout`), and give the `type` line its own directive.
When a Go error is in a generated file of a leaf, map it to the port or the wire that produced it.

### M9. The conformance cases are too thin to measure enforcement

Location: conformance/*.txtar, conformance/conformance_test.go.

- Each rule has one positive case of one shape. A10 (K0116), A12 (K0117), and A15 (K0202) each
  test the plainest shape, so the bypasses in H3, M2, and L1 pass the suite.
- There is no near-miss accept case for the rule checks (for example a local variable named like
  a port, a func-literal handler, or a library var read).
- W04 (stop signal) and S07 (stack trace names the .vulcan line) are missing. status.md lists
  W04. S07 works (the M2 panic trace names `orders.vulcan:17`), but nothing pins it.
- No case covers K0201, K0603, K0604 for a type wire with settings, K0602 for two types or two
  defaults, a missing port method (A06 has only the wrong-type variant), a duration setting, or
  a flag default.
- conformance.md requires differential tests: each accepting run case is also built as plain Go
  and must give the same output. The runner has none.
- S10 checks only the exit code, not that the command "prints that it is not in release 1".

Why it matters: plan section 6 derives the "enforced by V" label from seeded violations of two
shapes. The compiler has only been tested against one shape per rule, so the label is at risk.

Fix: for each rule in scope, add at least two violating shapes and one near-miss accept case, as
the grader will. Add W04 on a host with signals (or test `rt.Main` with a cancelled context), S07,
the missing codes above, and the differential runner.

## Low

### L1. K0116 misses a `range` assignment to a port

Location: internal/check/rules.go:157-171. `for Checkout = range ch {}` (a channel of the port
type) is accepted. Only `AssignStmt` and `&` are checked. Fix: also check `RangeStmt` with
`Tok == token.ASSIGN` (Key and Value).

### L2. A wrong fake in `vulcan.Bind` fails at run time, not at compile time

Location: vulcan/vulcan.go:17-19. The comment says "The compiler checks impl against the port
(K0108)". No checker code does this. A fake with `Pay(amt int)` for a port with
`Pay(amt money.Money)` passes `vulcan check` and panics in the test: `vulcan.Bind: orders.fake
does not satisfy the port`. Fix: in `leafRules`, check the static type of the third argument
against the port interface and report K0108 at the argument; or correct the comment.

### L3. W01 and §6 disagree about when a container exits

§6 says the process runs until a stop signal or a startup error, and that only a container with
no entry leaf exits 0 when all startup functions return nil. W01 has an entry leaf whose startup
returns, and expects exit 0. `rt.Main` follows W01 (program.go:183-186). Neither the spec nor
status.md records the difference. Fix: change §6 (or W01) so that they agree, and say which one
the PoC follows.

### L4. K0122 false positives

Location: internal/blank/blank.go:64. The regular expression runs on the raw source, so
`//go:linkname` at the start of a line inside a raw string or a block comment gives K0122
(reproduced with a raw string). Go itself rejects a directive that is not alone on its line
("misplaced compiler directive"), so I found no bypass. Fix: look for the directive in the
scanner's comment tokens (`go/scanner` with `ScanComments`) instead of the raw bytes.

### L5. K0513 false positive for a type named `invariant`

Location: internal/blank/blank.go:172-179. `type ( level int; invariant func() bool )` (on
separate lines) gives K0513. Fix: report only when `invariant` follows a struct type and is
followed by an expression, not a type.

### L6. Flag and settings details differ from §5.3

- A flag can fill only exactly `string`, `int`, `int64`, `float64`, `bool`, or `time.Duration`
  (wiring.go:589-609). A named type (`type Port int`) or another number type is rejected, and with
  K0602, whose meaning is "one flag with two types, or two flags with one variable". §5.3 says "a
  string, a number, a bool, or a time.Duration".
- `flag("key", default: "abc")` in a `vulcan:"secret"` field is accepted (wiring.go:541-548). It
  puts a literal secret in source, which K0605 exists to prevent.
- A tag `vulcan:"secret,..."` is not recognized.
- `shutdown_timeout` given by a flag is silently ignored (link.go, `writeMain`).
- After a stop signal, a startup that does not stop within the timeout gives exit 1
  (program.go:172-174); §6 says the exit code is 0 after a stop signal.

Fix: accept any type whose underlying type is one of these kinds, use a separate message (or
K0604), treat a default on a secret flag as K0605, parse the tag's options, reject a flag in
`shutdown_timeout` or support it, and align the timeout exit code with §6 or document it.

### L7. One instance per settings value compares raw text

Location: internal/check/wiring.go:180-187. `Billing.NewPayer{Fee: 1}` and
`Billing.NewPayer{ Fee: 1 }` make two instances. §5.3 says two settings are the same when their
text is identical after `vulcan fmt`. Fix: compare a normalized form (the parsed items, with each
value's text after `go/format`).

### L8. Generated code holds local paths

Location: internal/gen/leaves.go:32-34, internal/driver/driver.go:100. Each `//line` directive
names the absolute path of the project, and `go build` runs without `-trimpath`. §11
"Reproducible builds" (T02) asks for no local path. This matters for P2, not for P1. Also,
driver.go:97 keeps the leading tab of a Go continuation line in the file name
(`.vulcan/build/\tvulcan_main/main.go`). Fix: use paths relative to the root with `-trimpath`,
and trim the line before matching.

### L9. Layout checks are looser or stricter than §5.2 and Appendix B.2

- A library outside `lib/` (for example `api/util/library.vulcan`) is accepted. §5.2 says
  libraries are in `lib/`.
- The leaf directory must be the lower-case leaf name (project.go:336), but K0004 compares the
  component name with the directory "ignoring case", as B.2 says. `api/Orders/` gives K0003.

Fix: report K0003 for a library outside `lib/`, and match leaf directories ignoring case (or
state the lower-case rule in the spec).

### L10. An unexported port method gives a misleading K0108

A port method `func pay(...)` can never be satisfied by another package, but the error says the
provider "has no method pay". Fix: reject an unexported port method name at the declaration
(K0001 or K0110), with a hint.

## What is done well

- The stage split follows decision 4: a token-based blanker, `go/packages` on a generated
  module, and rules on `go/types` facts. Each stage is small and easy to read.
- Positions in code files are exact. The `//line` directive per file and the blanker that keeps
  columns give Go errors and stack traces at the right `.vulcan` line and column (S06, and the M2
  panic trace).
- Wiring is careful where it matters: method sets use `types.Implements`, the pointer-receiver
  case has a specific hint (A07), a type wire provides `new(T)`, cycles name the path, and
  instances are shared per constructor and settings.
- The runtime follows the §6 order for the normal path: flags with environment fallback and one
  report for all problems, construction in topological order with immediate binding, parallel
  setup, panics recovered in setup, startup, and construction.
- Deferred constructs give K0009 with the section, not a syntax error, as syntax.md asks.
- The conformance runner checks exact positions and exact error sets, and the whole suite runs
  in about 20 seconds. `vulcan check` on W01 takes under 1 second.
- Known limits are written down in docs/status.md.

## Resolution (2026-10-27)

Each fix has a conformance case; the suite has 84 cases, and all pass. The blanker and the
declaration parsers have fuzz tests (`FuzzFile`, `FuzzParse`); after the fixes, 1.4 million and
0.6 million inputs found no crash.

| Finding | Status | Change | Cases |
|---|---|---|---|
| H1 | Fixed | The checker evaluates each `system.vulcan` value as a constant, and the generated code holds the constant's value, so it needs no import of `time` or of a library | P20, P25 |
| H2 | Fixed | `limit` is a clause only at the struct's own bracket level, after the field's type or as `limit N` on its own line | P21, S03, S04, unit tests |
| H3 | Fixed | K0202 inspects every declaration, and walks a field of a struct var and an element of an array var to the var; `range` assignments too | A15b, A15c, A15d, near miss A15e |
| H4 | Fixed | K0606 (new): a value or a flag default that is not a constant of the field's type, at its `system.vulcan` position. K0604 for an unexported settings type. An import of `time` in `system.vulcan` is harmless | P22, P23, P24, P25 |
| M1 | Fixed | The token slice ends with padding EOF tokens; clause scans stop at EOF; `End` is clamped to the source. Only the first parse error of a file is reported, and type errors after it are dropped | P26, unit tests, fuzz |
| M2 | Fixed | K0117 follows direct calls to the leaf's functions and methods, and does not follow a function literal that is stored and not called | A12b, near miss A12c |
| M3 | Fixed | K0603 for each exported settings field with no value, for leaves and constructors. Spec App. B.2 says so | P27 |
| M4 | Fixed | A failed startup is reported at once; the others get at most `shutdown_timeout`; exit 1. After a stop signal the exit is 0, as §6 says | P28 |
| M5 | Fixed | A `.go` file gives K0002 with a hint to rename it. Non-code files and `testdata/` of a package are copied to the build directory | P29, P30 |
| M6 | Fixed | The build directory is read-only (`-mod=readonly`); a missing module is K0209 at the import; `vulcan mod tidy` adds it, and keeps the project's own `replace` lines | P31; tidy checked by hand |
| M7 | Fixed | A value is a constant (K0606), so a function literal or a call cannot run code. `system.vulcan` imports only libraries (K0110) | P32, P33 |
| M8 | Fixed | K0110 for a port declared twice, a port method declared twice, and a port that has the name of a declaration of its leaf; these stop before type checking. Line directives sit right before each identifier | P34, P35 |
| M9 | Partly | Second shapes and near misses for K0116, K0117, K0202; cases for K0201, K0602, K0603, K0604, K0606, a missing port method, durations, flag defaults, S07, and S10's message. Open: W04 needs a host with signals; the differential runner of conformance.md | many |
| L1 | Fixed | K0116 for a `range` assignment to a port | A10b, near miss A10c |
| L2 | Fixed | The checker checks the fake of `vulcan.Bind` against the port (K0108) | P37 |
| L3 | Recorded | The PoC exits 0 when all startup functions return, as W01 expects; `docs/status.md` says so | W01 |
| L4 | Fixed | Directives are found in comment tokens, not raw text | P38 |
| L5 | Fixed | K0513 only for `invariant` after a struct type and before an expression | P39 |
| L6 | Partly | A flag default in a secret field is K0605; the tag's options are parsed; `shutdown_timeout` must be a constant (K0606); exit 0 after a stop signal. Open: flags for named types | P23 |
| L7 | Fixed | Instances compare normalized settings | |
| L8 | Partly | The tab of a continuation line is trimmed. Open: `-trimpath` and relative `//line` paths | |
| L9 | Fixed | A library outside `lib/` is K0003; leaf directories match names ignoring case | P40 |
| L10 | Fixed | An unexported port method is K0110 at the declaration | P36 |
