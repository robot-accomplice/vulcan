# Stockroom-lite: functional description

Stockroom-lite is a small order, inventory, purchasing, and payment service. It keeps a product
catalog with stock levels, takes orders that reserve stock, takes payment through an external card
processor, ships orders, and handles cancellations and refunds. It buys stock from suppliers with
purchase orders, and receives their deliveries into stock. It tells other systems what happened
by webhooks: signed HTTP requests to the addresses that they subscribe. It is one process with an
HTTP API and a SQLite database.

This description is the same for every version of the application. It names no programming
language and no framework.

## Parts

```
                          Customer (HTTP client)
                                  |
                                [web]  ---- HTTP API
             ____________________/ | \_____________________
            /                      |                       \
     [inventory] <------------ [orders] ----> [billing]     [events] ---- [webhooks] ---- subscribers (HTTP)
        ^                          |              |            ^
        |                          +--------------|------------+
   [purchasing] ------------------------------------------------+
                                                  |
                                              [payments] ---- card processor (HTTP)

     inventory, orders, billing, purchasing, and events each also call [store] ---- SQLite file
     [shop]: shared types, used by all parts
```

Who calls whom:

- web calls inventory, orders, purchasing, and events.
- orders calls inventory, billing, events, and store.
- purchasing calls inventory, events, and store.
- billing calls payments and store.
- inventory calls store.
- events calls webhooks and store.
- payments calls the card processor; webhooks calls the subscribers; store uses the SQLite file.

| Part | Job |
|---|---|
| web | The HTTP API: decodes requests, calls inventory, orders, purchasing, and events, encodes responses and errors |
| inventory | The catalog and the stock levels: products, receipts of stock, reservations |
| orders | The order life cycle: create, pay, ship, cancel, refund |
| billing | Charges and refunds, and the ledger of each order's payments |
| purchasing | Suppliers and purchase orders: order stock from a supplier, receive its deliveries |
| events | Webhook subscriptions, the events that orders and purchasing raise, and their deliveries |
| store | Persistence in SQLite |
| payments | The client of the external card processor |
| webhooks | The client that sends webhooks to the subscribers |
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
- **Supplier:** `id` (`s-1`, `s-2`, … in order of creation), `name` (1 to 100 characters; no two
  suppliers have the same name).
- **Purchase order:** `id` (`po-1`, `po-2`, … in order of creation), `supplier_id`, `status`,
  `lines` (each `sku`, `qty` ordered (1 to 10,000), `received` so far, and `cost_cents`, the unit
  cost (more than 0, at most 100,000,000)), `total_cents` (the sum of qty × cost). For each line,
  `received` is 0 or more and never more than `qty`.
- **Purchase order status:** `open` → `received`; `open` → `cancelled`.
- **Webhook subscription:** `id` (`wh-1`, `wh-2`, … in order of creation), `url` (an absolute
  `http` or `https` URL with a host, at most 2,000 characters), `events` (the event types that it
  receives: at least one, each once), `secret` (made by the service when the subscription is
  created).
- **Event:** `id` (`ev-1`, `ev-2`, … in order of creation), `type`, `subject` (the id of the order
  or of the purchase order). The types:

  | Type | Raised when |
  |---|---|
  | `order.created` | an order is created |
  | `order.paid` | an order is paid |
  | `order.shipped` | an order is shipped |
  | `order.cancelled` | an order is cancelled |
  | `order.refunded` | a refund of an order is made (not the refund that a cancellation makes) |
  | `purchase_order.received` | the last units of a purchase order are received, so that it is `received` |

- **Delivery:** one event sent to one subscription: `event_id`, `type`, `subject`, `status`
  (`delivered` or `failed`), `attempts` (1 or more), `response_status` (the HTTP status of the
  last attempt, or 0 when there was no response).

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
7. A purchase order is for a supplier that exists, and each of its lines names a product of the
   catalog, once. Creating it changes no stock.
8. Receiving a delivery for an open purchase order adds each delivered line's quantity to the
   product's `on_hand`, and to the line's `received`. All lines of a delivery are received, or
   none. A line's `received` never goes above its `qty`: a delivery that would make it do so is a
   `conflict`. A delivery line for a product that is not on the purchase order is `invalid`.
9. When every line of a purchase order is received in full, its status becomes `received`.
10. An open purchase order can be cancelled. The units already received stay in stock. A received
    or cancelled purchase order cannot be received into or cancelled.
11. Each action in the table of event types raises one event, after its change is stored. Each
    subscription that lists the event's type, and that exists when the event is raised, gets one
    delivery of it, at once.
