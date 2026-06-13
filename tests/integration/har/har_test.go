//go:build integration

package har_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

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

	// Flush the HAR writer.
	env.Proxy.Shutdown()

	data, err := os.ReadFile(harPath)
	require.NoError(t, err)

	var parsed harFile
	require.NoError(t, json.Unmarshal(data, &parsed))

	require.Len(t, parsed.Log.Entries, 1)
	entry := parsed.Log.Entries[0]
	assert.Equal(t, http.MethodGet, entry.Request.Method)
	assert.Contains(t, entry.Request.URL, "/recorded/path")
	assert.Equal(t, http.StatusOK, entry.Response.Status)
}
