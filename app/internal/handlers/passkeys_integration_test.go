package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/handlers"
	"app/internal/models"
	"app/internal/services"
	"app/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
)

func TestPasskeys_Integration_ChallengeAndAccountScope(t *testing.T) {
	const origin = "https://calciotto.example.test"
	const secondOrigin = "https://calciotto.example.test:8443"
	const keyPath = "/players/me/passkeys"
	const loginOptionsPath = "/auth/passkeys/options"
	const loginPath = "/auth/passkeys/login"
	tx := testutil.BeginTx(t, testutil.OpenDB(t))
	players := services.NewPlayerService(tx)
	auth := services.NewAuthService(tx, uuid.NewString())
	service, err := services.NewPasskeyService(auth, []string{origin, secondOrigin})
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	keys := handlers.NewPasskeyHandler(service, handlers.NewAuthHandler(auth))
	router.POST(keyPath+"/options", handlers.AuthMiddleware(auth), keys.RegistrationOptions)
	router.POST(keyPath, handlers.AuthMiddleware(auth), keys.Register)
	router.GET(keyPath, handlers.AuthMiddleware(auth), keys.List)
	router.DELETE(keyPath+"/:id", handlers.AuthMiddleware(auth), keys.Remove)
	router.POST(loginOptionsPath, keys.LoginOptions)
	router.POST(loginPath, keys.Login)
	password := uuid.NewString()
	newAccount := func(name, email string) (uuid.UUID, string) {
		t.Helper()
		id, err := players.CreatePlayer(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := auth.Signup(id, email, password); err != nil {
			t.Fatal(err)
		}
		token, err := auth.GenerateToken(id)
		if err != nil {
			t.Fatal(err)
		}
		return id, token
	}
	ownerID, ownerToken := newAccount("Passkey owner", "passkey-owner@example.test")
	otherID, otherToken := newAccount("Other player", "passkey-other@example.test")
	send := func(method, path, site, token string, body any, expectedStatus int) *httptest.ResponseRecorder {
		t.Helper()
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", site)
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != expectedStatus {
			t.Fatalf("%s %s status = %d, want %d", method, path, recorder.Code, expectedStatus)
		}
		return recorder
	}
	challengeCount := func(id uuid.UUID, want int64) {
		t.Helper()
		var count int64
		if err := tx.Model(&models.PasskeyChallenge{}).Where("id = ?", id).Count(&count).Error; err != nil || count != want {
			t.Fatalf("challenge count = %d, want %d (error %v)", count, want, err)
		}
	}
	credentials := map[string]string{"password": password, "name": "Phone"}
	send(http.MethodPost, keyPath+"/options", origin, "", credentials, http.StatusUnauthorized)
	send(http.MethodGet, keyPath, origin, ownerToken, nil, http.StatusForbidden)
	addFeature := func(id uuid.UUID, feature string) {
		t.Helper()
		if err := tx.Create(&models.PlayerFeature{PlayerID: id, Feature: feature}).Error; err != nil {
			t.Fatal(err)
		}
	}
	// design-v2 is launched to everyone; a row for it changes nothing, the
	// passkeys row is what opens enrollment.
	addFeature(ownerID, models.FeatureDesignV2)
	send(http.MethodPost, keyPath+"/options", origin, ownerToken, credentials, http.StatusForbidden)
	send(http.MethodPost, keyPath+"/options", origin, otherToken, credentials, http.StatusForbidden)
	addFeature(ownerID, models.FeaturePasskeys)
	addFeature(otherID, models.FeaturePasskeys)
	send(http.MethodPost, keyPath+"/options", origin, ownerToken, map[string]string{"password": uuid.NewString()}, http.StatusBadRequest)
	enrollment := send(http.MethodPost, keyPath+"/options", origin, ownerToken, credentials, http.StatusOK)
	var creation struct {
		ID        uuid.UUID                                   `json:"challenge_id"`
		PublicKey protocol.PublicKeyCredentialCreationOptions `json:"publicKey"`
	}
	if err := json.Unmarshal(enrollment.Body.Bytes(), &creation); err != nil {
		t.Fatal(err)
	}
	selection := creation.PublicKey.AuthenticatorSelection
	if creation.PublicKey.RelyingParty.ID != "calciotto.example.test" || selection.UserVerification != protocol.VerificationRequired || selection.ResidentKey != protocol.ResidentKeyRequirementRequired {
		t.Fatal("enrollment must require user verification and a discoverable credential for the frontend host")
	}
	proof := map[string]any{"challenge_id": creation.ID, "credential": map[string]any{}}
	send(http.MethodPost, keyPath, origin, otherToken, proof, http.StatusBadRequest)
	send(http.MethodPost, keyPath, secondOrigin, ownerToken, proof, http.StatusBadRequest)
	send(http.MethodPost, loginPath, origin, "", proof, http.StatusUnauthorized)
	challengeCount(creation.ID, 1)
	if err := tx.Delete(&models.PlayerFeature{PlayerID: ownerID, Feature: models.FeaturePasskeys}).Error; err != nil {
		t.Fatal(err)
	}
	send(http.MethodPost, keyPath, origin, ownerToken, proof, http.StatusForbidden)
	send(http.MethodDelete, keyPath+"/"+uuid.NewString(), origin, ownerToken, nil, http.StatusForbidden)
	addFeature(ownerID, models.FeaturePasskeys)
	send(http.MethodPost, keyPath, origin, ownerToken, proof, http.StatusBadRequest)
	challengeCount(creation.ID, 0)

	send(http.MethodPost, loginOptionsPath, "https://other.example.test", "", nil, http.StatusForbidden)
	send(http.MethodPost, loginOptionsPath, "", "", nil, http.StatusForbidden)
	loginOptions := send(http.MethodPost, loginOptionsPath, origin, "", nil, http.StatusOK)
	var assertion struct {
		ID        uuid.UUID                                  `json:"challenge_id"`
		PublicKey protocol.PublicKeyCredentialRequestOptions `json:"publicKey"`
	}
	if err := json.Unmarshal(loginOptions.Body.Bytes(), &assertion); err != nil {
		t.Fatal(err)
	}
	if assertion.PublicKey.UserVerification != protocol.VerificationRequired || len(assertion.PublicKey.AllowedCredentials) != 0 || loginOptions.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("sign-in options must require verification without disclosing credentials")
	}
	proof["challenge_id"] = assertion.ID
	if err := tx.Model(&models.PasskeyChallenge{}).Where("id = ?", assertion.ID).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	rejected := send(http.MethodPost, loginPath, origin, "", proof, http.StatusUnauthorized)
	if len(rejected.Result().Cookies()) != 0 {
		t.Fatal("an expired challenge issued a session cookie")
	}
	challengeCount(assertion.ID, 1)
	send(http.MethodPost, loginOptionsPath, origin, "", nil, http.StatusOK)
	challengeCount(assertion.ID, 0)

	key := models.Passkey{PlayerID: ownerID, RPID: "calciotto.example.test", CredentialID: []byte("fixture-id"), Name: "Owner phone"}
	if err := tx.Create(&key).Error; err != nil {
		t.Fatal(err)
	}
	if other := send(http.MethodGet, keyPath, origin, otherToken, nil, http.StatusOK); other.Body.String() != "[]" {
		t.Fatal("another account saw the passkey")
	}
	send(http.MethodDelete, keyPath+"/"+key.ID.String(), origin, otherToken, nil, http.StatusNoContent)
	var stored models.Passkey
	if err := tx.Take(&stored, "id = ?", key.ID).Error; err != nil {
		t.Fatal("another account removed the passkey")
	}
	send(http.MethodDelete, keyPath+"/"+key.ID.String(), origin, ownerToken, nil, http.StatusNoContent)
	if remaining := send(http.MethodGet, keyPath, origin, ownerToken, nil, http.StatusOK); remaining.Body.String() != "[]" {
		t.Fatal("the removed passkey was still listed")
	}
}
