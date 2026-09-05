// cmd/richseed enriches an already-seeded dev database into a realistic,
// varied dataset for manual testing — a one-off complement to cmd/seed
// (which stays the minimal, idempotent tool documented in CLAUDE.md), not a
// replacement for it. Run `go run ./cmd/seed` first: this expects the
// "Default" group and its 12 players (including "thibaut") to already
// exist.
//
// It: gives every credential-less ("ghost") player left in the database a
// real email/password; creates three more groups; adds thibaut to each as a
// plain member (he stays the "Default" group's only admin — a random other
// player becomes each new group's admin instead); randomly spreads the rest
// of the player pool across the three new groups; and, in every one of the
// four groups, creates a mix of matches spanning several seasons — plain
// already-played matches, a recently-played scheduled match with a composed
// roster and Man of the Match votes, and one scheduled match in each sign-up
// state the app supports (not yet open, open with a waiting list, closed but
// not yet composed, closed and already composed ahead of kick-off).
//
// Unlike cmd/seed, this is NOT idempotent for matches — re-running it adds
// another full spread on top of whatever is already there. Group creation
// and membership are safe to re-run (existing groups/memberships are
// reused/skipped).
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const richSeedAuthSecret = "richseed-only-unused-jwt-secret"
const richSeedPassword = "user1234"

type newGroupSpec struct {
	Name  string
	Teams [2]services.TeamSpec
}

var extraGroups = []newGroupSpec{
	{
		Name: "Les Copains du Mardi",
		Teams: [2]services.TeamSpec{
			{Name: "Les Rouges", Colour: "red"},
			{Name: "Les Bleus", Colour: "blue"},
		},
	},
	{
		Name: "Foot du Vendredi",
		Teams: [2]services.TeamSpec{
			{Name: "Panthers", Colour: "black"},
			{Name: "Tigers", Colour: "orange"},
		},
	},
	{
		Name: "Ligue Amateur Sud",
		Teams: [2]services.TeamSpec{
			{Name: "FC Nord", Colour: "green"},
			{Name: "FC Sud", Colour: "yellow"},
		},
	},
}

func main() {
	recentOnly := flag.Bool("recent-only", false, "only (re)generate the recently-played composed match + MOTM votes per group — for recovering from a bad run without duplicating every other match type")
	flag.Parse()

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	groupService := services.NewGroupService(db)
	membershipService := services.NewGroupMembershipService(db)
	teamService := services.NewTeamService(db)
	matchService := services.NewMatchService(db)
	registrationService := services.NewMatchRegistrationService(db)
	voteService := services.NewMatchVoteService(db)
	matchPlayerService := services.NewMatchPlayerService(db)
	authService := services.NewAuthService(db, richSeedAuthSecret)

	giveGhostsAccounts(db, authService)

	var allPlayers []models.Player
	if err := db.Find(&allPlayers).Error; err != nil {
		log.Fatalf("failed to load players: %v", err)
	}
	if len(allPlayers) == 0 {
		log.Fatal("no players found — run `go run ./cmd/seed` first")
	}

	thibautIdx := -1
	for i, p := range allPlayers {
		if strings.EqualFold(p.Name, "thibaut") {
			thibautIdx = i
			break
		}
	}
	if thibautIdx == -1 {
		log.Fatal(`no player named "thibaut" found — run 'go run ./cmd/seed' first`)
	}
	thibaut := allPlayers[thibautIdx]
	others := make([]models.Player, 0, len(allPlayers)-1)
	for i, p := range allPlayers {
		if i != thibautIdx {
			others = append(others, p)
		}
	}

	var defaultGroup models.Group
	if err := db.Where("name = ?", "Default").First(&defaultGroup).Error; err != nil {
		log.Fatalf(`failed to find the "Default" group — run 'go run ./cmd/seed' first: %v`, err)
	}

	groups := []models.Group{defaultGroup}
	for _, spec := range extraGroups {
		group := findOrCreateGroup(db, groupService, spec)
		groups = append(groups, *group)

		addMemberIfMissing(membershipService, group.ID, thibaut.ID, models.RoleMember)

		admin := others[rand.Intn(len(others))]
		addMemberIfMissing(membershipService, group.ID, admin.ID, models.RoleAdmin)

		for _, p := range others {
			if p.ID == admin.ID {
				continue
			}
			if rand.Float64() < 0.6 {
				addMemberIfMissing(membershipService, group.ID, p.ID, models.RoleMember)
			}
		}
		// A coin-flip roster can come up thin — top it up so every group has
		// enough members for a believable match.
		ensureMinMembers(membershipService, group.ID, others, 6)

		log.Printf("group %q ready: admin=%s, thibaut is a member", group.Name, admin.Name)
	}

	for _, group := range groups {
		roster, err := membershipService.GetPlayersByGroupID(group.ID)
		if err != nil {
			log.Fatalf("failed to load roster for group %q: %v", group.Name, err)
		}
		teams, err := teamService.GetTeamsByGroupID(group.ID)
		if err != nil || len(teams) != 2 {
			log.Fatalf("failed to load teams for group %q: %v", group.Name, err)
		}
		if *recentOnly {
			generateRecentComposedMatch(group, teams, roster, matchService, matchPlayerService, voteService)
		} else {
			generateMatches(group, teams, roster, matchService, matchPlayerService, registrationService, voteService)
		}
	}

	log.Println("done — rich dataset ready.")
}

