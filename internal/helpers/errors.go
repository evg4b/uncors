package helpers

import "github.com/evg4b/uncors/internal/errordef"

func HandleCriticalError(err error) {
	if err != nil {
		panic(errordef.NewCriticalError(err))
	}
}
