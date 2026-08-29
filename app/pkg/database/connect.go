package database

import (
	"fmt"
	"log"
	"os"

	"app/internal/models"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func InitDB() (*gorm.DB, error) {
	host := getEnv("DB_HOST", "db")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "calciotto")
	password := os.Getenv("DB_PASSWORD")
	dbname := getEnv("DB_NAME", "calciotto")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database!")

	// Auto-migration des modèles pour créer les tables
	err = db.AutoMigrate(&models.Group{}, &models.Player{}, &models.Team{}, &models.Match{}, &models.MatchPlayer{}, &models.GroupMembership{}, &models.PasswordResetToken{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Tables created successfully!")

	// Player.Name used to carry a uniqueIndex tag (see PlayerService.CreatePlayer's
	// duplicate check). AuthService.SignupNewPlayer deliberately allows two
	// accounts to share a display name, so the tag was dropped from the model —
	// but AutoMigrate only ever adds schema, it never removes an index that's no
	// longer declared. A database created before this change keeps the old
	// idx_players_name constraint forever unless it's dropped explicitly here.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_players_name`).Error; err != nil {
		log.Fatal(err)
	}

	// GroupMembership.Role used to have "owner" as its privileged value, with
	// exactly one owner per group. The model now uses "admin" (models.RoleAdmin)
	// and allows several per group — but renaming the Go constant does nothing
	// to rows already stored: AutoMigrate only ever touches schema, never data.
	// Without this rewrite every pre-existing group would keep role = 'owner',
	// which no longer matches models.RoleAdmin, so its creator would be locked
	// out of every admin-gated action (removing a member, changing a role,
	// creating a match, editing scores) with no admin left able to grant it
	// back. Idempotent: after the first run no 'owner' row is left to update.
	if err := db.Exec(`UPDATE group_memberships SET role = 'admin' WHERE role = 'owner'`).Error; err != nil {
		log.Fatal(err)
	}

	return db, nil
}
