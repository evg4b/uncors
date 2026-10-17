package helpers

import "io"

func CloseSafe(resource io.Closer) {
	if resource == nil {
		return
	}

	_ = resource.Close()
}
