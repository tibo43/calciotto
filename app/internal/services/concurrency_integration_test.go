package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

func TestConcurrentPasswordResetConsumesTokenOnce(t *testing.T) {
	db := testutil.OpenDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	email := uuid.NewString() + "@example.test"
	playerID := newClaimedPlayer(t, db, "Concurrent reset fixture "+uuid.NewString(), email, "fixture-password")
	t.Cleanup(func() {
		cleanup := db.Where("player_id = ?", playerID).Delete(&models.PasswordResetToken{})
		if cleanup.Error != nil {
			t.Error(cleanup.Error)
		}
		if err := db.Delete(&models.Player{}, "id = ?", playerID).Error; err != nil {
			t.Error(err)
		}
	})
	auth := services.NewAuthService(db.WithContext(ctx), testJWTSecret)
	token := requestResetToken(t, auth, email)
	results := runTogether(
		func() error { return auth.ResetPassword(token, "first-new-password") },
		func() error { return auth.ResetPassword(token, "second-new-password") },
	)
	successes := 0
	for _, err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, services.ErrInvalidResetToken) {
			t.Fatalf("unexpected reset result: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful resets = %d, want one", successes)
	}
	var player models.Player
	if err := db.Take(&player, "id = ?", playerID).Error; err != nil || player.TokenVersion != 1 {
		t.Fatalf("password changed %d times, lookup error %v", player.TokenVersion, err)
	}
}

func TestConcurrentDeparturesKeepAnAdministrator(t *testing.T) {
	db := testutil.OpenDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	group, err := services.NewGroupService(db).CreateGroup("Concurrent group fixture", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatal(err)
	}
	playerIDs := make([]uuid.UUID, 0, 3)
	t.Cleanup(func() {
		for _, model := range []any{&models.GroupMembership{}, &models.Team{}} {
			if err := db.Where("group_id = ?", group.ID).Delete(model).Error; err != nil {
				t.Error(err)
			}
		}
		if err := db.Delete(&models.Group{}, "id = ?", group.ID).Error; err != nil {
			t.Error(err)
		}
		if len(playerIDs) > 0 {
			if err := db.Where("id IN ?", playerIDs).Delete(&models.Player{}).Error; err != nil {
				t.Error(err)
			}
		}
	})
	memberships := services.NewGroupMembershipService(db.WithContext(ctx))
	for _, role := range []string{models.RoleAdmin, models.RoleAdmin, models.RoleMember} {
		playerID, err := services.NewPlayerService(db).CreatePlayer("Concurrent member " + uuid.NewString())
		if err != nil {
			t.Fatal(err)
		}
		playerIDs = append(playerIDs, playerID)
		if err := memberships.AddPlayerToGroupWithRole(group.ID, playerID, role); err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range runTogether(
		func() error { return memberships.LeaveGroup(group.ID, playerIDs[0]) },
		func() error { return memberships.LeaveGroup(group.ID, playerIDs[1]) },
	) {
		if err != nil {
			t.Fatal(err)
		}
	}
	role, err := memberships.GetRole(group.ID, playerIDs[2])
	if err != nil || role != models.RoleAdmin {
		t.Fatalf("remaining member role = %q, lookup error %v", role, err)
	}
}

func runTogether(first, second func() error) [2]error {
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, run := range []func() error{first, second} {
		go func() { <-start; results <- run() }()
	}
	close(start)
	return [2]error{<-results, <-results}
}

// Test the cursor against the full read so equal-date matches stay on
// distinct pages while the API keeps their original representation.
func TestMatchPages_Integration_StableForEqualDates(t *testing.T) {
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	group, err := services.NewGroupService(tx).CreateGroup("Paged matches fixture", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatal(err)
	}
	service := services.NewMatchService(tx)
	date := models.DateOf(time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC))
	for range 3 {
		if _, err := service.CreateMatch(services.MatchSpec{Date: date}, group.ID); err != nil {
			t.Fatal(err)
		}
	}
	first, cursor, err := service.GetMatchesDetailsPage(group.ID, "2026-2027", "", 2)
	if err != nil || len(first) != 2 || cursor == "" {
		t.Fatalf("first page has %d matches, cursor=%t, error=%v", len(first), cursor != "", err)
	}
	second, tail, err := service.GetMatchesDetailsPage(group.ID, "2026-2027", cursor, 2)
	if err != nil || len(second) != 1 || tail != "" {
		t.Fatalf("last page has %d matches, cursor=%t, error=%v", len(second), tail != "", err)
	}
	all, err := service.GetMatchesDetails(group.ID, "2026-2027")
	if err != nil || len(all) != 3 ||
		all[0].ID != first[0].ID || all[1].ID != first[1].ID || all[2].ID != second[0].ID {
		t.Fatal("page boundaries disagree with the full read")
	}
}
