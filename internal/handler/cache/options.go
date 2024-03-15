package cache

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/tui/monitor"
	"github.com/patrickmn/go-cache"
)

type MiddlewareOption = func(*Middleware)

func WithLogger(logger contracts.Logger) MiddlewareOption {
	return func(m *Middleware) {
		m.logger = logger
	}
}

func WithMethods(methods []string) MiddlewareOption {
	return func(m *Middleware) {
		m.methods = methods
	}
}

func WithGlobs(globs config.CacheGlobs) MiddlewareOption {
	return func(m *Middleware) {
		m.pathGlobs = globs
	}
}

func WithCacheStorage(storage *cache.Cache) MiddlewareOption {
	return func(m *Middleware) {
		m.storage = storage
	}
}

func WithRequestTracker(tracker monitor.RequestTracker) MiddlewareOption {
	return func(m *Middleware) {
		m.tracker = tracker
	}
}
