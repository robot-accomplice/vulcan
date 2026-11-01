package rules

import (
	"crypto/tls"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// GoVersion is the Go version that init writes in a new go.mod: the current stable toolchain.
const GoVersion = "1.27.3"

// DefaultsName names the built-in ruleset of this tool.
const DefaultsName = "defaults v1"

//go:embed defaults/v1.rules
var defaultsV1 []byte

// Defaults returns the built-in ruleset.
func Defaults() []byte { return defaultsV1 }

// maxSource bounds what a source can be.
const maxSource = 8 << 20

// Client reads an https source. A test replaces it.
var Client = &http.Client{
	Timeout:   30 * time.Second,
	Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return errors.New("a source redirects away from https")
		}
		if len(via) >= 5 {
			return errors.New("a source redirects too many times")
		}
		return nil
	},
}

// Tool is what differs between the commands of vulcan and vulcan-lint.
type Tool struct {
	Name     string
	FindRoot func(dir string) (string, error)          // the root of a project, from a directory in it
	Scaffold func(root string, stdout io.Writer) error // creates what the project lacks
	Edges    func(root string) ([]Edge, error)         // the current wires and imports, for the lock
	Findings func(root string) ([]Entry, error)        // each current rule finding, for the baseline
}

// IsDefaults reports whether an origin is a built-in ruleset.
func IsDefaults(origin string) bool { return strings.HasPrefix(origin, "defaults ") }

// IsURL reports whether a source is a URL.
func IsURL(source string) bool { return strings.Contains(source, "://") }

// ReadSource reads a local file or an https URL. Nothing else reads a source, and only init and
// rules upgrade call it.
func ReadSource(source, base string) ([]byte, error) {
	if IsURL(source) {
		u, err := url.Parse(source)
		if err != nil {
			return nil, err
		}
		if u.Scheme != "https" {
			return nil, fmt.Errorf("the source %s is not https; a source is a local file or an https URL", source)
		}
		resp, err := Client.Get(source)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("the source %s answers %s", source, resp.Status)
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, maxSource+1))
		if err != nil {
			return nil, err
		}
		if len(b) > maxSource {
			return nil, fmt.Errorf("the source %s is larger than %d bytes", source, maxSource)
		}
		return b, nil
	}
	if !filepath.IsAbs(source) {
		source = filepath.Join(base, filepath.FromSlash(source))
	}
	return os.ReadFile(source)
}

// parseSource reads a source as rules. Its own head does not count, because the project records
// where it read the rules.
func parseSource(b []byte, name string) (*File, error) {
	f, probs := Parse(b)
	if len(probs) > 0 {
		var msgs []string
		for _, p := range probs {
			msgs = append(msgs, fmt.Sprintf("%s:%d: %s", name, p.Line, p.Msg))
		}
		return nil, errors.New(strings.Join(msgs, "\n"))
	}
	return f, nil
}

// withHead gives the text of a source with the head of a project.
func withHead(src []byte, origin, hash string) string {
	var out []string
	head := fmt.Sprintf("origin %s\nhash %s\nlists %s", origin, hash, Lists())
	done := false
	for _, line := range strings.Split(string(Normalize(src)), "\n") {
		key, _, _ := strings.Cut(strings.TrimSpace(line), " ")
		if key == "origin" || key == "hash" || key == "lists" {
			if !done {
				out, done = append(out, head), true
			}
			continue
		}
		out = append(out, line)
	}
	text := strings.Join(out, "\n")
	if !done {
		text = head + "\n\n" + text
	}
	return text
}

// Init is the command init [--rules SOURCE] [dir].
func (t Tool) Init(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	source := fs.String("rules", "", "read the rules from this local file or https URL, not from "+DefaultsName)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dir := "."
	if fs.NArg() > 1 {
		fmt.Fprintf(stderr, "%s init: too many arguments\n", t.Name)
		return 2
	}
	if fs.NArg() == 1 {
		dir = fs.Arg(0)
	}
	root, err := filepath.Abs(dir)
	if err == nil {
		err = os.MkdirAll(root, 0o755)
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s init: %v\n", t.Name, err)
		return 1
	}
	if _, err := os.Stat(filepath.Join(root, FileName)); err == nil {
		fmt.Fprintf(stderr, "%s init: %s exists, and the team owns it; %s init does not replace it (see %s rules upgrade)\n", t.Name, FileName, t.Name, t.Name)
		return 1
	}
	origin := DefaultsName
	src := Defaults()
	if *source != "" {
		cwd, _ := os.Getwd()
		src, err = ReadSource(*source, cwd)
		if err != nil {
			fmt.Fprintf(stderr, "%s init: %v\n", t.Name, err)
			return 1
		}
		origin = originName(*source, cwd, root)
	}
	if _, err := parseSource(src, sourceLabel(*source)); err != nil {
		fmt.Fprintf(stderr, "%s init: %v\n", t.Name, err)
		return 1
	}
	if err := t.Scaffold(root, stdout); err != nil {
		fmt.Fprintf(stderr, "%s init: %v\n", t.Name, err)
		return 1
	}
	hash := Hash(src)
	if err := writeFiles(root, withHead(src, origin, hash), src); err != nil {
		fmt.Fprintf(stderr, "%s init: %v\n", t.Name, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s and %s: origin %s, hash %s\n", FileName, OriginName, origin, hash)
	return 0
}

func sourceLabel(source string) string {
	if source == "" {
		return DefaultsName
	}
	return source
}

// originName is how the head names a source: a local file relative to the project when it is under
// the project, so that the project can move.
func originName(source, cwd, root string) string {
	if IsURL(source) {
		return source
	}
	abs := source
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(cwd, filepath.FromSlash(abs))
	}
	if rel, err := filepath.Rel(root, abs); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(abs)
}

func writeFiles(root, rulesText string, origin []byte) error {
	if err := os.WriteFile(filepath.Join(root, FileName), []byte(rulesText), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, OriginName), Normalize(origin), 0o644)
}

