package urlreplacer_test

import (
	"net/url"
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
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
					"locahost": string(rune(0x7f)),
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

	t.Run("shodul return replacer", func(t *testing.T) {
		actual, err := urlreplacer.NewURLReplacerFactory(map[string]string{
			"//localhost": "https://github.com",
		})

		assert.NotNil(t, actual)
		assert.NoError(t, err)
	})
}

func TestUrlReplacerFactoryMake(t *testing.T) {
	factory, _ := urlreplacer.NewURLReplacerFactory(map[string]string{
		"http://localhost": "https://github.com",
	})

	t.Run("shoduld return error when mapping not found", func(t *testing.T) {
		parsedURL, _ := url.Parse("http://unknow.com")

		actual, err := factory.Make(parsedURL)

		assert.Nil(t, actual)
		assert.EqualError(t, err, "mapping not found")
	})

	t.Run("shoduld return replacer wihout error", func(t *testing.T) {
		parsedURL, _ := url.Parse("http://localhost")

		actual, err := factory.Make(parsedURL)

		assert.NotNil(t, actual)
		assert.NoError(t, err)
	})

	t.Run("shoduld return error when requst sheme and mapping sheme not equal", func(t *testing.T) {
		parsedURL, err := url.Parse("https://localhost")
		if err != nil {
			t.Fatal(err)
		}

		actual, err := factory.Make(parsedURL)

		assert.Nil(t, actual)
		assert.EqualError(t, err, "mapping not found")
	})
}
