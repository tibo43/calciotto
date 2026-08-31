package services

import (
	"app/internal/models"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrMatchNotFound is returned when the requested match either does not
// exist at all, or exists but does not belong to the group it was requested
// under — the latter is the mechanism that stops a match ID from one group
// from being used to read another group's data, so it deliberately shares
// the same sentinel/404 as a plain missing match rather than leaking a
// distinct "forbidden" signal.
var ErrMatchNotFound = errors.New("match not found")

// The three sentinels below reject an incoherent *schedule* at creation time.
// A match with no scheduling at all stays perfectly valid — see MatchSpec.
var (
	// ErrIncompleteSchedule guards the all-or-nothing rule: a scheduled match
	// with no maximum number of players, or with no moment at which sign-ups
	// open, has no defined behaviour at all (nobody could ever register, or
	// nobody could ever land on the waiting list), so a half-filled schedule
	// is refused rather than stored and interpreted later.
	ErrIncompleteSchedule = errors.New("a scheduled match requires a kick-off time, a registration opening time and a maximum number of players")
	// ErrRegistrationOpensAfterKickoff rejects a window that never opens
	// before the match starts: kick-off is a hard backstop on registrations
	// (see RegistrationWindowError), so opening sign-ups at or after it would
	// create a match nobody can ever sign up for.
	ErrRegistrationOpensAfterKickoff = errors.New("registrations must open strictly before kick-off")
	// ErrInvalidMaxPlayers rejects a non-positive roster size, which would
	// put every single sign-up on the waiting list.
	ErrInvalidMaxPlayers = errors.New("maximum number of players must be greater than zero")
)

// MatchSpec is everything CreateMatch needs to know about the match being
// created, as a struct rather than a growing positional parameter list — the
// same call GroupService.CreateGroup makes with TeamSpec.
//
// The three scheduling fields are all-or-nothing: supply none of them for the
// original behaviour (an admin recording a match, scored right away), or all
// three to open the match to sign-ups. Date is only read in the unscheduled
// case: when ScheduledAt is set, Date is *derived* from it so the calendar day
// and the kick-off timestamp can never drift apart (see models.Match).
type MatchSpec struct {
	Date                models.Date
	ScheduledAt         *time.Time
	RegistrationOpensAt *time.Time
	MaxPlayers          *int
}

// IsScheduled mirrors models.Match.IsScheduled: a spec asks for a scheduled
// match iff it carries a kick-off time.
func (spec MatchSpec) IsScheduled() bool {
	return spec.ScheduledAt != nil
}

// validate enforces the schedule's internal coherence. Note what it
// deliberately does *not* check: a ScheduledAt in the past is accepted, since
// backfilling a match that already happened is legitimate and the unscheduled
// flow has always accepted any date.
func (spec MatchSpec) validate() error {
	if spec.ScheduledAt == nil && spec.RegistrationOpensAt == nil && spec.MaxPlayers == nil {
		return nil
	}
	if spec.ScheduledAt == nil || spec.RegistrationOpensAt == nil || spec.MaxPlayers == nil {
		return ErrIncompleteSchedule
	}
	if !spec.RegistrationOpensAt.Before(*spec.ScheduledAt) {
		return ErrRegistrationOpensAfterKickoff
	}
	if *spec.MaxPlayers <= 0 {
		return ErrInvalidMaxPlayers
	}
	return nil
}

type MatchService struct {
	DB *gorm.DB
}

func NewMatchService(db *gorm.DB) *MatchService {
	return &MatchService{DB: db}
}

// CreateMatch creates a match in groupID, optionally scheduled for a future
// kick-off with player sign-ups (see MatchSpec for the all-or-nothing rule).
//
// When the spec is scheduled, Date is derived from ScheduledAt's calendar day
// rather than taken from the spec: Date remains the field seasons, ordering and
// the existing JSON contract are built on, so it must stay a single write path
// — a caller able to supply both could store a match dated one day and kicking
// off on another. models.DateOf does that in the kick-off's own location, so an
// evening kick-off keeps the day the client meant.
func (s *MatchService) CreateMatch(spec MatchSpec, groupID uuid.UUID) (uuid.UUID, error) {
	if err := spec.validate(); err != nil {
		return uuid.Nil, err
	}

	match := &models.Match{
		GroupID:             groupID,
		Date:                spec.Date,
		ScheduledAt:         spec.ScheduledAt,
		RegistrationOpensAt: spec.RegistrationOpensAt,
		MaxPlayers:          spec.MaxPlayers,
	}
	if spec.IsScheduled() {
		match.Date = models.DateOf(*spec.ScheduledAt)
	}

	result := s.DB.Create(match)
	if result.Error != nil {
		return uuid.Nil, result.Error
	}
	return match.ID, nil
}

// GetGroupIDByMatchID answers "which group does this match belong to", and
// returns ErrMatchNotFound when the id names no match at all.
//
// It exists for the /matches/:id/... routes, whose path carries a *match* id
// and no group id: the authorization middleware has to derive the group from
// the match before it can check membership or admin rights (see
// RequireGroupMembershipByMatchPathParam). Deriving it is the whole point —
// letting the caller name the group instead, the way POST /matches does with
// its body, would let a member of group A act on a match in group B simply by
// supplying their own group id.
//
// It lives here rather than in the handler because touching the database is
// the service layer's job, and it selects only group_id: nothing upstream
// needs the rest of the row.
func (s *MatchService) GetGroupIDByMatchID(matchID uuid.UUID) (uuid.UUID, error) {
	var match models.Match
	if err := s.DB.Select("group_id").Where("id = ?", matchID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, ErrMatchNotFound
		}
		return uuid.Nil, err
	}
	return match.GroupID, nil
}

