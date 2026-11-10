#!/usr/bin/env python3
"""The procedure reader's view of a sealed file (plan-coverage.md, rule 6).

    python reader.py --sealed <file> --out <dir> [--config commands.json]

The sealed file is opened into a temporary directory, which is deleted at the end. Only a WHITELIST of
fields goes to --out: for each run its label, run number, arm, change id (or request path id), the name,
exit status and kept output of each tool, and the file counts of a trace run; and the faults and the arm
list. Nothing else is copied, so no time, interval, draw, order, seed, load sample or latency figure can pass.

Order does not leak: the runs are sorted by arm, then change id, then by a hash of their own content, and
`run` is the number in that sorted list, not the number of the harness (which is the order of the runs).
"""
import argparse
import hashlib
import json
import os
import sys
import tempfile

import common as C

# The whitelist. A key that is not named here never reaches the output.
RUN_KEYS = ("arm", "change", "path")           # row 15 names a change, row 19 a request path
TOOL_KEYS = ("name", "status")                 # plus the kept output text, read from the file named by "output"
COUNT_KEYS = ("tool", "files_opened", "cache_files_opened")
RUN_DIRS = ("runs", "trace")


def load(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def read_text(d, rel):
    with open(os.path.join(d, rel), encoding="utf-8", errors="replace", newline="") as f:
        return f.read()


def view_of_run(d, rec):
    """The whitelisted view of one raw run record."""
    v = {k: rec[k] for k in RUN_KEYS if k in rec}
    if "tools" in rec:
        v["tools"] = [dict({k: t[k] for k in TOOL_KEYS if k in t}, output=read_text(d, t["output"]) if "output" in t else "")
                      for t in rec["tools"]]
    if "counts" in rec:
        v["counts"] = [{k: c[k] for k in COUNT_KEYS if k in c} for c in rec["counts"]]
    return v


def build(d, label):
    """Returns (index dict, {file name: run view}) from an opened sealed directory."""
    views = []
    for sub in RUN_DIRS:
        base = os.path.join(d, sub)
        if os.path.isdir(base):
            views += [view_of_run(d, load(os.path.join(base, n))) for n in sorted(os.listdir(base)) if n.endswith(".json")]
    key = lambda v: (v["arm"], v.get("change", v.get("path", "")), hashlib.sha256(json.dumps(v, sort_keys=True).encode()).hexdigest())
    views.sort(key=key)
    runs, files = [], {}
    for i, v in enumerate(views, 1):
        v = dict(v, label=label, run=i)
        name = "run-%05d.json" % i
        files[name] = v
        runs.append({"run": i, "file": name, "arm": v["arm"], **{k: v[k] for k in ("change", "path") if k in v}})
    faults = load(os.path.join(d, "faults.json")) if os.path.exists(os.path.join(d, "faults.json")) else []
    arms = sorted(load(os.path.join(d, "plan.json"))["arms"])
    return {"label": label, "arms": arms, "faults": sorted(faults), "runs": runs}, files


def write_json(path, obj):
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write(json.dumps(obj, indent=1, sort_keys=True) + "\n")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--sealed", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--config", default=os.path.join(C.HERE, "commands.json"))
    args = ap.parse_args()
    try:
        cfg = C.load_config(args.config)
        if os.path.isdir(args.out) and os.listdir(args.out):
            raise C.Abort("the output directory is not empty")
        os.makedirs(cfg["work_root"], exist_ok=True)
        tmp = tempfile.mkdtemp(prefix="vt-read-", dir=cfg["work_root"])
        try:
            C.open_sealed(cfg, args.sealed, tmp)
            label = os.path.basename(args.sealed)[: -len(".sealed")] if args.sealed.endswith(".sealed") else os.path.basename(args.sealed)
            index, files = build(tmp, load(os.path.join(tmp, "MANIFEST.json")).get("label", label))
        finally:
            C.rmtree(tmp)
        os.makedirs(args.out, exist_ok=True)
        for name, v in files.items():
            write_json(os.path.join(args.out, name), v)
        write_json(os.path.join(args.out, "index.json"), index)
    except C.Abort as e:
        print("abort:", str(e)[:120], file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