// Rules is the command rules diff|upgrade|lock.
func (t Tool) Rules(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintf(stderr, "%s rules: the commands are rules diff, rules upgrade, rules lock, and rules baseline\n", t.Name)
		return 2
	}
	switch args[0] {
	case "diff":
		return t.diff(args[1:], stdout, stderr)
	case "upgrade":
		return t.upgrade(args[1:], stdout, stderr)
	case "lock":
		return t.lock(args[1:], stdout, stderr)
	case "baseline":
		return t.baseline(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "%s rules: unknown command %q; the commands are rules diff, rules upgrade, rules lock, and rules baseline\n", t.Name, args[0])
	return 2
}

func (t Tool) rootOf(args []string, stderr io.Writer) (string, bool) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	root, err := t.FindRoot(dir)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", t.Name, err)
		return "", false
	}
	return root, true
}

// readProject reads the rules file and the origin, and checks the origin against its hash.
func readProject(root string) (*File, *File, []byte, error) {
	b, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("no %s in %s; run the command init", FileName, root)
	}
	f, probs := ParseProject(b)
	if len(probs) > 0 {
		return nil, nil, nil, fmt.Errorf("%s:%d: %s", FileName, probs[0].Line, probs[0].Msg)
	}
	ob, err := os.ReadFile(filepath.Join(root, OriginName))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("no %s in %s, so the rules have nothing to compare with", OriginName, root)
	}
	if Hash(ob) != f.Hash {
		return nil, nil, nil, fmt.Errorf("%s has the hash %s, but the head of %s records %s", OriginName, Hash(ob), FileName, f.Hash)
	}
	of, err := parseSource(ob, OriginName)
	if err != nil {
		return nil, nil, nil, err
	}
	return f, of, ob, nil
}

// paramChanges says how two forms of one rule differ.
func paramChanges(a, b *Rule) []string {
	var out []string
	if a.Level != b.Level {
		out = append(out, fmt.Sprintf("level %s -> %s", a.Level, b.Level))
	}
	if a.Kind != b.Kind {
		out = append(out, fmt.Sprintf("kind %s -> %s", a.Kind, b.Kind))
	}
	names := map[string]bool{}
	for k := range a.Params {
		names[k] = true
	}
	for k := range b.Params {
		names[k] = true
	}
	var keys []string
	for k := range names {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		x, y := strings.Join(a.Params[k], " "), strings.Join(b.Params[k], " ")
		if x != y {
			out = append(out, fmt.Sprintf("%s: %s -> %s", k, x, y))
		}
	}
	if a.Reason != b.Reason {
		out = append(out, fmt.Sprintf("reason: %s -> %s", a.Reason, b.Reason))
	}
	if a.Source != b.Source {
		out = append(out, fmt.Sprintf("source: %s -> %s", a.Source, b.Source))
	}
	return out
}