12. A delivery is a success when the subscriber answers with a 2xx status within 5 seconds. A
    failed delivery never fails or undoes the action that raised the event; it is kept as
    `failed`.
13. Retrying a subscription sends each of its failed deliveries again, oldest first, and counts
    the attempt. A delivered delivery is never sent again.
14. Deleting a subscription ends its deliveries, and removes it and its deliveries.

## HTTP API

All bodies are JSON. An error response has the body `{"error": "<code>", "message": "<text>"}`.

| Code | HTTP status | When |
|---|---|---|
| `invalid` | 400 | A field is missing or out of range |
| `not_found` | 404 | No such product, order, supplier, purchase order, or subscription |
| `conflict` | 409 | The product or the supplier exists, or the order or the purchase order is not in a state that allows the action |
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
| `POST /suppliers` | `{"name"}` | 201 the supplier `{"id", "name"}` |
| `GET /suppliers` | | 200 a list of suppliers, by id |
| `GET /suppliers/{id}` | | 200 the supplier |
| `GET /suppliers/{id}/purchase-orders` | | 200 a list of that supplier's purchase orders, by id |
| `POST /purchase-orders` | `{"supplier_id", "lines": [{"sku", "qty", "cost_cents"}]}`; at least one line, each sku once | 201 the purchase order, `open` |
| `GET /purchase-orders/{id}` | | 200 the purchase order |
| `POST /purchase-orders/{id}/receive` | `{"lines": [{"sku", "qty"}]}`; at least one line, qty 1 to 10,000, each sku once | 200 the purchase order |
| `POST /purchase-orders/{id}/cancel` | | 200 the purchase order, `cancelled` |
| `POST /webhooks` | `{"url", "events": ["<type>", …]}` | 201 `{"id", "url", "events", "secret"}` |
| `GET /webhooks` | | 200 a list of subscriptions `{"id", "url", "events"}`, by id |
| `GET /webhooks/{id}` | | 200 `{"id", "url", "events"}` |
| `DELETE /webhooks/{id}` | | 204, no body |
| `GET /webhooks/{id}/deliveries` | | 200 a list of deliveries, in the order of their events |
| `POST /webhooks/{id}/retry` | | 200 the list of deliveries, after the retry |

An order in a response is `{"id", "customer", "status", "lines": [{"sku", "qty", "price_cents"}],
"total_cents", "charged_cents", "refunded_cents", "payment_id"}`.

A purchase order in a response is `{"id", "supplier_id", "status", "lines": [{"sku", "qty",
"received", "cost_cents"}], "total_cents"}`. A delivery in a response is `{"event_id", "type",
"subject", "status", "attempts", "response_status"}`. Only the response that creates a
subscription shows its secret.

The `supplier_id` of a new purchase order that names no supplier is `not_found`, and so is a
product of a new purchase order that is not in the catalog. A supplier name that exists is a
`conflict`. An action on a purchase order that is not `open` is a `conflict`. An unknown event type
is `invalid`.

## The card processor

The card processor is an external HTTP service. Every request carries the header
`Authorization: Bearer <key>`.

| Method and path | Request | Response |
|---|---|---|
| `POST /charges` | `{"customer", "amount_cents"}` | 201 `{"id": "ch-N"}`, or 402 `{"error": "declined"}` |
| `POST /refunds` | `{"charge_id", "amount_cents"}` | 201 `{"id": "rf-N"}`; 404 for an unknown charge; 409 when the refunds would exceed the charge |

The test processor (`poc/fakepay`) declines every charge whose amount in cents ends in 13
(`amount_cents % 100 == 13`).

## The subscribers

A subscriber is an external HTTP service at the URL of a subscription. A delivery is one request:

- `POST <url>`, with the body `{"id": "ev-N", "type": "<type>", "subject": "<id>"}`;
- the header `Content-Type: application/json`;
- the header `Stockroom-Event: <type>`;
- the header `Stockroom-Signature: sha256=<hex>`, where `<hex>` is the HMAC-SHA256 of the exact
  body bytes with the subscription's secret as the key, in lower-case hex.

The service waits at most 5 seconds for the answer. It does not follow redirects: a redirect is a
failed delivery.

## Running

The process takes these settings, as command-line flags or environment variables:

| Flag | Variable | Default | Use |
|---|---|---|---|
| `--addr` | `ADDR` | `:8080` | The HTTP address |
| `--db` | `DB` | `stockroom.db` | The SQLite file |
| `--payments-url` | `PAYMENTS_URL` | none (required) | The card processor's base URL |
| `--payments-key` | `PAYMENTS_KEY` | none (required) | The card processor's key |

It stops on an interrupt signal, after the requests in progress finish.
