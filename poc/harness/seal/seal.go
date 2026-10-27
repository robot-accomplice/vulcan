// Package seal keeps the hidden material of the evaluation away from the agents and the builder
// (docs/poc/plan.md, section 11). Seal packs a directory into one file encrypted with AES-256-GCM,
// and Manifest lists the SHA-256 of each file, so that the hashes can be committed before the
// runs. Open unpacks the file for grading.
package seal

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Key reads a 32-byte key in hex.
func Key(hexKey string) ([]byte, error) {
	k, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil || len(k) != 32 {
		return nil, errors.New("the key is 64 hex digits (32 bytes)")
	}
	return k, nil
}

// NewKey returns a random key in hex.
func NewKey() string {
	k := make([]byte, 32)
	if _, err := rand.Read(k); err != nil {
		panic(err)
	}
	return hex.EncodeToString(k)
}

// Manifest returns "sha256  path" for each file of dir, sorted by path.
func Manifest(dir string) (string, error) {
	var lines []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		sum := sha256.Sum256(b)
		lines = append(lines, hex.EncodeToString(sum[:])+"  "+filepath.ToSlash(rel))
		return nil
	})
	sort.Slice(lines, func(i, j int) bool { return lines[i][66:] < lines[j][66:] })
	return strings.Join(lines, "\n") + "\n", err
}

// Seal packs dir and encrypts it into out.
func Seal(dir, out string, key []byte) error {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		return err
	})
	if err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	return os.WriteFile(out, gcm.Seal(nonce, nonce, buf.Bytes(), nil), 0o600)
}

// Open decrypts a sealed file and unpacks it into dir.
func Open(sealed, dir string, key []byte) error {
	b, err := os.ReadFile(sealed)
	if err != nil {
		return err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return err
	}
	if len(b) < gcm.NonceSize() {
		return errors.New("the sealed file is too short")
	}
	plain, err := gcm.Open(nil, b[:gcm.NonceSize()], b[gcm.NonceSize():], nil)
	if err != nil {
		return fmt.Errorf("the key does not open the sealed file: %w", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		name := filepath.FromSlash(f.Name)
		if strings.Contains(f.Name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("bad path %q in the sealed file", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}
		target := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
