package services_test

import (
	"errors"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"
	"app/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// predictionEnv is the fixture every test below starts from: a scheduled
// match in a fresh group, with its own two team ids on hand — composing the
// roster (or not) is left to each test, since "is the roster composed yet"
// is exactly the thing several of these tests need to vary.
type predictionEnv struct {
	tx          *gorm.DB
	predictions *services.MatchPredictionService
	matches     *services.MatchService
	players     *services.PlayerService
	groupID     uuid.UUID
	matchID     uuid.UUID
	teamAID     uuid.UUID
	teamBID     uuid.UUID
}

func newPredictionEnv(t *testing.T, label string, kickoff time.Time) *predictionEnv {
	t.Helper()

	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	teamService := services.NewTeamService(tx)
	matchService := services.NewMatchService(tx)

	group, err := groupService.CreateGroup("Zzz Predictions "+label, services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}

	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		t.Fatalf("failed to load group's teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected CreateGroup to create exactly 2 teams, got %d", len(teams))
	}

	opensAt := kickoff.Add(-24 * time.Hour)
	maxPlayers := 10
	matchID, err := matchService.CreateMatch(services.MatchSpec{
		Date:                models.DateOf(kickoff),
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &opensAt,
		MaxPlayers:          &maxPlayers,
	}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}

	return &predictionEnv{
		tx:          tx,
		predictions: services.NewMatchPredictionService(tx, matchService),
		matches:     matchService,
		players:     services.NewPlayerService(tx),
		groupID:     group.ID,
		matchID:     matchID,
		teamAID:     teams[0].ID,
		teamBID:     teams[1].ID,
	}
}

// composeRoster puts one freshly created player on each team, so there is a
// full, derivable-result-capable roster to predict against.
func (e *predictionEnv) composeRoster(t *testing.T) {
	t.Helper()
	alice, err := e.players.CreatePlayer("Zzz Prediction Alice " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	bob, err := e.players.CreatePlayer("Zzz Prediction Bob " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := e.matches.UpdateMatch(e.matchID, e.groupID, []models.TeamWithPlayers{
		{ID: e.teamAID, Players: []models.PlayerCustom{{ID: alice}}},
		{ID: e.teamBID, Players: []models.PlayerCustom{{ID: bob}}},
	}); err != nil {
		t.Fatalf("failed to compose the roster: %v", err)
	}
}

func TestMatchPredictionService_PredictBeforeRosterComposed(t *testing.T) {
	env := newPredictionEnv(t, "not-composed", time.Now().Add(48*time.Hour))
	predictor, err := env.players.CreatePlayer("Zzz Predictor " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create predictor: %v", err)
	}

	err = env.predictions.Predict(env.matchID, predictor, &env.teamAID)
	if !errors.Is(err, services.ErrPredictionsNotAvailable) {
		t.Errorf("Predict before the roster is composed = %v, want ErrPredictionsNotAvailable", err)
	}
}

func TestMatchPredictionService_PredictOnUnscheduledMatch(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	matchService := services.NewMatchService(tx)
	playerService := services.NewPlayerService(tx)
	predictionService := services.NewMatchPredictionService(tx, matchService)

	group, err := groupService.CreateGroup("Zzz Predictions Unscheduled", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("failed to create group: %v", err)
	}
	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.Date(time.Now())}, group.ID)
	if err != nil {
		t.Fatalf("failed to create match: %v", err)
	}
	predictor, err := playerService.CreatePlayer("Zzz Predictor " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create predictor: %v", err)
	}

	err = predictionService.Predict(matchID, predictor, nil)
	if !errors.Is(err, services.ErrPredictionsNotAvailable) {
		t.Errorf("Predict on an unscheduled match = %v, want ErrPredictionsNotAvailable", err)
	}
}

