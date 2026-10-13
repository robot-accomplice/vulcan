package audit_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"shop/audit"
	"shop/shop"
)

// testDSNEnv names a scratch Postgres database for this test. The test writes to
// the billing_audit table there, so never point it at a shared database.
const testDSNEnv = "SHOP_TEST_DATABASE_URL"

// The Postgres store returns exactly the lines written for one order, oldest
// first, with every field intact. It needs a real database, so it skips
// without one.
func TestPostgresRoundTrip(t *testing.T) {
	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("set %s to run against Postgres", testDSNEnv)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	p := audit.NewPostgres(db)
	if err := p.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	// Run twice: Migrate must be safe to repeat at every start.
	if err := p.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	order := shop.OrderID("test-" + time.Now().Format(time.RFC3339Nano))
	at := time.Now().UTC().Truncate(time.Microsecond) // Postgres keeps microseconds
	want := []audit.Entry{
		{Order: order, Customer: "c1", Kind: audit.Charge, Amount: 1000, At: at},
		{Order: order, Customer: "c1", Kind: audit.Refund, Amount: 400, At: at.Add(time.Second)},
	}
	for _, e := range want {
		if err := p.Record(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Record(ctx, audit.Entry{Order: order + "-other", Customer: "c2", Kind: audit.Charge, Amount: 1, At: at}); err != nil {
		t.Fatal(err)
	}

	got, err := p.ForOrder(ctx, order)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Order != want[i].Order || got[i].Customer != want[i].Customer ||
			got[i].Kind != want[i].Kind || got[i].Amount != want[i].Amount || !got[i].At.Equal(want[i].At) {
			t.Fatalf("line %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}
