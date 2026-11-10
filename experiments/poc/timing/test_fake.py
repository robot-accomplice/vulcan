#!/usr/bin/env python3
"""Tests of the timing harness on a FAKE arm set (commands.fake.json). No real arm runs here.

    python test_fake.py

The seal key is a throwaway key from `harness newkey`, set in the environment of this process only, and never printed.
"""
import json
import os
import re
import subprocess
import sys

import numpy as np

import common as C

HERE = os.path.dirname(os.path.abspath(__file__))
FAKE = os.path.join(HERE, "commands.fake.json")
CHANGES = os.path.join(HERE, "changes.fake.json")
REQUESTS = os.path.join(HERE, "requests.fake.json")
SIM = os.path.normpath(os.path.join(HERE, "..", "sim", "warmup-sim.py"))
cfg = C.load_config(FAKE)
WORK = cfg["work_root"]
results = []
all_output = []


def check(name, ok, why=""):
    results.append(ok)
    print(("PASS " if ok else "FAIL ") + name + ("" if ok else ": " + why))


def py(script, *args, arms=None, config=FAKE):
    extra = ["--changes", CHANGES] if script == "row15.py" else ["--requests", REQUESTS]
    if arms:
        extra += ["--arms", arms]
    p = subprocess.run([sys.executable, os.path.join(HERE, script), "--config", config] + extra + list(args),
                       capture_output=True, text=True, cwd=HERE)
    all_output.append(p.stdout + p.stderr)
    return p


def open_sealed(label, key=None):
    """Open a sealed file into a temp directory and return its path."""
    out = os.path.join(WORK, "opened-%s-%s" % (label, "own" if key is None else "other"))
    C.rmtree(out)
    env = dict(os.environ)
    if key:
        env["VULCAN_SEAL_KEY"] = key
    p = subprocess.run([C.harness_exe(cfg), "open", "-in", os.path.join(WORK, "sealed", label + ".sealed"), "-dir", out], env=env, capture_output=True)
    return p.returncode, out


def read_runs(label):
    rc, d = open_sealed(label)
    assert rc == 0
    runs = []
    for n in sorted(os.listdir(os.path.join(d, "runs"))):
        runs.append(json.load(open(os.path.join(d, "runs", n))))
    return d, runs


def gate_config(name, **gate):
    """A copy of the fake config with other load_gate limits, written under the work root."""
    c = {k: v for k, v in C.load_config(FAKE).items() if not k.startswith("_")}
    c["load_gate"] = dict(c["load_gate"], **gate)
    path = os.path.join(WORK, name)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(c, f)
    return path


def gate_ok(g):
    """A gate record that passed: the samples of CPU use and free memory, the wait, and the verdict."""
    return (isinstance(g, dict) and g.get("passed") is True and isinstance(g.get("waited_s"), float) and g["waited_s"] >= 0
            and len(g.get("samples", [])) >= 1
            and all(isinstance(s["cpu_pct"], float) and 0 <= s["cpu_pct"] <= 100 and isinstance(s["free_gb"], float) and s["free_gb"] > 0 for s in g["samples"]))


def cache_markers(arm):
    d = os.path.join(C.arm_cfg(cfg, arm)["_ctx"]["cache"], "fakecache")
    return sorted(f for f in os.listdir(d) if f.startswith("marker_"))


def read_trace(label):
    rc, d = open_sealed(label)
    assert rc == 0
    return [json.load(open(os.path.join(d, "trace", n))) for n in sorted(os.listdir(os.path.join(d, "trace")))]


# What the reader may write (the whitelist) and what it must never write.
READER_KEYS = {"label", "run", "arm", "change", "path", "tools", "name", "status", "output", "counts", "tool", "files_opened",
               "cache_files_opened", "arms", "faults", "runs", "file"}
DROPPED_KEYS = {"index", "draw", "position_in_draw", "order", "plan", "seed", "batch", "machine", "unit", "cmd", "ns", "wall_ns", "steps_ns",
                "wall_ns_total", "clean", "findings", "dropped", "base_status", "load_gate", "samples", "waited_s", "passed", "skipped",
                "samples_ns", "median_ns", "residual_fall_pct", "unsettled", "warm_blocks", "invalid", "created_utc", "config_sha256", "kind"}


def walk(o):
    """Every (key, value) pair and every bare number of a JSON value."""
    if isinstance(o, dict):
        for k, v in o.items():
            yield k, v
            yield from walk(v)
    elif isinstance(o, list):
        for v in o:
            yield from walk(v)


