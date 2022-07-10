package urlreplacer_test

import (
	"testing"

	"github.com/evg4b/uncors/inernal/urlreplacer"
	"github.com/stretchr/testify/assert"
)

func TestSimpleReplacer_ToTarget(t *testing.T) {
	r := urlreplacer.NewSimpleReplacer(map[string]string{
		"http://localhost:3000": "https://test.com",
		"//host1:8080":          "//api.test.com",
		"//host2:8080":          "http//api.test2.com",
	})

	t.Run("should return for not registred host", func(t *testing.T) {
		actual, err := r.ToTarget("https://not-registred-host:3000")

		assert.EqualError(t, err, "failed to find mapping for host 'not-registred-host:3000'")
		assert.Empty(t, actual)
	})

	t.Run("should return for no registred scheme", func(t *testing.T) {
		actual, err := r.ToTarget("https://localhost:3000")

		assert.EqualError(t, err, "failed to find mapping for scheme 'https' and host 'localhost:3000'")
		assert.Empty(t, actual)
	})

	t.Run("ToTarget", func(t *testing.T) {
		t.Run("when mappong has scheme", func(t *testing.T) {

		})

		tests := []struct {
			name        string
			url         string
			expected    string
			expectedErr string
		}{
			{
				name:     "correctly transform http url mapping without scheme",
				url:      "http://host2:8080/",
				expected: "http://api.test2.com/",
			},
			{
				name:     "correctly transform https url in mapping without scheme",
				url:      "https://host2:8080/",
				expected: "https://api.test2.com/",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual, err := r.ToTarget(tt.url)

				assert.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

}
