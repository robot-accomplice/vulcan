package acceptance

// The acceptance tests of the two services added in round 2 (docs/poc/plan-r2.md, section 4):
// purchasing, and webhooks.

import (
	"encoding/json"
	"strings"
	"testing"

	st "github.com/robot-accomplice/vulcan/poc/stockroomtest"
)

func list(t *testing.T, a *st.App, path string) []map[string]any {
	t.Helper()
	status, out := a.Do("GET", path, nil)
	l, ok := out.([]any)
	if status != 200 || !ok {
		t.Fatalf("GET %s: status %d, body %v", path, status, out)
	}
	var ms []map[string]any
	for _, e := range l {
		ms = append(ms, e.(map[string]any))
	}
	return ms
}

func TestSuppliers(t *testing.T) {
	a := st.Start(t)
	s := a.Want("POST", "/suppliers", map[string]any{"name": "Acme"}, 201, "")
	if s["id"] != "s-1" || s["name"] != "Acme" {
		t.Fatalf("supplier: %v", s)
	}
	a.Want("POST", "/suppliers", map[string]any{"name": "Acme"}, 409, "conflict")
	a.Want("POST", "/suppliers", map[string]any{"name": ""}, 400, "invalid")
	a.Want("POST", "/suppliers", map[string]any{"name": strings.Repeat("x", 101)}, 400, "invalid")
	a.Want("POST", "/suppliers", map[string]any{"name": "Bolt Co"}, 201, "")
	if l := list(t, a, "/suppliers"); len(l) != 2 || l[0]["id"] != "s-1" || l[1]["name"] != "Bolt Co" {
		t.Fatalf("suppliers: %v", l)
	}
	if s := a.Want("GET", "/suppliers/s-2", nil, 200, ""); s["name"] != "Bolt Co" {
		t.Fatalf("supplier s-2: %v", s)
	}
	a.Want("GET", "/suppliers/s-9", nil, 404, "not_found")
	a.Want("GET", "/suppliers/bogus", nil, 404, "not_found")
	a.Want("GET", "/suppliers/s-9/purchase-orders", nil, 404, "not_found")
	if l := list(t, a, "/suppliers/s-1/purchase-orders"); len(l) != 0 {
		t.Fatalf("purchase orders of a new supplier: %v", l)
	}
}

func purchase(a *st.App, supplier string, lines ...any) map[string]any {
	a.T.Helper()
	var ls []map[string]any
	for i := 0; i < len(lines); i += 3 {
		ls = append(ls, map[string]any{"sku": lines[i], "qty": lines[i+1], "cost_cents": lines[i+2]})
	}
	return a.Want("POST", "/purchase-orders", map[string]any{"supplier_id": supplier, "lines": ls}, 201, "")
}

func receipt(lines ...any) map[string]any {
	var ls []map[string]any
	for i := 0; i < len(lines); i += 2 {
		ls = append(ls, map[string]any{"sku": lines[i], "qty": lines[i+1]})
	}
	return map[string]any{"lines": ls}
}

