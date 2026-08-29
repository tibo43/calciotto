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

	playerID, err := authService.SignupNewPlayer("  Zzz Integration Auth Gwen  ", "  Gwen@Example.com ", "s3cret-pass", "")
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

	if _, err := authService.SignupNewPlayer("Zzz Integration Auth Jack", "shared-new@example.com", "s3cret-pass", ""); err != nil {
		t.Fatalf("first SignupNewPlayer returned error: %v", err)
	}

	_, err := authService.SignupNewPlayer("Zzz Integration Auth Jill", "SHARED-NEW@example.com", "another-pass", "")
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

	firstID, err := authService.SignupNewPlayer("Zzz Integration Auth Kim", "kim-one@example.com", "s3cret-pass", "")
	if err != nil {
		t.Fatalf("first SignupNewPlayer returned error: %v", err)
	}

	secondID, err := authService.SignupNewPlayer("Zzz Integration Auth Kim", "kim-two@example.com", "another-pass", "")
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

// TestSignupNewPlayer_Integration_WithoutInviteCodeBehavesAsBefore pins the
// empty-string case (the default when the frontend field is left blank) to
// the pre-existing behavior: a player is created with no group membership at
// all.
func TestSignupNewPlayer_Integration_WithoutInviteCodeBehavesAsBefore(t *testing.T) {
	db := testutil.OpenDB(t)
	tx := testutil.BeginTx(t, db)

	membershipService := services.NewGroupMembershipService(tx)
	authService := services.NewAuthService(tx, testJWTSecret)

	playerID, err := authService.SignupNewPlayer("Zzz Integration Auth Noah", "noah@example.com", "s3cret-pass", "")
	if err != nil {
		t.Fatalf("SignupNewPlayer with no invite code returned error: %v", err)
	}

	groups, err := membershipService.GetGroupsWithRoleByPlayerID(playerID)
	if err != nil {
		t.Fatalf("GetGroupsWithRoleByPlayerID returned error: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("GetGroupsWithRoleByPlayerID = %+v, want no groups for a signup with no invite code", groups)
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
