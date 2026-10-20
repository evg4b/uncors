//go:build integration

package proxy_test

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type backend struct {
	*integration.Backend

	base string
	host string
}

func newMuxBackend(t *testing.T) *backend {
	t.Helper()

	self := &backend{}
	self.Backend = integration.NewBackend(t, backendMux(t, self).ServeHTTP)
	self.base = self.URL()

	parsed, err := url.Parse(self.base)
	require.NoError(t, err)

	self.host = parsed.Hostname()

	return self
}

func backendMux(t *testing.T, self *backend) http.Handler {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/echo", func(writer http.ResponseWriter, request *http.Request) {
		_, err := io.WriteString(writer, "echo:"+request.URL.Path)
		assert.NoError(t, err)
	})
	mux.HandleFunc("/data", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Backend", "served")
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_, err := io.WriteString(writer, `{"id":1}`)
		assert.NoError(t, err)
	})
	mux.HandleFunc("/sized", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", "1234")

		if request.Method != http.MethodHead {
			_, err := io.WriteString(writer, strings.Repeat("x", 1234))
			assert.NoError(t, err)
		}
	})
	mux.HandleFunc("/hop", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Connection", "X-Hop")
		writer.Header().Set("X-Hop", "1")
		writer.Header().Set("Keep-Alive", "timeout=5")
		_, err := io.WriteString(writer, "hop")
		assert.NoError(t, err)
	})
	mux.HandleFunc("/redirect", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", self.base+"/target")
		writer.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/set-cookie", func(writer http.ResponseWriter, _ *http.Request) {
		http.SetCookie(writer, &http.Cookie{Name: "sid", Value: "abc", Domain: self.host, Path: "/"})
		_, err := io.WriteString(writer, "cookie set")
		assert.NoError(t, err)
	})
	mux.HandleFunc("/read-cookie", func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie("token")
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, writeErr := io.WriteString(writer, "missing")
			assert.NoError(t, writeErr)

			return
		}

		_, writeErr := io.WriteString(writer, "token="+cookie.Value)
		assert.NoError(t, writeErr)
	})
	mux.HandleFunc("/", func(writer http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(writer, "ok")
		assert.NoError(t, err)
	})

	return mux
}

func newProxyEnv(t *testing.T) (*integration.Env, *backend) {
	t.Helper()

	back := newMuxBackend(t)
	env := integration.New(t, back.Backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://app.example.local"),
			To:   back.AsHost(),
		}},
	})

	return env, back
}

func proxyURL(env *integration.Env, path string) string {
	return env.URL("app.example.local", path)
}

func TestProxyHandler(t *testing.T) {
	env, back := newProxyEnv(t)
	backendURL := back.base

	t.Run("forwards method, path and query verbatim to the backend", func(t *testing.T) {
		// Append the query directly: the path joiner would percent-encode "?".
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo")+"?limit=10&sort=asc"))

		require.Equal(t, http.StatusOK, result.Response.StatusCode)
		require.True(t, result.HasBackendRequest())

		dump := result.BackendRequest(t)
		assert.True(t, strings.HasPrefix(dump, "GET /echo?limit=10&sort=asc HTTP/1.1"),
			"request line must preserve path and query, got:\n%s", dump)
	})

	t.Run("forwards the request body", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodPost, proxyURL(env, "/echo"))
		req.Body = io.NopCloser(strings.NewReader(`{"payload":true}`))
		req.ContentLength = int64(len(`{"payload":true}`))

		result := env.Do(t, req)

		assert.Contains(t, result.BackendRequest(t), `{"payload":true}`)
	})

	t.Run("forwards the request Content-Length instead of chunking the body", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodPost, proxyURL(env, "/echo"))
		req.Body = io.NopCloser(strings.NewReader("hello"))
		req.ContentLength = 5

		dump := env.Do(t, req).RawBackendRequest(t)

		assert.Contains(t, dump, "Content-Length: 5\r\n")
		assert.NotContains(t, dump, "Transfer-Encoding")
	})

	t.Run("forwards custom request headers", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("X-Trace-Id", "trace-123")

		result := env.Do(t, req)

		assert.Contains(t, result.BackendRequest(t), "X-Trace-Id: trace-123")
	})

	t.Run("returns the backend status, headers and body unchanged", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/data")))

		assert.Equal(t, http.StatusCreated, result.Response.StatusCode)
		assert.Equal(t, "served", result.Response.Header.Get("X-Backend"))
		assert.JSONEq(t, `{"id":1}`, result.BodyString())
	})

	t.Run("adds CORS headers reflecting the request Origin", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("Origin", "https://app.example.local")

		result := env.Do(t, req)

		assert.Equal(t, "https://app.example.local",
			result.Response.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", result.Response.Header.Get("Access-Control-Allow-Credentials"))
	})

	t.Run("rewrites the Origin request header to the target host", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("Origin", "https://app.example.local")

		result := env.Do(t, req)

		assert.Contains(t, result.RawBackendRequest(t), "Origin: "+backendURL)
	})

	t.Run("forwards an Origin of another host unchanged", func(t *testing.T) {
		for _, origin := range []string{"http://a.test", "http://127.0.0.1:3000", "null"} {
			req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
			req.Header.Set("Origin", origin)

			result := env.Do(t, req)

			require.Equal(t, http.StatusOK, result.Response.StatusCode, "origin %q", origin)
			assert.Contains(t, result.RawBackendRequest(t), "Origin: "+origin+"\r\n")
		}
	})

	t.Run("rewrites the Location response header back to the source host", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/redirect")))

		assert.Equal(t, http.StatusFound, result.Response.StatusCode)

		expected := fmt.Sprintf("https://app.example.local:%d/target", env.PortFor("app.example.local"))
		assert.Equal(t, expected, result.Response.Header.Get("Location"))
	})

	t.Run("rewrites the Referer request header to the target host", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("Referer", "https://app.example.local/from/page")

		result := env.Do(t, req)

		assert.Contains(t, result.RawBackendRequest(t), "Referer: "+backendURL+"/from/page")
	})

	t.Run("forwards request cookies to the backend", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/read-cookie"))
		req.AddCookie(&http.Cookie{Name: "token", Value: "xyz"})

		result := env.Do(t, req)

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "token=xyz", result.BodyString())
	})

	t.Run("forwards Set-Cookie and marks it Secure, rewriting the backend domain away", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/set-cookie")))

		cookies := result.Response.Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "sid", cookies[0].Name)
		assert.Equal(t, "abc", cookies[0].Value)
		assert.True(t, cookies[0].Secure)
		assert.NotContains(t, cookies[0].Domain, "127.0.0.1")
	})

	t.Run("forwarded request and response both match snapshot", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo")))

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}

