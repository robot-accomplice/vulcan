# Audit 3: the development plan

Date: 2026-10-21. Commit audited: `ac0b08a`.

Scope: [docs/development-plan.md](../docs/development-plan.md), and the parts of decisions 1 to 6
and `docs/release-1/` that the plan depends on. The owner asked for an antagonistic audit: the
plan counts as accepted only if it survives one.

**Independence.** This audit is not independent. The same assistant wrote audit 2, decisions 1 to
6, and the release-1 scope. It tried to attack its own work, but an outside reviewer would find
things that it cannot. Finding B3 is partly about this.

## Verdict

**The plan does not survive as written.** Its sequence (decide, build, measure, judge) is sound,
and most of its milestones are concrete. But the evaluation in milestone 4 cannot answer the
question that the plan exists to answer, and the plan spends its largest costs before its first
real stop point. Three findings block acceptance. Each has a direct amendment (see
"Amendments"). With amendments A1 to A6, the plan would survive this audit.

## Blockers

### B1. The evaluation cannot tell a language from a linter

The plan's stop rule is "stop or pivot if ordinary Go achieves the same outcomes with materially
less total effort" (milestone 4). The Go arm of decision 6 gets a document of the rules and a
dependency linter such as `depguard` (decision 6, section 1). That is not the strongest form of
"ordinary Go". Go already has the means to enforce most of Vulcan's architecture rules without a
new language:

- G1, G3, and G4 (no references between leaves, no cycles, private types) are import-graph and
  type checks that a `golang.org/x/tools/go/analysis` analyzer can do in a few hundred lines.
- G5 and G6 (network and acquiring reach) are a call-graph analysis, the same one that §5.4c
  specifies, and `x/tools` provides the call graph.
- Consumer-owned ports are already the Go idiom (§5.3 says so).
- Contracts are the part that needs new syntax. Even they can be written as structured comments
  and checked by generated code.

If the Go arm gets a set of such analyzers, P1 and P2 compare the same rules in two forms. If it
does not, the comparison measures a strong tool against a weak one, and a pass says nothing about
the language. The plan does not name the strongest alternative, "Vulcan's rules as Go analyzers",
so it cannot decide between continuing the language and pivoting to that alternative.

### B2. The primary outcomes pass by construction

P1 counts hidden-rule violations that reach review, and P2 counts violations stopped before
review. The author writes the hidden rules and the three tempting tasks (decision 6, section 2),
knowing which rules the Vulcan compiler enforces. In the Vulcan arm, a change that breaks an
enforced rule cannot build, so it cannot reach review. P1 and P2 then measure whether the
compiler enforces its own rules, which the conformance suite already shows. The decision rule
gives "continue" on outcomes that were never in doubt.

Two weaker points add to this. Only 3 of 12 tasks are tempting, so the violation counts come
mostly from 3 tasks; a bootstrap by task over 3 informative clusters gives intervals that mean
little. And nothing measures whether a violation in the Go arm costs anything, so P1 cannot show
that removing violations matters.

### B3. The people who want a result also judge it

- The owner builds both arms of Stockroom (acceptance criteria), so the quality of the baseline
  is in the hands of the party who prefers Vulcan.
- The "operator" judges false positives (M8) and answers agents' questions. The reviewer times
  reviews (M6). Decision 6 does not say who they are. In a one-person project, they are the
  owner.
- The tasks, the hidden rules, and the thresholds come from the same party, and from the same
  assistant that wrote the language decisions.

Sealing the files before the runs prevents later edits. It does not prevent bias in what was
sealed.

## High

### H1. The plan spends most of its cost before its first stop point

The only go or no-go decision is the exit gate of milestone 4. Before it, milestones 1 to 3
build a compiler for a large language subset, a conformance suite of 85 cases, and the same
application twice, to 32 criteria. The plan has no estimate and no time box, and no earlier point
where poor progress stops the work. A cheaper test exists: H-A and H-B are about rules, not syntax,
so a small version of milestone 4 can run with the rules as Go analyzers (B1) before any compiler
exists. If the rules do not change outcomes in that form, a compiler will not change them.

### H2. "Inconclusive" has no consequence

Decision 6 allows one extension, then reports "inconclusive" (section 6). Neither the plan nor
the decision says what the project does next. With 12 tasks and 5 to 10 runs, the 10%
non-inferiority bounds (N1, N2) are likely to be inconclusive. The project then continues by
default, which makes "inconclusive" a pass.

### H3. The compiler can change during the evaluation

Decision 6 records the compiler version of each run, but it does not freeze it. Milestone 4 also
counts "compiler fixes" in total time. If a compiler bug is fixed during the runs, the early and
late runs of the Vulcan arm use different tools, and the Go arm does not change. The protocol
needs a frozen tag and a rule for a blocking bug.

## Medium

### M1. The plan text is out of date

- Line 3: "draft for discussion, 2026-10-19".
- Line 14: "It does not change the specification". Decisions 1 to 5 changed it.
- Lines 25 and 26: command spelling and module paths "remain open until milestone 0". Decision 5
  closed them.
- Lines 42 to 48: milestone 0, item 1 describes the spec before decision 1 and a "preferred
  direction" (reject deferred mutations) that decision 1 did not take.
- The traceability table covers only the nine findings of the lost audit 1, not audit 2.

