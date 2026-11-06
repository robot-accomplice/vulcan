# Plan: close each principle of the initial evaluation

Status: **draft 12**: draft 10 accepted by the owner (D1a, 2026-11-05), plus the AE1 to AE6 fixes (draft 11)
and the AF1 to AF6 fixes of the eleventh audit, both before the phase A dry run. AE7 and R17 stay open. The amendment of section 9 is made in
plan-r3, hypotheses-r3 and the views draft. Its rows for D1d and D6 wait on those decisions. Its audit is
[audit-plan-coverage.md](../../experiments/poc/audit-plan-coverage.md): draft 1 had 12 High, 15
Medium and 6 Low findings; draft 2, after two reruns, had 6 High open; draft 3 had 1 High (W1);
draft 4 had 1 High (Y1); draft 5 had no High, and 2 Medium (Z1, Z2) and 3 Low open; draft 6 had no
High, and 2 Medium (AA1, AA2) and 5 Low open; draft 7 had no High or Medium, and 8 Low open; draft 8 had no High, and 1 Medium (AC1) and 7 Low
open; draft 9 had no High or Medium, and 9 Low open. Section 11 maps each open finding to the change
that answers it. The tenth rerun found 0 High, 0 Medium and 7 Low (AE1 to AE7) on draft 10. Draft 11 fixes AE1 to AE6. The eleventh rerun, on draft 11, found 0 High, 0 Medium and 8 Low open (AE7, R17 and the new AF1 to AF6). Draft 12 fixes AF1 to AF6. AE7 is fixed before the C3 seed writers start.

Each column reads one **confidence sequence**, which stays valid when batches are added without
limit (rule 5). There are two kinds of column (rule 6):
- **Agent columns.** Units are tasks or seeds, drawn at random from a **sealed pool** that many writers write
  before the column's first batch. The bound is about that pool. A writer-level interval beside each
  verdict shows the step from the pool to writers in general.
- **Fixed-outcome columns** (seeds and rules, no agents). The unit is a fresh writer, and each
  writer's seeds are evaluated once. The bound is about writers in general.

Both models sit inside each agent unit (rule 7). So the method's assumption holds by design. The
cost, simulated in section 6, is much larger than draft 2 said.

## 0. Why

The owner's rule (2026-11-03): the initial evaluation is complete only when no principle in the
findings report ([findings-report.html](../../experiments/poc/findings-report.html), "Where we are,
by principle") has a "Not built" or a "Not tested" entry. Analysis alone does not close an entry. A
count that stands in for the claim does not close it.

This plan gives each principle, and each part of a principle that needs its own verdict:
- what must be built, in each arm;
- the test, its one primary measure, its margin, and its batch;
- what closes each column.

It does not predict a result. Any verdict closes a column, "Loss measured" and "Within margin"
included. **Only a verdict closes a column (owner, 2026-11-03).** An interval that stays unclear, a
test that the model can't be tempted into, and a gate that stops a test do not close a column.

## 1. Rules for every row

1. **Arms.**
   - P is plain Go, as a Go team works today (rule 2).
   - L is Vulcan's model as a Go linter, library and primer.
   - V is Vulcan.
2. **The same common tools in every arm.**
   - Each arm gets `go vet`, `staticcheck`, `errcheck`, `exhaustive`, `dupl` and `gopls` (in the
     check of row 15, the command `gopls check`), pinned at their latest stable versions. L and V add their own checks on top.
   - P also gets the tool a Go team uses for each job that the model does, where one exists: an
     architecture linter (row 2), a schema compatibility check (row 10), and a conventions file
     (row 12).
   - Each arm reaches each tool by the same path, a command line, and each prompt names the tools in
     the same words.
   - **On V,** the Go tools run on the lowered Go, with `//line` positions back to the `.vulcan`
     files. **One filter rule in every arm:** a finding in a file whose first line is the standard
     `// Code generated ... DO NOT EDIT.` header is dropped. This drops V's wrappers and `main`, the
     output of L's generator, and any generated file in P. The record gives the count dropped in each
     arm. No other filter exists.
3. **Equal build.** A feature built for V gets its best equal form in L, and in P where a row says
   so. A fresh session builds each form, from one written brief and with the same budget. The
   builder of one form does not build another. Where L shares code with V (`internal/rules`), the
   row says so, because then V against L can show only what the language forms add.
4. **Parity audit.** An independent session audits each new build before its test. Gate: no open
   High finding, as plan-r3 step 3.
5. **Verdicts: one confidence sequence per column.**
   - **The quantity.** Each column has one primary measure: a difference D between two conditions,
     read per unit (rule 8), and a margin X. Both are in section 2, before any data.
   - **The sign.** D is signed so that a positive value favors the first-named condition: V, or the
     condition with the feature. For a rate of a bad outcome (rule breaks, invented symbols,
     `undefined:` failures, races that pass, bypasses, harms) and for a cost (tokens, turns, time),
     D is the other condition's value minus the first-named condition's value.
   - **The estimand** depends on the kind of column (rule 6), with each replication level of rule 7
     at equal weight inside the unit, and after the clipping stated in section 2:
     - **Agent column:** the mean of D over the column's sealed pool. The bound of this rule is about
       the pool. Its writers are a sample of possible writers, and the bound does not cover the step
       from the pool to all writers (section 12). Each verdict says so (below).
     - **Fixed-outcome column:** the mean, over writers who write from the column's brief, of a
       writer's mean D. The bound is about those writers in general.
   - **The method.** A 95% confidence sequence for the estimand: the hedged-capital betting method
     of Waudby-Smith and Ramdas (2023, JRSS B, theorem 3), with the plug-in bets of that paper,
     θ = 1/2 and a truncation of 1/2. Each unit's D is mapped to [0, 1] from its stated range.
   - **Its assumption.** Each new unit has the same conditional mean, given the units before it. The
     design keeps this: agent units are independent draws from one sealed pool, with replacement,
     and fixed-outcome units are fresh writers from one brief (rule 6); every replication level is
     inside each unit (rule 7); and nothing about a batch changes with the data (rule 6). A writer's
     effect is then part of the estimand, not a shift shared by a batch (Y1). Under it, the sequence
     covers the estimand at every time at once, with a probability of at least 95%. So a reader may
     look after any unit, add batches without limit, and stop at any time, and the chance that any
     verdict of the column is false stays at most 5%. **One alpha covers the whole series of a
     column.** The interval that a reader uses is the running intersection.
   - **The outcomes** (views draft, section 3), read at the end of each batch:

     | Outcome | Condition |
     |---|---|
     | Gain measured | The interval lies above +X |
     | Loss measured | The interval lies below −X |
     | Within margin | The interval lies within ±X. The record gives the interval, and adds "(excludes 0)" when it does |
     | Open | Anything else |

     The column closes at the first batch end where one of the first three holds. Each outcome has
     the same burden: the same interval, the same X, in each direction. "Within margin" replaces the
     draft 3 name "No measured gain", because a loss or a harm smaller than X also falls there (W4).
   - **What a verdict states** (Z1, AA2, AA5). On an agent column with a pool, the record writes the
     verdict as "for this pool (N writers)", for example "Gain measured, for this pool (20
     writers)". Beside it, the record gives the **writer-level interval**: the mean D of each writer
     over its drawn units, and a t-interval over those writer means at a nominal 95%. It is read
     once, when the column closes, and no verdict reads it. Where it does not meet the verdict's own
     condition (above +X for a Gain, below −X for a Loss, within ±X for Within margin), the record
     says "the step to other writers is not shown".
     - **Its limit.** The close is a stopping time of the sequence. So, given a verdict, the
       interval's coverage is lower than 95%, and the absence of the note is not evidence that the
       verdict reaches other writers. Section 6 gives two rates at which a verdict wrong for writers
       in general carries no note: over all simulations (about 5% to 8%), and given a verdict
       (about 15% to 31%). A reader who holds a verdict meets the second (AC7).
     - **On rate columns near 0** (every agent column with a pool, for example rows 3, 6a, 6b, 7, 13b,
       16, N3, N4, N5, N6), most writer means can be 0, and the interval is then narrow or of zero
       width. The record gives the count of writers with a nonzero mean beside it, and says "not
       informative" when fewer than half of the pool's writers (fewer than 10 of 20) have one.
     - **Columns without writers.** Rows 15 and 19 read a fixed set of changes or request paths
       (rule 6), and row 18 reads fresh attacker sessions. Their verdicts say "for this set of
       changes", "for this set of paths" or "for these attacker prompts", with no writer-level
       interval.
     - On a fixed-outcome column, the verdict is about writers in general, and it carries no "for
       this pool".
   - **An empty intersection** is not a verdict. The column stays open, and the record names it as a
     failure of the method's assumption. Its campaign continues unchanged for the other columns. Any
     change to this column makes a new column with a new estimand, by an amendment before its data
     (Y4).
   - **Accepted rules.** Where hypotheses-r3 or the views draft has an accepted rule for the column,
     its threshold sets X, and its own condition is reported beside the verdict as the "hr3
     reading". No verdict reads the hr3 reading. Section 9 lists every such change.
   - **Measures with no natural bound** (token ratios, times, counts): the unit's value is clipped to
     a range stated in section 2 before any data. The record gives the number of clipped units in
     each arm.
   - **Runs that do not reach the end point** (W6). For a cost to an end point (tokens to a correct
     program, tokens to the first passing check, turns from a rejection to a pass), a run that fails
     or hits its budget counts its total cost at the stop. So every run has a value, and a task has a
     value in every condition. The task's value is the log of its mean cost over runs. The record
     gives the count of runs that did not reach the end point, per arm.
