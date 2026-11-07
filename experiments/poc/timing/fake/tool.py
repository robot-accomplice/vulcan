"""A FAKE check tool for testing the harness. It sleeps, reports a finding for each TODO line of the
files it reads, exits 1 when a file holds BREAK, and writes a marker into the cache directory, so a test
can see that a restore removed what an earlier run wrote."""
import argparse
import json
import os
import subprocess
import sys
import time

ap = argparse.ArgumentParser()
ap.add_argument("--sleep", type=float, default=0.0)
ap.add_argument("--cache")
ap.add_argument("--out", help="a directory in the work tree that the tool writes, like .vulcan")
ap.add_argument("--const-status", type=int, help="always exit with this status and print one finding")
ap.add_argument("--spawn", action="store_true", help="start a child that also reads a file, for the trace")
ap.add_argument("--child", action="store_true")
ap.add_argument("paths", nargs="*")
a = ap.parse_args()

log = os.environ.get("FAKE_TRACE_LOG")


def event(**kw):
    if log:
        with open(log, "a", encoding="utf-8", newline="\n") as f:
            f.write(json.dumps(kw) + "\n")


def files(paths):
    for p in paths:
        if os.path.isdir(p):
            for d, _, fs in os.walk(p):
                for n in sorted(fs):
                    if n.endswith(".go"):
                        yield os.path.join(d, n)
        else:
            yield p


time.sleep(a.sleep)
if a.cache:
    os.makedirs(a.cache, exist_ok=True)
    with open(os.path.join(a.cache, "marker_%d" % os.getpid()), "w") as f:
        f.write("x")
    event(pid=os.getpid(), op="CreateFile", path=os.path.join(a.cache, "marker_read"), access="Generic Read")
if a.out:
    os.makedirs(a.out, exist_ok=True)
    with open(os.path.join(a.out, "built"), "w") as f:
        f.write("x")
if a.child:
    for p in files(a.paths):
        event(pid=os.getpid(), op="CreateFile", path=os.path.abspath(p), access="Generic Read")
    sys.exit(0)
if a.const_status is not None:
    print("pkg/b.go:1:1: constant finding")
    sys.exit(a.const_status)
bad = 0
for p in files(a.paths):
    event(pid=os.getpid(), op="CreateFile", path=os.path.abspath(p), access="Generic Read")
    event(pid=os.getpid(), op="CreateFile", path=os.path.abspath(p), access="Generic Read")  # counted once for each path
    event(pid=os.getpid(), op="CreateFile", path=os.path.abspath(p), access="Read Attributes")  # not a read
    with open(p, encoding="utf-8") as f:
        for i, line in enumerate(f, 1):
            if "TODO" in line:
                print("%s:%d:1: TODO found" % (p.replace(os.sep, "/"), i))
                bad = 1
            if "BREAK" in line:
                print("%s:%d:1: BREAK found" % (p.replace(os.sep, "/"), i))
                bad = 1
event(pid=os.getpid(), op="CreateFile", path="C:\\elsewhere\\outside.go", access="Generic Read")  # not under the tree
if a.spawn:
    c = subprocess.Popen([sys.executable, __file__, "--child"] + a.paths)
    event(pid=os.getpid(), op="Process Create", child=c.pid)
    c.wait()
sys.exit(bad)
