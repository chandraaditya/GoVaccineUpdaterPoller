package main

import (
	"GoVaccineUpdaterPoller/districts"
	"GoVaccineUpdaterPoller/notifier"
	"GoVaccineUpdaterPoller/poller"
	"GoVaccineUpdaterPoller/webhook"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const rounds = 100

var (
	log logr.Logger
)

func main() {
	l := getLogger("development")
	log = zapr.NewLogger(l)
	go webhook.StartWebhookServer()
	startPolling(log.WithName("start.polling"))
}

func startPolling(log logr.Logger) {
	clientForNotifier := &http.Client{
		Transport: &http.Transport{},
	}
	notifierClient := notifier.NewNotifier()
	polr := poller.NewPoller(0, log.WithName("poller"))
	webhookDistricts, err := webhook.NewDistricts()
	if err != nil {
		log.Error(err, err.Error())
		os.Exit(1)
	}
	districtsFromWebhook := webhookDistricts.GetDistricts()
	districtsToPoll := districts.GetDistrictsToPoll(districtsFromWebhook, 1, 0)
	log.V(1).Info("", "districtsToPoll", districtsToPoll)
	districtsMap, err := districts.GetDistrictsMap()
	if err != nil {
		log.Error(err, err.Error())
		os.Exit(1)
	}
	round := 0
	avgTime := 0.0
	for {
		if round >= rounds {
			status := webhook.Status{}
			status.UpdateFrequency = avgTime / float64(rounds)
			status.LastUpdated = time.Now().Unix()
			marshal, err := json.Marshal(status)
			if err != nil {
				log.Error(err, err.Error())
			} else {
				err = ioutil.WriteFile("status_static_files/status.json", marshal, 0644)
				if err != nil {
					log.Error(err, err.Error())
				}
			}
			avgTime = 0.0
			err = webhookDistricts.UpdateDistricts()
			if err != nil {
				log.Error(err, err.Error())
				continue
			}
			districtsMapTemp, err := districts.GetDistrictsMap()
			if err != nil {
				log.Error(err, err.Error())
			} else {
				districtsMap = districtsMapTemp
			}
			districtsFromWebhook = webhookDistricts.GetDistricts()
			districtsToPoll = districts.GetDistrictsToPoll(districtsFromWebhook, 1, 0)
			round = 0
		}
		start := time.Now()
		requests := polr.GeneratePollRequests(districtsToPoll, 7)
		sessions := polr.RunRequests(requests)
		notifierClient.Notify(sessions, clientForNotifier, webhookDistricts, districtsMap, log.WithName("notifier.notify"))
		fmt.Println(round, time.Since(start))
		avgTime += time.Since(start).Seconds()
		round++
	}
}

func getLogger(logfmt string) *zap.Logger {
	var l *zap.Logger
	var err error

	switch logfmt {
	case "production":
		l, err = zap.NewProduction()
	case "development":
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		l, err = config.Build()
	default:
		err = fmt.Errorf("unknown log format: %v", logfmt)
	}
	if err != nil {
		panic(fmt.Sprintf("log initialization failed: %v", err))
	}
	zap.ReplaceGlobals(l)
	return l
}
