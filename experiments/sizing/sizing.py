"""Pre-recruiting sizing check for decision 6 (audits 8 and 9).

Estimates, by simulation, the probability that each gate passes under fixed scenarios, for the
designs in decision 6's search space. It is a feasibility check run before recruiting, not the
frozen simulation that the analyst writes later.

Model:
- Tasks: half are tempting in milestone 4 (three quarters in milestone 0.5); every tempting task
  targets the tested set (binding rule).
- Violations: a task has a heterogeneity factor g ~ Gamma(mean 1, CV), shared by both arms. A run
  has Poisson(rate * g) violations of the targeted rules. Ordinary tasks have none.
- Success: a task has base success p ~ Beta(mean 0.7, sd about 0.15), shared, plus a per-arm task
  offset N(0, INTER_S) (task x arm interaction). A run is a **clean success** when it succeeds and,
  on a tempting task, has no violation of a targeted rule (decision 6, N0 and N1).
- Working time: a task has a log-median mu ~ N(0, 0.5), plus a per-arm task offset
  N(0, INTER_T); a run adds N(0, SIGMA_RUN). A run that is not a clean success counts as the
  budget, 3 times the task's median.
- Coupled cost (scenario L+coupled): on a tempting task, arm V is tempted as often as arm R
  violates (probability 1 - exp(-rate_R * g) per run); a tempted V run is COUPLED_T times slower
  and fails with extra probability COUPLED_F.
- Net costs (scenarios L+half and L+cost): V's per-run probability of a clean success is set
  that many points below R's on the same task, and V's clean runs are that much slower. A cost is
  thus stated in the quantity that N0 and N1 measure, after R's rule-breaking runs are counted as
  failures.
- Raw costs (scenario L+raw): V's base success is that many points lower and its runs that much
  slower on every task, before R's rule-breaking runs are counted.
- Ordinary-task check (N6): on the untargeted tasks alone, the point estimates of V's success gap
  and of the ratio of mean working times are within the margins.
- Cost pilot: 8 ordinary tasks, 5 runs per arm, the same point estimates. After a failed first
  try and a named change, a second try runs on 8 new tasks, and passes only if the point estimates
  over both tries together (16 tasks) meet the margins. The simulation assumes that the change has
  no effect, which is the case the rule must not reward.
- Escaped defects: Poisson(0.1) holdout failures per accepted change, in every arm; N2 compares
  per-task means over tasks where both arms have an accepted change.
- False positives: 2% of V's runs.
- Tests: exact sign-flip over informative tasks (all 2^n patterns up to n = 16, else 20,000
  random patterns), alpha 0.05, the half-rate condition, at least 5 informative tasks and 10
  events. N0 to N3 use one-sided 95% t-bounds over tasks, as decision 6, section 8 defines them.
- Agreement floor of the blind graders (a rule considered and dropped): each grader marks a true violation with probability q and
  a non-violation with probability 0.005; the second grader re-judges every pair the first marked
  and 30% of the rest; agreement = both / (both + first only + second only / 0.3), checked only
  when the estimated number of marked pairs is at least 20. The table gives P(agreement < 0.8).
- Not modelled (each would lower P(Continue) somewhat): reviewer and blind-grader disagreement and
  the dual-review and dual-grader conditions of Q2, N0, N1, and N3; N1 without the reviewer's
  minutes; N4 and N5 (assumed to pass); the four-rule minimum (assumed met).

Usage: python -I -B sizing.py   (writes results.md next to this file)
"""
import itertools
import os

import numpy as np
from scipy.stats import t as student_t

RNG = np.random.default_rng(20261021)
CV = 1.0
SIGMA_RUN = 0.5
INTER_S = 0.05
INTER_T = 0.10
COUPLED_T = 1.3
COUPLED_F = 0.05
MARGIN_S = 0.12
MARGIN_T = 1.15
SIGNS = {}


def sign_matrix(n):
    if n not in SIGNS:
        if n <= 16:
            SIGNS[n] = np.array(list(itertools.product((-1.0, 1.0), repeat=n)))
        else:
            SIGNS[n] = RNG.choice((-1.0, 1.0), size=(20000, n))
    return SIGNS[n]


