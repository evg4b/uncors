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
					ClearTime:      30 * time.Second,
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
					ClearTime: 30 * time.Second,
					Methods: []string{
						http.MethodGet,
						http.MethodPost,
					},
				},
				error: "test.expiration-time must be greater than 0",
			},
			{
				name: "config with empty clear time",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					Methods: []string{
						http.MethodGet,
						http.MethodPost,
					},
				},
				error: "test.clear-time must be greater than 0",
			},

			{
				name: "config with empty methods",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					ClearTime:      30 * time.Second,
				},
				error: "methods must not be empty",
			},
			{
				name: "config with invalid methods",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					ClearTime:      30 * time.Second,
					Methods: []string{
						"invalid",
					},
				},
				error: "test.methods[0] must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
			{
				name: "config with invalid methods",
				value: config.CacheConfig{
					ExpirationTime: 5 * time.Minute,
					ClearTime:      30 * time.Second,
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