func TestPurchaseOrders(t *testing.T) {
	a := st.Start(t)
	a.Product("A-1", "Lamp", 1500, 2)
	a.Product("B-2", "Bulb", 250, 0)
	a.Want("POST", "/suppliers", map[string]any{"name": "Acme"}, 201, "")
	po := purchase(a, "s-1", "A-1", 10, 900, "B-2", 4, 100)
	if po["id"] != "po-1" || po["supplier_id"] != "s-1" || po["status"] != "open" || st.Num(po, "total_cents") != 9400 {
		t.Fatalf("purchase order: %v", po)
	}
	lines, _ := po["lines"].([]any)
	if len(lines) != 2 {
		t.Fatalf("lines: %v", po["lines"])
	}
	l0 := lines[0].(map[string]any)
	if l0["sku"] != "A-1" || st.Num(l0, "qty") != 10 || st.Num(l0, "received") != 0 || st.Num(l0, "cost_cents") != 900 {
		t.Fatalf("first line: %v", l0)
	}
	a.Stock(t, "A-1", 2, 0) // creating a purchase order changes no stock
	a.Stock(t, "B-2", 0, 0)
	for _, bad := range []map[string]any{
		{"supplier_id": "s-1", "lines": []any{}},
		{"supplier_id": "s-1", "lines": []any{map[string]any{"sku": "A-1", "qty": 0, "cost_cents": 1}}},
		{"supplier_id": "s-1", "lines": []any{map[string]any{"sku": "A-1", "qty": 10001, "cost_cents": 1}}},
		{"supplier_id": "s-1", "lines": []any{map[string]any{"sku": "A-1", "qty": 1, "cost_cents": 0}}},
		{"supplier_id": "s-1", "lines": []any{map[string]any{"sku": "A-1", "qty": 1, "cost_cents": 100000001}}},
		{"supplier_id": "s-1", "lines": []any{map[string]any{"sku": "A-1", "qty": 1, "cost_cents": 1}, map[string]any{"sku": "A-1", "qty": 1, "cost_cents": 1}}},
	} {
		a.Want("POST", "/purchase-orders", bad, 400, "invalid")
	}
	a.Want("POST", "/purchase-orders", map[string]any{"supplier_id": "s-9", "lines": []any{map[string]any{"sku": "A-1", "qty": 1, "cost_cents": 1}}}, 404, "not_found")
	a.Want("POST", "/purchase-orders", map[string]any{"supplier_id": "s-1", "lines": []any{map[string]any{"sku": "NOPE", "qty": 1, "cost_cents": 1}}}, 404, "not_found")
	if g := a.Want("GET", "/purchase-orders/po-1", nil, 200, ""); st.Num(g, "total_cents") != 9400 {
		t.Fatalf("get: %v", g)
	}
	a.Want("GET", "/purchase-orders/po-9", nil, 404, "not_found")
	a.Want("GET", "/purchase-orders/o-1", nil, 404, "not_found")
	a.Want("POST", "/suppliers", map[string]any{"name": "Bolt Co"}, 201, "")
	purchase(a, "s-2", "B-2", 1, 100)
	purchase(a, "s-1", "B-2", 2, 100)
	if l := list(t, a, "/suppliers/s-1/purchase-orders"); len(l) != 2 || l[0]["id"] != "po-1" || l[1]["id"] != "po-3" {
		t.Fatalf("purchase orders of s-1: %v", l)
	}
}

func TestReceivePurchase(t *testing.T) {
	a := st.Start(t)
	a.Product("A-1", "Lamp", 1500, 2)
	a.Product("B-2", "Bulb", 250, 0)
	a.Product("C-3", "Cord", 300, 0)
	a.Order("ada", "A-1", 1)
	a.Want("POST", "/suppliers", map[string]any{"name": "Acme"}, 201, "")
	purchase(a, "s-1", "A-1", 10, 900, "B-2", 4, 100)

	po := a.Want("POST", "/purchase-orders/po-1/receive", receipt("A-1", 6), 200, "")
	if po["status"] != "open" || st.Num(po["lines"].([]any)[0].(map[string]any), "received") != 6 {
		t.Fatalf("after a partial delivery: %v", po)
	}
	a.Stock(t, "A-1", 8, 1) // the reservation of the order stays
	a.Stock(t, "B-2", 0, 0)

	// All lines or none: B-2 is fine, A-1 is over, so neither is received.
	a.Want("POST", "/purchase-orders/po-1/receive", receipt("B-2", 1, "A-1", 5), 409, "conflict")
	a.Stock(t, "B-2", 0, 0)
	a.Want("POST", "/purchase-orders/po-1/receive", receipt("B-2", 1, "C-3", 1), 400, "invalid") // C-3 is not on it
	a.Stock(t, "B-2", 0, 0)
	for _, bad := range []map[string]any{
		receipt(),
		receipt("B-2", 0),
		receipt("B-2", 10001),
		receipt("B-2", 1, "B-2", 1),
	} {
		a.Want("POST", "/purchase-orders/po-1/receive", bad, 400, "invalid")
	}
	if g := a.Want("GET", "/purchase-orders/po-1", nil, 200, ""); st.Num(g["lines"].([]any)[1].(map[string]any), "received") != 0 {
		t.Fatalf("after refused deliveries: %v", g)
	}

	po = a.Want("POST", "/purchase-orders/po-1/receive", receipt("B-2", 4, "A-1", 4), 200, "")
	if po["status"] != "received" {
		t.Fatalf("after the last units: %v", po)
	}
	a.Stock(t, "A-1", 12, 1)
	a.Stock(t, "B-2", 4, 0)
	a.Want("POST", "/purchase-orders/po-1/receive", receipt("A-1", 1), 409, "conflict")
	a.Want("POST", "/purchase-orders/po-1/cancel", nil, 409, "conflict")
	a.Want("POST", "/purchase-orders/po-9/receive", receipt("A-1", 1), 404, "not_found")
	a.Order("bob", "B-2", 4) // the received units can be ordered
}

