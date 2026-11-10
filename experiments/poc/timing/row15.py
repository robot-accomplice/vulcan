#!/usr/bin/env python3
"""Row 15 (fast verification): setup, timed runs, trace runs, and an untimed confirm.

    python row15.py --config commands.json --changes changes.json setup
    python row15.py ... run   --label dry-m1 --seed S --batch 1 --n 20
    python row15.py ... trace --label trace-m1 --all
    python row15.py ... confirm [--audit]

Steps of one run (plan-coverage.md, rule 6): (1) restore the closed list of caches from the snapshot,
(2) drop the file cache, (3) read the program files back, (4) apply the one-file change, (5) time the
check. The change is reverted after the run. Before step 1 of each timed run and trace run, and outside the timed
interval, the load gate of rule 6 samples the machine (common.load_gate); a run that fails it is skipped. Every raw record is sealed and the raw directory is deleted.
"""
import argparse
import json
import os
import re
import subprocess
import sys
import time

import common as C

DROP_TIMEOUT_S = 300
TRACER_STOP_TIMEOUT_S = 300


def state_path(a):
    return os.path.join(a["_ctx"]["armdir"], "baseline.json")


def load_baseline(a):
    with open(state_path(a), encoding="utf-8") as f:
        return json.load(f)


def guard_wipe(cfg, path):
    if not os.path.abspath(path).lower().startswith(os.path.abspath(cfg["work_root"]).lower() + os.sep) or "arms" not in path:
        raise C.Abort("refusing to wipe a path outside the work root")
    C.rmtree(path)


def prepare(cfg, arm, a):
    """A fresh work tree (and a pristine copy for reverts), and empty cache directories."""
    ctx = a["_ctx"]
    guard_wipe(cfg, ctx["armdir"])
    os.makedirs(ctx["armdir"])
    C.copytree(a["app_src"], os.path.join(ctx["armdir"], "pristine"))
    C.copytree(os.path.join(ctx["armdir"], "pristine"), ctx["tree"])
    for e in a["closed_list"]:
        if not e["path"].startswith(ctx["tree"]):
            os.makedirs(e["path"], exist_ok=True)


def change_files(spec):
    return {"files": spec["files"], "pkgs": spec["pkgs"]}


def full_check(cfg, arm, a, setup_env=True):
    """One full check of the unchanged base (all packages, all files). Untimed: no duration is kept."""
    env = C.arm_env(cfg, arm, setup=setup_env)
    out = []
    for tool in a["check"]:
        r = C.run_tool(tool, a["_ctx"], env, a["_ctx"]["tree"], None, "full", cfg["constants"]["tool_timeout_s"])
        r["findings"], r["dropped"] = C.normalize(r["output"], a["_ctx"]["tree"], r["cwd"])
        out.append(r)
    return out


def setup_arm(cfg, arm, changes, faults):
    a = C.arm_cfg(cfg, arm)
    ctx = a["_ctx"]
    prepare(cfg, arm, a)
    for r in full_check(cfg, arm, a):
        if r["status"] in (C.STATUS_LAUNCH, C.STATUS_TIMEOUT):
            faults.add("%s tool-launch %s" % (arm, r["name"]))
    C.snapshot(a)
    env = C.arm_env(cfg, arm)
    base = {}
    for ch in changes:
        spec = ch["arms"].get(arm)
        if not spec:
            continue
        if not C.git_apply(ctx["tree"], spec["patch"], check=True):
            faults.add("%s patch-does-not-apply" % arm)
        for tool in a["check"]:
            cmd, _ = C.tool_command(tool, ctx, ctx["tree"], change_files(spec), "change")
            key = C.rendered_key(tool, cmd)
            if key in base:
                continue
            r = C.run_tool(tool, ctx, env, ctx["tree"], change_files(spec), "change", cfg["constants"]["tool_timeout_s"])
            f, d = C.normalize(r["output"], ctx["tree"], r["cwd"])
            base[key] = {"status": r["status"], "findings": f, "dropped": d}
    C.restore(a)
    with open(state_path(a), "w", encoding="utf-8", newline="\n") as fh:
        json.dump(base, fh, indent=1, sort_keys=True)


def run_drop(cfg):
    cmd = C.subst(cfg["drop"]["cmd"], {"here": C.HERE, "repo": cfg.get("repo", "")})
    try:
        p = subprocess.run(cmd, capture_output=True, timeout=DROP_TIMEOUT_S)
    except (OSError, subprocess.TimeoutExpired):
        raise C.Abort("drop command could not run")
    if p.returncode != 0:
        raise C.Abort("drop command failed")


