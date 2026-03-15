package validators_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheConfigValidator(t *testing.T) {
	const field = "test"

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.CacheConfig
		}{
			{
				name: "full filled config",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        100 * 1024 * 1024,
					Methods: []string{
						http.MethodGet,
						http.MethodPost,
					},
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.CacheConfigValidator{
					Field: field,
					Value: test.value,
				})

				assert.Empty(t, errors.Errors)
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.CacheConfig
			error string
		}{
			{
				name: "config with empty expiration time",
				value: config.CacheConfig{
					MaxSize: 100 * 1024 * 1024,
					Methods: []string{
						http.MethodGet,
						http.MethodPost,
					},
				},
				error: "test.expiration-time must be greater than 0",
			},
			{
				name: "config with zero max size",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        0,
					Methods: []string{
						http.MethodGet,
						http.MethodPost,
					},
				},
				error: "test.max-size must be greater than 0",
			},
			{
				name: "config with negative max size",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        -1,
					Methods: []string{
						http.MethodGet,
						http.MethodPost,
					},
				},
				error: "test.max-size must be greater than 0",
			},
			{
				name: "config with empty methods",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        100 * 1024 * 1024,
				},
				error: "methods must not be empty",
			},
			{
				name: "config with invalid methods",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        100 * 1024 * 1024,
					Methods: []string{
						"invalid",
					},
				},
				error: "test.methods[0] must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
			{
				name: "config with invalid second method",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					MaxSize:        100 * 1024 * 1024,
					Methods: []string{
						http.MethodGet,
						"invalid",
						http.MethodPost,
					},
				},
				error: "test.methods[1] must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.CacheConfigValidator{
					Field: field,
					Value: test.value,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
