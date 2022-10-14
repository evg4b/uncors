package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/pkg/urlx"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNewUrlReplacerFactory(t *testing.T) {
	t.Run("should return error when", func(t *testing.T) {
		tests := []struct {
			name    string
			mapping map[string]string
		}{
			{
				name:    "mappings is empty",
				mapping: make(map[string]string),
			},
			{
				name: "source url is incorrect",
				mapping: map[string]string{
					string(rune(0x7f)): "https://github.com",
				},
			},
			{
				name: "target url is incorrect ",
				mapping: map[string]string{
					"localhost": string(rune(0x7f)),
				},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				actual, err := urlreplacer.NewURLReplacerFactory(testCase.mapping)

				assert.Nil(t, actual)
				assert.Error(t, err)
			})
		}
	})

	t.Run("should return replacers", func(t *testing.T) {
		actual, err := urlreplacer.NewURLReplacerFactory(map[string]string{
			"localhost": "https://github.com",
		})

		assert.NotNil(t, actual)
		assert.NoError(t, err)
	})
}

func TestFactoryMake(t *testing.T) {
	factory, err := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://server1.com":  "https://mappedserver1.com",
		"https://server2.com": "https://mappedserver2.com",
	})
	testutils.CheckNoError(t, err)

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
			parseURL, err := urlx.Parse(testCase.url)
			testutils.CheckNoError(t, err)

			got, got1, err := factory.Make(parseURL)
			if len(testCase.err) > 0 {
				assert.Nil(t, got1)
				assert.Nil(t, got)
				assert.EqualError(t, err, testCase.err)
			} else {
				assert.NotNil(t, got1)
				assert.NotNil(t, got)
				assert.NoError(t, err)
			}
		})
	}
}
