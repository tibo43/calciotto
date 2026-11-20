package database

import (
	"app/internal/models"

	"gorm.io/gorm"
)

// AddMotmNotifiedAt adds matches.motm_notified_at to an existing matches
// table and backfills it in the same transaction, so the column can never be
// there without its backfill, even if the process dies halfway. A database
// that already has the column, or has no matches table yet (a fresh one,
// which AutoMigrate creates with the column), is left to AutoMigrate.
func AddMotmNotifiedAt(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasTable(&models.Match{}) || migrator.HasColumn(&models.Match{}, "MotmNotifiedAt") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE matches ADD COLUMN IF NOT EXISTS motm_notified_at timestamptz`).Error; err != nil {
			return err
		}
		return BackfillMotmNotifiedAt(tx)
	})
}

// BackfillMotmNotifiedAt marks as announced every match whose Man of the
// Match announcement time (services.MotmAnnounceAt: 09:00 Europe/Rome on
// D+2, the day the vote closes) has already passed, with that time as its
// value. Without it, the first tick after the deploy would announce every
// vote closed in the last MotmAnnounceWindow (the tick's own bound, which
// already leaves older ones alone). A match whose vote is still open, or
// closed today before 09:00 in Rome, keeps NULL and is announced at its
// time like any later match. It runs only with the statement that adds the
// column (AddMotmNotifiedAt), never on a later start: there, a NULL on a
// closed vote is an announcement still waiting for its tick.
func BackfillMotmNotifiedAt(db *gorm.DB) error {
	return db.Exec(`
		UPDATE matches
		SET motm_notified_at = ((date + 2)::timestamp + interval '9 hours') AT TIME ZONE 'Europe/Rome'
		WHERE motm_notified_at IS NULL
		  AND ((date + 2)::timestamp + interval '9 hours') AT TIME ZONE 'Europe/Rome' <= now()
	`).Error
}
