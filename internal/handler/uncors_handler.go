package handler

import (
	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

type (
	CacheMiddlewareFactory   = func(globs config.CacheGlobs) contracts.Middleware
	StaticMiddlewareFactory  = func(path string, dir config.StaticDirectory) contracts.Middleware
	MockHandlerFactory       = func(response config.Response) contracts.Handler
	ScriptHandlerFactory     = func(script config.Script) contracts.Handler
	RewriteMiddlewareFactory = func(rewrite config.RewritingOption) contracts.Middleware
	OptionsMiddlewareFactory = func(options config.OptionsHandling) contracts.Middleware
	HARMiddlewareFactory     = func(harConfig config.HARConfig) contracts.Middleware
)

// MiddlewareFunc adapts an ordinary func into a contracts.Middleware.
type MiddlewareFunc func(contracts.Handler) contracts.Handler

func (f MiddlewareFunc) Wrap(next contracts.Handler) contracts.Handler {
	return f(next)
}
