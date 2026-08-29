package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const testPlayersJWTSecret = "zzz-integration-test-players-secret"

// playersEnv mirrors main.go's wiring for POST /players: authRequired +
// requireGroupAdmin (body/query-resolving, since group_id travels in the
// JSON body here just like POST /matches).
type playersEnv struct {
	memberships *services.GroupMembershipService
	players     *services.PlayerService
	auth        *services.AuthService
	router      *gin.Engine
}

func newPlayersEnv(t *testing.T, tx *gorm.DB) *playersEnv {
	t.Helper()

	membershipService := services.NewGroupMembershipService(tx)
	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testPlayersJWTSecret)
	playerHandler := handlers.NewPlayerHandler(playerService, groupService, membershipService)

	authRequired := handlers.AuthMiddleware(authService)
	requireGroupAdmin := handlers.RequireGroupAdmin(membershipService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/players", authRequired, requireGroupAdmin, playerHandler.CreatePlayer)

	return &playersEnv{
		memberships: membershipService,
		players:     playerService,
		auth:        authService,
		router:      router,
	}
}

func (e *playersEnv) authenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
	t.Helper()
	id, err := e.players.CreatePlayer(name)
	if err != nil {
		t.Fatalf("failed to create player %q: %v", name, err)
	}
	if err := e.auth.Signup(id, email, "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up %q: %v", name, err)
	}
	token, err := e.auth.Login(email, "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in %q: %v", name, err)
	}
	return id, token
}

func (e *playersEnv) do(token string, body any) *httptest.ResponseRecorder {
	raw, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// playersAdminGroup builds a group with one admin and one plain member, both
// authenticated, and returns the group id plus the two tokens — same helper
// shape as matchAdminEnv.matchAdminGroup.
func (e *playersEnv) playersAdminGroup(t *testing.T, tx *gorm.DB, name, adminEmail, memberEmail string) (uuid.UUID, string, string) {
	t.Helper()

	group, err := services.NewGroupService(tx).CreateGroup(name, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, adminToken := e.authenticatedPlayer(t, name+" Admin", adminEmail)
	if err := e.memberships.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}

	memberID, memberToken := e.authenticatedPlayer(t, name+" Member", memberEmail)
	if err := e.memberships.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	return group.ID, adminToken, memberToken
}

// TestCreatePlayer_Integration_NoToken covers the gating change itself: a
// request with no Authorization header at all is now rejected, where before
// this task the route was completely open.
func TestCreatePlayer_Integration_NoToken(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPlayersEnv(t, tx)

	groupID, _, _ := env.playersAdminGroup(t, tx,
		"Zzz Players No Token", "players-no-token-admin@example.com", "players-no-token-member@example.com")

	rec := env.do("", map[string]string{"name": "Zzz Ghost No Token", "group_id": groupID.String()})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated POST /players returned status %d, want 401, body: %s", rec.Code, rec.Body.String())
	}
}

// TestCreatePlayer_Integration_NonAdminForbidden covers the same gating for an
// authenticated caller who belongs to the target group but only as a plain
// member — that must still be rejected, with 403.
func TestCreatePlayer_Integration_NonAdminForbidden(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPlayersEnv(t, tx)

	groupID, _, memberToken := env.playersAdminGroup(t, tx,
		"Zzz Players Non Admin", "players-non-admin-admin@example.com", "players-non-admin-member@example.com")

	rec := env.do(memberToken, map[string]string{"name": "Zzz Ghost Non Admin", "group_id": groupID.String()})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("plain member POST /players returned status %d, want 403, body: %s", rec.Code, rec.Body.String())
	}

	members, err := env.memberships.GetPlayersByGroupID(groupID)
	if err != nil {
		t.Fatalf("GetPlayersByGroupID returned error: %v", err)
	}
	for _, member := range members {
		if member.Name == "Zzz Ghost Non Admin" {
			t.Fatalf("a forbidden POST /players still created the player: %+v", member)
		}
	}
}

