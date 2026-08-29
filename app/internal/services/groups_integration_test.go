package services_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

func TestCreateGroup_Integration_CreatesDefaultTeams(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)

	group, err := groupService.CreateGroup("Zzz Integration Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	if group.ID == uuid.Nil {
		t.Fatal("CreateGroup returned a nil group ID")
	}
	if group.Name != "Zzz Integration Group" {
		t.Errorf("group.Name = %q, want %q", group.Name, "Zzz Integration Group")
	}

	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("GetTeamsByGroupID returned error: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected exactly 2 teams for a new group, got %d: %+v", len(teams), teams)
	}
	colours := map[string]bool{}
	names := map[string]bool{}
	for _, team := range teams {
		if team.GroupID != group.ID {
			t.Errorf("team %+v has GroupID %s, want %s", team, team.GroupID, group.ID)
		}
		colours[team.Colour] = true
		names[team.Name] = true
	}
	if !colours["black"] || !colours["white"] {
		t.Errorf("expected teams with colours black and white, got %+v", teams)
	}
	if !names["Black"] || !names["White"] {
		t.Errorf("expected teams named Black and White, got %+v", teams)
	}
}

// TestCreateGroup_Integration_CustomTeamSpecs pins that the caller's own
// team names/colours are what get persisted — not some default — now that
// CreateGroup takes them explicitly instead of hardcoding black/white.
func TestCreateGroup_Integration_CustomTeamSpecs(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)

	specs := [2]services.TeamSpec{
		{Name: "Les Rouges", Colour: "red"},
		{Name: "Les Bleus", Colour: "blue"},
	}
	group, err := groupService.CreateGroup("Zzz Integration Custom Teams Group", specs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("GetTeamsByGroupID returned error: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected exactly 2 teams, got %d: %+v", len(teams), teams)
	}
	got := map[string]string{}
	for _, team := range teams {
		got[team.Name] = team.Colour
	}
	for _, spec := range specs {
		if got[spec.Name] != spec.Colour {
			t.Errorf("team %q has colour %q, want %q (teams: %+v)", spec.Name, got[spec.Name], spec.Colour, teams)
		}
	}
}

// TestCreateGroup_Integration_RequiresTeamNameAndColour covers both
// validation sentinels CreateGroup added alongside explicit TeamSpecs: an
// empty (or whitespace-only) name or colour on either team must reject the
// whole creation, and must not leave a half-created group behind.
func TestCreateGroup_Integration_RequiresTeamNameAndColour(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)

	cases := []struct {
		name    string
		teams   [2]services.TeamSpec
		wantErr error
	}{
		{
			name: "missing first team name",
			teams: [2]services.TeamSpec{
				{Name: "  ", Colour: "red"},
				{Name: "White", Colour: "white"},
			},
			wantErr: services.ErrTeamNameRequired,
		},
		{
			name: "missing second team colour",
			teams: [2]services.TeamSpec{
				{Name: "Black", Colour: "black"},
				{Name: "White", Colour: ""},
			},
			wantErr: services.ErrTeamColourRequired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			group, err := groupService.CreateGroup("Zzz Integration Invalid Teams Group "+tc.name, tc.teams)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("CreateGroup error = %v, want %v", err, tc.wantErr)
			}
			if group != nil {
				t.Errorf("CreateGroup returned a non-nil group on validation failure: %+v", group)
			}
		})
	}
}

func TestGroupMembership_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	groupA, err := groupService.CreateGroup("Zzz Integration Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Integration Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	aliceID, err := playerService.CreatePlayer("Zzz Integration Membership Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}

	// Alice belongs to both groups.
	if err := membershipService.AddPlayerToGroup(groupA.ID, aliceID); err != nil {
		t.Fatalf("failed to add alice to group A: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(groupB.ID, aliceID); err != nil {
		t.Fatalf("failed to add alice to group B: %v", err)
	}

	groups, err := membershipService.GetGroupsByPlayerID(aliceID)
	if err != nil {
		t.Fatalf("GetGroupsByPlayerID returned error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected alice to belong to 2 groups, got %d: %+v", len(groups), groups)
	}

	membersA, err := membershipService.GetPlayersByGroupID(groupA.ID)
	if err != nil {
		t.Fatalf("GetPlayersByGroupID returned error: %v", err)
	}
	if len(membersA) != 1 || membersA[0].ID != aliceID {
		t.Errorf("group A members = %+v, want just alice", membersA)
	}

	// Duplicate membership must be rejected by the unique index.
	if err := membershipService.AddPlayerToGroup(groupA.ID, aliceID); err == nil {
		t.Error("expected adding an already-member player to the same group to fail, got nil error")
	}
}

