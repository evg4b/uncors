package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/stretchr/testify/assert"
)

func TestNormaiseMappings(t *testing.T) {
	t.Run("custom port handling", func(t *testing.T) {
		httpPort, httpsPort := 3000, 3001
		testsCases := []struct {
			name     string
			mappings map[string]string
			expected map[string]string
		}{
			{
				name: "correctly set http and https ports",
				mappings: map[string]string{
					"localhost": "github.com",
				},
				expected: map[string]string{
					"http://localhost:3000":  "github.com",
					"https://localhost:3001": "github.com",
				},
			},
			{
				name: "correctly set http port",
				mappings: map[string]string{
					"http://localhost": "https://github.com",
				},
				expected: map[string]string{
					"http://localhost:3000": "https://github.com",
				},
			},
			{
				name: "corrctly set https port",
				mappings: map[string]string{
					"https://localhost": "https://github.com",
				},
				expected: map[string]string{
					"https://localhost:3001": "https://github.com",
				},
			},
			{
				name: "corrctly set mixed ports",
				mappings: map[string]string{
					"host1":         "https://github.com",
					"host2":         "http://github.com",
					"http://host3":  "http://api.github.com",
					"https://host4": "https://api.github.com",
				},
				expected: map[string]string{
					"http://host1:3000":  "https://github.com",
					"https://host1:3001": "https://github.com",
					"http://host2:3000":  "http://github.com",
					"https://host2:3001": "http://github.com",
					"http://host3:3000":  "http://api.github.com",
					"https://host4:3001": "https://api.github.com",
				},
			},
		}
		for _, testCase := range testsCases {
			t.Run(testCase.name, func(t *testing.T) {
				actual, err := urlreplacer.NormaiseMappings(testCase.mappings, httpPort, httpsPort)

				assert.NoError(t, err)
				assert.EqualValues(t, testCase.expected, actual)
			})
		}
	})

}
