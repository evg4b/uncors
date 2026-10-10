package testutils

import (
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/spf13/afero"
)

// TrackingFs wraps an afero.Fs and records the files opened through it, and
// counts those not closed yet, so tests can detect leaked file handles.
type TrackingFs struct {
	afero.Fs

	open   atomic.Int64
	mu     sync.Mutex
	opened []string
}

func NewTrackingFs(fs afero.Fs) *TrackingFs {
	return &TrackingFs{Fs: fs}
}

// OpenFiles returns the number of files that are still open.
func (f *TrackingFs) OpenFiles() int64 {
	return f.open.Load()
}

// Opened returns the names of every open attempt, successful or not.
func (f *TrackingFs) Opened() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.opened)
}

func (f *TrackingFs) Open(name string) (afero.File, error) {
	file, err := f.Fs.Open(name)

	return f.track(name, file, err)
}

func (f *TrackingFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file, err := f.Fs.OpenFile(name, flag, perm)

	return f.track(name, file, err)
}

func (f *TrackingFs) Create(name string) (afero.File, error) {
	file, err := f.Fs.Create(name)

	return f.track(name, file, err)
}

func (f *TrackingFs) track(name string, file afero.File, err error) (afero.File, error) {
	f.mu.Lock()
	f.opened = append(f.opened, name)
	f.mu.Unlock()

	if err != nil {
		return file, err
	}

	f.open.Add(1)

	return &trackedFile{File: file, fs: f}, nil
}

type trackedFile struct {
	afero.File

	fs   *TrackingFs
	once sync.Once
}

func (t *trackedFile) Close() error {
	t.once.Do(func() { t.fs.open.Add(-1) })

	return t.File.Close()
}
