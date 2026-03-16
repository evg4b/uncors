package har_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

	makeRequest := func(method, url string) *http.Request {
		req, err := http.NewRequest(method, url, nil)
		require.NoError(t, err)

		return req
	}

	newMiddleware := func(t *testing.T) (*har.Middleware, *har.Writer) {
		t.Helper()
		path := filepath.Join(t.TempDir(), "test.har")
		w := har.NewWriter(path)
		m := har.NewMiddleware(har.WithWriter(w))

		return m, w
	}

	t.Run("passes request to next handler", func(t *testing.T) {
		m, w := newMiddleware(t)

		defer w.Close() //nolint:errcheck

		called := false
		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, r *contracts.Request) {
			called = true
			rw.WriteHeader(http.StatusOK)
		})

		rec := httptest.NewRecorder()
		m.Wrap(next).ServeHTTP(makeWriter(rec), makeRequest(http.MethodGet, "http://example.com/path"))

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("records response body correctly", func(t *testing.T) {
		m, w := newMiddleware(t)

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			rw.Header().Set("Content-Type", "text/plain")
			rw.WriteHeader(http.StatusOK)
			fmt.Fprint(rw, "hello")
		})

		rec := httptest.NewRecorder()
		m.Wrap(next).ServeHTTP(makeWriter(rec), makeRequest(http.MethodGet, "http://example.com/"))

		require.NoError(t, w.Close())

		assert.Equal(t, "hello", rec.Body.String())
	})

	t.Run("records request with query string", func(t *testing.T) {
		m, w := newMiddleware(t)

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) {
			rw.WriteHeader(http.StatusNoContent)
		})

		req := makeRequest(http.MethodGet, "http://example.com/search?q=foo&page=2")
		rec := httptest.NewRecorder()
		m.Wrap(next).ServeHTTP(makeWriter(rec), req)

		require.NoError(t, w.Close())

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("restores request body for downstream handlers", func(t *testing.T) {
		m, w := newMiddleware(t)

		defer w.Close() //nolint:errcheck

		body := "request-payload"
		var received string

		next := contracts.HandlerFunc(func(rw contracts.ResponseWriter, r *contracts.Request) {
			b, _ := io.ReadAll(r.Body)
			received = string(b)
			rw.WriteHeader(http.StatusOK)
		})

		req, err := http.NewRequest(http.MethodPost, "http://example.com/api", strings.NewReader(body))
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		m.Wrap(next).ServeHTTP(makeWriter(rec), req)

		assert.Equal(t, body, received)
	})
}
