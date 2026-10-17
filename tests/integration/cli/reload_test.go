//go:build integration

package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	outputWait = 10 * time.Second
	outputTick = 20 * time.Millisecond
)

type output struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (o *output) Write(data []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.buffer.Write(data)
}

func (o *output) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()

	return o.buffer.String()
}

func TestReloadWithBrokenMappingKeepsServing(t *testing.T) {
	backend := integration.NewBackend(t, nil)
	port := testutils.GetFreePort(t)
	home := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "uncors.yaml")

	writeConfig := func(from string) {
		content := fmt.Sprintf("mappings:\n  - from: %s\n    to: %s\n", from, backend.URL())
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
	}

	writeConfig(fmt.Sprintf("http://localhost:%d", port))

	stdout := &output{}
	cmd := integration.UncorsCommand(t, []string{"-c", configPath, "--interactive=false"})

	cmd.Env = append(os.Environ(), "HOME="+home)
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	require.NoError(t, cmd.Start())

	exited := make(chan struct{})

	var exitErr error

	go func() {
		exitErr = cmd.Wait()

		close(exited)
	}()

	t.Cleanup(func() {
		_ = cmd.Process.Kill()

		<-exited
	})

	url := fmt.Sprintf("http://localhost:%d/", port)
	get := func() (int, error) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
		if err != nil {
			return 0, err
		}

		response, err := (&http.Client{Transport: &http.Transport{DisableKeepAlives: true}}).Do(request)
		if err != nil {
			return 0, err
		}

		defer response.Body.Close()

		_, err = io.Copy(io.Discard, response.Body)

		return response.StatusCode, err
	}

	require.Eventually(t, func() bool {
		code, err := get()

		return err == nil && code == http.StatusOK
	}, outputWait, outputTick, "uncors did not start: %s", stdout)

	writeConfig(fmt.Sprintf("http://{x}.{x}.localhost:%d", port))

	require.Eventually(t, func() bool {
		return strings.Contains(stdout.String(), "Failed to reload configuration, keeping the previous one")
	}, outputWait, outputTick, "no reload failure reported: %s", stdout)

	select {
	case <-exited:
		t.Fatalf("uncors exited after a broken reload: %v\n%s", exitErr, stdout)
	default:
	}

	code, err := get()
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, code)

	require.NoError(t, cmd.Process.Signal(syscall.SIGINT))

	ctx, cancel := context.WithTimeout(t.Context(), outputWait)
	defer cancel()

	select {
	case <-exited:
		require.NoError(t, exitErr, stdout.String())
	case <-ctx.Done():
		t.Fatalf("uncors did not stop on SIGINT: %s", stdout)
	}
}
