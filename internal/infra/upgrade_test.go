package infra_test

import (
	"net/http"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/stretchr/testify/assert"
)

func TestIsUpgrade(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		expected bool
	}{
		{
			name:     "websocket handshake",
			header:   http.Header{"Connection": {"Upgrade"}, "Upgrade": {"websocket"}},
			expected: true,
		},
		{
			name:     "upgrade among other connection tokens",
			header:   http.Header{"Connection": {"keep-alive, upgrade"}, "Upgrade": {"h2c"}},
			expected: true,
		},
		{
			name:     "upgrade in a repeated connection header",
			header:   http.Header{"Connection": {"keep-alive", " UPGRADE "}, "Upgrade": {"websocket"}},
			expected: true,
		},
		{
			name:     "upgrade header without the connection token",
			header:   http.Header{"Connection": {"keep-alive"}, "Upgrade": {"websocket"}},
			expected: false,
		},
		{
			name:     "connection token without the upgrade header",
			header:   http.Header{"Connection": {"Upgrade"}},
			expected: false,
		},
		{
			name:     "token that only contains upgrade",
			header:   http.Header{"Connection": {"upgrade-insecure"}, "Upgrade": {"websocket"}},
			expected: false,
		},
		{
			name:     "regular request",
			header:   http.Header{},
			expected: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, infra.IsUpgrade(testCase.header))
		})
	}
}
