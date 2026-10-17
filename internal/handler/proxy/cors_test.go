package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/handler/proxy"
	"github.com/evg4b/uncors/internal/server"
	"github.com/evg4b/uncors/internal/urlreplacer"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testconstants"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyHandlerCors(t *testing.T) {
	origin := hosts.Localhost.HTTPPort(3000).String()

	serve := func(t *testing.T, request *http.Request, upstream http.Header) http.Header {
		t.Helper()

		handler := proxy.NewProxyHandler(
			proxy.WithHTTPClient(testutils.NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					Status:     "200 OK",
					StatusCode: http.StatusOK,
					Header:     upstream,
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    req,
				}
			})),
			proxy.WithURLReplacerFactory(lo.Must(urlreplacer.NewURLReplacerFactory(config.Mappings{
				{From: hosts.Parse("http://premium.local.com"), To: hosts.Parse("https://premium.api.com")},
			}))),
		)

		recorder := httptest.NewRecorder()

		err := handler.ServeHTTP(server.NewResponseRecorder(recorder), request)
		require.NoError(t, err)

		return testutils.ReadHeader(t, recorder)
	}

	t.Run("exposes the upstream headers to the origin", func(t *testing.T) {
		request := premiumRequest(t, http.MethodGet, nil)
		request.Header.Set(headers.Origin, origin)

		header := serve(t, request, http.Header{
			headers.ContentType:                []string{"application/json"},
			headers.ContentLength:              []string{"0"},
			headers.CacheControl:               []string{"no-cache"},
			headers.ETag:                       []string{`"v1"`},
			"X-Total-Count":                    []string{"10"},
			headers.SetCookie:                  []string{"session=1"},
			headers.AccessControlExposeHeaders: []string{"X-Upstream"},
			headers.AccessControlAllowHeaders:  []string{"X-Upstream"},
		})

		assert.Equal(t, origin, header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "true", header.Get(headers.AccessControlAllowCredentials))
		assert.Equal(t, testconstants.AllMethods, header.Get(headers.AccessControlAllowMethods))
		assert.Equal(t, "Etag, Vary, X-Total-Count", header.Get(headers.AccessControlExposeHeaders))
		assert.Empty(t, header.Values(headers.AccessControlAllowHeaders))
		assert.Equal(t, []string{headers.Origin}, header.Values(headers.Vary))
	})

	t.Run("keeps the upstream Vary", func(t *testing.T) {
		request := premiumRequest(t, http.MethodGet, nil)
		request.Header.Set(headers.Origin, origin)

		header := serve(t, request, http.Header{
			headers.Vary: []string{"Accept-Encoding"},
		})

		assert.Equal(t, []string{"Accept-Encoding", headers.Origin}, header.Values(headers.Vary))
	})

	t.Run("does not repeat Origin in the upstream Vary", func(t *testing.T) {
		request := premiumRequest(t, http.MethodGet, nil)
		request.Header.Set(headers.Origin, origin)

		header := serve(t, request, http.Header{
			headers.Vary: []string{"accept-encoding, origin"},
		})

		assert.Equal(t, []string{"accept-encoding, origin"}, header.Values(headers.Vary))
	})

	t.Run("exposes every header without an origin", func(t *testing.T) {
		request := premiumRequest(t, http.MethodGet, nil)

		header := serve(t, request, http.Header{
			"X-Total-Count": []string{"10"},
			headers.Vary:    []string{"Accept-Encoding"},
		})

		assert.Equal(t, "*", header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, "*", header.Get(headers.AccessControlExposeHeaders))
		assert.Equal(t, []string{"Accept-Encoding"}, header.Values(headers.Vary))
	})

	t.Run("answers a forwarded preflight with the requested headers", func(t *testing.T) {
		request := premiumRequest(t, http.MethodOptions, nil)
		request.Header.Set(headers.Origin, origin)
		request.Header.Set(headers.AccessControlRequestMethod, http.MethodPut)
		request.Header.Set(headers.AccessControlRequestHeaders, "X-Api-Key")

		header := serve(t, request, http.Header{})

		assert.Equal(t, origin, header.Get(headers.AccessControlAllowOrigin))
		assert.Equal(t, http.MethodPut, header.Get(headers.AccessControlAllowMethods))
		assert.Equal(t, "X-Api-Key", header.Get(headers.AccessControlAllowHeaders))
		assert.Equal(t, []string{
			headers.Origin,
			headers.AccessControlRequestMethod,
			headers.AccessControlRequestHeaders,
		}, header.Values(headers.Vary))
	})
}
