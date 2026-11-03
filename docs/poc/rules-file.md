# The rules file: `init`, `rules`, and the rule kinds

This is what step 2 of [plan-r3.md](plan-r3.md) built, with amendment 6 of its section 9. `vulcan` and `vulcan-lint` have the same commands and read the same format. Both call the package `internal/rules`, so a rule means the same in each arm. `poc/golint/conformance/gen_rule_cases.py` writes the same cases for both.

`defaults v1` is a draft. It is not frozen, and no blind test has run on it. Its list of what it cannot state is in [NOT-STATED.md](../../experiments/poc/r3/defaults/NOT-STATED.md).

## Files of a project

| File | Who owns it | Who reads it |
|---|---|---|
| `rules.vulcan` | The team | Each check. It is what the check enforces |
| `rules.origin` | `init` and `rules upgrade` write it | `rules diff` and `rules upgrade` only. It is the source as it was read, so a diff needs no network |
| `rules.lock` | `rules lock` writes it | A check, only when a rule with `map locked` is on |
| `rules.baseline` | `rules baseline` writes it, and the team owns it | Each check and build, when the file exists. It holds rule findings that a check does not fail on. It joins the rules file and the lock in `PROTECTED` at the later step that sets that up |

A project with no `rules.vulcan` has no rules. A check reads no source of rules, so the rules need no network and cannot change because a server did. The go command can still fetch a module that the cache lacks, in both arms. A check does not read `hash` or `rules.origin`, so a changed origin does not fail a check. `rules diff` and `rules upgrade` refuse it. Nothing in these files runs.

## Format

```text
origin defaults v1
hash sha256:...
lists sha256:...

# A comment stays when an upgrade changes other rules.
rule ERR-08
  kind forbid-call
  level error
  funcs os.Exit log.Fatal
  in leaf gateway entry library
  reason Only main may end the process, and main is generated here.
  source Uber Go Style Guide (https://...)
```

Each rule has an id, a kind, its parameters, a level, a reason in one line, and a source. A level is `error`, `warn`, or `off`. A finding of a rule at `warn` is printed, and the check still passes. A word in `in`, `except`, `from`, `deny`, `allow`, or `load` is a kind (`leaf`, `gateway`, `entry`, `library`, `main`) or the directory of a part. `leaf` means a plain leaf. A file with a fault is an error (K0801, VL0801). So is a directory that is not a part, a word of `require` that a place does not take, and a place with nothing to apply. So is each name with the project's module path that the project lacks, in `funcs`, `deferred-ok`, `deny-types`, `types`, `param-type`, and `type`, and a type or a field of a table. A field of a `literal` rule is a fault when its type lacks it, and that is read for a type of the project, and for another type that a package uses. This holds for a name inside a container, such as `map[string]shop/lib/money.Order`, and for a prefix, which some name must have. A name in `main` is a fault in both arms, because no rule reads `main`. A name in `lib/contract` is not read in either. A port is named by its port name in both arms, for example `funcs (shop/api/billing.Ledger).Save`. A port has no field in V, because it is a package name there, and L's constructor takes a port as a parameter, so a rule `at field` or `at param` on a port type can fire in L only. A name outside the project is not checked: `funcs os.Exitt`, `type net/http.Clent`, and `deny-types context.Contex` are accepted, and the rule then rejects nothing. A field of a type outside the project that no package uses is not read, and changes nothing.

`lists` in the head is the version of what the tool holds and the file does not: the effect lists, `form`, and `at`. It does not cover the code of a kind, or the model's own lists.

## Commands

| Command | What it does |
|---|---|
| `init [--rules SOURCE] [dir]` | Writes `rules.vulcan` and `rules.origin`. `SOURCE` is a local file or an `https` URL, and the head records it and its hash. It creates the files of a project that the directory lacks, with strict mode on. It refuses to replace a `rules.vulcan` |
| `rules diff [dir]` | Shows how the rules differ from their origin: deleted, turned off, changed, added |
| `rules upgrade [--rules SOURCE] [--accept HASH] [dir]` | Shows the difference from the newer ruleset and changes nothing. With the hash that it printed, it applies the difference. A rule that the team changed or deleted stays as the team has it. If the source changed after the team read the difference, the hash does not match and nothing changes. It also shows when the tool's lists differ from `lists` in the head, as `rules diff` does, because the same rules then enforce more or less; accepting records the new version |
| `rules baseline [dir]` | Writes each current rule finding to `rules.baseline`. It refuses a model error and a fault in the rules |
| `rules lock [dir]` | Writes the current wires, and the imports of libraries, to `rules.lock`. An error of a rule does not stop it, so it can clear the error of a new dependency. Any other error stops it, in both arms |

## Rule kinds