def signflip_p(d):
    d = d[d != 0]
    n = len(d)
    if n < 5:
        return None
    return float(((sign_matrix(n) * np.abs(d)).sum(axis=1) >= d.sum() - 1e-12).mean())


def p1_pass(better, worse):
    """better, worse: (tasks, runs) violation counts on targeted tasks."""
    if better.sum() + worse.sum() < 10:
        return False
    rb, rw = better.mean(axis=1), worse.mean(axis=1)
    p = signflip_p(rw - rb)
    return p is not None and p < 0.05 and rb.mean() <= 0.5 * rw.mean()


def tbound(x):
    x = np.asarray(x, dtype=float)
    n = len(x)
    if n < 2:
        return np.inf
    return x.mean() + student_t.ppf(0.95, n - 1) * x.std(ddof=1) / np.sqrt(n)


def gamma_factor(n):
    k = 1.0 / CV ** 2
    return RNG.gamma(k, 1.0 / k, size=n)


def milestone4(tasks, runs, tempting, sc):
    rate_r, rate_v, cost_s, cost_t, coupled, inter_s, inter_t = sc[:7]
    raw = len(sc) > 7 and sc[7]
    g = np.concatenate([gamma_factor(tempting), np.zeros(tasks - tempting)])
    tempt = np.arange(tasks) < tempting
    vr = RNG.poisson(np.outer(rate_r * g, np.ones(runs)))
    vv = RNG.poisson(np.outer(rate_v * g, np.ones(runs)))
    if not p1_pass(vv[tempt], vr[tempt]):                                     # Q2
        return False
    base = RNG.beta(4.9, 2.1, size=tasks)
    pr = np.clip(base + RNG.normal(0, inter_s, tasks), 0, 1)
    pv = np.clip(base + RNG.normal(0, inter_s, tasks), 0, 1)
    if cost_s and raw:
        pv = np.clip(pv - cost_s, 0, 1)
    elif cost_s:
        # Net cost: V's clean-success probability is cost_s below R's clean-success probability.
        zr, zv = np.exp(-rate_r * g), np.exp(-rate_v * g)
        pv = np.clip((pr * zr - cost_s) / zv, 0, 1)
    sr = RNG.random((tasks, runs)) < pr[:, None]
    sv = RNG.random((tasks, runs)) < pv[:, None]
    mu = RNG.normal(0, 0.5, size=tasks)
    tr = np.exp((mu + RNG.normal(0, inter_t, tasks))[:, None] + RNG.normal(0, SIGMA_RUN, (tasks, runs)))
    tv = np.exp((mu + RNG.normal(0, inter_t, tasks))[:, None] + RNG.normal(0, SIGMA_RUN, (tasks, runs))
                + np.log(1 + cost_t))
    if coupled:
        tempted = RNG.random((tasks, runs)) < (1 - np.exp(-rate_r * g))[:, None]
        tv = np.where(tempted, tv * COUPLED_T, tv)
        sv = sv & ~(tempted & (RNG.random((tasks, runs)) < COUPLED_F))
    clean_r = sr & (vr == 0)
    clean_v = sv & (vv == 0)
    if tbound(clean_r.mean(axis=1) - clean_v.mean(axis=1)) > MARGIN_S:           # N0
        return False
    budget = np.exp(mu + np.log(3.0))[:, None]
    tr = np.where(clean_r, tr, budget)
    tv = np.where(clean_v, tv, budget)
    if tbound(np.log(tv.mean(axis=1) / tr.mean(axis=1))) > np.log(MARGIN_T):     # N1
        return False
    kr, kv = sr.sum(axis=1), sv.sum(axis=1)                                       # N2
    both = (kr > 0) & (kv > 0)
    dr = np.array([RNG.poisson(0.1, k).mean() if k else 0.0 for k in kr])
    dv = np.array([RNG.poisson(0.1, k).mean() if k else 0.0 for k in kv])
    if tbound((dv - dr)[both]) > 0.1:
        return False
    if tbound((RNG.random((tasks, runs)) < 0.02).mean(axis=1)) > 0.1:            # N3
        return False
    o = ~tempt                                                                    # N6
    if o.sum() and not ordinary_ok(clean_r[o], clean_v[o], tr[o], tv[o]):
        return False
    return True


