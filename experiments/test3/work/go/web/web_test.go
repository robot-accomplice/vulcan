package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shop/audit"
	"shop/billing"
	"shop/orders"
)

// The three routes work end to end against the real services.
func TestPlaceCancelGet(t *testing.T) {
	trail := &audit.Memory{}
	h := Handler(orders.New(billing.NewLedger(trail)), trail)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/orders", strings.NewReader(`{"customer":"c1","amount":1250}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("place: %d %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/orders/o-1/cancel", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel: %d %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/orders/o-1", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"cancelled"`) {
		t.Fatalf("get: %d %s", rec.Code, rec.Body)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/orders/o-404", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get missing: %d", rec.Code)
	}
}

// The audit route shows the charge from Place and the refund from Cancel, in
// that order, and answers 404 for an order that does not exist.
func TestOrderAuditTrail(t *testing.T) {
	trail := &audit.Memory{}
	h := Handler(orders.New(billing.NewLedger(trail)), trail)

	for _, req := range []*http.Request{
		httptest.NewRequest("POST", "/orders", strings.NewReader(`{"customer":"c1","amount":1250}`)),
		httptest.NewRequest("POST", "/orders/o-1/cancel", nil),
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code >= 300 {
			t.Fatalf("%s %s: %d %s", req.Method, req.URL, rec.Code, rec.Body)
		}
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/orders/o-1/audit", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("audit: %d %s", rec.Code, rec.Body)
	}
	var got []audit.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("audit body %q: %v", rec.Body, err)
	}
	if len(got) != 2 ||
		got[0].Kind != audit.Charge || got[0].Amount != 1250 ||
		got[1].Kind != audit.Refund || got[1].Amount != 1250 {
		t.Fatalf("audit trail: %+v", got)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/orders/o-404/audit", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("audit for missing order: %d", rec.Code)
	}
}
