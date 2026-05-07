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
	DefaultMaxSize        = 100 * 1024 * 1024 // 100 MB
)

func setDefaultValues(instance *viper.Viper) {
	instance.SetDefault("cache-config.expiration-time", DefaultExpirationTime)
	instance.SetDefault("cache-config.max-size", DefaultMaxSize)
	instance.SetDefault("cache-config.methods", []string{http.MethodGet})
	instance.SetDefault("interactive", true)
}
