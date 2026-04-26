package urlreplacer

import (
	"testing"

	"github.com/evg4b/uncors/testing/hosts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCases = []struct {
	name            string
	url             string
	expectedRegexp  string
	expectedPattern string
}{
	{
		name:            hosts.Localhost.Host(),
		url:             hosts.Localhost.Host(),
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?localhost(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}localhost${path}",
	},
	{
		name:            "localhost with port",
		url:             "localhost:3000",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?localhost(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}localhost:3000${path}",
	},
	{
		name:            "single placeholder",
		url:             "{tenant}",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<tenant>.+)(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${tenant}${path}",
	},
	{
		name:            "single placeholder with port",
		url:             "{tenant}:3001",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<tenant>.+)(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${tenant}:3001${path}",
	},
	{
		name:            "single placeholder with url part",
		url:             "demo.{tenant}.com",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?demo\.(?P<tenant>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}demo.${tenant}.com${path}",
	},
	{
		name:            "single placeholder with url part and port",
		url:             "api.{tenant}.com:3001",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?api\.(?P<tenant>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}api.${tenant}.com:3001${path}",
	},
	{
		name:            "multiple placeholders with url part",
		url:             "{region}.host.{tenant}.com",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<region>.+)\.host\.(?P<tenant>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${region}.host.${tenant}.com${path}",
	},
	{
		name:            "multiple placeholders with url part and port",
		url:             "{region}.host.{tenant}.com:3001",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<region>.+)\.host\.(?P<tenant>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${region}.host.${tenant}.com:3001${path}",
	},
	{
		name:            "host with default http port",
		url:             "{tenant}.api.com:80",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<tenant>.+)\.api\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${tenant}.api.com:80${path}",
	},
	{
		name:            "host with default https port",
		url:             "{tenant}.api.com:443",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<tenant>.+)\.api\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${tenant}.api.com:443${path}",
	},
}

func TestWildCardToRegexp(t *testing.T) {
	t.Run("regexp", func(t *testing.T) {
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				compiledRegexp, err := wildCardToRegexp(testCase.url)

				require.NoError(t, err)
				assert.Equal(t, testCase.expectedRegexp, compiledRegexp.String())
			})
		}
	})

	t.Run("extracted keys", func(t *testing.T) {
		testCases := []struct {
			name     string
			url      string
			expected []string
		}{
			{name: "no placeholders", url: hosts.Localhost.Host(), expected: []string{}},
			{name: "single placeholder", url: "{tenant}", expected: []string{"tenant"}},
			{name: "two placeholders", url: "{region}.{tenant}.com", expected: []string{"region", "tenant"}},
			{name: "three placeholders", url: "api.{env}.{region}.{tenant}.com", expected: []string{"env", "region", "tenant"}},
		}
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.Equal(t, testCase.expected, extractKeys(testCase.url))
			})
		}
	})

	t.Run("error handling", func(t *testing.T) {
		_, err := wildCardToRegexp("localhost:")

		require.EqualError(t, err, `failed to build url glob: port "//localhost:": empty port`)
	})
}

func TestWildCardToReplacePattern(t *testing.T) {
	t.Run("pattern", func(t *testing.T) {
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				assert.Equal(t, testCase.expectedPattern, wildCardToReplacePattern(testCase.url))
			})
		}
	})
}