// giveGhostsAccounts finds every Player with no Email (a "ghost" — the
// admin-created, credential-less roster entry the app itself can no longer
// create, but that can still exist from before that feature was removed) and
// attaches a real email/password to it, the same seedPassword convention
// cmd/seed uses for every player it creates.
func giveGhostsAccounts(db *gorm.DB, authService *services.AuthService) {
	var ghosts []models.Player
	if err := db.Where("email IS NULL").Find(&ghosts).Error; err != nil {
		log.Fatalf("failed to load ghost players: %v", err)
	}
	if len(ghosts) == 0 {
		log.Println("no ghost players found — everyone already has an account")
		return
	}
	for _, p := range ghosts {
		base := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p.Name), " ", "."))
		for attempt := 0; ; attempt++ {
			email := base + "@mail.com"
			if attempt > 0 {
				email = fmt.Sprintf("%s%d@mail.com", base, attempt+1)
			}
			err := authService.Signup(p.ID, email, richSeedPassword)
			if err == nil {
				log.Printf("gave %q a real account (%s / %s)", p.Name, email, richSeedPassword)
				break
			}
			if errors.Is(err, services.ErrEmailAlreadyUsed) {
				continue
			}
			log.Fatalf("failed to give %q an account: %v", p.Name, err)
		}
	}
}

func findOrCreateGroup(db *gorm.DB, groupService *services.GroupService, spec newGroupSpec) *models.Group {
	var existing models.Group
	if err := db.Where("name = ?", spec.Name).First(&existing).Error; err == nil {
		log.Printf("group %q already exists (%s) — reusing it", existing.Name, existing.ID)
		return &existing
	}
	group, err := groupService.CreateGroup(spec.Name, spec.Teams)
	if err != nil {
		log.Fatalf("failed to create group %q: %v", spec.Name, err)
	}
	log.Printf("created group %q (%s)", group.Name, group.ID)
	return group
}

func addMemberIfMissing(membershipService *services.GroupMembershipService, groupID, playerID uuid.UUID, role string) {
	isMember, err := membershipService.IsMember(groupID, playerID)
	if err != nil {
		log.Fatalf("failed to check membership: %v", err)
	}
	if isMember {
		return
	}
	if err := membershipService.AddPlayerToGroupWithRole(groupID, playerID, role); err != nil {
		log.Fatalf("failed to add member: %v", err)
	}
}

