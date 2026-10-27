# Hidden material of the PoC evaluation (P5)

Written 2026-10-27 by separate agent sessions, each with only its own inputs (docs/poc/plan.md,
section 11): a rule writer that read only docs/poc/stockroom.md; a labeller and a task writer that
read the rules, the description, and the Go version only; and grader sessions for the tests and
detectors, the seeded violations, and the reference solutions of each arm. The builder session
saw counts and validation results by id only.

The material is sealed (AES-256-GCM, `harness seal`). It is stored outside the repository.

| Item | Value |
|---|---|
| SHA-256 of the sealed file (resealed after strict mode, below) | `d89b3e08473e9fe28229052308a6022d31f6138e1c5da43a6a99d11347cb2bcd` |
| SHA-256 of the manifest (the SHA-256 of each of its 342 files) | `3b9fe077225e04f5878bd5cf9783582b222546202454f0bb29e82cf143ff0d10` |

## Contents

- 16 rules, 4 in each class (A boundaries, B effects, C values, D other); 8 labelled
  analyzer-decidable. 8 are checked by an HTTP rule test and 8 by a source detector.
- 14 tasks: 8 tempting (T01 to T08), 4 ordinary (T09 to T12), and 2 pilots (P01, P02). Every rule
  of classes A to C is targeted by a tempting task; class D by three tasks; R16 by none.
- 14 acceptance test packages (88 tests), and 14 holdout tests.
- 72 seeded violations: 2 or 3 shapes for each rule, in each arm.
- A reference solution of each task in each arm.

## Validation (`harness validate`)

| Items | Result |
|---|---|
| Base applications, both arms | 2 ok |
| Reference solutions | 28 ok |
| Seeded violations | 72 ok: each is caught by its rule's test or detector |

Eleven seeded cases also break another rule (R02 with R01 or R04; R04 with R07; R10 with R15; R12
with R09; R13 with R12 and R16); four of the eleven only in arm V.

**Enforced by V** (vulcan check rejects every seeded case of the rule in arm V), with strict
mode on: 2 of 16 rules, R03 and R05. The compiler rejects 9 of the 36 seeded cases of arm V:

| Rules | Class | Analyzer-decidable | Seeded cases that vulcan check rejects |
|---|---|---|---|
| R01, R02, R04 | A | yes | 1 of 2 each |
| R03 | A | yes | 2 of 2 |
| R05 | B | yes | 2 of 2 |
| R06, R07 | B | yes | 1 of 2 each |
| R08 | B | yes | 0 of 2 |
| R09 to R12 | C | no | 0 of 2 or 3 each |
| R13 to R16 | D | no | 0 of 2 or 3 each |

Rules of class C can still fail at run time in arm V through contracts; the label counts only
what the check command rejects (plan section 4).

## Change after the material was written: strict mode

The first validation ran with strict mode deferred (plan section 10). It labelled 1 of 16 rules
enforced by V (R03), and the compiler rejected 5 of the 36 seeded cases of arm V: 1 of 2 for R01,
R02, and R04, 2 of 2 for R03, and none for classes B to D.

The owner then decided to implement strict mode as the spec already defines it (§5.4d, decision
2: K0123, K0124, K0125, K0210, K0211) and to turn it on in arm V (`config { strict: true }` in
`poc/app/vulcan/system.vulcan`; the guide describes it). This is a change of the compiler after
the hidden rules existed. To limit its effect on the result:

- Only checks that the spec decided before the PoC were added. No check was designed from the
  rules, and the builder session did not read the seeded cases, the tasks, or the tests. It saw
  only the ids and counts above.
- Seven overlays of the material (two reference solutions and five seeded cases) carry their own
  `system.vulcan`. The builder added the same `config { strict: true }` line to each, by a
  script, without reading them, so that they match the new base of arm V.
- The material was validated again (the same 2 + 28 + 72 items, all ok; the same eleven cases
  also break another rule), the labels in meta.json were rewritten, and the material was sealed
  again. The hashes above are of the new seal.

The report of the results must name the rules that strict mode changed (R05, and the partial
gains of R06 and R07), so that a reader can take the result with or without them.