| Kind | Parameters | Code | What it rejects |
|---|---|---|---|
| `forbid-call` | `funcs`, and optional `in`, `except`, `files` | K0802, VL0802 | A use of a listed function, method, or package-level value. A name that ends in `*` is a prefix. `files` is `all`, `code`, or `test` |
| `only-in` | `effect`, `in`, and optional `files` | K0803, VL0803 | A use of an effect outside the listed parts. The effects are `network`, `dial`, `serve`, `database`, `files`, `clock`, `randomness`, and `processes` (`internal/rules/effects.go`) |
| `depends` | `from`, and `deny` or `allow` or `packages`, and optional `except`, `via`, `files` | K0804, VL0804 | A wire or an import from a part to a part that the rule does not allow, or an import of a listed package. `via` is `wire`, `import`, or `any` |
| `depends` | `map locked` | K0806, VL0806 | A wire or an import of a library that `rules.lock` does not hold. Off in `defaults v1` (plan-r3, decision 1) |
| `transitions` | `type`, `field`, `allow`, and optional `initial`, `load` | K0805, VL0805 | A write of a status field that does not follow the table |
| `declaration` | `at`, and optional `names`, `except-names`, `types`, `deny-types`, `deny-names`, `require`, `to`, `in`, `except`, `files` | K0807, VL0807 | A declared name, type, tag, doc comment, or comment. `at` says what it reads: `port-signature`, `port-method`, `field`, `settings-field`, `embedded`, `param`, `var`, `doc`, `library`, `external`, `config-entry`, or `wired-port` |
| `call-form` | `funcs` or `param-type`, and `require`, `deny-text`, or `require-text`, and optional `arg`, `unless-comment`, `in`, `except`, `files` | K0808, VL0808 | An argument of a call: `require` is `constant`, `not-nil`, `unexported-type`, or `tagged=KEY`. The texts are regular expressions on a constant. By default it reads the first parameter of a string type |
| `literal` | `type` with `require`, `deny`, or `below`; or `deny-text` or `names`; and optional `in`, `except`, `files` | K0809, VL0809 | A value of a field in a literal of a type, in `new`, in a `var` with no value, or in an assignment; or a string in the code |
| `code-form` | `form`, and optional `limit`, `similarity`, `funcs`, `deferred-ok`, `unless-comment`, `in`, `except`, `files` | K0810, VL0810 | A form of code. `form` is `dropped-error`, `discard-error`, `error-compare`, `one-value-assertion`, `channel-size`, `function-complexity`, `switch-exhaustive`, `test-logic`, `suppression`, `return-effect`, `retry-loop`, `duplicate-function`, `unused-name`, or `deprecated-use` |

### The shape kinds

Amendment 6 of plan-r3 adds four kinds for the 39 wanted rules that the first draft listed under "shape". No kind follows the flow of a value, so the 33 flow rules stay listed as not stated. `unless-comment yes` removes a finding when a comment is on its line or the line above it. That is the rule exactly: any comment counts, also a doc comment. So in a function on one line, the doc comment above it removes a finding in its body, and in the same function on more lines it does not (audit N38). These kinds also read what the two arms hold in different forms, so each arm gives the shared package its facts: its ports, with their doc comments and `requires` clauses; which functions have `ensures`; and its externals, libraries, and entries of the check configuration.

### `transitions`

A table names one field of one struct: `type lib/shop.Order`, `field Status`, `allow pending->paid paid->shipped *->cancelled`. The states are the values of constants. The check reads one function at a time, and it accepts a write only when the new value is a constant of the table and the state before it is known and allows the change. A state is known from a guard (`if`, an early return, `switch`, `==`, `!=`, `&&`, `||`) or from an earlier write.

It rejects what it cannot follow: a write through a value that it cannot name, a write after a call that can change the value, a guard through a copy or a function, a write in a loop of a guard from outside the loop, and the address of the field. A value starts in a state of `initial`. A part in `load` may write a value that is not a constant, because that is how a value comes back from a store. A test file can build a value in any state.

Limits, the same in both arms:

- It reads the field in Go. A gateway that writes a status with SQL is outside it.
- A write of a whole struct from another value (`*o = other`) is not a write of the field.
- A function that changes a value through a pointer that another function holds is not followed.

## Holes H6 and H7

- **H6.** A function with no body in a leaf or a library is K0126 and VL0126, and so is an assembly file or an object file (`.s`, `.S`, `.sx`, `.syso`) in its directory, also one for another system. A function that a `//go:linkname` names stays K0122 and VL0122 only. In L, H6 and VL0122 read `main` too (audit N39). V has no `main` that a person writes, so V has no such place.
- **H7.** `net/textproto.Dial` is in both network lists, so it is K0205 and VL0205, and K0125 and VL0125 in an entry leaf in strict mode.

## What the parity audit added to the model

[parity-r3.md](../../experiments/poc/parity-r3.md) found these. Each is the same in both arms, unless it says otherwise.

