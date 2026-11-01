package rewrite

import (
	"github.com/evg4b/uncors/internal/contracts"
)

type rewriteKeyType string

const RewriteHostKey rewriteKeyType = "__uncors_rewrite_host"

func GetRewriteHost(request *contracts.Request) string {
	host, _ := request.Context().Value(RewriteHostKey).(string)

	return host
}
