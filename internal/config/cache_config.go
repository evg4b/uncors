package config

import (
	"fmt"
	"slices"
	"time"

	multierror "github.com/hashicorp/go-multierror"
)

type CacheGlobs []string

func (g CacheGlobs) Clone() CacheGlobs {
	return slices.Clone(g)
}

type CacheConfig struct {
	ExpirationTime time.Duration `yaml:"expiration-time"`
	MaxSize        int64         `yaml:"max-size"`
	Methods        []string      `yaml:"methods"`
}

func (c *CacheConfig) Clone() *CacheConfig {
	return &CacheConfig{
		ExpirationTime: c.ExpirationTime,
		MaxSize:        c.MaxSize,
		Methods:        slices.Clone(c.Methods),
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
