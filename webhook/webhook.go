package webhook

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"
)

type Districts struct {
	DistrictsWithDestinations map[uint32][]Config
}

type Config struct {
	ApiKey                string   `json:"api_key"`
	SlotOpenWebhook       string   `json:"slot_open_webhook,omitempty"`
	SlotClosedWebhook     string   `json:"slot_closed_webhook,omitempty"`
	DistrictsSubscribedTo []uint32 `json:"districts_subscribed_to"`
}

type APIKeys struct {
	ApiKeys []string `json:"api_keys"`
}

func VerifyAPIKey(key string) bool {
	var apiKeys APIKeys
	keys, err := ioutil.ReadFile("webhook_configs/keys.json")
	if err != nil {
		log.Println("error reading webhook keys", err)
		return false
	}
	err = json.Unmarshal(keys, &apiKeys)
	if err != nil {
		log.Println(err)
		return false
	}
	for i := range apiKeys.ApiKeys {
		if key == apiKeys.ApiKeys[i] {
			return true
		}
	}
	return false
}

func NewDistricts() (Districts, error) {
	w := Districts{}
	err := w.UpdateDistricts()
	if err != nil {
		log.Println("unable to create new webhook", err)
		return w, err
	}
	return w, nil
}

func (w *Districts) UpdateDistricts() error {
	keys, err := ioutil.ReadFile("webhook_configs/keys.json")
	if err != nil {
		log.Println("error reading webhook keys", err)
		return err
	}
	var apiKeys APIKeys
	err = json.Unmarshal(keys, &apiKeys)
	if err != nil {
		log.Println(err)
		return err
	}
	newList := map[uint32][]Config{}
	for i := range apiKeys.ApiKeys {
		configFile, err := ioutil.ReadFile("webhook_configs/" + apiKeys.ApiKeys[i] + ".json")
		if err != nil {
			log.Println("error reading webhook config json:", err)
			continue
		}
		var config Config
		err = json.Unmarshal(configFile, &config)
		for i := range config.DistrictsSubscribedTo {
			currentDistrict := config.DistrictsSubscribedTo[i]
			if _, ok := newList[currentDistrict]; ok {
				newList[currentDistrict] = append(newList[currentDistrict], config)
			} else {
				newList[currentDistrict] = []Config{config}
			}
		}
	}
	w.DistrictsWithDestinations = newList
	return nil
}

func (w *Districts) GetDistricts() []uint32 {
	slice := make([]uint32, 0)
	for i := range w.DistrictsWithDestinations {
		slice = append(slice, i)
	}
	return slice
}

func (w *Districts) GetOpenWebhooksForDistrict(district uint32) []*url.URL {
	webhooks := make([]*url.URL, 0)
	configs := w.DistrictsWithDestinations[district]
	for _, config := range configs {
		URL, err := url.ParseRequestURI(config.SlotOpenWebhook)
		if err != nil {
			continue
		}
		webhooks = append(webhooks, URL)
	}
	return webhooks
}

func (w *Districts) GetCloseWebhooksForDistrict(district uint32) []*url.URL {
	webhooks := make([]*url.URL, 0)
	configs := w.DistrictsWithDestinations[district]
	for _, config := range configs {
		URL, err := url.ParseRequestURI(config.SlotClosedWebhook)
		if err != nil {
			continue
		}
		webhooks = append(webhooks, URL)
	}
	return webhooks
}
