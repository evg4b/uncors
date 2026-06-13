//go:build integration

package har_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type harFile struct {
	Log struct {
		Entries []struct {
			Request struct {
				Method string `json:"method"`
				URL    string `json:"url"`
			} `json:"request"`
			Response struct {
				Status int `json:"status"`
			} `json:"response"`
		} `json:"entries"`
	} `json:"log"`
}

func TestHARMiddleware(t *testing.T) {
	// The HAR writer uses the real filesystem and flushes on Close, so use a
	// real temp path and shut the proxy down before reading the file.
	harPath := filepath.Join(t.TempDir(), "out.har")

	env := harness.New(t, harness.WithMapping(func(mapping *config.Mapping) {
		mapping.HAR = config.HARConfig{File: harPath}
	}))

	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		env.Proxy.HTTPSURL("/recorded/path"),
		nil,
	)
	require.NoError(t, err)

	response, err := env.Client.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)

	// The writer enqueues the entry asynchronously after the response is sent and
	// rewrites the file (atomic temp+rename) on every entry, so poll until the
	// entry lands rather than racing the writer goroutine.
	var parsed harFile

	require.Eventually(t, func() bool {
		data, err := os.ReadFile(harPath)
		if err != nil {
			return false
		}

		parsed = harFile{}
		if json.Unmarshal(data, &parsed) != nil {
			return false
		}

		return len(parsed.Log.Entries) == 1
	}, 2*time.Second, 20*time.Millisecond)

	entry := parsed.Log.Entries[0]
	assert.Equal(t, http.MethodGet, entry.Request.Method)
	assert.Contains(t, entry.Request.URL, "/recorded/path")
	assert.Equal(t, http.StatusOK, entry.Response.Status)
}
