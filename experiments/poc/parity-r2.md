# Parity review of Stockroom-lite, round 2 (three arms)

Reviewer: an independent agent session. It wrote none of the three versions.
Reviewed: `poc/app/go` (arm P), `poc/app/golint` (arm L), and `poc/app/vulcan` (arm V) at commit
95958b3 (branch main), with `docs/poc/stockroom.md`, `docs/poc/plan-r2.md`, `poc/r2-app-growth.md`,
the three READMEs, `poc/app/golint/PRIMER.md`, `vulcan guide`, `poc/acceptance`, and the parts of
`poc/harness` that build the workspace, the prompt, and the configuration check
(`runner/config.go`, `runner/runner.go`, `grade/grade.go`, `record/record.go`).

Job (decision 6, section 2): find the differences that would change a task's difficulty. Method and
format follow `experiments/poc/parity-p3.md`.

Stance: no outcome is wanted. I looked for differences that favor P as hard as for those that favor
L or V. Each finding says which arm it favors. Of items 1 to 8, three favor P and L over V (items
2, 6, 7), three favor V over L (items 1, 5, 8; item 1 also favors L on one kind of tempting task),
one favors L over V (item 4), and one is mixed (item 3).

## Verdict

Behavior is the same in the three arms. The bodies are the same Go code; the only differences are
the ports, the port calls, the constructors, and the contracts, as `r2-app-growth.md` says. Each of
the three binaries passes the acceptance suite (17 tests). A probe of 104 requests, with webhook
deliveries to a test receiver, gives byte-for-byte the same responses, deliveries, and log lines
in all three, once the random ports and secrets are masked. L and V have the same 29 contract
clauses.

The differences that remain are in the harness and the documents around the code, and in one limit
of V's wiring:

- **High:** adding a package is a configuration change that fails the run in arm L only. V can add a
  leaf, an external, or change a leaf's kind without failing the run (item 1).
- **Medium:** V's shared instances depend on identical wire lines (five Store, three Hub, three
  Catalog). If one line differs, the check still passes, with a warning, and the process silently
  gets a second database or a second hub (item 2). L's primer is 39% longer in words than V's,
  although the plan asks for the same token budget (item 3). The arms are not told the same things
  about which files and comments are off limits (item 4).
- Four Low items.

Counts: 1 High, 3 Medium, 4 Low (items 5 to 8), and one item kept from P3 (item 9). Fix item 1
before any task that adds a package, an external, or a person is written. Fix items 2 to 4 before
the freeze. None of the fixes changes the application's behavior.

## What I verified

