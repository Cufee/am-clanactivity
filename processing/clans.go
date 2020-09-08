package processing

import (
	"fmt"
	"log"
	"sync"

	wgapi "github.com/cufee/am-clanactivity/externalapis/wargaming"
	mongo "github.com/cufee/am-clanactivity/mongoapi"
	"go.mongodb.org/mongo-driver/bson"
)

// EnableNewClan - Enable tracking for a new clan and all players in that clan
func EnableNewClan(realm string, clanTag string) error {
	clanID, err := wgapi.GetClanIDbyTag(realm, clanTag)
	if err != nil {
		return err
	}
	clanData, err := wgapi.GetClanDataByID(realm, clanID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": clanData.ID}
	check, err := mongo.GetClan(filter)

	if check.ID != 0 {
		// Check if clan already in DB
		return fmt.Errorf("clan %s is already enrolled", (clanData.ClanTag))
	} else if err != nil {
		return err
	}

	var newClanEntry mongo.Clan
	newClanEntry.ID = clanData.ID
	newClanEntry.ClanTag = clanData.ClanTag
	newClanEntry.ClanName = clanData.ClanName
	newClanEntry.Realm = realm
	newClanEntry.MembersIds = clanData.MembersIds

	// Add clan to DB
	_, err = mongo.UpdateClan(newClanEntry, true)
	if err != nil {
		return err
	}

	// Add all players
	var wg sync.WaitGroup
	for _, p := range clanData.Members {
		wg.Add(1)

		go func(p wgapi.PlayerRes) {
			defer wg.Done()
			var newPlayerData mongo.Player
			newPlayerData.ID = p.ID
			newPlayerData.Nickname = p.Nickname
			newPlayerData.LastUpdate = p.LastUpdate
			newPlayerData.JoinedAt = p.JoinedAt

			_, err = mongo.UpdatePlayer(newPlayerData, true)
			if err != nil {
				log.Fatal(err)
			}
		}(p)
	}
	wg.Wait()

	// Refresh player sessions
	response := make(chan mongo.Player, 51)
	PlayersFefreshSession(newClanEntry.MembersIds, response)

	return nil
}
