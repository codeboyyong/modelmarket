package httpapi

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPostgresConcurrentReservationsCannotOverspend(t *testing.T) {
	databaseURL := os.Getenv("MM_INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MM_INTEGRATION_DATABASE_URL is not set")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err = db.ExecContext(ctx, `update user_wallets set paid_credits = 10000, promotional_credits = 5000 where id = 'wallet-demo'`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, `delete from user_ledger_transactions where idempotency_key like 'reserve_%_integration_%' or idempotency_key like 'release_%_integration_%'`)
		_, _ = db.ExecContext(ctx, `update user_wallets set paid_credits = 10000, promotional_credits = 5000 where id = 'wallet-demo'`)
	}()
	app := &App{DB: db}
	type result struct {
		id          string
		reservation creditReservation
		err         error
	}
	results := make(chan result, 2)
	var start sync.WaitGroup
	start.Add(1)
	for _, id := range []string{"integration_a", "integration_b"} {
		go func(requestID string) {
			start.Wait()
			reservation, reserveErr := app.reserveProjectCredits(ctx, "project-demo", requestID, 10000)
			results <- result{id: requestID, reservation: reservation, err: reserveErr}
		}(id)
	}
	start.Done()
	first, second := <-results, <-results
	succeeded, failed := first, second
	if succeeded.err != nil {
		succeeded, failed = failed, succeeded
	}
	if succeeded.err != nil {
		t.Fatalf("both reservations failed: %v / %v", first.err, second.err)
	}
	if failed.err == nil || failed.err.Error() != "insufficient_credits" {
		t.Fatalf("second reservation should fail, got %v", failed.err)
	}
	if err := app.releaseProjectCredits(ctx, succeeded.id, succeeded.reservation); err != nil {
		t.Fatal(err)
	}
	var paid, promotional int64
	if err := db.QueryRowContext(ctx, `select paid_credits, promotional_credits from user_wallets where id = 'wallet-demo'`).Scan(&paid, &promotional); err != nil {
		t.Fatal(err)
	}
	if paid != 10000 || promotional != 5000 {
		t.Fatalf("wallet not restored: paid=%d promotional=%d", paid, promotional)
	}
}
