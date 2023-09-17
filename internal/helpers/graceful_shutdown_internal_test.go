package helpers

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	t.Run("shutdown after system signal", WithGoroutines(func(t *testing.T, env Env) {
		var systemSig chan<- os.Signal

		notifyFn = func(c chan<- os.Signal, sig ...os.Signal) {
			systemSig = c
		}

		t.Cleanup(func() {
			notifyFn = signal.Notify
		})

		called := false
		env.Go(func() {
			err := GracefulShutdown(context.Background(), func(ctx context.Context) error {
				called = true

				return nil
			})
			assert.NoError(t, err)
		})

		env.CheckAfterAll(func() {
			assert.True(t, called)
		})

		systemSig <- syscall.SIGTERM
	}))
}
