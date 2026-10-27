# Hidden material of the PoC evaluation (P5)

Written 2026-10-27 by separate agent sessions, each with only its own inputs (docs/poc/plan.md,
section 11): a rule writer that read only docs/poc/stockroom.md; a labeller and a task writer that
read the rules, the description, and the Go version only; and grader sessions for the tests and
detectors, the seeded violations, and the reference solutions of each arm. The builder session
saw counts and validation results by id only.

The material is sealed (AES-256-GCM, `harness seal`). It is stored outside the repository.

| Item | Value |
|---|---|
| SHA-256 of the sealed file | `570d8a30affa01a000c394e691844081777784e9924de9938fe78f9003ed44fc` |
| SHA-256 of the manifest (the SHA-256 of each of its 342 files) | `4cfcda6cdc924120749d610fe2f0611069bc90a0b867af7c36a9ea1f5c2303c5` |

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

**Enforced by V** (vulcan check rejects every seeded case of the rule in arm V): 1 of 16 rules,
R03. The compiler rejects 5 of the 36 seeded cases of arm V:

| Rules | Class | Analyzer-decidable | Seeded cases that vulcan check rejects |
|---|---|---|---|
| R01, R02, R04 | A | yes | 1 of 2 each |
| R03 | A | yes | 2 of 2 |
| R05 to R08 | B | yes | 0 of 2 each |
| R09 to R12 | C | no | 0 of 2 or 3 each |
| R13 to R16 | D | no | 0 of 2 or 3 each |

Rules of class C can still fail at run time in arm V through contracts; the label counts only
what the check command rejects (plan section 4).
