package helpers

import (
	"strings"
	"unsafe"
)

func AssertIsDefined(value any, message ...string) {
	if (*[2]uintptr)(unsafe.Pointer(&value))[1] == 0 {
		message := strings.Join(message, " ")
		if len(message) == 0 {
			message = "Required variable is not defined"
		}

		panic(message)
	}
}
