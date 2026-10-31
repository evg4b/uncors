package infra

import (
	"net/http"
	"strings"

	"github.com/evg4b/uncors/internal/headers"
)

func IsUpgrade(header http.Header) bool {
	if header.Get(headers.Upgrade) == "" {
		return false
	}

	for _, value := range header.Values("Connection") {
		for token := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(token), "upgrade") {
				return true
			}
		}
	}

	return false
}
