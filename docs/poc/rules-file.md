# The rules file: `init`, `rules`, and the rule kinds

This is what step 2 of [plan-r3.md](plan-r3.md) built, with amendment 6 of its section 9. `vulcan` and `vulcan-lint` have the same commands and read the same format. Both call the package `internal/rules`, so a rule means the same in each arm. `poc/golint/conformance/gen_rule_cases.py` writes the same cases for both.

`defaults v1` is a draft. It is not frozen, and no blind test has run on it. Its list of what it cannot state is in [NOT-STATED.md](../../experiments/poc/r3/defaults/NOT-STATED.md).

## Files of a project

| File | Who owns it | Who reads it |
|---|---|---|
| `rules.vulcan` | The team | Each check. It is what the check enforces |
| `rules.origin` | `init` and `rules upgrade` write it | `rules diff` and `rules upgrade` only. It is the source as it was read, so a diff needs no network |
| `rules.lock` | `rules lock` writes it | A check, only when a rule with `map locked` is on |

A project with no `rules.vulcan` has no rules. A check reads no source, so it needs no network and cannot change because a server did. Nothing in these files runs.

## Format

```text
origin defaults v1
hash sha256:...

# A comment stays when an upgrade changes other rules.
rule ERR-08
  kind forbid-call
  level error
  funcs os.Exit log.Fatal
  in leaf gateway entry library
  reason Only main may end the process, and main is generated here.
  source Uber Go Style Guide (https://...)
```

Each rule has an id, a kind, its parameters, a level, a reason in one line, and a source. A level is `error`, `warn`, or `off`. A finding of a rule at `warn` is printed, and the check still passes. A word in `in`, `except`, `from`, `deny`, `allow`, or `load` is a kind (`leaf`, `gateway`, `entry`, `library`, `main`) or the directory of a part. `leaf` means a plain leaf. A file with a fault is an error (K0801, VL0801), and so is a directory that is not a part, so that a typing error cannot switch a rule off without a word.

## Commands

| Command | What it does |
|---|---|
| `init [--rules SOURCE] [dir]` | Writes `rules.vulcan` and `rules.origin`. `SOURCE` is a local file or an `https` URL, and the head records it and its hash. It creates the files of a project that the directory lacks, with strict mode on. It refuses to replace a `rules.vulcan` |
| `rules diff [dir]` | Shows how the rules differ from their origin: deleted, turned off, changed, added |
| `rules upgrade [--rules SOURCE] [--accept HASH] [dir]` | Shows the difference from the newer ruleset and changes nothing. With the hash that it printed, it applies the difference. A rule that the team changed or deleted stays as the team has it. If the source changed after the team read the difference, the hash does not match and nothing changes |
| `rules lock [dir]` | Writes the current wires, and the imports of libraries, to `rules.lock`. An error of a rule does not stop it, so it can clear the error of a new dependency. Any other error stops it |

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

Amendment 6 of plan-r3 adds four kinds for the 39 wanted rules that the first draft listed under "shape". No kind follows the flow of a value, so the 33 flow rules stay listed as not stated. `unless-comment yes` removes a finding when a comment is on its line or the line above it. These kinds also read what the two arms hold in different forms, so each arm gives the shared package its facts: its ports, with their doc comments and `requires` clauses; which functions have `ensures`; and its externals, libraries, and entries of the check configuration.

### `transitions`

A table names one field of one struct: `type lib/shop.Order`, `field Status`, `allow pending->paid paid->shipped *->cancelled`. The states are the values of constants. The check reads one function at a time, and it accepts a write only when the new value is a constant of the table and the state before it is known and allows the change. A state is known from a guard (`if`, an early return, `switch`, `==`, `!=`, `&&`, `||`) or from an earlier write.

It rejects what it cannot follow: a write through a value that it cannot name, a write after a call that can change the value, a guard through a copy or a function, a write in a loop of a guard from outside the loop, and the address of the field. A value starts in a state of `initial`. A part in `load` may write a value that is not a constant, because that is how a value comes back from a store. A test file can build a value in any state.

Limits, the same in both arms:

- It reads the field in Go. A gateway that writes a status with SQL is outside it.
- A write of a whole struct from another value (`*o = other`) is not a write of the field.
- A function that changes a value through a pointer that another function holds is not followed.

## Holes H6 and H7

- **H6.** A function with no body in a leaf or a library is K0126 and VL0126, and so is an assembly file or an object file (`.s`, `.S`, `.sx`, `.syso`) in its directory, also one for another system. A function that a `//go:linkname` names stays K0122 and VL0122 only.
- **H7.** `net/textproto.Dial` is in both network lists, so it is K0205 and VL0205, and K0125 and VL0125 in an entry leaf in strict mode.

## Judgment calls, and which way each leans

