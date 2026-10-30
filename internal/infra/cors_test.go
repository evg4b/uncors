package infra_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const expectedAllowMethods = "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS"

func corsHeaders(origin, exposeHeaders string) http.Header {
	header := http.Header{
		headers.AccessControlAllowOrigin:      []string{origin},
		headers.AccessControlAllowCredentials: []string{"true"},
		headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
		headers.AccessControlMaxAge:           []string{"86400"},
	}

	if exposeHeaders != "" {
		header[headers.AccessControlExposeHeaders] = []string{exposeHeaders}
	}

	if origin != "*" {
		header[headers.Vary] = []string{headers.Origin}
	}

	return header
}

func preflightHeaders(origin, allowHeaders, allowMethods string, vary ...string) http.Header {
	header := http.Header{
		headers.AccessControlAllowOrigin:      []string{origin},
		headers.AccessControlAllowCredentials: []string{"true"},
		headers.AccessControlAllowHeaders:     []string{allowHeaders},
		headers.AccessControlAllowMethods:     []string{allowMethods},
		headers.AccessControlMaxAge:           []string{"86400"},
		headers.AccessControlExposeHeaders:    []string{"*"},
	}

	if len(vary) > 0 {
		header[headers.Vary] = vary
	}

	return header
}

func withHeader(header http.Header, name string, values ...string) http.Header {
	header[name] = values

	return header
}

func TestWriteCorsHeaders(t *testing.T) {
	origin := hosts.Localhost.HTTPPort(4000).String()

	tests := []struct {
		name     string
		header   http.Header
		origin   string
		expected http.Header
	}{
		{
			name:     "Empty headers without origin",
			header:   http.Header{},
			origin:   "",
			expected: corsHeaders("*", "*"),
		},
		{
			name:     "Empty headers with origin",
			header:   http.Header{},
			origin:   origin,
			expected: corsHeaders(origin, "Date, Vary"),
		},
		{
			name: "Override existing headers",
			header: http.Header{
				headers.AccessControlAllowOrigin:      []string{hosts.Github.HTTPS().String()},
				headers.AccessControlAllowCredentials: []string{"false"},
				headers.AccessControlAllowMethods:     []string{"GET, OPTIONS"},
				headers.AccessControlAllowHeaders:     []string{"*"},
				headers.AccessControlExposeHeaders:    []string{"*"},
			},
			origin:   "",
			expected: corsHeaders("*", "*"),
		},
		{
			name: "Do not change existing headers",
			header: http.Header{
				"X-Data": []string{hosts.Github.HTTPS().String()},
			},
			origin: "",
			expected: withHeader(
				corsHeaders("*", "*"),
				"X-Data", hosts.Github.HTTPS().String(),
			),
		},
		{
			name: "Expose response headers to the origin",
			header: http.Header{
				"X-Total-Count":         []string{"10"},
				"X-Request-Id":          []string{"42"},
				headers.ContentEncoding: []string{"gzip"},
				headers.CacheControl:    []string{"no-cache"},
				headers.ContentLanguage: []string{"en"},
				headers.ContentLength:   []string{"10"},
				headers.ContentType:     []string{"text/plain"},
				headers.Expires:         []string{"0"},
				headers.LastModified:    []string{"Wed, 21 Oct 2015 07:28:00 GMT"},
				headers.Pragma:          []string{"no-cache"},
				headers.SetCookie:       []string{"session=1"},
			},
			origin: origin,
			expected: http.Header{
				headers.AccessControlAllowOrigin:      []string{origin},
				headers.AccessControlAllowCredentials: []string{"true"},
				headers.AccessControlAllowMethods:     []string{expectedAllowMethods},
				headers.AccessControlMaxAge:           []string{"86400"},
				headers.AccessControlExposeHeaders:    []string{"Content-Encoding, Date, Vary, X-Request-Id, X-Total-Count"},
				headers.Vary:                          []string{headers.Origin},
				"X-Total-Count":                       []string{"10"},
				"X-Request-Id":                        []string{"42"},
				headers.ContentEncoding:               []string{"gzip"},
				headers.CacheControl:                  []string{"no-cache"},
				headers.ContentLanguage:               []string{"en"},
				headers.ContentLength:                 []string{"10"},
				headers.ContentType:                   []string{"text/plain"},
				headers.Expires:                       []string{"0"},
				headers.LastModified:                  []string{"Wed, 21 Oct 2015 07:28:00 GMT"},
				headers.Pragma:                        []string{"no-cache"},
				headers.SetCookie:                     []string{"session=1"},
			},
		},
		{
			name: "Keep existing Vary",
			header: http.Header{
				headers.Vary: []string{"Accept-Encoding"},
			},
			origin: origin,
			expected: withHeader(
				corsHeaders(origin, "Date, Vary"),
				headers.Vary, "Accept-Encoding", headers.Origin,
			),
		},
		{
			name: "Do not repeat Origin in Vary",
			header: http.Header{
				headers.Vary: []string{"Accept-Encoding, ORIGIN"},
			},
			origin: origin,
			expected: withHeader(
				corsHeaders(origin, "Date, Vary"),
				headers.Vary, "Accept-Encoding, ORIGIN",
			),
		},
		{
			name: "Do not add Origin to Vary with a wildcard",
			header: http.Header{
				headers.Vary: []string{"*"},
			},
			origin: origin,
			expected: withHeader(
				corsHeaders(origin, "Date, Vary"),
				headers.Vary, "*",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infra.WriteCorsHeaders(tt.header, tt.origin)

			assert.Equal(t, tt.expected, tt.header)
		})
	}

	t.Run("Exposes Date written by the server later", func(t *testing.T) {
		header := http.Header{}

		infra.WriteCorsHeaders(header, origin)

		assert.Equal(t, "Date, Vary", header.Get(headers.AccessControlExposeHeaders))
	})

	t.Run("Exposes Date once", func(t *testing.T) {
		header := http.Header{"Date": []string{"Wed, 21 Oct 2015 07:28:00 GMT"}}

		infra.WriteCorsHeaders(header, origin)

		assert.Equal(t, "Date, Vary", header.Get(headers.AccessControlExposeHeaders))
	})

	t.Run("Is idempotent", func(t *testing.T) {
		header := http.Header{"X-Total-Count": []string{"10"}}

		infra.WriteCorsHeaders(header, origin)
		infra.WriteCorsHeaders(header, origin)

		assert.Equal(t, withHeader(
			corsHeaders(origin, "Date, Vary, X-Total-Count"),
			"X-Total-Count", "10",
		), header)
	})
}

