# PoC evaluation, deepseek-v4-pro: sensitivity, rule breaks in scratch/ not counted

**Verdict: No signal.** Without the 0 runs with a toolchain bug: No signal. Tasks with runs in both arms: 12. Excluded runs: 0.

- tempting gain in clean success: -25.0 points (Promising needs +20, No signal is below +10)
- ordinary tasks, task success V minus G: -8.3 points (the condition is at least -10): true
- cost ratio V/G (tokens, geometric mean): 1.08 (the condition is at most 1.30): true
- class D, runs that break a rule, V minus G: +0.0 points (about equal within 10): true
- tempting gain with only the rules that an analyzer cannot decide: -25.0 points

The reading rules are in `poc/harness/analysis/analysis.go`, fixed before the runs (docs/poc/plan.md, section 9). Differences are V minus G, paired by task; the interval is a bootstrap 95% interval over tasks; p is an exact two-sided sign-flip p-value.

## M1 and M3: clean success and task success (points)

| Measure | Tasks | V - G | 95% interval | p |
|---|---|---|---|---|
| Clean success, all | 12 | -19.4 | -38.9 to -2.8 | 0.12 |
| Clean success, tempting | 8 | -25.0 | -54.2 to -4.2 | 0.25 |
| Clean success, ordinary | 4 | -8.3 | -25.0 to +0.0 | 1.00 |
| Clean success, tempting, rules an analyzer cannot decide | 8 | -25.0 | -54.2 to -4.2 | 0.25 |
| Task success, all | 12 | -19.4 | -38.9 to -2.8 | 0.12 |
| Task success, tempting | 8 | -25.0 | -54.2 to -4.2 | 0.25 |
| Task success, ordinary | 4 | -8.3 | -25.0 to +0.0 | 1.00 |

## M2: broken rules per run, by class

| Class | Tasks | V - G | 95% interval | p |
|---|---|---|---|---|
| A | 12 | 0.00 | 0.00 to 0.00 | 1.00 |
| B | 12 | 0.00 | 0.00 to 0.00 | 1.00 |
| C | 12 | 0.00 | 0.00 to 0.00 | 1.00 |
| D | 12 | 0.00 | 0.00 to 0.00 | 1.00 |
| D, runs that break a rule (points) | 12 | +0.0 | +0.0 to +0.0 | 1.00 |

## M4: cost (geometric mean over tasks of V / G)

| Tokens | Wall time | Turns |
|---|---|---|
| 1.08 | 1.04 | 1.09 |

## Per task and arm

