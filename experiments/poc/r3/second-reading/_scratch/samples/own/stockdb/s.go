package stockdb

import "database/sql"

const q = "UPDATE orders SET status = 'x'"

// Bad writes a table of another part, in a constant and in a built string.
func Bad(db *sql.DB, name string) {
	_, _ = db.Exec(q)
	_, _ = db.Exec("UPDATE " + name + " SET a = 1")
	_, _ = db.Exec("INSERT INTO stock VALUES (1)")
}
