package main

import (
	webapi 	"github.com/cufee/am-clanactivity/api"
)

func main() {
	// Run app
    webapi.HandleRequests(10000)
}