def ordinary_ok(sr, sv, tr, tv):
    """Point estimates on ordinary tasks: success gap and ratio of mean times within the margins."""
    if sr.mean() - sv.mean() > MARGIN_S:
        return False
    return np.exp(np.mean(np.log(tv.mean(axis=1) / tr.mean(axis=1)))) <= MARGIN_T


def pilot_try(cost_s, cost_t, tasks, runs):
    base = RNG.beta(4.9, 2.1, size=tasks)
    pr = np.clip(base + RNG.normal(0, INTER_S, tasks), 0, 1)
    pv = np.clip(base - cost_s + RNG.normal(0, INTER_S, tasks), 0, 1)
    sr = RNG.random((tasks, runs)) < pr[:, None]
    sv = RNG.random((tasks, runs)) < pv[:, None]
    mu = RNG.normal(0, 0.5, size=tasks)
    tr = np.exp((mu + RNG.normal(0, INTER_T, tasks))[:, None] + RNG.normal(0, SIGMA_RUN, (tasks, runs)))
    tv = np.exp((mu + RNG.normal(0, INTER_T, tasks))[:, None] + RNG.normal(0, SIGMA_RUN, (tasks, runs))
                + np.log(1 + cost_t))
    budget = np.exp(mu + np.log(3.0))[:, None]
    return sr, sv, np.where(sr, tr, budget), np.where(sv, tv, budget)


def pilot_pass(cost_s, cost_t, tasks=8, runs=5, tries=2):
    """One try, or a failed first try followed by a second try judged on both tries pooled."""
    first = pilot_try(cost_s, cost_t, tasks, runs)
    if ordinary_ok(*first) or tries == 1:
        return ordinary_ok(*first)
    second = pilot_try(cost_s, cost_t, tasks, runs)
    pooled = [np.concatenate((a, b)) for a, b in zip(first, second)]
    return ordinary_ok(*pooled)


def m4_power(tasks, runs, tempting, sc, sims):
    return float(np.mean([milestone4(tasks, runs, tempting, sc) for _ in range(sims)]))


def m05_q1(targeted, runs, sims):
    ok = 0
    for _ in range(sims):
        g = gamma_factor(targeted)
        ok += p1_pass(RNG.poisson(np.outer(0.05 * g, np.ones(runs))),
                      RNG.poisson(np.outer(0.4 * g, np.ones(runs))))
    return ok / sims


def m05_gate2_rate(targeted, runs, true_rate, sims):
    hits = 0
    for _ in range(sims):
        g = gamma_factor(targeted)
        hits += RNG.poisson(np.outer(true_rate * g, np.ones(runs))).mean() >= 0.2
    return hits / sims


def agreement_trip(true_pos, negatives, q, sims, fp=0.005, share=0.3):
    trips = 0
    for _ in range(sims):
        f_pos = RNG.random(true_pos) < q
        s_pos = RNG.random(true_pos) < q
        f_neg = RNG.random(negatives) < fp
        s_neg = RNG.random(negatives) < fp
        # Re-judged: every pair the first marked, and a 30% sample of the pairs it did not mark.
        sampled_pos = ~f_pos & (RNG.random(true_pos) < share)
        sampled_neg = ~f_neg & (RNG.random(negatives) < share)
        both = (f_pos & s_pos).sum() + (f_neg & s_neg).sum()
        first_only = (f_pos & ~s_pos).sum() + (f_neg & ~s_neg).sum()
        second_only = (sampled_pos & s_pos).sum() + (sampled_neg & s_neg).sum()
        marked = both + first_only + second_only / share
        if marked >= 20 and both / marked < 0.8:
            trips += 1
    return trips / sims


def scenarios(r):
    # (rate_R, rate_V, cost_success, cost_time, coupled, interaction_s, interaction_t)
    return {
        "Z": (r, r, 0, 0, False, INTER_S, INTER_T),
        "L": (r, r / 8, 0, 0, False, INTER_S, INTER_T),
        "L+coupled": (r, r / 8, 0, 0, True, INTER_S, INTER_T),
        "L+half": (r, r / 8, 0.06, 0.07, False, INTER_S, INTER_T),
        "L+inter": (r, r / 8, 0, 0, False, 0.10, 0.20),
        "L+cost": (r, r / 8, 0.15, 0.20, False, INTER_S, INTER_T),
        "L+raw": (r, r / 8, 0.12, 0.15, False, INTER_S, INTER_T, True),
    }


