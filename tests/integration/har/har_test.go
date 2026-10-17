//go:build integration

package har_test

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/spf13/afero"
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
		data, err := afero.ReadFile(env.FS, harPath)
		if err != nil {
			return false
		}

		parsed = harFile{}
		if json.Unmarshal(data, &parsed) != nil {
			return false
		}

		return len(parsed.Log.Entries) == 1
	}, 5*time.Second, 20*time.Millisecond)

	entry := parsed.Log.Entries[0]
	assert.Equal(t, http.MethodGet, entry.Request.Method)
	assert.Contains(t, entry.Request.URL, "/recorded/path")
	assert.Equal(t, http.StatusOK, entry.Response.Status)
}

func TestHARMiddlewareKeepsContentLength(t *testing.T) {
	type received struct {
		contentLength    int64
		transferEncoding []string
		body             string
	}

	seen := make(chan received, 1)

	backend := integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		assert.NoError(t, err)

		seen <- received{
			contentLength:    request.ContentLength,
			transferEncoding: request.TransferEncoding,
			body:             string(body),
		}

		writer.WriteHeader(http.StatusOK)
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://har.local"),
			To:   backend.AsHost(),
			HAR:  config.HARConfig{File: filepath.Join(t.TempDir(), "out.har")},
		}},
	})

	req := integration.NewRequest(t, http.MethodPost, env.URL("har.local", "/upload"))
	req.Body = io.NopCloser(strings.NewReader("hello"))
	req.ContentLength = 5

	result := env.Do(t, req)
	require.Equal(t, http.StatusOK, result.Response.StatusCode)

	got := <-seen
	assert.Equal(t, int64(5), got.contentLength)
	assert.Empty(t, got.transferEncoding)
	assert.Equal(t, "hello", got.body)
}
