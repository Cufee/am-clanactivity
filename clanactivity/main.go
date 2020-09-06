package main

import (
	// proc 	"github.com/cufee/am-clanactivity/processing"
	// mongo 	"github.com/cufee/am-clanactivity/mongoapi"
	webapi 	"github.com/cufee/am-clanactivity/api"
	// "github.com/cufee/am-clanactivity/externalapis/wargaming"
)

func main() {
	// Run app
    webapi.HandleRequests(10000)
}