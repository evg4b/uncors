// nolint: lll, dupl
package urlreplacer_test

import (
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestReplacerToSourceMapping(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"demo.localhost.com":           "https://demo.api.com",
		"custom.domain":                "http://customdomain.com",
		"custompost.localhost.com":     "https://customdomain.com:8080",
	})
	testutils.CheckNoError(t, err)

	tests := []struct {
		name       string
		requestURL string
		url        string
		expected   string
	}{
		{
			name:       "from https to http",
			requestURL: "http://premium.localhost.com",
			url:        "https://premium.api.com/api/info",
			expected:   "http://premium.localhost.com/api/info",
		},
		{
			name:       "from http to https",
			requestURL: "https://base.localhost.com",
			url:        "http://base.api.com/api/info",
			expected:   "https://base.localhost.com/api/info",
		},
		{
			name:       "from http to https with custom port",
			requestURL: "https://base.localhost.com:4200",
			url:        "http://base.api.com/api/info",
			expected:   "https://base.localhost.com:4200/api/info",
		},
		{
			name:       "from https to http with custom port",
			requestURL: "http://premium.localhost.com:3000",
			url:        "https://premium.api.com/api/info",
			expected:   "http://premium.localhost.com:3000/api/info",
		},
		{
			name:       "from https to http with custom port",
			requestURL: "http://custompost.localhost.com:3000",
			url:        "https://customdomain.com:8080/api/info",
			expected:   "http://custompost.localhost.com:3000/api/info",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			parsedURL, err := urlx.Parse(testCase.requestURL)
			testutils.CheckNoError(t, err)

			replacer, err := factory.Make(parsedURL)
			testutils.CheckNoError(t, err)

			t.Run("ToSource", func(t *testing.T) {
				actual, err := replacer.StringToSource(testCase.url)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, actual)
			})

			t.Run("ToSourceURL", func(t *testing.T) {
				parsedTargetURL, err := urlx.Parse(testCase.url)
				testutils.CheckNoError(t, err)

				actual, err := replacer.URLToSource(parsedTargetURL)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, actual.String())
			})
		})
	}
}

func TestReplacerToSourceMappingError(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"demo.localhost.com":           "https://demo.api.com",
		"custom.domain":                "http://customdomain.com",
		"custompost.localhost.com":     "https://customdomain.com:8080",
	})
	testutils.CheckNoError(t, err)

	t.Run("ToSource", func(t *testing.T) {
		tests := []struct {
			name          string
			requestURL    string
			url           string
			expectedError string
		}{
			{
				name:          "scheme in mapping and in url are not equal",
				requestURL:    "http://demo.localhost.com",
				url:           "http://demo.api.com",
				expectedError: "filed transform 'http://demo.api.com' to source url:  url scheme and mapping scheme is not equal",
			},
			{
				name:          "url is invalid",
				requestURL:    "http://demo.localhost.com",
				url:           "http://demo:.:a:pi.com",
				expectedError: "filed transform 'http://demo:.:a:pi.com' to source url:  filed parse url for replacing: parse \"http://demo:.:a:pi.com\": invalid port \":pi.com\" after host",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				parsedURL, err := urlx.Parse(testCase.requestURL)
				testutils.CheckNoError(t, err)

				replacer, err := factory.Make(parsedURL)
				testutils.CheckNoError(t, err)

				actual, err := replacer.StringToSource(testCase.url)

				assert.Empty(t, actual)
				assert.EqualError(t, err, testCase.expectedError)
			})
		}
	})

	t.Run("URLToSource", func(t *testing.T) {
		tests := []struct {
			name          string
			requestURL    string
			url           string
			expectedError string
		}{
			{
				name:          "scheme in mapping and in url are not equal",
				requestURL:    "http://demo.localhost.com",
				url:           "http://demo.api.com",
				expectedError: "filed transform 'http://demo.localhost.com' to source url:  url scheme and mapping scheme is not equal",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				parsedURL, err := urlx.Parse(testCase.requestURL)
				testutils.CheckNoError(t, err)

				replacer, err := factory.Make(parsedURL)
				testutils.CheckNoError(t, err)

				parsedTargetURL, err := urlx.Parse(testCase.requestURL)
				testutils.CheckNoError(t, err)

				actual, err := replacer.URLToSource(parsedTargetURL)

				assert.Empty(t, actual)
				assert.EqualError(t, err, testCase.expectedError)
			})
		}
	})
}

