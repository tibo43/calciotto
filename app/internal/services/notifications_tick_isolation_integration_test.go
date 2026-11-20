package services_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"app/internal/models"
	"app/internal/services"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// One step of the tick failing (here, the predictions_settled query) must not
// stop the others: sign-ups still open, and the error names the failed step.
func TestNotifications_Integration_TickFailingStepDoesNotStopTheOthers(t *testing.T) {
	env := newNotificationEnv(t)
	now := time.Now()
	_, readerEndpoint := env.member(t, "Reader", true)
	matchID := env.scheduledMatch(t, now.Add(72*time.Hour), now.Add(-10*time.Minute), 10)

	// A second handle on the same transaction, with its own callbacks, so the
	// injected failure stays out of every other test.
	failing, err := gorm.Open(postgres.New(postgres.Config{Conn: env.tx.Statement.ConnPool}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	injected := errors.New("injected query failure")
	if err := failing.Callback().Query().After("gorm:query").Register("test:fail_settled", func(db *gorm.DB) {
		if strings.Contains(db.Statement.SQL.String(), "predictions_settled_notified_at IS NULL") {
			db.AddError(injected)
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	notifications := services.NewNotificationService(failing, env.sender, services.NewMatchService(failing))

	result, err := notifications.Tick(now)
	if !errors.Is(err, injected) || !strings.Contains(err.Error(), services.PushTypePredictionsSettled) {
		t.Fatalf("Tick error = %v, want the injected failure, named after predictions_settled", err)
	}
	if result.RegistrationOpen != 1 {
		t.Errorf("RegistrationOpen = %d, want 1: the other steps must still run", result.RegistrationOpen)
	}
	if got := env.sender.sent[readerEndpoint]; len(got) != 1 || got[0].Type != services.PushTypeRegistrationOpen || got[0].URL != services.MatchDeepLink(matchID) {
		t.Errorf("member got %+v, want one registration_open", got)
	}
	var match models.Match
	if err := env.tx.First(&match, "id = ?", matchID).Error; err != nil || match.RegistrationOpenNotifiedAt == nil {
		t.Errorf("registration_open_notified_at not set (%v)", err)
	}
}
