package database

import (
	"context"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool returns a pgxpool.Pool connected to the test database.
// It skips the test if DATABASE_URL is not set.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// testIDCounter provides unique IDs across all tests in the package.
// Starts well above zero to avoid conflicts with any existing data.
var testIDCounter int64 = 100000

func nextID() int64 {
	return atomic.AddInt64(&testIDCounter, 1)
}