func TestReplacerToTargetMapping(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"demo.localhost.com":           "https://demo.api.com",
		"custom.domain":                "http://customdomain.com",
		"custompost.localhost.com":     "https://customdomain.com:8080",
		"*.star.com":                   "*.com",
	})
	testutils.CheckNoError(t, err)

	tests := []struct {
		name       string
		requestURL *url.URL
		url        string
		expected   string
	}{
		{
			name: "from https to https",
			requestURL: &url.URL{
				Host:   "premium.localhost.com",
				Scheme: "http",
			},
			url:      "http://premium.localhost.com/api/info",
			expected: "https://premium.api.com/api/info",
		},
		{
			name: "from http to https",
			requestURL: &url.URL{
				Host:   "base.localhost.com",
				Scheme: "https",
			},
			url:      "https://base.localhost.com/api/info",
			expected: "http://base.api.com/api/info",
		},
		{
			name: "from http to https with custom port",
			requestURL: &url.URL{
				Host:   "base.localhost.com:4200",
				Scheme: "https",
			},
			url:      "https://base.localhost.com:4200/api/info",
			expected: "http://base.api.com/api/info",
		},
		{
			name: "from https to http with custom port",
			requestURL: &url.URL{
				Host:   "premium.localhost.com:3000",
				Scheme: "http",
			},
			url:      "http://premium.localhost.com:3000/api/info",
			expected: "https://premium.api.com/api/info",
		},
		{
			name: "from https to http with custom port",
			requestURL: &url.URL{
				Host:   "custompost.localhost.com:3000",
				Scheme: "http",
			},
			url:      "http://custompost.localhost.com:3000/api/info",
			expected: "https://customdomain.com:8080/api/info",
		},
		{
			name: "* matcher",
			requestURL: &url.URL{
				Host:   "test.star.com:3000",
				Scheme: "http",
			},
			url:      "http://test.star.com:3000/api/info",
			expected: "http://test.com/api/info",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			replacer, err := factory.Make(testCase.requestURL)
			testutils.CheckNoError(t, err)

			actual, err := replacer.ToTarget(testCase.url)

			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestReplacerToTargetMappingErrors(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://premium.localhost.com": "https://premium.api.com",
		"https://base.localhost.com":   "http://base.api.com",
		"demo.localhost.com":           "https://demo.api.com",
		"custom.domain":                "http://customdomain.com",
		"custompost.localhost.com":     "https://customdomain.com:8080",
		"*.star.com":                   "*.com",
	})
	testutils.CheckNoError(t, err)

	t.Run("ToTarget", func(t *testing.T) {
		tests := []struct {
			name       string
			requestURL string
			url        string
		}{
			{
				name:       "scheme in mapping and in url are not equal",
				requestURL: "https://base.localhost.com",
				url:        "http://base.localhost.com/api/info",
			},
			{
				name:       "url is invalid",
				requestURL: "http://demo.localhost.com",
				url:        "http://demo.localh::$ost.com",
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				parsedURL, err := urlx.Parse(testCase.requestURL)
				testutils.CheckNoError(t, err)

				replacer, err := factory.Make(parsedURL)
				testutils.CheckNoError(t, err)

				actual, err := replacer.ToTarget(testCase.url)

				assert.Empty(t, actual)
				assert.Error(t, err)
			})
		}
	})
}

