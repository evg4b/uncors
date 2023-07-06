package config

import (
	"time"

	"github.com/spf13/viper"
)

const (
	DefaultExpirationTime = 30 * time.Minute
	DefaultClearTime      = 30 * time.Minute
)

func setDefaultValues(instance *viper.Viper) {
	instance.SetDefault("cache-config.expiration-time", DefaultExpirationTime)
	instance.SetDefault("cache-config.clear-time", DefaultClearTime)
}
