package router_test

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type countingMiddleware struct {
	calls int
}

func (m *countingMiddleware) ServeHTTP(
	writer contracts.ResponseWriter,
	request *contracts.Request,
	next contracts.Next,
) error {
	m.calls++

	return next(writer, request)
}

func TestRouterUpgradeSkipsCacheAndHAR(t *testing.T) {
	cacheMiddleware := &countingMiddleware{}
	routerInstance := newRouter(t, afero.NewMemMapFs(), config.Mappings{{
		From:  hosts.Parse("http://localhost"),
		To:    hosts.Parse("https://upstream.local"),
		Cache: config.CacheGlobs{"/**"},
		HAR:   config.HARConfig{File: filepath.Join(t.TempDir(), "test.har")},
	}}, upstream(t, http.StatusOK, "proxied"), router.ForRouterWithCacheMiddlewareFactory(
		func(config.CacheGlobs) contracts.Middleware { return cacheMiddleware },
	))

	upgrade := serve(t, routerInstance, http.MethodGet, "/socket", map[string]string{
		"Connection": "keep-alive, Upgrade",
		"Upgrade":    "websocket",
	})

	assert.Equal(t, "proxied", upgrade.Body.String())
	assert.Zero(t, cacheMiddleware.calls)

	regular := serve(t, routerInstance, http.MethodGet, "/socket", nil)

	assert.Equal(t, "proxied", regular.Body.String())
	assert.Equal(t, 1, cacheMiddleware.calls)
}
