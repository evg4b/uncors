package config

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

type CacheGlobs []string

func (g CacheGlobs) Clone() CacheGlobs {
	return slices.Clone(g)
}

type CacheConfig struct {
	ExpirationTime time.Duration `yaml:"expiration-time,omitempty"`
	MaxSize        int64         `yaml:"max-size,omitempty"`
	Methods        []string      `yaml:"methods,omitempty"`
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
		errs = append(errs, &ValidationError{joinPath(field, "methods") + " must not be empty"})
	}

	for i, method := range c.Methods {
		errs = append(errs, validateCacheMethod(joinPath(field, "methods", index(i)), method))
	}

	return errors.Join(errs...)
}

var cacheableMethods = slices.DeleteFunc(slices.Clone(allowedMethods), func(method string) bool {
	return method == http.MethodConnect
})

func validateCacheMethod(field, value string) error {
	if !slices.Contains(cacheableMethods, value) {
		return &ValidationError{fmt.Sprintf("%s must be one of %s", field, strings.Join(cacheableMethods, ", "))}
	}

	return nil
}
