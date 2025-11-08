package base_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config/validators/base"

	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathValidator(t *testing.T) {
	const field = "field"

	t.Run("absolute path", func(t *testing.T) {
		t.Run("should not register errors for", func(t *testing.T) {
			tests := []struct {
				name  string
				value string
			}{
				{
					name:  "root path",
					value: "/",
				},
				{
					name:  "valid path",
					value: "/api/info",
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					errors := validate.Validate(&base.PathValidator{
						Field: field,
						Value: test.value,
					})

					assert.False(t, errors.HasAny())
				})
			}
		})

		t.Run("should register errors for invalid hosts", func(t *testing.T) {
			t.Skip("This test is not implemented yet.")

			tests := []struct {
				name  string
				value string
				error string
			}{
				{
					name:  "empty path",
					value: "",
					error: "field must not be empty",
				},
				{
					name:  "path without /",
					value: "api/info",
					error: "field must be absolute and start with /",
				},
				{
					name:  "path with query",
					value: "/api/info?demo=1",
					error: "field must not contain a query",
				},
				{
					name:  "path with host",
					value: "demo.com/api/info?demo=1",
					error: "field must be absolute and start with /",
				},
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					errors := validate.Validate(&base.PathValidator{
						Field: field,
						Value: test.value,
					})

					require.EqualError(t, errors, test.error)
				})
			}
		})
	})
}
