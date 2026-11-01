package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// BaselineName is the file of the findings that a team has accepted for now. It is a plain file that
// the team owns, with the rules file and the lock.
const BaselineName = "rules.baseline"

// The kinds of finding that a stale entry is looked for with.
const (
	GroupCode  = "code"
	GroupWires = "wires"
	GroupDecls = "decls"
)

// Entry is the key of a rule's finding: what stays the same when code above it moves. It has no line
// number and no file, so an edit, or a function that moves to another file of its part, leaves it
// old. A rename of the function makes it new, and the old entry stale. It has no receiver, because a
// helper is a method in L and can be a plain function in V.
type Entry struct {
	Rule, Group, Part, Symbol, Text string
}

func clean(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func undash(s string) string {
	if s == "-" {
		return ""
	}
	return s
}

// Line writes an entry with its count, as one line of the file.
func (e Entry) Line(count int) string {
	return strings.Join([]string{strconv.Itoa(count), e.Rule, e.Group, dash(e.Part), dash(e.Symbol), clean(e.Text)}, "\t")
}

// Entry gives the key of a finding.
func (f Finding) Entry() Entry {
	return Entry{f.Rule, f.Group, f.Part, f.Symbol, clean(f.Text)}
}

// Baseline is a parsed file.
type Baseline struct {
	Counts map[Entry]int
	Lines  map[Entry]int // from 1
}

const baselineHead = `# The findings that the team accepted when it wrote this file with the command rules baseline. A check
# or a build fails only on a finding that this file does not hold. An old finding still prints.
# Each line: count, rule, group, part, symbol, text. It has no line number: an edit does not make a
# finding new, and one more finding with the same key is new. A model check is never here.
# Review a change to this file: it can hide a rule's findings.
`

// WriteBaseline writes the file, in a fixed order, so that a change to it is a small diff.
func WriteBaseline(root string, entries []Entry) (int, error) {
	counts := map[Entry]int{}
	for _, e := range entries {
		counts[e]++
	}
	var lines []string
	for e, n := range counts {
		lines = append(lines, e.Line(n))
	}
	slices.SortFunc(lines, func(a, b string) int {
		_, x, _ := strings.Cut(a, "\t")
		_, y, _ := strings.Cut(b, "\t")
		return strings.Compare(x, y)
	})
	text := baselineHead + strings.Join(lines, "\n")
	if len(lines) > 0 {
		text += "\n"
	}
	return len(entries), os.WriteFile(filepath.Join(root, BaselineName), []byte(text), 0o644)
}

// ReadBaseline reads the file of a project. It gives nil when the project has none.
func ReadBaseline(root string) (*Baseline, []Problem) {
	b, err := os.ReadFile(filepath.Join(root, BaselineName))
	if err != nil {
		return nil, nil
	}
	out := &Baseline{Counts: map[Entry]int{}, Lines: map[Entry]int{}}
	var probs []Problem
	for i, line := range strings.Split(string(Normalize(b)), "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.SplitN(line, "\t", 6)
		n := 0
		if len(f) == 6 {
			n, err = strconv.Atoi(f[0])
		}
		if len(f) != 6 || err != nil || n < 1 {
			probs = append(probs, Problem{File: BaselineName, Line: i + 1, Msg: BaselineName + " has a line that is not: count, rule, group, part, symbol, text, with tabs between them"})
			continue
		}
		e := Entry{f[1], f[2], undash(f[3]), undash(f[4]), f[5]}
		out.Counts[e] += n
		out.Lines[e] = i + 1
	}
	return out, probs
}

// Size gives the findings and the keys of the file.
func (b *Baseline) Size() (findings, keys int) {
	for _, n := range b.Counts {
		findings += n
	}
	return findings, len(b.Counts)
}

// Scope says which entries of the file a call of Classify has read in full.
type Scope struct {
	Group string
	Part  string // "*" for each part of the group
}

func (sc Scope) holds(e Entry) bool {
	return e.Group == sc.Group && (sc.Part == "*" || sc.Part == e.Part)
}

// Classify marks each finding that the baseline holds as old: it prints, and it fails nothing. With
// more findings of a key than the file holds, the first in the code's order are old, and the others
// are new. For a complete scope, it also gives each entry that no finding used, which is stale. A
// stale entry is reported, and fails nothing.
func (s *Set) Classify(findings []Finding, scope Scope, complete bool) ([]Finding, []Problem) {
	if s == nil || s.Baseline == nil {
		return findings, nil
	}
	used := map[Entry]int{}
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		e := f.Entry()
		used[e]++
		if used[e] <= s.Baseline.Counts[e] {
			f.Old = true
			f.Warn = true
			f.Msg = "old finding, in " + BaselineName + ": " + f.Msg
		}
		out = append(out, f)
	}
	if !complete {
		return out, nil
	}
	var stale []Problem
	for e, n := range s.Baseline.Counts {
		if scope.holds(e) && used[e] < n {
			stale = append(stale, Problem{File: BaselineName, Line: s.Baseline.Lines[e], Msg: fmt.Sprintf("stale entry of %s: it holds %d of rule %s in %s, and %d remain; run the command rules baseline to drop it", BaselineName, n, e.Rule, dash(e.Part), used[e])})
		}
	}
	slices.SortFunc(stale, func(a, b Problem) int { return a.Line - b.Line })
	return out, stale
}

// StaleParts reports the entries of a part that the project no longer has.
func (s *Set) StaleParts(parts map[string]Part) []Problem {
	if s == nil || s.Baseline == nil {
		return nil
	}
	var stale []Problem
	for e, n := range s.Baseline.Counts {
		if e.Part != "" && parts[e.Part].Dir == "" {
			stale = append(stale, Problem{File: BaselineName, Line: s.Baseline.Lines[e], Msg: fmt.Sprintf("stale entry of %s: it holds %d of rule %s in %s, which is not a part of the project; run the command rules baseline to drop it", BaselineName, n, e.Rule, e.Part)})
		}
	}
	slices.SortFunc(stale, func(a, b Problem) int { return a.Line - b.Line })
	return stale
}
