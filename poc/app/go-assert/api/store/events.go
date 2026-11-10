package store

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"stockroom/lib/shop"
)

// AddSubscription stores a new webhook subscription and returns its id.
func (d *DB) AddSubscription(sub shop.Subscription) (shop.SubscriptionID, error) {
	res, err := d.db.Exec(`insert into subscriptions (url, events, secret) values (?, ?, ?)`, sub.URL, joinTypes(sub.Events), sub.Secret)
	if err != nil {
		return "", err
	}
	key, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	return shop.SubscriptionID("wh-" + strconv.FormatInt(key, 10)), nil
}

func joinTypes(types []shop.EventType) string {
	s := make([]string, len(types))
	for i, t := range types {
		s[i] = string(t)
	}
	return strings.Join(s, ",")
}

func scanSubscription(r scanner) (shop.Subscription, error) {
	var key int64
	var u, events, secret string
	if err := r.Scan(&key, &u, &events, &secret); err != nil {
		return shop.Subscription{}, err
	}
	sub := shop.Subscription{ID: shop.SubscriptionID("wh-" + strconv.FormatInt(key, 10)), URL: u, Secret: secret}
	for t := range strings.SplitSeq(events, ",") {
		sub.Events = append(sub.Events, shop.EventType(t))
	}
	return sub, nil
}

// Subscription returns a webhook subscription.
func (d *DB) Subscription(id shop.SubscriptionID) (shop.Subscription, error) {
	key, err := keyOf("wh-", string(id), "subscription")
	if err != nil {
		return shop.Subscription{}, err
	}
	sub, err := scanSubscription(d.db.QueryRow(`select id, url, events, secret from subscriptions where id = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return shop.Subscription{}, shop.Errorf(shop.CodeNotFound, "no subscription %s", id)
	}
	return sub, err
}

// Subscriptions returns every webhook subscription, by id.
func (d *DB) Subscriptions() ([]shop.Subscription, error) {
	rows, err := d.db.Query(`select id, url, events, secret from subscriptions order by id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []shop.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// DeleteSubscription deletes a webhook subscription and its deliveries.
func (d *DB) DeleteSubscription(id shop.SubscriptionID) error {
	key, err := keyOf("wh-", string(id), "subscription")
	if err != nil {
		return err
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`delete from deliveries where subscription_id = ?`, key); err != nil {
		return err
	}
	res, err := tx.Exec(`delete from subscriptions where id = ?`, key)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shop.Errorf(shop.CodeNotFound, "no subscription %s", id)
	}
	return tx.Commit()
}

// AddEvent stores a new event and returns its id.
func (d *DB) AddEvent(e shop.Event) (shop.EventID, error) {
	res, err := d.db.Exec(`insert into events (type, subject) values (?, ?)`, string(e.Type), e.Subject)
	if err != nil {
		return "", err
	}
	key, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	return shop.EventID("ev-" + strconv.FormatInt(key, 10)), nil
}

// deliveryKeys returns the numbers of the subscription and the event of a delivery.
func deliveryKeys(dl shop.Delivery) (sub, event int64, err error) {
	if sub, err = keyOf("wh-", string(dl.Subscription), "subscription"); err != nil {
		return 0, 0, err
	}
	if event, err = keyOf("ev-", string(dl.Event.ID), "event"); err != nil {
		return 0, 0, err
	}
	return sub, event, nil
}

// AddDelivery stores a new delivery.
func (d *DB) AddDelivery(dl shop.Delivery) error {
	sub, event, err := deliveryKeys(dl)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`insert into deliveries (subscription_id, event_id, status, attempts, response) values (?, ?, ?, ?, ?)`,
		sub, event, string(dl.Status), dl.Attempts, dl.Response)
	return err
}

// SaveDelivery stores the status, the attempts, and the last response of a delivery.
func (d *DB) SaveDelivery(dl shop.Delivery) error {
	sub, event, err := deliveryKeys(dl)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`update deliveries set status = ?, attempts = ?, response = ? where subscription_id = ? and event_id = ?`,
		string(dl.Status), dl.Attempts, dl.Response, sub, event)
	return err
}

// Deliveries returns the deliveries of a subscription, in the order of their events.
func (d *DB) Deliveries(id shop.SubscriptionID) ([]shop.Delivery, error) {
	key, err := keyOf("wh-", string(id), "subscription")
	if err != nil {
		return nil, err
	}
	rows, err := d.db.Query(`select e.id, e.type, e.subject, d.status, d.attempts, d.response
		from deliveries d join events e on e.id = d.event_id where d.subscription_id = ? order by e.id`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []shop.Delivery
	for rows.Next() {
		var event int64
		var typ, subject, status string
		var attempts, response int
		if err := rows.Scan(&event, &typ, &subject, &status, &attempts, &response); err != nil {
			return nil, err
		}
		out = append(out, shop.Delivery{
			Subscription: id,
			Event:        shop.Event{ID: shop.EventID("ev-" + strconv.FormatInt(event, 10)), Type: shop.EventType(typ), Subject: subject},
			Status:       shop.DeliveryStatus(status), Attempts: attempts, Response: response,
		})
	}
	return out, rows.Err()
}
