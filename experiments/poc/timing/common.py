"""Shared parts of the timing harness for rows 15 and 19 of the Vulcan evaluation plan
(docs/poc/plan-coverage.md, section 1, rule 6).

The builder of this harness never sees a real arm's time. So:
  * every raw record goes to a raw directory under the work root, and the raw directory is sealed
    with `harness seal` and deleted when the run ends;
  * the only lines that a command prints are the ones that emit() allows, and none of them shows a
    time, a count or an order for one arm.
"""
import csv
import hashlib
import json
import os
import random
import re
import shutil
import stat
import subprocess
import sys
import tempfile
import time

import psutil

HERE = os.path.dirname(os.path.abspath(__file__))

# Exit codes of robocopy below this value are success (0 to 7 are "copied" or "nothing to copy").
ROBOCOPY_FAIL_MIN = 8
STATUS_TIMEOUT = -9999
STATUS_LAUNCH = -9998
# Files that robocopy copies in the restore; /MIR also purges what the snapshot lacks.
ROBOCOPY_FLAGS = ["/MIR", "/COPY:DAT", "/DCOPY:DAT", "/R:1", "/W:1", "/NFL", "/NDL", "/NJH", "/NJS", "/NP", "/NC", "/NS", "/MT:16"]
READ_CHUNK = 1 << 20
BYTES_PER_GB = 1 << 30
GENERATED = re.compile(r"^// Code generated .* DO NOT EDIT\.\s*$")
POSITION = re.compile(r"(\.(?:go|vulcan)):\d+(?::\d+)?(?:,\d+)?")
FILE_TOKEN = re.compile(r"(?P<p>(?:[A-Za-z]:)?[\w.\\/\-]+?\.(?:go|vulcan))(?=[:,\s)]|$)")
PLACEHOLDER = re.compile(r"\{[a-z_0-9]+\}")
SKIP_DIRS = {".vulcan", "testdata", "vendor", ".git"}


class Abort(Exception):
    """The run cannot go on. The raw records so far are still sealed."""


# ---------------------------------------------------------------- config

def load_config(path):
    """Read a JSON file with full-line // comments (the intended Windows values stay as comments)."""
    with open(path, encoding="utf-8") as f:
        lines = [ln for ln in f.read().split("\n") if not ln.lstrip().startswith("//")]
    cfg = json.loads("\n".join(lines))
    cfg["_path"] = os.path.abspath(path)
    cfg["_sha256"] = hashlib.sha256(open(path, "rb").read()).hexdigest()
    return cfg


def subst(x, ctx):
    """Replace {name} in strings, lists and dicts. A name that ctx lacks stays, and is checked later."""
    if isinstance(x, str):
        for k, v in ctx.items():
            x = x.replace("{" + k + "}", str(v))
        return x
    if isinstance(x, list):
        return [subst(i, ctx) for i in x]
    if isinstance(x, dict):
        return {k: subst(v, ctx) for k, v in x.items()}
    return x


def splice(items, sets):
    """Replace each "@name" item of a list by the items of sets[name]. An item {"@": name, key: value, ...} does the
    same and adds the keys to each spliced item (the same tools, with a cwd of their own)."""
    out = []
    for it in items:
        if isinstance(it, str) and it.startswith("@"):
            out.extend(splice(sets[it[1:]], sets))
        elif isinstance(it, dict) and "@" in it:
            extra = {k: v for k, v in it.items() if k != "@"}
            out.extend(dict(t, **extra) for t in splice(sets[it["@"]], sets))
        else:
            out.append(it)
    return out


def arm_names(cfg):
    return sorted(cfg["arms"])


def arm_ctx(cfg, arm):
    work = cfg["work_root"]
    armdir = os.path.join(work, "arms", arm)
    ctx = {
        "here": HERE, "repo": cfg.get("repo", ""), "work": work, "arm": arm, "armdir": armdir,
        "tree": os.path.join(armdir, "tree"), "cache": os.path.join(armdir, "cache"),
        "tools": cfg.get("tools_dir", ""), "gobin": cfg.get("gobin", ""), "goroot": cfg.get("goroot", ""),
    }
    return {k: str(v) for k, v in ctx.items()}


def arm_cfg(cfg, arm):
    """The arm's section with set references spliced and placeholders filled."""
    ctx = arm_ctx(cfg, arm)
    a = dict(cfg["arms"][arm])
    sets = cfg.get("sets", {})
    for key in ("check", "closed_list", "program_paths"):
        a[key] = splice(a.get(key, []), sets)
    a = subst(a, ctx)
    a["_ctx"] = ctx
    return a


