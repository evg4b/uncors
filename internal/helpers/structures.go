package helpers

func ApplyOptions[T any](service *T, options []func(*T)) *T {
	for _, option := range options {
		option(service)
	}

	return service
}
