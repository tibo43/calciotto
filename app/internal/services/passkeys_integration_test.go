package services_test

import (
	"testing"

	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/google/uuid"
)

func TestPasskeys_Integration_AccountRecoveryClearsCredentials(t *testing.T) {
	const email = "passkey-recovery@example.test"
	const origin = "https://calciotto.example.test"
	for _, name := range []string{"password reset", "account deletion"} {
		t.Run(name, func(t *testing.T) {
			tx := testutil.BeginTx(t, testutil.OpenDB(t))
			auth := services.NewAuthService(tx, testJWTSecret)
			password := uuid.NewString()
			playerID := newClaimedPlayer(t, tx, "Passkey recovery", email, password)
			otherID := newClaimedPlayer(t, tx, "Other account", "passkey-recovery-other@example.test", password)
			for _, id := range []uuid.UUID{playerID, otherID} {
				key := models.Passkey{PlayerID: id, RPID: "calciotto.example.test", CredentialID: id[:], Name: "Phone"}
				if err := tx.Create(&key).Error; err != nil {
					t.Fatal(err)
				}
			}
			keys, err := services.NewPasskeyService(auth, []string{origin})
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := keys.BeginRegistration(playerID, origin, password, "New phone"); err != nil {
				t.Fatal(err)
			}
			if name == "password reset" {
				err = auth.ResetPassword(requestResetToken(t, auth, email), uuid.NewString())
			} else {
				err = auth.DeleteAccount(playerID, password)
			}
			if err != nil {
				t.Fatal(err)
			}
			if owned, err := keys.List(playerID); err != nil || len(owned) != 0 {
				t.Fatalf("passkeys after recovery = %d (error %v)", len(owned), err)
			}
			if other, err := keys.List(otherID); err != nil || len(other) != 1 {
				t.Fatalf("other player's passkeys = %d (error %v)", len(other), err)
			}
			var pending int64
			if err := tx.Model(&models.PasskeyChallenge{}).Where("player_id = ?", playerID).Count(&pending).Error; err != nil || pending != 0 {
				t.Fatalf("pending passkey enrollments = %d (error %v)", pending, err)
			}
		})
	}
}
