package urlt_test

import (
	"testing"

	"github.com/evg4b/uncors/pkg/urlt"
)

func TestParseHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		want    urlt.Host
		wantErr bool
	}{
		{
			name: "simple host",
			host: "example.com",
			want: urlt.Host{Hostname: "example.com"},
		},
		{
			name: "host with port",
			host: "example.com:8080",
			want: urlt.Host{Hostname: "example.com", Port: "8080"},
		},
		{
			name:    "invalid host",
			host:    "example.com:invalidport",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := urlt.ParseHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseHost() = %v, want %v", got, tt.want)
			}
		})
	}
}
