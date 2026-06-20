//go:build integration

package domains_test

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/testing/hosts"
	"github.com/evg4b/uncors/testing/integration"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/gkampitakis/go-snaps/snaps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLocalDomainMapping maps a real local domain to the backend and reaches it
// purely through the in-memory host resolver (no /etc/hosts, no real DNS).
func TestLocalDomainMapping(t *testing.T) {
	backend := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "from backend")
		assert.NoError(t, err)
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://api.example.local"),
			To:   backend.AsHost(),
		}},
	})

	t.Run("exact local domain routes to backend over TLS", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet,
			env.URL("api.example.local", "/users")))
		defer result.Response.Body.Close()

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "from backend", result.BodyString())

		// SNI carried the domain, so the proxy minted a leaf for api.example.local.
		require.NotNil(t, result.Response.TLS)
		require.NotEmpty(t, result.Response.TLS.PeerCertificates)
		assert.Contains(t, result.Response.TLS.PeerCertificates[0].DNSNames, "api.example.local")

		assert.True(t, result.HasBackendRequest())

		snaps.MatchSnapshot(t, result.BackendRequest(t))
		snaps.MatchSnapshot(t, result.ResponseDump(t))
	})

	t.Run("domain absent from the resolver is unreachable", func(t *testing.T) {
		// Not registered via the harness and not in real DNS (.invalid never resolves),
		// so the connection cannot be established.
		url := fmt.Sprintf("https://unmapped.invalid:%d/", env.PortFor("api.example.local"))
		req := integration.NewRequest(t, http.MethodGet, url)

		resp, err := env.Client.Do(req) //nolint:bodyclose
		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

// TestPlaceholderDomainMapping maps a {placeholder} domain so a single mapping
// serves any subdomain, each resolved in-memory to the loopback proxy.
func TestPlaceholderDomainMapping(t *testing.T) {
	backend := integration.NewBackend(t, func(w http.ResponseWriter, r *http.Request) {
		_, err := io.WriteString(w, "served "+r.URL.Path) //nolint:gosec
		assert.NoError(t, err)
	})
	env := integration.New(t, backend, &config.UncorsConfig{
		Mappings: config.Mappings{{
			From: hosts.Parse("https://{tenant}.example.local"),
			To:   backend.AsHost(),
		}},
	})

	tenants := []string{"acme", "globex", "initech"}

	t.Run("every subdomain routes through one placeholder mapping", func(t *testing.T) {
		for _, tenant := range tenants {
			result := env.Do(t, integration.NewRequest(t, http.MethodGet,
				env.URL(tenant+".example.local", "/dashboard")))

			body, err := io.ReadAll(result.Response.Body)
			require.NoError(t, err)
			result.Response.Body.Close()

			assert.Equal(t, http.StatusOK, result.Response.StatusCode)
			assert.Equal(t, "served /dashboard", string(body))

			// The placeholder label flowed through SNI into a per-host leaf.
			require.NotNil(t, result.Response.TLS)
			require.NotEmpty(t, result.Response.TLS.PeerCertificates)
			assert.Contains(t, result.Response.TLS.PeerCertificates[0].DNSNames, tenant+".example.local")
		}

		assert.Equal(t, len(tenants), backend.Count())
	})
}

// TestSharedPortDistinctRemotes maps two different local domains onto a single
// shared local listener port, each forwarding to a different remote host. It
// proves the proxy dispatches purely by the Host header: every request reaches
// only its own remote, and the forwarded request carries that remote's address.
func TestSharedPortDistinctRemotes(t *testing.T) {
	shopRemote := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "shop remote")
		assert.NoError(t, err)
	})
	blogRemote := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "blog remote")
		assert.NoError(t, err)
	})

	// One local port shared by both domains; an explicit port is respected by the
	// harness, so the two mappings land on the same listener.
	port := strconv.Itoa(testutils.GetFreePort(t))
	env := integration.New(t, shopRemote, &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Parse("https://shop.local:" + port), To: shopRemote.AsHost()},
			{From: hosts.Parse("https://blog.local:" + port), To: blogRemote.AsHost()},
		},
	})

	// Both domains resolve to the same local listener port.
	require.Equal(t, env.PortFor("shop.local"), env.PortFor("blog.local"))

	t.Run("shop.local reaches only the shop remote", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("shop.local", "/catalog")))
		defer result.Response.Body.Close()

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "shop remote", result.BodyString())
		assert.Equal(t, 1, shopRemote.Count())
		assert.Equal(t, 0, blogRemote.Count())

		// The forwarded request carries the shop remote's own host:port.
		snaps.MatchSnapshot(t, result.BackendRequest(t))
	})

	t.Run("blog.local reaches only the blog remote", func(t *testing.T) {
		shopBefore := shopRemote.Count()

		response, err := env.Client.Do(integration.NewRequest(t, http.MethodGet, env.URL("blog.local", "/posts")))
		require.NoError(t, err)

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "blog remote", string(body))
		assert.Equal(t, 1, blogRemote.Count())
		assert.Equal(t, shopBefore, shopRemote.Count(), "blog request must not reach the shop remote")
	})
}
