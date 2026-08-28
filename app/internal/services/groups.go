package services

import (
	"crypto/rand"
	"errors"
	"strings"

	"app/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defaultTeamColours are the two teams every group is given automatically.
var defaultTeamColours = []string{"black", "white"}

var (
	ErrInviteCodeNotFound = errors.New("no group matches this invite code")
	ErrAlreadyMember      = errors.New("player is already a member of this group")
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

// CreateGroup creates a new group along with its two default teams
// (black/white), since every group must always have exactly these two teams,
// plus the invite code other players will use to join it.
//
// It deliberately says nothing about who is creating the group: the creator's
// membership is added by GroupHandler.CreateGroup from the JWT, the same way
// PlayerHandler.CreatePlayer attaches a new player to a group. That keeps this
// signature usable from cmd/seed, which has no authenticated caller.
func (s *GroupService) CreateGroup(name string) (*models.Group, error) {
	inviteCode, err := s.uniqueInviteCode()
	if err != nil {
		return nil, err
	}

	group := &models.Group{Name: name, InviteCode: inviteCode}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		if result := tx.Create(group); result.Error != nil {
			return result.Error
		}
		teamService := NewTeamService(tx)
		for _, colour := range defaultTeamColours {
			team := &models.Team{GroupID: group.ID, Colour: colour}
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

// GetDefaultGroup returns whichever group's (random) UUID sorts first — it
// has no relation to any particular caller or player. That makes it unsafe
// as a fallback for anything authenticated: creating a second group can
// silently flip which group is "default" and, combined with a group
// membership check, lock out every existing user who never passes a
// group_id explicitly (see the incident this was pulled from PlayerHandler
// and MatchHandler's read/list paths for). Only use it where there is no
// authenticated player to resolve a real group for instead — currently just
// PlayerHandler.CreatePlayer, whose route is intentionally public. Anywhere
// behind AuthMiddleware, use GroupMembershipService.GetFirstGroupForPlayer.
func (s *GroupService) GetDefaultGroup() (*models.Group, error) {
	var group models.Group
	result := s.DB.Order("id").First(&group)
	if result.Error != nil {
		return nil, result.Error
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

	var group models.Group
	if err := s.DB.First(&group, "invite_code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteCodeNotFound
		}
		return nil, err
	}

	membershipService := NewGroupMembershipService(s.DB)
	isMember, err := membershipService.IsMember(group.ID, playerID)
	if err != nil {
		return nil, err
	}
	if isMember {
		return nil, ErrAlreadyMember
	}
	if err := membershipService.AddPlayerToGroup(group.ID, playerID); err != nil {
		return nil, err
	}
	return &group, nil
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
