package validators_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/config/validators"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gobuffalo/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticValidator(t *testing.T) {
	const assetsPath = "/assets"
	const staticPath = "/static"
	const indexFilePath = "/static/index.html"

	fs := testutils.FsFromMap(t, map[string]string{
		indexFilePath: indexFilePath,
	})

	t.Run("should not register errors if response is valid", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.StaticDirectory
		}{
			{
				name: "valid static directory with index",
				value: config.StaticDirectory{
					Path:  assetsPath,
					Dir:   staticPath,
					Index: "index.html",
				},
			},
			{
				name: "valid static directory without index",
				value: config.StaticDirectory{
					Path: assetsPath,
					Dir:  staticPath,
				},
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.StaticValidator{
					Field: "test",
					Value: test.value,
					Fs:    fs,
				})

				assert.False(t, errors.HasAny())
			})
		}
	})

	t.Run("should register errors if response is invalid", func(t *testing.T) {
		tests := []struct {
			name  string
			value config.StaticDirectory
			error string
		}{
			{
				name: "empty path",
				value: config.StaticDirectory{
					Path: "",
					Dir:  staticPath,
				},
				error: "test.path must not be empty",
			},
			{
				name: "empty directory",
				value: config.StaticDirectory{
					Path: assetsPath,
					Dir:  "",
				},
				error: "test.directory must not be empty",
			},
			{
				name: "empty directory",
				value: config.StaticDirectory{
					Path:  assetsPath,
					Dir:   staticPath,
					Index: "index.php",
				},
				error: "test.index /static/index.php does not exist",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				errors := validate.Validate(&validators.StaticValidator{
					Field: "test",
					Value: test.value,
					Fs:    fs,
				})

				require.EqualError(t, errors, test.error)
			})
		}
	})
}
