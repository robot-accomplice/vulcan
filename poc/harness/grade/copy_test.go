package grade

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/robot-accomplice/vulcan/internal/rules"
)

// TestCopyForBuild is audit N34: the grader's copy keeps each regular file but the rules files, passes
// over links, junctions, and FIFOs, where the host allows each, and does not block.
func TestCopyForBuild(t *testing.T) {
	src := filepath.Join(t.TempDir(), "ws")
	put := func(rel string) {
		if err := writeFile(filepath.Join(src, filepath.FromSlash(rel)), []byte(rel)); err != nil {
			t.Fatal(err)
		}
	}
	keep := []string{"api/orders/seed.db", "api/orders/x.go", ".git/HEAD", "notes/rules.vulcan"}
	drop := []string{rules.FileName, rules.OriginName, rules.LockName, rules.BaselineName}
	for _, rel := range append(append([]string{}, keep...), drop...) {
		put(rel)
	}
	var links []string
	if err := os.Symlink(filepath.Join(src, "api"), filepath.Join(src, "link")); err == nil {
		links = append(links, "link")
	} else {
		t.Logf("no symlink on this host: %v", err)
	}
	if err := os.Symlink(filepath.Join(src, "gone"), filepath.Join(src, "broken")); err == nil {
		links = append(links, "broken")
	}
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(src, "junction"), filepath.Join(src, "api")).CombinedOutput(); err == nil {
			links = append(links, "junction")
		} else {
			t.Logf("no junction on this host: %v %s", err, out)
		}
	}
	if err := mkfifo(filepath.Join(src, "fifo")); err == nil {
		links = append(links, "fifo")
	} else {
		t.Logf("no FIFO on this host: %v", err)
	}
	t.Logf("entries that are not regular files: %v", links)
	dst := filepath.Join(t.TempDir(), "copy")
	done := make(chan error, 1)
	go func() { done <- copyForBuild(src, dst, time.Minute) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the copy blocks")
	}
	for _, rel := range keep {
		if b, err := os.ReadFile(filepath.Join(dst, filepath.FromSlash(rel))); err != nil || string(b) != rel {
			t.Errorf("%s is not copied: %v", rel, err)
		}
	}
	for _, rel := range append(drop, links...) {
		if _, err := os.Lstat(filepath.Join(dst, rel)); err == nil {
			t.Errorf("%s is copied", rel)
		}
	}
	if err := copyForBuild(src, filepath.Join(t.TempDir(), "late"), -time.Second); err == nil {
		t.Error("a copy past its time limit passes")
	}
}
