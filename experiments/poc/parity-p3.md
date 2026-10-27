# Parity review of Stockroom-lite (P3)

Reviewer: an independent agent session. It did not write either version.
Reviewed: `poc/app/vulcan` and `poc/app/go` at commit 174f2ac (branch main), with
`docs/poc/plan.md`, `docs/poc/stockroom.md`, `docs/poc/guide.md`, `poc/acceptance`, and `poc/fakepay`.

## Verdict

The two versions are close to a line-for-line match. The leaf bodies are the same Go code. The
error codes, the messages, the validation, the locking, the SQL, and the flag handling match.
Both binaries pass the shared acceptance suite. A hand probe of 25 requests (bad bodies, bad
states, refunds, unknown ids, and flag errors) gave the same status and body in both, with one
exception (item 7).

The differences that remain are not in the application logic. They are in the check commands,
in one wiring trap of arm V that the primer does not explain, and in a few small extras. Fix the
two High items and the Medium items before the freeze. None of them needs a change to the
application's behavior.

Counts: 2 High, 4 Medium, 5 Low.

## What I verified

- Built the compiler (`go build ./cmd/vulcan`), then `vulcan build` in `poc/app/vulcan`, and
  `go build ./cmd/stockroom` in `poc/app/go`. Binaries in a temporary directory outside the
  repository. Both built.
- `vulcan check` and `vulcan test` pass in arm V. `go vet ./...` and `go test ./...` pass in
  arm G. Neither version has a test file.
- `STOCKROOM_BIN=<v.exe> go test ./poc/acceptance -count=1`: ok. The same with `g.exe`: ok.
- `diff` of every body file. Store and the errors are identical. Payments differs only in the
  `vulcan:"secret"` tag. The other files differ only in the port declarations, the port calls
  (`s.store.X` against `Store.X`), the constructors, and the contracts.
- Ran both binaries with no flags, with `--name value` flags, with an unknown flag, with `-h`, and
  with a bad payments URL. The messages and exit codes match, except the prefix (`stockroom:`
  against `Stockroom/Api:`) and the order of the lines.
- Ran both binaries against `fakepay` with the same 25 requests (item 7 shows the one difference).
- Tried two changes in a scratch copy of arm V (items 2 and 9), and one in a scratch copy of arm G
  (item 1).

Note: `vulcan build` and `vulcan check` write their generated Go code into
`poc/app/vulcan/.vulcan/build` inside the repository. That directory is in `.gitignore`, so
`git status` stays clean. I did not create it on purpose and did not delete it.

## Differences that change a task's difficulty

### 1. High. The check commands differ in strictness: arm G runs full `go vet`, arm V does not

- Plan section 2: arm G checks with `go vet ./... && go test ./...`; arm V with
  `vulcan check && vulcan test`.
- `cmd/vulcan/main.go` and `internal/driver` never run `go vet`. `vulcan test` runs `go test`,
  which runs only vet's small default subset (atomic, bool, buildtags, errorsas, ifaceassert,
  nilfunc, printf, stringintconv, tests). It does not run copylocks, lostcancel, unusedresult,
  shadowing checks, and the rest.
- Verified: I changed `func (s *Service) Get` to a value receiver in both versions
  (`orders.vulcan` line 64, `orders.go` line 91). `orders.Service` holds a `sync.Mutex`. Arm G:
  `go vet` fails with "Get passes lock by value". Arm V: `vulcan check` and `vulcan test` both
  pass.
- Favors: arm V on task success and on the check loop (M5); arm G on correctness. Either way it
  is noise, not a language effect.
- Example task: "add a context with a timeout to the payment call". In arm G, a forgotten
  `cancel()` fails the check (lostcancel), and the agent fixes it. In arm V the same defect
  passes the check, and the run may end earlier with a latent bug.
- Fix: give both arms the same Go analysis. Either make arm V's check
  `vulcan check && vulcan test && go vet ./...` run inside `.vulcan/build` (the harness can do it,
  with positions mapped back by the `//line` directives), or make `vulcan check` run `go vet` on
  the generated code. Do not drop vet from arm G instead: it is what a Go team runs.