6. **The next batch.** The same rule for every column, written now:
   - **Kinds of column.** An **agent column** reads runs whose outcome varies from run to run. A
     **fixed-outcome column** reads a check or a tool on seeds or rules, with no agent: a check
     rejects a seeded case or it does not, so a unit evaluated again gives the same D (Z2).
     - **Agent columns with a task pool:** every column of C1, C2, C2v, C4, C5, C7 and
       calibration (rows 6a, 6b).
     - **Agent columns with a seed pool** (AA1): the seeded columns that agents or reviewers read,
       C3a (row 3, review part), C3b (row 4), C3c (row 5d) and N5. A seed's outcome varies with the
       reviewer or the run, so these use the pool rule, not the writer unit.
     - **Agent columns without writers** (AA5): rows 15 and 19 (timing runs on a fixed set) and row
       18 (attacker sessions).
     - **Fixed-outcome columns:** rows 2, 5a, 5b, 5c, 9, 10a, 10b, 11a, 11b and 11c. Row 5c's
       verifier runs with a resource limit (the Z3 solver's `rlimit`), not a time limit, so its
       outcome is fixed. Row 11c runs `go test -race` in every arm, and its result depends on
       scheduling. It runs once a seed, by one fixed procedure: `-count=20`, `GOMAXPROCS=4`, on
       machine 1 of rule 7, with no other load (AA5). The writer unit below is valid whether or not a
       seed's outcome is fixed, because each seed is evaluated once.
   - **Agent columns: a sealed pool, written first** (Y1). Before a campaign's first batch, fresh
     writers write the whole pool from one brief. A task pool is 20 writers of 20 tasks each (400
     tasks), or 40 writers of 10 tasks each by D15's rule. A seed pool is 20 writers of 10 seeds each
     (200 seeds). For C3a, C3b and C3c, a seed writer draws one task at random, by a seed fixed
     before the writers write: C3a from C2's task pool, C3b and C3c from C1's task pool (AD3). A
     fresh session makes one run of that task, under the main model, in each arm, for the seed
     writer only. This run is not a run of any campaign, so the diff always exists, and no draw
     depends on a campaign's progress. The writer takes the diff of that run in each arm and plants
     the same seed in each (AB3, AC4). The C3 seed pools are written after the task pools of C1 and
     C2 are sealed (phase C). N5's seed pool is idiom seeds in
     Go-looking text, written by the 20 writers. Rank 3's existing idiom seeds are history, as T1 is
     (AB4). A fresh session checks the pool against its brief. The check reads the brief and the
     tasks or seeds only, and runs nothing in any arm. For C3 it also reads the seeds in each arm's
     diff, to confirm that they are equal. The pool is then sealed. The builder sees its
     size only.
   - **Agent columns: each batch draws from the pool** at random, with replacement, by a seed fixed
     before the first batch. A unit drawn twice is run again with fresh runs or fresh reviewer
     sessions, and no state carries over from an earlier run: no cache, no kept workspace and no
     fixed sampling seed (Z2). So the units are independent draws, and the pool never runs out.
   - **Agent columns without writers** (AA5). Rows 15 and 19: one fresh session writes a fixed set of
     40 one-file changes (row 15) or names the request paths of the base apps (row 19), before any
     timing. Each batch draws 20 of them at random, with replacement, and runs each draw again on
     both machines. Row 15 runs from a **warm cache in a fixed state, set the same way in every arm**
     (AC1). The **check** of row 15 is one command an arm (AD8). P runs `go vet`, `staticcheck`,
     `errcheck`, `exhaustive`, `dupl`, `gopls check` and its architecture linter. L runs the same
     common tools and L's analyzers. V runs its check command, which runs the same common tools on
     the lowered Go and V's own checks. The warnings and fixes of row 11 are off in every arm, so the
     check is Base (section 5, AE3). The record writes the three commands before phase A'.
     - **The scope of a change** (AE4, AF2). In every arm, each tool reads the changed Go file and every
       Go file that the arm's own build rewrites because of the change. In P and L, this is the
       changed file. In V, it is the lowered Go of the changed `.vulcan` file, and any other
       lowered file that the lowering rewrites. A tool that takes packages reads the packages of
       those files.
     - **Whole-program tools** (AF2). A tool that checks the whole module or program runs on the
       whole module in every arm. These are P's architecture linter, and any analyzer of L and any
       check of V that reads files beyond the changed file. No arm limits such a tool to the
       changed files. The commands that the record writes before the dry run name, for each tool in
       each arm, the unit that it reads: file, package, module or whole program.
     - **The set of 40 changes holds no change of a signature or an export** (AF2), so no change
       breaks a dependent file. The fresh session that writes the set follows this rule, and checks
       that each change builds in every arm. The row then measures an incremental check after a
       change inside one file.
     - **The count of files and packages** (AF2). The record counts what each tool reads by one
       method in every arm: a trace of file opens, in a separate untimed run, never in a timed run.
       A trace run does steps 1 to 4 below for one change in one arm, and then runs the check under
       the trace in place of step 5. The record makes one trace run for each change of the set in
       each arm, once, in phase A' before batch 1 (120 untimed runs). The dry run tests the method
       on its row 15 draws (60 untimed runs). A tool for which the method gives no count is a fault
       of the procedure.
     - **The steps of one row 15 run, in this order** (AE5, AF3). (1) The record restores the caches of
       this **closed list** to their state just after one full check of the unchanged base app
       (AD1). In every arm: the Go build cache, the Go module cache, the `staticcheck` cache, the
       `gopls` cache and the build outputs in the working tree. P adds the cache of its
       architecture linter. L adds the cache of its analyzers. V adds its lowering cache and its
       lowered Go files. (2) The record drops the file cache of the system, in every arm. The drop
       follows the restore, so the copy of the caches does not warm the file cache of any arm. The
       record names one fixed method of the drop for each machine, before any timing. A method is
       part of its machine (rule 7). (3) The record reads the program files back into the file
       cache. The drop also removes the files of the tools themselves: the Go toolchain,
       `staticcheck`, `gopls`, `errcheck`, `exhaustive`, `dupl`, and P's architecture linter, L's
       analyzers or V's binary. A developer's second check finds them in the file cache. So the
       record reads once, and runs no tool, every file of a closed list of program files, which the
       commands name for each arm: the executables, and the toolchain tree that the check loads.
       The read is the same in every arm and writes nothing. (4) The record applies the one-file
       change. (5) The record times the check. Step 4 follows steps 2 and 3, so the changed file is
       in the file cache in every arm. Row 19 has no restore and no drop: its warm-up follows
       (below).
     - Any cache that an arm's tool writes outside the closed list is a parity-audit finding (rule
       4): the audit lists the paths that each check writes during one full check, and the build is
       not accepted until each path is in the list. The audit also lists the program files that
       each check loads, and a file outside the program list is a finding too (AF3).
     - So row 15 measures an incremental check, no run depends on an earlier one, and every cache of
       the list starts in the same kind of state in each arm. The record lists the caches restored
       in each arm (AC2), and keeps each run's tool output, so that a warm arm shows.
     - In rows 15 and 19, the arms of a draw run in a random order, fixed by a seed before the first
       batch, and no other load runs on the machine (AD1).
     - **The warm-up of row 19** (AD2, AE6). Row 19 runs in steady state in every arm: a run starts
       a fresh process and sends warm-up requests in blocks of 50. From block 4, the warm-up ends
       at the first block where the median time of the last two blocks (100 requests) is within 5%
       of the median time of the two blocks before them. A path that has not settled after 12
       blocks (600 requests) stops there, and the record counts these paths per arm. Then the run
       times the next 100 requests of its path. The run's time is their median (AC2). For each run,
       the record also gives the median of the timed 100 requests divided by the median of the 100
       requests before them, as the residual drift. Beside the verdict, the record reports by arm
       the share of runs whose residual drift differs from 1 by more than 2%, and the share where
       it differs by more than 10%, the margin of row 19. The 2% is one fifth of the margin: a
       drift of that size in one arm moves its time by one fifth of the margin. Both shares are
       limits. No verdict reads them, and no run leaves the verdict. They are sealed with the
       column and reported when it closes (AF5).
     - Row 18: each pair is four fresh attacker sessions from one fixed prompt per arm, so the pairs
       are independent with one mean. No writer-level interval applies (rule 5).
   - **The dry run of phase A** (AD8, AE1, AE2). Phase A has no new build, so arm P has no linter
     and no assertions yet. The dry run runs one batch of row 15 and one batch of row 19, by the
     steps above, on the base apps, with what each arm ships today. Row 15: P runs the common tools
     only; L runs the common tools and L's analyzers as they are; V runs its check command as it
     is. Row 19: P runs the base code with no assertions; V runs the base code. The record writes
     these commands before the dry run. The dry run sizes the machine time and tests the
     procedure. Its figure is a **lower bound**, because the check of P in phase A' is longer
     (its linter) and the code of P in row 19 differs (its assertions).
     - **Its outputs are sealed** (AE2, AF1) like the running interval of an open column. This
       covers every output that carries the direction of an arm: the times, the count of unsettled
       paths per arm, the residual drift, the file counts and the kept tool output. The record
       gives the total machine time of row 15 and the total of row 19, each over all arms and both
       machines (AF6), and never a figure for an arm. No column reads the dry run. A fault of the
       procedure (a command that fails, a cache outside the closed list, a tool with no file
       count) is reported as the fault only, with no figure of any arm. The person who fixes a
       fault sees the fault, the two totals and the count of faults.
     - **No rule changes after the dry run** (AF1), except by an amendment (rule 13) with its
       reason, made before the batch 1 data. This covers the band, the cap, the set of 40 changes,
       the drop method, the lists of caches and program files, and the scope. An amendment may cite
       a fault, the totals or the count of faults. It may not cite a figure of an arm.
     - The memo session of rule 6 reads the two totals as lower bounds before batch 1 of the
       timing campaign. It multiplies each by the batch counts of its own row (section 6): 4 to 6
       for row 15 and 9 to 20 for row 19. It reads the total of batch 1 of phase A' before batch 2.
   - **Fixed-outcome columns: the unit is a writer** (Z2). Fresh writers write from the column's
     brief: 10 seeds a writer for a seed column, and one rule set of about 28 rules a writer for
     row 2. Each seed or rule is evaluated once in each condition. The unit's D is the writer's mean
     D over its seeds. The first read is at 20 writers (rules: at 10); then a batch is 10 fresh
     writers. No writer is drawn twice. The columns of one row (against L, against P) read the same
     writers.
   - **History, not units** (AA3). T1 (plan-r3) and the 29 existing clauses of hr3 rank 1 step 1
     (row 5c) were written before this rule. T1 was written under an earlier model and with no corpus draw.
     They are reported as history beside
     rows 2 and 5c, as T2 is, and are not units of the sequence. Row 2's rule writers get round 2's
     brief and T1's labeler procedure, so that the history and the sequence measure the same thing.
   - **The writer check** (AA6). A fresh session checks each writer's seeds or rules against the
     brief, and reads nothing that any arm produced. A writer that fails the check is replaced by a
     fresh writer, and the record gives the count replaced per column. The estimand is then writers
     whose work passes the check.
   - **The design does not change.** A next batch uses the same primary measure, the same margin,
     the same conditions and the same pool, or for a fixed-outcome column the same brief. No column
     changes its measure after data.
   - **Every batch pools.** The sequence reads all batches of the column, from the first. No batch is
     left out.
   - **Funding is per campaign, all or none.** The next batch of a campaign runs every open column of
     that campaign together. The owner decides on a campaign's next batch from a memo that a fresh
     session writes. That session sees which columns are open and the cost of a batch, and not the
     direction or the value of any interval. A campaign that stops leaves all its open columns open.
     The same rule funds the next 10 writers of a fixed-outcome row, for all its open columns
     together. Rows 15 and 19 are the timing campaign, and row 18 is the attacker campaign
     (section 5, AD7). The same rule funds each.
   - **A closed column's conditions leave the campaign** when no open column of that campaign reads
     them. This rule is fixed now, so it is not a choice after data.
   - **Conditions are fixed before a campaign's first batch** (D13), from calibration's pooled values
     only. No condition is cut after a batch (W7).
   - **The running interval of an open column is sealed,** and so are its secondary sequences of
     rule 7, which show its direction too (Y5). The outputs of the dry run of phase A are sealed in
     the same way (AE2, AF1). The analysis session that reads them reports only
     "open" or a verdict, to the owner, to the memo session and to the findings report, until the
     column closes or its campaign stops (W7).
   - So no choice made after data can select a column by its direction, and none can make a verdict
     false beyond the 5% of rule 5.
7. **Replication is inside each unit** (W1). Each column names its level in section 2:
   - **Model:** every task runs in every condition under both the main model and the second model
     (D5). The unit's D is the mean of its two model differences.
   - **Corpus:** writers are split into two halves by a fixed random draw: an agent pool's writers
     before the pool is sealed, and a fixed-outcome column's writers by a draw of each writer's half
     before it writes. The secondary sequences read each half alone. The verdict reads all units.
   - **Machine:** every change or request runs on both machines, and the unit's D is the mean of the
     two. The record names the two machines, machine 1 and machine 2, before any timing (AB6). Each
      timing column (rows 15 and 19) uses these two machines for every run of both arms of a pair,
      with their hardware and system fixed for the whole column. A change of machine makes a new
      column, by an amendment before its data (AC2).
   - So every unit has the same mean, the estimand of rule 5, whatever the difference between the
     levels or between writers.
   - The verdict reads the one sequence. The record also gives the sequence of each model, corpus or
     machine alone, as secondary. If the two secondary sequences reach different verdicts, or one
     closes and the other does not, the record says "differs by model" (or corpus, or machine) next
     to the verdict, as a limit.
   - No disagreement is resolved by choosing one side, and no batch is repeated after a result.
8. **Unit of analysis.** Campaigns are analyzed by task: the unit is a task, and its D is the
   difference between its mean over runs in the two conditions, signed by rule 5, with both models
   inside it (rule 7). Runs of one task are not
   independent, so the unit absorbs their correlation. Seeded tests that agents or reviewers read (C3a,
   C3b, C3c, N5) use the seed as the unit, drawn from a seed pool, with both models inside it (rule 6,
   AA1). Seeded tests that a check reads, and label tests, use the writer as the unit: the mean over
   its seeds or rules, each paired across arms (Z2).
9. **One primary measure per column.** The verdict reads it alone. Other measures are reported as
   secondary, and no verdict reads them.
10. **Features on, per campaign.** Section 5 lists which features are on in each condition of each
    campaign. A column closes only on a comparison where the feature under test is the only
    difference, or where section 5 names the bundle that the column measures.
