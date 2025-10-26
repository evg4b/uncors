package validators_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/require"
)

func TestUncorsConfigValidator(t *testing.T) {
	mapFs := testutils.FsFromMap(t, map[string]string{})

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value *config.UncorsConfig
		}{
			{
				name: "minimal config",
				value: &config.UncorsConfig{
					HTTPPort: 8080,
					Mappings: []config.Mapping{
						{From: hosts.Localhost.Port(8080), To: hosts.Localhost.HTTPSPort(8443)},
					},
					CacheConfig: config.CacheConfig{
						ClearTime:      10 * time.Minute,
						ExpirationTime: 10 * time.Minute,
						Methods:        []string{http.MethodGet},
					},
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validators.ValidateConfig(test.value, mapFs)

				require.NoError(t, errors)
			})
		}
	})

	t.Run("should register errors for invalid config", func(t *testing.T) {
		tests := []struct {
			name  string
			value *config.UncorsConfig
			error string
		}{
			{
				name: "invalid http port",
				value: &config.UncorsConfig{
					HTTPPort: 0,
					Mappings: []config.Mapping{
						{From: hosts.Localhost.Port(8080), To: hosts.Localhost.HTTPSPort(8443)},
					},
					CacheConfig: config.CacheConfig{
						ClearTime:      10 * time.Minute,
						ExpirationTime: 10 * time.Minute,
						Methods:        []string{http.MethodGet},
					},
				},
				error: "http-port must be between 1 and 65535",
			},
			{
				name: "invalid mapping",
				value: &config.UncorsConfig{
					HTTPPort: 8080,
					Mappings: []config.Mapping{},
					CacheConfig: config.CacheConfig{
						ClearTime:      10 * time.Minute,
						ExpirationTime: 10 * time.Minute,
						Methods:        []string{http.MethodGet},
					},
				},
				error: "mappings must not be empty",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validators.ValidateConfig(test.value, mapFs)

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
