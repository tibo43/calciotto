package services

import (
	"strings"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TeamService struct {
	DB *gorm.DB
}

func NewTeamService(db *gorm.DB) *TeamService {
	return &TeamService{DB: db}
}

func (s *TeamService) GetTeams() ([]models.Team, error) {
	var teams []models.Team
	result := s.DB.Find(&teams)
	if result.Error != nil {
		return nil, result.Error
	}
	return teams, nil
}

func (s *TeamService) CreateTeam(team *models.Team) error {
	team.ID = uuid.New()
	result := s.DB.Create(team)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *TeamService) GetTeamByID(id uuid.UUID) (*models.Team, error) {
	var team models.Team
	result := s.DB.First(&team, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &team, nil
}

// GetTeamsByGroupID returns the teams belonging to a given group (normally
// exactly the 2 default teams created alongside the group).
func (s *TeamService) GetTeamsByGroupID(groupID uuid.UUID) ([]models.Team, error) {
	var teams []models.Team
	result := s.DB.Where("group_id = ?", groupID).Find(&teams)
	if result.Error != nil {
		return nil, result.Error
	}
	return teams, nil
}

// UpdateTeam renames a team and/or changes its colour. The lookup is scoped
// to both teamID and groupID in the same query — exactly the same pattern
// MatchService.GetMatchDetailsByID uses for matches — so a team belonging to
// a different group reads as gorm.ErrRecordNotFound rather than being
// silently updatable: RequireGroupAdminByPathParam only proves the caller
// administers groupID, it says nothing about whether teamID actually belongs
// to it.
//
// It returns the updated team so the handler can hand the caller a fresh
// copy without a follow-up fetch.
func (s *TeamService) UpdateTeam(teamID, groupID uuid.UUID, name, colour string) (*models.Team, error) {
	name = strings.TrimSpace(name)
	colour = strings.TrimSpace(colour)
	if name == "" {
		return nil, ErrTeamNameRequired
	}
	if colour == "" {
		return nil, ErrTeamColourRequired
	}

	var team models.Team
	if err := s.DB.Where("id = ? AND group_id = ?", teamID, groupID).First(&team).Error; err != nil {
		return nil, err
	}

	team.Name = name
	team.Colour = colour
	if err := s.DB.Save(&team).Error; err != nil {
		return nil, err
	}
	return &team, nil
}