// GetMatchesDetails returns every match of a group, optionally narrowed to a
// single season. The season is applied in Go, via the same
// FilterMatchesBySeason the standings already use, rather than as a SQL
// predicate: a season is a derived label (models.SeasonOf), not a stored
// column, so there is nothing to filter on in the query. An empty season means
// "no filtering" — which is what every standings caller passes, since they run
// their own FilterMatchesBySeason pass on the result.
func (s *MatchService) GetMatchesDetails(groupID uuid.UUID, season string) ([]models.MatchWithDetails, error) {
	var rowsMatches []models.RowsMatchDetails
	var err error

	// Execute the SQL query and scan the results into a flat structure
	result := s.DB.Raw(`
        SELECT matches.id as match_id, matches.group_id as match_group_id, matches.date as match_date,
               matches.scheduled_at as match_scheduled_at,
               matches.registration_opens_at as match_registration_opens_at,
               matches.registrations_closed_at as match_registrations_closed_at,
               matches.max_players as match_max_players,
               teams.id as team_id, teams.name as team_name, teams.colour as team_colour,
               players.id as player_id, players.name as player_name,
			   match_players.goals_scored as goals_scored
        FROM matches
        LEFT JOIN match_players ON match_players.match_id = matches.id
        LEFT JOIN teams ON teams.id = match_players.team_id
        LEFT JOIN players ON players.id = match_players.player_id
		WHERE matches.group_id = ?
		ORDER BY match_date DESC
    `, groupID).Scan(&rowsMatches)

	if result.Error != nil {
		return nil, result.Error
	}

	// Sign-up counts come from their own query, never from the join above — see
	// RowsMatchDetails for why joining match_registrations there would corrupt
	// every score it also produces.
	//
	// Skipped entirely for a group with no scheduled match, which is every group
	// that predates this feature — and this function is also what the four
	// StandingsService calls go through, so an unconditional second round-trip
	// would be charged to every standings request for nothing.
	var registrationCounts map[uuid.UUID]int
	for _, rowMatches := range rowsMatches {
		if rowMatches.MatchScheduledAt != nil {
			registrationCounts, err = s.registrationCountsByGroup(groupID)
			if err != nil {
				return nil, err
			}
			break
		}
	}

	// Map the flat results into the hierarchical structure
	matchesMap := make(map[uuid.UUID]*models.MatchWithDetails)

	for _, rowMatches := range rowsMatches {
		// Get or create the match
		match, exists := matchesMap[rowMatches.MatchID]
		if !exists {
			match = &models.MatchWithDetails{
				ID:                    rowMatches.MatchID,
				GroupID:               rowMatches.MatchGroupID,
				Date:                  rowMatches.MatchDate,
				ScheduledAt:           rowMatches.MatchScheduledAt,
				RegistrationOpensAt:   rowMatches.MatchRegistrationOpensAt,
				RegistrationsClosedAt: rowMatches.MatchRegistrationsClosedAt,
				MaxPlayers:            rowMatches.MatchMaxPlayers,
				Teams:                 []models.TeamWithPlayers{},
			}
			// Present iff the match is scheduled — a nil count means "there is
			// nothing to sign up for here", which is not the same as zero
			// sign-ups. This is the only place a match is built in this
			// function, including for a match with no match_players rows at all
			// (the LEFT JOIN still yields one row for it, with a NULL team),
			// which is precisely the normal state of a scheduled match.
			if match.IsScheduled() {
				count := registrationCounts[rowMatches.MatchID]
				match.RegistrationCount = &count
			}
			matchesMap[rowMatches.MatchID] = match
		}

		// Get or create the team
		var team *models.TeamWithPlayers
		for i := range match.Teams {
			if match.Teams[i].ID == rowMatches.TeamID {
				team = &match.Teams[i]
				break
			}
		}
		if team == nil {
			newTeam := models.TeamWithPlayers{
				ID:      rowMatches.TeamID,
				Name:    rowMatches.TeamName,
				Colour:  rowMatches.TeamColour,
				Score:   0,
				Players: []models.PlayerCustom{},
			}
			match.Teams = append(match.Teams, newTeam)
			team = &match.Teams[len(match.Teams)-1] // Point to the newly appended team
		}

		// Add the player to the team
		team.Players = append(team.Players, models.PlayerCustom{
			ID:          rowMatches.PlayerID,
			Name:        rowMatches.PlayerName,
			GoalsScored: rowMatches.GoalsScored,
		})
	}

	// Fetch this group's teams once, used below to backfill any team that has no roster yet in a given match.
	var allTeams []models.Team
	if result := s.DB.Where("group_id = ?", groupID).Find(&allTeams); result.Error != nil {
		return nil, result.Error
	}

	// Convert the map to a slice
	var matches []models.MatchWithDetails
	for _, match := range matchesMap {
		// Filter out teams with missing ID or Colour
		var validTeams []models.TeamWithPlayers
		for _, team := range match.Teams {
			if team.ID != uuid.Nil && team.Colour != "" && len(team.Players) > 0 {
				for _, player := range team.Players {
					// Update the team's score based on the number of goals scored by players
					team.Score += player.GoalsScored
				}
				validTeams = append(validTeams, team)
			}
		}

		// Backfill any team that has no players assigned yet in this match, so a match
		// always shows every team instead of only the ones with a roster so far.
		for _, allTeam := range allTeams {
			found := false
			for _, team := range validTeams {
				if team.ID == allTeam.ID {
					found = true
					break
				}
			}
			if !found {
				validTeams = append(validTeams, models.TeamWithPlayers{
					ID:      allTeam.ID,
					Name:    allTeam.Name,
					Colour:  allTeam.Colour,
					Score:   0,
					Players: []models.PlayerCustom{},
				})
			}
		}

		// Only include matches with valid teams
		if len(validTeams) > 0 {
			match.Teams = validTeams
			matches = append(matches, *match)
		}
	}

	// Sort matches by date in descending order
	sort.Slice(matches, func(i, j int) bool {
		return matches[j].Date.Before(matches[i].Date)
	})
	// Sort matched teams same colour in first position
	for _, match := range matches {
		sort.Slice(match.Teams, func(i, j int) bool {
			return match.Teams[i].Colour < match.Teams[j].Colour
		})
	}

	// sort players in each team by their goal number in descending order
	for _, match := range matches {
		for _, team := range match.Teams {
			sort.Slice(team.Players, func(i, j int) bool {
				return team.Players[i].GoalsScored > team.Players[j].GoalsScored
			})
		}
	}
	return FilterMatchesBySeason(matches, season), nil
}

