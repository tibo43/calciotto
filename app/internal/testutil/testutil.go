// Package testutil provides shared helpers for integration tests that need a
// real database connection. It is not a _test.go file itself so that it can
// be imported from _test.go files across multiple packages.
package testutil

import (
	"sync"
	"testing"

	"app/pkg/database"

	"gorm.io/gorm"
)

var (
	once     sync.Once
	sharedDB *gorm.DB
	initErr  error
)

// OpenDB connects to the database configured via the standard DB_* env vars
// (see pkg/database.InitDB) and skips the calling test if it isn't reachable,
// so `go test ./...` never fails just because no Postgres is running locally.
// The connection is shared across the whole test binary run.
func OpenDB(t *testing.T) *gorm.DB {
	t.Helper()
	once.Do(func() {
		sharedDB, initErr = database.InitDB()
	})
	if initErr != nil {
		t.Skipf("skipping integration test: database not reachable: %v", initErr)
	}
	return sharedDB
}

// BeginTx starts a transaction on db and registers a rollback via t.Cleanup,
// so anything the test creates is undone regardless of pass/fail — no
// leftover rows in the real database, no manual truncation between tests.
func BeginTx(t *testing.T, db *gorm.DB) *gorm.DB {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin test transaction: %v", tx.Error)
	}
	t.Cleanup(func() {
		if err := tx.Rollback().Error; err != nil {
			t.Logf("warning: failed to roll back test transaction: %v", err)
		}
	})
	return tx
}
