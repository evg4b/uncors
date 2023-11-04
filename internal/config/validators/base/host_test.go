package base_test

import (
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config/validators/base"

	"github.com/stretchr/testify/require"

	"github.com/evg4b/uncors/testing/hosts"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestHostValidator(t *testing.T) {
	const field = "field"

	t.Run("should not register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
		}{
			{
				name:  "valid host",
				value: hosts.Localhost.Host(),
			},
			{
				name:  "valid host with http scheme",
				value: hosts.Github.HTTP(),
			},
			{
				name:  "valid host with https scheme",
				value: hosts.Github.HTTPS(),
			},
			{
				name:  "valid ip address",
				value: hosts.Loopback.Host(),
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&base.HostValidator{
					Field: field,
					Value: test.value,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("should register errors for invalid hosts", func(t *testing.T) {
		tests := []struct {
			name  string
			value string
			error string
		}{
			{
				name:  "empty host",
				value: "",
				error: "field must not be empty",
			},
			{
				name:  "too long host",
				value: strings.Repeat("a", 256),
				error: "field must not be longer than 255 characters, but got 256",
			},
			{
				name:  "host with path",
				value: "example.com/path",
				error: "field must not contain path",
			},
			{
				name:  "host with query",
				value: "example.com?query=1",
				error: "field must not contain query",
			},
			{
				name:  "host with unsupported scheme",
				value: hosts.Localhost.Scheme("ftp"),
				error: "field scheme must be http or https",
			},
			{
				name:  "host is not valid",
				value: "loca:::lhost",
				error: "field is not valid host",
			},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&base.HostValidator{
					Field: field,
					Value: test.value,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
