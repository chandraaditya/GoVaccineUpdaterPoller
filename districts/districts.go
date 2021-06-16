package districts

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
)

type State struct {
	StateId   int    `json:"state_id,omitempty"`
	StateName string `json:"state_name,omitempty"`
}

type States struct {
	States []State `json:"states,omitempty"`
	Ttl    int     `json:"ttl,omitempty"`
}

type District struct {
	DistrictId   int    `json:"district_id,omitempty"`
	DistrictName string `json:"district_name,omitempty"`
}

type Districts struct {
	Districts []District `json:"districts,omitempty"`
	Ttl       int        `json:"ttl,omitempty"`
}

type districtState struct {
	district string
	state    string
}

type Map struct {
	getStateName    map[int]string
	getStateID      map[string]int
	getDistrictInfo map[int]districtState
	getDistrictID   map[string]map[string]int
	districts       []int
}

func (m *Map) GetStateID(name string) int {
	return m.getStateID[name]
}

func (m *Map) GetStateName(id int) string {
	return m.getStateName[id]
}

func (m *Map) GetDistrictInformation(id int) (string, string) {
	return m.getDistrictInfo[id].state, m.getDistrictInfo[id].district
}

func (m *Map) GetDistrictID(state string, district string) int {
	return m.getDistrictID[state][district]
}

func (m *Map) VerifyDistrict(id int) bool {
	for i := range m.districts {
		if id == m.districts[i] {
			return true
		}
	}
	return false
}

// GetDistrictsToPoll chunkNo is 0 indexed
func GetDistrictsToPoll(districts []int, chunks int, chunkNo int) []int {
	localSlice := districts
	sort.Slice(localSlice, func(i, j int) bool { return localSlice[i] < localSlice[j] })
	numberOfDistricts := len(localSlice)
	numPerBucket := numberOfDistricts / chunks
	startIndex := numPerBucket * chunkNo
	endIndex := numPerBucket*chunkNo + numPerBucket
	if chunkNo == chunks-1 {
		endIndex = numberOfDistricts
	}
	return localSlice[startIndex:endIndex]
}

func (m *Map) GetDistricts() []int {
	return m.districts
}

func GetDistrictsMap() (Map, error) {
	m := Map{
		getStateName:    map[int]string{},
		getStateID:      map[string]int{},
		getDistrictInfo: map[int]districtState{},
		getDistrictID:   map[string]map[string]int{},
	}

	client := &http.Client{}
	u := "https://api.cowin.gov.in/api/v2/admin/location/states"
	parsedURL, err := url.Parse(u)
	if err != nil {
		log.Println(err)
		return Map{}, err
	}
	req := &http.Request{
		Method: "GET",
		URL:    parsedURL,
		Header: map[string][]string{
			"cache-control": {"no-cache"},
			"pragma":        {"no-cache"},
		},
	}

	do, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return Map{}, err
	}

	var states States
	err = json.NewDecoder(do.Body).Decode(&states)
	if err != nil {
		log.Println(err)
		return Map{}, err
	}

	for i := 0; i < len(states.States); i++ {
		m.getStateID[states.States[i].StateName] = states.States[i].StateId
		m.getStateName[states.States[i].StateId] = states.States[i].StateName

		var districts Districts

		u = "https://api.cowin.gov.in/api/v2/admin/location/districts/" + fmt.Sprint(states.States[i].StateId)
		parsedURL, err = url.Parse(u)
		req.URL = parsedURL
		do, err = client.Do(req)
		if err != nil {
			log.Println(err)
			return Map{}, err
		}
		err = json.NewDecoder(do.Body).Decode(&districts)
		if err != nil {
			log.Println(err)
			return Map{}, err
		}

		for j := 0; j < len(districts.Districts); j++ {
			m.getDistrictInfo[districts.Districts[j].DistrictId] = districtState{
				district: districts.Districts[j].DistrictName,
				state:    states.States[i].StateName,
			}
			if _, ok := m.getDistrictID[states.States[i].StateName]; ok {
				m.getDistrictID[states.States[i].StateName][districts.Districts[j].DistrictName] = districts.Districts[j].DistrictId
			} else {
				m.getDistrictID[states.States[i].StateName] = map[string]int{}
				m.getDistrictID[states.States[i].StateName][districts.Districts[j].DistrictName] = districts.Districts[j].DistrictId
			}
			m.districts = append(m.districts, districts.Districts[j].DistrictId)
		}

	}
	return m, nil
}
