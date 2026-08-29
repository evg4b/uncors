package har

import (
	"errors"
	"sync"

	"github.com/spf13/afero"
)

// WriterPool hands out one Writer per archive path.
//
// A recording belongs to the file it is written to, not to the configuration
// generation that started it: saving a config file should not truncate the
// archive or hand a second writer the same path. One writer per path means the
// archive keeps accumulating across reloads, and there is never more than one
// goroutine or one journal per file.
type WriterPool struct {
	fs             afero.Fs
	creatorVersion string

	mu      sync.Mutex
	writers map[string]*Writer
}

func NewWriterPool(fs afero.Fs, creatorVersion string) *WriterPool {
	return &WriterPool{
		fs:             fs,
		creatorVersion: creatorVersion,
		writers:        map[string]*Writer{},
	}
}

// For returns the writer recording to path, creating it on first use.
func (p *WriterPool) For(path string) *Writer {
	p.mu.Lock()
	defer p.mu.Unlock()

	if writer, ok := p.writers[path]; ok {
		return writer
	}

	writer := NewWriter(p.fs, path, WithCreatorVersion(p.creatorVersion))
	p.writers[path] = writer

	return writer
}

// Close finalises every archive: each writer flushes what it holds and removes
// its journal.
func (p *WriterPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	errs := make([]error, 0, len(p.writers))

	for path, writer := range p.writers {
		errs = append(errs, writer.Close())

		delete(p.writers, path)
	}

	return errors.Join(errs...)
}
