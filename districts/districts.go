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
	StateId uint32 `json:"state_id,omitempty"`
	StateName string `json:"state_name,omitempty"`
}

type States struct {
	States []State `json:"states,omitempty"`
	Ttl uint32 `json:"ttl,omitempty"`
}

type District struct {
	DistrictId uint32 `json:"district_id,omitempty"`
	DistrictName string `json:"district_name,omitempty"`
}

type Districts struct {
	Districts []District `json:"districts,omitempty"`
	Ttl uint32 `json:"ttl,omitempty"`
}

type districtState struct {
	district string
	state string
}

type Map struct {
	getStateName map[uint32]string
	getStateID map[string]uint32
	getDistrictInfo map[uint32]districtState
	getDistrictID map[string]map[string]uint32
	districts []uint32
}

func (m *Map) GetStateID(name string) uint32 {
	return m.getStateID[name]
}

func (m *Map) GetStateName(id uint32) string {
	return m.getStateName[id]
}

func (m *Map) GetDistrictInformation(id uint32)  (string, string) {
	return m.getDistrictInfo[id].state, m.getDistrictInfo[id].district
}

func (m *Map) GetDistrictID(state string, district string) uint32 {
	return m.getDistrictID[state][district]
}

// GetDistrictsToPoll chunkNo is 0 indexed
func (m* Map) GetDistrictsToPoll(chunks int, chunkNo int) []uint32 {
	localSlice := m.districts
	sort.Slice(localSlice, func(i, j int) bool { return localSlice[i] < localSlice[j] })
	numberOfDistricts := len(localSlice)
	numPerBucket := numberOfDistricts/chunks
	startIndex := numPerBucket * chunkNo
	endIndex := numPerBucket * chunkNo + numPerBucket
	if chunkNo == chunks - 1 {
		endIndex = numberOfDistricts
	}
	return localSlice[startIndex:endIndex]
}

func (m *Map) GetDistricts() []uint32 {
	return m.districts
}

func GetDistrictsData() (Map, error) {
	m := Map{
		getStateName: map[uint32]string{},
		getStateID: map[string]uint32{},
		getDistrictInfo: map[uint32]districtState{},
		getDistrictID: map[string]map[string]uint32{},
	}

	client := &http.Client{}
	u := "https://api.cowin.gov.in/api/v2/admin/location/states"
	parsedURL, err := url.Parse(u)
	if err != nil {
		log.Fatalln(err)
	}
	req := &http.Request{
		Method: "GET",
		URL: parsedURL,
		Header: map[string][]string{
			"cache-control":{"no-cache"},
			"pragma":{"no-cache"},
		},
	}

	do, err := client.Do(req)
	if err != nil {
		return Map{}, err
	}

	var states States
	err = json.NewDecoder(do.Body).Decode(&states)
	if err != nil {
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
			return Map{}, err
		}
		err = json.NewDecoder(do.Body).Decode(&districts)
		if err != nil {
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
				m.getDistrictID[states.States[i].StateName] = map[string]uint32{}
				m.getDistrictID[states.States[i].StateName][districts.Districts[j].DistrictName] = districts.Districts[j].DistrictId
			}
			m.districts = append(m.districts, districts.Districts[j].DistrictId)
		}

	}
	return m, nil
}

