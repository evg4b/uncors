package har

import (
	"encoding/json"
	"os"
	"sync"
)

const (
	harVersion     = "1.2"
	creatorName    = "uncors"
	creatorVersion = "dev"
	// entryChanBuffer is the capacity of the entry channel.
	// Senders never block as long as fewer than this many entries are in-flight.
	entryChanBuffer = 4096
	// harFileMode is the permission bits used when writing the HAR file.
	harFileMode = 0o600
)

// Writer asynchronously appends HAR entries to a file.
// All public methods are safe for concurrent use.
// The internal goroutine serializes writes so the file is never corrupted.
type Writer struct {
	path    string
	entries chan Entry
	done    chan struct{}
	once    sync.Once
	wg      sync.WaitGroup
	mu      sync.Mutex
	all     []Entry
}

// NewWriter creates a Writer that records entries to path.
// The background goroutine starts immediately.
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

// AddEntry enqueues an entry for writing. It never blocks the caller:
// if the internal buffer is full the entry is silently dropped.
func (w *Writer) AddEntry(entry Entry) {
	select {
	case w.entries <- entry:
	default:
		// drop entry rather than block the request goroutine
	}
}

// Close flushes all pending entries to disk and stops the background goroutine.
// Calling Close more than once is safe; subsequent calls are no-ops.
// It implements io.Closer.
func (w *Writer) Close() error {
	w.once.Do(func() {
		close(w.done)
		w.wg.Wait()
	})

	return nil
}

// run is the single goroutine responsible for accumulating entries and
// flushing them to disk. It exits when done is closed and the entry
// channel has been fully drained.
func (w *Writer) run() {
	defer w.wg.Done()

	for {
		select {
		case entry := <-w.entries:
			w.append(entry)
			w.flush()

		case <-w.done:
			// Drain any entries that arrived before the channel was closed.
			for {
				select {
				case entry := <-w.entries:
					w.append(entry)
				default:
					w.flush()

					return
				}
			}
		}
	}
}

func (w *Writer) append(entry Entry) {
	w.mu.Lock()
	w.all = append(w.all, entry)
	w.mu.Unlock()
}

// flush serialises the full in-memory HAR to a temp file then atomically
// renames it over the target path, so readers always see a valid file.
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

	tmp := w.path + ".tmp"

	err = os.WriteFile(tmp, data, harFileMode)
	if err != nil {
		return
	}

	_ = os.Rename(tmp, w.path)
}
