package certs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHostMatcher(t *testing.T) {
	matcher := newHostMatcher([]string{"localhost", "API.local", "{tenant}.local.dev", "{app}.{env}.example.com", "::1"})

	for _, testCase := range []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"LOCALHOST", true},
		{"localhost.", true},
		{"api.local", true},
		{"other.local", false},
		{"acme.local.dev", true},
		{"a.b.local.dev", false},
		{"local.dev", false},
		{".local.dev", false},
		{"acme.local.dev.evil.com", false},
		{"web.prod.example.com", true},
		{"web.example.com", false},
		{"::1", true},
		{"[::1]", true},
		{"0:0:0:0:0:0:0:1", true},
		{"127.0.0.1", false},
		{"", false},
	} {
		t.Run(testCase.host, func(t *testing.T) {
			assert.Equal(t, testCase.want, matcher.match(testCase.host))
		})
	}

	t.Run("nil matcher allows nothing", func(t *testing.T) {
		var empty *hostMatcher

		assert.False(t, empty.match("localhost"))
	})
}
