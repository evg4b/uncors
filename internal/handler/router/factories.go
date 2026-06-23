package router

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

type (
	// CacheMiddlewareFactory creates a cache middleware for the given cache configuration.
	CacheMiddlewareFactory = func(globs config.CacheGlobs) contracts.Middleware
)
