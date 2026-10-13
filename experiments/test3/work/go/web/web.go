// Package web serves the HTTP API for orders.
package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"shop/audit"
	"shop/orders"
	"shop/shop"
)

// Orders is what the web API needs from the orders service.
type Orders interface {
	Place(customer shop.CustomerID, amount shop.Money) (orders.Order, error)
	Cancel(id shop.OrderID) (orders.Order, error)
	Get(id shop.OrderID) (orders.Order, error)
}

// AuditTrail is what the web API needs to read Billing's audit lines.
type AuditTrail interface {
	ForOrder(ctx context.Context, order shop.OrderID) ([]audit.Entry, error)
}

type placeRequest struct {
	Customer shop.CustomerID `json:"customer"`
	Amount   shop.Money      `json:"amount"`
}

// Handler returns the routes:
//
//	POST /orders              {"customer":"c1","amount":1250}
//	POST /orders/{id}/cancel
//	GET  /orders/{id}
//	GET  /orders/{id}/audit
func Handler(o Orders, a AuditTrail) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var req placeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if req.Customer == "" {
			http.Error(w, "customer is required", http.StatusBadRequest)
			return
		}
		ord, err := o.Place(req.Customer, req.Amount)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, ord)
	})
	mux.HandleFunc("POST /orders/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		ord, err := o.Cancel(shop.OrderID(r.PathValue("id")))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ord)
	})
	mux.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		ord, err := o.Get(shop.OrderID(r.PathValue("id")))
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, ord)
	})
	mux.HandleFunc("GET /orders/{id}/audit", func(w http.ResponseWriter, r *http.Request) {
		id := shop.OrderID(r.PathValue("id"))
		// An unknown order is a 404, not an empty trail.
		if _, err := o.Get(id); err != nil {
			writeErr(w, err)
			return
		}
		trail, err := a.ForOrder(r.Context(), id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, trail)
	})
	return mux
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, orders.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, orders.ErrBadAmount):
		status = http.StatusBadRequest
	case errors.Is(err, orders.ErrCancelled), errors.Is(err, orders.ErrShipped):
		status = http.StatusConflict
	}
	http.Error(w, err.Error(), status)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) // the status is already sent; nothing left to report to
}
