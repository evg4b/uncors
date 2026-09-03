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

func TestContainerClosesProcessLifetimeResources(t *testing.T) {
	t.Run("closes the request tracker it built", func(t *testing.T) {
		container := NewContainer()
		tracker := container.RequestTracker()

		require.NoError(t, container.Close())

		_, open := <-tracker.Events()
		require.False(t, open, "Close must close the tracker's event channel")
	})

	t.Run("is idempotent", func(t *testing.T) {
		container := NewContainer()
		container.RequestTracker()
		container.Server()

		require.NoError(t, container.Close())
		require.NoError(t, container.Close())
	})

	t.Run("closes nothing it did not build", func(t *testing.T) {
		require.NoError(t, NewContainer().Close())
	})
}