- **Wires in L (VL0909, amendment 7).** L reads a wire only in a plain form: the argument is a call of a leaf's constructor, or a local variable with one value that is such a call or such a variable. Each other wire in main is an error: a helper, a closure, a field, a slice, a map, a variable with a second value, a constructor as a value, and main's own type. V declares its wires, so it has no such limit. This is a cost of L's form.
- **An import that no part holds (VL0002 in L, K0008 in V).** `go` does not list a package under `testdata`, so L reads it through the import.
- **What brings the code in (K0002, VL0002, the same rule in both arms).** Both arms read each `replace` of the root `go.mod` and of a root `go.work`, and reject each one whose target is a local path, inside the project or outside it (plan-r3, amendment 12). A local path is relative or absolute, in the form of any host: `./x`, `../x`, `/x`, `C:/x`, `\\server\share`, `..\x`. The check reads the text of `go.mod` alone, and asks the file system nothing, so a link, a junction, an 8.3 path, `/proc/self/cwd`, and a target that does not exist have no effect on the result. A `replace` by a module path and a version passes. An `ignore` directive of the root `go.mod` fails too, in both arms (audit N41): the go command leaves an ignored directory out of `./...`, so no check loads it, and it still builds that directory when something imports it. A host that is not Windows refuses a path with a backslash, so `go.mod` does not parse there; the check then reads that line, so the `replace` fails in the same way on each host. One function in `internal/rules` does this for both arms, and each arm prints its own code. A root `go.work` also fails by itself, in both, because it changes what the go command builds. V ignored one before, so V is stricter on a harmless `go.work` than it was. Each arm stops there and reports nothing else of the project: the go command reads these in a way that differs by host, and this gives one result on Windows and on Linux. L reads them before its driver loads a package.
- **A go.mod below the root (K0002, VL0002).** A walk also rejects it, by any case of its name, whether anything replaces it or not. It leaves out two places at the root, each for a reason: `.vulcan/build` holds V's own `go.mod`, and `.git/objects` is what git alone writes, and it can be large. No `replace` can reach either, because a `replace` by a local path is rejected. The walk also rejects each link to a directory in the project, a symlink or a junction, at the link (audit N40). The go command does not follow a link when it expands `./...`, so L's check did not load a leaf behind one, and `go build` follows it. V gave K0003 there. Now both arms give K0002 or VL0002 at the link, from the same walk. A go.mod that nothing replaces brings in nothing. The walk stays because V's K0002 rejects a Go file in a plain directory and `go ./...` does not list a nested module, so without the walk L would pass that file, and because the owner's rule for N10 says so.
- **A Go file in a directory that the walk skips.** A `.go` file under `testdata`, a name that starts with `_` or `.`, or `vendor` passes in both arms when nothing imports it, also a fixture. An import of such a path fails in both (VL0002 in L, K0008 in V). In a plain directory that no part holds, a Go file fails in both: K0002 at the file in V, and VL0002 at the package in L.
- **A project that uses `go mod vendor` (K0002, VL0002).** `vendor/modules.txt` at the root, by any case of its name, fails in both arms, because the build then reads vendored code. So such a project passes neither check until it removes `vendor` and builds from the module cache. V also gives K0209 for a module that the cache lacks, and V never reads `vendor`.
- **A file that the build ignores (K0901, VL0901).** Behind a build tag or for another system. Without this, what a rule reads would depend on the host.
- **A package with a type error.** A rule does not read it, in either arm. A type error in a test leaves the package without its tests. Both arms fail, and they print other things: V also prints model codes, and the findings of each part that depends on the broken one, and L's driver skips those parts. So V's agent reads more on such a run. Equal text would need a change of L's driver or of V's stages, and this does not change what passes.
- **A part named `vendor`.** V skips `vendor` by name at the root only, so `lib/vendor` can be a part in both arms. The go command lists it at any depth. V still skips `testdata`, and a name that starts with `_` or `.`, at any depth, because the go command does too.
- **A wire in code that does not run.** L reads each constructor call in `main`, so a helper that `main` never calls still adds an edge, and a wire rule fires on it (case `LW12`). VL0909 holds in every mode, with no rules file too.

## The baseline

`rules baseline` records each current rule finding in `rules.baseline`, a plain file. A check or a build then fails only on a finding that the file does not hold (plan-r3, amendment 10). All rules stay on, and an old finding still prints, marked as old. With no file, the behavior is as before. `vulcan build` refuses only on a new finding, so V builds where L builds. V's check exits 1 and L's exits 3 on a new finding, and each exits 0 with only old ones. `rules diff` shows the size of the file. A model check (K or VL code outside the rule kinds) never goes in it: the command refuses a project with one, and a fault in the rules. A stale entry is reported (K0811, VL0811), and fails nothing.

**The key** is the rule's id, a group (`code`, `wires`, or `decls`), the part, the function that holds the finding with no receiver, and the finding's text without the rule's reason, with a count. It has no line number and no file.