11. **Harms.** Each harm of hypotheses-r3 section 4 (N1 to N6) and of the views draft (NV1 to NV3) is
    a column of its own, read by rule 5 (section 4). A gain on a row whose campaign has a "Loss
    measured" harm is reported as **"Gain with a measured harm"**, with both results side by side.
    The owner reads both. No rule nets one against the other.
12. **Blindness.** Fresh, isolated sessions write rules, tasks, seeds, questions and labels, and
    judge matches. The material is sealed, as in rounds 1 to 3. The builder sees ids, counts and
    pass/fail only. Where a judge or reviewer can see the arm (for example a `.vulcan` file), the
    record names this as a limit, and every arm gets one fixed prompt.
13. **Phase A' changes.** A phase A' result may drop a later condition only if its own column is
    already closed. Any other change is an amendment, with its reason, before the data it affects.
14. **Pilots and calibration.** No pilot run, pilot task or calibration run counts in a main result,
    except rows 6a and 6b, whose test is the calibration workspaces, and N5, whose test is its own
    seed pool, which runs in the calibration campaign (AC3). Besides D15's writer count
    (AA4), calibration may set only the
    pooled variance and the correlation of runs within a task, for the cost estimate. It sets no
    batch and no margin.
15. **Leans.** Each design input that leans toward an arm is named in the record, as plan-r3
    section 0 does. Section 8 lists the known ones.
16. **Many columns.** Rule 5 holds the error at 5% per column, not over the plan. With about 45
    columns, about 2 false verdicts are possible by chance alone. The report says this beside the
    table.

## 2. The principles, their parts, and their tests

"Measure" names the primary measure and the unit. X is the margin of rule 5. "hr3 reading" is the
accepted rule reported beside the verdict. "Repl." is the level of rule 7. Points are percentage
points of the unit's rate. Every D is signed by rule 5: a positive value favors V, or the condition
with the feature. Where a row writes "A minus B" for a cost or a bad outcome, B is that condition.
On a fixed-outcome column (rule 6), "per seed" and "per rule" give each seed's or rule's D; the
unit is the writer, and its D is the mean over its seeds or rules (rule 8).

### Rows 1 to 4: the language, the architecture, and review

| Row | Column | Build | Test and primary measure | X | hr3 reading | Repl. |
|---|---|---|---|---|---|---|
| 1. Bodies are plain Go (N1) | Against L | None | C1: V against L, base. Task success | 5 points | hr3 N1 (5 points on success, 5% on tokens) | Model |
| | Against P | None | C1: V against P. Task success. Round 1 stays as history and is not a batch | 12 points | None | Model |
| 2. The compiler enforces the architecture | Against L | None | Fresh rule writers (rule 6); T1 is history (AA3). Per rule: enforced by V (1 or 0) minus enforced by L | 4 rules in 28 (14 points) | hr3 "U below 4" | Corpus |
| | Against P | P's architecture linter (`depguard` or `go-arch-lint`, latest stable) with **a default configuration**, written by a fresh session from published Go practice, before any blind rule is seen, and frozen with `defaults v1` (R3, option A) | The same rule writers. Per rule: enforced by V minus enforced by P with its linter. P without its linter is secondary | 14 points | None | Corpus |
| 3. Architecture rules leave review | Against L, against P | Tempting tasks (section 3) | C2: per task, the rate of runs with a rule break in final code, judged by one procedure on the Go that each arm builds (V's lowered Go). Secondary: breaks fixed during the run, and P−linter | 12 points | None | Model |
| | Review part | Seeded architecture breaks in diffs | C3a: fresh reviewer sessions on the seed writers' diffs of C2 tasks (AD3), some with seeded breaks, with and without a passed check, in each arm. Per seed: found | 12 points | None | Model |
| 4. No gain on reading or logic defects (premise H3) | Against L, against P | Seeded logic defects in Go bodies only, by a fresh session, the same seed planted in each arm's own diff (rule 6, AC4) | C3b: reviewer sessions on the seed writers' diffs of C1 tasks (AD3). Per seed: found. Secondary: tokens read per defect | 12 points. The premise predicts Within margin | None | Model |

**Row 2 and T2.** T2 is in sample for the dependency check, which was designed after the round 2
diagnostic (plan-r3 section 0). So T2 is not a batch of row 2. It is reported as history, without the
dependency check, as plan-r3 section 5 says (R12). The sequence reads fresh rule writers only. T1 is
history too, beside the verdict (AA3).

### Rows 5 to 9: values, contracts and tests

| Row | Column | Build | Test and primary measure | X | hr3 reading | Repl. |
|---|---|---|---|---|---|---|
| 5a. Contracts at run and test time | Against L, against P | P's best equal: assertions in the provider, from the same neutral statement of each contract that V and L had, by a fresh builder | A1 (no agents): per seed, caught at check, test or run time | 12 points | None | Corpus |
| 5b. Exact arithmetic in contracts | Against L, against P | Spec C08 in V, or the claim narrowed by D8. L: a contract library with big integers. P: assertions in plain Go | A1b: seeded contracts whose Go arithmetic overflows. Per seed: caught | 10 points (new rule, section 9) | None | Corpus |
| 5c. Static proof (T4, T5) | Against L | hr3 rank 1: the verifier, trap mode, L's checked arithmetic | Rank 1 step 1: per clause, proved in V (same fact) minus proved in L. Fresh clause writers; the 29 existing clauses are history (AA3) | 10 points | Rank 1 table; T4's 30% share is reported per arm | Corpus |
| | Against P | See D12 | — | — | — | — |
| 5d. Contracts that reviewers can audit | Against L, against P | Seeded contracts that do not match a written intent. P's equal is its assertions and doc comments | C3c: per seed, mismatch found | 12 points | None | Model |
| 6a. Safer body meanings: faults shown (T8) | Against P | hr3 rank 3: the trap rewriter | Calibration workspaces of arm P, plain and rewritten. Per workspace: 1 if the rewriter shows a real fault, else 0 (a fresh session classifies). Batch: 36 workspaces | 5 points | Rank 3 (10 of 108) | Model |
| | Against L | L: a checked-math library and an analyzer that requires it, by a fresh builder | Per calibration task: V rewritten's rate of real faults shown minus L's with its library | 5 points | Rank 3, per arm | Model |
| 6b. Safer body meanings: false alarms (a harm, W5) | Against P | As 6a | Per workspace: 0 minus 1 if the rewriter raises a false alarm (a trap on a correct program), else 0 | 2 points (new rule) | Rank 3 (1 of 108) | Model |
| | Against L | As 6a | Per calibration task: L's false-alarm rate minus V rewritten's | 2 points (new rule) | Rank 3, per arm | Model |
| 7. Illegal states can't be built (T7) | Against L | hr3 rank 4: `enum`, `match`, `nozero` in V; L's best equal by a fresh builder | C2v: per task, the rate of runs that break a value rule in final code | 12 points | Rank 4 (form ratio 1.25, bypasses within 1), with form and holes as secondary | Model |
| | Against P | P has `exhaustive` (rule 2) | C2v, as above. A3 on seeded cases is secondary | 12 points | None | Model |
| 8. Less work for a correct program (T21) | Against L | hr3 rank 2: the L generator | C1: per task, log of mean input tokens to a correct program (a run that does not reach one counts its total tokens, rule 5), L+gen minus V. Clipped to ±ln 4 | ln 1.05 | Rank 2 (bounds 1.05 and 0.95) | Model |
| | Against P | P's best equal holds the same guarantees: the contracts as assertions (5a) and the architecture linter (2) | C1: per task, log of mean input tokens, as above, P+ minus V. Clipped to ±ln 4 | ln 1.15 | None | Model |
| 9. Tests with fakes bound to ports | Against L, against P | P's best equal: an interface, the contract as assertions in the real provider, and a shared test suite that each fake must pass. A fresh labeler defines a test defect | A4 (no agents): per seeded fake that drifts from the contract, caught | 10 points (new rule, section 9) | None | Corpus |

Rows 6a and 6b are one-sided by nature: a rate of faults shown cannot read a Loss, and a rate of
false alarms cannot read a Gain. Rule 5's equal burden holds in the direction each can move (Y5).

### Rows 10 to 15: processes, intelligence, tools and speed

| Row | Column | Build | Test and primary measure | X | hr3 reading | Repl. |
|---|---|---|---|---|---|---|
| 10a. Compatibility across processes (T16) | Against L, against P | Section 7: serialized messages, producer and consumer as separate binaries. L: a Go library and a lock check, stated as shared or not with V's code. P: a schema with `buf breaking` | A5: per seeded break that the Go compiler does not catch (renamed field, changed meaning, removed optional field), old consumer against new producer: rejected. C4 is secondary | 5 points | T16 (more than 1 of 20) | Corpus |
| 10b. Dead letters | Against L, against P | Built in the subset, or the claim narrowed by D9 | A5b: per seeded poison message: delivered to a dead letter, not lost, not looping | 10 points | None | Corpus |
| 11a. Duplication warnings | Against L, against P | V and L warnings, by spec §9. P: `dupl` | A6a: per duplicate seeded by a fresh writer from published clone types 1 to 3, without the spec: warned. Secondary: duplicates in final code in C2, warnings on against off | 10 points | None | Corpus |
| 11b. Masked-error warnings | Against L, against P | V and L warnings. All arms: `errcheck` | A6b: per seeded masked error: warned. Secondary: masked errors in final code in C2, warnings on against off | 10 points | None | Corpus |
| 11c. Race freedom (T20) | Against L | hr3 T20 build | Per seeded race: rejected at check time | 10 points | T20 (bypasses within 1, false rejections) | Corpus |
| | Against P | None. `go test -race` runs in every arm | Per seeded race: it reaches a passing change in the arm, by any tool, check or test (R7, an outcome in every arm) | 10 points | None | Corpus |
| 11d. Machine-applicable fixes (T22) | V against V−fixes | Fixes in V and L. P: the fixes of `staticcheck` and `gopls` | C2: per task, log of mean turns from a rejection to a pass (a run with no pass counts its turns to the stop), V−fixes minus V. Clipped to ±ln 4. L against L−fixes is secondary | ln 1.25 | T22 (a fall of 20%, arms within 12 points) | Model |
| 12. Tools teach the model | V against V−hints | Hints and `explain` in V and L. "−hints": the checker reports the rule and the place only; no fix text, no `explain`. **The primer stays in every condition** (R4) | C1: per task, success, V against V−hints. L against L−hints, and the difference of differences, are secondary | 12 points | None | Model |
| 13a. Small context (T25) | Against L, against P | `vulcan context` in V; the same pack from a `gopls`-based tool in L, which makes V against L a test of the code form only; P: `gopls` and search | C5: per task, log of mean input tokens to the first passing check after the first edit (a run with no pass counts its total), each arm with its tool, the other arm minus V. Clipped to ±ln 4. Without the tool is secondary | ln 1.15 | T25 (0.87 either way) | Model |
| 13b. Exact symbol index (R4) | Against L, against P | `vulcan query` in V; the same in L; P: `gopls` | C5: per task, the rate of runs with an invented symbol, counted on the Go that each arm builds, by one script | 12 points | None | Model |
| 14. The architecture can be seen | As the views draft | The views draft: `diagram`, `doc`, `archdiff`, with equal forms in L and P | C6: T27 to T30 and NV1 to NV3, each read by rule 5 with the views draft's margins. T29a needs a measure and a margin in the views amendment before C6 starts (R17). Row 14 closes only when every column of the views draft has a verdict | Views draft | Views draft | As the views draft |
| 15. Fast verification | Against L, against P | None | A7: per one-file change on the base apps, with the check of rule 6, from its warm cache, log of check time, P minus V. Per-leaf test time is secondary. Clipped to ±ln 4 | ln 1.2 | None | Machine |

### Rows 16 to 19: claims that draft 1 left out

| Row | Column | Build | Test and primary measure | X | hr3 reading | Repl. |
|---|---|---|---|---|---|---|
| 16. Type information on unfinished programs (R5, T24) | Against L, against P | The hole command in V; the `go/types` equal in L; P: `gopls` | C1: per task, the rate of runs with an `undefined:` build failure, each arm with its tool (V+hole, L+hole, P). The effect within each arm is secondary | 12 points | T24 (V's benefit over L's) | Model |
| 17. One way to write a thing (spec §2, T23) | Against L, against P | None | C1: per task, first-attempt pass rate: the first build, check and test run passes. This exists in every arm (R7). Rejections before the first pass are secondary, V and L only | 12 points | T23 | Model |
| 18. The checks list every path around them (gap list, `vulcan audit`) | Against L | `vulcan audit` in V; the same list in L | Holes stage of hr3 section 5. Per pair of attacker sessions (four sessions: one an arm under each model, paired by their place in a schedule fixed before the batch): working bypasses that the gap list does not name, each arm's count clipped to 0 to 5, then L minus V (W3) | 1 bypass | None | Model |
| | Against P | See D12 | — | — | — | — |
| 19. Performance like Go (spec §2, §11.3) | Against P | None | A8: per run of one request path of the base apps, in steady state (rule 6), log of time, P minus V (positive: V is faster), where P is the same code with assertions in place of contracts. Micro-benchmarks of port calls and checks are secondary. Clipped to ±ln 4 | ln 1.1 | None | Machine |

## 3. Tasks that tempt a rule break

Round 1 could not test rows 2 and 3, because no arm broke a rule. C2 needs tasks where a break is the
short path.

- **C2's pool** (rule 6, Z3): 20 fresh writers write C2's pool of 400 tasks from one brief. It is
  written in phase B, before calibration, so D15's rule does not apply to it, nor to C2v's pool or
  to a pool that replaces either after a failed gate (AA4). The brief is made
  from the blind rule set of plan-r3 T1. Each task names a goal, not a rule. A fresh session checks
  the pool against the brief, and the pool is sealed.
- **Pilot:** 10 tasks drawn at random from the pool, 2 runs an arm, in all three arms. The gate reads
  P's break rate with prose rules only, **P without its linter** (R9): at least one third of runs.
  Each arm's rate is reported. Selection on P's breaks leans toward V and L against P, and the record
  says so. Pilot runs do not count in C2, and the 10 pilot tasks leave the pool before its first
  batch.
- If the gate fails, the pool is not used. 20 fresh writers write a second pool from a revised
  brief. The brief's writer sees only P's rate per pilot task. The second pool is a new estimand,
  checked and sealed in the same way, and it gets its own pilot of 10 tasks.
- If the second pilot also fails, rows 2 and 3 (task columns) stay open, and the record gives each
  arm's pilot rate. The next step, written now (D1c): a third pool by 20 fresh writers, with tasks of
  a larger change, and the second model of D5. The second model is the replication model, named
  before any pilot and used in every arm (S5). It is not chosen for its breaks.
- **If the plan-r3 gate fails** (fewer than 8 rules in S or 4 in N on T1), plan-r3 stops its own
  tasks. A gate does not close a column, so rows 2 and 3 task columns stay open. If D1d is yes, C2
  then runs on **the union of the rules that any arm's check enforces**, P's linter included (S4).
  The record gives the result per rule and per arm, and the share of T1's rules that the set
  covers.
- **What each arm may edit:** the files of `PROTECTED` in plan-r3 are the same for every arm. V's
  agent must edit `system.vulcan` for ordinary work, and L's manifest is protected only where V's
  is too. The record states the list for each arm.
- **C2v** (row 7) has its own pool of 400 tasks by 20 fresh writers, from the value rules of the T7
  forms, and the same pilot, gate and pool rules on P (R16, Z3). It runs whether or not rank 4's form and holes show a gap, because only a
  verdict closes row 7.

## 4. Harms

Each harm is a column, read by rule 5. By the sign of rule 5, "Loss measured" means the harm is
measured in V (or in the checked arm), beyond X. A harm smaller than X reads "Within margin", with
its interval.

| Harm | Campaign | Measure, unit, X |
|---|---|---|
| N1, the cost of a new language | C1 | Row 1 against L |
| N2, turns spent on annotations and the solver | C1, on tasks that touch annotated code (R15) | Per task, log of mean turns, L minus V. X ln 1.1 (hr3 N2) |
| N3, runs that do not stop | C1, C2 | Per task, the rate of budget hits, P minus each checked arm. X 10 points (hr3 N3) |
| N4, weakened contracts and lists | C1, C2 | Per task, the rate of runs that weaken one, by a scripted diff against the base. L minus V, and P minus each checked arm. X 5 points (hr3 N4). Each arm's rate is secondary |
| N5, Go-looking text with another meaning | Calibration, on N5's seed pool (rule 6), in every case (R15) | Per seed, the rate of agent faults, L minus V's stricter mode. X 12 points (hr3 N5). Rank 3's idiom seeds are history (AB4) |
| N6, conflicts on the central file | C7: chains of changes by parallel runs, in every arm | Per chain, the rate of merge failures, P minus V and P minus L. X 5 points (hr3 N6) |
| NV1 to NV3 | C6 | Views draft section 5 |

## 5. Features on, per campaign

"Base" is what each arm ships today, with the default ruleset frozen as `defaults v1`, and P's
linter configuration frozen with it. Each column closes on the comparison in section 2.

| Campaign | Conditions | Features on | Columns it closes |
|---|---|---|---|
| C1, ordinary tasks | P (with conventions file); P+ (5a assertions and 2 linter); L; L+gen; V; L−hints; V−hints; L+hole; V+hole | Base only, except the feature named by the condition | Rows 1, 8, 12, 16, 17; N1 to N4 |
| C2, tempting architecture tasks | P (with its linter); P−linter; L; V; L−fixes; V−fixes; L−warnings; V−warnings | Base, fixes (11d) and row 11's warnings | Row 3 and its secondary; 11d; secondary of 11a and 11b; N3, N4 |
| C2v, value-rule tasks | P; L+T7; V+T7 | Base and the T7 forms | Row 7 |
| C3, reviewer sessions | Seed writers' diffs of C1 and C2 tasks (AD3), by arm, with seeds | None | Rows 3 (review), 4, 5d |
| C4, cross-service tasks | P; L; V | Base and the 10a subset | Row 10a (secondary) |
| C5, a larger codebase | P; P−tools; L; L−context; V; V−context | Base and `context`, `query` | Rows 13a, 13b |
| C6, views | As the views draft | Base and the views tools | Row 14 |
| C7, parallel chains | P; L; V | Base | N6 |
| Timing, rows 15 and 19 (AD7) | Row 15: P; L; V. Row 19: P; V | Base, with the warnings and fixes of row 11 off (AE3). Row 19: P has assertions in place of contracts | Rows 15, 19 |
| Attackers, row 18 (AD7) | L; V | Base and `vulcan audit` | Row 18 |
| Calibration | P; L (with the checked library); V. For N5 only: L and V (stricter mode), on N5's seed pool (AB4) | Base | Rows 6a, 6b, N5 |

## 6. Order, batches and cost

### Order

| Phase | Work | Rows |
|---|---|---|
| A. No agents, no new build | A7, A8, on the base apps, as a dry run that sizes the machine time; no verdict reads it, and its outputs are sealed (AD8, AE1, AE2, AF1) | 15, 19 (dry run) |
| B. Builds, each with a fresh builder of each form and a parity audit, in the order below | See "Build order" | All rows with a build |
| A'. No agents, after their builds | Plan-r3 T1 with the P column (after P's linter), as history; row 2's fresh rule writers (rule 6); A1, A1b, rank 1 step 1, A3, A4, A5, A5b, A6a, A6b, T20 seeds, rank 4 form and holes; A7 and A8, after builds 1 and 5, with the check of rule 6 (AD8, AE3) | 2, 5, 7 (secondary), 9, 10, 11, 15, 19 |
| C. Agent campaigns | Calibration, C1, C2 (after its pilot), C2v (after its pilot), C4, C5, C6, C7, row 18's holes (attacker sessions, AA5), then C3 on the seed writers' diffs (AD3) | 1, 3, 4, 6, 7, 8, 11d, 12, 13, 14, 16, 17, 18, harms |

