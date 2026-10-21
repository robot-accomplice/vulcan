# Audit 9: independent re-audit of the development plan (summary)

Commit audited: `98033f9`. Written by a sixth freshly spawned agent, independent of the
conversation that wrote the plan. It reproduced `experiments/sizing/results.md` byte for byte and
ran model variants. **This file is a condensed record**, saved when the session's usage limit was
reached; the full report is in that session's transcript.

## Verdict

**No. One Blocker remains.**

## Blocker

**B1. N0 and N1 count rule-breaking shortcuts as success.** On tempting tasks, arm R's easiest path
breaks a U\* rule and still passes acceptance and review, quickly. Arm V's checker blocks that path,
so V looks slower and less successful exactly where enforcement works. The sizing model assumes V
costs nothing, so it misses this. Simulated P(Continue | L) at 28 × 10 falls from 0.85 to 0.71
(blocked V runs 1.2× slower), 0.53 (1.3× slower, 5 points more failures), 0.16 (1.5×, 20 points).
**Fix:** count success and time for N0 and N1, in every arm, only for accepted changes that keep
the hidden rules their task targets; or compute N0 and N1 only on ordinary tasks, sized for them.
Then re-run the pre-recruiting check with a coupled-cost and a half-margin-cost scenario.

## High (summary)

- **H1.** Only P(Continue | L) binds at the sizing gate; Z and L+cost are about 0 everywhere. A
  half-margin real cost gives P(Continue) about 0.21; task × arm interaction lowers it to 0.61-0.73.
- **H2.** The agent's cost in Vulcan is first seen in milestone 4. Add a cost pilot after
  milestone 2 (6 to 8 ordinary tasks, V against R).
- **H3.** X6 counts every failed or non-building run as failing all holdout tests, which makes N2 a
  hidden, stricter success bound. Count X6 only in accepted changes.
- **H4.** Gate 2's rate test is unsized and noisy (a true rate of 0.4 fails it 18% of the time at 4
  tasks × 5 runs), and the number of U\*-targeting tasks in milestone 0.5 is not fixed.
- **H5.** Arm C has no gap rule: semantics a generator could deliver, but a 15-day generator lacks,
  stay in U\*.

## Medium and Low (titles)

M1 the simulation omits dual-reviewer conditions, N4, N5, the four-rule minimum, and samples 2,000
sign patterns; M2 boxes do not fit loads (milestone 0.5 pre-freeze work alone is about 14 weeks);
M3 six people cannot meet the separations as described; M4 row 5 labels a clear negative "not
judged"; M5 the model changes between milestones; M6 residual reviewer bias, and a reversed
acceptance is undefined; M7 the milestone-3 gate has gaps. L1 to L11: undefined "margins of P's",
N4 is not a t-bound, "smallest design" has no order, estimators undefined, release only after
milestone 4, unanswered-question bias, plan "Stop" wording, `keel` names in decision 4, table
order, spec §7.3 check removal against release 1, no box between milestones.

## Status of audit 8's findings

B1 resolved (binding targets, sizing ran, fixed designs, honest margin change), with new problems
(zero-cost model, gate 2 unmodelled). H1 partly; H2 resolved but created this audit's H3; H3 mostly;
H4 partly.
