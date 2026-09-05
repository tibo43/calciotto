// cmd/addmember is a one-off admin fix for a player who ended up in no group
// at all (e.g. an account created before invite codes were mandatory, or one
// whose only membership was removed) — it adds them to a specific group as a
// plain member, directly through the service layer, the same reasoning
// cmd/perfsetup uses for bypassing the disabled POST /groups: there is no
// HTTP route for an admin to add a player who isn't already a member of any
// group of their own to add them via.
//
// It deliberately does NOT take an -is-favorite flag: GroupMembershipService.
// AddPlayerToGroupWithRole already sets IsFavorite automatically on a
// player's very first membership ever (see its own doc comment), which is
// exactly this tool's use case — a player with zero existing memberships.
// The tool verifies that precondition itself and refuses (rather than
// silently leaving the player with a membership but no favorite, or with two
// favorites) if the player unexpectedly already belongs to a group.
//
// Run against production the same way as cmd/perfsetup:
//
//	DATABASE_URL="postgres://...neon.tech/..." go run ./cmd/addmember \
//	  -player-id b939bdc7-7e7f-45bb-81a1-7d11d0e2ea3e \
//	  -group-id a9fc8c47-0b76-42f6-9ff5-1e2b0f094ea0
package main

import (
	"flag"
	"fmt"
	"log"

	"app/internal/models"
	"app/internal/services"
	"app/pkg/database"

	"github.com/google/uuid"
)

func main() {
	playerIDFlag := flag.String("player-id", "", "UUID of the player to add (required)")
	groupIDFlag := flag.String("group-id", "", "UUID of the group to add them to (required)")
	role := flag.String("role", models.RoleMember, "role to grant: \"member\" or \"admin\"")
	flag.Parse()

	if *playerIDFlag == "" || *groupIDFlag == "" {
		log.Fatal("both -player-id and -group-id are required")
	}
	if *role != models.RoleMember && *role != models.RoleAdmin {
		log.Fatalf("invalid -role %q: must be %q or %q", *role, models.RoleMember, models.RoleAdmin)
	}

	playerID, err := uuid.Parse(*playerIDFlag)
	if err != nil {
		log.Fatalf("invalid -player-id %q: %v", *playerIDFlag, err)
	}
	groupID, err := uuid.Parse(*groupIDFlag)
	if err != nil {
		log.Fatalf("invalid -group-id %q: %v", *groupIDFlag, err)
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	playerService := services.NewPlayerService(db)
	membershipService := services.NewGroupMembershipService(db)

	player, err := playerService.GetPlayerByID(playerID)
	if err != nil {
		log.Fatalf("no player found with id %s: %v", playerID, err)
	}

	var group models.Group
	if err := db.First(&group, "id = ?", groupID).Error; err != nil {
		log.Fatalf("no group found with id %s: %v", groupID, err)
	}

	isMember, err := membershipService.IsMember(groupID, playerID)
	if err != nil {
		log.Fatalf("failed to check membership: %v", err)
	}
	if isMember {
		log.Fatalf("%q (%s) is already a member of %q — nothing to do", player.Name, playerID, group.Name)
	}

	existingGroups, err := membershipService.GetGroupsByPlayerID(playerID)
	if err != nil {
		log.Fatalf("failed to check existing memberships: %v", err)
	}
	if len(existingGroups) > 0 {
		log.Fatalf(
			"%q (%s) already belongs to %d group(s) — this tool only handles a player with zero groups (where AddPlayerToGroupWithRole's automatic IsFavorite=true is guaranteed correct). Use PATCH /groups/:id/favorite instead if a favorite needs to move.",
			player.Name, playerID, len(existingGroups),
		)
	}

	if err := membershipService.AddPlayerToGroupWithRole(groupID, playerID, *role); err != nil {
		log.Fatalf("failed to add player to group: %v", err)
	}

	fmt.Printf("Added %q (%s) to %q (%s) as %s, is_favorite=true (their first and only group).\n",
		player.Name, playerID, group.Name, groupID, *role)
}
