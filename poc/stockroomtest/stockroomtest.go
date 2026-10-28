// Package stockroomtest runs Stockroom-lite for black-box tests over HTTP: the acceptance
// suite, and the hidden tests of the PoC evaluation (docs/poc/plan.md, section 6). The binary is
// the one that STOCKROOM_BIN names; the card processor is fakepay, in process.
package stockroomtest

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/robot-accomplice/vulcan/poc/fakepay"
)

// App is one running instance of the application.
type App struct {
	T    *testing.T
	Base string
	Pay  *fakepay.Server
}

func FreeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

// start runs the application with a new database, and stops it when the test ends.
func Start(t *testing.T) *App {
	t.Helper()
	bin := os.Getenv("STOCKROOM_BIN")
	if bin == "" {
		t.Skip("STOCKROOM_BIN is not set")
	}
	pay := fakepay.New("test-key")
	ps := httptest.NewServer(pay)
	t.Cleanup(ps.Close)
	addr := FreeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, bin, "--addr="+addr, "--db="+filepath.Join(t.TempDir(), "stockroom.db"),
		"--payments-url="+ps.URL, "--payments-key=test-key")
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		_ = cmd.Wait()
		if t.Failed() {
			t.Logf("application output:\n%s", logs.String())
		}
	})
	a := &App{T: t, Base: "http://" + addr, Pay: pay}
	for i := 0; ; i++ {
		resp, err := http.Get(a.Base + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return a
			}
		}
		if i > 100 {
			t.Fatalf("the application did not start: %v\n%s", err, logs.String())
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// do sends a request and decodes the JSON response into a map or a list.
func (a *App) Do(method, path string, body any) (int, any) {
	a.T.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, a.Base+path, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.T.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var out any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// want checks a status, and for an error the error code.
func (a *App) Want(method, path string, body any, status int, code string) map[string]any {
	a.T.Helper()
	got, out := a.Do(method, path, body)
	m, _ := out.(map[string]any)
	if got != status {
		a.T.Fatalf("%s %s: status %d, want %d; body %v", method, path, got, status, out)
	}
	if code != "" && (m == nil || m["error"] != code) {
		a.T.Fatalf("%s %s: error %v, want %q", method, path, m, code)
	}
	return m
}

func Num(m map[string]any, k string) int64 {
	f, _ := m[k].(float64)
	return int64(f)
}

func (a *App) Product(sku, name string, price, qty int64) {
	a.T.Helper()
	a.Want("POST", "/products", map[string]any{"sku": sku, "name": name, "price_cents": price}, 201, "")
	if qty > 0 {
		a.Want("POST", "/products/"+sku+"/receive", map[string]any{"qty": qty}, 200, "")
	}
}

func (a *App) Order(customer string, lines ...any) map[string]any {
	a.T.Helper()
	var ls []map[string]any
	for i := 0; i < len(lines); i += 2 {
		ls = append(ls, map[string]any{"sku": lines[i], "qty": lines[i+1]})
	}
	return a.Want("POST", "/orders", map[string]any{"customer": customer, "lines": ls}, 201, "")
}

func (a *App) Stock(t *testing.T, sku string, onHand, reserved int64) {
	t.Helper()
	p := a.Want("GET", "/products/"+sku, nil, 200, "")
	if Num(p, "on_hand") != onHand || Num(p, "reserved") != reserved || Num(p, "available") != onHand-reserved {
		t.Fatalf("stock of %s: %v, want on_hand %d reserved %d", sku, p, onHand, reserved)
	}
}

// Receiver is a webhook subscriber for tests: it records each request, and answers 204, or 500
// while it is set to fail.
type Receiver struct {
	URL string

	mu       sync.Mutex
	hooks    []Hook
	failNext int
}

// Hook is one webhook that a Receiver got.
type Hook struct {
	Event     string // the Stockroom-Event header
	Signature string // the Stockroom-Signature header
	Body      []byte
}

// NewReceiver starts a receiver, and stops it when the test ends.
func NewReceiver(t *testing.T) *Receiver {
	t.Helper()
	rc := &Receiver{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rc.mu.Lock()
		defer rc.mu.Unlock()
		rc.hooks = append(rc.hooks, Hook{Event: r.Header.Get("Stockroom-Event"), Signature: r.Header.Get("Stockroom-Signature"), Body: body})
		if rc.failNext > 0 {
			rc.failNext--
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	rc.URL = srv.URL
	return rc
}

// FailNext makes the next n requests fail with status 500.
func (rc *Receiver) FailNext(n int) {
	rc.mu.Lock()
	rc.failNext = n
	rc.mu.Unlock()
}

// Hooks returns the webhooks received so far, failed ones included, in order.
func (rc *Receiver) Hooks() []Hook {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return append([]Hook(nil), rc.hooks...)
}

// Signed reports whether a hook carries the signature of its body with the secret.
func (h Hook) Signed(secret string) bool {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(h.Body)
	return h.Signature == "sha256="+hex.EncodeToString(m.Sum(nil))
}
