package har_test

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/andybalholm/brotli"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/har"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	harlib "github.com/evg4b/uncors/pkg/har"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/klauspost/compress/zstd"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeHARRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)

	return req
}

func newHARMiddleware(t *testing.T, opts ...har.MiddlewareOption) (*har.Middleware, *har.Writer, string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.har")
	harWriter := har.NewWriter(path, har.WithFileSystem(afero.NewOsFs()))
	opts = append([]har.MiddlewareOption{har.WithWriter(harWriter)}, opts...)
	mdlw := har.NewMiddleware(opts...)

	return mdlw, harWriter, path
}

func readHARFile(t *testing.T, path string) harlib.HAR {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var archive harlib.HAR
	require.NoError(t, json.Unmarshal(data, &archive))

	return archive
}

func compressBody(t *testing.T, algo string, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	switch algo {
	case "gzip":
		w := gzip.NewWriter(&buf)
		_, err := w.Write(data)
		require.NoError(t, err)
		require.NoError(t, w.Close())
	case "deflate":
		w := zlib.NewWriter(&buf)
		_, err := w.Write(data)
		require.NoError(t, err)
		require.NoError(t, w.Close())
	case "br":
		w := brotli.NewWriter(&buf)
		_, err := w.Write(data)
		require.NoError(t, err)
		require.NoError(t, w.Close())
	case "zstd":
		w, err := zstd.NewWriter(&buf)
		require.NoError(t, err)
		_, err = w.Write(data)
		require.NoError(t, err)
		require.NoError(t, w.Close())
	}

	return buf.Bytes()
}

