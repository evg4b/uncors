package tui

import (
	"github.com/evg4b/uncors/internal/contracts"
)

func PrintResponse(logger contracts.Logger, request *contracts.Request, statusCode int) {
	logger.Print(printResponse(request, statusCode)) //nolint:forbidigo
}
