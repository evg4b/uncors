//go:build integration

package har_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
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

	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://har.local"),
			To:   backend.AsHost(),
			HAR:  config.HARConfig{File: harPath},
		}},
	})

	result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("har.local", "/recorded/path")))
	require.NoError(t, result.Response.Body.Close())
	require.Equal(t, http.StatusOK, result.Response.StatusCode)

	// The writer enqueues entries asynchronously; poll until the entry lands.
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
