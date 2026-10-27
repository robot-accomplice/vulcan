// Package acceptance holds the black-box acceptance tests of Stockroom-lite
// (docs/poc/stockroom.md). They run one version of the application, the binary that
// STOCKROOM_BIN names, against the test card processor, over HTTP. The same tests serve both
// versions (arm V and arm G).
//
//	STOCKROOM_BIN=/path/to/stockroom go test ./poc/acceptance
package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/robot-accomplice/vulcan/poc/fakepay"
)

// app is one running instance of the application.
type app struct {
	t    *testing.T
	base string
	pay  *fakepay.Server
}

func freeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

// start runs the application with a new database, and stops it when the test ends.
func start(t *testing.T) *app {
	t.Helper()
	bin := os.Getenv("STOCKROOM_BIN")
	if bin == "" {
		t.Skip("STOCKROOM_BIN is not set")
	}
	pay := fakepay.New("test-key")
	ps := httptest.NewServer(pay)
	t.Cleanup(ps.Close)
	addr := freeAddr(t)
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
	a := &app{t: t, base: "http://" + addr, pay: pay}
	for i := 0; ; i++ {
		resp, err := http.Get(a.base + "/health")
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
func (a *app) do(method, path string, body any) (int, any) {
	a.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, a.base+path, r)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		a.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var out any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// want checks a status, and for an error the error code.
func (a *app) want(method, path string, body any, status int, code string) map[string]any {
	a.t.Helper()
	got, out := a.do(method, path, body)
	m, _ := out.(map[string]any)
	if got != status {
		a.t.Fatalf("%s %s: status %d, want %d; body %v", method, path, got, status, out)
	}
	if code != "" && (m == nil || m["error"] != code) {
		a.t.Fatalf("%s %s: error %v, want %q", method, path, m, code)
	}
	return m
}

func num(m map[string]any, k string) int64 {
	f, _ := m[k].(float64)
	return int64(f)
}

func (a *app) product(sku, name string, price, qty int64) {
	a.t.Helper()
	a.want("POST", "/products", map[string]any{"sku": sku, "name": name, "price_cents": price}, 201, "")
	if qty > 0 {
		a.want("POST", "/products/"+sku+"/receive", map[string]any{"qty": qty}, 200, "")
	}
}

func (a *app) order(customer string, lines ...any) map[string]any {
	a.t.Helper()
	var ls []map[string]any
	for i := 0; i < len(lines); i += 2 {
		ls = append(ls, map[string]any{"sku": lines[i], "qty": lines[i+1]})
	}
	return a.want("POST", "/orders", map[string]any{"customer": customer, "lines": ls}, 201, "")
}

func (a *app) stock(t *testing.T, sku string, onHand, reserved int64) {
	t.Helper()
	p := a.want("GET", "/products/"+sku, nil, 200, "")
	if num(p, "on_hand") != onHand || num(p, "reserved") != reserved || num(p, "available") != onHand-reserved {
		t.Fatalf("stock of %s: %v, want on_hand %d reserved %d", sku, p, onHand, reserved)
	}
}

func TestHealth(t *testing.T) {
	a := start(t)
	m := a.want("GET", "/health", nil, 200, "")
	if m["status"] != "ok" {
		t.Fatalf("health: %v", m)
	}
}

func TestProducts(t *testing.T) {
	a := start(t)
	m := a.want("POST", "/products", map[string]any{"sku": "B-2", "name": "Bulb", "price_cents": 250}, 201, "")
	if m["sku"] != "B-2" || num(m, "price_cents") != 250 || num(m, "on_hand") != 0 || num(m, "available") != 0 {
		t.Fatalf("created product: %v", m)
	}
	a.want("POST", "/products", map[string]any{"sku": "B-2", "name": "Again", "price_cents": 1}, 409, "conflict")
	a.product("A-1", "Lamp", 1500, 0)
	for _, bad := range []map[string]any{
		{"sku": "", "name": "x", "price_cents": 1},
		{"sku": "has space", "name": "x", "price_cents": 1},
		{"sku": "C-3", "name": "", "price_cents": 1},
		{"sku": "C-3", "name": "x", "price_cents": 0},
	} {
		a.want("POST", "/products", bad, 400, "invalid")
	}
	_, list := a.do("GET", "/products", nil)
	l, _ := list.([]any)
	if len(l) != 2 || l[0].(map[string]any)["sku"] != "A-1" || l[1].(map[string]any)["sku"] != "B-2" {
		t.Fatalf("products by sku: %v", list)
	}
	a.want("GET", "/products/NOPE", nil, 404, "not_found")
	p := a.want("POST", "/products/A-1/receive", map[string]any{"qty": 7}, 200, "")
	if num(p, "on_hand") != 7 || num(p, "available") != 7 {
		t.Fatalf("after receive: %v", p)
	}
	a.want("POST", "/products/A-1/receive", map[string]any{"qty": 0}, 400, "invalid")
	a.want("POST", "/products/NOPE/receive", map[string]any{"qty": 1}, 404, "not_found")
}

func TestOrderReservesStock(t *testing.T) {
	a := start(t)
	a.product("A-1", "Lamp", 1500, 5)
	a.product("B-2", "Bulb", 250, 1)
	o := a.order("ada", "A-1", 2, "B-2", 1)
	if o["id"] != "o-1" || o["status"] != "pending" || num(o, "total_cents") != 3250 || num(o, "charged_cents") != 0 {
		t.Fatalf("order: %v", o)
	}
	lines, _ := o["lines"].([]any)
	if len(lines) != 2 || num(lines[0].(map[string]any), "price_cents") != 1500 {
		t.Fatalf("lines: %v", o["lines"])
	}
	a.stock(t, "A-1", 5, 2)
	a.stock(t, "B-2", 1, 1)
	// All lines or none: B-2 has nothing left, so A-1 is not reserved either.
	a.want("POST", "/orders", map[string]any{"customer": "bob", "lines": []any{
		map[string]any{"sku": "A-1", "qty": 1}, map[string]any{"sku": "B-2", "qty": 1}}}, 409, "out_of_stock")
	a.stock(t, "A-1", 5, 2)
	a.want("POST", "/orders", map[string]any{"customer": "bob", "lines": []any{map[string]any{"sku": "NOPE", "qty": 1}}}, 404, "not_found")
	for _, bad := range []map[string]any{
		{"customer": "", "lines": []any{map[string]any{"sku": "A-1", "qty": 1}}},
		{"customer": "bob", "lines": []any{}},
		{"customer": "bob", "lines": []any{map[string]any{"sku": "A-1", "qty": 0}}},
		{"customer": "bob", "lines": []any{map[string]any{"sku": "A-1", "qty": 1}, map[string]any{"sku": "A-1", "qty": 1}}},
	} {
		a.want("POST", "/orders", bad, 400, "invalid")
	}
	a.stock(t, "A-1", 5, 2)
	o2 := a.order("ada", "A-1", 1)
	if o2["id"] != "o-2" {
		t.Fatalf("second order id: %v", o2["id"])
	}
	_, list := a.do("GET", "/orders?customer=ada", nil)
	if l, _ := list.([]any); len(l) != 2 || l[0].(map[string]any)["id"] != "o-1" {
		t.Fatalf("orders of ada: %v", list)
	}
	_, list = a.do("GET", "/orders?customer=nobody", nil)
	if l, ok := list.([]any); !ok || len(l) != 0 {
		t.Fatalf("orders of nobody: %v", list)
	}
	a.want("GET", "/orders/o-99", nil, 404, "not_found")
	a.want("GET", "/orders/bogus", nil, 404, "not_found")
}

func TestPayShip(t *testing.T) {
	a := start(t)
	a.product("A-1", "Lamp", 1500, 5)
	a.order("ada", "A-1", 2)
	o := a.want("POST", "/orders/o-1/pay", nil, 200, "")
	if o["status"] != "paid" || num(o, "charged_cents") != 3000 || o["payment_id"] == "" {
		t.Fatalf("paid: %v", o)
	}
	if amount, _, ok := a.pay.Charge(o["payment_id"].(string)); !ok || amount != 3000 {
		t.Fatalf("processor charge: %d %v", amount, ok)
	}
	a.want("POST", "/orders/o-1/pay", nil, 409, "conflict")
	if a.pay.Charges() != 1 {
		t.Fatalf("charges at the processor: %d, want 1", a.pay.Charges())
	}
	o = a.want("POST", "/orders/o-1/ship", nil, 200, "")
	if o["status"] != "shipped" {
		t.Fatalf("shipped: %v", o)
	}
	a.stock(t, "A-1", 3, 0)
	a.want("POST", "/orders/o-1/ship", nil, 409, "conflict")
	a.order("bob", "A-1", 1)
	a.want("POST", "/orders/o-2/ship", nil, 409, "conflict") // pending, not paid
}

func TestPaymentFailures(t *testing.T) {
	a := start(t)
	a.product("D-1", "Odd", 113, 5) // 113 cents ends in 13: declined
	a.product("A-1", "Lamp", 1500, 5)
	a.order("ada", "D-1", 1)
	a.want("POST", "/orders/o-1/pay", nil, 402, "payment_declined")
	if o := a.want("GET", "/orders/o-1", nil, 200, ""); o["status"] != "pending" || num(o, "charged_cents") != 0 {
		t.Fatalf("after a decline: %v", o)
	}
	a.order("ada", "A-1", 1)
	a.pay.FailNext(1)
	a.want("POST", "/orders/o-2/pay", nil, 502, "payment_unavailable")
	if o := a.want("GET", "/orders/o-2", nil, 200, ""); o["status"] != "pending" {
		t.Fatalf("after a failure: %v", o)
	}
	a.want("POST", "/orders/o-2/pay", nil, 200, "")
	_, l := a.do("GET", "/orders/o-1/payments", nil)
	if list, ok := l.([]any); !ok || len(list) != 0 {
		t.Fatalf("ledger after a decline: %v", l)
	}
}

func TestCancel(t *testing.T) {
	a := start(t)
	a.product("A-1", "Lamp", 1500, 5)
	a.order("ada", "A-1", 2)
	o := a.want("POST", "/orders/o-1/cancel", nil, 200, "")
	if o["status"] != "cancelled" || num(o, "refunded_cents") != 0 {
		t.Fatalf("cancelled pending: %v", o)
	}
	a.stock(t, "A-1", 5, 0)
	a.want("POST", "/orders/o-1/cancel", nil, 409, "conflict")
	a.want("POST", "/orders/o-1/pay", nil, 409, "conflict")

	a.order("ada", "A-1", 3)
	a.want("POST", "/orders/o-2/pay", nil, 200, "")
	a.want("POST", "/orders/o-2/refunds", map[string]any{"amount_cents": 1000}, 200, "")
	o = a.want("POST", "/orders/o-2/cancel", nil, 200, "")
	if o["status"] != "cancelled" || num(o, "charged_cents") != 4500 || num(o, "refunded_cents") != 4500 {
		t.Fatalf("cancelled paid: %v", o)
	}
	if _, refunded, _ := a.pay.Charge(o["payment_id"].(string)); refunded != 4500 {
		t.Fatalf("refunded at the processor: %d", refunded)
	}
	a.stock(t, "A-1", 5, 0)

	a.order("ada", "A-1", 1)
	a.want("POST", "/orders/o-3/pay", nil, 200, "")
	a.want("POST", "/orders/o-3/ship", nil, 200, "")
	a.want("POST", "/orders/o-3/cancel", nil, 409, "conflict")
	a.want("POST", "/orders/o-99/cancel", nil, 404, "not_found")
}

func TestRefunds(t *testing.T) {
	a := start(t)
	a.product("A-1", "Lamp", 1500, 5)
	a.order("ada", "A-1", 2)
	a.want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 100}, 409, "conflict") // pending
	a.want("POST", "/orders/o-1/pay", nil, 200, "")
	a.want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 0}, 400, "invalid")
	a.want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": -5}, 400, "invalid")
	o := a.want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 1200}, 200, "")
	if num(o, "refunded_cents") != 1200 || o["status"] != "paid" {
		t.Fatalf("after a refund: %v", o)
	}
	a.want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 1801}, 409, "conflict")
	a.want("POST", "/orders/o-1/ship", nil, 200, "")
	o = a.want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 1800}, 200, "")
	if num(o, "refunded_cents") != 3000 || o["status"] != "shipped" {
		t.Fatalf("refund of a shipped order: %v", o)
	}
	a.want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 1}, 409, "conflict")
	_, l := a.do("GET", "/orders/o-1/payments", nil)
	list, _ := l.([]any)
	if len(list) != 3 {
		t.Fatalf("ledger: %v", l)
	}
	var charged, refunded int64
	for i, e := range list {
		m := e.(map[string]any)
		switch m["kind"] {
		case "charge":
			charged += num(m, "amount_cents")
			if i != 0 {
				t.Fatalf("the charge is not first: %v", l)
			}
		case "refund":
			refunded += num(m, "amount_cents")
		}
		if m["provider_id"] == "" {
			t.Fatalf("entry without provider id: %v", m)
		}
	}
	if charged != 3000 || refunded != 3000 {
		t.Fatalf("ledger sums: %d %d", charged, refunded)
	}
	a.want("GET", "/orders/o-99/payments", nil, 404, "not_found")
}

