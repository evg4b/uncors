package infra_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUpstream = errors.New("dial failed")

func request(t *testing.T) *http.Request {
	t.Helper()

	return httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://api.local/users", nil)
}

func TestStatusFor(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "explicit status", err: infra.NewHTTPStatusError(http.StatusNotFound, "nope", nil), expected: 404},
		{name: "client went away", err: context.Canceled, expected: 499},
		{name: "aborted handler", err: http.ErrAbortHandler, expected: 499},
		{name: "timed out", err: context.DeadlineExceeded, expected: 504},
		{name: "connection refused", err: syscall.ECONNREFUSED, expected: 502},
		{name: "dns failure", err: &net.DNSError{Err: "no such host"}, expected: 502},
		{name: "anything else", err: errUpstream, expected: 500},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			code, _ := infra.StatusFor(testCase.err)

			assert.Equal(t, testCase.expected, code)
		})
	}
}

func TestHTTPError(t *testing.T) {
	t.Run("reports the status the error asked for", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		infra.HTTPError(recorder, request(t), infra.NewHTTPStatusError(
			http.StatusNotFound,
			"this host is not mapped in the uncors configuration",
			nil,
		))

		body := testutils.ReadBody(t, recorder)

		assert.Equal(t, http.StatusNotFound, recorder.Code)
		assert.Contains(t, body, "Error 404")
		assert.Contains(t, body, "this host is not mapped")
		assert.Contains(t, body, "GET http://api.local/users")
	})

	// The client is the wrong audience for a stack trace, and ReadMemStats stops
	// the world — neither belongs on a path any request can reach.
	t.Run("leaks neither a stack trace nor memory statistics", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		infra.HTTPError(recorder, request(t), errUpstream)

		body := testutils.ReadBody(t, recorder)

		assert.NotContains(t, body, "goroutine ")
		assert.NotContains(t, body, "Stack trace")
		assert.NotContains(t, body, "Memory usage")
		assert.NotContains(t, body, "TotalAlloc")
	})

	t.Run("writes nothing when the client is already gone", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		infra.HTTPError(recorder, request(t), context.Canceled)

		assert.Empty(t, testutils.ReadBody(t, recorder))
	})

	t.Run("writes correct headers", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		infra.HTTPError(recorder, request(t), errUpstream)

		header := recorder.Header()

		require.NotNil(t, header[headers.ContentType])
		assert.Equal(t, "nosniff", header.Get(headers.XContentTypeOptions))
		assert.Empty(t, header[headers.SetCookie])
	})
}