| Change | Result | Which way it leans |
|---|---|---|
| An edit above a finding | Old. No line number in the key | Toward fewer false alarms |
| A function moved to another file of its part | Old. No file in the key, so an arm's other file names do not matter | Toward fewer false alarms; a move adds no break. With a file in the key, each refactoring would fail |
| A function renamed | New, and the old entry is stale | Strict: a team sees it. An agent can run `rules baseline` again, which is the harm below |
| One more finding of a key | New. Which of them prints as new is the last in the code's order | Strict on count, and loose on which one |
| A finding replaced by another of the same key in one function | Old | Loose: the count is the same, and the key cannot tell them apart. The name of the call is in the text of `discard-error` and `call-form`, so another call there is new |
| A finding moved to a method of another type, with the same method name | Old. No receiver in the key | Loose |
| A finding at package level moved between variables | Old. Outside a function the key has no symbol | Loose |
| A finding that moves to another function | New, and the old entry is stale | Strict |
| A method that becomes a plain function | Old. No receiver in the key, because a helper is a method in L and can be a plain function in V | Toward equal keys in the arms |
| A changed reason in the rules file | Old. The key holds the finding's text, not the reason | Toward fewer false alarms |

**Equal keys.** `TestBaselineKeysAgree` writes the file in each arm for forms that one arm has alone: V's lowered return and clauses, L's contract calls, a port in `component.vulcan` against `ports.go`, a wire in `system.vulcan` against `main`, and a library. The files are the same. On the base applications, the keys differ only where the applications do: V's `startup` is L's `Serve`, and L's `lib/shop` has no package comment.

**The tool's version.** A baseline holds the text of each finding, as the tool writes it. A new version of the tool can change a text, for example to name a call (audit N29), and the hash of the rules file stays the same. A baseline from the older tool then gives new findings and stale entries, and `rules baseline` writes it again (audit N36). So a baseline is valid for one version of the tool, and an upgrade of the tool must write it again.

**The harm.** A team can hide a rule's findings in this file, and an agent can too unless it is protected: `rules baseline` makes each current finding old. So it must join `PROTECTED`, at that step. `rules diff` shows its size so that a reviewer can see it. It also lowers what a default shows an agent on old code (plan-r3, section 0). `QN-base-apps.md` measures with no baseline, so it does not hide a finding.

## `harness validate` with a rules file

A workspace counts as rejected only by a new finding. Each item records what fired: the rule's id, the digits of its code, the part, and the rule's text. A label needs `-match`, a file with a decision on each pair of a team rule and a rule that fired (plan-r3, section 5). A model check counts as a rule that fired, named `model-0205` and so on, with each word that names an arm taken out. With no match file, the command reports the rejected items, writes no enforced label, and says why. `-export-judge` writes the pairs for a fresh judge, with no arm named and no set sizes. A case that only an unrelated rule rejects does not count, and the report lists it. With no rules file, nothing changes: `TestValidateNoRulesFile`. [N17-demo.md](../../experiments/poc/r3/defaults/N17-demo.md) gives the demo.

**With a rules file, a run is graded so that `Builds` means one thing in each arm: the code compiles** (plan-r3, amendment 11). The grader builds V from a copy with no rules files, and records the check apart, in L and V, in the field `final_check` of the run. `vulcan build` does not change for its users. `TaskSuccess` then reads the same in each arm for a rule's finding. A model error still refuses V's build, and L's `go build` does not read it, so `Builds` differs there. So `TaskSuccess` counts a code of the shared model check in `final_check` as no success, in both arms (audit N33). The shared codes are those that both arms emit, from the table of `neutral.go`. A code that one arm alone emits (19 in V, 8 in L) does not fail M3: `final_check` lists it as `one_arm`, and `record.CountModel` counts those runs, as a difference between the arms. It also counts, in each arm, the runs with a shared code (`Model`), and the runs that a shared code alone decides (`Decided`). `Decided` is L's count alone: in V such a code stops the build, so V's count is 0 by definition (audit N42). A report compares `Model` between the arms, not `Decided`. With no rules file, a record has no such field, and nothing changes.

The copy for V's build (`copyForBuild`, audit N34) holds each regular file of the workspace but the four rules files, also a `*.db` and `.git`. It passes over each entry that is not a regular file or a directory, a symlink, a junction, a socket, or a FIFO, and it opens none of them. It stops with an error after two minutes. L builds in the workspace itself, so this copy is V's alone. A known limit, in V alone (audit N43): on a grader that runs on Linux, a source file that is a symlink, or a file with mode 000, gives V no build, and L builds. The harness grades on Windows, from a workspace on the Windows drive, and there neither form gives a difference. A link to a directory fails the check in both arms (N40).

What an agent sees during a run, with one new finding (`TestAgentK35`):

