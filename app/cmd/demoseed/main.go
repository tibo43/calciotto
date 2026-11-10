// cmd/demoseed wipes the local dev database and rebuilds a complete demo
// dataset whose every date is computed from the moment it runs — re-running
// it later "moves the clock forward": yesterday's match is always yesterday,
// the next composed match is always three days out, the open sign-up list
// always opened yesterday. It is a self-contained alternative to the
// cmd/seed + cmd/richseed pair, which stack on top of each other and drift
// as the calendar moves on.
//
// What it builds, per group:
//   - last season: a handful of plain, already-played unscheduled matches,
//     so the season selector and the Elo history behind the odds aren't empty;
//   - this season: six weekly scheduled matches already played (results,
//     sign-up lists, Man of the Match votes, settled predictions), the most
//     recent one yesterday so its voting window is still open;
//   - upcoming: one composed match with predictions open (+3 days), one with
//     sign-ups closed but no teams yet (+5), one with sign-ups open and a
//     waiting list (+7), and one whose sign-ups haven't opened yet (+14).
//
// thibaut is left without a prediction, a vote, or a sign-up on the matches
// still open to them, so logging in as him shows every action as available.
//
// Wherever a window is still open, the data goes through the real services
// (Register, Vote, Predict), exactly the code path the API uses. Where the
// window has already closed — a sign-up list or prediction for a match that
// kicked off last week — those services would rightly refuse, so the rows
// are written directly, timestamped as if made before kick-off.
//
// It TRUNCATEs every application table. It refuses to run when
// DATABASE_URL is set, which is how production is reached (see
// pkg/database/connect.go): this is a local dev tool only.
package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"
	_ "time/tzdata"

	"app/internal/models"
	"app/internal/services"
	"app/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const demoAuthSecret = "demoseed-only-unused-jwt-secret"

// demoPassword is shared by every account, the same convention as cmd/seed.
const demoPassword = "user1234"

var playerNames = []string{
	"thibaut", "matthias", "manfredi", "damien", "vincent", "pierre",
	"anthony", "jacopo", "mattheo", "ryan", "connor", "marcos",
	"lucas", "hugo", "enzo", "nathan",
}

type groupSpec struct {
	name    string
	teams   [2]services.TeamSpec
	admin   string
	members []string // besides the admin
	kickoff [2]int   // hour, minute
}

// thibaut is the admin of Default and a plain member of the other two, so
// one login sees both the admin and the member side of the app.
var groupSpecs = []groupSpec{
	{
		name:    "Default",
		teams:   services.DefaultTeamSpecs,
		admin:   "thibaut",
		members: []string{"matthias", "manfredi", "damien", "vincent", "pierre", "anthony", "jacopo", "mattheo", "ryan", "connor", "marcos", "lucas", "hugo"},
		kickoff: [2]int{20, 30},
	},
	{
		name:    "Les Copains du Mardi",
		teams:   [2]services.TeamSpec{{Name: "Les Rouges", Colour: "red"}, {Name: "Les Bleus", Colour: "blue"}},
		admin:   "matthias",
		members: []string{"thibaut", "damien", "pierre", "jacopo", "ryan", "marcos", "lucas", "enzo", "nathan", "hugo"},
		kickoff: [2]int{19, 0},
	},
	{
		name:    "Foot du Vendredi",
		teams:   [2]services.TeamSpec{{Name: "Panthers", Colour: "black"}, {Name: "Tigers", Colour: "orange"}},
		admin:   "vincent",
		members: []string{"thibaut", "manfredi", "anthony", "mattheo", "connor", "enzo", "nathan", "hugo", "lucas"},
		kickoff: [2]int{21, 0},
	},
}

// The schedule, in days relative to today.
var (
	playedOffsets       = []int{-36, -29, -22, -15, -8, -1}
	composedOffset      = 3
	closedOffset        = 5
	openOffset          = 7
	notYetOpenOffset    = 14
	notYetOpenOpensDays = 4
)

var rng *rand.Rand

type env struct {
	db            *gorm.DB
	loc           *time.Location
	now           time.Time
	matches       *services.MatchService
	matchPlayers  *services.MatchPlayerService
	registrations *services.MatchRegistrationService
	votes         *services.MatchVoteService
	predictions   *services.MatchPredictionService
	playerIDs     map[string]uuid.UUID
}