def revert(a, spec, pristine_hash, raw, arm):
    ctx = a["_ctx"]
    ok = C.git_apply(ctx["tree"], spec["patch"], reverse=True)
    outputs = [e["path"] for e in a["closed_list"] if e["path"].startswith(ctx["tree"])]
    if not ok or C.tree_hash(ctx["tree"], outputs) != pristine_hash:
        raw.fault(arm, "revert-mismatch")
        C.rmtree(ctx["tree"])
        C.copytree(os.path.join(ctx["armdir"], "pristine"), ctx["tree"])


def prepare_run(cfg, a, spec, raw, arm, steps):
    """Steps 1 to 4. Returns False when the change does not apply."""
    t = time.perf_counter_ns()
    C.restore(a)
    steps["restore"] = time.perf_counter_ns() - t
    t = time.perf_counter_ns()
    run_drop(cfg)
    steps["drop"] = time.perf_counter_ns() - t
    t = time.perf_counter_ns()
    steps["program_files"] = C.read_program_files(a["program_paths"])
    steps["program_read"] = time.perf_counter_ns() - t
    if steps["program_files"][2]:
        raw.fault(arm, "program-file-unreadable")
    t = time.perf_counter_ns()
    ok = C.git_apply(a["_ctx"]["tree"], spec["patch"])
    steps["apply"] = time.perf_counter_ns() - t
    if not ok:
        raw.fault(arm, "patch-does-not-apply")
    return ok


def timed_run(cfg, arm, a, base, ch, spec, raw, draw, pos, pristine_hash):
    ctx = a["_ctx"]
    env = C.arm_env(cfg, arm)
    idx = raw.next_index()
    head = {"index": idx, "draw": draw, "position_in_draw": pos, "arm": arm, "machine": cfg.get("machine"), "change": ch["id"]}
    gate = C.load_gate(cfg, raw, arm)
    if not gate["passed"]:
        # Skipped: no step and no tool ran. The slot still counts in `runs`.
        raw.json("runs/%05d.json" % idx, dict(head, load_gate=gate, skipped="load-gate", tools=[]))
        return
    steps = {}
    t0 = time.perf_counter_ns()
    applied = prepare_run(cfg, a, spec, raw, arm, steps)
    tools = []
    if applied:
        t = time.perf_counter_ns()
        for tool in a["check"]:
            tools.append(C.run_tool(tool, ctx, env, ctx["tree"], change_files(spec), "change", cfg["constants"]["tool_timeout_s"]))
        steps["check"] = time.perf_counter_ns() - t
    wall = time.perf_counter_ns() - t0
    raw.wall_ns += wall
    if applied:
        revert(a, spec, pristine_hash, raw, arm)
    recs = []
    for r in tools:
        f, d = C.normalize(r["output"], ctx["tree"], r["cwd"])
        b = base.get(C.rendered_key(next(t for t in a["check"] if t["name"] == r["name"]), r["cmd"]))
        if b is None:
            raise C.Abort("no baseline for a command of " + arm)
        clean = r["status"] == b["status"] and f == b["findings"]
        if r["status"] in (C.STATUS_LAUNCH, C.STATUS_TIMEOUT):
            raw.fault(arm, "tool-launch-or-timeout", "tool " + r["name"])
        elif not clean:
            raw.fault(arm, "not-clean", "tool " + r["name"])
        rel = "out/%05d.%s.txt" % (idx, r["name"])
        raw.write(rel, r["output"])
        recs.append({"name": r["name"], "unit": r["unit"], "cmd": r["cmd"], "status": r["status"], "ns": r["ns"], "clean": clean,
                     "findings": len(f), "dropped": d, "base_status": b["status"], "output": rel})
    raw.json("runs/%05d.json" % idx, dict(head, load_gate=gate, steps_ns=steps, wall_ns=wall, tools=recs))


def cmd_run(cfg, args, changes):
    arms = args.arms or C.arm_names(cfg)
    ids = args.only.split(",") if args.only else [c["id"] for c in changes]
    byid = {c["id"]: c for c in changes}
    if not ids:
        raise C.Abort("no changes file: write changes.json (see changes.example.json)")
    plan = C.make_plan(args.seed, args.batch, ids, arms, args.n)
    return execute(cfg, args, byid, arms, plan, trace=False)


