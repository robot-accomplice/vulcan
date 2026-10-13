// Command api runs Orders, Billing, and the web API in one process.
package main

import (
	"flag"
	"log"
	"net/http"

	"shop/billing"
	"shop/bus"
	"shop/orders"
	"shop/shop"
	"shop/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	svc := orders.New(billing.NewLedger())

	// The bus that carries OrderShipped from the worker. In this base it is in-memory,
	// so it stands in for the channel between the two processes.
	events := bus.New()
	events.Subscribe(svc.HandleShipped)
	events.DeadLetter = func(ev shop.OrderShipped, err error) {
		log.Printf("dead letter: order %s: %v", ev.Order, err)
	}

	log.Printf("api listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, web.Handler(svc)))
}