def times_of(runs):
    """Every recorded time of the runs, as ns and as the s and ms that a reader could derive."""
    ns = []
    for r in runs:
        ns += [r[k] for k in ("wall_ns", "median_ns") if k in r] + list(r.get("steps_ns", {}).values() if isinstance(r.get("steps_ns"), dict) else [])
        ns += [t["ns"] for t in r.get("tools", [])] + list(r.get("samples_ns", []))
    ns = [x for x in ns if isinstance(x, (int, float)) and x > 1000]
    return {float(x) for x in ns} | {x / 1e9 for x in ns} | {x / 1e6 for x in ns}


def reader_view(label):
    out = os.path.join(WORK, "reader-" + label)
    C.rmtree(out)
    p = subprocess.run([sys.executable, os.path.join(HERE, "reader.py"), "--config", FAKE, "--sealed", os.path.join(WORK, "sealed", label + ".sealed"), "--out", out],
                       capture_output=True, text=True, cwd=HERE)
    all_output.append(p.stdout + p.stderr)
    return p, out


def reader_files(out):
    return {n: open(os.path.join(out, n), encoding="utf-8").read() for n in sorted(os.listdir(out))}


def check_reader(label, raw_runs):
    p, out = reader_view(label)
    check("reader %s: exits 0 and prints nothing" % label, p.returncode == 0 and not p.stdout.strip(), p.stdout + p.stderr)
    files = reader_files(out)
    index = json.loads(files["index.json"])
    views = [json.loads(v) for n, v in files.items() if n != "index.json"]
    check("reader %s: one file per run, plus index.json" % label, len(views) == len(raw_runs) == len(index["runs"]) and sorted(files) == sorted(["index.json"] + [r["file"] for r in index["runs"]]),
          "%d views, %d raw runs" % (len(views), len(raw_runs)))
    keys = {k for text in files.values() for k, _ in walk(json.loads(text))}
    check("reader %s: every key is on the whitelist and none of the dropped keys is in any file" % label, keys <= READER_KEYS and not (keys & DROPPED_KEYS), str(keys - READER_KEYS))
    check("reader %s: no key names a time, a wall or a step" % label, not any(k.endswith("_ns") or k.endswith("_s") or "time" in k or "wall" in k or "step" in k for k in keys))
    recorded = times_of(raw_runs)
    # The run file names (run-00004.json) are sorted positions, not times; leave them out of the scan.
    tokens = {float(t) for text in files.values() for t in re.findall(r"\d+\.\d+(?:[eE][+-]?\d+)?|\d{4,}", re.sub(r"run-\d+\.json", "", text))}
    floats = {v for text in files.values() for _, v in walk(json.loads(text)) if isinstance(v, float)}
    check("reader %s: no float and no number in any file equals a recorded time" % label, recorded and not ((tokens | floats) & recorded), str(sorted((tokens | floats) & recorded)[:3]))
    rl = [(r["arm"], r.get("change", r.get("path"))) for r in index["runs"]]
    check("reader %s: runs are sorted by arm then id, not by run order" % label, rl == sorted(rl) and [r["run"] for r in index["runs"]] == list(range(1, len(rl) + 1)))
    check("reader %s: the arm list of the plan and the label are given" % label, index["arms"] == sorted(index["arms"]) and index["label"] == label and len(index["arms"]) >= 2)
    check("reader %s: the temp directory is gone" % label, not [n for n in os.listdir(WORK) if n.startswith("vt-read-")], str(os.listdir(WORK)))
    return index, views


