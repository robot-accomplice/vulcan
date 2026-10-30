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

SQL_FUNCS = """(*database/sql.DB).Query (*database/sql.DB).QueryRow (*database/sql.DB).Exec (*database/sql.DB).Prepare (*database/sql.DB).QueryContext (*database/sql.DB).QueryRowContext (*database/sql.DB).ExecContext (*database/sql.DB).PrepareContext (*database/sql.Tx).Query (*database/sql.Tx).QueryRow (*database/sql.Tx).Exec (*database/sql.Tx).Prepare (*database/sql.Tx).QueryContext (*database/sql.Tx).QueryRowContext (*database/sql.Tx).ExecContext (*database/sql.Tx).PrepareContext (*database/sql.Conn).QueryContext (*database/sql.Conn).QueryRowContext (*database/sql.Conn).ExecContext (*database/sql.Conn).PrepareContext"""

SLOG_FUNCS = """log/slog.Debug log/slog.Info log/slog.Warn log/slog.Error log/slog.DebugContext log/slog.InfoContext log/slog.WarnContext log/slog.ErrorContext log/slog.Log (*log/slog.Logger).Debug (*log/slog.Logger).Info (*log/slog.Logger).Warn (*log/slog.Logger).Error (*log/slog.Logger).DebugContext (*log/slog.Logger).InfoContext (*log/slog.Logger).WarnContext (*log/slog.Logger).ErrorContext (*log/slog.Logger).Log"""

RETURN_EFFECTS = """(*database/sql.Tx).Commit (*database/sql.Tx).Exec (*database/sql.Tx).ExecContext (*database/sql.DB).Exec (*database/sql.DB).ExecContext
(*net/http.Client).Do (*net/http.Client).Post (*net/http.Client).Get net/smtp.SendMail os.WriteFile os.Remove"""

NUMS = "int int8 int16 int32 int64 uint uint8 uint16 uint32 uint64 float32 float64"

# The rule entries that state each wanted rule: (suffix, kind, parameters), and optionally a level. A
# suffix makes the id of a second entry for one wanted rule. The wishlist's own choice asks for a
# warning in CONC-03, CON-03, and CODE-03.
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
    "ARCH-03": [("", "declaration", {"at": "port-signature", "deny-types": "*net/http.Request net/http.ResponseWriter *net/http.Response *database/sql.DB *database/sql.Tx *database/sql.Rows net.Conn"})],
    "ARCH-06": [("", "declaration", {"at": "config-entry", "require": "comment"})],
    "ARCH-21": [("", "declaration", {"at": "library", "deny-names": "util common misc helpers shared base"})],
    "ERR-01": [("", "code-form", {"form": "dropped-error", "deferred-ok": "(*database/sql.Tx).Rollback (*database/sql.Rows).Close (*database/sql.Stmt).Close (*database/sql.Conn).Close (*os.File).Close (io.Closer).Close", "in": CODE})],
    "ERR-03": [("", "code-form", {"form": "discard-error", "unless-comment": "yes", "in": CODE})],
    "ERR-04": [("", "code-form", {"form": "error-compare", "in": CODE})],
    "ERR-09": [("", "code-form", {"form": "one-value-assertion", "in": CODE})],
    "ERR-13": [("", "call-form", {"funcs": "errors.New fmt.Errorf", "require-text": r"^[a-z] [^.:!?\n]$", "in": CODE})],
    "CTX-01": [("", "declaration", {"at": "param", "types": "context.Context", "require": "first name=ctx", "in": CODE})],
    "CTX-02": [("", "declaration", {"at": "field", "deny-types": "context.Context", "in": CODE})],
    "CTX-03": [("", "call-form", {"param-type": "context.Context", "require": "not-nil", "in": CODE})],
    "HTTP-11": [("", "literal", {"type": "crypto/tls.Config", "deny": "InsecureSkipVerify=true", "below": "MinVersion=771", "in": CODE, "files": "code"})],
    "HTTP-13": [("", "call-form", {"funcs": "(*net/http.ServeMux).Handle (*net/http.ServeMux).HandleFunc", "require": "constant", "require-text": r"^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s", "in": "entry"})],
    "HTTP-17": [("", "literal", {"type": "net/http.Cookie", "require": "Secure=true HttpOnly=true SameSite=2,3", "in": "entry"})],
    "SEC-07": [("", "call-form", {"funcs": SQL_FUNCS, "require": "constant", "unless-comment": "yes", "in": CODE})],
    "DATA-03": [("", "call-form", {"funcs": SQL_FUNCS, "deny-text": r"(?i)\b(BEGIN|COMMIT|ROLLBACK)\b", "in": CODE})],
    "CONC-03": [("", "code-form", {"form": "channel-size", "unless-comment": "yes", "in": CODE}, "warn")],
    "OBS-02": [("", "call-form", {"funcs": SLOG_FUNCS, "require": "constant", "in": CODE})],
    "CON-03": [("", "code-form", {"form": "return-effect", "funcs": RETURN_EFFECTS, "unless-comment": "yes", "in": CODE}, "warn")],
    "CODE-02": [("", "code-form", {"form": "suppression", "in": CODE})],
    "CODE-03": [("", "code-form", {"form": "function-complexity", "limit": "15", "in": CODE}, "warn")],
    "CODE-04": [("", "code-form", {"form": "switch-exhaustive", "in": CODE})],
    "CODE-05": [
        (".json", "call-form", {"funcs": "encoding/json.Marshal encoding/json.MarshalIndent (*encoding/json.Encoder).Encode", "arg": "0", "require": "tagged=json", "in": CODE}),
        (".xml", "call-form", {"funcs": "encoding/xml.Marshal encoding/xml.MarshalIndent (*encoding/xml.Encoder).Encode", "arg": "0", "require": "tagged=xml", "in": CODE}),
    ],
    "CODE-06": [("", "declaration", {"at": "embedded", "in": "library"})],
    "CODE-08": [("", "declaration", {"at": "doc", "in": CODE, "files": "code"})],
    "TEST-04": [("", "code-form", {"form": "test-logic"})],
    "HTTP-01": [
        ("", "forbid-call", {"funcs": "net/http.ListenAndServe net/http.ListenAndServeTLS net/http.Serve", "in": "entry"}),
        (".timeouts", "literal", {"type": "net/http.Server", "require": "ReadHeaderTimeout ReadTimeout WriteTimeout IdleTimeout", "in": "entry"}),
    ],
    "HTTP-02": [
        ("", "forbid-call", {"funcs": "net/http.Get net/http.Post net/http.Head net/http.PostForm net/http.DefaultClient", "in": "gateway"}),
        (".timeout", "literal", {"type": "net/http.Client", "require": "Timeout", "in": "gateway"}),
    ],
}

