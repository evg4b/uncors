package router

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

type (
	CacheMiddlewareFactory = func(globs config.CacheGlobs) contracts.Middleware
)