// GetMatchDetailsByID returns the match with the given id, scoped to
// groupID — a match belonging to a different group is treated as not found,
// so a match ID from one group can't be used to read another group's data.
func (s *MatchService) GetMatchDetailsByID(id uuid.UUID, groupID uuid.UUID) (*models.MatchWithDetails, error) {

	var rowsMatch []*models.RowsMatchDetails

	// Execute the SQL query and scan the results into a flat structure
	result := s.DB.Raw(`
        SELECT matches.id as match_id, matches.group_id as match_group_id, matches.date as match_date,
               matches.scheduled_at as match_scheduled_at,
               matches.registration_opens_at as match_registration_opens_at,
               matches.registrations_closed_at as match_registrations_closed_at,
               matches.max_players as match_max_players,
               teams.id as team_id, teams.name as team_name, teams.colour as team_colour,
               players.id as player_id, players.name as player_name,
			   match_players.goals_scored as goals_scored
        FROM matches
        LEFT JOIN match_players ON match_players.match_id = matches.id
        LEFT JOIN teams ON teams.id = match_players.team_id
        LEFT JOIN players ON players.id = match_players.player_id
		WHERE matches.id = ? AND matches.group_id = ?
		ORDER BY match_date DESC`, id, groupID).Scan(&rowsMatch)

	if result.Error != nil {
		return nil, result.Error
	}

	if len(rowsMatch) == 0 {
		return nil, ErrMatchNotFound
	}

	// Initialisez l'objet match. The Match* columns repeat identically on every
	// row of the join, so row 0 is as good as any — including when it is the
	// single all-NULL-team row of a match with no roster yet, which is the
	// normal state of a scheduled match nobody has been assigned to.
	match := &models.MatchWithDetails{
		ID:                    id,
		GroupID:               rowsMatch[0].MatchGroupID,
		Date:                  rowsMatch[0].MatchDate,
		ScheduledAt:           rowsMatch[0].MatchScheduledAt,
		RegistrationOpensAt:   rowsMatch[0].MatchRegistrationOpensAt,
		RegistrationsClosedAt: rowsMatch[0].MatchRegistrationsClosedAt,
		MaxPlayers:            rowsMatch[0].MatchMaxPlayers,
		Teams:                 []models.TeamWithPlayers{},
	}

	// A separate count rather than another join, for the reason spelled out on
	// RowsMatchDetails; present iff the match is scheduled, as in
	// GetMatchesDetails.
	if match.IsScheduled() {
		count, err := s.registrationCount(id)
		if err != nil {
			return nil, err
		}
		match.RegistrationCount = &count
	}

	for _, rowMatch := range rowsMatch {
		if rowMatch.TeamID == uuid.Nil {
			// No match_players row at all for this match yet.
			continue
		}

		// Get or create the team
		var team *models.TeamWithPlayers
		for i := range match.Teams {
			if match.Teams[i].ID == rowMatch.TeamID {
				team = &match.Teams[i]
				break
			}
		}

		if team == nil {
			newTeam := models.TeamWithPlayers{
				ID:      rowMatch.TeamID,
				Name:    rowMatch.TeamName,
				Colour:  rowMatch.TeamColour,
				Score:   0,
				Players: []models.PlayerCustom{},
			}
			match.Teams = append(match.Teams, newTeam)
			team = &match.Teams[len(match.Teams)-1] // Point to the newly appended team
		}

		team.Players = append(team.Players, models.PlayerCustom{
			ID:          rowMatch.PlayerID,
			Name:        rowMatch.PlayerName,
			GoalsScored: rowMatch.GoalsScored,
		})
	}

	// Calculez le score pour chaque équipe
	for i := range match.Teams {
		score := 0
		for _, player := range match.Teams[i].Players {
			score += player.GoalsScored
		}
		match.Teams[i].Score = score
	}

	// Backfill any team that has no players assigned yet in this match, so the match
	// always shows every team instead of only the ones with a roster so far.
	var allTeams []models.Team
	if result := s.DB.Where("group_id = ?", groupID).Find(&allTeams); result.Error != nil {
		return nil, result.Error
	}
	for _, allTeam := range allTeams {
		found := false
		for _, team := range match.Teams {
			if team.ID == allTeam.ID {
				found = true
				break
			}
		}
		if !found {
			match.Teams = append(match.Teams, models.TeamWithPlayers{
				ID:      allTeam.ID,
				Name:    allTeam.Name,
				Colour:  allTeam.Colour,
				Score:   0,
				Players: []models.PlayerCustom{},
			})
		}
	}

	// Triez les équipes par couleur (ordre alphabétique)
	if len(match.Teams) > 0 {
		sort.Slice(match.Teams, func(i, j int) bool {
			return match.Teams[i].Colour < match.Teams[j].Colour
		})
	}

	// Triez les joueurs dans chaque équipe par leur nombre de buts en ordre décroissant
	for i := range match.Teams {
		team := &match.Teams[i]
		sort.Slice(team.Players, func(k, l int) bool {
			return team.Players[k].GoalsScored > team.Players[l].GoalsScored
		})
	}

	return match, nil
}

