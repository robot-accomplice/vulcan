#!/usr/bin/env python3
"""Writes the draft of defaults v1 from the wishlist (plan-r3, sections 4 and 6, step 2).

Reads   experiments/poc/r3/wishlist/wishlist.json (not changed)
Writes  internal/rules/defaults/v1.rules          the draft ruleset
        experiments/poc/r3/defaults/coverage.json one row for each wanted rule
        experiments/poc/r3/defaults/NOT-STATED.md the list of what no kind states

The wanted text, reason, and sources come from the wishlist as they are. A rule that no kind can
state stays in the list with its reason. Run it from the root of the repository.
"""
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[4]
WISH = json.loads((ROOT / "experiments/poc/r3/wishlist/wishlist.json").read_text(encoding="utf-8"))
OUT = ROOT / "experiments/poc/r3/defaults"

CODE = "leaf gateway entry library"
LEAVES = "leaf gateway entry"

MUTATORS = """os.Setenv os.Unsetenv os.Clearenv os.Chdir net/http.Handle net/http.HandleFunc
expvar.Publish expvar.NewInt expvar.NewFloat expvar.NewMap expvar.NewString
flag.Parse flag.Set flag.String flag.Int flag.Int64 flag.Uint flag.Uint64 flag.Bool flag.Float64 flag.Duration
flag.StringVar flag.IntVar flag.Int64Var flag.UintVar flag.Uint64Var flag.BoolVar flag.Float64Var flag.DurationVar
flag.Var flag.Func flag.BoolFunc flag.TextVar
log.SetOutput log.SetFlags log.SetPrefix log/slog.SetDefault log/slog.SetLogLoggerLevel
runtime/debug.SetGCPercent runtime/debug.SetMemoryLimit runtime/debug.SetMaxThreads runtime/debug.SetMaxStack"""

NO_CTX = """net/http.NewRequest net/http.Get net/http.Head net/http.Post net/http.PostForm
(*net/http.Client).Get (*net/http.Client).Head (*net/http.Client).Post (*net/http.Client).PostForm
(*database/sql.DB).Query (*database/sql.DB).QueryRow (*database/sql.DB).Exec (*database/sql.DB).Begin
(*database/sql.DB).Prepare (*database/sql.DB).Ping
(*database/sql.Tx).Query (*database/sql.Tx).QueryRow (*database/sql.Tx).Exec (*database/sql.Tx).Prepare
(*database/sql.Stmt).Query (*database/sql.Stmt).QueryRow (*database/sql.Stmt).Exec
net.Dial net.DialTimeout (*net.Dialer).Dial crypto/tls.Dial (*crypto/tls.Dialer).Dial
os/exec.Command"""

# The rule entries that state each wanted rule: (suffix, kind, parameters). A suffix makes the id of
# a second entry for one wanted rule.
STATED = {
    "ARCH-01": [("", "depends", {"from": "leaf", "packages": "net/http net database/sql os/exec"})],
    "ARCH-02": [("", "depends", {"from": "library", "packages": "net/http database/sql"})],
    "ARCH-07": [("", "depends", {"from": LEAVES, "deny": LEAVES, "via": "import"})],
    "ARCH-10": [("", "only-in", {"effect": "network", "in": "gateway entry"})],
    "ARCH-11": [("", "only-in", {"effect": "dial", "in": "gateway"})],
    "ARCH-19": [("", "forbid-call", {"funcs": MUTATORS, "in": CODE})],
    "ERR-08": [("", "forbid-call", {"funcs": "os.Exit log.Fatal log.Fatalf log.Fatalln log.Panic log.Panicf log.Panicln (*log.Logger).Fatal* (*log.Logger).Panic*", "in": CODE})],
    "CTX-06": [("", "forbid-call", {"funcs": NO_CTX, "in": CODE})],
    "HTTP-07": [("", "forbid-call", {"funcs": "net/http.Handle net/http.HandleFunc net/http.DefaultServeMux", "in": "entry"})],
    "SEC-10": [("", "depends", {"from": CODE, "packages": "crypto/md5 crypto/sha1 crypto/des crypto/rc4"})],
    "SEC-30": [
        (".imports", "depends", {"from": CODE, "packages": "unsafe syscall C"}),
        (".calls", "forbid-call", {"funcs": "(reflect.Value).Call (reflect.Value).CallSlice (reflect.Value).Method (reflect.Value).MethodByName", "in": CODE}),
    ],
    "SEC-31": [("", "forbid-call", {"funcs": "encoding/gob.NewDecoder", "in": "entry gateway"})],
    "DATA-15": [("", "forbid-call", {"funcs": "time.Now time.Since time.Until", "in": "leaf library", "files": "code"})],
    "OBS-01": [("", "forbid-call", {"funcs": "log.* (*log.Logger).* fmt.Print fmt.Printf fmt.Println", "in": CODE})],
    "TEST-05": [("", "forbid-call", {"funcs": "time.Sleep", "files": "test"})],
}

