package router

import (
	"github.com/evg4b/uncors/internal/config"
)

// gorilla/mux matches in registration order, so path-only routes go last.
func registerMatchedRoutes[T any](
	items []T,
	matcher func(*T) *config.RequestMatcher,
	register func(int, *T),
) {
	var defaults []int

	for i := range items {
		if !matcher(&items[i]).IsPathOnly() {
			register(i, &items[i])
		} else {
			defaults = append(defaults, i)
		}
	}

	for _, i := range defaults {
		register(i, &items[i])
	}
}
