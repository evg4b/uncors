package router

import (
	"github.com/evg4b/uncors/internal/config"
)

// registerMatchedRoutes registers routes in two passes: specific matchers first, path-only matchers second.
// This ensures specific routes take priority over catch-all path routes in gorilla/mux.
func registerMatchedRoutes[T any](
	items []T,
	matcher func(*T) *config.RequestMatcher,
	register func(*T),
) {
	var defaults []*T

	for i := range items {
		item := &items[i]
		if !matcher(item).IsPathOnly() {
			register(item)
		} else {
			defaults = append(defaults, item)
		}
	}

	for _, item := range defaults {
		register(item)
	}
}
