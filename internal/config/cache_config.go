package config

import (
	"fmt"
	"time"

	multierror "github.com/hashicorp/go-multierror"
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

func (c *CacheConfig) Validate(field string) error {
	var errs *multierror.Error

	errs = multierror.Append(errs, ValidateDuration(joinPath(field, "expiration-time"), c.ExpirationTime, false))

	if c.MaxSize <= 0 {
		msg := fmt.Sprintf("%s must be greater than 0", joinPath(field, "max-size"))
		errs = multierror.Append(errs, &ValidationError{msg})
	}

	if len(c.Methods) == 0 {
		errs = multierror.Append(errs, &ValidationError{"methods must not be empty"})
	}

	for i, method := range c.Methods {
		errs = multierror.Append(errs, ValidateMethod(joinPath(field, "methods", index(i)), method, false))
	}

	return joinErrors(errs)
}
