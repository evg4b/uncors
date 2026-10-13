package rewrite_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/rewrite"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareLogs(t *testing.T) {
	recorded := testutils.Telemetry(t)
	middleware := rewrite.NewMiddleware(rewrite.WithRewritingOptions(&config.RewritingOption{
		From: "/original",
		To:   "/rewritten",
		Host: urlt.Host{Hostname: "example.com"},
	}))
	handler := infra.Mddleware(middleware, infra.HandlerFunc(func(contracts.ResponseWriter, *contracts.Request) error {
		return nil
	}))

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/original", nil)
	require.NoError(t, handler.ServeHTTP(server.NewResponseRecorder(httptest.NewRecorder()), request))

	records := recorded.Records()
	require.Len(t, records, 1)
	assert.Equal(t, "rewrote request", records[0].Message)
	assert.Equal(t, map[string]string{
		"pattern": "/original",
		"from":    "/original",
		"to":      "example.com/rewritten",
	}, records[0].Attrs)
}