func TestMiddleware_Wrap(t *testing.T) {
	t.Run("passes request to next handler", func(t *testing.T) {
		mdlw, harWriter, _ := newHARMiddleware(t)

		defer testutils.Close(t, harWriter)

		called := false
		next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
			called = true

			rw.WriteHeader(http.StatusOK)

			return nil
		})

		rec := httptest.NewRecorder()
		rr := server.NewResponseRecorder(rec)
		err := infra.Wrap(mdlw, next).ServeHTTP(rr, makeHARRequest(t, "http://example.com/path"))
		require.NoError(t, err)

		assert.True(t, called)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("records response body correctly", func(t *testing.T) {
		mdlw, harWriter, _ := newHARMiddleware(t)

		next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
			rw.Header().Set("Content-Type", "text/plain")
			rw.WriteHeader(http.StatusOK)
			fmt.Fprint(rw, "hello")

			return nil
		})

		rec := httptest.NewRecorder()
		rr := server.NewResponseRecorder(rec)
		err := infra.Wrap(mdlw, next).ServeHTTP(rr, makeHARRequest(t, "http://example.com/"))
		require.NoError(t, err)

		require.NoError(t, harWriter.Close())

		assert.Equal(t, "hello", rec.Body.String())
	})

	t.Run("records request with query string", func(t *testing.T) {
		mdlw, harWriter, _ := newHARMiddleware(t)

		next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
			rw.WriteHeader(http.StatusNoContent)

			return nil
		})

		req := makeHARRequest(t, "http://example.com/search?q=foo&page=2")
		rec := httptest.NewRecorder()
		rr := server.NewResponseRecorder(rec)
		err := infra.Wrap(mdlw, next).ServeHTTP(rr, req)
		require.NoError(t, err)

		require.NoError(t, harWriter.Close())

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("restores request body for downstream handlers", func(t *testing.T) {
		mdlw, harWriter, _ := newHARMiddleware(t)

		defer testutils.Close(t, harWriter)

		body := "request-payload"

		var received string

		next := infra.HandlerFunc(func(rw contracts.ResponseWriter, r *contracts.Request) error {
			b, _ := io.ReadAll(r.Body)
			received = string(b)

			rw.WriteHeader(http.StatusOK)

			return nil
		})

		req, err := http.NewRequestWithContext(
			context.Background(), http.MethodPost, "http://example.com/api", strings.NewReader(body),
		)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		rr := server.NewResponseRecorder(rec)
		serveErr := infra.Wrap(mdlw, next).ServeHTTP(rr, req)
		require.NoError(t, serveErr)

		assert.Equal(t, body, received)
	})

	t.Run("secure headers not captured by default", func(t *testing.T) {
		mdlw, harWriter, path := newHARMiddleware(t)

		next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
			http.SetCookie(rw, &http.Cookie{Name: "session", Value: "abc"}) // nolint: gosec
			rw.Header().Set("Www-Authenticate", `Bearer realm="api"`)
			rw.WriteHeader(http.StatusOK)

			return nil
		})

		req := makeHARRequest(t, "http://example.com/")
		req.AddCookie(&http.Cookie{Name: "token", Value: "secret"}) // nolint: gosec
		req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiJ9")

		rec := httptest.NewRecorder()
		rr := server.NewResponseRecorder(rec)
		err := infra.Wrap(mdlw, next).ServeHTTP(rr, req)
		require.NoError(t, err)

		require.NoError(t, harWriter.Close())

		archive := readHARFile(t, path)
		require.Len(t, archive.Log.Entries, 1)

		entry := archive.Log.Entries[0]

		assert.Empty(t, entry.Request.Cookies)
		assert.Empty(t, entry.Response.Cookies)

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

	t.Run("uses https scheme for TLS requests", func(t *testing.T) {
		mdlw, harWriter, path := newHARMiddleware(t)

		next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
			rw.WriteHeader(http.StatusOK)

			return nil
		})

		req := makeHARRequest(t, "https://example.com/")
		req.TLS = &tls.ConnectionState{}

		rec := httptest.NewRecorder()
		rr := server.NewResponseRecorder(rec)
		err := infra.Wrap(mdlw, next).ServeHTTP(rr, req)
		require.NoError(t, err)

		require.NoError(t, harWriter.Close())

		archive := readHARFile(t, path)
		require.Len(t, archive.Log.Entries, 1)

		assert.True(t,
			strings.HasPrefix(archive.Log.Entries[0].Request.URL, "https://"),
			"URL must use https scheme for TLS requests",
		)
	})

	t.Run("secure headers captured when WithCaptureSecureHeaders(true)", func(t *testing.T) {
		mdlw, harWriter, path := newHARMiddleware(t, har.WithCaptureSecureHeaders(true))

		next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
			http.SetCookie(rw, &http.Cookie{Name: "session", Value: "abc"}) // nolint: gosec
			rw.WriteHeader(http.StatusOK)

			return nil
		})

		req := makeHARRequest(t, "http://example.com/")
		req.AddCookie(&http.Cookie{Name: "token", Value: "secret"}) // nolint: gosec
		req.Header.Set("Authorization", "Bearer token123")

		rec := httptest.NewRecorder()
		rr := server.NewResponseRecorder(rec)
		serveErr := infra.Wrap(mdlw, next).ServeHTTP(rr, req)
		require.NoError(t, serveErr)

		require.NoError(t, harWriter.Close())

		archive := readHARFile(t, path)
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
}

func TestMiddleware_Wrap_Decompression(t *testing.T) {
	tests := []struct {
		encoding string
	}{
		{encoding: "gzip"},
		{encoding: "deflate"},
		{encoding: "br"},
		{encoding: "zstd"},
	}

	for _, testCase := range tests {
		t.Run(testCase.encoding, func(t *testing.T) {
			mdlw, harWriter, path := newHARMiddleware(t)

			const originalBody = `{"status":"ok"}`

			compressed := compressBody(t, testCase.encoding, []byte(originalBody))

			next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
				rw.Header().Set("Content-Type", "application/json")
				rw.Header().Set("Content-Encoding", testCase.encoding)
				rw.WriteHeader(http.StatusOK)

				_, err := rw.Write(compressed)
				if err != nil {
					return err
				}

				return nil
			})

			rec := httptest.NewRecorder()
			rr := server.NewResponseRecorder(rec)
			serveErr := infra.Wrap(mdlw, next).ServeHTTP(rr, makeHARRequest(t, "http://example.com/api"))
			require.NoError(t, serveErr)

			require.NoError(t, harWriter.Close())

			archive := readHARFile(t, path)
			require.Len(t, archive.Log.Entries, 1)

			content := archive.Log.Entries[0].Response.Content
			assert.JSONEq(t, originalBody, content.Text)
			assert.Empty(t, content.Encoding, "encoding field should be empty for decoded text")
		})
	}
}