func TestWriteCorsHeadersForOptions(t *testing.T) {
	testAllowHeaders := "Content-Type, Authorization"
	xCustomHeader := "X-Custom-Header"

	tests := []struct {
		name            string
		reqHeader       http.Header
		expectedHeaders http.Header
	}{
		{
			name:            "Empty request headers",
			reqHeader:       http.Header{},
			expectedHeaders: preflightHeaders("*", "*", expectedAllowMethods),
		},
		{
			name: "With Origin header",
			reqHeader: http.Header{
				headers.Origin: []string{hosts.Localhost.HTTPPort(4000).String()},
			},
			expectedHeaders: preflightHeaders(
				hosts.Localhost.HTTPPort(4000).String(), "*", expectedAllowMethods,
				headers.Origin,
			),
		},
		{
			name: "With AccessControlRequestHeaders",
			reqHeader: http.Header{
				headers.AccessControlRequestHeaders: []string{testAllowHeaders},
			},
			expectedHeaders: preflightHeaders(
				"*", testAllowHeaders, expectedAllowMethods,
				headers.AccessControlRequestHeaders,
			),
		},
		{
			name: "With AccessControlRequestMethod",
			reqHeader: http.Header{
				headers.AccessControlRequestMethod: []string{"POST"},
			},
			expectedHeaders: preflightHeaders("*", "*", "POST", headers.AccessControlRequestMethod),
		},
		{
			name: "With all request headers",
			reqHeader: http.Header{
				headers.Origin:                      []string{hosts.Github.HTTPS().String()},
				headers.AccessControlRequestHeaders: []string{xCustomHeader},
				headers.AccessControlRequestMethod:  []string{"PUT"},
			},
			expectedHeaders: preflightHeaders(
				hosts.Github.HTTPS().String(), xCustomHeader, "PUT",
				headers.Origin, headers.AccessControlRequestMethod, headers.AccessControlRequestHeaders,
			),
		},
		{
			name: "Ignore unrelated request headers",
			reqHeader: http.Header{
				headers.Origin:      []string{"http://localhost:3000"},
				xCustomHeader:       []string{"custom-value"},
				headers.UserAgent:   []string{"Mozilla/5.0"},
				headers.ContentType: []string{"application/json"},
			},
			expectedHeaders: preflightHeaders("http://localhost:3000", "*", expectedAllowMethods, headers.Origin),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respHeader := http.Header{}
			infra.WriteCorsHeadersForOptions(respHeader, tt.reqHeader)

			assert.Equal(t, tt.expectedHeaders, respHeader)
		})
	}

	t.Run("Overwrite existing response headers", func(t *testing.T) {
		respHeader := http.Header{
			headers.AccessControlAllowOrigin:  []string{"*"},
			headers.AccessControlAllowMethods: []string{"GET"},
			headers.Vary:                      []string{"Accept-Encoding, Origin"},
		}

		infra.WriteCorsHeadersForOptions(respHeader, http.Header{
			headers.Origin:                     []string{hosts.Example.HTTP().String()},
			headers.AccessControlRequestMethod: []string{"DELETE"},
		})

		assert.Equal(t, preflightHeaders(
			hosts.Example.HTTP().String(), "*", "DELETE",
			"Accept-Encoding, Origin", headers.AccessControlRequestMethod,
		), respHeader)
	})
}

