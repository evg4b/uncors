package helpers

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Env struct {
	wg       *sync.WaitGroup
	afterAll []func()
}

func (e *Env) Go(action func()) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		action()
	}()

	for _, f := range e.afterAll {
		f()
	}
}

func (e *Env) CheckAfterAll(action func()) {
	e.afterAll = append(e.afterAll, action)
}

func WithGoroutines(test func(t *testing.T, env Env)) func(t *testing.T) {
	return func(t *testing.T) {
		env := Env{wg: &sync.WaitGroup{}}
		test(t, env)
		env.wg.Wait()
	}
}

func TestGracefulShutdown(t *testing.T) {
	t.Run("shutdown when context is done", WithGoroutines(func(t *testing.T, env Env) {
		ctx, cancel := context.WithCancel(context.Background())

		called := false
		env.Go(func() {
			err := GracefulShutdown(ctx, func(ctx context.Context) error {
				called = true

				return nil
			})
			assert.NoError(t, err)
		})

		env.CheckAfterAll(func() {
			assert.True(t, called)
		})

		cancel()
	}))
}
