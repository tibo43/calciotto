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
// It is intended to run once against production, pointed at the real
// database via DATABASE_URL — the same connection selection InitDB always
// uses (see pkg/database/connect.go's appDSN, which prefers DATABASE_URL over
// DB_HOST/... whenever it's set). Run it like:
//
//	DATABASE_URL="postgres://...neon.tech/..." go run ./cmd/perfsetup
//
// Every created row's ID is printed at the end specifically so
// devops/perf-cleanup.sql can be filled in and run afterwards — there is no
// automatic teardown here, on purpose: a load test needs a deliberate,
// reviewed cleanup step, not one bundled into the setup script that created
// the data in the first place.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/pkg/database"
)

func main() {
	groupName := flag.String("group-name", "Perf Load Test", "name of the throwaway group to create")
	adminEmail := flag.String("admin-email", "perfload-admin@perfload.test", "email for the throwaway admin account (reserved .test TLD, never emailed)")
	adminPassword := flag.String("admin-password", "PerfLoad!2026", "password for the throwaway admin account")
	maxPlayers := flag.Int("max-players", 60, "MaxPlayers on the scheduled match — keep it above the number of accounts you intend to sign up, or some will land on the waiting list on purpose")
	kickoffInDays := flag.Int("kickoff-in-days", 7, "days from now for the match's kick-off — far enough out that it won't close registrations mid-test")
	frontendURL := flag.String("frontend-url", "", "optional: your Vercel frontend URL, to print a ready-to-use signup link (e.g. https://calciotto.example.com)")
	flag.Parse()

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	groupService := services.NewGroupService(db)
	playerService := services.NewPlayerService(db)
	authService := services.NewAuthService(db, "perfsetup-only-unused-jwt-secret")
	membershipService := services.NewGroupMembershipService(db)
	matchService := services.NewMatchService(db)

	group, err := groupService.CreateGroup(*groupName, [2]services.TeamSpec{
		{Name: "Red", Colour: "red"},
		{Name: "Blue", Colour: "blue"},
	})
	if err != nil {
		log.Fatalf("failed to create group: %v", err)
	}

	adminID, err := playerService.CreatePlayer("Perf Admin")
	if err != nil {
		log.Fatalf("failed to create admin player: %v", err)
	}
	if err := authService.Signup(adminID, *adminEmail, *adminPassword); err != nil {
		log.Fatalf("failed to attach credentials to admin player: %v", err)
	}
	if err := membershipService.AddPlayerToGroupWithRole(group.ID, adminID, models.RoleAdmin); err != nil {
		log.Fatalf("failed to add admin to group: %v", err)
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
	fmt.Printf("Admin login:  %s / %s\n", *adminEmail, *adminPassword)
	fmt.Printf("Match id:     %s\n", matchID)
	fmt.Printf("Kick-off:     %s\n", kickoff.Format(time.RFC3339))
	fmt.Printf("Max players:  %d\n", *maxPlayers)
	if *frontendURL != "" {
		fmt.Printf("Signup link:  %s/signup?invite=%s\n", *frontendURL, group.InviteCode)
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run the Playwright load script (frontend/loadtest/run.js) with this invite code.")
	fmt.Println("  2. Once done, fill in the group id above into devops/perf-cleanup.sql and run it against the same database.")
	fmt.Println()
}