func TestCancelPurchase(t *testing.T) {
	a := st.Start(t)
	a.Product("A-1", "Lamp", 1500, 0)
	a.Want("POST", "/suppliers", map[string]any{"name": "Acme"}, 201, "")
	purchase(a, "s-1", "A-1", 10, 900)
	a.Want("POST", "/purchase-orders/po-1/receive", receipt("A-1", 3), 200, "")
	po := a.Want("POST", "/purchase-orders/po-1/cancel", nil, 200, "")
	if po["status"] != "cancelled" {
		t.Fatalf("cancelled: %v", po)
	}
	a.Stock(t, "A-1", 3, 0) // the received units stay
	a.Want("POST", "/purchase-orders/po-1/cancel", nil, 409, "conflict")
	a.Want("POST", "/purchase-orders/po-1/receive", receipt("A-1", 1), 409, "conflict")
	a.Want("POST", "/purchase-orders/po-9/cancel", nil, 404, "not_found")
}

type hookBody struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Subject string `json:"subject"`
}

func decodeHook(t *testing.T, h st.Hook) hookBody {
	t.Helper()
	var b hookBody
	if err := json.Unmarshal(h.Body, &b); err != nil {
		t.Fatalf("webhook body %q: %v", h.Body, err)
	}
	return b
}

