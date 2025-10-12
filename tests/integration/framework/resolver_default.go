//go:build !integration_test

package framework

import (
	"context"
	"net"
)

// defaultTestResolver uses the system's default DNS resolver.
type defaultTestResolver struct{}

func (r *defaultTestResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

// newTestDNSResolver creates the default DNS resolver.
// This version is compiled when the integration_test build tag is NOT set.
func newTestDNSResolver() testDNSResolver {
	return &defaultTestResolver{}
}