// registrationCountsByGroup returns how many players have signed up per match,
// for every match of groupID that has at least one sign-up. Matches with none
// are simply absent from the map, so a plain lookup yields the zero value.
//
// This exists as its own query rather than as a fifth LEFT JOIN in
// GetMatchesDetails because match_registrations is a *second* one-to-many on
// matches: joined alongside match_players it would multiply the rows (one per
// registration × one per assigned player), and every score and goal total in
// this file is derived by summing those rows. One extra round-trip is the price
// of not silently corrupting all of them.
//
// It stays on the query builder — the group scope is a subquery on matches, not
// a join whose columns are needed, so there is nothing here that raw SQL would
// express better.
func (s *MatchService) registrationCountsByGroup(groupID uuid.UUID) (map[uuid.UUID]int, error) {
	var rows []struct {
		MatchID uuid.UUID
		Total   int
	}
	if err := s.DB.Model(&models.MatchRegistration{}).
		Select("match_id, COUNT(*) AS total").
		Where("match_id IN (?)", s.DB.Model(&models.Match{}).
			Select("id").Where("group_id = ?", groupID)).
		Group("match_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		counts[row.MatchID] = row.Total
	}
	return counts, nil
}

// registrationCount is registrationCountsByGroup for a single match. The match
// id is already known to belong to the authorized group by the time this runs
// (GetMatchDetailsByID scopes its own lookup on group_id), so there is nothing
// left to scope here.
func (s *MatchService) registrationCount(matchID uuid.UUID) (int, error) {
	var total int64
	if err := s.DB.Model(&models.MatchRegistration{}).
		Where("match_id = ?", matchID).
		Count(&total).Error; err != nil {
		return 0, err
	}
	return int(total), nil
}

