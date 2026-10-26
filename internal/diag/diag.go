// Package diag holds compiler diagnostics (spec Appendix E).
package diag

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

// Severity is "error" or "warn".
type Severity string

const (
	Error Severity = "error"
	Warn  Severity = "warn"
)

// Pos is a position in a source file. File is relative to the project root, with slashes.
type Pos struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Col  int    `json:"col"`
}

func (p Pos) String() string {
	if p.File == "" {
		return "-"
	}
	if p.Line == 0 {
		return p.File
	}
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col)
}

// Diagnostic is one finding of the compiler.
type Diagnostic struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Pos      Pos      `json:"pos"`
	Message  string   `json:"message"`
	Fix      string   `json:"fix,omitempty"`
}

// List collects diagnostics.
type List struct {
	Items []Diagnostic
	seen  map[string]bool
}

// Errorf adds an error.
func (l *List) Errorf(pos Pos, code, format string, args ...any) {
	l.add(Diagnostic{Severity: Error, Code: code, Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// Warnf adds a warning.
func (l *List) Warnf(pos Pos, code, format string, args ...any) {
	l.add(Diagnostic{Severity: Warn, Code: code, Pos: pos, Message: fmt.Sprintf(format, args...)})
}

// Add adds a diagnostic, once.
func (l *List) Add(d Diagnostic) { l.add(d) }

func (l *List) add(d Diagnostic) {
	d.Pos.File = filepath.ToSlash(d.Pos.File)
	key := fmt.Sprintf("%s|%s|%s|%s", d.Code, d.Pos, d.Message, d.Severity)
	if l.seen == nil {
		l.seen = map[string]bool{}
	}
	if l.seen[key] {
		return
	}
	l.seen[key] = true
	l.Items = append(l.Items, d)
}

// HasErrors reports whether the list holds an error.
func (l *List) HasErrors() bool {
	for _, d := range l.Items {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// Sort orders the diagnostics by file, line, column, and code.
func (l *List) Sort() {
	sort.SliceStable(l.Items, func(i, j int) bool {
		a, b := l.Items[i], l.Items[j]
		if a.Pos.File != b.Pos.File {
			return a.Pos.File < b.Pos.File
		}
		if a.Pos.Line != b.Pos.Line {
			return a.Pos.Line < b.Pos.Line
		}
		if a.Pos.Col != b.Pos.Col {
			return a.Pos.Col < b.Pos.Col
		}
		return a.Code < b.Code
	})
}

// Print writes the diagnostics as text, one per line.
func (l *List) Print(w io.Writer) {
	for _, d := range l.Items {
		fmt.Fprintf(w, "%s: %s %s: %s\n", d.Pos, d.Severity, d.Code, d.Message)
		if d.Fix != "" {
			fmt.Fprintf(w, "\tfix: %s\n", strings.ReplaceAll(d.Fix, "\n", "\n\t"))
		}
	}
}

// PrintJSON writes the diagnostics as one JSON document.
func (l *List) PrintJSON(w io.Writer) error {
	items := l.Items
	if items == nil {
		items = []Diagnostic{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"diagnostics": items, "errors": l.HasErrors()})
}
