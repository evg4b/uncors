package har_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddleware_Wrap(t *testing.T) {
	makeWriter := func(rec *httptest.ResponseRecorder) contracts.ResponseWriter {
		return contracts.WrapResponseWriter(rec)
	}

	makeRequest := func(method, rawURL string) *http.Request {
		req, err := http.NewRequestWithContext(context.Background(), method, rawURL, nil)
		require.NoError(t, err)

		return req
	}

	newMiddleware := func(t *testing.T, opts ...har.MiddlewareOption) (*har.Middleware, *har.Writer, string) {
		t.Helper()

		path := filepath.Join(t.TempDir(), "test.har")
		harWriter := har.NewWriter(path)
		opts = append([]har.MiddlewareOption{har.WithWriter(harWriter)}, opts...)
		mdlw := har.NewMiddleware(opts...)

		return mdlw, harWriter, path
	}

	readHAR := func(t *testing.T, path string) har.HAR {
		t.Helper()

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var archive har.HAR
		require.NoError(t, json.Unmarshal(data, &archive))

		return archive
	}

	t.Run("passes request to next handler", func(t *testing.T) {
		mdlw, harWriter, _ := newMiddleware(t)

		defer harWriter.Close() //nolint:errcheck

		called := false
		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			called = true

			rw.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		mdlw.Wrap(next).ServeHTTP(makeWriter(rec), makeRequest(http.MethodGet, "http://example.com/path"))

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("records response body correctly", func(t *testing.T) {
		mdlw, harWriter, _ := newMiddleware(t)

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			rw.Header().Set("Content-Type", "text/plain")
			rw.WriteHeader(http.StatusOK)
			fmt.Fprint(rw, "hello")
		})

		rec := httptest.NewRecorder()
		mdlw.Wrap(next).ServeHTTP(makeWriter(rec), makeRequest(http.MethodGet, "http://example.com/"))

		require.NoError(t, harWriter.Close())

		assert.Equal(t, "hello", rec.Body.String())
	})

	t.Run("records request with query string", func(t *testing.T) {
		mdlw, harWriter, _ := newMiddleware(t)

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})

		req := makeRequest(http.MethodGet, "http://example.com/search?q=foo&page=2")
		rec := httptest.NewRecorder()
		mdlw.Wrap(next).ServeHTTP(makeWriter(rec), req)

		require.NoError(t, harWriter.Close())

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("restores request body for downstream handlers", func(t *testing.T) {
		mdlw, harWriter, _ := newMiddleware(t)

		defer harWriter.Close() //nolint:errcheck

		body := "request-payload"

		var received string

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, r *contracts.Request) {
			b, _ := io.ReadAll(r.Body)
			received = string(b)

			rw.WriteHeader(http.StatusOK)
		})

		req, err := http.NewRequestWithContext(
			context.Background(), http.MethodPost, "http://example.com/api", strings.NewReader(body),
		)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		mdlw.Wrap(next).ServeHTTP(makeWriter(rec), req)

		assert.Equal(t, body, received)
	})

	t.Run("secure headers not captured by default", func(t *testing.T) {
		mdlw, harWriter, path := newMiddleware(t) // captureSecureHeaders defaults to false

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			http.SetCookie(rw, &http.Cookie{Name: "session", Value: "abc"})
			rw.Header().Set("Www-Authenticate", `Bearer realm="api"`)
			rw.WriteHeader(http.StatusOK)
		})

		req := makeRequest(http.MethodGet, "http://example.com/")
		req.AddCookie(&http.Cookie{Name: "token", Value: "secret"})
		req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiJ9")

		rec := httptest.NewRecorder()
		mdlw.Wrap(next).ServeHTTP(makeWriter(rec), req)

		require.NoError(t, harWriter.Close())

		archive := readHAR(t, path)
		require.Len(t, archive.Log.Entries, 1)

		entry := archive.Log.Entries[0]

		// Cookies arrays must be empty.
		assert.Empty(t, entry.Request.Cookies)
		assert.Empty(t, entry.Response.Cookies)

		// Sensitive headers must not appear.
		blocked := []string{"Cookie", "Authorization"}
		for _, name := range blocked {
			for _, nv := range entry.Request.Headers {
				assert.NotEqual(t, name, nv.Name, "request header %s must be stripped", name)
			}
		}

		for _, nv := range entry.Response.Headers {
			assert.NotEqual(t, "Set-Cookie", nv.Name)
			assert.NotEqual(t, "Www-Authenticate", nv.Name)
		}
	})

	t.Run("secure headers captured when WithCaptureSecureHeaders(true)", func(t *testing.T) {
		mdlw, harWriter, path := newMiddleware(t, har.WithCaptureSecureHeaders(true))

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			http.SetCookie(rw, &http.Cookie{Name: "session", Value: "abc"})
			rw.WriteHeader(http.StatusOK)
		})

		req := makeRequest(http.MethodGet, "http://example.com/")
		req.AddCookie(&http.Cookie{Name: "token", Value: "secret"})
		req.Header.Set("Authorization", "Bearer token123")

		rec := httptest.NewRecorder()
		mdlw.Wrap(next).ServeHTTP(makeWriter(rec), req)

		require.NoError(t, harWriter.Close())

		archive := readHAR(t, path)
		require.Len(t, archive.Log.Entries, 1)

		entry := archive.Log.Entries[0]

		assert.NotEmpty(t, entry.Request.Cookies)
		assert.NotEmpty(t, entry.Response.Cookies)

		var hasAuth bool

		for _, nv := range entry.Request.Headers {
			if nv.Name == "Authorization" {
				hasAuth = true

				break
			}
		}

		assert.True(t, hasAuth, "Authorization header must be present when captureSecureHeaders is true")
	})

	t.Run("gzip response body is decoded in HAR", func(t *testing.T) {
		mdlw, harWriter, path := newMiddleware(t)

		const originalBody = `{"status":"ok"}`

		var buf bytes.Buffer

		gz := gzip.NewWriter(&buf)
		_, err := gz.Write([]byte(originalBody))
		require.NoError(t, err)
		require.NoError(t, gz.Close())

		compressed := buf.Bytes()

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			rw.Header().Set("Content-Type", "application/json")
			rw.Header().Set("Content-Encoding", "gzip")
			rw.WriteHeader(http.StatusOK)
			_, _ = rw.Write(compressed)
		})

		rec := httptest.NewRecorder()
		mdlw.Wrap(next).ServeHTTP(makeWriter(rec), makeRequest(http.MethodGet, "http://example.com/api"))

		require.NoError(t, harWriter.Close())

		archive := readHAR(t, path)
		require.Len(t, archive.Log.Entries, 1)

		content := archive.Log.Entries[0].Response.Content
		assert.JSONEq(t, originalBody, content.Text)
		assert.Empty(t, content.Encoding, "encoding field should be empty for plain text")
	})
}
