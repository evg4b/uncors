package validators_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestMethodValidator(t *testing.T) {
	const field = "test-field"

	t.Run("should not register errors for", func(t *testing.T) {
		httpMethods := []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodConnect,
			http.MethodOptions,
			http.MethodTrace,
		}
		for _, method := range httpMethods {
			t.Run(fmt.Sprintf("http method %s", method), func(t *testing.T) {
				errors := validate.Validate(&validators.MethodValidator{
					Field: field,
					Value: method,
				})

				assert.False(t, errors.HasAny())
			})
		}

		t.Run("empty value when empty value is allowed", func(t *testing.T) {
			errors := validate.Validate(&validators.MethodValidator{
				Field:      field,
				Value:      "",
				AllowEmpty: true,
			})

			assert.False(t, errors.HasAny())
		})
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			Value string
			error string
		}{
			{
				name:  "empty value when empty value is not allowed",
				Value: "",
				error: "test-field must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
			{
				name:  "invalid value",
				Value: "invalid",
				error: "test-field must be one of GET, HEAD, POST, PUT, PATCH, DELETE, CONNECT, OPTIONS, TRACE",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.MethodValidator{
					Field: field,
					Value: test.Value,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