func TestMatchesAndStandings_Integration_ScopedPerGroup(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	standingsService := services.NewStandingsService(tx, services.NewGroupMembershipService(tx))

	groupA, err := groupService.CreateGroup("Zzz Integration Scoped Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group A: %v", err)
	}
	groupB, err := groupService.CreateGroup("Zzz Integration Scoped Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group B: %v", err)
	}

	teamsA, err := teamService.GetTeamsByGroupID(groupA.ID)
	if err != nil || len(teamsA) != 2 {
		t.Fatalf("failed to load group A teams: err=%v teams=%+v", err, teamsA)
	}
	teamsB, err := teamService.GetTeamsByGroupID(groupB.ID)
	if err != nil || len(teamsB) != 2 {
		t.Fatalf("failed to load group B teams: err=%v teams=%+v", err, teamsB)
	}

	aliceID, err := playerService.CreatePlayer("Zzz Integration Scoped Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration Scoped Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}

	matchAID, err := matchService.CreateMatch(models.Date(time.Now()), groupA.ID)
	if err != nil {
		t.Fatalf("failed to create match in group A: %v", err)
	}
	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID: matchAID,
		Teams: []models.TeamWithPlayers{
			{ID: teamsA[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 2}}},
			{ID: teamsA[1].ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 0}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch (group A) returned error: %v", err)
	}

	matchBID, err := matchService.CreateMatch(models.Date(time.Now()), groupB.ID)
	if err != nil {
		t.Fatalf("failed to create match in group B: %v", err)
	}
	carolID, err := playerService.CreatePlayer("Zzz Integration Scoped Carol")
	if err != nil {
		t.Fatalf("failed to create player carol: %v", err)
	}
	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID: matchBID,
		Teams: []models.TeamWithPlayers{
			{ID: teamsB[0].ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 5}}},
			{ID: teamsB[1].ID, Players: []models.PlayerCustom{{ID: carolID, GoalsScored: 0}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch (group B) returned error: %v", err)
	}

	// GetMatchesDetails for group A must not include group B's match, and its
	// team backfill must only ever show group A's 2 teams.
	matchesA, err := matchService.GetMatchesDetails(groupA.ID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails(groupA) returned error: %v", err)
	}
	if len(matchesA) != 1 || matchesA[0].ID != matchAID {
		t.Fatalf("GetMatchesDetails(groupA) = %+v, want only matchA", matchesA)
	}
	if len(matchesA[0].Teams) != 2 {
		t.Fatalf("expected exactly group A's 2 teams, got %d: %+v", len(matchesA[0].Teams), matchesA[0].Teams)
	}

	matchesB, err := matchService.GetMatchesDetails(groupB.ID, "")
	if err != nil {
		t.Fatalf("GetMatchesDetails(groupB) returned error: %v", err)
	}
	if len(matchesB) != 1 || matchesB[0].ID != matchBID {
		t.Fatalf("GetMatchesDetails(groupB) = %+v, want only matchB", matchesB)
	}

	// Fetching group A's match while scoped to group B must behave as not
	// found — specifically services.ErrMatchNotFound, so the handler can map
	// it to a 404 rather than a generic 500.
	if _, err := matchService.GetMatchDetailsByID(matchAID, groupB.ID); !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("expected GetMatchDetailsByID(matchA, groupB) to fail with ErrMatchNotFound, got: %v", err)
	}

	// Standings must be computed per group: bob lost in group A (0 points) but
	// won in group B (3 points) — the two must not be mixed together.
	pointsA, err := standingsService.GetPointsStandings(groupA.ID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings(groupA) returned error: %v", err)
	}
	bobA := pointsRowByID(pointsA, bobID)
	if bobA == nil || bobA.Points != 0 || bobA.Played != 1 {
		t.Errorf("bob's group A standing = %+v, want 0 points from 1 match", bobA)
	}

	pointsB, err := standingsService.GetPointsStandings(groupB.ID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings(groupB) returned error: %v", err)
	}
	bobB := pointsRowByID(pointsB, bobID)
	if bobB == nil || bobB.Points != 3 || bobB.Played != 1 {
		t.Errorf("bob's group B standing = %+v, want 3 points from 1 match", bobB)
	}

	aliceA := pointsRowByID(pointsA, aliceID)
	if aliceA == nil || aliceA.Points != 3 {
		t.Errorf("alice's group A standing = %+v, want 3 points (won 2-0)", aliceA)
	}
	if aliceInB := pointsRowByID(pointsB, aliceID); aliceInB != nil {
		t.Errorf("alice never played in group B, but got a standings row: %+v", aliceInB)
	}
}

