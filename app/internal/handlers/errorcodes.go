package handlers

import (
	"errors"
	"strconv"
	"strings"

	"app/internal/services"

	"github.com/gin-gonic/gin"
)

// Every error response keeps its English "error" text, and adds a stable
// "code" when the error is one the app shows to a person. The frontend writes
// the message in the player's language from the code (apiError.<code> in
// src/locales/*.json) and falls back to the English text for a response with
// no code. Codes are snake_case and are never renamed: they are the contract
// with the bundles. testdata/error_codes.json lists every code with its
// English text, and the frontend tests read it to check the bundles.

type errorCode struct {
	err  error
	code string
}

// sentinelCodes are the errors of the services layer, matched with errors.Is
// (so a wrapped error keeps its code). Server-side failures (a lost database,
// a cursor or page size only a script can get wrong) have no code on purpose.
var sentinelCodes = []errorCode{
	{services.ErrEmailRequired, "email_required"},
	{services.ErrPasswordRequired, "password_required"},
	{services.ErrPasswordTooShort, "password_too_short"},
	{services.ErrPlayerNotFound, "player_not_found"},
	{services.ErrPlayerAlreadyClaimed, "player_already_claimed"},
	{services.ErrEmailAlreadyUsed, "email_already_used"},
	{services.ErrInvalidCredentials, "invalid_credentials"},
	{services.ErrInvalidToken, "invalid_token"},
	{services.ErrInvalidResetToken, "invalid_reset_token"},
	{services.ErrPasswordResetUnavailable, "password_reset_unavailable"},
	{services.ErrInviteCodeRequired, "invite_code_required"},
	{services.ErrInviteCodeNotFound, "invite_code_not_found"},
	{services.ErrAlreadyMember, "already_member"},
	{services.ErrEmptyPlayerName, "player_name_required"},
	{services.ErrPlayerNameAlreadyUsed, "player_name_taken"},
	{services.ErrInvalidPosition, "invalid_position"},
	{services.ErrTeamNameRequired, "team_name_required"},
	{services.ErrGroupDescriptionTooLong, "group_description_too_long"},
	{services.ErrGroupWebsiteLabelTooLong, "group_link_label_too_long"},
	{services.ErrInvalidGroupWebsite, "group_link_invalid"},
	{services.ErrLastMember, "last_member"},
	{services.ErrCannotRemoveSelf, "cannot_remove_self"},
	{services.ErrInvalidRole, "invalid_role"},
	{services.ErrCannotChangeOwnRole, "cannot_change_own_role"},
	{services.ErrLastAdmin, "last_admin"},
	{services.ErrGroupAdminRequired, "admin_required"},
	{services.ErrMatchNotFound, "match_not_found"},
	{services.ErrTeamNotInGroup, "team_not_in_group"},
	{services.ErrPlayerNotInGroup, "roster_player_not_in_group"},
	{services.ErrIncompleteSchedule, "incomplete_schedule"},
	{services.ErrRegistrationOpensAfterKickoff, "registration_opens_after_kickoff"},
	{services.ErrInvalidMaxPlayers, "invalid_max_players"},
	{services.ErrRegisterCreatorOnUnscheduledMatch, "creator_on_unscheduled_match"},
	{services.ErrMatchNotScheduled, "match_not_scheduled"},
	{services.ErrRegistrationsNotOpenYet, "registrations_not_open_yet"},
	{services.ErrRegistrationsClosed, "registrations_closed"},
	{services.ErrAlreadyRegistered, "already_registered"},
	{services.ErrNotRegistered, "not_registered"},
	{services.ErrPredictionsNotAvailable, "predictions_not_available"},
	{services.ErrPredictionsClosed, "predictions_closed"},
	{services.ErrPredictedTeamNotInMatch, "predicted_team_not_in_match"},
	{services.ErrCannotVoteForSelf, "cannot_vote_for_self"},
	{services.ErrVotedForPlayerNotOnRoster, "voted_player_not_on_roster"},
	{services.ErrVoterNotOnRoster, "voter_not_on_roster"},
	{services.ErrVotingClosed, "voting_closed"},
	{services.ErrPasskeyOrigin, "passkeys_unavailable_origin"},
	{services.ErrPasskeysDisabled, "passkeys_unavailable_account"},
	{services.ErrPasskeyName, "passkey_name_too_long"},
	{services.ErrPasskeyProof, "passkey_not_verified"},
	{services.ErrInvalidPushSubscription, "invalid_push_subscription"},
	{services.ErrUnsupportedLocale, "unsupported_language"},
}

// messageCodes are the handler's own English messages that a person can see
// (a rate limit, a wrong password, a feature that is not open yet), matched
// by their exact text, or the text of a service error.
var messageCodes = map[string]string{
	"too many requests, please try again later":         "rate_limited",
	"too many attempts, try again later":                "too_many_attempts",
	"incorrect password":                                "incorrect_password",
	"request body is too large":                         "body_too_large",
	"the Agoranking is not enabled for this account":    "agoranking_not_enabled",
	"joining a group is not available yet":              "joining_not_available",
	"group creation is not available yet":               "group_creation_not_available",
	"passkeys are temporarily unavailable":              "passkeys_temporarily_unavailable",
	"invalid group details":                             "invalid_group_details",
	"player is not a member of this group":              "player_not_in_this_group",
	"not a member of this group":                        "not_a_member",
	"group not found":                                   "group_not_found",
	"team not found in this group":                      "team_not_found",
	"player or group not found":                         "player_or_group_not_found",
	"unable to update the invitation":                   "invitation_update_failed",
	"unable to complete signup":                         "signup_failed",
	"unable to add the player":                          "add_player_failed",
	"could not save your positions":                     "positions_save_failed",
	"could not load your positions":                     "positions_load_failed",
	"could not save group details":                      "group_details_save_failed",
	"could not load positions":                          "group_positions_load_failed",
	"could not load your language":                      "language_load_failed",
	"could not save your language":                      "language_save_failed",
	"authenticated player does not belong to any group": "not_in_any_group",
}

// errorBody is the JSON body of an error response for a service error.
func errorBody(err error) gin.H {
	body := gin.H{"error": err.Error()}
	for _, entry := range sentinelCodes {
		if errors.Is(err, entry.err) {
			body["code"] = entry.code
			if entry.err == services.ErrPasswordTooShort {
				body["params"] = gin.H{"min": services.MinPasswordLength}
			}
			break
		}
	}
	return body
}

// messageBody is the JSON body of an error response for a fixed message.
func messageBody(message string) gin.H {
	body := gin.H{"error": message}
	if code, ok := messageCodes[message]; ok {
		body["code"] = code
		return body
	}
	// The same text as a service error ("not an admin of this group") has
	// that error's code.
	for _, entry := range sentinelCodes {
		if entry.err.Error() == message {
			body["code"] = entry.code
			break
		}
	}
	return body
}

// errorCatalogue is every code with its English text, sorted by code: what
// testdata/error_codes.json holds.
func errorCatalogue() map[string]string {
	catalogue := map[string]string{}
	for _, entry := range sentinelCodes {
		message := entry.err.Error()
		if entry.err == services.ErrPasswordTooShort {
			message = strings.Replace(message, strconv.Itoa(services.MinPasswordLength), "{min}", 1)
		}
		catalogue[entry.code] = message
	}
	for message, code := range messageCodes {
		catalogue[code] = message
	}
	return catalogue
}
