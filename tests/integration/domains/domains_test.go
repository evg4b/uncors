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

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "from backend", result.BodyString())

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

			body := result.BodyString()

			assert.Equal(t, http.StatusOK, result.Response.StatusCode)
			assert.Equal(t, "served /dashboard", body)

			require.NotNil(t, result.Response.TLS)
			require.NotEmpty(t, result.Response.TLS.PeerCertificates)
			assert.Contains(t, result.Response.TLS.PeerCertificates[0].DNSNames, tenant+".example.local")
		}

		assert.Equal(t, len(tenants), backend.Count())
	})
}

func TestSharedPortDistinctRemotes(t *testing.T) {
	shopRemote := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "shop remote")
		assert.NoError(t, err)
	})
	blogRemote := integration.NewBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "blog remote")
		assert.NoError(t, err)
	})

	port := strconv.Itoa(testutils.GetFreePort(t))
	env := integration.New(t, shopRemote, &config.UncorsConfig{
		Mappings: config.Mappings{
			{From: hosts.Parse("https://shop.local:" + port), To: shopRemote.AsHost()},
			{From: hosts.Parse("https://blog.local:" + port), To: blogRemote.AsHost()},
		},
	})

	require.Equal(t, env.PortFor("shop.local"), env.PortFor("blog.local"))

	t.Run("shop.local reaches only the shop remote", func(t *testing.T) {
		result := env.Do(t, integration.NewRequest(t, http.MethodGet, env.URL("shop.local", "/catalog")))

		assert.Equal(t, http.StatusOK, result.Response.StatusCode)
		assert.Equal(t, "shop remote", result.BodyString())
		assert.Equal(t, 1, shopRemote.Count())
		assert.Equal(t, 0, blogRemote.Count())

		snaps.MatchSnapshot(t, result.BackendRequest(t))
	})

	t.Run("blog.local reaches only the blog remote", func(t *testing.T) {
		shopBefore := shopRemote.Count()

		response, err := env.Client.Do(integration.NewRequest(t, http.MethodGet, env.URL("blog.local", "/posts")))
		require.NoError(t, err)

		defer testutils.Close(t, response.Body)

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "blog remote", string(body))
		assert.Equal(t, 1, blogRemote.Count())
		assert.Equal(t, shopBefore, shopRemote.Count(), "blog request must not reach the shop remote")
	})
}
