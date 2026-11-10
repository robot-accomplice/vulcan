#!/usr/bin/env python3
"""Row 19 (performance like Go): build each arm's app, then time one request path in steady state.

    python row19.py --config commands.json --requests requests.json setup
    python row19.py ... run --label dry19-m1 --seed S --batch 1 --n 20

One run (plan-coverage.md, rule 6, "The warm-up of row 19"): start a fresh process, send warm-up
requests in blocks of 50; from block 4 the warm-up ends at the first block where the median of the
last two blocks is within 5% of the median of the two blocks before them; a path not settled after
12 blocks stops there and is flagged. Then time the next 100 requests. The run's time is their median.
The residual fall is 100 * ln(median of the 100 requests before the timed ones / median of the timed 100).
"""
import argparse
import http.client
import json
import math
import os
import re
import socket
import statistics
import subprocess
import sys
import time

import common as C

READY_POLL_S = 0.005
READY_TIMEOUT_S = 60
EXIT_WAIT_S = 30


def warm_settled(samples, nb, w):
    """The rule of the plan, the 2v2 rule of experiments/poc/sim/warmup-sim.py: after nb blocks, from block
    `from_block`, the last two blocks against the two before them."""
    if nb < w["from_block"]:
        return False
    k = w["block"]
    last = statistics.median(samples[(nb - 2) * k: nb * k])
    prev = statistics.median(samples[(nb - 4) * k: (nb - 2) * k])
    return abs(last / prev - 1) <= w["band"]


def steady_state(send, w):
    """Drive `send` (one timed request, returns ns) by the rule. Returns the record of the measurement."""
    k = w["block"]
    samples, stop, unsettled = [], None, False
    for nb in range(1, w["cap_blocks"] + 1):
        samples.extend(send() for _ in range(k))
        if warm_settled(samples, nb, w):
            stop = nb
            break
    if stop is None:
        stop, unsettled = w["cap_blocks"], True
    timed = [send() for _ in range(w["timed"])]
    before = samples[stop * k - w["timed"]: stop * k]
    med_t, med_b = statistics.median(timed), statistics.median(before)
    return {"warm_blocks": stop, "unsettled": unsettled, "median_ns": med_t,
            "residual_fall_pct": 100 * math.log(med_b / med_t), "samples_ns": samples + timed}


def wait_port(port, timeout, proc=None):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        if proc is not None and proc.poll() is not None:
            return False
        try:
            socket.create_connection(("127.0.0.1", port), timeout=1).close()
            return True
        except OSError:
            time.sleep(READY_POLL_S)
    return False


def stop(proc):
    if proc and proc.poll() is None:
        proc.kill()
    if proc:
        try:
            proc.wait(timeout=EXIT_WAIT_S)
        except subprocess.TimeoutExpired:
            pass


class Fault(Exception):
    pass


def requester(conn, req, ctx):
    body = json.dumps(req["body"]).encode() if "body" in req else None
    hdr = {"Content-Type": "application/json"} if body else {}
    path = C.subst(req["path"], ctx)

    def send():
        t = time.perf_counter_ns()
        conn.request(req["method"], path, body=body, headers=hdr)
        r = conn.getresponse()
        r.read()
        dt = time.perf_counter_ns() - t
        if r.status != req.get("expect", 200):
            raise Fault("bad-status")
        return dt
    return send


def setup_arm(cfg, arm, faults):
    a = C.arm_cfg(cfg, arm)
    r = a["row19"]
    ctx = a["_ctx"]
    d = os.path.join(ctx["armdir"], "row19")
    C.rmtree(d)
    os.makedirs(d)
    C.copytree(r["app_src"], os.path.join(d, "app"))
    status, out, _ = C.run_cmd(C.expand(r["build"]["cmd"], ctx), C.subst(r["build"]["cwd"], ctx), C.arm_env(cfg, arm, setup=True), cfg["constants"]["tool_timeout_s"])
    if status != 0:
        faults.add("%s build-failed" % arm)