### 2. High. In arm V, the one shared database depends on three identical wire lines, and the primer does not say so

- Arm V, `system.vulcan` lines 15 to 17: `Orders.Store`, `Inventory.Store`, and `Billing.Store`
  each say `Store.Open{ Path: flag("db", default: "stockroom.db") }`. The compiler builds one
  `*store.DB` only because the three lines are identical (generated `vulcan_main/main.go`).
- Verified: I changed only `Billing.Store` to `flag("db2", default: "x.db")`. `vulcan check`
  passes, and the generated `main` now opens two databases, one for Billing alone. No diagnostic.
  Two `sql.DB` handles on one file also break the comment in `store.go` line 68 ("One
  connection: SQLite allows one writer, and the transactions below rely on it").
- Arm G, `cmd/stockroom/main.go` lines 64 and 74 to 76: one `store.Open` call, and `db` is passed
  to three constructors. There is nothing to keep in sync.
- `docs/poc/guide.md` section 2 never says that wires with the same constructor and the same
  settings share one instance, or that different settings make a second instance.
- Favors: arm G. It adds a cost and a silent failure to arm V that is not one of the measured
  features (ports, gateways, contracts, invariants).
- Example tasks: "add a busy timeout setting to the database" (a new field in
  `store.Settings` must get the same value on three lines in V, one line in G); "show the stock
  on the order page" done by giving Web a Store port (the fourth copy of the line must match
  exactly); "rename the `--db` flag".
- Fix: in `docs/poc/guide.md` section 2, add one sentence: "Wires that name the same
  constructor with the same settings get one shared instance; any difference in the settings
  makes a second instance." Add a one-line comment above the three lines in `system.vulcan`
  that says they must stay identical so that the store is shared. Better, if the compiler can do
  it before the freeze: a diagnostic (or a warning) when one constructor of a gateway is wired
  with two different settings.

### 3. Medium. Arm V's check runs the race detector on a host that has cgo; arm G's does not

- `cmd/vulcan/main.go` lines 173 and 208 to 214: `vulcan test` adds `-race` when
  `driver.RaceSupported()` is true (cgo enabled, and gcc on Windows). Plan section 2 gives arm G
  plain `go test ./...`.
- On this host (Windows, no gcc) the detector is off, and every `vulcan test` prints a warning
  instead: `warn K0705: vulcan test ran without the race detector...`. On a Linux run host with
  gcc, arm V's tests run with `-race` and are slower, and they can fail on races that arm G's
  check never sees.
- Favors: arm G on cost and task success where the host has cgo; arm V on correctness. On a host
  without cgo, the warning is extra text that only arm V sees, and an agent may spend turns on
  it.
- Example task: "cache the catalog in memory". An agent adds a map without a lock and a
  concurrent test. Arm V fails with `-race`, arm G passes.
- Fix: make the two arms equal on the run host. Either run arm G with `go test -race ./...`
  when the host supports it, or run arm V with `vulcan test --race=false`. Pick the run host
  first, and write the choice into the harness and the task briefs. If the host has no cgo,
  run arm V with `--race=false` so that the K0705 warning does not appear.

### 4. Medium. Arm G's hand-written flag parser needs three edits for a new flag, one of them easy to miss

- Arm G, `cmd/stockroom/main.go` line 47: `known := map[string]bool{"addr": true, ...}`. A new
  flag must be added to this map, read with `s.get`, and passed to a settings struct. If the map
  is missed, the binary rejects the flag at start ("unknown or malformed flag"), and every
  acceptance test fails at start.
- Arm V: one `flag("x")` in `system.vulcan`.
- Generated wiring is an intended difference. But the `known` map is a second list of the same
  names that a small Go program would not keep, so arm G pays more than the language requires.
- Favors: arm V.
- Example task: "add a `--max-order-cents` setting".
- Fix: in arm G, keep one table of flag specs (name, default, required) and build both the
  `known` set and the reads from it, so that a new flag is one line in the table plus its use.
  Also fix the comment on line 37: it says "setting reads", the type is `settings`, and the
  method is `get`.

