package mock

import (
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/spf13/afero"
)

type Handler struct {
	response config.Response
	logger   contracts.Logger
	fs       afero.Fs
	after    func(duration time.Duration) <-chan time.Time
}

func NewMockHandler(options ...HandlerOption) *Handler {
	handler := &Handler{}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	response := h.response
	header := writer.Header()

	if response.Delay > 0 {
		h.logger.Debugf("Delay %s for %s", response.Delay, request.URL.RequestURI())
		ctx := request.Context()
		url := request.URL.RequestURI()
	waitingLoop:
		for {
			select {
			case <-ctx.Done():
				writer.WriteHeader(http.StatusServiceUnavailable)
				h.logger.Debugf("Delay is canceled (url: %s)", url)

				return
			case <-h.after(response.Delay):
				h.logger.Debugf("Delay is complete (url: %s)", url)

				break waitingLoop
			}
		}
	}

	infra.WriteCorsHeaders(header)
	for key, value := range response.Headers {
		header.Set(key, value)
	}

	if len(h.response.File) > 0 {
		err := h.serveFileContent(writer, request)
		if err != nil {
			infra.HTTPError(writer, err)

			return
		}
	} else {
		h.serveRawContent(writer)
	}

	h.logger.PrintResponse(&http.Response{
		Request:    request,
		StatusCode: writer.StatusCode(),
	})
}

func normaliseCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}
