package mock

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tracking"

	"github.com/evg4b/uncors/internal/config"
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/go-http-utils/headers"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	response *config.Response
	fs       afero.Fs
	after    func(duration time.Duration) <-chan time.Time
}

const contentTypeSniffLen = 512

var ErrResponseIsNotDefined = errors.New("response is not defined")

func NewMockHandler(options ...HandlerOption) *Handler {
	return helpers.ApplyOptions(&Handler{}, options)
}

func (h *Handler) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request) error {
	request, span := telemetry.StartRequest(request, "mock.serve")
	defer span.End()

	tracking.Claim(request.Context(), events.HandlerMock)

	if span.IsRecording() {
		span.SetAttributes(h.attributes()...)
	}

	if h.waitDelay(writer, request) {
		telemetry.MockServed(request.Context(), http.StatusServiceUnavailable, h.response.Delay, true)

		if telemetry.Enabled() {
			telemetry.Debug(request.Context(), "mock delay cancelled by the client",
				slog.Duration("delay", h.response.Delay))
		}

		return nil
	}

	err := h.writeResponse(writer, request)
	if err != nil {
		telemetry.Fail(span, err)

		return err
	}

	telemetry.MockServed(request.Context(), helpers.NormaliseStatusCode(h.response.Code), h.response.Delay, false)

	if telemetry.Enabled() {
		h.logServed(request)
	}

	return nil
}

func (h *Handler) logServed(request *contracts.Request) {
	attrs := []slog.Attr{
		slog.Int("status", helpers.NormaliseStatusCode(h.response.Code)),
		slog.Duration("delay", h.response.Delay),
	}

	if h.response.IsFile() {
		attrs = append(attrs, slog.String("file", h.response.File))
	}

	telemetry.Debug(request.Context(), "served mock response", attrs...)
}

func (h *Handler) attributes() []attribute.KeyValue {
	response := h.response
	attrs := []attribute.KeyValue{
		attribute.Int("uncors.mock.status", helpers.NormaliseStatusCode(response.Code)),
		attribute.String("uncors.mock.delay", response.Delay.String()),
		attribute.Int("uncors.mock.headers", len(response.Headers)),
	}

	switch {
	case response.IsFile():
		attrs = append(attrs,
			attribute.String("uncors.mock.source", "file"),
			attribute.String("uncors.mock.file", response.File),
		)
	case response.IsRaw():
		attrs = append(attrs,
			attribute.String("uncors.mock.source", "raw"),
			attribute.Int("uncors.mock.raw_bytes", len(response.Raw)),
		)
	}

	return attrs
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

	if span := trace.SpanFromContext(request.Context()); span.IsRecording() {
		span.SetAttributes(attribute.Int64("uncors.mock.file_bytes", stat.Size()))
	}

	code := helpers.NormaliseStatusCode(h.response.Code)
	if code == http.StatusOK {
		http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

		return nil
	}

	return writeFileWithStatus(writer, file, stat, code)
}

func writeFileWithStatus(writer http.ResponseWriter, file io.Reader, stat os.FileInfo, code int) error {
	header := writer.Header()

	body := file

	if len(header.Get(headers.ContentType)) == 0 {
		contentType := mime.TypeByExtension(filepath.Ext(stat.Name()))
		if len(contentType) == 0 {
			sniff := make([]byte, contentTypeSniffLen)

			read, err := io.ReadFull(file, sniff)
			if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
				return fmt.Errorf("failed to read file: %w", err)
			}

			contentType = http.DetectContentType(sniff[:read])
			body = io.MultiReader(bytes.NewReader(sniff[:read]), file)
		}

		header.Set(headers.ContentType, contentType)
	}

	if !bodyAllowed(code) {
		writer.WriteHeader(code)

		return nil
	}

	if len(header.Get(headers.ContentLength)) == 0 {
		header.Set(headers.ContentLength, strconv.FormatInt(stat.Size(), 10))
	}

	writer.WriteHeader(code)

	_, err := io.Copy(writer, body)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	return nil
}

func bodyAllowed(code int) bool {
	return code >= http.StatusOK && code != http.StatusNoContent && code != http.StatusNotModified
}

func (h *Handler) waitDelay(writer contracts.ResponseWriter, request *contracts.Request) bool {
	if h.response.Delay <= 0 {
		return false
	}

	span := trace.SpanFromContext(request.Context())
	if span.IsRecording() {
		span.AddEvent("uncors.mock.delay_started")
	}

	select {
	case <-request.Context().Done():
		if span.IsRecording() {
			span.AddEvent("uncors.mock.delay_cancelled")
		}

		writer.WriteHeader(http.StatusServiceUnavailable)

		return true
	case <-h.after(h.response.Delay):
		if span.IsRecording() {
			span.AddEvent("uncors.mock.delay_finished")
		}

		return false
	}
}