# A wanted rule of which a kind states one part: the entries, and what stays not stated, with why.
PARTLY = {
    "API-04": ([("", "depends", {"from": CODE, "packages": "net/http/pprof expvar"})],
               "outside-source", "The ban on the debug and profiling packages is stated, in every build. An owner and an end date for each old API version are not in the source."),
    "CTX-07": ([("", "call-form", {"funcs": "context.WithValue", "arg": "1", "require": "unexported-type", "in": CODE})],
               "meaning", "The unexported key type is stated. Whether a value holds only request-scoped data is for a person to judge."),
    "CTX-08": ([("", "declaration", {"at": "wired-port", "to": "gateway"})],
               "whole-program", "A port that a gateway provides is stated. A plain leaf that reaches a network or an acquiring function itself, and a provider that reaches one through its own ports, need what a provider reaches, and L's analyzers read one package at a time."),
    "HTTP-09": ([("", "literal", {"type": "net/http.Client", "require": "CheckRedirect", "in": "gateway"})],
                "meaning", "That a client sets `CheckRedirect` is stated. Whether that function checks the target against an allow-list is for a person to judge."),
    "SEC-01": ([("", "literal", {"deny-text": r"(sk_live_|sk_test_|AKIA[0-9A-Z]{16}|ghp_[A-Za-z0-9]{20,}|-----BEGIN[A-Z\s]*PRIVATE\sKEY)", "names": "*password* *Password* *secret* *Secret* *apiKey* *ApiKey* *APIKey* *token* *Token*", "in": CODE})],
               "meaning", "A string in the code that has the form of a key, or that a name marks as a secret, is stated. Declaration files and fixtures are not read, and only a person can say that a value is real."),
    "SEC-02": ([("", "declaration", {"at": "settings-field", "names": "*Key *Token *Password *Secret *Credential *Credentials", "require": "vulcan=secret", "in": LEAVES})],
               "meaning", "A settings field whose name shows a credential is stated. A credential under another name is not found, as the wishlist's note says."),
    "DATA-14": ([("", "declaration", {"at": "field param var", "names": "*At *Time *Timeout *Deadline *Duration *Interval *TTL *Delay at time timeout deadline duration interval ttl delay", "deny-types": "string " + NUMS, "in": CODE})],
                "meaning", "A value whose name shows a time or a period is stated. A value under another name is not found, as the wishlist's note says."),
    "PAY-01": ([("", "declaration", {"at": "field param var", "names": "*Amount* *amount* *Price* *price* *Total* *total* *Cost* *cost* *Charge* *charge* *Refund* *refund* *Fee* *fee* *Balance* *balance* *Money* *money* *Cents* *cents*", "deny-types": "float32 float64", "in": CODE})],
               "project-facts", "A float whose name shows money is stated. The name of the project's money type would give a stronger check, as the wishlist's note says."),
    "RES-02": ([("", "code-form", {"form": "retry-loop", "in": "leaf gateway"})],
               "flow", "A loop that calls `time.Sleep` is stated: it needs a condition, a growing wait, and a random value in its function. A retry in another form is not found, as the wishlist's note says, and that the wait grows or is random is read from the form only."),
    "CON-01": ([("", "declaration", {"at": "port-method", "require": "requires", "except-names": "Get* List* Find* Read* Is* Has* Count* Load* Lookup* Exists*", "in": LEAVES})],
               "meaning", "That a clause exists is stated, on each port method that takes arguments and whose name does not show a read. Whether a method changes state, and whether a clause is enough, are for a person to judge."),
    "CODE-09": ([("", "code-form", {"form": "duplicate-function", "limit": "60", "similarity": "90", "in": CODE, "files": "code"})],
                "whole-program", "Near-duplicates in one package are stated. A duplicate in another package needs a view of the whole module, and L's analyzers read one package at a time."),
    "CODE-11": ([(".deprecated", "code-form", {"form": "deprecated-use", "in": CODE}),
                 (".unused", "code-form", {"form": "unused-name", "in": CODE})],
                "whole-program", "A use of a deprecated function, and an unexported name at package level that nothing uses, are stated. An unused exported name needs a view of the whole module, and an unused method or field is not read."),
    "BLD-07": ([("", "literal", {"deny-text": r"^(https?://|\d+\.\d+\.\d+\.\d+|localhost|[a-z0-9.-]+\.(com|net|org|io|dev|local|internal)(:\d+)?$|[a-z0-9.-]*:\d+$)", "in": LEAVES, "files": "code"})],
               "meaning", "An address or a host name in a string of a leaf is stated. Whether any other value is specific to a deployment is for a person to judge."),
}

