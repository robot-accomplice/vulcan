"""A FAKE tracer. `start LOG` makes the log, `export LOG CSV` writes a Process Monitor style CSV from the
events that fake/tool.py logs."""
import csv
import json
import sys

if sys.argv[1] == "export":
    with open(sys.argv[3], "w", newline="", encoding="utf-8") as out:
        w = csv.writer(out)
        w.writerow(["Time of Day", "Process Name", "PID", "Operation", "Path", "Result", "Detail"])
        with open(sys.argv[2], encoding="utf-8") as f:
            for ln in f:
                e = json.loads(ln)
                if e["op"] == "Process Create":
                    w.writerow(["0", "python.exe", e["pid"], "Process Create", "python.exe", "SUCCESS", "PID: %d, Command line: x" % e["child"]])
                else:
                    w.writerow(["0", "python.exe", e["pid"], "CreateFile", e["path"], "SUCCESS", "Desired Access: " + e["access"]])
elif sys.argv[1] == "start":
    open(sys.argv[2], "w").close()
