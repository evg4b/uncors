package infra_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"syscall"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const expectedPage = `
███████  ██████   ██████      ███████ ██████  ██████   ██████  ██████  
██      ██  ████ ██  ████     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████ ██ ██ ██ ██ ██ ██     █████   ██████  ██████  ██    ██ ██████  
     ██ ████  ██ ████  ██     ██      ██   ██ ██   ██ ██    ██ ██   ██ 
███████  ██████   ██████      ███████ ██   ██ ██   ██  ██████  ██   ██ 


An error occurred: net/http: abort Handler
`

func writeError(t *testing.T, request *http.Request, err error) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	infra.HTTPError(server.NewResponseRecorder(recorder), request, err)

	return recorder
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()

	return httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://api.local/data", nil)
}

func TestHttpError(t *testing.T) {
	recorder := writeError(t, newRequest(t), http.ErrAbortHandler)
	body := testutils.ReadBody(t, recorder)

	t.Run("write correct page", func(t *testing.T) {
		assert.Contains(t, body, expectedPage)
	})

	t.Run("write correct headers", func(t *testing.T) {
		header := recorder.Header()

		assert.NotNil(t, header[headers.ContentType])
		assert.NotNil(t, header[headers.ContentEncoding])
		assert.NotNil(t, header[headers.CacheControl])
		assert.NotNil(t, header[headers.Pragma])
		assert.NotNil(t, header[headers.XContentTypeOptions])
		assert.Nil(t, header[headers.SetCookie])
	})

	t.Run("does not include memory usage", func(t *testing.T) {
		assert.NotContains(t, body, "Memory usage:")
	})
}

func TestHttpErrorCorsHeaders(t *testing.T) {
	t.Run("allows the request origin", func(t *testing.T) {
		request := newRequest(t)
		request.Header.Set(headers.Origin, "http://localhost:3000")

		recorder := writeError(t, request, errHandler)

		assert.Equal(t, "http://localhost:3000", recorder.Header().Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", recorder.Header().Get(headers.AccessControlAllowCredentials))
	})

	t.Run("allows any origin without an Origin header", func(t *testing.T) {
		recorder := writeError(t, newRequest(t), errHandler)

		assert.Equal(t, "*", recorder.Header().Get(headers.AccessControlAllowOrigin))
	})
}

func TestHttpErrorStatus(t *testing.T) {
	refused := &url.Error{
		Op:  "Get",
		URL: "http://upstream.local",
		Err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
	}
	timedOut := &url.Error{
		Op:  "Get",
		URL: "http://upstream.local",
		Err: &net.DNSError{Err: "i/o timeout", Name: "upstream.local", IsTimeout: true},
	}
	parseErr := &url.Error{Op: "parse", URL: "http://[::1", Err: errHandler}

	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "host not mapped",
			err:      infra.ErrHostNotMapped,
			expected: http.StatusNotFound,
		},
		{
			name:     "upstream connection refused",
			err:      fmt.Errorf("failed to execute request: %w", refused),
			expected: http.StatusBadGateway,
		},
		{
			name:     "upstream dial error",
			err:      &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			expected: http.StatusBadGateway,
		},
		{
			name:     "upstream timeout",
			err:      fmt.Errorf("failed to execute request: %w", timedOut),
			expected: http.StatusGatewayTimeout,
		},
		{
			name:     "deadline exceeded",
			err:      fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
			expected: http.StatusGatewayTimeout,
		},
		{
			name:     "invalid url",
			err:      parseErr,
			expected: http.StatusInternalServerError,
		},
		{
			name:     "other error",
			err:      errHandler,
			expected: http.StatusInternalServerError,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := writeError(t, newRequest(t), testCase.err)

			assert.Equal(t, testCase.expected, recorder.Code)
			assert.Contains(t, testutils.ReadBody(t, recorder), testCase.err.Error())
		})
	}
}

func TestHttpErrorCancelledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://api.local/data", nil)
	recorder := httptest.NewRecorder()
	writer := server.NewResponseRecorder(recorder)

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		infra.HTTPError(writer, request, fmt.Errorf("failed to execute request: %w", context.Canceled))
	})
	assert.Zero(t, writer.StatusCode())
	assert.Empty(t, recorder.Body.String())
	assert.Empty(t, recorder.Header())
}

func TestHttpErrorStartedResponse(t *testing.T) {
	tests := []struct {
		name  string
		start func(http.ResponseWriter)
	}{
		{
			name: "status sent",
			start: func(writer http.ResponseWriter) {
				writer.WriteHeader(http.StatusOK)
				fmt.Fprint(writer, "partial")
			},
		},
		{
			name: "body sent without a status",
			start: func(writer http.ResponseWriter) {
				fmt.Fprint(writer, "partial")
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			writer := server.NewResponseRecorder(recorder)
			testCase.start(writer)

			assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
				infra.HTTPError(writer, newRequest(t), errHandler)
			})
			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "partial", recorder.Body.String())
		})
	}
}

func TestCastToHTTPHandlerAbortsStartedResponse(t *testing.T) {
	failing := infra.HandlerFunc(func(writer contracts.ResponseWriter, _ *contracts.Request) error {
		writer.WriteHeader(http.StatusOK)
		fmt.Fprint(writer, "partial")

		return errHandler
	})
	handler := infra.CastToHTTPHandler(failing)

	recorder := httptest.NewRecorder()

	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		handler.ServeHTTP(server.NewResponseRecorder(recorder), newRequest(t))
	})
	require.Equal(t, "partial", recorder.Body.String())
}