# Each other wanted rule: why no kind states it.
REASONS = {
    "model": "The model check enforces it, and no rule kind states it, so it is not in the rules file and a team cannot delete it there.",
    "shape": "It is a shape of the code that no kind reads. No wanted rule stays here: amendment 6 added the kinds.",
    "flow": "It needs the flow of a value or the paths of a function. `transitions` follows one status field only, and no other kind follows flow.",
    "meaning": "Only a person or a reviewing model can judge it. It depends on what the code means.",
    "run-time": "Only a test, a probe, or the deployment shows a breach, when the code runs.",
    "project-facts": "It needs the facts of one project. A ruleset written before a project exists cannot name them.",
    "outside-source": "It is about CI, the repository, or a process, which a check of the source does not read.",
    "declaration": "It is about a declaration of the system (a config key or an external). No kind reads declarations.",
    "deferred": "It is about topics, remote ports, or transports, which release 1 defers.",
    "whole-program": "It needs a view of the whole module. L's analyzers read one package at a time.",
}

NOT_STATED = {
    "model": ["ARCH-04", "ARCH-08", "ARCH-09", "ARCH-12", "ARCH-18"],
    "shape": [],
    "flow": ["ARCH-22", "ERR-02", "ERR-06", "ERR-11", "ERR-12", "ERR-14", "CTX-04", "CTX-05", "HTTP-03", "HTTP-04", "HTTP-05", "HTTP-06", "HTTP-08", "HTTP-10",
             "SEC-05", "SEC-08", "SEC-09", "SEC-13", "SEC-14", "SEC-28", "DATA-01", "DATA-02", "DATA-04", "DATA-05", "DATA-06", "DATA-07", "DATA-10",
             "RES-05", "CONC-01", "CONC-02", "OBS-03", "OBS-05", "CODE-07"],
    "meaning": ["CON-02", "ARCH-13", "ARCH-14", "ARCH-15", "ARCH-20", "ERR-05", "ERR-07", "ERR-10", "ERR-15", "CTX-09", "HTTP-18", "HTTP-20",
                "SEC-03", "SEC-04", "SEC-06", "SEC-11", "SEC-12", "SEC-15", "SEC-19", "SEC-20", "SEC-21", "SEC-22", "SEC-23", "SEC-24", "SEC-26", "SEC-27",
                "DATA-08", "DATA-09", "DATA-12", "DATA-13", "PAY-02", "PAY-03", "PAY-04", "PAY-06", "PAY-17", "PAY-20", "RES-01", "RES-03", "RES-04", "RES-06", "RES-07",
                "CONC-05", "CONC-06", "CONC-07", "OBS-04", "API-01", "API-05", "API-06", "API-07", "TEST-01", "TEST-02", "TEST-03", "TEST-07", "TEST-10"],
    "run-time": ["ARCH-16", "HTTP-12", "HTTP-14", "HTTP-15", "HTTP-19", "SEC-29", "PAY-07", "PAY-08", "CONC-04", "OBS-06", "OBS-07", "CON-04", "TEST-06", "TEST-08", "TEST-09", "BLD-09"],
    "project-facts": ["ARCH-17", "HTTP-16", "HTTP-21", "SEC-16", "SEC-17", "SEC-18", "SEC-32", "SEC-33", "SEC-34", "PAY-05", "PAY-09", "PAY-10", "PAY-11", "PAY-12", "PAY-13", "PAY-14", "PAY-15", "PAY-16", "PAY-18", "PAY-19", "PAY-21"],
    "outside-source": ["ARCH-25", "DATA-11", "API-02", "API-03", "CODE-01", "CODE-10", "TEST-11", "BLD-01", "BLD-02", "BLD-03", "BLD-04", "BLD-05", "BLD-06", "BLD-08"],
    "declaration": ["ARCH-05", "ARCH-23", "ARCH-24"],
    "whole-program": [],
    "deferred": ["SEC-25", "MSG-01", "MSG-02", "MSG-03", "MSG-04", "MSG-05", "MSG-06", "MSG-07"],
}

