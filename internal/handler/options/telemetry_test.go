package options_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/options"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareLogs(t *testing.T) {
	recorded := testutils.Telemetry(t)
	handler := infra.Wrap(options.NewMiddleware(options.WithCode(http.StatusNoContent)),
		infra.HandlerFunc(func(contracts.ResponseWriter, *contracts.Request) error { return nil }))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "http://app.local")
	request.Header.Set("Access-Control-Request-Method", http.MethodPut)
	require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request))

	records := recorded.Records()
	require.Len(t, records, 1)
	assert.Equal(t, "answered preflight request", records[0].Message)
	assert.Equal(t, map[string]string{
		"origin":           "http://app.local",
		"requested_method": http.MethodPut,
		"status":           "204",
	}, records[0].Attrs)
}