func TestProxyOverHTTP(t *testing.T) {
	back := newMuxBackend(t)
	env := integration.New(t, back.Backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("http://plain.local"),
			To:   back.AsHost(),
		}},
	})

	t.Run("forwards over plain HTTP without TLS", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("plain.local", "/echo")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "echo:/echo", result.BodyString())
		assert.Nil(t, result.Response.TLS, "plain HTTP response must not carry TLS state")
		assert.True(t, result.HasBackendRequest())

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}

func TestProxyMultipleMappings(t *testing.T) {
	alpha := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "alpha")
		assert.NoError(t, err)
	})
	beta := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "beta")
		assert.NoError(t, err)
	})

	// Both From hosts deliberately share the same port (no explicit port means
	// the harness assigns one per host; reuse it so they land on one listener).
	port := strconv.Itoa(testutils.GetFreePort(t))
	env := integration.New(t, alpha, &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Parse("https://alpha.local:" + port), To: alpha.AsHost()},
			{From: hosts.Parse("https://beta.local:" + port), To: beta.AsHost()},
		},
	})

	t.Run("each host is routed to its own backend", func(t *testing.T) {
		alphaResult := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("alpha.local", "/")))
		defer testutils.Close(t, alphaResult.Response.Body)

		betaResult := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("beta.local", "/")))
		defer testutils.Close(t, betaResult.Response.Body)

		assert.Equal(t, "alpha", alphaResult.BodyString())
		assert.Equal(t, "beta", betaResult.BodyString())

		assert.Equal(t, 1, alpha.Count())
		assert.Equal(t, 1, beta.Count())
	})
}

func TestProxyPlaceholderMapping(t *testing.T) {
	back := newMuxBackend(t)
	env := integration.New(t, back.Backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://{tenant}.api.local"),
			To:   back.AsHost(),
		}},
	})

	tenants := []string{"acme", "globex", "initech"}

	t.Run("every subdomain is matched and forwarded with the path preserved", func(t *testing.T) {
		for _, tenant := range tenants {
			result := env.Do(t, integration.NewRequest(t, http.MethodGet,
				env.URL(tenant+".api.local", "/echo")))

			assert.Equal(t, http.StatusOK, result.Response.StatusCode)
			assert.Equal(t, "echo:/echo", result.BodyString())
			require.True(t, result.HasBackendRequest(), "tenant %q must reach the backend", tenant)

			require.NotNil(t, result.Response.TLS)
			require.NotEmpty(t, result.Response.TLS.PeerCertificates)
			assert.Contains(t, result.Response.TLS.PeerCertificates[0].DNSNames, tenant+".api.local")
		}

		assert.Equal(t, len(tenants), back.Count())
	})
}
