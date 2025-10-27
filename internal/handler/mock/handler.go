package mock

import (
	"errors"
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

var ErrResponseIsNotDefined = errors.New("response is not defined")

func NewMockHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) {
	if h.waiteDelay(writer, request) {
		return
	}

	if err := h.writeResponse(writer, request); err != nil {
		h.logger.Error("Mock handler error", "error", err, "url", request.URL.String())
		infra.HTTPError(writer, err)

		return
	}

	tui.PrintResponse(h.logger, request, writer.StatusCode())
}

func (h *Handler) writeResponse(writer contracts.ResponseWriter, request *contracts.Request) error {
	header := writer.Header()
	response := h.response

	origin := request.Header.Get("Origin")
	infra.WriteCorsHeaders(header, origin)
	for key, value := range response.Headers {
		header.Set(key, value)
	}

	switch {
	case response.IsFile():
		if err := h.serveFileContent(writer, request); err != nil {
			return err
		}
	case response.IsRaw():
		if err := h.serveRawContent(writer); err != nil {
			return err
		}
	default:
		return ErrResponseIsNotDefined
	}

	return nil
}

func (h *Handler) waiteDelay(writer contracts.ResponseWriter, request *contracts.Request) bool {
	response := h.response

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
