package services

import (
	"crypto/rand"
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TeamSpec is one team's name, as supplied by the admin creating a group —
// see CreateGroup, which takes exactly two of these (a group always has
// exactly two teams), in the order the teams are then shown.
type TeamSpec struct {
	Name string
}

// DefaultTeamSpecs are the team specs cmd/seed uses, and that most tests
// reach for when the actual team names don't matter to the scenario under
// test.
var DefaultTeamSpecs = [2]TeamSpec{
	{Name: "Black"},
	{Name: "White"},
}

var (
	ErrInviteCodeNotFound       = errors.New("no group matches this invite code")
	ErrAlreadyMember            = errors.New("player is already a member of this group")
	ErrTeamNameRequired         = errors.New("team name must not be empty")
	ErrGroupDescriptionTooLong  = errors.New("description must be at most 1000 characters")
	ErrGroupWebsiteLabelTooLong = errors.New("link label must be at most 100 characters")
	ErrInvalidGroupWebsite      = errors.New("enter a valid HTTP or HTTPS address up to 2048 characters")
)

const (
	inviteCodeLength = 8
	// inviteCodeAlphabet leaves out the characters that get mixed up when a
	// code is read out loud or copied by hand (0/O, 1/I/L), since the whole
	// point of an invite code is to be passed around between humans.
	inviteCodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	// inviteCodeMaxAttempts bounds the re-roll loop in uniqueInviteCode: with
	// 31^8 possibilities a collision is already improbable, so exhausting the
	// attempts means something else is wrong.
	inviteCodeMaxAttempts = 5
)

type GroupService struct {
	DB *gorm.DB
}

func NewGroupService(db *gorm.DB) *GroupService {
	return &GroupService{DB: db}
}

// CreateGroup creates a new group along with its two teams — built from the
// two TeamSpecs the caller provides, since a group's teams no longer have a
// reasonable default name to invent on its behalf — plus the invite code
// other players will use to join it.
//
// It deliberately says nothing about who is creating the group: the creator's
// membership is added by GroupHandler.CreateGroup from the JWT, the same way
// PlayerHandler.CreatePlayer attaches a new player to a group.
func (s *GroupService) CreateGroup(name string, teams [2]TeamSpec) (*models.Group, error) {
	inviteCode, err := s.uniqueInviteCode()
	if err != nil {
		return nil, err
	}

	group := &models.Group{Name: name, InviteCode: inviteCode, InviteExpiresAt: models.InviteNeverExpires}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		trimmed := make([]TeamSpec, len(teams))
		for i, spec := range teams {
			trimmed[i] = TeamSpec{Name: strings.TrimSpace(spec.Name)}
			if trimmed[i].Name == "" {
				return ErrTeamNameRequired
			}
		}

		if result := tx.Create(group); result.Error != nil {
			return result.Error
		}
		teamService := NewTeamService(tx)
		for position, spec := range trimmed {
			team := &models.Team{GroupID: group.ID, Name: spec.Name, Position: position}
			if err := teamService.CreateTeam(team); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (s *GroupService) GetGroups() ([]models.Group, error) {
	var groups []models.Group
	result := s.DB.Find(&groups)
	if result.Error != nil {
		return nil, result.Error
	}
	return groups, nil
}

func (s *GroupService) GetGroupByID(id uuid.UUID) (*models.Group, error) {
	var group models.Group
	result := s.DB.First(&group, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &group, nil
}

func (s *GroupService) UpdateGroupDetails(groupID, adminID uuid.UUID, details models.GroupDetails) (*models.Group, error) {
	details.Description = strings.TrimSpace(details.Description)
	details.WebsiteURL = strings.TrimSpace(details.WebsiteURL)
	details.WebsiteLabel = strings.TrimSpace(details.WebsiteLabel)
	if utf8.RuneCountInString(details.Description) > 1000 {
		return nil, ErrGroupDescriptionTooLong
	}
	if details.WebsiteURL == "" {
		details.WebsiteLabel = ""
	} else {
		website, err := url.Parse(details.WebsiteURL)
		if err != nil || len(details.WebsiteURL) > 2048 || website.Hostname() == "" ||
			website.User != nil || (website.Scheme != "http" && website.Scheme != "https") {
			return nil, ErrInvalidGroupWebsite
		}
		if utf8.RuneCountInString(details.WebsiteLabel) > 100 {
			return nil, ErrGroupWebsiteLabelTooLong
		}
	}

	var group models.Group
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, "id = ?", groupID).Error; err != nil {
			return err
		}
		if err := verifyGroupAdmin(tx, groupID, adminID); err != nil {
			return err
		}
		group.GroupDetails = details
		return tx.Model(&group).Select("description", "website_url", "website_label").Updates(&group).Error
	})
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// ChangeInvitation also serializes with redemption of the current code.
func (s *GroupService) ChangeInvitation(groupID, adminID uuid.UUID, rotate bool) (*models.Group, error) {
	var group models.Group
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, "id = ?", groupID).Error; err != nil {
			return err
		}
		if err := verifyGroupAdmin(tx, groupID, adminID); err != nil {
			return err
		}
		group.InviteExpiresAt = time.Now()
		if rotate {
			code, err := NewGroupService(tx).uniqueInviteCode()
			if err != nil {
				return err
			}
			group.InviteCode = code
			group.InviteExpiresAt = models.InviteNeverExpires
		}
		return tx.Model(&group).Select("invite_code", "invite_expires_at").Updates(&group).Error
	})
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func groupForActiveInvite(db *gorm.DB, code string) (*models.Group, error) {
	var group models.Group
	if err := db.First(&group, "invite_code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteCodeNotFound
		}
		return nil, err
	}
	if !group.InviteExpiresAt.After(time.Now()) {
		return nil, ErrInviteCodeNotFound
	}
	return &group, nil
}

