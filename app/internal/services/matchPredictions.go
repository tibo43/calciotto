package services

import (
	"errors"
	"time"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrPredictionsNotAvailable covers both "this match has no schedule" and
	// "the roster isn't fully composed on both sides yet" — predicting a
	// winner only makes sense once there's a kick-off to predict ahead of
	// and two actual rosters to pick between. Unlike ErrPredictionsClosed,
	// this is never a timing-dependent state a retry could fix on its own —
	// an unscheduled match will never open a prediction window at all.
	ErrPredictionsNotAvailable = errors.New("predictions are not available for this match")

	// ErrPredictionsClosed is returned once kick-off has passed — the same
	// "state conflict, not a malformed request" posture as
	// ErrRegistrationsClosed/ErrVotingClosed, since the identical request
	// would have succeeded a moment earlier.
	ErrPredictionsClosed = errors.New("predictions are closed for this match")

	// ErrPredictedTeamNotInMatch rejects a prediction naming a team id that
	// isn't one of this specific match's own two teams.
	ErrPredictedTeamNotInMatch = errors.New("the predicted team is not one of this match's teams")
)

// PredictionWindowError reports why match is not currently accepting a
// prediction at instant now, or nil when it is — a pure function mirroring
// RegistrationWindowError/VotingWindowError's own split from their thin
// service callers.
//
// rosterComposed is passed in rather than derived here: answering it means
// loading the match's own teams/players, a database read this function
// deliberately stays free of, the same reasoning ComputeRegistrationPositions
// takes an already-known maxPlayers rather than querying for it itself.
func PredictionWindowError(match models.Match, rosterComposed bool, now time.Time) error {
	if !match.IsScheduled() || !rosterComposed {
		return ErrPredictionsNotAvailable
	}
	if !now.Before(*match.ScheduledAt) {
		return ErrPredictionsClosed
	}
	return nil
}

// rosterFullyComposed is stricter than the frontend's own teamsAreComposed
// (true once *either* team has a player, which is enough to start showing a
// roster UI): predicting a winner between two teams needs both sides to
// actually exist, or "the other team" is nothing in particular yet.
func rosterFullyComposed(teams []models.TeamWithPlayers) bool {
	return len(teams) == 2 && len(teams[0].Players) > 0 && len(teams[1].Players) > 0
}

// MatchPredictionService owns a match's win/draw/loss predictions and the
// Elo-derived odds they're settled against. Unlike MatchVoteService/
// MatchRegistrationService, it holds a *MatchService rather than querying
// match_players directly: deriving odds needs a match's full composed
// roster (to average each side's rating) and the group's whole match
// history (to know those ratings as of right before this match — see
// EloRatingsBeforeMatch), both of which only MatchService's flatten/
// reconstruct read path produces.
//
// Like the other match-scoped services, it does not check that the caller
// belongs to the match's group — that is authorization, enforced one layer
// up by RequireGroupMembershipByMatchPathParam. It also does not require the
// predictor to be on the match's own roster, unlike Man of the Match voting:
// judging who played well requires having played, guessing who wins does
// not — any group member may predict.
type MatchPredictionService struct {
	DB           *gorm.DB
	MatchService *MatchService
}

func NewMatchPredictionService(db *gorm.DB, matchService *MatchService) *MatchPredictionService {
	return &MatchPredictionService{DB: db, MatchService: matchService}
}

// Predict casts or replaces playerID's prediction for matchID — an upsert,
// the same "changing your mind is the ordinary case" reasoning as
// MatchVoteService.Vote, not a one-shot action refused on a repeat.
// predictedTeamID nil predicts a draw.
func (s *MatchPredictionService) Predict(matchID, playerID uuid.UUID, predictedTeamID *uuid.UUID) error {
	match, err := s.findMatch(matchID)
	if err != nil {
		return err
	}

	details, err := s.matchDetails(*match)
	if err != nil {
		return err
	}
	composed := rosterFullyComposed(details.Teams)

	if err := PredictionWindowError(*match, composed, time.Now()); err != nil {
		return err
	}

	if predictedTeamID != nil {
		onTeam := false
		for _, team := range details.Teams {
			if team.ID == *predictedTeamID {
				onTeam = true
				break
			}
		}
		if !onTeam {
			return ErrPredictedTeamNotInMatch
		}
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.MatchPrediction
		err := tx.Where("match_id = ? AND player_id = ?", matchID, playerID).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return tx.Create(&models.MatchPrediction{MatchID: matchID, PlayerID: playerID, PredictedTeamID: predictedTeamID}).Error
		case err != nil:
			return err
		default:
			return tx.Model(&existing).Update("predicted_team_id", predictedTeamID).Error
		}
	})
}

