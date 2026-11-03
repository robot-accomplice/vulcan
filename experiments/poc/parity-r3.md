# Parity audit of round 3, step 2 (arms L and V)

Reviewer: an independent agent session. It built none of step 2.
Reviewed at commit 1f8a7fe (branch main): `docs/poc/plan-r3.md` (sections 2 to 6, 9), `docs/poc/rules-file.md`, `internal/rules`, `internal/check/ruleset.go`, `poc/golint/rulecheck`, the cases `R*`, `H6*`, and `H7*`, `experiments/poc/r3/wishlist/wishlist.json`, `experiments/poc/r3/defaults/NOT-STATED.md`, and `QN-base-apps.md`.
Job: plan-r3, section 6, step 3. Find each place where the arms differ, each place where they are held equal by a limit, and each place where the draft misstates the wishlist.
Stance: no outcome is wanted. I looked for differences that favor L as hard as for those that favor V. I did not read anything under `C:\Users\jonat\vulcan-poc`. I fixed nothing, and I did not freeze `defaults v1`.

## Verdict

Step 3 has a gate: no open High finding. **The gate does not pass.** There are 4 High, 13 Medium, and 10 Low findings.

The shared package gives the same result in both arms for most programs. Of about 370 cases that I wrote, about 40 show a difference in the result. Each one comes from what an arm gives the shared package, or from code of the shared package that reads a form only one arm has. In the other cases the arms agree, or both reject with other codes (for example `w6`). So the shared cases hide what the adapters do. The generator of the cases shares the same blind spot (section 1).

| Direction | Findings | Effect on the sets of plan-r2 |
|---|---|---|
| V accepts, L rejects | H1, M1, M2, M3, M4, L3 | A rule can land in G (L only) because of a defect |
| L accepts, V rejects | H2, H3, L2, L4, L5 | A rule can land in U (V only) because of a defect, or because of a real difference (H2) |
| V fails with no diagnostic, L reports | H4 | Task difficulty in V, not a set |
| The same in both | M5 to M13, L8, L9 | Set sizes move in both arms |
| Mixed | L1, L6, L7, L10 | Small |

A count of findings per direction is not a measure of which arm is better. It is a count of what this audit found.

The builder's own gates hold on what they test. `go test ./internal/rules ./poc/golint/conformance` passes, and `go test ./conformance -run 'TestConformance/(R|H[67])'` passes.

## How I ran it

- Built `vulcan` and `vulcan-lint` from commit 1f8a7fe into a scratch directory. All runs used scratch copies, with `GOFLAGS=`, `GOWORK=off`, and `GOPROXY=off`, on Windows, with Go 1.27.3.
- Each case starts from `conformance/R01.txtar` without its file `extra` (V), and from `poc/golint/conformance/testdata/base.txtar` with `R01.txtar` on top, without `extra` (L). A case adds one file `extra.vulcan` or `extra.go` with the same Go text, and one rules file. The check is `vulcan check` in V and `vulcan-lint ./...` in L. L's exit code for a finding is 3, and V's is 1.
- The program in each finding below is the file `extra`, unless the finding says otherwise. Each id, such as `lc2`, is one run. The scripts stayed in the scratch directory, so they are not in the repository. The text of each case that a High finding needs is here.
- I wrote two scratch prototypes for section 2. `wp` loads one module with `go/packages`. `factdemo` is a `go/analysis` analyzer with package facts under `multichecker`, the driver of `vulcan-lint`.
- For one test I extracted commit 1f8a7fe with `git archive`, added one name to `internal/rules/effects.go` in the copy, and built that tool in the scratch directory.

## High

### H1. V reads its own generated `/*line*/` comments as a comment that gives a reason

- **Where:** `internal/rules/shape.go:150-162` (`commentNear`) skips only a comment that starts with `//line`. `internal/rules/shape.go:1300-1307` (`commentInside`) skips none. `internal/lower/lower.go:282` and `:288` write `/*line ...*/` into each lowered `return` and clause. `internal/gen/leaves.go:57` writes a header comment that also counts on lines 1 and 2 of a file.
- **Effect:** in V, a rule with `unless-comment yes`, and `CODE-04`, do not fire on a line with a `/*line*/` comment or on the line after it. L has no such comments. `CON-03` has `unless-comment yes` in `defaults v1` and reads only a `return` in a function with `ensures`, so with its default parameters it never fires in V. With `unless-comment` removed, it fires in both (`re1n`).
- **Case `lc2`, rule `ERR-03`, in `api/store`:**

  ```go
  package store

  func g() error { return nil }

  func F(a int) int
      ensures a > 0
  {
      if a > 1 { return 1 }
      _ = g()
      return 2
  }
  ```

  L's form has `defer contract.Ensures(func() bool { return a > 0 })` as its first statement. **V exit 0. L exit 3, VL0810.**
- **Other runs with the same result:** `lc3` (`ERR-03`, discard on a return line), `lc4` (`SEC-07`, a built query after a return), `lc5` (`CODE-04`, a `default` with a return), `re1`, `re5`, `re6` (`CON-03`), `da3` and `da4` (`ERR-03` on line 1 or 2 of a file, the header comment).
- **Favors:** V passes more. A rule can land in G with no real difference behind it.
- **Why the tests pass:** the case for `return-effect`, R130, has no `unless-comment` (`poc/golint/conformance/gen_rule_cases.py:582`). The cases with `unless-comment` (R90, R113, R114, R120, R121) hold no `ensures`.
- **Smallest fix:** make `commentNear` and `commentInside` skip each comment that starts with `/*line` or `//line`, and the header. Add R-cases for `unless-comment` with `ensures` in both arms.

### H2. L does not see a wire that main makes through a helper, a closure, a field, a slice, or a map

- **Where:** `poc/golint/rulecheck/rulecheck.go:384-412` (`provider`) returns `""` for each form that it cannot read, and `Wires` then makes no edge (`:351`). `docs/poc/rules-file.md:90` (call 8) says this is "a form that VL0907 mostly stops".
- **Effect:** `CTX-08`, `DEP-GATEWAY`, `DEP-ENTRY`, each `depends` with `via wire`, and `map locked` do not read that wire in L. V's wires are declared in `system.vulcan`, so V always reads them.
- **Case `w1`, rule `CTX-08`, in L's `cmd/shop/main.go`:**

  ```go
  func mk() billing.Ledger { return store.Open() }

  func main() {
      payer := billing.NewPayer(mk())
      ...
  ```

  **L exit 0. V, with the same architecture, exit 1, K0807.** The plain form `billing.NewPayer(store.Open())` gives VL0807 (`w0`).
- **Result by form:** eight of nine forms pass in L, and VL0907 stops one. Pass: helper that returns the interface (`w1`), helper that takes it (`w2`), struct field (`w3`), slice (`w4`), closure (`w5`), a variable that gets a second value (`w7`), map (`w8`), function variable (`w9`). Stopped: an adapter type in main (`w6`, VL0907). With the lock on, `w1` also hides a new dependency: V K0806, L exit 0 (`lk1`, `lk2`).
- **Favors:** L passes more. This is the only finding here that comes from a real difference of the two designs: a declared wire against a wire made by code. A reader can hold that it is a true member of U. A reader can also hold that a linter can reject what it cannot read, as VL0907 does already, and that U then stays empty. The audit does not choose. The choice changes what T1 can say, so the owner makes it before the freeze.
- **Smallest fix, if the owner wants the arms equal:** in `Wires`, when `provider` returns `""` for an interface argument that is not nil, report a new VL code that says main must pass a value whose provider can be read. On the base application each wire can be read: the arms write the same lock (`TestRulesCommands`). If the owner keeps the difference, correct call 8 and name it as a U candidate in plan-r3, section 0.

### H3. In L, code under `testdata` passes every check

- **Where:** `poc/golint/layout/layout.go:47-51` reads only the packages that `./...` lists, and `go` does not list a directory named `testdata`. `poc/golint/imports/imports.go:38-42` does not reject an import of a module path that the manifest lacks.
- **Case `td1`, rule `ERR-08`:**

  ```go
  package orders

  import "shop/api/orders/testdata/sub"

  func F() { sub.X() }
  ```

  with `api/orders/testdata/sub/s.go` that calls `os.Exit(1)`. **L exit 0. V exit 1, K0008.** `go build ./cmd/shop` and `go vet ./...` also pass in L, so the code is in the binary. `td2` does the same with `testdata` at the root of the module. A package inside a leaf with an ordinary name is rejected in both (`sub1`).
- **Effect:** this leaves the whole model and each rule behind, in L only.
- **Favors:** L passes more.
- **Smallest fix:** in `imports.go`, reject an import of a path in the module that `m.PartOf` does not find.

### H4. In V, a typing error in a test makes the check panic, with defaults on

- **Where:** `internal/rules/shape.go:1342` (`isTable`) calls `.Underlying()` on a type that `go/types` did not record. V runs the rules after a type error. L's driver skips its analyzers in a package with a type error.
- **Case `cg`, rule `TEST-04`, file `api/orders/x_test.go` or `.vulcan`:**

  ```go
  func TestX(t *testing.T) {
      cases := []struct{ A int }{{1}}
      for _, c := range casess {
          _ = c
      }
      _ = cases
  }
  ```

  **V exit 2: `panic: runtime error: invalid memory address or nil pointer dereference`, and no diagnostic. L exit 1: `undefined: casess`.**
- **Favors:** L. An agent in V sees a stack trace for a typing error.
- **Smallest fix:** in `isTable`, return false for a nil type. Add a case with a type error for each form.

## Medium

### M1. V does not read the names and types of a port method's parameters

- **Where:** `internal/rules/shape.go:480-488` and `:540-547`. `portParams` checks only the place and the name of a listed type, and never calls `check`. In L, `ports.go` is in `u.Files`, so `u.params` reads it. In V, `vulcan_ports.go` is not in `u.Files`.
- **Case `pp1`, `PAY-01`:** the port declares `Refund(amount float64) error`, and the provider declares `Refund(x float64) error`. **V exit 0. L exit 3, VL0807.** `pp2` is the same with `DATA-14`. When the provider uses the same name, both fire (`p_pay`), in V at the provider only.
- **Favors:** V passes more.
- **Smallest fix:** call `check` with `v` and `v.Type()` in `portParams`.

### M2. V has no equal of VL0901: a file behind a build tag passes

- **Where:** `poc/golint/layout/layout.go:79-85` has it. Neither `internal/project/project.go` nor `internal/check` has a form of it.
- **Case `bt2`, `ERR-08`:** `//go:build linux` on a file that calls `os.Exit(1)`, on Windows. **V exit 0. L exit 3, VL0901.** The same for `//go:build ignore` (`bt1`), `foo` (`bt3`), `// +build foo` (`bt5`), a file `extra_linux` (`bt6`), and `import "C"` on this host, which has no C compiler (`as4`). So in V what a rule reads depends on the host.
- **Favors:** V passes more. On the host that checks it, the code is not built, so its effect on a run is small. On another system it is built.
- **Smallest fix:** give V the same rule, with `IgnoredFiles` of `go/packages` or the `go list` field.

### M3. `CODE-09` counts more nodes in L for one port call

- **Where:** `internal/rules/shape.go:1518-1535` counts AST nodes. L's `p.ledger.Save(x)` has one more selector and one more identifier than V's `Ledger.Save(x)`.
- **Case `dp4`:** two functions of `api/billing`, each with four copies of `if err := Ledger.Save(x); err != nil { return err }` (`p.ledger.Save` in L). **V exit 0. L exit 3, VL0810.** With three copies neither fires, and with five both fire (`dp3`, `dp5`). So the limit of 60 is at another place in each arm.
- **Favors:** V passes more, near the limit.
- **Smallest fix:** count a selection of a port field in L as V's form, or measure in statements. The base applications do not reach this limit (10 and 10).

### M4. `CODE-11.deprecated` in V does not read a deprecated function of the module itself

- **Where:** `internal/rules/shape.go:1676`. The check needs a file name that ends in `.go`, and V's `//line` gives `.vulcan`.
- **Case `de2`:** `lib/money` has `// Deprecated: use New.` on `Old`, and a leaf calls `money.Old()`. **V exit 0. L exit 3, VL0810.** A deprecated function of the standard library fires in both (`de1`).
- **Favors:** V passes more.
- **Smallest fix:** use `PositionFor(fn.Pos(), false)`, which gives the file in the build directory. Check that lowering keeps the doc comment.

### M5. Both arms panic on valid code: a switch over an alias of a type with unexported constants

- **Where:** `internal/rules/shape.go:1263`, the assertion `.Type.(*types.Named)`. The name before it is already unaliased, this one is not.
- **Case `crash3`, `CODE-04`:** `type S int; type Z = S; const (a S = iota; b); func F(s Z) int { switch s { case a: return 1 }; return 0 }`. **V exit 2, L exit 2, both `panic: interface conversion: types.Type is *types.Alias`.** An exported constant comes first in the `||`, so `sw4` does not reach the assertion.
- **Favors:** neither.
- **Smallest fix:** use `named.Obj().Pkg()`, which the function already has.

### M6. Five rules listed as stated are stated in part, by the builder's own test

`RES-02` is "in part" because a property "is read from the form only" (`NOT-STATED.md:127`). The same holds for these. Each is in both arms.

| Rule | Why it is in part | Case |
|---|---|---|
| `HTTP-01` | A timeout from a variable that is zero passes. The check reads a literal, and not that the entry serves through it. A server built outside the entry is not read | `h_l5`, `h_l7` |
| `HTTP-02` | `Timeout: z` with a zero variable passes, and so does `(*http.Client)(&X{})` with `type X http.Client` | `h_l1`, `h_l2`, `h_l4` |
| `HTTP-13` | It reads `ServeMux` patterns. `http.Server{Handler: h}` with no mux has no route to read, and passes | `h_c18` |
| `HTTP-17` | `w.Header().Set("Set-Cookie", "a=b")` passes | `h_l11` |
| `SEC-07` | The wishlist's decision says dynamic text "must come from a fixed allow-list in the same function, with a comment". `unless-comment yes` accepts any comment, with no allow-list | `h_c3` |

If the owner accepts this, the counts become 38 stated and 18 in part, and of the 39 wanted rules of amendment 6, 23 stated, 15 in part, 1 not stated. `HTTP-01` and `HTTP-02` are outside the 39. **Favors:** neither. The present list leans toward a higher count of stated rules. **Smallest fix:** change `coverage.json` and run `gen_defaults.py`. No code changes.

### M7. A typing error in a rule can still switch it off without a word

- **Where:** `docs/poc/rules-file.md:33` says that a typing error cannot do that. `internal/rules/check.go:86-111` and `shape.go:58-126` check directories and some words only.
- **Cases, each exit 0 in both, with code that the correct rule rejects:** `funcs os.Exitt` (`ty1`), `type net/http.Clent` (`ty2`), `deny-types context.Contex` (`ty3`), `require requirs` at `port-method` (`ty5`), `at field` with nothing to deny (`ty6`), `type lib/money.Ordr` and `field Statuss` of a table (`ty7`, `ty8`), `to api/nope` (`ty9`). A level, a kind, and a word of `in` are reported (`ty11` to `ty14`).
- **Favors:** neither. It matters because T1's labeler writes shapes before a check runs, so a typing error counts as a rule that is not enforced.
- **Smallest fix:** reject an unknown `require` word and `at` with no parameter to apply. Report a `type` or `field` of a table that the project lacks. Correct the sentence in `rules-file.md` for names outside the project.

### M8. "No hidden layer" and "what the check enforces is what the file says" do not hold as written

- **Where:** `docs/poc/plan-r3.md:64-65`. `docs/poc/rules-file.md:85` (call 3) discloses part of this. The plan was not amended.
- **Runs, in strict mode, for 36 witnesses of 36 rule entries, each with the rule on, at `level off`, and with no rules file:** the rule's own finding stops in 36 of 36. The program stays rejected for 7 of them, the same code in both arms: `ARCH-07` (0101), `ARCH-10` (0205), `ARCH-11` (0125), `ARCH-19` (0211), `HTTP-07` (0211), `SEC-30.imports` (0124), `SEC-30.calls` (0124). This matches the list in `NOT-STATED.md:16`.
- **A new build changes what a rule enforces, and no command shows it (case `cj`):** the copy of the tool adds `net/rpc/jsonrpc.Dial` to `dialFuncs` in `internal/rules/effects.go:47`. The project and its `rules.vulcan` do not change. Old tool: exit 0. New tool: V K0803, L VL0803. `rules upgrade` with the new tool says "the source has not changed". The effect lists, `form`, and `at` live in the tool. `CODE-11.deprecated` also reads the doc comments of the toolchain's own files (`shape.go:1639-1664`), so a new Go release can add findings. I read that last point in the code, and did not run it.
- **Favors:** neither. Plan-r3, section 5 asks "which default enforced each rule", and the 7 above can land in S with the default deleted.
- **Smallest fix:** amend plan-r3, section 2 to say that the model checks and the tool's lists stay. Add a version of the lists to the head, so that `rules upgrade` can show a change.

### M9. The reason "L's analyzers read one package at a time" is not a limit of L

See section 2. `NOT-STATED.md:103`, `:139`, `:143`, `rules-file.md:97-98` (calls 15 and 16), and `coverage.json` state it. **Favors:** in words, against L. In effect, neither, because both arms are held. **Smallest fix:** say that the limit is the design of `rules.Unit`, in both arms. Whether to build the wider forms is the owner's decision.

### M10. Nothing says how T1 counts a rule at level `warn`

- **Where:** `docs/poc/plan-r3.md:66` says `error` or `off`. `experiments/poc/labels-r2.md:28` says a rule is enforced when the check rejects each seeded case. `defaults v1` has 3 rules at `warn` (`CONC-03`, `CON-03`, `CODE-03`), and each passes the check in both arms (`ch1`, `h_e26`, `ty18` to `ty20`).
- **Favors:** neither. By the present definition, a team rule that only these defaults match is in N, though the tool prints a finding.
- **Smallest fix:** a sentence in plan-r3, section 5, and an amendment of section 2 for the third level.

### M11. A call through an interface or a generic constraint leaves `forbid-call`, `only-in`, and `call-form`

The same in both arms, exit 0, with the real rule. **Favors:** neither.

| Case | Rule | Program |
|---|---|---|
| `h_f6` | `ERR-08` | `type f interface{ Fatal(...any) }; var x f = log.Default(); x.Fatal("x")` |
| `h_f7` | `ERR-08` | `func G[T interface{ Fatal(...any) }](t T) { t.Fatal("x") }` |
| `h_c2` | `SEC-07` | A local interface with `Exec(string, ...any) (sql.Result, error)`, and `d.Exec(q)` |
| `h_o10` | `only-in database` | The same interface |
| `h_c1`, `h_c8` | `SEC-07`, `OBS-02` | A method value: `f := d.Exec; f(q)`. `forbid-call` reads this form and `call-form` does not, because `calledFunc` needs the call to name the function |

