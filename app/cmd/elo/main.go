// Command elo prints a per-group Elo rating table for the repo owner's own,
// private inspection. It is read-only: unlike cmd/seed, it never writes
// anything to the database — every rating is recomputed fresh, in memory,
// each time it runs (see internal/services/elo.go's ComputeEloStandings and
// CLAUDE.md's "Elo rating" section for why nothing is ever persisted).
//
// There is deliberately no HTTP route or frontend surface for this feature
// today — see CLAUDE.md — so this CLI is the only way to see the numbers at
// all right now.
//
// Usage (from app/, with a reachable Postgres — see CLAUDE.md's Gotchas
// section for the DB_HOST/DB_PORT/... env vars needed outside Docker):
//
//	go run ./cmd/elo -group "Default"
//	go run ./cmd/elo -group 3fa85f64-5717-4562-b3fc-2c963f66afa6
//	go run ./cmd/elo -all
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"text/tabwriter"

	"app/internal/models"
	"app/internal/services"
	"app/pkg/database"

	"github.com/google/uuid"
)

func main() {
	groupArg := flag.String("group", "", "print the Elo table for one group, looked up by exact name (case-insensitive) or by UUID")
	all := flag.Bool("all", false, "print the Elo table for every group in the database")
	flag.Parse()

	groupSet := *groupArg != ""
	if groupSet == *all {
		log.Fatalf("usage: exactly one of -group <name-or-id> or -all is required")
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	groupService := services.NewGroupService(db)
	groupMembershipService := services.NewGroupMembershipService(db)
	standingsService := services.NewStandingsService(db, groupMembershipService)

	groups, err := groupService.GetGroups()
	if err != nil {
		log.Fatalf("failed to load groups: %v", err)
	}

	var targets []models.Group
	if *all {
		targets = groups
	} else {
		group, err := resolveGroup(*groupArg, groups)
		if err != nil {
			log.Fatalf("%v", err)
		}
		targets = []models.Group{*group}
	}

	for i, group := range targets {
		if i > 0 {
			fmt.Println()
		}
		rows, err := standingsService.GetEloStandings(group.ID)
		if err != nil {
			log.Fatalf("failed to compute Elo standings for group %q (%s): %v", group.Name, group.ID, err)
		}
		printEloTable(group, rows)
	}
}

// resolveGroup finds one group by exact UUID or by case-insensitive exact
// name — tries UUID first, since a name is never itself a valid UUID string
// (the two spaces don't overlap), falling back to a name search otherwise.
func resolveGroup(value string, groups []models.Group) (*models.Group, error) {
	if id, err := uuid.Parse(value); err == nil {
		for i := range groups {
			if groups[i].ID == id {
				return &groups[i], nil
			}
		}
		return nil, fmt.Errorf("no group with id %s", id)
	}
	for i := range groups {
		if strings.EqualFold(groups[i].Name, value) {
			return &groups[i], nil
		}
	}
	return nil, fmt.Errorf("no group named %q", value)
}

// printEloTable prints one group's Elo table: rank, player, rating (rounded
// to the nearest integer for display only — GetEloStandings' own float64 is
// what every computation used, this is purely cosmetic), and games played.
// A departed player's name is suffixed " (left the group)" rather than the
// row being hidden or altered — same convention as PointsStandingsTable.vue/
// ScorersTable.vue's "(left the group)" tag on the frontend, just rendered
// as plain text here since there is no UI for this feature.
func printEloTable(group models.Group, rows []models.EloStandingRow) {
	fmt.Printf("Group: %s (%s)\n", group.Name, group.ID)
	if len(rows) == 0 {
		fmt.Println("  no completed matches yet")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "Rank\tPlayer\tRating\tGames")
	for i, row := range rows {
		name := row.Name
		if !row.IsMember {
			name += " (left the group)"
		}
		fmt.Fprintf(w, "%d\t%s\t%d\t%d\n", i+1, name, int(math.Round(row.Rating)), row.GamesPlayed)
	}
	w.Flush()
}
