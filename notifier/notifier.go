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
	NotifiedDose1 map[string]notified
	NotifiedDose2 map[string]notified
}

type notified struct {
	Session    parser.Session
	TimeCaught time.Time
}

func NewNotifier() Notifier {
	return Notifier{
		NotifiedDose1: map[string]notified{},
		NotifiedDose2: map[string]notified{},
	}
}

func (n *Notifier) Notify(sessions map[string]parser.Session, client *http.Client, webhook webhook.Districts, districtMap districts2.Map, log logr.Logger) {
	count := 0
	for i := range n.NotifiedDose1 {
		if _, ok := sessions[n.NotifiedDose1[i].Session.SessionId]; !ok {
			if count >= 100 {
				count = 0
				runtime.Gosched()
			}
			count++
			n.ZeroSlotsLeft(webhook, client, districtMap, n.NotifiedDose1[i].Session.SessionId, 1, log.WithName("zero"))
		}
	}
	count = 0
	for i := range n.NotifiedDose2 {
		if _, ok := sessions[n.NotifiedDose2[i].Session.SessionId]; !ok {
			if count >= 100 {
				count = 0
				runtime.Gosched()
			}
			count++
			n.ZeroSlotsLeft(webhook, client, districtMap, n.NotifiedDose2[i].Session.SessionId, 2, log.WithName("zero"))
		}
	}
	count = 0
	for i := range sessions {
		if count >= 100 {
			count = 0
			runtime.Gosched()
		}
		session := sessions[i]
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
}

func (n *Notifier) SlotsOpen(webhook webhook.Districts, client *http.Client, districtMap districts2.Map, session parser.Session, dose int, log logr.Logger) {
	districtID := districtMap.GetDistrictID(session.StateName, session.DistrictName)
	URLs := webhook.GetOpenWebhooksForDistrict(districtID)
	if dose == 1 {
		if _, ok := n.NotifiedDose1[session.SessionId]; !ok {
			log.V(1).Info("open slots",
				"district_name", session.DistrictName,
				"dose", dose,
				"center_id", session.CenterId,
				"session_id", session.SessionId,
			)
			webhookSession := parser.OpenWebhook{
				Dose:    1,
				Session: session,
			}
			marshal, err := json.Marshal(&webhookSession)
			if err != nil {
				log.V(1).Error(err, "unable to marshal")
				return
			}
			go SendToWebhooks(URLs, marshal, client)
			n.NotifiedDose1[session.SessionId] = notified{
				Session:    session,
				TimeCaught: time.Now(),
			}
		}
	}
	if dose == 2 {
		if _, ok := n.NotifiedDose2[session.SessionId]; !ok {
			log.V(1).Info("open slots",
				"district_name", session.DistrictName,
				"dose", dose,
				"center_id", session.CenterId,
				"session_id", session.SessionId,
			)
			webhookSession := parser.OpenWebhook{
				Dose:    2,
				Session: session,
			}
			marshal, err := json.Marshal(&webhookSession)
			if err != nil {
				log.V(1).Error(err, "unable to marshal")
				return
			}
			go SendToWebhooks(URLs, marshal, client)
			n.NotifiedDose2[session.SessionId] = notified{
				Session:    session,
				TimeCaught: time.Now(),
			}
		}
	}
}

func (n *Notifier) ZeroSlotsLeft(webhook webhook.Districts, client *http.Client, districtMap districts2.Map, sessionId string, dose int, log logr.Logger) {
	if dose == 1 {
		session := n.NotifiedDose1[sessionId]
		if _, ok := n.NotifiedDose1[sessionId]; ok {
			log.V(1).Info("zero slots",
				"district_name", session.Session.DistrictName,
				"dose", dose,
				"center_id", session.Session.CenterId,
				"session_id", session.Session.SessionId,
			)
			districtID := districtMap.GetDistrictID(session.Session.StateName, session.Session.DistrictName)
			URLs := webhook.GetCloseWebhooksForDistrict(districtID)
			webhookSession := parser.CloseWebhook{
				Dose:            1,
				Session:         session.Session,
				DurationOpenFor: time.Since(session.TimeCaught).String(),
			}
			marshal, err := json.Marshal(&webhookSession)
			if err != nil {
				log.V(1).Error(err, "unable to marshal")
				return
			}
			go SendToWebhooks(URLs, marshal, client)
			delete(n.NotifiedDose1, sessionId)
		}
	}
	if dose == 2 {
		session := n.NotifiedDose2[sessionId]
		if _, ok := n.NotifiedDose2[sessionId]; ok {
			log.V(1).Info("zero slots",
				"district_name", session.Session.DistrictName,
				"dose", dose,
				"center_id", session.Session.CenterId,
				"session_id", session.Session.SessionId,
			)
			districtID := districtMap.GetDistrictID(session.Session.StateName, session.Session.DistrictName)
			URLs := webhook.GetCloseWebhooksForDistrict(districtID)
			webhookSession := parser.CloseWebhook{
				Dose:            2,
				Session:         session.Session,
				DurationOpenFor: time.Since(session.TimeCaught).String(),
			}
			marshal, err := json.Marshal(&webhookSession)
			if err != nil {
				log.V(1).Error(err, "unable to marshal")
				return
			}
			go SendToWebhooks(URLs, marshal, client)
			delete(n.NotifiedDose2, sessionId)
		}
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
	defer (*wg).Done()
	_, err := client.Post(parsedURL.String(), "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println("unable to send to webhook:", err)
	}
	c <- struct{}{}
}
