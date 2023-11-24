package helpers_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/helpers"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestPassedOrOsFs(t *testing.T) {
	type testStruct struct {
		fs afero.Fs
	}

	t.Run("should assign os fs if passed param is nil", func(t *testing.T) {
		data := testStruct{}
		helpers.PassedOrOsFs(&data.fs)

		assert.NotNil(t, data.fs)
	})

	t.Run("should skip if passed param is defined", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		data := testStruct{fs}

		helpers.PassedOrOsFs(&data.fs)

		assert.Equal(t, fs, data.fs)
	})
}
