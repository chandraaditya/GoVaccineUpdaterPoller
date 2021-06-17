package webhook

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"log"
	"net/url"
)

type Districts struct {
	DistrictsWithDestinations map[int][]Config
}

type Config struct {
	SlotOpenWebhook       string `json:"slot_open_webhook,omitempty" mapstructure:"slot-open-webhook"`
	SlotClosedWebhook     string `json:"slot_closed_webhook,omitempty" mapstructure:"slot-closed-webhook"`
	DistrictsSubscribedTo []int  `json:"districts" mapstructure:"districts"`
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
	var c APIKeys

	if err := viper.Unmarshal(&c); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	config = &c

	log.Println(c)

	newList := map[int][]Config{}
	for _, cnf := range c.ApiKeys {
		for _, currentDistrict := range cnf.DistrictsSubscribedTo {
			if _, ok := newList[currentDistrict]; ok {
				newList[currentDistrict] = append(newList[currentDistrict], cnf)
			} else {
				newList[currentDistrict] = []Config{cnf}
			}
		}
	}
	log.Println(w.DistrictsWithDestinations)
	w.DistrictsWithDestinations = newList
	log.Println(w.DistrictsWithDestinations)
	return nil
}

func (w *Districts) GetDistricts() []int {
	slice := make([]int, 0)
	for i := range w.DistrictsWithDestinations {
		slice = append(slice, i)
	}
	return slice
}

func (w *Districts) GetOpenWebhooksForDistrict(district int) []*url.URL {
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

func (w *Districts) GetCloseWebhooksForDistrict(district int) []*url.URL {
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