# A wanted rule of which a kind states one part: the entries, and what stays not stated, with why.
PARTLY = {
    "HTTP-01": ([("", "forbid-call", {"funcs": "net/http.ListenAndServe net/http.ListenAndServeTLS net/http.Serve", "in": "entry"})],
                "shape", "The ban on the package-level functions is stated. That the four timeouts of `http.Server` are set is a value of a field, which no kind reads."),
    "HTTP-02": ([("", "forbid-call", {"funcs": "net/http.Get net/http.Post net/http.Head net/http.PostForm net/http.DefaultClient", "in": "gateway"})],
                "shape", "The ban on the package-level functions and on `http.DefaultClient` is stated. That a client has a non-zero `Timeout` is a value of a field, which no kind reads."),
    "API-04": ([("", "depends", {"from": CODE, "packages": "net/http/pprof expvar"})],
               "outside-source", "The ban on the debug and profiling packages is stated, in every build. An owner and an end date for each old API version are not in the source."),
}

# Each other wanted rule: why no kind states it.
REASONS = {
    "model": "The model check enforces it, and no rule kind states it, so it is not in the rules file and a team cannot delete it there.",
    "shape": "It is a shape of the code (a form of syntax, a type, a field value, a name, or a comment). No kind reads that shape.",
    "flow": "It needs the flow of a value or the paths of a function. `transitions` follows one status field only, and no other kind follows flow.",
    "meaning": "Only a person or a reviewing model can judge it. It depends on what the code means.",
    "run-time": "Only a test, a probe, or the deployment shows a breach, when the code runs.",
    "project-facts": "It needs the facts of one project. A ruleset written before a project exists cannot name them.",
    "outside-source": "It is about CI, the repository, or a process, which a check of the source does not read.",
    "declaration": "It is about a declaration of the system (a config key or an external). No kind reads declarations.",
    "deferred": "It is about topics, remote ports, or transports, which release 1 defers.",
}

NOT_STATED = {
    "model": ["ARCH-04", "ARCH-08", "ARCH-09", "ARCH-12", "ARCH-18"],
    "shape": ["ARCH-03", "ARCH-06", "ARCH-21", "ERR-01", "ERR-03", "ERR-04", "ERR-09", "ERR-13", "CTX-01", "CTX-02", "CTX-03", "CTX-07", "CTX-08",
              "HTTP-09", "HTTP-11", "HTTP-13", "HTTP-17", "SEC-01", "SEC-02", "SEC-07", "DATA-03", "DATA-14", "PAY-01", "RES-02", "CONC-03", "OBS-02",
              "CON-01", "CON-02", "CON-03", "CODE-02", "CODE-03", "CODE-04", "CODE-05", "CODE-06", "CODE-08", "CODE-09", "CODE-11", "TEST-04", "BLD-07"],
    "flow": ["ARCH-22", "ERR-02", "ERR-06", "ERR-11", "ERR-12", "ERR-14", "CTX-04", "CTX-05", "HTTP-03", "HTTP-04", "HTTP-05", "HTTP-06", "HTTP-08", "HTTP-10",
             "SEC-05", "SEC-08", "SEC-09", "SEC-13", "SEC-14", "SEC-28", "DATA-01", "DATA-02", "DATA-04", "DATA-05", "DATA-06", "DATA-07", "DATA-10",
             "RES-05", "CONC-01", "CONC-02", "OBS-03", "OBS-05", "CODE-07"],
    "meaning": ["ARCH-13", "ARCH-14", "ARCH-15", "ARCH-20", "ERR-05", "ERR-07", "ERR-10", "ERR-15", "CTX-09", "HTTP-18", "HTTP-20",
                "SEC-03", "SEC-04", "SEC-06", "SEC-11", "SEC-12", "SEC-15", "SEC-19", "SEC-20", "SEC-21", "SEC-22", "SEC-23", "SEC-24", "SEC-26", "SEC-27",
                "DATA-08", "DATA-09", "DATA-12", "DATA-13", "PAY-02", "PAY-03", "PAY-04", "PAY-06", "PAY-17", "PAY-20", "RES-01", "RES-03", "RES-04", "RES-06", "RES-07",
                "CONC-05", "CONC-06", "CONC-07", "OBS-04", "API-01", "API-05", "API-06", "API-07", "TEST-01", "TEST-02", "TEST-03", "TEST-07", "TEST-10"],
    "run-time": ["ARCH-16", "HTTP-12", "HTTP-14", "HTTP-15", "HTTP-19", "SEC-29", "PAY-07", "PAY-08", "CONC-04", "OBS-06", "OBS-07", "CON-04", "TEST-06", "TEST-08", "TEST-09", "BLD-09"],
    "project-facts": ["ARCH-17", "HTTP-16", "HTTP-21", "SEC-16", "SEC-17", "SEC-18", "SEC-32", "SEC-33", "SEC-34", "PAY-05", "PAY-09", "PAY-10", "PAY-11", "PAY-12", "PAY-13", "PAY-14", "PAY-15", "PAY-16", "PAY-18", "PAY-19", "PAY-21"],
    "outside-source": ["ARCH-25", "DATA-11", "API-02", "API-03", "CODE-01", "CODE-10", "TEST-11", "BLD-01", "BLD-02", "BLD-03", "BLD-04", "BLD-05", "BLD-06", "BLD-08"],
    "declaration": ["ARCH-05", "ARCH-23", "ARCH-24"],
    "deferred": ["SEC-25", "MSG-01", "MSG-02", "MSG-03", "MSG-04", "MSG-05", "MSG-06", "MSG-07"],
}

