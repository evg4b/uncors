package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/engine"
	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type stubClient struct{}

func (stubClient) Do(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("ok")),
	}, nil
}

// BenchmarkRequestDisabled serves a proxied request through the router with
// tracing off: its allocations are the cost the instrumentation must not add.
func BenchmarkRequestDisabled(b *testing.B) {
	mappings := config.Mappings{{From: hosts.Localhost.HTTP(), To: hosts.Github.HTTPS()}}

	generation, err := engine.New(engine.WithFs(afero.NewMemMapFs())).BuildGeneration(&config.UncorsConfig{})
	require.NoError(b, err)

	b.Cleanup(func() { _ = generation.Close() })

	muxRouter, err := router.NewRouter(mappings,
		router.WithFactory(generation),
		router.ForRouterWithDefaultHandler(proxy.NewProxyHandler(
			proxy.WithURLReplacerFactory(urlreplacer.NewURLReplacerFactory(mappings)),
			proxy.WithHTTPClient(stubClient{}),
		)),
	)
	require.NoError(b, err)

	handler := infra.CastToContractsHandler(muxRouter)
	instance := server.New(nil, events.NoopPublisher{})

	b.ReportAllocs()

	for b.Loop() {
		request := httptest.NewRequestWithContext(b.Context(), http.MethodGet, "http://localhost/api/users?page=1", nil)
		server.HandleRequest(instance, handler, httptest.NewRecorder(), request)
	}
}
