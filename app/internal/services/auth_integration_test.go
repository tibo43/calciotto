package services_test

import (
	"errors"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testJWTSecret = "zzz-integration-test-secret"

func TestSignup_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Alice")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	if err := authService.Signup(playerID, "  Alice@Example.com ", "s3cret-pass"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	token, err := authService.Login("alice@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("Login after successful signup returned error: %v", err)
	}
	if token == "" {
		t.Fatal("Login returned an empty token")
	}
}

func TestSignup_Integration_AlreadyClaimedPlayerFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Bob")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}

	if err := authService.Signup(playerID, "bob@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("first Signup returned error: %v", err)
	}

	err = authService.Signup(playerID, "bob-other@example.com", "another-pass")
	if !errors.Is(err, services.ErrPlayerAlreadyClaimed) {
		t.Errorf("second Signup on same player error = %v, want ErrPlayerAlreadyClaimed", err)
	}
}

func TestSignup_Integration_EmailAlreadyUsedByAnotherPlayerFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	carolID, err := playerService.CreatePlayer("Zzz Integration Auth Carol")
	if err != nil {
		t.Fatalf("failed to create player carol: %v", err)
	}
	daveID, err := playerService.CreatePlayer("Zzz Integration Auth Dave")
	if err != nil {
		t.Fatalf("failed to create player dave: %v", err)
	}

	if err := authService.Signup(carolID, "shared@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("Signup for carol returned error: %v", err)
	}

	err = authService.Signup(daveID, "SHARED@example.com", "another-pass")
	if !errors.Is(err, services.ErrEmailAlreadyUsed) {
		t.Errorf("Signup for dave with carol's email error = %v, want ErrEmailAlreadyUsed", err)
	}
}

func TestSignup_Integration_UnknownPlayerFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	err := authService.Signup(uuid.New(), "ghost@example.com", "s3cret-pass")
	if !errors.Is(err, services.ErrPlayerNotFound) {
		t.Errorf("Signup for unknown player error = %v, want ErrPlayerNotFound", err)
	}
}

func TestSignupNewPlayer_Integration_Success(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	group, err := services.NewGroupService(tx).CreateGroup("Zzz Signup Success Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	playerID, err := authService.SignupNewPlayer("  Zzz Integration Auth Gwen  ", "  Gwen@Example.com ", "s3cret-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer returned error: %v", err)
	}
	if playerID == uuid.Nil {
		t.Fatal("SignupNewPlayer returned a nil player ID")
	}

	token, err := authService.Login("gwen@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("Login after SignupNewPlayer returned error: %v", err)
	}
	if token == "" {
		t.Fatal("Login returned an empty token")
	}

	decodedID, err := authService.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if decodedID != playerID {
		t.Errorf("ParseToken player_id = %s, want %s", decodedID, playerID)
	}
}

func TestSignupNewPlayer_Integration_EmptyNameFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	_, err := authService.SignupNewPlayer("   ", "empty-name@example.com", "s3cret-pass", "")
	if !errors.Is(err, services.ErrEmptyPlayerName) {
		t.Errorf("SignupNewPlayer with an empty name error = %v, want ErrEmptyPlayerName", err)
	}
}

func TestSignupNewPlayer_Integration_EmptyEmailFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	_, err := authService.SignupNewPlayer("Zzz Integration Auth Holly", "   ", "s3cret-pass", "")
	if !errors.Is(err, services.ErrEmailRequired) {
		t.Errorf("SignupNewPlayer with an empty email error = %v, want ErrEmailRequired", err)
	}
}

func TestSignupNewPlayer_Integration_EmptyPasswordFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	_, err := authService.SignupNewPlayer("Zzz Integration Auth Ivy", "ivy@example.com", "", "")
	if !errors.Is(err, services.ErrPasswordRequired) {
		t.Errorf("SignupNewPlayer with an empty password error = %v, want ErrPasswordRequired", err)
	}
}