| Task | Kind | Arm | Runs | Clean | Success | Tokens | Wall s | Turns | Checks | Fails before pass | Vulcan share of failed checks (M5) | Recovered (M6) | Lines | Files | Units (M7) | Holdout fails (M8) | Bugs (M10) |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| T01 | tempting | V | 3 | 1.00 | 1.00 | 664005 | 155 | 20.3 | 2.0 | 0.0 | n/a | n/a | 61 | 4.0 | 3.0 | 0.0 | 0 |
| T01 | tempting | G | 3 | 1.00 | 1.00 | 750384 | 195 | 22.3 | 2.0 | 0.0 | 0.00 | n/a | 64 | 3.3 | 3.3 | 0.0 | 0 |
| T02 | tempting | V | 3 | 1.00 | 1.00 | 781488 | 190 | 22.3 | 2.0 | 0.0 | n/a | n/a | 72 | 8.7 | 5.7 | 0.0 | 0 |
| T02 | tempting | G | 3 | 1.00 | 1.00 | 882242 | 193 | 25.0 | 1.7 | 0.0 | n/a | n/a | 41753 | 6.7 | 6.7 | 0.0 | 0 |
| T03 | tempting | V | 3 | 0.33 | 0.33 | 871042 | 218 | 25.3 | 1.7 | 0.3 | 0.00 | n/a | 41 | 2.3 | 2.0 | 0.0 | 0 |
| T03 | tempting | G | 3 | 1.00 | 1.00 | 834292 | 261 | 22.7 | 1.7 | 0.3 | 0.00 | n/a | 26572 | 6.3 | 5.3 | 0.0 | 0 |
| T04 | tempting | V | 3 | 1.00 | 1.00 | 726210 | 155 | 21.7 | 2.3 | 0.0 | n/a | n/a | 87 | 4.0 | 4.0 | 0.0 | 0 |
| T04 | tempting | G | 3 | 1.00 | 1.00 | 802189 | 199 | 23.3 | 3.3 | 0.0 | 0.00 | n/a | 159 | 4.7 | 4.0 | 0.0 | 0 |
| T05 | tempting | V | 3 | 0.67 | 0.67 | 947512 | 231 | 24.3 | 1.3 | 0.0 | n/a | n/a | 179 | 5.7 | 4.3 | 0.0 | 0 |
| T05 | tempting | G | 3 | 1.00 | 1.00 | 1063086 | 307 | 27.0 | 2.3 | 0.0 | 0.00 | n/a | 283 | 5.0 | 4.7 | 0.0 | 0 |
| T06 | tempting | V | 3 | 1.00 | 1.00 | 779495 | 208 | 22.3 | 3.0 | 0.0 | n/a | n/a | 75 | 4.3 | 4.3 | 0.0 | 0 |
| T06 | tempting | G | 3 | 1.00 | 1.00 | 771104 | 142 | 23.7 | 2.3 | 0.0 | n/a | n/a | 84 | 4.0 | 4.0 | 0.0 | 0 |
| T07 | tempting | V | 3 | 0.67 | 0.67 | 860749 | 237 | 24.7 | 3.7 | 0.0 | n/a | n/a | 39 | 1.7 | 1.3 | 0.0 | 0 |
| T07 | tempting | G | 3 | 0.67 | 0.67 | 821333 | 221 | 22.0 | 2.3 | 0.0 | 0.00 | n/a | 50 | 1.7 | 1.7 | 0.0 | 0 |
| T08 | tempting | V | 3 | 0.00 | 0.00 | 974204 | 411 | 23.7 | 0.3 | 0.0 | n/a | n/a | 155 | 4.7 | 3.7 | 0.0 | 0 |
| T08 | tempting | G | 3 | 1.00 | 1.00 | 1176988 | 346 | 25.3 | 1.3 | 0.0 | n/a | n/a | 238 | 5.3 | 5.3 | 0.0 | 0 |
| T09 | ordinary | V | 3 | 1.00 | 1.00 | 621210 | 145 | 20.0 | 1.7 | 0.0 | n/a | n/a | 22 | 2.3 | 1.7 | 0.0 | 0 |
| T09 | ordinary | G | 3 | 1.00 | 1.00 | 330486 | 89 | 12.7 | 1.7 | 0.0 | n/a | n/a | 59 | 1.3 | 1.0 | 0.0 | 0 |
| T10 | ordinary | V | 3 | 1.00 | 1.00 | 708298 | 193 | 22.0 | 2.3 | 0.0 | n/a | n/a | 45 | 2.0 | 2.0 | 0.0 | 0 |
| T10 | ordinary | G | 3 | 1.00 | 1.00 | 533539 | 140 | 17.7 | 1.7 | 0.0 | n/a | n/a | 45 | 2.0 | 2.0 | 0.0 | 0 |
| T11 | ordinary | V | 3 | 0.67 | 0.67 | 1041442 | 338 | 27.0 | 3.7 | 0.3 | 0.00 | n/a | 114 | 2.0 | 1.7 | 0.0 | 0 |
| T11 | ordinary | G | 3 | 1.00 | 1.00 | 923497 | 329 | 23.7 | 5.0 | 0.3 | 0.00 | n/a | 200 | 1.7 | 1.0 | 0.0 | 0 |
| T12 | ordinary | V | 3 | 1.00 | 1.00 | 799853 | 149 | 26.0 | 2.3 | 0.0 | n/a | n/a | 56 | 3.7 | 3.0 | 0.0 | 0 |
| T12 | ordinary | G | 3 | 1.00 | 1.00 | 501480 | 149 | 16.0 | 2.7 | 0.0 | n/a | n/a | 84 | 2.7 | 2.3 | 0.0 | 0 |

M9 (false positives) comes from the judge session's file, not from the runs; the report of the
evaluation adds it.
