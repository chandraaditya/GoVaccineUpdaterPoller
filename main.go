package main

import (
	"GoVaccineUpdaterPoller/districts"
	"GoVaccineUpdaterPoller/notifier"
	"GoVaccineUpdaterPoller/poller"
	"GoVaccineUpdaterPoller/webhook"
	"encoding/json"
	"fmt"
	"golang.org/x/net/http2"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const rounds = 100

func main() {
	go webhook.StartWebhookServer()
	startPolling()
}

func startPolling() {
	client := &http.Client{
		Transport: &http2.Transport{},
	}
	clientForNotifier := &http.Client{
		Transport: &http.Transport{},
	}
	notifierClient := notifier.NewNotifier()
	webhookDistricts, err := webhook.NewDistricts()
	if err != nil {
		log.Fatalln(err)
	}
	districtsFromWebhook := webhookDistricts.GetDistricts()
	districtsToPoll := districts.GetDistrictsToPoll(districtsFromWebhook, 1, 0)
	log.Println(districtsToPoll)
	districtsMap, err := districts.GetDistrictsMap()
	if err != nil {
		log.Fatalln(err)
	}
	round := 0
	avgTime := 0.0
	for {
		if round >= rounds {
			status := webhook.Status{}
			status.UpdateFrequency = avgTime / float64(rounds)
			status.LastUpdated = time.Now().Unix()
			marshal, err := json.Marshal(status)
			if err != nil {
				log.Println(err)
			} else {
				err = ioutil.WriteFile("status_static_files/status.json", marshal, 0644)
				if err != nil {
					log.Println(err)
				}
			}
			avgTime = 0.0
			err = webhookDistricts.UpdateDistricts()
			if err != nil {
				log.Println(err)
				continue
			}
			districtsMapTemp, err := districts.GetDistrictsMap()
			if err != nil {
				log.Println(err)
			} else {
				districtsMap = districtsMapTemp
			}
			districtsFromWebhook = webhookDistricts.GetDistricts()
			districtsToPoll = districts.GetDistrictsToPoll(districtsFromWebhook, 1, 0)
			round = 0
		}
		start := time.Now()
		urls := poller.GenURLs(districtsToPoll, 7)
		sessions := poller.RunRequests(urls, client, 0)
		notifierClient.Notify(sessions, clientForNotifier, webhookDistricts, districtsMap)
		fmt.Println(round, time.Since(start))
		avgTime += time.Since(start).Seconds()
		round++
	}
}
