package router_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/di"
	"github.com/evg4b/uncors/internal/handler/router"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	spaIndexBody  = "SPA-INDEX"
	mockHealth    = "MOCK-HEALTH"
	upstreamBody  = "FROM-UPSTREAM"
	healthPath    = "/api/health"
	rewrittenPath = "/v2/api/health"
)

// harArchive is the slice of the HAR schema this test asserts on.
type harArchive struct {
	Log struct {
		Entries []struct {
			Request struct {
				URL string `json:"url"`
			} `json:"request"`
		} `json:"entries"`
	} `json:"log"`
}

func upstreamHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(upstreamBody))
	})
}

func newMappingRouter(t *testing.T, mapping config.Mapping, upstream http.Handler) *router.Router {
	t.Helper()

	container := di.NewContainer(di.WithFs(testutils.FsFromMap(t, map[string]string{
		"/dist/index.html": spaIndexBody,
	})))

	t.Cleanup(func() {
		require.NoError(t, container.Close())
	})

	instance, err := router.NewRouter(config.Mappings{mapping}, newDeps(t, container, upstream))
	require.NoError(t, err)

	return instance
}

func serve(t *testing.T, handler http.Handler, method, url string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), method, url, nil)
	infra.NormaliseRequest(request)

	handler.ServeHTTP(infra.NewResponseRecorder(recorder), request)

	return recorder
}

// The flagship "SPA with API proxying" configuration: a static mount at "/" used
// to shadow every mock in the same mapping.
func TestStaticMountDoesNotShadowMocks(t *testing.T) {
	instance := newMappingRouter(t, config.Mapping{
		From:    hosts.Localhost.HTTP(),
		To:      hosts.Localhost.HTTPS(),
		Statics: config.StaticDirectories{{Dir: "/dist", Path: "/", Index: "index.html"}},
		Mocks: config.Mocks{
			{
				Matcher:  config.RequestMatcher{Path: healthPath},
				Response: config.Response{Code: http.StatusOK, Raw: mockHealth},
			},
		},
	}, upstreamHandler())

	t.Run("the mock answers its own path", func(t *testing.T) {
		recorder := serve(t, instance, http.MethodGet, "http://localhost"+healthPath)

		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, mockHealth, recorder.Body.String())
	})

	t.Run("the static mount still answers everything else", func(t *testing.T) {
		recorder := serve(t, instance, http.MethodGet, "http://localhost/some/spa/route")

		assert.Equal(t, spaIndexBody, recorder.Body.String())
	})
}

// HAR is a property of the mapping, so locally produced responses belong in the
// archive too.
func TestHARRecordsLocallyServedResponses(t *testing.T) {
	harFile := filepath.Join(t.TempDir(), "test.har")

	instance := newMappingRouter(t, config.Mapping{
		From: hosts.Localhost.HTTP(),
		To:   hosts.Localhost.HTTPS(),
		HAR:  config.HARConfig{File: harFile},
		Mocks: config.Mocks{
			{
				Matcher:  config.RequestMatcher{Path: healthPath},
				Response: config.Response{Code: http.StatusOK, Raw: mockHealth},
			},
		},
	}, upstreamHandler())

	recorder := serve(t, instance, http.MethodGet, "http://localhost"+healthPath)
	require.Equal(t, mockHealth, recorder.Body.String())

	assert.Eventually(t, func() bool {
		content, err := os.ReadFile(harFile) //nolint:gosec // a path this test created
		if err != nil {
			return false
		}

		var archive harArchive

		if json.Unmarshal(content, &archive) != nil {
			return false
		}

		for _, entry := range archive.Log.Entries {
			if strings.HasSuffix(entry.Request.URL, healthPath) {
				return true
			}
		}

		return false
	}, 2*time.Second, 20*time.Millisecond, "the mocked request was not recorded")
}

// A preflight is answered before the router decides what serves the path, so a
// mocked path gets CORS handling like any other.
func TestOptionsIsAnsweredForMockedPaths(t *testing.T) {
	instance := newMappingRouter(t, config.Mapping{
		From: hosts.Localhost.HTTP(),
		To:   hosts.Localhost.HTTPS(),
		Mocks: config.Mocks{
			{
				Matcher:  config.RequestMatcher{Path: healthPath},
				Response: config.Response{Code: http.StatusOK, Raw: mockHealth},
			},
		},
	}, upstreamHandler())

	recorder := serve(t, instance, http.MethodOptions, "http://localhost"+healthPath)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEmpty(t, recorder.Header().Get(headers.AccessControlAllowMethods))
	assert.NotEqual(t, mockHealth, recorder.Body.String())
}

// A rewritten request re-enters routing, so it can be answered by a mock.
func TestRewriteRedispatchesIntoTheMapping(t *testing.T) {
	instance := newMappingRouter(t, config.Mapping{
		From:     hosts.Localhost.HTTP(),
		To:       hosts.Localhost.HTTPS(),
		Rewrites: config.RewriteOptions{{From: "/old-api/{path}", To: "/v2/api/{path}"}},
		Mocks: config.Mocks{
			{
				Matcher:  config.RequestMatcher{Path: rewrittenPath},
				Response: config.Response{Code: http.StatusOK, Raw: mockHealth},
			},
		},
	}, upstreamHandler())

	recorder := serve(t, instance, http.MethodGet, "http://localhost/old-api/health")

	assert.Equal(t, mockHealth, recorder.Body.String())
}

func TestRewriteLoopIsBounded(t *testing.T) {
	instance := newMappingRouter(t, config.Mapping{
		From: hosts.Localhost.HTTP(),
		To:   hosts.Localhost.HTTPS(),
		Rewrites: config.RewriteOptions{
			{From: "/ping", To: "/pong"},
			{From: "/pong", To: "/ping"},
		},
	}, upstreamHandler())

	recorder := serve(t, instance, http.MethodGet, "http://localhost/ping")

	assert.Equal(t, http.StatusLoopDetected, recorder.Code, "a rewrite loop must terminate")
}
