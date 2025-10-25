package config_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
)

func TestOptionsClone(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		config config.OptionsHandling
	}{
		{
			name: "filled",
			config: config.OptionsHandling{
				Disabled: true,
				Headers: map[string]string{
					headers.ContentType:  "application/json",
					headers.CacheControl: "no-cache",
				},
				Code: 200,
			},
		},
		{
			name:   "empty",
			config: config.OptionsHandling{},
		},
		{
			name: "nil headers",
			config: config.OptionsHandling{
				Disabled: false,
				Code:     404,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			options := testCase.config

			clonedMock := options.Clone()

			t.Run("not same", func(t *testing.T) {
				assert.NotSame(t, &options, &clonedMock)
			})

			t.Run("equals values", func(t *testing.T) {
				assert.Equal(t, options, clonedMock)
			})

			t.Run("not same headers map", func(t *testing.T) {
				assert.NotSame(t, &options.Headers, &clonedMock.Headers)
			})

			t.Run("equals headers map", func(t *testing.T) {
				assert.Equal(t, options.Headers, clonedMock.Headers)
			})

			t.Run("equals code", func(t *testing.T) {
				assert.Equal(t, options.Code, clonedMock.Code)
			})

			t.Run(("rquals disabled"), func(t *testing.T) {
				assert.Equal(t, options.Disabled, clonedMock.Disabled)
			})
		})
	}
}
