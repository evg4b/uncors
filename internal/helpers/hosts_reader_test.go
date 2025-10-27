package helpers_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestReadHostsFile(t *testing.T) {
	t.Run("should parse hosts file correctly", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		hostsContent := `# This is a comment
127.0.0.1 localhost
127.0.0.1 api.local app.local
::1       ipv6.local

# Another comment
192.168.1.1 external.com
`
		hostsPath := helpers.GetHostsFilePath()
		err := afero.WriteFile(fs, hostsPath, []byte(hostsContent), 0644)
		require.NoError(t, err)

		hosts, err := helpers.ReadHostsFile(fs)
		require.NoError(t, err)
		assert.NotNil(t, hosts)

		assert.Equal(t, "127.0.0.1", hosts["localhost"])
		assert.Equal(t, "127.0.0.1", hosts["api.local"])
		assert.Equal(t, "127.0.0.1", hosts["app.local"])
		assert.Equal(t, "::1", hosts["ipv6.local"])
		assert.Equal(t, "192.168.1.1", hosts["external.com"])
	})

	t.Run("should handle non-existent file", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		hosts, err := helpers.ReadHostsFile(fs)
		assert.Error(t, err)
		assert.Nil(t, hosts)
	})

	t.Run("should skip invalid lines", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		hostsContent := `127.0.0.1 localhost
invalid-line

127.0.0.2 test.local
`
		hostsPath := helpers.GetHostsFilePath()
		err := afero.WriteFile(fs, hostsPath, []byte(hostsContent), 0644)
		require.NoError(t, err)

		hosts, err := helpers.ReadHostsFile(fs)
		require.NoError(t, err)
		assert.NotNil(t, hosts)

		assert.Equal(t, "127.0.0.1", hosts["localhost"])
		assert.Equal(t, "127.0.0.2", hosts["test.local"])
		assert.Len(t, hosts, 2)
	})
}

func TestGetHostsFilePath(t *testing.T) {
	path := helpers.GetHostsFilePath()
	assert.NotEmpty(t, path)
	// Path should be either /etc/hosts (Unix) or C:\Windows\System32\drivers\etc\hosts (Windows)
	assert.Contains(t, []string{"/etc/hosts", `C:\Windows\System32\drivers\etc\hosts`}, path)
}