func TestWebhookSubscriptions(t *testing.T) {
	a := st.Start(t)
	rc := st.NewReceiver(t)
	for _, bad := range []map[string]any{
		{"url": "", "events": []any{"order.paid"}},
		{"url": "ftp://example.com/x", "events": []any{"order.paid"}},
		{"url": "/relative", "events": []any{"order.paid"}},
		{"url": "http://example.com/" + strings.Repeat("x", 2000), "events": []any{"order.paid"}},
		{"url": rc.URL, "events": []any{}},
		{"url": rc.URL, "events": []any{"order.lost"}},
		{"url": rc.URL, "events": []any{"order.paid", "order.paid"}},
	} {
		a.Want("POST", "/webhooks", bad, 400, "invalid")
	}
	s := a.Want("POST", "/webhooks", map[string]any{"url": rc.URL, "events": []any{"order.paid", "purchase_order.received"}}, 201, "")
	if s["id"] != "wh-1" || s["url"] != rc.URL || s["secret"] == nil || s["secret"] == "" {
		t.Fatalf("subscription: %v", s)
	}
	if ev, _ := s["events"].([]any); len(ev) != 2 || ev[0] != "order.paid" || ev[1] != "purchase_order.received" {
		t.Fatalf("events: %v", s["events"])
	}
	s2 := a.Want("POST", "/webhooks", map[string]any{"url": rc.URL, "events": []any{"order.created"}}, 201, "")
	if s2["secret"] == s["secret"] {
		t.Fatalf("two subscriptions with one secret")
	}
	l := list(t, a, "/webhooks")
	if len(l) != 2 || l[0]["id"] != "wh-1" || l[1]["id"] != "wh-2" {
		t.Fatalf("subscriptions: %v", l)
	}
	for _, m := range append(l, a.Want("GET", "/webhooks/wh-1", nil, 200, "")) {
		if _, ok := m["secret"]; ok {
			t.Fatalf("a secret is shown after the creation: %v", m)
		}
	}
	a.Want("GET", "/webhooks/wh-9", nil, 404, "not_found")
	a.Want("GET", "/webhooks/bogus", nil, 404, "not_found")
	if status, _ := a.Do("DELETE", "/webhooks/wh-2", nil); status != 204 {
		t.Fatalf("delete: status %d", status)
	}
	a.Want("GET", "/webhooks/wh-2", nil, 404, "not_found")
	a.Want("GET", "/webhooks/wh-2/deliveries", nil, 404, "not_found")
	a.Want("DELETE", "/webhooks/wh-2", nil, 404, "not_found")
	a.Want("POST", "/webhooks/wh-2/retry", nil, 404, "not_found")
	if l := list(t, a, "/webhooks"); len(l) != 1 {
		t.Fatalf("subscriptions after a delete: %v", l)
	}
}

func TestWebhookDeliveries(t *testing.T) {
	a := st.Start(t)
	rc := st.NewReceiver(t)
	a.Product("A-1", "Lamp", 1500, 5)
	a.Order("ada", "A-1", 1) // ev-1, before any subscription
	all := []any{"order.created", "order.paid", "order.shipped", "order.cancelled", "order.refunded", "purchase_order.received"}
	s := a.Want("POST", "/webhooks", map[string]any{"url": rc.URL, "events": all}, 201, "")
	secret := s["secret"].(string)
	only := st.NewReceiver(t)
	a.Want("POST", "/webhooks", map[string]any{"url": only.URL, "events": []any{"order.shipped"}}, 201, "")
	if len(rc.Hooks()) != 0 {
		t.Fatalf("a subscription got an event from before it: %v", rc.Hooks())
	}

	a.Want("POST", "/orders/o-1/pay", nil, 200, "")                                     // ev-2
	a.Want("POST", "/orders/o-1/refunds", map[string]any{"amount_cents": 100}, 200, "") // ev-3
	a.Want("POST", "/orders/o-1/ship", nil, 200, "")                                    // ev-4
	a.Order("bob", "A-1", 1)                                                            // ev-5
	a.Want("POST", "/orders/o-2/pay", nil, 200, "")                                     // ev-6
	a.Want("POST", "/orders/o-2/cancel", nil, 200, "")                                  // ev-7: no order.refunded
	a.Want("POST", "/suppliers", map[string]any{"name": "Acme"}, 201, "")
	purchase(a, "s-1", "A-1", 2, 900)
	a.Want("POST", "/purchase-orders/po-1/receive", receipt("A-1", 1), 200, "") // not yet received: no event
	a.Want("POST", "/purchase-orders/po-1/receive", receipt("A-1", 1), 200, "") // ev-8

	want := []hookBody{
		{"ev-2", "order.paid", "o-1"}, {"ev-3", "order.refunded", "o-1"}, {"ev-4", "order.shipped", "o-1"},
		{"ev-5", "order.created", "o-2"}, {"ev-6", "order.paid", "o-2"}, {"ev-7", "order.cancelled", "o-2"},
		{"ev-8", "purchase_order.received", "po-1"},
	}
	hooks := rc.Hooks()
	if len(hooks) != len(want) {
		t.Fatalf("webhooks: got %d, want %d: %v", len(hooks), len(want), hooks)
	}
	for i, h := range hooks {
		if b := decodeHook(t, h); b != want[i] {
			t.Fatalf("webhook %d: %+v, want %+v", i, b, want[i])
		}
		if h.Event != want[i].Type {
			t.Fatalf("webhook %d: Stockroom-Event %q, want %q", i, h.Event, want[i].Type)
		}
		if !h.Signed(secret) {
			t.Fatalf("webhook %d: bad signature %q", i, h.Signature)
		}
	}
	if h := only.Hooks(); len(h) != 1 || decodeHook(t, h[0]) != want[2] {
		t.Fatalf("the order.shipped subscription got %v", h)
	}
	d := list(t, a, "/webhooks/wh-1/deliveries")
	if len(d) != len(want) {
		t.Fatalf("deliveries: %v", d)
	}
	for i, m := range d {
		if m["event_id"] != want[i].ID || m["type"] != want[i].Type || m["subject"] != want[i].Subject ||
			m["status"] != "delivered" || st.Num(m, "attempts") != 1 || st.Num(m, "response_status") != 204 {
			t.Fatalf("delivery %d: %v", i, m)
		}
	}
}