| | Check | Build | Test |
|---|---|---|---|
| V | `vulcan check` 1 | `vulcan build` 1 | `vulcan test` 1 |
| L | `vulcan-lint` 3 | `go build` 0 | `go test` 0 |

So an agent in V cannot run its tests with a new finding, and one in L can. That stays a cost of V's form, and the runs measure it. The grader's result is another thing: with the same finding, each arm has `Builds` true, and its check failed.

`harness validate` prints the sets twice: as each arm ships, with a model check as a rule that fired, which the gate of plan-r3, section 5 reads; and for the rules file alone. It lists apart each team rule that a code of one arm decides. A label is written only from the first. It writes no label when a base is rejected.

The judge reads a model check as one sentence for each digit of its code, from a table (`poc/harness/grade/neutral.go`), the same whichever arm fired, under an id with no digit. So a code that one arm alone has reads like any other. The id is a hash of the code with no salt, and the sentence of a one-arm code describes a form that one arm has, so a judge who reads `neutral.go`, or knows both forms, can find the arm of 27 codes (audit N35). A judge does not read this repository (plan-r3, section 5), so this stays. A salt would not change it, because a judge could still read the sentence. `TestModelChecksTable` holds the table to the code, and a scan of the export rejects each word that names an arm.

## Limits

Each holds in both arms.

- **Strict test (the main count):** a rule that a limit reaches is stated in part.
- **Looser test (beside it):** only a limit that the wanted text names changes a status.

| Status | Strict | Looser, the builder's reading | Looser, the auditor's reading |
|---|---:|---:|---:|
| Stated | 8 | 41 | 34 |
| In part | 48 | 15 | 22 |
| Not stated | 154 | 154 | 154 |

The readings of the looser test differ on what "names" means, for 7 rules. `NOT-STATED.md` lists them.

[NOT-STATED.md](../../experiments/poc/r3/defaults/NOT-STATED.md) gives each rule under both. Neither test changes a wanted rule's text.

| Limit | What passes | Why it stays |
|---|---|---|
| A call through a local interface or a generic constraint | `type f interface{ Fatal(...any) }; var x f = log.Default(); x.Fatal("x")`, for `forbid-call`, `only-in`, and `call-form` | Which value the interface holds is flow. A match by method name and signature needs the listed type, which a package need not import, and it rejects an honest interface of the same form. `call-form` does reject a listed method as a value |
| A type defined from a listed type | `type M float64`, `type C context.Context`, `type C net.Conn` | It is another type. An underlying match would reject `time.Duration` under `DATA-14`. For `PAY-01` (no standard type is defined from a float for money), and for `CTX-01`, `CTX-02`, and `ARCH-03` (a match by `types.Identical` on an interface has no false alarm of that form), it would be safe. Leaving it out there is a choice, not a limit of the check: the parameter is not built, and these rules stay in part |
| A break through a helper or another part | `TEST-05` and `TEST-04` read only the test file or the test body, and `SEC-31` and `HTTP-07` only the entry leaf or the gateway, so a helper or a library passes. `SEC-10` reads imports, so `crypto.MD5` of package `crypto` passes (cases `R214` to `R217`, `R249`) | To follow a call is flow. The audit's scripts for its forms `s18` to `s21` and `s25` are not in the repository, so these cases come from its words |
| Any comment | Where a comment removes a finding or gives a reason, also an empty `//` | Whether a comment gives a reason is for a person to judge |
| A module from outside, by its module path | A `replace` by a module path and a version, and a module that the project requires, pass in both arms: no rule reads their code. A `go.work` in a parent directory reaches L when `GOWORK` is not set, and the harness sets it off | The check reads the project. A `replace` by a local path, inside or outside, fails in both arms (amendment 12), and so does a link or a junction in the project to a directory, because the walk rejects each link to a directory (audit N37, N40). The cost: a team that uses a local `replace`, a monorepo for example, cannot use these rules as they are |
| A local copy of the receiver in `CODE-09` | `q := p; q.save(x)` counts as a selector, so it counts more in L than `save(x)` does in V: at limit 60, L fires at 10 such lines and V at 16 | The form reads the receiver by name. To follow a copy is flow |
| A value in a variable | `Timeout: z` with a zero `z`, and a text in a variable | Each kind that reads a constant reads only a constant. Following the variable is flow |
| A finite list | A function that a list does not name | The audit's names are added. It tried names that it chose, not each name of the standard library |
| A name outside the project in a rule | `funcs os.Exitt` | The check does not load what a rule names |
| `-json` of `vulcan-lint` | A finding at `warn`, an old finding, a stale entry, and what brings code in are not in it, and `vulcan --json` has them. With a new finding, `vulcan-lint -json` exits 0 and `vulcan check --json` exits 1 | A `go/analysis` driver has no warning level |
| An external with no text | In V a description is the doc comment that `at doc` and `at external` read, and in L a comment is. Each form is that arm's own | Not a difference in what is rejected: each arm rejects its own form without it |
| One package at a time | `CTX-08`, `CODE-09`, and `CODE-11` | The design of `rules.Unit`, in both arms. It is not a limit of L, which can read other packages through facts, or of V, which holds them all. The owner did not build the wider forms (amendment 9) |