func TestSignupNewPlayer_Integration_DuplicateEmailFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	group, err := services.NewGroupService(tx).CreateGroup("Zzz Signup Duplicate Email Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	if _, err := authService.SignupNewPlayer("Zzz Integration Auth Jack", "shared-new@example.com", "s3cret-pass", group.InviteCode); err != nil {
		t.Fatalf("first SignupNewPlayer returned error: %v", err)
	}

	_, err = authService.SignupNewPlayer("Zzz Integration Auth Jill", "SHARED-NEW@example.com", "another-pass", group.InviteCode)
	if !errors.Is(err, services.ErrEmailAlreadyUsed) {
		t.Errorf("SignupNewPlayer with an already-used email error = %v, want ErrEmailAlreadyUsed", err)
	}
}

// TestSignupNewPlayer_Integration_DuplicateNameSucceeds asserts the
// deliberate behavior change this method introduces: unlike
// PlayerService.CreatePlayer (used by the separate ghost-player admin flow),
// SignupNewPlayer never rejects a name that's already in use. Two unrelated
// people can share a display name/nickname across different groups.
func TestSignupNewPlayer_Integration_DuplicateNameSucceeds(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	group, err := services.NewGroupService(tx).CreateGroup("Zzz Signup Duplicate Name Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	firstID, err := authService.SignupNewPlayer("Zzz Integration Auth Kim", "kim-one@example.com", "s3cret-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("first SignupNewPlayer returned error: %v", err)
	}

	secondID, err := authService.SignupNewPlayer("Zzz Integration Auth Kim", "kim-two@example.com", "another-pass", group.InviteCode)
	if err != nil {
		t.Errorf("second SignupNewPlayer with a duplicate name returned error: %v, want nil (names are not unique)", err)
	}
	if firstID == secondID {
		t.Error("two signups with the same name got the same player ID")
	}

	// Both accounts must be independently usable.
	if _, err := authService.Login("kim-one@example.com", "s3cret-pass"); err != nil {
		t.Errorf("Login for the first Kim account returned error: %v", err)
	}
	if _, err := authService.Login("kim-two@example.com", "another-pass"); err != nil {
		t.Errorf("Login for the second Kim account returned error: %v", err)
	}
}

