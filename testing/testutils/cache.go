package testutils

import "github.com/evg4b/uncors/internal/contracts"

func CachedHeader(name, value string) contracts.CachedHeader {
	return contracts.CachedHeader{
		Name:  name,
		Value: []string{value},
	}
}
