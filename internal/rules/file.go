// Package rules is the rules file of plan-r3, section 2: its format, the commands init, rules
// diff, rules upgrade, and rules lock, and the rule kinds forbid-call, only-in, depends, and
// transitions. `vulcan` and `vulcan-lint` both call this package, so a rule means the same thing
// in each. Rules are data: nothing in a rules file runs. A check reads only the rules file and
// the lock of its own project, and never a source.
package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

// The files of a project, at its root.
const (
	FileName   = "rules.vulcan" // the rules that the team owns, and that the check reads
	OriginName = "rules.origin" // the source as it was read; only diff and upgrade read it
	LockName   = "rules.lock"   // the locked map of dependencies
)

// The levels of a rule.
const (
	Error = "error"
	Warn  = "warn"
	Off   = "off"
)

// Rule is one rule of a rules file.
type Rule struct {
	ID     string
	Kind   string
	Level  string
	Params map[string][]string
	Reason string
	Source string
	Line   int // of the rule line, from 1

	start, end int // the lines of the block, with its comments: [start, end), from 0
}

// Problem is a rules file that the check cannot read.
type Problem struct {
	File string `json:",omitempty"` // the rules file when it is empty
	Line int
	Msg  string
}

// File is a parsed rules file.
type File struct {
	Origin string
	Hash   string
	Lists  string // the version of the tool's lists when the head was written
	Rules  []*Rule

	originLine, hashLine int // from 0, or -1
	lines                []string
}

// Normalize gives the bytes that a hash reads: a checkout on Windows does not change a hash.
func Normalize(b []byte) []byte {
	return []byte(strings.ReplaceAll(string(b), "\r\n", "\n"))
}

