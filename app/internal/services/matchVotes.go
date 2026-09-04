package services

import (
	"errors"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// ErrCannotVoteForSelf rejects a vote where the voter and the candidate are
	// the same player. There is no legitimate self-nomination for "best player
	// on the pitch", unlike e.g. self-registering for a match.
	ErrCannotVoteForSelf = errors.New("you cannot vote for yourself for man of the match")

	// ErrVotedForPlayerNotOnRoster rejects a vote for a player with no
	// MatchPlayer row for this match, on either team. This is what scopes
	// voting to actual participants — the voter can be anyone in the group
	// (see MatchVoteService.Vote's own comment), but the candidate must have
	// actually played.
	ErrVotedForPlayerNotOnRoster = errors.New("the player voted for is not on this match's roster")
)

// MatchVoteService owns a match's Man of the Match votes and the tally
// derived from them. Like MatchRegistrationService, it deliberately does not
// check that the voter belongs to the match's group — that is authorization,
// enforced one layer up by RequireGroupMembershipByMatchPathParam on the
// route.
//
// There is no admin close/reopen concept here, unlike MatchRegistrationService:
// voting is always open for any match with a composed roster, for as long as
// the match exists. There is no product need yet for a windowing mechanism —
// a match already played does not stop being judgeable, and reopening a
// closed vote would just be extra machinery protecting nothing.
type MatchVoteService struct {
	DB *gorm.DB
}

func NewMatchVoteService(db *gorm.DB) *MatchVoteService {
	return &MatchVoteService{DB: db}
}

// Vote casts or replaces voterID's vote for votedForID in matchID.
//
// Unlike MatchRegistrationService.Register, this is deliberately an upsert
// rather than a one-shot action refused on a duplicate (ErrAlreadyRegistered):
// a player changing their mind about who deserves MOTM before or after the
// final whistle is the ordinary case, not a mistake — so a second call from
// the same voter replaces their first vote instead of being rejected or
// creating a second row. The composite unique index on (match_id, voter_id)
// is what makes "at most one *current* vote per voter per match" the real
// guarantee under a race; the transaction below is what makes the visible
// behaviour update-in-place rather than a rejection.
//
// Voter eligibility is deliberately broader than "played in the match": any
// group member can judge who the best player was, including a sub who did
// not get on or a member who only watched — enforced by the route's
// membership check, not by this service. The candidate, on the other hand,
// must actually be on the roster (ErrVotedForPlayerNotOnRoster) — that is
// what keeps "who was the best player" scoped to people who could plausibly
// have been.
func (s *MatchVoteService) Vote(matchID, voterID, votedForID uuid.UUID) error {
	if _, err := s.findMatch(matchID); err != nil {
		return err
	}
	if voterID == votedForID {
		return ErrCannotVoteForSelf
	}

	onRoster, err := s.playerOnRoster(matchID, votedForID)
	if err != nil {
		return err
	}
	if !onRoster {
		return ErrVotedForPlayerNotOnRoster
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.MatchVote
		err := tx.Where("match_id = ? AND voter_id = ?", matchID, voterID).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return tx.Create(&models.MatchVote{MatchID: matchID, VoterID: voterID, VotedForID: votedForID}).Error
		case err != nil:
			return err
		default:
			return tx.Model(&existing).Update("voted_for_id", votedForID).Error
		}
	})
}

// Unvote removes voterID's vote for matchID, if any.
//
// It is a no-op success when the caller had not voted — the same "a retried
// request must not fail for nothing" philosophy as
// MatchRegistrationService.ReopenRegistrations — rather than a 404, and
// deliberately does not check that matchID names a real match first: deleting
// a vote that was never cast, for a match that may not even exist any more,
// is still exactly nothing to do.
func (s *MatchVoteService) Unvote(matchID, voterID uuid.UUID) error {
	return s.DB.Where("match_id = ? AND voter_id = ?", matchID, voterID).Delete(&models.MatchVote{}).Error
}

// ListVotes returns matchID's tally — every candidate with at least one vote,
// ordered by vote count desc then name asc — plus which player callerID has
// voted for, if any.
//
// The tally is a GORM query-builder aggregation (Group/Count plus one join to
// players for names), not raw SQL: this is a single join with a GROUP BY,
// exactly the case the query builder is for, unlike the multi-table
// flatten/reconstruct matches.go needs raw SQL for (see CLAUDE.md).
func (s *MatchVoteService) ListVotes(matchID, callerID uuid.UUID) (*models.MatchVoteSummary, error) {
	if _, err := s.findMatch(matchID); err != nil {
		return nil, err
	}

	tally, err := s.tallyForMatch(matchID)
	if err != nil {
		return nil, err
	}

	summary := &models.MatchVoteSummary{Tally: tally}

	var myVote models.MatchVote
	err = s.DB.Where("match_id = ? AND voter_id = ?", matchID, callerID).First(&myVote).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// The caller has not voted; MyVoteFor stays nil.
	case err != nil:
		return nil, err
	default:
		summary.MyVoteFor = &myVote.VotedForID
	}

	return summary, nil
}

