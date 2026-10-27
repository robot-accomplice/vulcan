// Command fakepay runs the test card processor of Stockroom-lite.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/robot-accomplice/vulcan/poc/fakepay"
)

func main() {
	addr := flag.String("addr", ":9090", "the HTTP address")
	key := flag.String("key", "test-key", "the key that clients send")
	flag.Parse()
	log.Printf("fakepay on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, fakepay.New(*key)))
}
