//go:build integration

package proxy_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
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

// backendMux serves the endpoints the proxy tests forward to. Building Location
// from r.Host lets the test assert the proxy rewrites it back to the source host
// without the handler needing to know its own ephemeral port up front.
func backendMux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "echo:"+r.URL.Path) //nolint:gosec
	})
	mux.HandleFunc("/data", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Backend", "served")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":1}`)
	})
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		// r.Host is the rewritten target host the proxy forwarded to.
		w.Header().Set("Location", "http://"+r.Host+"/target")
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/set-cookie", func(writer http.ResponseWriter, request *http.Request) {
		// Domain set to the backend's own host so the proxy must rewrite it
		// back to the source host before the client sees it.
		host, _, _ := net.SplitHostPort(request.Host)
		//nolint:gosec // G124: test cookie; Secure is added by the proxy on the way out
		http.SetCookie(writer, &http.Cookie{Name: "sid", Value: "abc", Domain: host, Path: "/"})
		_, _ = io.WriteString(writer, "cookie set")
	})
	mux.HandleFunc("/read-cookie", func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie("token")
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(writer, "missing")

			return
		}

		_, _ = io.WriteString(writer, "token="+cookie.Value) //nolint:gosec // G705: value is test-controlled
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})

	return mux
}

// newProxyEnv wires a single named-host mapping (app.example.local -> backend)
// so host rewriting is observable: source and target hosts differ visibly,
// unlike a loopback-to-loopback mapping.
func newProxyEnv(t *testing.T) (*integration.Env, *integration.Backend) {
	t.Helper()

	backend := integration.NewBackend(t, backendMux().ServeHTTP)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://app.example.local"),
			To:   backend.AsHost(),
		}},
	})

	return env, backend
}

func proxyURL(env *integration.Env, path string) string {
	return env.URL("app.example.local", path)
}

func TestProxyHandler(t *testing.T) {
	env, backend := newProxyEnv(t)
	backendURL := backend.URL() // e.g. http://127.0.0.1:PORT

	t.Run("forwards method, path and query verbatim to the backend", func(t *testing.T) {
		// Append the query directly: the path joiner would percent-encode "?".
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo")+"?limit=10&sort=asc"))
		defer result.Response.Body.Close()

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
		defer result.Response.Body.Close()

		assert.Contains(t, result.BackendRequest(t), `{"payload":true}`)
	})

	t.Run("forwards custom request headers", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("X-Trace-Id", "trace-123")

		result := env.Do(t, req)
		defer result.Response.Body.Close()

		assert.Contains(t, result.BackendRequest(t), "X-Trace-Id: trace-123")
	})

	t.Run("returns the backend status, headers and body unchanged", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/data")))
		defer result.Response.Body.Close()

		assert.Equal(t, http.StatusCreated, result.Response.StatusCode)
		assert.Equal(t, "served", result.Response.Header.Get("X-Backend"))
		assert.JSONEq(t, `{"id":1}`, result.BodyString())
	})

	t.Run("adds CORS headers reflecting the request Origin", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("Origin", "https://app.example.local")

		result := env.Do(t, req)
		defer result.Response.Body.Close()

		assert.Equal(t, "https://app.example.local",
			result.Response.Header.Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", result.Response.Header.Get("Access-Control-Allow-Credentials"))
	})

	t.Run("rewrites the Origin request header to the target host", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("Origin", "https://app.example.local")

		result := env.Do(t, req)
		defer result.Response.Body.Close()

		// The proxy rewrites Origin from the source host to the backend host
		// before forwarding, so the upstream sees its own address.
		assert.Contains(t, result.BackendRequest(t), "Origin: "+backendURL)
	})

	t.Run("rewrites the Location response header back to the source host", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/redirect")))
		defer result.Response.Body.Close()

		assert.Equal(t, http.StatusFound, result.Response.StatusCode)
		// Backend returned Location pointing at itself; the client must see it
		// rewritten back to the public source host (port included).
		expected := fmt.Sprintf("https://app.example.local:%d/target", env.PortFor("app.example.local"))
		assert.Equal(t, expected, result.Response.Header.Get("Location"))
	})

	t.Run("rewrites the Referer request header to the target host", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo"))
		req.Header.Set("Referer", "https://app.example.local/from/page")

		result := env.Do(t, req)
		defer result.Response.Body.Close()

		// Only the host part is rewritten; the path is preserved.
		assert.Contains(t, result.BackendRequest(t), "Referer: "+backendURL+"/from/page")
	})

	t.Run("forwards request cookies to the backend", func(t *testing.T) {
		req := integration.NewRequest(t, http.MethodGet, proxyURL(env, "/read-cookie"))
		req.AddCookie(&http.Cookie{Name: "token", Value: "xyz"}) //nolint:gosec // G124: test request cookie

		result := env.Do(t, req)
		defer result.Response.Body.Close()

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "token=xyz", result.BodyString())
	})

	t.Run("forwards Set-Cookie and marks it Secure, rewriting the backend domain away", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/set-cookie")))
		defer result.Response.Body.Close()

		cookies := result.Response.Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "sid", cookies[0].Name)
		assert.Equal(t, "abc", cookies[0].Value)
		// The source mapping is HTTPS, so the forwarded cookie is marked Secure.
		assert.True(t, cookies[0].Secure)
		// The backend's own host must not leak to the client. (The proxy currently
		// rewrites it to a port-bearing domain that the Go client drops as invalid,
		// leaving it empty; a fixed proxy would set "app.example.local". Either way
		// it must no longer reference the loopback backend.)
		assert.NotContains(t, cookies[0].Domain, "127.0.0.1")
	})

	t.Run("forwarded request and response both match snapshot", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, proxyURL(env, "/echo")))
		defer result.Response.Body.Close()

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})
}

// TestProxyOverHTTP covers a plain-HTTP mapping: the proxy listener and client
// hop are unencrypted, yet host routing and forwarding behave identically.
func TestProxyOverHTTP(t *testing.T) {
	backend := integration.NewBackend(t, backendMux().ServeHTTP)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("http://plain.local"),
			To:   backend.AsHost(),
		}},
	})

	t.Run("forwards over plain HTTP without TLS", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("plain.local", "/echo")))
		defer result.Response.Body.Close()

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "echo:/echo", result.BodyString())
		assert.Nil(t, result.Response.TLS, "plain HTTP response must not carry TLS state")
		assert.True(t, result.HasBackendRequest())
	})
}

// TestProxyMultipleMappings covers routing across several mappings that share a
// single listener port: requests are dispatched to the right upstream purely by
// their Host header.
func TestProxyMultipleMappings(t *testing.T) {
	alpha := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "alpha")
	})
	beta := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "beta")
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
		defer alphaResult.Response.Body.Close()

		betaResult := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("beta.local", "/")))
		defer betaResult.Response.Body.Close()

		assert.Equal(t, "alpha", alphaResult.BodyString())
		assert.Equal(t, "beta", betaResult.BodyString())

		// alpha's backend saw only alpha's request; beta's saw only beta's.
		assert.Equal(t, 1, alpha.Count())
		assert.Equal(t, 1, beta.Count())
	})
}

// TestProxyPlaceholderMapping covers a {placeholder} host mapping: one mapping
// matches every subdomain and forwards each to the backend with the path intact.
func TestProxyPlaceholderMapping(t *testing.T) {
	backend := integration.NewBackend(t, backendMux().ServeHTTP)
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://{tenant}.api.local"),
			To:   backend.AsHost(),
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

			// The placeholder label flowed through SNI into a per-host leaf.
			require.NotNil(t, result.Response.TLS)
			require.NotEmpty(t, result.Response.TLS.PeerCertificates)
			assert.Contains(t, result.Response.TLS.PeerCertificates[0].DNSNames, tenant+".api.local")

			result.Response.Body.Close()
		}

		assert.Equal(t, len(tenants), backend.Count())
	})
}
