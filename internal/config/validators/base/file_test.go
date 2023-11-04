package base_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config/validators/base"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileValidator(t *testing.T) {
	const field = "test"

	t.Run("should not register error if file error", func(t *testing.T) {
		path := "/demo/file.go"
		errors := validate.Validate(&base.FileValidator{
			Field: field,
			Value: path,
			Fs: testutils.FsFromMap(t, map[string]string{
				path: "package validators",
			}),
		})

		assert.False(t, errors.HasAny())
	})

	fs := testutils.FsFromMap(t, map[string]string{
		"file.go": "package validators",
	})
	testutils.CheckNoError(t, fs.Mkdir("/demo", 0o755))

	tests := []struct {
		name  string
		path  string
		error string
	}{
		{
			name:  "file does not exist",
			path:  "file_does_not_exist.go",
			error: "test file does not exist",
		},
		{
			name:  "file is not accessible",
			path:  "/demo",
			error: "test is a directory",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errors := validate.Validate(&base.FileValidator{
				Field: field,
				Value: test.path,
				Fs:    fs,
			})

			require.EqualError(t, errors, test.error)
		})
	}
}
