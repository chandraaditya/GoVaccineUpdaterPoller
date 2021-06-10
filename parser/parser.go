package parser

import (
	"encoding/json"
	"log"
)

type Session struct {
	CenterId               int      `json:"center_id"`
	Name                   string   `json:"name"`
	Address                string   `json:"address"`
	StateName              string   `json:"state_name"`
	DistrictName           string   `json:"district_name"`
	BlockName              string   `json:"block_name"`
	Pincode                int      `json:"pincode"`
	From                   string   `json:"from"`
	To                     string   `json:"to"`
	Lat                    int      `json:"lat"`
	Long                   int      `json:"long"`
	FeeType                string   `json:"fee_type"`
	SessionId              string   `json:"session_id"`
	Date                   string   `json:"date"`
	AvailableCapacityDose1 int      `json:"available_capacity_dose1"`
	AvailableCapacityDose2 int      `json:"available_capacity_dose2"`
	AvailableCapacity      int      `json:"available_capacity"`
	Fee                    string   `json:"fee"`
	MinAgeLimit            int      `json:"min_age_limit"`
	Vaccine                string   `json:"vaccine"`
	Slots                  []string `json:"slots"`
}

type AllSessions struct {
	Sessions []Session `json:"sessions"`
}

type CloseWebhook struct {
	Dose            int     `json:"dose"`
	Session         Session `json:"session"`
	DurationOpenFor string  `json:"durationOpenFor"`
}

type OpenWebhook struct {
	Dose    int     `json:"dose"`
	Session Session `json:"session"`
}

func ParseSessions(jsonBytes []byte) ([]Session, error) {
	sessions := &AllSessions{}
	err := json.Unmarshal(jsonBytes, sessions)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return sessions.Sessions, nil
}