// ListPredictions returns matchID's predictions plus the odds they were/are
// made against and the caller's own prediction, if any. Unlike
// MatchVoteService.ListVotes it is never gated on the prediction window —
// the whole picture (and, once played, the actual result) is the point once
// a match is done, the same "never gate the read" posture ListVotes itself
// already takes past its own window.
func (s *MatchPredictionService) ListPredictions(matchID, callerID uuid.UUID) (*models.MatchPredictionSummary, error) {
	match, err := s.findMatch(matchID)
	if err != nil {
		return nil, err
	}
	details, err := s.matchDetails(*match)
	if err != nil {
		return nil, err
	}

	var rows []struct {
		PlayerID        uuid.UUID
		Name            string
		PredictedTeamID *uuid.UUID
	}
	if err := s.DB.Model(&models.MatchPrediction{}).
		Select("match_predictions.player_id AS player_id, players.name AS name, match_predictions.predicted_team_id AS predicted_team_id").
		Joins("JOIN players ON players.id = match_predictions.player_id").
		Where("match_predictions.match_id = ?", matchID).
		Order("players.name ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	summary := &models.MatchPredictionSummary{Entries: make([]models.MatchPredictionEntry, 0, len(rows))}
	for _, row := range rows {
		entry := models.MatchPredictionEntry{PlayerID: row.PlayerID, Name: row.Name, PredictedTeamID: row.PredictedTeamID}
		summary.Entries = append(summary.Entries, entry)
		if row.PlayerID == callerID {
			summary.MyPrediction = &entry
		}
	}

	odds, err := s.oddsForMatch(*match, details)
	if err != nil {
		return nil, err
	}
	summary.Odds = odds

	return summary, nil
}

// TallyPredictionsForMatches returns every prediction entry for every match
// in matchIDs, keyed by match id — what
// StandingsService.GetPredictionStandings needs for a whole group's matches
// in one round-trip, mirroring MatchVoteService.TallyVotesForMatches
// exactly.
func (s *MatchPredictionService) TallyPredictionsForMatches(matchIDs []uuid.UUID) (map[uuid.UUID][]models.MatchPredictionEntry, error) {
	byMatch := make(map[uuid.UUID][]models.MatchPredictionEntry)
	if len(matchIDs) == 0 {
		return byMatch, nil
	}

	var rows []struct {
		MatchID         uuid.UUID
		PlayerID        uuid.UUID
		Name            string
		PredictedTeamID *uuid.UUID
	}
	if err := s.DB.Model(&models.MatchPrediction{}).
		Select("match_predictions.match_id AS match_id, match_predictions.player_id AS player_id, players.name AS name, match_predictions.predicted_team_id AS predicted_team_id").
		Joins("JOIN players ON players.id = match_predictions.player_id").
		Where("match_predictions.match_id IN (?)", matchIDs).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		byMatch[row.MatchID] = append(byMatch[row.MatchID], models.MatchPredictionEntry{
			PlayerID:        row.PlayerID,
			Name:            row.Name,
			PredictedTeamID: row.PredictedTeamID,
		})
	}
	return byMatch, nil
}

// oddsForMatch derives match's current odds from its own composed roster
// and the group's match history, or nil if the roster isn't fully composed
// on both sides yet (see rosterFullyComposed) — there is nothing to average
// a rating over on an empty side.
func (s *MatchPredictionService) oddsForMatch(match models.Match, details *models.MatchWithDetails) (*models.MatchOdds, error) {
	if !rosterFullyComposed(details.Teams) {
		return nil, nil
	}

	allMatches, err := s.MatchService.GetMatchesDetails(match.GroupID, "")
	if err != nil {
		return nil, err
	}

	ratings := EloRatingsBeforeMatch(allMatches, match.ID)
	odds := ComputeMatchOdds(
		teamAverageRating(details.Teams[0], ratings),
		teamAverageRating(details.Teams[1], ratings),
	)
	return &odds, nil
}

// matchDetails loads match's own teams/rosters, scoped to its own group —
// already known from findMatch, so there is no group ambiguity to resolve
// here the way MatchHandler.resolveScopedGroupID has to for a caller-facing
// route.
func (s *MatchPredictionService) matchDetails(match models.Match) (*models.MatchWithDetails, error) {
	return s.MatchService.GetMatchDetailsByID(match.ID, match.GroupID)
}

// findMatch loads a match by id alone, the same way
// MatchVoteService.findMatch/MatchRegistrationService.findMatch do and for
// the same reason: the player-facing routes reach a match through
// middleware that already authorized the caller against the group carrying
// it, so there is no group to scope against here — only existence needs
// checking.
func (s *MatchPredictionService) findMatch(matchID uuid.UUID) (*models.Match, error) {
	var match models.Match
	if err := s.DB.Where("id = ?", matchID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return &match, nil
}