**Where:** `internal/rules/check.go:204-222`, `shape.go:657-674`. **Smallest fix:** for `call-form`, read a selection whose receiver is an interface with a listed method's name and signature. Otherwise list these as limits in `rules-file.md`.

### M12. An alias, a defined type, or a container leaves `declaration`

The same in both arms, exit 0. **Where:** `internal/rules/shape.go:404-410` compares a type string. **Favors:** neither. Each program breaks a wanted rule that the list holds as stated or stated in part.

| Case | Rule | Program |
|---|---|---|
| `h_d1`, `h_d5` | `CTX-02` | `type C = context.Context; type T struct{ X C }`, also embedded |
| `h_d2`, `h_d3` | `CTX-02` | `*context.Context`, `[]context.Context` |
| `h_d4` | `CTX-02` | `type B[V any] struct{ X V }; type T = B[context.Context]` |
| `h_d6`, `h_d8` | `CTX-02`, `CTX-01` | `type I interface{ context.Context }` |
| `h_d7` | `CTX-01` | `func F(a int, x C)` with the alias |
| `h_d9`, `h_d10`, `h_d12` | `PAY-01` | `type M = float64`, `type M float64`, `Amounts []float64` |
| `h_d13`, `h_d14` | `DATA-14` | `type S int; Timeout S`; a named result `(timeout int)` |

`ARCH-03` holds for an alias (`y9`). **Smallest fix:** `types.Unalias` before the compare, and read a named result. A defined type stays a limit.

### M13. The finite lists miss names that the list does not disclose

`NOT-STATED.md:20` names holes H3 and H7 only. Each of these is exit 0 in both arms.

| Case | Rule | Program |
|---|---|---|
| `h_o2` | `ARCH-10`, and K0205 and VL0205 | `var d net.Dialer; d.DialTCP(...)`, a method of Go 1.26. `DialUDP`, `DialIP`, and `DialUnix` are absent too |
| `h_o1`, `h_o9` | `ARCH-11`, `only-in processes` | `c := exec.Cmd{Path: "x"}; c.Run()` |
| `h_o6` | `only-in randomness` | `io.ReadFull(rand.Reader, b)` of `crypto/rand` |
| `h_c7` | `OBS-02` | `slog.LogAttrs(c, slog.LevelInfo, m)`, which the rule lists neither as a function nor as a method |
| `h_o7`, `h_o8` | `only-in clock`, `only-in files` | `context.WithTimeout`, `template.ParseFiles` |
| `h7b` | `ARCH-10` | `jsonrpc.Dial` (disclosed, call 10) |

**Where:** `internal/rules/effects.go:39-90`, `internal/rules/defaults/v1.rules:436`. **Favors:** neither. **Smallest fix:** add the names, and `(*os/exec.Cmd).Run`, `.Start`, `.Output`, `.CombinedOutput`. The model lists need the same names.

## Low

| # | Finding | Case | Where | Favors | Smallest fix |
|---|---|---|---|---|---|
| L1 | `ARCH-06` reads comments in another way in each arm. When the entry before has a line comment: V rejects, L accepts (`cfg4c`). For a block comment above: V rejects, L accepts (`cfg5`). For an entry with `//` in its text: V accepts, L rejects (`cfg6`) | `"a.B", // why` then `"c.D"` | `shape.go:166-172`, `rulecheck.go:250-258` | Mixed | One function that takes the lines of a comment in each arm |
| L2 | In L, a function named `vulcan_x` or `Vulcan_X` leaves `unused-name`, the doc rule, complexity, and duplicates. V rejects the name (K0005) | `func vulcan_x() {}`, `nm1`, `nm2` | `shape.go:175-177` | L | Give `Unit` a field that says this arm generates such names, or reject them in L |
| L3 | `HasEnsures` in V does not match a generic receiver, so `CON-03` is silent there | `func (b *B[T]) F(t *sql.Tx) error ensures true { return t.Commit() }`, `re4n` | `internal/check/ruleset.go:84-96` | V | Unwrap `IndexExpr` and `IndexListExpr` |
| L4 | `at var` reads V's lowered temporary `vulcan_t0`. `defaults v1` is safe, because it names its variables | `rule at var deny-types float64`, a function with a float result and `ensures`, `k2`: V rejects, L accepts | `shape.go:458-475` | L | Skip `generatedName` in `check` |
| L5 | `rules lock` in L runs with a model error. V refuses, and `rules-file.md:42` says that any other error stops it | A leaf that imports a leaf, `cf`: V exit 1, no lock. L exit 0, lock written | `rulecheck.go:416-429` | L | Run the analyzers, or correct the sentence |
| L6 | A finding at `warn` is not in L's `-json`. V's `--json` has it. `rules diff` calls `warn -> off` and `error -> warn` "changed", and only `error -> off` "turned off" | `ce`, `cc` | `rulecheck.go:107`, `commands.go:327-330` | Mixed | Say "turned off" for each change to `off`; document the JSON gap |
| L7 | One finding has another direction in each arm: V says `Subscriptions` is a near-duplicate of `Products`, and L says the opposite. So `QN-base-apps.md:7` holds by rule and part, and not by function. `TestDefaultsOnBase` compares only rule names, so 1 finding against 100 passes | Base applications | `shape.go:1567`, `poc/golint/conformance/rules_test.go:301-310` | Neither | Compare by rule, part, and count; order pairs by name |
| L8 | A check can use the network for a module that the cache lacks, in both arms. The rules file does not need it. I saw two requests to a local proxy from each arm | `require example.com/nope`, with a `go.sum` line and an import. Requests: `/@v/v1.0.0.zip`, `/@v/v1.0.0.mod` | `internal/check/load.go:37-39` | Neither | Say "the rules" in `plan-r3.md:72`, or set `GOPROXY=off` in a check |
| L9 | A check does not read the hash. `hash` may be absent, may be `sha256:0`, and `rules.origin` may be changed, and the check passes. `rules diff` and `rules upgrade` do refuse | `ca`, `cd` | `file.go:170-176` | Neither | None needed if this is meant. Say so in `rules-file.md` |
| L10 | For `at external` with `require comment`, V's required text counts as a comment, and L needs a real comment. Not in `defaults v1` | `x1`: V exit 0, L VL0807 | `ruleset.go:123` | L stricter | Treat the two as one, or document it |

Noted, outside this audit: `init` writes `go 1.26.0` in both arms (`internal/driver/rulesfile.go:23`, `poc/golint/scaffold/scaffold.go`), and the toolchain here is 1.27.3. `poc/app/*/PROTECTED` does not yet list `rules.vulcan` or `rules.lock`, which plan-r3, section 6 asks for at a later step.

## Section 1: parity in each direction

**Result: each kind, each form, and each place ran in both arms. The differences are the findings above.**

| Part of the model | Cases | Result |
|---|---|---|
| `forbid-call` | `f1`, `h_f*` | Equal |
| `only-in`, all 8 effects | `oi_*`, `h_o*` | Equal |
| `depends`: `packages`, `deny`, `allow`, `via`, `except`, blank and dot imports | `dp*`, `y2` to `y4` | Equal. External test package: K0004 and VL0004 reject it in both |
| `depends`, `map locked` | `lk1`, `lk2`, `y8` | H2 |
| `transitions` | R-cases, and `tr1` to `tr4` with `requires`, `ensures`, and a method | Equal. `requires` is no guard in either arm |
| `call-form` | `cf*`, `h_c*` | Equal, with H1 for `unless-comment` |
| `literal` | `li*`, `h_l*` | Equal |
| `code-form` `dropped-error` | `dr1` to `dr5` | Equal |
| `discard-error` | `da*`, `lc1` to `lc3` | H1 |
| `error-compare` | `ec*` | Equal |
| `one-value-assertion` | `ov*` | Equal |
| `channel-size` | `ch*` | Equal at `warn`: exit 0 in both |
| `function-complexity` | `fc*`, `cx*` with clauses at the limit | Equal |
| `switch-exhaustive` | `sw*`, `lc5`, `crash3` | H1, M5 |
| `test-logic` | `tl*`, `cg` | H4 |
| `suppression` | `su*` | Equal |
| `return-effect` | `re*` | H1, L3 |
| `retry-loop` | `rt*` | Equal |
| `duplicate-function` | `du2`, `dp3` to `dp5` | M3 |
| `unused-name` | `un*`, `nm1` | L2 |
| `deprecated-use` | `de1` to `de3` | M4 |
| `at port-signature` | `R70`, `y9` | Equal |
| `at port-method` | `p_clause2`, `y10` | Equal |
| `at field`, `embedded`, `var` | `de_f*`, `de_e*`, `de_v*` | Equal, with L4 |
| `at settings-field` | `de_s1` | Equal |
| `at param` | `de_p*`, `pp*` | M1 |
| `at doc` | `de_d*`, `p_doc*` | Equal |
| `at library` | `x4`, `x5`, `lib1`, `lib2` | Equal: K0004 and VL0004 hold a package name to its directory |
| `at external` | `x1` | L10 |
| `at config-entry` | `cfg*` | L1. `allow_network` and `allow_acquiring` are K0009 in V and have no field in L |
| `at wired-port` | `w0` to `w9`, `ty18` | H2 |

A function call in a port clause gives K0503 and VL0503 in both, so a port clause cannot show a difference through `forbid-call` (`p_clause`, `p_clause3`). In L the wrapper in `ports.go` is read by the rules, and in V the wrapper is generated and is not, so L adds a second finding. The exit is a failure in both.

A package with a type error: V also prints rule findings, and L prints only the type error (`crash6`). Both fail.

## Section 2: equal by constraint

The builder held `CTX-08`, `CODE-09`, and `CODE-11` to one package because "L's analyzers read one package at a time".

**The stated reason is not a limit of L.**

- `factdemo` runs under `multichecker`, the driver of `vulcan-lint`, on a copy of `poc/app/golint`. Each package exports a fact, and the pass of main reads them all: `MAIN SEES stockroom/api/payments:3 stockroom/api/store:18 stockroom/api/web:10 stockroom/api/webhooks:3`.
- `rulecheck.Edges` (`poc/golint/rulecheck/rulecheck.go:416-454`) already loads the whole module with `go/packages`, in the same binary.
- V holds each package in `Checker.Full`. It calls `CheckUnit` one package at a time because that is the shape of `rules.Unit`.

So the limit is the design of the shared package, in both arms.

**Does it lower either arm's reach? Yes, both, by the same amount on the base applications.**

| Rule | Form today | Whole-program form, by a prototype, on the base applications | V | L |
|---|---|---|---:|---:|
| `CTX-08` | Ports that a gateway provides | Each wired port whose provider reaches a network or database use, through its own wires. On these applications each provider does, so a `wired-port` rule `to leaf gateway` gives the count | 69 (today 32) | 69 (today 32) |
| `CODE-09` | Inside one package: 10 | Added pairs across packages, limit 60, similarity 90: `orders.Pay ~ purchasing.Cancel`, `orders.Ship ~ purchasing.Cancel` | 2 | 2 |
| `CODE-11` | Unexported, package level | Exported names that no package of the module uses, apart from generated names and `arch` | 0 | 0 |

The prototype's network list is five packages, not `effects.go`, and it does not read test files. A witness: `h_e17` shows two copies of one function in two parts, and both arms pass today.

**Could the shared package suppress a real difference (set U)?**

- On these three rules, on the base applications: no. The wider forms give the same result in each arm.
- In general: it can. A rule kind in one shared package can differ between the arms only through an adapter. So any member of U or G on these kinds is an adapter defect or a form that only one arm has, and not a measure of the language. Call 1 says this in part. **T1 cannot show that a second builder of the linter would reach the same result.**
- H2 is the case to watch. The current `CTX-08` already differs on `w1`. A whole-program form that follows wires would widen that difference, unless H2 is fixed.
- M3 applies to a cross-package `CODE-09` too.

**What each choice leans toward:**

| Choice | Leans toward | Leans against |
|---|---|---|
| Hold V to one package | "U stays empty", plan-r3's prediction. V cannot show what its in-memory view could | V, if V's view could give more with less work. The prototype shows no such gain on these rules |
| Hold L to one package | The same prediction | L: the reason in the documents says L cannot. Staticcheck's unused check and the facts above show that a linter can |
| Leave all three "in part" | A lower count in T1, in both arms, so against the gate of section 5 | Nothing in parity |
| Build the wider forms | A higher count in both arms, so toward the gate. They then need the fix of H2 and M3 to stay equal | More findings on the base applications: 69 against 32 for `CTX-08` |
| One shared package at all | U empty by construction on these kinds. Fewer places to read | Independence: call 1 says it gives no test of whether two builders would agree |

I state no preference.

## Section 3: holes, in both arms

Each row is the same in both arms unless the finding says otherwise.

| Thing tried | Result | Cases |
|---|---|---|
| Aliases | `forbid-call` and `literal` see through an alias of a type. `declaration` does not (M12). `switch-exhaustive` panics (M5) | `h_f4`, `h_d1`, `crash3` |
| Method values | `forbid-call` rejects them. `call-form` does not (M11) | `h_f1`, `h_f2`, `h_c1` |
| Generics | A constraint with a listed method passes (M11). A type argument passes (M12) | `h_f7`, `h_d4` |
| Embedding | A promoted method is rejected. A local interface that embeds `context.Context` passes | `h_f5`, `h_d6` |
| Dot imports | Rejected | `h_f3`, `dp5` |
| Build tags | V passes, L rejects (M2). `_x` and `.x` file names pass in both, and are not built | `bt*` |
| Generated files | A `Code generated` header gives no exemption | `gen1`, `gen2` |
| `//go:linkname` | K0122 and VL0122, with or without a body. H6 and H7 hold as written | `ln1`, `ln2`, `as2`, `h7a` |
| `testdata` | L passes, V rejects (H3) | `td1`, `td2` |
| Interface over a listed call | Passes (M11) | `h_f6`, `h_c2` |
| A value in a variable | Passes each rule that reads a constant | `h_l9`, `h_c5`, `h_c13` |
| A helper that holds the break | `TEST-04` and `TEST-05` read only the test body or the test file | `h_e18`, `h_e21` |
| Other forms of a retry | `<-time.After`, `goto`, recursion pass. A multiply and any random call in the function count as backoff and jitter (disclosed: "in part") | `h_e22` to `h_e25` |
| `var _ = g()` | Passes `ERR-03`, which reads only assignments | `h_e4` |
| A concrete or defined error type | Passes `ERR-01`, which compares the string `error` | `h_e1`, `h_e2` |
| `crypto.MD5` of package `crypto` | Passes `SEC-10` | `h_e28` |

## Section 4: fidelity

**Counts:** the recount gives 43 stated, 13 in part, 154 not stated, and for amendment 6, 26, 12, and 1. No rule comment in `v1.rules` differs from its wanted text. `CON-02`, the 1 that moved, is about purpose, so I agree with "meaning". `HTTP-01` and `HTTP-02` are stated outside the 39, as the list says. By M6, I hold them to be in part.

A sample of 30 rules. "As text" means that the runs found no gap in the form that the text names.

| Rule | Status in the list | Against the wanted text | Case |
|---|---|---|---|
| `ARCH-03` | stated | As text | `y9` |
| `ARCH-06` | stated | As text. Also reads `mutators`. The two `allow_` keys do not exist in release 1 | `key_*` |
| `ARCH-07` | stated | As text | `dp3` |
| `ARCH-10` | stated | Narrower: M13 | `h_o2` |
| `ARCH-19` | stated | Narrower, and the text says "such as" | `h_f15` to `h_f17` |
| `ARCH-21` | stated | As text | `lib2` |
| `ERR-01` | stated | Narrower: a concrete error type. Wider: `fmt.Println`, `bytes.Buffer.Write` | `h_e2`, `dr1` |
| `ERR-03` | stated | Narrower: `var _ =`, and any comment passes | `h_e4`, `h_e5` |
| `ERR-04` | stated | Narrower: through a variable. Wider: two concrete values | `h_e7`, `h_e8` |
| `ERR-08` | stated | As text | `h_f1` to `h_f5` |
| `ERR-09` | stated | As text | `h_e9` |
| `ERR-13` | stated | Narrower: a variable. Wider: a format verb first | `h_c13`, `h_c14` |
| `CTX-01` | stated | Narrower: alias, local interface | `h_d7`, `h_d8` |
| `CTX-02` | stated | Narrower: M12 | `h_d1` to `h_d6` |
| `CTX-03` | stated | Narrower: `context.Context(nil)` | `h_c10` |
| `CTX-07` | in part | Wider: a key of type `any` is rejected, though its value can be unexported | `h_c11` |
| `CTX-08` | in part | Its reason is wrong: M9 | section 2 |
| `HTTP-01` | stated | Narrower and wider: M6, `h_l6` | `h_l5`, `h_l6` |
| `HTTP-02` | stated | Narrower: M6 | `h_l1` |
| `HTTP-11` | stated | Narrower: a variable | `h_l9` |
| `HTTP-13` | stated | Narrower: M6 | `h_c18` |
| `HTTP-17` | stated | Narrower: M6. Wider, disclosed: each cookie | `h_l11` |
| `SEC-01` | in part | Narrower, disclosed. A key in a struct tag or by concatenation passes | `h_l15`, `h_l17` |
| `SEC-02` | in part | Narrower: a settings struct with a nested struct is not read | `h_d16` |
| `SEC-07` | stated | Wider exit: M6. Narrower: M11 | `h_c1` to `h_c3` |
| `DATA-03` | stated | Narrower: a variable. Wider: a column named `begin` | `h_c5`, `h_c6` |
| `OBS-01` | stated | As text. `println` and `fmt.Fprintln(os.Stdout, ...)` are not named in it | `h_f12`, `h_f13` |
| `OBS-02` | stated | Narrower: M13 | `h_c7` |
| `CODE-02` | stated | Narrower: `//gosec:disable`, `/* nolint */` | `h_e12`, `h_e13` |
| `CODE-08` | stated | Wider, disclosed. An empty `//` passes | `h_d21` |
| `TEST-04` | stated | Wider: a table of strings is rejected. Disclosed: `if err != nil` | `h_e19` |

## Section 5: false alarms on the base applications

Measured again: V 284, L 285, hash `sha256:4d47eceb52365d0cc00b2a21f2db656977dec6519c1bb4d0f678690892224d37`, as `QN-base-apps.md` says. No rule at `warn` fires. A compare by rule, part, and text gives two rows of difference: `library shop has no doc comment` in L only, and the direction of one `CODE-09` pair (L7).

