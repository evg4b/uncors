package helpers

import "io"

func CloseSafe(resource io.Closer) {
	if resource == nil {
		return
	}

	err := resource.Close()
	if err != nil {
		panic(err)
	}
}
