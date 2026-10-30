package har

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/afero"
)

const (
	DefaultQueueSize     = 4096
	DefaultFlushInterval = time.Second

	flushCostFactor = 10

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

type encodedEntry struct {
	started time.Time
	data    []byte
}

type spooledEntry struct {
	started time.Time
	offset  int64
	size    int64
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
	encoded    []encodedEntry
	spool      afero.File
	spoolEnd   int64
	spooled    []spooledEntry
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
		cost  time.Duration
	)

	for {
		select {
		case entry := <-w.entries:
			w.append(entry)
			w.drain()

			if tick == nil {
				timer = time.NewTimer(max(w.flushInterval, cost*flushCostFactor))
				tick = timer.C
			}

		case <-tick:
			tick = nil

			cost = w.flush()

		case <-w.done:
			if timer != nil {
				timer.Stop()
			}

			if !w.discarded.Load() {
				w.drain()
				w.flush()
			}

			w.removeSpool()

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

	index := len(w.encoded)
	for index > 0 && w.encoded[index-1].started.After(entry.StartedDateTime) {
		index--
	}

	w.encoded = slices.Insert(w.encoded, index, encodedEntry{started: entry.StartedDateTime, data: data})
	w.size += len(data)
}

func (w *Writer) flush() time.Duration {
	start := time.Now()

	w.mu.Lock()

	err := w.fs.MkdirAll(filepath.Dir(w.path), dirMode)
	if err != nil {
		err = fmt.Errorf("cannot create directory: %w", err)
	} else {
		err = w.spoolEncoded()
	}

	if err == nil {
		err = w.write()
	}

	entries, size := len(w.spooled)+len(w.encoded), w.size

	w.mu.Unlock()

	duration := time.Since(start)

	if w.onFlush != nil {
		w.onFlush(Flush{
			Path:     w.path,
			Entries:  entries,
			Bytes:    size,
			Start:    start,
			Duration: duration,
			Err:      err,
		})
	}

	return duration
}

func (w *Writer) spoolEncoded() error {
	if len(w.encoded) == 0 {
		return nil
	}

	if w.spool == nil {
		spool, err := afero.TempFile(w.fs, filepath.Dir(w.path), "."+filepath.Base(w.path)+".*.spool")
		if err != nil {
			return fmt.Errorf("cannot create spool file: %w", err)
		}

		w.spool = spool
	}

	_, err := w.spool.Seek(w.spoolEnd, io.SeekStart)
	if err != nil {
		return fmt.Errorf("cannot write spool file: %w", err)
	}

	buffered := bufio.NewWriter(w.spool)
	added := make([]spooledEntry, 0, len(w.encoded))
	offset := w.spoolEnd

	for _, entry := range w.encoded {
		_, _ = buffered.Write(entry.data)
		added = append(added, spooledEntry{started: entry.started, offset: offset, size: int64(len(entry.data))})
		offset += int64(len(entry.data))
	}

	err = buffered.Flush()
	if err != nil {
		return fmt.Errorf("cannot write spool file: %w", err)
	}

	w.spoolEnd = offset
	w.spooled = mergeByStart(w.spooled, added)
	w.encoded = w.encoded[:0]

	return nil
}

func mergeByStart(spooled, added []spooledEntry) []spooledEntry {
	if len(spooled) == 0 || !added[0].started.Before(spooled[len(spooled)-1].started) {
		return append(spooled, added...)
	}

	merged := make([]spooledEntry, 0, len(spooled)+len(added))

	for len(spooled) > 0 && len(added) > 0 {
		if added[0].started.Before(spooled[0].started) {
			merged = append(merged, added[0])
			added = added[1:]
		} else {
			merged = append(merged, spooled[0])
			spooled = spooled[1:]
		}
	}

	return append(append(merged, spooled...), added...)
}

func (w *Writer) removeSpool() {
	if w.spool == nil {
		return
	}

	_ = w.spool.Close()
	_ = w.fs.Remove(w.spool.Name())
	w.spool = nil
}

func (w *Writer) write() error {
	dir := filepath.Dir(w.path)

	tmp, err := afero.TempFile(w.fs, dir, filepath.Base(w.path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}

	err = errors.Join(w.writeArchive(tmp), tmp.Close())
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

func (w *Writer) writeArchive(file io.Writer) error {
	buffered := bufio.NewWriter(file)

	_, _ = buffered.Write(w.head)
	_ = buffered.WriteByte('[')

	for i, entry := range w.spooled {
		if i > 0 {
			_ = buffered.WriteByte(',')
		}

		_ = buffered.WriteByte('\n')
		_, _ = buffered.WriteString(entryPrefix)

		_, err := io.Copy(buffered, io.NewSectionReader(w.spool, entry.offset, entry.size))
		if err != nil {
			return fmt.Errorf("cannot read spool file: %w", err)
		}
	}

	if len(w.spooled) > 0 {
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