// DeleteMatch removes the match with the given id, scoped to groupID the same
// way GetMatchDetailsByID is — a match belonging to a different group (or not
// existing at all) is reported as ErrMatchNotFound rather than being
// reachable. MatchPlayer.MatchID has no ON DELETE CASCADE, so every
// match_players row for this match is deleted first, inside the same
// transaction as the match itself, or the match delete would fail with a
// foreign-key violation instead of a clean application-level result.
func (s *MatchService) DeleteMatch(matchID, groupID uuid.UUID) error {
	var match models.Match
	if err := s.DB.Where("id = ? AND group_id = ?", matchID, groupID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMatchNotFound
		}
		return err
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("match_id = ?", matchID).Delete(&models.MatchPlayer{}).Error; err != nil {
			return err
		}
		// Sign-ups go the same way. MatchRegistration declares no association
		// on Match, so unlike match_players there is no FK forcing this — but
		// leaving a deleted match's sign-up list behind would be dead rows
		// nothing can ever read or clean up again.
		if err := tx.Where("match_id = ?", matchID).Delete(&models.MatchRegistration{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Match{}, "id = ?", matchID).Error
	})
}

func (s *MatchService) UpdateMatch(match models.MatchWithDetails) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var dbMatchPlayers []models.MatchPlayer

		for i := range match.Teams {
			team := &match.Teams[i]
			result := tx.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Find(&dbMatchPlayers)
			if result.Error != nil {
				return result.Error
			}
			for j := range team.Players {
				player := &team.Players[j]
				// Check if the player already exists in the team composition
				exists := false
				for _, dbMatchPlayer := range dbMatchPlayers {
					if dbMatchPlayer.PlayerID == player.ID {
						exists = true
						break
					}
				}
				if !exists {
					// Create a new team composition if it doesn't exist
					newMatchPlayer := models.MatchPlayer{
						MatchID:     match.ID,
						TeamID:      team.ID,
						PlayerID:    player.ID,
						GoalsScored: player.GoalsScored,
					}
					result := tx.Create(&newMatchPlayer)
					if result.Error != nil {
						return result.Error
					}
				}
			}

			for _, dbMatchPlayer := range dbMatchPlayers {
				toDelete := true
				for j := range team.Players {
					player := &team.Players[j]
					if dbMatchPlayer.PlayerID == player.ID {
						toDelete = false
						result := tx.Model(&models.MatchPlayer{}).Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Where("player_id = ?", player.ID).Update("goals_scored", player.GoalsScored)
						if result.Error != nil {
							return result.Error
						}
						break
					}
				}
				if toDelete {
					result := tx.Where("match_id = ?", match.ID).Where("team_id = ?", team.ID).Where("player_id = ?", dbMatchPlayer.PlayerID).Delete(&models.MatchPlayer{})
					if result.Error != nil {
						return result.Error
					}
				}
			}
		}
		return nil
	})
}