func TestMatchPredictionService_PredictClosedAfterKickoff(t *testing.T) {
	// A kick-off already in the past: CreateMatch deliberately allows
	// backfilling a match this way (see MatchSpec.validate's own comment),
	// so this is a legitimate state to test against, not a contrived one.
	env := newPredictionEnv(t, "past-kickoff", time.Now().Add(-time.Hour))
	env.composeRoster(t)
	predictor, err := env.players.CreatePlayer("Zzz Predictor " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create predictor: %v", err)
	}

	err = env.predictions.Predict(env.matchID, predictor, &env.teamAID)
	if !errors.Is(err, services.ErrPredictionsClosed) {
		t.Errorf("Predict after kick-off = %v, want ErrPredictionsClosed", err)
	}
}

func TestMatchPredictionService_PredictRejectsATeamNotInTheMatch(t *testing.T) {
	env := newPredictionEnv(t, "wrong-team", time.Now().Add(48*time.Hour))
	env.composeRoster(t)
	predictor, err := env.players.CreatePlayer("Zzz Predictor " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create predictor: %v", err)
	}

	someOtherTeamID := uuid.New()
	err = env.predictions.Predict(env.matchID, predictor, &someOtherTeamID)
	if !errors.Is(err, services.ErrPredictedTeamNotInMatch) {
		t.Errorf("Predict with a team id not in this match = %v, want ErrPredictedTeamNotInMatch", err)
	}
}

func TestMatchPredictionService_PredictAndListRoundTrip(t *testing.T) {
	env := newPredictionEnv(t, "roundtrip", time.Now().Add(48*time.Hour))
	env.composeRoster(t)
	predictor, err := env.players.CreatePlayer("Zzz Predictor " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create predictor: %v", err)
	}

	if err := env.predictions.Predict(env.matchID, predictor, &env.teamAID); err != nil {
		t.Fatalf("Predict returned error: %v", err)
	}

	summary, err := env.predictions.ListPredictions(env.matchID, predictor)
	if err != nil {
		t.Fatalf("ListPredictions returned error: %v", err)
	}
	if len(summary.Entries) != 1 {
		t.Fatalf("expected exactly 1 entry, got %d: %+v", len(summary.Entries), summary.Entries)
	}
	if summary.MyPrediction == nil || summary.MyPrediction.PredictedTeamID == nil || *summary.MyPrediction.PredictedTeamID != env.teamAID {
		t.Errorf("MyPrediction = %+v, want a prediction for team A (%s)", summary.MyPrediction, env.teamAID)
	}
	// Roster is fully composed, so odds must be present — with no prior
	// history both sides are an even match (see
	// standings_test.go's own testEvenOdds for the same figures pinned
	// directly against ComputeMatchOdds).
	if summary.Odds == nil {
		t.Fatal("Odds is nil, want a value now that the roster is fully composed")
	}
	if summary.Odds.TeamAOdds <= 1 || summary.Odds.DrawOdds <= 1 || summary.Odds.TeamBOdds <= 1 {
		t.Errorf("Odds = %+v, want every multiplier above 1", summary.Odds)
	}

	// Changing your mind is an upsert, not a second row.
	if err := env.predictions.Predict(env.matchID, predictor, &env.teamBID); err != nil {
		t.Fatalf("Predict (changing prediction) returned error: %v", err)
	}
	summary, err = env.predictions.ListPredictions(env.matchID, predictor)
	if err != nil {
		t.Fatalf("ListPredictions (after change) returned error: %v", err)
	}
	if len(summary.Entries) != 1 {
		t.Fatalf("expected the change to replace, not add, a row — got %d entries: %+v", len(summary.Entries), summary.Entries)
	}
	if summary.MyPrediction == nil || summary.MyPrediction.PredictedTeamID == nil || *summary.MyPrediction.PredictedTeamID != env.teamBID {
		t.Errorf("MyPrediction after change = %+v, want team B (%s)", summary.MyPrediction, env.teamBID)
	}

	// A draw is represented as a nil PredictedTeamID — round-trip it
	// through an actual update (not just a create) to a nullable column,
	// the one GORM behaviour worth verifying against a real database rather
	// than trusting by inspection.
	if err := env.predictions.Predict(env.matchID, predictor, nil); err != nil {
		t.Fatalf("Predict (draw) returned error: %v", err)
	}
	summary, err = env.predictions.ListPredictions(env.matchID, predictor)
	if err != nil {
		t.Fatalf("ListPredictions (after draw) returned error: %v", err)
	}
	if summary.MyPrediction == nil {
		t.Fatal("MyPrediction is nil after predicting a draw, want a present entry with a nil team id")
	}
	if summary.MyPrediction.PredictedTeamID != nil {
		t.Errorf("MyPrediction.PredictedTeamID = %v after predicting a draw, want nil", *summary.MyPrediction.PredictedTeamID)
	}
}