### 5. Medium. Neither version has a README, and the way to build, run, and check differs in discoverability

- Neither `poc/app/vulcan` nor `poc/app/go` has a README. Arm V has `vulcan guide` (the primer,
  with the commands and the layout). Arm G has nothing beyond `go.mod` and the package comments.
- Neither workspace says how to run the service against a card processor. `poc/fakepay` and
  `poc/acceptance` are outside both workspaces, and `--payments-url` is required, so neither arm
  can run the binary end to end without writing its own fake.
- In arm V, `go build` or `go vet` in the workspace fails (the `.vulcan` files are not Go, and
  `go.mod` requires `github.com/robot-accomplice/vulcan/vulcan v0.0.0`, which only the generated
  build directory replaces). The primer says to use `vulcan mod tidy`, but not that the `go`
  commands do not work in the workspace.
- Favors: it depends on the task. Arm V knows its commands from the primer. Neither arm can run
  the binary end to end.
- Example task: a failure-handling task ("when the card processor times out, leave the order
  pending and return payment_unavailable") is much easier to check by running the binary.
- Fix: add one README to each workspace, word for word the same except the language name and the
  commands: the layout (one line for each leaf or package), the check command of the arm, how to
  build and run, and how to start the fake processor (ship `fakepay` as a built binary, or as a
  command on the PATH, in both workspaces). In arm V's README, say that `go build` does not work
  in the workspace and that `vulcan build` does.

### 6. Medium. Arm V has a contract that encodes current behavior, not a rule of the business

- Arm V, `api/orders/orders.vulcan` line 22: `ensures err == nil => ... && len(result.Lines) ==
  len(wants)`. No rule in `docs/poc/stockroom.md` says that each wanted line becomes exactly one
  order line. It is an implementation fact. Arm G has no counterpart, as expected.
- The other contracts each encode a rule of `stockroom.md` (rules 2, 4, 5, and the stock limits)
  and are the treatment under test. This one adds cost to arm V on an ordinary task, and does not
  guard a rule.
- Favors: arm G on ordinary tasks (it is measured as Vulcan's cost, but it is the builder's
  choice, not the language's).
- Example task: "accept the same SKU on two lines and merge them into one line". Arm V panics with
  a contract failure on the first test, and the agent must find and change the `ensures`.
- Fix: in arm V, drop `&& len(result.Lines) == len(wants)` from the `ensures` of `Create`, and keep
  `result.Status == shop.Pending && result.Charged == 0`. Check the remaining contracts against the
  same test: each one should map to a numbered rule or a data limit of `stockroom.md`.

### 7. Low. A price overflow gives a different response in the two versions

- Both versions accept `price_cents` up to the int64 limit, and `shop.Total` overflows. This is a
  shared defect: an order with `price_cents` 9223372036854775807 and `qty` 2 is created with
  `total_cents` -2 in both.
- Paying it: arm G answers 502 `payment_unavailable` ("the card processor answered 400"). Arm V
  panics on the port contract `Orders.Billing.Charge requires o.Total > 0`, and the client gets a
  closed connection with no response.
- This is a contract doing its job, so it is mostly intended. It matters only if a holdout test or
  a task touches large amounts.
- Favors: neither, unless a task touches it.
- Example task: "add a per-order limit of 1,000,000 cents". Both arms are fine. A holdout test
  that sends huge prices would see different results.
- Fix: none needed for parity. Tell the grader and the task writer about the shared overflow so
  that no holdout test depends on it, or add the same upper bound on `price_cents` and `qty` to
  both versions (in `inventory` and `orders`) before the freeze.

### 8. Low. Arm V's generated code sits inside the workspace, with no "do not edit" mark on the leaf files

- `vulcan check` writes `.vulcan/build/` in the workspace, with a Go copy of every leaf file and
  the generated `main`. The generated `main.go` and `vulcan_ports.go` say "DO NOT EDIT". The
  leaf copies (for example `.vulcan/build/api/billing/billing.go`) start with a `//line`
  directive and no such mark.
- A `grep -r` in Bash finds every match twice in arm V. An agent may edit a generated copy, which
  the next check overwrites. Ripgrep (the Grep tool) skips hidden directories by default, so the
  effect is smaller there.
- Favors: arm G, a little.
- Fix: in the harness, start each arm V run with no `.vulcan/` directory (as committed). If the
  compiler can do it before the freeze, put the build directory outside the workspace (for
  example in the user cache) or add the "Code generated ... DO NOT EDIT." line to each leaf copy.

### 9. Low. Arm V's Web wraps three port calls in closures that it does not need

- Arm V, `api/web/web.vulcan` lines 54 to 56:
  `orderAction(func(id shop.OrderID) (shop.Order, error) { return Orders.Pay(id) })`.
  Arm G, `api/web/web.go` lines 84 to 86: `orderAction(a.orders.Pay)`.
- Verified: `orderAction(Orders.Pay)` passes `vulcan check` in a scratch copy, so this is not a
  compiler limit.
- Favors: arm G, a little (longer lines to read and copy for a new action route).
- Fix: in arm V, use `orderAction(Orders.Pay)`, `orderAction(Orders.Ship)`, and
  `orderAction(Orders.Cancel)`.

### 10. Low. Small layout and doc differences

- Arm G, `api/store/store.go` and `api/payments/payments.go` have no package comment. Arm V has
  the same text in `api/store/component.vulcan` ("Store keeps the catalog, the stock, the orders,
  and the ledger in SQLite.") and `api/payments/component.vulcan`. Fix: add
  `// Package store keeps the catalog, the stock, the orders, and the ledger in SQLite.` and
  `// Package payments is the client of the card processor.` to arm G.
- Arm V, `system.vulcan` lines 3 to 5, describes `Customer`, `Database`, and `PaymentProvider`
  in words. Arm G's `main.go` has no such overview. This is part of the language, so it is fine,
  but one comment block above `run()` in arm G with the same three lines and the wiring (which
  package uses which) would make the overview equal.
- `billing.accountOf` is a method of `*Ledger` in arm G (`billing.go` line 48) and a plain
  function in arm V (`billing.vulcan` line 39). This follows from ports being package names. No
  fix needed.
- Favors: arm V, a little.

### 11. Low. Startup error text differs

- Arm G prefixes errors with `stockroom:`; arm V with `Stockroom/Api:` and a stage line
  ("startup failed:" and "constructor Payments.NewClient: ..."). The required-flag lines are
  the same text. No acceptance test reads these messages.
- Fix: none needed. If a task asks for a startup check, write its test to look at the exit code
  and a substring that both versions print (for example the flag name).

## Intended differences that are bigger than they need to be

- Item 6 (the `len(result.Lines)` ensures) is the one contract that goes beyond the rules.
- Item 2 is a gap in the primer more than in the language: the sharing rule exists in the
  compiler but not in `guide.md`.
- Item 9 is extra code in arm V that the language does not require.

## Places where one version could not express what the other does

- I found none in the application code. Arm V needed no workaround for a compiler limit: the
  closures of item 9 are optional, and everything else in the bodies is the same Go.
- Arm V cannot name one instance and wire it to several ports. It can only repeat the same
  constructor and settings (item 2). Arm G passes one value. This is a limit of the wiring syntax
  in this release; the fix above is a sentence in the primer, or a diagnostic.

## Intended differences that are fine as they are

- Ports in `component.vulcan` against consumer-owned interfaces at the top of each Go package. The
  method sets and their doc comments are the same.
- Port calls by name (`Store.Order(id)`) against struct fields (`s.store.Order(id)`), and empty
  constructors (`NewLedger()`) against constructors with interface parameters.
- `system.vulcan` wiring and flags against `cmd/stockroom/main.go`. The flag names, the variables,
  the defaults, the required flags, and the messages match.
- `Web.startup` against `(*API).Serve`. The bodies are the same, including the 10 second
  shutdown.
- The contracts and invariants of arm V that encode rules of `stockroom.md`: the `account`
  invariant and the `charge` and `refund` contracts in Billing (rule 5 and 6), the port contracts
  on `Billing.Charge` and `Billing.Refund` (rules 2 and 5), the `ensures` of `Pay`, `Ship`,
  `Cancel`, and `Refund` (rules 2 to 5), the `shop.Stock` invariant (the stock limits in the Data
  section), and `requires qty > 0` and `requires len(lines) > 0`. Every rule of `stockroom.md` that
  the API exposes is also enforced by plain code in both versions (the status checks in `orders`,
  the refund limit in `billing`, the availability check in `store`, the `qty` checks in
  `inventory`). So no business rule is enforced by a contract alone in arm V and by code in arm G.
  The contracts only add a second line of defence, and they document the rules in the source,
  which is part of the treatment by plan section 4. One caveat for the analysis: the stock limits
  (`reserved <= on_hand`, each 0 or more) are written in arm V's source and nowhere in arm G's. A
  task that tempts a change of stock (a write-off, a backorder) gets both a check and a hint in
  arm V. That is the intended effect, but the report should say that the hint is part of it.
- `Key string \`vulcan:"secret"\`` in arm V's `payments.Settings`. It makes a default key a
  compile error (K0605). Arm G has no counterpart. It is a language feature and fits a class B or
  C rule.
- The `lib/shop` library rules of arm V (no package variable written after init, no file or
  network target). Arm G's `lib/shop` is the same code without the rules.
- Test fakes: `vulcan.Bind(t, &Port, fake)` against passing fakes to constructors. Both are short.
  Arm V's tests cannot use `t.Parallel` with `Bind`; that is inherent.
- Build time: `vulcan build` took about 6 s and `vulcan test` about 5 s on this host, against about
  2 s for `go build`. It is inherent to the toolchain and is part of the measured cost.

## Shared notes for the later phases

- Both `go.mod` files say `go 1.26.0`, while the host runs Go 1.27.3. They are equal, so this is
  not a parity issue. If the toolchain is upgraded, upgrade both in one change before the freeze.
- Neither version has a test. If a task asks the agent to add tests, the two arms start from the
  same place.
- The shared overflow of item 7 should be known to the grader before it writes holdout tests.

## Resolution (2026-10-27)

Both versions pass the acceptance suite (10 tests, now with the data limits), and the compiler's
131 conformance cases pass.

| Item | Status | Change |
|---|---|---|
| 1 | Fixed | `vulcan check` runs `go vet` on the generated code and reports each finding as an error at its `.vulcan` position (K0706, conformance V01). A finding on the line of a contract clause is in generated check code and is dropped; the generated functions of an invariant now sit on that line. Arm G keeps `go vet ./...` |
| 2 | Fixed | The primer says that wires with the same constructor and settings share one instance; `system.vulcan` says so above the three Store wires; K0607 warns when one constructor is wired with two settings (conformance V02) |
| 3 | Fixed | The run host has no cgo, so arm V's check is `vulcan check && vulcan test --race=false` (no K0705 warning), and arm G's runs no race detector either; plan section 2 says so |
| 4 | Fixed | Arm G reads its flags from one table |
| 5 | Fixed | Each workspace has a README, the same text except the language and the commands, with the check command and how to run against `fakepay`; arm V's says that the `go` commands do not work there. Plan section 2: the harness puts `fakepay` on the PATH in both arms |
| 6 | Fixed | `Create`'s `ensures` keeps only the status and the zero charge |
| 7 | Fixed | Both versions limit `price_cents` to 100,000,000 and `qty` to 10,000 (`shop.MaxPrice`, `shop.MaxQty`); `stockroom.md` states the limits; acceptance tests cover them |
| 8 | Fixed | Every generated copy of a code file starts with a "Code generated ... DO NOT EDIT" line; the primer says not to edit `.vulcan/build`; plan section 2: an arm V run starts with no `.vulcan/` |
| 9 | Fixed | Arm V's Web passes `Orders.Pay`, `Orders.Ship`, and `Orders.Cancel` directly |
| 10 | Fixed | Package comments on arm G's store and payments, and the overview above `run()` |
| 11 | Not changed | As the review says, no test reads the startup prefix |
