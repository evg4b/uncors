package har

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/afero"
)

const (
	DefaultQueueSize     = 4096
	DefaultFlushInterval = time.Second

	dirMode        = 0o755
	entryPrefix    = "      "
	entriesClosing = "\n    "
)

var (
	DefaultCreator = Creator{Name: "uncors", Version: "dev"}

	ErrEncodeEntry = errors.New("har: cannot encode entry")
)

type Flush struct {
	Path     string
	Entries  int
	Bytes    int
	Start    time.Time
	Duration time.Duration
	Err      error
}

type Writer struct {
	path          string
	fs            afero.Fs
	creator       Creator
	queueSize     int
	flushInterval time.Duration
	onFlush       func(Flush)
	onError       func(error)

	head, tail []byte
	entries    chan Entry
	done       chan struct{}
	once       sync.Once
	wg         sync.WaitGroup
	mu         sync.Mutex
	encoded    [][]byte
	size       int

	discarded atomic.Bool
}

type WriterOption = func(*Writer)

func WithFileSystem(fs afero.Fs) WriterOption {
	return func(w *Writer) {
		w.fs = fs
	}
}

func WithCreator(creator Creator) WriterOption {
	return func(w *Writer) {
		w.creator = creator
	}
}

func WithQueueSize(size int) WriterOption {
	return func(w *Writer) {
		w.queueSize = size
	}
}

func WithFlushInterval(interval time.Duration) WriterOption {
	return func(w *Writer) {
		w.flushInterval = interval
	}
}

func WithFlushHandler(handler func(Flush)) WriterOption {
	return func(w *Writer) {
		w.onFlush = handler
	}
}

func WithErrorHandler(handler func(error)) WriterOption {
	return func(w *Writer) {
		w.onError = handler
	}
}

func NewWriter(path string, options ...WriterOption) *Writer {
	writer := &Writer{
		path:          path,
		fs:            afero.NewOsFs(),
		creator:       DefaultCreator,
		queueSize:     DefaultQueueSize,
		flushInterval: DefaultFlushInterval,
		done:          make(chan struct{}),
	}

	for _, option := range options {
		option(writer)
	}

	writer.head, writer.tail = archiveFrame(writer.creator)
	writer.entries = make(chan Entry, writer.queueSize)
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
				timer = time.NewTimer(w.flushInterval)
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
		if w.onError != nil {
			w.onError(fmt.Errorf("%w: %w", ErrEncodeEntry, err))
		}

		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.encoded = append(w.encoded, data)
	w.size += len(data)
}

func (w *Writer) flush() {
	start := time.Now()

	w.mu.Lock()
	snapshot := w.encoded[:len(w.encoded):len(w.encoded)]
	size := w.size
	w.mu.Unlock()

	err := w.write(snapshot)
	if w.onFlush == nil {
		return
	}

	w.onFlush(Flush{
		Path:     w.path,
		Entries:  len(snapshot),
		Bytes:    size,
		Start:    start,
		Duration: time.Since(start),
		Err:      err,
	})
}

func (w *Writer) write(entries [][]byte) error {
	dir := filepath.Dir(w.path)

	err := w.fs.MkdirAll(dir, dirMode)
	if err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	tmp, err := afero.TempFile(w.fs, dir, filepath.Base(w.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}

	err = errors.Join(w.writeArchive(tmp, entries), tmp.Close())
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

func (w *Writer) writeArchive(file io.Writer, entries [][]byte) error {
	buffered := bufio.NewWriter(file)

	_, _ = buffered.Write(w.head)
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
	_, _ = buffered.Write(w.tail)

	return buffered.Flush()
}

func archiveFrame(creator Creator) ([]byte, []byte) {
	data, err := json.MarshalIndent(New(creator), "", "  ")
	if err != nil {
		panic(err)
	}

	index := bytes.LastIndex(data, []byte("[]"))
	if index < 0 {
		panic("har: cannot find the entries list in the archive frame")
	}

	return data[:index], append(data[index+len("[]"):], '\n')
}