func ensureMinMembers(membershipService *services.GroupMembershipService, groupID uuid.UUID, pool []models.Player, min int) {
	members, err := membershipService.GetPlayersByGroupID(groupID)
	if err != nil {
		log.Fatalf("failed to count members: %v", err)
	}
	if len(members) >= min {
		return
	}
	memberIDs := make(map[uuid.UUID]bool, len(members))
	for _, m := range members {
		memberIDs[m.ID] = true
	}
	shuffled := make([]models.Player, len(pool))
	copy(shuffled, pool)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	for _, p := range shuffled {
		if len(members) >= min {
			break
		}
		if memberIDs[p.ID] {
			continue
		}
		addMemberIfMissing(membershipService, groupID, p.ID, models.RoleMember)
		memberIDs[p.ID] = true
		members = append(members, p)
	}
}

// generateMatches builds a full spread of match types for one group, across
// several seasons: plain already-played matches in each of the three most
// recent past seasons, one recently-played scheduled match (composed roster
// + Man of the Match votes) and, in the current season, one scheduled match
// in each sign-up state — not yet open, open (with a waiting list), closed
// but uncomposed, and closed with the roster already built ahead of kick-off.
func generateMatches(
	group models.Group,
	teams []models.Team,
	roster []models.Player,
	matchService *services.MatchService,
	matchPlayerService *services.MatchPlayerService,
	registrationService *services.MatchRegistrationService,
	voteService *services.MatchVoteService,
) {
	now := time.Now()
	currentSeasonStartYear := now.Year()
	if now.Month() < time.September {
		currentSeasonStartYear--
	}

	for _, startYear := range []int{currentSeasonStartYear - 3, currentSeasonStartYear - 2, currentSeasonStartYear - 1} {
		for _, date := range randomSundaysInSeason(startYear, 2) {
			createPlayedUnscheduledMatch(matchService, matchPlayerService, group.ID, teams, roster, date)
		}
	}

	generateRecentComposedMatch(group, teams, roster, matchService, matchPlayerService, voteService)

	createOpenSignupsMatch(matchService, registrationService, group.ID, roster, now.AddDate(0, 0, 10))
	createNotYetOpenMatch(matchService, group.ID, now.AddDate(0, 0, 20))
	createClosedUncomposedMatch(matchService, registrationService, group.ID, roster, now.AddDate(0, 0, 14))
	createClosedComposedMatch(matchService, registrationService, matchPlayerService, group.ID, teams, roster, now.AddDate(0, 0, 7))

	log.Printf("group %q: generated a full spread of matches (%d roster members)", group.Name, len(roster))
}

// generateRecentComposedMatch creates the one recently-played, composed-
// roster scheduled match (plus its Man of the Match votes) that generateMatches
// also creates — factored out so `-recent-only` can regenerate just this part
// without duplicating everything else, e.g. after a bad run left this one
// match's kick-off too far in the past for the voting window to still be open.
func generateRecentComposedMatch(
	group models.Group,
	teams []models.Team,
	roster []models.Player,
	matchService *services.MatchService,
	matchPlayerService *services.MatchPlayerService,
	voteService *services.MatchVoteService,
) {
	// Comfortably inside the 2-calendar-day Man of the Match voting window
	// (VotingWindowError closes it the day after the match, at midnight) —
	// -2 days sits right on the boundary and can already read as closed
	// depending on the time of day the script runs.
	recentKickoff := time.Now().AddDate(0, 0, -1)
	recentMatchID, composed := createComposedScheduledMatch(matchService, matchPlayerService, group.ID, teams, roster, recentKickoff, true)
	castRandomVotes(voteService, recentMatchID, roster, composed)
	log.Printf("group %q: recreated the recently-played composed match with votes", group.Name)
}

