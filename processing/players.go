package processing

import (
	"log"
	"math"
	"time"

	"sync"

	wgapi "github.com/cufee/am-clanactivity/externalapis/wargaming"
	mongo "github.com/cufee/am-clanactivity/mongoapi"
	"go.mongodb.org/mongo-driver/bson"
)

// PlayersFefreshSession - Refresh sessions for a list of players
func PlayersFefreshSession(players []int, realm string, channel chan mongo.Player) {
	// Waitgroup for player update goroutines
	var wg sync.WaitGroup
	// Loop througp player IDs and start goroutines
	for _, playerID := range players {
		filter := bson.M{"_id": playerID}
		playerData, err := mongo.GetPlayer(filter)
		if err != nil {
			// Start a go routine to add a player to DB
			wg.Add(1)
			go func(pid int) {
				defer wg.Done()
				// Get player data
				playerRes, err := wgapi.GetPlayerDataByID(realm, pid)
				if err != nil {
					log.Println(err)
					return
				}
				// Get player battles
				battles, err := GetPlayerVehBattles(pid)
				if err != nil {
					log.Println(err)
				}
				// Add player to DB
				var newPlayerData mongo.Player
				newPlayerData.ID = pid
				newPlayerData.Battles = battles
				newPlayerData.Nickname = playerRes.Nickname
				newPlayerData.PremiumExpiration = 0
				newPlayerData.SessionBattles = 0
				newPlayerData.SessionRating = 0
				_, err = mongo.UpdatePlayer(newPlayerData, true)
				if err != nil {
					log.Println(err)
					return
				}
				channel <- newPlayerData
				return
			}(playerID)
			continue
		}
		wg.Add(1)
		go calcPlayerRating(playerData, channel, &wg)
	}
	start := time.Now()
	log.Println("Starting wg.Wait()")
	wg.Wait()
	log.Println("wg.Wait() took", (time.Now().Sub(start)))
	close(channel)
	return
}

// GetPlayerVehBattles - Get player battles total from adding all vehicle battles
func GetPlayerVehBattles(pid int) (battles int, err error) {
	vehicles, err := wgapi.GetVehicleStats(pid)
	if err != nil {
		return 0, err
	}
	for _, t := range vehicles {
		battles += int(t.All.Battles)
	}
	return battles, nil
}

// calcPlayerRating - Caculate player rating and return updated playerData to the channel
func calcPlayerRating(playerData mongo.Player, playersChannel chan mongo.Player, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		playersChannel <- playerData
		// // Add playerData to the channel and finish waitgroup
		// if playerData.SessionBattles > 0 {
		// 	log.Println("Sent to channel")
		// }
	}()
	// Used at the bottom to calculate session rating
	oldBattles := playerData.Battles

	// // Check current player battles
	// liveBattles, err := wgapi.GetLiveBattles(playerData.ID)
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }

	if oldBattles == 0 {
		playerData.SessionRating = 0
		playerData.SessionBattles = 0
		return
	}
	vehicles, err := wgapi.GetVehicleStats(playerData.ID)
	if err != nil {
		log.Println(err)
		playerData.SessionRating = -1
		playerData.SessionBattles = -1
		return
	}

	var battles float64
	var rawRating float64

	for _, tank := range vehicles {
		filter := bson.M{"tank_id": tank.TankID}
		tankAvgData, err := mongo.GetTankAvg(filter)
		if err != nil {
			// No tank average data, no need to spam log/report
			log.Println("No average for", tank.TankID)
			continue
		}
		if tankAvgData.All.Battles == 0 || tank.All.Battles == 0 {
			log.Println("Bad average data for", tank.TankID)
			continue
		}

		// Expected values for WN8
		expDef := tankAvgData.All.DroppedCapturePoints / tankAvgData.All.Battles
		expFrag := tankAvgData.Special.KillsPerBattle
		expSpot := tankAvgData.Special.SpotsPerBattle
		expDmg := tankAvgData.Special.DamagePerBattle
		expWr := tankAvgData.Special.Winrate

		// Actual performance
		pDef := tank.All.DroppedCapturePoints / tank.All.Battles
		pFrag := tank.All.Frags / tank.All.Battles
		pSpot := tank.All.Spotted / tank.All.Battles
		pDmg := tank.All.DamageDealt / tank.All.Battles
		pWr := tank.All.Wins / tank.All.Battles * 100

		// Calculate WN8 metrics
		rDef := pDef / expDef
		rFrag := pFrag / expFrag
		rSpot := pSpot / expSpot
		rDmg := pDmg / expDmg
		rWr := pWr / expWr

		adjustedWr := math.Max(0, ((rWr - 0.71) / (1 - 0.71)))
		adjustedDmg := math.Max(0, ((rDmg - 0.22) / (1 - 0.22)))
		adjustedDef := math.Max(0, (math.Min(adjustedDmg+0.1, (rDef-0.10)/(1-0.10))))
		adjustedSpot := math.Max(0, (math.Min(adjustedDmg+0.1, (rSpot-0.38)/(1-0.38))))
		adjustedFrag := math.Max(0, (math.Min(adjustedDmg+0.2, (rFrag-0.12)/(1-0.12))))

		rating := math.Round(((980 * adjustedDmg) + (210 * adjustedDmg * adjustedFrag) + (155 * adjustedFrag * adjustedSpot) + (75 * adjustedDef * adjustedFrag) + (145 * math.Min(1.8, adjustedWr))))

		ratingWeighted := rating * tank.All.Battles
		battles += tank.All.Battles
		rawRating += ratingWeighted
	}

	if int(battles) < oldBattles {
		log.Println("Current battles cnt is less than old battles cnt for", playerData.Nickname)
		playerData.Battles = int(battles)
		playerData.SessionRating = 0
		playerData.SessionBattles = 0
		_, err := mongo.UpdatePlayer(playerData, false)
		if err != nil {
			log.Println(err)
		}
		return
	}

	// oldBattles defined at the start of this func
	playerData.SessionBattles = int(battles) - oldBattles
	if playerData.SessionBattles == 0 {
		// log.Println("Session battles is 0 for", playerData.Nickname)
		playerData.SessionRating = 0
		return
	}
	oldRating := playerData.AverageRating
	playerData.AverageRating = int(math.Round(rawRating / battles))
	playerData.Battles = int(battles)
	sessionRatingWeighted := (playerData.AverageRating * int(battles)) - (oldRating * oldBattles)
	sessionRating := sessionRatingWeighted / (int(battles) - oldBattles)
	playerData.SessionRating = sessionRating

	// log.Println(playerData.SessionBattles, playerData.SessionRating, playerData.Nickname)
	return
}
