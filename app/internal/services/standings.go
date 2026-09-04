package services

import (
	"sort"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type StandingsService struct {
	MatchService      *MatchService
	MembershipService *GroupMembershipService
	PlayerService     *PlayerService
	VoteService       *MatchVoteService
}

func NewStandingsService(db *gorm.DB, membershipService *GroupMembershipService) *StandingsService {
	return &StandingsService{
		MatchService:      NewMatchService(db),
		MembershipService: membershipService,
		PlayerService:     NewPlayerService(db),
		VoteService:       NewMatchVoteService(db),
	}
}

// The GetMatchesDetails calls below deliberately pass an empty season: these
// callers load every match of the group and do their own FilterMatchesBySeason
// pass (or, for GetSeasons, need every season there is). Pushing the season
// down would either double-filter or, in GetSeasons' case, collapse the list to
// the one season asked for.
func (s *StandingsService) GetPointsStandings(groupID uuid.UUID, season string) ([]models.PointsStandingRow, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID, "")
	if err != nil {
		return nil, err
	}
	rows := ComputePointsStandings(FilterMatchesBySeason(matches, season))

	currentMembers, err := s.currentMemberIDs(groupID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].IsMember = currentMembers[rows[i].PlayerID]
	}
	return rows, nil
}

func (s *StandingsService) GetScorers(groupID uuid.UUID, season string) ([]models.ScorerRow, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID, "")
	if err != nil {
		return nil, err
	}
	rows := ComputeScorers(FilterMatchesBySeason(matches, season))

	currentMembers, err := s.currentMemberIDs(groupID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].IsMember = currentMembers[rows[i].PlayerID]
	}
	return rows, nil
}

// GetMotmStandings ranks a group's players by how many matches' Man of the
// Match award they have won, following the exact same shape as
// GetPointsStandings/GetScorers: load the group's matches, season-filter in
// Go, run a pure Compute* function, then tag IsMember as a post-processing
// step. The one difference is the extra data source — a match's votes are
// not part of MatchWithDetails at all (the same reason RegistrationCount
// isn't: see CLAUDE.md's "flatten/reconstruct pattern"), so they are loaded
// separately, once, for every match in the (already season-filtered) result.
func (s *StandingsService) GetMotmStandings(groupID uuid.UUID, season string) ([]models.MotmStandingRow, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID, "")
	if err != nil {
		return nil, err
	}
	matches = FilterMatchesBySeason(matches, season)

	matchIDs := make([]uuid.UUID, len(matches))
	for i, match := range matches {
		matchIDs[i] = match.ID
	}
	votesByMatch, err := s.VoteService.TallyVotesForMatches(matchIDs)
	if err != nil {
		return nil, err
	}

	rows := ComputeMotmStandings(matches, votesByMatch)

	currentMembers, err := s.currentMemberIDs(groupID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].IsMember = currentMembers[rows[i].PlayerID]
	}
	return rows, nil
}

// currentMemberIDs returns the set of player IDs currently belonging to
// groupID, as a post-processing step for GetPointsStandings/GetScorers: a
// player who has since been removed from the group still keeps their
// historical rows (standings are computed from match history, ComputePoints
// Standings/ComputeScorers know nothing about membership), so this is what
// lets those two callers tag each row IsMember: true/false without touching
// the pure aggregation functions themselves. GetPlayerProfile deliberately
// does not call this — every group it iterates is, by construction, one the
// caller currently belongs to, so tagging IsMember there would be trivially
// true for every row and isn't worth the extra query.
func (s *StandingsService) currentMemberIDs(groupID uuid.UUID) (map[uuid.UUID]bool, error) {
	members, err := s.MembershipService.GetPlayersByGroupID(groupID)
	if err != nil {
		return nil, err
	}
	ids := make(map[uuid.UUID]bool, len(members))
	for _, member := range members {
		ids[member.ID] = true
	}
	return ids, nil
}

// GetSeasons lists the seasons a group actually has matches in. Seasons have
// no table of their own — they're derived from the match dates already loaded
// for the group, so no extra query is needed.
func (s *StandingsService) GetSeasons(groupID uuid.UUID) ([]string, error) {
	matches, err := s.MatchService.GetMatchesDetails(groupID, "")
	if err != nil {
		return nil, err
	}
	return ComputeSeasons(matches), nil
}