// randomSundaysInSeason returns n distinct Sundays (oldest first) drawn from
// the season starting September 1st of startYear.
func randomSundaysInSeason(startYear, n int) []time.Time {
	start := time.Date(startYear, time.September, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(startYear+1, time.August, 31, 0, 0, 0, 0, time.Local)
	d := start
	for d.Weekday() != time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	var all []time.Time
	for !d.After(end) {
		all = append(all, d)
		d = d.AddDate(0, 0, 7)
	}
	rand.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	if n > len(all) {
		n = len(all)
	}
	picked := all[:n]
	sort.Slice(picked, func(i, j int) bool { return picked[i].Before(picked[j]) })
	return picked
}

func capAt(n, max int) int {
	if n > max {
		return max
	}
	return n
}

// splitRoster shuffles roster, keeps at most maxSize of it, and splits what's
// left in half — the same mechanical split cmd/seed's own match generation
// uses.
func splitRoster(roster []models.Player, maxSize int) [][]models.Player {
	shuffled := make([]models.Player, len(roster))
	copy(shuffled, roster)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	if maxSize > 0 && maxSize < len(shuffled) {
		shuffled = shuffled[:maxSize]
	}
	mid := len(shuffled) / 2
	return [][]models.Player{shuffled[:mid], shuffled[mid:]}
}

// weightedGoals mimics a realistic scoreline: mostly 0, occasionally a goal,
// rarely a brace or more — the same distribution cmd/seed uses.
func weightedGoals() int {
	switch r := rand.Intn(10); {
	case r < 6:
		return 0
	case r < 9:
		return 1
	default:
		return rand.Intn(3) + 2
	}
}

func createPlayedUnscheduledMatch(matchService *services.MatchService, matchPlayerService *services.MatchPlayerService, groupID uuid.UUID, teams []models.Team, roster []models.Player, date time.Time) {
	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.DateOf(date)}, groupID)
	if err != nil {
		log.Fatalf("failed to create unscheduled match: %v", err)
	}
	split := splitRoster(roster, capAt(len(roster), 16))
	for i, team := range teams {
		for _, p := range split[i] {
			mp := &models.MatchPlayer{MatchID: matchID, TeamID: team.ID, PlayerID: p.ID, GoalsScored: weightedGoals()}
			if err := matchPlayerService.CreateMatchPlayer(mp); err != nil {
				log.Fatalf("failed to assign player to match: %v", err)
			}
		}
	}
}

// createComposedScheduledMatch creates a scheduled match whose roster is
// already built (bypassing the sign-up list entirely, the same way an admin
// composing teams manually would end up), returning which players ended up
// on it so the caller can cast Man of the Match votes for them.
func createComposedScheduledMatch(matchService *services.MatchService, matchPlayerService *services.MatchPlayerService, groupID uuid.UUID, teams []models.Team, roster []models.Player, kickoff time.Time, played bool) (uuid.UUID, []uuid.UUID) {
	opensAt := kickoff.AddDate(0, 0, -7)
	maxPlayers := 10 + rand.Intn(7)
	spec := services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}
	matchID, err := matchService.CreateMatch(spec, groupID)
	if err != nil {
		log.Fatalf("failed to create scheduled match: %v", err)
	}
	split := splitRoster(roster, capAt(len(roster), maxPlayers))
	var composed []uuid.UUID
	for i, team := range teams {
		for _, p := range split[i] {
			goals := 0
			if played {
				goals = weightedGoals()
			}
			mp := &models.MatchPlayer{MatchID: matchID, TeamID: team.ID, PlayerID: p.ID, GoalsScored: goals}
			if err := matchPlayerService.CreateMatchPlayer(mp); err != nil {
				log.Fatalf("failed to assign player to match: %v", err)
			}
			composed = append(composed, p.ID)
		}
	}
	return matchID, composed
}

func castRandomVotes(voteService *services.MatchVoteService, matchID uuid.UUID, voters []models.Player, candidates []uuid.UUID) {
	if len(candidates) == 0 || len(voters) == 0 {
		return
	}
	shuffled := make([]models.Player, len(voters))
	copy(shuffled, voters)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	for _, voter := range shuffled[:capAt(len(shuffled), 5)] {
		candidate := candidates[rand.Intn(len(candidates))]
		if candidate == voter.ID {
			continue // skip rather than retry — a missed vote is harmless here
		}
		if err := voteService.Vote(matchID, voter.ID, candidate); err != nil {
			log.Printf("warning: failed to cast a Man of the Match vote: %v", err)
		}
	}
}