func TestCorsWriter(t *testing.T) {
	origin := hosts.Localhost.HTTPPort(4000).String()

	newWriter := func(origin string) (*httptest.ResponseRecorder, *infra.CorsWriter) {
		recorder := httptest.NewRecorder()

		return recorder, infra.NewCorsWriter(server.NewResponseRecorder(recorder), originRequest(t, origin))
	}

	t.Run("exposes the headers set before WriteHeader", func(t *testing.T) {
		recorder, writer := newWriter(origin)

		writer.Header().Set("X-Total-Count", "10")
		writer.WriteHeader(http.StatusCreated)
		writer.Header().Set("X-Late", "1")

		response := recorder.Result()
		require.NoError(t, response.Body.Close())

		assert.Equal(t, http.StatusCreated, response.StatusCode)
		assert.Equal(t, "Date, Vary, X-Total-Count", response.Header.Get(headers.AccessControlExposeHeaders))
		assert.Equal(t, []string{headers.Origin}, response.Header.Values(headers.Vary))
		assert.Equal(t, http.StatusCreated, writer.StatusCode())
	})

	t.Run("exposes the headers set before Write", func(t *testing.T) {
		recorder, writer := newWriter(origin)

		writer.Header().Set("X-Request-ID", "42")
		_, err := writer.Write([]byte("ok"))
		require.NoError(t, err)

		response := recorder.Result()
		require.NoError(t, response.Body.Close())

		assert.Equal(t, "Date, Vary, X-Request-Id", response.Header.Get(headers.AccessControlExposeHeaders))
	})

	t.Run("keeps exposed headers set by the handler", func(t *testing.T) {
		recorder, writer := newWriter(origin)

		writer.Header().Set("X-Total-Count", "10")
		writer.Header().Set(headers.AccessControlExposeHeaders, "X-Custom")
		writer.WriteHeader(http.StatusOK)

		assert.Equal(t, "X-Custom", recorder.Header().Get(headers.AccessControlExposeHeaders))
	})

	t.Run("exposes every header without an origin", func(t *testing.T) {
		recorder, writer := newWriter("")

		writer.Header().Set("X-Total-Count", "10")
		writer.WriteHeader(http.StatusOK)

		assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlExposeHeaders))
		assert.Empty(t, recorder.Header().Values(headers.Vary))
	})

	t.Run("exposes headers once", func(t *testing.T) {
		recorder, writer := newWriter(origin)

		writer.Header().Set("X-First", "1")
		writer.ExposeHeaders()
		writer.Header().Set("X-Second", "2")
		writer.ExposeHeaders()

		assert.Equal(t, "Date, Vary, X-First", recorder.Header().Get(headers.AccessControlExposeHeaders))
	})

	t.Run("replaces upstream CORS headers", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Set(headers.AccessControlAllowHeaders, "*")
		recorder.Header().Set(headers.AccessControlExposeHeaders, "*")

		writer := infra.NewCorsWriter(server.NewResponseRecorder(recorder), originRequest(t, origin))
		writer.WriteHeader(http.StatusOK)

		assert.Empty(t, recorder.Header().Values(headers.AccessControlAllowHeaders))
		assert.Equal(t, "Date, Vary", recorder.Header().Get(headers.AccessControlExposeHeaders))
	})

	t.Run("unwraps to the underlying writer", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		inner := server.NewResponseRecorder(recorder)

		assert.Same(t, inner, infra.NewCorsWriter(inner, originRequest(t, origin)).Unwrap())
	})

	t.Run("withholds CORS headers from a rejected origin", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		recorder.Header().Set(headers.AccessControlAllowOrigin, "*")

		request := testutils.RejectOrigin(t, originRequest(t, origin))
		writer := infra.NewCorsWriter(server.NewResponseRecorder(recorder), request)

		writer.Header().Set("X-Total-Count", "10")
		writer.Header().Set(headers.AccessControlAllowCredentials, "true")
		writer.WriteHeader(http.StatusOK)

		assert.Equal(t, http.Header{
			"X-Total-Count": []string{"10"},
			headers.Vary:    []string{headers.Origin},
		}, recorder.Header())
	})
}

