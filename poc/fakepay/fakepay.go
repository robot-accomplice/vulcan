// Package fakepay is the test card processor of Stockroom-lite (docs/poc/stockroom.md). It keeps
// its charges in memory. It declines every charge whose amount in cents ends in 13.
package fakepay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Server is a card processor.
type Server struct {
	Key string

	mu       sync.Mutex
	charges  map[string]*charge
	order    []string
	nextCh   int
	nextRf   int
	refunds  int
	failNext int // the next n requests fail with 500
}

type charge struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Amount   int64  `json:"amount_cents"`
	Refunded int64  `json:"refunded_cents"`
}

// New returns a processor that accepts the key.
func New(key string) *Server {
	return &Server{Key: key, charges: map[string]*charge{}}
}

// FailNext makes the next n requests fail with status 500.
func (s *Server) FailNext(n int) {
	s.mu.Lock()
	s.failNext = n
	s.mu.Unlock()
}

// Charge returns a charge, for tests.
func (s *Server) Charge(id string) (amount, refunded int64, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.charges[id]
	if !ok {
		return 0, 0, false
	}
	return c.Amount, c.Refunded, true
}

// Charges returns the number of charges, for tests.
func (s *Server) Charges() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.charges)
}

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ServeHTTP serves POST /charges, POST /refunds, and GET /charges/{id}.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+s.Key {
		reply(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failNext > 0 {
		s.failNext--
		reply(w, http.StatusInternalServerError, map[string]string{"error": "unavailable"})
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/charges":
		var req struct {
			Customer string `json:"customer"`
			Amount   int64  `json:"amount_cents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 || req.Customer == "" {
			reply(w, http.StatusBadRequest, map[string]string{"error": "invalid"})
			return
		}
		if req.Amount%100 == 13 {
			reply(w, http.StatusPaymentRequired, map[string]string{"error": "declined"})
			return
		}
		s.nextCh++
		c := &charge{ID: fmt.Sprintf("ch-%d", s.nextCh), Customer: req.Customer, Amount: req.Amount}
		s.charges[c.ID] = c
		s.order = append(s.order, c.ID)
		reply(w, http.StatusCreated, map[string]string{"id": c.ID})
	case r.Method == http.MethodPost && r.URL.Path == "/refunds":
		var req struct {
			ChargeID string `json:"charge_id"`
			Amount   int64  `json:"amount_cents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
			reply(w, http.StatusBadRequest, map[string]string{"error": "invalid"})
			return
		}
		c, ok := s.charges[req.ChargeID]
		if !ok {
			reply(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		if c.Refunded+req.Amount > c.Amount {
			reply(w, http.StatusConflict, map[string]string{"error": "exceeds_charge"})
			return
		}
		c.Refunded += req.Amount
		s.nextRf++
		reply(w, http.StatusCreated, map[string]string{"id": fmt.Sprintf("rf-%d", s.nextRf)})
	case r.Method == http.MethodGet && len(r.URL.Path) > len("/charges/") && r.URL.Path[:len("/charges/")] == "/charges/":
		c, ok := s.charges[r.URL.Path[len("/charges/"):]]
		if !ok {
			reply(w, http.StatusNotFound, map[string]string{"error": "not_found"})
			return
		}
		reply(w, http.StatusOK, c)
	default:
		reply(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	}
}
