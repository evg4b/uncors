package rewrite

import (
	"errors"

	"github.com/evg4b/uncors/internal/contracts"
)

var ErrInvalidHost = errors.New("rewrite host has invalid type")

type rewriteKeyType string

// RewriteHostKey is a key used to store host rewriting configuration in the request context.
var RewriteHostKey rewriteKeyType = "__uncors_rewrite_host"

// GetRewriteHost extracts the rewrite host value from the request context.
// It returns the host string and any error encountered during extraction.
// If no host value is found in the context, it returns an empty string and nil error.
// If the value exists but is not a string, it returns ErrInvalidHost.
func GetRewriteHost(request *contracts.Request) (string, error) {
	value := request.Context().Value(RewriteHostKey)

	if value == nil {
		return "", nil
	}

	if host, ok := value.(string); ok {
		return host, nil
	}

	return "", ErrInvalidHost
}

// IsRewriteRequest checks if the provided request contains a rewrite host in its context.
// It returns true if the request has a rewrite host value set under RewriteHostKey,
// false otherwise.
func IsRewriteRequest(request *contracts.Request) bool {
	return request.Context().Value(RewriteHostKey) != nil
}
