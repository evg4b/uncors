package urlreplacer_test

import (
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/stretchr/testify/assert"
)

func TestReplacerToSource(t *testing.T) {
	factory, _ := urlreplacer.NewUrlReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"//demo.localhost.com":         "https://demo.api.com",
		"//custom.domain":              "http://customdomain.com",
		"//custompost.localhost.com":   "https://customdomain.com:8080",
	})

	t.Run("should correctly map url", func(t *testing.T) {
		tests := []struct {
			name      string
			requerUrl *url.URL
			url       string
			expected  string
		}{
			{
				name: "from https to http",
				requerUrl: &url.URL{
					Host:   "premium.localhost.com",
					Scheme: "http",
				},
				url:      "https://premium.api.com/api/info",
				expected: "http://premium.localhost.com/api/info",
			},
			{
				name: "from http to https",
				requerUrl: &url.URL{
					Host:   "base.localhost.com",
					Scheme: "https",
				},
				url:      "http://base.api.com/api/info",
				expected: "https://base.localhost.com/api/info",
			},
			{
				name: "from http to https with custom port",
				requerUrl: &url.URL{
					Host:   "base.localhost.com:4200",
					Scheme: "https",
				},
				url:      "http://base.api.com/api/info",
				expected: "https://base.localhost.com:4200/api/info",
			},
			{
				name: "from https to http with custom port",
				requerUrl: &url.URL{
					Host:   "premium.localhost.com:3000",
					Scheme: "http",
				},
				url:      "https://premium.api.com/api/info",
				expected: "http://premium.localhost.com:3000/api/info",
			},
			{
				name: "from https to http with custom port",
				requerUrl: &url.URL{
					Host:   "custompost.localhost.com:3000",
					Scheme: "http",
				},
				url:      "https://customdomain.com:8080/api/info",
				expected: "http://custompost.localhost.com:3000/api/info",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r, _ := factory.Make(tt.requerUrl)

				actual, err := r.ToSource(tt.url)

				assert.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("should return error when", func(t *testing.T) {
		tests := []struct {
			name          string
			requerUrl     *url.URL
			url           string
			expectedError string
		}{
			{
				name: "scheme in mapping and in url are not equal",
				requerUrl: &url.URL{
					Host:   "demo.localhost.com",
					Scheme: "http",
				},
				url:           "http://demo.api.com",
				expectedError: "target url scheme in mapping (https) and in query (http) are not equal",
			},
			{
				name: "url is invalid",
				requerUrl: &url.URL{
					Host:   "demo.localhost.com",
					Scheme: "http",
				},
				url:           "http://demo:.:a:pi.com",
				expectedError: "filed transform url http://demo:.:a:pi.com to source: parse \"http://demo:.:a:pi.com\": invalid port \":pi.com\" after host",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r, _ := factory.Make(tt.requerUrl)

				actual, err := r.ToSource(tt.url)

				assert.Empty(t, actual)
				assert.EqualError(t, err, tt.expectedError)
			})
		}
	})
}

func TestReplacerToTarget(t *testing.T) {
	factory, _ := urlreplacer.NewUrlReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"//demo.localhost.com":         "https://demo.api.com",
		"//custom.domain":              "http://customdomain.com",
		"//custompost.localhost.com":   "https://customdomain.com:8080",
	})

	t.Run("should correctly map url", func(t *testing.T) {
		tests := []struct {
			name      string
			requerUrl *url.URL
			url       string
			expected  string
		}{
			{
				name: "from https to https",
				requerUrl: &url.URL{
					Host:   "premium.localhost.com",
					Scheme: "http",
				},
				url:      "http://premium.localhost.com/api/info",
				expected: "https://premium.api.com/api/info",
			},
			{
				name: "from http to https",
				requerUrl: &url.URL{
					Host:   "base.localhost.com",
					Scheme: "https",
				},
				url:      "https://base.localhost.com/api/info",
				expected: "http://base.api.com/api/info",
			},
			{
				name: "from http to https with custom port",
				requerUrl: &url.URL{
					Host:   "base.localhost.com:4200",
					Scheme: "https",
				},
				url:      "https://base.localhost.com:4200/api/info",
				expected: "http://base.api.com/api/info",
			},
			{
				name: "from https to http with custom port",
				requerUrl: &url.URL{
					Host:   "premium.localhost.com:3000",
					Scheme: "http",
				},
				url:      "http://premium.localhost.com:3000/api/info",
				expected: "https://premium.api.com/api/info",
			},
			{
				name: "from https to http with custom port",
				requerUrl: &url.URL{
					Host:   "custompost.localhost.com:3000",
					Scheme: "http",
				},
				url:      "http://custompost.localhost.com:3000/api/info",
				expected: "https://customdomain.com:8080/api/info",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r, _ := factory.Make(tt.requerUrl)

				actual, err := r.ToTarget(tt.url)

				assert.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("should return error when", func(t *testing.T) {
		tests := []struct {
			name          string
			requerUrl     *url.URL
			url           string
			expectedError string
		}{
			{
				name: "scheme in mapping and in url are not equal",
				requerUrl: &url.URL{
					Host:   "base.localhost.com",
					Scheme: "https",
				},
				url:           "http://base.localhost.com/api/info",
				expectedError: "target url scheme in mapping (https) and in query (http) are not equal",
			},
			{
				name: "url is invalid",
				requerUrl: &url.URL{
					Host:   "demo.localhost.com",
					Scheme: "http",
				},
				url:           "http://demo.localh::ost.com",
				expectedError: "filed transform url http://demo.localh::ost.com to target: parse \"http://demo.localh::ost.com\": invalid port \":ost.com\" after host",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r, _ := factory.Make(tt.requerUrl)

				actual, err := r.ToTarget(tt.url)

				assert.Empty(t, actual)
				assert.EqualError(t, err, tt.expectedError)
			})
		}
	})
}
