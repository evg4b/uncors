package har

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/contracts"
	"github.com/evg4b/uncors/internal/headers"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	harlib "github.com/evg4b/uncors/pkg/har"
	"go.opentelemetry.io/otel/attribute"
)

const maxRecordedRequestBody = 10 << 20

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

	reqBody, streamed, err := recordRequestBody(req)
	if err != nil {
		telemetry.Fail(span, err)

		return err
	}

	writer.EnableBodyCapture()

	err = next(writer, req)

	elapsed := time.Since(start)
	entry := m.buildEntry(req, writer.Captured(), start, elapsed, reqBody)

	if streamed != nil {
		entry.Request.BodySize = streamed.read.Load()
		entry.Request.PostData = &harlib.PostData{MimeType: req.Header.Get(headers.ContentType)}
	}

	reqBodySize := entry.Request.BodySize
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

func recordRequestBody(req *http.Request) ([]byte, *countingReader, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, maxRecordedRequestBody+1))
	if err != nil {
		_ = req.Body.Close()

		return nil, nil, fmt.Errorf("har: cannot read request body: %w", err)
	}

	if len(body) > maxRecordedRequestBody {
		streamed := &countingReader{reader: io.MultiReader(bytes.NewReader(body), req.Body)}
		req.Body = readCloser{Reader: streamed, Closer: req.Body}

		return nil, streamed, nil
	}

	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))

	return body, nil, nil
}

type countingReader struct {
	reader io.Reader
	read   atomic.Int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read.Add(int64(n))

	return n, err
}

type readCloser struct {
	io.Reader
	io.Closer
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

	harResponse := harlib.NewResponse(response, capture.Body, m.options...)
	if capture.Truncated {
		harResponse.BodySize = capture.Written
		harResponse.Content.Comment = fmt.Sprintf("Only the first %d bytes of the body are recorded", len(capture.Body))
	}

	return harlib.NewEntry(start, elapsed, harlib.NewRequest(req, reqBody, m.options...), harResponse)
}
