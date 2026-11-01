package infra

import (
	"net/http"
	"net/textproto"
	"strings"

	"github.com/evg4b/uncors/internal/headers"
)

func IsUpgrade(header http.Header) bool {
	return header.Get(headers.Upgrade) != "" && HasToken(header.Values("Connection"), "upgrade")
}

func HasToken(values []string, token string) bool {
	for _, value := range values {
		for item := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(textproto.TrimString(item), token) {
				return true
			}
		}
	}

	return false
}
