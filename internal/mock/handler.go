package mock

import (
	"fmt"
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infrastructure"
)

type Handler struct {
	mock   Mock
	logger contracts.Logger
}

func NewMockHandler(options ...HandlerOption) *Handler {
	handler := &Handler{}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	updateRequest(request)

	header := writer.Header()
	infrastructure.WriteCorsHeaders(header)
	if len(header.Get("Content-Type")) == 0 {
		contentType := http.DetectContentType([]byte(handler.mock.Response.RawContent))
		header.Set("Content-Type", contentType)
	}

	writer.WriteHeader(handler.mock.Response.Code)
	if _, err := fmt.Fprint(writer, handler.mock.Response.RawContent); err != nil {
		return // TODO: add error handler
	}

	handler.logger.PrintResponse(&http.Response{
		Request:    request,
		StatusCode: handler.mock.Response.Code,
	})
}

func updateRequest(request *http.Request) {
	request.URL.Host = request.Host

	if request.TLS != nil {
		request.URL.Scheme = "https"
	} else {
		request.URL.Scheme = "http"
	}
}