def main():
    C.rmtree(WORK)
    os.makedirs(WORK)
    key = subprocess.run([C.harness_exe(cfg), "newkey"], capture_output=True, text=True).stdout.strip()
    os.environ["VULCAN_SEAL_KEY"] = key

    # --- setup of the fake arms
    p = py("row15.py", "setup")
    check("row15 setup prints only 'setup: done'", p.returncode == 0 and p.stdout.strip() == "setup: done", p.stdout + p.stderr)
    p = py("row19.py", "setup")
    check("row19 setup prints only 'setup: done'", p.returncode == 0 and p.stdout.strip() == "setup: done", p.stdout + p.stderr)

    # --- arm order: seeded and reproducible
    ids, arms = ["c1", "c2"], ["A", "B", "C"]
    p1, p2 = C.make_plan("s1", "1", ids, arms, 20), C.make_plan("s1", "1", ids, arms, 20)
    p3, p4 = C.make_plan("s2", "1", ids, arms, 20), C.make_plan("s1", "2", ids, arms, 20)
    orders = {tuple(o) for _, o in p1}
    check("plan: same seed and batch give the same plan", p1 == p2)
    check("plan: another seed or batch gives another plan", p1 != p3 and p1 != p4)
    check("plan: the arm order varies over draws", len(orders) > 1)
    q = py("row15.py", "run", "--label", "r1", "--seed", "s1", "--batch", "1", "--n", "6", "--only", "c1")
    q2 = py("row15.py", "run", "--label", "r2", "--seed", "s1", "--batch", "1", "--n", "6", "--only", "c1")
    d1, runs1 = read_runs("r1")
    d2, runs2 = read_runs("r2")
    plan1, plan2 = json.load(open(os.path.join(d1, "plan.json")))["plan"], json.load(open(os.path.join(d2, "plan.json")))["plan"]
    seq = lambda rs: [(r["draw"], r["arm"], r["change"]) for r in rs]
    check("two harness runs with one seed run the same arm order", plan1 == plan2 and seq(runs1) == seq(runs2))
    check("the plan in the record is the plan of make_plan", [(c, o) for c, o in plan1] == C.make_plan("s1", "1", ["c1"], arms, 6))

    # --- clean change, and the printed lines
    allowed = C.ALLOWED_OUTPUT
    lines = q.stdout.strip().split("\n")
    check("clean change: no fault", "faults: none" in lines, q.stdout + q.stderr)
    check("clean change survives shifted line numbers and the generated-file filter (findings equal the base)",
          all(t["clean"] for r in runs1 for t in r["tools"]) and any(t["dropped"] > 0 for r in runs1 for t in r["tools"] if t["name"] == "fake-mod"))
    check("a tool with a nonzero base status counts as clean when it matches the base",
          all(t["clean"] and t["status"] == 3 for r in runs1 for t in r["tools"] if t["name"] == "fake-status"))
    check("only the allowed lines print (runs, faults, row15 total)",
          all(any(a.match(l) for a in allowed) for l in lines) and len(lines) == 3 and lines[0] == "runs: 18" and lines[2].startswith("row15_total_wall_s: "), q.stdout)
    check("nothing printed names an arm or a per-run figure", not re.search(r"\b[ABC]\b|fake-", q.stdout))
    check("the raw directory is gone after the seal", not os.path.exists(os.path.join(WORK, "raw", "r1")))
    check("the sealed file exists and is not plain text", os.path.getsize(os.path.join(WORK, "sealed", "r1.sealed")) > 0 and b"fake-file" not in open(os.path.join(WORK, "sealed", "r1.sealed"), "rb").read())
    rc, _ = open_sealed("r1")
    check("the sealed file opens with the test key", rc == 0)
    other = subprocess.run([C.harness_exe(cfg), "newkey"], capture_output=True, text=True).stdout.strip()
    rc2, _ = open_sealed("r1", other)
    check("the sealed file does not open with another key", rc2 != 0)
    man = json.load(open(os.path.join(d1, "MANIFEST.json")))
    check("the manifest lists the files with sha256 and the run count",
          man["runs"] == 18 and man["config_sha256"] == cfg["_sha256"] and all(len(f["sha256"]) == 64 for f in man["files"]) and any(f["path"] == "plan.json" for f in man["files"]))
    check("the manifest hashes match the opened files", all(C.sha256_file(os.path.join(d1, f["path"])) == f["sha256"] for f in man["files"]))
    ok_wall = abs(sum(r["wall_ns"] for r in runs1) / 1e9 - float(lines[2].split(": ")[1])) < 0.2
    check("the printed total is the sum of the whole-run wall times", ok_wall)

    # --- load gate that passes: the samples are kept with every run
    check("load gate: every timed run keeps its samples of CPU use and free memory, and passed", all(gate_ok(r.get("load_gate")) for r in runs1))
    slow = gate_config("commands.gate-slow.json", sample_s=5)
    py("row15.py", "run", "--label", "g0", "--seed", "s1", "--n", "1", "--only", "c1", arms="C", config=slow)
    _, runs_g0 = read_runs("g0")
    check("load gate: the gate time is outside the timed interval (a 5 s sample waits 5 s but the wall time is shorter)",
          len(runs_g0) == 1 and gate_ok(runs_g0[0]["load_gate"]) and runs_g0[0]["load_gate"]["waited_s"] >= 5 and runs_g0[0]["wall_ns"] / 1e9 < 4, str(runs_g0[0]["wall_ns"] / 1e9))

    # --- dirty change is a fault
    q = py("row15.py", "run", "--label", "r3", "--seed", "s1", "--n", "3", "--only", "c2")
    lines = q.stdout.strip().split("\n")
    check("a dirty check is reported as a procedure fault (not-clean, per arm and tool, deduplicated)",
          any(l.startswith("fault: A not-clean tool fake-file") for l in lines) and any(l.startswith("fault: C not-clean tool fake-file") for l in lines)
          and not any("fake-status" in l for l in lines) and len(lines) == len(set(lines)), q.stdout)
    check("fault lines show no time and no order", not any(re.search(r"\d+\.\d|order|ns\b", l) for l in lines if l.startswith("fault")))
    _, runs3 = read_runs("r3")
    check("the run still records every tool's exit status", all("status" in t for r in runs3 for t in r["tools"]))
    check("the change is reverted after a dirty run (the next run applies cleanly)", "patch-does-not-apply" not in q.stdout and "revert-mismatch" not in q.stdout)

    # --- a change left in the tree by an aborted run is found and reset before the next run
    aa = C.arm_cfg(cfg, "A")
    left = [c for c in json.load(open(CHANGES, encoding="utf-8"))["changes"] if c["id"] == "c2"][0]["arms"]["A"]["patch"]
    check("setup: the left-over change applies to A's tree", C.git_apply(aa["_ctx"]["tree"], os.path.join(HERE, left)))
    q = py("row15.py", "run", "--label", "r3b", "--seed", "s1", "--n", "3", "--only", "c2")
    check("a tree left changed by an aborted run is a fault and is reset, so the change applies",
          "fault: A tree-not-pristine-at-start" in q.stdout and "patch-does-not-apply" not in q.stdout and "revert-mismatch" not in q.stdout, q.stdout)

    # --- restore really restores
    a = C.arm_cfg(cfg, "A")
    live = [e["path"] for e in a["closed_list"]][0]
    snap = C.snap_path(a["_ctx"]["armdir"], 0, a["closed_list"][0]["name"])
    sentinel = os.path.join(live, "sentinel_written_after_snapshot")
    ro = os.path.join(live, "readonly_new_file")
    open(sentinel, "w").write("x")
    open(ro, "w").write("x")
    os.chmod(ro, 0o444)
    kept = [f for f in os.listdir(snap)]
    victim = os.path.join(live, kept[0]) if kept else None
    if victim:
        os.chmod(victim, 0o666)
        open(victim, "w").write("changed after the snapshot")
        os.chmod(victim, 0o444)
    C.restore(a)
    check("restore: a file written into a cache dir is gone", not os.path.exists(sentinel) and not os.path.exists(ro))
    check("restore: a file changed after the snapshot has its snapshot content again",
          victim is None or open(victim).read() == open(os.path.join(snap, kept[0])).read())
    check("restore: the directory equals the snapshot", sorted(os.listdir(live)) == sorted(os.listdir(snap)))
    outs = os.path.join(a["_ctx"]["tree"], ".fakeout")
    check("restore: the tree's build outputs (like .vulcan) are restored as well", os.path.isdir(outs) == os.path.isdir(C.snap_path(a["_ctx"]["armdir"], 1, a["closed_list"][1]["name"])))
    # markers: each run restores first, so the cache holds the markers of the last run only
    q = py("row15.py", "run", "--label", "r4", "--seed", "s9", "--n", "5", "--only", "c1")
    for arm in "ABC":
        arm_a = C.arm_cfg(cfg, arm)
        d = os.path.join(arm_a["_ctx"]["cache"], "fakecache")
        in_snap = len([f for f in os.listdir(C.snap_path(arm_a["_ctx"]["armdir"], 0, "fake cache")) if f.startswith("marker_")])
        n = len([f for f in os.listdir(d) if f.startswith("marker_")])
        # 5 runs per arm have happened; if a restore were missing, the markers of the earlier runs would still be here
        check("restore in a run: cache of arm %s holds the snapshot's markers and those of the last run only" % arm, n == in_snap + 1, "found %d, snapshot %d" % (n, in_snap))

    # --- the go command's module fetch notes are not findings (they made a clean go vet base read 9)
    f, _ = C.normalize(b"go: downloading modernc.org/sqlite v1.61.0\nx.go:3:1: real finding\n", WORK)
    check("go: downloading lines are not counted as findings", len(f) == 1 and "real finding" in f[0], f)

    # --- a drop that fails stops the run loudly
    bad = C.load_config(FAKE)
    bad["drop"] = {"cmd": [sys.executable, "-c", "import sys; sys.exit(1)"]}
    try:
        import row15
        row15.run_drop(bad)
        check("a failing drop command aborts", False, "no abort")
    except C.Abort:
        check("a failing drop command aborts", True)
    # The real drop and tracer of commands.json need admin and change the machine, so no test runs them.
    src = open(os.path.join(HERE, "commands.json"), encoding="utf-8").read()
    check("commands.json holds the intended drop and tracer as comments, and no -accepteula in any file",
          "RAMMap64.exe\", \"-Et\"" in src and "Procmon64.exe" in src and not any(
              "accepteula" in open(os.path.join(HERE, f), encoding="utf-8").read().lower().replace("do not add -accepteula here", "").replace("no -accepteula", "")
              for f in ("common.py", "row15.py", "row19.py", "commands.json", "commands.fake.json")))

    # --- trace
    q = py("row15.py", "trace", "--label", "t1", "--seed", "s1", "--all")
    lines = q.stdout.strip().split("\n")
    check("trace prints only runs, faults and the trace total", all(any(a.match(l) for a in allowed) for l in lines) and lines[-1].startswith("trace_total_wall_s: "), q.stdout + q.stderr)
    check("a tool with no file count is a procedure fault", "fault: B no-file-count tool fake-status" in lines, q.stdout)
    rc, d = open_sealed("t1")
    tr = [json.load(open(os.path.join(d, "trace", n))) for n in sorted(os.listdir(os.path.join(d, "trace")))]
    fa = [c for r in tr if r["arm"] == "A" and r["change"] == "c1" for c in r["counts"]]
    byname = {c["tool"]: c for c in fa}
    check("trace counts a file once for each path, only reads, only under the tree, children included",
          byname["fake-file"]["files_opened"] == 1 and byname["fake-mod"]["files_opened"] == 3, str(fa))
    check("trace counts cache opens as a second figure", byname["fake-file"]["cache_files_opened"] == 1, str(fa))
    csvp = os.path.join(WORK, "unit.csv")
    tree = "C:\\w\\tree"
    rows = [("100", "Process Create", "x", "SUCCESS", "PID: 200, Command line: y"), ("200", "CreateFile", tree + "\\child.go", "SUCCESS", "Desired Access: Generic Read"),
            ("300", "CreateFile", tree + "\\other.go", "SUCCESS", "Desired Access: Generic Read"), ("100", "CreateFile", tree + "\\x.vulcan", "SUCCESS", "Desired Access: Read Data/List Directory"),
            ("100", "CreateFile", tree + "\\n.go", "NAME NOT FOUND", "Desired Access: Generic Read"), ("100", "CreateFile", tree + "\\a.go", "SUCCESS", "Desired Access: Read Attributes")]
    with open(csvp, "w", newline="") as f:
        f.write("Time of Day,Process Name,PID,Operation,Path,Result,Detail\n")
        for pid, op, path, res, det in rows:
            f.write('0,p.exe,%s,%s,"%s",%s,"%s"\n' % (pid, op, path, res, det))
    check("trace parser: the process tree, a read access, a success, one path once",
          C.parse_trace_csv(csvp, tree, 100, ["C:\\w\\cache"]) == (2, 0))

    check("load gate: every trace run keeps its samples and passed", all(gate_ok(r.get("load_gate")) for r in tr))

    # --- row 19: the warm-up rule against the simulation's 2v2 rule on fixed sequences
    check_warmup_against_sim()

    # --- row 19 on the fake app
    q = py("row19.py", "run", "--label", "w1", "--seed", "s1", "--n", "3", arms="A,B")
    lines = q.stdout.strip().split("\n")
    check("row19 prints only runs, faults and the row19 total", all(any(a.match(l) for a in allowed) for l in lines) and len(lines) == 3 and lines[0] == "runs: 6"
          and lines[1] == "faults: none" and lines[2].startswith("row19_total_wall_s: "), q.stdout + q.stderr)
    d, runs = read_runs("w1")
    A = [r for r in runs if r["arm"] == "A"]
    B = [r for r in runs if r["arm"] == "B"]
    check("row19: the fast-decay arm settles within the cap, the slow one is flagged unsettled",
          A and B and not any(r["unsettled"] for r in A) and all(r["unsettled"] and r["warm_blocks"] == 12 for r in B), str([(r["arm"], r["unsettled"], r["warm_blocks"]) for r in runs]))
    check("row19: a run times 100 requests after the warm-up and keeps the median and the residual fall",
          all(len(r["samples_ns"]) == r["warm_blocks"] * 50 + 100 and "residual_fall_pct" in r and r["median_ns"] > 0 for r in runs))
    check("row19: the fake decay shows as a positive residual fall for the unsettled arm", all(r["residual_fall_pct"] > 0 for r in B))
    check("row19: the median is the median of the timed 100", all(abs(r["median_ns"] - float(np.median(r["samples_ns"][-100:]))) < 1 for r in runs))
    check("row19: the total printed is the sum of the whole-run wall times", abs(sum(r["wall_ns"] for r in runs) / 1e9 - float(lines[2].split(": ")[1])) < 0.2)
    check("row19: the raw directory is gone", not os.path.exists(os.path.join(WORK, "raw", "w1")))
    check("load gate: every row19 run keeps its samples and passed", all(gate_ok(r.get("load_gate")) for r in runs))

    # --- load gate that never passes: the run is skipped, nothing runs, and the fault is reported
    never = gate_config("commands.gate-never.json", cpu_max_pct=-1, wait_max_s=0.5)
    before = cache_markers("A")
    q = py("row15.py", "run", "--label", "g15", "--seed", "s1", "--n", "2", "--only", "c1", arms="A", config=never)
    lines = q.stdout.strip().split("\n")
    _, rg = read_runs("g15")
    check("load gate: a row15 run that never passes prints 'fault: A load-gate' and counts its two slots", "fault: A load-gate" in lines and "runs: 2" in lines and "faults: none" not in lines, q.stdout + q.stderr)
    check("load gate: a skipped row15 run ran no step and no tool, and keeps its failed samples",
          len(rg) == 2 and all(r["skipped"] == "load-gate" and r["tools"] == [] and "wall_ns" not in r and "steps_ns" not in r
                               and r["load_gate"]["passed"] is False and len(r["load_gate"]["samples"]) > 1 for r in rg), str(rg))
    q = py("row15.py", "trace", "--label", "g15t", "--seed", "s1", "--all", arms="A", config=never)
    lines = q.stdout.strip().split("\n")
    rt = read_trace("g15t")
    check("load gate: a trace run that never passes prints 'fault: A load-gate' and counts no file", "fault: A load-gate" in lines and "runs: 2" in lines
          and len(rt) == 2 and all(r["skipped"] == "load-gate" and r["counts"] == [] and "wall_ns" not in r for r in rt), q.stdout + q.stderr)
    check("load gate: no tool of a skipped row15 run touched the cache", cache_markers("A") == before)
    q = py("row19.py", "run", "--label", "g19", "--seed", "s1", "--n", "2", arms="A", config=never)
    lines = q.stdout.strip().split("\n")
    _, r19 = read_runs("g19")
    check("load gate: a row19 run that never passes prints 'fault: A load-gate' and starts no app",
          "fault: A load-gate" in lines and "runs: 2" in lines and len(r19) == 2
          and all(r["skipped"] == "load-gate" and "samples_ns" not in r and "wall_ns" not in r and r["load_gate"]["passed"] is False for r in r19), q.stdout + q.stderr)

    # --- the procedure reader's view
    index, views = check_reader("r1", runs1)
    raw_out = sorted(open(os.path.join(d1, t["output"]), encoding="utf-8", errors="replace", newline="").read() for r in runs1 for t in r["tools"])
    check("reader r1: the kept output and exit status of every tool are shown",
          sorted(t["output"] for v in views for t in v["tools"]) == raw_out and any(t["output"] for v in views for t in v["tools"])
          and sorted((v["arm"], t["name"], t["status"]) for v in views for t in v["tools"]) == sorted((r["arm"], t["name"], t["status"]) for r in runs1 for t in r["tools"]))
    tr_runs = read_trace("t1")
    pt, out_t = reader_view("t1")
    vt = [json.loads(v) for n, v in reader_files(out_t).items() if n != "index.json"]
    check("reader t1: the file counts of every trace run are shown, and no other number",
          pt.returncode == 0 and sorted(json.dumps(v["counts"], sort_keys=True) for v in vt) == sorted(json.dumps(r["counts"], sort_keys=True) for r in tr_runs)
          and not ({k for v in vt for k, _ in walk(v)} & DROPPED_KEYS), pt.stderr)
    _, runs3_ = read_runs("r3")
    index3, _ = check_reader("r3", runs3_)
    check("reader r3: the fault list of the raw is shown", "A not-clean tool fake-file" in index3["faults"] and index3["faults"] == sorted(index3["faults"]), str(index3["faults"]))
    index19, views19 = check_reader("w1", runs)
    check("reader w1: a row19 run shows its arm and request path id only", all(set(v) == {"label", "run", "arm", "path"} for v in views19) and len(views19) == 6)
    check("reader: it refuses a non-empty output directory and a wrong key",
          subprocess.run([sys.executable, os.path.join(HERE, "reader.py"), "--config", FAKE, "--sealed", os.path.join(WORK, "sealed", "r1.sealed"), "--out", os.path.join(WORK, "reader-r1")],
                         capture_output=True, cwd=HERE).returncode != 0
          and subprocess.run([sys.executable, os.path.join(HERE, "reader.py"), "--config", FAKE, "--sealed", os.path.join(WORK, "sealed", "r1.sealed"), "--out", os.path.join(WORK, "reader-bad")],
                             capture_output=True, cwd=HERE, env=dict(os.environ, VULCAN_SEAL_KEY=other)).returncode != 0
          and not [n for n in os.listdir(WORK) if n.startswith("vt-read-")])

    # --- a V-like arm: the common tools run in the lowered module (cwd) on the lowered files (OW2)
    check_lowered_tools()

    # --- a seal key never printed
    blob = "\n".join(all_output)
    check("the seal key is not in any output of any command", key not in blob)

    C.rmtree(WORK)
    print("%d of %d checks passed" % (sum(results), len(results)))
    return 0 if all(results) else 1


