package poller

import (
	"GoVaccineUpdaterPoller/parser"
	"fmt"
	"github.com/go-logr/logr"
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

func RunRequests(urls []*url.URL, client *http.Client, sleepDurationBetweenCalls time.Duration, log logr.Logger) (sessions map[string]*parser.Session) {
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
		go RunRequest(link, client, c, &wg, log)
		workersInitCount++
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	statuses := make(map[int]int)
	noOfSessions := 0
	for sessionsReturned := range c {
		if _, ok := statuses[sessionsReturned.StatusCode]; ok {
			statuses[sessionsReturned.StatusCode]++
		} else {
			statuses[sessionsReturned.StatusCode] = 1
		}
		if sessionsReturned.StatusCode == 200 {
			noOfSessions += len(sessionsReturned.Session)
			for i := 0; i < len(sessionsReturned.Session); i++ {
				sessions[sessionsReturned.Session[i].SessionId] = sessionsReturned.Session[i]
			}
		}
	}
	log.V(1).Info("Run complete.", "no_of_sessions", noOfSessions)
	return
}

func IgnoreError(_ error) {}

func RunRequest(parsedURL *url.URL, client *http.Client, c chan SessionsReturned, wg *sync.WaitGroup, log logr.Logger) {
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
	start := time.Now()
	for retries > 0 {
		log.V(1).Info("Outgoing", "url", parsedURL, "retry", 3-retries)
		resp, err := client.Do(req)
		if err != nil {
			statusCode = -1
			retries--
			log.V(1).Error(err, "Outgoing failed with error.")
			time.Sleep(retrySleepDuration)
			continue
		}
		if resp.StatusCode != 200 {
			statusCode = resp.StatusCode
			retries--
			log.V(1).Info("Outgoing failed with non 200.", "status_code", statusCode)
			time.Sleep(retrySleepDuration)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			statusCode = -1
			retries--
			log.V(1).Error(err, "Outgoing failed with body reading.")
			time.Sleep(retrySleepDuration)
			continue
		}
		err = resp.Body.Close()
		if err != nil {
			IgnoreError(err)
		}
		sessionsReturned.Session, err = parser.ParseSessions(body)
		if err != nil {
			statusCode = -1
			retries--
			time.Sleep(retrySleepDuration)
			continue
		}
		statusCode = resp.StatusCode
		break
	}
	sessionsReturned.StatusCode = statusCode
	log.V(1).Info("Completed", "url", parsedURL, "retry", 3-retries, "time", time.Since(start), "no_of_sessions", len(sessionsReturned.Session))
	c <- sessionsReturned
	return
}

func GenURLs(districtsToPoll []uint32, days int) (urls []*url.URL) {
	location, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Println(err)
		return
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
