# Stockroom

Stockroom is an order, inventory, and payment service: an HTTP API and a SQLite database. It is
written in Go.

## Layout

| Path | Part |
|---|---|
| `cmd/stockroom` | The wiring of the parts, and the settings |
| `api/web` | The HTTP API |
| `api/inventory` | The catalog and the stock levels |
| `api/orders` | The order life cycle: create, pay, ship, cancel, refund |
| `api/billing` | Charges, refunds, and the ledger of each order |
| `api/store` | Persistence in SQLite |
| `api/payments` | The client of the card processor |
| `lib/shop` | Shared value types and errors |

## Check, build, and run

Check (run this before you finish a change):

```bash
go vet ./... && go test ./...
```

Build and run, with the test card processor in another terminal:

```bash
fakepay -addr 127.0.0.1:9090 -key test-key
```

```bash
go run ./cmd/stockroom --addr=127.0.0.1:8080 --db=stockroom.db --payments-url=http://127.0.0.1:9090 --payments-key=test-key
```

The test card processor declines every charge whose amount in cents ends in 13.