func TestMatchPredictionService_ListPredictionsNoOddsWhenNotComposed(t *testing.T) {
	env := newPredictionEnv(t, "no-odds", time.Now().Add(48*time.Hour))
	// Deliberately not composing the roster.

	summary, err := env.predictions.ListPredictions(env.matchID, uuid.New())
	if err != nil {
		t.Fatalf("ListPredictions returned error: %v", err)
	}
	if summary.Odds != nil {
		t.Errorf("Odds = %+v, want nil when the roster isn't fully composed yet", summary.Odds)
	}
	if summary.MyPrediction != nil {
		t.Errorf("MyPrediction = %+v, want nil — the caller in this test never predicted", summary.MyPrediction)
	}
}

func TestMatchPredictionService_ListPredictionsUnknownMatchNotFound(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)
	matchService := services.NewMatchService(tx)
	predictionService := services.NewMatchPredictionService(tx, matchService)

	_, err := predictionService.ListPredictions(uuid.New(), uuid.New())
	if !errors.Is(err, services.ErrMatchNotFound) {
		t.Errorf("ListPredictions for an unknown match = %v, want ErrMatchNotFound", err)
	}
}

// TestDeleteMatch_Integration_RemovesPredictions: like votes and sign-ups,
// MatchPrediction declares no association on Match, so DeleteMatch has to
// remove a match's predictions itself.
func TestDeleteMatch_Integration_RemovesPredictions(t *testing.T) {
	env := newPredictionEnv(t, "delete-match", time.Now().Add(48*time.Hour))
	env.composeRoster(t)
	predictor, err := env.players.CreatePlayer("Zzz Predictor " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create predictor: %v", err)
	}
	if err := env.predictions.Predict(env.matchID, predictor, &env.teamAID); err != nil {
		t.Fatalf("Predict returned error: %v", err)
	}

	if err := env.matches.DeleteMatch(env.matchID, env.groupID); err != nil {
		t.Fatalf("DeleteMatch returned error: %v", err)
	}

	var remaining int64
	if err := env.tx.Model(&models.MatchPrediction{}).Where("match_id = ?", env.matchID).
		Count(&remaining).Error; err != nil {
		t.Fatalf("failed to count predictions: %v", err)
	}
	if remaining != 0 {
		t.Errorf("%d predictions survived the match deletion, want 0", remaining)
	}
}

// TestUpdateMatch_Integration_RecordsResultOnlyAfterKickoff: composing a
// roster ahead of kick-off is not recording a result, so it must not settle
// anything.
func TestUpdateMatch_Integration_RecordsResultOnlyAfterKickoff(t *testing.T) {
	env := newPredictionEnv(t, "before-kickoff-save", time.Now().Add(48*time.Hour))
	env.composeRoster(t)

	var match models.Match
	if err := env.tx.First(&match, "id = ?", env.matchID).Error; err != nil {
		t.Fatalf("failed to reload match: %v", err)
	}
	if match.ResultRecordedAt != nil {
		t.Errorf("ResultRecordedAt = %v after a save before kick-off, want nil", match.ResultRecordedAt)
	}
}

