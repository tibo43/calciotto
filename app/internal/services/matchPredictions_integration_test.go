package services_test

import (
	"errors"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

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
