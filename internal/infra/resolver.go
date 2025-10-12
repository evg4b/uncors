package infra

import (
	"context"
)

// DNSResolver is a custom resolver interface for DNS lookups.
type DNSResolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}
