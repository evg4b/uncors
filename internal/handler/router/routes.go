package router

import (
	"github.com/evg4b/uncors/internal/config"
)

// gorilla/mux matches in registration order, so path-only routes go last.
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
