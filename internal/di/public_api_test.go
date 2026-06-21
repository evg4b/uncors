package di_test

import (
	"testing"

	"github.com/evg4b/uncors/internal/di"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestContaienr(t *testing.T) {
	container := di.NewContainer()

	t.Run("fs", func(t *testing.T) {
		fs := container.Fs()

		assert.NotNil(t, fs)
		assert.IsType(t, &afero.MemMapFs{}, fs)
	})
}
