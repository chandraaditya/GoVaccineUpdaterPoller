package notifier

import (
	districts2 "GoVaccineUpdaterPoller/districts"
	"GoVaccineUpdaterPoller/parser"
	"GoVaccineUpdaterPoller/webhook"
	"bytes"
	"encoding/json"
	"github.com/go-logr/logr"
	"log"
	"net/http"
	"net/url"
	"runtime"
	"sync"
	"time"
)

type Notifier struct {
	cache *NotificationCache
}

func NewNotifier(cacheType string, namedLogger logr.Logger, redisHost string, redisPassword string, dbIndex int, ttl time.Duration) Notifier {
	return Notifier{
		cache: NewCache(cacheType, namedLogger, redisHost, redisPassword, dbIndex, ttl),
	}
}

func (n *Notifier) Notify(sessions chan parser.Session, client *http.Client, webhook webhook.Districts, districtMap districts2.Map, log logr.Logger) {
	count := 0
	//sessionIds := map[string]bool{}
	for session := range sessions {
		if count >= 100 {
			count = 0
			runtime.Gosched()
		}
		//sessionIds[session.SessionId] = true
		if session.AvailableCapacityDose1 > 0 {
			count++
			n.SlotsOpen(webhook, client, districtMap, session, 1, log.WithName("open"))
		} else {
			count++
			n.ZeroSlotsLeft(webhook, client, districtMap, session.SessionId, 1, log.WithName("zero"))
		}
		if session.AvailableCapacityDose2 > 0 {
			count++
			n.SlotsOpen(webhook, client, districtMap, session, 2, log.WithName("open"))
		} else {
			count++
			n.ZeroSlotsLeft(webhook, client, districtMap, session.SessionId, 2, log.WithName("zero"))
		}
	}

	//count = 0
	//for i := range n.NotifiedDose1 {
	//	if _, ok := sessionIds[n.NotifiedDose1[i].Session.SessionId]; !ok {
	//		if count >= 100 {
	//			count = 0
	//			runtime.Gosched()
	//		}
	//		count++
	//		n.ZeroSlotsLeft(webhook, client, districtMap, n.NotifiedDose1[i].Session.SessionId, 1, log.WithName("zero"))
	//	}
	//}
	//count = 0
	//for i := range n.NotifiedDose2 {
	//	if _, ok := sessionIds[n.NotifiedDose2[i].Session.SessionId]; !ok {
	//		if count >= 100 {
	//			count = 0
	//			runtime.Gosched()
	//		}
	//		count++
	//		n.ZeroSlotsLeft(webhook, client, districtMap, n.NotifiedDose2[i].Session.SessionId, 2, log.WithName("zero"))
	//	}
	//}

}

func (n *Notifier) SlotsOpen(webhook webhook.Districts, client *http.Client, districtMap districts2.Map, session parser.Session, dose int, log logr.Logger) {
	districtID := districtMap.GetDistrictID(session.StateName, session.DistrictName)
	URLs := webhook.GetOpenWebhooksForDistrict(districtID)
	if ok := (*n.cache).Contains(dose, session.SessionId); !ok {
		log.Info("open slots",
			"district_name", session.DistrictName,
			"dose", dose,
			"available_capacity", session.AvailableCapacity,
			"center_id", session.CenterId,
			"session_id", session.SessionId,
		)
		webhookSession := parser.OpenWebhook{
			Dose:    dose,
			Session: session,
		}
		marshal, err := json.Marshal(&webhookSession)
		if err != nil {
			log.V(1).Error(err, "unable to marshal")
			return
		}
		go SendToWebhooks(URLs, marshal, client)
		(*n.cache).Put(dose, session)
	}
}

func (n *Notifier) ZeroSlotsLeft(webhook webhook.Districts, client *http.Client, districtMap districts2.Map, sessionId string, dose int, log logr.Logger) {
	if ok := (*n.cache).Contains(dose, sessionId); ok {
		session, caughtAt := (*n.cache).Get(dose, sessionId)
		log.Info("zero slots",
			"district_name", session.DistrictName,
			"dose", dose,
			"available_capacity", session.AvailableCapacity,
			"center_id", session.CenterId,
			"session_id", session.SessionId,
		)
		districtID := districtMap.GetDistrictID(session.StateName, session.DistrictName)
		URLs := webhook.GetCloseWebhooksForDistrict(districtID)
		webhookSession := parser.CloseWebhook{
			Dose:            dose,
			Session:         session,
			DurationOpenFor: time.Since(caughtAt).Truncate(time.Second).String(),
		}
		marshal, err := json.Marshal(&webhookSession)
		if err != nil {
			log.V(1).Error(err, "unable to marshal")
			return
		}
		go SendToWebhooks(URLs, marshal, client)
		(*n.cache).Remove(dose, sessionId)
	}
}

func SendToWebhooks(URLs []*url.URL, body []byte, client *http.Client) {
	c := make(chan struct{})
	var wg sync.WaitGroup
	workersInitCount := 0
	for _, link := range URLs {
		if workersInitCount >= 10 {
			workersInitCount = 0
			runtime.Gosched()
		}
		wg.Add(1)
		go SendToWebhook(link, body, client, c, &wg)
		workersInitCount++
	}
	go func() {
		wg.Wait()
		close(c)
	}()
	for range c {
	}
}

func SendToWebhook(parsedURL *url.URL, body []byte, client *http.Client, c chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := client.Post(parsedURL.String(), "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("unable to send to webhook:", err)
	}
	c <- struct{}{}
}