type group struct {
	spec    groupSpec
	id      uuid.UUID
	teams   []models.Team
	members []uuid.UUID // admin first
}

func main() {
	randSeed := flag.Int64("rand-seed", time.Now().UnixNano(), "random seed for goals and picks; reuse a logged value to get the same scores again")
	flag.Parse()

	if os.Getenv("DATABASE_URL") != "" {
		log.Fatal("DATABASE_URL is set — refusing to wipe what is probably production; demoseed is for the local dev database only")
	}
	rng = rand.New(rand.NewSource(*randSeed))
	log.Printf("random seed: %d", *randSeed)

	loc, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		log.Fatalf("failed to load Europe/Paris: %v", err)
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	if err := wipe(db); err != nil {
		log.Fatalf("failed to wipe the database: %v", err)
	}

	matchService := services.NewMatchService(db)
	e := &env{
		db:            db,
		loc:           loc,
		now:           time.Now().In(loc),
		matches:       matchService,
		matchPlayers:  services.NewMatchPlayerService(db),
		registrations: services.NewMatchRegistrationService(db),
		votes:         services.NewMatchVoteService(db),
		predictions:   services.NewMatchPredictionService(db, matchService),
		playerIDs:     make(map[string]uuid.UUID, len(playerNames)),
	}

	createPlayers(e)
	for _, spec := range groupSpecs {
		g := createGroup(e, spec)
		buildGroupHistory(e, g)
		log.Printf("group %q ready (%d members, admin %s)", spec.name, len(g.members), spec.admin)
	}

	log.Printf("done — every account is <name>@mail.com / %s", demoPassword)
}

// wipe empties every application table in one statement; CASCADE covers
// the foreign keys (match_players → matches/players/teams) whatever order
// they were created in.
func wipe(db *gorm.DB) error {
	return db.Exec(`TRUNCATE match_predictions, match_votes, match_registrations, match_players,
		matches, password_reset_tokens, group_memberships, teams, groups, players CASCADE`).Error
}

func createPlayers(e *env) {
	playerService := services.NewPlayerService(e.db)
	authService := services.NewAuthService(e.db, demoAuthSecret)
	for _, name := range playerNames {
		id, err := playerService.CreatePlayer(name)
		if err != nil {
			log.Fatalf("failed to create player %q: %v", name, err)
		}
		if err := authService.Signup(id, name+"@mail.com", demoPassword); err != nil {
			log.Fatalf("failed to give %q an account: %v", name, err)
		}
		e.playerIDs[name] = id
	}
	log.Printf("created %d players", len(playerNames))
}

func createGroup(e *env, spec groupSpec) *group {
	created, err := services.NewGroupService(e.db).CreateGroup(spec.name, spec.teams)
	if err != nil {
		log.Fatalf("failed to create group %q: %v", spec.name, err)
	}
	membershipService := services.NewGroupMembershipService(e.db)
	g := &group{spec: spec, id: created.ID}

	add := func(name, role string) {
		id, ok := e.playerIDs[name]
		if !ok {
			log.Fatalf("group %q lists unknown player %q", spec.name, name)
		}
		if err := membershipService.AddPlayerToGroupWithRole(created.ID, id, role); err != nil {
			log.Fatalf("failed to add %q to %q: %v", name, spec.name, err)
		}
		g.members = append(g.members, id)
	}
	add(spec.admin, models.RoleAdmin)
	for _, name := range spec.members {
		add(name, models.RoleMember)
	}

	g.teams, err = services.NewTeamService(e.db).GetTeamsByGroupID(created.ID)
	if err != nil || len(g.teams) != 2 {
		log.Fatalf("failed to load the teams of %q: %v", spec.name, err)
	}
	return g
}

func buildGroupHistory(e *env, g *group) {
	for _, date := range lastSeasonDates(e.now) {
		createUnscheduledPlayed(e, g, date)
	}
	for _, offset := range playedOffsets {
		createScheduledPlayed(e, g, e.kickoff(g, offset))
	}
	createComposedUpcoming(e, g, e.kickoff(g, composedOffset))
	createClosedUncomposed(e, g, e.kickoff(g, closedOffset))
	createOpenSignups(e, g, e.kickoff(g, openOffset))
	createNotYetOpen(e, g, e.kickoff(g, notYetOpenOffset))
}

