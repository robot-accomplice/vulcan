package driver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/diag"
	"github.com/robot-accomplice/vulcan/internal/rules"
)

// RulesTool gives the commands init and rules their Vulcan parts.
func RulesTool() rules.Tool {
	return rules.Tool{Name: "vulcan", FindRoot: FindRoot, Scaffold: scaffold, Edges: edges, Findings: findings}
}

// scaffold writes go.mod and system.vulcan, each only when the project lacks it.
func scaffold(root string, stdout io.Writer) error {
	name := moduleName(filepath.Base(root))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		if err := os.WriteFile(filepath.Join(root, "go.mod"), fmt.Appendf(nil, "module %s\n\ngo %s\n", name, rules.GoVersion), 0o644); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "wrote go.mod")
	}
	if _, err := os.Stat(filepath.Join(root, "system.vulcan")); err != nil {
		text := fmt.Sprintf("system %s {\n    config { strict: true }\n\n    container Api {\n    }\n}\n", systemName(name))
		if err := os.WriteFile(filepath.Join(root, "system.vulcan"), []byte(text), 0o644); err != nil {
			return err
		}
		fmt.Fprintln(stdout, "wrote system.vulcan")
	}
	return nil
}

func moduleName(s string) string {
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' {
			return r
		}
		return -1
	}, strings.ToLower(s))
	if s == "" {
		return "app"
	}
	return s
}

func systemName(module string) string {
	var b strings.Builder
	for _, part := range strings.FieldsFunc(module, func(r rune) bool { return r == '-' || r == '_' || r == '.' }) {
		b.WriteString(strings.ToUpper(part[:1]) + part[1:])
	}
	if s := b.String(); s != "" && s[0] >= 'A' && s[0] <= 'Z' {
		return s
	}
	return "App"
}

// findings gives what rules baseline writes: each rule finding, with tests. It refuses a model error
// and a fault in the rules, because the file holds only rule findings.
func findings(root string) ([]rules.Entry, error) {
	r, err := checkWith(root, true, true)
	if err != nil {
		return nil, err
	}
	for _, d := range r.Diags.Items {
		if d.Severity == diag.Error && (!strings.HasPrefix(d.Code, "K08") || d.Code == "K0801") {
			return nil, fmt.Errorf("the project has errors; fix them first: %s %s %s", d.Pos, d.Code, d.Message)
		}
	}
	var out []rules.Entry
	for _, f := range r.Findings {
		out = append(out, f.Entry())
	}
	return out, nil
}

// edges gives what rules lock writes. It runs the check, and refuses a project with an error that
// is not a rule's: an error of a rule must not stop the lock that would clear it.
func edges(root string) ([]rules.Edge, error) {
	r, err := Check(root, false)
	if err != nil {
		return nil, err
	}
	for _, d := range r.Diags.Items {
		if d.Severity == diag.Error && !strings.HasPrefix(d.Code, "K08") {
			return nil, fmt.Errorf("the project has errors; fix them first: %s %s %s", d.Pos, d.Code, d.Message)
		}
	}
	return r.Edges, nil
}
