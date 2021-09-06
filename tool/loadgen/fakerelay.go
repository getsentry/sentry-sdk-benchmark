package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// relayMetrics returns /debug/vars exposed variables from the Fake Relay
// instance.
func relayMetrics(url string) map[string]interface{} {
	log.Printf("Fetching fakerelay stats from %q", url)
	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url + "/debug/vars")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var m map[string]interface{}
	err = dec.Decode(&m)
	if err != nil {
		panic(err)
	}
	return m
}