def timed_run(cfg, arm, a, req, raw, draw, pos):
    r, ctx = a["row19"], a["_ctx"]
    k = cfg["constants"]
    ctx = dict(ctx, port=k["port"], pay_port=k["pay_port"], db=os.path.join(ctx["armdir"], "row19", "run.db"))
    if os.path.exists(ctx["db"]):
        os.remove(ctx["db"])
    idx = raw.next_index()
    helper = app = None
    rec = {"index": idx, "draw": draw, "position_in_draw": pos, "arm": arm, "machine": cfg.get("machine"), "path": req["id"]}
    rec["load_gate"] = C.load_gate(cfg, raw, arm)
    if not rec["load_gate"]["passed"]:
        # Skipped: no helper and no app started. The slot still counts in `runs`.
        rec["skipped"] = "load-gate"
        raw.json("runs/%05d.json" % idx, rec)
        return
    try:
        if r.get("helper"):
            helper = subprocess.Popen(C.expand(r["helper"], ctx), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if not wait_port(k["pay_port"], READY_TIMEOUT_S, helper):
                raise Fault("helper-not-ready")
        t0 = time.perf_counter_ns()
        app = subprocess.Popen(C.expand(r["run"], ctx), cwd=os.path.join(ctx["armdir"], "row19", "app"),
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if not wait_port(k["port"], READY_TIMEOUT_S, app):
            raise Fault("app-not-ready")
        conn = http.client.HTTPConnection("127.0.0.1", k["port"], timeout=60)
        for s in req.get("setup", []):
            requester(conn, s, ctx)()
        res = steady_state(requester(conn, req["request"], ctx), cfg["warmup"])
        wall = time.perf_counter_ns() - t0
        raw.wall_ns += wall
        rec.update(res, wall_ns=wall)
        conn.close()
    except (Fault, OSError, http.client.HTTPException) as e:
        raw.fault(arm, "run-failed", str(e) if isinstance(e, Fault) else "request-error")
        rec["invalid"] = True
    finally:
        stop(app)
        stop(helper)
        if os.path.exists(ctx["db"]):
            try:
                os.remove(ctx["db"])
            except OSError:
                pass
    raw.json("runs/%05d.json" % idx, rec)


def cmd_run(cfg, args, requests):
    if not requests:
        raise C.Abort("no requests file: write requests.json (see requests.example.json)")
    arms = args.arms or [x for x in C.arm_names(cfg) if "row19" in cfg["arms"][x]]
    C.preflight_seal(cfg)
    a_cfg = {arm: C.arm_cfg(cfg, arm) for arm in arms}
    byid = {r["id"]: r for r in requests}
    ids = args.only.split(",") if args.only else list(byid)
    plan = C.make_plan(args.seed, args.batch, ids, arms, args.n)
    raw = C.Raw(cfg, args.label)
    status = 0
    try:
        raw.json("plan.json", {"seed": args.seed, "batch": args.batch, "plan": plan, "arms": arms, "warmup": cfg["warmup"]})
        for draw, (pid, order) in enumerate(plan):
            for pos, arm in enumerate(order):
                timed_run(cfg, arm, a_cfg[arm], byid[pid], raw, draw, pos)
    except C.Abort as e:
        raw.fault("-", "abort", re.sub(r"[^A-Za-z0-9_ ./:=-]", "_", str(e))[:80])
        status = 1
    finally:
        raw.seal({"kind": "row19", "machine": cfg.get("machine"), "seed": args.seed, "batch": args.batch, "arms": arms,
                  "wall_ns_total": raw.wall_ns})
    C.report(raw, "row19_total_wall_s")
    return status


def cmd_setup(cfg, args, requests):
    faults = set()
    for arm in args.arms or [x for x in C.arm_names(cfg) if "row19" in cfg["arms"][x]]:
        setup_arm(cfg, arm, faults)
    C.emit(["fault: " + f for f in sorted(faults)] + ["setup: done"])
    return 1 if faults else 0


def cmd_confirm(cfg, args, requests):
    """Start each arm's app once, send one request, print the status. No duration is measured or printed."""
    k = cfg["constants"]
    for arm in args.arms or [x for x in C.arm_names(cfg) if "row19" in cfg["arms"][x]]:
        a = C.arm_cfg(cfg, arm)
        r, ctx = a["row19"], a["_ctx"]
        ctx = dict(ctx, port=k["port"], pay_port=k["pay_port"], db=os.path.join(ctx["armdir"], "row19", "run.db"))
        helper = app = None
        try:
            if r.get("helper"):
                helper = subprocess.Popen(C.expand(r["helper"], ctx), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
                wait_port(k["pay_port"], READY_TIMEOUT_S, helper)
            app = subprocess.Popen(C.expand(r["run"], ctx), cwd=os.path.join(ctx["armdir"], "row19", "app"), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            ok = wait_port(k["port"], READY_TIMEOUT_S, app)
            code = None
            if ok:
                conn = http.client.HTTPConnection("127.0.0.1", k["port"], timeout=30)
                conn.request("GET", cfg["confirm_path"])
                code = conn.getresponse().status
            print("%s started=%s GET %s -> %s" % (arm, ok, cfg["confirm_path"], code))
        finally:
            stop(app)
            stop(helper)
    return 0


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--config", default=os.path.join(C.HERE, "commands.json"))
    ap.add_argument("--requests", default=os.path.join(C.HERE, "requests.json"))
    ap.add_argument("--work")
    ap.add_argument("--arms", type=lambda s: s.split(","))
    sub = ap.add_subparsers(dest="cmd", required=True)
    sub.add_parser("setup")
    sub.add_parser("confirm")
    r = sub.add_parser("run")
    r.add_argument("--label", required=True)
    r.add_argument("--seed", required=True)
    r.add_argument("--batch", default="1")
    r.add_argument("--n", type=int, default=20)
    r.add_argument("--only")
    args = ap.parse_args()
    cfg = C.load_config(args.config)
    if args.work:
        cfg["work_root"] = args.work
    requests = []
    if os.path.exists(args.requests):
        with open(args.requests, encoding="utf-8") as f:
            requests = json.load(f)["requests"]
    try:
        return {"setup": cmd_setup, "run": cmd_run, "confirm": cmd_confirm}[args.cmd](cfg, args, requests)
    except C.Abort as e:
        print("abort:", re.sub(r"[^A-Za-z0-9_ ./:=-]", "_", str(e))[:120], file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
