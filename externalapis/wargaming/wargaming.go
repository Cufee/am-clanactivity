package externalapis

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cufee/am-clanactivity/config"
	utils "github.com/cufee/am-clanactivity/externalapis/utils"
)

// ClanInfoRes - JSON response from WG API
type clanInfoRes struct {
	Data []ClanInfo `json:"data"`
}

// ClanInfo - Short clan info struct, used to get ClanID
type ClanInfo struct {
	ClanID int    `json:"clan_id"`
	Tag    string `json:"tag"`
	Name   string `json:"name"`
}

// ClanMembersRes - JSON response from WG clans detailed API
type clanMembersRes struct {
	Data map[string]ClanDetails `json:"data"`
}

// ClanDetails -
type ClanDetails struct {
	ID         int                  `json:"clan_id"`
	ClanName   string               `json:"name"`
	ClanTag    string               `json:"tag"`
	MembersIds []int                `json:"members_ids"`
	Members    map[string]PlayerRes `json:"members"`
}

// PlayerResRaw -
type PlayerResRaw struct {
	Data map[string]PlayerRes `json:"data"`
}

// PlayerRes - player data response from WG
type PlayerRes struct {
	ID         int       `json:"account_id"`
	JoinedAt   int       `json:"joined_at"`
	Nickname   string    `json:"account_name"`
	Role       string    `json:"role"`
	LastUpdate time.Time `json:"last_update"`
}

// PID - PlayedID for parsing json
type pid map[string][]VehicleStats

// PlayerVehiclesRes - JSON response from WG API
type playerVehiclesRes struct {
	Data pid `json:"data"`
}

// VehicleStats -
type VehicleStats struct {
	All struct {
		Spotted              float64 `json:"spotted,omitempty"`
		Hits                 float64 `json:"hits,omitempty"`
		Frags                float64 `json:"frags,omitempty"`
		MaxXp                int     `json:"max_xp,omitempty"`
		Wins                 float64 `json:"wins,omitempty"`
		Losses               float64 `json:"losses,omitempty"`
		CapturePoints        float64 `json:"capture_points,omitempty"`
		Battles              float64 `json:"battles,omitempty"`
		DamageDealt          float64 `json:"damage_dealt,omitempty"`
		DamageReceived       float64 `json:"damage_received,omitempty"`
		MaxFrags             float64 `json:"max_frags,omitempty"`
		Shots                float64 `json:"shots,omitempty"`
		Xp                   float64 `json:"xp,omitempty"`
		SurvivedBattles      float64 `json:"survived_battles,omitempty"`
		DroppedCapturePoints float64 `json:"dropped_capture_points,omitempty"`
	} `json:"all"`
	LastBattleTime int `json:"last_battle_time,omitempty"`
	MarkOfMastery  int `json:"mark_of_mastery,omitempty"`
	TankID         int `json:"tank_id"`
}

// Players
var wgAPIVehicles string = fmt.Sprintf("/wotb/tanks/stats/?application_id=%s&account_id=", config.WgAPIAppID)
var wgAPIBaseStats string = fmt.Sprintf("/wotb/account/info/?application_id=%s&fields=statistics.all.battles&account_id=", config.WgAPIAppID)

// Clans
var wgAPIClanInfo string = fmt.Sprintf("/wotb/clans/list/?application_id=%s&search=", config.WgAPIAppID)
var wgAPIClanDetails string = fmt.Sprintf("/wotb/clans/info/?application_id=%s&fields=clan_id,name,tag,is_clan_disbanded,members_ids,updated_at,members&extra=members&clan_id=", config.WgAPIAppID)

// getAPIDomain - Get WG API domain using realm or playerID length
func getAPIDomain(realm string, playerID int) (string, error) {
	realm = strings.ToUpper(realm)
	pIDLen := len(strconv.Itoa(playerID))

	if realm == "NA" || pIDLen == 10 {
		return "http://api.wotblitz.com", nil

	} else if realm == "EU" || pIDLen == 9 {
		return "http://api.wotblitz.eu", nil

	} else if realm == "RU" || pIDLen == 8 {
		return "http://api.wotblitz.ru", nil

	} else {
		message := fmt.Sprintf("Realm %s not found", realm)
		return "", errors.New(message)
	}
}

// GetVehicleStats - Get current vehicle stats for a player by playerID
func GetVehicleStats(playerID int) ([]VehicleStats, error) {
	domain, err := getAPIDomain("", playerID)
	if err != nil {
		var result []VehicleStats
		return result, err
	}
	// Get current stats
	playerIDStr := strconv.Itoa(playerID)
	finalURL := domain + wgAPIVehicles + playerIDStr
	response := new(playerVehiclesRes)

	err = utils.GetJSON(finalURL, response)
	if err != nil {
		// Check error and retry if timed out or reached limit
		panic("Not implemented")
	}
	return response.Data[playerIDStr], nil

}

// GetClanIDbyTag - Find clanID by tag and realm
func GetClanIDbyTag(realm string, clanTag string) (int, error) {
	realm = strings.ToUpper(realm)
	clanTag = strings.ToUpper(clanTag)

	domain, err := getAPIDomain(realm, 0)
	if err != nil {
		return 0, err
	}
	// Search for clan by tag
	fullURL := domain + wgAPIClanInfo + clanTag
	var response = new(clanInfoRes)
	err = utils.GetJSON(fullURL, response)
	if err != nil {
		return 0, err
	}
	// Loop through results and math the tag
	var clanFoundID int
	for _, clan := range response.Data {
		if clan.Tag == clanTag {
			clanFoundID = clan.ClanID
			break
		}
	}
	if clanFoundID == 0 {
		message := fmt.Sprintf("Clan %s not found on %s.", clanTag, realm)
		return 0, errors.New(message)
	}
	return clanFoundID, nil
}

// GetClanDataByID - Get clan detailed data from clanID and realm
func GetClanDataByID(realm string, clanID int) (ClanDetails, error) {
	realm = strings.ToUpper(realm)
	domain, err := getAPIDomain(realm, 0)
	if err != nil {
		var result ClanDetails
		return result, err
	}
	fullURL := domain + wgAPIClanDetails + strconv.Itoa(clanID)
	var response = new(clanMembersRes)
	err = utils.GetJSON(fullURL, response)
	if err != nil {
		var result ClanDetails
		return result, err
	}
	var result ClanDetails = response.Data[strconv.Itoa(clanID)]
	if result.ID != clanID {
		message := fmt.Sprintf("Detailed clan response ID %v is not matching requested clan ID %v", result.ID, clanID)
		var result ClanDetails
		return result, errors.New(message)
	}
	return result, nil
}

// GetPlayerDataByID - Get player data from player ID
func GetPlayerDataByID(realm string, pid int) (PlayerRes, error) {
	realm = strings.ToUpper(realm)
	domain, err := getAPIDomain(realm, 0)
	if err != nil {
		var result PlayerRes
		return result, err
	}
	fullURL := domain + wgAPIBaseStats + strconv.Itoa(pid)
	var response = new(PlayerResRaw)
	err = utils.GetJSON(fullURL, response)
	if err != nil {
		var result PlayerRes
		return result, err
	}

	var playerData PlayerRes
	playerData.ID = response.Data[(strconv.Itoa(pid))].ID
	playerData.Nickname = response.Data[(strconv.Itoa(pid))].Nickname

	return playerData, nil
}