This says whether a finding is true by the wanted text. It does not say whether a rule should stay on: that is the owner's decision.

| Rule | Findings (V) | Read | True by the wanted text | Notes |
|---|---:|---:|---|---|
| `CODE-08` | 126 | All, by a heuristic on the source line | All, on the widest reading of "exported name" | 69 are port methods, 56 are struct fields or grouped values, 1 is a function. The text says "port" and "exported name". 125 of 126 depend on that reading. Line 22 of `NOT-STATED.md` discloses fields and interface methods |
| `CON-01` | 51 | All, by name, and the store's bodies | 34 true, **17 false** | 17 are reads: `Store.Payments`, `.Subscription`, `.Deliveries`, `.Product`, `.Order`, `.OrdersOf`, `.Supplier`, `.Purchase`, `.PurchasesOf`, and `Stock.Product` (2), `Billing.Payments`, `Catalog.Product`, `Orders.Payments`, `Purchasing.Supplier`, `Events.Subscription`, `Events.Deliveries`. The text says "changes state". Call 14 discloses the form |
| `CTX-06` | 46 | All, by name | All | `(*sql.Tx).Exec` 12, `(*sql.DB).Query` 9, `.Exec` 8, `.Begin` 7, `.QueryRow` 5, `(*sql.Tx).QueryRow` 3, `http.NewRequest` 2 |
| `CTX-08` | 32 | All, by provider | All | The 29 store methods reach the database. `Charge`, `Refund`, and `Send` reach the network |
| `CODE-09` | 10 | 4 pairs | The form is near. The content is another matter | `Ship`/`Pay`, `refund`/`receive`, `Subscriptions`/`Products`, `toPurchaseJSON`/`toOrderJSON` differ in ports or types. Merging needs a generic or a function value. A reader can hold either view |
| `ERR-01` | 6 | All | All | Explicit `rows.Close()` (4), `db.Close()`, `io.Writer.Write` |
| `ERR-03` | 4 | All | All | Two are `n, _ := res.RowsAffected()`, a tuple that the wishlist's decision also covers |
| `ERR-13` | 2 | All | **2 false** | `"%w; and the release of its stock failed: %v"` and `"%s: %w"`. The error string starts with the value of the verb, so the check cannot say |
| `HTTP-01.timeouts` | 3 | All | All | One literal gives three findings, so the count is of fields |
| `HTTP-09` | 1 | All | All | `api/payments` has no `CheckRedirect` |
| `OBS-01` | 3 | All | All | `log.Printf` |

Of 284: 19 false by the text, 10 disputable, and 125 true only on a wide reading.

## Section 6: claims

| Claim | Result | Evidence |
|---|---|---|
| "No hidden layer" | Does not hold as written | M8: 7 of 36 stay rejected with the rule gone, and the effect lists live in the tool |
| A deleted rule stops firing | Holds for the rule's own finding: 36 of 36 witnesses, with `level off` and with no file, in both arms | The 36 witnesses of M8, R04, R71, R89, R98, R111 |
| A check needs no network | Holds for the rules. A rules file whose origin names a server that does not answer passes, and the source can be gone. Does not hold for the check as a whole | `cd`, L8 |
| `upgrade` never changes a project unasked | Holds. `rules upgrade`, `rules diff`, a check, `init` again, and `--accept` with a wrong hash leave each file's content and time the same, in both arms | `cb` |
| It never changes because a compiler did | The text of the rules holds. What they enforce does not | M8, `cj` |
| The hash of a source is recorded | Holds. The head holds the SHA-256 of the source with CRLF as LF, the same in both arms. `rules diff` refuses a changed `rules.origin` or a changed hash line. `http`, `file`, and `ftp` sources are refused. A check does not read the hash | `ca`, L9 |
| The level `warn` | Passes the check and prints, in both arms, for code, wires, imports, and declarations. Plan-r3 does not name it, and call 12 discloses it. T1's reading is open | M10, L6, `ty18` to `ty20` |

## What I could not check

- A real `https` source. The only client path is in `internal/rules/commands.go:31-44`, and a test replaces it. I ran the refusals of other schemes only.
- A Linux host, and a host with a C compiler. Case `as4` ran with no C compiler, so there `import "C"` is an ignored file and not a cgo package. M2 and the rule `SEC-30.imports` need a run on such a host.
- `vulcan-lint` flags such as `-rules=false`, which `multichecker` gives. I did not run them. The plan fixes the check command.
- That a new Go release changes the findings of `deprecated-use`. I read the code, and did not run it with two toolchains.
- Each name in each list. I tried names that I chose, not all of the standard library.
- `transitions`, beyond the 20 existing cases and 4 of mine, which hold contracts. No program of mine reached a difference there.
- Whether a fresh session judges section 5 the same way. Plan-r3, section 5 asks a fresh session for QN. This is a sample for this audit, not that measure.
- T1, T2, T3, and QN themselves. They run after the freeze.

## Rerun

Reviewed at commit 737b2c5 (branch main), with the fixes ac416e3 and 737b2c5, and plan-r3, sections 2, 5, and 9 (amendments 7 to 9). The same reviewer ran this. The sections above stay as they were at e2b1ab6.

### Result

**Step 3's gate (no open High finding) does not pass.** Two High findings are open, and both are new here: N1 and N2.

| Severity | Open now | Which |
|---|---:|---|
| High | 2 | N1, N2 |
| Medium | 2 | M6, M7 |
| Low | 8 | M12 (rest), N3 to N9 |

Of the 27 findings of the first audit, 17 are fixed by a result that I ran, 7 are listed as a limit and I hold the limit acceptable (M8, M11, M13, L6, L8, L9, L10), 1 is listed as a limit that is not a true limit for each rule it reaches (M12), and 2 are still open (M6, M7). M3 is in the 17 for port calls only. The contract form of the same count is N2. H3 is in the 17 for import paths inside the module. The nested module is N1.

The 4 High findings of the first audit are fixed. H2 is fixed by the owner's choice: L now rejects each form of the first audit.

### How I ran it

- A fresh build of 737b2c5. No script was lost. I reran each case of `b1` to `b5`, `b7`, and `c2` with no change to its text, and I compared each result with the saved output. Of 377 rows, 75 changed.
- I replaced `c3` with new runs, because its patch was for the old list. I fixed one fault of my own: the cases `ty15` to `ty17` of `b5` had no rules file because of my harness. `c1` holds the correct runs of those three.
- I wrote 166 new cases for this rerun: 67 on H2, H3, H4, and M2; 71 on M3, M6, the widened types, and comments; 18 on nested modules, names in rules, and forms of main; 10 on defined types and generated text.
- I rebuilt scratch tools with a changed list, and with a changed kind, from `git archive` of 737b2c5. A third scratch build holds five forms of the `CODE-09` count behind an environment variable.
- `go test ./internal/rules ./poc/golint/... ./conformance` passes. The generators `gen_cases.py` and `gen_rule_cases.py`, run on a scratch copy, reproduce each archive under `testdata`.

A row with the tag DIFF is not always a difference in the result. The rows that remain are: V and L reject with other codes (`w*`, `td*`, `sub1`, `lk2`), a form that one arm alone can have (`nm3`, `k9`, `g11`, `g12`, `key_*`), V's `warn` word that my reader does not parse (`ch1`, `ty18` to `ty20`), or a fault of my case (`du1`, `cfg4`, `x2`, `x3`, `y1`, `h7c`). `h_d16` and `h_d17` are a constructor that L reads and V does not, because it has no wire. They are as before. The rows where one arm passes and the other rejects, on a form that both can write, are the findings below.

### The 27 findings

| Finding | Result | Run | Note |
|---|---|---|---|
| H1 | Fixed | `lc2` to `lc5`, `re1`, `re5`, `re6`, `da3`, `da4`: V now rejects with L. `q1` to `q7`: comment forms agree | Both arms now scan the file that a person wrote (`shape.go`, `ScanComments`) |
| H2 | Fixed, by amendment 7 | `w1` to `w5`, `w7` to `w9`: L VL0909. `w6`: VL0907 and VL0909. `lk2`: VL0909 | See "H2" below |
| H3 | Fixed for an import path in the module | `td1`, `td2`, `t1`, `t2`, `t4`, `t5`, `t7`, `t8`, `t11`: VL0002. `t3`: the go command refuses | A nested module and `vendor` still pass: N1 |
| H4 | Fixed | `cg`, `crash2`: V K0008, no panic. 15 broken programs with all defaults on: no panic in either arm | See "H4" below |
| M1 | Fixed | `pp1`, `pp2`: V rejects at `component.vulcan` | |
| M2 | Fixed | `bt1` to `bt3`, `bt5`, `bt6`, `as4`, `g1` to `g5`, `g7`, `g8`: K0901 and VL0901 | See "M2" below |
| M3 | Fixed for port calls | `dp3` to `dp5`, `m3p3` to `m3p6`: both fire at 5 | The contract form is open: N2 |
| M4 | Fixed | `de2`: V rejects | |
| M5 | Fixed | `crash3`: a finding in both, no panic | |
| M6 | **Open** | See "M6" below | 17 rules moved. The test is not applied the same way to the others |
| M7 | **Open, narrower** | `ty5` to `ty9`: K0801 and VL0801. `ty22` to `ty24`: exit 0 in both | See "M7" below |
| M8 | Limit, acceptable | Plan-r3, section 2 now says it. A changed effect list: `rules upgrade` and `rules diff` show it (`cn`). A changed kind: exit 3 becomes exit 0, and upgrade says "the source has not changed" (`co`) | `rules-file.md` says that `lists` does not cover the code of a kind or the model's lists. That matches the runs |
| M9 | Fixed | No document holds the old reason. `rules-file.md`, calls 15 and 16, and amendment 9 name `rules.Unit` | |
| M10 | Fixed, by amendment 8 | Plan-r3, sections 2 and 5 name `warn` and say that it does not count as enforced. Exit 0 in both | |
| M11 | Method values fixed. Interface: limit, acceptable | `h_c1`, `h_c8`, `z14`, `z15` reject. `h_f6`, `h_f7`, `h_c2`, `h_o10` pass in both | To know which value an interface holds is flow. A match by name and signature is possible, and rejects an honest interface. The limit is true for an exact answer. Rules with a listed method moved to in part |
| M12 | Alias, pointer, slice, type argument, embedded interface, named result fixed. **Defined type: listed, not a true limit for each rule** | `h_d1` to `h_d9`, `h_d12`, `h_d14` reject. `z10`, `z11`, `h_d10`, `h_d13`, `s1c`, `s1d` pass | See "M12" below |
| M13 | Names fixed. Finite list: limit, acceptable | `h_o1`, `h_o2`, `h_o6` to `h_o9`, `h_c7`, `h7b` reject in both. `h_o5` (`tls.Client`) passes | A list can only hold names that someone tried. `rules-file.md` says this |
| L1 | Fixed | `cfg4c`, `cfg5`, `cfg6`: the same result in both | |
| L2 | Fixed | `nm1`, `nm2`: L reads the name | |
| L3 | Fixed | `re4n`: both fire | |
| L4 | Fixed | `k2`: both pass | The fix adds N5 |
| L5 | Fixed | `cf`, `cl`: both refuse a model error, write with a rule's error, and write with a fault in the rules file. L refuses VL0909 | |
| L6 | Words fixed. JSON: limit, acceptable | `cc`: "turned off" for `warn -> off`. `ce`: L's `-json` holds no `warn`, V's does | |
| L7 | Fixed | A compare by rule, part, and text gives one row: `library shop has no doc comment`, in L. The pair `Products` and `Subscriptions` now reads the same | See "`TestDefaultsOnBase`" below |
| L8 | Limit, acceptable | Each arm still sends 2 requests to a local proxy for a module that the cache lacks. `rules-file.md` now says so | |
| L9 | Limit, acceptable | A check still passes with a changed hash. `rules-file.md` now says so | |
| L10 | Limit, acceptable | `x1`: V exit 0, L VL0807 | Each arm rejects its own form when it is absent |

### New findings

#### N1. High. In L, a nested module or a vendored package passes every check

- **Where:** `poc/golint/imports/imports.go:45` asks `m.InModule(p)`. A path with another module's name is not in the module. `layout.go` reads only what `./...` lists, and `go` does not list a directory with its own `go.mod`, or `vendor`.
- **Case `t9`, rule `ERR-08`:** `go.mod` holds `require example.com/evil v0.0.0` and `replace example.com/evil => ./third`. `third/go.mod` is `module example.com/evil`, and `third/s.go` calls `os.Exit(1)`. A leaf calls `evil.X()`. **L exit 0. V exit 1, K0002.** `go build ./cmd/shop` and `go vet ./...` pass in L.
- **Other runs:** `t13` (a deeper directory), `t14` (`net.Dial` in the nested module, so the model is passed too): L exit 0, V K0002. `t10` (`vendor/example.com/evil` with `modules.txt`): L exit 0, V K0209.
- **A module outside the project passes in both** (`t12`), and that is N8. The difference is that V has no place for the code inside the project.
- **Favors:** L passes more. This is the same escape as H3, by another path, so I give it the same severity.
- **Smallest fix:** walk the module root in `layout.go`, and reject a `.go` file in a directory that no part holds, as V's K0002 does. This covers `testdata`, `_x`, `.x`, `vendor`, and a nested module, with one rule.

#### N2. High. `CODE-09` counts V's lowered contract code, so the limit is at another place in each arm

- **Where:** `internal/rules/shape.go:1779` (`tokens`) reads the lowered code in V. A `return` in a function with `ensures` is lowered to `{ var vulcan_t0 T; vulcan_t0 = e; if !(c) { vulcan_rt.Fail(...) }; return vulcan_t0 }`. `skipped` leaves out only the `if`. In L the same statement is `return e`. In L a `contract.Requires(c)` leaves an `ExprStmt` in the count, and in V the whole `if` is skipped.
- **Case `m3e3`, limit 60, similarity 90:**

  ```go
  func A(x string) int
      ensures x != ""
  {
      if x == "x" { return 1 }
      if x == "x" { return 1 }
      if x == "x" { return 1 }
      return 0
  }
  ```

  and a function `B` with the same form. L's form has `defer contract.Ensures(...)`. **V exit 1, K0810. L exit 0.**
- **By form, the number of guards where each arm first fires:**

  | Form | V | L |
  |---|---:|---:|
  | Port calls | 5 | 5 |
  | `ensures` with returns | 3 | Above 7 (the last that I ran) |
  | `requires` | Above 8 | 8 |
  | `ensures` with `old` | 7 | Above 8 |

- **Why I judge this differently:** M3 was Medium, because the case that I ran was near the limit, and it had one form. These cases show a factor of more than two in the place of the limit, on the ordinary form of a Stockroom function. `Pay` and `Ship` have this form. It has the root of H1, a rule that reads lowered code as if a person wrote it.
- **Favors:** in V with `ensures` and `old`, L passes more. With `requires`, V passes more. A rule can land in U or in G because of this.
- **Smallest fix:** count what a person wrote. Count a lowered return block as a `return`, and leave out each statement that holds only a contract call. A scratch form with this closes the `ensures` return and `requires` rows (`m3e3`, `m3e5`, `m3e7`, `m3r8`), and leaves the base applications at 10 and 10 with the same pairs. It does not yet close `old` (`m3o7`).

#### Low

| # | Finding | Case | Where | Favors | Smallest fix |
|---|---|---|---|---|---|
| N3 | `typeIn` reads a map key, and a container for `types`. So `PerRouteTimeout map[string]time.Duration` and `Deadline map[string]time.Time` fail `DATA-14`, which asks for `time.Time` and `time.Duration`. `more []context.Context` and `m map[string]context.Context` are treated as a context under `CTX-01` | `z1`, `z2`, `z6`, `z7` | `shape.go:653` | Neither | Do not walk a map key for `deny-types`. Use the exact type for `types` |
| N4 | A `literal` rule reads the text of V's generated `vulcan_rt.Fail`. A function `local` or `dev` with a clause gives `"orders.local"`, which `BLD-07` rejects. A team's `deny-text ^ensures$` rejects each `ensures` | `bl1`, `bl2`, `bl3`: V rejects, L exit 0 | `shape.go:1118` | L passes more | Skip the arguments of a `vulcan_rt` call |
| N5 | `at var` now skips V's temporaries, and still reads L's variable for `old`. A team's rule `at var deny-types int` rejects `o := contract.Old(n)` in L only | `k3`: was the same, now V exit 0, L VL0807 | `shape.go:533` | V passes more | Skip a variable whose value is a contract call |
| N6 | After H4, the arms print other things for a project with a type error. Both fail. V also prints model codes and findings of parts that L skips: `e1` K0205, `e5` a rule break in a leaf that imports a broken library, `e8` a wire rule, `e9` a library rule. L's driver skips each package that depends on the broken one | `e1`, `e5`, `e8`, `e9`. Agree: `e2`, `e3`, `e4`, `e6`, `e7` | `internal/check/ruleset.go:181` | V's agent reads more per run | Document it. Equal text needs a change of L's driver or of V's stages |
| N7 | VL0909 is on with `Strict: false` and with no rules file, but the analyzer's `Doc` says "in strict mode". A helper that main never calls adds an edge (`v14`), so a wire rule fires on code that does not run | `w24`, `v14` | `poc/golint/wiring/wiring.go:22` | L stricter | Correct `Doc`. Note the edge in `rules-file.md` |
| N8 | A module outside the project passes in both arms, with a `replace`. The Limits table of `rules-file.md` does not list this | `t12`: exit 0 in both | `docs/poc/rules-file.md:93-103` | Neither | One row |
| N9 | Call 22 says "No finding on the base applications changed", and does not name the other form or the source of the numbers. See "M3" below | | `docs/poc/rules-file.md:131` | Neither | One sentence |

### H2: VL0909

**Does it reject each unreadable form? Yes, of each form that I tried.** The first audit's eight no longer pass. These do not pass either: a conversion (`v3`), a method of the provider (`v4`), a package variable of a leaf (`v5`), a package variable of main (`v1`), a helper's result (`v7`, `v8`), the address taken (`v11`), a second value (`v15`), a type switch (`v18`), a type assertion (`v19`). Other routes are rejected by older checks: a variadic constructor (`v30`, VL0604), a generic constructor (`v31`), a setter (`v32`), a field (`v33`), a package variable of the leaf (`v34`, VL0116).