# What a reader needs next to a row of the list.
# The 39 wanted rules that the first draft listed under "shape" (plan-r3, amendment 6).
SHAPE39 = """ARCH-03 ARCH-06 ARCH-21 ERR-01 ERR-03 ERR-04 ERR-09 ERR-13 CTX-01 CTX-02 CTX-03 CTX-07 CTX-08 HTTP-09 HTTP-11 HTTP-13
HTTP-17 SEC-01 SEC-02 SEC-07 DATA-03 DATA-14 PAY-01 RES-02 CONC-03 OBS-02 CON-01 CON-02 CON-03 CODE-02 CODE-03 CODE-04 CODE-05 CODE-06
CODE-08 CODE-09 CODE-11 TEST-04 BLD-07""".split()

NOTES = {
    "CON-02": "Moved from shape. Which rule must stop an effect is what the code means. A kind can read a clause, not its purpose.",
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
        for entry in entries:
            suffix, kind, ps = entry[:3]
            level = entry[3] if len(entry) > 3 else "error"
            rid = i + suffix
            ids.append(rid)
            blocks.append(block(rid, kind, level, ps, r["reason"], source_line(r), r["rule"]))
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
          "Stated forms that hold a finite list, or are wider than the words, for the same reason: `ERR-13` reads `errors.New` and `fmt.Errorf` only. `HTTP-13` reads the patterns of `http.ServeMux`, where a method in the pattern gives 405 for the others. `HTTP-17` reads each cookie of an entry leaf, not only a session cookie. `ERR-01` allows a bare `defer` of a listed `Close` or `Rollback`, and it cannot tell a read handle from a write handle. `CON-03` lists its effects and adds each port call. `CODE-08` holds for each exported name, also a struct field and an interface method. `CODE-08` and `CON-01` name no topic, because release 1 has none.", "",
          "`HTTP-01` and `HTTP-02` were stated in part, with \"shape\" as the reason for the rest. The kind `literal` states the rest, so both are now stated. They were not among the 39.", "",
          "## Not stated, by reason", "",
          "| Reason | Wanted rules | Meaning |", "|---|---:|---|"]
    for reason, text_ in REASONS.items():
        count = sum(1 for c in coverage if c["status"] == "not-stated" and c["reason"] == reason)
        if not count:
            continue
        md.append(f"| {reason} | {count} | {text_} |")
    md += ["", "## Stated in part, by the reason for the rest", "",
           "| Reason | Wanted rules |", "|---|---:|"]
    for reason in REASONS:
        count = sum(1 for c in coverage if c["status"] == "partly" and c["reason"] == reason)
        if count:
            md.append(f"| {reason} | {count} |")
    md += ["", "## The 39 wanted rules of amendment 6", "",
           "| Rule | Outcome | Entries |", "|---|---|---|"]
    row_by = {c["id"]: c for c in coverage}
    for i in SHAPE39:
        c = row_by[i]
        what = {"stated": "stated", "partly": f"in part ({c['reason']})", "not-stated": f"not stated ({c['reason']})"}[c["status"]]
        md.append(f"| {i} | {what} | {', '.join(f'`{e}`' for e in c['entries'])} |")
    n39 = lambda st: sum(1 for i in SHAPE39 if row_by[i]["status"] == st)
    md += ["", f"Of the 39: {n39('stated')} stated, {n39('partly')} in part, {n39('not-stated')} not stated.", "", "## Stated in part", ""]
    for c in coverage:
        if c["status"] == "partly":
            md += [f"**{c['id']}.** {by_id[c['id']]['rule']}", "", f"- Reason: {c['reason']}. {c['not_stated_part']}", ""]
    for reason in REASONS:
        if not NOT_STATED[reason]:
            continue
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
