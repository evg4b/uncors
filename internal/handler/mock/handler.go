package mock

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/go-http-utils/headers"
	"github.com/spf13/afero"
)

type Handler struct {
	response config.Response
	output   contracts.Output
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

	err := h.writeResponse(writer, request)
	if err != nil {
		log.Printf("ERROR: Mock handler error: %s (URL: %s)", err.Error(), request.URL.String())
		infra.HTTPError(writer, err)

		return
	}

	h.output.Request(helpers.ToRequestData(request, writer.StatusCode()))
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
		err := h.serveFileContent(writer, request)
		if err != nil {
			return err
		}
	case response.IsRaw():
		err := h.serveRawContent(writer)
		if err != nil {
			return err
		}
	default:
		return ErrResponseIsNotDefined
	}

	return nil
}

func (h *Handler) serveRawContent(writer http.ResponseWriter) error {
	response := h.response

	header := writer.Header()
	if len(header.Get(headers.ContentType)) == 0 {
		contentType := http.DetectContentType([]byte(response.Raw))
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

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to receive file information: %w", err)
	}

	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

	return nil
}

func (h *Handler) waiteDelay(writer contracts.ResponseWriter, request *contracts.Request) bool {
	response := h.response

	if response.Delay > 0 {
		log.Printf("Delay %s for %s", response.Delay, request.URL.RequestURI())
		ctx := request.Context()
		url := request.URL.RequestURI()

	waitingLoop:
		for {
			select {
			case <-ctx.Done():
				writer.WriteHeader(http.StatusServiceUnavailable)
				log.Printf("Delay is canceled (url: %s)", url)

				return true
			case <-h.after(response.Delay):
				log.Printf("Delay is complete (url: %s)", url)

				break waitingLoop
			}
		}
	}

	return false
}
