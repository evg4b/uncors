// Package helpers holds small generic utilities over the language's own types.
// Nothing here knows anything about uncors: anything that does belongs in the
// package that owns the concept.
package helpers

// ApplyOptions applies functional options to a value under construction.
func ApplyOptions[T any](service *T, options []func(*T)) *T {
	for _, option := range options {
		option(service)
	}

	return service
}
