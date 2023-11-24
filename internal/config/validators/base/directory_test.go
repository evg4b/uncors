package base_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
)

func TestDirectoryValidator(t *testing.T) {
	const field = "test"
	const path = "/demo"

	fs := testutils.FsFromMap(t, map[string]string{
		"file.go": "package validators",
	})
	testutils.CheckNoError(t, fs.Mkdir(path, 0o755))

	t.Run("should not register error if file error", func(t *testing.T) {
		errors := validate.Validate(&base.DirectoryValidator{
			Field: field,
			Value: path,
			Fs:    fs,
		})

		assert.False(t, errors.HasAny())
	})

	t.Run("should register error for", func(t *testing.T) {
		tests := []struct {
			name  string
			path  string
			error string
		}{
			{
				name:  "empty path",
				path:  "",
				error: "test must not be empty",
			},
			{
				name:  "directory does not exist",
				path:  "directory_does_not_exist",
				error: "test directory does not exist",
			},
			{
				name:  "file instead of directory",
				path:  "file.go",
				error: "test is not a directory",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&base.DirectoryValidator{
					Field: field,
					Value: test.path,
					Fs:    fs,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
