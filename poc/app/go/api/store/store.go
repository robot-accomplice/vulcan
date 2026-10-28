// Package store keeps the catalog, the stock, the orders, the ledger, the purchase orders, and the
// webhooks in SQLite.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"stockroom/lib/shop"
)

// Settings name the database file.
type Settings struct {
	Path string
}

// DB is the database of Stockroom.
type DB struct {
	db *sql.DB
}

const schema = `
create table if not exists products (
	sku   text primary key,
	name  text not null,
	price integer not null
);
create table if not exists stock (
	sku      text primary key references products(sku),
	on_hand  integer not null,
	reserved integer not null
);
create table if not exists orders (
	id         integer primary key autoincrement,
	customer   text not null,
	status     text not null,
	total      integer not null,
	charged    integer not null,
	refunded   integer not null,
	payment_id text not null
);
create table if not exists order_lines (
	order_id integer not null references orders(id),
	pos      integer not null,
	sku      text not null,
	qty      integer not null,
	price    integer not null,
	primary key (order_id, pos)
);
create table if not exists payments (
	id          integer primary key autoincrement,
	order_id    integer not null references orders(id),
	kind        text not null,
	amount      integer not null,
	provider_id text not null
);
create table if not exists suppliers (
	id   integer primary key autoincrement,
	name text not null unique
);
create table if not exists purchase_orders (
	id          integer primary key autoincrement,
	supplier_id integer not null references suppliers(id),
	status      text not null,
	total       integer not null
);
create table if not exists purchase_lines (
	purchase_id integer not null references purchase_orders(id),
	pos         integer not null,
	sku         text not null,
	qty         integer not null,
	received    integer not null,
	cost        integer not null,
	primary key (purchase_id, pos)
);
create table if not exists subscriptions (
	id     integer primary key autoincrement,
	url    text not null,
	events text not null,
	secret text not null
);
create table if not exists events (
	id      integer primary key autoincrement,
	type    text not null,
	subject text not null
);
create table if not exists deliveries (
	id              integer primary key autoincrement,
	subscription_id integer not null references subscriptions(id),
	event_id        integer not null references events(id),
	status          text not null,
	attempts        integer not null,
	response        integer not null,
	unique (subscription_id, event_id)
);
`

// Open opens the database file, and creates its tables when they do not exist.
func Open(s Settings) (*DB, error) {
	db, err := sql.Open("sqlite", s.Path)
	if err != nil {
		return nil, err
	}
	// One connection: SQLite allows one writer, and the transactions below rely on it.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create the tables: %w", err)
	}
	return &DB{db: db}, nil
}

// AddProduct adds a product with no stock.
func (d *DB) AddProduct(p shop.Product) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRow(`select count(*) from products where sku = ?`, string(p.SKU)).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return shop.Errorf(shop.CodeConflict, "product %s exists", p.SKU)
	}
	if _, err := tx.Exec(`insert into products (sku, name, price) values (?, ?, ?)`, string(p.SKU), p.Name, int64(p.Price)); err != nil {
		return err
	}
	if _, err := tx.Exec(`insert into stock (sku, on_hand, reserved) values (?, 0, 0)`, string(p.SKU)); err != nil {
		return err
	}
	return tx.Commit()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanItem(r scanner) (shop.Item, error) {
	var sku, name string
	var price int64
	var onHand, reserved int
	if err := r.Scan(&sku, &name, &price, &onHand, &reserved); err != nil {
		return shop.Item{}, err
	}
	return shop.Item{
		Product: shop.Product{SKU: shop.SKU(sku), Name: name, Price: shop.Money(price)},
		Stock:   shop.Stock{OnHand: onHand, Reserved: reserved},
	}, nil
}

const itemQuery = `select p.sku, p.name, p.price, s.on_hand, s.reserved from products p join stock s on s.sku = p.sku`

