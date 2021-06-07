package poller

import (
	"GoVaccineUpdaterPoller/parser"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type SessionsReturned struct {
	Session    []*parser.Session
	StatusCode int
}

func RunRequests(urls []*url.URL, client *http.Client, sleepDurationBetweenCalls time.Duration) (sessions map[string]*parser.Session) {
	sessions = map[string]*parser.Session{}
	c := make(chan SessionsReturned)
	var wg sync.WaitGroup
	workersInitCount := 0
	for _, link := range urls {
		if workersInitCount >= 100 {
			workersInitCount = 0
			runtime.Gosched()
		}
		time.Sleep(sleepDurationBetweenCalls)
		wg.Add(1)
		go RunRequest(link, client, c, &wg)
		workersInitCount++
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	statuses := make(map[int]int)
	for sessionsReturned := range c {
		if _, ok := statuses[sessionsReturned.StatusCode]; ok {
			statuses[sessionsReturned.StatusCode]++
		} else {
			statuses[sessionsReturned.StatusCode] = 1
		}
		if sessionsReturned.StatusCode == 200 {
			for i := 0; i < len(sessionsReturned.Session); i++ {
				sessions[sessionsReturned.Session[i].SessionId] = sessionsReturned.Session[i]
			}
		}
	}
	return
}

func IgnoreError(_ error) {}

func RunRequest(parsedURL *url.URL, client *http.Client, c chan SessionsReturned, wg *sync.WaitGroup) {
	defer (*wg).Done()
	req := &http.Request{
		Method: "GET",
		URL:    parsedURL,
		Header: map[string][]string{
			"cache-control": {"no-cache"},
			"pragma":        {"no-cache"},
		},
	}
	sessionsReturned := SessionsReturned{}
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
		if err != nil {
			IgnoreError(err)
		}
		sessionsReturned.Session = parser.ParseSessions(body)
		statusCode = resp.StatusCode
		break
	}
	sessionsReturned.StatusCode = statusCode
	c <- sessionsReturned
	return
}

func GenURLs(districtsToPoll []uint32, days int) (urls []*url.URL) {
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Fatalln(err)
	}
	timeInUTC := time.Now()
	today := timeInUTC.In(location)
	urls = make([]*url.URL, 0)
	for i := 0; i < len(districtsToPoll); i++ {
		for j := 0; j < days; j++ {
			requestDate := today.AddDate(0, 0, j)
			URL := "https://api.cowin.gov.in/api/v2/appointment/sessions/public/findByDistrict?district_id=" + fmt.Sprint(districtsToPoll[i]) + "&date=" + strconv.Itoa(requestDate.Day()) + "-" + strconv.Itoa(int(requestDate.Month())) + "-" + strconv.Itoa(requestDate.Year())
			parsedURL, err := url.Parse(URL)
			if err != nil {
				IgnoreError(err)
			}
			urls = append(urls, parsedURL)
		}
	}
	return
}
