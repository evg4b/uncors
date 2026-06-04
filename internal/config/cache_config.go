package config

import (
	"errors"
	"fmt"
	"slices"
	"time"
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
	var errs []error

	errs = append(errs, ValidateDuration(joinPath(field, "expiration-time"), c.ExpirationTime, false))

	if c.MaxSize <= 0 {
		msg := fmt.Sprintf("%s must be greater than 0", joinPath(field, "max-size"))
		errs = append(errs, &ValidationError{msg})
	}

	if len(c.Methods) == 0 {
		errs = append(errs, &ValidationError{"methods must not be empty"})
	}

	for i, method := range c.Methods {
		errs = append(errs, ValidateMethod(joinPath(field, "methods", index(i)), method, false))
	}

	return errors.Join(errs...)
}