// kickoff is today+offsetDays at the group's usual time, Paris time.
func (e *env) kickoff(g *group, offsetDays int) time.Time {
	day := e.now.AddDate(0, 0, offsetDays)
	return time.Date(day.Year(), day.Month(), day.Day(), g.spec.kickoff[0], g.spec.kickoff[1], 0, 0, e.loc)
}

// lastSeasonDates returns four Sundays spread over the season before the
// current one (seasons start September 1st, see models.SeasonOf).
func lastSeasonDates(now time.Time) []time.Time {
	startYear := now.Year() - 1
	if now.Month() < time.September {
		startYear--
	}
	var dates []time.Time
	for _, days := range []int{30, 90, 160, 240} {
		d := time.Date(startYear, time.September, 1, 0, 0, 0, 0, now.Location()).AddDate(0, 0, days)
		for d.Weekday() != time.Sunday {
			d = d.AddDate(0, 0, 1)
		}
		dates = append(dates, d)
	}
	return dates
}

func createUnscheduledPlayed(e *env, g *group, date time.Time) {
	matchID, err := e.matches.CreateMatch(services.MatchSpec{Date: models.DateOf(date)}, g.id)
	if err != nil {
		log.Fatalf("failed to create an unscheduled match: %v", err)
	}
	composeRoster(e, g, matchID, pickPlayers(g.members, 12), true)
}

// createScheduledPlayed is a match that went through the whole flow and is
// now over: sign-ups, closed list, composed teams, a result, Man of the
// Match votes and settled predictions. Every window is closed by now, so
// the rows are written directly with timestamps from before kick-off.
func createScheduledPlayed(e *env, g *group, kickoff time.Time) {
	opensAt := kickoff.AddDate(0, 0, -7)
	maxPlayers := 12
	matchID := createScheduled(e, g, kickoff, opensAt, maxPlayers)

	signedUp := pickPlayers(g.members, maxPlayers+2)
	for i, playerID := range signedUp {
		insert(e, &models.MatchRegistration{MatchID: matchID, PlayerID: playerID, CreatedAt: opensAt.Add(time.Duration(i+1) * time.Hour)})
	}
	closedAt := kickoff.Add(-24 * time.Hour)
	if err := e.db.Model(&models.Match{}).Where("id = ?", matchID).Update("registrations_closed_at", closedAt).Error; err != nil {
		log.Fatalf("failed to close sign-ups: %v", err)
	}

	roster := signedUp[:min(maxPlayers, len(signedUp))]
	composeRoster(e, g, matchID, roster, true)

	// Yesterday's match is still inside the Man of the Match window, so its
	// votes go through the service — and thibaut's is left for him to cast.
	votingOpen := kickoff.After(e.now.AddDate(0, 0, -2))
	for _, voter := range pickPlayers(roster, len(roster)*2/3) {
		candidate := pickOther(roster, voter)
		switch {
		case !votingOpen:
			insert(e, &models.MatchVote{MatchID: matchID, VoterID: voter, VotedForID: candidate, CreatedAt: kickoff.Add(2 * time.Hour)})
		case voter != e.playerIDs["thibaut"]:
			if err := e.votes.Vote(matchID, voter, candidate); err != nil {
				log.Fatalf("failed to cast a Man of the Match vote: %v", err)
			}
		}
	}

	for _, predictor := range pickPlayers(g.members, len(g.members)*2/3) {
		insert(e, &models.MatchPrediction{MatchID: matchID, PlayerID: predictor, PredictedTeamID: randomPick(g), CreatedAt: opensAt.Add(36 * time.Hour)})
	}
}

// createComposedUpcoming: sign-ups closed, teams composed, kick-off ahead —
// the one state where predictions are open, so they go through Predict.
func createComposedUpcoming(e *env, g *group, kickoff time.Time) {
	maxPlayers := 12
	matchID := createScheduled(e, g, kickoff, e.now.AddDate(0, 0, -4), maxPlayers)
	signedUp := register(e, matchID, pickPlayers(g.members, maxPlayers+1))
	closeSignups(e, g, matchID)
	composeRoster(e, g, matchID, signedUp[:min(maxPlayers, len(signedUp))], false)

	for _, predictor := range pickPlayers(without(g.members, e.playerIDs["thibaut"]), len(g.members)/2) {
		if err := e.predictions.Predict(matchID, predictor, randomPick(g)); err != nil {
			log.Fatalf("failed to cast a prediction: %v", err)
		}
	}
}

