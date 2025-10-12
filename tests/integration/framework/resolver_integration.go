//go:build integration_test

package framework

import (
	"context"
	"net"
	"strings"
)

// integrationTestDNSResolver is a custom DNS resolver for integration tests
// that redirects test domains to localhost.
type integrationTestDNSResolver struct {
	testDomains map[string]bool
}

// newTestDNSResolver creates a custom DNS resolver for integration tests.
// This version is compiled only when the integration_test build tag is set.
func newTestDNSResolver() testDNSResolver {
	return &integrationTestDNSResolver{
		testDomains: map[string]bool{
			// Test domains that should resolve to localhost
			"test.local":         true,
			"api.test.local":     true,
			"backend.test.local": true,
			"app.test.local":     true,
		},
	}
}

// LookupHost implements custom DNS resolution for integration tests.
// If the host matches a test domain, it returns localhost (127.0.0.1).
// Otherwise, it falls back to the default resolver.
func (r *integrationTestDNSResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	// Remove port if present
	hostWithoutPort := host
	if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
		hostWithoutPort = host[:colonIdx]
	}

	// Check if this is a test domain
	if r.testDomains[hostWithoutPort] {
		return []string{"127.0.0.1"}, nil
	}

	// Check for wildcard test domains (*.test.local)
	if strings.HasSuffix(hostWithoutPort, ".test.local") {
		return []string{"127.0.0.1"}, nil
	}

	// Fall back to default resolver for non-test domains
	return net.DefaultResolver.LookupHost(ctx, hostWithoutPort)
}
