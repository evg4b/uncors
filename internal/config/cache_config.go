package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrInvalidCacheConfig is returned when the cache-config YAML value is not a mapping.
var ErrInvalidCacheConfig = errors.New("expected a mapping for cache-config")

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
	ExpirationTime time.Duration `yaml:"-"`
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

// UnmarshalYAML implements custom decoding so that the "expiration-time" field
// can be expressed as a human-readable duration string (e.g. "30m", "1h").
// Other fields are decoded by the standard yaml.v3 machinery.
// Only fields present in the YAML node are updated; existing values (defaults)
// are preserved for absent keys.
func (c *CacheConfig) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return ErrInvalidCacheConfig
	}

	for i := 0; i+1 < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]

		switch keyNode.Value {
		case "expiration-time":
			dur, err := time.ParseDuration(strings.ReplaceAll(valNode.Value, " ", ""))
			if err != nil {
				return fmt.Errorf("invalid expiration-time %q: %w", valNode.Value, err)
			}

			c.ExpirationTime = dur
		case "max-size":
			err := valNode.Decode(&c.MaxSize)
			if err != nil {
				return err
			}
		case "methods":
			err := valNode.Decode(&c.Methods)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
