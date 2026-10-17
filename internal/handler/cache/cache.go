package cache

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/samber/lo"
)

func CalcCost(value *contracts.CachedResponse) int64 {
	cost := len(value.Body)
	for _, header := range value.Headers {
		cost += len(header.Name) + lo.Reduce(header.Value, func(acc int, value string, _ int) int {
			return acc + len(value)
		}, 0)
	}

	for _, name := range value.Vary {
		cost += len(name)
	}

	return int64(cost)
}

const (
	numCounters = 1e5
	bufferItems = 64
)

type RistrettoCache struct {
	storage *ristretto.Cache[string, contracts.CachedResponse]
	ttl     time.Duration
}

func NewRistrettoCache(maxSize int64, ttl time.Duration) (*RistrettoCache, error) {
	storage, err := ristretto.NewCache(&ristretto.Config[string, contracts.CachedResponse]{
		NumCounters: numCounters,
		MaxCost:     maxSize,
		BufferItems: bufferItems,
	})
	if err != nil {
		return nil, err
	}

	return &RistrettoCache{
		storage: storage,
		ttl:     ttl,
	}, nil
}

func (cs *RistrettoCache) Get(key string) (contracts.CachedResponse, bool) {
	return cs.storage.Get(key)
}

func (cs *RistrettoCache) Set(key string, value contracts.CachedResponse) bool {
	cost := CalcCost(&value)
	if cost > cs.storage.MaxCost() {
		return false
	}

	return cs.storage.SetWithTTL(key, value, cost, cs.ttl)
}

func (cs *RistrettoCache) Wait() {
	cs.storage.Wait()
}

func (cs *RistrettoCache) Close() error {
	cs.storage.Close()

	return nil
}
