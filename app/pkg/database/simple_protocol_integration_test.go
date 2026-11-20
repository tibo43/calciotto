package database_test

import (
	"testing"

	"app/internal/testutil"

	"gorm.io/gorm"
)

// The app's pool must not prepare statements on the server: behind Neon's
// pooler, a SELECT * prepared before a migration adds a column fails with
// "cached plan must not change result type" (SQLSTATE 0A000) until the
// pooler's connections are recycled. Run the same SELECT * on one connection
// before and after adding a column.
func TestAppPool_SelectStarSurvivesAnAddedColumn(t *testing.T) {
	db := testutil.OpenDB(t)
	err := db.Connection(func(conn *gorm.DB) error {
		if err := conn.Exec(`CREATE TEMP TABLE simple_protocol_probe (id int)`).Error; err != nil {
			return err
		}
		if err := conn.Exec(`INSERT INTO simple_protocol_probe VALUES (1)`).Error; err != nil {
			return err
		}
		for i := 0; i < 2; i++ {
			var rows []map[string]any
			if err := conn.Raw(`SELECT * FROM simple_protocol_probe WHERE id = ?`, 1).Scan(&rows).Error; err != nil {
				return err
			}
		}
		if err := conn.Exec(`ALTER TABLE simple_protocol_probe ADD COLUMN extra int`).Error; err != nil {
			return err
		}
		var rows []map[string]any
		if err := conn.Raw(`SELECT * FROM simple_protocol_probe WHERE id = ?`, 1).Scan(&rows).Error; err != nil {
			return err
		}
		if len(rows) != 1 || len(rows[0]) != 2 {
			t.Errorf("rows = %v, want one row with both columns", rows)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("SELECT * after ADD COLUMN: %v", err)
	}
}
