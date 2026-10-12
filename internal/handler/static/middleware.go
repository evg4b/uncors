package static

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/tracking"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/infra"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

var errNotHandled = errors.New("request is not handled")

type Middleware struct {
	fs     afero.Fs
	index  string
	prefix string
}

func NewStaticMiddleware(options ...MiddlewareOption) *Middleware {
	return helpers.ApplyOptions(&Middleware{}, options)
}

func (h *Middleware) ServeHTTP(writer contracts.ResponseWriter, request *contracts.Request, next contracts.Next) error {
	original := request

	request, span := telemetry.StartRequest(request, "static.serve")
	defer span.End()

	filePath := h.extractFilePath(request)

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("uncors.static.prefix", h.prefix),
			attribute.String("uncors.static.file", filePath),
		)
	}

	file, stat, err := h.openFile(filePath)
	defer helpers.CloseSafe(file)

	if err != nil {
		if errors.Is(err, errNotHandled) {
			if span.IsRecording() {
				span.SetAttributes(attribute.Bool("uncors.static.fall_through", true))
			}

			telemetry.StaticRequest(request.Context(), telemetry.ResultFallThrough)
			// The default handler is not part of this span.
			span.End()

			return next(writer, original)
		}

		tracking.Claim(request.Context(), events.HandlerStatic)
		telemetry.Fail(span, err)
		telemetry.StaticRequest(request.Context(), telemetry.ResultError)

		return err
	}

	tracking.Claim(request.Context(), events.HandlerStatic)
	http.ServeContent(writer, request, stat.Name(), stat.ModTime(), file)

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("uncors.static.served", stat.Name()),
			attribute.Bool("uncors.static.index_used", stat.Name() != filepath.Base(filePath)),
			attribute.Int64("uncors.static.file_bytes", stat.Size()),
			attribute.String("uncors.static.modified", stat.ModTime().UTC().Format(time.RFC3339)),
			semconv.HTTPResponseStatusCode(helpers.NormaliseStatusCode(writer.StatusCode())),
		)
	}

	telemetry.StaticRequest(request.Context(), telemetry.ResultServed)

	return nil
}

func (h *Middleware) Wrap(next contracts.Handler) contracts.Handler {
	return infra.HandlerFunc(func(writer contracts.ResponseWriter, request *contracts.Request) error {
		return h.ServeHTTP(writer, request, func(w contracts.ResponseWriter, r *contracts.Request) error {
			return next.ServeHTTP(w, r)
		})
	})
}

func (h *Middleware) extractFilePath(request *http.Request) string {
	// Cleaning a rooted path resolves every ".." segment, so the result cannot
	// point outside the static directory.
	return filepath.Clean("/" + strings.TrimPrefix(request.URL.Path, h.prefix))
}

func (h *Middleware) openFile(filePath string) (afero.File, os.FileInfo, error) {
	file, err := h.fs.Open(filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("failed to open file: %w", err)
		}

		return h.openIndexFile()
	}

	stat, err := file.Stat()
	if err != nil {
		return file, nil, fmt.Errorf("failed to get information about file: %w", err)
	}

	if !stat.IsDir() {
		return file, stat, nil
	}

	// A directory is served by the index file, so its own handle is released
	// before the index file takes its place.
	err = file.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to close directory: %w", err)
	}

	return h.openIndexFile()
}

// On error the returned file, when not nil, is still open.
func (h *Middleware) openIndexFile() (afero.File, os.FileInfo, error) {
	if len(h.index) == 0 {
		return nil, nil, errNotHandled
	}

	file, err := h.fs.Open(h.index)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open index file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		return file, nil, fmt.Errorf("failed to get information about index file: %w", err)
	}

	return file, stat, nil
}

type MiddlewareOption = func(*Middleware)

func WithFileSystem(fs afero.Fs) MiddlewareOption {
	return func(h *Middleware) {
		h.fs = fs
	}
}

func WithIndex(index string) MiddlewareOption {
	return func(h *Middleware) {
		h.index = index
	}
}

func WithPrefix(prefix string) MiddlewareOption {
	return func(h *Middleware) {
		h.prefix = prefix
	}
}
