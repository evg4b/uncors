package rewrite

import (
	"errors"

	"github.com/evg4b/uncors/internal/contracts"
)

var ErrInvalidHost = errors.New("rewrite host has invalid type")

type rewriteKeyType string

var RewriteHostKey rewriteKeyType = "__uncors_rewrite_host"

// GetRewriteHost extracts the rewrite host from the request context.
// Returns ErrInvalidHost if the value exists but is not a string.
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

func IsRewriteRequest(request *contracts.Request) bool {
	return request.Context().Value(RewriteHostKey) != nil
}
