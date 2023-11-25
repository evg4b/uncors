package validators_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/testing/hosts"

	"github.com/stretchr/testify/require"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestMappingValidator(t *testing.T) {
	const field = "mapping"
	const demoJSONPath = "/tmp/demo.json"

	t.Run("should not register errors for", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			demoJSONPath: "{}",
		})

		tests := []struct {
			name  string
			value config.Mapping
		}{
			{
				name: "full filled mapping",
				value: config.Mapping{
					From: "localhost",
					To:   hosts.Github.Host(),
					Statics: []config.StaticDirectory{
						{Path: "/", Dir: "/tmp"},
						{Path: "/", Dir: "/tmp"},
					},
					Mocks: []config.Mock{
						{
							Path:   "/api/info",
							Method: http.MethodGet,
							Response: config.Response{
								Code: 200,
								Raw:  "test",
							},
						},
						{
							Path:   "/api/info/demo",
							Method: http.MethodGet,
							Response: config.Response{
								Code: 300,
								File: demoJSONPath,
							},
						},
					},
					Cache: config.CacheGlobs{
						"/api/constants",
						"/**",
					},
				},
			},
			{
				name: "mapping without mocks and statics and caches",
				value: config.Mapping{
					From:    "localhost",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache:   config.CacheGlobs{},
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.MappingValidator{
					Field: field,
					Value: test.value,
					Fs:    fs,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("should register errors for", func(t *testing.T) {
		fs := testutils.FsFromMap(t, map[string]string{
			demoJSONPath: "{}",
		})

		tests := []struct {
			name  string
			value config.Mapping
			error string
		}{
			{
				name: "mapping without from",
				value: config.Mapping{
					From:    "",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache:   config.CacheGlobs{},
				},
				error: "mapping.from must not be empty",
			},
			{
				name: "mapping without to",
				value: config.Mapping{
					From:    "localhost",
					To:      "",
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache:   config.CacheGlobs{},
				},
				error: "mapping.to must not be empty",
			},
			{
				name: "mapping with invalid statics",
				value: config.Mapping{
					From: "localhost",
					To:   hosts.Github.Host(),
					Statics: []config.StaticDirectory{
						{Path: "/", Dir: "/tmp"},
						{Path: "/", Dir: ""},
					},
					Mocks: []config.Mock{},
					Cache: config.CacheGlobs{},
				},
				error: "mapping.statics[1].directory must not be empty",
			},
			{
				name: "mapping with invalid mocks",
				value: config.Mapping{
					From:    "localhost",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks: []config.Mock{
						{
							Path:   "/api/user",
							Method: "invalid",
							Response: config.Response{
								Code: 200,
								Raw:  "test",
							},
						},
					},
					Cache: config.CacheGlobs{},
				},
				error: "mapping.mocks[0].method must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
			{
				name: "mapping with invalid mocks",
				value: config.Mapping{
					From:    "localhost",
					To:      hosts.Github.Host(),
					Statics: []config.StaticDirectory{},
					Mocks:   []config.Mock{},
					Cache: config.CacheGlobs{
						"/api/info[",
					},
				},
				error: "mapping.cache[0] is not a valid glob pattern",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.MappingValidator{
					Field: field,
					Value: test.value,
					Fs:    fs,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