**Does it reject a plain form or a valid program?** The forms that a Go team writes pass: a local variable, a chain of them, parentheses, a wire in an `if` init, a wire in a helper, the `run` idiom (`w20`, `w21`), and an app struct (`w22`). These valid programs are rejected: a package variable of main (`v1`), a conversion to the port type (`v3`), a provider whose address is taken for another reason (`v11`), and two branches that assign the same constructor (`w23`). `rules-file.md` lists the second-value form. These are the cost that amendment 7 accepts, and V has no equal.

Amendment 7 is the owner's choice, so I do not rank it. The plan names the lean, and that is correct: it leans toward an empty U on wires.

### H3: places

| Place | L | V |
|---|---|---|
| `testdata`, in a leaf, a library, or the root | VL0002 | K0008 |
| `_x` and `.x` | VL0002 | K0008 |
| `vendor/sub`, by the module's path | The go command refuses | K0008 |
| `vendor/example.com/evil`, with `modules.txt` | **Exit 0** (N1) | K0209 |
| A nested module with a `replace` | **Exit 0** (N1) | K0002 |

### H4: type errors

V reads no package with a type error, and a type error in a test leaves the package without its tests. The change is small: V's model codes stay, and a rule's finding in that package stops. The case `crash6` had K0008 and K0802, and now has K0008. Compared with L, the exit is a failure in both, and the text differs as N6 says. There was no panic in 15 broken programs with all defaults on, in either arm.

### M2: each form

K0901 and VL0901 agree on each form that the build ignores: `//go:build` for another system, `ignore`, a word, the old `// +build`, a GOOS suffix, a GOARCH suffix, both, a test file, a library, a negated tag, `cgo`, and `import "C"` on this host. A file for this host is read, and its rule fires in both (`g6`, `bt4`, `bt7`). So a program can pass on one host and fail on another, the same in both arms. L's message is at the first file of the package, and V's is at the file itself. Two forms give other codes and the same failure: the only file of a leaf (`g9`, V K0901 and K0107, L a build error), and a file with another package name (`g14`, V K0004, L VL0901). A file named `_x` or `.x` passes in both, and is not built.

### M3: which count

The numbers below are for the base applications, with limit 60 and similarity 90, in scratch builds. Each row has the same pairs in both arms, unless it says otherwise.

| Form | Pairs |
|---|---:|
| Each node, the form before the fixes | 10 |
| The builder's form: a chain of names counts as a selector of one name | 10 |
| Only a selection of a port field | 10 |
| The builder's form, and the fix of N2 | 10 |
| A chain of names is one token | 5 |
| One token, and the fix of N2 | 4 in V, 5 in L |

- **Was the form chosen for its reason, or to fit the base count?** I cannot see the intent. The form has a reason that does not need the count: it gives L's chain the same three nodes as V's `Ledger.Save`, so V's count for a port call does not change. But that reason does not choose between the rows: one token also equalizes port calls. The base count is the only thing in the record that separates them, and the record does not say that a form of 5 was tried. That is N9.
- **Which count is right by the wanted text?** The text does not decide. `CODE-09` says "near-duplicate functions" and gives no number. The limit 60 and the similarity 90 are the builder's. At similarity 80, 85, 90, and 95 the builder's form gives 15, 12, 10, and 6 pairs. So 10 is a result of two parameters, and no test of a form. Five pairs that the one-token form removes are `CreateOrder` and `CreatePurchase`, `OrdersOf` and `PurchasesOf`, `Products` and `Subscriptions`, `createPurchase` and `receivePurchase`, `receive` and `refund`. A lower limit does not bring them back (40 gives 6), so the change is in similarity and not in scale. A reader can hold either view of those pairs.
- **Is the limit at the same place in each arm?** For port calls, yes. For contracts, no: N2.

### M6: the test for stated and in part

The recount of `coverage.json` gives 26 stated, 30 in part, and 154 not stated, and the 17 moved rules are as the builder says.

**The test is not applied the same way.** `rules-file.md:93` says: "A rule that a limit reaches is stated in part". Call 23 moves the rules with a method in their list. `SEC-07` moved because any comment removes the finding. These stay stated, and each is reached by the same reason. I ran each one, and each passes in both arms.

| Reason | Rule | Case |
|---|---|---|
| A value in a variable (Limits table) | `ERR-13`, `HTTP-11`, `ERR-04`, `CTX-03` | `s2`, `s3`, `s8`, `s10` |
| A type defined from a listed type (Limits table) | `ARCH-03`, `ERR-01` | `s1c`, `s1d`, `s4`, `s5` |
| A finite list (Limits table) | `ARCH-19`, `CODE-02` | `s11` to `s14` |
| Any comment, also an empty `//` (the reason of `SEC-07`) | `ARCH-06`, `ERR-03`, `CONC-03`, `CODE-04`, `CODE-08` | `s28`, `s6`, `s22`, `s16`, `s17` |

`ARCH-03` through a defined pointer to a struct does fire (`s1b`, `s1e`), and by my reading of `carried` the reason is that a chain of struct fields under `sql.DB` or `sql.Rows` holds a listed type. A defined interface passes.

**Recount by the sentence as written:** 13 stated, 43 in part, 154 not stated. The 13 are `ARCH-01`, `ARCH-02`, `ARCH-07`, `ARCH-21`, `ERR-09`, `HTTP-07`, `SEC-10`, `SEC-31`, `DATA-15`, `CODE-03`, `CODE-06`, `TEST-04`, `TEST-05`. Four of these have a form gap that no limit names: `s18` and `s19` (`TEST-05`), `s20` (`TEST-04`), `s21` (`SEC-10`), `s25` (`SEC-31`). If those count too, the result is 9 stated and 47 in part. Those are my reading, so I do not hold them as part of this finding.

A reader can also hold another test: "constants only" and "no comment reading" are limits of each kind, so they do not change a status, and only a limit that the wanted text names does. By that test, some of the 17 moved rules would move back (`HTTP-01`, `HTTP-02`). Either test is coherent. The present list holds neither, and it leans toward a higher count of stated rules. **Favors:** neither arm. **Smallest fix:** write one test, and run it on each rule in `coverage.json`. No code changes.

### M7: typing errors

`ty5` to `ty9` are now faults. `rules-file.md:33` says "A name outside the project is not checked", and that holds (`ty1` to `ty3`, `ty25`). **A name inside the project is not checked either**, and the sentence implies that it is: `funcs shop/lib/money.Nope` (`ty22`), `type shop/lib/money.Recept` (`ty23`), `deny-types shop/lib/money.Recept` (`ty24`), each exit 0 in both. A field of a `literal` is the same (`ty29`, `ty30`). This is the case that matters for T1: a labeler's shapes name the project's parts. The check holds the package, so `ResolveTypes` can read these names as it reads a table's type. **Favors:** neither. **Smallest fix:** extend `ResolveTypes` to each name with the project's module path. Correct the sentence.

### M12: a defined type

`rules-file.md:98` gives the reason: "An underlying match would reject `time.Duration` under `DATA-14`". That holds for `DATA-14`. It does not reach the others. `type M float64` under `PAY-01` (`z11`) has no such case, because no standard type is defined from a float for money. `type C context.Context` under `CTX-02` (`z10`) and `type C net.Conn` under `ARCH-03` (`s1c`) are identical interfaces, so a match with `types.Identical` on the underlying type rejects them with no false alarm of that form. So this is listed as a limit, and it is a choice for each rule but `DATA-14`. `PAY-01`, `CTX-01`, and `CTX-02` are in part, so their status is honest. `ARCH-03` is M6. **Favors:** neither. **Smallest fix:** a parameter for an underlying match, on the rules where it is safe, or say that this is a choice.

### Older cases `A09` and `P36`

The change is proper. Each has a stub main with `web.New(place{})`, main's own type. Amendment 7 names that form, and `VL0909` is at that argument (17:17). Each case keeps its code (VL0112, VL0110), and adds one line to `want.txt`. Only these two archives hold the stub. `gen_cases.py` makes them. The V forms did not change.

`git diff 1f8a7fe HEAD` shows no change under `poc/app`, `poc/acceptance`, or `poc/harness`. Each base application passes its check with no rules file, as before. **I could not check the seeded cases**, which are under the path that this audit may not read. A seeded case in L whose main has a form of VL0909 would now fail for that reason. T2 reads them, and it is run with the frozen tool, so its record can say so.

### The `lists` line

- Both arms write the same value, and it does not change between runs. V and L agree: `sha256:e28cae27...`.
- A changed effect: exit changes, and both `rules upgrade` and `rules diff` say so (`cn`). An accept records it (`TestListsInUpgrade`).
- With no `lists` line: a check passes, and `rules diff` prints `lists  here, and sha256:...`, with an empty value. A `lists` line in a rule is a fault. A source's own `lists` line does not reach the head.
- A check does not read it, so a new tool can change a result with no word from the check itself. `rules-file.md` says what `lists` covers, and that holds.

### `TestDefaultsOnBase`

It now compares by rule, part, and count, and it adds one to V's count for `CODE-08 lib/shop`. My compare by text agrees: that one row is the only difference on the base applications (`library shop has no doc comment`), and V's `library.vulcan` has a comment where L's Go files have none. So the named difference is true, and it is a difference of the applications. If L's files gain a package comment, the test fails, and that is the right signal. The test does not read the text, so two other findings with the same count would pass. My compare by text is not in the repository. I accept this, and I do not list it as a finding.

### Section 2 of the first audit

Amendment 9 keeps the three rules at one package, and the documents now say why. The counts of 69, 2, and 0 for the wider forms in both arms do not depend on these fixes: I reran the 69 (`c1`), and did not rerun the prototype `wp`. The lean of a shared package stays, and plan-r3, section 5 now says it.

### What I could not check

- The sealed seeded cases, as above.
- A Linux host, and a host with a C compiler, as before. So `g6`, `bt4`, and `bt7` hold for Windows, and `as4` holds with no C compiler.
- A real `https` source, as before.
- The builder's intent for M3. I read the record and ran the forms.
- Each form of main that a Go program can have. I tried those above.

## Second rerun

Reviewed at commit 15ef49e (branch main), with the fixes ae55653 and 15ef49e. The same reviewer ran this. The sections above stay as they were at b49bf27. This rerun is narrow, as the owner asked: the fixes, plus each existing case script for a regression.

### Result

**Step 3's gate (no open High finding) does not pass.** N1 and N2 are fixed. One High finding is open, and it is new here: N10. V has the same escape that N1 was in L. It was in V from the start, and my earlier runs did not try it.

| Severity | Open now | Which |
|---|---:|---|
| High | 1 | N10 |
| Medium | 1 | N11 |
| Low | 5 | N9 (rest), N12, N13, N14, N15 |

Of the findings open after the first rerun: N1, N2, M6 (main count), M7, N3, N4, N5 are fixed by a result that I ran. M12, N6, N7, N8 are listed, and I hold the text honest. N9 is correct in its main text, and has a rest.

### How I ran it

- A fresh build of 15ef49e. I reran `b1` to `b5`, `b7`, `c1`, `c2`, `a1`, `a2`, `a3`, and `a5`, with no change to their text, and compared them with the saved output of the first rerun. Of 543 rows, 36 changed. Each change is one of the fixes below, and none is a regression. The rows `td1`, `td2`, and `t1` to `t8` also gain the new finding at the file.
- I wrote 72 new cases. `a6`: fixtures, names in rules, and witnesses for the 9 stated rules. `a7`: a nested module in each directory that V skips. `a8` and `a9`: forms of a call of the leaf's own helper.
- `go test ./internal/rules ./poc/golint/... ./conformance` passes, with `-parallel 2` for `conformance`. A first run under full parallelism, with other work on the host, had one failure: `R20`, with a `fork/exec` error of the compiler. It passes alone, so I hold this as a fault of the host.
- One scratch build, from `git archive` of 15ef49e, prints the tokens of each function that `CODE-09` reads. So I compared the count directly, on 26 forms and on each function of both base applications.

### Results for each finding

| Finding | Result | Run |
|---|---|---|
| N1 | Fixed | `t9`, `t13`, `t14`, `t10`: L now VL0002. See "N1" below |
| N2 | Fixed for contracts | `m3e3`, `m3e5`, `m3e7`, `m3r8`, `m3o7`: the same result in both arms. Port calls: both at 5. See "N2" below |
| M6 | Main count fixed | Recount below. A rest is N13 |
| M7 | Fixed | `ty5` to `ty9`, `ty22` to `ty24`, `ty29`, `ty30`, `m1`, `m3`, `m5`, `m6`, `m15`, `m17`, `m18`: K0801 and VL0801. `ty1` to `ty3`, `m16`: exit 0 in both, as `rules-file.md` says. A rest is N12 |
| M12 | Listed, acceptable | `z10`, `z11`, `s1c`, `s1d` pass in both. `rules-file.md` now says that this is a choice for `PAY-01`, `CTX-01`, `CTX-02`, and `ARCH-03`, and a limit for `DATA-14`. That matches the code |
| N3 | Fixed | `z1`, `z2`, `z5`, `z6`, `z7` pass in both. `z3`, `z8`, `z12` still reject, and each is a value of the listed type |
| N4 | Fixed | `bl1`, `bl2`, `bl3`: exit 0 in both. `bl4`: a text that a person wrote in a clause fires in both |
| N5 | Fixed | `k3`: exit 0 in both |
| N6 | Listed, acceptable | `e1`, `e5`, `e8`, `e9` as before. `rules-file.md` states it. Both arms fail |
| N7 | Fixed | The analyzer's `Doc` no longer says "in strict mode". `rules-file.md` lists the edge of `v14` |
| N8 | Listed, acceptable | `t12`: exit 0 in both. A row in the Limits table states it |
| N9 | Main text correct. A rest is open | See "N9" below |

### New findings

#### N10. High. In V, a nested module under a directory that V skips passes every check

- **Where:** `internal/project/project.go:153`. The walk of the project skips a directory whose name starts with `.` or `_`, and `testdata` and `vendor`. K0002 is in the part that the walk reaches. `internal/gen/leaves.go` then copies the `replace` with an absolute path.
- **Case `ne1`, rules `ERR-08` and `ARCH-10`:** `go.mod` holds `require example.com/evil v0.0.0` and `replace example.com/evil => ./testdata/evil`. `testdata/evil/go.mod` is `module example.com/evil`, and `testdata/evil/s.go` calls `os.Exit(1)` and `net.Dial`. A leaf calls `evil.X()`. **V exit 0. `vulcan build` writes the binary. L exit 3, VL0002.**
- **Other runs:** `ne2` (`_x/evil`), `ne3` (`.x/evil`), `ne4` (`api/orders/testdata/evil`): V exit 0, L VL0002. `ne6` (`vendor/evil`): V exit 0, L a build error. `ne5` (`third`): both reject.
- **Why High:** it is the escape of H3 and N1: code inside the project that leaves the model and each rule, with exit 0, in one arm. I gave those High, so I give this High. The direction is the other way: V passes more. A rule can land in G because of it.
- **Why the record misses it:** `rules-file.md` says "V ignores a `.go` file under `testdata`, a `_` name, a `.` name, or `vendor` when nothing imports it". The cases above do import it. The Limits row for a module outside the project does not cover it, because these are inside the project.
- **Smallest fix:** make V's walk report a `go.mod` below the root in each directory, also a skipped one, with K0002.

#### N11. Medium. `CODE-09` counts a call of the leaf's own helper as two more nodes in L

- **Where:** `internal/rules/shape.go`, `tokens`. A chain of names counts as a selector of one name, so `a.save` is three nodes, and `save` is one. In L a helper that uses a port is a method, because the port is a field. In V the same helper can be a plain function, because a port is a package name.
- **Case `m3d10`:** two functions of `api/billing`, each with 10 lines `save(x)` in V and `p.save(x)` in L. **V exit 0. L exit 3, VL0810.** The same through 14 lines. Both fire at 15. With an `if err := save(x)` form both fire at 5 (`m3h5`), so the steps there hide it.
- **On the base applications:** by the probe, `api/billing` `Refund` is 126 in V and 128 in L, and `api/web` `Handler` is 237 and 283, for each route's `a.refund` against `refund`. At limit 60 and similarity 90 the findings stay equal. At similarity 80, L has 14 findings and V has 13, so one pair crosses.
- **Severity:** the same kind and size as the first M3, which was Medium. It is not High: the place of the limit moves by a factor of 1.5 at most in a function made only of such calls, and much less in code with other statements. N2's contract form was a factor of more than two on the common form.
- **Favors:** V passes more, near the limit. **Smallest fix:** count a selection of a method on the function's own receiver as an identifier. Or list it as a limit.

#### Low

| # | Finding | Case | Where | Favors | Smallest fix |
|---|---|---|---|---|---|
| N12 | Names in a rule still differ at the edges. `shop/cmd/shop.X`: V K0801, L exit 0. `shop/lib/contract.Nope`: V exit 0, L VL0801. `(shop/api/billing.Ledger).Save`, a port method: a fault in both, and in L it is false, because `LookupFieldOrMethod` on a pointer to an interface finds nothing. L also applies the rule. `deny-types shop/api/billing.Ledger`: V silent, L VL0807, because V's port type has another name. `map[string]shop/lib/money.Nope` and a prefix `shop/lib/money.Nop*`: no fault in either | `m7`, `m9`, `m11`, `m12`, `m4`, `m2` | `internal/rules/shape.go`, `projectName`, `resolveNames` | Mixed | Look up on the type, not on a pointer to an interface. Treat `cmd` and `lib/contract` the same in both. Say that a port cannot be named in V |
| N13 | M6, the rest. See below | | `coverage.json` | Neither | One row for `HTTP-07`. A rule for what "names" means under the looser test |
| N14 | L rejects a `.go` file that nothing imports, under `testdata`, `_x`, `.x`, or `vendor`. V passes it. See "N1" below | `fx1` to `fx8`, `fx10`, `fx12`, `fx15` | `poc/golint/layout/layout.go`, `hidden` | V passes more | Drop `testdata`, `_`, and `.` from `hidden`, and keep `go.mod` and `vendor` |
| N15 | V counts one more token, `*ast.EmptyStmt`, at the end of a function with no result and a lowered contract | `f_void`, `Shrink`; `api/billing` `charge` and `refund`: 6 and 5 | `shape.go`, `tokens` | L passes more, by one token | Skip `EmptyStmt`, or the `;` in the lowering |

### N1: the fixture question

**Yes, the fix adds a difference between the arms.** The programs that L alone rejects are these. Each passes in V, and each is a `.go` file that nothing imports:

