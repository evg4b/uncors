package urlt_test

import (
	"testing"

	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/stretchr/testify/assert"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		want    *urlt.Host
		wantErr bool
	}{
		{
			name: "simple host",
			host: "example.com",
			want: &urlt.Host{Hostname: "example.com"},
		},
		{
			name: "sceme depended host",
			host: "//example.com",
			want: &urlt.Host{Hostname: "example.com"},
		},
		{
			name: "http host",
			host: "http://example.com",
			want: &urlt.Host{Scheme: "http", Hostname: "example.com"},
		},
		{
			name: "https host",
			host: "https://example.com",
			want: &urlt.Host{Scheme: "https", Hostname: "example.com"},
		},
		{
			name: "host with port",
			host: "example.com:8080",
			want: &urlt.Host{Hostname: "example.com", Port: "8080"},
		},
		{
			name: "sceme depended with port",
			host: "//example.com:8080",
			want: &urlt.Host{Hostname: "example.com", Port: "8080"},
		},
		{
			name: "http host with port",
			host: "http://example.com:8080",
			want: &urlt.Host{Scheme: "http", Hostname: "example.com", Port: "8080"},
		},
		{
			name: "https host with port",
			host: "https://example.com:8080",
			want: &urlt.Host{Scheme: "https", Hostname: "example.com", Port: "8080"},
		},
		{
			name:    "invalid host",
			host:    "example.com:invalidport",
			wantErr: true,
		},
		{
			name:    "url",
			host:    "http://example.com/demo",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := urlt.ParseHost(tt.host)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
