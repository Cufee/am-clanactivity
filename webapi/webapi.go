package api

import (
	"log"
	"strconv"

	"sync"

	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	mongo "github.com/cufee/am-clanactivity/mongoapi"
	proc "github.com/cufee/am-clanactivity/processing"
	"go.mongodb.org/mongo-driver/bson"
)

type exportJSON struct {
	Clan    mongo.Clan     `json:"clan_data,omitempty"`
	Members []mongo.Player `json:"players"`
}

type reqClanInfo struct {
	Tag   string `json:"clan_tag"`
	Realm string `json:"clan_realm"`
	ID    string `json:"clan_id"`
}

// HandleRequests - start API
func HandleRequests(PORT int) {
	log.Println("Starting webserver on", PORT)
	hostPORT := ":" + strconv.Itoa(PORT)

	myRouter := mux.NewRouter().StrictSlash(true)
	// myRouter.HandleFunc("/clans", updateClanActivity)
	myRouter.HandleFunc("/clan", addNewClan).Methods("POST")
	myRouter.HandleFunc("/clan", updateClanActivity).Methods("PUT")
	myRouter.HandleFunc("/clan", exportClanActivity).Methods("GET")

	log.Fatal(http.ListenAndServe(hostPORT, myRouter))
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
	log.Println("Request - ", code)
}

func respondWithCode(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
	log.Println("Request - ", code)
}

// GET
func exportClanActivity(w http.ResponseWriter, r *http.Request) {
	var request reqClanInfo
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	clanTag := request.Tag
	clanRealm := request.Realm
	if clanTag == (reqClanInfo{}.Tag) || clanRealm == (reqClanInfo{}.Realm) {
		// Check if both Tag and Realm are provided
		respondWithError(w, http.StatusBadRequest, ("Clan tag or realm not provided"))
		return
	}

	filter := bson.M{"clan_tag": clanTag}
	clanData, err := mongo.GetClan(filter)
	if err != nil {
		// Error 404
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	var export exportJSON
	export.Clan = clanData

	response := make(chan mongo.Player, 51)
	proc.PlayersFefreshSession(clanData.MembersIds, response)
	for r := range response {
		export.Members = append(export.Members, r)
	}
	// Send response
	respondWithJSON(w, http.StatusOK, export)

	return
}

// PUT
func updateClanActivity(w http.ResponseWriter, r *http.Request) {
	var request reqClanInfo
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	clanTag := request.Tag
	clanRealm := request.Realm
	if clanTag == (reqClanInfo{}.Tag) || clanRealm == (reqClanInfo{}.Realm) {
		// Check if both Tag and Realm are provided
		respondWithError(w, http.StatusBadRequest, ("Clan tag or realm not provided"))
		return
	}

	filter := bson.M{"clan_tag": clanTag}
	clanData, err := mongo.GetClan(filter)
	if err != nil {
		// Error 404
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}
	var export exportJSON
	export.Clan = clanData

	response := make(chan mongo.Player, 51)
	proc.PlayersFefreshSession(clanData.MembersIds, response)

	var wg sync.WaitGroup

	for r := range response {
		export.Members = append(export.Members, r)
		wg.Add(1)
		go func(p mongo.Player) {
			defer wg.Done()

			_, err := mongo.UpdatePlayer(p, true)
			if err != nil {
				log.Fatal(err)
			}
		}(r)
	}
	// Need to implement sorting the dict by SessionBattles - Currenlty done client side

	// Send response
	respondWithJSON(w, http.StatusOK, export)

	// Wait for player updates to finish
	wg.Wait()
}

// POST
func addNewClan(w http.ResponseWriter, r *http.Request) {
	var request reqClanInfo
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	clanTag := request.Tag
	clanRealm := request.Realm
	if clanTag == (reqClanInfo{}.Tag) || clanRealm == (reqClanInfo{}.Realm) {
		// Check if both Tag and Realm are provided
		respondWithError(w, http.StatusBadRequest, ("Clan tag or realm not provided"))
	}

	err = proc.EnableNewClan(clanRealm, clanTag)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithCode(w, http.StatusOK)
}
