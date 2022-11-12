package helpers

import "strings"

func AssertIsDefined(value interface{}, message ...string) {
	if value == nil {
		message := strings.Join(message, " ")
		if len(message) == 0 {
			message = "Requared variable is not defined"
		}
		panic(message)
	}
}
