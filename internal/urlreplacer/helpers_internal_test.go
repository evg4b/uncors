package urlreplacer

import (
	"net/url"
	"testing"

	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestWildCardToRegexp(t *testing.T) {
	t.Run("regexp", func(t *testing.T) {
		testCases := []struct {
			name     string
			url      string
			expected string
		}{
			{
				name:     "localhost",
				url:      "localhost",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?localhost(?P<path>[\/?].*)?$`,
			},
			{
				name:     "localhost with port",
				url:      "localhost:3000",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?localhost:3000(?P<path>[\/?].*)?$`,
			},
			{
				name:     "single star",
				url:      "*",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)(?P<path>[\/?].*)?$`,
			},
			{
				name:     "single star with port",
				url:      "*:3001",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+):3001(?P<path>[\/?].*)?$`,
			},
			{
				name:     "single star with url part",
				url:      "demo.*.com",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?demo\.(?P<part1>.+)\.com(?P<path>[\/?].*)?$`,
			},
			{
				name:     "single star with url part and port",
				url:      "api.*.com:3001",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?api\.(?P<part1>.+)\.com:3001(?P<path>[\/?].*)?$`,
			},
			{
				name:     "multiple stars with url part",
				url:      "*.host.*.com",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.host\.(?P<part2>.+)\.com(?P<path>[\/?].*)?$`,
			},
			{
				name:     "multiple stars with url part and port",
				url:      "*.host.*.com:3001",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.host\.(?P<part2>.+)\.com:3001(?P<path>[\/?].*)?$`,
			},
			{
				name:     "host with default http port",
				url:      "*.api.com:80",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.api\.com(:80)?(?P<path>[\/?].*)?$`,
			},
			{
				name:     "host with default https port",
				url:      "*.api.com:443",
				expected: `^(?P<scheme>(http(s?):)?\/\/)?(?P<part1>.+)\.api\.com(:443)?(?P<path>[\/?].*)?$`,
			},
		}
		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				parsedPattern, err := urlx.Parse(testCase.url)
				testutils.CheckNoError(t, err)

				regexp, _, err := wildCardToRegexp(parsedPattern)

				assert.NoError(t, err)
				assert.Equal(t, testCase.expected, regexp.String())
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
				url:      "localhost",
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
			t.Run(testCase.name, func(t *testing.T) {
				_, _, err := wildCardToRegexp(&testCase.parsedPattern)

				assert.EqualError(t, err, testCase.expected)
			})
		}
	})
}
