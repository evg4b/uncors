package mock

import (
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/configuration"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infrastructure"
	"github.com/spf13/afero"
)

type internalHandler struct {
	response configuration.Response
	logger   contracts.Logger
	fs       afero.Fs
	after    func(duration time.Duration) <-chan time.Time
}

func (handler *internalHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	response := handler.response
	header := writer.Header()

	if response.Delay > 0 {
		handler.logger.Debugf("Delay %s for %s", response.Delay, request.URL.RequestURI())
		ctx := request.Context()
		select {
		case <-ctx.Done():
			writer.WriteHeader(http.StatusServiceUnavailable)
			handler.logger.Debugf("Delay for %s canceled", request.URL.RequestURI())

			return
		case <-handler.after(response.Delay):
		}
	}

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