// GetPlayerProfile returns one player's record across every group they belong
// to. Overall and PerGroup are computed from the same matches, so they always
// agree: each group's matches are loaded and season-filtered exactly like the
// group-scoped standings endpoints do, then ComputePointsStandings/
// ComputeMotmStandings each run once per group for PerGroup and once over the
// concatenation for Overall — the same "load once, compute at two scopes"
// shape GetMotmStandings itself uses, just repeated per group here instead of
// once for a single one.
//
// A group the player is a member of but has never played a match in still
// gets a PerGroup entry, zeroed — being in the group is what puts the row
// there, not having played. Same for Overall: a player with no matches at all
// gets a zero row, not an error.
func (s *StandingsService) GetPlayerProfile(playerID uuid.UUID, season string) (*models.PlayerProfileStats, error) {
	groups, err := s.MembershipService.GetGroupsByPlayerID(playerID)
	if err != nil {
		return nil, err
	}
	// GetGroupsByPlayerID has no ORDER BY, so sort here — otherwise the
	// profile's per-group table can reshuffle between two identical requests.
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })

	// The zero row still carries the player's identity, so a profile with no
	// matches anywhere is a complete row of zeroes rather than a blank.
	zeroRow := models.PointsStandingRow{PlayerID: playerID}
	if player, err := s.PlayerService.GetPlayerByID(playerID); err == nil {
		zeroRow.Name = player.Name
	}

	profile := &models.PlayerProfileStats{
		Overall:  models.PlayerOverallStanding{PointsStandingRow: zeroRow},
		PerGroup: make([]models.PlayerGroupStanding, 0, len(groups)),
	}

	var allMatches []models.MatchWithDetails
	// Match ids are globally unique, so merging each group's votesByMatch
	// into one combined map for the Overall computation below can never
	// collide two different groups' entries under the same key.
	allVotesByMatch := make(map[uuid.UUID][]models.MatchVoteTally)
	for _, group := range groups {
		matches, err := s.MatchService.GetMatchesDetails(group.ID, "")
		if err != nil {
			return nil, err
		}
		matches = FilterMatchesBySeason(matches, season)
		allMatches = append(allMatches, matches...)

		row := zeroRow
		if found := findPointsRow(ComputePointsStandings(matches), playerID); found != nil {
			row = *found
		}

		matchIDs := make([]uuid.UUID, len(matches))
		for i, match := range matches {
			matchIDs[i] = match.ID
		}
		votesByMatch, err := s.VoteService.TallyVotesForMatches(matchIDs)
		if err != nil {
			return nil, err
		}
		for id, tally := range votesByMatch {
			allVotesByMatch[id] = tally
		}

		profile.PerGroup = append(profile.PerGroup, models.PlayerGroupStanding{
			PointsStandingRow: row,
			GroupID:           group.ID,
			GroupName:         group.Name,
			MotmAwards:        findMotmAwards(ComputeMotmStandings(matches, votesByMatch), playerID),
		})
	}

	if found := findPointsRow(ComputePointsStandings(allMatches), playerID); found != nil {
		profile.Overall.PointsStandingRow = *found
	}
	profile.Overall.MotmAwards = findMotmAwards(ComputeMotmStandings(allMatches, allVotesByMatch), playerID)
	return profile, nil
}

// findPointsRow picks one player out of a computed standings table.
func findPointsRow(rows []models.PointsStandingRow, playerID uuid.UUID) *models.PointsStandingRow {
	for i := range rows {
		if rows[i].PlayerID == playerID {
			return &rows[i]
		}
	}
	return nil
}

// findMotmAwards picks one player's award count out of a computed Man of the
// Match leaderboard, defaulting to 0 — a player with no awards is simply
// absent from the computed rows, not an error.
func findMotmAwards(rows []models.MotmStandingRow, playerID uuid.UUID) int {
	for _, row := range rows {
		if row.PlayerID == playerID {
			return row.Awards
		}
	}
	return 0
}

