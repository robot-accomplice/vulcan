# Round 3: a default ruleset, written at `init`

Status: accepted by the owner, 2026-10-30, with the amendments in section 9. Steps 1 to 7 go ahead.
Step 8 waits (section 8, decision 3).

## 0. Why this plan

Round 2 stopped at step 3 ([labels-r2.md](../../experiments/poc/labels-r2.md)). Of 28 team rules
written without Vulcan material, `vulcan check` and `vulcan-lint` each enforced the same 3, against
the 8 that its plan required. A diagnostic found the other 25 either outside the model, only
checkable at run time, or declarable through one mechanism. Encoding them could reach 7 at most.

The owner's direction (2026-10-30): a team should not have to write the basics. A new project
should start with a ruleset of good current practice, written when the project is created, that
the team can tune, override, or delete, with the option of another rule source.

This plan does two things:

1. It builds that: `vulcan init`, `vulcan-lint init`, the rules file, and a first ruleset,
   `defaults v1`.
2. It measures what the ruleset covers on rules written blind, and only then decides whether the
   agent runs that round 2 never reached are worth their cost.

**What this plan does not answer.** Defaults are not tied to the language: any rule that `vulcan
init` can write, `vulcan-lint init` can write too. So this plan may give the model more to enforce,
but it gives no reason to expect a gain of the language over the linter. *Prediction, to be
measured:* set U (rules only V enforces) stays empty. Whether a language feature could do what a
linter cannot is a separate question, for [hypotheses-r3.md](hypotheses-r3.md). A
feature from that work would join this plan only by the owner's decision, before the freeze of
section 4.

**Inputs of the design that lean one way**, named so that a reader can weigh them:

- The idea of defaults came after the round 2 result, and is meant to raise the number of rules
  that the checks enforce. The blind tests of section 5 are there to measure that, not to assume it.
- The dependency check of section 3 was designed after reading the diagnostic's summary, which
  says that no check can state which part may depend on which. It saw no rule text, but it is not
  blind to that finding. So round 2's rule set is not a clean test of it (section 5).
- The `transitions` rule kind and the fix of holes H6 and H7 (section 9) come from
  [hypotheses-r3.md](hypotheses-r3.md), whose author read round 2's class counts and the
  diagnostic's summary. So round 2's rule set is not a clean test of them either.
- The shape kinds (section 9) widen what the defaults can state, and so lean toward a higher count
  of enforced rules on T1. They were chosen by the owner after the builder reported that 39 wanted
  rules were out of reach for that reason. They come from the blind wishlist and from no rule set
  or result of a blind test, and they apply the same in L and V.

## 1. Questions

| # | Question | How it is decided |
|---|---|---|
| QD | What share of a team's rules, written blind, do the frozen defaults and the checks enforce? | Set sizes on a fresh rule set, by class (section 5) |
| QN | How often do the defaults reject code that a team would accept? | Findings on code that no one wrote for them (section 5) |
| QA, QV, QR, QC | Round 2's questions: does the model help, does the language add to it, is review shorter, and what does each cost? | As in [plan-r2.md](plan-r2.md), section 3, with its verdicts unchanged. They run only if the gate of section 5 passes |

## 2. The product: `init` and the rules file

**`vulcan init [--rules SOURCE]`** creates a project and writes its rules file. Without
`--rules`, the file is `defaults v1`, built into the tool. With it, the file comes from `SOURCE`.
`vulcan-lint init` does the same for a Go module, with the same rules and the same switches.

**The rules file is an ordinary file that the team owns** (working name `rules.vulcan`; arm L
reads the same format).

- To tune, override, or delete a rule, edit the file. What the check enforces is what the file
  says. There is no hidden layer of built-in rules, so deleting a rule really deletes it.
- Each rule has an id, a kind, its parameters, `error` or `off`, a reason in one line, and the
  source of the practice.
- The head of the file names its origin: `defaults v1`, or a source and hash.

**Other sources.** `SOURCE` is a local file, or an `https` URL. The project records the hash of
what it read. Rules are data: the tool never runs anything from a source. It reads a source only
at `init` and at an upgrade, never during a check, so a check needs no network and cannot change
because a server did.