def main():
    out = ["# Pre-recruiting sizing check", "",
           "Generated by `sizing.py` (seed 20261021). Model and its limits: see the docstring of "
           "`sizing.py`. Success for N0 and N1 is a clean success: an accepted change that keeps "
           "its targeted rules. Margins 12 points and 1.15.", ""]
    sims = 1000
    out += ["## Milestone 0.5", "", "P(Q1 passes | L), every tempting task targeting S (%d "
            "simulations per cell):" % sims, "", "| Targeted tasks | 5 runs | 10 runs |", "|---|---|---|"]
    for k in (8, 12, 16):
        out.append("| %d | %.2f | %.2f |" % (k, m05_q1(k, 5, sims), m05_q1(k, 10, sims)))
    out += ["", "Gate 2's rate test, P(estimate of R's U* rate >= 0.2), at 5 runs (%d simulations "
            "per cell):" % sims, "", "| Tasks targeting U* | true 0.1 | true 0.2 | true 0.4 |",
            "|---|---|---|---|"]
    for k in (4, 6, 8):
        out.append("| %d | %.2f | %.2f | %.2f |" % (k, *(m05_gate2_rate(k, 5, x, sims) for x in (0.1, 0.2, 0.4))))
    out += ["", "## Cost pilot (end of milestone 2)", "", "8 ordinary tasks, 5 runs per arm, point "
            "estimates against the margins (12 points, 1.15); a second try is judged on both tries pooled. "
            "P(pass), 3,000 simulations per cell:",
            "", "| V's raw cost (success, time) | One try | With the pooled second try |", "|---|---|---|"]
    for c, tt in ((0, 0), (0.06, 0.07), (0.12, 0.15), (0.20, 0.30)):
        one = np.mean([pilot_pass(c, tt, tries=1) for _ in range(3000)])
        two = np.mean([pilot_pass(c, tt, tries=2) for _ in range(3000)])
        out.append("| %d points, %d%% | %.2f | %.2f |" % (round(c * 100), round(tt * 100), one, two))
    out += ["", "## Agreement floor of the blind graders (considered and dropped)", "",
            "P(weighted positive agreement < 0.8) for two honest graders, with 60 true judged "
            "violations among 2,000 judged pairs and a false-positive rate of 0.005 (2,000 "
            "simulations per cell). A fixed floor trips almost always, so decision 6 re-judges "
            "every decisive pair instead:", "",
            "| Each grader's detection rate q | P(floor trips) |", "|---|---|"]
    for q in (0.95, 0.90, 0.85, 0.80, 0.70):
        out.append("| %.2f | %.2f |" % (q, agreement_trip(60, 2000, q, 2000)))
    names = ("Z", "L", "L+coupled", "L+inter", "L+half", "L+cost", "L+raw")
    sims = 600
    out += ["", "## Milestone 4: P(Continue) by scenario (%d simulations per cell)" % sims, "",
            "Half of the tasks are tempting. Scenarios: Z no effect; L large effect (V's rate one eighth "
            "of R's); L+coupled, tempted V runs 1.3 times slower and 5 points more failures; L+inter, "
            "task x arm interaction sd 0.10 (success) and 0.20 (log time); L+half, a net cost of half "
            "the margins (6 points, 7%); L+cost, a net cost of 15 points and 20%; L+raw, a raw cost "
            "equal to the margins (12 points, 15%) on every task. Net costs are stated against R's "
            "clean success. Continue includes the ordinary-task check N6.", "",
            "| Tasks | Runs | r | " + " | ".join(names) + " |", "|---|---|---|" + "---|" * len(names)]
    for tasks in (28, 32, 36):
        for runs in (8, 10):
            for r in (0.4, 0.2):
                sc = scenarios(r)
                row = [m4_power(tasks, runs, tasks // 2, sc[n], sims) for n in names]
                out.append("| %d | %d | %.1f | " % (tasks, runs, r) + " | ".join("%.2f" % x for x in row) + " |")
    path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "results.md")
    open(path, "w", encoding="utf-8", newline="\n").write("\n".join(out) + "\n")
    print("\n".join(out))


if __name__ == "__main__":
    main()
