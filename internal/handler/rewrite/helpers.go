package rewrite

import (
	"errors"
	"net/http"
)

var ErrInvalidHost = errors.New("rewrite host has invalid type")

type rewriteKeyType string

const RewriteHostKey rewriteKeyType = "__uncors_rewrite_host"

func GetRewriteHost(request *http.Request) (string, error) {
	value := request.Context().Value(RewriteHostKey)

	if value == nil {
		return "", nil
	}

	if host, ok := value.(string); ok {
		return host, nil
	}

	return "", ErrInvalidHost
}

func IsRewriteRequest(request *http.Request) bool {
	return request.Context().Value(RewriteHostKey) != nil
}