// TestCreateGroup_Integration_GeneratesInviteCode pins the invite code's
// shape: every group gets its own, drawn from the unambiguous alphabet, so it
// can be read out loud or typed by hand without confusion.
func TestCreateGroup_Integration_GeneratesInviteCode(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)

	first, err := groupService.CreateGroup("Zzz Integration Invite Group A", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	second, err := groupService.CreateGroup("Zzz Integration Invite Group B", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	if first.InviteCode == second.InviteCode {
		t.Errorf("two groups share the invite code %q", first.InviteCode)
	}
	for _, group := range []*models.Group{first, second} {
		if len(group.InviteCode) != 8 {
			t.Errorf("invite code %q has length %d, want 8", group.InviteCode, len(group.InviteCode))
		}
		if strings.ContainsAny(group.InviteCode, "01OIL") {
			t.Errorf("invite code %q contains an ambiguous character", group.InviteCode)
		}
		if group.InviteCode != strings.ToUpper(group.InviteCode) {
			t.Errorf("invite code %q is not upper-case, so case normalization on join would not match it", group.InviteCode)
		}
	}
}

// TestJoinByInviteCode_Integration covers the service's own contract: a valid
// code enrolls the player, an unknown one and a second join report distinct
// sentinel errors the handler maps to 404 and 400.
func TestJoinByInviteCode_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Integration Join Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}
	playerID, err := playerService.CreatePlayer("Zzz Integration Join Player")
	if err != nil {
		t.Fatalf("CreatePlayer returned error: %v", err)
	}

	joined, err := groupService.JoinByInviteCode(playerID, group.InviteCode)
	if err != nil {
		t.Fatalf("JoinByInviteCode returned error: %v", err)
	}
	if joined.ID != group.ID {
		t.Errorf("JoinByInviteCode returned group %s, want %s", joined.ID, group.ID)
	}
	isMember, err := membershipService.IsMember(group.ID, playerID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !isMember {
		t.Error("player is not a member after JoinByInviteCode")
	}

	if _, err := groupService.JoinByInviteCode(playerID, group.InviteCode); !errors.Is(err, services.ErrAlreadyMember) {
		t.Errorf("re-joining error = %v, want ErrAlreadyMember", err)
	}
	if _, err := groupService.JoinByInviteCode(playerID, "ZZZZZZZZ"); !errors.Is(err, services.ErrInviteCodeNotFound) {
		t.Errorf("unknown code error = %v, want ErrInviteCodeNotFound", err)
	}
}

// TestGetPlayersWithRoleByGroupID_Integration covers the new member-listing
// query backing GET /groups/:id/players: a mix of an admin and a plain
// member in the same group must come back each tagged with their own role,
// not both defaulting to one or the other.
func TestGetPlayersWithRoleByGroupID_Integration(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	playerService := services.NewPlayerService(tx)
	membershipService := services.NewGroupMembershipService(tx)

	group, err := groupService.CreateGroup("Zzz Integration Roles Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Zzz Integration Roles Admin")
	if err != nil {
		t.Fatalf("failed to create admin player: %v", err)
	}
	memberID, err := playerService.CreatePlayer("Zzz Integration Roles Member")
	if err != nil {
		t.Fatalf("failed to create member player: %v", err)
	}

	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add admin: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, memberID); err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	players, err := membershipService.GetPlayersWithRoleByGroupID(group.ID)
	if err != nil {
		t.Fatalf("GetPlayersWithRoleByGroupID returned error: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 members, got %d: %+v", len(players), players)
	}

	var gotAdmin, gotMember *models.PlayerWithRole
	for i := range players {
		switch players[i].ID {
		case adminID:
			gotAdmin = &players[i]
		case memberID:
			gotMember = &players[i]
		}
	}
	if gotAdmin == nil || gotAdmin.Role != models.RoleAdmin {
		t.Errorf("admin row = %+v, want role %q", gotAdmin, models.RoleAdmin)
	}
	if gotMember == nil || gotMember.Role != models.RoleMember {
		t.Errorf("member row = %+v, want role %q", gotMember, models.RoleMember)
	}
}

