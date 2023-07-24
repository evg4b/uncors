package config

import (
	"time"
)

type CacheGlobs []string

func (g CacheGlobs) Clone() CacheGlobs {
	if g == nil {
		return nil
	}

	cacheGlobs := make(CacheGlobs, 0, len(g))
	cacheGlobs = append(cacheGlobs, g...)

	return cacheGlobs
}

type CacheConfig struct {
	ExpirationTime time.Duration `mapstructure:"expiration-time"`
	ClearTime      time.Duration `mapstructure:"clear-time"`
	Methods        []string      `mapstructure:"methods"`
}

func (c *CacheConfig) Clone() *CacheConfig {
	var methods []string
	if c.Methods != nil {
		methods = append(methods, c.Methods...)
	}

	return &CacheConfig{
		ExpirationTime: c.ExpirationTime,
		ClearTime:      c.ClearTime,
		Methods:        methods,
	}
}
