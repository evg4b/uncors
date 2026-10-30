//go:build integration

package har_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/pkg/har"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/samber/lo"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const recording = "/recordings/session.har"

func headerValue(headers []har.NameValue, name string) (string, bool) {
	for _, header := range headers {
		if strings.EqualFold(header.Name, name) {
			return header.Value, true
		}
	}

	return "", false
}

func newRecordingEnv(t *testing.T, capture bool) *integration.Env {
	t.Helper()

	backend := integration.NewBackend(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("X-Upstream", "yes")

		if request.URL.Path == "/api/items" {
			writer.Header().Set("Set-Cookie", "sid=secret")
		}

		writer.WriteHeader(http.StatusCreated)

		body, err := io.ReadAll(request.Body)
		assert.NoError(t, err)

		_, err = io.WriteString(writer, `{"received":`+string(body)+`}`)
		assert.NoError(t, err)
	})

	return integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From:    hosts.Parse("https://har.local"),
			To:      backend.AsHost(),
			HAR:     config.HARConfig{File: recording, CaptureSecureHeaders: capture},
			Cache:   config.CacheGlobs{"/cached"},
			Statics: config.StaticDirectories{{Path: "/assets", Dir: "/www"}},
			Mocks: config.Mocks{{
				Matcher:  config.RequestMatcher{Path: "/mock"},
				Response: config.Response{Code: http.StatusOK, Raw: "mock"},
			}},
			Scripts: config.Scripts{{
				Matcher: config.RequestMatcher{Path: "/script"},
				Script:  `response:WriteHeader(200)`,
			}},
		}},
	}, integration.WithFile("/www/app.js", "console.log('hi')"))
}

func postJSON(t *testing.T, env *integration.Env) {
	t.Helper()

	req := integration.NewRequest(t, http.MethodPost, env.RawURL("har.local", "/api/items?b=2&a=1"))
	req.Body = io.NopCloser(strings.NewReader(`{"id":7}`))
	req.ContentLength = 8
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request", "visible")
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("X-Api-Key", "key")
	req.Header.Set("Cookie", "session=abc")

	result := env.Do(t, req)
	require.Equal(t, http.StatusCreated, result.Response.StatusCode)
	require.JSONEq(t, `{"received":{"id":7}}`, result.BodyString())
}

func readRecording(t *testing.T, env *integration.Env) har.HAR {
	t.Helper()

	env.Shutdown(t)

	data, err := afero.ReadFile(env.FS, recording)
	require.NoError(t, err)

	var parsed har.HAR
	require.NoError(t, json.Unmarshal(data, &parsed))

	return parsed
}

func TestHARRecordsTheExchange(t *testing.T) {
	env := newRecordingEnv(t, false)
	postJSON(t, env)

	parsed := readRecording(t, env)
	require.Len(t, parsed.Log.Entries, 1)

	recorded := parsed.Log.Entries[0]

	t.Run("records the request as the client sent it", func(t *testing.T) {
		assert.Equal(t, "uncors", parsed.Log.Creator.Name)
		assert.Equal(t, http.MethodPost, recorded.Request.Method)
		assert.Equal(t, env.RawURL("har.local", "/api/items?b=2&a=1"), recorded.Request.URL)
		assert.Equal(t, []har.NameValue{{Name: "b", Value: "2"}, {Name: "a", Value: "1"}}, recorded.Request.QueryString)

		value, ok := headerValue(recorded.Request.Headers, "X-Request")
		assert.True(t, ok)
		assert.Equal(t, "visible", value)

		require.NotNil(t, recorded.Request.PostData)
		assert.Equal(t, "application/json", recorded.Request.PostData.MimeType)
		assert.JSONEq(t, `{"id":7}`, recorded.Request.PostData.Text)
	})

	t.Run("records the response the client got", func(t *testing.T) {
		assert.Equal(t, http.StatusCreated, recorded.Response.Status)
		assert.Equal(t, "application/json", recorded.Response.Content.MimeType)
		assert.JSONEq(t, `{"received":{"id":7}}`, recorded.Response.Content.Text)

		value, ok := headerValue(recorded.Response.Headers, "X-Upstream")
		assert.True(t, ok)
		assert.Equal(t, "yes", value)
	})

	t.Run("leaves secure headers and cookies out", func(t *testing.T) {
		for _, name := range []string{"Authorization", "X-Api-Key", "Cookie"} {
			_, ok := headerValue(recorded.Request.Headers, name)
			assert.False(t, ok, name)
		}

		_, ok := headerValue(recorded.Response.Headers, "Set-Cookie")
		assert.False(t, ok)
		assert.Empty(t, recorded.Request.Cookies)
	})
}

func TestHARCapturesSecureHeaders(t *testing.T) {
	env := newRecordingEnv(t, true)
	postJSON(t, env)

	parsed := readRecording(t, env)
	require.Len(t, parsed.Log.Entries, 1)

	recorded := parsed.Log.Entries[0]

	for name, expected := range map[string]string{
		"Authorization": "Bearer token",
		"X-Api-Key":     "key",
		"Cookie":        "session=abc",
	} {
		value, ok := headerValue(recorded.Request.Headers, name)
		assert.True(t, ok, name)
		assert.Equal(t, expected, value, name)
	}

	value, ok := headerValue(recorded.Response.Headers, "Set-Cookie")
	assert.True(t, ok)
	assert.Equal(t, "sid=secret; Secure", value, "the header is recorded as the client got it")
}

func TestHARRecordsOnlyTheProxyPath(t *testing.T) {
	env := newRecordingEnv(t, false)

	send := func(method, path string) {
		req := integration.NewRequest(t, method, env.URL("har.local", path))
		req.Header.Set("Origin", "https://front.local")
		env.Do(t, req)
	}

	send(http.MethodGet, "/mock")
	send(http.MethodGet, "/script")
	send(http.MethodGet, "/assets/app.js")
	send(http.MethodGet, "/assets/missing.js")
	send(http.MethodOptions, "/preflight")
	send(http.MethodGet, "/cached")

	require.Eventually(t, func() bool {
		return !env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("har.local", "/cached"))).HasBackendRequest()
	}, 2*time.Second, 10*time.Millisecond)

	parsed := readRecording(t, env)

	paths := make([]string, 0, len(parsed.Log.Entries))

	for _, recorded := range parsed.Log.Entries {
		parsedURL, err := url.Parse(recorded.Request.URL)
		require.NoError(t, err)

		paths = append(paths, recorded.Request.Method+" "+parsedURL.Path)
	}

	assert.NotContains(t, paths, "GET /mock")
	assert.NotContains(t, paths, "GET /script")
	assert.NotContains(t, paths, "GET /assets/app.js")
	assert.Contains(t, paths, "GET /assets/missing.js")
	assert.Contains(t, paths, "OPTIONS /preflight")
	assert.GreaterOrEqual(t, lo.Count(paths, "GET /cached"), 2, "cache misses and hits are both recorded: %v", paths)
}