func TestReplacerSecure(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://localhost.com":  "https://premium.api.com",
		"https://localhost.net": "http://test.api.com",
		"localhost.us":          "http://api.us",
		"localhost.dev":         "https://api.dev",
		"http://localhost.biz":  "api.biz",
		"https://localhost.io":  "api.io",
		"demo.xyz":              "api.xyz",
	})
	testutils.CheckNoError(t, err)

	t.Run("IsSourceSecure", func(t *testing.T) {
		tests := []struct {
			name       string
			requestURL string
			expected   bool
		}{
			{
				name:       "should be false for http source mapping",
				requestURL: "http://localhost.com/api",
				expected:   false,
			},
			{
				name:       "should be true for https source mapping",
				requestURL: "https://localhost.net/api",
				expected:   true,
			},
			{
				name:       "should be true for not set source mapping called via https",
				requestURL: "https://localhost.us/api",
				expected:   true,
			},
			{
				name:       "should be true for not set source mapping called via http",
				requestURL: "http://localhost.us/api",
				expected:   false,
			},
			{
				name:       "should be true for not set source mapping called via https",
				requestURL: "https://localhost.dev/api",
				expected:   true,
			},
			{
				name:       "should be true for not set source mapping called via http",
				requestURL: "http://localhost.dev/api",
				expected:   false,
			},
			{
				name:       "should be false for http source mapping called via http",
				requestURL: "http://localhost.biz/api",
				expected:   false,
			},
			{
				name:       "should be true for https source mapping called via https",
				requestURL: "https://localhost.io/api",
				expected:   true,
			},
			{
				name:       "should be true for not set both mappings called via https",
				requestURL: "https://demo.xyz/api",
				expected:   true,
			},
			{
				name:       "should be true for not set both mappings called via http",
				requestURL: "http://demo.xyz/api",
				expected:   false,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				parsedURL, err := urlx.Parse(testCase.requestURL)
				testutils.CheckNoError(t, err)

				replacer, err := factory.Make(parsedURL)
				testutils.CheckNoError(t, err)

				actual := replacer.IsSourceSecure()

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})

	t.Run("IsTargetSecure", func(t *testing.T) {
		tests := []struct {
			name       string
			requestURL string
			url        string
			expected   bool
		}{
			{
				name:       "should be true for https target mapping",
				requestURL: "http://localhost.com/api",
				expected:   true,
			},
			{
				name:       "should be false for http target mapping",
				requestURL: "https://localhost.net/api",
				expected:   false,
			},
			{
				name:       "should be false for http taget mapping called via https",
				requestURL: "https://localhost.us/api",
				expected:   false,
			},
			{
				name:       "should be false for http taget mapping called via http",
				requestURL: "http://localhost.us/api",
				expected:   false,
			},
			{
				name:       "should be true for https taget mapping called via https",
				requestURL: "https://localhost.dev/api",
				expected:   true,
			},
			{
				name:       "should be true for https taget mapping called via http",
				requestURL: "http://localhost.dev/api",
				expected:   true,
			},
			{
				name:       "should be false for not set taget mapping called via http",
				requestURL: "http://localhost.biz/api",
				expected:   false,
			},
			{
				name:       "should be true for not set taget mapping called via https",
				requestURL: "https://localhost.io/api",
				expected:   true,
			},
			{
				name:       "should be true for not set both mappings called via https",
				requestURL: "https://demo.xyz/api",
				expected:   true,
			},
			{
				name:       "should be true for not set both mappings called via http",
				requestURL: "http://demo.xyz/api",
				expected:   false,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				parsedURL, err := urlx.Parse(testCase.requestURL)
				testutils.CheckNoError(t, err)

				replacer, err := factory.Make(parsedURL)
				testutils.CheckNoError(t, err)

				actual := replacer.IsTargetSecure()

				assert.Equal(t, testCase.expected, actual)
			})
		}
	})
}
