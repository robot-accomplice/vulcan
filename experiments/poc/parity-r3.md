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