// TestPredictionStandings_Integration_SettledOnlyOnceResultRecorded: a match
// that has kicked off still settles nothing until an admin saves its score;
// that save settles it straight away, and a later correction keeps the
// first recording time.
func TestPredictionStandings_Integration_SettledOnlyOnceResultRecorded(t *testing.T) {
	env := newPredictionEnv(t, "settle-on-record", time.Now().Add(-2*time.Hour))
	alice, err := env.players.CreatePlayer("Zzz Settle Alice " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	bob, err := env.players.CreatePlayer("Zzz Settle Bob " + uuid.NewString())
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	membership := services.NewGroupMembershipService(env.tx)
	for _, id := range []uuid.UUID{alice, bob} {
		if err := membership.AddPlayerToGroup(env.groupID, id); err != nil {
			t.Fatalf("failed to add member: %v", err)
		}
	}
	// The roster and the prediction are written directly: going through
	// UpdateMatch here would itself record the result (kick-off has passed),
	// and Predict refuses after kick-off.
	for _, mp := range []models.MatchPlayer{
		{MatchID: env.matchID, TeamID: env.teamAID, PlayerID: alice},
		{MatchID: env.matchID, TeamID: env.teamBID, PlayerID: bob},
	} {
		if err := env.tx.Create(&mp).Error; err != nil {
			t.Fatalf("failed to compose roster: %v", err)
		}
	}
	if err := env.tx.Create(&models.MatchPrediction{MatchID: env.matchID, PlayerID: alice, PredictedTeamID: &env.teamAID}).Error; err != nil {
		t.Fatalf("failed to create prediction: %v", err)
	}

	standings := services.NewStandingsService(env.tx, membership)
	rows, err := standings.GetPredictionStandings(env.groupID, "")
	if err != nil {
		t.Fatalf("GetPredictionStandings returned error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("standings = %+v before the result was recorded, want none", rows)
	}

	saveScore := func(aliceGoals int) {
		t.Helper()
		if err := env.matches.UpdateMatch(env.matchID, env.groupID, []models.TeamWithPlayers{
			{ID: env.teamAID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: aliceGoals}}},
			{ID: env.teamBID, Players: []models.PlayerCustom{{ID: bob}}},
		}); err != nil {
			t.Fatalf("UpdateMatch returned error: %v", err)
		}
	}
	saveScore(2)

	rows, err = standings.GetPredictionStandings(env.groupID, "")
	if err != nil {
		t.Fatalf("GetPredictionStandings returned error: %v", err)
	}
	if len(rows) != 1 || rows[0].PlayerID != alice || rows[0].Correct != 1 {
		t.Fatalf("standings = %+v after the result was recorded, want alice with 1 correct prediction", rows)
	}

	var first models.Match
	if err := env.tx.First(&first, "id = ?", env.matchID).Error; err != nil || first.ResultRecordedAt == nil {
		t.Fatalf("ResultRecordedAt not set after a save past kick-off (err %v)", err)
	}
	saveScore(3)
	var second models.Match
	if err := env.tx.First(&second, "id = ?", env.matchID).Error; err != nil {
		t.Fatalf("failed to reload match: %v", err)
	}
	if second.ResultRecordedAt == nil || !second.ResultRecordedAt.Equal(*first.ResultRecordedAt) {
		t.Errorf("ResultRecordedAt moved from %v to %v on a correction, want the first recording kept", first.ResultRecordedAt, second.ResultRecordedAt)
	}
}