func TestWebhookFailures(t *testing.T) {
	a := st.Start(t)
	rc := st.NewReceiver(t)
	a.Product("A-1", "Lamp", 1500, 5)
	a.Want("POST", "/webhooks", map[string]any{"url": rc.URL, "events": []any{"order.created", "order.paid"}}, 201, "")
	a.Want("POST", "/webhooks", map[string]any{"url": "http://" + st.FreeAddr(t) + "/hooks", "events": []any{"order.created"}}, 201, "")

	rc.FailNext(1)
	o := a.Order("ada", "A-1", 1) // the failed deliveries do not fail the order
	if o["id"] != "o-1" {
		t.Fatalf("order: %v", o)
	}
	a.Stock(t, "A-1", 5, 1)
	a.Want("POST", "/orders/o-1/pay", nil, 200, "")
	d := list(t, a, "/webhooks/wh-1/deliveries")
	if len(d) != 2 || d[0]["status"] != "failed" || st.Num(d[0], "response_status") != 500 || st.Num(d[0], "attempts") != 1 ||
		d[1]["status"] != "delivered" {
		t.Fatalf("deliveries after a failure: %v", d)
	}
	if d := list(t, a, "/webhooks/wh-2/deliveries"); len(d) != 1 || d[0]["status"] != "failed" || st.Num(d[0], "response_status") != 0 {
		t.Fatalf("delivery to a closed port: %v", d)
	}

	_, out := a.Do("POST", "/webhooks/wh-1/retry", nil)
	r, _ := out.([]any)
	if len(r) != 2 {
		t.Fatalf("retry: %v", out)
	}
	if m := r[0].(map[string]any); m["status"] != "delivered" || st.Num(m, "attempts") != 2 || st.Num(m, "response_status") != 204 {
		t.Fatalf("retried delivery: %v", m)
	}
	if m := r[1].(map[string]any); st.Num(m, "attempts") != 1 {
		t.Fatalf("a delivered delivery was sent again: %v", m)
	}
	hooks := rc.Hooks()
	if len(hooks) != 3 || decodeHook(t, hooks[2]).ID != "ev-1" {
		t.Fatalf("webhooks after a retry: %v", hooks)
	}
	a.Want("POST", "/webhooks/wh-1/retry", nil, 200, "")
	if len(rc.Hooks()) != 3 {
		t.Fatalf("a retry with nothing failed sent %d webhooks", len(rc.Hooks())-3)
	}

	if status, _ := a.Do("DELETE", "/webhooks/wh-1", nil); status != 204 {
		t.Fatalf("delete: status %d", status)
	}
	a.Order("bob", "A-1", 1)
	if len(rc.Hooks()) != 3 {
		t.Fatalf("a deleted subscription got a webhook")
	}
}
