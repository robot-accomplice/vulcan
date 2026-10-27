# Stockroom-lite: functional description

Stockroom-lite is a small order, inventory, and payment service. It keeps a product catalog
with stock levels, takes orders that reserve stock, takes payment through an external card
processor, ships orders, and handles cancellations and refunds. It is one process with an HTTP
API and a SQLite database.

This description is the same for every version of the application. It names no programming
language and no framework.

## Parts

```
            Customer (HTTP client)
                   |
                 [web]  ---- HTTP API
                /     \
        [inventory]  [orders] ---------- [billing]
              |        |                  |      \
              +------[store]--------------+   [payments] ---- card processor (HTTP)
                       |
                    SQLite file
                [shop]: shared types, used by all parts
```

| Part | Job |
|---|---|
| web | The HTTP API: decodes requests, calls inventory and orders, encodes responses and errors |
| inventory | The catalog and the stock levels: products, receipts of stock, reservations |
| orders | The order life cycle: create, pay, ship, cancel, refund |
| billing | Charges and refunds, and the ledger of each order's payments |
| store | Persistence in SQLite |
| payments | The client of the external card processor |
| shop | Shared value types and errors |

## Data

- **Money** is an integer number of cents. There are no fractional cents.
- **Product:** `sku` (1 to 32 characters: letters, digits, `-`), `name` (1 to 100 characters),
  `price_cents` (more than 0, at most 100,000,000).
- **Quantities:** a receipt of stock and a line of an order have a `qty` of 1 to 10,000.
- **Stock** of a product: `on_hand` (units in the stockroom), `reserved` (units held by orders that
  are not shipped or cancelled), `available` = `on_hand` − `reserved`. Each is 0 or more, and
  `reserved` is never more than `on_hand`.
- **Order:** `id` (`o-1`, `o-2`, … in order of creation), `customer` (1 to 64 characters),
  `status`, `lines` (each `sku`, `qty`, and the `price_cents` at the time of the order),
  `total_cents` (the sum of qty × price), `charged_cents`, `refunded_cents`, `payment_id` (the card
  processor's charge id, or empty).
- **Status:** `pending` → `paid` → `shipped`; `pending` or `paid` → `cancelled`.
- **Payment entry:** `kind` (`charge` or `refund`), `amount_cents`, `provider_id`.

## Rules of the business

1. An order reserves its stock when it is created. All lines are reserved, or none.
2. Paying an order charges its total once. A declined or failed charge leaves the order pending.
3. Shipping a paid order takes its units out of stock: `on_hand` and `reserved` both go down by
   each line's quantity.
4. Cancelling a pending order releases its reservation. Cancelling a paid order also refunds
   whatever is charged and not yet refunded. A shipped or cancelled order cannot be cancelled.
5. A refund is for a paid or shipped order, is more than 0, and is never more than
   `charged_cents` − `refunded_cents`. A refund of 0 or less is `invalid`; a refund of more than
   can be refunded is a `conflict`.
6. The ledger of an order lists each charge and each refund, in order, and its sums equal the
   order's `charged_cents` and `refunded_cents`.

## HTTP API

All bodies are JSON. An error response has the body `{"error": "<code>", "message": "<text>"}`.

| Code | HTTP status | When |
|---|---|---|
| `invalid` | 400 | A field is missing or out of range |
| `not_found` | 404 | No such product or order |
| `conflict` | 409 | The product exists, or the order is not in a state that allows the action |
| `out_of_stock` | 409 | Not enough available stock for an order |
| `payment_declined` | 402 | The card processor declined the charge |
| `payment_unavailable` | 502 | The card processor could not be reached, or failed |

| Method and path | Request | Response |
|---|---|---|
| `GET /health` | | 200 `{"status": "ok"}` |
| `POST /products` | `{"sku", "name", "price_cents"}` | 201 the product with its stock |
| `GET /products` | | 200 a list of products with stock, by sku |
| `GET /products/{sku}` | | 200 `{"sku", "name", "price_cents", "on_hand", "reserved", "available"}` |
| `POST /products/{sku}/receive` | `{"qty"}`, qty > 0 | 200 the product with its stock |
| `POST /orders` | `{"customer", "lines": [{"sku", "qty"}]}`; at least one line, qty > 0, each sku once | 201 the order |
| `GET /orders/{id}` | | 200 the order |
| `GET /orders?customer=c` | | 200 a list of that customer's orders, by id |
| `POST /orders/{id}/pay` | | 200 the order, `paid` |
| `POST /orders/{id}/ship` | | 200 the order, `shipped` |
| `POST /orders/{id}/cancel` | | 200 the order, `cancelled` |
| `POST /orders/{id}/refunds` | `{"amount_cents"}` | 200 the order |
| `GET /orders/{id}/payments` | | 200 a list of payment entries, in order |

An order in a response is `{"id", "customer", "status", "lines": [{"sku", "qty", "price_cents"}],
"total_cents", "charged_cents", "refunded_cents", "payment_id"}`.

## The card processor

The card processor is an external HTTP service. Every request carries the header
`Authorization: Bearer <key>`.

| Method and path | Request | Response |
|---|---|---|
| `POST /charges` | `{"customer", "amount_cents"}` | 201 `{"id": "ch-N"}`, or 402 `{"error": "declined"}` |
| `POST /refunds` | `{"charge_id", "amount_cents"}` | 201 `{"id": "rf-N"}`; 404 for an unknown charge; 409 when the refunds would exceed the charge |

The test processor (`poc/fakepay`) declines every charge whose amount in cents ends in 13
(`amount_cents % 100 == 13`).

## Running

The process takes these settings, as command-line flags or environment variables:

| Flag | Variable | Default | Use |
|---|---|---|---|
| `--addr` | `ADDR` | `:8080` | The HTTP address |
| `--db` | `DB` | `stockroom.db` | The SQLite file |
| `--payments-url` | `PAYMENTS_URL` | none (required) | The card processor's base URL |
| `--payments-key` | `PAYMENTS_KEY` | none (required) | The card processor's key |

It stops on an interrupt signal, after the requests in progress finish.
