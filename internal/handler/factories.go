package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

type (
	// CacheMiddlewareFactory creates a cache middleware for the given cache configuration.
	CacheMiddlewareFactory = func(globs config.CacheGlobs) contracts.Middleware

	// StaticMiddlewareFactory creates a static file serving middleware for the given directory.
	StaticMiddlewareFactory = func(path string, dir config.StaticDirectory) contracts.Middleware

	// MockHandlerFactory creates a mock handler for the given response configuration.
	MockHandlerFactory = func(response config.Response) contracts.Handler

	// ScriptHandlerFactory creates a script handler for the given script configuration.
	ScriptHandlerFactory = func(script config.Script) contracts.Handler

	// RewriteMiddlewareFactory creates a rewrite middleware for the given rewrite option.
	RewriteMiddlewareFactory = func(rewrite config.RewritingOption) contracts.Middleware

	// OptionsMiddlewareFactory creates an OPTIONS handler middleware for the given configuration.
	OptionsMiddlewareFactory = func(options config.OptionsHandling) contracts.Middleware

	// HARMiddlewareFactory creates a HAR (HTTP Archive) recording middleware for the given configuration.
	HARMiddlewareFactory = func(harConfig config.HARConfig) contracts.Middleware
)
