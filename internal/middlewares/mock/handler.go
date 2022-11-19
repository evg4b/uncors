package mock

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/go-http-utils/headers"
)

type internalHandler struct {
	response Response
	logger   contracts.Logger
}

func (handler *internalHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	response := handler.response
	header := writer.Header()
	infrastructure.WriteCorsHeaders(header)
	for key, value := range response.Headers {
		header.Set(key, value)
	}
	if len(header.Get(headers.ContentType)) == 0 {
		contentType := http.DetectContentType([]byte(response.RawContent))
		header.Set(headers.ContentType, contentType)
	}

	writer.WriteHeader(normaliseCode(response.Code))
	if _, err := fmt.Fprint(writer, response.RawContent); err != nil {
		return // TODO: add error handler
	}

	handler.logger.PrintResponse(&http.Response{
		Request:    request,
		StatusCode: response.Code,
	})
}

func normaliseCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}
