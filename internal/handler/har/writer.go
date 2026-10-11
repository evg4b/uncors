package har

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
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
	path    string
	entries chan Entry
	done    chan struct{}
	once    sync.Once
	wg      sync.WaitGroup
	mu      sync.Mutex
	all     []Entry
}

func NewWriter(path string) *Writer {
	writer := &Writer{
		path:    path,
		entries: make(chan Entry, entryChanBuffer),
		done:    make(chan struct{}),
	}

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
		return
	}

	err = os.MkdirAll(filepath.Dir(w.path), harDirMode)
	if err != nil {
		slog.Error("har: cannot create directory", "path", w.path, "error", err)

		return
	}

	tmp := w.path + ".tmp"

	err = os.WriteFile(tmp, data, harFileMode)
	if err != nil {
		slog.Error("har: cannot write temp file", "path", tmp, "error", err)

		return
	}

	err = os.Rename(tmp, w.path)
	if err != nil {
		slog.Error("har: cannot rename temp file", "from", tmp, "to", w.path, "error", err)
		_ = os.Remove(tmp)
	}
}
