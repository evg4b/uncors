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

	return int64(cost)
}

const (
	bufferItems = 64
	// expectedEntrySize is the response size the counter budget is sized around.
	// Ristretto wants roughly ten counters per item it expects to hold, so the
	// budget has to follow MaxSize rather than being a fixed number.
	expectedEntrySize = 4 << 10 // 4 KiB
	countersPerEntry  = 10
	minNumCounters    = 1 << 10
)

type RistrettoCache struct {
	storage *ristretto.Cache[string, contracts.CachedResponse]
	ttl     time.Duration
}

func NewRistrettoCache(maxSize int64, ttl time.Duration) *RistrettoCache {
	storage, err := ristretto.NewCache(&ristretto.Config[string, contracts.CachedResponse]{
		NumCounters: numCountersFor(maxSize),
		MaxCost:     maxSize,
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

// Set admits a response. Admission is asynchronous by design — that is the
// reason ristretto is here — so this never blocks the request that produced the
// response.
func (cs *RistrettoCache) Set(key string, value contracts.CachedResponse) {
	cs.storage.SetWithTTL(key, value, CalcCost(&value), cs.ttl)
}

// Wait blocks until pending admissions have been processed. Tests need it to be
// deterministic; the request path must not pay for that.
func (cs *RistrettoCache) Wait() {
	cs.storage.Wait()
}

func numCountersFor(maxSize int64) int64 {
	counters := maxSize / expectedEntrySize * countersPerEntry
	if counters < minNumCounters {
		return minNumCounters
	}

	return counters
}

// Close releases the underlying cache. It satisfies io.Closer so the app can
// free the previous cache on config reload/shutdown.
func (cs *RistrettoCache) Close() error {
	cs.storage.Close()

	return nil
}
