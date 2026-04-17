package validators_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheConfigValidator(t *testing.T) {
	const field = "test"

	t.Run("should not register errors for", func(t *testing.T) {
		var errs validators.Errors
		validators.ValidateCacheConfig(field, config.CacheConfig{
			ExpirationTime: 5 * time.Minute,
			MaxSize:        100 * 1024 * 1024,
			Methods:        []string{http.MethodGet, http.MethodPost},
		}, &errs)
		assert.False(t, errs.HasAny())
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.CacheConfig
			error string
		}{
			{
				name:  "empty expiration time",
				value: config.CacheConfig{MaxSize: 100 * 1024 * 1024, Methods: []string{http.MethodGet}},
				error: "test.expiration-time must be greater than 0",
			},
			{
				name:  "zero max size",
				value: config.CacheConfig{ExpirationTime: 5 * time.Minute, MaxSize: 0, Methods: []string{http.MethodGet}},
				error: "test.max-size must be greater than 0",
			},
			{
				name:  "negative max size",
				value: config.CacheConfig{ExpirationTime: 5 * time.Minute, MaxSize: -1, Methods: []string{http.MethodGet}},
				error: "test.max-size must be greater than 0",
			},
			{
				name:  "empty methods",
				value: config.CacheConfig{ExpirationTime: 5 * time.Minute, MaxSize: 100 * 1024 * 1024},
				error: "methods must not be empty",
			},
			{
				name: "invalid method",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        100 * 1024 * 1024,
					Methods:        []string{"invalid"},
				},
				error: "test.methods[0] must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
			{
				name: "invalid second method",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        100 * 1024 * 1024,
					Methods:        []string{http.MethodGet, "invalid", http.MethodPost},
				},
				error: "test.methods[1] must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var errs validators.Errors
				validators.ValidateCacheConfig(field, test.value, &errs)
				require.EqualError(t, errs, test.error)
			})
		}
	})
}