// TestStandings_Integration_IsMemberAfterRemoval covers the new IsMember tag
// on GetPointsStandings/GetScorers: a player removed from the group must
// still appear in both — their historical goals aren't lost, since standings
// are computed from match history, not current membership — but tagged
// IsMember: false, while a player still in the group is IsMember: true.
func TestStandings_Integration_IsMemberAfterRemoval(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	playerService := services.NewPlayerService(tx)
	matchService := services.NewMatchService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	standingsService := services.NewStandingsService(tx, membershipService)

	group, err := groupService.CreateGroup("Zzz Integration IsMember Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil || len(teams) != 2 {
		t.Fatalf("failed to load group teams: err=%v teams=%+v", err, teams)
	}

	aliceID, err := playerService.CreatePlayer("Zzz Integration IsMember Alice")
	if err != nil {
		t.Fatalf("failed to create player alice: %v", err)
	}
	bobID, err := playerService.CreatePlayer("Zzz Integration IsMember Bob")
	if err != nil {
		t.Fatalf("failed to create player bob: %v", err)
	}

	if err := membershipService.AddPlayerToGroupWithRole(group.ID, aliceID, models.RoleAdmin); err != nil {
		t.Fatalf("failed to add alice: %v", err)
	}
	if err := membershipService.AddPlayerToGroup(group.ID, bobID); err != nil {
		t.Fatalf("failed to add bob: %v", err)
	}

	matchID, err := matchService.CreateMatch(models.Date(time.Now()), group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}
	if err := matchService.UpdateMatch(models.MatchWithDetails{
		ID: matchID,
		Teams: []models.TeamWithPlayers{
			{ID: teams[0].ID, Players: []models.PlayerCustom{{ID: aliceID, GoalsScored: 2}}},
			{ID: teams[1].ID, Players: []models.PlayerCustom{{ID: bobID, GoalsScored: 1}}},
		},
	}); err != nil {
		t.Fatalf("UpdateMatch returned error: %v", err)
	}

	// Bob leaves the group; alice (an admin) removes him.
	if err := membershipService.RemoveMember(group.ID, aliceID, bobID); err != nil {
		t.Fatalf("RemoveMember returned error: %v", err)
	}

	points, err := standingsService.GetPointsStandings(group.ID, "")
	if err != nil {
		t.Fatalf("GetPointsStandings returned error: %v", err)
	}
	alicePoints, bobPoints := pointsRowByID(points, aliceID), pointsRowByID(points, bobID)
	if alicePoints == nil || !alicePoints.IsMember {
		t.Errorf("alice's points row = %+v, want IsMember true", alicePoints)
	}
	if bobPoints == nil || bobPoints.IsMember {
		t.Errorf("bob's points row = %+v, want IsMember false after removal", bobPoints)
	}
	if bobPoints != nil && bobPoints.GoalsFor != 1 {
		t.Errorf("bob's points row = %+v, want his goal kept despite removal", bobPoints)
	}

	scorers, err := standingsService.GetScorers(group.ID, "")
	if err != nil {
		t.Fatalf("GetScorers returned error: %v", err)
	}
	aliceScorer, bobScorer := scorerRowByID(scorers, aliceID), scorerRowByID(scorers, bobID)
	if aliceScorer == nil || !aliceScorer.IsMember {
		t.Errorf("alice's scorer row = %+v, want IsMember true", aliceScorer)
	}
	if bobScorer == nil || bobScorer.IsMember {
		t.Errorf("bob's scorer row = %+v, want IsMember false after removal", bobScorer)
	}
}
