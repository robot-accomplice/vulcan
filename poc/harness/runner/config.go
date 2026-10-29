package runner

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/robot-accomplice/vulcan/poc/golint/model"
	"github.com/robot-accomplice/vulcan/poc/harness/record"
)

// ProtectedFile names, in a template, the files of the arm's check configuration, one path or
// glob per line ('#' starts a comment). No arm may change them unless the brief asks (round 2,
// F7; decision 6, section 2).
const ProtectedFile = "PROTECTED"

// suppressionRE finds in-source suppressions of a check.
var suppressionRE = regexp.MustCompile(`//\s*(nolint|lint:ignore|lint:file-ignore|vulcan-lint:ignore)\b`)

// configBlockRE finds the config block of system.vulcan.
var configBlockRE = regexp.MustCompile(`(?s)\bconfig\s*\{[^}]*\}`)

// configChanges lists the changes to the arm's check configuration between the base and the
// final workspace: a changed protected file, a changed config block of system.vulcan (strict,
// strict_exempt, strict_interfaces, strict_shared, allow_*), a changed kind of a part of the
// base, and new in-source suppressions. An added part is not a change (parity review R2, item 1).
func configChanges(arm, base, ws string) []string {
	var out []string
	for _, rel := range protected(base) {
		a, errA := os.ReadFile(filepath.Join(base, rel))
		b, errB := os.ReadFile(filepath.Join(ws, rel))
		if errA != nil || errB != nil || string(normal(a)) != string(normal(b)) {
			out = append(out, rel+": changed")
		}
	}
	if arm == record.ArmV {
		a, _ := os.ReadFile(filepath.Join(base, "system.vulcan"))
		b, _ := os.ReadFile(filepath.Join(ws, "system.vulcan"))
		if configBlock(a) != configBlock(b) {
			out = append(out, "system.vulcan: config block changed")
		}
	}
	kinds := map[string]func(string) map[string]string{record.ArmL: goKinds, record.ArmV: vulcanKinds}[arm]
	if kinds != nil {
		before, after := kinds(base), kinds(ws)
		for _, dir := range sortedKeys(before) {
			if k, ok := after[dir]; ok && k != before[dir] {
				out = append(out, dir+": kind changed from "+before[dir]+" to "+k)
			}
		}
	}
	if n := suppressions(ws) - suppressions(base); n > 0 {
		out = append(out, "suppressions added in the source")
	}
	sort.Strings(out)
	return out
}

// goKinds returns the kind of each part of arm L's manifest (arch), by directory: its
// constructor and its party, such as "gateway for Database".
func goKinds(root string) map[string]string {
	m := model.Read(root)
	out := map[string]string{}
	for _, p := range m.Parts {
		k := p.Kind
		if p.For != "" {
			k += " for " + p.For
		}
		out[p.Dir] = k
	}
	return out
}

// kindLineRE finds the kind lines of a component.vulcan: gateway for X, entry for Y.
var kindLineRE = regexp.MustCompile(`(?m)^\s*(gateway|entry)\s+for\s+(\w+)`)

// vulcanKinds returns the kind of each leaf of arm V, by the directory of its component.vulcan:
// its gateway for and entry for lines, or "leaf".
func vulcanKinds(root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == ".vulcan" || d.Name() == ".git") {
			return filepath.SkipDir
		}
		if d.IsDir() || d.Name() != "component.vulcan" {
			return nil
		}
		b, _ := os.ReadFile(p)
		var lines []string
		for _, m := range kindLineRE.FindAllStringSubmatch(string(normal(b)), -1) {
			lines = append(lines, m[1]+" for "+m[2])
		}
		sort.Strings(lines)
		k := "leaf"
		if len(lines) > 0 {
			k = strings.Join(lines, ", ")
		}
		rel, _ := filepath.Rel(root, filepath.Dir(p))
		out[filepath.ToSlash(rel)] = k
		return nil
	})
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normal(b []byte) []byte { return []byte(strings.ReplaceAll(string(b), "\r\n", "\n")) }

func configBlock(b []byte) string {
	return strings.Join(strings.Fields(configBlockRE.FindString(string(b))), " ")
}

// protected lists the files that ProtectedFile names in the base workspace.
func protected(base string) []string {
	b, err := os.ReadFile(filepath.Join(base, ProtectedFile))
	if err != nil {
		return nil
	}
	seen := map[string]bool{ProtectedFile: true}
	out := []string{ProtectedFile}
	for _, line := range strings.Split(string(normal(b)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		matches, _ := filepath.Glob(filepath.Join(base, filepath.FromSlash(line)))
		for _, m := range matches {
			rel, _ := filepath.Rel(base, m)
			if rel = filepath.ToSlash(rel); !seen[rel] {
				seen[rel] = true
				out = append(out, rel)
			}
		}
	}
	sort.Strings(out)
	return out
}

// suppressions counts the in-source suppressions of the product source.
func suppressions(root string) int {
	n := 0
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && (d.Name() == ".vulcan" || d.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") || strings.HasSuffix(p, ".vulcan") {
			b, _ := os.ReadFile(p)
			n += len(suppressionRE.FindAll(b, -1))
		}
		return nil
	})
	return n
}

// strayFiles lists files that the run left in the workspace outside the product source: under a
// top-level name that the base does not have, or binary (F3). The grader and the diff ignore them.
func strayFiles(base, ws string) []string {
	tops := topEntries(base)
	var out []string
	_ = filepath.WalkDir(ws, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(ws, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if skipped(rel + "/") {
				return filepath.SkipDir
			}
			return nil
		}
		if skipped(rel) {
			return nil
		}
		if !tops[strings.Split(rel, "/")[0]] {
			out = append(out, rel)
			return nil
		}
		if b, err := os.ReadFile(p); err == nil && isBinary(b) {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out
}
