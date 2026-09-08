package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestGroupIDResolution_HandlerAndMiddlewareAgree pins the property the shared
// resolveScopedGroupID exists for: for one and the same request, the group a
// middleware authorizes against (resolveGroupIDForMembership) and the group the
// handler behind it then acts on (resolveGroupID) are the *same* group.
//
// That matters because DELETE /matches/:id resolves its group twice — once in
// requireGroupAdmin and once in MatchHandler.DeleteMatch — so a divergence
// between the two would mean an admin of group A having their delete applied to
// group B. Before this refactor the two functions were independent copies of
// the same rules, agreeing by coincidence rather than by construction; this
// test is what would catch them drifting apart again.
func TestGroupIDResolution_HandlerAndMiddlewareAgree(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	membershipService := services.NewGroupMembershipService(tx)
	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)

	firstGroup, err := groupService.CreateGroup("Zzz Scope Resolution First", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create the first group: %v", err)
	}
	otherGroup, err := groupService.CreateGroup("Zzz Scope Resolution Other", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create the other group: %v", err)
	}
	playerID, err := playerService.CreatePlayer("Zzz Scope Resolution Player")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(firstGroup.ID, playerID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add the player to the first group: %v", err)
	}

	// Stands in for AuthMiddleware: the only thing both resolvers need from it
	// is "player_id" in the context.
	authenticated := func(c *gin.Context) {
		c.Set("player_id", playerID)
		c.Next()
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	// Both resolvers on one request, in the order production wires them:
	// middleware first, handler second.
	router.DELETE("/both", authenticated, func(c *gin.Context) {
		groupID, ok := resolveGroupIDForMembership(c, membershipService)
		if !ok {
			return
		}
		c.Set("middleware_group_id", groupID)
		c.Next()
	}, func(c *gin.Context) {
		middlewareGroupID := c.MustGet("middleware_group_id").(uuid.UUID)
		handlerGroupID, ok := resolveGroupID(c, membershipService)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"middleware": middlewareGroupID, "handler": handlerGroupID})
	})
	// The handler's resolver alone, to compare how each reports the same
	// malformed input.
	router.DELETE("/handler-only", authenticated, func(c *gin.Context) {
		groupID, ok := resolveGroupID(c, membershipService)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"handler": groupID})
	})

	do := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, path, nil))
		return rec
	}

	agreedGroup := func(t *testing.T, path string) uuid.UUID {
		t.Helper()
		rec := do(path)
		if rec.Code != http.StatusOK {
			t.Fatalf("DELETE %s returned status %d, want 200, body: %s", path, rec.Code, rec.Body.String())
		}
		var body struct {
			Middleware uuid.UUID `json:"middleware"`
			Handler    uuid.UUID `json:"handler"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("failed to unmarshal %s response: %v", path, err)
		}
		if body.Middleware != body.Handler {
			t.Fatalf("DELETE %s: middleware authorized group %s while the handler acted on %s", path, body.Middleware, body.Handler)
		}
		return body.Handler
	}

	// An explicit group_id: both must read it, and neither may substitute the
	// caller's own first group for it.
	if got := agreedGroup(t, "/both?group_id="+otherGroup.ID.String()); got != otherGroup.ID {
		t.Errorf("with an explicit group_id both resolved %s, want %s", got, otherGroup.ID)
	}

	// No group_id at all: both must fall back to the same group — the caller's
	// first — rather than one falling back and the other failing.
	if got := agreedGroup(t, "/both"); got != firstGroup.ID {
		t.Errorf("with no group_id both resolved %s, want the caller's first group %s", got, firstGroup.ID)
	}

	// A malformed group_id: same status and same message from either side, so
	// the two never disagree about what counts as a valid group id either.
	middlewareRec := do("/both?group_id=not-a-uuid")
	handlerRec := do("/handler-only?group_id=not-a-uuid")
	if middlewareRec.Code != http.StatusBadRequest || handlerRec.Code != http.StatusBadRequest {
		t.Fatalf("malformed group_id: middleware status %d, handler status %d, want 400 from both", middlewareRec.Code, handlerRec.Code)
	}
	if middlewareRec.Body.String() != handlerRec.Body.String() {
		t.Errorf("malformed group_id answered differently:\n middleware: %s\n handler:    %s", middlewareRec.Body.String(), handlerRec.Body.String())
	}
}
