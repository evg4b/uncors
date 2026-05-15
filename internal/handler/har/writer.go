package har

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
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
	// harDirMode is the permission bits used when creating parent directories.
	harDirMode = 0o755
)

// Writer asynchronously appends HAR entries to a file.
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

// AddEntry enqueues an entry for writing.
func (w *Writer) AddEntry(entry Entry) {
	select {
	case w.entries <- entry:
	default:
		// drop entry rather than block the request goroutine
	}
}

// Close flushes all pending entries to disk and stops the background goroutine.
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
		log.Printf("har: cannot create directory for %q: %v", w.path, err)

		return
	}

	tmp := w.path + ".tmp"

	err = os.WriteFile(tmp, data, harFileMode)
	if err != nil {
		log.Printf("har: cannot write temp file %q: %v", tmp, err)

		return
	}

	err = os.Rename(tmp, w.path)
	if err != nil {
		log.Printf("har: cannot rename %q to %q: %v", tmp, w.path, err)
		_ = os.Remove(tmp)
	}
}