def cmd_trace(cfg, args, changes):
    arms = args.arms or C.arm_names(cfg)
    byid = {c["id"]: c for c in changes}
    if args.all:
        plan = [(c["id"], C.make_plan(args.seed, "trace-" + c["id"], [c["id"]], arms, 1)[0][1]) for c in changes]
    else:
        ids = args.only.split(",") if args.only else list(byid)
        plan = C.make_plan(args.seed, args.batch, ids, arms, args.n)
    return execute(cfg, args, byid, arms, plan, trace=True)


def tracer_cmd(tcfg, key, ctx):
    return C.subst(tcfg[key], ctx)


def traced_run(cfg, arm, a, ch, spec, raw, draw, pos, pristine_hash):
    """A trace run: steps 1 to 4, then the check under the tracer in place of step 5. Untimed in the record,
    but its wall time is the trace total."""
    ctx = a["_ctx"]
    tcfg = cfg["tracer"]
    env = C.arm_env(cfg, arm)
    idx = raw.next_index()
    head = {"index": idx, "draw": draw, "position_in_draw": pos, "arm": arm, "machine": cfg.get("machine"), "change": ch["id"]}
    gate = C.load_gate(cfg, raw, arm)
    if not gate["passed"]:
        raw.json("trace/%05d.json" % idx, dict(head, load_gate=gate, skipped="load-gate", counts=[]))
        return
    tdir = os.path.join(cfg["work_root"], "trace_tmp")
    os.makedirs(tdir, exist_ok=True)
    tctx = {"log": os.path.join(tdir, "t%05d.pml" % idx), "csv": os.path.join(tdir, "t%05d.csv" % idx), "here": C.HERE}
    env.update({k: C.subst(v, tctx) for k, v in tcfg.get("env", {}).items()})
    steps = {}
    t0 = time.perf_counter_ns()
    applied = prepare_run(cfg, a, spec, raw, arm, steps)
    pids = []
    if applied:
        try:
            tp = subprocess.Popen(C.subst(tcfg["start"], tctx), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        except OSError:
            raise C.Abort("tracer could not start")
        time.sleep(tcfg.get("wait_s", 0))
        if tp.poll() not in (None, 0):
            raise C.Abort("tracer start failed")
        for tool in a["check"]:
            cmd, cwd = C.tool_command(tool, ctx, ctx["tree"], change_files(spec), "change")
            p = subprocess.Popen(cmd, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
            pids.append((tool["name"], p.pid))
            p.communicate(timeout=cfg["constants"]["tool_timeout_s"])
        if subprocess.run(C.subst(tcfg["stop"], tctx), capture_output=True).returncode != 0:
            raise C.Abort("tracer stop failed")
        try:
            tp.wait(timeout=TRACER_STOP_TIMEOUT_S)
        except subprocess.TimeoutExpired:
            raise C.Abort("tracer did not stop")
        if subprocess.run(C.subst(tcfg["export"], tctx), capture_output=True).returncode != 0:
            raise C.Abort("tracer export failed")
    wall = time.perf_counter_ns() - t0
    raw.wall_ns += wall
    counts = []
    if applied:
        cache_dirs = [e["path"] for e in a["closed_list"]]
        for name, pid in pids:
            src, cache = C.parse_trace_csv(tctx["csv"], ctx["tree"], pid, cache_dirs)
            if src == 0:
                raw.fault(arm, "no-file-count", "tool " + name)
            counts.append({"tool": name, "files_opened": src, "cache_files_opened": cache})
        for k in ("log", "csv"):
            if os.path.exists(tctx[k]):
                os.remove(tctx[k])
        revert(a, spec, pristine_hash, raw, arm)
    raw.json("trace/%05d.json" % idx, dict(head, load_gate=gate, wall_ns=wall, counts=counts))


def execute(cfg, args, byid, arms, plan, trace):
    if not byid:
        raise C.Abort("no changes file: write changes.json (see changes.example.json)")
    C.preflight_seal(cfg)
    arm_cfgs = {arm: C.arm_cfg(cfg, arm) for arm in arms}
    bases = {} if trace else {arm: load_baseline(arm_cfgs[arm]) for arm in arms}
    raw = C.Raw(cfg, args.label)
    status = 0
    try:
        raw.json("plan.json", {"seed": args.seed, "batch": args.batch, "plan": plan, "arms": arms,
                               "closed_list": {arm: [e["name"] for e in arm_cfgs[arm]["closed_list"]] for arm in arms}})
        pristine = {arm: C.tree_hash(os.path.join(arm_cfgs[arm]["_ctx"]["armdir"], "pristine")) for arm in arms}
        # A run that aborted between the change and its revert leaves the change in the tree. Reset such a tree first.
        for arm in arms:
            a = arm_cfgs[arm]
            outputs = [e["path"] for e in a["closed_list"] if e["path"].startswith(a["_ctx"]["tree"])]
            if C.tree_hash(a["_ctx"]["tree"], outputs) != pristine[arm]:
                raw.fault(arm, "tree-not-pristine-at-start")
                C.rmtree(a["_ctx"]["tree"])
                C.copytree(os.path.join(a["_ctx"]["armdir"], "pristine"), a["_ctx"]["tree"])
        for draw, (cid, order) in enumerate(plan):
            ch = byid[cid]
            for pos, arm in enumerate(order):
                spec = ch["arms"][arm]
                if trace:
                    traced_run(cfg, arm, arm_cfgs[arm], ch, spec, raw, draw, pos, pristine[arm])
                else:
                    timed_run(cfg, arm, arm_cfgs[arm], bases[arm], ch, spec, raw, draw, pos, pristine[arm])
    except C.Abort as e:
        raw.fault("-", "abort", re.sub(r"[^A-Za-z0-9_ ./:=-]", "_", str(e))[:80])
        status = 1
    finally:
        raw.seal({"kind": "row15-trace" if trace else "row15", "machine": cfg.get("machine"), "seed": args.seed,
                  "batch": args.batch, "arms": arms, "wall_ns_total": raw.wall_ns})
    C.report(raw, "trace_total_wall_s" if trace else "row15_total_wall_s")
    return status


def cmd_setup(cfg, args, changes):
    arms = args.arms or C.arm_names(cfg)
    faults = set()
    for arm in arms:
        setup_arm(cfg, arm, changes, faults)
    C.emit(["fault: " + f for f in sorted(faults)] + ["setup: done"])
    return 1 if faults else 0


def cmd_confirm(cfg, args, changes):
    """One untimed full check of the unchanged base in each arm. Prints the exit status and the count of
    findings of each tool, and never a duration."""
    for arm in args.arms or C.arm_names(cfg):
        a = C.arm_cfg(cfg, arm)
        prepare(cfg, arm, a)
        start = time.time()
        for r in full_check(cfg, arm, a):
            print("%s %s unit=%s status=%d findings=%d dropped=%d" % (arm, r["name"], r["unit"], r["status"], len(r["findings"]), r["dropped"]))
        if args.audit:
            for root in cfg.get("audit_roots", []):
                root = os.path.expandvars(root)
                for d, _, fs in os.walk(root):
                    for f in fs:
                        p = os.path.join(d, f)
                        try:
                            if os.path.getmtime(p) >= start - 1:
                                print("%s audit-path-written-outside-closed-list %s" % (arm, p))
                        except OSError:
                            pass
    return 0


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default=os.path.join(C.HERE, "commands.json"))
    ap.add_argument("--changes", default=os.path.join(C.HERE, "changes.json"))
    ap.add_argument("--work", help="override work_root")
    ap.add_argument("--arms", type=lambda s: s.split(","))
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("setup")
    c = sub.add_parser("confirm")
    c.add_argument("--audit", action="store_true")
    for name in ("run", "trace"):
        r = sub.add_parser(name)
        r.add_argument("--label", required=True)
        r.add_argument("--seed", required=True)
        r.add_argument("--batch", default="1")
        r.add_argument("--n", type=int, default=20)
        r.add_argument("--only", help="comma-separated change ids to draw from")
        if name == "trace":
            r.add_argument("--all", action="store_true", help="every change once in every arm")
    args = ap.parse_args()
    cfg = C.load_config(args.config)
    if args.work:
        cfg["work_root"] = args.work
    changes = C.load_changes(args.changes) if os.path.exists(args.changes) else []
    try:
        return {"setup": cmd_setup, "run": cmd_run, "trace": cmd_trace, "confirm": cmd_confirm}[args.cmd](cfg, args, changes)
    except C.Abort as e:
        print("abort:", re.sub(r"[^A-Za-z0-9_ ./:=-]", "_", str(e))[:120], file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
