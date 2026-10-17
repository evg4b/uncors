package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlt"
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
					{From: urlt.Host{Hostname: string(rune(0x7f))}, To: hosts.Github.HTTPS()},
				},
			},
			{
				name: "target url is incorrect ",
				mapping: config.Mappings{
					{From: hosts.Localhost.Host(), To: urlt.Host{Hostname: string(rune(0x7f))}},
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				factory, err := urlreplacer.NewURLReplacerFactory(testCase.mapping)
				require.Error(t, err)
				assert.Nil(t, factory)
			})
		}
	})

	t.Run("should return replacers", func(t *testing.T) {
		actual, err := urlreplacer.NewURLReplacerFactory(config.Mappings{
			{From: hosts.Localhost.Host(), To: hosts.Github.HTTPS()},
		})
		require.NoError(t, err)

		assert.NotNil(t, actual)
	})
}

func TestFactoryMake(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(config.Mappings{
		{From: hosts.Parse("http://server1.com"), To: hosts.Parse("https://mappedserver1.com")},
		{From: hosts.Parse("https://server2.com"), To: hosts.Parse("https://mappedserver2.com")},
	})
	require.NoError(t, err)

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
			parseURL, err := urlt.Parse(testCase.url)
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

func TestFactoryMakeIgnoresHostSuffixInPath(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(config.Mappings{
		{From: hosts.Parse("http://{tenant}.local.dev"), To: hosts.Parse("https://{tenant}.api.com")},
		{From: hosts.Parse("http://other.host"), To: hosts.Parse("https://other.com")},
	})
	require.NoError(t, err)

	requestURL, err := urlt.Parse("http://other.host/files/x.local.dev?q=.local.dev")
	require.NoError(t, err)

	target, _, err := factory.Make(requestURL)
	require.NoError(t, err)

	actual, err := target.Replace(urlt.URL_String(requestURL))
	require.NoError(t, err)
	assert.Equal(t, "https://other.com/files/x.local.dev?q=.local.dev", actual)
}
