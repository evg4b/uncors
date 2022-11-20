package mock

import (
	"net/http"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/spf13/afero"
)

type internalHandler struct {
	response Response
	logger   contracts.Logger
	fs       afero.Fs
}

func (handler *internalHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	response := handler.response
	header := writer.Header()
	infrastructure.WriteCorsHeaders(header)
	for key, value := range response.Headers {
		header.Set(key, value)
	}

	var err error
	if len(handler.response.File) > 0 {
		err = handler.serveFileContent(writer, request)
	} else {
		err = handler.serveRawContent(writer)
	}

	if err != nil {
		// TODO: add error handling
		return
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