**Upgrades.** A new compiler never changes a project's rules. `vulcan rules upgrade` shows the
difference from the newer ruleset as a diff to accept or refuse. `vulcan rules diff` shows how the
project differs from its origin, so that a reviewer can see what the team turned off.

**Rule kinds.** A rule file can use only kinds that the checkers implement, the same in L and V.
Which kinds to build comes from step 1 of section 6, not from this list. The first candidates:

| Kind | Meaning | Status |
|---|---|---|
| `forbid-call` | These functions may not be called, or only by these parts | The strict mode list `mutators` does this now, with an error message written for another purpose |
| `only-in` | This effect (network, database, files, clock, randomness, processes) only in parts of this kind | Effects exist; the clock, randomness, and processes need checking |
| `depends` | Which kinds or parts may depend on which | New (section 3) |
| `transitions` | A declared table of the legal changes of a status field, checked at each write | New (section 9; theory T10) |

**Limit of defaults.** A ruleset written before a project exists cannot name that project's parts.
It can speak only of kinds (entry, gateway, leaf, library) and of the standard library. A rule such
as "orders does not use billing" remains a team's own, which is what section 3 is for.

## 3. The dependency check

Each class A seed of round 2 adds a port and a wire, and both checks accept it. Two designs can
reject that:

| Design | How it works | Cost |
|---|---|---|
| By kind | Rules on kinds: libraries depend on no leaf; gateways call no other leaf; entries are called by none | Cheap; it catches few project-specific breaks |
| A locked map | `vulcan rules lock` writes the current wiring into a protected file; a new dependency fails until that file changes | It catches each new dependency, and also each *legitimate* one, so a feature that needs a new wire must change a protected file |

The draft builds both, with the lock on by default only if the owner decides so (section 8). A
lock could make the check look stronger by making any change look wrong. So its cost is measured:
under the rule of F7, a run that changes the lock is no clean success unless its task allows it,
and the ordinary tasks must include some that truly need a new wire. If the lock is on, N0, N1, and
N6 of plan-r2 show what it costs.

## 4. `defaults v1`, and the freeze

A fresh session writes the wishlist, before any new check exists:

- **Inputs:** published practice (layering and dependency rules, OWASP, common Go lints, payment
  and idempotency practice), the model's terms, and the Vulcan spec. It may search the web.
- **Barred:** anything under `C:\Users\jonat\vulcan-poc`, and any mention of the rule classes or
  the set sizes of round 2. Its brief names no target number.
- **Isolation:** the README, `docs/status.md`, `experiments/poc`, and `docs/poc` now state those
  classes and sizes. So the session works in its own folder outside this repository, with only the
  spec and the primer, and it does not read this repository.
- **Output:** each wanted rule, with its reason and source, whether or not it thinks a check can
  do it.

The builder then implements what it can, in both checkers. Each wanted rule that it cannot
implement is listed with the reason. That list is itself a result: it says what the model cannot
state. A fresh session audits L and V for parity on each kind. Then `defaults v1` is frozen, and
its hash is committed before any blind test runs. A change after the freeze makes it `v2`, and the
tests run again.

The applications stay as they are. The rules file is added beside them, so round 2's seeded cases
still apply unchanged. A default that fires on a base application is counted under QN, and is not
quietly removed: it stays on or goes off by a decision that is written down.

## 5. The blind tests

| Test | Rule set | What it can say |
|---|---|---|
| T1 (gate) | A fresh set: a new rule-writer session, the same brief as round 2, written after the freeze; a new labeler, the same procedure (every shape written before a check runs on it; no shape changed because of a check) | Out of sample for all of `defaults v1` and every check |
| T2 (reported) | Round 2's sealed set, with its 168 seeded cases unchanged | Out of sample for the defaults; not for the dependency check (section 0). Reported with and without that check |
| T3 (reported) | A second labeler on T1's rules, on a sample of them | Round 2 had no measure of the labeler's agreement |
| QN (reported) | The findings of `defaults v1` on code not written for it: the applications of this repository, and open-source Go services if the owner allows the download (section 8) | How noisy the defaults are. A sample of findings is judged by a fresh session |

**Gate (unchanged from plan-r2, section 4).** At least 8 rules in S and 4 in N on T1, with both
checks run as each arm ships. If not, no tasks are written and the round stops again, and its
record says what was covered, by class. The bar is not lowered, and rules are not replaced.