def check_lowered_tools():
    tree = os.path.join(WORK, "vlike", "tree")
    low = os.path.join(tree, ".vulcan", "build")
    os.makedirs(os.path.join(low, "api", "inventory"))
    gen = "// Code generated by vulcan; DO NOT EDIT.\npackage inventory\n"
    with open(os.path.join(low, "api", "inventory", "w.go"), "w", newline="\n") as f:
        f.write(gen)
    with open(os.path.join(low, "api", "inventory", "inventory.go"), "w", newline="\n") as f:
        f.write("// Code generated by vulcan; edit the .vulcan file.\npackage inventory\n")
    echo = lambda *a: [sys.executable, "-c", "import os,sys; print(os.path.normcase(os.getcwd()))\nprint(' '.join(sys.argv[1:]))", *a]
    # a change as load_changes builds it for a V entry: the source file is .vulcan, "files" are the lowered Go (relative to the tree)
    spec = {"file": "api/inventory/inventory.vulcan", "lowered": [".vulcan/build/api/inventory/inventory.go"]}
    spec["files"] = spec["lowered"]
    spec["pkgs"] = C.pkgs_of(spec["files"])
    chg = {"files": spec["files"], "pkgs": spec["pkgs"]}
    env = dict(os.environ)
    own = {"name": "t-pkg", "unit": "package", "cmd": echo("{pkgs}"), "cwd": low}
    r = C.run_tool(own, {}, env, tree, chg, "change", 60)
    cwd_line, args_line = r["output"].decode().replace("\r", "").split("\n")[:2]
    check("a tool with its own cwd runs there", cwd_line == os.path.normcase(low), cwd_line)
    check("a V-like arm's package tool gets the packages of the lowered files, relative to the lowered module (not of the source file, not under .vulcan/build)",
          args_line == "./api/inventory", args_line)
    r = C.run_tool(dict(own, name="t-file", unit="file", cmd=echo("{files}")), {}, env, tree, chg, "change", 60)
    check("a V-like arm's file tool gets the lowered files, relative to the lowered module", r["output"].decode().replace("\r", "").split("\n")[1] == "api/inventory/inventory.go", r["output"])
    r = C.run_tool(dict(own, name="t-full"), {}, env, tree, None, "full", 60)
    check("a V-like arm's full check gets ./... in the lowered module", r["output"].decode().replace("\r", "").split("\n")[:2] == [os.path.normcase(low), "./..."], r["output"])
    r = C.run_tool({k: v for k, v in own.items() if k != "cwd"}, {}, env, tree, chg, "change", 60)
    check("a tool with no cwd still runs in the tree on the files of the change as given (P and L)",
          r["output"].decode().replace("\r", "").split("\n")[:2] == [os.path.normcase(tree), "./.vulcan/build/api/inventory"], r["output"])
    try:
        C.run_tool(own, {}, env, tree, {"files": ["api/inventory/inventory.go"], "pkgs": ["./api/inventory"]}, "change", 60)
        check("a change file outside the cwd of a tool is refused", False, "no abort")
    except C.Abort:
        check("a change file outside the cwd of a tool is refused", True)
    # the spliced tools are the same entries as the common set, plus the cwd
    sets = {"common": [{"name": "t1", "unit": "package", "cmd": ["x", "{pkgs}"]}, {"name": "t2", "unit": "file", "cmd": ["y", "{files}"]}]}
    got = C.splice([{"name": "first", "unit": "u", "cmd": ["z"]}, {"@": "common", "cwd": "{tree}\\.vulcan\\build"}], sets)
    check("splice: an @ item adds the tools of the set with the cwd added, same name, unit and cmd, and leaves the set alone",
          [t["name"] for t in got] == ["first", "t1", "t2"] and all(t.get("cwd") == "{tree}\\.vulcan\\build" for t in got[1:])
          and [{k: t[k] for k in ("name", "unit", "cmd")} for t in got[1:]] == sets["common"] and all("cwd" not in t for t in sets["common"]), str(got))
    a = C.arm_cfg(C.load_config(os.path.join(HERE, "commands.json")), "V")
    names = [t["name"] for t in a["check"]]
    real = C.load_config(os.path.join(HERE, "commands.json"))
    p = C.arm_cfg(real, "P")
    # Rule 2: V gets the common tools, not P's own architecture linter.
    common_names = [t["name"] for t in real["sets"]["common_tools"]]
    p_common = [t for t in p["check"] if t["name"] in common_names]
    check("commands.json: V runs vulcan-check first, then the common tools in P's order, each with the cmd of P and the cwd of the lowered module",
          names == ["vulcan-check"] + [t["name"] for t in p_common]
          and [{k: t[k] for k in ("name", "unit", "cmd")} for t in a["check"][1:]] == p_common
          and all(t["cwd"] == os.path.join(a["_ctx"]["tree"], ".vulcan", "build") for t in a["check"][1:]) and "cwd" not in a["check"][0], str(names))
    # normalize: paths are against the cwd of the tool, and the header test is on the file the path names
    with open(os.path.join(low, "api", "inventory", "inventory.vulcan"), "w", newline="\n") as f:
        f.write("package inventory\n")
    f, d = C.normalize(b"api/inventory/w.go:3:1: in wrapper\napi/inventory/inventory.go:4:2: in lowered, not standard header\n", tree, low)
    check("normalize: relative paths are against the cwd of the tool; a standard generated header is dropped and a non-standard one is kept",
          d == 1 and len(f) == 1 and "inventory.go" in f[0], str((f, d)))


