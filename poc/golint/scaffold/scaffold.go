// Package scaffold writes what vulcan-lint init creates in a Go module that lacks it: go.mod, the
// manifest, and the contract package. files/schema.go.txt and files/contract.go.txt are copies of
// the workspace's, which a test compares.
package scaffold

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/robot-accomplice/vulcan/internal/rules"
	"github.com/robot-accomplice/vulcan/poc/golint/model"
)

//go:embed files
var files embed.FS

const manifest = `package arch

// Manifest is the architecture of the system.
var Manifest = System{
	Parts: []Part{
		Library("lib/contract"),
	},
}
`

const config = `package arch

// Config is the check configuration of vulcan-lint.
var Config = Settings{
	Strict: true,
}
`

// Tool gives the commands init and rules their parts in vulcan-lint.
func Tool(edges func(string) ([]rules.Edge, error)) rules.Tool {
	return rules.Tool{Name: "vulcan-lint", FindRoot: findRoot, Scaffold: write, Edges: edges}
}

func findRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		if filepath.Dir(d) == d {
			return "", fmt.Errorf("no go.mod in %s or a parent directory", abs)
		}
	}
}

// write writes each file that the project lacks. A module that already has a manifest gets no new
// part of one.
func write(root string, stdout io.Writer) error {
	schema, err := files.ReadFile("files/schema.go.txt")
	if err != nil {
		return err
	}
	contract, err := files.ReadFile("files/contract.go.txt")
	if err != nil {
		return err
	}
	out := []struct {
		rel  string
		data []byte
	}{{"go.mod", fmt.Appendf(nil, "module %s\n\ngo %s\n", moduleName(filepath.Base(root)), rules.GoVersion)}}
	if _, err := os.Stat(filepath.Join(root, model.ManifestDir)); err != nil {
		out = append(out,
			struct {
				rel  string
				data []byte
			}{"arch/schema.go", schema},
			struct {
				rel  string
				data []byte
			}{"arch/arch.go", []byte(manifest)},
			struct {
				rel  string
				data []byte
			}{"arch/config.go", []byte(config)},
			struct {
				rel  string
				data []byte
			}{"lib/contract/contract.go", contract},
		)
	}
	for _, f := range out {
		full := filepath.Join(root, filepath.FromSlash(f.rel))
		if _, err := os.Stat(full); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, f.data, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s\n", f.rel)
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
