package router

import (
	"net/http"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
)

// Deps are the handlers and middleware factories the router needs to turn a set
// of mappings into a request graph. Stating them as a value the composition root
// fills in keeps the router a leaf package: it does not know how any of these
// are built, and it cannot reach back into the composition root to find out.
type Deps struct {
	// Proxy handles everything no mapping-specific route matched.
	Proxy http.Handler
	// Static serves a directory mounted at path.
	Static func(path string, dir config.StaticDirectory) contracts.Middleware
	// Rewrite rewrites matching request paths.
	Rewrite func(rewriting *config.RewritingOption) contracts.Middleware
	// HAR records traffic of a mapping into an archive.
	HAR func(harConfig *config.HARConfig) contracts.Middleware
	// Options answers CORS preflight requests.
	Options func(cfg config.OptionsHandling) contracts.Middleware
	// Cache caches responses matching the given globs.
	Cache func(globs config.CacheGlobs) contracts.Middleware
	// Mock answers with a configured response.
	Mock func(response *config.Response) http.Handler
	// Script answers with the result of a Lua script.
	Script func(scriptConfig *config.Script) http.Handler
}
