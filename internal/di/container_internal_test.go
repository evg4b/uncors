package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainerCloseError(t *testing.T) {
	t.Run("collects errors from closers", func(t *testing.T) {
		expectedErr := errors.New("close error")

		container := NewContainer()
		container.closers = append(container.closers, closerFunc(func() error {
			return expectedErr
		}))

		err := container.Close()

		require.Error(t, err)
		assert.ErrorIs(t, err, expectedErr)
	})

	t.Run("joins multiple closer errors", func(t *testing.T) {
		err1 := errors.New("first error")
		err2 := errors.New("second error")

		container := NewContainer()
		container.closers = append(container.closers,
			closerFunc(func() error { return err1 }),
			closerFunc(func() error { return err2 }),
		)

		err := container.Close()

		require.Error(t, err)
		assert.ErrorIs(t, err, err1)
		assert.ErrorIs(t, err, err2)
	})
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }
