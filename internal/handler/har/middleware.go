package har

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	harlib "github.com/evg4b/uncors/pkg/har"
	"go.opentelemetry.io/otel/attribute"
)

type EntryWriter interface {
	AddEntry(entry harlib.Entry) bool
	Path() string
}

type Middleware struct {
	writer               EntryWriter
	captureSecureHeaders bool
	options              []harlib.Option
}

func NewMiddleware(opts ...MiddlewareOption) *Middleware {
	middleware := helpers.ApplyOptions(&Middleware{}, opts)

	helpers.AssertIsDefined(middleware.writer, "HARMiddleware: Writer is not configured")

	middleware.options = []harlib.Option{harlib.WithSecureHeaders(middleware.captureSecureHeaders)}

	return middleware
}

func (m *Middleware) ServeHTTP(writer contracts.ResponseWriter, req *contracts.Request, next contracts.Next) error {
	req, span := telemetry.StartRequest(req, "har.record")
	defer span.End()

	start := time.Now()

	var reqBody []byte

	if req.Body != nil && req.Body != http.NoBody {
		body, err := io.ReadAll(req.Body)
		_ = req.Body.Close()

		if err != nil {
			telemetry.Fail(span, err)

			return fmt.Errorf("har: cannot read request body: %w", err)
		}

		reqBody = body
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	reqBodySize := int64(len(reqBody))

	writer.EnableBodyCapture()

	err := next(writer, req)

	elapsed := time.Since(start)
	entry := m.buildEntry(req, writer.Captured(), start, elapsed, reqBody)
	recorded := m.writer.AddEntry(entry)
	telemetry.HAREntry(req.Context(), recorded)

	if !recorded && telemetry.Enabled() {
		telemetry.Warn(req.Context(), "HAR entry dropped: the write queue is full",
			slog.String("file", m.writer.Path()), slog.String("url", helpers.LogURL(req.URL)))
	}

	if span.IsRecording() {
		span.SetAttributes(
			attribute.String("uncors.har.file", m.writer.Path()),
			attribute.Int64("uncors.har.request_body_bytes", reqBodySize),
			attribute.Int64("uncors.har.response_body_bytes", entry.Response.BodySize),
			attribute.Bool("uncors.har.recorded", recorded),
			attribute.Bool("uncors.har.secure_headers", m.captureSecureHeaders),
		)
	}

	return err
}

func (m *Middleware) buildEntry(
	req *http.Request,
	capture contracts.ResponseCapture,
	start time.Time,
	elapsed time.Duration,
	reqBody []byte,
) harlib.Entry {
	response := &http.Response{
		StatusCode: capture.StatusCode,
		Proto:      req.Proto,
		Header:     capture.Header,
	}

	return harlib.NewEntry(start, elapsed,
		harlib.NewRequest(req, reqBody, m.options...),
		harlib.NewResponse(response, capture.Body, m.options...),
	)
}