def check_warmup_against_sim():
    import row19
    w = cfg["warmup"]
    simsrc = open(SIM, encoding="utf-8").read().split("for rule,maxb,thr in")[0]
    ns = {}
    exec(compile(simsrc, SIM, "exec"), ns)

    class Stub:
        def __init__(self, v):
            self.v = v

        def normal(self, mu, sd, size):
            return self.v[:size]
    n = np.arange(w["cap_blocks"] * w["block"] + w["timed"])
    seqs = {
        "flat": 1000 + (n * 37) % 11,
        "fast exponential": 1000 * (1 + 3 * np.exp(-n / 60)),
        "never settles": 1000 * (1 + 5 * np.exp(-n / 400)),
        "step down at 250": 1000 * np.where(n < 250, 1.3, 1.0),
        "power law p=1": 1000 * (1 + (1 + n / 50) ** -1.0),
        "settles late": 1000 * (1 + 2 * np.exp(-n / 150)),
    }
    for name, t in seqs.items():
        it = iter(t.tolist())
        res = row19.steady_state(lambda: next(it), w)
        # the simulation with its noise replaced by log(t / (1 + excess)), and a time constant so small that the excess is 0 from request 1 on
        ns["rng"] = Stub(np.log(t / (1 + np.where(n == 0, 1.0, 0.0))))
        _, fall, _ = ns["run"](1e-9, 0.0, "2v2", w["cap_blocks"], w["band"], 1, "exp", True)
        ns["rng"] = Stub(np.log(t / (1 + np.where(n == 0, 1.0, 0.0)))[: w["cap_blocks"] * w["block"]])
        _, uns = ns["run"](1e-9, 0.0, "2v2", w["cap_blocks"], w["band"], 1, "exp", False)
        check("warm-up %-18s same residual fall and unsettled flag as sim 2v2 (stop at block %d)" % (name, res["warm_blocks"]),
              round(res["residual_fall_pct"], 1) == fall and res["unsettled"] == (uns == 100), "harness %.1f/%s, sim %s/%s" % (res["residual_fall_pct"], res["unsettled"], fall, uns))
    flat = iter(seqs["flat"].tolist())
    check("warm-up: a flat sequence stops at block 4 (the first block of the rule)", row19.steady_state(lambda: next(flat), w)["warm_blocks"] == 4)
    nev = iter(seqs["never settles"].tolist())
    r = row19.steady_state(lambda: next(nev), w)
    check("warm-up: a sequence that never settles stops at the cap of 12 blocks and times the next 100", r["unsettled"] and r["warm_blocks"] == 12 and len(r["samples_ns"]) == 700)


if __name__ == "__main__":
    sys.exit(main())
