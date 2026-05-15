package config

import (
	"net/http"
	"time"
)

const (
	defaultHTTPPort       = 80
	defaultHTTPSPort      = 443
	DefaultExpirationTime = 30 * time.Minute
	DefaultMaxSize        = 100 * 1024 * 1024 // 100 MB
)

// defaultConfig returns a new UncorsConfig with all default values applied.
func defaultConfig() *UncorsConfig {
	return &UncorsConfig{
		Mappings: Mappings{},
		CacheConfig: CacheConfig{
			ExpirationTime: DefaultExpirationTime,
			MaxSize:        DefaultMaxSize,
			Methods:        []string{http.MethodGet},
		},
		Interactive: true,
	}
}
