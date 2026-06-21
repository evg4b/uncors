package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/testutils"
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
	t.Run("returns error for non-existent file on Watch", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		watcher := config.NewWatcher("/no/such/file.yaml")
		err := watcher.Watch(ctx, func() {})
		assert.Error(t, err)
	})

	t.Run("invokes onChange on file write", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		called := make(chan struct{}, 1)

		ctx := t.Context()

		watcher := config.NewWatcher(configFile)
		err := watcher.Watch(ctx, func() {
			select {
			case called <- struct{}{}:
			default:
			}
		})
		require.NoError(t, err)

		defer testutils.Close(t, watcher)

		require.NoError(t, os.WriteFile(configFile, []byte("proxy: localhost:8080"), 0o600))
		assert.True(t, waitForCall(called, watcherTimeout), "onChange was not called after file write")
	})

	t.Run("does not invoke onChange after Close", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		called := make(chan struct{}, 1)

		ctx := t.Context()

		watcher := config.NewWatcher(configFile)
		err := watcher.Watch(ctx, func() {
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

		ctx := t.Context()

		watcher := config.NewWatcher(configFile)
		err := watcher.Watch(ctx, func() {
			callCount++

			called <- struct{}{}
		})
		require.NoError(t, err)

		defer testutils.Close(t, watcher)

		// Write multiple times in quick succession.
		for range 5 {
			require.NoError(t, os.WriteFile(configFile, []byte("proxy: change"), 0o600))
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

		ctx := t.Context()

		watcher := config.NewWatcher(configFile)
		err := watcher.Watch(ctx, func() {})
		require.NoError(t, err)

		assert.NoError(t, watcher.Close())
	})

	t.Run("invokes onChange on atomic save (write-temp + rename)", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		called := make(chan struct{}, 4)

		ctx := t.Context()

		watcher := config.NewWatcher(configFile)
		err := watcher.Watch(ctx, func() {
			select {
			case called <- struct{}{}:
			default:
			}
		})
		require.NoError(t, err)

		defer testutils.Close(t, watcher)

		// Simulate the atomic-save pattern used by vim/IntelliJ/VS Code: write a
		// sibling temp file, then rename it over the target. This replaces the
		// target's inode, which a file-level watch would miss. Repeat to confirm
		// the watch keeps firing across multiple saves.
		atomicSave := func(content string) {
			tmp := filepath.Join(tmpDir, "config.yaml.tmp")
			require.NoError(t, os.WriteFile(tmp, []byte(content), 0o600))
			require.NoError(t, os.Rename(tmp, configFile))
		}

		atomicSave("proxy: localhost:8080")
		assert.True(t, waitForCall(called, watcherTimeout), "onChange not called after first atomic save")

		atomicSave("proxy: localhost:9090")
		assert.True(t, waitForCall(called, watcherTimeout), "onChange not called after second atomic save")
	})

	t.Run("stops watching when context is cancelled", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: \"\""), 0o600))

		called := make(chan struct{}, 1)
		ctx, cancel := context.WithCancel(context.Background())

		watcher := config.NewWatcher(configFile)
		err := watcher.Watch(ctx, func() {
			select {
			case called <- struct{}{}:
			default:
			}
		})
		require.NoError(t, err)

		defer testutils.Close(t, watcher)

		// Cancel the context
		cancel()
		time.Sleep(50 * time.Millisecond)

		// Write to file after context is cancelled
		require.NoError(t, os.WriteFile(configFile, []byte("proxy: changed"), 0o600))
		assert.False(t, waitForCall(called, 100*time.Millisecond), "onChange was called after context cancelled")
	})
}
