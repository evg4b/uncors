package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/urlparser"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUrlReplacerFactory(t *testing.T) {
	t.Run("should return error when", func(t *testing.T) {
		tests := []struct {
			name    string
			mapping config.Mappings
		}{
			{
				name:    "mappings is empty",
				mapping: make(config.Mappings, 0),
			},
			{
				name: "source url is incorrect",
				mapping: config.Mappings{
					{From: string(rune(0x7f)), To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "target url is incorrect ",
				mapping: config.Mappings{
					{From: hosts.Localhost.Host(), To: string(rune(0x7f))},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				assert.Panics(t, func() {
					urlreplacer.NewURLReplacerFactory(testCase.mapping)
				})
			})
		}
	})

	t.Run("should return replacers", func(t *testing.T) {
		actual := urlreplacer.NewURLReplacerFactory(config.Mappings{
			{From: hosts.Localhost.Host(), To: hosts.Github.HTTPS()},
		})

		assert.NotNil(t, actual)
	})
}

func TestFactoryMake(t *testing.T) {
	factory := urlreplacer.NewURLReplacerFactory(config.Mappings{
		{From: "http://server1.com", To: "https://mappedserver1.com"},
		{From: "https://server2.com", To: "https://mappedserver2.com"},
	})

	tests := []struct {
		name string
		url  string
		err  string
	}{
		{
			name: "mapped http server",
			url:  "http://server1.com",
		},
		{
			name: "mapped https server",
			url:  "https://server2.com",
		},
		{
			name: "unknown server",
			url:  "https://server3.com",
			err:  "mapping not found",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			parseURL, err := urlparser.Parse(testCase.url)
			testutils.CheckNoError(t, err)

			got, got1, err := factory.Make(parseURL)
			if len(testCase.err) > 0 {
				assert.Nil(t, got1)
				assert.Nil(t, got)
				require.EqualError(t, err, testCase.err)
			} else {
				assert.NotNil(t, got1)
				assert.NotNil(t, got)
				require.NoError(t, err)
			}
		})
	}
}
