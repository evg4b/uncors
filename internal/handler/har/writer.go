package har

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

const (
	harVersion      = "1.2"
	creatorName     = "uncors"
	creatorVersion  = "dev"
	entryChanBuffer = 4096
	harFileMode     = 0o600
	harDirMode      = 0o755
)

type Writer struct {
	path      string
	publisher events.Publisher
	entries   chan Entry
	done      chan struct{}
	once      sync.Once
	wg        sync.WaitGroup
	mu        sync.Mutex
	all       []Entry

	// Set after a failed flush, so a failure streak produces one notice.
	failing bool
}

type WriterOption = func(*Writer)

func WithPublisher(publisher events.Publisher) WriterOption {
	return func(w *Writer) {
		w.publisher = publisher
	}
}

func NewWriter(path string, options ...WriterOption) *Writer {
	writer := helpers.ApplyOptions(&Writer{
		path:      path,
		publisher: events.NoopPublisher{},
		entries:   make(chan Entry, entryChanBuffer),
		done:      make(chan struct{}),
	}, options)

	writer.wg.Add(1)

	go writer.run()

	return writer
}

func (w *Writer) AddEntry(entry Entry) {
	select {
	case w.entries <- entry:
	default:
		// drop entry rather than block the request goroutine
	}
}

func (w *Writer) Close() error {
	w.once.Do(func() {
		close(w.done)
		w.wg.Wait()
	})

	return nil
}

func (w *Writer) run() {
	defer w.wg.Done()

	for {
		select {
		case entry := <-w.entries:
			w.append(entry)
			w.drain()
			w.flush()

		case <-w.done:
			w.drain()
			w.flush()

			return
		}
	}
}

func (w *Writer) drain() {
	for {
		select {
		case entry := <-w.entries:
			w.append(entry)
		default:
			return
		}
	}
}

func (w *Writer) append(entry Entry) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.all = append(w.all, entry)
}

func (w *Writer) flush() {
	_, span := telemetry.Start(context.Background(), "har.flush")
	defer span.End()

	w.mu.Lock()
	snapshot := make([]Entry, len(w.all))
	copy(snapshot, w.all)
	w.mu.Unlock()

	archive := HAR{
		Log: Log{
			Version: harVersion,
			Creator: Creator{Name: creatorName, Version: creatorVersion},
			Entries: snapshot,
		},
	}

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		telemetry.Fail(span, err)

		return
	}

	span.SetAttributes(
		attribute.String("uncors.har.path", w.path),
		attribute.Int("uncors.har.entries", len(snapshot)),
		attribute.Int("uncors.har.bytes", len(data)),
	)

	err = w.write(data)
	if err != nil {
		telemetry.Fail(span, err)
		w.reportFailure(err)

		return
	}

	w.failing = false
}

func (w *Writer) write(data []byte) error {
	err := os.MkdirAll(filepath.Dir(w.path), harDirMode)
	if err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	tmp := w.path + ".tmp"

	err = os.WriteFile(tmp, data, harFileMode)
	if err != nil {
		return fmt.Errorf("cannot write temp file: %w", err)
	}

	err = os.Rename(tmp, w.path)
	if err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("cannot rename temp file: %w", err)
	}

	return nil
}

// Runs on the writer goroutine only, like every flush.
func (w *Writer) reportFailure(err error) {
	if w.failing {
		return
	}

	w.failing = true
	w.publisher.Publish(events.Notice{
		Level: events.NoticeError,
		Text:  fmt.Sprintf("HAR: cannot write %s: %v", w.path, err),
	})
}
