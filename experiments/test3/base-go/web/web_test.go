package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"shop/billing"
	"shop/orders"
)

// The three routes work end to end against the real services.
func TestPlaceCancelGet(t *testing.T) {
	h := Handler(orders.New(billing.NewLedger()))

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
