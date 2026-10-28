package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSandboxIsolation runs a shell in the sandbox of a real WSL distribution, named by
// VULCAN_TEST_WSL, and checks what the run can and cannot reach.
func TestSandboxIsolation(t *testing.T) {
	distro := os.Getenv("VULCAN_TEST_WSL")
	if distro == "" {
		t.Skip("set VULCAN_TEST_WSL to a WSL distribution")
	}
	dir := t.TempDir()
	tools := filepath.Join(dir, "tools")
	mod := filepath.Join(dir, "mod")
	for _, d := range []string{filepath.Join(dir, "run", "workspace"), filepath.Join(tools, "bin"), filepath.Join(mod, "cache", "download")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	marker := "vulcan-sandbox-probe-" + filepath.Base(dir)
	probe := `out=/tmp/run/workspace/probe.txt
{
echo "mnt=$(ls /mnt | wc -l)"
echo "userhome=$(ls -A /home/*/ 2>/dev/null | wc -l)"
echo "token=${ANTHROPIC_AUTH_TOKEN:-none}"
echo "tools_ro=$(touch /tmp/tools/x 2>/dev/null && echo no || echo yes)"
echo "ws=$(pwd)"
echo "uid=$(id -u)"
} > "$out"
touch /tmp/` + marker + `
echo '{"type":"result","subtype":"success","is_error":false,"result":"ok"}'
`
	c := &Config{OutDir: filepath.Join(dir, "run"), Arm: "P", BaseURL: "https://example.test/anthropic", Token: "tok-123"}
	c.Sandbox = &Sandbox{Distro: distro, Tools: tools, ModCache: mod}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd, cleanup, err := c.command(ctx, []string{"sh", "-c", probe, "prompt"})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	b, err := os.ReadFile(filepath.Join(dir, "run", "workspace", "probe.txt"))
	if err != nil {
		t.Fatalf("no probe: %v\n%s", err, out)
	}
	got := string(b)
	for _, want := range []string{"mnt=0\n", "token=tok-123\n", "tools_ro=yes\n", "ws=/tmp/run/workspace\n", "uid=0\n", "userhome=0\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("probe lacks %q:\n%s", want, got)
		}
	}
	check := exec.Command("wsl.exe", "-d", distro, "--", "sh", "-c", "ls /tmp/"+marker+" 2>/dev/null; ls -A \"$HOME\" | head -1")
	host, _ := check.CombinedOutput()
	if strings.Contains(string(host), marker) {
		t.Fatal("the run's /tmp file is in the real /tmp")
	}
	script, _ := os.ReadFile(filepath.Join(dir, "run", "sandbox.sh"))
	if strings.Contains(string(script), "tok-123") {
		t.Fatal("the script holds the key")
	}
}
