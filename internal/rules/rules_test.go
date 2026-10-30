package rules

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const oneRule = "rule A\n  kind forbid-call\n  level error\n  funcs os.Exit\n  reason R.\n  source S.\n"

func TestDefaultsParse(t *testing.T) {
	f, probs := Parse(Defaults())
	if len(probs) > 0 {
		t.Fatalf("defaults v1: %v", probs)
	}
	if len(f.Rules) == 0 {
		t.Fatal("defaults v1 has no rules")
	}
	for _, r := range f.Rules {
		if r.ID == "DEP-LOCK" && r.Level != Off {
			t.Errorf("the locked map is on in defaults v1; plan-r3, decision 1 has it off")
		}
	}
}

// TestCoverage checks that each wanted rule of the wishlist has one row, and that each stated rule
// keeps the wishlist's reason, word for word.
func TestCoverage(t *testing.T) {
	var wish struct {
		Rules []struct{ ID, Rule, Reason string } `json:"rules"`
	}
	var cov struct {
		Rules []struct {
			ID      string   `json:"id"`
			Status  string   `json:"status"`
			Entries []string `json:"entries"`
			Reason  *string  `json:"reason"`
		} `json:"rules"`
	}
	root := filepath.Join("..", "..", "experiments", "poc", "r3")
	for file, v := range map[string]any{filepath.Join(root, "wishlist", "wishlist.json"): &wish, filepath.Join(root, "defaults", "coverage.json"): &cov} {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatal(err)
		}
	}
	if len(wish.Rules) != 210 || len(cov.Rules) != len(wish.Rules) {
		t.Fatalf("%d wanted rules, %d rows", len(wish.Rules), len(cov.Rules))
	}
	f, _ := Parse(Defaults())
	text := string(Defaults())
	used := map[string]bool{}
	for i, c := range cov.Rules {
		w := wish.Rules[i]
		if c.ID != w.ID {
			t.Errorf("row %d is %s, the wishlist has %s", i, c.ID, w.ID)
		}
		switch c.Status {
		case "stated", "partly":
			if len(c.Entries) == 0 {
				t.Errorf("%s is %s and has no entry", c.ID, c.Status)
			}
			for _, id := range c.Entries {
				used[id] = true
				r := f.Find(id)
				if r == nil {
					t.Errorf("%s: no rule %s", c.ID, id)
				} else if r.Reason != w.Reason {
					t.Errorf("%s: the reason differs from the wishlist", id)
				}
			}
			if !strings.Contains(text, "# "+w.Rule+"\n") {
				t.Errorf("%s: the wanted text is not above its rule", c.ID)
			}
		case "not-stated":
			if c.Reason == nil || len(c.Entries) > 0 {
				t.Errorf("%s: not stated needs a reason, and no entry", c.ID)
			}
		default:
			t.Errorf("%s: status %q", c.ID, c.Status)
		}
	}
	for _, r := range f.Rules {
		if !used[r.ID] && !strings.HasPrefix(r.ID, "DEP-") {
			t.Errorf("rule %s comes from no wanted rule", r.ID)
		}
	}
}

func TestReadSource(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ok" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(oneRule))
	}))
	defer srv.Close()
	old := Client
	Client = srv.Client()
	defer func() { Client = old }()
	b, err := ReadSource(srv.URL+"/ok", "")
	if err != nil || string(b) != oneRule {
		t.Fatalf("https: %q, %v", b, err)
	}
	if _, err := ReadSource(srv.URL+"/no", ""); err == nil {
		t.Error("a 404 reads as a source")
	}
	if _, err := ReadSource("http://example.com/x", ""); err == nil || !strings.Contains(err.Error(), "https") {
		t.Errorf("plain http: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "r"), []byte(oneRule), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, err := ReadSource("r", dir); err != nil || string(b) != oneRule {
		t.Errorf("a local file: %q, %v", b, err)
	}
}

// TestLoadReadsNoSource: a check reads rules.vulcan, even when its origin is a URL that nothing
// answers, and when rules.origin is gone.
func TestLoadReadsNoSource(t *testing.T) {
	dir := t.TempDir()
	text := "origin https://nothing.invalid/rules\nhash sha256:0\n\n" + oneRule
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	s := Load(dir)
	if s == nil || len(s.Probs) > 0 || len(s.on) != 1 {
		t.Fatalf("load: %+v", s)
	}
	if err := os.Remove(filepath.Join(dir, FileName)); err != nil {
		t.Fatal(err)
	}
	if s := Load(dir); s != nil {
		t.Errorf("with no rules file, load gives rules: %+v", s)
	}
}

func TestHashIgnoresLineEnds(t *testing.T) {
	if Hash([]byte("a\r\nb\n")) != Hash([]byte("a\nb\n")) {
		t.Error("a checkout on Windows changes a hash")
	}
}

func TestParseProblems(t *testing.T) {
	for text, want := range map[string]string{
		"rule A\n  kind nope\n  level error\n":                                                                      "the kind",
		"rule A\n  kind forbid-call\n  level maybe\n  funcs x\n  reason R\n  source S\n":                            "the level",
		"rule A\n  kind forbid-call\n  level error\n  funcs x\n  source S\n":                                        "no reason",
		"rule A\n  kind forbid-call\n  level error\n  funcs x\n  reason R\n":                                        "no source",
		"rule A\n  kind forbid-call\n  level error\n  reason R\n  source S\n":                                       "needs funcs",
		"rule A\n  kind forbid-call\n  level error\n  funcs x\n  nope y\n  reason R\n  source S\n":                  "no parameter",
		"rule A\n  kind only-in\n  level error\n  effect rain\n  in leaf\n  reason R\n  source S\n":                 "the effect",
		"rule A\n  kind only-in\n  level error\n  effect clock\n  in sideways\n  reason R\n  source S\n":            "not a kind",
		"rule A\n  kind depends\n  level error\n  from leaf\n  reason R\n  source S\n":                              "needs deny",
		"rule A\n  kind depends\n  level error\n  map locked\n  from leaf\n  reason R\n  source S\n":                "takes no from",
		"rule A\n  kind transitions\n  level error\n  type lib/x.T\n  field S\n  allow a\n  reason R\n  source S\n": "allow holds",
		"stray line\n": "not a line of the head",
	} {
		_, probs := Parse([]byte(text))
		found := false
		for _, p := range probs {
			found = found || strings.Contains(p.Msg, want)
		}
		if !found {
			t.Errorf("%q: %v, want %q", text, probs, want)
		}
	}
}
