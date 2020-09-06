package api

import (	
    "log"
	"strconv"
	"strings"

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

    log.Fatal(http.ListenAndServe(hostPORT, myRouter))
}

func exportClanActivity(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
	log.Println(vars["tag"])

	clanTag := strings.ToUpper(vars["tag"])
	filter := bson.M{"clan_tag": clanTag}
	clanData, err := mongo.GetClan(filter)
	if err != nil {
		log.Println(err)
	}
	var export exportJSON
	export.Clan = clanData
	
	response := make(chan mongo.Player, 51)
	proc.PlayersFefreshSession(clanData.MembersIds, response)

	for r := range response {
		export.Members = append(export.Members, r)
	}
	json.NewEncoder(w).Encode(export)
}