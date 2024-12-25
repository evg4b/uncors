package rewrite

import (
	"errors"

	"github.com/evg4b/uncors/internal/contracts"
)

var ErrInvalidHost = errors.New("rewrite host has invalid type")

type rewriteKeyType string

var rewriteHostKey rewriteKeyType = "__uncors_rewrite_host"

func GetRewriteHost(request *contracts.Request) (string, error) {
	value := request.Context().Value(rewriteHostKey)

	if value == nil {
		return "", nil
	}

	if host, ok := value.(string); ok {
		return host, nil
	}

	return "", ErrInvalidHost
}