func (t Tool) diff(args []string, stdout, stderr io.Writer) int {
	root, ok := t.rootOf(args, stderr)
	if !ok {
		return 1
	}
	f, of, _, err := readProject(root)
	if err != nil {
		fmt.Fprintf(stderr, "%s rules diff: %v\n", t.Name, err)
		return 1
	}
	fmt.Fprintf(stdout, "origin %s\nhash %s\n", f.Origin, f.Hash)
	if f.Lists != Lists() {
		fmt.Fprintf(stdout, "lists %s here, and %s in this tool: a rule can enforce more or less than when the head was written\n", f.Lists, Lists())
	}
	n := 0
	for _, r := range of.Rules {
		p := f.Find(r.ID)
		switch {
		case p == nil:
			n++
			fmt.Fprintf(stdout, "- rule %s: deleted by the team\n", r.ID)
		case !Equal(r, p):
			n++
			what := "changed by the team"
			if p.Level == Off && r.Level != Off {
				what = "turned off by the team"
			}
			fmt.Fprintf(stdout, "~ rule %s: %s: %s\n", r.ID, what, strings.Join(paramChanges(r, p), "; "))
		}
	}
	for _, r := range f.Rules {
		if of.Find(r.ID) == nil {
			n++
			fmt.Fprintf(stdout, "+ rule %s: added by the team\n", r.ID)
		}
	}
	if n == 0 {
		fmt.Fprintln(stdout, "the rules are the same as their origin")
	}
	if base, _ := ReadBaseline(root); base != nil {
		findings, keys := base.Size()
		fmt.Fprintf(stdout, "%s holds %d findings, in %d keys: a check does not fail on these\n", BaselineName, findings, keys)
	} else {
		fmt.Fprintf(stdout, "no %s\n", BaselineName)
	}
	return 0
}

// baseline writes each current rule finding to the file. It refuses a project with a model error or a
// fault in its rules, because no model check goes in the file.
func (t Tool) baseline(args []string, stdout, stderr io.Writer) int {
	root, ok := t.rootOf(args, stderr)
	if !ok {
		return 1
	}
	if _, err := os.Stat(filepath.Join(root, FileName)); err != nil {
		fmt.Fprintf(stderr, "%s rules baseline: no %s in %s; run the command init\n", t.Name, FileName, root)
		return 1
	}
	entries, err := t.Findings(root)
	if err != nil {
		fmt.Fprintf(stderr, "%s rules baseline: %v\n", t.Name, err)
		return 1
	}
	n, err := WriteBaseline(root, entries)
	if err != nil {
		fmt.Fprintf(stderr, "%s rules baseline: %v\n", t.Name, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s: %d findings\n", BaselineName, n)
	return 0
}

// change is one step of an upgrade.
type change struct {
	line  string
	id    string
	apply string // "add", "replace", "remove", or "" when the team's form stays
}

func planUpgrade(f, of, nf *File) []change {
	var out []change
	for _, r := range nf.Rules {
		old, mine := of.Find(r.ID), f.Find(r.ID)
		switch {
		case old == nil && mine == nil:
			out = append(out, change{"+ rule " + r.ID + ": new", r.ID, "add"})
		case old == nil:
			out = append(out, change{"= rule " + r.ID + ": new in the source, and the team already has a rule with this id, so the team's rule stays", r.ID, ""})
		case Equal(old, r):
		case mine == nil:
			out = append(out, change{"= rule " + r.ID + ": changed in the source, but the team deleted it, so it stays deleted: " + strings.Join(paramChanges(old, r), "; "), r.ID, ""})
		case Equal(old, mine):
			out = append(out, change{"~ rule " + r.ID + ": " + strings.Join(paramChanges(old, r), "; "), r.ID, "replace"})
		default:
			out = append(out, change{"= rule " + r.ID + ": changed in the source, but the team changed it too, so the team's rule stays: " + strings.Join(paramChanges(old, r), "; "), r.ID, ""})
		}
	}
	for _, old := range of.Rules {
		if nf.Find(old.ID) != nil {
			continue
		}
		mine := f.Find(old.ID)
		switch {
		case mine == nil:
		case Equal(old, mine):
			out = append(out, change{"- rule " + old.ID + ": removed from the source", old.ID, "remove"})
		default:
			out = append(out, change{"= rule " + old.ID + ": removed from the source, but the team changed it, so the team's rule stays", old.ID, ""})
		}
	}
	return out
}

// applyUpgrade changes only the blocks that the plan names, so each comment of the team stays.
func applyUpgrade(f, nf *File, plan []change, origin, hash string) string {
	lines := slices.Clone(f.lines)
	type edit struct {
		start, end int
		text       string
	}
	var edits []edit
	var added []string
	for _, c := range plan {
		switch c.apply {
		case "add":
			added = append(added, nf.block(nf.Find(c.id)))
		case "replace":
			r := f.Find(c.id)
			edits = append(edits, edit{r.start, r.end, strings.TrimSuffix(nf.block(nf.Find(c.id)), "\n")})
		case "remove":
			r := f.Find(c.id)
			edits = append(edits, edit{r.start, r.end, "\x00"})
		}
	}
	slices.SortFunc(edits, func(a, b edit) int { return b.start - a.start })
	for _, e := range edits {
		repl := []string{e.text}
		if e.text == "\x00" {
			repl = nil
			if e.end < len(lines) && strings.TrimSpace(lines[e.end]) == "" {
				e.end++ // the blank line after the block
			}
		}
		lines = slices.Replace(lines, e.start, e.end, repl...)
	}
	hasLists := false
	for i, line := range lines {
		key, _, _ := strings.Cut(strings.TrimSpace(line), " ")
		if key == "origin" {
			lines[i] = "origin " + origin
		}
		if key == "hash" {
			lines[i] = "hash " + hash
		}
		if key == "lists" {
			lines[i], hasLists = "lists "+Lists(), true
		}
		if key == "rule" {
			break
		}
	}
	if !hasLists {
		if i := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), "hash ") }); i >= 0 {
			lines = slices.Insert(lines, i+1, "lists "+Lists())
		}
	}
	text := strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
	for _, a := range added {
		text += "\n" + a
	}
	return text
}

