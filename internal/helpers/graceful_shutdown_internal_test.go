package helpers

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Env struct {
	wg       *sync.WaitGroup
	afterAll []func()
	mutex    sync.Mutex
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

func (e *Env) Lock() {
	e.mutex.Lock()
}

func (e *Env) Unlock() {
	e.mutex.Unlock()
}

func WithGoroutines(test func(t *testing.T, env *Env)) func(t *testing.T) {
	return func(t *testing.T) {
		env := Env{wg: &sync.WaitGroup{}, mutex: sync.Mutex{}}
		test(t, &env)
		env.wg.Wait()
		for _, f := range env.afterAll {
			f()
		}
	}
}

func TestGracefulShutdown(t *testing.T) {
	t.Run("shutdown when context is done", WithGoroutines(func(t *testing.T, env *Env) {
		ctx, cancel := context.WithCancel(context.Background())

		called := &atomic.Bool{}

		env.Go(func() {
			err := GracefulShutdown(ctx, func(_ context.Context) error {
				called.Store(true)

				return nil
			})
			require.NoError(t, err)
		})

		env.CheckAfterAll(func() {
			assert.False(t, called.Load())
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
			t.Run(testCase.name, WithGoroutines(func(t *testing.T, env *Env) {
				env.Lock()
				var systemSig chan<- os.Signal
				notifyFn = func(c chan<- os.Signal, _ ...os.Signal) {
					systemSig = c
					env.Unlock()
				}

				t.Cleanup(func() {
					notifyFn = signal.Notify
				})

				called := false
				env.Go(func() {
					err := GracefulShutdown(context.Background(), func(_ context.Context) error {
						called = true

						return nil
					})
					require.NoError(t, err)
				})

				<-time.After(50 * time.Millisecond)
				env.Lock()
				systemSig <- testCase.signal

				env.CheckAfterAll(func() {
					assert.True(t, called)
				})
			}))
		}
	})

	t.Run("apply additional ui fix for SIGINT signal", WithGoroutines(func(t *testing.T, env *Env) {
		var systemSig chan<- os.Signal
		env.Lock()
		notifyFn = func(c chan<- os.Signal, _ ...os.Signal) {
			systemSig = c
			env.Unlock()
		}
		called := false
		sigintFix = func() {
			called = true
		}

		t.Cleanup(func() {
			notifyFn = signal.Notify
		})

		env.Go(func() {
			err := GracefulShutdown(context.Background(), func(_ context.Context) error {
				return nil
			})
			require.NoError(t, err)
		})

		<-time.After(50 * time.Millisecond)
		env.Lock()
		systemSig <- syscall.SIGINT

		env.CheckAfterAll(func() {
			assert.True(t, called)
		})
		env.Unlock()
	}))
}
