// Command worker runs Picking. It reads one shipment per line from stdin,
// "<order-id> <note>", and announces each one.
package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	"shop/bus"
	"shop/picking"
	"shop/shop"
)

func main() {
	// In this base the bus is in-memory; it stands in for the channel to the api process.
	p := picking.New(bus.New())
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		id, note, _ := strings.Cut(strings.TrimSpace(in.Text()), " ")
		if id == "" {
			continue
		}
		if err := p.Ship(shop.OrderID(id), note); err != nil {
			log.Printf("ship %s: %v", id, err)
			continue
		}
		log.Printf("shipped %s", id)
	}
	if err := in.Err(); err != nil {
		log.Fatalf("read stdin: %v", err)
	}
}
