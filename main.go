package main

import (
	"GoVaccineUpdaterPoller/districts"
	"GoVaccineUpdaterPoller/parser"
	"fmt"
	"golang.org/x/net/http2"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"
)

func main() {
	districtsData, err := districts.GetDistrictsData()
	if err != nil {
		return
	}
	transport := &http2.Transport{}
	client := &http.Client{
		Transport: transport,
	}
	for {
		urls := genURLsForSevenDays(districtsData.GetDistrictsToPoll(4, 0))
		runRequests(urls, client)
	}
}

func runRequests(urls []*url.URL, client *http.Client) {
	c := make(chan int)
	var wg sync.WaitGroup
	start := time.Now()
	workersInitCount := 0
	for _, link := range urls {
		if workersInitCount >= 100 {
			workersInitCount = 0
			runtime.Gosched()
		}
		wg.Add(1)
		go runRequest(link, client, c, &wg)
		workersInitCount++
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	statuses := make(map[int]int)
	for status := range c {
		if _, ok := statuses[status]; ok {
			statuses[status]++
		} else {
			statuses[status] = 1
		}
	}
	log.Println(statuses, time.Since(start).Seconds())
	return
}

func ignoreError(_ error) {}

func runRequest(parsedURL *url.URL, client *http.Client, c chan int, wg *sync.WaitGroup) {
	defer (*wg).Done()
	req := &http.Request{
		Method: "GET",
		URL: parsedURL,
		Header: map[string][]string{
			"cache-control":{"no-cache"},
			"pragma":{"no-cache"},
		},
	}
	retrySleepDuration := 1 * time.Millisecond
	retries := 3
	statusCode := -1
	for retries > 0 {
		resp, err := client.Do(req)
		if err != nil {
			statusCode = -1
			retries--
			time.Sleep(retrySleepDuration)
			continue
		}
		if resp.StatusCode != 200 {
			statusCode = resp.StatusCode
			retries--
			time.Sleep(retrySleepDuration)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			statusCode = -1
			retries--
			time.Sleep(retrySleepDuration)
			continue
		}
		err = resp.Body.Close()
		if err != nil {ignoreError(err)}
		_ = parser.ParseSessions(body)
		statusCode = resp.StatusCode
		break
	}
	c <- statusCode
	return
}

func genURLsForSevenDays(districtsToPoll []uint32) (urls []*url.URL) {
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Fatalln(err)
	}
	timeInUTC := time.Now()
	today := timeInUTC.In(location)
	urls = make([]*url.URL, 0)
	for i := 0; i < len(districtsToPoll); i++ {
		for j := 0; j < 7; j++ {
			requestDate := today.AddDate(0,0, j)
			URL := "https://api.cowin.gov.in/api/v2/appointment/sessions/public/findByDistrict?district_id=" + fmt.Sprint(districtsToPoll[i]) + "&date=" + strconv.Itoa(requestDate.Day()) + "-" + strconv.Itoa(int(requestDate.Month())) + "-" + strconv.Itoa(requestDate.Year())
			parsedURL, err := url.Parse(URL)
			if err != nil {ignoreError(err)}
			urls = append(urls, parsedURL)
		}
	}
	return
}
