package api

import (
	"log"
	"time"	
	"strconv"
	"strings"

	"sync"

    "net/http"
    "encoding/json"
	"github.com/gorilla/mux"
    // "github.com/gorilla/handlers"
	"go.mongodb.org/mongo-driver/bson"
	
	mongo 	"github.com/cufee/am-clanactivity/mongoapi"
	proc 	"github.com/cufee/am-clanactivity/processing"
)

type exportJSON struct {
	Clan mongo.Clan			`json:"clan_data,omitempty"`
	Members []mongo.Player	`json:"players"`
}

// HandleRequests - start API
func HandleRequests(PORT int) {
	log.Println("Starting webserver on", PORT)
	hostPORT := ":" + strconv.Itoa(PORT)

	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/clans/{tag}", exportClanActivity)
	myRouter.HandleFunc("/clans/update/{tag}", updateClanActivity)

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

func exportClanActivity(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
    vars := mux.Vars(r)

	clanTag := strings.ToUpper(vars["tag"])
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
	log.Println(vars["tag"], "- request took", time.Since(start))
}

func updateClanActivity(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
    vars := mux.Vars(r)

	clanTag := strings.ToUpper(vars["tag"])
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
	// Need to implement sorting the dict by SessionBattles

	// Send response
	respondWithJSON(w, http.StatusOK, export)
	// Wait for player updates to finish
	wg.Wait()
	log.Println(vars["tag"], "- request took", time.Since(start))
}