| Place | L | V |
|---|---|---|
| `testdata/x.go`, in the root, a leaf, or a library (`fx1`, `fx2`, `fx7`, `fx12`) | VL0002 | exit 0 |
| `_x/x.go`, also in a leaf (`fx3`, `fx6`) | VL0002 | exit 0 |
| `.x/x.go`, `.github/tools/x.go` (`fx4`, `fx8`) | VL0002 | exit 0 |
| `vendor/x.go`, also in a leaf (`fx5`, `fx15`) | VL0002 | exit 0 |
| A nested module under `testdata`, not imported (`fx10`) | VL0002 | exit 0 |

These agree: a nested module in a plain directory (`fx9`), `Testdata` and `node_modules`, which `go` lists (`fx16`, `fx17`), a `.txt` fixture (`fx13`).

**How much it matters:** little. The file is not built, so no rule's result depends on it, and neither application has one. The cost is that an agent in L must move a Go fixture, or a small tool under `.github`, that an agent in V can keep. `rules-file.md` says "L is stricter on one form", and that is correct for these programs. The walk also was not needed for them: `imports.go` already rejects an import of such a path. Only `go.mod` and `vendor` need the walk.

**The honest limit, if the owner does not change it:** the sentence in `rules-file.md`, with N10 corrected. **The smallest fix for both N10 and N14 is one rule in each arm:** a `go.mod` below the root fails in each directory, in both arms, and an unimported file in a skipped directory passes in both.

The H3 table now reads:

| Place | L | V |
|---|---|---|
| `testdata`, imported, in a leaf, a library, or the root | VL0002 | K0008 |
| `_x` and `.x`, imported | VL0002 | K0008 |
| `vendor/sub`, by the module's path | VL0002, and the go command refuses | K0008 |
| `vendor/example.com/evil`, with `modules.txt` | VL0002 | K0209 |
| A nested module in a plain directory, with a `replace` | VL0002 | K0002 |
| **A nested module under `testdata`, `_x`, or `.x`, with a `replace`** | VL0002 | **Exit 0 (N10)** |
| **A nested module under `vendor`, with a `replace`** | A build error | **Exit 0 (N10)** |

### N2: the limit, and the similarity counts

**Contract forms: the limit is at the same place, within one token.** The probe gives the same token list in both arms on 24 of 26 forms: returns with `ensures`, named results and a bare return, a tuple, two values, a switch, a loop, a closure, `requires`, `old`, a literal with an invariant, a pointer to one, `Keep`, labels, a person's `defer`, `select`, a nested block, one line, and a person's own block of the same shape as a lowered one. The two others are N15.

**Base applications:** 118 of the 131 functions in V have the same list in L. The other 13 are constructors, which hold port parameters in L, wrappers in `ports.go`, which only L has, `startup` against `Serve`, `Refund` and `Handler` (N11), and `charge` and `refund` (N15).

**Port calls:** both at 5 (`m3p3` to `m3p6`). **A call of the leaf's own helper: no** (N11).

**The builder's numbers are correct for V, and not for L at 80.**

| Similarity (limit 60) | V | L | Same pairs |
|---:|---:|---:|---|
| 80 | 13 | **14** | No: L has `Charge` and `Refund` twice |
| 85 | 12 | 12 | Yes |
| 90 | 10 | 10 | Yes |
| 95 | 6 | 6 | Yes |
| 100 | 4 | 4 | Yes |

`rules-file.md` says "13, 12, 10, and 6 pairs ... in each arm". That is a small error. It comes from N11. At similarity 90, the limit gives V and L the same pairs at each of 30, 40, 50, 60, 70, 80, and 100 (34, 12, 10, 10, 8, 7, 6).

### M6: the recount

`coverage.json` gives `status` 9, 47, 154, and `loose_status` 41, 15, 154. Both counts hold. The 39 of amendment 6 give 4, 34, 1, and 24, 14, 1.

**The strict test is applied the same way, with one exception.** The 9 are `ARCH-01`, `ARCH-02`, `ARCH-07`, `ARCH-21`, `ERR-09`, `HTTP-07`, `DATA-15`, `CODE-03`, and `CODE-06`. I tried a break of each by each limit that could reach it. The one that passes is `HTTP-07` through a library (`st5`), which is the same route as `SEC-31` through a library (`s25`), and `SEC-31` is in part. The call is `http.HandleFunc`, which `ARCH-19` and K0211 reject in a library when all defaults are on. So the break does not pass the ruleset, and it does pass the rule. By the sentence as written this gives 8, 48, 154. `st1` and `st2` are no break of the text, because it names imports.

**The looser test is not applied the same way.** It says "only a limit that the wanted text names changes a status". `HTTP-13` is in part because the text names "each entry route". By that reading these also name what the limit reaches, and they are listed as stated: `ARCH-06`, `ERR-03`, `CODE-02`, `CODE-04` ("gives the reason", "with a reason"), `CONC-03` ("a comment defends"), `HTTP-02` ("a non-zero `Timeout`", and `h_l1` passes), and `HTTP-01` ("serves through", which the rule does not read). By my reading this gives 34, 22, 154. `rules-file.md`, call 23, says that this class is the builder's reading, so the record is honest about it. It leans toward a higher count under the looser test. The main count is the strict one, by the owner's decision, so I rank this Low. Another reader of "names" can reach another number.

### N9: the record of `CODE-09`

**Does the record state this correctly?** Call 22 of `rules-file.md` now states the forms, the counts, the order, and the reason: the builder built the one-token form, saw 10 fall to 5, and changed it because the brief of that fix said not to reduce findings. That agrees with my runs (10, 10, 10, 5). It corrects what the first rerun could say: I wrote that I could not see the intent. Three small gaps remain, and these are the rest of N9:

- It says "the brief of that fix", and not that the coordinating session wrote it. A reader should know that the instruction came from outside the builder.
- `QN-base-apps.md` holds the row `CODE-09 | 10 | 10`, with no word of this. That is where a reader of QN looks.
- "in each arm" at similarity 80, as above.

**Should `CODE-09` as built count as a tuned rule when it is read later?** Yes, for the base applications. The form of the count was selected by its result on those applications, in the direction that keeps findings. So this rule's count there is not a measure on code that was not used to shape it, and that is what QN asks of those applications (plan-r3, section 5). The record does not say whether the limit 60 and the similarity 90 were also set with these applications in view. They were in the first build of the kind, and the count depends on them.

The tuning does not reach the blind tests. No T1 or T2 rule text or result went into it, so a later `CODE-09` result on T1 has no such lean. On a QN of other code, the form stays, and it is selected. I do not choose a form. Which one is right is not something that the wanted text or these runs decide.

Smallest fix: a sentence in `QN-base-apps.md` beside the row, and the words "the coordinating session's brief" in call 22.

### What I could not check

- The sealed seeded cases, as before.
- A Linux host, a host with a C compiler, and a real `https` source, as before.
- How the limit 60 and the similarity 90 were first chosen. The record does not say.
- Each form of a lowered statement. I tried 26 forms and each function of the base applications.

## Third rerun

Reviewed at commit 02f2464 (branch main), with the fixes a5177bb and the upgrade 02f2464. The same reviewer ran this. The sections above stay as they were at 17ccad3. This rerun is narrow, as the owner asked: the changes, each existing case script for a regression, and the separate question on `harness validate`.

### Result

**Step 3's gate (no open High finding) does not pass.** N10, N11, and N14 are fixed. Two High findings are open, and both are new here. N16 is a parity finding: the same escape, by a directory that the new rule skips, in L only. N17 is the separate question: it is not a parity difference in the rule kinds, it is in the harness, and it would invalidate the labels of T1. By the wording of the gate it is an open High. Whether it blocks step 3 or the start of step 5 is the owner's call. N16 alone keeps the gate closed.

| Severity | Open now | Which |
|---|---:|---|
| High | 2 | N16, N17 |
| Medium | 1 | N18 |
| Low | 4 | N19, N20, N21, N22 |

### How I ran it

- A fresh build of 02f2464. I reran `b1` to `b5`, `b7`, `c1`, `c2`, `a1` to `a3`, and `a5` to `a9` with no change to their text, and compared them with the saved output of the second rerun. Of 613 rows, 43 changed. Each change is one of the fixes below, and none is a regression.
- I wrote 47 new cases in `a10` and `a11`, and I ran `harness validate` on the demo material.
- **WSL gives a Linux host**, with Go 1.27.3 and a C compiler. Earlier sections list that as not checked. I ran 35 cases there, the ones that can differ by host.
- `go test ./internal/rules ./poc/harness/... ./poc/golint/... ./conformance` passes, with `-parallel 2` for `conformance`.

### Results for each finding

| Finding | Result | Run |
|---|---|---|
| N10 | Fixed for each directory but two | `ne1` to `ne5`: K0002 and VL0002. `ne6`: V K0002, L a build error. On Linux the same. The rest is N16 and N18 |
| N14 | Fixed | `fx1` to `fx8`, `fx12`, `fx15`: exit 0 in both. `fx10`: both reject the `go.mod`. `fx16`, `fx17`: both reject |
| N1 | Holds | `t9`, `t13`, `t14`: K0002, VL0002. `t10`: V K0002 and K0209, L VL0002. On Linux the same |
| N11 | Fixed for the receiver | `m3d10` to `m3d15`, `m3h3` to `m3h9`, `m3s*`, `m3c*`: the same in both. The rest is N21 |
| N12 | Fixed | `m2`, `m4`, `m7`: K0801 and VL0801. `m9`, `m14`: exit 0 in both. `m11`, `r5`: a port by name fires in both, in the same files. `r6`, `r7`: a typing error in it is a fault in both. `r9`: `return-effect` reads a port by name in both. `m12`, `r1`: at a field, L only, as `rules-file.md` says. The rest is N20 |
| N13 | Fixed | The recount below |
| N15 | Fixed | `f_void`, `Shrink`: the same tokens. The probe: 26 of 26 forms are the same |
| N9 | Fixed | Call 22 names the coordinating session. `QN-base-apps.md` has the note. "In each arm" is now true: see below. The spec's numbers that the note gives are right: `duplicate_similarity` 90 and `duplicate_min_tokens` 50 (`docs/spec.md:1959-1960`). I cannot check what the note says of the builder's order of work |

### The rule of `go.mod`, attacked

`rules.ForeignRoots` is a walk by name, and a `replace` is what brings the code in. So I attacked the walk.

| Attack | Windows, V | Windows, L | Linux |
|---|---|---|---|
| `go.mod` in `third`, `testdata`, `_x`, `.x`, `vendor` | K0002 | VL0002, or a build error for `vendor` | The same |
| **`.vulcan/build/evil`** (`d2`) | K0008, because each check empties it | **Exit 0, and `go build` passes** | The same |
| **`.vulcan/bin/evil`** (`d1`) | **Exit 0, and `vulcan build` passes** | **Exit 0, and `go build` passes** | The same |
| **`x/.git/evil`** (`d4`) | **Exit 0, and `vulcan build` passes** | **Exit 0, and `go build` passes** | The same |
| **`.git/evil` in a real repository** (`gi`) | **Exit 0** | **Exit 0, and `go build` passes** | Not run |
| `.vulcan/evil` (`d5`) | K0002 | VL0002 | Not run |
| **`third/GO.MOD`** (`u1`) | K0002, for the `.go` file | **Exit 0, and `go build` passes** | Both reject |
| **`testdata/evil/GO.MOD`, `_x/evil/Go.Mod`** (`u2`, `u3`) | **Exit 0, and it builds** | **Exit 0, and it builds** | Both reject |
| **`vendor/MODULES.TXT`** (`u4`) | K0209 | **Exit 0, and `go build` passes** | Both reject |
| `VENDOR/modules.txt` (`u5`) | K0002, K0209 | VL0002 | Both reject |
| `vendor` with no `modules.txt` (`u6`, `u7`), `vendor` in a leaf (`u8`) | K0209 | A go command error | Not run |
| A replace to a directory with no `go.mod` | K0002, K0008 | A go command error | Not run |
| A replace to a path outside the project (`t12`) | Exit 0 | Exit 0 | Listed in the Limits table |
| A junction in the project to a module outside | Exit 0 | Exit 0 | N19 |
| `go.work` with `use` or `replace` of a module outside, `GOWORK` not set | K0209 | **Exit 0** | N19 |
| The same, `GOWORK=off`, as `harness` sets it (`grade.go:150`) | K0209 | A go command error | Not run |

**`vendor` in a path.** The builder's `layout` leaves a package alone when `vendor` is in its path. That leaves no escape that I can find. An import of `lib/vendor/sub`, `api/orders/vendor/sub`, or of `sub` resolved in a leaf's `vendor` fails in both (`p1` to `p4`: K0008 in V, a go command error in L). An unimported package passes in both, also a `main` and a package with a type error (`p6`, `p7`), and it is not built. A part named `vendor` is not left alone: L reads it (`p5`). V skips that name, so it gives K0008. That is N22.

### New findings

#### N16. High. In L alone, a nested module under `.vulcan/build` passes every check, and on a case-insensitive host so does `GO.MOD`

- **Where:** `internal/rules/check.go`, `ForeignRoots`, called from `poc/golint/layout/layout.go`, `unlisted`, with `.vulcan/build` and `.vulcan/bin` to skip. L has no use for those directories. The name is compared with `d.Name() == "go.mod"` and `rel == "vendor/modules.txt"`, exactly.
- **Case `d2`, rules `ERR-08` and `ARCH-10`:** `go.mod` holds `require example.com/evil v0.0.0` and `replace example.com/evil => ./.vulcan/build/evil`. `.vulcan/build/evil/go.mod` is `module example.com/evil`, and `s.go` there calls `os.Exit(1)` and `net.Dial`. A leaf calls `evil.X()`. **L exit 0, and `go build ./cmd/shop` passes. V exit 1, K0008.**
- **Case `u1`, on Windows:** `third/GO.MOD` in place of `go.mod`. **L exit 0, and `go build` passes. V K0002.** `u4`: `vendor/MODULES.TXT`, L exit 0, V K0209. On Linux the go command does not read these names, and both reject. The harness can run on either: `.harness` holds `harness.exe` and Linux tools.
- **Why High:** it is the escape of H3, N1, and N10: code inside the project that leaves the model and each rule, with exit 0, in one arm. I gave those High, so I give this High. A form this unusual is unlikely in a run. That was true of N1 and N10 too.
- **Favors:** L passes more. A rule can land in U because of it.
- **Smallest fix:** read the `replace` lines of the root `go.mod`, and of a root `go.work`, and reject a relative target inside the root, in both arms. That closes each row above that the walk misses, by what actually brings the code in, and it needs no list of directories. Keep the walk for a `go.mod` that nothing replaces, if that is wanted. Compare the name of `modules.txt` with `strings.EqualFold`.

#### N17. High. With `defaults v1` on the base applications, each seeded case counts as rejected, and each base and reference fails

This is the separate question. It is not a difference in the rule kinds, and it is in both arms.

- **Where:** `poc/harness/grade/validate.go:198`, `it.Rejected = err != nil`, reads only the exit code of the check. `:215` and `:220` require `!it.Rejected` of a base and a reference. `:145`, `enforced = enforced && it.Rejected`, makes the label. `:235`, `caught = true` when the check rejects a seeded case of a test rule. `cmdValidate` writes the labels with `-write-labels` before it returns its error for the problems (`poc/harness/cmd/harness/main.go`).
- **What the code does:** plan-r3 adds the rules file beside the applications, and section 6 says that `harness validate` runs on the templates that `init` makes. The base applications give 284 and 285 findings. So each workspace exits non-zero before any seeded change is read.
- **Run:** scratch copies of the repository, the demo material, and two harmless seeded shapes of the behavior rule `RT`. Each adds one documented function that breaks nothing.

  | | No rules file | `defaults v1` from `init` |
  |---|---|---|
  | Base, L and V | Pass | **Fail** |
  | Reference, L and V | Pass | **Fail** |
  | `RT`, harmless shapes, L and V | Not rejected, and the item fails | **Rejected, and the item passes** |
  | `enforced by L`, `enforced by V` | `RD` | **`RD`, `RT`** |
  | `set S`, `set N` | `RD`; `RT` | **`RD`, `RT`; none** |

  The command does report the base and the reference as problems, and it exits 1. So a reader who reads the summary sees that something is wrong. The set line and the labels are wrong all the same.
- **The arms differ too.** With defaults on, the base in L has `builds true` and its holdout passes. In V it has `builds false`, and the holdout does not run, because `vulcan build` runs the check. So the reference has acceptance `{Passed:0 Failed:1}` in V and `{Passed:2 Failed:0}` in L. An agent's run in step 8 has the same difference.
- **Why High:** it makes the main measure of T1 meaningless. Each rule with a seeded shape lands in S, and N is empty. So the gate of section 5, at least 8 in S and 4 in N, would read 8 for a reason that has nothing to do with the rules, and would fail on N for the same reason. The failure of the base would show this, but no text in the plan says what to do instead.
- **Favors:** neither arm in S. V has the worse grading.
- **Smallest fix.** The design is the owner's. Each of these answers it, at another cost:
  - Compare with the base: a workspace is rejected only by a diagnostic that the base of its arm does not have. This keeps the plan's "rules file beside the applications". It needs a stable key for a diagnostic, without a line number. It does not stop a seeded case that breaks an unrelated default by the way, such as an exported name with no comment (CODE-08). So a label should also name the rule that fired.
  - Turn off, in the template, each default that fires on the base, by a decision that is written down, as plan-r3, section 4 allows. Then the exit code means what it did. This changes what T1 can enforce.
  - Label with the rules file that the rule under test needs, and no other. Then a seeded case is read against that rule alone.

  V's build needs an answer in each: `Grade` must get a binary from a project with findings, or each base finding must be off.

#### N18. Medium. In both arms, a nested module under `.git` or `.vulcan/bin` passes every check

- **Where:** `ForeignRoots` skips each directory named `.git`, at any depth, and `.vulcan/bin`. So `rules-file.md:88`, "It fails in each directory", does not hold.
- **Cases:** `gi` (`.git/evil` in a real repository), `d4` (`x/.git/evil`), `d1` (`.vulcan/bin/evil`): exit 0 in both, and each arm builds. `u2`, `u3` on Windows: `GO.MOD` under `testdata` or `_x`, the same.
- **Severity:** a hole in both arms, so Medium, as M11 and M12 were. It is no difference between the arms, and it can change no set but N.
- **Smallest fix:** the fix of N16 closes it.

#### Low

