package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"

	"app/internal/services"
)

const errorCatalogueFile = "testdata/error_codes.json"

// The frontend reads this file to check that every code has a message in every
// language, and that the English message is the one the API sends. After
// adding or rewording an error, regenerate it with UPDATE_ERROR_CODES=1.
func TestErrorCatalogue_MatchesTheGoldenFile(t *testing.T) {
	want, err := json.MarshalIndent(errorCatalogue(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	want = append(want, '\n')
	if os.Getenv("UPDATE_ERROR_CODES") != "" {
		if err := os.WriteFile(errorCatalogueFile, want, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(errorCatalogueFile)
	if err != nil {
		t.Fatalf("read %s: %v", errorCatalogueFile, err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s is out of date; regenerate it with UPDATE_ERROR_CODES=1 go test ./internal/handlers -run ErrorCatalogue", errorCatalogueFile)
	}
}

func TestErrorCodes_AreUniqueSnakeCase(t *testing.T) {
	snake := regexp.MustCompile(`^[a-z]+(_[a-z]+)*$`)
	seen := map[string]string{}
	check := func(code, what string) {
		if !snake.MatchString(code) {
			t.Errorf("code %q (%s) is not snake_case", code, what)
		}
		if other, dup := seen[code]; dup {
			t.Errorf("code %q is used by %s and %s", code, other, what)
		}
		seen[code] = what
	}
	for _, entry := range sentinelCodes {
		check(entry.code, entry.err.Error())
	}
	for message, code := range messageCodes {
		check(code, message)
	}
}

func TestErrorBody_CarriesTheCodeAndKeepsTheEnglishText(t *testing.T) {
	body := errorBody(services.ErrRegistrationsClosed)
	if body["error"] != "registrations for this match are closed" || body["code"] != "registrations_closed" {
		t.Errorf("body = %v", body)
	}
	if _, has := body["params"]; has {
		t.Errorf("body = %v, no params expected", body)
	}
}

func TestErrorBody_FindsAWrappedError(t *testing.T) {
	body := errorBody(fmt.Errorf("saving: %w", services.ErrVotingClosed))
	if body["code"] != "voting_closed" || body["error"] != "saving: voting for MVP is closed for this match" {
		t.Errorf("body = %v", body)
	}
}

func TestErrorBody_NoCodeForAnUnlistedError(t *testing.T) {
	body := errorBody(errors.New("database is down"))
	if _, has := body["code"]; has || body["error"] != "database is down" {
		t.Errorf("body = %v", body)
	}
}

func TestErrorBody_NamesTheMinimumPasswordLength(t *testing.T) {
	body := errorBody(services.ErrPasswordTooShort)
	params, _ := body["params"].(map[string]any)
	if body["code"] != "password_too_short" {
		t.Fatalf("body = %v", body)
	}
	if got, want := fmt.Sprint(body["params"]), fmt.Sprint(map[string]any{"min": services.MinPasswordLength}); got != want {
		t.Errorf("params = %v (%v), want %v", got, params, want)
	}
}

func TestMessageBody_CodesOnlyTheListedMessages(t *testing.T) {
	if body := messageBody("incorrect password"); body["code"] != "incorrect_password" {
		t.Errorf("body = %v", body)
	}
	if body := messageBody("something else"); body["code"] != nil || body["error"] != "something else" {
		t.Errorf("body = %v", body)
	}
}

func TestMessageBody_ACodedServiceErrorTextKeepsItsCode(t *testing.T) {
	if body := messageBody("not an admin of this group"); body["code"] != "admin_required" {
		t.Errorf("body = %v", body)
	}
}