func createOpenSignupsMatch(matchService *services.MatchService, registrationService *services.MatchRegistrationService, groupID uuid.UUID, roster []models.Player, kickoff time.Time) {
	opensAt := time.Now().AddDate(0, 0, -1)
	maxPlayers := 8 + rand.Intn(5) // deliberately smallish, to make a waiting list likely
	spec := services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}
	matchID, err := matchService.CreateMatch(spec, groupID)
	if err != nil {
		log.Fatalf("failed to create open-signups match: %v", err)
	}
	shuffled := make([]models.Player, len(roster))
	copy(shuffled, roster)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	for _, p := range shuffled[:capAt(len(shuffled), maxPlayers+3)] {
		if err := registrationService.Register(matchID, p.ID); err != nil {
			log.Printf("warning: failed to register a player for open sign-ups: %v", err)
		}
	}
}

func createNotYetOpenMatch(matchService *services.MatchService, groupID uuid.UUID, kickoff time.Time) {
	opensAt := time.Now().AddDate(0, 0, 3)
	maxPlayers := 12 + rand.Intn(5)
	spec := services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}
	if _, err := matchService.CreateMatch(spec, groupID); err != nil {
		log.Fatalf("failed to create not-yet-open match: %v", err)
	}
}

func createClosedUncomposedMatch(matchService *services.MatchService, registrationService *services.MatchRegistrationService, groupID uuid.UUID, roster []models.Player, kickoff time.Time) {
	opensAt := time.Now().AddDate(0, 0, -5)
	maxPlayers := 10 + rand.Intn(7)
	spec := services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}
	matchID, err := matchService.CreateMatch(spec, groupID)
	if err != nil {
		log.Fatalf("failed to create closed-uncomposed match: %v", err)
	}
	shuffled := make([]models.Player, len(roster))
	copy(shuffled, roster)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	for _, p := range shuffled[:capAt(len(shuffled), maxPlayers)] {
		if err := registrationService.Register(matchID, p.ID); err != nil {
			log.Printf("warning: failed to register a player before closing: %v", err)
		}
	}
	if err := registrationService.CloseRegistrations(matchID, groupID); err != nil {
		log.Fatalf("failed to close registrations: %v", err)
	}
}

func createClosedComposedMatch(matchService *services.MatchService, registrationService *services.MatchRegistrationService, matchPlayerService *services.MatchPlayerService, groupID uuid.UUID, teams []models.Team, roster []models.Player, kickoff time.Time) {
	opensAt := time.Now().AddDate(0, 0, -6)
	maxPlayers := 10 + rand.Intn(7)
	spec := services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}
	matchID, err := matchService.CreateMatch(spec, groupID)
	if err != nil {
		log.Fatalf("failed to create closed-composed match: %v", err)
	}
	shuffled := make([]models.Player, len(roster))
	copy(shuffled, roster)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	signedUp := shuffled[:capAt(len(shuffled), maxPlayers)]
	for _, p := range signedUp {
		if err := registrationService.Register(matchID, p.ID); err != nil {
			log.Printf("warning: failed to register a player before closing: %v", err)
		}
	}
	if err := registrationService.CloseRegistrations(matchID, groupID); err != nil {
		log.Fatalf("failed to close registrations: %v", err)
	}
	split := splitRoster(signedUp, len(signedUp))
	for i, team := range teams {
		for _, p := range split[i] {
			mp := &models.MatchPlayer{MatchID: matchID, TeamID: team.ID, PlayerID: p.ID, GoalsScored: 0}
			if err := matchPlayerService.CreateMatchPlayer(mp); err != nil {
				log.Fatalf("failed to assign player to match: %v", err)
			}
		}
	}
}
