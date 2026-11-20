package services_test

import (
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"
)

// result_recorded tells the service worker which recipients are on design
// v2, so it can name the vote the way their screens do: designV2 is true for
// a player with the design-v2 feature and absent for the others.
func TestNotifications_Integration_ResultRecordedFlagsDesignV2(t *testing.T) {
	env := newNotificationEnv(t)
	alice, aliceEndpoint := env.member(t, "Alice", true)
	bob, bobEndpoint := env.member(t, "Bob", true)
	if err := env.tx.Create(&models.PlayerFeature{PlayerID: alice, Feature: models.FeatureDesignV2}).Error; err != nil {
		t.Fatalf("enable design-v2: %v", err)
	}

	kickoff := time.Now().Add(-2 * time.Hour)
	matchID := env.scheduledMatch(t, kickoff, kickoff.Add(-72*time.Hour), 10)
	if err := env.matches.UpdateMatch(matchID, env.groupID, []models.TeamWithPlayers{
		{ID: env.teamA.ID, Players: []models.PlayerCustom{{ID: alice, GoalsScored: 2}}},
		{ID: env.teamB.ID, Players: []models.PlayerCustom{{ID: bob, GoalsScored: 1}}},
	}); err != nil {
		t.Fatalf("UpdateMatch (result): %v", err)
	}

	aliceResult := messagesOfType(env.sender.sent[aliceEndpoint], services.PushTypeResultRecorded)
	if len(aliceResult) != 1 || !aliceResult[0].DesignV2 {
		t.Fatalf("Alice (design-v2) got %+v, want one result_recorded with designV2", env.sender.sent[aliceEndpoint])
	}
	bobResult := messagesOfType(env.sender.sent[bobEndpoint], services.PushTypeResultRecorded)
	if len(bobResult) != 1 || bobResult[0].DesignV2 {
		t.Fatalf("Bob (v1) got %+v, want one result_recorded without designV2", env.sender.sent[bobEndpoint])
	}
}