// FilterMatchesBySeason keeps only the matches belonging to season (a label as
// produced by models.SeasonOf). An empty season means "no filtering", so
// callers that don't scope by season keep the previous behaviour.
func FilterMatchesBySeason(matches []models.MatchWithDetails, season string) []models.MatchWithDetails {
	if season == "" {
		return matches
	}
	filtered := make([]models.MatchWithDetails, 0, len(matches))
	for _, match := range matches {
		if models.SeasonOf(match.Date) == season {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

// ComputeSeasons returns the sorted, deduplicated season labels covered by an
// already-loaded set of matches. Like the Compute* functions below, it is a
// pure function of its input: which matches to consider is the caller's call.
//
// It counts *every* match, including a scheduled one nobody has played yet — so
// scheduling a match into a future season makes that season appear in
// GET /standings/seasons before a ball has been kicked. That is a decision, not
// an oversight, and there is a test pinning it: GET /matches/details is filtered
// by whichever season the frontend has selected, so dropping a future season
// from this list would make the upcoming match unreachable in the UI, which is
// strictly worse than one extra dropdown entry. The known cost — the frontend
// preselects the most recent season, so scheduling into a new season shifts the
// default view — is accepted for now and tracked as a separate improvement.
func ComputeSeasons(matches []models.MatchWithDetails) []string {
	seen := make(map[string]bool, len(matches))
	seasons := make([]string, 0, len(matches))
	for _, match := range matches {
		season := models.SeasonOf(match.Date)
		if seen[season] {
			continue
		}
		seen[season] = true
		seasons = append(seasons, season)
	}
	sort.Strings(seasons)
	return seasons
}

// ComputePointsStandings aggregates win/draw/loss points (3/1/0) per player over
// an already-loaded set of matches. It is a pure function of its input — data
// selection (which matches to include, e.g. scoped to a group) is the caller's
// responsibility, so that scope can change later without touching this logic.
func ComputePointsStandings(matches []models.MatchWithDetails) []models.PointsStandingRow {
	type acc struct {
		name                                    string
		played, won, drawn, lost, goals, points int
	}
	stats := make(map[uuid.UUID]*acc)

	rowFor := func(id uuid.UUID, name string) *acc {
		row, ok := stats[id]
		if !ok {
			row = &acc{name: name}
			stats[id] = row
		}
		return row
	}

	applyResult := func(team models.TeamWithPlayers, result string) {
		for _, player := range team.Players {
			row := rowFor(player.ID, player.Name)
			row.played++
			row.goals += player.GoalsScored
			switch result {
			case "won":
				row.won++
				row.points += 3
			case "drawn":
				row.drawn++
				row.points++
			default:
				row.lost++
			}
		}
	}

	for _, match := range matches {
		// Only count matches where both sides have a full roster.
		if len(match.Teams) != 2 {
			continue
		}
		teamA, teamB := match.Teams[0], match.Teams[1]
		if len(teamA.Players) == 0 || len(teamB.Players) == 0 {
			continue
		}

		switch {
		case teamA.Score > teamB.Score:
			applyResult(teamA, "won")
			applyResult(teamB, "lost")
		case teamB.Score > teamA.Score:
			applyResult(teamB, "won")
			applyResult(teamA, "lost")
		default:
			applyResult(teamA, "drawn")
			applyResult(teamB, "drawn")
		}
	}

	rows := make([]models.PointsStandingRow, 0, len(stats))
	for id, row := range stats {
		rows = append(rows, models.PointsStandingRow{
			PlayerID: id,
			Name:     row.name,
			Played:   row.played,
			Won:      row.won,
			Drawn:    row.drawn,
			Lost:     row.lost,
			GoalsFor: row.goals,
			Points:   row.points,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Points != rows[j].Points {
			return rows[i].Points > rows[j].Points
		}
		if rows[i].GoalsFor != rows[j].GoalsFor {
			return rows[i].GoalsFor > rows[j].GoalsFor
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// ComputeScorers totals goals scored per player over an already-loaded set of
// matches. Like ComputePointsStandings, it takes the match slice as input so
// scoping (e.g. to a group) is the caller's concern, not this function's.
func ComputeScorers(matches []models.MatchWithDetails) []models.ScorerRow {
	type acc struct {
		name          string
		played, goals int
	}
	stats := make(map[uuid.UUID]*acc)

	for _, match := range matches {
		for _, team := range match.Teams {
			for _, player := range team.Players {
				row, ok := stats[player.ID]
				if !ok {
					row = &acc{name: player.Name}
					stats[player.ID] = row
				}
				row.played++
				row.goals += player.GoalsScored
			}
		}
	}

	rows := make([]models.ScorerRow, 0, len(stats))
	for id, row := range stats {
		rows = append(rows, models.ScorerRow{
			PlayerID: id,
			Name:     row.name,
			Played:   row.played,
			Goals:    row.goals,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Goals != rows[j].Goals {
			return rows[i].Goals > rows[j].Goals
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// ComputeMotmStandings ranks players by how many matches' Man of the Match
// award they have won, over an already-loaded set of matches and their vote
// tallies keyed by match id (as produced by
// MatchVoteService.TallyVotesForMatches). Like ComputePointsStandings/
// ComputeScorers, it is a pure function of its inputs — which matches and
// which votes to consider is entirely the caller's concern, so scoping (by
// group, by season) never has to touch this logic.
//
// A tie for the most votes in a match is not resolved here either: every
// player ComputeMotmWinners returns for that match has their award count
// incremented, so a three-way tie in one match increments three rows rather
// than picking a winner.
func ComputeMotmStandings(matches []models.MatchWithDetails, votesByMatch map[uuid.UUID][]models.MatchVoteTally) []models.MotmStandingRow {
	type acc struct {
		name   string
		awards int
	}
	stats := make(map[uuid.UUID]*acc)

	for _, match := range matches {
		tally := votesByMatch[match.ID]
		if len(tally) == 0 {
			continue
		}

		// Names come from the tally itself (players.name, resolved at query
		// time) rather than from the match roster: a tally entry only exists
		// for a player who actually received a vote.
		nameByID := make(map[uuid.UUID]string, len(tally))
		for _, candidate := range tally {
			nameByID[candidate.PlayerID] = candidate.Name
		}

		for _, winner := range ComputeMotmWinners(tally) {
			row, ok := stats[winner]
			if !ok {
				row = &acc{name: nameByID[winner]}
				stats[winner] = row
			}
			row.awards++
		}
	}

	rows := make([]models.MotmStandingRow, 0, len(stats))
	for id, row := range stats {
		rows = append(rows, models.MotmStandingRow{PlayerID: id, Name: row.name, Awards: row.awards})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Awards != rows[j].Awards {
			return rows[i].Awards > rows[j].Awards
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}
