package webhook

import (
	districts2 "GoVaccineUpdaterPoller/districts"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
	"strconv"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"time"
)

func StartWebhookServer(log logr.Logger) {
	s := &server{
		log: log,
	}
	http.HandleFunc("/update_districts", s.UpdateDistricts)
	http.HandleFunc("/status", s.Status)
	panic(http.ListenAndServe(":8080", nil))
}

type server struct {
	log logr.Logger
}

func (s *server) UpdateDistricts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte(`{"message": "this endpoint only supports post http requests contact the developer for more information"}`))
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			s.log.V(1).Error(err, err.Error())
			return
		}
		return
	}

	var cnf Config
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&cnf)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(`{"message": "bad request"}`))
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			s.log.V(1).Error(err, err.Error())
			return
		}
		return
	}

	apiKey := r.Header.Get("X-Api-Key")
	verified := VerifyAPIKey(apiKey)
	if !verified {
		s.log.V(1).Info("invalid api key")
		w.WriteHeader(http.StatusUnauthorized)
		_, err = w.Write([]byte(`{"message": "unauthorized request"}`))
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			s.log.V(1).Error(err, err.Error())
			return
		}
		return
	}

	_, errPRU1 := url.ParseRequestURI(cnf.SlotOpenWebhook)
	_, errPRU2 := url.ParseRequestURI(cnf.SlotClosedWebhook)
	if errPRU1 != nil && errPRU2 != nil {
		s.log.V(1).Error(errPRU1, errPRU1.Error())
		s.log.V(1).Error(errPRU2, errPRU2.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte(`{"message": "invalid slot_open_webhook or slot_closed_webhook, either one of the fields must be valid endpoints, but both are not required"}`))
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			s.log.V(1).Error(err, err.Error())
			return
		}
		return
	}

	districtsMap, err := districts2.GetDistrictsMap()
	if err != nil {
		s.log.V(1).Error(err, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(`{"message": "internal server error"}`))
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			s.log.V(1).Error(err, err.Error())
			return
		}
		return
	}

	for i := range cnf.DistrictsSubscribedTo {
		if !districtsMap.VerifyDistrict(cnf.DistrictsSubscribedTo[i]) {
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte(`{"message": "` + strconv.Itoa(cnf.DistrictsSubscribedTo[i]) + ` is not a valid district"}`))
			if err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				s.log.V(1).Error(err, err.Error())
				return
			}
			return
		}
	}

	viper.Set("api-keys."+apiKey+".slot-open-webhook", cnf.SlotOpenWebhook)
	viper.Set("api-keys."+apiKey+".slot-closed-webhook", cnf.SlotClosedWebhook)
	viper.Set("api-keys."+apiKey+".districts", cnf.DistrictsSubscribedTo)

	err = viper.WriteConfig()

	if err != nil {
		s.log.V(1).Error(err, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte(`{"message": "internal server error"}`))
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			s.log.V(1).Error(err, err.Error())
			return
		}
		return
	}

	err = r.Body.Close()
	if err != nil {
		s.log.V(1).Error(err, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(`{"message": "successfully updated"}`))
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		s.log.V(1).Error(err, err.Error())
		return
	}
}

type Status struct {
	UpdateFrequency float64 `json:"update_frequency"`
	LastUpdated     int64   `json:"last_updated"`
}

func (s *server) Status(w http.ResponseWriter, _ *http.Request) {
	var status Status
	keys, err := ioutil.ReadFile("status_static_files/status.json")
	if err != nil {
		s.log.V(1).Error(err, err.Error())
		_, err = fmt.Fprintf(w, "unknown error")
		if err != nil {
			return
		}
	}
	err = json.Unmarshal(keys, &status)
	if err != nil {
		s.log.V(1).Error(err, err.Error())
		_, err = fmt.Fprintf(w, "unknown error")
		if err != nil {
			s.log.V(1).Error(err, err.Error())
			return
		}
	}
	tm := time.Unix(status.LastUpdated, 0).Local()
	_, err = fmt.Fprintf(w, "update frequency: %fs\nlast updated: %s", status.UpdateFrequency, tm.String())
	if err != nil {
		s.log.V(1).Error(err, err.Error())
		_, err = fmt.Fprintf(w, "unknown error")
		if err != nil {
			s.log.V(1).Error(err, err.Error())
			return
		}
	}
}