def arm_env(cfg, arm, setup=False):
    ctx = arm_ctx(cfg, arm)
    env = dict(os.environ)
    for k in cfg.get("unset_env", []):
        env.pop(k, None)
    env.update(subst(cfg.get("env", {}), ctx))
    env.update(subst(cfg["arms"][arm].get("env", {}), ctx))
    if setup:
        env.update(subst(cfg.get("setup_env", {}), ctx))
    return env


def check_resolved(tokens):
    for t in tokens:
        if PLACEHOLDER.search(t):
            raise Abort("unresolved placeholder in command: " + t)


def expand(tokens, ctx, files=None, pkgs=None):
    """Fill a command template. A token {files} or {pkgs} becomes several arguments."""
    out = []
    for t in tokens:
        if t == "{files}":
            out.extend(files or [])
        elif t == "{pkgs}":
            out.extend(pkgs or [])
        else:
            out.append(subst(t, ctx))
    check_resolved(out)
    return out


# ---------------------------------------------------------------- files

def _force(fn, p, exc):
    os.chmod(p, stat.S_IWRITE)
    fn(p)


def rmtree(path):
    if os.path.lexists(path):
        shutil.rmtree(path, onexc=_force)


def copytree(src, dst):
    shutil.copytree(src, dst, ignore=shutil.ignore_patterns(".vulcan", "*.db", "__pycache__"))


def sha256_file(path):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(READ_CHUNK), b""):
            h.update(chunk)
    return h.hexdigest()


def go_files(tree):
    out = []
    for d, dirs, names in os.walk(tree):
        dirs[:] = sorted(x for x in dirs if x not in SKIP_DIRS)
        for n in sorted(names):
            if n.endswith(".go"):
                out.append(os.path.relpath(os.path.join(d, n), tree).replace("\\", "/"))
    return out


def robocopy(src, dst):
    p = subprocess.run(["robocopy", src, dst] + ROBOCOPY_FLAGS, capture_output=True)
    if p.returncode >= ROBOCOPY_FAIL_MIN:
        raise Abort("robocopy failed (%d) for %s" % (p.returncode, os.path.basename(dst)))


def snap_path(armdir, i, name):
    safe = re.sub(r"[^A-Za-z0-9]+", "_", name).strip("_")
    return os.path.join(armdir, "snapshot", "%02d_%s" % (i, safe))


def snapshot(a):
    """Copy every entry of the closed list to the snapshot. An entry that does not exist is recorded as absent."""
    for i, e in enumerate(a["closed_list"]):
        s = snap_path(a["_ctx"]["armdir"], i, e["name"])
        rmtree(s)
        if os.path.isdir(e["path"]):
            robocopy(e["path"], s)


def restore(a):
    """Step 1: put each cache of the closed list back to the snapshot."""
    for i, e in enumerate(a["closed_list"]):
        s = snap_path(a["_ctx"]["armdir"], i, e["name"])
        if os.path.isdir(s):
            robocopy(s, e["path"])
        else:
            rmtree(e["path"])


def read_program_files(paths):
    """Step 3: read every file of the closed list of program files, and run nothing. Returns (files, bytes, errors)."""
    n = size = errs = 0
    for p in paths:
        if os.path.isdir(p):
            todo = (os.path.join(d, f) for d, _, fs in os.walk(p) for f in fs)
        else:
            todo = [p]
        for f in todo:
            try:
                with open(f, "rb") as fh:
                    while True:
                        b = fh.read(READ_CHUNK)
                        if not b:
                            break
                        size += len(b)
                n += 1
            except OSError:
                errs += 1
    return n, size, errs


# ---------------------------------------------------------------- tools and findings

def run_cmd(cmd, cwd, env, timeout):
    """Run one command. Returns (status, output bytes, duration ns). Never raises for a tool failure."""
    t0 = time.perf_counter_ns()
    try:
        p = subprocess.run(cmd, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=timeout)
        status, out = p.returncode, p.stdout
    except subprocess.TimeoutExpired as e:
        status, out = STATUS_TIMEOUT, e.stdout or b""
    except OSError as e:
        status, out = STATUS_LAUNCH, str(e).encode()
    return status, out, time.perf_counter_ns() - t0