**Report on T1 and T2, whatever the outcome:** the sets S, G, U, and N, by class; the share of each
class that moved compared with no defaults; and which default enforced each rule. A rule may land
in S only because of a default that the team would have turned off, so QN is read next to it.

*Predictions, to be measured:* classes A and B move, and classes C and D mostly do not, because
values and behavior are not what these rule kinds state. U stays empty.

## 6. Work order

| Step | Who | Gate |
|---|---|---|
| 1. The wishlist of defaults | A fresh session (section 4) | Each wanted rule has a source |
| 2. The rules file, `init`, `--rules`, `rules diff`, `rules upgrade`, `rules lock`, the rule kinds, the fix of holes H6 and H7, in both checkers; conformance cases | Builder or a fresh session | The same cases pass in L and V; a deleted rule stops firing; H6 and H7 are rejected in both |
| 3. Parity audit of step 2 | A fresh session | No open High finding |
| 4. Freeze `defaults v1`, commit its hash; templates made by `init` | Builder | |
| 5. T1: fresh rule writer, then labeler; T3 | Fresh sessions | **The gate of section 5** |
| 6. T2 and QN; the record in `experiments/poc`, summary in the README | Builder (ids and counts only) | |
| 7. The owner reads the result, and decides whether the runs are worth their cost | Owner | |
| 8. From here, as plan-r2 steps 4 to 9: tasks, audit, seal, pilot, calibration, sizing, main runs, replication, results | As plan-r2 | Its gates |

The harness needs little: `PROTECTED` lists the rules file and the lock (F7), and `harness validate`
runs on the templates that `init` makes. Arm P keeps the team's rules as prose, as in round 2. A
default that matches no team rule is not graded in any arm.

## 7. Cost

Steps 1 to 6 are sessions and local checks, with no provider spend beyond them. Step 8 costs what
plan-r2 estimated (about 370 runs, plus the replication), under its cost gate. The funds are not
committed until step 7.

## 8. Decisions for the owner

| # | Decision | The draft assumes |
|---|---|---|
| 1 | Is the wire lock on by default in a new project? It is the strongest boundary check and the most intrusive | Off in `defaults v1`. T1 is reported both ways, and the gate reads it **off**, so that a check that rejects every new dependency cannot pass the gate by itself |
| 2 | May the QN test download a few open-source Go services? | Only with the owner's yes; otherwise QN reads this repository's applications, and says so |
| 3 | Do the agent runs wait for [hypotheses-r3.md](hypotheses-r3.md), so that one round tests its shortlisted features too? | Yes: steps 1 to 7 go ahead now, because they are cheap, and step 8 waits |
| 4 | Is a URL enough for `--rules`, or does a team need a git source or a registry? | A file or an `https` URL, with a recorded hash |

The owner accepted these assumptions on 2026-10-30. Decision 2 stays as written: until the owner
says yes, QN reads this repository's applications only.

## 9. Amendments at acceptance

| # | Amendment | Where |
|---|---|---|
| 1 | Fix holes H6 (a body-less Go function with an assembly file) and H7 (`net/textproto.Dial`) before the freeze, with the same fix in L and V | Section 6, step 2 |
| 2 | The author of the wishlist works in an isolated folder | Section 4 |
| 3 | Add a `transitions` rule kind (theory T10) before the freeze. It is a coverage feature in both arms, not a claim for the language. The wishlist's author is not told of it | Section 2 |
| 4 | A second session repeats the verdicts of [hypotheses-r3.md](hypotheses-r3.md) blind, before any feature from that document is built | That document, section 9 |
| 5 | A first look at theory T8, on round 1's 72 final workspaces. It is exploratory, and no stop rule reads it | That document, section 9 |
| 6 | Add rule kinds for the 39 wanted rules that step 2 lists under "shape", in both checkers, with no wanted rule reworded. Flow kinds stay out, and those 33 rules stay listed as not stated. Added 2026-10-30: section 2 lets the wishlist decide the kinds, but the first brief for step 2 named four, so this corrects that brief. The gate of section 5 does not change | Section 2; [NOT-STATED.md](../../experiments/poc/r3/defaults/NOT-STATED.md) |
