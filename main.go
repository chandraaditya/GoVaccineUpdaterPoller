package main

import (
	"GoVaccineUpdaterPoller/districts"
	"GoVaccineUpdaterPoller/notifier"
	"GoVaccineUpdaterPoller/parser"
	"GoVaccineUpdaterPoller/poller"
	"GoVaccineUpdaterPoller/webhook"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	log    logr.Logger
	logOpt string
)

func init() {

	// Predence order is CLI -> ENV -> FILE

	pflag.String("log", "development", "log format: production or development")
	pflag.String("config", "", "Configuration location")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("/etc/poller/")   // path to look for the config file in
	viper.AddConfigPath("$HOME/.poller/") // call multiple times to add many search paths
	viper.AddConfigPath(".")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	//defaults
	viper.SetDefault("log", "development")
	viper.SetDefault("poller.exit", false)
	viper.SetDefault("poller.no-of-days", 7)
	viper.SetDefault("poller.chunks.total", 1)
	viper.SetDefault("poller.chunks.current", 0)

	viper.SetDefault("notifier.cache-type", "in-memory")
	viper.SetDefault("notifier.redis.host", "localhost:6379")
	viper.SetDefault("notifier.redis.ttl", 24*time.Hour)

	viper.AutomaticEnv()
}

func main() {
	pflag.Parse()
	_ = viper.BindPFlags(pflag.CommandLine)
	configLocation := viper.GetString("config")
	if configLocation != "" {
		println("Adding log file " + configLocation)
		viper.SetConfigFile(configLocation)
	}
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	logOpt = viper.GetString("log")
	if logOpt == "development" {
		viper.Debug()
	}
	l := getLogger(logOpt)
	log = zapr.NewLogger(l)
	defer func() { _ = l.Sync() }()

	go webhook.StartWebhookServer(log.WithName("server"))
	startPolling(log.WithName("start.polling"))
}

func startPolling(log logr.Logger) {
	clientForNotifier := &http.Client{
		Transport: &http.Transport{},
	}
	notifierClient := notifier.NewNotifier(viper.GetString("notifier.cache-type"),
		log.WithName("notifier"), viper.GetString("notifier.redis.host"),
		viper.GetString("notifier.redis.password"), viper.GetInt("notifier.redis.db-index"), viper.GetDuration("notifier.redis.ttl"))
	polr := poller.NewPoller(100*time.Millisecond, log.WithName("poller"))
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
	rounds := viper.GetInt("poller.no-of-rounds")
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
			districtsMapTemp, err := districts.GetDistrictsMap()
			if err != nil {
				log.Error(err, err.Error())
			} else {
				districtsMap = districtsMapTemp
			}
			districtsFromWebhook = webhookDistricts.GetDistricts()
			districtsToPoll = districts.GetDistrictsToPoll(districtsFromWebhook, viper.GetInt("poller.chunks.total"), viper.GetInt("poller.chunks.current"))
			round = 0
			if viper.GetBool("poller.exit") {
				return
			}
		}
		start := time.Now()
		requests := polr.GeneratePollRequests(districtsToPoll, viper.GetInt("poller.no-of-days"))
		responseChannel := make(chan parser.Session)
		go polr.RunRequests(requests, responseChannel)
		notifierClient.Notify(responseChannel, clientForNotifier, webhookDistricts, districtsMap, log.WithName("notifier.notify"))
		log.Info("round over", "round number", round, "time taken", time.Since(start))
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
