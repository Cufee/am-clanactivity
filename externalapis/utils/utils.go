package externalapis

import (
	"encoding/json"
	"net/http"
	"time"
)

// Global HTTP client
var clientHTTP = &http.Client{Timeout: 10 * time.Second}

// GetJSON - Send a GET request to URL and return JSON result into target interface
func GetJSON(url string, target interface{}) error {
	r, err := clientHTTP.Get(url)
	// log.Println(r.Status)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(target)
}
