package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const testMatchVoteJWTSecret = "zzz-integration-test-match-vote-secret"

// voteEnv mirrors registrationEnv's shape for the three vote routes, wired
// exactly as main.go wires them: authRequired + RequireGroupMembershipByMatchPathParam
// for all three, since voting has no admin-only action at all.
type voteEnv struct {
	groups      *services.GroupService
	memberships *services.GroupMembershipService
	players     *services.PlayerService
	teams       *services.TeamService
	matches     *services.MatchService
	votes       *services.MatchVoteService
	auth        *services.AuthService
	router      *gin.Engine
}

func newVoteEnv(t *testing.T, tx *gorm.DB) *voteEnv {
	t.Helper()

	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testMatchVoteJWTSecret)
	matchService := services.NewMatchService(tx)
	voteService := services.NewMatchVoteService(tx)
	voteHandler := handlers.NewMatchVoteHandler(voteService)

	authRequired := handlers.AuthMiddleware(authService)
	requireGroupMemberByMatchID := handlers.RequireGroupMembershipByMatchPathParam(matchService, membershipService, "id")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/matches/:id/votes", authRequired, requireGroupMemberByMatchID, voteHandler.Vote)
	router.DELETE("/matches/:id/votes", authRequired, requireGroupMemberByMatchID, voteHandler.Unvote)
	router.GET("/matches/:id/votes", authRequired, requireGroupMemberByMatchID, voteHandler.ListVotes)

	return &voteEnv{
		groups:      services.NewGroupService(tx),
		memberships: membershipService,
		players:     services.NewPlayerService(tx),
		teams:       services.NewTeamService(tx),
		matches:     matchService,
		votes:       voteService,
		auth:        authService,
		router:      router,
	}
}

func (e *voteEnv) authenticatedPlayer(t *testing.T, name, email string) (uuid.UUID, string) {
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

func (e *voteEnv) do(method, path, token string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, req)
	return rec
}

// voteGroup is a group with an admin, two plain members, and a match whose
// roster is the admin plus the first member — leaving the second member as a
// non-playing voter, exactly the "watched but did not play" case voter
// eligibility is meant to cover.
type voteGroup struct {
	id           uuid.UUID
	adminID      uuid.UUID
	adminToken   string
	member1ID    uuid.UUID
	member1Token string
	member2ID    uuid.UUID
	member2Token string
	matchID      uuid.UUID
}

func (e *voteEnv) newGroupWithComposedMatch(t *testing.T, label string) voteGroup {
	t.Helper()

	group, err := e.groups.CreateGroup("Zzz Votes "+label, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, adminToken := e.authenticatedPlayer(t, "Zzz Vote "+label+" Admin", "vote-"+label+"-admin@example.com")
	if err := e.memberships.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}
	member1ID, member1Token := e.authenticatedPlayer(t, "Zzz Vote "+label+" Member1", "vote-"+label+"-member1@example.com")
	if err := e.memberships.AddPlayerToGroup(group.ID, member1ID); err != nil {
		t.Fatalf("failed to add member1: %v", err)
	}
	member2ID, member2Token := e.authenticatedPlayer(t, "Zzz Vote "+label+" Member2", "vote-"+label+"-member2@example.com")
	if err := e.memberships.AddPlayerToGroup(group.ID, member2ID); err != nil {
		t.Fatalf("failed to add member2: %v", err)
	}

	teams, err := e.teams.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load teams: %v", err)
	}
	black := teams[0]

	matchID, err := e.matches.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}
	if err := e.matches.UpdateMatch(models.MatchWithDetails{
		ID: matchID,
		Teams: []models.TeamWithPlayers{
			{ID: black.ID, Players: []models.PlayerCustom{{ID: adminID}, {ID: member1ID}}},
		},
	}); err != nil {
		t.Fatalf("failed to compose roster: %v", err)
	}

	return voteGroup{
		id: group.ID, matchID: matchID,
		adminID: adminID, adminToken: adminToken,
		member1ID: member1ID, member1Token: member1Token,
		member2ID: member2ID, member2Token: member2Token,
	}
}

