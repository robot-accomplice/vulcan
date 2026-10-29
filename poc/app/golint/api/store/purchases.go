package store

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"stockroom/lib/contract"
	"stockroom/lib/shop"
)

// keyOf returns the number of an id such as s-1, or not_found for an id of another form.
func keyOf(prefix, id, what string) (int64, error) {
	n, err := strconv.ParseInt(strings.TrimPrefix(id, prefix), 10, 64)
	if err != nil || !strings.HasPrefix(id, prefix) {
		return 0, shop.Errorf(shop.CodeNotFound, "no %s %s", what, id)
	}
	return n, nil
}

// AddSupplier stores a new supplier and returns its id; a supplier with the same name is a
// conflict.
func (d *DB) AddSupplier(name string) (shop.SupplierID, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRow(`select count(*) from suppliers where name = ?`, name).Scan(&n); err != nil {
		return "", err
	}
	if n > 0 {
		return "", shop.Errorf(shop.CodeConflict, "supplier %q exists", name)
	}
	res, err := tx.Exec(`insert into suppliers (name) values (?)`, name)
	if err != nil {
		return "", err
	}
	key, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return shop.SupplierID("s-" + strconv.FormatInt(key, 10)), nil
}

// Supplier returns a supplier.
func (d *DB) Supplier(id shop.SupplierID) (shop.Supplier, error) {
	key, err := keyOf("s-", string(id), "supplier")
	if err != nil {
		return shop.Supplier{}, err
	}
	var name string
	err = d.db.QueryRow(`select name from suppliers where id = ?`, key).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return shop.Supplier{}, shop.Errorf(shop.CodeNotFound, "no supplier %s", id)
	}
	if err != nil {
		return shop.Supplier{}, err
	}
	return shop.Supplier{ID: id, Name: name}, nil
}

// Suppliers returns every supplier, by id.
func (d *DB) Suppliers() ([]shop.Supplier, error) {
	rows, err := d.db.Query(`select id, name from suppliers order by id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []shop.Supplier
	for rows.Next() {
		var key int64
		var name string
		if err := rows.Scan(&key, &name); err != nil {
			return nil, err
		}
		out = append(out, shop.Supplier{ID: shop.SupplierID("s-" + strconv.FormatInt(key, 10)), Name: name})
	}
	return out, rows.Err()
}

// CreatePurchase stores a new purchase order and returns its id.
func (d *DB) CreatePurchase(po shop.PurchaseOrder) (shop.PurchaseID, error) {
	supplier, err := keyOf("s-", string(po.Supplier), "supplier")
	if err != nil {
		return "", err
	}
	tx, err := d.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`insert into purchase_orders (supplier_id, status, total) values (?, ?, ?)`, supplier, string(po.Status), int64(po.Total))
	if err != nil {
		return "", err
	}
	key, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	for i, l := range po.Lines {
		if _, err := tx.Exec(`insert into purchase_lines (purchase_id, pos, sku, qty, received, cost) values (?, ?, ?, ?, ?, ?)`,
			key, i, string(l.SKU), l.Qty, l.Received, int64(l.Cost)); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return shop.PurchaseID("po-" + strconv.FormatInt(key, 10)), nil
}

// Purchase returns a purchase order.
func (d *DB) Purchase(id shop.PurchaseID) (shop.PurchaseOrder, error) {
	key, err := keyOf("po-", string(id), "purchase order")
	if err != nil {
		return shop.PurchaseOrder{}, err
	}
	var supplier, total int64
	var status string
	err = d.db.QueryRow(`select supplier_id, status, total from purchase_orders where id = ?`, key).Scan(&supplier, &status, &total)
	if errors.Is(err, sql.ErrNoRows) {
		return shop.PurchaseOrder{}, shop.Errorf(shop.CodeNotFound, "no purchase order %s", id)
	}
	if err != nil {
		return shop.PurchaseOrder{}, err
	}
	po := shop.PurchaseOrder{
		ID: id, Supplier: shop.SupplierID("s-" + strconv.FormatInt(supplier, 10)), Status: shop.PurchaseStatus(status), Total: shop.Money(total),
	}
	rows, err := d.db.Query(`select sku, qty, received, cost from purchase_lines where purchase_id = ? order by pos`, key)
	if err != nil {
		return shop.PurchaseOrder{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var sku string
		var qty, received int
		var cost int64
		if err := rows.Scan(&sku, &qty, &received, &cost); err != nil {
			return shop.PurchaseOrder{}, err
		}
		po.Lines = append(po.Lines, contract.Valid(shop.PurchaseLine{SKU: shop.SKU(sku), Qty: qty, Received: received, Cost: shop.Money(cost)}))
	}
	return po, rows.Err()
}

// PurchasesOf returns the purchase orders of a supplier, by id.
func (d *DB) PurchasesOf(s shop.SupplierID) ([]shop.PurchaseOrder, error) {
	supplier, err := keyOf("s-", string(s), "supplier")
	if err != nil {
		return nil, err
	}
	rows, err := d.db.Query(`select id from purchase_orders where supplier_id = ? order by id`, supplier)
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
	var out []shop.PurchaseOrder
	for _, k := range keys {
		po, err := d.Purchase(shop.PurchaseID("po-" + strconv.FormatInt(k, 10)))
		if err != nil {
			return nil, err
		}
		out = append(out, po)
	}
	return out, nil
}

// SavePurchase stores the status of a purchase order and the units received on each line.
func (d *DB) SavePurchase(po shop.PurchaseOrder) error {
	key, err := keyOf("po-", string(po.ID), "purchase order")
	if err != nil {
		return err
	}
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`update purchase_orders set status = ? where id = ?`, string(po.Status), key); err != nil {
		return err
	}
	for i, l := range po.Lines {
		if _, err := tx.Exec(`update purchase_lines set received = ? where purchase_id = ? and pos = ?`, l.Received, key, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}
