// Package lintest builds the small modules that the tests of vulcan-lint check.
package lintest

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/tools/txtar"
)

// Workspace is arm L's workspace, poc/app/golint: the source of the manifest's schema and of the
// contract package that every test module gets.
func Workspace() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "app", "golint")
}

// Shared are the files that each test module gets from the workspace, unless its archive has
// them: the schema of the manifest and the contract package.
var Shared = []string{"arch/schema.go", "lib/contract/contract.go"}

// Write writes the files of archives, in order (a later file replaces an earlier one of the
// same name; a file whose content is "DELETE\n" removes it), and the shared files, into a new
// directory. It adds "module shop" when no archive has a go.mod.
func Write(t *testing.T, archives ...*txtar.Archive) string {
	t.Helper()
	files := map[string][]byte{}
	for _, name := range Shared {
		b, err := os.ReadFile(filepath.Join(Workspace(), filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = b
	}
	files["go.mod"] = []byte("module shop\n\ngo 1.26.0\n")
	for _, a := range archives {
		for _, f := range a.Files {
			if string(f.Data) == "DELETE\n" {
				delete(files, f.Name)
				continue
			}
			files[f.Name] = f.Data
		}
	}
	dir := t.TempDir()
	for name, data := range files {
		if strings.HasSuffix(name, ".txt") && !strings.Contains(name, "/") {
			continue // want.txt and other notes of the case
		}
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Read reads a txtar archive.
func Read(t *testing.T, path string) *txtar.Archive {
	t.Helper()
	a, err := txtar.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
