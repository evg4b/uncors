//go:build integration

package har_test

import (
	"encoding/json"
	"net/http"
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
	// The HAR writer records through the injected filesystem, which the harness
	// keeps in memory.
	const harPath = "/har/out.har"

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
		data, err := afero.ReadFile(env.Fs, harPath)
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

// "records every request that passes through a mapping" used to mean "everything
// the mapping proxied": mocks and scripts bypassed the recorder entirely, so a
// developer comparing a HAR against what their frontend sent would conclude the
// requests were never made.
func TestHARRecordsLocallyProducedResponses(t *testing.T) {
	const harPath = "/har/local.har"

	backend := integration.NewBackend(t, nil)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://local.har.local"),
			To:   backend.AsHost(),
			HAR:  config.HARConfig{File: harPath},
			Mocks: config.Mocks{
				{
					Matcher:  config.RequestMatcher{Path: "/api/mocked"},
					Response: config.Response{Code: http.StatusOK, Raw: `{"mocked": true}`},
				},
			},
			Scripts: config.Scripts{
				{
					Matcher: config.RequestMatcher{Path: "/api/scripted"},
					Script:  `response:WriteHeader(200) response:WriteString("scripted")`,
				},
			},
		}},
	})

	for _, path := range []string{"/api/mocked", "/api/scripted", "/api/proxied"} {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("local.har.local", path)))
		require.NoError(t, result.Response.Body.Close())
		require.Equal(t, http.StatusOK, result.Response.StatusCode, path)
	}

	require.Eventually(t, func() bool {
		data, err := afero.ReadFile(env.Fs, harPath)
		if err != nil {
			return false
		}

		var parsed harFile
		if json.Unmarshal(data, &parsed) != nil {
			return false
		}

		recorded := map[string]bool{}
		for _, entry := range parsed.Log.Entries {
			recorded[entry.Request.URL] = true
		}

		for _, path := range []string{"/api/mocked", "/api/scripted", "/api/proxied"} {
			if !containsPath(recorded, path) {
				return false
			}
		}

		return true
	}, 2*time.Second, 20*time.Millisecond, "every response the mapping produced must be recorded")
}

func containsPath(recorded map[string]bool, path string) bool {
	for url := range recorded {
		if strings.HasSuffix(url, path) {
			return true
		}
	}

	return false
}
