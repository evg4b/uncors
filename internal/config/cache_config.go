package config

import (
	"fmt"
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
	ExpirationTime time.Duration `yaml:"expiration-time"`
	MaxSize        int64         `yaml:"max-size"`
	Methods        []string      `yaml:"methods"`
}

func (c *CacheConfig) Clone() *CacheConfig {
	var methods []string
	if c.Methods != nil {
		methods = append(methods, c.Methods...)
	}

	return &CacheConfig{
		ExpirationTime: c.ExpirationTime,
		MaxSize:        c.MaxSize,
		Methods:        methods,
	}
}

func (c *CacheConfig) Validate(field string, errs *Errors) {
	ValidateDuration(joinPath(field, "expiration-time"), c.ExpirationTime, false, errs)

	if c.MaxSize <= 0 {
		errs.add(fmt.Sprintf("%s must be greater than 0", joinPath(field, "max-size")))
	}

	if len(c.Methods) == 0 {
		errs.add("methods must not be empty")
	}

	for i, method := range c.Methods {
		ValidateMethod(joinPath(field, "methods", index(i)), method, false, errs)
	}
}
