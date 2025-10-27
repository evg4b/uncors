package helpers_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/stretchr/testify/assert"
)

func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv4 loopback variant", "127.0.0.2", true},
		{"IPv6 loopback", "::1", true},
		{"IPv6 loopback full", "0:0:0:0:0:0:0:1", true},
		{"Non-localhost IPv4", "192.168.1.1", false},
		{"Non-localhost IPv6", "2001:db8::1", false},
		{"Invalid IP", "invalid", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.IsLocalhost(tt.ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHostInHostsFile(t *testing.T) {
	hosts := map[string]string{
		"localhost":   "127.0.0.1",
		"api.local":   "127.0.0.1",
		"app.local":   "::1",
		"external.io": "192.168.1.100",
	}

	tests := []struct {
		name     string
		hostname string
		expected bool
	}{
		{"localhost with IPv4", "localhost", true},
		{"custom host with IPv4", "api.local", true},
		{"custom host with IPv6", "app.local", true},
		{"external host", "external.io", false},
		{"non-existent host", "unknown.host", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.IsHostInHostsFile(tt.hostname, hosts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHostsFilePath(t *testing.T) {
	path := helpers.GetHostsFilePath()
	assert.NotEmpty(t, path)
	// Path should be either /etc/hosts (Unix) or C:\Windows\System32\drivers\etc\hosts (Windows)
	assert.Contains(t, []string{"/etc/hosts", `C:\Windows\System32\drivers\etc\hosts`}, path)
}
