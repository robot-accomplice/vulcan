package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestChangedLines(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"x\ny\nz\n", "x\ny\nz\n", 0},
		{"x\ny\nz\n", "x\nY\nz\n", 2},
		{"x\nz\n", "x\ny\nz\n", 1},
		{"", "a\nb\n", 2},
		{"a\nb\nc\nd\n", "b\nc\n", 2},
	} {
		if got := changedLines(splitLines([]byte(c.a)), splitLines([]byte(c.b))); got != c.want {
			t.Errorf("%q -> %q: %d, want %d", c.a, c.b, got, c.want)
		}
	}
	if unitOf("api/orders/orders.go") != "api/orders" || unitOf("system.vulcan") != "system.vulcan" {
		t.Fatal("unitOf")
	}
}

func TestParseVerdict(t *testing.T) {
	v, ok := parseVerdict("Looks fine.\n```\n{\"verdict\": \"reject\", \"broken_rules\": [\"R2\"], \"comments\": \"x\"}\n```")
	if !ok || v.Verdict != "reject" || len(v.Broken) != 1 {
		t.Fatalf("%+v %v", v, ok)
	}
	if _, ok := parseVerdict("no verdict here"); ok {
		t.Fatal("a missing verdict parsed")
	}
	if _, ok := parseVerdict(`{"verdict": "maybe"}`); ok {
		t.Fatal("an unknown verdict parsed")
	}
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestConfigChangesAndStrays(t *testing.T) {
	base, ws := t.TempDir(), t.TempDir()
	for _, root := range []string{base, ws} {
		write(t, root, "PROTECTED", "lint/*.json\n")
		write(t, root, "lint/rules.json", "{}\n")
		write(t, root, "system.vulcan", "system S {\n    config { strict: true }\n}\n")
		write(t, root, "api/orders/orders.go", "package orders\n")
	}
	if got := configChanges("V", base, ws); len(got) != 0 {
		t.Fatalf("no change: %v", got)
	}
	write(t, ws, "lint/rules.json", "{\"off\": true}\n")
	write(t, ws, "system.vulcan", "system S {\n    config { strict: true, strict_exempt: [\"Orders\"] }\n}\n")
	write(t, ws, "api/orders/orders.go", "package orders\n\nvar x = 1 //nolint\n")
	got := configChanges("V", base, ws)
	want := []string{"lint/rules.json: changed", "suppressions added in the source", "system.vulcan: config block changed"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v", got)
	}
	write(t, ws, "scratch/notes.md", "x")
	write(t, ws, "api/orders/stockroom.exe", "MZ\x00\x01")
	if got := strayFiles(base, ws); strings.Join(got, "|") != "api/orders/stockroom.exe|scratch/notes.md" {
		t.Fatalf("strays %v", got)
	}
}

// TestConfigChangesKinds checks that a changed kind of a part of the base fails the run in arms L
// and V, and that an added part does not (parity review R2, item 1).
func TestConfigChangesKinds(t *testing.T) {
	const manifest = "package arch\n\nconst Database External = \"Database\"\n\nvar Manifest = System{\n\tParts: []Part{\n%s\t},\n}\n"
	parts := "\t\tMain(\"cmd/shop\"),\n\t\tLeaf(\"api/orders\"),\n\t\tGateway(\"api/store\", Database),\n"
	lBase, lWS := t.TempDir(), t.TempDir()
	for _, root := range []string{lBase, lWS} {
		write(t, root, "PROTECTED", "arch/config.go\n")
		write(t, root, "arch/config.go", "package arch\n\nvar Config = Settings{Strict: true}\n")
		write(t, root, "arch/arch.go", fmt.Sprintf(manifest, parts))
	}
	write(t, lWS, "arch/arch.go", fmt.Sprintf(manifest, parts+"\t\tLeaf(\"api/reports\"),\n"))
	if got := configChanges("L", lBase, lWS); len(got) != 0 {
		t.Fatalf("L, an added part: %v", got)
	}
	write(t, lWS, "arch/arch.go", fmt.Sprintf(manifest, strings.Replace(parts, "Leaf(\"api/orders\")", "Gateway(\"api/orders\", Database)", 1)))
	if got := configChanges("L", lBase, lWS); strings.Join(got, "|") != "api/orders: kind changed from leaf to gateway for Database" {
		t.Fatalf("L, a changed kind: %v", got)
	}
	write(t, lWS, "arch/config.go", "package arch\n\nvar Config = Settings{Strict: false}\n")
	if got := configChanges("L", lBase, lWS); len(got) != 2 || got[1] != "arch/config.go: changed" {
		t.Fatalf("L, a changed configuration: %v", got)
	}

	vBase, vWS := t.TempDir(), t.TempDir()
	for _, root := range []string{vBase, vWS} {
		write(t, root, "system.vulcan", "system S {\n    config { strict: true }\n}\n")
		write(t, root, "api/orders/component.vulcan", "component Orders {\n    // not a gateway for Bank\n}\n")
		write(t, root, "api/web/component.vulcan", "component Web {\n    entry for Customer\n}\n")
	}
	write(t, vWS, "api/reports/component.vulcan", "component Reports {\n    gateway for Warehouse\n}\n")
	if got := configChanges("V", vBase, vWS); len(got) != 0 {
		t.Fatalf("V, an added part: %v", got)
	}
	write(t, vWS, "api/orders/component.vulcan", "component Orders {\n    gateway for Bank\n}\n")
	write(t, vWS, "api/web/component.vulcan", "component Web {\n    entry for Customer\n    gateway for Bank\n}\n")
	want := "api/orders: kind changed from leaf to gateway for Bank|api/web: kind changed from entry for Customer to entry for Customer, gateway for Bank"
	if got := configChanges("V", vBase, vWS); strings.Join(got, "|") != want {
		t.Fatalf("V, changed kinds: %v", got)
	}
	if got := configChanges("P", vBase, vWS); len(got) != 0 {
		t.Fatalf("P has no kinds: %v", got)
	}
}

func TestSandboxScript(t *testing.T) {
	if got := wslPath(`C:\Users\x\run 1`); got != "/mnt/c/Users/x/run 1" {
		t.Fatalf("wslPath %q", got)
	}
	if got := shQuote("it's"); got != `'it'\''s'` {
		t.Fatalf("shQuote %q", got)
	}
	c := &Config{OutDir: `C:\runs\001`, BaseURL: "https://example.test/anthropic", Token: "secret", Model: "m"}
	sb := &Sandbox{Distro: "Ubuntu", Tools: `C:\tools`, ModCache: `C:\mod`}
	s := sb.script(c, []string{"claude", "-p", "say 'hi'"})
	for _, want := range []string{"mount --bind '/mnt/c/runs/001' /tmp/run", "mount -t tmpfs tmpfs /mnt",
		"export ANTHROPIC_AUTH_TOKEN=\"${VULCAN_SB_TOKEN:-}\"", `exec 'claude' '-p' 'say '\''hi'\'''`} {
		if !strings.Contains(s, want) {
			t.Fatalf("script lacks %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "secret") {
		t.Fatal("the script holds the key")
	}
}

// TestRunWithReview runs the scripted agent on the Go app: the reviewer rejects once, the agent
// resumes, the reviewer accepts, and a second review follows.
func TestRunWithReview(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the scripted agent")
	}
	dir := t.TempDir()
	agent := filepath.Join(dir, "fakeagent"+exeSuffix())
	if out, err := exec.Command("go", "build", "-o", agent, "../fakeagent").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	t.Setenv("FAKEAGENT_STATE", filepath.Join(dir, "reviews"))
	t.Setenv("FAKEAGENT_REJECTS", "1")
	c := &Config{Template: "../../app/go", Arm: "P", Task: "T1", Index: 1, Brief: "Add a thing.",
		OutDir: filepath.Join(dir, "run"), Agent: []string{agent},
		Review: &Review{MaxResubmits: 2, Agent: []string{agent}}}
	r, err := Run(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Reviews) != 3 || r.Reviews[0].Accept || !r.Reviews[1].Accept || r.Reviews[2].Round != 0 || !r.Reviews[0].Valid {
		t.Fatalf("reviews %+v", r.Reviews)
	}
	if r.Rejections() != 1 || !r.Accepted() || r.Reviews[0].Rules[0] != "R03" {
		t.Fatalf("rejections %d accepted %v", r.Rejections(), r.Accepted())
	}
	if r.Turns != 2 || len(r.Checks) != 2 || r.TokensOut != 100 {
		t.Fatalf("turns %d checks %d out %d", r.Turns, len(r.Checks), r.TokensOut)
	}
	if !strings.Contains(c.Prompt(), filepath.Join(c.OutDir, "scratch")) {
		t.Fatal("the prompt does not name the scratch directory")
	}
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