// TestListPredictions_Integration_OthersHiddenUntilKickoff: while predictions
// are open a caller sees only their own pick (and how many were made); at
// kick-off everyone's is revealed.
func TestListPredictions_Integration_OthersHiddenUntilKickoff(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kickoff  time.Time
		revealed bool
	}{
		{"before kick-off", time.Now().Add(48 * time.Hour), false},
		{"after kick-off", time.Now().Add(-time.Hour), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newPredictionEnv(t, "reveal-"+tc.name, tc.kickoff)
			me, err := env.players.CreatePlayer("Zzz Reveal Me " + uuid.NewString())
			if err != nil {
				t.Fatalf("failed to create player: %v", err)
			}
			other, err := env.players.CreatePlayer("Zzz Reveal Other " + uuid.NewString())
			if err != nil {
				t.Fatalf("failed to create player: %v", err)
			}
			// Written directly so the same setup works on both sides of
			// kick-off (Predict refuses after it).
			for _, p := range []models.MatchPrediction{
				{MatchID: env.matchID, PlayerID: me, PredictedTeamID: &env.teamAID},
				{MatchID: env.matchID, PlayerID: other, PredictedTeamID: &env.teamBID},
			} {
				if err := env.tx.Create(&p).Error; err != nil {
					t.Fatalf("failed to create prediction: %v", err)
				}
			}

			summary, err := env.predictions.ListPredictions(env.matchID, me)
			if err != nil {
				t.Fatalf("ListPredictions returned error: %v", err)
			}
			if summary.Count != 2 || summary.Revealed != tc.revealed {
				t.Errorf("Count/Revealed = %d/%v, want 2/%v", summary.Count, summary.Revealed, tc.revealed)
			}
			wantEntries := 1
			if tc.revealed {
				wantEntries = 2
			}
			if len(summary.Entries) != wantEntries {
				t.Fatalf("got %d entries, want %d: %+v", len(summary.Entries), wantEntries, summary.Entries)
			}
			for _, entry := range summary.Entries {
				if !tc.revealed && entry.PlayerID != me {
					t.Errorf("another player's pick leaked before kick-off: %+v", entry)
				}
			}
			if summary.MyPrediction == nil || summary.MyPrediction.PlayerID != me {
				t.Errorf("MyPrediction = %+v, want the caller's own", summary.MyPrediction)
			}
		})
	}
}

// TestBackfillResultRecordedAt_Integration: a composed scheduled match played
// before ResultRecordedAt existed is marked recorded at startup, while an
// uncomposed one, and a composed one kicking off after the release (whose
// NULL means the score really isn't in yet), are left alone.
func TestBackfillResultRecordedAt_Integration(t *testing.T) {
	beforeRelease := time.Date(2026, time.October, 4, 20, 0, 0, 0, time.UTC)
	// Already kicked off, but after the release: its NULL is a real
	// "score not saved yet".
	afterRelease := time.Now().Add(-5 * time.Minute)

	old := newPredictionEnv(t, "backfill-old", beforeRelease)
	old.composeRoster(t)
	oldEmpty := newPredictionEnv(t, "backfill-old-empty", beforeRelease)
	recent := newPredictionEnv(t, "backfill-recent", afterRelease)
	recent.composeRoster(t)

	// Composing after kick-off records the result; undo that to reproduce a
	// row saved before the column existed. Each env has its own transaction.
	for _, env := range []*predictionEnv{old, oldEmpty, recent} {
		if err := env.tx.Model(&models.Match{}).Where("id = ?", env.matchID).
			Update("result_recorded_at", nil).Error; err != nil {
			t.Fatalf("failed to clear result_recorded_at: %v", err)
		}
		if err := database.BackfillResultRecordedAt(env.tx); err != nil {
			t.Fatalf("BackfillResultRecordedAt: %v", err)
		}
	}

	recordedAt := func(env *predictionEnv) *time.Time {
		t.Helper()
		var match models.Match
		if err := env.tx.First(&match, "id = ?", env.matchID).Error; err != nil {
			t.Fatalf("failed to reload match: %v", err)
		}
		return match.ResultRecordedAt
	}

	if got := recordedAt(old); got == nil || !got.Equal(beforeRelease) {
		t.Errorf("old composed match: ResultRecordedAt = %v, want its kick-off %v", got, beforeRelease)
	}
	if got := recordedAt(oldEmpty); got != nil {
		t.Errorf("old uncomposed match: ResultRecordedAt = %v, want nil", got)
	}
	if got := recordedAt(recent); got != nil {
		t.Errorf("match after the release: ResultRecordedAt = %v, want nil", got)
	}
}
