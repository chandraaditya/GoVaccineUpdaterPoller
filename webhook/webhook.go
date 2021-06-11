package webhook

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/url"
)

type Districts struct {
	DistrictsWithDestinations map[uint32][]Config
}

type Config struct {
	SlotOpenWebhook       string   `json:"slot_open_webhook,omitempty" mapstructure:"slot-open-webhook"`
	SlotClosedWebhook     string   `json:"slot_closed_webhook,omitempty" mapstructure:"slot-closed-webhook"`
	DistrictsSubscribedTo []uint32 `json:"districts_subscribed_to" mapstructure:"districts"`
}

type APIKeys struct {
	ApiKeys map[string]Config `json:"api_keys" mapstructure:"api-keys"`
}

var config *APIKeys

func VerifyAPIKey(key string) bool {
	_, ok := config.ApiKeys[key]
	return ok
}

func NewDistricts() (Districts, error) {
	w := Districts{}
	err := w.UpdateDistricts()
	if err != nil {
		log.Println("unable to create new webhook", err)
		return w, err
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		_ = w.UpdateDistricts()
	})
	return w, nil
}

func (w *Districts) UpdateDistricts() error {
	//var config APIKeys
	if err := viper.Unmarshal(&config); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

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
	for _, cnf := range apiKeys.ApiKeys {
		for _, currentDistrict := range cnf.DistrictsSubscribedTo {
			if _, ok := newList[currentDistrict]; ok {
				newList[currentDistrict] = append(newList[currentDistrict], cnf)
			} else {
				newList[currentDistrict] = []Config{cnf}
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
