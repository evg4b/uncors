//go:build integration

package domains_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/evg4b/uncors/tests/integration/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func get(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)

	response, err := client.Do(request)
	require.NoError(t, err)

	return response
}

// TestLocalDomainMapping maps a real local domain to the backend and reaches it
// purely through the in-memory host resolver (no /etc/hosts, no real DNS).
func TestLocalDomainMapping(t *testing.T) {
	env := harness.New(t,
		harness.WithBackendHandler(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(writer, "from backend")
		}),
		harness.WithDomain("api.example.local"),
	)
	route := env.Route(t, "api.example.local")

	t.Run("exact local domain routes to backend over TLS", func(t *testing.T) {
		env.Backend.Reset()

		response := get(t, env.Client, route.URL("", "/users"))
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		require.NoError(t, err)

		// A 200 proves the Host header matched the mapping (an unmapped host
		// would return the router's "host not mapped" error instead).
		assert.Equal(t, http.StatusOK, response.StatusCode)
		assert.Equal(t, "from backend", string(body))

		// SNI carried the domain, so the proxy minted a leaf for it.
		require.NotNil(t, response.TLS)
		require.NotEmpty(t, response.TLS.PeerCertificates)
		assert.Contains(t, response.TLS.PeerCertificates[0].DNSNames, "api.example.local")

		// The backend received exactly the forwarded request.
		requests := env.Backend.Requests()
		require.Len(t, requests, 1)
		assert.True(t, strings.HasPrefix(requests[0], "GET /users HTTP/1.1"))
	})

	t.Run("domain absent from the resolver is unreachable", func(t *testing.T) {
		// Not registered via WithDomain and not in real DNS (.invalid never
		// resolves), so the connection cannot be established.
		url := fmt.Sprintf("https://unmapped.invalid:%d/", route.Port())

		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		require.NoError(t, err)

		response, err := env.Client.Do(request) //nolint:bodyclose // call fails before a body exists
		require.Error(t, err)
		assert.Nil(t, response)
	})
}

// TestPlaceholderDomainMapping maps a {placeholder} domain so a single mapping
// serves any subdomain, each resolved in-memory to the loopback proxy.
func TestPlaceholderDomainMapping(t *testing.T) {
	env := harness.New(t,
		harness.WithBackendHandler(func(writer http.ResponseWriter, request *http.Request) {
			// Echo the path so the test can confirm the proxied target. The path
			// is test-controlled, not attacker input (gosec G705 false positive).
			_, _ = io.WriteString(writer, "served "+request.URL.Path) //nolint:gosec
		}),
		harness.WithDomain("{tenant}.example.local"),
	)
	route := env.Route(t, "{tenant}.example.local")

	tenants := []string{"acme", "globex", "initech"}

	t.Run("every subdomain routes through one placeholder mapping", func(t *testing.T) {
		env.Backend.Reset()

		for _, tenant := range tenants {
			response := get(t, env.Client, route.URL(tenant, "/dashboard"))

			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			_ = response.Body.Close()

			assert.Equal(t, http.StatusOK, response.StatusCode)
			assert.Equal(t, "served /dashboard", string(body))

			// The placeholder label flowed through SNI into a per-host leaf.
			require.NotNil(t, response.TLS)
			require.NotEmpty(t, response.TLS.PeerCertificates)
			assert.Contains(t, response.TLS.PeerCertificates[0].DNSNames, tenant+".example.local")
		}

		// One request per tenant reached the backend; no duplicates.
		assert.Equal(t, len(tenants), env.Backend.Count())
	})
}
