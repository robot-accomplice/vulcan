# Draft: theories of model views (architecture, diagrams, drift, documents)

Status: **draft for the owner**, 2026-11-03. Not accepted. It changes no plan, no gate, and no
accepted theory. [hypotheses-r3.md](hypotheses-r3.md) stays as accepted. The numbers T27 to T30
continue its list, so that an amendment can take them in as they are.

Every prior here is a **prediction by the session that wrote it**, to be measured. No result is
wanted. "A declared model gives no measured gain over plain Go for these views" is an acceptable end.

## 0. Why

The project names four strengths for AI-assisted development: the extraction of the architecture,
diagrams, drift, and generated documents. No round measures them:

- Rounds 1 and 2 measure if an agent keeps the rules of an architecture.
- Round 3's 26 theories are about enforcement and correctness. None is about a view of the system.
- The spec describes `vulcan diagram` (§5.6), `vulcan doc`, `vulcan archdiff` (§9), `vulcan context`,
  and `vulcan audit` (§10). Release 1 defers each of them: `cmd/vulcan/main.go` refuses them with
  exit 2.

So today there is nothing to measure. Each theory below needs a build first.

What exists now, in both arms:

- A reader of the declared model: `internal/project` and `internal/decl` in V, `poc/golint/model` in L.
- `rules lock` and the `depends` kind, which fail a check on a new dependency edge (K0806, VL0806).
  This is drift of dependencies against a locked file. It is not drift between code and a view.
- `rules diff`, which shows what changed in the rules file.

## 1. What is compared

The model is declared in both V and L:

- V: `system.vulcan` and each `component.vulcan`. The wires are the source of `main`.
- L: `arch/arch.go` and `ports.go`. VL0909 holds the wires in `main` to their plain form.
- P: no declared model.

So there are two different questions. The report keeps them apart.

| Comparison | What it tests | Predicted verdict (section 1 of hypotheses-r3) |
|---|---|---|
| V against L | The language: the same view from a model in Vulcan's form or in Go's | Linter-equal for each theory, because L declares the same facts and both checks hold them to the code |
| V and L against P | A declared, checked model against what plain Go gives | Not a language test. It tells if the model is worth its cost, which both L and V pay |

For P there are two sources of a view, and each is a condition of the test:

- **P1, extracted by a tool.** The import graph from `go list`, and the constructor calls in `main`.
- **P2, written by an agent.** A fresh agent reads the code and writes the view. This is the common
  practice of AI-assisted development today, so it is the real alternative.

**Parity.** Each view has one format for every arm and every condition: a graph in JSON (parts,
edges by kind, effects, contracts), with a renderer. The comparison reads the graph, not the
picture. Each arm's generator reads its own model. A test like `TestDefaultsOnBase` checks that V and
L give the same graph on the base applications, with each difference named.

## 2. Ground truth

Each theory compares a view with a graph of the program as it runs. The same tool makes this graph
for every arm, from the Go that each arm builds (V lowers to Go):

- **Static:** the call graph of `go/ssa`, with each edge between parts, and each effect: network,
  database, file, process.
- **Run time:** a trace of each call that crosses a part, under the acceptance tests.
- **Edge kinds:** wire, import, effect.

Where the static graph and the trace disagree, the report gives both and does not merge them. A path
around the model (the escape hatches of the parity audits, the gap list of `vulcan audit`) is ground
truth too: a view that does not show it states less than the program does.

## 3. Verdicts, the same for every comparison

Each measure gives a difference D between two conditions, with a 95% interval, and a margin X that is
set before any data:

| Verdict | Meaning |
|---|---|
| Gain measured | The interval lies above +X |
| Loss measured | The interval lies below −X |
| No measured gain | The interval lies within ±X |
| Unclear | Anything else. One more batch, sized before it is run, then read the same way |

The same X, the same size, and the same replication apply to each direction and to each pair of
conditions. A verdict that only one run shows is not a result.

## 4. The theories

### T27. Extraction of the architecture

- **Claim.** A declared, checked model gives a graph of the architecture that matches the program,
  with no reading of function bodies.
- **Linter test: Linter-equal (prediction).** L declares the same parts, ports, and wires, and VL0909
  holds the wires to `main`. Against P, a tool sees imports and not wires, and an agent's graph can be
  wrong. That is a test of the model, not of the language.
- **Moves.** No rule class. It is a view theory.
- **Cost.** A generator in each arm, about 300 lines each, on the existing model readers. An extractor
  for P1. The ground-truth tool, about 500 lines, shared.
- **Test.** *Measure:* the precision and the recall of edges against ground truth, by edge kind. On
  the base applications, on the final workspaces of past runs in this repository (round 1, and
  `experiments/poc/data/p5-deepseek`), and on the final workspaces of step 8. *Margin:* X = 10
  points. *Size:* set after a first count of edges on the base applications. No agent run is needed.
- **Prior (prediction).** 85% that V and L are within 2 points. 60% that V and L have a recall of
  wires higher than P2's by more than 10 points.

### T28. Diagrams for an agent and a reviewer

- **Claim.** A diagram generated from the model lets an agent or a reviewer answer questions about the
  architecture correctly, with fewer tokens.
