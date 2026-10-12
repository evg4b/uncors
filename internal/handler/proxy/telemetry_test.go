package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
)

func TestProxyHandlerLogs(t *testing.T) {
	serve := func(t *testing.T, status int) []testutils.LogRecord {
		t.Helper()

		recorded := testutils.Telemetry(t)
		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(testutils.NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: status,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    req,
				}
			})),
			proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(config.Mappings{
				{From: hosts.Parse("http://premium.local.com"), To: hosts.Parse("https://premium.api.com")},
			})),
		)

		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://premium.local.com/api", nil)
		helpers.NormaliseRequest(request)
		require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request))

		records := recorded.Records()
		require.Len(t, records, 2)
		assert.Equal(t, "forwarding request upstream", records[0].Message)
		assert.Equal(t, "https://premium.api.com/api", records[0].Attrs["target"])

		return records
	}

	t.Run("upstream success", func(t *testing.T) {
		response := serve(t, http.StatusOK)[1]

		assert.Equal(t, log.SeverityDebug, response.Severity)
		assert.Equal(t, "upstream responded", response.Message)
		assert.Equal(t, "200", response.Attrs["status"])
	})

	t.Run("upstream server error", func(t *testing.T) {
		response := serve(t, http.StatusBadGateway)[1]

		assert.Equal(t, log.SeverityWarn, response.Severity)
		assert.Equal(t, "upstream responded with a server error", response.Message)
		assert.Equal(t, "502", response.Attrs["status"])
	})
}
