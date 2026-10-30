package engine

import "github.com/evg4b/uncors/internal/contracts"

func (g *Generation) CacheStore() contracts.Cache {
	return g.cacheStore
}