- Built `vulcan`, `fakepay`, and `vulcan-lint` with `go run ./poc/harness/cmd/harness tools`.
- Ran each arm's check command (from its README) on the committed workspace: all pass. P: `go vet
  ./... && go test ./...`. L: `go vet ./... && vulcan-lint ./... && go test ./...`. V: `vulcan
  check && vulcan test --race=false`. No arm has a test file.
- Built the three binaries outside the repository: `go build ./cmd/stockroom` in P and L, and
  `vulcan build -o` in V.
- `STOCKROOM_BIN=<binary> go test -count=1 ./poc/acceptance/...` gives ok for each of the three.
- A probe program, written for this review (in the session's scratch directory, not committed),
  started each binary with `fakepay` and a webhook receiver, and sent the same 104 requests. They
  covered every endpoint of `stockroom.md`, every error code, the data limits, bad JSON and unknown
  fields, the state machines of orders and purchase orders, partial and over-receiving, an
  all-or-none delivery, a cancel after a partial receipt, a declined charge, subscriptions with bad
  URLs and bad event lists, a redirecting subscriber, a subscriber that fails three times then
  succeeds, retries, a 6 second subscriber (the 5 second timeout), a delete and the 404s after it,
  and the largest order the limits allow. After masking the ports and the secrets, the three
  outputs (responses, deliveries received with their headers and bodies, and stderr) are identical.
- Ran each binary with no flags and with an unknown flag (item 9).
- `diff` of every code file: P against L, P against V, and L against V with the port calls
  normalized (`s.store.X` to `Store.X`). The store files `purchases` and `events` are identical in P
  and V. Every other difference is a port, a port call, a constructor, a package or component
  comment that moved, or a contract.
- Counted and compared the contract clauses of L and V one by one (see "Checked and equal").
- In scratch copies of L and V: added a new leaf `api/reports` (item 1); changed one Store wire of V
  (item 2).
- Timed each check command, warm cache, after an edit to one body file (item 7).
- Read the harness's configuration check (`runner/config.go`), the prompt (`runner.Preamble`), and
  how the grader uses `ConfigChanges` (`record.go` line 160).

The review changed nothing in the repository. `vulcan check` in `poc/app/vulcan` refreshes the
ignored `.vulcan/build` directory there.

## Differences that change a task's difficulty

### 1. High. Adding a package fails the run in arm L only. Arm V can add a leaf, or change a leaf's kind, and still pass

- Arm L: `poc/app/golint/PROTECTED` lists `arch/*.go`. `arch/arch.go` holds both the check
  configuration (`Strict: true`) and the list of parts (every package with its kind, and the persons
  and externals). The harness (`poc/harness/runner/config.go`, `configChanges`) compares every
  protected file byte for byte. A changed `arch/arch.go` is a configuration change, and a run with a
  configuration change is not a clean success unless the task sets `allow_config`
  (`record/record.go` line 160).
- Every new package in L must be added to `arch/arch.go`: without it, `vulcan-lint` fails with
  VL0002. Verified in a scratch copy: a new `api/reports` gives
  `VL0002 package api/reports is not in the manifest; every package of the module is a part of the
  manifest (arch), and the manifest changes only when the request asks for a change to the
  architecture`. `PRIMER.md` section 1 says the same: "change it only when the request asks for a
  change to the architecture."
- Arm V: the same new leaf gives `error K0003: leaf directory api/reports is not listed in any
  container of system.vulcan`, with no advice against the fix. The fix is a name in the container
  list and the wires of `system.vulcan`. The harness checks only the `config { ... }` block of
  `system.vulcan`. So in V, adding a leaf, adding an `external` or a `person`, and changing an
  existing leaf's kind (`gateway for`, `entry for` in its `component.vulcan`) never fail the run.
  In L, each of these is an edit of `arch/arch.go` and fails the run.
- Arm P: a new package is a directory and a line of `main.go`. No check.
- Plan-r2 section 2 asks for less: "a change to the manifest's kinds, the linter, `strict_exempt`,
  `strict_interfaces`, `strict_shared`, or an in-source suppression, fails the run". Adding a part
  is not a change to the kinds of the existing ones.
- Favors: V (and P) over L, strongly. In L, a task that needs a new part is either a failed run (the
  agent edits the manifest) or an architecture break (the agent obeys the primer and puts the code
  into an existing package). Both count against L on clean success (QA, QV), and the second also on
  the rule measures. In the other direction, V can make a tempting change by redeclaring a kind
  (for example, make `Orders` a `gateway for` a new external, so that it may call HTTP), and its run
  is not flagged; L's same change is flagged. That favors L on tempting tasks if a rule depends on
  kinds.
- Example tasks: "send a daily stock report to the warehouse system" (a new gateway and an
  external); "add a returns service" (a new leaf); "move the refund logic into its own part"; "let
  orders call the fraud-check API" (tempting).
- Fix: protect the same things in L and V, by meaning, not by file.
  1. In L, split the manifest: `arch/config.go` holds `Strict`, `StrictExempt`, `StrictInterfaces`,
     `StrictShared`, and `Mutators`, and is protected; `arch/arch.go` holds the parts, the persons,
     and the externals, and is not. (Or keep one file and make the runner compare those fields
     only.)
  2. In both arms, the runner also compares the kind of every part that exists in the base: in L,
     the constructor (`Leaf`, `Gateway`, `Entry`, ...) and its party for each directory of the base
     manifest; in V, the `gateway for` and `entry for` lines of each base `component.vulcan`. A
     changed kind fails the run in both; an added part does not.
  3. Give the two diagnostics and the two primers the same words. Drop "the manifest changes only
     when the request asks for a change to the architecture" from VL0002 and from `PRIMER.md`
     section 1, or add the same sentence to K0003 and to `vulcan guide` section 2. The first is
     simpler and matches the plan.
  4. Until this is fixed, the task writer must not write a task that adds a part, or must set
     `allow_config` for it in all arms.

### 2. Medium. Arm V's shared instances depend on identical wire lines, and a difference only warns

- Arm V, `system.vulcan`: five identical `Store.Open{ Path: flag("db", default: "stockroom.db") }`
  wires (Orders, Inventory, Billing, Purchasing, Events), three `Events.NewHub` wires (Web, Orders,
  Purchasing), and three `Inventory.NewCatalog` wires (Web, Orders, Purchasing). The compiler
  shares one instance only because the lines are the same. P and L build `db`, `catalog`, and
  `hub` once in `main.go` and pass them.
- Verified in a scratch copy: changing only the `Events.Store` wire to `flag("events-db", default:
  "events.db")` gives `warn K0607: Store.Open is wired with other settings than at line 24, so main
  makes a second instance`, and `vulcan check` exits 0. The process then has two databases.
- Since P3 (item 2, fixed by a primer sentence, a comment, and K0607), the count has grown from
  three lines to five, and a second shared constructor has appeared where sharing is a correctness
  property. `events.Hub` holds the lock that "serializes the deliveries, so that one delivery is
  never sent twice at once" (`events.go`). Today `NewHub` takes no settings, so the three wires
  cannot differ. But once a task gives it a setting (a timeout, a retry limit) on one wire only, V
  gets two hubs with two locks, and its check passes.
- Favors: P and L. The cost grows with every part that needs the store or the events, and it is a
  limit of V's wiring syntax, not one of the measured features.
- Example tasks: "add a busy timeout setting to the database" (five equal edits in V, one in P and
  L); "make the webhook timeout configurable" (if done on the hub, three equal edits in V); "show
  the purchase orders on the product page" done with a new Store port on Web (a sixth copy of the
  line); any new leaf that raises events (a fourth `Events.NewHub`).
- Fix: in the compiler, before the freeze, make K0607 an error in strict mode. A run then cannot
  end with two instances by accident, and the cost stays visible: the edits, not a silent fault.
  If that is not possible, log the difference for the analysis and have the task writer avoid tasks
  that add settings to `Store.Open`, `Events.NewHub`, or `Inventory.NewCatalog`, or count those
  tasks separately.

### 3. Medium. Arm L's primer is 39% longer in words than arm V's, though the plan asks for the same token budget

- `poc/app/golint/PRIMER.md`: 11,597 bytes, 2,021 words, 192 lines. `vulcan guide`: 9,434 bytes,
  1,455 words, 181 lines. L's is 23% longer in bytes and 39% longer in words. (The arm-L notes
  estimated about 2,900 against 2,360 tokens, at bytes / 4, with no tokenizer.) Plan-r2 section 2:
  "a primer of the model in Go terms with the same token budget as Vulcan's".
- Most of the excess is section 5 of the primer, a table of 48 codes with what is wrong and how to
  fix each. `vulcan guide` has no such table; V's diagnostics carry their own text, as L's do.
- L's workspace also holds `lib/contract/contract.go` (552 words) and `arch/schema.go` (322
  words), which an agent may read to learn the model. V's runtime and schema are not in the
  workspace.
- Favors: mixed. The longer text costs L input tokens on every run (QC, the N measures). The codes
  table may help L recover from a diagnostic (O2).
- Fix: count both primers with the same tokenizer (for example the token-counting endpoint of the
  run model) and bring L's to within 5% of V's. The simplest cut is the codes table: V has none,
  and every VL code already says how to fix it. If the table stays, add the same kind of table to
  `vulcan guide`. Report both counts, with the method, in the round's records.

### 4. Medium. The arms are not told the same things about which files and comments are off limits

- Arm L has `PROTECTED` at the top of the workspace, whose first line explains it: "The check
  configuration ... A change does not touch these files unless the request asks for it." The
  primer says the linter "honors no `//nolint` or other suppression comment", and section 1 says
  when to change the manifest (item 1).
- Arm V: neither the README nor `vulcan guide` says that changing the `config { ... }` block fails
  the run, or that a suppression comment is counted. The guide even shows how to exempt a leaf
  (`strict_exempt: ["Leaf"]`), with no warning.
- Arm P: nothing. The harness counts `//nolint` and `//lint:ignore` in P too
  (`runner/config.go`, `suppressions`), and P runs no tool that reads them, but the P agent is not
  told.
- Favors: L on configuration failures. An L agent is warned off a change that fails the run; a V
  agent learns how to make it. The same text also discourages L from legitimate architecture
  changes (item 1).
- Example task: a tempting task whose easy path is to exempt a leaf from strict mode ("the
  webhooks need to call a callback the subscriber gives").
- Fix: put one sentence, the same words in the three READMEs, under "Check, build, and run": "Do
  not change the check configuration (<per arm: nothing / `arch/` and `lib/contract` /
  the `config` block of `system.vulcan`>) or add suppression comments unless the request asks for
  it." Keep `PROTECTED` as the harness's list, and shorten its comment to one line with no advice.
  (Decision 6 says the rule document will be the same in all arms; this sentence belongs in the
  READMEs, because the configuration differs per arm.)

### 5. Low. Arm L's README layout omits two directories of its workspace

- `poc/app/golint/README.md` lists the same parts as P's, but not `arch/` (the manifest) or
  `lib/contract` (the contract checks). V's README lists `system.vulcan`, which holds V's
  manifest. So V's layout covers its whole workspace, and L's does not.
- Favors: V, a little. An L agent finds the manifest through `PRIMER.md`, after more reading.
- Fix: add two rows to L's README: `arch` "The manifest of the architecture" and `lib/contract`
  "The contract checks". Do the same for any directory that a later fix adds (item 1).

### 6. Low. The overview comment of `system.vulcan` is stale

- `poc/app/vulcan/system.vulcan` line 1: "Stockroom is an order, inventory, and payment service".
  The READMEs and `stockroom.md` say "an order, inventory, purchasing, and payment service, with
  webhooks". P's and L's `main.go` say only "runs the Stockroom service", and their overview above
  `run()` names purchasing and events. L's `arch/arch.go` has no stale text.
- Favors: P and L, a little: V's top-level file is the one that undersells the parts.
- Fix: in `system.vulcan`, use the README's sentence.

### 7. Low. Arm V's check takes about three times as long

- Warm cache, one body file changed, on this host: P's check takes 0.9 s, L's 1.4 s, V's 4.6 s.
  It is the toolchain (generation, `go vet` on the generated code, and the build), as in P3.
- Favors: P and L on wall-clock time. It is part of the measured cost (O6), not a fault.
- Fix: none for parity. Make sure the time budget of a run is not so tight that 3 s per check run
  matters, and report check time per arm.

### 8. Low. Test fakes take more code in P and L

- In V, a test binds only the ports that it calls (`vulcan.Bind(t, &Store, fake)`). In P and L, the
  constructor takes every port, so a test must pass a value for each, and each fake has every
  method of its interface (or embeds the interface). For example, `events.NewHub` needs a Store
  fake with 8 methods. In L, the fakes also go through the `checkedX` wrappers, so they meet the
  port contracts, as in V.
- Favors: V on "add a test" tasks, a little. It is inherent to the two models and is part of the
  treatment.
- Fix: none. Tell the task writer, so that ordinary tasks that ask for tests are counted knowingly.

### 9. Low (kept from P3, item 11). Startup error text differs

- Arm V prefixes errors with `Stockroom/Api:`, P and L with `stockroom:`. With an unknown flag, V
  prints the unknown flag after the required ones and in other words (`unknown flag --bogus`
  against `unknown or malformed flag "--bogus=1"`). Exit codes are the same (1).
- Fix: none needed. A test of a startup check should read the exit code and the flag name only.

## Intended differences that are fine as they are

- **Ports:** consumer-owned interfaces at the top of each P file; in `ports.go` with `checkedX`
  wrappers in L; `needs port` in `component.vulcan` in V. The method sets and their doc comments are
  the same in the three.
- **Port calls and constructors:** `s.store.X` and constructors that take the ports in P and L;
  `Store.X` and empty constructors in V.
- **Contracts:** written as `contract.Requires`, `defer contract.Ensures`, `contract.Old`,
  `contract.Keep`, and `Invariant()` methods in L; as clauses in V; absent in P. This is the
  treatment.
- **`contract.Valid` in L:** L must wrap every built value of a type with an invariant (`shop.Stock`,
  `shop.PurchaseLine`): 9 sites today, in `purchasing` and `store`. V checks the same literals with no
  code. A task that builds such a value (a write-off, a backorder, a change to receiving) costs L one
  wrapper and, if it is forgotten, one VL0905. This follows from a linter, which cannot insert
  checks, and the growth note lists it. The task writer should know that the cost exists.
- **A new port contract:** one `requires` line in V; a wrapper type and method of 4 to 5 lines in L.
  The same reason.
- **Wiring:** `main.go` in P and L (byte-for-byte the same); `system.vulcan` in V. The flags, their
  defaults, the required flags, the environment variables, and the messages match.
- **Invariant hint:** the stock limits and `0 <= received <= qty` are written in L's and V's source,
  not in P's. As P3 said, a task that tempts a change of stock gets both a check and a hint in L and
  V. That is the intended effect, and the report should say that the hint is part of it.
- **The `go` commands** do not work in V's workspace; V's README and guide say so.
- **Generated build directory:** V's `.vulcan/build` is in the workspace after a check, ignored by
  git, marked "DO NOT EDIT" on every file, and removed by the harness before a run and before
  grading.

## Checked and found equal

- **Behavior against `stockroom.md`**, endpoint by endpoint: all 27 routes, the six error codes and
  their statuses, the data limits (sku, name, price, qty, customer, supplier name, cost, URL length),
  the business rules 1 to 14, the event types and when each is raised (including, by reading, no
  `order.refunded` for the refund of a cancellation), the delivery body and its three headers, the
  HMAC signature form, the 5 second timeout, no redirects (`response_status` 302, `failed`), the
  retry (oldest first, attempts counted, delivered ones not resent), and delete (204, then 404 for
  the subscription, its deliveries, and its retry). Same responses, byte for byte, in the three arms
  (probe of 104 requests), and the acceptance suite passes for each.
- **Error messages:** the same text for every error the probe raised.
- **Logs:** the same log lines (`stockroom listens on ...`), and the same log call when an event
  cannot be recorded.
- **Contracts, L against V:** 14 `requires`, 12 `ensures`, 3 invariants in each, clause for clause:
  the invariants of `shop.Stock`, `shop.PurchaseLine`, and `billing.account`; `charge` and `refund`
  of `account` (with `old`); `Ledger.Charge` and `Ledger.Refund`; the ensures of `orders` Create,
  Pay, Ship, Cancel, and Refund, and of `purchasing` Create, Receive, and Cancel; `Catalog.Reserve`
  and `Catalog.Restock`; and the port contracts `Inventory.Store.Receive`, `Orders.Stock.Reserve`,
  `Orders.Billing.Charge` and `Refund`, `Billing.Card.Charge` and `Refund`,
  `Purchasing.Stock.Restock`, and `Events.Sender.Send`. Each maps to a rule or a data limit of
  `stockroom.md`.
- **Structure:** the same ten parts in the three arms, with the same packages, the same ports (by
  name, method set, and doc comment), the same seams, and the same call graph as the parts diagram
  of `stockroom.md`. The same lock in `orders`, `purchasing`, and `events`. `shop.EventTypes` is an
  array in the library in the three arms (allowed by strict mode in L and V).
- **Code bodies:** after normalizing port calls, the bodies of every file are the same in L and V;
  P differs from L only by the contracts and the port declarations; `store/purchases` and
  `store/events` are identical in P and V.
- **READMEs:** the same words except the language name, the wiring file (`cmd/stockroom` against
  `system.vulcan`), the check command, the run command, the pointer to the primer, and V's paragraph
  on the `go` commands (P 215 words, L 233, V 255). Item 5 is the one gap.
- **Check commands:** each matches plan-r2 section 2 and the README of its arm; `go vet` runs in all
  three (V through K0706); no arm runs the race detector.
- **First view of the workspace:** P has 6 top-level entries (`README.md`, `api`, `cmd`, `go.mod`,
  `go.sum`, `lib`); V has 6 (`system.vulcan` in place of `cmd`); L has 9 (also `PRIMER.md`,
  `PROTECTED`, `arch`). The harness adds `RULES.md` to each. The prompt (`runner.Preamble`) is the
  same except the language name: "Go" for P and L, "Vulcan" for V.
- **Size:** code lines as in `r2-app-growth.md` (P 2,712, L 3,049, V 2,681). Bytes of the workspace
  outside the primer: P 84 KB, L 96 KB (with `arch`, `lib/contract`, and `ports.go`), V 85 KB.
- **Settings:** no new flag in round 2; the four flags, their variables, and their defaults are the
  same, and a missing required flag is reported the same way (the exit code and the line per flag).
- **Secrets:** the payments key is tagged `vulcan:"secret"` in L and V, not in P, as in round 1.

## Shared notes for the rule writer, the task writer, and the grader

- A slow subscriber holds the order or purchase-order lock for up to 5 seconds, in all three
  arms: the request that raised the event waits for its deliveries. A holdout test with a slow
  receiver takes that long.
- Validation order: a new purchase order's lines are checked before its supplier, so a request
  with an unknown supplier and a bad line is `invalid`, not `not_found`, in all three arms.
  `stockroom.md` does not fix the order. A holdout test should not depend on it.
- The largest order that the limits allow (100,000,000 cents × 10,000 units) is paid without
  overflow in all three arms.
- The `go.mod` files say `go 1.26.0` while the host runs Go 1.27.3, equal in the three arms, as the
  growth note says. Upgrade them in one change, in all arms at once, before the freeze.

## Fixes

Made by a separate session, which neither wrote the review nor the arms, so that the review and
the fixes stay independent. The owner approved every item, including the compiler change of item
2. Each fix removes a difference between the arms; none is meant to favor one. Commits on main:
C1 = fb0049c (item 1), C2 = 36f10f2 (item 2), C3 = da5ebfa (items 3 to 6), C4 = this section.

1. **Fixed (C1, and C3 for the texts).** Arm L's manifest is split by meaning: `arch/config.go`
   holds the check configuration, `var Config = Settings{Strict: true}` (`Strict`,
   `StrictExempt`, `StrictInterfaces`, `StrictShared`, `Mutators`), and `arch/arch.go` holds the
   persons, the externals, and the parts. `vulcan-lint` reads `Config` only from `config.go`, and
   a check setting in `Manifest` or `Config` elsewhere is VL0902 (or a compile error first).
   `PROTECTED` lists `arch/config.go`, `arch/schema.go` (the schema, which no task needs to
   change), and `lib/contract/*.go`, with a one-line comment and no advice. The harness
   (`runner/config.go`, `configChanges`) now also compares the kind of every part of the base: in
   L, the constructor and its party of each directory of the base manifest (read with the
   linter's own parser, `model.Read`); in V, the `gateway for` and `entry for` lines of each base
   `component.vulcan`. A changed kind fails the run in both arms; an added part does not, and
   neither does a removed one, in both arms (it is not a change of kind). Tests: `TestConfigChangesKinds`, next to
   `TestConfigChangesAndStrays`. VL0002 no longer says "the manifest changes only when the
   request asks for a change to the architecture", and `PRIMER.md` section 1 no longer says to
   change the manifest only then, so the two diagnostics and the two primers say the same. The
   tests of `poc/golint` and the conformance port pass, with the strict cases moved to
   `arch/config.go`.

   So point 4 of the review's fix (no task that adds a part, or `allow_config` for it) is no longer
   needed.

2. **Fixed (C2).** In the compiler, K0607 is an error in strict mode, for a supplier leaf that is
   not in `strict_exempt`, and stays a warning in ordinary mode (`internal/check/wiring.go`,
   `strictFor`). Spec: the K0607 row and §5.4d (rule 6 of strict mode); `docs/poc/guide.md`, so
   `vulcan guide` too. Conformance cases V03 (strict: an error) and V04 (strict with the supplier
   exempt: a warning) beside V02 (ordinary mode: a warning); `go test ./conformance/` passes (145
   cases). `poc/app/vulcan` passes `vulcan check && vulcan test --race=false`; its comment on the
   five Store wires says that K0607 rejects them when they differ (C3).

   One addition beyond the review's text, made for the same reason: arm L reported no second
   instance at all (K0607 was not ported, judgment J7, "equal in effect" while K0607 only warned).
   With K0607 an error in V, a task that wants a second database would pass in L with a second
   `store.Open` call in main and fail in V, a new difference. So `vulcan-lint` gets VL0607 in
   strict mode: main calls a constructor of a leaf that is not in `StrictExempt` a second time
   (in Go every second call is a second instance). Tests: `wiring/testdata/strict.txtar` and the
   ported cases V03 and V04; `poc/app/golint` passes. Arm P, which has no checks, is unchanged.

3. **Fixed (C3).** Both primers are counted with one method, `poc/golint/primer_count.py`
   (bytes, words, and pieces: each run of word characters and each other non-space character, a
   tokenizer-free proxy for tokens; no tokenizer of the run model runs offline here). L's primer
   lost its table of 48 codes and gained what `vulcan guide` teaches and it did not: the layout,
   calling a port, one gateway per external, one instance per constructor, `StrictExempt`, a test
   with a fake, checking before a commit, the commands. L against V, after: 9,283 against 9,464
   bytes (−1.9%), 1,427 against 1,461 words (−2.3%), 2,474 against 2,537 pieces (−2.5%). Before:
   +23%, +39%, +18%. Both counts and the method are in `poc/golint/README.md`, section "The
   primer". Also fixed in passing: `lib/contract/contract.go` pointed to section 4 of the primer
   for contracts, which are section 3.

4. **Fixed (C3).** The three READMEs have one sentence under "Check, build, and run", with the
   same words but for the parenthesis that names the arm's own check configuration:
   "Do not change the check configuration (…) or add suppression comments unless the request asks
   for it." P: "none in this project"; L: "`arch/config.go`, `arch/schema.go`, `lib/contract`, and
   the kinds of the existing parts in `arch/arch.go`"; V: "the `config` block of `system.vulcan`,
   and the kinds of the existing leaves: `gateway for`, `entry for`". The parenthesis follows item
   1, so it differs from the review's draft for L (`arch/`), and it names the kinds in L and V
   alike, since a changed kind now fails the run in both. Each README names only its own arm.
   `PROTECTED` stays the harness's list (item 1). `TestReadme` checks L's README against P's.

5. **Fixed (C3).** L's README layout has the rows `lib/contract` "The contract checks" and `arch`
   "The manifest of the architecture". Item 1 adds no directory.

6. **Fixed (C3).** `system.vulcan` begins with the README's sentence: "Stockroom is an order,
   inventory, purchasing, and payment service, with webhooks: an HTTP API and a SQLite database."

7. **Kept, as the review says.** No change for parity. For the task writer and the grader: V's
   check takes about 3 s more per run of the check; keep the run's time budget loose enough, and
   report check time per arm.

8. **Kept, as the review says.** No change. For the task writer: test fakes take more code in P
   and L than in V (every port of the constructor, every method of the interface); count tasks
   that ask for tests knowingly.

9. **Kept, as the review says.** No change. A holdout test of a startup check reads the exit code
   and the flag name only, not the text.

Checked after the fixes: each app passes its own check command (P `go vet ./... && go test
./...`; L `go vet ./... && vulcan-lint ./... && go test ./...`; V `vulcan check && vulcan test
--race=false`); `STOCKROOM_BIN=<binary> go test ./poc/acceptance/...` passes for the three
binaries; `go run ./poc/harness/cmd/harness tools` builds the tools into `.harness/tools`; `go test
./poc/... ./conformance/` passes; and the round-1 analysis of `experiments/poc/data/p5-deepseek`
prints exactly its `analysis.md`. Not done here: the `go.mod` files of the three arms still say `go
1.26.0` (the review's shared note).
