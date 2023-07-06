package config

import (
	"time"
)

type CacheGlobs []string

func (globs CacheGlobs) Clone() CacheGlobs {
	if globs == nil {
		return nil
	}

	cacheGlobs := make(CacheGlobs, 0, len(globs))
	cacheGlobs = append(cacheGlobs, globs...)

	return cacheGlobs
}

type CacheConfig struct {
	ExpirationTime time.Duration `mapstructure:"expiration-time"`
	ClearTime      time.Duration `mapstructure:"clear-time"`
}

func (config *CacheConfig) Clone() *CacheConfig {
	return &CacheConfig{
		ExpirationTime: config.ExpirationTime,
		ClearTime:      config.ClearTime,
	}
}
