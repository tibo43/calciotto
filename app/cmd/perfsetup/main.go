// cmd/perfsetup is a one-off tool for a load-test dry run: it creates,
// directly through the service layer (bypassing HTTP — POST /groups is
// disabled, see CLAUDE.md's "Group bootstrapping" section), exactly the three
// things a load test needs before any UI-driven traffic starts:
//
//  1. A fresh group with two teams, isolated from every real group so the
//     test's 50 accounts never touch real data.
//  2. An admin player with real credentials, so whoever runs the test can log
//     in and watch the sign-up list fill up, close registrations, etc.
//  3. A scheduled match in that group with sign-ups already open (kick-off
//     days in the future, registration opened in the past), so 50 real
//     signup-then-Participate flows driven through the UI have something to
//     register for immediately.
//
// Run it against a disposable local calciotto_perf database using the
// DB_* variables. See frontend/loadtest/README.md for the local setup.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/pkg/database"

	"github.com/google/uuid"
)

func main() {
	groupName := flag.String("group-name", "Perf Load Test", "name of the throwaway group to create")
	adminEmail := flag.String("admin-email", "perfload-admin@perfload.test", "email for the throwaway admin account (reserved .test TLD, never emailed)")
	adminPassword := flag.String("admin-password", "", "password for the throwaway admin account (defaults to PERF_ADMIN_PASSWORD)")
	maxPlayers := flag.Int("max-players", 60, "MaxPlayers on the scheduled match — keep it above the number of accounts you intend to sign up, or some will land on the waiting list on purpose")
	kickoffInDays := flag.Int("kickoff-in-days", 7, "days from now for the match's kick-off — far enough out that it won't close registrations mid-test")
	frontendURL := flag.String("frontend-url", "", "optional loopback frontend origin for printing a local signup link")
	existingPlayerID := flag.String("existing-player-id", "", "optional UUID of a player in the local test database to add as an admin")
	flag.Parse()

	password := *adminPassword
	if password == "" {
		password = os.Getenv("PERF_ADMIN_PASSWORD")
	}
	if len(password) == 0 || len(password) > 72 {
		log.Fatal("set PERF_ADMIN_PASSWORD to a password of 1 to 72 bytes for the local test admin")
	}
	if *frontendURL != "" {
		frontend, err := url.Parse(*frontendURL)
		if err != nil || (frontend.Scheme != "http" && frontend.Scheme != "https") ||
			!net.ParseIP(frontend.Hostname()).IsLoopback() || frontend.User != nil ||
			frontend.RawQuery != "" || frontend.Fragment != "" || (frontend.Path != "" && frontend.Path != "/") {
			log.Fatal("-frontend-url must be a loopback HTTP(S) origin")
		}
	}

	db, err := database.InitLocalPerfDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	groupService := services.NewGroupService(db)
	playerService := services.NewPlayerService(db)
	authService := services.NewAuthService(db, "perfsetup-only-unused-jwt-secret")
	membershipService := services.NewGroupMembershipService(db)
	matchService := services.NewMatchService(db)

	group, err := groupService.CreateGroup(*groupName, [2]services.TeamSpec{
		{Name: "Red"},
		{Name: "Blue"},
	})
	if err != nil {
		log.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Perf Admin")
	if err != nil {
		log.Fatalf("failed to create admin player: %v", err)
	}
	if err := authService.Signup(adminID, *adminEmail, password); err != nil {
		log.Fatalf("failed to attach credentials to admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		log.Fatalf("failed to add admin to group: %v", err)
	}

	var existingPlayerName string
	if *existingPlayerID != "" {
		id, err := uuid.Parse(*existingPlayerID)
		if err != nil {
			log.Fatalf("invalid -existing-player-id %q: %v", *existingPlayerID, err)
		}
		player, err := playerService.GetPlayerByID(id)
		if err != nil {
			log.Fatalf("no player found with id %s: %v", id, err)
		}
		existingPlayerName = player.Name
		isMember, err := membershipService.IsMember(group.ID, id)
		if err != nil {
			log.Fatalf("failed to check membership for %s: %v", id, err)
		}
		if !isMember {
			if err := membershipService.AddPlayerToGroupWithRole(group.ID, id, models.RoleAdmin); err != nil {
				log.Fatalf("failed to add existing player %s to group: %v", id, err)
			}
		}
	}

	now := time.Now()
	registrationOpensAt := now.Add(-1 * time.Hour)
	kickoff := now.AddDate(0, 0, *kickoffInDays)
	matchID, err := matchService.CreateMatch(services.MatchSpec{
		ScheduledAt:         &kickoff,
		RegistrationOpensAt: &registrationOpensAt,
		MaxPlayers:          maxPlayers,
	}, group.ID)
	if err != nil {
		log.Fatalf("failed to create scheduled match: %v", err)
	}

	fmt.Println()
	fmt.Println("=== Perf load-test fixtures created ===")
	fmt.Printf("Group:        %s (id=%s)\n", group.Name, group.ID)
	fmt.Printf("Invite code:  %s\n", group.InviteCode)
	fmt.Printf("Admin email:  %s\n", *adminEmail)
	if *existingPlayerID != "" {
		fmt.Printf("Also added:   %s (id=%s) as admin — log in on your normal account to watch\n", existingPlayerName, *existingPlayerID)
	}
	fmt.Printf("Match id:     %s\n", matchID)
	fmt.Printf("Kick-off:     %s\n", kickoff.Format(time.RFC3339))
	fmt.Printf("Max players:  %d\n", *maxPlayers)
	if *frontendURL != "" {
		fmt.Printf("Signup link:  %s/signup?invite=%s\n", strings.TrimRight(*frontendURL, "/"), group.InviteCode)
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run the Playwright load script (frontend/loadtest/run.js) with this invite code.")
	fmt.Println("  2. Remove the disposable calciotto_perf database when finished.")
	fmt.Println()
}
