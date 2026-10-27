// Command stockroom runs the Stockroom service (docs/poc/stockroom.md).
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"stockroom/api/billing"
	"stockroom/api/inventory"
	"stockroom/api/orders"
	"stockroom/api/payments"
	"stockroom/api/store"
	"stockroom/api/web"
)

// setting reads --name=value from the command line, else the variable NAME (with - as _), else
// the default. A setting with no default is required.
type settings struct {
	args     map[string]string
	problems []string
}

func (s *settings) get(name, def string, required bool) string {
	if v, ok := s.args[name]; ok {
		return v
	}
	env := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if v, ok := os.LookupEnv(env); ok {
		return v
	}
	if required {
		s.problems = append(s.problems, fmt.Sprintf("flag --%s (or %s) is required", name, env))
	}
	return def
}

func main() {
	os.Exit(run())
}

func run() int {
	s := &settings{args: map[string]string{}}
	known := map[string]bool{"addr": true, "db": true, "payments-url": true, "payments-key": true}
	for _, a := range os.Args[1:] {
		name, val, ok := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !strings.HasPrefix(a, "--") || !ok || !known[name] {
			s.problems = append(s.problems, fmt.Sprintf("unknown or malformed flag %q", a))
			continue
		}
		s.args[name] = val
	}
	addr := s.get("addr", ":8080", false)
	dbPath := s.get("db", "stockroom.db", false)
	payURL := s.get("payments-url", "", true)
	payKey := s.get("payments-key", "", true)
	if len(s.problems) > 0 {
		fmt.Fprintf(os.Stderr, "stockroom: reading flags failed:\n%s\n", strings.Join(s.problems, "\n"))
		return 1
	}
	db, err := store.Open(store.Settings{Path: dbPath})
	if err != nil {
		fmt.Fprintf(os.Stderr, "stockroom: open the database: %v\n", err)
		return 1
	}
	card, err := payments.NewClient(payments.Settings{URL: payURL, Key: payKey})
	if err != nil {
		fmt.Fprintf(os.Stderr, "stockroom: %v\n", err)
		return 1
	}
	catalog := inventory.NewCatalog(db)
	ledger := billing.NewLedger(card, db)
	service := orders.NewService(catalog, ledger, db)
	api := web.New(catalog, service)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := api.Serve(ctx, web.Settings{Addr: addr}); err != nil {
		fmt.Fprintf(os.Stderr, "stockroom: %v\n", err)
		return 1
	}
	return 0
}