// tallyForMatch is ListVotes' aggregation, factored out so
// TallyVotesForMatches can reuse the exact same shape for many matches at
// once (see its own comment for why that one is a separate query rather than
// a loop over this).
func (s *MatchVoteService) tallyForMatch(matchID uuid.UUID) ([]models.MatchVoteTally, error) {
	var tally []models.MatchVoteTally
	if err := s.DB.Model(&models.MatchVote{}).
		Select("match_votes.voted_for_id AS player_id, players.name AS name, COUNT(*) AS votes").
		Joins("JOIN players ON players.id = match_votes.voted_for_id").
		Where("match_votes.match_id = ?", matchID).
		Group("match_votes.voted_for_id, players.name").
		Order("votes DESC, players.name ASC").
		Scan(&tally).Error; err != nil {
		return nil, err
	}
	if tally == nil {
		tally = []models.MatchVoteTally{}
	}
	return tally, nil
}

// TallyVotesForMatches returns the tally for every match in matchIDs, keyed
// by match id — what StandingsService.GetMotmStandings needs for a whole
// group's matches in one round-trip rather than one query per match. A match
// with no votes at all is simply absent from the map.
func (s *MatchVoteService) TallyVotesForMatches(matchIDs []uuid.UUID) (map[uuid.UUID][]models.MatchVoteTally, error) {
	byMatch := make(map[uuid.UUID][]models.MatchVoteTally)
	if len(matchIDs) == 0 {
		return byMatch, nil
	}

	var rows []struct {
		MatchID  uuid.UUID
		PlayerID uuid.UUID
		Name     string
		Votes    int
	}
	if err := s.DB.Model(&models.MatchVote{}).
		Select("match_votes.match_id AS match_id, match_votes.voted_for_id AS player_id, players.name AS name, COUNT(*) AS votes").
		Joins("JOIN players ON players.id = match_votes.voted_for_id").
		Where("match_votes.match_id IN (?)", matchIDs).
		Group("match_votes.match_id, match_votes.voted_for_id, players.name").
		Order("votes DESC, players.name ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		byMatch[row.MatchID] = append(byMatch[row.MatchID], models.MatchVoteTally{
			PlayerID: row.PlayerID,
			Name:     row.Name,
			Votes:    row.Votes,
		})
	}
	return byMatch, nil
}

// playerOnRoster reports whether playerID has a MatchPlayer row for matchID,
// on either team.
func (s *MatchVoteService) playerOnRoster(matchID, playerID uuid.UUID) (bool, error) {
	var count int64
	if err := s.DB.Model(&models.MatchPlayer{}).
		Where("match_id = ? AND player_id = ?", matchID, playerID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// findMatch loads a match by id alone, the same way
// MatchRegistrationService.findMatch does and for the same reason: the
// player-facing routes reach a match through middleware that already
// authorized the caller against the group carrying it, so there is no group
// to scope against here — only existence needs checking.
func (s *MatchVoteService) findMatch(matchID uuid.UUID) (*models.Match, error) {
	var match models.Match
	if err := s.DB.Where("id = ?", matchID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return &match, nil
}

// ComputeMotmWinners is the entire "who won this match's award" rule, kept as
// a pure function of an already-computed tally — the same split as
// ComputeRegistrationPositions/ComputePointsStandings, so it is unit-testable
// without a database.
//
// Ties are inclusive by product decision: if several candidates are tied for
// the most votes, every one of them gets the award for that match, rather
// than an arbitrary tie-break picking one. tally may be in any order and is
// not mutated; an empty tally (nobody voted, or nobody voted for the
// eventual roster — e.g. every vote was later withdrawn) returns no winners.
func ComputeMotmWinners(tally []models.MatchVoteTally) []uuid.UUID {
	max := 0
	for _, candidate := range tally {
		if candidate.Votes > max {
			max = candidate.Votes
		}
	}
	if max == 0 {
		return nil
	}

	var winners []uuid.UUID
	for _, candidate := range tally {
		if candidate.Votes == max {
			winners = append(winners, candidate.PlayerID)
		}
	}
	return winners
}
