package har

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/afero"
)

const (
	harVersion  = "1.2"
	creatorName = "uncors"
	// entryChanBuffer is the capacity of the entry channel.
	// Senders never block as long as fewer than this many entries are in-flight.
	entryChanBuffer = 4096
	// harFileMode is the permission bits used when writing the HAR file.
	harFileMode = 0o600
	// harDirMode is the permission bits used when creating parent directories.
	harDirMode = 0o755
	// materialiseInterval is how often the .har document is rebuilt from the
	// journal, so that it stays usable while a session is still running. It
	// bounds the rebuild rate by wall clock rather than by traffic, which is
	// what keeps a busy session from rewriting the archive per request.
	materialiseInterval = 250 * time.Millisecond
	// defaultMaxEntries bounds a long running recording. Entries beyond it are
	// dropped rather than growing the archive without limit.
	defaultMaxEntries = 100_000
)

// writerSeq makes each writer's temporary file unique, so that two writers over
// the same path can never rename each other's partial output over the target.
var writerSeq atomic.Uint64

// Writer records HAR entries to a file.
//
// Entries are appended to a journal as they arrive, which costs the same for the
// first entry and the ten thousandth, and the .har document is rebuilt from that
// journal on a slow timer and on Close. Holding the whole archive in memory and
// re-serialising it after every batch made both memory and I/O grow with the
// length of the session.
type Writer struct {
	fs             afero.Fs
	path           string
	journalPath    string
	tempPath       string
	creatorVersion string
	maxEntries     int

	entries chan Entry
	done    chan struct{}
	once    sync.Once
	wg      sync.WaitGroup

	journal   afero.File
	encoder   *json.Encoder
	count     int
	dirty     bool
	truncated bool
}

// WriterOption configures a Writer.
type WriterOption = func(*Writer)

// WithCreatorVersion records which uncors build produced the archive.
func WithCreatorVersion(version string) WriterOption {
	return func(w *Writer) {
		w.creatorVersion = version
	}
}

// WithMaxEntries bounds how many entries an archive keeps.
func WithMaxEntries(maxEntries int) WriterOption {
	return func(w *Writer) {
		w.maxEntries = maxEntries
	}
}

// NewWriter creates a Writer that records entries to path.
func NewWriter(fs afero.Fs, path string, options ...WriterOption) *Writer {
	writer := &Writer{
		fs:             fs,
		path:           path,
		journalPath:    path + ".jsonl",
		tempPath:       fmt.Sprintf("%s.%d.tmp", path, writerSeq.Add(1)),
		creatorVersion: "dev",
		maxEntries:     defaultMaxEntries,
		entries:        make(chan Entry, entryChanBuffer),
		done:           make(chan struct{}),
	}

	for _, option := range options {
		option(writer)
	}

	writer.wg.Add(1)

	go writer.run()

	return writer
}

// TempPath is the file this writer stages the archive in before publishing it.
// It is unique per writer, so two writers over the same path can never rename
// each other's partial output over the target.
func (w *Writer) TempPath() string {
	return w.tempPath
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

	ticker := time.NewTicker(materialiseInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-w.entries:
			w.record(entry)

		case <-ticker.C:
			w.materialise(false)

		case <-w.done:
			w.drain()
			// The archive is always written on close, even when nothing was
			// recorded: an empty but valid HAR file is what a user who enabled
			// recording expects to find.
			w.materialise(true)
			w.closeJournal()

			return
		}
	}
}

// drain appends every entry currently queued without blocking.
func (w *Writer) drain() {
	for {
		select {
		case entry := <-w.entries:
			w.record(entry)
		default:
			return
		}
	}
}

func (w *Writer) record(entry Entry) {
	if w.maxEntries > 0 && w.count >= w.maxEntries {
		if !w.truncated {
			w.truncated = true

			slog.Warn("har: entry limit reached, further entries are dropped",
				"path", w.path, "limit", w.maxEntries)
		}

		return
	}

	err := w.appendJournal(entry)
	if err != nil {
		slog.Error("har: cannot record entry", "path", w.journalPath, "err", err)

		return
	}

	w.count++
	w.dirty = true
}

