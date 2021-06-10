package poller

import (
	"GoVaccineUpdaterPoller/parser"
	"fmt"
	"github.com/go-logr/logr"
	"golang.org/x/net/http2"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"sync"
	"time"
)

type SessionsReturned struct {
	Sessions   []parser.Session
	StatusCode int
}

type Poller struct {
	log                       logr.Logger
	location                  *time.Location
	client                    *http.Client
	sleepDurationBetweenCalls time.Duration
}

type DistrictPollRequest struct {
	DistrictId uint32
	Date       time.Time
}

func (r DistrictPollRequest) GetUrl() *url.URL {
	URL := "https://api.cowin.gov.in/api/v2/appointment/sessions/public/findByDistrict?district_id=" + fmt.Sprint(r.DistrictId) + "&date=" + strconv.Itoa(r.Date.Day()) + "-" + strconv.Itoa(int(r.Date.Month())) + "-" + strconv.Itoa(r.Date.Year())
	parsedURL, _ := url.Parse(URL)
	return parsedURL
}

func (p Poller) GeneratePollRequests(districtsToPoll []uint32, days int) (districtsPollRequests []*DistrictPollRequest) {
	timeInUTC := time.Now()
	today := timeInUTC.In(p.location)
	districtsPollRequests = make([]*DistrictPollRequest, 0)
	for i := 0; i < len(districtsToPoll); i++ {
		for j := 0; j < days; j++ {
			districtPollRequest := &DistrictPollRequest{
				DistrictId: districtsToPoll[i],
				Date:       today.AddDate(0, 0, j),
			}
			districtsPollRequests = append(districtsPollRequests, districtPollRequest)
		}
	}
	return
}

func NewPoller(sleepDurationBetweenCalls time.Duration, log logr.Logger) Poller {
	location, _ := time.LoadLocation("Asia/Kolkata")
	client := &http.Client{
		Transport: &http2.Transport{},
	}
	return Poller{
		log:                       log,
		location:                  location,
		client:                    client,
		sleepDurationBetweenCalls: sleepDurationBetweenCalls,
	}
}

func (p Poller) RunRequests(districtsPollRequests []*DistrictPollRequest) (sessions map[string]parser.Session) {
	sessions = map[string]parser.Session{}
	c := make(chan SessionsReturned)
	var wg sync.WaitGroup
	workersInitCount := 0
	for _, districtsPollRequest := range districtsPollRequests {
		if workersInitCount >= 100 {
			workersInitCount = 0
			runtime.Gosched()
		}
		time.Sleep(p.sleepDurationBetweenCalls)
		wg.Add(1)
		go RunRequest(districtsPollRequest.GetUrl(), p.client, c, &wg, p.log)
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
			noOfSessions += len(sessionsReturned.Sessions)
			for i := 0; i < len(sessionsReturned.Sessions); i++ {
				sessions[sessionsReturned.Sessions[i].SessionId] = sessionsReturned.Sessions[i]
			}
		}
	}
	p.log.V(1).Info("Run complete.", "no_of_sessions", noOfSessions)
	return
}

func IgnoreError(_ error) {}

func RunRequest(parsedURL *url.URL, client *http.Client, c chan SessionsReturned, wg *sync.WaitGroup, log logr.Logger) {
	defer wg.Done()
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
		sessionsReturned.Sessions, err = parser.ParseSessions(body)
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
	log.V(1).Info("Completed", "url", parsedURL, "retry", 3-retries, "time", time.Since(start), "no_of_sessions", len(sessionsReturned.Sessions))
	c <- sessionsReturned
	return
}
