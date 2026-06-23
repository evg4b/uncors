package di

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errCloseError  = errors.New("close error")
	errFirstError  = errors.New("first error")
	errSecondError = errors.New("second error")
)

func TestContainerCloseError(t *testing.T) {
	t.Run("collects errors from closers", func(t *testing.T) {
		container := NewContainer()
		container.closers = append(container.closers, closerFunc(func() error {
			return errCloseError
		}))

		err := container.Close()

		require.Error(t, err)
		require.ErrorIs(t, err, errCloseError)
	})

	t.Run("joins multiple closer errors", func(t *testing.T) {
		container := NewContainer()
		container.closers = append(container.closers,
			closerFunc(func() error { return errFirstError }),
			closerFunc(func() error { return errSecondError }),
		)

		err := container.Close()

		require.Error(t, err)
		require.ErrorIs(t, err, errFirstError)
		require.ErrorIs(t, err, errSecondError)
	})
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }
