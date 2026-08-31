package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"app/internal/models"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// schemaMigrationLockKey is an arbitrary, fixed Postgres advisory-lock key
// used to serialize applySchema across every process that might run it
// concurrently against the same database. Two cases hit this in practice:
// several backend replicas each calling InitDB() on startup in production,
// and `go test ./...` in CI, where each package (internal/services,
// internal/handlers, ...) is its own test binary and runs concurrently by
// default — testutil.OpenDB's sync.Once only dedupes InitDB calls *within*
// one package's binary, not across them. Without serializing, two processes
// can both see a table as "not yet created" and issue a concurrent
// CREATE TABLE, which Postgres doesn't fully serialize on its own: one loses
// a race on the implicit row type it registers in pg_type, surfacing as
// "duplicate key value violates unique constraint pg_type_typname_nsp_index"
// (SQLSTATE 23505) — not an app bug, just two DDL statements arriving at
// once. The numeric value itself is arbitrary; it only needs to be stable
// and unlikely to collide with an unrelated advisory lock in this database
// (nothing else in this codebase takes one).
const schemaMigrationLockKey = 726750310

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// appDSN is what the running app queries through. A managed provider (Neon and
// friends) hands out a single connection URL rather than a set of discrete
// fields, so DATABASE_URL wins whenever it's set; the DB_HOST/DB_PORT/... path
// below stays the default for the local docker-compose Postgres, which has no
// TLS and therefore keeps sslmode=disable.
func appDSN() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "db"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "calciotto"),
		os.Getenv("DB_PASSWORD"),
		getEnv("DB_NAME", "calciotto"),
	)
}

// migrate applies the schema. When a separate direct URL is configured it opens
// its own short-lived connection for the DDL and closes it again, so exactly one
// pool — the app's — outlives startup; otherwise it reuses the pool it's given.
// Either way, applySchema itself only runs while withSchemaMigrationLock holds
// the advisory lock — see schemaMigrationLockKey for why.
func migrate(appDB *gorm.DB) error {
	unpooled := os.Getenv("DATABASE_URL_UNPOOLED")
	if unpooled == "" || unpooled == appDSN() {
		return withSchemaMigrationLock(appDB, applySchema)
	}

	db, err := gorm.Open(postgres.Open(unpooled), &gorm.Config{})
	if err != nil {
		return err
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	return withSchemaMigrationLock(db, applySchema)
}

// withSchemaMigrationLock pins a single physical connection out of db's pool,
// holds schemaMigrationLockKey on it for the duration of fn, and releases it
// afterwards. A session-level advisory lock is tied to whichever connection
// took it, so fn must keep running on that same connection throughout —
// which is why fn receives a *gorm.DB backed by the pinned *sql.Conn
// (gorm.io/driver/postgres.Config.Conn accepts one directly) rather than the
// pool-backed db passed in, which could hand fn's own queries a different
// physical connection than the one holding the lock. A concurrent caller
// elsewhere — another process, another `go test` package — blocks inside
// pg_advisory_lock until this one releases, so schema application across
// every process sharing this database is fully serialized rather than merely
// made less likely to race.
func withSchemaMigrationLock(db *gorm.DB, fn func(*gorm.DB) error) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", schemaMigrationLockKey); err != nil {
		return err
	}
	defer func() {
		if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", schemaMigrationLockKey); err != nil {
			log.Printf("warning: failed to release schema migration advisory lock: %v", err)
		}
	}()

	connDB, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{})
	if err != nil {
		return err
	}
	return fn(connDB)
}

func applySchema(db *gorm.DB) error {
	// Auto-migration des modèles pour créer les tables
	// Match's four scheduling columns (added with the sign-up feature) need no
	// backfill of their own: they are all nullable, and NULL is exactly the
	// "this match was never scheduled" state every pre-existing match is in —
	// unlike GroupMembership.IsFavorite below, whose non-null default(false)
	// broke an invariant and had to be repaired.
	if err := db.AutoMigrate(&models.Group{}, &models.Player{}, &models.Team{}, &models.Match{}, &models.MatchPlayer{}, &models.MatchRegistration{}, &models.GroupMembership{}, &models.PasswordResetToken{}); err != nil {
		return err
	}

	log.Println("Tables created successfully!")

	// Player.Name used to carry a uniqueIndex tag (see PlayerService.CreatePlayer's
	// duplicate check). AuthService.SignupNewPlayer deliberately allows two
	// accounts to share a display name, so the tag was dropped from the model —
	// but AutoMigrate only ever adds schema, it never removes an index that's no
	// longer declared. A database created before this change keeps the old
	// idx_players_name constraint forever unless it's dropped explicitly here.
	if err := db.Exec(`DROP INDEX IF EXISTS idx_players_name`).Error; err != nil {
		return err
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
		return err
	}

	// GroupMembership.IsFavorite is new: AutoMigrate adds the column with its
	// declared default (false) for every existing row, leaving every
	// pre-existing player with zero favorites — breaking the "always exactly
	// one, as long as you belong to at least one group" invariant the rest of
	// GroupMembershipService relies on (AddPlayerToGroupWithRole only grants it
	// to a brand-new player's very first membership, which doesn't help anyone
	// who already had memberships before this column existed). Backfill: for
	// every player with no favorite yet, promote their oldest membership
	// (DISTINCT ON ... ORDER BY player_id, created_at picks exactly that row
	// per player). Idempotent — after the first run every player already has a
	// favorite, so the NOT EXISTS guard makes this a no-op on every later start.
	return db.Exec(`
		UPDATE group_memberships gm
		SET is_favorite = true
		FROM (
			SELECT DISTINCT ON (player_id) id
			FROM group_memberships
			ORDER BY player_id, created_at ASC
		) AS oldest
		WHERE gm.id = oldest.id
		AND NOT EXISTS (
			SELECT 1 FROM group_memberships gm2
			WHERE gm2.player_id = gm.player_id AND gm2.is_favorite = true
		)
	`).Error
}

func InitDB() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(appDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to the database!")

	if err := migrate(db); err != nil {
		log.Fatal(err)
	}

	return db, nil
}
