package cache

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/evg4b/uncors/internal/contracts"
)

func CalcCost(value *contracts.CachedResponse) int64 {
	cost := len(value.Body)
	for _, header := range value.Headers {
		cost += len(header.Name) + len(header.Value)
	}

	return int64(cost)
}

const (
	numCounters = 1e5
	bufferItems = 64
	ttl         = 10 * time.Minute
)

type RistrettoCache struct {
	storage *ristretto.Cache[string, contracts.CachedResponse]
	ttl     time.Duration
}

func NewCache(maxBytes int64) *RistrettoCache {
	storage, err := ristretto.NewCache(&ristretto.Config[string, contracts.CachedResponse]{
		NumCounters: numCounters,
		MaxCost:     maxBytes,
		BufferItems: bufferItems,
	})
	if err != nil {
		panic(err)
	}

	return &RistrettoCache{
		storage: storage,
		ttl:     ttl,
	}
}

func (cs *RistrettoCache) Get(key string) (contracts.CachedResponse, bool) {
	return cs.storage.Get(key)
}

func (cs *RistrettoCache) Set(key string, value *contracts.CachedResponse) {
	cs.storage.SetWithTTL(key, *value, CalcCost(value), cs.ttl)
}
