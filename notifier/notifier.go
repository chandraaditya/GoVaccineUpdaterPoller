package notifier

import (
	"GoVaccineUpdaterPoller/parser"
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
	for i := range sessions {
		session := sessions[i]
		if session.GetAvailableCapacityDose1() > 0 {
			n.SlotsOpen(session, 1)
		} else {
			n.ZeroSlotsLeft(session.GetSessionId(), 1)
		}
		if session.GetAvailableCapacityDose2() > 0 {
			n.SlotsOpen(session, 2)
		} else {
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
		if _, ok := n.NotifiedDose1[sessionId]; ok {
			delete(n.NotifiedDose1, sessionId)
		}
	}
	if dose == 2 {
		if _, ok := n.NotifiedDose2[sessionId]; ok {
			delete(n.NotifiedDose2, sessionId)
		}
	}
}
