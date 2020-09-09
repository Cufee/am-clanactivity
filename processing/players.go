package processing

import (
	"errors"
	"log"
	"math"

	wgapi "github.com/cufee/am-clanactivity/externalapis/wargaming"
	mongo "github.com/cufee/am-clanactivity/mongoapi"
	"go.mongodb.org/mongo-driver/bson"
)

// PlayersFefreshSession - Refresh sessions for a list of players
func PlayersFefreshSession(players []int, realm string, channel chan mongo.Player) {
	defer log.Println("Finished PlayersFefreshSession")
	// Loop througp player IDs and start goroutines
	for _, playerID := range players {
		filter := bson.M{"_id": playerID}
		playerData, err := mongo.GetPlayer(filter)
		if err != nil {
			// Get player data
			playerRes, err := wgapi.GetPlayerDataByID(realm, playerID)
			if err != nil {
				log.Println(err)
				return
			}
			// Get player battles
			battles, err := GetPlayerVehBattles(playerID)
			if err != nil {
				log.Println(err)
			}
			// Add player to DB
			var newPlayerData mongo.Player
			newPlayerData.ID = playerID
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
			continue
		}
		calcPlayerRating(playerData, channel)
	}

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
func calcPlayerRating(playerData mongo.Player, playersChannel chan mongo.Player) {
	defer log.Println("Finished calcPlayerRating for", playerData.Nickname)
	defer func() {
		playersChannel <- playerData
	}()

	// Get live vehicle stats
	vehicles, err := wgapi.GetVehicleStats(playerData.ID)
	if err != nil {
		log.Println(err)
		playerData.SessionRating = 0
		playerData.SessionBattles = 0
		return
	}

	// Calcualte Raw rating and get total battles
	battles, rawRating, err := CalcVehicleRawRating(vehicles)
	if err != nil {
		log.Println(err)
		playerData.AverageRating = 0
		playerData.SessionRating = 0
		playerData.SessionBattles = 0
	}

	oldBattles := playerData.Battles

	if oldBattles == 0 {
		playerData.Battles = int(battles)
		playerData.AverageRating = int(math.Round(rawRating / battles))
		playerData.SessionRating = 0
		// Update player record
		_, err := mongo.UpdatePlayer(playerData, false)
		if err != nil {
			log.Println(err)
		}
		return
	}

	if int(battles) < oldBattles {
		log.Println("Current battles cnt is less than old battles cnt for", playerData.Nickname)
		playerData.Battles = int(battles)
		playerData.SessionRating = 0
		playerData.SessionBattles = 0
		// Update player record
		_, err := mongo.UpdatePlayer(playerData, false)
		if err != nil {
			log.Println(err)
		}
		return
	}

	// oldBattles defined at the start of this func
	playerData.SessionBattles = int(battles) - oldBattles
	if playerData.SessionBattles == 0 {
		playerData.AverageRating = int(math.Round(rawRating / battles))
		playerData.SessionRating = 0
		return
	}

	oldRating := playerData.AverageRating
	playerData.AverageRating = int(math.Round(rawRating / battles))
	playerData.Battles = int(battles)
	sessionRatingWeighted := (playerData.AverageRating * int(battles)) - (oldRating * oldBattles)
	sessionRating := sessionRatingWeighted / (int(battles) - oldBattles)
	playerData.SessionRating = sessionRating

	return
}

// CalcVehicleRawRating - Calculate rating for a slice of VehicleStats structs.
func CalcVehicleRawRating(vehicles []wgapi.VehicleStats) (battles float64, rawRating float64, err error) {
	if len(vehicles) == 0 {
		return 0, 0, errors.New("VehicleStats slice empty")
	}
	// Calculate rating for all vehicles in go routines
	// Create channels for battles and rawRating
	for _, tank := range vehicles {
		filter := bson.M{"tank_id": tank.TankID}
		tankAvgData, err := mongo.GetTankAvg(filter)
		if err != nil {
			// No tank average data, no need to spam log/report
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

	return battles, rawRating, nil
}
