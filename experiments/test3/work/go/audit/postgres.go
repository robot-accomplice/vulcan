package audit

import (
	"context"
	"database/sql"
	"fmt"

	"shop/shop"
)

// Schema creates the audit table. It is safe to run more than once.
const Schema = `
CREATE TABLE IF NOT EXISTS billing_audit (
	id       BIGSERIAL   PRIMARY KEY,
	order_id TEXT        NOT NULL,
	customer TEXT        NOT NULL,
	kind     TEXT        NOT NULL CHECK (kind IN ('charge', 'refund')),
	amount   BIGINT      NOT NULL CHECK (amount > 0),
	at       TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS billing_audit_order_id ON billing_audit (order_id, id);
`

// Postgres stores audit lines in the billing_audit table.
// The caller opens db with a registered Postgres driver.
type Postgres struct {
	db *sql.DB
}

// NewPostgres returns a store that uses db.
func NewPostgres(db *sql.DB) *Postgres { return &Postgres{db: db} }

// Migrate creates the table if it does not exist.
func (p *Postgres) Migrate(ctx context.Context) error {
	if _, err := p.db.ExecContext(ctx, Schema); err != nil {
		return fmt.Errorf("audit: migrate: %w", err)
	}
	return nil
}

// Record inserts e.
func (p *Postgres) Record(ctx context.Context, e Entry) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO billing_audit (order_id, customer, kind, amount, at) VALUES ($1, $2, $3, $4, $5)`,
		string(e.Order), string(e.Customer), string(e.Kind), int64(e.Amount), e.At)
	if err != nil {
		return fmt.Errorf("audit: record %s %s %d: %w", e.Kind, e.Order, e.Amount, err)
	}
	return nil
}

// ForOrder returns the lines for order, oldest first.
func (p *Postgres) ForOrder(ctx context.Context, order shop.OrderID) ([]Entry, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT order_id, customer, kind, amount, at FROM billing_audit WHERE order_id = $1 ORDER BY id`,
		string(order))
	if err != nil {
		return nil, fmt.Errorf("audit: read %s: %w", order, err)
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Order, &e.Customer, &e.Kind, &e.Amount, &e.At); err != nil {
			return nil, fmt.Errorf("audit: read %s: %w", order, err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit: read %s: %w", order, err)
	}
	return out, nil
}
