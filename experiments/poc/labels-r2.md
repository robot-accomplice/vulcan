# Round 2, step 3: rules and labels

Written 2026-10-29. Step 3 of [plan-r2.md](../../docs/poc/plan-r2.md) (section 4). This record
gives ids, counts, and hashes only: the rules and the cases are hidden material, and the builder
session has not read them.

## Result

The stop rule of plan-r2 (section 4) applies: "At least 8 rules in S and 4 in N, or the round
stops before tasks are written." Set S has 3 rules. **Round 2 stops here, before tasks.**

| Set | Meaning | Rules | Ids |
|---|---|---|---|
| S | L and V enforce it | 3 | R07, R08, R09 |
| G | L only | 0 | |
| U | V only | 0 | |
| N | neither | 25 | R01 to R06, R10 to R28 |

By class (A boundaries, B effects, C values, D other; 7 rules each):

| Class | S | N |
|---|---|---|
| A | 1 | 6 |
| B | 2 | 5 |
| C | 0 | 7 |
| D | 0 | 7 |

A rule is enforced by an arm when that arm's check rejects every seeded case of it:
`vulcan-lint ./...` in arm L, `vulcan check` in arm V. Arm P has no check beyond the build.

What was measured:

- **L and V agree on every case.** Each arm's check rejects 6 of its 56 seeded cases: both shapes
  of R07, R08, and R09, and no case of any other rule. No rule is rejected in some of its cases
  and not in others.
- **Set U is empty.** The desk test ([desk-test-r2.md](../../docs/poc/desk-test-r2.md)) predicted
  that every check of the PoC compiler is "Same in Go". These labels measure that prediction on 28
  rules written without Vulcan material, and they do not contradict it.
- **Few of the rules are ones that either check decides.** 3 of 14 rules of classes A and B (the
  classes that architecture checks address), and none of classes C and D. A class C rule can still
  fail at run time through a contract in L or V. The label counts only what the check rejects, as
  in round 1.

Limits of this result:

- **One rule writer and one labeler.** A different writer could write rules that the checks
  decide more often, and a different labeler could pick other shapes. No second labeler measured
  agreement. The labels are the measure of these 28 rules, not of all rules a team could hold.
- **The shapes were written before either check ran on them**, and no shape was changed after
  the first validation (0 fixes). This keeps the labels from being tuned to a check, in either
  direction.
- **The stop rule was set before the labels**, in the plan the owner accepted. The round is not
  continued with changed rules: rules chosen after seeing the labels would no longer be a fair
  sample.

## How it was made

| Step | Session | Input | Output |
|---|---|---|---|
| Rules | A fresh rule writer, in a folder that held only a copy of [stockroom.md](../../docs/poc/stockroom.md) (no Vulcan material), told to favor no tool or style of enforcement | The functional description and its neutral diagram | 28 rules, 7 per class; 15 checked by an HTTP test, 13 by a source detector |
| Labels | A fresh labeler, in the repository | The rules, the three applications, the harness | A test or detector for each rule; 2 seeded shapes per rule in each arm (56 per arm, 168 in all), the same shape in P, L, and V; `harness validate -write-labels` |

## Validation (`harness validate`, arms P, L, V)

| Items | Result |
|---|---|
| Seeded cases | 168 ok: each is flagged by its own rule's test or detector |
| Base applications | 3 build, and no rule fires on them; the validator lists them as problems only because no holdout tests exist yet (they come with the tasks, step 4) |
| Fixes after the first validation | 0 |

74 seeded cases also break another rule (listed by id in the validator's output).

## The material

The material is in `C:\Users\jonat\vulcan-poc\r2` on the owner's machine, outside the repository.

| File | SHA-256 |
|---|---|
| `manifest.txt` (the SHA-256 of each of the material's 306 files) | `39f0f784da496f83601d55a6d550c8ab43a6678f564a51a284cb6041bd4f77b7` |
| `validate-summary.txt` | `9560ea7f377d34a3afb2131ee69f893a6406ecaa1e5ac2fc4fe4885e71ae5578` |
| `validate/validation.json` | `9c4e12cef43431df3592e12f81d8e01560b981bb9063f93ae9853822dd919861` |
