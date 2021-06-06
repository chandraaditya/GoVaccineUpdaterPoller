package main

import (
	"GoVaccineUpdaterPoller/notifier"
	"GoVaccineUpdaterPoller/poller"
	"fmt"
	"golang.org/x/net/http2"
	"net/http"
	"time"
)

func main() {
	//districtsData, err := districts.GetDistrictsData()
	//if err != nil {
	//	log.Fatalln(err)
	//	return
	//}
	transport := &http2.Transport{}
	client := &http.Client{
		Transport: transport,
	}
	notifierClient := notifier.NewNotifier()
	for i := 0; i < 1; i++ {
		start := time.Now()
		districtsToPoll := []uint32{265, 294}
		//districtsToPoll := districtsData.GetDistrictsToPoll(1,0)
		urls := poller.GenURLs(districtsToPoll, 7)
		sessions := poller.RunRequests(urls, client, 0)
		notifierClient.Notify(sessions)
		fmt.Println(time.Since(start))
	}
}