func decodeVoteSummary(t *testing.T, rec *httptest.ResponseRecorder) models.MatchVoteSummary {
	t.Helper()
	var summary models.MatchVoteSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("failed to unmarshal vote summary from %s: %v", rec.Body.String(), err)
	}
	return summary
}

// TestVote_Integration_NonPlayingMemberCanVoteForARosterPlayer is the central
// eligibility divergence from sign-ups: member2 never played, but can still
// judge who did.
func TestVote_Integration_NonPlayingMemberCanVoteForARosterPlayer(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newVoteEnv(t, tx)

	group := env.newGroupWithComposedMatch(t, "eligibility")
	path := "/matches/" + group.matchID.String() + "/votes"

	rec := env.do(http.MethodPost, path, group.member2Token, map[string]string{"voted_for_id": group.adminID.String()})
	if rec.Code != http.StatusOK {
		t.Fatalf("non-playing member's vote returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	summary := decodeVoteSummary(t, rec)
	if len(summary.Tally) != 1 || summary.Tally[0].PlayerID != group.adminID || summary.Tally[0].Votes != 1 {
		t.Errorf("tally = %+v, want a single vote for the admin", summary.Tally)
	}
	if summary.MyVoteFor == nil || *summary.MyVoteFor != group.adminID {
		t.Errorf("MyVoteFor = %v, want the admin's id", summary.MyVoteFor)
	}
}

// TestVote_Integration_SelfVoteRejected checks the 400 mapping for a self-vote.
func TestVote_Integration_SelfVoteRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newVoteEnv(t, tx)

	group := env.newGroupWithComposedMatch(t, "selfvote")
	path := "/matches/" + group.matchID.String() + "/votes"

	rec := env.do(http.MethodPost, path, group.adminToken, map[string]string{"voted_for_id": group.adminID.String()})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("self-vote returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}

// TestVote_Integration_NotOnRosterRejected checks the 400 mapping for voting
// for a real group member who never played in this match.
func TestVote_Integration_NotOnRosterRejected(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newVoteEnv(t, tx)

	group := env.newGroupWithComposedMatch(t, "notonroster")
	path := "/matches/" + group.matchID.String() + "/votes"

	rec := env.do(http.MethodPost, path, group.member1Token, map[string]string{"voted_for_id": group.member2ID.String()})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("voting for a non-roster player returned status %d, want 400, body: %s", rec.Code, rec.Body.String())
	}
}

// TestVote_Integration_UpsertOverHTTP: a second POST from the same voter
// changes their vote rather than being refused, unlike the sign-up route's
// ErrAlreadyRegistered/409.
func TestVote_Integration_UpsertOverHTTP(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newVoteEnv(t, tx)

	group := env.newGroupWithComposedMatch(t, "upsert")
	path := "/matches/" + group.matchID.String() + "/votes"

	firstRec := env.do(http.MethodPost, path, group.member2Token, map[string]string{"voted_for_id": group.adminID.String()})
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first vote returned status %d, want 200, body: %s", firstRec.Code, firstRec.Body.String())
	}

	secondRec := env.do(http.MethodPost, path, group.member2Token, map[string]string{"voted_for_id": group.member1ID.String()})
	if secondRec.Code != http.StatusOK {
		t.Fatalf("changed vote returned status %d, want 200 (an upsert, not a conflict), body: %s", secondRec.Code, secondRec.Body.String())
	}
	summary := decodeVoteSummary(t, secondRec)
	if summary.MyVoteFor == nil || *summary.MyVoteFor != group.member1ID {
		t.Fatalf("MyVoteFor after changing = %v, want %s", summary.MyVoteFor, group.member1ID)
	}
	total := 0
	for _, c := range summary.Tally {
		total += c.Votes
	}
	if total != 1 {
		t.Errorf("total votes across the tally = %d, want 1 (the voter's single, current vote)", total)
	}
}

// TestUnvote_Integration_NoOpSuccess: DELETE with no existing vote is 200,
// never 404 — the same "must not fail for nothing" contract as
// ReopenRegistrations.
func TestUnvote_Integration_NoOpSuccess(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newVoteEnv(t, tx)

	group := env.newGroupWithComposedMatch(t, "unvotenoop")
	path := "/matches/" + group.matchID.String() + "/votes"

	rec := env.do(http.MethodDelete, path, group.member2Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE votes with no existing vote returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var body map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal unvote response: %v", err)
	}
	if !body["unvoted"] {
		t.Errorf("unvote response = %v, want {\"unvoted\": true}", body)
	}

	// Cast, then remove: the tally must return to empty.
	if rec := env.do(http.MethodPost, path, group.member2Token, map[string]string{"voted_for_id": group.adminID.String()}); rec.Code != http.StatusOK {
		t.Fatalf("vote returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	if rec := env.do(http.MethodDelete, path, group.member2Token, nil); rec.Code != http.StatusOK {
		t.Fatalf("DELETE votes returned status %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	listRec := env.do(http.MethodGet, path, group.member2Token, nil)
	summary := decodeVoteSummary(t, listRec)
	if len(summary.Tally) != 0 {
		t.Errorf("tally = %+v after the only vote was withdrawn, want empty", summary.Tally)
	}
}

// TestMatchVotes_Integration_OtherGroupGets404 mirrors the equivalent
// registration test: an admin of a different group must not be able to reach
// this match at all, on any of the three routes.
func TestMatchVotes_Integration_OtherGroupGets404(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newVoteEnv(t, tx)

	owner := env.newGroupWithComposedMatch(t, "owner")
	outsider := env.newGroupWithComposedMatch(t, "outsider")
	path := "/matches/" + owner.matchID.String() + "/votes"

	cases := []struct {
		method string
		body   any
	}{
		{http.MethodPost, map[string]string{"voted_for_id": owner.adminID.String()}},
		{http.MethodDelete, nil},
		{http.MethodGet, nil},
	}
	for _, tc := range cases {
		rec := env.do(tc.method, path, outsider.adminToken, tc.body)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s as an admin of another group returned status %d, want 404, body: %s", tc.method, path, rec.Code, rec.Body.String())
		}
	}

	summary, err := env.votes.ListVotes(owner.matchID, owner.adminID)
	if err != nil {
		t.Fatalf("ListVotes returned error: %v", err)
	}
	if len(summary.Tally) != 0 {
		t.Errorf("an outsider's rejected requests still created %d vote(s)", len(summary.Tally))
	}
}

// TestMatchVotes_Integration_UnknownAndMalformedID mirrors the registration
// route's own equivalent test.
func TestMatchVotes_Integration_UnknownAndMalformedID(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	env := newVoteEnv(t, tx)

	group := env.newGroupWithComposedMatch(t, "badid")

	unknown := "/matches/" + uuid.New().String() + "/votes"
	malformed := "/matches/not-a-uuid/votes"

	for _, tc := range []struct {
		method string
		path   string
		want   int
	}{
		{http.MethodPost, unknown, http.StatusNotFound},
		{http.MethodDelete, unknown, http.StatusNotFound},
		{http.MethodGet, unknown, http.StatusNotFound},
		{http.MethodPost, malformed, http.StatusBadRequest},
		{http.MethodDelete, malformed, http.StatusBadRequest},
		{http.MethodGet, malformed, http.StatusBadRequest},
	} {
		var body any
		if tc.method == http.MethodPost {
			body = map[string]string{"voted_for_id": group.adminID.String()}
		}
		rec := env.do(tc.method, tc.path, group.adminToken, body)
		if rec.Code != tc.want {
			t.Errorf("%s %s returned status %d, want %d, body: %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
		}
	}
}
