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
	Methods        []string      `mapstructure:"methods"`
}

func (config *CacheConfig) Clone() *CacheConfig {
	var methods []string = nil
	if config.Methods != nil {
		methods = append(methods, config.Methods...)
	}

	return &CacheConfig{
		ExpirationTime: config.ExpirationTime,
		ClearTime:      config.ClearTime,
		Methods:        methods,
	}
}
