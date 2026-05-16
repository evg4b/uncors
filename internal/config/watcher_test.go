package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const watcherTimeout = 500 * time.Millisecond

// waitForCall blocks until fn fires or the timeout elapses, returning true on success.
func waitForCall(ch <-chan struct{}, timeout time.Duration) bool {
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestNewConfigWatcher(t *testing.T) {
	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := config.NewWatcher("/no/such/file.yaml", func() {})
		assert.Error(t, err)
	})

	t.Run("invokes onChange on file write", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		called := make(chan struct{}, 1)

		watcher, err := config.NewWatcher(configFile, func() {
			select {
			case called <- struct{}{}:
			default:
			}
		})
		require.NoError(t, err)

		defer watcher.Close()

		require.NoError(t, os.WriteFile(configFile, []byte("proxy: localhost:8080"), 0o600))
		assert.True(t, waitForCall(called, watcherTimeout), "onChange was not called after file write")
	})

	t.Run("does not invoke onChange after Close", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		called := make(chan struct{}, 1)

		watcher, err := config.NewWatcher(configFile, func() {
			select {
			case called <- struct{}{}:
			default:
			}
		})
		require.NoError(t, err)

		require.NoError(t, watcher.Close())

		require.NoError(t, os.WriteFile(configFile, []byte("proxy: changed"), 0o600))
		assert.False(t, waitForCall(called, 100*time.Millisecond), "onChange was called after Close")
	})

	t.Run("debounces rapid successive writes", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		callCount := 0
		called := make(chan struct{}, 10)

		watcher, err := config.NewWatcher(configFile, func() {
			callCount++

			called <- struct{}{}
		})
		require.NoError(t, err)

		defer watcher.Close()

		// Write multiple times in quick succession.
		for i := range 5 {
			require.NoError(t, os.WriteFile(configFile, []byte("proxy: change"), 0o600))

			_ = i
		}

		// Wait for the first (and hopefully only) callback.
		assert.True(t, waitForCall(called, watcherTimeout), "onChange was never called")

		// Give any extra calls a chance to arrive.
		time.Sleep(50 * time.Millisecond)

		assert.LessOrEqual(t, callCount, 3, "too many onChange calls for rapid writes (expected debouncing)")
	})

	t.Run("Close returns nil on first call", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte(""), 0o600))

		watcher, err := config.NewWatcher(configFile, func() {})
		require.NoError(t, err)

		assert.NoError(t, watcher.Close())
	})
}
