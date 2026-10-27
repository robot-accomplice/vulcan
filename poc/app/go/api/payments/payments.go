package payments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stockroom/lib/shop"
)

// Settings say where the card processor is, and the key to send it.
type Settings struct {
	URL string
	Key string
}

// Client calls the card processor.
type Client struct {
	base string
	key  string
	http *http.Client
}

// NewClient returns a client of the card processor at s.URL.
func NewClient(s Settings) (*Client, error) {
	u, err := url.Parse(s.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("payments-url %q is not an http or https URL", s.URL)
	}
	if s.Key == "" {
		return nil, fmt.Errorf("payments-key is empty")
	}
	return &Client{base: strings.TrimSuffix(s.URL, "/"), key: s.Key, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

// post sends a JSON request, and decodes a JSON response into out. It returns the status.
func (c *Client) post(path string, in, out any) (int, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, shop.Errorf(shop.CodeUnavailable, "the card processor cannot be reached: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 == 2 && out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, shop.Errorf(shop.CodeUnavailable, "the card processor sent a bad response: %v", err)
		}
	}
	return resp.StatusCode, nil
}

// Charge charges a customer, and returns the processor's charge id.
func (c *Client) Charge(customer shop.CustomerID, amount shop.Money) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	status, err := c.post("/charges", map[string]any{"customer": customer, "amount_cents": amount}, &out)
	switch {
	case err != nil:
		return "", err
	case status == http.StatusPaymentRequired:
		return "", shop.Errorf(shop.CodeDeclined, "the card processor declined the charge")
	case status != http.StatusCreated:
		return "", shop.Errorf(shop.CodeUnavailable, "the card processor answered %d", status)
	}
	return out.ID, nil
}

// Refund refunds part of a charge, and returns the processor's refund id.
func (c *Client) Refund(chargeID string, amount shop.Money) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	status, err := c.post("/refunds", map[string]any{"charge_id": chargeID, "amount_cents": amount}, &out)
	switch {
	case err != nil:
		return "", err
	case status == http.StatusConflict:
		return "", shop.Errorf(shop.CodeConflict, "the card processor refused a refund larger than the charge")
	case status != http.StatusCreated:
		return "", shop.Errorf(shop.CodeUnavailable, "the card processor answered %d", status)
	}
	return out.ID, nil
}