func TestBadRequests(t *testing.T) {
	a := start(t)
	req, _ := http.NewRequest("POST", a.base+"/products", bytes.NewReader([]byte("{not json")))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("bad JSON: status %d", resp.StatusCode)
	}
	a.want("GET", "/orders", nil, 400, "invalid") // no customer
}

func TestRestartKeepsData(t *testing.T) {
	// The database file outlives the process.
	bin := os.Getenv("STOCKROOM_BIN")
	if bin == "" {
		t.Skip("STOCKROOM_BIN is not set")
	}
	db := filepath.Join(t.TempDir(), "keep.db")
	pay := httptest.NewServer(fakepay.New("k"))
	defer pay.Close()
	run := func(f func(base string)) {
		addr := freeAddr(t)
		ctx, cancel := context.WithCancel(context.Background())
		cmd := exec.CommandContext(ctx, bin, "--addr="+addr, "--db="+db, "--payments-url="+pay.URL, "--payments-key=k")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() { cancel(); _ = cmd.Wait() }()
		base := "http://" + addr
		for i := 0; i < 100; i++ {
			if resp, err := http.Get(base + "/health"); err == nil {
				resp.Body.Close()
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		f(base)
	}
	run(func(base string) {
		b, _ := json.Marshal(map[string]any{"sku": "K-1", "name": "Kept", "price_cents": 10})
		resp, err := http.Post(base+"/products", "application/json", bytes.NewReader(b))
		if err != nil || resp.StatusCode != 201 {
			t.Fatalf("create: %v %v", err, resp)
		}
		resp.Body.Close()
	})
	run(func(base string) {
		resp, err := http.Get(base + "/products/K-1")
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("after a restart: %v %v", err, resp)
		}
		resp.Body.Close()
	})
}

func TestMissingSettings(t *testing.T) {
	bin := os.Getenv("STOCKROOM_BIN")
	if bin == "" {
		t.Skip("STOCKROOM_BIN is not set")
	}
	cmd := exec.Command(bin, "--addr=127.0.0.1:0", fmt.Sprintf("--db=%s", filepath.Join(t.TempDir(), "x.db")))
	cmd.Env = append(os.Environ(), "PAYMENTS_URL=", "PAYMENTS_KEY=")
	cmd.Env = filterEnv(cmd.Env, "PAYMENTS_URL", "PAYMENTS_KEY")
	out, err := cmd.CombinedOutput()
	ee, ok := err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 1 || !bytes.Contains(out, []byte("payments-url")) {
		t.Fatalf("without payments settings: %v\n%s", err, out)
	}
}

func filterEnv(env []string, names ...string) []string {
	var out []string
	for _, e := range env {
		keep := true
		for _, n := range names {
			if len(e) > len(n) && e[:len(n)+1] == n+"=" {
				keep = false
			}
		}
		if keep {
			out = append(out, e)
		}
	}
	return out
}