// TestSignupNewPlayer_Integration_WithValidInviteCodeJoinsGroup covers the
// happy path of the optional invite_code parameter: a brand-new player is
// created AND enrolled as a member of the group owning the code, in the same
// call.
func TestSignupNewPlayer_Integration_WithValidInviteCodeJoinsGroup(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	groupService := services.NewGroupService(tx)
	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	group, err := groupService.CreateGroup("Zzz Signup Invite Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	playerID, err := authService.SignupNewPlayer("Zzz Integration Auth Liam", "liam@example.com", "s3cret-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer with a valid invite code returned error: %v", err)
	}
	if playerID == uuid.Nil {
		t.Fatal("SignupNewPlayer returned a nil player ID")
	}

	isMember, err := membershipService.IsMember(group.ID, playerID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !isMember {
		t.Error("player signed up with a valid invite code is not a member of that group")
	}

	// Also lower-cased/whitespace-padded, mirroring how a human would type it
	// back — normalizeInviteCode must handle this the same way JoinByInviteCode
	// does.
	groups, err := membershipService.GetGroupsWithRoleByPlayerID(playerID)
	if err != nil {
		t.Fatalf("GetGroupsWithRoleByPlayerID returned error: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != group.ID {
		t.Errorf("GetGroupsWithRoleByPlayerID = %+v, want exactly one entry for group %s", groups, group.ID)
	}
}

// TestSignupNewPlayer_Integration_WithInvalidInviteCodeFailsAtomically is the
// most important guarantee this feature adds: an unknown/invalid invite code
// must fail the whole signup, not just skip joining a group — no orphaned
// Player row left behind with no group membership.
func TestSignupNewPlayer_Integration_WithInvalidInviteCodeFailsAtomically(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	_, err := authService.SignupNewPlayer("Zzz Integration Auth Mia", "mia@example.com", "s3cret-pass", "NOTAREALCODE")
	if !errors.Is(err, services.ErrInviteCodeNotFound) {
		t.Errorf("SignupNewPlayer with an invalid invite code error = %v, want ErrInviteCodeNotFound", err)
	}

	var count int64
	if err := tx.Model(&models.Player{}).Where("email = ?", "mia@example.com").Count(&count).Error; err != nil {
		t.Fatalf("failed to count players by email: %v", err)
	}
	if count != 0 {
		t.Errorf("player row was created despite the invalid invite code failing the transaction (count = %d, want 0)", count)
	}
}

// TestSignupNewPlayer_Integration_WithoutInviteCodeFailsRequired pins the
// current, mandatory-invite-code behavior: self-service group creation/
// joining is disabled (see GroupHandler.CreateGroup/JoinGroup), so signup is
// the only way left into a group, and an empty (or whitespace-only) code must
// fail fast with ErrInviteCodeRequired — no player row left behind, mirroring
// TestSignupNewPlayer_Integration_WithInvalidInviteCodeFailsAtomically's own
// "no orphaned account" guarantee.
func TestSignupNewPlayer_Integration_WithoutInviteCodeFailsRequired(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	_, err := authService.SignupNewPlayer("Zzz Integration Auth Noah", "noah@example.com", "s3cret-pass", "   ")
	if !errors.Is(err, services.ErrInviteCodeRequired) {
		t.Errorf("SignupNewPlayer with no invite code error = %v, want ErrInviteCodeRequired", err)
	}

	var count int64
	if err := tx.Model(&models.Player{}).Where("email = ?", "noah@example.com").Count(&count).Error; err != nil {
		t.Fatalf("failed to count players by email: %v", err)
	}
	if count != 0 {
		t.Errorf("player row was created despite the missing invite code failing signup (count = %d, want 0)", count)
	}
}

func TestLogin_Integration_WrongPasswordFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Erin")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := authService.Signup(playerID, "erin@example.com", "correct-pass"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	_, err = authService.Login("erin@example.com", "wrong-pass")
	if !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("Login with wrong password error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_Integration_UnknownEmailFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	_, err := authService.Login("nobody-zzz@example.com", "whatever")
	if !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("Login with unknown email error = %v, want ErrInvalidCredentials (must not leak which case failed)", err)
	}
}

func TestLogin_Integration_TokenContainsPlayerID(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	playerService := services.NewPlayerService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := playerService.CreatePlayer("Zzz Integration Auth Frank")
	if err != nil {
		t.Fatalf("failed to create player: %v", err)
	}
	if err := authService.Signup(playerID, "frank@example.com", "s3cret-pass"); err != nil {
		t.Fatalf("Signup returned error: %v", err)
	}

	token, err := authService.Login("frank@example.com", "s3cret-pass")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}

	decodedID, err := authService.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken on a freshly issued token returned error: %v", err)
	}
	if decodedID != playerID {
		t.Errorf("ParseToken player_id = %s, want %s", decodedID, playerID)
	}
}

// TestDeleteAccount_Integration_WrongPasswordFails pins DeleteAccount's
// safety confirmation: a wrong password must refuse the whole deletion, and
// leave the account exactly as it was (still able to log in).
func TestDeleteAccount_Integration_WrongPasswordFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	group, err := services.NewGroupService(tx).CreateGroup("Zzz Delete Account Wrong Password Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	playerID, err := authService.SignupNewPlayer("Zzz Integration Auth Oscar", "oscar@example.com", "correct-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer returned error: %v", err)
	}

	if err := authService.DeleteAccount(playerID, "wrong-pass"); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("DeleteAccount with wrong password error = %v, want ErrInvalidCredentials", err)
	}

	if _, err := authService.Login("oscar@example.com", "correct-pass"); err != nil {
		t.Errorf("Login after a refused DeleteAccount returned error: %v, account must be untouched", err)
	}
}

// TestDeleteAccount_Integration_EmptyPasswordFails pins the same up-front
// guard Signup/ResetPassword already share for an empty password.
func TestDeleteAccount_Integration_EmptyPasswordFails(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	group, err := services.NewGroupService(tx).CreateGroup("Zzz Delete Account Empty Password Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	playerID, err := authService.SignupNewPlayer("Zzz Integration Auth Paul", "paul@example.com", "s3cret-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer returned error: %v", err)
	}

	if err := authService.DeleteAccount(playerID, ""); !errors.Is(err, services.ErrPasswordRequired) {
		t.Errorf("DeleteAccount with an empty password error = %v, want ErrPasswordRequired", err)
	}
}

