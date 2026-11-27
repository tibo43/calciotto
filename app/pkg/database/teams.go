package database

import (
	"app/internal/models"

	"gorm.io/gorm"
)

// ReplaceTeamColourWithPosition drops teams.colour, which nothing displays
// any more (design v2 shows no team colour), and keeps the one thing it
// silently carried: the order of a group's two teams. Matches listed their
// teams sorted by colour, so a group's left and right team were whichever
// colour sorted first. teams.position records that same order (0, then 1)
// before the colour goes, so no match swaps its two sides after the deploy.
//
// It runs before AutoMigrate, and only while the colour column still exists:
// a fresh database never has it, and after the first start there is nothing
// left to do. AutoMigrate never drops a column, hence the explicit DROP.
func ReplaceTeamColourWithPosition(db *gorm.DB) error {
	migrator := db.Migrator()
	if !migrator.HasTable(&models.Team{}) || !migrator.HasColumn(&models.Team{}, "colour") {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE teams ADD COLUMN IF NOT EXISTS position integer NOT NULL DEFAULT 0`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			UPDATE teams SET position = ranked.position
			FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY group_id ORDER BY colour, id) - 1 AS position
				FROM teams
			) AS ranked
			WHERE teams.id = ranked.id
		`).Error; err != nil {
			return err
		}
		return tx.Exec(`ALTER TABLE teams DROP COLUMN colour`).Error
	})
}
