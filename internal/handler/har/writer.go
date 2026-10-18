package har

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	harlib "github.com/evg4b/uncors/pkg/har"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type writerConfig struct {
	fs        afero.Fs
	publisher events.Publisher
}

type WriterOption = func(*writerConfig)

func WithPublisher(publisher events.Publisher) WriterOption {
	return func(c *writerConfig) {
		c.publisher = publisher
	}
}

func WithFileSystem(fs afero.Fs) WriterOption {
	return func(c *writerConfig) {
		c.fs = fs
	}
}

func NewWriter(path string, options ...WriterOption) *harlib.Writer {
	config := helpers.ApplyOptions(&writerConfig{publisher: events.NoopPublisher{}}, options)

	helpers.AssertIsDefined(config.fs, "HARWriter: File system is not configured")

	reporter := &flushReporter{path: path, publisher: config.publisher}

	return harlib.NewWriter(path,
		harlib.WithFileSystem(config.fs),
		harlib.WithFlushHandler(reporter.flushed),
		harlib.WithErrorHandler(reporter.failed),
	)
}

type flushReporter struct {
	path      string
	publisher events.Publisher

	// Set after a failed flush, so a failure streak produces one notice.
	failing bool
}

// Runs on the writer goroutine only, like every flush.
func (r *flushReporter) flushed(flush harlib.Flush) {
	ctx, span := telemetry.StartAt(context.Background(), "har.flush", flush.Start)
	defer span.End(trace.WithTimestamp(flush.Start.Add(flush.Duration)))

	span.SetAttributes(
		attribute.String("uncors.har.path", flush.Path),
		attribute.Int("uncors.har.entries", flush.Entries),
		attribute.Int("uncors.har.bytes", flush.Bytes),
	)

	telemetry.HARFlushed(ctx, flush.Err)

	if flush.Err != nil {
		telemetry.Fail(span, flush.Err)
		telemetry.Error(ctx, "cannot write HAR file", slog.String("file", flush.Path), telemetry.Err(flush.Err))
		r.reportFailure(flush.Err)

		return
	}

	telemetry.Debug(ctx, "HAR file written",
		slog.String("file", flush.Path), slog.Int("entries", flush.Entries), slog.Int("bytes", flush.Bytes))

	r.failing = false
}

func (r *flushReporter) failed(err error) {
	telemetry.Error(context.Background(), "cannot encode HAR entry", slog.String("file", r.path), telemetry.Err(err))
}

func (r *flushReporter) reportFailure(err error) {
	if r.failing {
		return
	}

	r.failing = true
	r.publisher.Publish(events.Notice{
		Level: events.NoticeError,
		Text:  fmt.Sprintf("HAR: cannot write %s: %v", r.path, err),
	})
}
