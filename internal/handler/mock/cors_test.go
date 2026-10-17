package mock_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/mock"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerCors(t *testing.T) {
	origin := hosts.Localhost.HTTPPort(3000).String()
	fileSystem := testutils.FsFromMap(t, map[string]string{
		textFile: textContent,
	})

	serve := func(t *testing.T, response config.Response, origin string) http.Header {
		t.Helper()

		handler := mock.NewHandler(
			mock.WithResponse(&response),
			mock.WithFileSystem(fileSystem),
		)

		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		if origin != "" {
			request.Header.Set(headers.Origin, origin)
		}

		recorder := httptest.NewRecorder()

		err := handler.ServeHTTP(server.NewResponseRecorder(recorder), request)
		require.NoError(t, err)

		return testutils.ReadHeader(t, recorder)
	}

	t.Run("exposes the headers of a raw response", func(t *testing.T) {
		header := serve(t, config.Response{
			Code: http.StatusOK,
			Raw:  textContent,
			Headers: map[string]string{
				"X-Total-Count":     "10",
				headers.ContentType: textPlain,
				headers.ETag:        `"v1"`,
			},
		}, origin)

		assert.Equal(t, origin, header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "Date, Etag, Vary, X-Total-Count", header.Get(headers.AccessControlExposeHeaders))
		assert.Equal(t, []string{headers.Origin}, header.Values(headers.Vary))
		assert.Empty(t, header.Values(headers.AccessControlAllowHeaders))
	})

	t.Run("exposes the headers of a file response", func(t *testing.T) {
		header := serve(t, config.Response{
			Code: http.StatusOK,
			File: textFile,
			Headers: map[string]string{
				"X-Request-Id": "42",
			},
		}, origin)

		assert.Equal(t, "Accept-Ranges, Date, Vary, X-Request-Id", header.Get(headers.AccessControlExposeHeaders))
		assert.Equal(t, []string{headers.Origin}, header.Values(headers.Vary))
	})

	t.Run("keeps the exposed headers set in the mock", func(t *testing.T) {
		header := serve(t, config.Response{
			Code: http.StatusOK,
			Raw:  textContent,
			Headers: map[string]string{
				"X-Total-Count":                    "10",
				headers.AccessControlExposeHeaders: "X-Custom",
			},
		}, origin)

		assert.Equal(t, "X-Custom", header.Get(headers.AccessControlExposeHeaders))
	})

	t.Run("exposes every header without an origin", func(t *testing.T) {
		header := serve(t, config.Response{
			Code:    http.StatusOK,
			Raw:     textContent,
			Headers: map[string]string{"X-Total-Count": "10"},
		}, "")

		assert.Equal(t, "*", header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "*", header.Get(headers.AccessControlExposeHeaders))
		assert.Empty(t, header.Values(headers.Vary))
	})
}
