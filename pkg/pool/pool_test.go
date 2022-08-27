// nolint: goerr113, errorlint, forcetypeassert
package pool_test

import (
	"context"
	"errors"
	"testing"

	"github.com/evg4b/uncors/pkg/pool"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
)

func TestZeroGroup(t *testing.T) {
	t.Run("should return multierror", func(t *testing.T) {
		err1 := errors.New("pool_test: error 1")
		err2 := errors.New("pool_test: error 2")
		err3 := errors.New("pool_test: error 3")
		err4 := errors.New("pool_test: error 4")

		goPool, _ := pool.WithContext(context.Background())
		goPool.Go(func() error { return err1 })
		goPool.Go(func() error { return err2 })
		goPool.Go(func() error { return err3 })
		goPool.Go(func() error { return err4 })

		err := goPool.Wait()

		assert.Error(t, err)
		assert.IsType(t, &multierror.Error{}, err)
		assert.Equal(t, len(err.(*multierror.Error).Errors), 4)
		assert.ErrorIs(t, err, err1)
		assert.ErrorIs(t, err, err2)
		assert.ErrorIs(t, err, err3)
		assert.ErrorIs(t, err, err4)
	})

	t.Run("should return nill where no gourutines", func(t *testing.T) {
		goPool, _ := pool.WithContext(context.Background())

		err := goPool.Wait()

		assert.NoError(t, err)
	})

	t.Run("should return nill where no erorors", func(t *testing.T) {
		goPool, _ := pool.WithContext(context.Background())
		goPool.Go(func() error { return nil })
		goPool.Go(func() error { return nil })

		err := goPool.Wait()

		assert.NoError(t, err)
	})
}
