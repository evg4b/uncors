package urlreplacer_test

import (
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

	t.Run("should return replacer", func(t *testing.T) {
		actual, err := urlreplacer.NewURLReplacerFactory(map[string]string{
			"localhost": "https://github.com",
		})

		assert.NotNil(t, actual)
		assert.NoError(t, err)
	})
}
