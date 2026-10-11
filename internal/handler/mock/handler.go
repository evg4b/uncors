package mock

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tracking"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/pkg/urlt"
	"github.com/go-http-utils/headers"
	"github.com/spf13/afero"
)

type Handler struct {
	response *config.Response
	fs       afero.Fs
	after    func(duration time.Duration) <-chan time.Time
	logger   *slog.Logger
}

const contentTypeSniffLen = 512

var ErrResponseIsNotDefined = errors.New("response is not defined")

func NewMockHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{logger: slog.Default()}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) error {
	tracking.Claim(request.Context(), events.HandlerMock)

	if h.waitDelay(writer, request) {
		return nil
	}

	err := h.writeResponse(writer, request)
	if err != nil {
		h.logger.Error("mock handler error", "url", urlt.URL_String(request.URL), "error", err)

		return err
	}

	return nil
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
		return h.serveFileContent(writer, request)
	case response.IsRaw():
		return h.serveRawContent(writer)
	default:
		return ErrResponseIsNotDefined
	}
}

func (h *Handler) serveRawContent(writer http.ResponseWriter) error {
	response := h.response

	header := writer.Header()
	if len(header.Get(headers.ContentType)) == 0 {
		sniff := response.Raw
		if len(sniff) > contentTypeSniffLen {
			sniff = sniff[:contentTypeSniffLen]
		}

		contentType := http.DetectContentType([]byte(sniff))
		header.Set(headers.ContentType, contentType)
	}

	writer.WriteHeader(helpers.NormaliseStatusCode(response.Code))
	_, err := fmt.Fprint(writer, response.Raw)

	return err
}

func (h *Handler) serveFileContent(writer http.ResponseWriter, request *http.Request) error {
	fileName := h.response.File

	file, err := h.fs.OpenFile(fileName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fileName, err)
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to receive file information: %w", err)
	}

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

	return nil
}

func (h *Handler) waitDelay(writer contracts.ResponseWriter, request *contracts.Request) bool {
	if h.response.Delay <= 0 {
		return false
	}

	select {
	case <-request.Context().Done():
		writer.WriteHeader(http.StatusServiceUnavailable)

		return true
	case <-h.after(h.response.Delay):
		return false
	}
}
