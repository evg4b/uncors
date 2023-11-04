package validators_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseValidator_IsValid(t *testing.T) {
	const file = "testdata/file.txt"

	fs := testutils.FsFromMap(t, map[string]string{
		file: "test",
	})

	t.Run("should not register errors if response is valid", func(t *testing.T) {
		errors := validate.Validate(&validators.ResponseValidator{
			Field: "test",
			Value: config.Response{
				Code:  200,
				File:  file,
				Delay: 3 * time.Second,
			},
			Fs: fs,
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should register errors for", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.Response
			error string
		}{
			{
				name: "code",
				value: config.Response{
					Code:  0,
					File:  file,
					Delay: 3 * time.Second,
				},
				error: "test.code code must be in range 100-599",
			},
			{
				name: "file",
				value: config.Response{
					Code:  200,
					File:  "testdata/unknown.txt",
					Delay: 3 * time.Second,
				},
				error: "test.file file does not exist",
			},
			{
				name: "delay",
				value: config.Response{
					Code:  200,
					File:  file,
					Delay: -1 * time.Second,
				},
				error: "test.delay must be greater than 0",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.ResponseValidator{
					Field: "test",
					Value: test.value,
					Fs:    fs,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}