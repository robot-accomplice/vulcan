# The demo of audit finding N17

Audit finding N17 ([parity-r3.md](../../parity-r3.md), third rerun): with `defaults v1` on the base applications, each seeded case counted as rejected, and each base and reference failed. Plan-r3, amendment 10 answers it with a baseline and a rule match.

The demo material is `poc/harness/testdata/material`, with the audit's two harmless shapes of the behavior rule `RT`, each of which adds one documented function that breaks nothing. `poc/harness/grade/validate_test.go`, `TestValidateWithRules`, runs it. The "Before" column is the audit's run at commit 02f2464. I did not repeat that run.

| | No rules file (unchanged) | `defaults v1`, before | `defaults v1` and a baseline, after, with no match file | The same, with a match file |
|---|---|---|---|---|
| Base, L and V | Pass | **Fail** | Pass | Pass |
| Reference, L and V | Pass | **Fail** | Pass | Pass |
| V builds, and runs its holdout | Yes | **No, builds false** | Yes, as L | Yes, as L |
| `RT`, harmless shapes, L and V | Not rejected, and the item fails | **Rejected, and the item passes** | Not rejected, and the item fails | Not rejected, and the item fails |
| `enforced by L`, `enforced by V` | `RD` | **`RD`, `RT`** | No label, and the command says why | `RD` |
| `set S`, `set N` | `RD`; `RT` | **`RD`, `RT`; none** | None printed | `RD`; `RT` |

The test adds a third shape, `RT/unrelated`, an exported function with no comment. It fires `CODE-08` in both arms. With the match file, the report says "rejected only by an unrelated rule, so it does not count", and `RT` stays in N. An agent in V still sees `vulcan build` refuse it, and one in L does not.

With no rules file, the items, the labels, and the summary are the same as at commit 73ad2eb: `TestValidateNoRulesFile` compares them with a file written at that commit, before the code that reads a rules file.

**The two sets, with the test's match file** (`model-0205` matches `RD`, and nothing else matches):

| | S | G | U | N |
|---|---|---|---|---|
| As each arm ships, with a model check as a rule that fired; the gate reads this | `RD` | none | none | `RT` |
| For the rules file alone | none | none | none | `RD`, `RT` |

No rule on this demo is decided by a code of one arm: `model-0205` is in both.

**A run's grade (plan-r3, amendment 11).** With the fix of N28, `RT/unrelated` in V also builds, and its check is recorded apart: `Builds` is true in both arms, and `final_check` has `passed: false` with `CODE-08`.

The decisions in the test's match file are the test's own, for a check of the mechanism. A fresh session makes them for T1.