| # | Finding | Case | Where | Favors | Smallest fix |
|---|---|---|---|---|---|
| N19 | The Limits row names "a `replace` to a path outside". A junction in the project to a module outside passes in both. A `go.work` in the project with a module outside passes in L when `GOWORK` is not set, and V gives K0209, because V sets it off. `harness` sets it off, so a run does not reach this | Junction, `go.work` runs | `docs/poc/rules-file.md:118`, `internal/check/load.go:38` | L, outside the harness | Say so in the row. Have L report a root `go.work` |
| N20 | `rules-file.md` says that a port rule "`at field` ... can fire in L only". `at param` does too, on L's constructor (`r2`) | `r2`: V exit 0, L VL0807 | `docs/poc/rules-file.md:33` | V | Add `at param` |
| N21 | The fix of N11 reads the receiver by name. A local copy of it is a selector, so `q := p; q.save(x)` still counts more in L: L fires at 10, V at 16. A plain function in L that takes the value is the same | `m3q10` to `m3q14` | `shape.go`, `tokens` | V, near the limit | List it |
| N22 | V skips a directory by its name at any depth, so a part named `vendor`, as `lib/vendor`, cannot exist in V: K0008. L reads it | `p5` | `internal/project/project.go:153` | L | Skip by name at the root only, or say so |

### N11: the similarity counts

The builder's claim holds. Each row has the same pairs in both arms.

| Similarity (limit 60) | V | L |
|---:|---:|---:|
| 80 | 13 | 13 |
| 85 | 12 | 12 |
| 90 | 10 | 10 |
| 95 | 6 | 6 |
| 100 | 4 | 4 |

At similarity 90, the limits 30, 40, 50, 60, 70, 80, and 100 give 34, 12, 10, 10, 8, 7, and 6 in each arm, with the same pairs. By the probe, 121 of V's 131 functions have the same token list in L. The other 10 are constructors, which hold port parameters in L, and names that only one arm has at that place: wrappers in `ports.go`, and `startup` against `Serve`, each 112 tokens. `Handler` and `Refund` now agree.

### The counts

`coverage.json` gives `status` 8, 48, 154; `loose_status` 41, 15, 154; `auditor_loose_status` 34, 22, 154. The 7 rules that the two looser readings differ on are `ARCH-06`, `ERR-03`, `HTTP-01`, `HTTP-02`, `CONC-03`, `CODE-02`, and `CODE-04`, as in the second rerun. The 39 give 4, 34, 1 by the strict test. The 8 stated rules are `ARCH-01`, `ARCH-02`, `ARCH-07`, `ARCH-21`, `ERR-09`, `DATA-15`, `CODE-03`, and `CODE-06`. I found no break by a named limit for these. N13 is closed. The second rerun's lean of the looser test stays, and it is now printed.

### The upgrade

- **Source:** only the `go` line changed, in `poc/app/go/go.mod`, `poc/app/golint/go.mod`, and `poc/app/vulcan/go.mod`. `git diff 1f8a7fe HEAD -- poc/app` holds these three lines and nothing else.
- **The arms agree:** each base passes its check with no rules file. With `defaults v1`, the hash is `sha256:14c3ca9a...`, V gives 284 and L gives 285, with the same counts by rule, and the one difference by text is the row of `lib/shop`.
- **Behavior:** `go test ./poc/acceptance/...` passes against the binary of each of the three arms.
- **Not checked:** the sealed seeded cases. `QN-base-apps.md` says that T2 must confirm them, and that is right.

### Linux

With the host available, these hold there: `bt*` and `g6`, `g7` give K0901 and VL0901 for a tag of another system, and a rule's finding for a tag of this one, in both arms. So the same program can pass one host's read and fail another's, the same in both arms. `lc2`, `re1`, `dp4`, `m3e5`, `m3d10`, `fx1`, `fx3`, `fx5`, `ne*`, `t9`, `t10`, and `t14` give the results above. `t12` holds a Windows path, so that run says nothing. `as4`, `import "C"` with a C compiler, gives V K0804, and L VL0804 and VL0122. Both reject, and L also reads a directive that cgo generates.

### What I could not check

- The sealed seeded cases, as before.
- A real `https` source, as before.
- What the builder's note says of its own order of work.

## Fourth rerun

Reviewed at commit db10d23 (branch main), with e2b9441, d641fbd, db10d23, and plan-r3, amendment 10. The same reviewer ran this. The sections above stay as they were at 1e1df8a. This rerun is narrow, as the owner asked: the three decisions, plus each existing case script for a regression, on Windows and in WSL.

### Result

**Step 3's gate (no open High finding) does not pass.** N16, N17, N18, N20, and N22 are fixed as reported. Two High findings are open, and both are new here. N23 is a fifth form of the module escape, in L alone. N28 is a difference in how a run is graded: with a new finding, V does not build and L does, so `TaskSuccess` differs by arm. N28 reaches step 8, not T1.

| Severity | Open now | Which |
|---|---:|---|
| High | 2 | N23, N28 |
| Medium | 4 | N24, N25, N26, N27 |
| Low | 2 | N29, N30 |

### How I ran it

- A fresh build of db10d23, for Windows and for Linux. I reran `b1` to `b5`, `b7`, `c1`, `c2`, `a1` to `a3`, and `a5` to `a10`, with no change to their text, and compared them with the saved output of the third rerun. Of 658 rows, 24 changed. Each is a fix below, and none is a regression.
- I wrote about 80 new cases: `a12` and `a15` on the `replace` check on Windows, `lx3.sh` on Linux, `a13` on the baseline. I reran 36 cases in WSL.
- `harness validate` ran on the demo material of the repository, and on a copy with a third rule `RB` and my own shapes, in scratch copies of the repository: no rules file; `defaults v1`; `defaults v1` with a baseline.
- `go test ./internal/rules ./poc/harness/... ./poc/golint/... ./conformance` passes, with `-parallel 2` for `conformance`.

### Results for each finding

| Finding | Result | Run |
|---|---|---|
| N16 | Fixed for its forms | `d2`, `u1`, `u4`, `u5`: K0002, VL0002, on Windows and Linux. The check now holds a fifth form: N23 |
| N17 | Fixed on the demo | With a baseline: base and reference pass in both, V builds and runs its holdout. Harmless shapes are not rejected. `RT` stays in N. See "`harness validate`". The rest is N25 |
| N18 | Fixed for its forms | `d1`, `d4`, `gi`-style targets by a plain path: K0002, VL0002. The rest is N24 |
| N19 | Listed, acceptable | `rp26`, junction runs: as the Limits row now says |
| N20 | Fixed | `r1`, `r2` fire in L only, and `rules-file.md` now names `at field` and `at param` |
| N21 | Listed, acceptable | `m3q10` to `m3q14`: L fires at 10, V at 16. A Limits row states it |
| N22 | Fixed | `p5`: a part `lib/vendor` is read in both |
| Counts | Hold | `status` 8, 48, 154. `loose_status` 41, 15, 154. `auditor_loose_status` 34, 22, 154 |
| Base findings | Hold | V 284, L 285, hash `sha256:14c3ca9a...`. By key, only `startup` against `Serve`, and `lib/shop`. With a baseline, each exits 0, with 284 and 285 old, none stale |

### The `replace` check, attacked

The check compares paths by text, with `filepath.Rel`. It does not ask the file system. The walk still skips `.git`, `.vulcan/build`, and `.vulcan/bin` at the root.

| Target | V | L |
|---|---|---|
| A version on the left, a block form, quoted, `.\third`, `./a/../third`, `../<dir>/third`, an absolute path with either slash, another case of the drive or of a directory, a junction in the project (`rp1` to `rp10`, `rp16`) | K0002 | VL0002 |
| A non-canonical version, an unknown directive, a bad `go` line, also with `GOFLAGS=-mod=mod` (`rp17` to `rp20`, `rp22`) | K0002 | VL0002 |
| `\\?\` path, 8.3 path, or a junction outside the project, to a plain directory (`es_third_*`) | K0002 by the walk | VL0002 by the walk |
| **The same three, to `.vulcan/build/evil`** (`es_build_*`) | K0008 | **Exit 0, and `go build` passes** |
| **The same three, to `.git/evil` or `.vulcan/bin/evil`** (`es_git_*`, `es_bin_*`), in a real repository | **Exit 0, and it builds** | **Exit 0, and it builds** |
| **Linux: `/proc/self/cwd/.git/evil`, `/proc/self/cwd/.vulcan/bin/evil`** (`li_*_proc`) | K0008 | **Exit 0, and `go build` passes** |
| **Linux: a symlink outside the project, back into `.git` or `.vulcan/bin`** (`li_*_link`) | **Exit 0, and it builds** | **Exit 0, and it builds** |
| Linux: a plain absolute path, `//` in it (`li_abs`, `li_dbl`) | K0002 | VL0002 |
| `-modfile` in `GOFLAGS`, also by a `go env` file (`rp23`, `rp24`) | K0209 | The go command refuses |
| `-overlay` in `GOFLAGS` (`rp25`) | K0002, K0209 | VL0002 |
| A `go.work` above the root, `GOWORK` not set (`rp26`) | K0209 | **Exit 0**. Listed. With `GOWORK=off`, as `harness` sets it, L fails too |
| `Main("lib/evil")`, a plain package that a leaf imports (`e1`) | K0008 | VL0003, VL0004 |

#### N23. High. In L alone, a path that does not read as inside the project reaches `.vulcan/build`, or on Linux `.git` and `.vulcan/bin`

- **Where:** `internal/rules/check.go`, `ForeignRoots`, the function `inside`, compares text. The walk skips those directories, and `poc/golint/cmd/vulcan-lint/main.go` calls it.
- **Case `es_build_unc`, on Windows, rules `ERR-08` and `ARCH-10`:** `go.mod` holds `replace example.com/evil => \\?\C:\...\project\.vulcan\build\evil`. That directory holds `go.mod` and a `.go` file that calls `os.Exit(1)` and `net.Dial`. A leaf calls `evil.X()`. **L exit 0, and `go build ./cmd/shop` passes. V exit 1, K0008.** `es_build_short` (an 8.3 path) and `es_build_junction` (`../o/.vulcan/build/evil`, with `o` a junction to the project) give the same.
- **Case `li_git_proc`, on Linux:** `replace example.com/evil => /proc/self/cwd/.git/evil`. **L exit 0, and `go build` passes. V exit 1, K0008.**
- **Why High:** it is the escape of H3, N1, N10, and N16: code inside the project that leaves the model and each rule, with exit 0, in one arm. I gave those High, so I give this High. These paths are unusual. So were the others.
- **Favors:** L passes more. A rule can land in U because of it.
- **Smallest fix:** ask the file system. Walk up from the target, and compare each directory with the root by `os.SameFile`. Stop skipping directories in the walk, or keep only `.git/objects`. This covers text forms, links, and `/proc`.

#### N24. Medium. In both arms, the same paths reach `.git` and `.vulcan/bin`

