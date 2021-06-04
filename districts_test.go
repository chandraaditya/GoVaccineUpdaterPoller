package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/http2"
	"log"
	"net/http"
	"net/url"
)

type State struct {
	StateId uint32 `json:"state_id,omitempty"`
	StateName string `json:"state_name,omitempty"`
}

type States struct {
	States []State `json:"states,omitempty"`
	Ttl uint32 `json:"ttl"`
}

func main() {
	getDistricts()
}

func getDistricts() {
	transport := &http2.Transport{}
	client := &http.Client{
		Transport: transport,
	}

	u := "https://api.cowin.gov.in/api/v2/admin/location/states"

	parsedURL, err := url.Parse(u)
	if err != nil {
		log.Fatalln(err)
	}

	req := &http.Request{
		Method: "GET",
		URL: parsedURL,
		Header: map[string][]string{
			"cache-control":{"no-cache"},
			"pragma":{"no-cache"},
		},
	}

	do, err := client.Do(req)
	if err != nil {
		return
	}
	var states States
	err = json.NewDecoder(do.Body).Decode(&states)
	if err != nil {
		log.Fatalln(err)
		return
	}


	fmt.Println(states)

}