// TestCreatePlayer_Integration_AdminCreatesInTargetGroup is the happy path:
// an admin of the target group creates a ghost player, and the new player
// actually ends up a member of *that* group — not the admin's first group,
// not GetDefaultGroup's arbitrary pick (removed).
func TestCreatePlayer_Integration_AdminCreatesInTargetGroup(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPlayersEnv(t, tx)

	groupID, adminToken, _ := env.playersAdminGroup(t, tx,
		"Zzz Players Admin Create", "players-admin-create-admin@example.com", "players-admin-create-member@example.com")

	rec := env.do(adminToken, map[string]string{"name": "Zzz Ghost Created", "group_id": groupID.String()})
	if rec.Code != http.StatusOK {
		t.Fatalf("admin POST /players returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var playerID uuid.UUID
	if err := json.Unmarshal(rec.Body.Bytes(), &playerID); err != nil {
		t.Fatalf("failed to decode created player id: %v", err)
	}

	members, err := env.memberships.GetPlayersByGroupID(groupID)
	if err != nil {
		t.Fatalf("GetPlayersByGroupID returned error: %v", err)
	}
	for _, member := range members {
		if member.ID == playerID {
			return
		}
	}
	t.Errorf("expected created player %s to be a member of group %s, members = %+v", playerID, groupID, members)
}

// TestCreatePlayer_Integration_DuplicateNameInSameGroupRejected covers the new
// per-group soft duplicate guard (GroupMembershipService.HasMemberNamed):
// creating a second "ghost" with a name that already exists in the same
// group's roster (case-insensitive) is rejected with 400, and nothing new is
// created.
func TestCreatePlayer_Integration_DuplicateNameInSameGroupRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPlayersEnv(t, tx)

	groupID, adminToken, _ := env.playersAdminGroup(t, tx,
		"Zzz Players Duplicate", "players-duplicate-admin@example.com", "players-duplicate-member@example.com")

	first := env.do(adminToken, map[string]string{"name": "Zzz Ghost Marco", "group_id": groupID.String()})
	if first.Code != http.StatusOK {
		t.Fatalf("first admin POST /players returned status %d, want 200, body: %s", first.Code, first.Body.String())
	}

	before, err := env.memberships.GetPlayersByGroupID(groupID)
	if err != nil {
		t.Fatalf("GetPlayersByGroupID returned error: %v", err)
	}

	second := env.do(adminToken, map[string]string{"name": "zzz ghost marco", "group_id": groupID.String()})
	if second.Code != http.StatusBadRequest {
		t.Fatalf("duplicate-name POST /players returned status %d, want 400, body: %s", second.Code, second.Body.String())
	}

	after, err := env.memberships.GetPlayersByGroupID(groupID)
	if err != nil {
		t.Fatalf("GetPlayersByGroupID returned error: %v", err)
	}
	if len(after) != len(before) {
		t.Errorf("a rejected duplicate-name POST /players changed group membership from %d to %d members", len(before), len(after))
	}
}

// TestCreatePlayer_Integration_SameNameDifferentGroupAllowed covers the "soft,
// per-group" half of the new guard: the same name used in an unrelated group
// by a different admin must succeed (200), since HasMemberNamed only looks
// within the target group.
func TestCreatePlayer_Integration_SameNameDifferentGroupAllowed(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newPlayersEnv(t, tx)

	groupA, adminTokenA, _ := env.playersAdminGroup(t, tx,
		"Zzz Players Cross Group A", "players-cross-group-a-admin@example.com", "players-cross-group-a-member@example.com")
	groupB, adminTokenB, _ := env.playersAdminGroup(t, tx,
		"Zzz Players Cross Group B", "players-cross-group-b-admin@example.com", "players-cross-group-b-member@example.com")

	inA := env.do(adminTokenA, map[string]string{"name": "Zzz Ghost Shared Name", "group_id": groupA.String()})
	if inA.Code != http.StatusOK {
		t.Fatalf("admin A POST /players returned status %d, want 200, body: %s", inA.Code, inA.Body.String())
	}

	inB := env.do(adminTokenB, map[string]string{"name": "Zzz Ghost Shared Name", "group_id": groupB.String()})
	if inB.Code != http.StatusOK {
		t.Fatalf("admin B POST /players (same name, different group) returned status %d, want 200, body: %s", inB.Code, inB.Body.String())
	}
}
