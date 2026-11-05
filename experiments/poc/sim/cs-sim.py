# Simulation of rule 5 of docs/poc/plan-coverage.md: a hedged-capital betting confidence sequence
# (Waudby-Smith and Ramdas 2023, JRSS B, theorem 3) on per-unit differences D in [-R, R], mapped to
# [0, 1], read at the end of each batch, with the four outcomes against a margin X.
#
# Usage: python cs-sim.py            runs every scenario, writes out/<scenario>.txt
#        python cs-sim.py NAME ...   runs the named scenarios only
# Seed 20261105 for every scenario. Results are in plan-coverage.md, section 6.
#
# Data models (all assumptions, to be replaced by calibration's pooled values):
#   success: a task's base rate p ~ Beta, mean BASE, ICC 0.3 between runs of one task; the other
#            condition has p + delta for each model (clipped to [0, 1]); RUNS runs per model per
#            condition. A unit's D is the mean over models of (rate with delta - rate without).
#   normal:  a unit's D ~ N(mu, sd), clipped to [-R, R] (log ratios of tokens, turns or time).
#   alternate: the success model, but whole batches alternate between model A and model B, as
#            draft 3's rule 7 did. Kept to show why draft 4 drops it (audit finding W1).
#   fixed:   a unit is a writer; each of its seeds or rules has one fixed outcome per condition
#            (Z2). The writer's D is the mean over its seeds.
#   attackers: Poisson counts of unnamed bypasses per arm and model, clipped to 0 to 5 (row 18, Z4).
import sys, os
import numpy as np
from scipy import stats

ALPHA = 0.05
C_TRUNC = 0.5
THETA = 0.5
GRID = np.linspace(0.0005, 0.9995, 2000)
PLAN_TASKS, PLAN_RUNS = 50, 2  # the plan's task batch: 50 tasks, 2 runs a model (chosen by equal_cost)
OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), "out")


def run_sequences(units, X, R, batch, first=0):
    """units: array (sims, n) of D values in [-R, R]. Returns verdicts and batch counts per sim.
    first: no read before this many units (Z2: a writer pool is read first when it is complete)."""
    sims, n = units.shape
    y = (units + R) / (2 * R)
    m = GRID[None, :]
    lkp = np.zeros((sims, GRID.size))
    lkm = np.zeros((sims, GRID.size))
    alive = np.ones((sims, GRID.size), bool)
    sum_x = np.zeros(sims)
    sum_sq = np.zeros(sims)
    mu_prev = np.full(sims, 0.5)
    sig2_prev = np.full(sims, 0.25)
    verdict = np.array(["open"] * sims, dtype=object)
    when = np.full(sims, -1)
    for t in range(1, n + 1):
        x = y[:, t - 1]
        lam = np.sqrt(2 * np.log(2 / ALPHA) / (sig2_prev * t * np.log(1 + t)))[:, None]
        lp = np.minimum(lam, C_TRUNC / m)
        lm = np.minimum(lam, C_TRUNC / (1 - m))
        lkp += np.log1p(lp * (x[:, None] - m))
        lkm += np.log1p(-lm * (x[:, None] - m))
        k = np.logaddexp(np.log(THETA) + lkp, np.log(1 - THETA) + lkm)
        alive &= k < np.log(1 / ALPHA)
        sum_x += x
        mu_prev = (0.5 + sum_x) / (t + 1)
        sum_sq += (x - mu_prev) ** 2
        sig2_prev = (0.25 + sum_sq) / (t + 1)
        if t % batch or t < first:
            continue
        undecided = verdict == "open"
        empty = undecided & ~alive.any(axis=1)
        verdict[empty] = "empty"
        when[empty] = t // batch
        ok = undecided & ~empty
        if not ok.any():
            continue
        lo = np.where(alive, m, np.inf).min(axis=1) * 2 * R - R
        hi = np.where(alive, m, -np.inf).max(axis=1) * 2 * R - R
        for name, cond in (("gain", lo > X), ("loss", hi < -X), ("within", (lo > -X) & (hi < X))):
            hit = ok & cond
            verdict[hit] = name
            when[hit] = t // batch
            ok &= ~hit
    return verdict, when


def success_units(rng, sims, n, deltas, base, runs, icc=0.3):
    s = 1 / icc - 1
    p = rng.beta(base * s, (1 - base) * s, (sims, n))
    d = np.zeros((sims, n))
    for delta in deltas:
        pa = np.clip(p + delta, 0, 1)
        d += rng.binomial(runs, pa) / runs - rng.binomial(runs, p) / runs
    return d / len(deltas)


