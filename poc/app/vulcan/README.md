# Stockroom

Stockroom is an order, inventory, purchasing, and payment service, with webhooks: an HTTP API
and a SQLite database. It is written in Vulcan (`vulcan guide` prints the primer).

## Layout

| Path | Part |
|---|---|
| `system.vulcan` | The wiring of the parts, and the settings |
| `api/web` | The HTTP API |
| `api/inventory` | The catalog and the stock levels |
| `api/orders` | The order life cycle: create, pay, ship, cancel, refund |
| `api/billing` | Charges, refunds, and the ledger of each order |
| `api/purchasing` | Suppliers and purchase orders: order stock from a supplier, receive its deliveries |
| `api/events` | Webhook subscriptions, the events, and their deliveries |
| `api/store` | Persistence in SQLite |
| `api/payments` | The client of the card processor |
| `api/webhooks` | The client that sends webhooks to the subscribers |
| `lib/shop` | Shared value types and errors |

## Check, build, and run

Check (run this before you finish a change):

```bash
vulcan check && vulcan test --race=false
```

Build and run, with the test card processor in another terminal:

```bash
fakepay -addr 127.0.0.1:9090 -key test-key
```

```bash
vulcan run -- --addr=127.0.0.1:8080 --db=stockroom.db --payments-url=http://127.0.0.1:9090 --payments-key=test-key
```

The test card processor declines every charge whose amount in cents ends in 13.

The `go` commands (`go build`, `go vet`) do not work in this directory, because the code is in
`.vulcan` files: use `vulcan build`, `vulcan check`, and `vulcan mod tidy`. `.vulcan/` is
generated; do not edit it.
