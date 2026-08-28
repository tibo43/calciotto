package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/handlers"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestCreatePlayer_Integration_AttachesToDefaultGroup exercises the same
// fallback CreateMatch uses (see resolveGroupID): when a POST /players body
// carries no group_id, the handler must still attach the new player to the
// system's default group via GroupMembershipService.
func TestCreatePlayer_Integration_AttachesToDefaultGroup(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerHandler := handlers.NewPlayerHandler(services.NewPlayerService(tx), groupService, membershipService)

	defaultGroup, err := groupService.GetDefaultGroup()
	if err != nil {
		t.Skipf("skipping: no group exists to act as the default (run `go run ./cmd/seed`): %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/players", playerHandler.CreatePlayer)

	body, _ := json.Marshal(map[string]string{"name": "Zzz Integration Player No Group"})
	req := httptest.NewRequest(http.MethodPost, "/players", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("CreatePlayer returned status %d, body: %s", rec.Code, rec.Body.String())
	}

	var playerID uuid.UUID
	if err := json.Unmarshal(rec.Body.Bytes(), &playerID); err != nil {
		t.Fatalf("failed to decode created player id: %v", err)
	}

	members, err := membershipService.GetPlayersByGroupID(defaultGroup.ID)
	if err != nil {
		t.Fatalf("GetPlayersByGroupID returned error: %v", err)
	}
	for _, member := range members {
		if member.ID == playerID {
			return
		}
	}
	t.Errorf("expected created player %s to be a member of the default group %s, members = %+v", playerID, defaultGroup.ID, members)
}
