package urlpattern_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/urlpattern"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		want    *urlpattern.Host
		wantErr bool
	}{
		// Basic hostname
		{
			name: "simple host",
			host: "example.com",
			want: &urlpattern.Host{Hostname: "example.com"},
		},
		{
			name: "scheme-relative host",
			host: "//example.com",
			want: &urlpattern.Host{Hostname: "example.com"},
		},
		{
			name: "http host",
			host: "http://example.com",
			want: &urlpattern.Host{Scheme: "http", Hostname: "example.com"},
		},
		{
			name: "https host",
			host: "https://example.com",
			want: &urlpattern.Host{Scheme: "https", Hostname: "example.com"},
		},
		{
			name: "host with port",
			host: "example.com:8080",
			want: &urlpattern.Host{Hostname: "example.com", Port: "8080"},
		},
		{
			name: "scheme-relative with port",
			host: "//example.com:8080",
			want: &urlpattern.Host{Hostname: "example.com", Port: "8080"},
		},
		{
			name: "http host with port",
			host: "http://example.com:8080",
			want: &urlpattern.Host{Scheme: "http", Hostname: "example.com", Port: "8080"},
		},
		{
			name: "https host with port",
			host: "https://example.com:8080",
			want: &urlpattern.Host{Scheme: "https", Hostname: "example.com", Port: "8080"},
		},

		// Subdomain
		{
			name: "subdomain",
			host: "sub.example.com",
			want: &urlpattern.Host{Hostname: "sub.example.com"},
		},
		{
			name: "deep subdomain",
			host: "a.b.c.example.com",
			want: &urlpattern.Host{Hostname: "a.b.c.example.com"},
		},
		{
			name: "subdomain with port",
			host: "sub.example.com:9000",
			want: &urlpattern.Host{Hostname: "sub.example.com", Port: "9000"},
		},

		// Localhost
		{
			name: "localhost",
			host: "localhost",
			want: &urlpattern.Host{Hostname: "localhost"},
		},
		{
			name: "localhost with port",
			host: "localhost:3000",
			want: &urlpattern.Host{Hostname: "localhost", Port: "3000"},
		},
		{
			name: "http localhost",
			host: "http://localhost",
			want: &urlpattern.Host{Scheme: "http", Hostname: "localhost"},
		},
		{
			name: "https localhost with port",
			host: "https://localhost:8443",
			want: &urlpattern.Host{Scheme: "https", Hostname: "localhost", Port: "8443"},
		},

		// IPv4
		{
			name: "ipv4 address",
			host: "192.168.0.1",
			want: &urlpattern.Host{Hostname: "192.168.0.1"},
		},
		{
			name: "ipv4 with port",
			host: "192.168.0.1:8080",
			want: &urlpattern.Host{Hostname: "192.168.0.1", Port: "8080"},
		},
		{
			name: "http ipv4",
			host: "http://192.168.0.1",
			want: &urlpattern.Host{Scheme: "http", Hostname: "192.168.0.1"},
		},
		{
			name: "https ipv4 with port",
			host: "https://192.168.0.1:443",
			want: &urlpattern.Host{Scheme: "https", Hostname: "192.168.0.1", Port: "443"},
		},

		// IPv6
		{
			name: "ipv6 address",
			host: "[::1]",
			want: &urlpattern.Host{Hostname: "::1"},
		},
		{
			name: "ipv6 with port",
			host: "[::1]:8080",
			want: &urlpattern.Host{Hostname: "::1", Port: "8080"},
		},
		{
			name: "http ipv6",
			host: "http://[::1]",
			want: &urlpattern.Host{Scheme: "http", Hostname: "::1"},
		},
		{
			name: "https ipv6 with port",
			host: "https://[::1]:8443",
			want: &urlpattern.Host{Scheme: "https", Hostname: "::1", Port: "8443"},
		},

		// Scheme normalization
		{
			name: "uppercase scheme",
			host: "HTTP://example.com",
			want: &urlpattern.Host{Scheme: "http", Hostname: "example.com"},
		},
		{
			name: "mixed case scheme",
			host: "HTTPs://example.com",
			want: &urlpattern.Host{Scheme: "https", Hostname: "example.com"},
		},

		// Non-http schemes
		{
			name: "ws scheme",
			host: "ws://example.com",
			want: &urlpattern.Host{Scheme: "ws", Hostname: "example.com"},
		},
		{
			name: "wss scheme with port",
			host: "wss://example.com:443",
			want: &urlpattern.Host{Scheme: "wss", Hostname: "example.com", Port: "443"},
		},
		{
			name: "ftp scheme",
			host: "ftp://files.example.com",
			want: &urlpattern.Host{Scheme: "ftp", Hostname: "files.example.com"},
		},

		// Port edge cases
		{
			name: "port zero",
			host: "example.com:0",
			want: &urlpattern.Host{Hostname: "example.com", Port: "0"},
		},
		{
			name: "port 65535",
			host: "example.com:65535",
			want: &urlpattern.Host{Hostname: "example.com", Port: "65535"},
		},
		{
			name: "trailing colon",
			host: "example.com:",
			want: &urlpattern.Host{Hostname: "example.com"},
		},

		// Errors — invalid input
		{
			name:    "empty string",
			host:    "",
			wantErr: true,
		},
		{
			name:    "invalid port",
			host:    "example.com:invalidport",
			wantErr: true,
		},
		{
			name:    "url with path",
			host:    "http://example.com/demo",
			wantErr: true,
		},
		{
			name:    "host with trailing slash",
			host:    "example.com/",
			wantErr: true,
		},
		{
			name:    "host with path",
			host:    "example.com/path",
			wantErr: true,
		},

		// Errors — invalid IPv6
		{
			name:    "ipv6 missing closing bracket",
			host:    "[::1",
			wantErr: true,
		},
		{
			name:    "bracket not at start",
			host:    "host[::1]",
			wantErr: true,
		},
		{
			name:    "ipv4 in brackets",
			host:    "[192.168.0.1]",
			wantErr: true,
		},
		{
			name:    "invalid ipv6 address",
			host:    "[invalid]",
			wantErr: true,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := urlpattern.Parse(testCase.host)

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestHost_String(t *testing.T) {
	tests := []struct {
		name string
		host urlpattern.Host
		want string
	}{
		{
			name: "hostname only",
			host: urlpattern.Host{Hostname: "example.com"},
			want: "example.com",
		},
		{
			name: "with http scheme",
			host: urlpattern.Host{Scheme: "http", Hostname: "example.com"},
			want: "http://example.com",
		},
		{
			name: "with port",
			host: urlpattern.Host{Hostname: "example.com", Port: "8080"},
			want: "example.com:8080",
		},
		{
			name: "scheme and port",
			host: urlpattern.Host{Scheme: "https", Hostname: "example.com", Port: "8443"},
			want: "https://example.com:8443",
		},
		{
			name: "ipv6 hostname only",
			host: urlpattern.Host{Hostname: "::1"},
			want: "[::1]",
		},
		{
			name: "ipv6 with scheme",
			host: urlpattern.Host{Scheme: "http", Hostname: "::1"},
			want: "http://[::1]",
		},
		{
			name: "ipv6 with port",
			host: urlpattern.Host{Hostname: "::1", Port: "8080"},
			want: "[::1]:8080",
		},
		{
			name: "ipv6 scheme and port",
			host: urlpattern.Host{Scheme: "https", Hostname: "::1", Port: "8443"},
			want: "https://[::1]:8443",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.host.String())
		})
	}
}

