package helpers

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

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
}

func (e *Env) CheckAfterAll(action func()) {
	e.afterAll = append(e.afterAll, action)
}

func WithGoroutines(test func(t *testing.T, env Env)) func(t *testing.T) {
	return func(t *testing.T) {
		env := Env{wg: &sync.WaitGroup{}}
		test(t, env)
		env.wg.Wait()
		for _, f := range env.afterAll {
			f()
		}
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

	t.Run("shutdown after system signal", func(t *testing.T) {
		tests := []struct {
			name   string
			signal os.Signal
		}{
			{
				name:   "SIGINT",
				signal: syscall.SIGINT,
			},
			{
				name:   "SIGTERM",
				signal: syscall.SIGTERM,
			},
			{
				name:   "SIGHUP",
				signal: syscall.SIGHUP,
			},
		}

		for _, testCase := range tests {
			t.Run(testCase.name, WithGoroutines(func(t *testing.T, env Env) {
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

				<-time.After(50 * time.Millisecond)
				systemSig <- testCase.signal

				env.CheckAfterAll(func() {
					assert.True(t, called)
				})
			}))
		}
	})

	t.Run("apply additional ui fix for SIGINT signal", WithGoroutines(func(t *testing.T, env Env) {
		var systemSig chan<- os.Signal
		notifyFn = func(c chan<- os.Signal, sig ...os.Signal) {
			systemSig = c
		}
		called := false
		sigintFix = func() {
			called = true
		}

		t.Cleanup(func() {
			notifyFn = signal.Notify
		})

		env.Go(func() {
			err := GracefulShutdown(context.Background(), func(ctx context.Context) error {
				return nil
			})
			assert.NoError(t, err)
		})

		<-time.After(50 * time.Millisecond)
		systemSig <- syscall.SIGINT

		env.CheckAfterAll(func() {
			assert.True(t, called)
		})
	}))
}