func TestWriteRequestCorsHeaders(t *testing.T) {
	origin := hosts.Localhost.HTTPPort(4000).String()

	t.Run("writes CORS headers for the request origin", func(t *testing.T) {
		header := http.Header{}

		infra.WriteRequestCorsHeaders(header, originRequest(t, origin))

		assert.Equal(t, corsHeaders(origin, "Date, Vary"), header)
	})

	t.Run("writes CORS headers without an origin", func(t *testing.T) {
		header := http.Header{}

		infra.WriteRequestCorsHeaders(header, originRequest(t, ""))

		assert.Equal(t, corsHeaders("*", "*"), header)
	})

	t.Run("writes preflight headers for an OPTIONS request", func(t *testing.T) {
		header := http.Header{}
		request := originRequest(t, origin)
		request.Method = "options"

		infra.WriteRequestCorsHeaders(header, request)

		assert.Equal(t, preflightHeaders(origin, "*", expectedAllowMethods, headers.Origin), header)
	})

	t.Run("withholds CORS headers from a rejected origin", func(t *testing.T) {
		header := corsHeaders(origin, "X-Total-Count")
		header.Set("X-Total-Count", "10")

		infra.WriteRequestCorsHeaders(header, testutils.RejectOrigin(t, originRequest(t, origin)))

		assert.Equal(t, http.Header{
			"X-Total-Count": []string{"10"},
			headers.Vary:    []string{headers.Origin},
		}, header)
	})
}

func TestWithholdCorsHeaders(t *testing.T) {
	header := http.Header{
		headers.AccessControlAllowOrigin:   []string{"*"},
		headers.AccessControlExposeHeaders: []string{"*"},
		"access-control-max-age":           []string{"86400"},
		"Access-Control":                   []string{"kept"},
		headers.Vary:                       []string{"Accept-Encoding, Origin"},
		headers.ContentType:                []string{"text/plain"},
	}

	infra.WithholdCorsHeaders(header)

	assert.Equal(t, http.Header{
		"Access-Control":    []string{"kept"},
		headers.Vary:        []string{"Accept-Encoding, Origin"},
		headers.ContentType: []string{"text/plain"},
	}, header)
}

func originRequest(t *testing.T, origin string) *http.Request {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://api.local/data", nil)
	if origin != "" {
		request.Header.Set(headers.Origin, origin)
	}

	return request
}