| # | Call | Reason | Leans |
|---|---|---|---|
| 1 | One package for the rule kinds and the commands, not one in each arm | The plan asks for the same behavior. Shared code cannot drift. A parity audit reads less, but it must still read what each arm gives the package: parts, files, and wires | Equal. It gives no test of whether two builders would agree |
| 2 | A rule does not read L's `main` | V's main is generated, so a rule that fired on L's main would have no equal in V. L's main already has VL0205, VL0907, and VL0908. An effect in L's main is still outside `forbid-call` and `only-in` | Looser in L, as call J4 of the linter README |
| 3 | The model's checks stay on, and a rules file does not switch them off | Round 2's seeded cases need them. So "no hidden layer" holds for the rules file only. A team that deletes `ARCH-10` no longer gets K0803, and still gets K0205 | Equal. A reader can hold that this is a hidden layer. The owner can decide otherwise |
| 4 | K0126 and VL0126 are on in every mode | `//go:linkname` is the nearest hole, and K0122 is on in every mode too | Stricter than a strict-only form, in both |
| 5 | `depends` also names packages (`packages`) | ARCH-01, ARCH-02, and others want "does not import", and an import of a package is a dependency. A rule that names a function would miss "even for a type". This parameter came from reading the wishlist, not from a blind set | Adds reach, in both |
| 6 | `forbid-call` also names package-level values | HTTP-02 and HTTP-07 name `http.DefaultClient` and `http.DefaultServeMux` | Adds reach, in both |
| 7 | The dependency rules of plan-r3, section 3, are in `defaults v1`, though the wishlist does not ask for them | Section 3 designs them for the defaults, and section 8 says that only the lock is off | Toward more rules being enforced. T2 reports with and without this check |
| 8 | L finds the provider of a wire by the type of the argument, else by the value that main assigned | V's wire names the provider. In L, main's own helper that returns an interface can hide it, and then there is no edge | Looser in L, in a form that VL0907 mostly stops |
| 9 | A lock holds wires and imports of libraries, and no edge to `lib/contract` | L's contracts are that package, and V's are in its runtime. A test checks that the arms write the same lock for Stockroom-lite | Equal |
| 10 | H7 adds one function | The brief names one. `net/rpc/jsonrpc.Dial`, `log/syslog.Dial`, and `log/syslog.New` have the same shape and are in neither list | Equal, and open. Theory T1 tests a closed list |
| 11 | Four shape kinds, and `form` and `at` name closed lists | One kind for each rule would add 30 kinds that differ only in the node they read. So the count of kinds depends on this grouping: `code-form` holds 14 small checks and `declaration` reads 12 places. A reader can count those as checks, not as 4 kinds | Neutral on what is enforced. It makes the design look smaller than its code |
| 12 | A level `warn`, which plan-r3, section 2 does not name | `CONC-03`, `CON-03`, and `CODE-03` each ask for a warning, in the wishlist's own decision, and an error would state another rule. In L a finding at `warn` is printed by the analyzer, because a `go/analysis` driver has no warning level | Equal in effect: the check passes in both. L prints no warning through its driver's own form |
| 13 | A name decides `DATA-14`, `PAY-01`, and `SEC-02`, with case-sensitive patterns | The wishlist's notes say that a tool uses names. These stay "stated in part", because a value under another name is not found | Looser than the wanted text, and each pattern can reject a value that is no time, money, or credential |
| 14 | `CON-01` reads a method as changing state unless its name starts with a read word, or it takes no argument | No kind can see what a method does | Both ways: a write named `GetOrCreate` is missed, and a read under another name is rejected. It stays "stated in part" |
| 15 | `CTX-08` holds only for a port that a gateway provides | What a provider reaches needs a view of the whole module, and L reads one package at a time. V could see more, so this is the same limit in both by choice | Looser than the wanted text. It stays "stated in part" |
| 16 | `CODE-09` compares functions inside one package, and `CODE-11` reads unexported names at package level | The same reason. V could read more, and does not, so that the arms stay equal | Looser than the wanted text. Both stay "stated in part" |
| 17 | A rule does not read L's `lib/contract` | V has that code in its runtime, so a rule that fired on it would have no equal in V | Equal |
| 18 | A required field of `literal` must be in the literal | A check that follows later assignments would be a flow kind. `s := &http.Server{}` with `s.ReadTimeout = x` after it is rejected | Stricter than the wanted text, in both |
| 19 | `test-logic` rejects `if err != nil` in a test body | The wanted text allows one table loop and no branch. A team that uses it may tune it | Stricter than common Go practice, as the wishlist's author chose |
| 20 | `HTTP-01` and `HTTP-02` are now stated, though they were not among the 39 | Each had a rest with the reason "shape", and `literal` states it. A team with a deadline on each call turns `HTTP-02.timeout` off, as the wishlist's decision allows | More rules enforced, in both. They are named here so that the count of 39 stays traceable |

## Tests

| Test | What it shows |
|---|---|
| `conformance` and `poc/golint/conformance`, cases `R01` to `R65`, `H6a` to `H6c`, `H7a`, `H7b`, and `R70` to `R137` for the shape kinds | The same case in V and in L: each kind, each effect, each `at` and `form`, a rule off, a deleted rule (R04, and for the shape kinds R71, R89, R98, R111), no rules file (R05), a fault in the file, and the holes |
| `TestRuleCasesAgree` | Each case lists the same codes in both arms |
| `TestRulesCommands` | `init`, `diff`, `upgrade`, and `lock` on copies of both base applications: the same words, and the same lock. The check passes after the source is removed |
| `TestDefaultsOnBase` | The same rules fire in both arms on the base applications |
| `TestTransitionsOnStockroom` | A real table passes on both applications, and a table without one change rejects that write in both |
| `TestCoverage` | Each of the 210 wanted rules has one row, and each stated rule keeps the wishlist's reason and text |
