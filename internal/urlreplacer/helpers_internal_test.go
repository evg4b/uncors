package urlreplacer

import (
	"net/url"
	"testing"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
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
		name:            "single star",
		url:             "*",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${part1}${path}",
	},
	{
		name:            "single star with port",
		url:             "*:3001",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${part1}:3001${path}",
	},
	{
		name:            "single star with url part",
		url:             "demo.*.com",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?demo\.(?P<part1>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}demo.${part1}.com${path}",
	},
	{
		name:            "single star with url part and port",
		url:             "api.*.com:3001",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?api\.(?P<part1>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}api.${part1}.com:3001${path}",
	},
	{
		name:            "multiple stars with url part",
		url:             "*.host.*.com",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.host\.(?P<part2>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${part1}.host.${part2}.com${path}",
	},
	{
		name:            "multiple stars with url part and port",
		url:             "*.host.*.com:3001",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.host\.(?P<part2>.+)\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${part1}.host.${part2}.com:3001${path}",
	},
	{
		name:            "host with default http port",
		url:             "*.api.com:80",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.api\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${part1}.api.com:80${path}",
	},
	{
		name:            "host with default https port",
		url:             "*.api.com:443",
		expectedRegexp:  `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.api\.com(:\d+)?(?P<path>[\/?].*)?$`,
		expectedPattern: "${scheme}${part1}.api.com:443${path}",
	},
}

func TestWildCardToRegexp(t *testing.T) {
	t.Run("regexp", func(t *testing.T) {
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				parsedPattern, err := urlx.Parse(testCase.url)
				testutils.CheckNoError(t, err)

				regexp, _, err := wildCardToRegexp(parsedPattern)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expectedRegexp, regexp.String())
			})
		}
	})

	t.Run("wildcard count", func(t *testing.T) {
		testCases := []struct {
			name     string
			url      string
			expected int
		}{
			{
				name:     "no wildcards",
				url:      hosts.Localhost.Host(),
				expected: 0,
			},
			{
				name:     "single star",
				url:      "*",
				expected: 1,
			},
			{
				name:     "two stars",
				url:      "*.*.com",
				expected: 2,
			},
			{
				name:     "three stars",
				url:      "api.*.*.*.com",
				expected: 3,
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				parsedPattern, err := urlx.Parse(testCase.url)
				testutils.CheckNoError(t, err)

				_, count, err := wildCardToRegexp(parsedPattern)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, count)
			})
		}
	})

	t.Run("error handling", func(t *testing.T) {
		testCases := []struct {
			name          string
			parsedPattern url.URL
			expected      string
		}{
			{
				name:          "incorrect port",
				parsedPattern: url.URL{Host: "localhost:"},
				expected:      `filed to build url glob: port "//localhost:": empty port`,
			},
		}
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				_, _, err := wildCardToRegexp(&testCase.parsedPattern)

				assert.EqualError(t, err, testCase.expected)
			})
		}
	})
}

func TestWildCardToReplacePattern(t *testing.T) {
	t.Run("pattern", func(t *testing.T) {
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				parsedPattern, err := urlx.Parse(testCase.url)
				testutils.CheckNoError(t, err)

				pattern, _ := wildCardToReplacePattern(parsedPattern)

				assert.Equal(t, testCase.expectedPattern, pattern)
			})
		}
	})

	t.Run("wildcard count", func(t *testing.T) {
		testCases := []struct {
			name     string
			url      string
			expected int
		}{
			{
				name:     "no wildcards",
				url:      hosts.Localhost.Host(),
				expected: 0,
			},
			{
				name:     "single star",
				url:      "*",
				expected: 1,
			},
			{
				name:     "two stars",
				url:      "*.*.com",
				expected: 2,
			},
			{
				name:     "three stars",
				url:      "api.*.*.*.com",
				expected: 3,
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				parsedPattern, err := urlx.Parse(testCase.url)
				testutils.CheckNoError(t, err)

				_, count := wildCardToReplacePattern(parsedPattern)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, count)
			})
		}
	})
}
