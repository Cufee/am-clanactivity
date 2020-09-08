package main

import (
	webapi "github.com/cufee/am-clanactivity/webapi"
)

func main() {
	// Run app
	webapi.HandleRequests(10000)
}