# What a reader needs next to a row of the list.
NOTES = {
    "ARCH-04": "K0123 and VL0123, in strict mode.",
    "ARCH-05": "`init` writes `strict: true` in the project that it creates. That is a template, not a rule, and a team can change it.",
    "ARCH-08": "K0111. In L, constructor injection cannot build a cycle.",
    "ARCH-09": "K0112 and VL0112.",
    "ARCH-12": "K0207 and VL0207.",
    "ARCH-18": "K0210 and VL0210, in strict mode.",
    "PAY-16": "The kind `transitions` states this once a team names its status field and its table. A default cannot name them.",
    "PAY-18": "Which containers hold money is a project fact, and the config key is a declaration.",
}

# Plan-r3, section 3, states these dependency rules. The wishlist does not.
PLAN3 = "docs/poc/plan-r3.md, section 3"
PLAN_RULES = [
    ("DEP-LIBRARY", "depends", "error", {"from": "library", "deny": LEAVES, "via": "import"}, "A library depends on no leaf.", PLAN3),
    ("DEP-GATEWAY", "depends", "error", {"from": "gateway", "deny": LEAVES, "via": "wire"}, "A gateway calls no other leaf.", PLAN3),
    ("DEP-ENTRY", "depends", "error", {"from": CODE, "deny": "entry", "via": "wire"}, "No part calls an entry leaf.", PLAN3),
    ("DEP-LOCK", "depends", "off", {"map": "locked"}, "A new dependency fails until rules.lock changes; off by the owner's decision 1 of plan-r3, section 8.", PLAN3),
]


def source_line(r):
    return "; ".join(f"{s['title']} ({s['url']})" if s.get("url") and s["url"] not in s["title"] else s["title"] for s in r["sources"])


def block(rid, kind, level, ps, reason, source, wanted=None):
    out = []
    if wanted:
        out.append(f"# {wanted}")
    out += [f"rule {rid}", f"  kind {kind}", f"  level {level}"]
    for k, v in ps.items():
        out.append(f"  {k} {' '.join(v.split())}")
    out += [f"  reason {reason}", f"  source {source}"]
    return "\n".join(out) + "\n"


