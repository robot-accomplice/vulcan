package seal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "rules", "R1"), 0o755)
	os.WriteFile(filepath.Join(src, "meta.json"), []byte(`{"tasks":[]}`), 0o644)
	os.WriteFile(filepath.Join(src, "rules", "R1", "main.go"), []byte("package main"), 0o644)
	key, _ := Key(NewKey())
	out := filepath.Join(t.TempDir(), "material.sealed")
	if err := Seal(src, out, key); err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := Open(out, dst, key); err != nil {
		t.Fatal(err)
	}
	m1, _ := Manifest(src)
	m2, _ := Manifest(dst)
	if m1 != m2 || m1 == "\n" {
		t.Fatalf("manifests differ:\n%s\n%s", m1, m2)
	}
	other, _ := Key(NewKey())
	if err := Open(out, t.TempDir(), other); err == nil {
		t.Fatal("a wrong key opened the file")
	}
}
