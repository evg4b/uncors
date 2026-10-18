package har

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/evg4b/uncors/internal/events"
	"github.com/evg4b/uncors/internal/helpers"
	"github.com/evg4b/uncors/internal/telemetry"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
)

const (
	harVersion      = "1.2"
	creatorName     = "uncors"
	creatorVersion  = "dev"
	entryChanBuffer = 4096
	harDirMode      = 0o755
	flushInterval   = time.Second
	entryPrefix     = "      "
	entriesClosing  = "\n    "
)

var archiveHead, archiveTail = archiveFrame()

type Writer struct {
	path      string
	fs        afero.Fs
	publisher events.Publisher
	entries   chan Entry
	done      chan struct{}
	once      sync.Once
	wg        sync.WaitGroup
	mu        sync.Mutex
	encoded   [][]byte
	size      int

	// Set after a failed flush, so a failure streak produces one notice.
	failing bool

	discarded atomic.Bool
}

type WriterOption = func(*Writer)

func WithPublisher(publisher events.Publisher) WriterOption {
	return func(w *Writer) {
		w.publisher = publisher
	}
}

func WithFileSystem(fs afero.Fs) WriterOption {
	return func(w *Writer) {
		w.fs = fs
	}
}

func NewWriter(path string, options ...WriterOption) *Writer {
	writer := helpers.ApplyOptions(&Writer{
		path:      path,
		publisher: events.NoopPublisher{},
		entries:   make(chan Entry, entryChanBuffer),
		done:      make(chan struct{}),
	}, options)

	helpers.AssertIsDefined(writer.fs, "HARWriter: File system is not configured")

	writer.wg.Add(1)

	go writer.run()

	return writer
}

// AddEntry queues the entry and reports whether it was queued: when the
// queue is full the entry is dropped rather than block the request.
func (w *Writer) AddEntry(entry Entry) bool {
	select {
	case w.entries <- entry:
		return true
	default:
		return false
	}
}

func (w *Writer) Path() string {
	return w.path
}

func (w *Writer) Discard() error {
	w.discarded.Store(true)

	return w.Close()
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

	var (
		timer *time.Timer
		tick  <-chan time.Time
	)

	for {
		select {
		case entry := <-w.entries:
			w.append(entry)
			w.drain()

			if tick == nil {
				timer = time.NewTimer(flushInterval)
				tick = timer.C
			}

		case <-tick:
			tick = nil

			w.flush()

		case <-w.done:
			if timer != nil {
				timer.Stop()
			}

			if !w.discarded.Load() {
				w.drain()
				w.flush()
			}

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
	data, err := json.MarshalIndent(entry, entryPrefix, "  ")
	if err != nil {
		telemetry.Error(context.Background(), "cannot encode HAR entry", slog.String("file", w.path), telemetry.Err(err))

		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.encoded = append(w.encoded, data)
	w.size += len(data)
}

func (w *Writer) flush() {
	ctx, span := telemetry.Start(context.Background(), "har.flush")
	defer span.End()

	w.mu.Lock()
	snapshot := w.encoded[:len(w.encoded):len(w.encoded)]
	size := w.size
	w.mu.Unlock()

	span.SetAttributes(
		attribute.String("uncors.har.path", w.path),
		attribute.Int("uncors.har.entries", len(snapshot)),
		attribute.Int("uncors.har.bytes", size),
	)

	err := w.write(snapshot)
	telemetry.HARFlushed(ctx, err)

	if err != nil {
		telemetry.Fail(span, err)
		telemetry.Error(ctx, "cannot write HAR file", slog.String("file", w.path), telemetry.Err(err))
		w.reportFailure(err)

		return
	}

	telemetry.Debug(ctx, "HAR file written",
		slog.String("file", w.path), slog.Int("entries", len(snapshot)), slog.Int("bytes", size))

	w.failing = false
}

func (w *Writer) write(entries [][]byte) error {
	dir := filepath.Dir(w.path)

	err := w.fs.MkdirAll(dir, harDirMode)
	if err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	tmp, err := afero.TempFile(w.fs, dir, filepath.Base(w.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}

	err = errors.Join(writeArchive(tmp, entries), tmp.Close())
	if err != nil {
		_ = w.fs.Remove(tmp.Name())

		return fmt.Errorf("cannot write temp file: %w", err)
	}

	err = w.fs.Rename(tmp.Name(), w.path)
	if err != nil {
		_ = w.fs.Remove(tmp.Name())

		return fmt.Errorf("cannot rename temp file: %w", err)
	}

	return nil
}

func writeArchive(file afero.File, entries [][]byte) error {
	buffered := bufio.NewWriter(file)

	_, _ = buffered.Write(archiveHead)
	_ = buffered.WriteByte('[')

	for i, entry := range entries {
		if i > 0 {
			_ = buffered.WriteByte(',')
		}

		_ = buffered.WriteByte('\n')
		_, _ = buffered.WriteString(entryPrefix)
		_, _ = buffered.Write(entry)
	}

	if len(entries) > 0 {
		_, _ = buffered.WriteString(entriesClosing)
	}

	_ = buffered.WriteByte(']')
	_, _ = buffered.Write(archiveTail)

	return buffered.Flush()
}

func archiveFrame() ([]byte, []byte) {
	data, err := json.MarshalIndent(HAR{
		Log: Log{
			Version: harVersion,
			Creator: Creator{Name: creatorName, Version: creatorVersion},
			Entries: []Entry{},
		},
	}, "", "  ")
	if err != nil {
		panic(err)
	}

	head, tail, found := bytes.Cut(data, []byte("[]"))
	if !found {
		panic("har: cannot find the entries list in the archive frame")
	}

	return head, append(tail, '\n')
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
