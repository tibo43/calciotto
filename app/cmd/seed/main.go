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

var playerNames = []string{
	"thibaut", "matthias", "manfredi", "damien", "vincent", "pierre",
	"anthony", "jacopo", "mattheo", "ryan", "connor", "marcos",
}

var teamColours = []string{"black", "white"}

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
	teamService := services.NewTeamService(db)
	matchService := services.NewMatchService(db)
	matchPlayerService := services.NewMatchPlayerService(db)

	players := make([]models.Player, 0, len(playerNames))
	for _, name := range playerNames {
		id, err := playerService.CreatePlayer(name)
		if err != nil {
			log.Fatalf("failed to create player %q: %v", name, err)
		}
		players = append(players, models.Player{BaseModel: models.BaseModel{ID: id}, Name: name})
	}
	log.Printf("created %d players", len(players))

	teams := make([]models.Team, 0, len(teamColours))
	for _, colour := range teamColours {
		team := &models.Team{Colour: colour}
		if err := teamService.CreateTeam(team); err != nil {
			log.Fatalf("failed to create team %q: %v", colour, err)
		}
		teams = append(teams, *team)
	}
	log.Printf("created %d teams", len(teams))

	for _, date := range lastSundays(*matchCount) {
		matchID, err := matchService.CreateMatch(date)
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
	for _, model := range []interface{}{&models.MatchPlayer{}, &models.Match{}, &models.Team{}, &models.Player{}} {
		if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
			return err
		}
	}
	return nil
}

// lastSundays returns the date (YYYY-MM-DD) of the last n Sundays, oldest first.
func lastSundays(n int) []string {
	dates := make([]string, 0, n)
	now := time.Now()
	lastSunday := now.AddDate(0, 0, -int(now.Weekday()))
	for i := n - 1; i >= 0; i-- {
		dates = append(dates, lastSunday.AddDate(0, 0, -7*i).Format("2006-01-02"))
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
