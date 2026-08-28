package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"app/internal/models"
	"app/internal/services"
	"app/pkg/database"

	"gorm.io/gorm"
)

// seedAuthSecret signs tokens AuthService.Signup never issues here — the seed
// script only claims each player's account, it never logs in — so this value
// has no security relevance and doesn't need to match the backend's real
// JWT_SECRET.
const seedAuthSecret = "seed-only-unused-jwt-secret"

// seedPassword is the shared password for every seeded player account, so
// anyone can log in as any of them locally without juggling per-player
// credentials.
const seedPassword = "user1234"

var playerNames = []string{
	"thibaut", "matthias", "manfredi", "damien", "vincent", "pierre",
	"anthony", "jacopo", "mattheo", "ryan", "connor", "marcos",
}

func main() {
	reset := flag.Bool("reset", false, "wipe existing players/teams/matches before seeding")
	matchCount := flag.Int("matches", 6, "number of past matches to generate")
	flag.Parse()

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if *reset {
		log.Println("resetting existing data...")
		if err := resetData(db); err != nil {
			log.Fatalf("failed to reset data: %v", err)
		}
	}

	var existingPlayers int64
	db.Model(&models.Player{}).Count(&existingPlayers)
	if existingPlayers > 0 {
		log.Println("database already contains players — skipping seed (rerun with -reset to start fresh)")
		return
	}

	playerService := services.NewPlayerService(db)
	groupService := services.NewGroupService(db)
	groupMembershipService := services.NewGroupMembershipService(db)
	teamService := services.NewTeamService(db)
	matchService := services.NewMatchService(db)
	matchPlayerService := services.NewMatchPlayerService(db)
	authService := services.NewAuthService(db, seedAuthSecret)

	group, err := groupService.CreateGroup("Default")
	if err != nil {
		log.Fatalf("failed to create default group: %v", err)
	}
	log.Printf("created group %q (%s)", group.Name, group.ID)

	players := make([]models.Player, 0, len(playerNames))
	for _, name := range playerNames {
		id, err := playerService.CreatePlayer(name)
		if err != nil {
			log.Fatalf("failed to create player %q: %v", name, err)
		}
		if err := groupMembershipService.AddPlayerToGroup(group.ID, id); err != nil {
			log.Fatalf("failed to add player %q to group: %v", name, err)
		}
		email := name + "@mail.com"
		if err := authService.Signup(id, email, seedPassword); err != nil {
			log.Fatalf("failed to sign up player %q: %v", name, err)
		}
		players = append(players, models.Player{BaseModel: models.BaseModel{ID: id}, Name: name})
	}
	log.Printf("created %d players, each with an account (name@mail.com / %s)", len(players), seedPassword)

	teams, err := teamService.GetTeamsByGroupID(group.ID)
	if err != nil {
		log.Fatalf("failed to load group's teams: %v", err)
	}
	log.Printf("group has %d teams", len(teams))

	for _, date := range lastSundays(*matchCount) {
		matchID, err := matchService.CreateMatch(date, group.ID)
		if err != nil {
			log.Fatalf("failed to create match: %v", err)
		}

		shuffled := make([]models.Player, len(players))
		copy(shuffled, players)
		rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

		mid := len(shuffled) / 2
		roster := [][]models.Player{shuffled[:mid], shuffled[mid:]}

		for i, team := range teams {
			for _, player := range roster[i] {
				mp := &models.MatchPlayer{
					MatchID:     matchID,
					TeamID:      team.ID,
					PlayerID:    player.ID,
					GoalsScored: weightedGoals(),
				}
				if err := matchPlayerService.CreateMatchPlayer(mp); err != nil {
					log.Fatalf("failed to assign player %q to match: %v", player.Name, err)
				}
			}
		}
		log.Printf("created match on %s with %d players", date, len(shuffled))
	}

	log.Println("done.")
}

// resetData wipes seeded tables in FK-safe order (children before parents).
func resetData(db *gorm.DB) error {
	for _, model := range []interface{}{&models.MatchPlayer{}, &models.GroupMembership{}, &models.Match{}, &models.Team{}, &models.Player{}, &models.Group{}} {
		if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
			return err
		}
	}
	return nil
}

// lastSundays returns the last n Sundays, oldest first.
func lastSundays(n int) []models.Date {
	dates := make([]models.Date, 0, n)
	now := time.Now()
	lastSunday := now.AddDate(0, 0, -int(now.Weekday()))
	for i := n - 1; i >= 0; i-- {
		dates = append(dates, models.Date(lastSunday.AddDate(0, 0, -7*i)))
	}
	return dates
}

// weightedGoals mimics a realistic scoreline: mostly 0, occasionally a goal, rarely a brace or more.
func weightedGoals() int {
	switch r := rand.Intn(10); {
	case r < 6:
		return 0
	case r < 9:
		return 1
	default:
		return rand.Intn(3) + 2
	}
}