// Product returns a product with its stock.
func (d *DB) Product(sku shop.SKU) (shop.Item, error) {
	it, err := scanItem(d.db.QueryRow(itemQuery+` where p.sku = ?`, string(sku)))
	if errors.Is(err, sql.ErrNoRows) {
		return shop.Item{}, shop.Errorf(shop.CodeNotFound, "no product %s", sku)
	}
	return it, err
}

// Products returns every product with its stock, by SKU.
func (d *DB) Products() ([]shop.Item, error) {
	rows, err := d.db.Query(itemQuery + ` order by p.sku`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []shop.Item
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Receive adds qty units to the stock of a product.
func (d *DB) Receive(sku shop.SKU, qty int) (shop.Item, error) {
	res, err := d.db.Exec(`update stock set on_hand = on_hand + ? where sku = ?`, qty, string(sku))
	if err != nil {
		return shop.Item{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return shop.Item{}, shop.Errorf(shop.CodeNotFound, "no product %s", sku)
	}
	return d.Product(sku)
}

// Reserve reserves the units of every line, or of none: it fails with out_of_stock when a line
// asks for more than is available.
func (d *DB) Reserve(lines []shop.Line) error {
	return d.changeStock(lines, func(st shop.Stock, qty int) (shop.Stock, error) {
		if st.Available() < qty {
			return st, shop.Errorf(shop.CodeOutOfStock, "only %d available", st.Available())
		}
		return shop.Stock{OnHand: st.OnHand, Reserved: st.Reserved + qty}, nil
	})
}

// Release releases the reservation of every line.
func (d *DB) Release(lines []shop.Line) error {
	return d.changeStock(lines, func(st shop.Stock, qty int) (shop.Stock, error) {
		return shop.Stock{OnHand: st.OnHand, Reserved: st.Reserved - qty}, nil
	})
}

// Ship takes the reserved units of every line out of stock.
func (d *DB) Ship(lines []shop.Line) error {
	return d.changeStock(lines, func(st shop.Stock, qty int) (shop.Stock, error) {
		return shop.Stock{OnHand: st.OnHand - qty, Reserved: st.Reserved - qty}, nil
	})
}

// Restock adds the units of every arrival to the stock, or of none.
func (d *DB) Restock(arrivals []shop.Arrival) error {
	lines := make([]shop.Line, len(arrivals))
	for i, a := range arrivals {
		lines[i] = shop.Line{SKU: a.SKU, Qty: a.Qty}
	}
	return d.changeStock(lines, func(st shop.Stock, qty int) (shop.Stock, error) {
		return shop.Stock{OnHand: st.OnHand + qty, Reserved: st.Reserved}, nil
	})
}

// changeStock applies change to the stock of each line in one transaction.
func (d *DB) changeStock(lines []shop.Line, change func(shop.Stock, int) (shop.Stock, error)) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, l := range lines {
		var onHand, reserved int
		err := tx.QueryRow(`select on_hand, reserved from stock where sku = ?`, string(l.SKU)).Scan(&onHand, &reserved)
		if errors.Is(err, sql.ErrNoRows) {
			return shop.Errorf(shop.CodeNotFound, "no product %s", l.SKU)
		}
		if err != nil {
			return err
		}
		next, err := change(shop.Stock{OnHand: onHand, Reserved: reserved}, l.Qty)
		if err != nil {
			return fmt.Errorf("%s: %w", l.SKU, err)
		}
		if _, err := tx.Exec(`update stock set on_hand = ?, reserved = ? where sku = ?`, next.OnHand, next.Reserved, string(l.SKU)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func orderKey(id shop.OrderID) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimPrefix(string(id), "o-"), 10, 64)
	if err != nil || !strings.HasPrefix(string(id), "o-") {
		return 0, shop.Errorf(shop.CodeNotFound, "no order %s", id)
	}
	return n, nil
}

// CreateOrder stores a new order and returns its id.
func (d *DB) CreateOrder(o shop.Order) (shop.OrderID, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`insert into orders (customer, status, total, charged, refunded, payment_id) values (?, ?, ?, ?, ?, ?)`,
		string(o.Customer), string(o.Status), int64(o.Total), int64(o.Charged), int64(o.Refunded), o.PaymentID)
	if err != nil {
		return "", err
	}
	key, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	for i, l := range o.Lines {
		if _, err := tx.Exec(`insert into order_lines (order_id, pos, sku, qty, price) values (?, ?, ?, ?, ?)`,
			key, i, string(l.SKU), l.Qty, int64(l.Price)); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return shop.OrderID("o-" + strconv.FormatInt(key, 10)), nil
}

// Order returns an order.
func (d *DB) Order(id shop.OrderID) (shop.Order, error) {
	key, err := orderKey(id)
	if err != nil {
		return shop.Order{}, err
	}
	o, err := d.scanOrder(d.db.QueryRow(`select id, customer, status, total, charged, refunded, payment_id from orders where id = ?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return shop.Order{}, shop.Errorf(shop.CodeNotFound, "no order %s", id)
	}
	return o, err
}

// OrdersOf returns the orders of a customer, by id.
func (d *DB) OrdersOf(c shop.CustomerID) ([]shop.Order, error) {
	rows, err := d.db.Query(`select id from orders where customer = ? order by id`, string(c))
	if err != nil {
		return nil, err
	}
	var keys []int64
	for rows.Next() {
		var k int64
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return nil, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var out []shop.Order
	for _, k := range keys {
		o, err := d.Order(shop.OrderID("o-" + strconv.FormatInt(k, 10)))
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func (d *DB) scanOrder(r scanner) (shop.Order, error) {
	var key, total, charged, refunded int64
	var customer, status, paymentID string
	if err := r.Scan(&key, &customer, &status, &total, &charged, &refunded, &paymentID); err != nil {
		return shop.Order{}, err
	}
	o := shop.Order{
		ID: shop.OrderID("o-" + strconv.FormatInt(key, 10)), Customer: shop.CustomerID(customer), Status: shop.Status(status),
		Total: shop.Money(total), Charged: shop.Money(charged), Refunded: shop.Money(refunded), PaymentID: paymentID,
	}
	rows, err := d.db.Query(`select sku, qty, price from order_lines where order_id = ? order by pos`, key)
	if err != nil {
		return shop.Order{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var sku string
		var qty int
		var price int64
		if err := rows.Scan(&sku, &qty, &price); err != nil {
			return shop.Order{}, err
		}
		o.Lines = append(o.Lines, shop.Line{SKU: shop.SKU(sku), Qty: qty, Price: shop.Money(price)})
	}
	return o, rows.Err()
}

// SaveOrder stores the status, the sums, and the payment id of an order.
func (d *DB) SaveOrder(o shop.Order) error {
	key, err := orderKey(o.ID)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`update orders set status = ?, charged = ?, refunded = ?, payment_id = ? where id = ?`,
		string(o.Status), int64(o.Charged), int64(o.Refunded), o.PaymentID, key)
	return err
}

// AddPayment adds an entry to the ledger of an order.
func (d *DB) AddPayment(id shop.OrderID, p shop.Payment) error {
	key, err := orderKey(id)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`insert into payments (order_id, kind, amount, provider_id) values (?, ?, ?, ?)`, key, p.Kind, int64(p.Amount), p.ProviderID)
	return err
}

// Payments returns the ledger of an order, in order.
func (d *DB) Payments(id shop.OrderID) ([]shop.Payment, error) {
	key, err := orderKey(id)
	if err != nil {
		return nil, err
	}
	rows, err := d.db.Query(`select kind, amount, provider_id from payments where order_id = ? order by id`, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []shop.Payment
	for rows.Next() {
		var p shop.Payment
		var amount int64
		if err := rows.Scan(&p.Kind, &amount, &p.ProviderID); err != nil {
			return nil, err
		}
		p.Amount = shop.Money(amount)
		out = append(out, p)
	}
	return out, rows.Err()
}
