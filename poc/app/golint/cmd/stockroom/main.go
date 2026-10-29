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
	"stockroom/api/events"
	"stockroom/api/inventory"
	"stockroom/api/orders"
	"stockroom/api/payments"
	"stockroom/api/purchasing"
	"stockroom/api/store"
	"stockroom/api/web"
	"stockroom/api/webhooks"
)

// flags are the settings of the process. Each reads --name=value from the command line, else
// the variable NAME (with - as _), else its default; a flag with no default is required.
var flags = []struct {
	name, def string
	required  bool
}{
	{"addr", ":8080", false},
	{"db", "stockroom.db", false},
	{"payments-url", "", true},
	{"payments-key", "", true},
}

// readFlags returns the value of each flag, or the list of problems.
func readFlags(args []string) (map[string]string, []string) {
	given := map[string]string{}
	known := map[string]bool{}
	for _, f := range flags {
		known[f.name] = true
	}
	var problems []string
	for _, a := range args {
		name, val, ok := strings.Cut(strings.TrimPrefix(a, "--"), "=")
		if !strings.HasPrefix(a, "--") || !ok || !known[name] {
			problems = append(problems, fmt.Sprintf("unknown or malformed flag %q", a))
			continue
		}
		given[name] = val
	}
	out := map[string]string{}
	for _, f := range flags {
		env := strings.ToUpper(strings.ReplaceAll(f.name, "-", "_"))
		if v, ok := given[f.name]; ok {
			out[f.name] = v
		} else if v, ok := os.LookupEnv(env); ok {
			out[f.name] = v
		} else if f.required {
			problems = append(problems, fmt.Sprintf("flag --%s (or %s) is required", f.name, env))
		} else {
			out[f.name] = f.def
		}
	}
	return out, problems
}

func main() {
	os.Exit(run())
}

// run wires the service and serves it. The customer calls web; web uses inventory, orders,
// purchasing, and events; orders uses inventory, billing, events, and store; purchasing uses
// inventory, events, and store; billing uses payments (the card processor) and store; events
// uses webhooks (the subscribers) and store; store keeps everything in the SQLite file.
func run() int {
	f, problems := readFlags(os.Args[1:])
	if len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "stockroom: reading flags failed:\n%s\n", strings.Join(problems, "\n"))
		return 1
	}
	addr, dbPath, payURL, payKey := f["addr"], f["db"], f["payments-url"], f["payments-key"]
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
	sender := webhooks.NewSender()
	hub := events.NewHub(db, sender)
	service := orders.NewService(catalog, ledger, db, hub)
	buying := purchasing.NewService(catalog, db, hub)
	api := web.New(catalog, service, buying, hub)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := api.Serve(ctx, web.Settings{Addr: addr}); err != nil {
		fmt.Fprintf(os.Stderr, "stockroom: %v\n", err)
		return 1
	}
	return 0
}