func serveHAR(t *testing.T, mdlw *har.Middleware, req *http.Request) {
	t.Helper()

	next := infra.HandlerFunc(func(rw contracts.ResponseWriter, _ *contracts.Request) error {
		rw.WriteHeader(http.StatusOK)

		return nil
	})

	err := infra.Wrap(mdlw, next).ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)
	require.NoError(t, err)
}

func TestMiddleware_PostData(t *testing.T) {
	t.Run("records a text body", func(t *testing.T) {
		mdlw, harWriter, path := newHARMiddleware(t)

		const body = `{"name":"uncors"}`

		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodPost, "http://example.com/api", strings.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		serveHAR(t, mdlw, req)
		require.NoError(t, harWriter.Close())

		archive := readHARFile(t, path)
		require.Len(t, archive.Log.Entries, 1)

		request := archive.Log.Entries[0].Request
		assert.Equal(t, int64(len(body)), request.BodySize)
		assert.Equal(t, &harlib.PostData{MimeType: "application/json", Text: body}, request.PostData)
	})

	t.Run("records only the size of a binary body", func(t *testing.T) {
		mdlw, harWriter, path := newHARMiddleware(t)

		body := []byte{0xff, 0xfe, 0x00, 0x01}

		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodPut, "http://example.com/upload", bytes.NewReader(body),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/octet-stream")

		serveHAR(t, mdlw, req)
		require.NoError(t, harWriter.Close())

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotContains(t, string(data), `"text"`)

		archive := readHARFile(t, path)
		require.Len(t, archive.Log.Entries, 1)

		request := archive.Log.Entries[0].Request
		assert.Equal(t, int64(len(body)), request.BodySize)
		assert.Equal(t, &harlib.PostData{MimeType: "application/octet-stream"}, request.PostData)
	})

	t.Run("omits postData without a body", func(t *testing.T) {
		mdlw, harWriter, path := newHARMiddleware(t)

		serveHAR(t, mdlw, makeHARRequest(t, "http://example.com/"))
		require.NoError(t, harWriter.Close())

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "postData")

		archive := readHARFile(t, path)
		require.Len(t, archive.Log.Entries, 1)
		assert.Equal(t, int64(0), archive.Log.Entries[0].Request.BodySize)
	})
}

func TestMiddleware_RequestBodyReadError(t *testing.T) {
	mdlw, harWriter, path := newHARMiddleware(t)

	readErr := io.ErrUnexpectedEOF
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example.com/api",
		io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(readErr)))
	require.NoError(t, err)

	called := false
	next := infra.HandlerFunc(func(contracts.ResponseWriter, *contracts.Request) error {
		called = true

		return nil
	})

	serveErr := infra.Wrap(mdlw, next).ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), req)

	require.ErrorIs(t, serveErr, readErr)
	assert.False(t, called)

	require.NoError(t, harWriter.Close())
	assert.Empty(t, readHARFile(t, path).Log.Entries)
}

func TestMiddleware_ResponseHTTPVersion(t *testing.T) {
	for _, proto := range []struct {
		name  string
		major int
		minor int
	}{
		{name: "HTTP/1.0", major: 1, minor: 0},
		{name: "HTTP/1.1", major: 1, minor: 1},
		{name: "HTTP/2.0", major: 2, minor: 0},
	} {
		t.Run(proto.name, func(t *testing.T) {
			mdlw, harWriter, path := newHARMiddleware(t)

			req := makeHARRequest(t, "http://example.com/")
			req.Proto, req.ProtoMajor, req.ProtoMinor = proto.name, proto.major, proto.minor

			serveHAR(t, mdlw, req)
			require.NoError(t, harWriter.Close())

			archive := readHARFile(t, path)
			require.Len(t, archive.Log.Entries, 1)
			assert.Equal(t, proto.name, archive.Log.Entries[0].Request.HTTPVersion)
			assert.Equal(t, proto.name, archive.Log.Entries[0].Response.HTTPVersion)
		})
	}
}

func TestNewMiddleware_NilWriterPanic(t *testing.T) {
	assert.PanicsWithValue(
		t,
		"HARMiddleware: Writer is not configured",
		func() { har.NewMiddleware() },
	)
}
