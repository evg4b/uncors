// nolint: lll, dupl
package urlreplacer_test

import (
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/stretchr/testify/assert"
)

func TestReplacerToSource(t *testing.T) {
	factory, _ := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"//demo.localhost.com":         "https://demo.api.com",
		"//custom.domain":              "http://customdomain.com",
		"//custompost.localhost.com":   "https://customdomain.com:8080",
	})

	t.Run("should correctly map url", func(t *testing.T) {
		tests := []struct {
			name      string
			requerURL *url.URL
			url       string
			expected  string
		}{
			{
				name: "from https to http",
				requerURL: &url.URL{
					Host:   "premium.localhost.com",
					Scheme: "http",
				},
				url:      "https://premium.api.com/api/info",
				expected: "http://premium.localhost.com/api/info",
			},
			{
				name: "from http to https",
				requerURL: &url.URL{
					Host:   "base.localhost.com",
					Scheme: "https",
				},
				url:      "http://base.api.com/api/info",
				expected: "https://base.localhost.com/api/info",
			},
			{
				name: "from http to https with custom port",
				requerURL: &url.URL{
					Host:   "base.localhost.com:4200",
					Scheme: "https",
				},
				url:      "http://base.api.com/api/info",
				expected: "https://base.localhost.com:4200/api/info",
			},
			{
				name: "from https to http with custom port",
				requerURL: &url.URL{
					Host:   "premium.localhost.com:3000",
					Scheme: "http",
				},
				url:      "https://premium.api.com/api/info",
				expected: "http://premium.localhost.com:3000/api/info",
			},
			{
				name: "from https to http with custom port",
				requerURL: &url.URL{
					Host:   "custompost.localhost.com:3000",
					Scheme: "http",
				},
				url:      "https://customdomain.com:8080/api/info",
				expected: "http://custompost.localhost.com:3000/api/info",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				r, _ := factory.Make(testCase.requerURL)

				actual, err := r.ToSource(testCase.url)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("should return error when", func(t *testing.T) {
		tests := []struct {
			name          string
			requerURL     *url.URL
			url           string
			expectedError string
		}{
			{
				name: "scheme in mapping and in url are not equal",
				requerURL: &url.URL{
					Host:   "demo.localhost.com",
					Scheme: "http",
				},
				url:           "http://demo.api.com",
				expectedError: "failed to transform url from https to http: scheme in mapping and query not matched",
			},
			{
				name: "url is invalid",
				requerURL: &url.URL{
					Host:   "demo.localhost.com",
					Scheme: "http",
				},
				url:           "http://demo:.:a:pi.com",
				expectedError: "filed transform url http://demo:.:a:pi.com to source: parse \"http://demo:.:a:pi.com\": invalid port \":pi.com\" after host",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				r, _ := factory.Make(testCase.requerURL)

				actual, err := r.ToSource(testCase.url)

				assert.Empty(t, actual)
				assert.EqualError(t, err, testCase.expectedError)
			})
		}
	})
}

func TestReplacerToTarget(t *testing.T) {
	factory, _ := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"//demo.localhost.com":         "https://demo.api.com",
		"//custom.domain":              "http://customdomain.com",
		"//custompost.localhost.com":   "https://customdomain.com:8080",
	})

	t.Run("should correctly map url", func(t *testing.T) {
		tests := []struct {
			name      string
			requerURL *url.URL
			url       string
			expected  string
		}{
			{
				name: "from https to https",
				requerURL: &url.URL{
					Host:   "premium.localhost.com",
					Scheme: "http",
				},
				url:      "http://premium.localhost.com/api/info",
				expected: "https://premium.api.com/api/info",
			},
			{
				name: "from http to https",
				requerURL: &url.URL{
					Host:   "base.localhost.com",
					Scheme: "https",
				},
				url:      "https://base.localhost.com/api/info",
				expected: "http://base.api.com/api/info",
			},
			{
				name: "from http to https with custom port",
				requerURL: &url.URL{
					Host:   "base.localhost.com:4200",
					Scheme: "https",
				},
				url:      "https://base.localhost.com:4200/api/info",
				expected: "http://base.api.com/api/info",
			},
			{
				name: "from https to http with custom port",
				requerURL: &url.URL{
					Host:   "premium.localhost.com:3000",
					Scheme: "http",
				},
				url:      "http://premium.localhost.com:3000/api/info",
				expected: "https://premium.api.com/api/info",
			},
			{
				name: "from https to http with custom port",
				requerURL: &url.URL{
					Host:   "custompost.localhost.com:3000",
					Scheme: "http",
				},
				url:      "http://custompost.localhost.com:3000/api/info",
				expected: "https://customdomain.com:8080/api/info",
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				r, _ := factory.Make(testCase.requerURL)

				actual, err := r.ToTarget(testCase.url)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("should return error when", func(t *testing.T) {
		tests := []struct {
			name          string
			requerURL     *url.URL
			url           string
			expectedError string
		}{
			{
				name: "scheme in mapping and in url are not equal",
				requerURL: &url.URL{
					Host:   "base.localhost.com",
					Scheme: "https",
				},
				url:           "http://base.localhost.com/api/info",
				expectedError: "failed to transform url from https to http: scheme in mapping and query not matched",
			},
			{
				name: "url is invalid",
				requerURL: &url.URL{
					Host:   "demo.localhost.com",
					Scheme: "http",
				},
				url:           "http://demo.localh::ost.com",
				expectedError: "filed transform url http://demo.localh::ost.com to target: parse \"http://demo.localh::ost.com\": invalid port \":ost.com\" after host",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				r, _ := factory.Make(testCase.requerURL)

				actual, err := r.ToTarget(testCase.url)

				assert.Empty(t, actual)
				assert.EqualError(t, err, testCase.expectedError)
			})
		}
	})
}