// TestDeleteAccount_Integration_AnonymizesAndCleansUp is the main happy-path
// test: it covers the account itself (anonymized, not erased, and no longer
// usable to log in), the admin-successor rule (mirroring
// GroupMembershipService.LeaveGroup) when the deleted player was a group's
// last admin, and the asymmetric handling of match-related rows — votes CAST
// by the deleted player are gone, but a vote FOR them (a historical fact
// about a match that already happened) survives untouched.
func TestDeleteAccount_Integration_AnonymizesAndCleansUp(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)
	membershipService := services.NewGroupMembershipService(tx)
	matchService := services.NewMatchService(tx)

	group, err := services.NewGroupService(tx).CreateGroup("Zzz Delete Account Cleanup Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	deletingPlayerID, err := authService.SignupNewPlayer("Zzz Integration Auth Quinn", "quinn@example.com", "s3cret-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer for the deleting player returned error: %v", err)
	}
	// SignupNewPlayer always joins as a plain member (see CLAUDE.md), and
	// CreateGroup itself assigns no admin at all (see GroupService.CreateGroup)
	// — promote Quinn to admin directly so they end up the group's *only*
	// admin, the case the successor rule exists for.
	if err := tx.Model(&models.GroupMembership{}).
		Where("group_id = ? AND player_id = ?", group.ID, deletingPlayerID).
		Update("role", models.RoleAdmin).Error; err != nil {
		t.Fatalf("failed to promote Quinn to admin directly: %v", err)
	}

	survivorID, err := authService.SignupNewPlayer("Zzz Integration Auth Rita", "rita@example.com", "s3cret-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer for the survivor returned error: %v", err)
	}

	matchID, err := matchService.CreateMatch(services.MatchSpec{Date: models.DateOf(time.Now())}, group.ID)
	if err != nil {
		t.Fatalf("CreateMatch returned error: %v", err)
	}
	if err := tx.Create(&models.MatchRegistration{MatchID: matchID, PlayerID: deletingPlayerID}).Error; err != nil {
		t.Fatalf("failed to create match registration: %v", err)
	}
	// Quinn votes for Rita (must be deleted along with Quinn's account) and
	// Rita votes for Quinn (must survive — a historical fact about the match).
	if err := tx.Create(&models.MatchVote{MatchID: matchID, VoterID: deletingPlayerID, VotedForID: survivorID}).Error; err != nil {
		t.Fatalf("failed to create Quinn's vote: %v", err)
	}
	if err := tx.Create(&models.MatchVote{MatchID: matchID, VoterID: survivorID, VotedForID: deletingPlayerID}).Error; err != nil {
		t.Fatalf("failed to create Rita's vote: %v", err)
	}

	if err := authService.DeleteAccount(deletingPlayerID, "s3cret-pass"); err != nil {
		t.Fatalf("DeleteAccount returned error: %v", err)
	}

	// The membership is gone outright, and Rita — the group's only remaining
	// member — was promoted to admin so the group isn't left adminless.
	isMember, err := membershipService.IsMember(group.ID, deletingPlayerID)
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if isMember {
		t.Error("deleted player is still a member of the group")
	}
	survivorRole, err := membershipService.GetRole(group.ID, survivorID)
	if err != nil {
		t.Fatalf("GetRole for the survivor returned error: %v", err)
	}
	if survivorRole != models.RoleAdmin {
		t.Errorf("survivor role = %q, want %q (successor promotion)", survivorRole, models.RoleAdmin)
	}

	var registrationCount int64
	if err := tx.Model(&models.MatchRegistration{}).Where("player_id = ?", deletingPlayerID).Count(&registrationCount).Error; err != nil {
		t.Fatalf("failed to count match registrations: %v", err)
	}
	if registrationCount != 0 {
		t.Errorf("match registration count for deleted player = %d, want 0", registrationCount)
	}

	var castVoteCount int64
	if err := tx.Model(&models.MatchVote{}).Where("voter_id = ?", deletingPlayerID).Count(&castVoteCount).Error; err != nil {
		t.Fatalf("failed to count votes cast by deleted player: %v", err)
	}
	if castVoteCount != 0 {
		t.Errorf("votes cast by deleted player = %d, want 0", castVoteCount)
	}

	var voteForDeletedPlayer models.MatchVote
	if err := tx.Where("voter_id = ? AND voted_for_id = ?", survivorID, deletingPlayerID).First(&voteForDeletedPlayer).Error; err != nil {
		t.Errorf("vote FOR the deleted player was removed, want it kept as historical record: %v", err)
	}

	var player models.Player
	if err := tx.First(&player, "id = ?", deletingPlayerID).Error; err != nil {
		t.Fatalf("failed to reload deleted player: %v", err)
	}
	if player.Name != "Deleted account" {
		t.Errorf("player name after deletion = %q, want %q", player.Name, "Deleted account")
	}
	if player.Email != nil {
		t.Errorf("player email after deletion = %q, want nil", *player.Email)
	}
	if player.PasswordHash != "" {
		t.Error("player password hash after deletion is not empty")
	}

	if _, err := authService.Login("quinn@example.com", "s3cret-pass"); !errors.Is(err, services.ErrInvalidCredentials) {
		t.Errorf("Login with the deleted account's old email error = %v, want ErrInvalidCredentials", err)
	}
}

