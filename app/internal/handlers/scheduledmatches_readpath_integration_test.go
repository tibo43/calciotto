package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const testScheduledReadJWTSecret = "zzz-integration-test-scheduled-read-secret"

// TestMatchesDetails_Integration_SchedulingOnTheWire checks the JSON actually
// served by GET /matches/details and GET /matches/:id/details, not just the Go
// structs behind them. Two things matter here and neither is visible from a
// struct assertion:
//
//   - an ordinary, unscheduled match must serialize to exactly the keys it did
//     before this feature existed. A frontend has been parsing this payload for
//     months and the sign-up UI does not exist yet, so any extra key — even a
//     null one — is an unforced change to a live contract. That is what the
//     omitempty tags on MatchWithDetails buy, and this is what pins them.
//   - a scheduled match must carry all five new keys, spelled in the DTO's
//     PascalCase convention, so the upcoming match card can render a kick-off
//     time and "N / MaxPlayers signed up" from this one response.
func TestMatchesDetails_Integration_SchedulingOnTheWire(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	registrationService := services.NewMatchRegistrationService(tx)
	authService := services.NewAuthService(tx, testScheduledReadJWTSecret)
	matchHandler := handlers.NewMatchHandler(matchService, membershipService)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/matches/details",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupMembership(membershipService),
		matchHandler.GetMatchesDetails)
	router.GET("/matches/:id/details",
		handlers.AuthMiddleware(authService),
		handlers.RequireGroupMembership(membershipService),
		matchHandler.GetMatchDetailsByID)

	group, err := groupService.CreateGroup("Zzz Scheduled Wire Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	bobID, err := playerService.CreatePlayer("Zzz Scheduled Wire Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, bobID); err != nil {
		t.Fatalf("failed to add bob to the group: %v", err)
	}
	if err := authService.Signup(bobID, "scheduled-wire-bob@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("failed to sign up bob: %v", err)
	}
	token, err := authService.Login("scheduled-wire-bob@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("failed to log in bob: %v", err)
	}

	// Fixed timestamps: the kick-off is far enough ahead that the registration
	// window is open, the opening time is in the past so signing up is allowed.
	kickoff := time.Date(2030, time.October, 5, 19, 30, 0, 0, time.UTC)
	opensAt := time.Date(2020, time.January, 1, 8, 0, 0, 0, time.UTC)
	maxPlayers := 16

	scheduledID, err := matchService.CreateMatch(services.MatchSpec{
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &opensAt,
		MaxPlayers:          &maxPlayers,
	}, group.ID)
	if err != nil {
		t.Fatalf("failed to create the scheduled match: %v", err)
	}
	if err := registrationService.Register(scheduledID, bobID); err != nil {
		t.Fatalf("failed to register bob: %v", err)
	}

	plainID, err := matchService.CreateMatch(services.MatchSpec{
		Date: models.Date(time.Date(2025, time.October, 5, 0, 0, 0, 0, time.UTC)),
	}, group.ID)
	if err != nil {
		t.Fatalf("failed to create the plain match: %v", err)
	}

	get := func(path string) []byte {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s returned status %d, want 200, body: %s", path, rec.Code, rec.Body.String())
		}
		return rec.Body.Bytes()
	}

	// The pre-feature key set (plus CreatedAt, added for the Man of the Match
	// voting window and always present, scheduled or not), and the five
	// scheduling keys added on top of it.
	baseKeys := []string{"ID", "GroupID", "Date", "CreatedAt", "Teams"}
	scheduleKeys := []string{"ScheduledAt", "RegistrationOpensAt", "MaxPlayers", "RegistrationCount"}

	assertPlain := func(source string, raw map[string]json.RawMessage) {
		t.Helper()
		for _, key := range baseKeys {
			if _, ok := raw[key]; !ok {
				t.Errorf("%s: unscheduled match is missing pre-existing key %q, got %v", source, key, keysOf(raw))
			}
		}
		if len(raw) != len(baseKeys) {
			t.Errorf("%s: unscheduled match has keys %v, want exactly %v — a client that knows nothing "+
				"about scheduling must see no new keys", source, keysOf(raw), baseKeys)
		}
		// Spelled out individually so a failure names the offending key.
		for _, key := range append(scheduleKeys, "RegistrationsClosedAt") {
			if _, ok := raw[key]; ok {
				t.Errorf("%s: unscheduled match carries %q, want it omitted", source, key)
			}
		}
	}

	assertScheduled := func(source string, raw map[string]json.RawMessage) {
		t.Helper()
		for _, key := range append(append([]string{}, baseKeys...), scheduleKeys...) {
			if _, ok := raw[key]; !ok {
				t.Errorf("%s: scheduled match is missing key %q, got %v", source, key, keysOf(raw))
			}
		}
		// Sign-ups are open, so this one stays absent rather than null.
		if _, ok := raw["RegistrationsClosedAt"]; ok {
			t.Errorf("%s: scheduled match carries RegistrationsClosedAt while sign-ups are open", source)
		}

		var typed struct {
			Date              string     `json:"Date"`
			ScheduledAt       *time.Time `json:"ScheduledAt"`
			MaxPlayers        *int       `json:"MaxPlayers"`
			RegistrationCount *int       `json:"RegistrationCount"`
		}
		body, _ := json.Marshal(raw)
		if err := json.Unmarshal(body, &typed); err != nil {
			t.Fatalf("%s: failed to decode the scheduled match: %v", source, err)
		}
		if typed.Date != "2030-10-05" {
			t.Errorf("%s: Date = %q, want 2030-10-05 (derived from the kick-off day)", source, typed.Date)
		}
		if typed.ScheduledAt == nil || !typed.ScheduledAt.Equal(kickoff) {
			t.Errorf("%s: ScheduledAt = %v, want %v", source, typed.ScheduledAt, kickoff)
		}
		if typed.MaxPlayers == nil || *typed.MaxPlayers != maxPlayers {
			t.Errorf("%s: MaxPlayers = %v, want %d", source, typed.MaxPlayers, maxPlayers)
		}
		if typed.RegistrationCount == nil || *typed.RegistrationCount != 1 {
			t.Errorf("%s: RegistrationCount = %v, want 1", source, typed.RegistrationCount)
		}
	}

	// The list endpoint, decoded as raw key maps so absent keys stay absent.
	var list []map[string]json.RawMessage
	if err := json.Unmarshal(get("/matches/details?group_id="+group.ID.String()), &list); err != nil {
		t.Fatalf("failed to decode GET /matches/details: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("GET /matches/details returned %d matches, want 2: %v", len(list), list)
	}
	byID := map[string]map[string]json.RawMessage{}
	for _, entry := range list {
		var id uuid.UUID
		if err := json.Unmarshal(entry["ID"], &id); err != nil {
			t.Fatalf("failed to decode a match id: %v", err)
		}
		byID[id.String()] = entry
	}
	assertScheduled("GET /matches/details", byID[scheduledID.String()])
	assertPlain("GET /matches/details", byID[plainID.String()])

	// ...and the detail endpoint, which builds its payload on a separate code
	// path and therefore has to be checked separately.
	var detail map[string]json.RawMessage
	if err := json.Unmarshal(get("/matches/"+scheduledID.String()+"/details?group_id="+group.ID.String()), &detail); err != nil {
		t.Fatalf("failed to decode GET /matches/:id/details (scheduled): %v", err)
	}
	assertScheduled("GET /matches/:id/details", detail)

	var plainDetail map[string]json.RawMessage
	if err := json.Unmarshal(get("/matches/"+plainID.String()+"/details?group_id="+group.ID.String()), &plainDetail); err != nil {
		t.Fatalf("failed to decode GET /matches/:id/details (plain): %v", err)
	}
	assertPlain("GET /matches/:id/details", plainDetail)
}

// keysOf renders a decoded payload's key set, since json.RawMessage values
// print as byte slices and drown the actual failure.
func keysOf(raw map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