Replication is inside each sequence (rule 7), so there is no separate phase for it.

**Build order** (R14), by the rows a build opens, so that a stop leaves a known set of rows built:
1. P's architecture linter configuration (row 2, C2).
2. The tempting tasks and the C2 pilot (rows 3, 11d).
3. The L generator (row 8).
4. Hints off, and the hole command and its L equal (rows 12, 16).
5. P's assertions and shared fake suites (rows 5a, 9).
6. The warnings and fixes of row 11 (11a, 11b, 11d).
7. The trap rewriter and L's checked-math library (row 6, N5).
8. The T7 forms and the C2v tasks (row 7).
9. `context`, `query`, `audit`, and the port of C5's codebase (rows 13, 18).
10. Exact arithmetic or D8 (row 5b).
11. The verifier, trap mode and L's checked arithmetic (row 5c).
12. The crossing subset with dead letters, or D9 (row 10).
13. The views tools (row 14).

**Build effort.** About 13 builds. Each has two or three forms by fresh builders and one parity audit.
Plan-r3's one build needed six audit reruns. At 2 to 3 builder sessions and 3 to 6 audit passes each,
phase B is about 65 to 115 sessions. This plan does not estimate calendar time.

**The C5 codebase.** An open-source Go service, with a license that allows it (D4). A fresh session
writes one structure plan for it. A second session ports it to V, and a third declares L's model,
both from that plan. P gets the same restructured Go without the model. P on the original layout is a
second P condition only if the owner asks. A parity audit reads all three.

### Batches

A batch is fixed now, by unit type. It does not change after data (rule 6).

| Unit | One batch |
|---|---|
| Task | 50 draws from the campaign's pool of 400 tasks; in each condition, 2 runs a task under each of the two models (200 runs a condition) |
| Seed (fixed-outcome) | The first read at 20 writers of 10 seeds; then 10 fresh writers of 10 seeds |
| Seed (agent, AA1) | 20 draws from the pool of 200 seeds; each draw read once in each condition under each model |
| Rule (fixed-outcome) | The first read at 10 fresh rule writers; then 10 fresh rule writers |
| Calibration workspace | 36 workspaces an arm under each model, from 12 draws from the calibration pool of 400 tasks |
| Attacker | 3 pairs of attacker sessions, both models in each pair |
| Timing | 20 draws from the fixed set of changes or request paths, each run on both machines (AA5) |

The task batch comes from the equal-cost comparison below: at 200 runs a condition, more tasks with
fewer runs reach a verdict in fewer runs.

### Simulation (W1, W2, W8)

