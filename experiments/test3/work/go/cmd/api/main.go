// Command api runs Orders, Billing, and the web API in one process.
// Billing writes its audit lines to Postgres.
package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq" // registers the "postgres" driver

	"shop/audit"
	"shop/billing"
	"shop/bus"
	"shop/orders"
	"shop/shop"
	"shop/web"
)

// startupTimeout bounds the database ping and migration at start.
const startupTimeout = 10 * time.Second

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dsn := flag.String("db", os.Getenv("DATABASE_URL"), "Postgres connection string for the audit trail (default $DATABASE_URL)")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("no audit database: set -db or DATABASE_URL")
	}
	db, err := sql.Open("postgres", *dsn)
	if err != nil {
		log.Fatalf("open audit database: %v", err)
	}
	defer db.Close()
	trail := audit.NewPostgres(db)
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("reach audit database: %v", err)
	}
	if err := trail.Migrate(ctx); err != nil {
		log.Fatal(err)
	}
	cancel()

	svc := orders.New(billing.NewLedger(trail))

	// The bus that carries OrderShipped from the worker. In this base it is in-memory,
	// so it stands in for the channel between the two processes.
	events := bus.New()
	events.Subscribe(svc.HandleShipped)
	events.DeadLetter = func(ev shop.OrderShipped, err error) {
		log.Printf("dead letter: order %s: %v", ev.Order, err)
	}

	log.Printf("api listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, web.Handler(svc, trail)))
}