// Hash is the hash of a source.
func Hash(b []byte) string {
	sum := sha256.Sum256(Normalize(b))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// params lists the parameters of each kind, and whether each is required ("!").
var params = map[string][]string{
	"forbid-call": {"funcs!", "in", "except", "files"},
	"only-in":     {"effect!", "in!", "files"},
	"depends":     {"from", "except", "deny", "allow", "packages", "via", "map", "files"},
	"transitions": {"type!", "field!", "allow!", "initial", "load"},
	"declaration": {"at!", "names", "except-names", "types", "deny-types", "deny-names", "require", "to", "in", "except", "files"},
	"call-form":   {"funcs", "param-type", "arg", "require", "deny-text", "require-text", "unless-comment", "in", "except", "files"},
	"literal":     {"type", "require", "deny", "below", "deny-text", "names", "in", "except", "files"},
	"code-form":   {"form!", "limit", "similarity", "funcs", "deferred-ok", "unless-comment", "in", "except", "files"},
}

// Kinds lists the rule kinds.
func Kinds() []string {
	var out []string
	for k := range params {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// Parse reads a rules file. It reports each thing that is wrong, by line. A source is parsed the
// same way, but its head is optional, because init writes the head.
func Parse(b []byte) (*File, []Problem) {
	f := &File{originLine: -1, hashLine: -1, lines: strings.Split(string(Normalize(b)), "\n")}
	var probs []Problem
	bad := func(line int, format string, args ...any) {
		probs = append(probs, Problem{Line: line + 1, Msg: fmt.Sprintf(format, args...)})
	}
	var r *Rule
	seen := map[string]bool{}
	comment := -1 // the first line of the comments right above the current line
	finish := func() {
		if r == nil {
			return
		}
		r.check(func(format string, args ...any) { bad(r.Line-1, format, args...) })
		r = nil
	}
	for i, line := range f.lines {
		text := strings.TrimSpace(line)
		switch {
		case text == "":
			comment = -1
			continue
		case strings.HasPrefix(text, "#"):
			if comment < 0 {
				comment = i
			}
			continue
		}
		key, rest, _ := strings.Cut(text, " ")
		rest = strings.TrimSpace(rest)
		switch {
		case key == "origin" && r == nil:
			f.Origin, f.originLine = rest, i
		case key == "hash" && r == nil:
			f.Hash, f.hashLine = rest, i
		case key == "lists" && r == nil:
			f.Lists = rest
		case key == "rule":
			finish()
			r = &Rule{ID: rest, Params: map[string][]string{}, Line: i + 1, start: i, end: i + 1}
			if comment >= 0 {
				r.start = comment
			}
			switch {
			case rest == "" || len(strings.Fields(rest)) != 1:
				bad(i, "a rule line is: rule ID")
			case seen[rest]:
				bad(i, "rule %s is declared twice", rest)
			}
			seen[rest] = true
			f.Rules = append(f.Rules, r)
		case r == nil:
			bad(i, "%s is not a line of the head; the head holds origin and hash, and each rule starts with: rule ID", key)
		default:
			r.end = i + 1
			switch key {
			case "kind":
				r.Kind = rest
			case "level":
				r.Level = rest
			case "reason":
				r.Reason = rest
			case "source":
				r.Source = rest
			default:
				if _, dup := r.Params[key]; dup {
					bad(i, "rule %s gives %s twice", r.ID, key)
				}
				r.Params[key] = strings.Fields(rest)
			}
		}
		comment = -1
	}
	finish()
	return f, probs
}

// ParseProject reads the rules file of a project, whose head must name its origin.
func ParseProject(b []byte) (*File, []Problem) {
	f, probs := Parse(b)
	if f.Origin == "" {
		probs = append(probs, Problem{Line: 1, Msg: "the head of the rules file names its origin: origin defaults v1, or origin SOURCE and hash sha256:..."})
	}
	return f, probs
}

// check reports what is wrong with one rule, apart from names that only a project can resolve.
func (r *Rule) check(bad func(string, ...any)) {
	allowed, ok := params[r.Kind]
	if !ok {
		bad("rule %s has the kind %q; the kinds are %s", r.ID, r.Kind, strings.Join(Kinds(), ", "))
		return
	}
	if r.Level != Error && r.Level != Warn && r.Level != Off {
		bad("rule %s has the level %q; a level is error, warn, or off", r.ID, r.Level)
	}
	if r.Reason == "" {
		bad("rule %s has no reason: a reason in one line", r.ID)
	}
	if r.Source == "" {
		bad("rule %s has no source: where the practice comes from", r.ID)
	}
	names := map[string]bool{}
	for _, p := range allowed {
		name, required := strings.CutSuffix(p, "!")
		names[name] = true
		if required && len(r.Params[name]) == 0 {
			bad("rule %s of kind %s needs %s", r.ID, r.Kind, name)
		}
	}
	for name := range r.Params {
		if !names[name] {
			bad("rule %s of kind %s has no parameter %s", r.ID, r.Kind, name)
		}
	}
	for _, v := range r.Params["files"] {
		if v != "all" && v != "code" && v != "test" {
			bad("rule %s: files is all, code, or test", r.ID)
		}
	}
	for _, key := range []string{"in", "except", "from", "deny", "allow", "load"} {
		if r.Kind == "transitions" && key == "allow" || r.Kind == "literal" && key == "deny" {
			continue
		}
		for _, v := range r.Params[key] {
			if !isSelector(v) {
				bad("rule %s: %s names %q, which is not a kind (%s) or the directory of a part", r.ID, key, v, strings.Join(partKinds, ", "))
			}
		}
	}
	switch r.Kind {
	case "only-in":
		for _, e := range r.Params["effect"] {
			if _, ok := effects[e]; !ok {
				bad("rule %s has the effect %q; the effects are %s", r.ID, e, strings.Join(EffectNames(), ", "))
			}
		}
	case "depends":
		r.checkDepends(bad)
	case "transitions":
		r.checkTransitions(bad)
	case "declaration", "call-form", "literal", "code-form":
		r.checkShape(bad)
	}
}

func (r *Rule) checkDepends(bad func(string, ...any)) {
	locked := len(r.Params["map"]) > 0
	if locked {
		if len(r.Params["map"]) != 1 || r.Params["map"][0] != "locked" {
			bad("rule %s: map is locked", r.ID)
		}
		for _, k := range []string{"from", "except", "deny", "allow", "packages", "via", "files"} {
			if len(r.Params[k]) > 0 {
				bad("rule %s: a rule with map locked reads the lock, and takes no %s", r.ID, k)
			}
		}
		return
	}
	if len(r.Params["from"]) == 0 {
		bad("rule %s of kind depends needs from, or map locked", r.ID)
	}
	if len(r.Params["deny"])+len(r.Params["packages"]) > 0 == (len(r.Params["allow"]) > 0) {
		bad("rule %s of kind depends needs deny or packages, or else allow", r.ID)
	}
	for _, v := range r.Params["via"] {
		if v != "wire" && v != "import" && v != "any" {
			bad("rule %s: via is wire, import, or any", r.ID)
		}
	}
}

func (r *Rule) checkTransitions(bad func(string, ...any)) {
	if t := r.Params["type"]; len(t) == 1 && !strings.Contains(t[0], ".") {
		bad("rule %s: type is the directory of a part and a type name, such as lib/shop.Order", r.ID)
	} else if len(t) > 1 {
		bad("rule %s: type names one type", r.ID)
	}
	if len(r.Params["field"]) > 1 {
		bad("rule %s: field names one field", r.ID)
	}
	for _, e := range r.Params["allow"] {
		from, to, ok := strings.Cut(e, "->")
		if !ok || from == "" || to == "" || to == "*" {
			bad("rule %s: allow holds changes such as paid->shipped, or *->cancelled", r.ID)
		}
	}
}

// Format writes one rule, with the parameters in a fixed order.
func Format(r *Rule) string {
	var b strings.Builder
	fmt.Fprintf(&b, "rule %s\n  kind %s\n  level %s\n", r.ID, r.Kind, r.Level)
	for _, p := range params[r.Kind] {
		name := strings.TrimSuffix(p, "!")
		if v := r.Params[name]; len(v) > 0 {
			fmt.Fprintf(&b, "  %s %s\n", name, strings.Join(v, " "))
		}
	}
	fmt.Fprintf(&b, "  reason %s\n  source %s\n", r.Reason, r.Source)
	return b.String()
}

// Equal reports whether two rules say the same; comments and line numbers do not count.
func Equal(a, b *Rule) bool { return Format(a) == Format(b) }

// Find returns the rule with an id, or nil.
func (f *File) Find(id string) *Rule {
	for _, r := range f.Rules {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// TextOf gives the text of a rule for a reader: the comment above it, which holds the wanted text in
// defaults v1, and its reason.
func (f *File) TextOf(id string) string {
	r := f.Find(id)
	if r == nil {
		return ""
	}
	var words []string
	for _, line := range f.lines[r.start : r.Line-1] {
		words = append(words, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#")))
	}
	return strings.TrimSpace(strings.Join(append(words, r.Reason), " "))
}

// block is the text of a rule, with the comments right above it.
func (f *File) block(r *Rule) string {
	return strings.Join(f.lines[r.start:r.end], "\n") + "\n"
}