def normalize(output, tree, cwd=None):
    """The findings of a tool output as a sorted list of lines.

    One filter rule (plan rule 2): a line about a file whose first line is the standard
    `// Code generated ... DO NOT EDIT.` header is dropped, and the count dropped is returned.
    Line and column numbers are removed, because a one-file change shifts them.
    Relative paths in the output are against cwd (the directory the tool ran in; the tree when None).
    Returns (findings, dropped).
    """
    text = output.decode("utf-8", "replace").replace("\r", "").replace("\\", "/")
    root = tree.replace("\\", "/").rstrip("/")
    base = (cwd or tree).replace("\\", "/").rstrip("/")
    kept, dropped, header = [], 0, {}
    for ln in text.split("\n"):
        ln = ln.strip()
        if not ln or ln.startswith("go: downloading "):  # module fetch notes from the go command, not findings
            continue
        m = FILE_TOKEN.search(ln)
        if m:
            p = m.group("p")
            full = p if re.match(r"^([A-Za-z]:)?/", p) else base + "/" + p
            if full not in header:
                try:
                    with open(full, encoding="utf-8", errors="replace") as f:
                        header[full] = bool(GENERATED.match(f.readline()))
                except OSError:
                    header[full] = False
            if header[full]:
                dropped += 1
                continue
        ln = re.sub(re.escape(root), "<tree>", ln, flags=re.I)
        kept.append(POSITION.sub(r"\1", ln))
    return sorted(kept), dropped


def tool_cwd(tool, tree):
    """Where a tool runs: its own "cwd" (V's lowered module, which has its own go.mod), else the work tree."""
    return tool.get("cwd", tree)


def tool_args(tool, scope, files, tree):
    """The files and packages that a tool gets: the change (scope "change") or the whole module ("full").
    Both are relative to the cwd of the tool. The files of a change are relative to the tree, so a tool with
    a cwd gets them made relative to it, and its packages are those of the made-relative files."""
    cwd = tool_cwd(tool, tree)
    if scope == "full":
        return go_files(cwd), ["./..."]
    if os.path.normcase(os.path.abspath(cwd)) == os.path.normcase(os.path.abspath(tree)):
        return files["files"], files["pkgs"]
    prefix = os.path.relpath(cwd, tree).replace("\\", "/").rstrip("/") + "/"
    rel = []
    for f in files["files"]:
        f = f.replace("\\", "/")
        if not f.startswith(prefix):
            raise Abort("a file of the change is outside the cwd of tool " + tool["name"])
        rel.append(f[len(prefix):])
    return rel, pkgs_of(rel)


def tool_command(tool, ctx, tree, change_files, scope):
    """The command of a tool and the directory it runs in. One place for the run, the baseline key and the trace."""
    files, pkgs = tool_args(tool, scope, change_files, tree)
    return expand(tool["cmd"], ctx, files, pkgs), tool_cwd(tool, tree)


def run_tool(tool, ctx, env, tree, change_files, scope, timeout):
    cmd, cwd = tool_command(tool, ctx, tree, change_files, scope)
    status, out, dur = run_cmd(cmd, cwd, env, timeout)
    return {"name": tool["name"], "unit": tool["unit"], "cmd": cmd, "cwd": cwd, "status": status, "output": out, "ns": dur}


def rendered_key(tool, cmd):
    return tool["name"] + "\0" + "\0".join(cmd)


# ---------------------------------------------------------------- changes

def pkgs_of(files):
    """The package directories of Go files, as ./dir (or . for the root)."""
    return sorted({"./" + os.path.dirname(f) if os.path.dirname(f) else "." for f in files})


def load_changes(path):
    with open(path, encoding="utf-8") as f:
        d = json.load(f)
    base = os.path.dirname(os.path.abspath(path))
    for c in d["changes"]:
        for arm, spec in c["arms"].items():
            spec["patch"] = os.path.join(base, spec["patch"])
            # The files that the tools read: the changed Go file, or the lowered Go of a .vulcan file.
            spec.setdefault("files", spec.get("lowered", [spec["file"]]))
            spec.setdefault("pkgs", pkgs_of(spec["files"]))
    return d["changes"]


def git_apply(tree, patch, reverse=False, check=False):
    cmd = ["git", "-c", "core.autocrlf=false", "-c", "core.eol=lf", "apply", "-p1", "--whitespace=nowarn"]
    if reverse:
        cmd.append("-R")
    if check:
        cmd.append("--check")
    cmd.append(patch)
    p = subprocess.run(cmd, cwd=tree, capture_output=True)
    return p.returncode == 0


