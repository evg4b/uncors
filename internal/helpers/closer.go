package helpers

import "io"

func CloseSafe(resource io.Closer) {
	if resource == nil {
		return
	}

	if err := resource.Close(); err != nil {
		panic(err)
	}
}
