package script_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"testing/iotest"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/script"
	"github.com/evg4b/uncors/internal/server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBodyRead = errors.New("connection reset")

func serveWithTimeout(t *testing.T, source string, timeout time.Duration, request *http.Request) error {
	t.Helper()

	handler := script.NewHandler(
		script.WithScript(&config.Script{Script: source, Timeout: timeout}),
		script.WithFileSystem(afero.NewMemMapFs()),
	)

	return handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request)
}

func TestScriptHandlerTimeout(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{name: "infinite loop", script: "while true do end"},
		{name: "loop that catches errors", script: "while true do pcall(function() while true do end end) end"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

			startedAt := time.Now()
			err := serveWithTimeout(t, testCase.script, 50*time.Millisecond, request)

			require.ErrorIs(t, err, script.ErrScriptTimeout)
			assert.Less(t, time.Since(startedAt), 5*time.Second)
		})
	}
}

func TestScriptHandlerRequestCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

	time.AfterFunc(50*time.Millisecond, cancel)

	err := serveWithTimeout(t, "while true do end", time.Hour, request)

	require.ErrorIs(t, err, context.Canceled)
	require.NotErrorIs(t, err, script.ErrScriptTimeout)
}

func TestScriptHandlerOsExit(t *testing.T) {
	tests := []struct {
		name   string
		script string
	}{
		{name: "global os", script: "os.exit(0)"},
		{name: "required os", script: `require("os").exit(1)`},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

			err := serveWithTimeout(t, testCase.script, time.Minute, request)

			require.ErrorContains(t, err, script.ErrOsExit.Error())
		})
	}

	t.Run("can be caught with pcall", func(t *testing.T) {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
		recorder := httptest.NewRecorder()

		handler := script.NewHandler(
			script.WithScript(&config.Script{
				Script: `local ok = pcall(os.exit)
response:WriteString(tostring(ok))`,
			}),
			script.WithFileSystem(afero.NewMemMapFs()),
		)

		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), request))
		assert.Equal(t, "false", recorder.Body.String())
	})
}

func serveBody(t *testing.T, maxBodySize int64, body io.Reader) (*httptest.ResponseRecorder, error) {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", body)
	recorder := httptest.NewRecorder()

	handler := script.NewHandler(
		script.WithScript(&config.Script{
			Script:      "response:WriteString(tostring(#request.body))",
			MaxBodySize: maxBodySize,
		}),
		script.WithFileSystem(afero.NewMemMapFs()),
	)

	return recorder, handler.ServeHTTP(server.NewResponseRecorder(recorder), request)
}

func TestScriptHandlerRequestBody(t *testing.T) {
	tests := []struct {
		name        string
		maxBodySize int64
		limit       int64
	}{
		{name: "default limit", maxBodySize: 0, limit: script.DefaultMaxBodySize},
		{name: "configured limit", maxBodySize: 16, limit: 16},
		{
			name:        "configured limit above default",
			maxBodySize: script.DefaultMaxBodySize + 8,
			limit:       script.DefaultMaxBodySize + 8,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Run("body at the limit", func(t *testing.T) {
				body := bytes.Repeat([]byte("a"), int(testCase.limit))

				recorder, err := serveBody(t, testCase.maxBodySize, bytes.NewReader(body))

				require.NoError(t, err)
				assert.Equal(t, strconv.FormatInt(testCase.limit, 10), recorder.Body.String())
			})

			t.Run("body over the limit", func(t *testing.T) {
				body := bytes.Repeat([]byte("a"), int(testCase.limit)+1)

				_, err := serveBody(t, testCase.maxBodySize, bytes.NewReader(body))

				require.ErrorIs(t, err, script.ErrRequestBodyTooLarge)
			})
		})
	}

	t.Run("read failure", func(t *testing.T) {
		_, err := serveBody(t, 0, iotest.ErrReader(errBodyRead))

		require.ErrorIs(t, err, script.ErrRequestBodyNotRead)
		require.ErrorIs(t, err, errBodyRead)
	})
}