// TestDeleteAccount_Integration_SoleGroupMemberLeavesGroupEmpty pins the one
// deliberate divergence from GroupMembershipService.LeaveGroup: leaving
// voluntarily refuses when the departing player is a group's only member
// (ErrLastMember), since there's no one to hand the group off to — but
// deleting the account outright cannot be refused the same way, so the group
// is simply left with no members at all.
func TestDeleteAccount_Integration_SoleGroupMemberLeavesGroupEmpty(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	group, err := services.NewGroupService(tx).CreateGroup("Zzz Delete Account Sole Member Group", services.DefaultTeamSpecs)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	playerID, err := authService.SignupNewPlayer("Zzz Integration Auth Sam", "sam@example.com", "s3cret-pass", group.InviteCode)
	if err != nil {
		t.Fatalf("SignupNewPlayer returned error: %v", err)
	}

	if err := authService.DeleteAccount(playerID, "s3cret-pass"); err != nil {
		t.Fatalf("DeleteAccount for a group's sole member returned error: %v, want nil", err)
	}

	var memberCount int64
	if err := tx.Model(&models.GroupMembership{}).Where("group_id = ?", group.ID).Count(&memberCount).Error; err != nil {
		t.Fatalf("failed to count memberships: %v", err)
	}
	if memberCount != 0 {
		t.Errorf("group membership count after its sole member deleted their account = %d, want 0", memberCount)
	}
}

func TestParseToken_Integration_RejectsInvalidAndExpiredTokens(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	authService := services.NewAuthService(tx, testJWTSecret)

	if _, err := authService.ParseToken("not-a-valid-jwt"); !errors.Is(err, services.ErrInvalidToken) {
		t.Errorf("ParseToken(garbage) error = %v, want ErrInvalidToken", err)
	}

	expiredClaims := jwt.MapClaims{
		"player_id": uuid.New().String(),
		"exp":       jwt.NewNumericDate(time.Now().Add(-time.Hour)),
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	signedExpired, err := expiredToken.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign expired test token: %v", err)
	}
	if _, err := authService.ParseToken(signedExpired); !errors.Is(err, services.ErrInvalidToken) {
		t.Errorf("ParseToken(expired) error = %v, want ErrInvalidToken", err)
	}

	wrongSecretToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"player_id": uuid.New().String(),
	})
	signedWrongSecret, err := wrongSecretToken.SignedString([]byte("a-different-secret"))
	if err != nil {
		t.Fatalf("failed to sign wrong-secret test token: %v", err)
	}
	if _, err := authService.ParseToken(signedWrongSecret); !errors.Is(err, services.ErrInvalidToken) {
		t.Errorf("ParseToken(wrong secret) error = %v, want ErrInvalidToken", err)
	}
}