## Judgment calls, and which way each leans

| # | Call | Reason | Leans |
|---|---|---|---|
| 1 | One package for the rule kinds and the commands, not one in each arm | The plan asks for the same behavior. Shared code cannot drift. A parity audit reads less, but it must still read what each arm gives the package: parts, files, and wires | Equal. It gives no test of whether two builders would agree |
| 2 | A rule does not read L's `main` | V's main is generated, so a rule that fired on L's main would have no equal in V. L's main already has VL0205, VL0907, and VL0908. An effect in L's main is still outside `forbid-call` and `only-in` | Looser in L, as call J4 of the linter README |
| 3 | The model's checks stay on, and a rules file does not switch them off | Round 2's seeded cases need them. So "no hidden layer" holds for the rules file only. A team that deletes `ARCH-10` no longer gets K0803, and still gets K0205. Plan-r3, section 2 now says this (amendment 8) | Equal |
| 4 | K0126 and VL0126 are on in every mode | `//go:linkname` is the nearest hole, and K0122 is on in every mode too | Stricter than a strict-only form, in both |
| 5 | `depends` also names packages (`packages`) | ARCH-01, ARCH-02, and others want "does not import", and an import of a package is a dependency. A rule that names a function would miss "even for a type". This parameter came from reading the wishlist, not from a blind set | Adds reach, in both |
| 6 | `forbid-call` also names package-level values | HTTP-02 and HTTP-07 name `http.DefaultClient` and `http.DefaultServeMux` | Adds reach, in both |
| 7 | The dependency rules of plan-r3, section 3, are in `defaults v1`, though the wishlist does not ask for them | Section 3 designs them for the defaults, and section 8 says that only the lock is off | Toward more rules being enforced. T2 reports with and without this check |
| 8 | L reads a wire only in a plain form, and rejects each other wire (VL0909) | The owner's choice, amendment 7. V's wire names its provider | Stricter in L than a typical Go main, and toward an empty U. It is a cost of L's form, which T21 and the form measures count |
| 9 | A lock holds wires and imports of libraries, and no edge to `lib/contract` | L's contracts are that package, and V's are in its runtime. A test checks that the arms write the same lock for Stockroom-lite | Equal |
| 10 | H7 adds one function | The brief names one. `net/rpc/jsonrpc.Dial`, `log/syslog.Dial`, and `log/syslog.New` have the same shape and are in neither list | Equal, and open. Theory T1 tests a closed list |
| 11 | Four shape kinds, and `form` and `at` name closed lists | One kind for each rule would add 30 kinds that differ only in the node they read. So the count of kinds depends on this grouping: `code-form` holds 14 small checks and `declaration` reads 12 places. A reader can count those as checks, not as 4 kinds | Neutral on what is enforced. It makes the design look smaller than its code |
| 12 | A level `warn`, which plan-r3, section 2 now names (amendment 8); a rule at `warn` does not count as enforced | `CONC-03`, `CON-03`, and `CODE-03` each ask for a warning, in the wishlist's own decision, and an error would state another rule. In L a finding at `warn` is printed by the analyzer, because a `go/analysis` driver has no warning level | Equal in effect: the check passes in both. L prints no warning through its driver's own form |
| 13 | A name decides `DATA-14`, `PAY-01`, and `SEC-02`, with case-sensitive patterns | The wishlist's notes say that a tool uses names. These stay "stated in part", because a value under another name is not found | Looser than the wanted text, and each pattern can reject a value that is no time, money, or credential |
| 14 | `CON-01` reads a method as changing state unless its name starts with a read word, or it takes no argument | No kind can see what a method does | Both ways: a write named `GetOrCreate` is missed, and a read under another name is rejected. It stays "stated in part" |
| 15 | `CTX-08` holds only for a port that a gateway provides | What a provider reaches needs a view of the whole module, and `rules.Unit` gives one package. That is the design of the shared package in both arms, not a limit of either (amendment 9) | Looser than the wanted text. It stays "stated in part" |
| 16 | `CODE-09` compares functions inside one package, and `CODE-11` reads unexported names at package level | The same reason: the design of `rules.Unit`, in both arms (amendment 9) | Looser than the wanted text. Both stay "stated in part" |
| 17 | A rule does not read L's `lib/contract`, and in V a library of that name is an ordinary library | V has L's contract code in its runtime, so a rule that fired on it would have no equal in V. `Unit` has a field for this, so only L skips it | Looser in L on that one directory, which `PROTECTED` covers there |
| 18 | A required field of `literal` must be in the literal | A check that follows later assignments would be a flow kind. `s := &http.Server{}` with `s.ReadTimeout = x` after it is rejected | Stricter than the wanted text, in both |
| 19 | `test-logic` rejects `if err != nil` in a test body | The wanted text allows one table loop and no branch. A team that uses it may tune it | Stricter than common Go practice, as the wishlist's author chose |
| 20 | `literal` also reaches `HTTP-01` and `HTTP-02`, though they were not among the 39 | Each had a rest with the reason "shape". A team with a deadline on each call turns `HTTP-02.timeout` off, as the wishlist's decision allows. Each stays in part | More rules enforced, in both. They are named here so that the count of 39 stays traceable |
| 21 | A comment is read from the file that a person wrote, and a comment on the line above counts only if it fills that line | One reading in both arms, which also leaves out V's generated comments | Stricter than before in both: `x := 1 // about x` gives no reason for the next line. An empty `//` still counts |
| 22 | `CODE-09` counts what a person wrote, and a chain of names as a selector of one name | So `p.ledger.Save` and `Ledger.Save` count the same. A lowered return block is a `return`, and a statement that holds only a contract does not count, in both arms. Forms of the chain that were tried, with limit 60 and similarity 90, on the base applications: each node, 10 pairs; a selector of one name, 10; only a selection of a port field, 10; one token for the chain, 5. The builder first built the one-token form, saw the base count fall from 10 to 5, and then changed to a selector of one name, because the brief of that fix, which the coordinating session wrote, said not to reduce findings on a base application unless a fix required it. So the base count is what separated these forms, and the record holds no other reason for the choice. Each form equalizes a port call. The chain form was not changed in this fix. The limit 60 and the similarity 90 are the builder's choices, and the wanted text gives no number. After the fix of the contract forms, and of a call of a leaf's own helper, at limit 60, the base applications give 13, 12, 10, and 6 pairs at similarity 80, 85, 90, and 95, in each arm, as measured after those fixes. Before the helper's fix L gave 14 at similarity 80. So the 10 is a result of two parameters, and no test of a form. A method on a function's own receiver counts as a name, so `p.save(x)` and `save(x)` are the same size, and an `EmptyStmt` does not count. `QN-base-apps.md` says how to read the row | Lowers the count for a longer chain in both, so such a pair reaches the limit later |
| 28 | A model error still refuses V's build in the grader, and `go build` does not read it in L | That is round 2's form, and a build past a model error is not one that V's stages can do: a wiring may not resolve. So `Builds` differs on a model error, and `final_check` records it in both | Toward L on success, for a run with a model error |
| 29 | The grader builds V from a copy with no rules files, not through a flag of `vulcan build` | A flag would be a way around the rules for an agent in a run | No effect on what an agent can do |
| 30 | A model check reaches the judge under a hash of its code, not its digit | The owner asked that a code of one arm not show. The sentence is one reading of each code, and it can be thinner than the message | Toward no sight of the arm, against a little detail for the judge |
| 31 | The walk leaves out `.vulcan/build` and `.git/objects` only | See above. Before, it left out `.git` and `.vulcan/bin` too | A go.mod that nothing replaces fails in more places, in both arms |
| 32 | Each `replace` by a local path fails, with no rules file too (amendment 12) | The owner's decision on N31. The check that walked the file system had a new form at each rerun. A rule on the text alone has one result on each host, and it closes N31, N32, and the listed limit N8. A rule only with a rules file would leave V's check open with no rules file. `internal/gen/leaves.go` no longer turns a relative `replace` into an absolute one, because a checked project has none | Stricter in both arms, equally. A team with a local `replace` cannot use these checks |
| 33 | A model code that is not in the table of `neutral.go` counts as shared | `TestModelChecksTable` holds the table to the code, so this is only a guard | Toward no success, in both arms |
| 34 | The copy holds `.git` and `.vulcan` too, and passes over each link | A copy of each regular file is the plain rule. `.git` gives V's binary its VCS stamp, as L's has. A link to a directory now fails the check in both arms (N40). A link to a file is left: on a grader on Linux, code that reads a file through one builds in L and not in V's graded copy (N43) | Toward L, on a grader on Linux only. The harness grades on Windows |
| 37 | N40: the walk rejects each link to a directory, not only a part's, and the check does not load parts by import path | One shared function, the same in both arms, and it needs no list of parts, so L reads it before its driver. A load by import path would change L's loader alone, and V would still give K0003, another code. A link to a directory that nothing uses fails too | Stricter, equally. A team that keeps a link to a directory in the project cannot use these rules as they are |
| 38 | N41: an `ignore` directive fails with no rules file too | The same reason as call 32: what brings code in is read with no rules file | Stricter, equally. A team that uses `ignore` for a large directory of data cannot use these rules as they are |
| 35 | N38: the rule is stated, not changed | A change would alter what passes in both arms, for an edge of a rule that removes a finding on any comment. Row "Any comment" of the limits already holds that a person judges the comment | Looser, equally |
| 36 | N35: the reversal of the id is accepted, and stated | See the judge's paragraph | Toward a judge's sight of the arm on 27 codes, if a judge reads the code |
| 24 | The key has no file and no receiver | See the table of the baseline. A brief named a file as an example | Toward fewer false alarms when code moves, and toward equal keys in the arms |
| 25 | A root `go.work` fails in both arms, and each arm stops at what brings code in | V ignored a `go.work` before. One rule and one stop give one result on each host. A team loses a `go.work` in the project, and an agent sees only those lines first | Stricter in V than before; equal in the arms |
| 26 | A model check counts as a rule that fired, for the judge | Plan-r3 says "a rule that matches". A team rule such as "only gateways reach the network" is what K0205 and VL0205 state, and with no such pair it could not be enforced by the model at all. Its text comes from the check's own message, with words that name an arm removed, which is a best effort | Toward more enforced rules than a reading that counts only rules-file rules. A judge can say no |
| 27 | With no match file, a seeded case that the check rejects still counts as caught for its item, and gives no label | The item says that something rejects it, and the label waits for the judge | No effect on a label |
| 23 | The main count uses the strict test, by the owner's decision | The sentence as written. It leans toward a low count of stated rules, in both arms. The looser test is printed beside it. The classing of a rule under the looser test is the builder's reading of the wanted text | A lower count of stated rules, in both |

