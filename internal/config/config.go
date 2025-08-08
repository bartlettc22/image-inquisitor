package config

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func Init() {
	logLevelStr := viper.GetString("log-level")
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		log.Fatalf("invalid --log-level: %s", logLevelStr)
	}
	log.SetLevel(logLevel)
	log.WithField("log-level", logLevel.String()).Debug("log level set")

	logFormat := viper.GetString("log-format")
	switch logFormat {
	case "logfmt":
		log.SetFormatter(&log.TextFormatter{
			ForceColors: true,
		})
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.Fatalf("invalid --log-format: %s", logFormat)
	}
}
