package config

import (
	"net/http"
	"time"

	"github.com/spf13/viper"
)

const (
	defaultHTTPPort       = 80
	defaultHTTPSPort      = 443
	DefaultExpirationTime = 30 * time.Minute
	DefaultClearTime      = 30 * time.Minute
)

func setDefaultValues(instance *viper.Viper) {
	instance.SetDefault("cache-config.expiration-time", DefaultExpirationTime)
	instance.SetDefault("cache-config.clear-time", DefaultClearTime)
	instance.SetDefault("cache-config.methods", []string{http.MethodGet})
}
