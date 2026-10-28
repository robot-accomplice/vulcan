// Package provider reads the balance of the provider's account (round 2, F8), so that the spend
// of a batch of runs is known from the provider's own books, not only from token counts.
package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Balance is the account's balance at one time.
type Balance struct {
	Time     time.Time `json:"time"`
	Order    int       `json:"after_order"` // the last run before the reading; 0 before the first
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
	Error    string    `json:"error,omitempty"`
}

// endpoint is the balance request of a provider, by the host of its Anthropic-compatible URL,
// and how to read the answer.
type endpoint struct {
	url  string
	read func([]byte) (float64, string, error)
}

var endpoints = map[string]endpoint{
	"api.deepseek.com": {"https://api.deepseek.com/user/balance", func(b []byte) (float64, string, error) {
		var v struct {
			Infos []struct {
				Currency string `json:"currency"`
				Total    string `json:"total_balance"`
			} `json:"balance_infos"`
		}
		if err := json.Unmarshal(b, &v); err != nil || len(v.Infos) == 0 {
			return 0, "", fmt.Errorf("deepseek balance: %v %s", err, b)
		}
		f, err := strconv.ParseFloat(v.Infos[0].Total, 64)
		return f, v.Infos[0].Currency, err
	}},
	"openrouter.ai": {"https://openrouter.ai/api/v1/credits", func(b []byte) (float64, string, error) {
		var v struct {
			Data struct {
				Credits float64 `json:"total_credits"`
				Usage   float64 `json:"total_usage"`
			} `json:"data"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			return 0, "", fmt.Errorf("openrouter credits: %v", err)
		}
		return v.Data.Credits - v.Data.Usage, "USD", nil
	}},
	"api.moonshot.ai": {"https://api.moonshot.ai/v1/users/me/balance", func(b []byte) (float64, string, error) {
		var v struct {
			Data struct {
				Available float64 `json:"available_balance"`
			} `json:"data"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			return 0, "", fmt.Errorf("moonshot balance: %v", err)
		}
		return v.Data.Available, "USD", nil
	}},
}

// Read reads the balance of the provider at baseURL. A provider with no known balance request,
// or a failed request, gives a Balance with Error set: the runs go on.
func Read(baseURL, token string, order int) Balance {
	out := Balance{Time: time.Now().UTC(), Order: order}
	host := baseURL
	if i := strings.Index(host, "://"); i >= 0 {
		host = host[i+3:]
	}
	host, _, _ = strings.Cut(host, "/")
	ep, ok := endpoints[host]
	if !ok {
		out.Error = "no balance request for " + host
		return out
	}
	req, err := http.NewRequest(http.MethodGet, ep.url, nil)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if resp.StatusCode != http.StatusOK {
		out.Error = fmt.Sprintf("status %d", resp.StatusCode)
		return out
	}
	amount, cur, err := ep.read(b)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	out.Amount, out.Currency = amount, cur
	return out
}