func (w *Writer) appendJournal(entry Entry) error {
	if w.journal == nil {
		err := w.fs.MkdirAll(filepath.Dir(w.path), harDirMode)
		if err != nil {
			return fmt.Errorf("cannot create directory: %w", err)
		}

		file, err := w.fs.OpenFile(w.journalPath, journalFlags, harFileMode)
		if err != nil {
			return fmt.Errorf("cannot open journal: %w", err)
		}

		w.journal = file
		w.encoder = json.NewEncoder(file)
	}

	err := w.encoder.Encode(entry)
	if err != nil {
		return fmt.Errorf("cannot append entry: %w", err)
	}

	return nil
}

// materialise rebuilds the .har document from the journal. The entries are
// streamed rather than held, so the cost is proportional to what was recorded
// since the process started, not to the square of it.
func (w *Writer) materialise(force bool) {
	if !w.dirty && !force {
		return
	}

	w.dirty = false

	if w.journal != nil {
		err := w.journal.Sync()
		if err != nil {
			slog.Error("har: cannot flush journal", "path", w.journalPath, "err", err)

			return
		}
	}

	err := w.writeArchive()
	if err != nil {
		slog.Error("har: cannot write archive", "path", w.path, "err", err)

		return
	}

	err = w.fs.Rename(w.tempPath, w.path)
	if err != nil {
		slog.Error("har: cannot publish archive", "from", w.tempPath, "to", w.path, "err", err)
		_ = w.fs.Remove(w.tempPath)
	}
}

func (w *Writer) writeArchive() error {
	var journal io.Reader = strings.NewReader("")

	if w.journal == nil {
		err := w.fs.MkdirAll(filepath.Dir(w.path), harDirMode)
		if err != nil {
			return fmt.Errorf("cannot create directory: %w", err)
		}
	} else {
		source, err := w.fs.Open(w.journalPath)
		if err != nil {
			return fmt.Errorf("cannot read journal: %w", err)
		}

		defer source.Close()

		journal = source
	}

	target, err := w.fs.OpenFile(w.tempPath, archiveFlags, harFileMode)
	if err != nil {
		return fmt.Errorf("cannot open archive: %w", err)
	}

	defer target.Close()

	err = writeEnvelope(target, w.creatorVersion, journal)
	if err != nil {
		return err
	}

	return nil
}

// writeEnvelope streams the journal entries into the HAR document structure.
func writeEnvelope(target io.Writer, creatorVersion string, journal io.Reader) error {
	// The header values are marshalled rather than formatted, so that escaping
	// is the JSON package's problem and not ours.
	version, err := json.Marshal(harVersion)
	if err != nil {
		return fmt.Errorf("cannot encode har version: %w", err)
	}

	creator, err := json.Marshal(Creator{Name: creatorName, Version: creatorVersion})
	if err != nil {
		return fmt.Errorf("cannot encode har creator: %w", err)
	}

	_, err = fmt.Fprintf(target, `{"log":{"version":%s,"creator":%s,"entries":[`, version, creator)
	if err != nil {
		return fmt.Errorf("cannot write archive header: %w", err)
	}

	err = copyEntries(target, journal)
	if err != nil {
		return err
	}

	_, err = target.Write([]byte("]}}"))
	if err != nil {
		return fmt.Errorf("cannot write archive footer: %w", err)
	}

	return nil
}

// copyEntries streams the journal's entries into the archive's entries array.
func copyEntries(target io.Writer, journal io.Reader) error {
	decoder := json.NewDecoder(journal)

	for index := 0; ; index++ {
		var entry json.RawMessage

		err := decoder.Decode(&entry)
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("cannot read journal entry: %w", err)
		}

		if index > 0 {
			_, err = target.Write([]byte(","))
			if err != nil {
				return fmt.Errorf("cannot write archive entry: %w", err)
			}
		}

		_, err = target.Write(entry)
		if err != nil {
			return fmt.Errorf("cannot write archive entry: %w", err)
		}
	}
}

func (w *Writer) closeJournal() {
	if w.journal == nil {
		return
	}

	err := w.journal.Close()
	if err != nil {
		slog.Error("har: cannot close journal", "path", w.journalPath, "err", err)
	}

	w.journal = nil

	// The journal has served its purpose. One left behind means the process was
	// killed mid-session; the recording can be recovered from it.
	err = w.fs.Remove(w.journalPath)
	if err != nil {
		slog.Warn("har: cannot remove journal", "path", w.journalPath, "err", err)
	}
}
