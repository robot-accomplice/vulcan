# The rules file: `init`, `rules`, and the rule kinds

This is what step 2 of [plan-r3.md](plan-r3.md) built. `vulcan` and `vulcan-lint` have the same commands and read the same format. Both call the package `internal/rules`, so a rule means the same in each arm. `poc/golint/conformance/gen_rule_cases.py` writes the same cases for both.

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

Each rule has an id, a kind, its parameters, `error` or `off`, a reason in one line, and a source. A word in `in`, `except`, `from`, `deny`, `allow`, or `load` is a kind (`leaf`, `gateway`, `entry`, `library`, `main`) or the directory of a part. `leaf` means a plain leaf. A file with a fault is an error (K0801, VL0801), and so is a directory that is not a part, so that a typing error cannot switch a rule off without a word.

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

## Tests

| Test | What it shows |
|---|---|
| `conformance` and `poc/golint/conformance`, cases `R01` to `R65`, `H6a` to `H6c`, `H7a`, `H7b` | The same case in V and in L: each kind, each effect, a rule off, a deleted rule (R04), no rules file (R05), a fault in the file, and the holes |
| `TestRuleCasesAgree` | Each case lists the same codes in both arms |
| `TestRulesCommands` | `init`, `diff`, `upgrade`, and `lock` on copies of both base applications: the same words, and the same lock. The check passes after the source is removed |
| `TestDefaultsOnBase` | The same rules fire in both arms on the base applications |
| `TestTransitionsOnStockroom` | A real table passes on both applications, and a table without one change rejects that write in both |
| `TestCoverage` | Each of the 210 wanted rules has one row, and each stated rule keeps the wishlist's reason and text |