`es_git_unc`, `es_git_short`, `es_git_junction`, `es_bin_unc`, `es_bin_short`, `es_bin_junction`, `li_git_link`, `li_bin_link`: exit 0 in both, and each builds. It is a hole in both, so Medium, as N18 was. A link outside the project is close to the listed limit of a module outside the project. A `\\?\` or 8.3 path is not: nothing outside the project exists. The fix of N23 closes it.

### The baseline, attacked

Each row is one project: `rules baseline`, then a change, then the check. Both arms gave the same result, the same counts of new, old, and stale, and equal keys, on each row. The one exception is the last row, where my case had a fault.

| Change | Result | Case | Note |
|---|---|---|---|
| An edit above; to another file; into a closure; a generic function | Old | `k1`, `k2`, `k8`, `k11` | As listed |
| One more in the function | New, the other old | `k3` | As listed |
| A rename; between parts; another call of the rule; from package level into a function | New, and stale | `k9`, `k13`, `k14`, `k16` | As listed |
| A swap in one function: another `os.Exit` | **Old** | `k4` | Listed: "Old, Loose" |
| ERR-03: `_ = h()` in place of `_ = g()` | **Old** | `k17` | The text does not name the call |
| SEC-07: `d.Query("select " + u + q)` in place of `d.Exec(q)` | **Old** | `k18` | The same. A worse statement passes |
| To a method of another type, of one name | **Old** | `k5` | Not listed as such. See N29 |
| A method to a plain function; a generic method to a plain function | Old | `k10`, `k12` | As listed |
| Package level: `var B = os.Exit` for `var A = os.Exit` | **Old** | `k15` | The key has no symbol there |
| A function `init` | No case | `k6`, `k7` | K0117 rejects `init` in a part, so the baseline refuses, in both |
| A test file, then code, of one function name | Old | `k19` | |
| A test and code of one name, one held; two files, one held | New is the later in order: `a_test`, `b` | `k20`, `k21` | The same in both arms. I expected this |
| A function with `ensures`, with `old`; a method of a type with an invariant | Old, equal keys | `k30`, `k31`, `k32` | V's lowering does not change a key |
| A status table with several states before | Old, 8 of 8 runs | `k34` | The text has one fixed order |
| A count of 98 added by hand | **Old, and stale** | `k23` | Each new finding of the key passes. The check prints K0811. This is the harm that `rules-file.md` names, for `PROTECTED` at a later step |
| A model check by hand: `K0205`, `VL0205`, `model-0205` | Rejected, and stale | `k24` | **A model check cannot get in.** No path in `Classify` reads one |
| ARCH-10 by hand, with its model check | K0205 stays | `k25` | The rule's finding is old, and the model rejects |
| An unknown rule, a bad line | K0801 in both for a bad line | `k26`, `k27` | My case also had an unused import, so its K0008 and build error say nothing |

**Equal keys.** Each row above writes the same file in both arms. On the base applications, 257 and 258 keys differ only by `startup` and `Serve`, and `lib/shop`. Where a fixture differs (`k33`, a port comment in `R01` that only L's form has), the keys differ with it.

**`-json`.** `vulcan-lint -json` puts a new finding on standard output, and old findings and stale entries on standard error, and it exits 0 with a new finding. `vulcan check --json` has them all, and exits 1. `harness` reads text and exit codes, so a run does not reach this. It matters only to a tool that reads the JSON of L. N30 records the exit code, which the Limits row does not.

#### N28. High. With a new finding, V does not build and L does, so `TaskSuccess` differs by arm

- **Where:** `poc/harness/record/record.go:155`, `TaskSuccess` is `r.Builds && r.Acceptance.OK()`. `poc/harness/grade/grade.go:164` builds V with `vulcan build`, which refuses a new finding, and `:166` builds L with `go build`, which reads no rule. Neither `TaskSuccess` nor `CleanSuccess` reads the final check.
- **Run `k35`, one new finding of `ERR-08`:**

  | | Check | Build | Test |
  |---|---|---|---|
  | V | `vulcan check` 1 | `vulcan build` **1** | `vulcan test` **1** |
  | L | `vulcan-lint` 3 | `go build` **0** | `go test` **0** |

- **Effect:** an agent that leaves one new finding of any default, such as an exported name with no comment (CODE-08), has a failed task in V. In L the same run can be a clean success. This form existed for a model error in round 2. `defaults v1` widens it to each of its 61 rules at `error`. `N17-demo.md` notes the build, and it does not name M1 or M3.
- **Why High:** a run's success differs by arm on an ordinary form of code. That is the scale of the first High of `parity-r2.md`. It does not reach T1, which has no agent run. It reaches step 8.
- **Favors:** L on M1 and M3. An agent in L also has less to fix before it can run a test.
- **Smallest fix, one of these. The choice is the owner's:**
  - Grade with the final check in both arms: a run with a new finding is no task success, in L too.
  - Give the grader a V build that does not refuse a rule's finding, and record the check apart, in both arms.

  Each gives one meaning of `Builds`.

### `harness validate`

**No rules file:** on the repository's demo material, the output is the same, byte for byte, as my run at 02f2464. `TestValidateNoRulesFile` passes.

**`defaults v1` and a baseline, with my shapes of a rule `RB`, "Code logs through log/slog, not through log", and a match file that says only `OBS-01` matches:**

| Shape | Fired, L and V | Item, and label |
|---|---|---|
| `count`: one more `log.Printf` beside an old one | `OBS-01` | Rejected, and matched |
| `both`: that, and an exported name with no comment | `CODE-08`, `OBS-01` | Rejected, and matched |
| `unrelated`: only an exported name with no comment | `CODE-08` | "rejected only by an unrelated rule, so it does not count" |
| `harmless` | None | Not rejected |
| `swap`: the old `log.Printf` with another text | None | Not rejected. The key holds the count |

- `count` with `both`: `RB` in S, in both arms. `count` with `unrelated`: `RB` in N. `count` with `swap`: `RB` in N. All read correctly, with the same caveat: a seeded case that only changes an old finding cannot be caught. A labeler must seed a new one.
- A `count` case is rejected by a rule that already fires there, so an old finding's rule still labels correctly.

#### N25. Medium. With no baseline and a match file, the base fails and the labels are still written

- **Where:** `poc/harness/grade/validate.go`, `noLabels` looks at a missing match. It does not look at a failed base. `cmdValidate` writes labels with `-write-labels`, and then returns its error.
- **Run:** defaults on, no baseline, a match file with `RB` and `OBS-01` true, `-only seeded/RB/harmless,base,reference -write-labels`. Base and reference fail in both. `RB/harmless` fired `OBS-01` from an old finding, so it matched. **`set S: 1 (RB)`, and `meta.json` holds `enforced_by_v: true`, `enforced_by_l: true`.** The exit is 1, and the problems print.
- **Severity:** it is N17 again, in a smaller form, and only if the template lacks a baseline. So Medium. The plan needs the baseline at step 4.
- **Smallest fix:** write no label when a base item is rejected.

#### N26. Medium. `-export-judge` shows the arm through the words of a model check

- **Where:** `poc/harness/grade/fired.go`, `armWords` and `neutral`. The builder calls it a best effort.
- **File names, paths, and code prefixes:** clean, on my run. No path, no `K` or `VL`, no arm name, no part. `validation.json` holds the arm, and the judge does not read it.
- **Wording is not clean.** On the demo, `RD` and `model-0205` read: "uses the network function database/sql.Open, but leaf Inventory is not a gateway or an entry leaf; move the call to a gateway leaf (gateway for <External>) and reach it through a port (§5.4c rule 1)". `gateway for <External>` and `§5.4c` are V's. L's text for the same code is another sentence. By a scan of the format strings after `neutral`, V's messages hold `§` (9), `container` (6), `component` (3); L's hold `contract.` (8), `checked` (3), `wrapper` (2), `main` (17 against 4).
- **A code can name the arm by itself.** 11 digits exist in L only, such as `model-0909`, `model-0907`, `model-0908`. 14 exist in V only, such as `model-0105`, `model-0111`. A pair with one of these says which arm fired.
- **The text that a pair shows is the least of the two by sort order**, so a judge of a shared code reads one arm's sentence and decides for both.
- **Severity:** plan-r3, section 5 says that the judge does not see the arm. A decision on a shared code applies to both arms, so the leak can bias little there. On a code of one arm it can bias a label. So Medium.
- **Smallest fix:** one neutral sentence for each digit, in a table, used in the export. Export a one-arm code with the same sentence form and no digit, or list those pairs apart.

#### N27. Medium. In both arms no rule reads `lib/contract`, and V has no reason for that

- **Where:** `internal/rules/check.go`, `CheckUnit`, returns nothing for `IsRuntimeLib(u.Part.Dir)`, in each arm. That is right for L, whose contract package is that directory (call 17). In V a library with that name is an ordinary library.
- **Case `e4`, rule `ERR-08`:** `lib/contract/evil` calls `os.Exit(1)`, and a leaf calls it. **V exit 0, L exit 0.** The model still reads it (`e3`: K0205, VL0205).
- **Under the harness:** L's `PROTECTED` lists `lib/contract/*.go`, so a new file there is a configuration change. V has no such list, and a new library is allowed. So an agent in V can use it, and one in L cannot.
- **Severity:** rules only, and the model stays. So Medium.
- **Favors:** V passes more. **Smallest fix:** give `Unit` a field, as `Generates`, so that only L skips it.

#### Low

| # | Finding | Case | Where | Favors | Smallest fix |
|---|---|---|---|---|---|
| N29 | The key is looser than the table of `rules-file.md` says: a finding moves between types of one method name (`k5`), and at package level between variables (`k15`), and stays old. The table lists a swap in one function and a method to a plain function. A text with no detail of the call makes a swap harder to see (`k17`, `k18`) | `k5`, `k15`, `k17`, `k18` | `internal/rules/baseline.go`, `shape.go` texts | Neither | Two rows in the table. A call's name in the text of `discard-error` and `call-form` |
| N30 | `vulcan-lint -json` exits 0 with a new finding, and `vulcan check --json` exits 1. The Limits row lists what is left out of the JSON, and not the exit | `k35` | `docs/poc/rules-file.md`, the Limits table | L, outside the harness | One sentence |

### A choice for the owner: a model check as a rule that fired

On the demo, `RD`, "Only store opens a database", has seeded cases that fire `ARCH-01`, `ARCH-10`, `ARCH-11`, and `model-0205`, in both arms.

| Reading | Judge's decisions | `RD` | S | N |
|---|---|---|---:|---:|
| A model check counts | `model-0205` yes, the rest no | Enforced | `RD` | `RB`, `RT` |
| It does not count | The rest no | **Not enforced** | none | `RB`, `RD`, `RT` |
| It does not count, and a default says the same | `ARCH-10` yes | Enforced | `RD` | `RB`, `RT` |
| No rules file (round 2's form) | None | Enforced by any rejection | `RD` | `RT` |

**If it counts:** the count is closer to round 2's, which read any rejection. A team rule that only the model states can be enforced. It leans toward more enforced rules, and so toward the gate. Each code of one arm can put a rule in G or U, by a form that only that arm has: the 11 and 14 of N26. So it leans away from an empty U and G.

**If it does not count:** the count reads only what the defaults add, which is what QD asks of the defaults. It leans toward fewer enforced rules, and away from the gate. A rule that the model rejects in both arms reads as not enforced, though both arms reject each case. That understates what each arm does. It leans toward an empty U and G. Six default rules restate a model check, so a judge's yes on those can give the same label as the third row.

### What I could not check

- The sealed seeded cases, as before.
- A real `https` source, as before.
- A fresh judge's decisions. Mine are for the mechanism.
- An agent's run. N28 is from `k35` and the code of `grade.go` and `record.go`.

## Fifth rerun

Reviewed at commit a4056dc (branch main), with 02cedb5, a4056dc, and plan-r3, amendment 11 (d94a9af). The same reviewer ran this. The sections above stay as they were at b4ace44. This rerun is narrow, as the owner asked: the fixes of the fourth rerun, the sets, and each existing case script for a regression, on Windows and in WSL where a host can change a result.

### Result

**Step 3's gate (no open High finding) does not pass.** One High finding is open, and it is new here. N31 is a sixth form of the module escape, in L alone, on Windows. It needs a directory outside the project. Section "Why N31 is High" gives what speaks for and against that rating, because the gate turns on it.

N23, N25, N26, N27, N29, and N30 are fixed or listed as reported. N24 is fixed except for one form (N32). The part of N28 for a rule's finding is fixed. The part for a model error stays, and I rate it Medium (N33). The fix of N28 adds a new difference (N34).

| Severity | Open now | Which |
|---|---:|---|
| High | 1 | N31 |
| Medium | 3 | N32, N33, N34 |
| Low | 4 | N35, N36, N37, N38 |

Listed, and not counted as open, as before: M8, M11, M12, M13, L6, L8, L9, L10, N6, N8, N19, N21, and now N30.

### How I ran it

- A fresh build of a4056dc, for Windows and for Linux. I reran `b1` to `b5`, `b7`, `c1`, `c2`, `a1` to `a3`, `a5` to `a10`, `a12`, and `a13`. `cmp.py` compares each case with the fourth rerun's output.
- New scripts in my scratch directory: `a17`, `a18`, `a23` (the escape on Windows), `lx4.sh` (the escape on Linux), `a20`, `a22` (the sets and the export), `a21` (the binary, an old baseline, the exits of `-json`), `a24`, `a25` (the names of calls), `audit5_test.go` (the real `Grader.Grade`, in exports of a4056dc and of b4ace44), and `lxc/main.go` (the text of `copyTemplate`, on Linux).
- I read nothing under `C:\Users\jonat\vulcan-poc`. I changed no file but this report, and I did not freeze `defaults v1`.

### Results for each finding

| Finding | Result | Run |
|---|---|---|
| N23 | Fixed for its forms. `es_build_unc`, `es_build_short`, `es_build_junction`, and each `li_*_proc`: K0002, VL0002. A sixth form: N31 | `a17`, `lx4.sh` |
| N24 | Fixed for its forms. `es_git_*`, `es_bin_*`, `li_*_link`: K0002, VL0002. One form is left: N32 | `a17`, `lx4.sh` |
| N25 | Fixed. With no baseline and a match file, the report says "no enforced label: the check rejects the base of arm L, V", and no labels file is written | `a20`, repo `c` |
| N26 | Fixed for its words. See the export scan. A smaller point: N35 | `a20`, `a22` |
| N27 | Fixed. `e4`: V exit 1, K0802, on a library named `contract`. L exit 0 on its `lib/contract`, which `poc/app/golint/PROTECTED` lists, and call 17 of `rules-file.md` states this | `a12` |
| N28 | Fixed for a rule's finding. Before (b4ace44): V `builds=false`, success false. After (a4056dc): V `builds=true`, success true, with `final_check` recorded. A model error stays: N33 | `audit5_test.go` |
| N29 | Fixed. `k17`, `k18`: new in both arms, with the old entry stale. `k5`, `k15` stay old, and the table of `rules-file.md` now states both moves. A port call has one text and one key in both arms (`a25`). An old baseline: N36 | `a13`, `a24`, `a25` |
| N30 | Listed. V exit 1, L exit 0. `rules-file.md` line 169 states it | `a21` |

**Regression.** 658 cases compared, and 2 changed: `a10 d1` and `a10 d3` each now report the nested `go.mod` as well as the `replace`, in both arms. Both stay K0002, VL0002, with the same exits. No verdict changed.

**Counts.** Unchanged: strict 8, 48, 154. Looser 41, 15, 154. Auditor's looser 34, 22, 154. Base findings: V 284, L 285. Keys: V 257, L 258. Hash of `defaults v1`: `sha256:14c3ca9a9ac1603d5227ff4275375f2fd38bded91be26887b921929972a6f427`, in both arms.

**Tests.** `go test ./conformance -parallel 2`, `./poc/harness/...`, and `./poc/golint/...` pass, on an export of a4056dc. `./internal/rules` passes in the repository. On that export its `TestCoverage` fails, and the repository has `core.autocrlf` true, so I read that as the line ends of my export (`git archive`), and not as a fault of the code.

### The `os.SameFile` walk, attacked

A leaf calls `evil.X()`, and `evil` calls `os.Exit` and `net.Dial`, with rules `ERR-08` and `ARCH-10`.

| Attack | V | L | Note |
|---|---|---|---|
| A target that does not exist, inside (`./later/evil`) | K0002 | VL0002 | A directory above it is the root |
| A target that does not exist, outside (`../later/evil`) | K0008 | exit 1 | The go command refuses it. So a directory made after the check needs a failing check first |
| A broken junction in the project | K0002 | VL0002 | |
| A broken junction outside | K0008 | exit 1 | |
| A cycle of junctions outside | K0008 | exit 1 | |
| A case-only difference (`./.GIT/OBJECTS/EVIL`) | K0002 | VL0002 | |
| `\\?\`, 8.3, a junction to the root, an admin share, `::$INDEX_ALLOCATION`, to each of `.git/evil`, `.vulcan/bin/evil`, `third`, `.vulcan/build/evil`, `.git/objects/evil` | K0002 | VL0002 | 15 forms, plus 3 |
| A volume path (`\\?\Volume{...}\`) | exit 1 | exit 1 | `go.mod` does not parse |
| A junction inside the project, to anything | K0002 | VL0002 | Stricter than `rules-file.md` line 164: N37 |
| A junction outside, directly to `.git/evil`, `.vulcan/bin/evil`, or `third` | K0002 | VL0002 | The walk finds the `go.mod`, not the `replace` check |
| **A junction outside, directly to `.vulcan/build/evil`** (`es_build_direct`, `ct_bld`). Also a junction to `.vulcan`, then `build/evil` (`es_up2`) | K0008 | **exit 0**, builds | **N31** |
| **A junction outside, directly to `.git/objects/evil`** (`es_obj_direct`, `ct_obj`). Also two junctions (`es_chain`), and a junction to `.git`, then `objects/evil` (`es_up`) | **exit 0**, builds | **exit 0**, builds | **N32** |
| A `go.mod` in `.git/objects/evil` or `.vulcan/build/evil`, with nothing that replaces it | exit 0 | exit 0 | It brings in nothing, as the code comment says |
| `import "shop/.vulcan/build/evil"`, with no replace | K0008 | exit 3 | |
| Linux: `/proc/self/cwd/`, a symlink to the root, a symlink directly to the target, an absolute path, for each of the five directories | K0002 | VL0002 | 15 forms, plus 1. `filepath.EvalSymlinks` follows a symlink, so N31 and N32 do not occur here |

Controls: a plain `../out` module outside the project exits 0 and builds in both (the listed limit). A plain `./third` inside gives K0002, VL0002.

### N28: the model error and the copy

**How often.** The repository holds one set of agent runs, `experiments/poc/data/p5-deepseek/runs.jsonl`: V and G, with no L. In it, 4 of 36 V runs did not build, and 0 of 36 G runs. All 4 are stopped runs. Their last recorded check: ok in 2, K0002 in 1, none in 1. So an upper bound is 4 in 36 (11 points), and the only one with a known model error is 1 in 36 (3 points). Each of the 4 would still need to pass acceptance in L's form to change a result, and I cannot say how many would. I have no round-2 data to count: it is not in this repository.

**The effect, by the real grader on the demo.** One run, one difference:

| Final code | V builds | V acceptance | V success | L builds | L acceptance | L success |
|---|---|---|---|---|---|---|
| Clean | true | 2 of 2 | true | true | 2 of 2 | true |
| One new rule finding, before (b4ace44) | **false** | failed | **false** | true | 2 of 2 | true |
| One new rule finding, after (a4056dc) | true | 2 of 2 | true | true | 2 of 2 | true |
| One model error, before and after | **false** | failed | **false** | true | 2 of 2 | **true** |

In V the holdout and each rule test also do not run (`grade.go:89`).

**Did plan-r2's parity accept it?** Not by name. The code was the same in round 2 (`git show e26680c:poc/harness/grade/grade.go`, lines 154 to 170). `parity-r2.md` records the two build commands and does not discuss the difference. `plan.md` defines M3 as "The acceptance tests pass, whatever the rules", and no text of `plan.md` or `plan-r2.md` says that a model error fails a task in one arm only. Amendment 11 names "a rule's finding", and it does not name a model error. The sentence "A model error stays as in round 2" is in the commit message of 02cedb5 and in a comment in `grade.go`.

**Does the copy give the same binary?**

| Comparison | Result |
|---|---|
| `vulcan build`, with the rules files and with none, from one path | Byte for byte equal: `73f14ed09172` both. The rules files do not change the code |
| Two real builds in one directory, at b4ace44 | Equal: `ffc9caecf579` |
| The graded binary (from `workspace-no-rules`) and a real build (from `workspace`), at a4056dc | Differ: 17804800 and 17803776 bytes. The graded one holds the text `-no-rules` 21 times. `go version -m` differs only in the first line, the file path |

So the copy changes the source paths in the binary, and nothing else that I found. A difference in behavior needs a program that prints its own source path, such as a stack trace. The copy also drops `.git`, so V's binary has no VCS stamp where L's can have one. That reads only through `debug.ReadBuildInfo`. I did not run a project with `.git` through this comparison.

**What the copy cannot do:** N34.

**The fail-first gaps.** I ran the "before" myself. For N28, the same test on an export of b4ace44 shows V `builds=false` on a new finding, so the test would have failed first. For N26, the export of the fourth rerun's tool gives `model-0205` with the text "api/inventory uses the network function database/sql.Open, but leaf Inventory is not a gateway or an entry leaf; ...". The new export gives `check-8f638d44` and one sentence. So each fix does what it says. The gap hides N34: `TestBuildsMeansCompiles` grades only files that the copy keeps.

### The sets

| Check | Result |
|---|---|
| The demo, with `model-0205` matching `RD`, as each arm ships | S `RD`, N `RT`. As `N17-demo.md` says |
| The demo, for the rules file alone | N `RD`, `RT`. As `N17-demo.md` says |
| The demo: decided by a code of one arm | None. As it says |
| My shape: `RL`, L's `main` calls a function of a leaf, `model-0907`, in L only | L enforces, V does not: G `RL`. Rules file alone: N. Listed: "decided by a code of one arm: RL (model-0907)" |
| My shape: `RV`, a wire names a missing constructor, `model-0107`, in V only | V enforces, L does not: U `RV`. Rules file alone: N. Listed: "RV (model-0107)" |
| My shape: `RC`, a type error, with the match false | V fires `model-0008`, L fires nothing. Each is "not enforced", N |
| The table of 69 codes in `neutral.go`, each `arm` mark, against the codes in each arm's source | No code is missing. Each one-arm mark holds: 19 V, 8 L |

These are correct on my shapes. One limit: `decidedByOneArm` reads model codes only. A rule kind that fires in one arm by form (`at field`, `at param`) shows in G or U, and it is not on this list. `rules-file.md` states that form. On the demo, my scratch repository also reports problems for `detector RD`, and those do not touch the sets. `TestValidateWithRules` in the repository runs this demo with its detector.

### The export, scanned

Run on the demo and on my shapes, with pairs from each arm.

| Scan | Result |
|---|---|
| A field or a value that names an arm | None |
| A path, a file extension (`.go`, `.vulcan`), a K or VL code, `model-` with digits | None. The only hit of my list is the text `ARCH-` in a rule's id, in both arms |
| Hashes that differ by arm | None. The id comes from the digits alone, so `model-0205` is `check-8f638d44` for each arm |
| The order of pairs | Sorted by team rule, then by id. Nothing from an arm |
| A pair from one arm | It has the same form as any other. `RL` and `RV` show this |
| What is left | N35 |

### New findings

#### N31. High. In L alone, on Windows, a junction outside the project reaches `.vulcan/build`

- **Where:** `internal/rules/check.go:454`, `insideRoot`. It walks up the path as written, and then the path from `filepath.EvalSymlinks`. By this result, neither reaches the root for a junction outside the project that leads to a directory inside it. The later walk for a `go.mod` skips `.vulcan/build`.
- **Case `ct_bld`:** `replace example.com/evil => ../pL`, with `pL` a junction to `<project>/.vulcan/build/evil`. That directory holds `go.mod` and a `.go` file that calls `os.Exit(1)` and `net.Dial`. A leaf calls `evil.X()`. **L exit 0, and `go build ./cmd/shop` passes. V exit 1, K0008**, as `vulcan check` clears that directory. `es_up2`, a junction to `.vulcan`, then `build/evil`, gives the same.
- **The text:** `rules-file.md` line 90 says "a link outside the project that leads back in is inside". This case does not hold to that.
- **Why High:** it is the escape of H3, N1, N10, N16, and N23: code inside the project that leaves the model and each rule, with exit 0, in one arm. `es_build_junction` needed a junction outside the project too, and it was part of N23's High.
- **Favors:** L passes more. A rule can land in U because of it.
- **Smallest fix:** in the walk, at each step that is a reparse point, follow it with `os.Readlink`, which reads a junction, and walk again. Or walk the two skipped directories for a directory that `os.SameFile` matches with a target.

##### Why N31 is High

The gate turns on this rating, so here is each side. I did not change the bar, so it stays High.

- **For High:** the wording of N23 fits it, word for word. The form is L only, with exit 0, and the bytes are inside the project.
- **Against, as context:** it needs a directory entry outside the project, and it occurs on Windows only. With that same power, a plain `../out` module already exits 0 in both arms (`ct_out`, the listed limit), so an agent in V can take code out of the rules too. So N31 gives L no power that V lacks. It differs only in where the bytes lie.

If the owner lists this form with the module outside the project, the High count is 0. That choice is the owner's, and this report does not make it.

#### N32. Medium. In both arms, on Windows, a junction outside the project reaches `.git/objects`

`es_obj_direct`, `ct_obj`, `es_chain`, `es_up`: exit 0 in both, and each builds. It is a hole in both, so Medium, as N18 and N24 were. The same fix as N31 closes it.

#### N33. Medium. With a model error in the final code, V does not build and L does, so `TaskSuccess` differs by arm

- **Where:** `poc/harness/grade/grade.go:166` to `:187`, `poc/harness/record/record.go:165`. `TaskSuccess` does not read `final_check`.
- **Case:** the table above. One model error, with behavior that passes: V success false, L success true.
- **Why Medium:** this is N28's form on a smaller set. N28 was High because it hit "an ordinary form of code": one new finding of any of 61 rules, which an agent can leave and still test in L. A model error stops `vulcan check`, `vulcan build`, and `vulcan test` in V at once, and it prints exit 3 in L, so a final state with one comes from a run that stopped or gave up. In the only data here, that is at most 4 of 36 runs, and 1 of 36 with a known model error. It rises to the level of N28 only if such ends are common in step 8. That is not measured.
- **Against this rating:** read as "a run's success differs by arm", it is High. The upper bound of 11 points is above the 10 points of `plan.md`'s "Promising" limit on ordinary tasks, though not all 4 would pass in L. I rate by the frequency that I can show, and I state this so that the owner can read it the other way.
- **Favors:** L on M3 and M1.
- **Smallest fix, one of these. The choice is the owner's:**
  - In `TaskSuccess` for a run with a rules file, count a model code in `final_check` as no success, in both arms.
  - State in plan-r3 that M3 keeps this difference, as a cost of V's form, and report in step 8 how many runs it decided. `final_check.fired` gives the count.

#### N34. Medium. The grader's copy can fail or hang in V, on a workspace that L builds

- **Where:** `poc/harness/grade/grade.go:177`, `copyTemplate` at `poc/harness/grade/validate.go:361`. It was written for templates, and it now reads an agent's final workspace. It is new in 02cedb5, and only with a rules file.
- **Cases, by the real grader on Windows, before and after:**

  | Final workspace | V before | V after | L |
  |---|---|---|---|
  | `//go:embed seed.db` in a leaf (the copy drops `*.db`) | builds, success | **no build, no success** | builds, success |
  | A junction to a directory, anywhere in the workspace, used by nothing | builds, success | **no build, no success** | builds, success |

- **Cases on Linux, by the same function's text alone:** a symlink to a directory, a broken symlink, and a socket each give an error, so V would not build. A FIFO blocks: **after 6 seconds the copy had not returned**, and the build step has no time limit that I found around it. I did not run these through the grader. Symlinks could not be created in my Windows test, so that row has no Windows result.
- **Why Medium:** a run's success differs by arm, in V's disfavor, with code that passes. The forms are not common, so it is a smaller version of N28.
- **Favors:** L on M3 and M1. The FIFO costs a grader's run, in V only.
- **Smallest fix:** copy each regular file except the four rules files, including `*.db`, and pass over what is not a regular file. Better for the embed case: a grader-only way to run the real build in the workspace with no rule's refusal. A test with an embedded file and a link.

#### Low

| # | Finding | Case | Where | Favors | Smallest fix |
|---|---|---|---|---|---|
| N35 | The sentences of one-arm codes describe a form that one arm has: "The start of the program..." (L), "A wiring...", "A startup function..." (V). The id is a plain SHA-256 of `model-NNNN`, so 10,000 tries reverse it, and `neutral.go` holds the arm. A judge sees only the pair, and plan-r3 says that a judge does not read this repository. A judge who knows both forms could guess an arm on 27 codes | `a22`, `RL`, `RV` | `poc/harness/grade/neutral.go` | Neither | Accept and list it, or add a per-export salt to the id |
| N36 | N29 changes the text in 6 keys and the hash of `defaults v1` stays. A baseline from the prior tool then fails: V exit 1, L exit 3, 6 new findings and 6 stale. `rules baseline` clears it. No baseline exists outside this repository, and `defaults v1` is not frozen, so nothing is broken today. After the freeze, a change of a message would do this with no change of the hash | `a21` | `internal/rules/baseline.go`, `shape.go` | Neither | One line in `rules-file.md`: the tool's version is part of what a baseline holds, or a version in the file |
| N37 | `rules-file.md` line 164 says that a link or a junction in the project to a module outside passes. It gives K0002, VL0002 in both (`ct_in_out`, `ct_nest`). The check is stricter than the text. Line 90 is N31 | `ct_in_out`, `ct_nest` | `docs/poc/rules-file.md:164` | Neither | Change the row |
| N38 | `unless-comment` takes the doc comment above a one-line function as the reason. `// Drop is.` over `func Drop() { _ = g() }` passes `ERR-03` in both, and the same code on separate lines fails in both. Equal, and not listed | `a25`, first form | `internal/rules/shape.go` | Neither | One row in Limits, or read only a comment inside the body |

### What I could not check

- The sealed seeded cases, as before.
- A real `https` source, as before.
- A fresh judge's decisions. Mine are for the mechanism.
- An agent's run. N33 and N34 are from the real grader on a seeded final state. How often a run ends in such a state in step 8 is not measured.
- Round-2 run data, which is not in this repository. The frequency in N33 is from `p5-deepseek` alone, with arms V and G.
- Windows symlinks, which my account could not create. Junctions stand in for them, and `EvalSymlinks` was tested on Linux.
- N34's Linux rows through the whole grader.

## Sixth rerun

Reviewed at commit 4752184 (branch main), with 4f4eecc, 8bd9c25, and plan-r3, amendment 12. 4752184 adds only `docs/poc/hypotheses-views.md`. The same reviewer ran this. The sections above stay as they were at 6038e4a. This rerun is narrow, as the owner asked: the fixes of the fifth rerun, the forms that can bring code into a build, the grader's copy, and each existing case script for a regression.

### Result

**Step 3's gate (no open High finding) does not pass.** Two High findings are open. Both are new here, and both are older than the fixes: the fixes did not add them. Each is code that the build holds and that no rule and no model check reads, with exit 0, in L alone:

- **N39.** An assembly file and a function with no body in L's `main`. H6 reads only leaves and libraries, and V has no `main` that a person writes.
- **N40.** A leaf's directory that is a junction. `go ./...` does not follow it, so L's check never loads the leaf, and `go build` does. V gives K0003.

Amendment 12 closes N31, N32, and N8 for each form of `replace` that I tried, in both arms. N33's fix holds as stated, with one Low point about a count (N42). N34's fix holds on the grading path that the harness uses. On a Linux host, two forms are left (N43). A form that both arms share is new: the `ignore` directive of `go.mod` (N41).

N35, N36, and N38 are now stated in `rules-file.md`, in judgment calls 35 and 36 and in the paragraph on baselines. I count them as listed, not open. N37's row is changed, and its new reason is wrong (a new form, Low).

| Severity | Open now | Which |
|---|---:|---|
| High | 2 | N39, N40 |
| Medium | 1 | N41 |
| Low | 3 | N37, N42, N43 |

Listed, and not counted as open, as before: M8, M11, M12, M13, L6, L8, L9, L10, N6, N19, N21, N30, and now N35, N36, N38. N8 is closed by amendment 12.

### How I ran it

- A fresh build of 4752184, for Windows and for Linux, and an export of it (`git archive`) for tests that call the grader's own functions.
- I reran 21 case scripts: those of the fifth rerun's regression, and `a4` and `c3`. `cmp6.py` compares each case with the fifth rerun's output.
- New scripts in my scratch directory: `a26`, `a27` (assembly and the reads of L's `main`), `a28` (a parent `go.work`, `-modfile`, the `ignore` directive, a module from a file proxy), `a29` (the `ignore` directive), `a30`, `lx6.sh`, and `audit6_test.go` (the grader's `copyForBuild`, then each arm's build, on Windows and on Linux), and junction runs on Windows (`s5` to `s8`).
- I read nothing under `C:\Users\jonat\vulcan-poc` and no sealed case. I changed no file but this report, and I did not freeze `defaults v1`.

### 1. What brings code into a build

| Form | V | L | Note |
|---|---|---|---|
| `replace` to a junction outside, to `.vulcan/build/evil` (N31: `ct_bld`) | K0002 | VL0002 | Closed |
| The same to `.git/objects/evil` (N32: `ct_obj`) | K0002 | VL0002 | Closed |
| `replace` to `../out` (N8: `ct_out`), `./third`, and each junction form inside (`ct_in*`, `ct_nest`) | K0002 | VL0002 | Closed, by the text alone |
| A `go.work` in the parent directory with a `replace`, `GOWORK` not set (`pw`) | K0209, no build | **exit 0, builds** | The listed limit, as the row of Limits states it. With `GOWORK=off`, as the grader and the runner set it: both fail (`pw_off`) |
| `GOFLAGS=-modfile=` a second file with a local `replace` (`modfile`) | K0209 | exit 1 | Both checks fail. The grader and the runner set `GOFLAGS` themselves, so a workspace cannot set it |
| A module by its module path from a file proxy outside, by `require` or by a `replace` with a version (`proxy`, `proxyrep`) | K0209 | exit 0, builds | The listed limit. V fails here only for want of a `go.sum` line. The grader sets `GOPROXY=off`, and the sandbox has its own module cache, so an agent cannot add a module there |
| An assembly file in L's `main` (`m_asm`) | No such place | **exit 0, builds, runs** | **N39** |
| A leaf's directory that is a junction, to `_keep`, `testdata`, `.keep` inside, or to a place outside (`s7*`, `s6`) | K0003 | **exit 0, builds** | **N40** |
| The `ignore` directive of `go.mod` on a part (`i1` to `i7`) | **exit 0, builds** | **exit 0, builds** | **N41** |
| A junction in the project to a module outside, with nothing that replaces it (`s8`) | exit 0 | exit 0 | Harmless, because no `replace` can reach it. `rules-file.md` says it fails: N37 |

**The backslash fallback (`refused`).** On Windows, R300 to R305 pass in both arms. The function reads a line only when the parser's error says "Windows path", and x/mod v0.42.0 gives that error only for a directory path with a backslash on a host that is not Windows. The go command uses the same parser, so it refuses the same file. I found no text that the go command accepts and the check reads otherwise. I did not run the Linux side myself. `TestReplaceByFileSystem` covers it in the repository.

**The build `go.mod` (`internal/gen/leaves.go`).** With the rewrite removed, the project's `go.mod` is copied as it is, and the runtime's `require` and `replace` are added. Each V build in this rerun and in the conformance tests used it, and each one built. A project's `go.mod` can no longer hold a relative `replace`, so the rewrite has nothing left to change.

### 2. N33

- **A shared model code is no success in both arms.** `TaskSuccess` reads `final_check.model`. In V, such a code also stops the build, so V fails M3 in either case. In L the build passes, and the code now fails M3. `TestModelError` shows each case.
- **One-arm codes are counted apart.** The table of `neutral.go` has 19 codes of V alone, 8 of L alone, and 42 shared, as `rules-file.md` says. A one-arm code goes to `one_arm` and does not fail M3. In V such a code still stops the build, so V fails M3 there, and `CountModel.OneArm` counts those runs. That is the difference as stated.
- **"An unknown code counts as shared."** The rule is the same in both arms. Its effect falls on L alone, because in V each model code already stops the build. So it moves L toward V's result, never the other way. `TestModelChecksTable` holds the table to the code, so the guard does not act today.
- **The count `Decided`:** N42.

### 3. N34

| Final workspace | Host | V: check, copy, build | L: check, build |
|---|---|---|---|
| Control | Windows, Linux | ok, ok, builds | ok, builds |
| A source file that is a symlink | Linux | ok, ok, **no build** (`undefined: Z`) | ok, **builds** |
| The same link, made in WSL on the Windows drive, as the sandbox would | Windows | **fail** (cannot open) | **fail**, no build |
| A file with mode 000 | Linux | ok, **copy error** | ok, **builds** |
| A directory with mode 000 | Linux | fail, copy error | fail, builds |
| `chmod 000` made in WSL on the Windows drive | Windows | No effect: the drive has no metadata, and the file stays readable | |
| A FIFO | Linux | ok, ok, builds | ok, builds |
| A FIFO made in WSL on the Windows drive | Windows | `mkfifo` is not supported there | |
| A leaf's directory that is a junction | Windows | K0003 | exit 0, builds (N40) |

The harness runs an agent in WSL with the workspace on the Windows drive, and it grades on Windows. On that path, the link and the mode give the same result in both arms, and a FIFO cannot be made. So the copy gives no difference there. On a Linux host that grades, two forms give V no build where L builds: N43. The builder's named lean, code that reads a file through a link, is one case of it. A source file that is a link is another. I found no form that builds in V and not in L.

### 4. N35 to N38

| Finding | Result |
|---|---|
| N35 | Stated, in the judge's paragraph, and judgment call 36 |
| N36 | Stated, in the paragraph "The tool's version" |
| N37 | The row is changed. Its new reason, "because the walk finds its `go.mod`", does not hold: `filepath.WalkDir` does not follow a junction, and `s8` passes in both arms. That is harmless, since no `replace` can reach it, but the text is wrong. Low |
| N38 | Stated, as the rule is, and judgment call 35 |

### 5. Regressions

**Regression.** 694 cases compared, in the fifth rerun's scripts and also `a4` and `c3`. One changed: `a3 t12`, "a module outside the project, with a replace", was exit 0 in both arms and is now K0002 and VL0002. That is amendment 12, the same in both arms. No other verdict or code changed.

**Tests.** In the repository at 4752184, with `-timeout 30m`: `go test ./...` at the root, which holds `conformance`, `internal/rules`, and `poc/harness`, and `go test ./...` in `poc/golint`. Both pass.

**Counts.** `internal/rules/defaults`, `experiments/poc/r3`, and `poc/app` have no change since 6038e4a. The base applications have no `replace`. So the counts of the fifth rerun stand, and so does the hash of `defaults v1`, `sha256:14c3ca9a9ac1603d5227ff4275375f2fd38bded91be26887b921929972a6f427`. `TestDefaultsOnBase` passes.

### New findings

#### N39. High. In L alone, assembly in `main` reaches the build, and no rule reads it

- **Where:** `poc/golint/imports/imports.go:29` and `:33`. `foreignCode` and `Bodyless` skip a part of kind `Main`. So does the `//go:linkname` check, at `:65` to `:70`.
- **Case `m_asm`:** in `cmd/shop/main.go`, `func evil() int`, with its body in `cmd/shop/evil_amd64.s`, and `main` ends the process when `evil()` returns 42. With `ERR-08` and `ARCH-10`: **L exit 0, `go build` passes, and the binary exits 7.** With no rules file, also exit 0. The same in a leaf gives VL0126 in L and K0126 in V (`m_leaf`). A bodyless function with `//go:linkname` in `main` also gives exit 0 (`m_link`).
- **What `main` may not do in Go:** L reads `main`'s Go code. `net.Dial` and `exec.Command` there give VL0205 (`a27`). Assembly is not read, so it can do what the Go code may not.
- **V:** V generates `main` from `system.vulcan`, and a person writes no Go code there. So V has no such place.
- **Why High:** it is the escape of H3, N1, N10, N16, N23, and N31: code inside the project that leaves the model and each rule, with exit 0, in one arm. Unlike N31, it needs nothing outside the project.
- **Favors:** L passes more.
- **Smallest fix:** apply H6 and the `//go:linkname` check to `main` too, in L.

#### N40. High. In L alone, a leaf's directory that is a junction leaves the check, and the build holds it

- **Where:** L runs its check on `./...`. The go command does not follow a junction or a symlink to a directory when it expands `./...`, so the leaf is not a package that the check loads. `go build ./cmd/shop` imports it through the junction. The walk for a `go.mod` does not follow it either.
- **Cases, on Windows:** `api/orders` is a junction to `_keep/orders`, `testdata/orders`, or `.keep/orders` inside the project (`s7*`), or to a directory outside it (`s6`). The leaf holds `os.Exit(2)`, which `ERR-08` forbids. **L exit 0, and `go build` passes.** `vulcan-lint ./api/orders` on the same leaf gives VL0802, so the code is there and the pattern is what leaves it out. V gives K0003, because it does not find the leaf's directory.
- **Linux:** not run. The go command's documentation says that `./...` does not follow a symlink to a directory either, so I expect the same result there, but I did not show it. A link that WSL makes on the Windows drive cannot be opened on Windows, so on the harness's grading path L's build fails too.
- **Why High:** the same wording as N39. The bytes are inside the project, in `_keep`, `testdata`, or `.keep`.
- **Against, as context:** it needs a junction, which an agent in the sandbox cannot make on the Windows drive. A run on the host can make one.
- **Favors:** L passes more.
- **Smallest fix:** load each part of the manifest by its import path, not by `./...` alone, or reject a part's directory that is a link, in both arms.

#### N41. Medium. In both arms, the `ignore` directive of `go.mod` takes a part out of each rule and the model

- **Where:** `go.mod` can say `ignore ./api/orders` (Go 1.25). The go command then leaves that directory out of `./...` and still builds it when it is imported. Both checks load packages through the go command, and `ForeignRoots` reads the `replace` lines of `go.mod`, not `ignore`.
- **Cases (`a29`):** a leaf with `func init() { os.Exit(2) }` gives K0117 and K0802 in V, and VL0117 and VL0802 in L, with no `ignore` (`i0`). With `ignore ./api/orders`, `ignore api/orders`, or `ignore ./lib` on a library: **exit 0 in both, each builds, and the binary exits 2 or 5**, also with no rules file (`i1`, `i3`, `i6`, `i7`). A leaf that imports a leaf passes in both (`i4`). An ignored plain directory that a leaf imports still fails in both (`i5`). With `ignore ./api`, V's binary exits 0 at once, and L's exits 2 (`i2`). I did not look further into that.
- **Why Medium:** a hole in both arms, as N18 and N32 were. Its reach is wider than theirs: one line in a file that the agent can write, in each arm, with no link and nothing outside the project.
- **Smallest fix:** reject an `ignore` directive in the root `go.mod`, in the shared function of amendment 12, as is done for a `replace` by a local path.

#### N42. Low. `CountModel.Decided` is 0 in V by its definition

`Decided` counts a run with a shared model code that builds and passes its acceptance tests. In V such a code stops the build, so V's count is always 0, and only L's count has meaning (`TestModelError` checks `Decided[V] == 0`). M3 is not affected. A report of step 8 that sets the two counts side by side would mislead. **Fix:** count, in each arm, the runs with a shared model code, or state that the count is L's alone.

#### N43. Low. On a Linux host, the copy for V's build still loses some workspaces that L builds

A source file that is a symlink, and a file with mode 000 anywhere, give V no build (`undefined: Z`, or a copy error) and L a build (`lx6.sh`). The harness grades on Windows, from a workspace on the Windows drive, and there neither form gives a difference. So this reaches only a grader on Linux. **Fix:** state it, or for a grader on Linux, copy the target of a link to a file.

### Decisions for the owner

- **N39 and N40 decide the gate.** Each is in L alone, and each has a small fix in L or in the shared loading. If the owner instead lists N40 as a limit of the host's links, the High count is 1, and N39 stays.
- **N41.** Whether to reject `ignore` with the same rule as a local `replace`. It is a one-line form in both arms.
- **N43.** Whether a grader on Linux is in scope. If not, it can be listed.

### What I could not check

- The sealed seeded cases, as before.
- A fresh judge's decisions, an agent's real run, and round-2 run data, as before.
- Windows symlinks, which my account could not create. Junctions stand in for them.
- On Linux: N39, N40, and the backslash fallback, which I did not run myself.
- The cause of `i2`.
