package helpers

import "strings"

func AssertIsDefined(value any, message ...string) {
	if value == nil {
		message := strings.Join(message, " ")
		if len(message) == 0 {
			message = "Required variable is not defined"
		}

		panic(message)
	}
}
