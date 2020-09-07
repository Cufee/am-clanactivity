package processing

import (
	mongo "github.com/cufee/am-clanactivity/mongoapi"
	wgapi "github.com/cufee/am-clanactivity/externalapis/wargaming"
)

func EnableNewClan(realm string, clanTag string) (error) {
	clanID, err := wgapi.GetClanIDbyTag(realm, clanTag)
	if err != nil{
		return err
	}
	clanData, err := wgapi.GetClanDataByID(realm, clanID)
	if err != nil {
		return err
	}

	var newClanEntry mongo.Clan
	newClanEntry.ID			= clanData.ID
	newClanEntry.ClanTag	= clanData.ClanTag
	newClanEntry.ClanName	= clanData.ClanName
	newClanEntry.Realm		= realm
	newClanEntry.MembersIds	= clanData.MembersIds
	
	// Add clan to DB
	_, err = mongo.UpdateClan(newClanEntry, true)
	if err != nil {
		return err
	}
	return nil
}