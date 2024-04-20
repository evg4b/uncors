package tui

import "github.com/evg4b/uncors/internal/contracts"

func PrintResponse(request *contracts.Request, statusCode int) {
	println(printResponse(request, statusCode)) //nolint:forbidigo
}