The script [cs-sim.py](../../experiments/poc/sim/cs-sim.py) runs every scenario with one command
(`python cs-sim.py`) and saves each output in [sim/out](../../experiments/poc/sim/out). Seed
20261105. Assumptions: a correlation of 0.3 between runs of one task, no task-by-arm interaction, no writer effect except in `writers` and `fixed`, and
for log measures a per-task standard deviation of D of 0.32 (the audit's value). Calibration measures
the first and the last. "Batches" is the median over all simulations, and the 90th percentile in
brackets, with "more than the cap" where a simulation did not close. Each row's cap is in the
output file.

**The method under the boundary and under different models** (`models_differ`, `success`):

| Case | False verdicts in 200 |
|---|---|
| One constant mean, true difference = +X (12 points, base 0.8) | 2 (1%) |
| One constant mean, true difference = −X | 6 (3%) |
| Models differ (+22 and +2 points), both inside each task (draft 4) | 1 (0.5%) |
| The same, batches alternate by model, A first (draft 3) | 22 (11%) |
| The same, batches alternate, B first | 37 (18.5%) |
| Models differ (+32 and −8 points), both inside each task (draft 4) | 3 (1.5%) |
| The same, batches alternate, A first | 115 (57.5%) |
| The same, batches alternate, B first | 132 (66%) |

So at this batch of 50 tasks, alternation breaks the bound, and the error grows with the difference
between models. The in-unit design of rule 7 stays within 5%. At draft 3's own batch of 20 tasks,
neither the audit's runs (section 11.2 of the audit) nor this script's first runs showed a failure of
alternation at reachable values: alternation fails when one batch can reach a verdict on one model
alone. The audit's section 10.8 case 2 asked for a value that cannot be reached, and the audit
withdrew it. The script now refuses such a target.

**Task success, X = 12 points** (`success`):

| True difference | Base rate | Outcome in 200 | Batches |
|---|---|---|---|
| 0 | 0.8 | Within margin, 200 | 2 (2) |
| +4 points | 0.8 | Within margin, 200 | 2 (4) |
| +6 points | 0.8 | Within margin, 200 | 3 (6) |
| +8 points | 0.8 | Within margin, 200 | 6 (13) |
| +10 points | 0.8 | Within margin, 123; open, 77 | 24 (more than 30) |
| −22 points | 0.8 | Loss, 200 | 2 (4) |
| −10 points | 0.6 | Within margin, 66; open, 134 | more than 30 |
| +15 points | 0.6 | Gain, 167; open, 33 | 14 (more than 30) |
| +18 points | 0.6 | Gain, 200 | 4 (8) |
| +21 points | 0.6 | Gain, 200 | 3 (4) |

**Every margin of the plan** (`margins`, 50 simulations a row; the second true value is half the
margin):

| Margin, measure | Batches at true 0 | Batches at X / 2 |
|---|---|---|
| 5 points, task success (row 1 against L) | 7 (12) | 15 (30) |
| ln 1.05, log tokens (row 8 against L) | 13 (19) | 30 (72) |
| ln 1.10, log turns (N2) | 4 (5) | 9 (17) |
| ln 1.15, log tokens (rows 8 against P, 13a) | 2 (3) | 4 (8) |
| ln 1.20, log time (row 15), batch 50 | 2 (2) | 3 (5) |
| ln 1.20, log time (row 15), timing batch of 20 (AA5) | 4 (5) | 6 (11) |
| ln 1.10, log time (row 19), timing batch of 20 (AA5) | 9 (11) | 20 (50) |
| 12 points, seeded agent draws, 1 run a model, batch 20, base 0.8 (C3a, C3b, C3c, N5; AA1) | 5 (8) | 12 (31) |
| 12 points, seeded agent draws, base 0.5, 1 run a model, batch 20 (AB2) | 8 (14) | 21 (more than 60) |
| 5 points, task success, base 0.5 (row 1 against L, N1, N4; AC6) | 10 (18) | 26 (72) |
| ln 1.25, log turns (row 11d) | 2 (2) | 2 (3) |

The base rate of a seeded agent row is not known before calibration. So the seeded rows have two
bases: 0.8, as the other success rows, and 0.5, which gives the most variance (AB2). The base of
the 5-point task success rows is not known either, so that row also has both bases (AC6). The
cost below gives both. The header of the `margins` output now says which rows use base 0.5.

**Equal cost** (`equal_cost`, 200 runs a condition a batch, X = 12 points, base 0.8):

| Design | Runs a condition to a verdict at true 0 | At +8 points | At −22 points |
|---|---|---|---|
| 20 tasks × 5 runs a model | 800 | 2,000 | 1,000 |
| 34 tasks × 3 runs a model | 408 | 1,224 | 612 |
| 50 tasks × 2 runs a model | 400 | 1,200 | 400 |

What it shows:
- A column on task success at X = 12 points closes in about 2 to 6 batches when the true difference
  is at least 4 points from the margin. Within about 2 points of the margin, it often stays open
  past 30 batches. At the margin itself it does not close in expectation.
- The small margins of accepted rules drive the cost: ln 1.05 (row 8 against L) needs about 13
  batches at a true 0, and 30 or more near half the margin.
- Fixed-outcome columns are in their own table below. They cost writer sessions, not agent runs.
- Round 1's −19.4 points would close as "Loss measured" in about 2 to 4 batches.

**Writers that differ** (`writers`, Y1; base 0.6, a population difference of exactly +X = 12 points,
50 tasks a batch, both models in each task; a writer shifts the difference of all its tasks by a
normal value with the standard deviation shown):

| Writer spread | One fresh writer a batch (draft 4): false verdicts | Sealed pool of 20 writers × 20 tasks (draft 5): false against the pool's mean | The pool's mean, standard deviation around the population's | Pool verdicts, all wrong for the population | Of those, the writer-level interval shows the step (Z1; one seed, see the three-seed table below) | Writer-level interval holds the population value, all 200 | Its median width |
|---|---|---|---|---|---|---|---|
| 0 | 4 of 200 (2%) | 2 of 200 (1%) | 0.2 points | 4 | 0 | 189 | 3.0 points |
| 10 points | 82 (41%) | 0 | 1.7 points | 57 | 44 (77%) | 187 | 8.0 points |
| 15 points | 144 (72%) | 0 | 2.4 points | 67 | 61 (91%) | 194 | 11.5 points |
| 20 points | 168 (84%) | 1 (0.5%) | 3.5 points | 96 | 87 (91%) | 191 | 14.9 points |

At a population difference of 0 and a 10-point spread, the pool closes Within margin in 2 batches
(90th percentile 4), with no false verdict. So the sealed pool keeps the bound for its own mean. Its
mean differs from the mean over all possible writers by about the writer spread divided by √20. At
the margin, that difference decides the verdict: in this table, 28% to 48% of the pool runs at a spread of 10 to 20 points reached a
verdict that is right for their pool and wrong for the population. The bound does not cover that
step (section 12). "Shows the step" means that the writer-level interval of rule 5 holds the
population value, so the record's note appears. At a spread of 0, the 4 false verdicts are errors of
the sequence itself, within its 5%, and the narrow writer interval does not show them. At a
population difference of 0 and a spread of 10 points, the writer-level interval holds 0 in 193 of
200 and lies within ±X in 172. Calibration, with its tasks from 20 writers, measures the spread for
D15. A pool of 40 writers is not simulated.

**The note over three seeds** (`writer_note`, AA2; seeds 20261105, 20261109 and 20261110, 200
simulations each, a population difference of exactly +X, so every verdict is wrong for writers in
general; each seed starts a fresh generator, so seed 20261105 differs a little from the table above):

| Writer spread | Verdicts in 600 | The note appears | Wrong for the population, with no note, over all 600 simulations | Range over the three seeds | The same, given a verdict | Range over the three seeds |
|---|---|---|---|---|---|---|
| 10 points | 147 | 102 (69%) | 45 of 600 (7.5%) | 5% to 10% | 45 of 147 (31%) | 25% to 42% |
| 15 points | 226 | 193 (85%) | 33 (5.5%) | 4% to 6.5% | 33 of 226 (15%) | 12% to 16% |
| 20 points | 259 | 217 (84%) | 42 (7.0%) | 3.5% to 10.5% | 42 of 259 (16%) | 10% to 22% |

So at the margin, a verdict that is wrong for writers in general and carries no note occurs in about
5% to 8% of simulations. The binomial intervals of the three rates run from about 4% to 10%. Given a
verdict, the rate is about 15% to 31% (exact intervals 23% to 39%, 10% to 20% and 12% to 21%). Here
every verdict is wrong, so these are the shares of verdicts that carry no note. Rule 5's limit
means this second rate. Both rates are about writers in general. The 5% of rule 5 is about the pool, so
neither is the same quantity. The note is a help to the reader, not a bound (rule 5, "Its limit").

**Rate columns** (`rates`, Y2): row 6b (36 workspaces a batch, both models, X = 2 points) closes Within
margin in 10 batches at a true rate of 0, 13 (17) at 0.5%, 22 (34) at 1%, and does not close in 60 at
2%. N6 (merge failures, base 10%, X = 5 points) closes in 4 (6) at a true 0 and 8 (19) at half the
margin.

**Fixed-outcome columns** (`fixed`, Z2; the unit is a writer, base rate 0.5, which gives the most
variance; 200 simulations a row; writers at the close, median and 90th percentile in brackets):

| Column type | Writer spread | At true 0 | At X / 2 | False verdicts at true X |
|---|---|---|---|---|
| Seeds, 10 a writer, X 10 points | 0 / 10 / 20 points | 90 (110) / 90 (120) / 100 (150) | 160 (280) / 160 (290) / 210 (more than 400) | 0 / 0 / 1 of 200 |
| Seeds, 40 a writer, X 10 points | 0 / 10 / 20 points | 80 (90) / 80 (100) / 90 (110) | 140 (180) / 150 (200) / 180 (290) | 0 / 0 / 3 |
| Seeds, 10 a writer, X 5 points (row 10a) | 0 / 10 / 20 points | 220 (340) / 250 (390) / 330 (510) | 500 (1,040) / 600 (more than 1,200) / 880 (more than 1,200) | 4 / 1 / 4 |
| Rules, 28 a writer, X 14 points (row 2) | 0 / 10 / 20 points | 60 (60) / 60 (70) / 60 (80) | 100 (120) / 100 (140) / 110 (160) | 1 / 0 / 0 |

What it shows: with the writer as the unit, the bound holds for writers in general at every spread
(at most 4 false verdicts in 200). The mirror at −X (`fixed_minus`, AA6) gives 1 and 2 false in 200
for seeds at X 10 points (spread 0 and 20), 3 and 2 at X 5 points, and 0 and 1 for rules. The cost is in writers: 40 seeds a writer save only about 10% of
writers, because the method's bets are bounded (truncation 1/2), so 10 seeds a writer stay. Row 10a's
margin of 5 points, from hr3 T16, needs about 220 to 880 writers.

**Row 18** (`attackers`, Z4; 3 pairs a batch, X 1 bypass; Poisson means are assumptions until the
holes stage): L 1 and V 1 close Within margin in 13 (15) batches; L 1 and V 0.5 in 22 (29); L 0.5 and
V 0.5 in 12 (14); L 2 and V 0.5 close as Gain measured in 34 (47); L 2 and V 1, a difference of
exactly X, do not close in 200 batches.

### Cost

**One convention in every cost row** (AC5): a batch range runs from the median at a true
difference of 0 to the median at X / 2. It is a typical value, not an upper bound. Every cost row follows it. The 90th
percentile is higher: for example, C3 reaches 31 batches at base 0.8 (about 14,900 sessions) and
more than 60 at base 0.5 (more than 28,800 sessions).

One batch of every agent campaign, with all its conditions (section 5), both models:

| Campaign | Conditions | One batch, all conditions | Slowest columns (batches, typical) |
|---|---|---|---|
| C1 | 9 | 1,800 runs | Row 8 against L (ln 1.05): 13 to 30. Row 1 against L, N1, N4 (5 points): 7 to 15 at base 0.8; 10 to 26 at base 0.5 |
| C2 | 8 | 1,600 runs | N4 (5 points): 7 to 15 at base 0.8; 10 to 26 at base 0.5. Row 3 (12 points): 2 to 3. N3 (10 points) is not simulated; N4 sets the end of C2 |
| C2v | 3 | 600 runs | Row 7, 12 points: 2 to 3 |
| C4 | 3 | 600 runs | Secondary only: 1 batch |
| C5 | 6 | 1,200 runs | Rows 13a, 13b: 2 to 4 |
| C6 | As the views draft | about 300 sessions | Views draft |
| C7 | 3 | 600 runs | N6, 5 points: 4 to 8 |
| C3 | Reviewer sessions | 480 sessions: 20 seed draws × 2 models × 12 conditions (C3a 6, C3b 3, C3c 3) | 12 points: 5 to 12 at base 0.8; 8 to 21 at base 0.5 (AA1, AB2) |
| Calibration | 3 arms | 216 runs | Row 6b, 2 points: 10 to 22 |

By rule 6, a condition leaves its campaign when no open column reads it. The estimate counts, for
each batch, the conditions that some open column still reads (Y2):

| Campaign | Estimate |
|---|---|
| C1 | 6 batches × 1,800; then P, L, L+gen and V for row 8, row 1, N1 and N4, 7 to 24 batches × 800: about 16,400 to 30,000 runs. At base 0.5 the 5-point columns take 10 to 26 batches. They close inside row 8's slowest column (13 to 30 batches in total), so C1 costs the same (AD5) |
| C2 | 6 batches × 1,600; then P, L and V for N4, 1 to 9 batches × 600: about 10,200 to 15,000 runs. At base 0.5, N4 takes 10 to 26 batches: 6 batches × 1,600; then 4 to 20 batches × 600: about 12,000 to 21,600 runs (AD5) |
| C2v | 2 to 3 batches × 600: about 1,200 to 1,800 runs |
| C5 | 2 to 4 batches × 1,200: about 2,400 to 4,800 runs |
| C4 | about 600 runs |
| C7 | 4 to 8 batches × 600: about 2,400 to 4,800 runs |
| C3 (AA1, AB2) | At base 0.8, 5 to 12 batches × 480: about 2,400 to 5,800 sessions. At base 0.5, 8 to 21 batches × 480: about 3,800 to 10,100 sessions |
| C6 | about 300 sessions a batch, 2 to 6 batches: about 600 to 1,800 sessions |
| N5 (AA1, AB2) | At base 0.8, 5 to 12 batches × 20 seed draws × 2 arms × 2 models: about 400 to 960 runs. At base 0.5, 8 to 21 batches × 80: about 640 to 1,700 runs |
| Row 18 (Z4) | 12 to 22 batches × 12 attacker sessions: about 150 to 270 sessions, from the `attackers` scenario |
| Calibration | 10 to 22 batches × 216: about 2,200 to 4,800 runs |
| **Runs and review sessions** | **about 39,000 to 71,000 at base 0.8; about 41,000 to 76,000 at base 0.5**. With the 5-point success rows at base 0.5 too (the C2 line above): about 42,000 to 82,000 |
| Timing runs (AB6) | Rows 15 and 19, no agent: 20 draws × 2 machines × the arms a batch. Row 15, 3 arms: 120 runs a batch, 4 to 6 batches, about 480 to 720. Row 19, 2 arms: 80 runs a batch, 9 to 20 batches, about 720 to 1,600. Machine time only; not in the total above. It has no duration: no run of the base apps has been timed, so any figure would be a guess. The dry run of phase A (one batch of each row, 200 timed runs, and 60 untimed trace runs of row 15, not in the range above) gives a lower bound for each row (AE1, AF2). The record states the total of each row over all arms and both machines, never per arm (AE2, AF6), and the total of batch 1 of phase A' before batch 2 (AC2). The memo session of rule 6 multiplies the total of row 15 by 4 to 6 batches and that of row 19 by 9 to 20, and reads both for the timing campaign (AD7). Phase A' adds 120 untimed trace runs of row 15, once (40 changes × 3 arms, AF2). Trace runs are machine time only |
| Task pools (Y3, Z3) | 7 task pools × 20 writers (40 for C1, C4, C5 and C7 if D15's rule applies), and a check of each pool: about 150 to 230 sessions, written once. A failed C2 or C2v pilot adds a pool of 20 writers |
| Seed pools (AA1) | 4 seed pools (C3a, C3b, C3c, N5) × 20 writers, and a check of each: about 84 sessions. Also one run of the drawn task in each arm for each writer of C3a, C3b and C3c (AD3): about 180 runs |
| Labelers (AA6) | One labeler a rule set (plan-r3's procedure), 60 to 110, and row 9's labeler: about 60 to 110 sessions |
| Fixed-outcome writers (Z2) | 8 seed rows at X 10 to 12 points × 90 to 210 writers, row 10a × 220 to 880, row 2 × 60 to 110 rule writers, and a check session per 10 writers: about 1,100 to 2,900 sessions. They grow with the batches, as agent runs do |

Columns whose true difference lies near their margin can need much more: the 90th percentile of row 8
against L at half its margin is 72 batches. Round 1 had 72 runs. This is the cost of the owner's rule
with a valid error rate. Earlier drafts gave lower figures that were not valid (draft 2, S1) or left
columns out (drafts 3 and 4, W2 and Y2; draft 5, row 18, Z4; draft 6, the seeded review columns, AA1). The owner decides per campaign batch (D7), by the memo of
rule 6. D13 can lower the cost, if it is decided before a campaign starts.

C1 and C2 replace plan-r3 step 8 (plan-r2's about 370 runs) and rank 2's 180 runs. They are not in
addition to them.

## 7. The crossing-processes subset

Spec §5.5 is milestone 5 of the development plan. This plan builds what the claim needs, not the full
feature:

- Two containers as separate binaries, with a transport that serializes each message.
- One topic and one remote port between them.
- `topics.lock`, and the check that rejects an incompatible change.
- Dead letters, unless D9 narrows the claim.
- A test runs an old consumer against a new producer.

L gets a Go library with the same transport and a lock check over Go types. Its record says whether it
shares V's evolution code. P gets a schema with `buf breaking`.

## 8. Leans that remain

| Lean | Direction | Why it stays |
|---|---|---|
| The pilot gate reads P's breaks (P without its linter) | Toward V and L, against P | A gate on V or L would select on the checks under test. Named, and the pilot is reported in all arms |
| A third pool with larger changes, after two failed pilots | Toward V and L, against P: the tasks move away from ordinary work | Only a verdict closes rows 2 and 3. The second model is fixed in advance, so it is not chosen for its breaks |
| C2 on the union of enforced rules, after a failed plan-r3 gate (D1d) | Toward V and L against P, by the selection of rules; also toward a task test that plan-r3 found not worth its cost | The union includes P's linter. Results are per rule, with T1's share covered |
| Arm P gets common tools that rounds 1 and 2 did not give it | Toward P | It is what a Go team uses (rule 2) |
| P's linter has a default configuration, as L and V have `defaults v1` | Against P on rules that a team would write for its own linter | The arms get the same information about the test (R3). Option B (a team rules file in every arm) is D14 |
| L may use every Go tool, with no cost of adoption | Toward L | Hr3 section 5 names it |
| A reviewer can see the arm | Either | It can't be hidden. One fixed prompt per arm |
| L's `context` and `query` give the same output as V's | Toward a null for V against L | The column then measures the code form only, as rule 3 says |
| Shared engine in rows 2 and 11 | Toward a null for V against L | Rule 3 |
| One writer plants the same seed in each arm's diff (C3; AB3) | Unknown: the planting can differ in difficulty by arm | The check confirms that the seeds are equal, not that they are equally hard to find. Rule 6 fixes the run (one run made for the seed writer, under the main model; AD3), so the writer does not choose the diff. That run is one sample, and its difficulty can differ by arm. Named as a limit (AC4) |
| Row 15 restores a warm cache in a fixed state in every arm (AC1) | Toward the arm whose cache saves most work after a one-file change; the direction is not known. The file cache is dropped after the restore and before each run, so it also leans toward the arm that reads fewer cache files from disk (AD1, AE5). The arm with the largest restored caches reads most from disk; which arm that is, the parity audit's list of paths shows. The direction is not known before it. The drop also removes the program files of the tools, and arms start different numbers of programs (P and L several, V one command that runs the same tools). Step 3 of the run reads the listed program files back in every arm (AF3), so this lean does not stay for the listed files. A program file outside the list stays a parity-audit finding. The warnings and fixes of row 11 are off in the timed check: toward V and L on time, because their shipped check does more work than the Base check that is timed, while P keeps `dupl` and `errcheck` in every arm (AF4) | It is the incremental check that the row claims. A cold full check is not measured. The list of caches is closed (rule 6), and a cache outside it is a parity-audit finding. The drop of the file cache is the same in every arm, but a run reads its restored caches from disk. The changed file is in the file cache in every arm. The warnings and fixes of row 11 are off in the check, so the row does not measure their time (AE3). The record names the row "the Base check" in its report (AF4). It times no second check with the warnings on (AF4): that sequence adds about 120 runs a batch, and the warnings come from build 6, which A7 and A8 do not wait for. The owner can fund it as a secondary sequence of the same rules |
| The common tools in the check of row 15, and the scope of a change (AE4) | Toward Within margin, in both directions: the time of the common tools is a cost of the same kind in every arm, so it moves the ratio of check times toward 1, and `gopls check` adds one more such term. Also against the arm whose build rewrites more files for one change (V, if the lowering fans out). Whole-program tools run on the whole module in every arm (AF2): their time does not shrink with the change, so it also moves the ratio toward 1. They lean against the arm with more whole-program work; which arm that is, the units that the commands name show. The set of changes holds no change of a signature or an export (AF2), so the row does not measure a change that breaks dependent files: this leans toward the arm whose check time grows most with dependents, which can be V if the lowering fans out; the direction is not known. The size of all of these is not known | The common tools are rule 2's, and the scope rule is the same in every arm (rule 6). The record gives the count of files each tool reads for each change, by a trace that is the same in every arm and runs outside the timed runs (AF2). The set of 40 holds no change of a signature or an export, so the row does not measure a check after a change that breaks a dependent file. No check has been timed |
| Row 19 runs in steady state: the warm-up ends when the last 100 requests have a median within 5% of the 100 before them, at most 600 requests (AD2, AE6) | Toward the arm that settles slower: a drift left in its timed window reads its time too high. The direction is not known. The rule leans less than the draft 10 rule (neighbouring 50-request blocks) for an exponential start-up, but it is not free of lean, and for a slowly decaying tail the lean is as large as for the old rule (AF5) | In a model with a start-up excess (audit, 17.5, AE6; not a measurement of the base apps), the draft 10 rule left more than 2% of excess in the timed window in 12% to 100% of the runs of its four slow cases (12, 82, 100 and 24; `sim/out/warmup.txt`, line `nb 8 0.05`). This rule, run on the same cases, leaves it in 0% to 8% of the runs (0, 8, 100 and 0), except for a very slow start (a time constant near 200 requests), where it leaves it in every run. The cap, the band and the form were chosen among seven settings by their score on these five exponential cases, so these cases do not test the rule (AF5). The model also runs power-law tails, with excess (1+n/50)^(-p) for p = 0.5 and p = 1.0, at noise 0.1, 0.3 and 0.5. They were added after the rule was fixed, and the rule was not changed for them. Both rules leave more than 2% of excess in 100% of those runs (lines `nb 8 0.05 pow` and `2v2 12 0.05 pow`). The share unsettled at the cap is 0% to 12% for the draft 10 rule and 0% to 2% for this rule. So the rule does not find a power-law tail: it stops at a block where the window median is stable, not where the excess is gone. The 2% bound is one fifth of the margin ln 1.1 (rule 6). The shares of runs with a residual drift above 2% and above 10%, by arm, are the limit that the record reports beside the verdict. Start-up and first-request time are not measured. A path that does not settle in 600 requests is counted per arm (rule 6) |
| One filter for generated code | Either | The same written rule in every arm; counts reported (rule 2) |
| A margin of 5 points (row 1 against L, hr3 N1) and ln 1.05 (row 8 against L, rank 2) | Toward more batches, not toward an arm | They are accepted rules. Small margins need many batches (section 6) |
| A run that does not reach its end point counts its total cost (rule 5) | Against the arm with more failed runs, on cost rows | Leaving such runs out leans the other way, toward that arm (W6). The count of such runs is reported per arm |
| Row 6 has a fault column and a separate false-alarm column, with X 2 points for the harm | Neither by design | Rank 3 weighed a false alarm at about ten faults. Two columns keep both visible, and rule 11 does not net them (W5) |
| Intervals of open columns are sealed until they close | Neither | It removes the choice of what to fund by direction (W7). The owner sees less while a campaign runs |

## 9. The amendment, and what it changes

If the owner accepts, these changes are made in one amendment to plan-r3, hypotheses-r3 and the views
draft:

| Document | Old | New |
|---|---|---|
| Plan-r3, step 4 | `defaults v1` is frozen | `defaults v1` and P's linter configuration are frozen together, before the blind rules are seen |
| Plan-r3, step 6 | "Arm P keeps the team's rules as prose" | P keeps the prose, and also gets the tools of rule 2. T1 reports a P column |
| Plan-r3, step 8 | Plan-r2 steps 4 to 9, about 370 runs | Campaigns C1 and C2 of this plan |
| Plan-r3, section 5 gate | The gate stops the round's tasks | If D1d is yes: C2 runs on the union of the rules that any arm's check enforces |
| Hr3, every stop rule | One-sided, point and count rules; "Unclear stays reported as unclear" | Each column reads rule 5. The accepted threshold sets X, and the accepted condition is the hr3 reading, reported and not read by a verdict. The mirror loss exists for each |
| Hr3, rank 2 | 20 tasks, 3 arms, 5 runs | Part of C1, with the conditions L, L+gen and V. X = ln 1.05 |
| Hr3, rank 3 | Calibration workspaces of plan-r3 or plan-r2; 10 of 108 faults and at most 1 of 108 false alarms | Calibration in arms P, L and V; batches of 36. Two columns: faults shown (6a, X 5 points) and false alarms (6b, a harm, X 2 points). They are not netted |
| Hr3, rank 4 | Runs only if form or holes show a gap | C2v runs in every case; form and holes are secondary |
| Hr3, T16 | Differ by more than 1 seed | X 5 points on seeds |
| Hr3, T20 | Bypasses within 1, false rejections | Per seed rejected, X 10 points; bypasses and false rejections secondary |
| Hr3, T22 | Fall of 20%, arms within 12 points | V against V−fixes, X ln 1.25 |
| Hr3, T23, T24 | V better by a bound above 0 | Rule 5 with X 12 points, mirror loss included |
| Hr3, T25 | Ratio bound below 0.87 either way | X ln 1.15 |
| Hr3, N1 to N6 | One-arm or point thresholds | Section 4 |
| Hr3, replication | For Gains and named cases | Every column, inside the sequence (rule 7) |
| Views draft, section 3 | "No measured gain"; "Unclear: one more batch, sized before it is run" | "Within margin"; rule 5 and rule 6 |
| Views draft | Not accepted | Accepted as the design of row 14, with its own decisions, if D6 is yes. T29a gets a measure and a margin |

**New rules, where no accepted rule existed:** the X of every column with "None" in its hr3 reading.
These include rows 5b and 9 (10 points), which draft 2 had cited as T26 by mistake (R2).

## 10. Decisions for the owner

| # | Decision | Recommendation |
|---|---|---|
| D1a | Accept this plan | **Decided (2026-11-05): accepted**, after the tenth rerun of the audit |
| D1b | Does an unresolved result close a column? | **Decided (2026-11-03): no.** Rule 5 makes this valid |
| D1c | If no task tempts a break after two pilots | **Decided: the rows stay open.** The next step is written in section 3 |
| D1d | If the plan-r3 gate fails, does C2 run on the union of enforced rules? | Yes, with the lean named. A gate does not close a column, so without this rows 2 and 3 can't close |
| D2 | The tools of rule 2 in every arm, with the one filter for generated code | Yes |
| D3 | The crossing subset of section 7 | Yes |
| D4 | The C5 codebase: an open-source Go service (needs the download yes) | Yes, chosen by a fresh session for license and size |
| D5 | The second model, named before any pilot | The owner's choice, from the providers already used |
| D6 | Accept the views draft as row 14's design, with its five decisions | Yes |
| D7 | Spend: each campaign's next batch, from a direction-blind memo, with open intervals sealed | Decide per campaign batch |
| D8 | Exact arithmetic (C08): build it, or narrow the README's claim to Go arithmetic | Build it. It is small next to the verifier |
| D9 | Dead letters: build them, or narrow the claim | Build them in the subset |
| D10 | Rows 16 to 19 | Add all four |
| D11 | The Z3 download for the verifier | Yes |
| D12 | Rows 5c and 18 against P: no Go team has static proof or a gap list. A P column there can only read 0 (R7). Drop these two P columns, with the reason in the report, so that the rows close on their L columns? Or keep them and build P a form that no team uses? | Drop them, and say so beside each row. The P comparison of contracts is row 5a |
| D13 | Cut conditions to lower the cost: for example, C1 without L+hole and V+hole (row 16 then reads C1 base against a separate small campaign), or C2 without the −warnings conditions | Decide once per campaign, before its first batch, from calibration's pooled values only. No cut after a batch (rule 6) |
| D14 | Row 2 against P: also report option B (a team rules file in every arm, by one fresh session, same budget) as secondary | Yes, as secondary. The verdict reads option A |
| D15 | Writers per task pool, by a rule fixed now (Z1, AA4): 20 writers of 20 tasks. At the end of calibration's tenth batch (120 draws, about 6 a writer), a fresh session estimates the writer spread on task success, V minus P and V minus L: the standard deviation of the writer means of D, less the part that run and task noise explain. If either is above 10 points, the pools of C1, C4, C5 and C7, written after that read, get 40 writers of 10 tasks. C2's and C2v's pools are written in phase B and keep 20. If calibration stops before its tenth batch, no read is made and every pool keeps 20. At about 6 units a writer, each writer mean has a noise of about 10 points, the size of the threshold (audit 13.7, AA4), so near 10 points the rule decides almost at random. The spread has no sign, so the rule selects no direction | Accept the rule. Writers cost sessions, not runs. 40 writers narrow the step from the pool to all writers by about √2; this is from the formula, not simulated |

## 11. Answers to the open findings of the audit

Draft 1's findings that the audit's section 8 closed are not repeated here.

| Finding | Change |
|---|---|
| AF1 | Rule 6: the seal on the dry run covers every output that carries an arm's direction: times, unsettled counts, residual drift, file counts and tool output. The person who fixes a fault sees the fault, the two totals and the count of faults. No rule of rows 15 and 19 changes after the dry run except by an amendment (rule 13), before the batch 1 data, that cites no figure of an arm. Section 6 (order, cost) matches |
| AF2 | Rule 6: whole-program tools (P's architecture linter, and any analyzer or check that reads beyond the changed file) run on the whole module in every arm. The commands name the unit that each tool reads. The set of 40 changes holds no change of a signature or an export. Files read are counted by a trace in separate untimed runs: one for each change in each arm, once, in phase A' (120 runs); the dry run tests the method (60 runs). I chose a trace outside the timed runs, because a trace in a timed run adds load that differs by arm. Section 8 restates the lean of the AE4 row; section 6 adds the untimed runs to the timing line (machine time only, no total moves) |
| AF3 | Rule 6: a new step 3 reads the listed program files back into the file cache after the drop, in every arm, so a run has five steps. I chose this over naming the lean alone, because the row claims the incremental check, and a developer's second check finds the tools in the file cache. It is the audit's second option in a form that needs no exemption from the drop on any machine. The parity audit lists the program files that each check loads. Section 8 restates the file-cache lean |
| AF4 | Section 8 (row 15): the lean is toward V and L on time, and the record names the row "the Base check". No second timed check with the warnings on, because it adds about 120 runs a batch and the warnings come from build 6. The owner can fund it as a secondary sequence |
| AF5 | Section 8 (row 19) and AE6 row: the draft 10 range reads "12% to 100%", as `warmup.txt` shows. `warmup-sim.py` adds held-out power-law tails (p = 0.5 and 1.0), run with `python -I`, seed unchanged, with the first nine lines of `warmup.txt` unchanged. Both rules leave more than 2% of excess in 100% of those runs, so the plan names the tail as a limit and does not change the rule. The section says which cases chose the settings. Rule 6: the record reports by arm the share of runs with a residual drift above 2% and above 10%, as limits beside the verdict, sealed until the column closes. The 2% is one fifth of the margin |
| AF6 | Rule 6 and section 6: the dry run gives the total of row 15 and the total of row 19 as two figures, each over all arms and both machines, never per arm. The memo session multiplies the first by 4 to 6 batches and the second by 9 to 20 |
| AE1 | Rule 6: the dry run of phase A is defined. It runs one batch of row 15 and one of row 19 on the base apps, by the steps of the rule, with what each arm ships today (P: common tools only, and no assertions in row 19). The record writes its commands before the dry run. Its figure is a lower bound. Section 6: the cost row says so |
| AE2 | Rule 6: the dry-run times are sealed like an open interval. The record gives the total over all arms and both machines, never per arm. No column reads the dry run. The memo session reads that total and the total of batch 1 of phase A'. Rule 13 reads "A phase A' result" |
| AE3 | Rule 6 and section 5: the check of row 15 has the warnings and fixes of row 11 off in every arm, which is Base. Phase A' runs A7 and A8 after builds 1 and 5, not 6. Section 8 says that the row does not measure those warnings |
| AE4 | Rule 6: one scope of a change in every arm (the changed Go file, plus the Go files that the arm's own build rewrites because of the change); the count of files and packages per tool is recorded. Rule 2 names `gopls check`. Section 8: a lean row for the common tools (toward Within margin) and for the scope (against the arm that rewrites more files) |
| AE5 | Rule 6: each row 15 run takes four steps in this order: restore, drop the file cache, apply the change, time. Row 19 has no restore and no drop. The drop method is named for each machine and is part of the machine (rule 7). Section 8 restates the file-cache lean |
| AE6 | Rule 6: the warm-up of row 19 compares the last two blocks (100 requests) with the two blocks before them, from block 4, with at most 12 blocks (600 requests); the residual drift is recorded. The cap rises from 8 to 12 blocks, because with this form 8 blocks leave 9% to 66% of the runs of the audit's slow cases unsettled. Run on those cases, the rule leaves more than 2% of excess in 0% to 8% of the runs, against 12% to 100% for draft 10 (AF5 corrected the range from 12% to 97%). A very slow start stays a named limit. The model is `experiments/poc/sim/warmup-sim.py` (output `sim/out/warmup.txt`, line `2v2 12 0.05`). Section 8 restates the lean |
| AD1 | Rule 6: the cache list of row 15 is closed and named for each arm. A cache outside it is a parity-audit finding (rule 4). The record drops the file cache of the system before each row 15 run, in every arm. The arms of a draw run in a random order, fixed by a seed before batch 1. Rows 15 and 19 run with no other load. Section 8 names the file-cache lean |
| AD2 | Rule 6: row 19 warms up in blocks of 50 requests and ends at the first block, from the second, whose median is within 5% of the block before it, with at most 8 blocks. The record counts unsettled paths per arm. The rule reads no arm difference and is set before any data. Section 8 restates the lean |
| AD3 | Rule 6: a seed writer draws a task from C2's pool (C3a) or C1's pool (C3b, C3c). One run of that task in each arm, made for the seed writer only, gives the diff. So the diff always exists. Sections 2, 5 and 6 and phase C match. Section 8 names the lean. About 180 runs are added to the seed pool cost |
| AD4 | Section 6: the medians of the convention. C2v is 2 to 3 batches, C5 2 to 4, row 18 12 to 22, and row 3 of C2 is 2 to 3. N3 is not simulated, and the text says so. The totals fall |
| AD5 | Section 6: C1 at base 0.5 costs the same as at base 0.8, because the 5-point columns close inside row 8's slowest column. A C2 line at base 0.5 is added (12,000 to 21,600 runs). The total with the 5-point rows at base 0.5 is about 42,000 to 82,000, after AD4 |
| AD6 | This section: the AB6 row and the AC6 row are corrected |
| AD7 | Section 5: a timing campaign (rows 15 and 19) and an attacker campaign (row 18). Rule 6 funds each as any campaign. The timing cost row names the reader of the machine-time total |
| AD8 | Rule 6: the check of row 15 is one command an arm, with its tools. Phase A is a dry run that sizes machine time and counts in no verdict. A7 and A8 run in phase A', after builds 1, 5 and 6 |
| AC1 | Rule 6: row 15 runs from a warm cache in a fixed state, set the same way in every arm. Before each run, the record restores every cache that the arm's check uses to its state after one full check of the unchanged base; then the one-file change is applied and timed. Row 15 measures an incremental check. Section 8 names the lean. Cold full checks are not measured. Draft 10 restates this (row AD1) |
| AC2 | Rule 6: every cache is named by the AC1 rule, not two. Row 19 runs in steady state after 100 warm-up requests, in every arm. Rule 7: two named machines fixed for each timing column. Section 6: the timing cost row states that it has no duration, and why. Draft 10 restates this (row AD1 and AD2) |
| AC3 | Rule 14 lists N5 with rows 6a and 6b as an exception. Rule 6, section 4 and rule 14 now agree |
| AC4 | Rule 6: a C3 seed writer plants in the diff of the first run, under the main model, of the drawn task in each arm, from a seed fixed before the writers write. The pools are written after C1 and C2. Row 4 reads "the same seed planted in each arm's own diff". Section 8 names the limit. Draft 10 restates this (row AD3) |
| AC5 | Section 6: one convention above the cost tables (median at true 0 to median at X / 2). Timing rows read 4 to 6 and 9 to 20 batches. The 90th percentile of C3 is given in a note |
| AC6 | Section 6: a `margins` row for 5-point success at base 0.5, and base 0.5 ranges for C1 and C2 (C1 as at base 0.8, AD5). The `margins` header says which rows use base 0.5 |
| AC7 | Rule 5, section 6, section 11 (AA2) and section 12: both rates of the writer note, over all simulations (about 5% to 8%) and given a verdict (about 15% to 31%). Rule 5's limit means the second |
| AB1 | Section 6: the batch table reads "10 fresh rule writers", and phase A' lists row 2's fresh rule writers and T1 as history. Rule 6 gives the reason T1 is history: an earlier model and no corpus draw. R12's row is restated |
| AB2 | Section 6: the seeded agent row and the cost of C3 and N5 are given at base 0.8 and at base 0.5, and the total at both. A `margins` row at base 0.5 is added |
| AB3 | Rule 6: a seed writer draws one task, takes its diff in each arm, and plants the same seed in each. The check session confirms that the seeds are equal in each arm's diff. Section 8 names the limit |
| AB4 | Rule 6: N5's unit is the seed pool, and rank 3's idiom seeds are history. Rule 14 no longer lists N5. Sections 4 and 5: N5 runs in calibration, in L and V (stricter mode), on its seed pool |
| AB5 | Section 6: the task pool cost reads "about 150 to 230". D15: every pool keeps 20 writers if calibration stops before its tenth batch |
| AB6 | Rule 6: changed by AC1 and AC2 (row 15 from a restored warm cache; row 19 in steady state). Rule 7: the record names machine 1 and machine 2. Section 6: a cost row for the timing runs |
| AB7 | Rule 5: the count rule applies to every agent column with a pool, and reads "fewer than half of the pool's writers (10 of 20)". Sections 6 and 12: the rate of a wrong verdict with no note has binomial intervals of about 4% to 10%, and is not compared with the 5% of rule 5 |
| AA1 | Rule 6: C3a, C3b, C3c and N5 are agent columns with a seed pool of 20 writers × 10 seeds, drawn with replacement, each draw read by fresh sessions under both models. Rule 8 matches. Section 6: their batch, a simulation at X 12 points (5 to 12 batches), and their cost (C3 about 2,400 to 5,800 sessions, N5 about 400 to 960 runs, 84 writer and check sessions) |
| AA2 | Rule 5: the interval is at a nominal 95%; given a verdict its coverage is lower, and a missing note is not evidence. Rate columns near 0 give the count of writers with a nonzero mean, and "not informative" under 10. Section 6: the `writer_note` scenario over three seeds: a wrong verdict with no note in 5.5% to 7.5% of simulations, and in 15% to 31% of verdicts (AC7) |
| AA3 | Rule 6: T1 and row 5c's 29 existing clauses are history, as T2 is. Only fresh writers are units. Row 2's writers get round 2's brief and T1's labeler procedure |
| AA4 | D15: the spread is read at calibration's tenth batch; only pools written after it (C1, C4, C5, C7) can get 40 writers; C2 and C2v keep 20 (section 3). Rule 14 names the exception. D15 states that its input is weak near 10 points |
| AA5 | Rule 6: rows 15 and 19 draw from a fixed set, row 18 uses fresh attacker sessions; their verdicts name the set or the prompts (rule 5). Section 6 simulates ln 1.20 and ln 1.10 at the timing batch of 20. Row 18 moves to phase C. Row 11c's procedure fixes `-count`, `GOMAXPROCS`, the machine and the load, in every arm |
| AA6 | Rule 6: a writer that fails the check is replaced, with the count recorded. Section 6: labelers are costed, and `fixed_minus` tests −X. Rule 6 writes "the Z3 solver's `rlimit`" |
| Z1 | Rule 5: an agent column's verdict says "for this pool (N writers)", with a writer-level interval beside it, read once at the close and read by no verdict, and a note where it does not meet the verdict's condition. Over three seeds (AA2), the note appears on 69% to 85% of verdicts that are wrong for writers in general. D15 is a rule fixed before calibration |
| Z2 | Rule 6: fixed-outcome columns (rows 2, 5a, 5b, 5c, 9, 10a, 10b, 11a, 11b, 11c) take the writer as the unit. Each seed is evaluated once, and the sequence runs over fresh writers, so the bound covers writers in general and no unit is a copy. Rule 6 also says that a repeated agent draw carries no state. Section 6 simulates the design and its writer cost |
| Z3 | Section 3: C2's and C2v's pools come from 20 writers each; the pilot draws 10 tasks, which leave the pool; a failed gate makes a new pool, which is a new estimand. Rule 6: the pool check reads only the brief and the tasks |
| Z4 | Section 6: row 18 is simulated (`attackers`) and costed; section 11's W1 row is restated |
| Y1 (High) | Rule 6: a sealed pool per campaign or column, written first by 20 writers (rules: 10), checked, then drawn at random with replacement. Rule 5 states the pool's mean as the estimand, and the assumption now holds by design. Section 6 simulates writer spreads of 0 to 20 points: one writer a batch gave 2% to 84% false verdicts, the pool 0% to 1% against its own mean. The step from the pool to all writers is a named limit (section 12) and D15 |
| Y2 | Section 6: cost per batch counts every condition that an open column still reads; row 6b and N6 are simulated; the total is about 37,000 to 70,000 |
| Y3 | Section 6: pool writing is in the cost, about 370 sessions, once per pool. With a pool, the number of tasks to write no longer grows with the batches |
| Y4 | Rule 5: after an empty intersection, the campaign goes on unchanged; a change makes a new column by amendment |
| Y5 | Rule 6 seals the secondary sequences too; row 18's pair is four sessions; rows 6a and 6b carry a one-sided note |
| W1 (High) | Rule 5 defines the estimand (the mean of D, replication levels at equal weight) and states the method's assumption. Rule 7 puts both models inside each task, a fixed random split of each pool's writers into two halves (draft 5), and both machines inside each change. Section 6 simulates the boundary and the old alternation. At reachable values (base 0.6), alternation gave 11% to 66% false verdicts, and the in-unit design 0.5% to 1.5%. The audit's own case 2 asked for a value that cannot be reached at a base rate of 0.8; the conclusion does not depend on it |
| W2 | Section 6 simulates every margin of the plan, and gives each campaign's cost by its slowest column. Rule 6: a closed column's conditions leave the campaign when no open column reads them |
| W3 | Rule 5: the sign rule. Rows 18 and 19 restated; row 18 clips each arm's count, then differences them, pairs sessions by a fixed schedule, and replicates by model |
| W4 | Rule 5: "Within margin" replaces "No measured gain", with the interval and "(excludes 0)" |
| W5 | Row 6 splits into 6a (faults shown) and 6b (false alarms, a harm, X 2 points). Section 9 names the change of weight; section 8 names it |
| W6 | Rule 5: a run that does not reach its end point counts its total cost; the task's value is the log of its mean cost; counts reported. Section 8 names the lean |
| W7 | Rule 6: conditions fixed before a campaign's first batch (D13); open intervals sealed until the column closes or its campaign stops |
| W8 | `cs-sim.py` runs every scenario from one command and saves every output in `sim/out/`. Medians are over all simulations, with "more than the cap" as a value. The equal-cost comparison is in section 6, and it changed the task batch |
| W9 | Rule 5: an empty intersection is not a verdict; the column stays open, and the record names it |
| R17 | Unchanged: C6 waits for T29a's measure and margin in the views amendment |
| S1 (High) | Rule 5: one anytime-valid confidence sequence per column, one alpha over the whole series. Rule 7: replication inside the sequence, all batches read together |
| S2 (High) | Rule 6: the next batch is the same design; every batch pools; funding is per campaign, all or none, from a direction-blind memo; no change of measure |
| S4 (High) | Section 3 and D1d: the union of the rules that any arm enforces, P's linter included; per-rule results; the lean named in section 8 |
| R2 (High) | Rule 5: every column reads the same four outcomes with a mirror loss; accepted rules set X and are reported as the hr3 reading; section 9 lists each change; rows 5b and 9 are new rules |
| R3 (High) | Row 2 against P: P's linter gets a default configuration from published practice, frozen with `defaults v1` (option A). Option B is secondary (D14) |
| R4 (High) | Row 12: the primer stays in every condition; the primary is V against V−hints; the difference of differences is secondary |
| R5 | Rule 8: the task is the unit, so clustering is inside the unit's value; section 6 simulates with a correlation of 0.3. Rule sets grow by fresh writers, one batch each |
| R6 | Row 6: batches of 36 workspaces, read by rule 5; no 108-workspace threshold |
| R7 | Rows 11c, 16, 17 against P read an outcome that exists in every arm. Rows 5c and 18 against P: D12 |
| R8 | Rows 11a and 11b: the final-code counts are read in C2, warnings on against off (section 5) |
| R9 | Section 3: the pilot gate reads P without its linter; C2 has a P−linter condition |
| R10 | Rule 2: tools run on lowered Go; one filter for generated code in every arm; counts reported |
| R11 | Section 6: T1's P column moves to phase A', after P's linter; row 5c's P build is D12 |
| R12 | Row 2: T2 is history, without the dependency check; the sequence reads fresh rule writers only, and T1 is history (AB1) |
| R14 | Section 6: build order by rows, and an estimate of 65 to 115 sessions |
| R15 | Section 4: N2 in C1, N5 with idiom seeds in every case; rule 11: "Gain with a measured harm" |
| R16 | Section 3: C2v has its own writer and pilot gate, and runs in every case |
| R17 | Row 14: T29a needs a measure and a margin before C6 |
| R18 | Row 19: an application request path is the primary; P's same work is defined |
| R19 | The status paragraph names section 11 |
| R20 | Rule 9 and section 2: each column names one primary measure; the rest are secondary |
| S3 | Rule 5 replaces the group sequential design; count and point rules become interval readings (section 9) |
| S5 | Section 3: the second model is D5's, named before any pilot, in every arm; the lean is named |
| S6 | Rule 14: calibration sets only the pooled variance and correlation, for cost; batches are fixed now |
| S7 | Section 6: simulated batches to a verdict, and a cost range |
| S8 | Rule 7: no batch is repeated after a result; both levels of each replication sit inside every unit (W1) |

## 12. What this plan does not claim

- It does not claim that any principle will show a gain.
- A closed column means a test gave a verdict. It does not mean the principle holds.
- Rule 5 holds 5% per column. Over about 45 columns, a few false verdicts are possible (rule 16).
- On an agent column, rule 5's bound is about the column's sealed pool. With 20 writers, the pool's
  mean differs from the mean over all possible writers by about the writer spread divided by √20
  (section 6). A verdict near its margin can be right for the pool and wrong for other writers. The
  verdict says "for this pool", and the writer-level interval beside it can show the step (rule 5).
  It is not a bound: at the margin, a verdict wrong for writers in general and with no note occurs
  in about 5% to 8% of simulations (binomial intervals from about 4% to 10%), and in about 15% to
  31% of the verdicts reached (section 6). D15
  can enlarge the pools. On a fixed-outcome column, the bound covers writers who write from the
  brief, not cases from other sources.
- Rule 5's bound needs its assumption: every unit has the same mean. The design keeps it (rules 6
  and 7), but a drift in the units that no rule controls, such as a provider changing a model under
  the same name, would break it. The record gives the model version of every run.
- The batch counts of section 6 come from a simulation with an assumed correlation and assumed
  effects. Real counts can be higher or lower.
- The builds of phase B test the claims, not finished features. A result on a PoC build is not a
  result on a release.
- All campaigns use one application family plus one ported service. A result does not generalize past
  them without more corpora.
