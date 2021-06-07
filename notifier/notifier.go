package notifier

import (
	"GoVaccineUpdaterPoller/parser"
	"fmt"
	"runtime"
	"time"
)

type Notifier struct {
	NotifiedDose1 map[string]notified
	NotifiedDose2 map[string]notified
}

type notified struct {
	Session    *parser.Session
	TimeCaught time.Time
}

func NewNotifier() Notifier {
	return Notifier{
		NotifiedDose1: map[string]notified{},
		NotifiedDose2: map[string]notified{},
	}
}

func (n *Notifier) Notify(sessions map[string]*parser.Session) {
	count := 0
	for i := range n.NotifiedDose1 {
		if _, ok := sessions[n.NotifiedDose1[i].Session.GetSessionId()]; !ok {
			if count >= 100 {
				count = 0
				runtime.Gosched()
			}
			count++
			n.ZeroSlotsLeft(n.NotifiedDose1[i].Session.GetSessionId(), 1)
		}
	}
	count = 0
	for i := range n.NotifiedDose2 {
		if _, ok := sessions[n.NotifiedDose2[i].Session.GetSessionId()]; !ok {
			if count >= 100 {
				count = 0
				runtime.Gosched()
			}
			count++
			n.ZeroSlotsLeft(n.NotifiedDose2[i].Session.GetSessionId(), 2)
		}
	}
	count = 0
	for i := range sessions {
		if count >= 100 {
			count = 0
			runtime.Gosched()
		}
		session := sessions[i]
		if session.GetAvailableCapacityDose1() > 0 {
			count++
			n.SlotsOpen(session, 1)
		} else {
			count++
			n.ZeroSlotsLeft(session.GetSessionId(), 1)
		}
		if session.GetAvailableCapacityDose2() > 0 {
			count++
			n.SlotsOpen(session, 2)
		} else {
			count++
			n.ZeroSlotsLeft(session.GetSessionId(), 2)
		}
	}
}

func (n *Notifier) SlotsOpen(session *parser.Session, dose int) {
	// TODO: Send notifications
	if dose == 1 {
		if _, ok := n.NotifiedDose1[session.GetSessionId()]; !ok {
			n.NotifiedDose1[session.GetSessionId()] = notified{
				Session:    session,
				TimeCaught: time.Now(),
			}
		}
	}
	if dose == 2 {
		if _, ok := n.NotifiedDose2[session.GetSessionId()]; !ok {
			n.NotifiedDose2[session.GetSessionId()] = notified{
				Session:    session,
				TimeCaught: time.Now(),
			}
		}
	}
}

func (n *Notifier) ZeroSlotsLeft(sessionId string, dose int) {
	// TODO: Send notifications
	if dose == 1 {
		session := n.NotifiedDose1[sessionId]
		if _, ok := n.NotifiedDose1[sessionId]; ok {
			fmt.Println(session.Session.GetSessionId(), "1", session.Session.GetMinAgeLimit(), session.Session.GetAvailableCapacityDose1(), time.Since(session.TimeCaught))
			delete(n.NotifiedDose1, sessionId)
		}
	}
	if dose == 2 {
		session := n.NotifiedDose2[sessionId]
		if _, ok := n.NotifiedDose2[sessionId]; ok {
			fmt.Println(session.Session.GetSessionId(), "2", session.Session.GetMinAgeLimit(), session.Session.GetAvailableCapacityDose2(), time.Since(session.TimeCaught))
			delete(n.NotifiedDose2, sessionId)
		}
	}
}