func TestHost_HostPort(t *testing.T) {
	tests := []struct {
		name string
		host urlpattern.Host
		want string
	}{
		{
			name: "hostname only",
			host: urlpattern.Host{Hostname: "example.com"},
			want: "example.com",
		},
		{
			name: "with port",
			host: urlpattern.Host{Hostname: "example.com", Port: "8080"},
			want: "example.com:8080",
		},
		{
			name: "scheme is ignored",
			host: urlpattern.Host{Scheme: "https", Hostname: "example.com", Port: "443"},
			want: "example.com:443",
		},
		{
			name: "ipv6 no port",
			host: urlpattern.Host{Hostname: "::1"},
			want: "[::1]",
		},
		{
			name: "ipv6 with port",
			host: urlpattern.Host{Hostname: "::1", Port: "8080"},
			want: "[::1]:8080",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.host.HostPort())
		})
	}
}

func TestHost_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		host urlpattern.Host
		want string
	}{
		{
			name: "hostname only",
			host: urlpattern.Host{Hostname: "example.com"},
			want: "example.com",
		},
		{
			name: "full url",
			host: urlpattern.Host{Scheme: "https", Hostname: "example.com", Port: "8443"},
			want: "https://example.com:8443",
		},
		{
			name: "ipv6",
			host: urlpattern.Host{Scheme: "http", Hostname: "::1", Port: "80"},
			want: "http://[::1]:80",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := testCase.host.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, testCase.want, string(got))
		})
	}
}

func TestHost_UnmarshalText(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
			want  urlpattern.Host
		}{
			{
				name:  "hostname only",
				input: "example.com",
				want:  urlpattern.Host{Hostname: "example.com"},
			},
			{
				name:  "full url",
				input: "https://example.com:8443",
				want:  urlpattern.Host{Scheme: "https", Hostname: "example.com", Port: "8443"},
			},
			{
				name:  "ipv6 with port",
				input: "http://[::1]:80",
				want:  urlpattern.Host{Scheme: "http", Hostname: "::1", Port: "80"},
			},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var h urlpattern.Host

				err := h.UnmarshalText([]byte(testCase.input))
				require.NoError(t, err)
				assert.Equal(t, testCase.want, h)
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := []struct {
			name  string
			input string
		}{
			{name: "empty string", input: ""},
			{name: "with path", input: "example.com/path"},
			{name: "invalid port", input: "example.com:notaport"},
		}
		for _, testCase := range tests {
			t.Run(testCase.name, func(t *testing.T) {
				var h urlpattern.Host

				err := h.UnmarshalText([]byte(testCase.input))
				assert.Error(t, err)
			})
		}
	})
}
