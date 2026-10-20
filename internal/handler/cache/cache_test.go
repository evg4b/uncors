package cache_test

import (
	"testing"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/handler/cache"
	"github.com/evg4b/uncors/testing/testutils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestCalcCost(t *testing.T) {
	tests := []struct {
		name     string
		input    *contracts.CachedResponse
		expected int64
	}{
		{
			name: "empty body and headers",
			input: &contracts.CachedResponse{
				Body:    []byte{},
				Headers: []contracts.CachedHeader{},
			},
			expected: 0,
		},
		{
			name: "body only",
			input: &contracts.CachedResponse{
				Body:    []byte("hello"),
				Headers: nil,
			},
			expected: 5,
		},
		{
			name: "headers only",
			input: &contracts.CachedResponse{
				Body: nil,
				Headers: []contracts.CachedHeader{
					testutils.CachedHeader("Content-Type", "json"),
					testutils.CachedHeader("X-Test", "1"),
				},
			},
			expected: 23,
		},
		{
			name: "mixed body and headers",
			input: &contracts.CachedResponse{
				Body: []byte("data"),
				Headers: []contracts.CachedHeader{
					testutils.CachedHeader("K", "V"),
				},
			},
			expected: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.CalcCost(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRistrettoCache(t *testing.T) {
	ttl := 1 * time.Minute
	storage := lo.Must(cache.NewRistrettoCache(1024, ttl))

	assert.NotNil(t, storage)

	t.Run("Get missing key", func(t *testing.T) {
		val, ok := storage.Get("non-existent")
		assert.False(t, ok)
		assert.Empty(t, val)
	})

	t.Run("Set and Get key", func(t *testing.T) {
		key := "test-key"
		value := contracts.CachedResponse{
			Body: []byte("test-body"),
			Headers: []contracts.CachedHeader{
				testutils.CachedHeader("H", "V"),
			},
		}

		assert.True(t, storage.Set(key, value))
		storage.Wait()

		got, ok := storage.Get(key)

		assert.True(t, ok)
		assert.Equal(t, value.Body, got.Body)
		assert.Equal(t, value.Headers, got.Headers)
	})

	t.Run("Set rejects a value larger than the storage", func(t *testing.T) {
		key := "large-key"

		assert.False(t, storage.Set(key, contracts.CachedResponse{Body: make([]byte, 2048)}))
		storage.Wait()

		_, ok := storage.Get(key)
		assert.False(t, ok)
	})

	t.Run("Wait call", func(t *testing.T) {
		assert.NotPanics(t, func() {
			storage.Wait()
		})
	})
}