// createClosedUncomposed: the admin closed the list but hasn't built the
// teams yet — the moment "Fill teams from sign-ups" is for.
func createClosedUncomposed(e *env, g *group, kickoff time.Time) {
	matchID := createScheduled(e, g, kickoff, e.now.AddDate(0, 0, -2), 12)
	register(e, matchID, pickPlayers(g.members, 11))
	closeSignups(e, g, matchID)
}

// createOpenSignups: more sign-ups than places, so there is a waiting list;
// thibaut isn't on it, so Participate is there to click.
func createOpenSignups(e *env, g *group, kickoff time.Time) {
	maxPlayers := 8
	matchID := createScheduled(e, g, kickoff, e.now.AddDate(0, 0, -1), maxPlayers)
	register(e, matchID, pickPlayers(without(g.members, e.playerIDs["thibaut"]), maxPlayers+2))
}

func createNotYetOpen(e *env, g *group, kickoff time.Time) {
	createScheduled(e, g, kickoff, e.now.AddDate(0, 0, notYetOpenOpensDays), 12)
}

func createScheduled(e *env, g *group, kickoff, opensAt time.Time, maxPlayers int) uuid.UUID {
	spec := services.MatchSpec{ScheduledAt: &kickoff, RegistrationOpensAt: &opensAt, MaxPlayers: &maxPlayers}
	matchID, err := e.matches.CreateMatch(spec, g.id)
	if err != nil {
		log.Fatalf("failed to create a scheduled match on %s: %v", kickoff.Format(time.RFC3339), err)
	}
	return matchID
}

func register(e *env, matchID uuid.UUID, players []uuid.UUID) []uuid.UUID {
	for _, playerID := range players {
		if err := e.registrations.Register(matchID, playerID); err != nil {
			log.Fatalf("failed to sign a player up: %v", err)
		}
	}
	return players
}

func closeSignups(e *env, g *group, matchID uuid.UUID) {
	if err := e.registrations.CloseRegistrations(matchID, g.id); err != nil {
		log.Fatalf("failed to close sign-ups: %v", err)
	}
}

// composeRoster splits players across the group's two teams, with goals
// (and the odd own goal) when the match has been played.
func composeRoster(e *env, g *group, matchID uuid.UUID, players []uuid.UUID, played bool) {
	for i, playerID := range players {
		mp := &models.MatchPlayer{MatchID: matchID, TeamID: g.teams[i%2].ID, PlayerID: playerID}
		if played {
			mp.GoalsScored = weightedGoals()
			if rng.Intn(25) == 0 {
				mp.OwnGoals = 1
			}
		}
		if err := e.matchPlayers.CreateMatchPlayer(mp); err != nil {
			log.Fatalf("failed to put a player on a team: %v", err)
		}
	}
}

func insert(e *env, row interface{}) {
	if err := e.db.Create(row).Error; err != nil {
		log.Fatalf("failed to insert %T: %v", row, err)
	}
}

// pickPlayers returns n of players in random order (all of them if n is
// larger), leaving the input untouched.
func pickPlayers(players []uuid.UUID, n int) []uuid.UUID {
	shuffled := append([]uuid.UUID(nil), players...)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled[:min(n, len(shuffled))]
}

func pickOther(players []uuid.UUID, not uuid.UUID) uuid.UUID {
	others := without(players, not)
	return others[rng.Intn(len(others))]
}

func without(players []uuid.UUID, not uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(players))
	for _, p := range players {
		if p != not {
			out = append(out, p)
		}
	}
	return out
}

// randomPick is a prediction: one of the two teams, or nil for a draw.
func randomPick(g *group) *uuid.UUID {
	switch rng.Intn(5) {
	case 0:
		return nil
	case 1, 2:
		return &g.teams[0].ID
	default:
		return &g.teams[1].ID
	}
}

// weightedGoals: mostly 0, sometimes 1, rarely more — same as cmd/seed.
func weightedGoals() int {
	switch r := rng.Intn(10); {
	case r < 6:
		return 0
	case r < 9:
		return 1
	default:
		return rng.Intn(3) + 2
	}
}