def tree_hash(tree, exclude=()):
    """Hash of every file of the tree except the listed paths (the build outputs of the closed list). Proves a revert."""
    skip = {os.path.normcase(os.path.abspath(p)) for p in exclude}
    h = hashlib.sha256()
    for d, dirs, names in os.walk(tree):
        dirs[:] = sorted(x for x in dirs if os.path.normcase(os.path.join(d, x)) not in skip)
        for n in sorted(names):
            p = os.path.join(d, n)
            h.update(os.path.relpath(p, tree).replace("\\", "/").encode())
            h.update(sha256_file(p).encode())
    return h.hexdigest()


# ---------------------------------------------------------------- load gate

def load_gate(cfg, raw, arm):
    """Plan rule 6: before a run, sample the machine every sample_s. The run may start when the total CPU use is
    at most cpu_max_pct and the free memory is at least mem_free_min_gb. It waits up to wait_max_s.
    A run that never passes is a fault (`load-gate`) and the caller must not run it.
    Returns the record that is kept with the run: passed, waited_s and every sample. Call it outside the timed interval."""
    g = cfg["load_gate"]
    samples = []
    t0 = time.monotonic()
    while True:
        cpu = psutil.cpu_percent(interval=g["sample_s"])
        free = psutil.virtual_memory().available / BYTES_PER_GB
        samples.append({"cpu_pct": cpu, "free_gb": free})
        passed = cpu <= g["cpu_max_pct"] and free >= g["mem_free_min_gb"]
        waited = time.monotonic() - t0
        if passed or waited >= g["wait_max_s"]:
            break
    if not passed:
        raw.fault(arm, "load-gate")
    return {"passed": passed, "waited_s": waited, "samples": samples}


# ---------------------------------------------------------------- plan

def make_plan(seed, batch, change_ids, arms, n):
    """The draws and the arm order of each draw. Same seed and batch give the same plan."""
    rng = random.Random("%s/%s" % (seed, batch))
    plan = []
    for _ in range(n):
        item = rng.choice(change_ids)
        order = list(arms)
        rng.shuffle(order)
        plan.append((item, order))
    return plan


# ---------------------------------------------------------------- raw records, seal, output

class Raw:
    """The raw directory of one command. Everything that carries an arm's direction goes here."""

    def __init__(self, cfg, label):
        self.cfg, self.label = cfg, label
        self.dir = os.path.join(cfg["work_root"], "raw", label)
        self.sealed = os.path.join(cfg["work_root"], "sealed", label + ".sealed")
        if os.path.exists(self.dir) or os.path.exists(self.sealed):
            raise Abort("label already used: " + label)
        os.makedirs(self.dir)
        os.makedirs(os.path.dirname(self.sealed), exist_ok=True)
        self.n = 0
        self.faults = set()
        self.wall_ns = 0

    def write(self, rel, data):
        p = os.path.join(self.dir, rel)
        os.makedirs(os.path.dirname(p), exist_ok=True)
        mode = "wb" if isinstance(data, bytes) else "w"
        with open(p, mode, **({} if mode == "wb" else {"encoding": "utf-8", "newline": "\n"})) as f:
            f.write(data)

    def json(self, rel, obj):
        self.write(rel, json.dumps(obj, indent=1, sort_keys=True) + "\n")

    def fault(self, arm, kind, what=""):
        self.faults.add("%s %s%s" % (arm, kind, (" " + what) if what else ""))

    def next_index(self):
        self.n += 1
        return self.n

    def seal(self, meta):
        if self.faults:
            self.json("faults.json", sorted(self.faults))
        files = []
        for d, _, names in os.walk(self.dir):
            for n in sorted(names):
                p = os.path.join(d, n)
                files.append({"path": os.path.relpath(p, self.dir).replace("\\", "/"), "bytes": os.path.getsize(p), "sha256": sha256_file(p)})
        files.sort(key=lambda x: x["path"])
        meta = dict(meta, label=self.label, runs=self.n, config_sha256=self.cfg["_sha256"],
                    created_utc=time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), files=files)
        # The manifest is written last, so it is not in its own file list.
        self.write("MANIFEST.json", json.dumps(meta, indent=1, sort_keys=True) + "\n")
        seal_dir(self.cfg, self.dir, self.sealed)
        rmtree(self.dir)


def harness_exe(cfg):
    return subst(cfg["harness"], {"repo": cfg.get("repo", ""), "here": HERE})


def _harness(cfg, args):
    if not os.environ.get("VULCAN_SEAL_KEY"):
        raise Abort("VULCAN_SEAL_KEY is not set")
    # The key stays in the environment of the child. It is never printed, and the child output is dropped.
    p = subprocess.run([harness_exe(cfg)] + args, env=os.environ, capture_output=True)
    if p.returncode != 0:
        raise Abort("harness %s failed (exit %d)" % (args[0], p.returncode))


