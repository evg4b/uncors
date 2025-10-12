//go:build !integration_test

package infra

import (
	"context"
	"net"
)

// defaultResolver uses the system's default DNS resolver.
type defaultResolver struct{}

func (r *defaultResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

// newDNSResolver creates the default DNS resolver.
// This version is compiled when the integration_test build tag is NOT set.
func newDNSResolver() DNSResolver {
	return &defaultResolver{}
}