func (t Tool) upgrade(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("rules upgrade", flag.ContinueOnError)
	fs.SetOutput(stderr)
	source := fs.String("rules", "", "read the newer rules from this local file or https URL, not from the origin")
	accept := fs.String("accept", "", "apply the upgrade to the ruleset with this hash, which the command printed")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root, ok := t.rootOf(fs.Args(), stderr)
	if !ok {
		return 1
	}
	fail := func(err error) int {
		fmt.Fprintf(stderr, "%s rules upgrade: %v\n", t.Name, err)
		return 1
	}
	f, of, _, err := readProject(root)
	if err != nil {
		return fail(err)
	}
	origin := f.Origin
	var src []byte
	switch {
	case *source != "":
		cwd, _ := os.Getwd()
		if src, err = ReadSource(*source, cwd); err != nil {
			return fail(err)
		}
		origin = originName(*source, cwd, root)
	case IsDefaults(origin):
		src, origin = Defaults(), DefaultsName
	default:
		if src, err = ReadSource(origin, root); err != nil {
			return fail(err)
		}
	}
	nf, err := parseSource(src, sourceLabel(*source))
	if err != nil {
		return fail(err)
	}
	hash := Hash(src)
	fmt.Fprintf(stdout, "from %s %s\nto %s %s\n", f.Origin, f.Hash, origin, hash)
	listsChanged := f.Lists != Lists()
	if listsChanged {
		fmt.Fprintf(stdout, "the tool's lists changed, so the same rules can enforce more or less: lists %s here, and %s in this tool\n", f.Lists, Lists())
	}
	if hash == f.Hash && origin == f.Origin && !listsChanged {
		fmt.Fprintln(stdout, "the source has not changed")
		return 0
	}
	plan := planUpgrade(f, of, nf)
	for _, c := range plan {
		fmt.Fprintln(stdout, c.line)
	}
	if *accept == "" {
		fmt.Fprintf(stdout, "nothing changed; to accept: %s rules upgrade", t.Name)
		if *source != "" {
			fmt.Fprintf(stdout, " --rules %s", *source)
		}
		fmt.Fprintf(stdout, " --accept %s\n", hash)
		return 0
	}
	if *accept != hash {
		return fail(fmt.Errorf("the source now has the hash %s, not %s; read this difference, and accept that hash", hash, *accept))
	}
	if err := writeFiles(root, applyUpgrade(f, nf, plan, origin, hash), src); err != nil {
		return fail(err)
	}
	fmt.Fprintf(stdout, "accepted: %s and %s now have origin %s, hash %s\n", FileName, OriginName, origin, hash)
	return 0
}

const lockHead = `# The locked map of dependencies. The command rules lock writes this file. With a depends rule
# that has map locked, a dependency that this file does not hold fails the check.
`

// IsRuntimeLib reports a part that no dependency counts: L's contract package, which V has in its
// runtime.
func IsRuntimeLib(dir string) bool { return dir == "lib/contract" }

func (t Tool) lock(args []string, stdout, stderr io.Writer) int {
	root, ok := t.rootOf(args, stderr)
	if !ok {
		return 1
	}
	edges, err := t.Edges(root)
	if err != nil {
		fmt.Fprintf(stderr, "%s rules lock: %v\n", t.Name, err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(root, LockName), []byte(FormatLock(edges)), 0o644); err != nil {
		fmt.Fprintf(stderr, "%s rules lock: %v\n", t.Name, err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", LockName)
	return 0
}

// FormatLock writes a lock, in a fixed order, so that a change to it is a small diff.
func FormatLock(edges []Edge) string {
	seen := map[string]bool{}
	var keys []string
	for _, e := range edges {
		if k := e.key(); !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	slices.Sort(keys)
	return lockHead + strings.Join(keys, "\n") + "\n"
}

// ReadLock reads a lock.
func ReadLock(root string) (map[string]bool, error) {
	b, err := os.ReadFile(filepath.Join(root, LockName))
	if err != nil {
		return nil, fmt.Errorf("there is no %s", LockName)
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(Normalize(b)), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			out[line] = true
		}
	}
	return out, nil
}