def main():
    rules = WISH["rules"]
    by_id = {r["id"]: r for r in rules}
    assigned = {}
    for reason, ids in NOT_STATED.items():
        for i in ids:
            assert i not in assigned, i
            assigned[i] = ("not-stated", reason)
    for i in STATED:
        assert i not in assigned, i
        assigned[i] = ("stated", None)
    for i, (_, reason, _) in PARTLY.items():
        assert i not in assigned, i
        assigned[i] = ("partly", reason)
    missing = [r["id"] for r in rules if r["id"] not in assigned]
    extra = [i for i in assigned if i not in by_id]
    assert not missing and not extra, (missing, extra)

    head = [
        "# The rules of this project. The team owns this file: edit a rule to tune it, set level off",
        "# to turn it off, or delete it. What the check enforces is what this file says.",
        "# The draft of defaults v1. It is not frozen (plan-r3, section 4).",
        "",
    ]
    blocks = []
    coverage = []
    for r in rules:
        i = r["id"]
        status, reason = assigned[i]
        entries = []
        if status == "stated":
            entries = STATED[i]
        elif status == "partly":
            entries = PARTLY[i][0]
        ids = []
        for suffix, kind, ps in entries:
            rid = i + suffix
            ids.append(rid)
            blocks.append(block(rid, kind, "error", ps, r["reason"], source_line(r), r["rule"]))
        row = {"id": i, "status": status, "entries": ids, "reason": reason}
        if status == "partly":
            row["not_stated_part"] = PARTLY[i][2]
        if i in NOTES:
            row["note"] = NOTES[i]
        coverage.append(row)
    for rid, kind, level, ps, reason, source in PLAN_RULES:
        blocks.append(block(rid, kind, level, ps, reason, source))
    text = "\n".join(head) + "\n" + "\n".join(blocks)
    (ROOT / "internal/rules/defaults/v1.rules").write_text(text, encoding="utf-8", newline="\n")
    (OUT / "coverage.json").write_text(json.dumps({"rules": coverage}, indent=1) + "\n", encoding="utf-8", newline="\n")

    n = lambda st: sum(1 for c in coverage if c["status"] == st)
    md = ["# What `defaults v1` cannot state", "",
          "Draft, not frozen. `gen_defaults.py` writes this file from the wishlist and `coverage.json`. The wanted text is the wishlist's.", "",
          "This list is a result of plan-r3, section 4: it says what the four rule kinds cannot state.", "",
          "| Status | Wanted rules |", "|---|---:|",
          f"| Stated by the kinds | {n('stated')} |",
          f"| Stated in part | {n('partly')} |",
          f"| Not stated | {n('not-stated')} |",
          f"| Total | {len(coverage)} |", "",
          "`defaults v1` also holds the 4 dependency rules of plan-r3, section 3 (`DEP-LIBRARY`, `DEP-GATEWAY`, `DEP-ENTRY`, and `DEP-LOCK`, which is off). The wishlist does not ask for them.", "",
          "A rule that the model already enforces and that a kind can state is in the file too: `ARCH-07`, `ARCH-10`, `ARCH-11`, `ARCH-19`, `HTTP-07` in part, and `SEC-30`. If a team deletes one of these, the rule stops firing, but its model check (K0101, K0205, K0125, K0211, K0124) stays. Plan-r3's \"no hidden layer\" holds for the rules file. It does not switch off the model. See `QN-base-apps.md`.", "",
          "Two stated rules are wider than their wanted text, because a kind cannot see a purpose. `SEC-10` bans the broken algorithms in all code, not only in security code; its note in the wishlist expects a written exception for a checksum, which here is an edit of `from` or `except`. `DATA-15` bans `time.Now`, `time.Since`, and `time.Until` in all code of a plain leaf or a library, not only for a business decision.", "",
          "Four stated rules hold a finite list where the wanted text is open or says \"such as\": `ARCH-19` (Appendix D's global mutators), `CTX-06` (the forms without a context in the standard library), and `ARCH-10` and `ARCH-11` (Appendix D's network functions, with holes H3 and H7).", "",
          "## Not stated, by reason", "",
          "| Reason | Wanted rules | Meaning |", "|---|---:|---|"]
    for reason, text_ in REASONS.items():
        count = sum(1 for c in coverage if c["status"] == "not-stated" and c["reason"] == reason)
        md.append(f"| {reason} | {count} | {text_} |")
    md += ["", "## Stated in part", ""]
    for c in coverage:
        if c["status"] == "partly":
            md += [f"**{c['id']}.** {by_id[c['id']]['rule']}", "", f"- Reason: {c['reason']}. {c['not_stated_part']}", ""]
    for reason in REASONS:
        md += [f"## {reason}", ""]
        for c in coverage:
            if c["status"] == "not-stated" and c["reason"] == reason:
                line = f"- **{c['id']}.** {by_id[c['id']]['rule']}"
                if "note" in c:
                    line += f" *{c['note']}*"
                md.append(line)
        md.append("")
    (OUT / "NOT-STATED.md").write_text("\n".join(md), encoding="utf-8", newline="\n")
    print(n("stated"), n("partly"), n("not-stated"))
    for reason in REASONS:
        print(reason, sum(1 for c in coverage if c["status"] == "not-stated" and c["reason"] == reason))


if __name__ == "__main__":
    main()