def open_sealed(cfg, sealed, dest):
    """Open a sealed file into the directory dest (the key is read from VULCAN_SEAL_KEY by the harness)."""
    _harness(cfg, ["open", "-in", sealed, "-dir", dest])


def seal_dir(cfg, d, out):
    _harness(cfg, ["seal", "-dir", d, "-out", out])
    if not os.path.isfile(out) or os.path.getsize(out) == 0:
        raise Abort("no sealed file written")
    tmp = tempfile.mkdtemp(prefix="vt-open-", dir=cfg["work_root"])
    try:
        open_sealed(cfg, out, tmp)
        mine = sorted(os.path.relpath(os.path.join(a, f), d) for a, _, fs in os.walk(d) for f in fs)
        back = sorted(os.path.relpath(os.path.join(a, f), tmp) for a, _, fs in os.walk(tmp) for f in fs)
        if mine != back:
            raise Abort("the sealed file does not open to the same files")
    finally:
        rmtree(tmp)


def preflight_seal(cfg):
    """Seal and open a tiny directory before any run, so a bad key or harness fails before hours of work."""
    os.makedirs(cfg["work_root"], exist_ok=True)
    tmp = tempfile.mkdtemp(prefix="vt-pre-", dir=cfg["work_root"])
    try:
        os.makedirs(os.path.join(tmp, "in"))
        with open(os.path.join(tmp, "in", "x.txt"), "w", newline="\n") as f:
            f.write("x\n")
        seal_dir(cfg, os.path.join(tmp, "in"), os.path.join(tmp, "pre.sealed"))
    finally:
        rmtree(tmp)


ALLOWED_OUTPUT = [
    re.compile(r"^runs: \d+$"),
    re.compile(r"^faults: none$"),
    re.compile(r"^fault: [A-Za-z0-9_ ./:=-]+$"),
    re.compile(r"^row15_total_wall_s: \d+\.\d$"),
    re.compile(r"^row19_total_wall_s: \d+\.\d$"),
    re.compile(r"^trace_total_wall_s: \d+\.\d$"),
    re.compile(r"^setup: done$"),
]


def emit(lines):
    """Print lines. A line that is not of an allowed shape stops the harness, so nothing else can leak."""
    for ln in lines:
        if not any(r.match(ln) for r in ALLOWED_OUTPUT):
            raise Abort("an output line of a disallowed shape was withheld")
    for ln in lines:
        print(ln)


def report(raw, total_key=None):
    lines = ["runs: %d" % raw.n]
    lines += ["fault: " + f for f in sorted(raw.faults)] or ["faults: none"]
    if total_key:
        lines.append("%s: %.1f" % (total_key, raw.wall_ns / 1e9))
    emit(lines)


def parse_trace_csv(path, tree, root_pid, cache_dirs):
    """Count what a traced process tree opened for reading. Returns (source files opened, cache files opened).

    Columns are those of a Process Monitor CSV export: Process Name, PID, Operation, Path, Result, Detail.
    A source read is an open (CreateFile, SUCCESS, a read access) of a *.go or *.vulcan path under the
    working tree, once for each path. A cache read is the same for any file under a cache directory.
    The tree of processes is the root process and every process it creates (Process Create rows).
    """
    low = lambda s: s.replace("/", "\\").lower().rstrip("\\")
    tree_l, cdirs = low(tree) + "\\", [low(c) + "\\" for c in cache_dirs]
    rows = []
    with open(path, newline="", encoding="utf-8-sig") as f:
        for r in csv.DictReader(f):
            rows.append(r)
    pids = {str(root_pid)}
    changed = True
    while changed:
        changed = False
        for r in rows:
            if r["Operation"] == "Process Create" and r["PID"] in pids:
                m = re.search(r"PID:\s*(\d+)", r.get("Detail", ""))
                if m and m.group(1) not in pids:
                    pids.add(m.group(1))
                    changed = True
    src, cache = set(), set()
    for r in rows:
        if r["Operation"] != "CreateFile" or r["PID"] not in pids or r["Result"] != "SUCCESS":
            continue
        det = r.get("Detail", "")
        if "Generic Read" not in det and "Read Data" not in det:
            continue
        p = low(r["Path"])
        if p.startswith(tree_l) and (p.endswith(".go") or p.endswith(".vulcan")):
            src.add(p)
        if any(p.startswith(c) for c in cdirs):
            cache.add(p)
    return len(src), len(cache)
