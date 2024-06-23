package mock

import (
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/tui"
	"github.com/spf13/afero"
)

type Handler struct {
	response config.Response
	logger   contracts.Logger
	fs       afero.Fs
	after    func(duration time.Duration) <-chan time.Time
}

func NewMockHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	response := h.response
	header := writer.Header()

	if h.waiteDelay(writer, request, response) {
		return
	}

	infra.WriteCorsHeaders(header)
	for key, value := range response.Headers {
		header.Set(key, value)
	}

	switch {
	case response.IsFake():
		h.serveFakeContent(writer, request)

		return
	case response.IsFile():
		err := h.serveFileContent(writer, request)
		if err != nil {
			infra.HTTPError(writer, err)

			return
		}
	case response.IsRaw():
		h.serveRawContent(writer)
	}

	tui.PrintResponse(h.logger, request, writer.StatusCode())
}

func (h *Handler) waiteDelay(writer contracts.ResponseWriter, request *contracts.Request, response config.Response) bool {
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

				return true
			case <-h.after(response.Delay):
				h.logger.Debugf("Delay is complete (url: %s)", url)

				break waitingLoop
			}
		}
	}

	return false
}

func normaliseCode(code int) int {
	if code == 0 {
		return http.StatusOK
	}

	return code
}