// JoinByInviteCode makes playerID a member of the group owning code. It is the
// only way into a group the caller isn't already in — POST /groups/:id/players
// requires the caller to be a member of the target group already, so without
// this a freshly created group could never gain a second member.
//
// The code is matched case-insensitively (stored upper-case, normalized on the
// way in) for the same reason AuthService normalizes emails: a code that gets
// typed by hand shouldn't fail on capitalization.
func (s *GroupService) JoinByInviteCode(playerID uuid.UUID, code string) (*models.Group, error) {
	code = normalizeInviteCode(code)
	if code == "" {
		return nil, ErrInviteCodeNotFound
	}

	var group *models.Group
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := lockPlayer(tx, playerID); err != nil {
			return err
		}
		var err error
		group, err = groupForActiveInvite(tx.Clauses(clause.Locking{Strength: "UPDATE"}), code)
		if err != nil {
			return err
		}
		membershipService := NewGroupMembershipService(tx)
		isMember, err := membershipService.IsMember(group.ID, playerID)
		if err != nil {
			return err
		}
		if isMember {
			return ErrAlreadyMember
		}
		return addGroupMembership(tx, group.ID, playerID, models.RoleMember)
	})
	if err != nil {
		return nil, err
	}
	return group, nil
}

// uniqueInviteCode draws a random invite code and re-rolls it if some group
// already holds it. The unique index on Group.InviteCode remains the actual
// guarantee — this loop only keeps an improbable collision from surfacing to
// the caller as a failed group creation.
func (s *GroupService) uniqueInviteCode() (string, error) {
	for attempt := 0; attempt < inviteCodeMaxAttempts; attempt++ {
		code, err := generateInviteCode()
		if err != nil {
			return "", err
		}
		var count int64
		if err := s.DB.Model(&models.Group{}).Where("invite_code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("failed to generate a unique invite code")
}

// generateInviteCode returns a cryptographically random code over
// inviteCodeAlphabet. Bytes landing in the incomplete final block of the
// alphabet are rejected rather than folded with a modulo, so every character
// stays equally likely.
func generateInviteCode() (string, error) {
	maxByte := byte(256 - (256 % len(inviteCodeAlphabet)))
	code := make([]byte, 0, inviteCodeLength)
	buf := make([]byte, 1)
	for len(code) < inviteCodeLength {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		if buf[0] >= maxByte {
			continue
		}
		code = append(code, inviteCodeAlphabet[int(buf[0])%len(inviteCodeAlphabet)])
	}
	return string(code), nil
}

func normalizeInviteCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