- **Linter test: Linter-equal (prediction).** The same graph, in the same format, in both arms.
- **Moves.** No rule class. It is a cost theory, near T25.
- **Cost.** The renderer, small. A set of questions, with answers from ground truth.
- **Test.** *Measure:* correct answers, input tokens, and turns. Each question is answered by a fresh
  agent in three conditions: the code only; the code and a generated diagram; the code and a
  diagram written by an agent (P2). This is done for each arm. A session that does not know the arms
  writes the questions, isolated as the wishlist's author was (plan-r3, amendment 2). *Margin:* 12
  points on accuracy, and a ratio bound of 0.87 on tokens, as T25. *Size:* 40 questions, 3 runs a
  condition. Set the size again after a pilot of 10 questions.
- **Prior (prediction).** 50% that a generated diagram helps either arm by more than the margin. 85%
  that V and L are equal.

### T29. Drift

Drift has two meanings. The test keeps them apart.

**T29a. A view that goes stale.**

- **Claim.** A view generated from the model cannot be stale. A view written by hand goes stale
  under agent work.
- **Linter test: Linter-equal (prediction).** Both arms can generate the view.
- **Test.** *Measure:* each run starts with an `ARCHITECTURE.md` and a diagram written by hand, which
  are true for the base. After the run, count the statements that are now false against ground truth,
  and how many the agent corrected. A generated view is rebuilt in the arms that have one. So this
  measures how much there is to gain: the rate at which a hand-written view goes stale. *Size:* the
  runs of step 8, at no extra run cost.
- **Prior (prediction).** 60% that at least 1 in 5 runs makes a statement of the hand-written view
  false.

**T29b. A change of the architecture that nobody decided.**

- **Claim.** `archdiff` in a review finds a change of the architecture that a review of the diff alone
  misses.
- **Linter test: Linter-equal (prediction).** An archdiff of two graphs is the same tool in either
  arm. In P it reads the graph of P1.
- **Note.** `rules lock` already fails a new dependency, and it fails the check. `archdiff` reports
  each change and fails nothing, so it also covers a change that is wanted.
- **Test.** *Measure:* the detection rate and the false alarms of a fresh reviewer agent, with and
  without the output of `archdiff`. The diffs are seeded: a new edge, a new effect, a removed port, a
  moved wire. Diffs with no change of the architecture are mixed in. An independent session writes
  them. *Margin:* 20 points on detection, 10 on false alarms. *Size:* 20 seeded diffs and 20 controls,
  3 reviews each.
- **Prior (prediction).** 70% that `archdiff` raises detection by more than 20 points in both arms.
  90% that V and L are equal.

### T30. Generated documents

- **Claim.** A document generated from the model and the contracts (the ports, their contracts, the
  mode of each leaf, and each gap, as ADR 0002, rule 2, asks) is correct where a written one is not.
  An agent that works from it invents fewer APIs (R4) and reads less.
- **Linter test: Linter-equal (prediction).** L's contracts are Go expressions, and `go doc` reads its
  comments.
- **Test.** *Measure, correctness:* a sample of statements from each document, checked against the
  code by a judge who does not know which source wrote it. *Measure, use:* ordinary tasks, with and
  without the document, in each arm: input tokens, `undefined:` build failures, and turns, as T24.
  *Margin:* 10 points on correctness. A ratio bound of 0.87 on tokens. *Size:* 100 statements for each
  source. 20 ordinary tasks for use.
- **Prior (prediction).** 60% that generated statements are correct more often than an agent's by
  more than 10 points. 30% that the document changes cost or failures by more than the margin. 85%
  that V and L are equal.

## 5. Theories of harm

Each harm can cancel a gain, so the round measures each with the same care.

| # | Claim | Measure, and what refutes it | Prior (prediction) |
|---|---|---|---|
| NV1 | A view is believed where it states too much: it shows the declared model and not a path around it | On each view, count the claims that the gap list or ground truth contradicts. Refuted if no view has one | 50% that at least one view states too much |
| NV2 | A view in the prompt costs tokens and gives nothing | Tokens with and without the view, on tasks that do not need it. Refuted if the rise is under 5% | 40% |
| NV3 | Nobody reads a generated document | In runs where it is offered, the share of runs that read it. Refuted if more than half do | 50% |

## 6. Order of work

1. **Build, with no agent runs.** The graph format, the generator in V and in L, the extractor for P1,
   the ground-truth tool, and the parity test on the base applications. The same work in each arm.
2. **T27,** on the stored workspaces.
3. **T29b and the correctness part of T30,** with fresh reviewer and judge sessions. These need no
   task runs.
4. **T28, T29a, and the use part of T30.** These need agent runs, so they join step 8, or a small
   separate batch. The owner chooses.

After step 2, the owner reads T27 and decides if steps 3 and 4 go ahead. This document does not set a
stop rule in advance.

## 7. Decisions for the owner

1. Keep this as a draft for review, or take it into hypotheses-r3 by an amendment.
2. The format of a view: a JSON graph with a renderer (my pick), or a standard such as Structurizr DSL.
3. Condition P2, views written by an agent. My recommendation is to keep it, because it is the real
   alternative today.
4. Agent runs for T28, T29a, and T30: inside step 8, or a separate batch.
5. Who writes the questions and the seeded diffs: an isolated session, as amendment 2 did for the
   wishlist.

## 8. What this does not claim

The project lists these four as strengths. Nothing in this repository measures them yet. My main
prediction is that they are strengths of a declared and checked model, which L and V share, and not
of the language. The tests above would show that, or show the opposite, in the same way.
