package config

import (
	"net/http"
	"time"
)

const (
	defaultHTTPPort  = 80
	defaultHTTPSPort = 443
	// DefaultListenAddress keeps the proxy reachable only from this machine.
	DefaultListenAddress  = "127.0.0.1"
	DefaultExpirationTime = Duration(30 * time.Minute)
	DefaultMaxSize        = 100 * 1024 * 1024 // 100 MB
)

func defaultConfig() *UncorsConfig {
	return &UncorsConfig{
		Mappings: Mappings{},
		CacheConfig: CacheConfig{
			ExpirationTime: DefaultExpirationTime,
			MaxSize:        DefaultMaxSize,
			Methods:        []string{http.MethodGet},
		},
		Listen:      DefaultListenAddress,
		Interactive: true,
	}
}