def alternate_units(rng, sims, n, delta_a, delta_b, base, runs, batch, a_first=True):
    s = 1 / 0.3 - 1
    p = rng.beta(base * s, (1 - base) * s, (sims, n))
    k = (np.arange(n) // batch) % 2 == (0 if a_first else 1)
    delta = np.where(k, delta_a, delta_b)[None, :]
    pa = np.clip(p + delta, 0, 1)
    return rng.binomial(runs, pa) / runs - rng.binomial(runs, p) / runs


def writer_units(rng, sims, n, delta, base, runs, batch, sd, design, writers=20, per_writer=20):
    """Y1: each writer shifts delta by w ~ N(0, sd) for all of its tasks; both models in each unit.
    design "one_writer": every batch is one fresh writer's tasks.
    design "pool": a sealed pool of writers x per_writer tasks, written first; units are drawn from it
    uniformly with replacement, so they are independent draws, and the estimand is the pool's mean.
    Returns the units, per simulation the true mean that the design's sequence estimates, and the
    writer of each unit (pool design only)."""
    s = 1 / 0.3 - 1
    if design == "one_writer":
        p = rng.beta(base * s, (1 - base) * s, (sims, n))
        w = np.repeat(rng.normal(0, sd, (sims, n // batch)), batch, axis=1)
    else:
        size = writers * per_writer
        pool_p = rng.beta(base * s, (1 - base) * s, (sims, size))
        pool_w = np.repeat(rng.normal(0, sd, (sims, writers)), per_writer, axis=1)
        pa_pool = np.clip(pool_p + delta + pool_w, 0, 1)
        truth = (pa_pool - pool_p).mean(axis=1)
        idx = rng.integers(0, size, (sims, n))
        p = np.take_along_axis(pool_p, idx, axis=1)
        w = np.take_along_axis(pool_w, idx, axis=1)
    pa = np.clip(p + delta + w, 0, 1)
    d = sum(rng.binomial(runs, pa) / runs - rng.binomial(runs, p) / runs for _ in range(2)) / 2
    if design == "one_writer":
        return d, np.full(sims, np.nan), None
    return d, truth, idx // per_writer


def writer_interval(units, wid, upto, writers=20):
    """Z1: the 95% t-interval over writer means, each writer's mean over its drawn units up to upto.
    A fixed-sample interval read once, when the column closes. Writers with no draw are left out."""
    lo, hi = np.empty(len(units)), np.empty(len(units))
    for i, (u, w, n) in enumerate(zip(units, wid, upto)):
        cnt = np.bincount(w[:n], minlength=writers)
        means = np.bincount(w[:n], u[:n], minlength=writers)[cnt > 0] / cnt[cnt > 0]
        half = stats.t.ppf(0.975, means.size - 1) * means.std(ddof=1) / np.sqrt(means.size)
        lo[i], hi[i] = means.mean() - half, means.mean() + half
    return lo, hi


def fixed_units(rng, sims, n_writers, per_writer, delta, base, sd, icc=0.3):
    """Z2: a unit is a writer. Each of its per_writer seeds (or rules) has a fixed outcome in each
    condition, drawn once: caught with p (condition B) or p + delta + w (condition A), w ~ N(0, sd)
    per writer. The writer's D is the mean over its seeds of A minus B. Every writer is new."""
    s = 1 / icc - 1
    p = rng.beta(base * s, (1 - base) * s, (sims, n_writers, per_writer))
    w = rng.normal(0, sd, (sims, n_writers, 1))
    pa = np.clip(p + delta + w, 0, 1)
    return (rng.binomial(1, pa) - rng.binomial(1, p)).mean(axis=2)


def solve_delta_writers(target, base, sd):
    """The delta whose mean clipped difference, over the writer spread, equals target."""
    rng = np.random.default_rng(1)
    s = 1 / 0.3 - 1
    p = rng.beta(base * s, (1 - base) * s, 400000)
    w = rng.normal(0, sd, 400000)
    lo, hi = -0.6, 0.6
    for _ in range(40):
        mid = (lo + hi) / 2
        if np.mean(np.clip(p + mid + w, 0, 1) - p) < target:
            lo = mid
        else:
            hi = mid
    got = np.mean(np.clip(p + (lo + hi) / 2 + w, 0, 1) - p)
    if abs(got - target) > 0.002:
        raise ValueError(f"target {target} is not reachable at base {base}, sd {sd} (best {got:.3f})")
    return (lo + hi) / 2


def false_rate(verdict, truth, X):
    """Verdicts that are false against the truth of each simulation."""
    wrong = ((verdict == "gain") & (truth <= X)) | ((verdict == "loss") & (truth >= -X)) \
        | ((verdict == "within") & ((truth <= -X) | (truth >= X)))
    return int(wrong.sum())


def true_success_diff(rng, deltas, base, icc=0.3):
    s = 1 / icc - 1
    p = rng.beta(base * s, (1 - base) * s, 400000)
    return np.mean([np.mean(np.clip(p + d, 0, 1) - p) for d in deltas])


def solve_delta(target, base):
    """The delta whose mean clipped difference equals target, by bisection on a fixed sample."""
    rng = np.random.default_rng(1)
    lo, hi = (0.0, 0.6) if target >= 0 else (-0.6, 0.0)
    for _ in range(40):
        mid = (lo + hi) / 2
        if true_success_diff(rng, [mid], base) < target:
            lo = mid
        else:
            hi = mid
    mid = (lo + hi) / 2
    got = true_success_diff(rng, [mid], base)
    if abs(got - target) > 0.002:
        raise ValueError(f"target {target} is not reachable at base {base} (best {got:.3f})")
    return mid


def summarize(verdict, when, cap, batch_runs=None):
    counts = {k: int((verdict == k).sum()) for k in ("gain", "loss", "within", "open", "empty")}
    b = np.where(verdict == "open", cap + 1, when)  # open counts as "more than cap"
    med, p90 = np.percentile(b, 50), np.percentile(b, 90)
    fmt = lambda v: f">{cap}" if v > cap else f"{v:.0f}"
    line = f"{counts}  batches, all sims: median {fmt(med)}, p90 {fmt(p90)}"
    if batch_runs:
        line += f"  (runs a condition at the median: {fmt(med) if med > cap else int(med * batch_runs)})"
    return line


def sc_success(out, rng):
    """Draft 4 design: both models inside every unit, the plan's task batch, X = 12 points."""
    sims, batch, cap = 200, PLAN_TASKS, 30
    out.write(f"success, both models in each unit, {PLAN_RUNS} runs a model, batch {batch}, X 0.12, {sims} sims, cap {cap}\n")
    for base, targets in ((0.8, [0, 0.04, 0.06, 0.08, 0.10, 0.12, -0.12, -0.22]), (0.6, [0.15, 0.18, 0.21, -0.10])):
        for target in targets:
            d = solve_delta(target, base)
            u = success_units(rng, sims, batch * cap, [d, d], base, PLAN_RUNS)
            out.write(f"base {base} true {target:+.2f}: " + summarize(*run_sequences(u, 0.12, 1, batch), cap) + "\n")


def sc_models_differ(out, rng):
    """W1: models with different effects. In-unit design (draft 4) against batch alternation (draft 3).
    Base 0.6, because at base 0.8 no difference above +0.20 is reachable (the mean of 1 - p)."""
    sims, batch, cap, base = 200, PLAN_TASKS, 30, 0.6
    for ta, tb in ((0.22, 0.02), (0.32, -0.08)):
        da, db = solve_delta(ta, base), solve_delta(tb, base)
        out.write(f"model A {ta:+.2f}, model B {tb:+.2f}, equal-weight mean +0.12 = X; base {base}; {sims} sims, cap {cap}\n")
        u = success_units(rng, sims, batch * cap, [da, db], base, PLAN_RUNS)
        out.write("  in each unit (draft 4): " + summarize(*run_sequences(u, 0.12, 1, batch), cap) + "\n")
        for first in (True, False):
            u = alternate_units(rng, sims, batch * cap, da, db, base, PLAN_RUNS, batch, a_first=first)
            out.write(f"  alternating batches, {'A' if first else 'B'} first (draft 3): "
                      + summarize(*run_sequences(u, 0.12, 1, batch), cap) + "\n")
    out.write("Every verdict at a true mean of exactly X is false; 'within' and 'gain' both count.\n")
    out.write("Check of audit 10.8 case 2 (A +0.22, B +0.02 at base 0.8): +0.22 is not reachable there.\n")


def sc_margins(out, rng):
    """W2: every margin the plan uses, with the batch of its unit."""
    rows = [
        ("success X 5 points", "s", 0.05, [0, 0.025], PLAN_TASKS, 100),
        ("seeds X 10 points (success model, 1 run)", "s1", 0.10, [0, 0.05], 20, 60),
        ("rules X 14 points (success model, 1 run, 28 a batch)", "s1", 0.14, [0, 0.07], 28, 60),
        ("log ratio X ln 1.05, sd 0.32", "n", np.log(1.05), [0, np.log(1.05) / 2], PLAN_TASKS, 300),
        ("log ratio X ln 1.10, sd 0.32", "n", np.log(1.10), [0, np.log(1.10) / 2], PLAN_TASKS, 150),
        ("log ratio X ln 1.15, sd 0.32", "n", np.log(1.15), [0, np.log(1.15) / 2], PLAN_TASKS, 100),
        ("log ratio X ln 1.20, sd 0.32", "n", np.log(1.20), [0, np.log(1.20) / 2], PLAN_TASKS, 60),
        ("log ratio X ln 1.25, sd 0.32", "n", np.log(1.25), [0, np.log(1.25) / 2], PLAN_TASKS, 60),
        # AA1, AA5: seeded agent columns (C3a, C3b, C3c, N5) and the timing batch of 20 (rows 15, 19)
        ("seeded agent draws X 12 points (success model, 1 run a model)", "s1", 0.12, [0, 0.06], 20, 60),
        ("log ratio X ln 1.10, sd 0.32, timing batch", "n", np.log(1.10), [0, np.log(1.10) / 2], 20, 300),
        ("log ratio X ln 1.20, sd 0.32, timing batch", "n", np.log(1.20), [0, np.log(1.20) / 2], 20, 150),
        ("seeded agent draws X 12 points, base 0.5 (success model, 1 run a model)", "s1h", 0.12, [0, 0.06], 20, 60),
        ("success X 5 points, base 0.5", "sh", 0.05, [0, 0.025], PLAN_TASKS, 100),
    ]
    sims = 50
    out.write(f"{sims} sims a row; success rows use base 0.8 unless the row name says base 0.5, both models in each unit, {PLAN_RUNS} runs a model (1 for seeds and rules)\n")
    for name, kind, X, targets, batch, cap in rows:
        for target in targets:
            if kind in ("s", "s1", "s1h", "sh"):
                base = 0.5 if kind in ("s1h", "sh") else 0.8
                d = solve_delta(target, base)
                u = success_units(rng, sims, batch * cap, [d, d], base, PLAN_RUNS if kind in ("s", "sh") else 1)
                R = 1
            else:
                R = np.log(4)
                u = np.clip(rng.normal(target, 0.32, (sims, batch * cap)), -R, R)
            out.write(f"{name}, true {target:+.3f}, batch {batch}, cap {cap}: "
                      + summarize(*run_sequences(u, X, R, batch), cap) + "\n")


def sc_equal_cost(out, rng):
    """W8: runs to a verdict at equal runs a batch: more tasks with fewer runs, or the reverse."""
    sims, cap, base = 200, 40, 0.8
    out.write(f"200 runs a condition a batch (both models); X 0.12; base {base}; {sims} sims; cap {cap}\n")
    for tasks, runs in ((20, 5), (34, 3), (50, 2)):
        for target in (0.0, 0.08, -0.22):
            d = solve_delta(target, base)
            u = success_units(rng, sims, tasks * cap, [d, d], base, runs)
            out.write(f"{tasks} tasks x {runs} runs a model, true {target:+.2f}: "
                      + summarize(*run_sequences(u, 0.12, 1, tasks), cap, batch_runs=tasks * runs * 2) + "\n")


def sc_writers(out, rng):
    """Y1: writers that differ. One fresh writer per batch (draft 4) against a sealed pool (draft 5)."""
    sims, batch, cap, base, X = 200, PLAN_TASKS, 30, 0.6, 0.12
    out.write(f"population difference +0.12 = X; base {base}; batch {batch}; {sims} sims; cap {cap}\n"
              "pool: 20 writers x 20 tasks, drawn with replacement; false is against the pool's own mean\n")
    out.write("writer interval (Z1): 95% t-interval over the 20 writer means, read once where the column closes\n")
    for sd in (0.0, 0.10, 0.15, 0.20):
        d = solve_delta_writers(X, base, sd)
        u, _, _ = writer_units(rng, sims, batch * cap, d, base, PLAN_RUNS, batch, sd, "one_writer")
        v, w = run_sequences(u, X, 1, batch)
        out.write(f"writer sd {sd:.2f}, one writer a batch: false {int((v != 'open').sum())} of {sims}; "
                  + summarize(v, w, cap) + "\n")
        u, truth, wid = writer_units(rng, sims, batch * cap, d, base, PLAN_RUNS, batch, sd, "pool")
        v, w = run_sequences(u, X, 1, batch)
        lo, hi = writer_interval(u, wid, np.where(v == "open", batch * cap, w * batch))
        closed = v != "open"
        holds = (lo <= X) & (X <= hi)
        out.write(f"writer sd {sd:.2f}, sealed pool: false against the pool {false_rate(v, truth, X)} of {sims}; "
                  f"verdicts {int(closed.sum())} (all false against the population); "
                  f"pool mean sd {truth.std():.3f}; " + summarize(v, w, cap) + "\n"
                  f"  writer interval holds the population value: {int(holds.sum())} of {sims}; "
                  f"in the verdicts, {int((holds & closed).sum())} of {int(closed.sum())}; "
                  f"median width {np.median(hi - lo):.3f}\n")
    for sd in (0.10,):
        d = solve_delta_writers(0.0, base, sd)
        u, truth, wid = writer_units(rng, sims, batch * cap, d, base, PLAN_RUNS, batch, sd, "pool")
        v, w = run_sequences(u, X, 1, batch)
        lo, hi = writer_interval(u, wid, np.where(v == "open", batch * cap, w * batch))
        out.write(f"writer sd {sd:.2f}, sealed pool, population difference 0: "
                  f"false against the pool {false_rate(v, truth, X)}; " + summarize(v, w, cap) + "\n"
                  f"  writer interval holds 0: {int(((lo <= 0) & (0 <= hi)).sum())} of {sims}; "
                  f"inside +-X: {int(((lo > -X) & (hi < X)).sum())}; median width {np.median(hi - lo):.3f}\n")


def sc_fixed(out, rng):
    """Z2: columns with a fixed outcome per unit (seeds, rules). The unit is a writer; each writer's
    seeds are evaluated once. The pool is read when its first writers are complete, then after each
    batch of 10 fresh writers. The truth is the mean over all writers; every verdict at truth X is false."""
    sims, base, step = 200, 0.5, 10
    for name, per, X, first, cap in (("seeds", 10, 0.10, 20, 40), ("seeds", 40, 0.10, 20, 40),
                                     ("seeds", 10, 0.05, 20, 120), ("rules", 28, 0.14, 10, 40)):
        out.write(f"{name}: {per} a writer; X {X}; first read at {first} writers, then every {step}; "
                  f"base {base}; {sims} sims; cap {cap} batches of {step}\n")
        n = first + step * cap
        for sd in (0.0, 0.10, 0.20):
            for target in (0.0, X / 2, X):
                d = solve_delta_writers(target, base, sd)
                u = fixed_units(rng, sims, n, per, d, base, sd)
                v, w = run_sequences(u, X, 1, step, first=first)
                extra = f"; false {int((v != 'open').sum())}" if target == X else ""
                out.write(f"  writer sd {sd:.2f}, true {target:+.3f}: " + summarize(v, w, cap) + extra + "\n")
    out.write(f"Batches are counted in steps of {step} writers from the start: writers at a close = {step} x batches.\n")


def sc_fixed_minus(out, rng):
    """AA6: the fixed-outcome design at true -X, the mirror of sc_fixed's +X row."""
    sims, base, step = 200, 0.5, 10
    for name, per, X, first, cap in (("seeds", 10, 0.10, 20, 40), ("seeds", 10, 0.05, 20, 120),
                                     ("rules", 28, 0.14, 10, 40)):
        out.write(f"{name}: {per} a writer; X {X}; true -X; first read at {first} writers; {sims} sims; cap {cap}\n")
        for sd in (0.0, 0.20):
            d = solve_delta_writers(-X, base, sd)
            u = fixed_units(rng, sims, first + step * cap, per, d, base, sd)
            v, w = run_sequences(u, X, 1, step, first=first)
            out.write(f"  writer sd {sd:.2f}: " + summarize(v, w, cap) + f"; false {int((v != 'open').sum())}\n")


def sc_writer_note(out, rng):
    """AA2: rule 5's note over three seeds. The note appears when the writer-level interval does not
    meet the verdict's own condition. At a population difference of exactly +X, every verdict is
    wrong for the population; 'wrong, no note' is the case a reader cannot see."""
    sims, batch, cap, base, X = 200, PLAN_TASKS, 30, 0.6, 0.12
    out.write(f"population difference +X = {X}; base {base}; pool 20 x 20; {sims} sims a seed; cap {cap}\n")
    for sd in (0.10, 0.15, 0.20):
        d = solve_delta_writers(X, base, sd)
        tot_v = tot_note = 0
        for seed in (20261105, 20261109, 20261110):
            r = np.random.default_rng(seed)
            u, truth, wid = writer_units(r, sims, batch * cap, d, base, PLAN_RUNS, batch, sd, "pool")
            v, w = run_sequences(u, X, 1, batch)
            lo, hi = writer_interval(u, wid, np.where(v == "open", batch * cap, w * batch))
            meets = ((v == "gain") & (lo > X)) | ((v == "loss") & (hi < -X)) \
                | ((v == "within") & (lo > -X) & (hi < X))
            closed = v != "open"
            note = closed & ~meets
            tot_v += int(closed.sum())
            tot_note += int(note.sum())
            out.write(f"writer sd {sd:.2f}, seed {seed}: verdicts {int(closed.sum())}; note {int(note.sum())}; "
                      f"wrong with no note {int((closed & meets).sum())} of {sims}\n")
        out.write(f"writer sd {sd:.2f}, three seeds: note on {tot_note} of {tot_v} verdicts; "
                  f"wrong with no note {tot_v - tot_note} of {3 * sims}\n")


def sc_attackers(out, rng):
    """Z4: row 18. A unit is a pair of attacker sessions under each model. Each arm's count of unnamed
    working bypasses is Poisson, clipped to 0 to 5; D is the mean over models of L minus V."""
    sims, batch, cap, X, R = 100, 3, 200, 1.0, 5
    out.write(f"row 18: batch {batch} pairs (12 sessions); X {X}; R {R}; {sims} sims; cap {cap}\n")
    for lam_l, lam_v in ((1.0, 1.0), (1.0, 0.5), (0.5, 0.5), (2.0, 0.5), (2.0, 1.0)):
        cnt = lambda lam: np.minimum(rng.poisson(lam, (sims, batch * cap)), 5)
        u = ((cnt(lam_l) - cnt(lam_v)) + (cnt(lam_l) - cnt(lam_v))) / 2
        out.write(f"L {lam_l}, V {lam_v}: " + summarize(*run_sequences(u, X, R, batch), cap) + "\n")
    out.write("The Poisson means are assumptions; the holes stage of hr3 section 5 replaces them.\n")


def sc_rates(out, rng):
    """Y2: rate columns. Row 6b (false alarms against none, X 2 points) and N6 (merge failures, X 5 points)."""
    sims, batch, cap = 100, 36, 60
    out.write(f"row 6b: unit = minus the mean of two false alarms (one a model); batch {batch}; X 0.02; {sims} sims; cap {cap}\n")
    for rate in (0.0, 0.005, 0.01, 0.02):
        u = -(rng.binomial(1, rate, (sims, batch * cap)) + rng.binomial(1, rate, (sims, batch * cap))) / 2
        out.write(f"false-alarm rate {rate:.3f}: " + summarize(*run_sequences(u, 0.02, 1, batch), cap) + "\n")
    sims, batch, cap = 50, PLAN_TASKS, 100
    out.write(f"N6: merge success model, base 0.9, {PLAN_RUNS} chains a model; batch {batch}; X 0.05; {sims} sims; cap {cap}\n")
    for target in (0.0, 0.025):
        d = solve_delta(target, 0.9)
        u = success_units(rng, sims, batch * cap, [d, d], 0.9, PLAN_RUNS)
        out.write(f"true {target:+.3f}: " + summarize(*run_sequences(u, 0.05, 1, batch), cap) + "\n")


SCENARIOS = {"success": sc_success, "models_differ": sc_models_differ, "margins": sc_margins,
             "equal_cost": sc_equal_cost, "writers": sc_writers, "rates": sc_rates,
             "fixed": sc_fixed, "attackers": sc_attackers, "fixed_minus": sc_fixed_minus,
             "writer_note": sc_writer_note}

if __name__ == "__main__":
    names = sys.argv[1:] or list(SCENARIOS)
    os.makedirs(OUT, exist_ok=True)
    for name in names:
        rng = np.random.default_rng(20261105)
        with open(os.path.join(OUT, f"{name}.txt"), "w") as out:
            SCENARIOS[name](out, rng)
        print(open(os.path.join(OUT, f"{name}.txt")).read())