A reader cannot tell which parts of the plan still bind.

### M2. The spec keeps promises that no milestone tests

- §2 and §11.3: "Resource use and speed must be competitive with Go, Java, C++, and Rust." The
  plan measures only against Go, defers LLVM with no date, and leaves Q22 open. This is the kind
  of retained stronger description that the exit gate of milestone 0 forbids.
- §5.5 (topics, remote ports, transports) reads as normative. Release 1 rejects it, and
  milestone 5 may never run, but the spec does not say "deferred" where a reader meets it. Audit 2
  (H11) asked for this marking, and it was not done.

### M3. Milestone 5 has no entry criterion

"Proceed only after the application establishes a need" names no test of the need. Without one,
the deferred distributed design can come back by preference.

### M4. "Without its author's intervention" names no one

The exit gate of milestone 3 needs a person who is not the author, but the plan does not say who,
or what they may ask.

## Low

### L1. The closure lowering of decision 1 changes caller depth

In a function with a risky deferred call, the body runs in a generated closure. A logging helper
that reports its caller with `runtime.Caller(1)` and `runtime.FuncForPC` then prints
`pkg.F.func1` instead of `pkg.F` (confirmed with Go 1.26.1). The `//line` directives keep the
line right, but not the function name, and code that walks more frames sees one more. This
changes the output of a correct body, which design rule 4 forbids. Go gives user code no way to
hide a frame, and a lowering without the closure (one generated deferred call that runs last)
cannot tell a recovered panic from a running one without calling `recover` itself. Until a better
lowering exists, decision 1 should state the difference as a limit, and the conformance suite
should pin it.

## What holds up

- The order of work (decide the guarantees, then build, then measure) is right, and milestone 0
  did its job: the decisions closed real holes (the recover path, `linkname`, unbounded claims).
- Release 1 rejects unsupported constructs with K0009 instead of ignoring them.
- The acceptance criteria are testable, and trace to concrete defects of test 3.
- Decision 6 is preregistered, scripted where it can be, and reports every run.

## Amendments

Each amendment closes the findings it names. With A1 to A6, this audit finds no blocker.

| # | Amendment | Closes |
|---|---|---|
| A1 | Add a third arm, or replace the Go arm's linter with it: **Go with Vulcan's rules as analyzers**, with contracts as checked comments where possible. The stop rule compares Vulcan with this arm. Pivoting to it is a named outcome. | B1 |
| A2 | Write the hidden rules from a source the compiler did not shape: for example, a person who has not read the spec writes rules for an order service, and the grading reports violations of rules that Vulcan enforces and rules that it does not, apart. Raise tempting tasks to at least half of the tasks. Add an outcome that measures the cost of a violation in the Go arm (defects or rework that it caused later). | B2 |
| A3 | Name the roles. The Go arm is built or reviewed by a Go developer who does not work on Vulcan. The operator and the reviewer are not the owner, or the owner's judgments (M8) are checked by a second person on a sample. | B3 |
| A4 | Add **milestone 0.5: the rules without a compiler.** Implement the architecture rules of decision 2 as Go analyzers (this is also the arm of A1), and run a reduced decision 6 on a small application. Stop if P1 and P2 fail there. Give milestones 1 to 3 a time box each; a miss of more than half its box stops for a re-plan. | H1 |
| A5 | Give "inconclusive" a consequence: an inconclusive primary outcome is treated as a fail for the decision rule; an inconclusive non-inferiority bound leads to "narrow". | H2 |
| A6 | Freeze the compiler at a tag for the main runs. A blocking compiler bug ends the affected run as a failure of the Vulcan arm; if more than 10% of runs end this way, the evaluation stops, the bug is fixed, and every Vulcan run is repeated. | H3 |
| A7 | Rewrite the plan's status, line 14, lines 25 and 26, and milestone 0 as "done, see decisions". Add audit 2 and audit 3 to the traceability table. | M1 |
| A8 | Mark the performance promise of §2 and §11.3 as a goal with no current test, and mark §5.5 as deferred in the spec. | M2 |
| A9 | Give milestone 5 an entry test: for example, a milestone-4 task that the one-container design cannot meet without a workaround that the evaluation counted. | M3 |
| A10 | Name the person of the milestone-3 exit gate, or use a fresh agent with only the README as the "person", and say which. | M4 |
| A11 | State L1 as a limit in decision 1 and §7.3, and add a conformance case that pins the reported caller. | L1 |

## Resolution

On 2026-10-21 the owner had amendments A1 to A11 applied, in the commit that adds this report:

- Decision 6 has three arms, named roles, hidden rules from a person who has not read the spec,
  six tempting tasks with follow-on tasks, a frozen compiler, and inconclusive results that count
  as fails (A1, A2, A3, A5, A6).
- The development plan has milestone 0.5, time boxes, an up-to-date text, an entry test for
  milestone 5, a named check for milestone 3, and a traceability table with audits 2 and 3 (A4,
  A7, A9, A10).
- The spec marks the performance promise as a goal and §5.5 as deferred (A8), and states the
  limit of L1 in §7.3, pinned by conformance case D12 (A11).

With these amendments, this audit finds no blocker, and the owner accepted the plan. The roles of
decision 6, section 2 are not yet filled. Until they are, B3 is answered on paper only.
