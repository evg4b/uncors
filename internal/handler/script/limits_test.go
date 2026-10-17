package script_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
		script.WithScript(&config.Script{Script: source}),
		script.WithFileSystem(afero.NewMemMapFs()),
		script.WithTimeout(timeout),
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

func TestScriptHandlerRequestBody(t *testing.T) {
	t.Run("body at the limit", func(t *testing.T) {
		body := strings.Repeat("a", script.MaxRequestBodySize)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body))
		recorder := httptest.NewRecorder()

		handler := script.NewHandler(
			script.WithScript(&config.Script{Script: "response:WriteString(tostring(#request.body))"}),
			script.WithFileSystem(afero.NewMemMapFs()),
		)

		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(recorder), request))
		assert.Equal(t, "10485760", recorder.Body.String())
	})

	t.Run("body over the limit", func(t *testing.T) {
		body := bytes.Repeat([]byte("a"), script.MaxRequestBodySize+1)
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", bytes.NewReader(body))

		err := serveWithTimeout(t, "response:WriteString('ok')", time.Minute, request)

		require.ErrorIs(t, err, script.ErrRequestBodyTooLarge)
	})

	t.Run("read failure", func(t *testing.T) {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", iotest.ErrReader(errBodyRead))

		err := serveWithTimeout(t, "response:WriteString('ok')", time.Minute, request)

		require.ErrorIs(t, err, script.ErrRequestBodyNotRead)
		require.ErrorIs(t, err, errBodyRead)
	})
}