## Tests

| Test | What it shows |
|---|---|
| `conformance` and `poc/golint/conformance`, cases `R01` to `R65`, `H6a` to `H6c`, `H7a`, `H7b`, `R70` to `R137` for the shape kinds, and `R140` to `R185`, `R190` to `R217`, `R220` to `R249`, and `R260` to `R308` for the audit, its reruns, and the baseline | The same case in V and in L: each kind, each effect, each `at` and `form`, a rule off, a deleted rule (R04, and for the shape kinds R71, R89, R98, R111), no rules file (R05), a fault in the file, and the holes |
| `LW*`, `LT*`, `LN1`, `LC1`, `VC1`, `VT*`, `ET1`, `NE6`, `FX9`, `FX16`, `FX17`, `LH1`, `LH2` | Forms where the arms differ in code or place: a wire in L's main, assembly and `//go:linkname` in L's main (N39), code under `testdata`, a nested module, a vendored package, a name with V's prefix in L, and a type error in a test. They are outside `TestRuleCasesAgree` |
| `TestBaselineCommands`, `TestBaselineKeysAgree`, `TestBaselineRefusesModelError` | A check and a build with a baseline, a new finding, a stale entry, equal keys, and no model check in the file |
| `TestReplaceByFileSystem` | Each `replace` by a local path fails, in each form of the path, inside the project and outside, on Windows and on Linux, in both arms. Cases `R271` and `R300` to `R305`: relative, absolute, a drive, UNC, backslashes, and a module path that passes. Cases `R306` to `R308`: an `ignore` directive, in each form (N41) |
| `TestLinkedPart` | A leaf's directory that is a link fails in both arms, at the link: a link to `_keep`, `testdata`, or `.keep` inside, or to a place outside, a junction on Windows and a symlink on Linux (N40). The plain project passes |
| `TestCopyForBuild`, `TestModelError` (`poc/harness/record`) | The grader's copy, and a model code in the final check, with the counts `Model` and `Decided` (N42) |
| `TestAgentK35` | What an agent sees with one new finding |
| `TestValidateNoRulesFile`, `TestValidateWithRules`, `TestBuildsMeansCompiles`, `TestNoLabelsWhenBaseRejected`, `TestModelChecksTable` (`poc/harness/grade`) | No change with no rules file, the demo of N17, one meaning of `Builds`, a model error as no success in both arms, an embedded `*.db` that builds in both, a link to a directory that fails in both (N40), no label on a rejected base, and the judge's table |
| `TestRuleCasesAgree` | Each case lists the same codes in both arms |
| `TestRulesCommands` | `init`, `diff`, `upgrade`, and `lock` on copies of both base applications: the same words, and the same lock. The check passes after the source is removed |
| `TestDefaultsOnBase` | The arms agree by rule, part, and count on the base applications, with one named difference in the applications |
| `TestListsInUpgrade`, `TestDiffSaysTurnedOff`, `TestLockRefusesModelError`, `TestInitIsClean` | The head's lists, the words of `rules diff`, a lock that an error stops, and the Go version that `init` writes |
| `TestTransitionsOnStockroom` | A real table passes on both applications, and a table without one change rejects that write in both |
| `TestCoverage` | Each of the 210 wanted rules has one row, and each stated rule keeps the wishlist's reason and text |
